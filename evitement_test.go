package tournoi_test

import (
	"math/rand"
	"testing"

	"github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/rules"
	"github.com/kevung/nicomaque/sim"
)

func mêmePaire(pr rules.Pairing, a, b rules.Ref) bool {
	return (pr.A == a && pr.B == b) || (pr.A == b && pr.B == a)
}

// TestEvitementPays : avoid_countries évite les rencontres de même pays comme avoid_clubs celles
// de même club ; avoid_except sort un club (le club hôte) de la comparaison ; avoid_rounds borne
// la préférence aux premières rondes.
func TestEvitementPays(t *testing.T) {
	e := func(id, club, pays string) rules.Entrant {
		return rules.Entrant{Player: rules.Ref(id), Lives: 99, Club: club, Country: pays}
	}
	phase := func(r rules.PhaseRules, round int) rules.Phase {
		r.Rounds, r.Length, r.Mode = 5, 7, rules.ModeRounds
		return rules.Phase{Format: rules.FormatSwissRounds, Length: 7, Rules: r, Round: round,
			Entrants: []rules.Entrant{e("a", "X", "fr"), e("b", "X", "FR "), e("c", "Y", "be"), e("d", "Y", "be")},
			Free:     []rules.Ref{"a", "b", "c", "d"}}
	}
	croisées := func(ph rules.Phase) (n int) {
		for g := int64(0); g < 20; g++ {
			for _, pr := range rules.Pair(ph, rules.Seed{Base: g}) {
				if mêmePaire(pr, "a", "b") || mêmePaire(pr, "c", "d") {
					n++
				}
			}
		}
		return n
	}
	if n := croisées(phase(rules.PhaseRules{}, 0)); n == 0 {
		t.Fatal("sans option, aucune rencontre de même pays en 20 tirages : le cas n'est pas couvert")
	}
	if n := croisées(phase(rules.PhaseRules{AvoidCountries: true}, 0)); n != 0 {
		t.Errorf("avoid_countries : %d rencontres de même pays", n)
	}
	// Le club hôte ne compte pas : X et Y exclus, plus rien n'est évité.
	if n := croisées(phase(rules.PhaseRules{AvoidClubs: true, AvoidExcept: []string{"X", "Y"}}, 0)); n == 0 {
		t.Error("avoid_except : les clubs exclus sont encore évités")
	}
	if n := croisées(phase(rules.PhaseRules{AvoidClubs: true, AvoidExcept: []string{"X"}}, 0)); n != 0 {
		t.Errorf("avoid_except : %d rencontres du club non exclu", n)
	}
	// avoid_rounds: 2 — la ronde 2 évite encore, la ronde 3 non.
	if n := croisées(phase(rules.PhaseRules{AvoidCountries: true, AvoidRounds: 2}, 1)); n != 0 {
		t.Errorf("avoid_rounds 2, ronde 2 : %d rencontres de même pays", n)
	}
	if n := croisées(phase(rules.PhaseRules{AvoidCountries: true, AvoidRounds: 2}, 2)); n == 0 {
		t.Error("avoid_rounds 2, ronde 3 : la préférence joue encore")
	}
}

// TestEvitementSimulation : 32 joueurs de cinq clubs, 5 rondes. Limitée aux deux premières
// rondes, la préférence y supprime les rencontres de club sans créer de revanche, et laisse les
// rondes suivantes s'apparier au classement.
func TestEvitementSimulation(t *testing.T) {
	compte := func(évite bool) (tôt, revanches int) {
		graines := int64(10)
		if testing.Short() {
			graines = 3
		}
		for k := int64(0); k < graines; k++ {
			cfg := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 5,
				AvoidClubs: évite, AvoidRounds: 2}}}
			if !évite {
				cfg.Phases[0].AvoidRounds = 0
			}
			vus := map[[2]tournoi.PlayerID]bool{}
			hook := func(s *tournoi.State, ev tournoi.Event) error {
				if ev.Kind != tournoi.EvMatchStarted {
					return nil
				}
				if ev.Round <= 2 && s.Players[ev.A].Club == s.Players[ev.B].Club {
					tôt++
				}
				p := [2]tournoi.PlayerID{min(ev.A, ev.B), max(ev.A, ev.B)}
				if vus[p] {
					revanches++
				}
				vus[p] = true
				return nil
			}
			r := sim.Run(cfg, sim.Champ(32, 6, 2, 2, 10, rand.New(rand.NewSource(k))), sim.Options{Seed: k, Hook: hook})
			if r.Err != nil || r.NMatches != 5*16 || len(r.State.Final) != 32 {
				t.Fatalf("graine %d : %v, %d matchs", k, r.Err, r.NMatches)
			}
		}
		return tôt, revanches
	}
	sans, _ := compte(false)
	avec, revanches := compte(true)
	if sans == 0 {
		t.Fatal("aucune rencontre de club sans l'option : le cas n'est pas couvert")
	}
	if avec != 0 || revanches != 0 {
		t.Errorf("avec l'option : %d rencontres de club aux rondes 1 et 2 (%d sans), %d revanches", avec, sans, revanches)
	}
}

func TestEvitementValidate(t *testing.T) {
	refus := map[string]tournoi.PhaseConfig{
		"rondes négatives":      {Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 5, AvoidClubs: true, AvoidRounds: -1},
		"rondes sans option":    {Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 5, AvoidRounds: 2},
		"rondes hors classique": {Kind: tournoi.KindSwissLives, Length: 7, AvoidClubs: true, AvoidRounds: 2},
		"exclus sans clubs":     {Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 5, AvoidCountries: true, AvoidExcept: []string{"X"}},
		"pays sur un tableau":   {Kind: tournoi.KindBracket, Length: 7, AvoidCountries: true},
	}
	for nom, p := range refus {
		c := tournoi.Config{Phases: []tournoi.PhaseConfig{p}}
		if err := c.Validate(); err == nil {
			t.Errorf("%s : configuration acceptée", nom)
		}
	}
	c := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 5,
		AvoidClubs: true, AvoidCountries: true, AvoidRounds: 2, AvoidExcept: []string{"Hôte"}}}}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}
