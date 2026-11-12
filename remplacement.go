package tournoi

import (
	"fmt"
	"reflect"
)

// ---- Remplacement d'un joueur, échange de places ----
//
// Une PLACE est tout ce que le tournoi a attaché à un identifiant : rang d'inscription, tête de
// série, place dans un tirage ou une poule, matchs qui l'attendent dans un graphe, exemptions
// reçues. Tant que son joueur n'a lancé aucun match, la place peut changer de joueur : le
// journal n'est pas réécrit, l'état dérive la substitution en renommant l'identifiant partout où
// il figure. Une demi-exemption annoncée appartient au joueur qui l'a demandée, pas à la place :
// elle se retire d'abord. Une indisponibilité aussi est au joueur : le remplaçant n'en hérite
// pas, et un échange la laisse à qui l'a déclarée.

// seatLocked : ce qui interdit de changer le joueur de la place de p.
func (s *State) seatLocked(p PlayerID) error {
	if _, ok := s.Players[p]; !ok {
		return fmt.Errorf("joueur %s inconnu", p)
	}
	if s.Withdrawn[p] {
		return fmt.Errorf("joueur %s retiré : sa place ne se reprend pas", p)
	}
	for _, id := range s.MatchOrder {
		if m := s.Matches[id]; m.Status != Cancelled && (m.A == p || m.B == p) {
			return fmt.Errorf("joueur %s a déjà joué (match %s)", p, m.ID)
		}
	}
	for _, ph := range s.Phases {
		if len(ph.HalfByes[p]) > 0 {
			return fmt.Errorf("joueur %s a une demi-exemption annoncée : la retirer d'abord", p)
		}
	}
	return nil
}

func (s *State) applyReplace(ev Event) error {
	if ev.Player == nil || ev.Player.ID == "" {
		return fmt.Errorf("remplaçant sans identifiant")
	}
	if ev.Player.ID == BYE {
		return fmt.Errorf("identifiant %s réservé à l'exemption", BYE)
	}
	if _, ok := s.Players[ev.Player.ID]; ok {
		return fmt.Errorf("joueur %s déjà inscrit", ev.Player.ID)
	}
	if err := s.seatLocked(ev.ID); err != nil {
		return err
	}
	fiche := *ev.Player
	fiche.Seed = s.Players[ev.ID].Seed // la tête de série est celle de la place
	delete(s.Unavailable, ev.ID)       // l'indisponibilité est celle du joueur, qui s'en va
	renamePlayers(reflect.ValueOf(s).Elem(), map[PlayerID]PlayerID{ev.ID: fiche.ID})
	s.Players[fiche.ID] = &fiche
	s.Warnings = s.check()
	return nil
}

func (s *State) applySwap(ev Event) error {
	if ev.A == ev.B {
		return fmt.Errorf("échange d'un joueur avec lui-même")
	}
	for _, p := range []PlayerID{ev.A, ev.B} {
		if err := s.seatLocked(p); err != nil {
			return err
		}
	}
	// Les fiches restent à leurs joueurs ; les places — phases, rang d'inscription, tête de
	// série — s'échangent.
	éch := map[PlayerID]PlayerID{ev.A: ev.B, ev.B: ev.A}
	renamePlayers(reflect.ValueOf(&s.Phases).Elem(), éch)
	renamePlayers(reflect.ValueOf(&s.Order).Elem(), éch)
	renamePlayers(reflect.ValueOf(&s.Matches).Elem(), éch) // les matchs annulés suivent la place
	renamePlayers(reflect.ValueOf(&s.Infos).Elem(), éch)
	renamePlayers(reflect.ValueOf(&s.Final).Elem(), éch)
	a, b := s.Players[ev.A], s.Players[ev.B]
	a.Seed, b.Seed = b.Seed, a.Seed
	s.Warnings = s.check()
	return nil
}

var playerIDType = reflect.TypeOf(PlayerID(""))

// renamePlayers remplace, dans toute la valeur, chaque identifiant de joueur par son image —
// valeurs, éléments et clés de cartes. Seuls les champs exportés sont parcourus : ceux qui ne
// le sont pas ne portent aucun joueur au-delà d'une proposition.
func renamePlayers(v reflect.Value, img map[PlayerID]PlayerID) {
	switch v.Kind() {
	case reflect.String:
		if v.Type() == playerIDType && v.CanSet() {
			if n, ok := img[PlayerID(v.String())]; ok {
				v.SetString(string(n))
			}
		}
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			renamePlayers(v.Elem(), img)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				renamePlayers(v.Field(i), img)
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			renamePlayers(v.Index(i), img)
		}
	case reflect.Map:
		if v.IsNil() {
			return
		}
		type paire struct{ k, e reflect.Value }
		var paires []paire
		for it := v.MapRange(); it.Next(); {
			k := reflect.New(v.Type().Key()).Elem()
			k.Set(it.Key())
			e := reflect.New(v.Type().Elem()).Elem()
			e.Set(it.Value())
			renamePlayers(k, img)
			renamePlayers(e, img)
			paires = append(paires, paire{k, e})
		}
		v.Clear()
		for _, p := range paires {
			v.SetMapIndex(p.k, p.e)
		}
	}
}
