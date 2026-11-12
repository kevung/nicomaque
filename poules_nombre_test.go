package tournoi_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/sim"
)

// tiragePoules : les groupes que le moteur propose pour ces joueurs.
func tiragePoules(t *testing.T, ph tournoi.PhaseConfig, joueurs []tournoi.Player) [][]tournoi.PlayerID {
	t.Helper()
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	st, _, err := tournoi.New(tournoi.Config{Phases: []tournoi.PhaseConfig{ph}}, 2, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range joueurs {
		if err := st.Apply(tournoi.PlayerAddedEvent(p, now)); err != nil {
			t.Fatal(err)
		}
	}
	acts := st.Propose()
	if len(acts) != 1 || acts[0].Kind != tournoi.ActDraw {
		t.Fatalf("propositions : %v", acts)
	}
	return acts[0].Draw.Groups
}

// TestPoulesSerpentin : avec des têtes de série, les poules se composent en serpentin.
func TestPoulesSerpentin(t *testing.T) {
	got := tiragePoules(t, tournoi.PhaseConfig{Kind: tournoi.KindRoundRobin, Length: 5, Groups: 2, Seeding: tournoi.SeedingManual}, joueursNumérotés(8, nil))
	want := [][]int{{1, 4, 5, 8}, {2, 3, 6, 7}}
	for g, têtes := range want {
		for i, n := range têtes {
			if len(got) != 2 || len(got[g]) != 4 || got[g][i] != j(n) {
				t.Fatalf("2 poules de 8 : %v, attendu %v", got, want)
			}
		}
	}
	got = tiragePoules(t, tournoi.PhaseConfig{Kind: tournoi.KindRoundRobin, Length: 5, Groups: 3, Seeding: tournoi.SeedingManual}, joueursNumérotés(7, nil))
	// 1 2 3 / 6 5 4 / 7 : A = 1, 6, 7 ; B = 2, 5 ; C = 3, 4
	if len(got) != 3 || fmt.Sprint(got[0]) != "[j01 j06 j07]" || fmt.Sprint(got[1]) != "[j02 j05]" || fmt.Sprint(got[2]) != "[j03 j04]" {
		t.Fatalf("3 poules de 7 : %v", got)
	}
}

// TestPoulesNombre : groups fixe le nombre de poules — tailles égales à un près, jamais plus
// d'une poule pour deux joueurs —, chaque paire d'une poule se rencontre exactement une fois, et
// le tournoi va au bout sur effectifs × nombres de poules × graines.
func TestPoulesNombre(t *testing.T) {
	effectifs, graines := []int{2, 3, 5, 8, 13, 32}, 4
	if testing.Short() {
		effectifs, graines = []int{2, 3, 5, 8, 13}, 2
	}
	for _, P := range effectifs {
		for _, ng := range []int{1, 2, 3, 5} {
			for k := 0; k < graines; k++ {
				nom := fmt.Sprintf("P=%d poules=%d graine=%d", P, ng, k)
				cfg := tournoi.Config{Phases: []tournoi.PhaseConfig{
					{Kind: tournoi.KindRoundRobin, Length: 5, Groups: ng, Qualifiers: 1},
					{Kind: tournoi.KindBracket, Length: 7}}}
				attendu := max(min(ng, P/2), 1)
				if attendu == 1 {
					// Une seule poule, un seul qualifié : c'est le tournoi entier. (Un tableau d'un
					// seul entrant ne se tire pas — limite du moteur antérieure à groups.)
					cfg.Phases = cfg.Phases[:1]
				}
				joueurs := sim.Champ(P, 6, 2, 2, 10, rand.New(rand.NewSource(int64(P*100+k))))
				r := sim.Run(cfg, joueurs, sim.Options{Seed: int64(k)})
				if r.Err != nil {
					t.Fatalf("%s : %v", nom, r.Err)
				}
				var groupes [][]tournoi.PlayerID
				for _, ev := range r.Journal {
					if ev.Kind == tournoi.EvDraw && ev.Phase == 0 && ev.Section == "" {
						groupes = ev.Draw.Groups
					}
				}
				if len(groupes) != attendu {
					t.Fatalf("%s : %d poules, %d attendues", nom, len(groupes), attendu)
				}
				lo, hi := 1<<30, 0
				poule := map[tournoi.PlayerID]int{}
				for g, grp := range groupes {
					lo, hi = min(lo, len(grp)), max(hi, len(grp))
					for _, p := range grp {
						poule[p] = g
					}
				}
				if hi-lo > 1 || len(poule) != P {
					t.Fatalf("%s : tailles de %d à %d, %d joueurs placés", nom, lo, hi, len(poule))
				}
				vus := map[[2]tournoi.PlayerID]int{}
				for _, id := range r.State.MatchOrder {
					m := r.State.Matches[id]
					if m.Phase != 0 || len(m.Section) < 6 || m.Section[:6] != "poule:" || m.Status == tournoi.Cancelled {
						continue
					}
					if poule[m.A] != poule[m.B] {
						t.Fatalf("%s : %s et %s ne sont pas de la même poule", nom, m.A, m.B)
					}
					vus[[2]tournoi.PlayerID{min(m.A, m.B), max(m.A, m.B)}]++
				}
				paires := 0
				for _, grp := range groupes {
					paires += len(grp) * (len(grp) - 1) / 2
				}
				if len(vus) != paires {
					t.Fatalf("%s : %d paires jouées, %d attendues", nom, len(vus), paires)
				}
				for pr, n := range vus {
					if n != 1 {
						t.Fatalf("%s : %v jouée %d fois", nom, pr, n)
					}
				}
				if countRank1(r.State.Final) != 1 || len(r.State.Final) != P {
					t.Fatalf("%s : classement %d joueurs, %d premiers", nom, len(r.State.Final), countRank1(r.State.Final))
				}
			}
		}
	}
}

func TestPoulesNombreValidate(t *testing.T) {
	refus := map[string]tournoi.PhaseConfig{
		"nombre et taille": {Kind: tournoi.KindRoundRobin, Length: 5, Groups: 2, GroupSize: 4},
		"négatif":          {Kind: tournoi.KindRoundRobin, Length: 5, Groups: -1},
		"sur un tableau":   {Kind: tournoi.KindBracket, Length: 5, Groups: 2},
		"seeding inconnu":  {Kind: tournoi.KindRoundRobin, Length: 5, Seeding: "elo"},
	}
	for nom, p := range refus {
		c := tournoi.Config{Phases: []tournoi.PhaseConfig{p}}
		if err := c.Validate(); err == nil {
			t.Errorf("%s : configuration acceptée", nom)
		}
	}
	c := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindRoundRobin, Length: 5, Groups: 2}}}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err != nil { // une configuration validée se revalide
		t.Fatalf("seconde validation : %v", err)
	}
}
