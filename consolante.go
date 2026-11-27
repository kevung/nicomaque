package tournoi

import (
	"fmt"
	"time"
)

// ---- Phases latérales : une consolante alimentée par les perdants d'une seule ronde ----
//
// Une phase latérale (PhaseConfig.ConsolationOf) se joue à côté de la phase principale, qui
// reste State.Current. Elle s'ouvre par side_phase_opened, qui porte ses entrants comme un
// tirage : rejouer un journal ne dépend pas de la règle « perdants de la ronde ». Elle finit
// quand phaseDone le dit, ou quand le directeur y renonce (side_phase_waived) avant tout match
// lancé : close sans jeu, elle ne retient plus la clôture.
//
// Un joueur n'a jamais deux matchs à la fois (busy regarde toutes les phases). Quand les deux
// phases le réclament, la principale l'emporte : la latérale ne lui propose rien tant qu'il
// figure dans une proposition de la principale.

// hasSides : la configuration porte au moins une phase latérale.
func (c *Config) hasSides() bool {
	for _, p := range c.Phases {
		if p.Side() {
			return true
		}
	}
	return false
}

// sideSource : les perdants des matchs joués de la ronde source de la phase latérale k, et si
// cette ronde est soldée. Une exemption, un forfait ou un tour non joué ne donnent pas de
// perdant. Les retirés y figurent : c'est à l'ouverture qu'on les écarte.
func (s *State) sideSource(k int) ([]PlayerID, bool) {
	src := s.Config.Phases[k].ConsolationOf
	ph := s.phaseOf(src.Phase)
	if ph == nil || ph.Side {
		return nil, false
	}
	var out []PlayerID
	if ph.Cfg.Kind == KindBracket {
		sec := ph.section(secMain)
		if !ph.Drawn || sec == nil || src.Round > len(sec.Rounds) {
			return nil, false
		}
		for _, i := range sec.Rounds[src.Round-1] {
			g := sec.Matches[i]
			if !g.Done && !g.Skipped {
				return nil, false
			}
			if m := s.Matches[g.MatchID]; m != nil && !g.Walkover && m.Status == Finished && !m.Forfeit {
				out = append(out, g.Loser)
			}
		}
		return out, true
	}
	if ph.Round < src.Round {
		return nil, false
	}
	if ph.Round == src.Round {
		engagé := s.engagedIn(ph, src.Round)
		for _, p := range ph.Roster {
			if !engagé[p] && s.remainingLives(ph, p) > 0 && !s.Withdrawn[p] {
				return nil, false // la ronde est encore ouverte
			}
		}
	}
	for _, id := range s.MatchOrder {
		m := s.Matches[id]
		if m.Phase != src.Phase || m.Round != src.Round {
			continue
		}
		switch {
		case m.Status == Running:
			return nil, false
		case m.Status == Finished && !m.Forfeit:
			out = append(out, m.Loser())
		}
	}
	return out, true
}

// sideOpenable : la phase latérale k peut s'ouvrir — pas encore ouverte, ronde source soldée,
// au moins deux perdants encore là. Rend ses entrants.
func (s *State) sideOpenable(k int) ([]PlayerID, bool) {
	if ph := s.phaseOf(k); ph != nil && (ph.Opened || ph.Waived) {
		return nil, false
	}
	losers, ok := s.sideSource(k)
	if !ok {
		return nil, false
	}
	var in []PlayerID
	for _, p := range losers {
		if !s.Withdrawn[p] {
			in = append(in, p)
		}
	}
	return in, len(in) >= 2
}

// sidesPending : une phase latérale reste à ouvrir ou à finir. La clôture l'attend ; une
// latérale qui n'a jamais eu deux perdants ne retient rien.
func (s *State) sidesPending() bool {
	for k, p := range s.Config.Phases {
		if !p.Side() {
			continue
		}
		if ph := s.phaseOf(k); ph != nil && ph.Opened {
			if !ph.Waived && !s.phaseDone(ph) {
				return true
			}
			continue
		}
		if _, ok := s.sideOpenable(k); ok {
			return true
		}
	}
	return false
}

// WaiverRefusal : le renoncement à une phase latérale est refusé parce qu'un de ses matchs est
// lancé (Reason : RefusalStarted ; Match : le premier). Un match annulé ne compte pas.
type WaiverRefusal struct {
	Phase  int     `json:"phase"`
	Reason string  `json:"reason"`
	Match  MatchID `json:"match"`
}

