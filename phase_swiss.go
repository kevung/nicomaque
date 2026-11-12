package tournoi

import (
	"fmt"

	"github.com/kevung/nicomaque/rules"
)

// ---- Suisse à vies : appariement dans le groupe de même nombre de défaites ----

// sumLives : somme des vies restantes des joueurs en vie.
func (s *State) sumLives(ph *PhaseState) int {
	n := 0
	for _, p := range ph.Entrants {
		n += s.remainingLives(ph, p)
	}
	return n
}

// swissDone : la phase suisse n'a plus rien à jouer (rules.PhaseDone).
//
// Suisse classique : la dernière ronde peut se lancer en plusieurs vagues. Tant qu'il en reste à
// confirmer, la phase n'est pas finie, même sans match en cours.
func (s *State) swissDone(ph *PhaseState) bool {
	if !rules.PhaseDone(s.bilan(ph)) {
		return false
	}
	if rolling(ph.Cfg) {
		return true
	}
	return ph.Cfg.Kind != KindSwissRounds || ph.Round == 0 || len(s.roundRest(ph, ph.Round)) == 0
}

// swissKind : les deux suisses — à vies, ou classique à rondes fixes — partagent la mécanique
// des rondes, des absences et des retardataires.
func swissKind(k string) bool { return k == KindSwissLives || noLives(k) }

// free : joueurs en vie sans match en cours, ni ici ni dans une autre épreuve de la salle, et
// disponibles pour la ronde r (0 = hors rondes ; voir absence.go).
func (s *State) free(ph *PhaseState, r int) []PlayerID {
	var out []PlayerID
	for _, p := range s.alive(ph) {
		if !s.busy(p) && !s.absent(p, r) && !s.ailleurs[p] && !containsRound(ph.HalfByes[p], r) {
			out = append(out, p)
		}
	}
	return out
}

// swissLength : longueur des matchs à lancer maintenant dans un suisse (rules.SwissLength).
func (s *State) swissLength(ph *PhaseState) int { return rules.SwissLength(s.bilan(ph)) }

// proposeSwiss : enveloppe de rules.Pair. Une ronde ouverte d'abord : ce qui en reste à
// confirmer (roundRest) ; sinon les appariements de rules, sur les joueurs libres.
func (s *State) proposeSwiss(ph *PhaseState) []Action {
	if s.swissDone(ph) {
		return nil
	}
	if rolling(ph.Cfg) && ph.Round <= 1 {
		// La ronde 1 se lance comme une ronde ordinaire : son plan, une fois ouvert, se
		// repropose à l'identique. Le fil de l'eau commence quand elle est entièrement lancée.
		if ph.Round == 1 {
			if rest := s.roundRest(ph, 1); len(rest) > 0 {
				return rest
			}
		} else {
			avant := s.snapshot(ph, s.free(ph, 1))
			avant.Rules.Mode = rules.ModeRounds
			return s.actions(ph, rules.Pair(avant, s.seed()))
		}
	}
	r := 0
	if byRounds(ph.Cfg) {
		if ph.Round > 0 {
			if rest := s.roundRest(ph, ph.Round); len(rest) > 0 {
				return rest
			}
		}
		if s.runningInPhase(ph) > 0 {
			return nil
		}
		r = ph.Round + 1
	}
	return s.actions(ph, rules.Pair(s.snapshot(ph, s.free(ph, r)), s.seed()))
}

// Suisse par rondes : tous les matchs de la ronde en même temps, byes aux groupes impairs
// (rules.PairSwissRound).
//
// UNE RONDE RESTE OUVERTE tant que l'un des joueurs appelés n'y est pas engagé. Une ronde de
// 12 matchs dans une salle de 6 tables se lance en deux vagues ; un bye se confirme avant les
// matchs. Le moteur ne proposait une ronde qu'avec zéro match en cours, puis appariait « la
// ronde suivante » : lancer la première vague faisait disparaître la seconde, et confirmer le
// bye seul faisait passer à la ronde N+1, l'exempté réapparié. Désormais, tant que la ronde
// Round est ouverte, ses appariements non lancés restent proposés, LES MÊMES ; la ronde suivante
// ne vient que quand la ronde est entièrement engagée et ses matchs finis.
//
// Les mêmes appariements, parce qu'ils sont RECALCULÉS à l'identique plutôt que stockés : le
// plan d'une ronde est fonction de l'état d'avant la ronde (beforeRound retire ce que la ronde a
// déjà produit), des joueurs appelés (PhaseState.Roster, relevé au premier événement de la
// ronde) et d'une graine propre à la ronde (rules.Seed.Round). Le journal ne porte que ce qui a été
// confirmé, comme toujours.
//
// La bascule vers un tableau (Target) demande que la somme des vies TOMBE JUSTE sur une
// puissance de 2. Chaque match la fait décroître d'une unité, mais une ronde en lance beaucoup
// d'un coup : sans garde-fou, la dernière ronde passe sous la cible et le tableau qui suit
// n'est plus complet. Mesuré avant correction sur 2 vies et une cible de 16 : la somme
// atterrissait entre 12 et 15 selon l'effectif, jamais sur 16, et le tableau se remplissait de
// byes qui n'avaient pas lieu d'être.
//
// Le mode continu avait déjà ce budget ; les rondes ne l'avaient pas. La DERNIÈRE ronde d'une
// phase à bascule est donc tronquée au nombre exact de matchs qui reste à jouer, et ne donne
// aucun bye : elle est la dernière, et un bye enregistré fausserait l'ordre d'appariement d'une
// ronde qui n'aura pas lieu.

