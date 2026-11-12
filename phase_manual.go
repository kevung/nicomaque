package tournoi

import (
	"fmt"
	"time"
)

// ---- Phase à appariement manuel ----
//
// Le moteur n'y apparie personne. Le directeur pose chaque match et chaque exemption dans une
// ronde ; le moteur ne vérifie que ce qui rendrait la ronde incohérente : un joueur de la phase,
// non retiré, au plus une fois par ronde, et des rondes numérotées sans trou. La comptabilité
// (victoires, exemptions, adversaires) et le classement sont ceux du suisse classique.

// proposeManual : l'attente, et le passage de phase ou la clôture dès qu'aucun match ne joue —
// proposés, jamais imposés : c'est le directeur qui dit quand la phase est finie.
func (s *State) proposeManual(ph *PhaseState) []Action {
	wait := Action{Kind: ActWait, Phase: ph.Index, Reason: ReasonManual}
	if s.runningInPhase(ph) > 0 {
		return []Action{wait}
	}
	if s.Current+1 < len(s.Config.Phases) {
		return []Action{wait, {Kind: ActNextPhase, Phase: ph.Index, Label: Label{Kind: LabelPhase, Text: PhaseName(s.Config.Phases[s.Current+1])}}}
	}
	return []Action{wait, {Kind: ActFinish, Phase: ph.Index}}
}

// manualCheck : ce qu'une phase manuelle refuse d'un match ou d'une exemption posés dans la
// ronde round. Sans effet sur les autres phases.
func (s *State) manualCheck(phase, round int, joueurs ...PlayerID) error {
	ph := s.phaseOf(phase)
	if ph == nil || ph.Cfg.Kind != KindManual {
		return nil
	}
	if ph.Done {
		return fmt.Errorf("phase %d terminée", phase)
	}
	if round < 1 || round > ph.Round+1 {
		return fmt.Errorf("phase %d : ronde %d invalide (rondes 1 à %d)", phase, round, ph.Round+1)
	}
	entrant := map[PlayerID]bool{}
	for _, p := range ph.Entrants {
		entrant[p] = true
	}
	for _, p := range joueurs {
		if !entrant[p] {
			return fmt.Errorf("joueur %s hors de la phase %d", p, phase)
		}
		if s.Withdrawn[p] {
			return fmt.Errorf("joueur %s retiré", p)
		}
		if s.inRound(ph, round, p) {
			return fmt.Errorf("joueur %s déjà placé dans la ronde %d", p, round)
		}
	}
	return nil
}

// inRound : le joueur a déjà un match non annulé ou une exemption dans la ronde.
func (s *State) inRound(ph *PhaseState, round int, p PlayerID) bool {
	if containsRound(ph.ByeRounds[p], round) {
		return true
	}
	for _, id := range s.MatchOrder {
		m := s.Matches[id]
		if m.Phase == ph.Index && m.Round == round && m.Status != Cancelled && (m.A == p || m.B == p) {
			return true
		}
	}
	return false
}

// ManualMatchEvent : le directeur pose le match a contre b dans la ronde round de la phase
// manuelle courante (round = dernière ronde + 1 ouvre une ronde). La longueur est celle de la
// phase ; la table reste à l'hôte (table_changed).
func (s *State) ManualMatchEvent(round int, a, b PlayerID, now time.Time) Event {
	ev := Event{Version: JournalVersion, Kind: EvMatchStarted, Time: now, Phase: s.Current, Round: round,
		MatchID: s.nextMatchID(), A: a, B: b, Label: Label{Kind: LabelRound, N: round}}
	if ph := s.phase(); ph != nil {
		ev.Length = ph.Length
	}
	return ev
}

// ManualByeEvent : le directeur exempte le joueur pour la ronde round de la phase manuelle
// courante. L'exemption compte un point, comme au suisse classique.
func (s *State) ManualByeEvent(round int, p PlayerID, now time.Time) Event {
	return Event{Version: JournalVersion, Kind: EvBye, Time: now, Phase: s.Current, Round: round, ID: p,
		Label: Label{Kind: LabelRound, N: round}}
}