func (r *WaiverRefusal) Error() string {
	return fmt.Sprintf("phase %d : on ne renonce plus à la consolante, le match %s y est lancé", r.Phase, r.Match)
}

// sideStarted : le premier match lancé, en cours ou fini, de la phase k.
func (s *State) sideStarted(k int) (MatchID, bool) {
	for _, id := range s.MatchOrder {
		if m := s.Matches[id]; m.Phase == k && m.Status != Cancelled {
			return id, true
		}
	}
	return "", false
}

// applySideWaived : la phase latérale est close sans jeu. Fermée, elle garde les entrants que
// porte l'événement (ceux qu'elle aurait eus), pour que son classement les dise abandonnés.
func (s *State) applySideWaived(ev Event) error {
	k := ev.Phase
	if k < 0 || k >= len(s.Config.Phases) || !s.Config.Phases[k].Side() {
		return fmt.Errorf("phase %d : pas une phase latérale", k)
	}
	if ph := s.phaseOf(k); ph != nil && ph.Waived {
		return fmt.Errorf("phase %d : déjà abandonnée", k)
	}
	if id, ok := s.sideStarted(k); ok {
		return &WaiverRefusal{Phase: k, Reason: RefusalStarted, Match: id}
	}
	if s.phaseOf(s.Config.Phases[k].ConsolationOf.Phase) == nil {
		return fmt.Errorf("phase %d : sa phase source n'a pas commencé", k)
	}
	if err := s.growPhases(k + 1); err != nil {
		return err
	}
	ph := s.Phases[k]
	if ev.Draw != nil {
		if ph.Opened || len(ev.Draw.Groups) != 1 {
			return fmt.Errorf("phase %d : side_phase_waived porte des entrants d'une phase ouverte, ou plusieurs groupes", k)
		}
		vus := map[PlayerID]bool{}
		for _, p := range ev.Draw.Groups[0] {
			if _, ok := s.Players[p]; !ok || vus[p] {
				return fmt.Errorf("phase %d : entrant %q inconnu ou répété", k, p)
			}
			vus[p] = true
		}
		for _, p := range ev.Draw.Groups[0] {
			s.enter(ph, p, livesFor(ph.Cfg))
		}
	}
	ph.Waived = true
	return nil
}

// waivedRanking : chaque entrant d'une phase abandonnée, au rang 0 (aucun prix), note waived.
func waivedRanking(ph *PhaseState) []Rank {
	out := make([]Rank, len(ph.Entrants))
	for i, p := range ph.Entrants {
		out[i] = Rank{Player: p, Note: Note{Kind: NoteWaived}}
	}
	return out
}

func (s *State) applySideOpened(ev Event) error {
	k := ev.Phase
	if k < 0 || k >= len(s.Config.Phases) || !s.Config.Phases[k].Side() {
		return fmt.Errorf("phase %d : pas une phase latérale", k)
	}
	if ev.Draw == nil || len(ev.Draw.Groups) != 1 || len(ev.Draw.Groups[0]) < 2 {
		return fmt.Errorf("phase %d : side_phase_opened attend un groupe d'au moins deux entrants", k)
	}
	if ph := s.phaseOf(k); ph != nil && (ph.Opened || ph.Waived) {
		return fmt.Errorf("phase %d : déjà ouverte ou abandonnée", k)
	}
	if src := s.phaseOf(s.Config.Phases[k].ConsolationOf.Phase); src == nil {
		return fmt.Errorf("phase %d : sa phase source n'a pas commencé", k)
	}
	vus := map[PlayerID]bool{}
	for _, p := range ev.Draw.Groups[0] {
		if _, ok := s.Players[p]; !ok || vus[p] {
			return fmt.Errorf("phase %d : entrant %q inconnu ou répété", k, p)
		}
		vus[p] = true
	}
	if err := s.growPhases(k + 1); err != nil {
		return err
	}
	ph := s.Phases[k]
	ph.Opened = true
	for _, p := range ev.Draw.Groups[0] {
		s.enter(ph, p, livesFor(ph.Cfg))
	}
	return nil
}