// openRound : au premier événement d'une ronde (match ou bye), relever les joueurs appelés —
// ceux qu'un Propose de cet instant aurait appariés. Appelée par Apply AVANT d'appliquer
// l'événement, quand ses joueurs sont encore libres.
func (s *State) openRound(ph *PhaseState, r int) {
	if !swissKind(ph.Cfg.Kind) || !(byRounds(ph.Cfg) || rolling(ph.Cfg) && r == 1) || r <= ph.Round {
		return
	}
	ph.Roster = s.free(ph, r)
}

// engagedIn : les joueurs déjà engagés dans la ronde r de la phase — un match non annulé de
// cette ronde, ou un bye de cette ronde.
func (s *State) engagedIn(ph *PhaseState, r int) map[PlayerID]bool {
	out := map[PlayerID]bool{}
	for _, id := range s.MatchOrder {
		m := s.Matches[id]
		if m.Phase == ph.Index && m.Round == r && m.Status != Cancelled {
			out[m.A], out[m.B] = true, true
		}
	}
	for p, rs := range ph.ByeRounds {
		for _, x := range rs {
			if x == r {
				out[p] = true
			}
		}
	}
	return out
}

// beforeRound : la phase telle qu'elle était avant la ronde r — victoires, défaites,
// adversaires et byes de la ronde r retirés. C'est l'état sur lequel la ronde a été appariée.
func (s *State) beforeRound(ph *PhaseState, r int) *PhaseState {
	v := *ph
	v.sansRonde = r
	v.Losses, v.Wins, v.Opponents = map[PlayerID]int{}, map[PlayerID]int{}, map[PlayerID][]PlayerID{}
	v.Byes = map[PlayerID]int{}
	for p, n := range ph.Byes {
		v.Byes[p] = n
	}
	for p, rs := range ph.ByeRounds {
		for _, x := range rs {
			if x == r {
				v.Byes[p]--
			}
		}
	}
	for _, id := range s.MatchOrder {
		m := s.Matches[id]
		if m.Phase != ph.Index || m.Status == Cancelled || m.Round == r {
			continue
		}
		v.Opponents[m.A] = append(v.Opponents[m.A], m.B)
		v.Opponents[m.B] = append(v.Opponents[m.B], m.A)
		if m.Status == Finished {
			v.Wins[m.Winner]++
			v.Losses[m.Loser()]++
		}
	}
	return &v
}