// proposeSides : après les propositions de la principale (main), celles de chaque phase
// latérale ouverte et non finie, puis l'ouverture de celles qui le peuvent, par indice.
//
// Les joueurs d'une proposition déjà faite comptent comme occupés pour les suivantes, et leurs
// tables comme prises : deux propositions ne se disputent ni un joueur ni une table.
func (s *State) proposeSides(main []Action, now time.Time, ext External) []Action {
	acts := main
	tables := append([]int(nil), ext.BusyTables...)
	pris := map[PlayerID]bool{}
	note := func(as []Action) {
		for _, a := range as {
			if a.Kind == ActStartMatch || a.Kind == ActBye {
				pris[a.A], pris[a.B] = true, true
				if a.Table > 0 {
					tables = append(tables, a.Table)
				}
			}
		}
		delete(pris, "")
	}
	note(main)
	for k, p := range s.Config.Phases {
		if !p.Side() {
			continue
		}
		ph := s.phaseOf(k)
		if ph != nil && ph.Waived {
			continue
		}
		nom := Label{Kind: LabelPhase, Text: PhaseName(p)}
		if ph != nil && ph.Opened {
			if s.phaseDone(ph) {
				continue
			}
			v := s.occupiedFor(ph, pris)
			as := v.proposeIn(ph, now, External{BusyTables: tables})
			note(as)
			acts = append(acts, as...)
			if _, lancé := s.sideStarted(k); !lancé {
				acts = append(acts, Action{Kind: ActWaiveSidePhase, Phase: k, Label: nom})
			}
			continue
		}
		if in, ok := s.sideOpenable(k); ok {
			acts = append(acts, Action{Kind: ActOpenSidePhase, Phase: k, Label: nom, Draw: &Draw{Groups: [][]PlayerID{in}}},
				Action{Kind: ActWaiveSidePhase, Phase: k, Label: nom, Draw: &Draw{Groups: [][]PlayerID{in}}})
		}
	}
	return acts
}

// occupiedFor : une copie de l'état pour proposer dans ph, où comptent comme occupés les
// joueurs qui jouent dans une autre phase et ceux que réclame déjà une proposition (pris).
// Ils rejoignent ceux de l'extérieur (ailleurs) : un graphe retient leur match, un suisse ne
// les apparie pas — et n'ouvre pas de ronde sans eux (waitOccupied).
func (s *State) occupiedFor(ph *PhaseState, pris map[PlayerID]bool) *State {
	occ := map[PlayerID]bool{}
	for p := range pris {
		occ[p] = true
	}
	for _, id := range s.MatchOrder {
		if m := s.Matches[id]; m.Status == Running && m.Phase != ph.Index {
			occ[m.A], occ[m.B] = true, true
		}
	}
	v := *s
	v.occupés = occ
	if len(occ) > 0 {
		v.ailleurs = make(map[PlayerID]bool, len(s.ailleurs)+len(occ))
		for p := range s.ailleurs {
			v.ailleurs[p] = true
		}
		for p := range occ {
			v.ailleurs[p] = true
		}
	}
	return &v
}

// waitOccupied : un suisse par rondes n'ouvre pas de ronde tant qu'un de ses joueurs joue dans
// une autre phase. Le relevé des appelés (openRound) l'oublierait, et il manquerait la ronde.
func (s *State) waitOccupied(ph *PhaseState) []Action {
	var out []Action
	for _, p := range s.alive(ph) {
		if s.occupés[p] && !s.absent(p, ph.Round+1) {
			out = append(out, Action{Kind: ActWait, Phase: ph.Index, Reason: ReasonPlayerBusy, A: p})
		}
	}
	return out
}

// sideWarnings : une correction de la ronde source après l'ouverture change ses perdants. La
// consolante n'est pas refaite — ses entrants sont dans le journal ; le TD est prévenu, joueur
// par joueur.
func (s *State) sideWarnings() []Warning {
	var w []Warning
	for _, ph := range s.Phases {
		if !ph.Side || !ph.Opened || ph.Waived {
			continue
		}
		losers, ok := s.sideSource(ph.Index)
		if !ok {
			continue
		}
		perdant := map[PlayerID]bool{}
		for _, p := range losers {
			perdant[p] = true
		}
		for _, p := range ph.Entrants {
			if !perdant[p] {
				w = append(w, Warning{Code: WarnConsolationNotLoser, Section: SidePhaseSection(ph.Index), Player: p})
			}
			delete(perdant, p)
		}
		for _, p := range losers {
			if perdant[p] && !s.Withdrawn[p] {
				w = append(w, Warning{Code: WarnConsolationMissing, Section: SidePhaseSection(ph.Index), Player: p})
			}
		}
	}
	return w
}