// roundRest : ce qui reste à confirmer de la ronde r, ouverte. Le plan est recalculé sur l'état
// d'avant la ronde, pour les joueurs appelés (rules.PairSwissRound) ; on garde ce dont aucun
// joueur n'est encore engagé, ni retiré, ni occupé. Les ORPHELINS — prévus contre un joueur qui a
// joué autre chose (un match lancé à la main) ou qui est parti — sont appariés entre eux
// (rules.PairSwissOrphans), dans le budget de la bascule s'il y en a une. Un orphelin seul
// attend la ronde suivante.
func (s *State) roundRest(ph *PhaseState, r int) []Action {
	if len(ph.Roster) == 0 {
		return nil
	}
	engagé := s.engagedIn(ph, r)
	v := s.beforeRound(ph, r)
	avant := s.snapshot(v, nil)
	graine := rules.Seed{Base: s.Seed, Events: s.NEvents, Phase: ph.Index}
	prévu, normal := rules.PairSwissRound(avant, r, refs(ph.Roster), graine)
	if !normal {
		return nil
	}
	plan := s.actions(ph, prévu)
	libre := func(p PlayerID) bool {
		return !engagé[p] && s.remainingLives(ph, p) > 0 && !s.busy(p) && !s.absent(p, r) && !s.ailleurs[p]
	}
	longueur := s.swissLength(ph) // la longueur est celle d'aujourd'hui (length_changed)
	var acts []Action
	pris := map[PlayerID]bool{}
	for _, a := range plan {
		switch {
		case a.Kind == ActBye && libre(a.A):
			acts = append(acts, a)
			pris[a.A] = true
		case a.Kind == ActStartMatch && libre(a.A) && libre(a.B):
			a.Length = longueur
			acts = append(acts, a)
			pris[a.A], pris[a.B] = true, true
		}
	}
	var orphelins []PlayerID
	for _, a := range plan {
		for _, p := range []PlayerID{a.A, a.B} {
			if p != "" && !pris[p] && libre(p) {
				orphelins = append(orphelins, p)
				pris[p] = true
			}
		}
	}
	if len(orphelins) < 2 {
		return acts
	}
	budget := 1 << 30
	if ph.Cfg.Target > 0 {
		budget = s.sumLives(ph) - s.runningInPhase(ph) - ph.Cfg.Target
		for _, a := range acts {
			if a.Kind == ActStartMatch {
				budget--
			}
		}
	}
	return append(acts, s.actions(ph, rules.PairSwissOrphans(avant, r, refs(orphelins), budget, longueur, graine))...)
}

// applyHalfBye pose ou retire une demi-exemption. Sur un suisse classique seulement, pour un
// joueur inscrit et non retiré, et pour une ronde qui n'est pas encore ouverte : une fois la
// ronde appariée, on n'y touche plus. Il reste toujours quelqu'un à apparier.
func (s *State) applyHalfBye(ev Event) error {
	if ev.Phase < 0 || ev.Phase >= len(s.Config.Phases) || ev.Phase < s.Current {
		return fmt.Errorf("demi-exemption : phase %d inconnue ou passée", ev.Phase)
	}
	cfg := s.Config.Phases[ev.Phase]
	if cfg.Kind != KindSwissRounds {
		return fmt.Errorf("demi-exemption : la phase %d n'est pas un suisse classique", ev.Phase)
	}
	if rolling(cfg) {
		return fmt.Errorf("demi-exemption : la phase %d se joue au fil de l'eau, sans ronde à manquer", ev.Phase)
	}
	if ev.Round < 1 || ev.Round > cfg.Rounds {
		return fmt.Errorf("demi-exemption : ronde %d hors de 1..%d", ev.Round, cfg.Rounds)
	}
	if _, ok := s.Players[ev.ID]; !ok || s.Withdrawn[ev.ID] {
		return fmt.Errorf("demi-exemption : joueur %s inconnu ou retiré", ev.ID)
	}
	ph := s.phaseOf(ev.Phase)
	if ph == nil {
		return fmt.Errorf("demi-exemption : la phase %d n'est pas commencée", ev.Phase)
	}
	if ev.Round <= ph.Round {
		return fmt.Errorf("demi-exemption : la ronde %d est déjà ouverte", ev.Round)
	}
	posée := containsRound(ph.HalfByes[ev.ID], ev.Round)
	if ev.Kind == EvHalfByeCancelled {
		if !posée {
			return fmt.Errorf("demi-exemption : %s n'en a pas à la ronde %d", ev.ID, ev.Round)
		}
		var reste []int
		for _, r := range ph.HalfByes[ev.ID] {
			if r != ev.Round {
				reste = append(reste, r)
			}
		}
		if len(reste) == 0 {
			delete(ph.HalfByes, ev.ID)
		} else {
			ph.HalfByes[ev.ID] = reste
		}
		return nil
	}
	if posée {
		return fmt.Errorf("demi-exemption : %s en a déjà une à la ronde %d", ev.ID, ev.Round)
	}
	if ph.Index == s.Current && len(ph.Entrants) > 0 {
		dedans, autres := false, 0
		for _, p := range s.alive(ph) {
			if p == ev.ID {
				dedans = true
			} else if !containsRound(ph.HalfByes[p], ev.Round) {
				autres++
			}
		}
		if !dedans {
			return fmt.Errorf("demi-exemption : %s n'est pas dans la phase %d", ev.ID, ev.Phase)
		}
		if autres == 0 {
			return fmt.Errorf("demi-exemption : plus personne à apparier à la ronde %d", ev.Round)
		}
	}
	if ph.HalfByes == nil {
		ph.HalfByes = map[PlayerID][]int{}
	}
	ph.HalfByes[ev.ID] = append(ph.HalfByes[ev.ID], ev.Round)
	return nil
}
