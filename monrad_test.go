package tournoi_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/kevung/nicomaque"
)

func monrad(rondes int) tournoi.Config {
	return tournoi.Config{Name: "Monrad", Phases: []tournoi.PhaseConfig{
		{Kind: tournoi.KindSwissRounds, Length: 7, Rounds: rondes, Pairing: tournoi.PairingMonrad}}}
}

// TestMonradInvariants : ceux du suisse classique, plus ce qui fait le Monrad — un match oppose
// deux joueurs de mêmes points, sauf flotteur (au plus un par groupe de points).
func TestMonradInvariants(t *testing.T) {
	effectifs, graines := []int{2, 3, 5, 8, 13, 32, 64, 100}, 8
	if testing.Short() {
		effectifs, graines = []int{2, 3, 5, 8, 13, 32}, 3
	}
	for _, P := range effectifs {
		for _, rondes := range []int{1, 4, 7} {
			for k := 0; k < graines; k++ {
				nom := fmt.Sprintf("P=%d rondes=%d graine=%d", P, rondes, k)
				r, b := jouerRondes(t, monrad(rondes), P, int64(k))
				if b.maxRonde != rondes || len(b.parRonde) != rondes {
					t.Fatalf("%s : %d rondes jouées, dernière %d", nom, len(b.parRonde), b.maxRonde)
				}
				for ronde, eng := range b.parRonde {
					if len(eng) != P {
						t.Fatalf("%s : ronde %d, %d joueurs engagés sur %d", nom, ronde, len(eng), P)
					}
					for p, n := range eng {
						if n != 1 {
							t.Fatalf("%s : ronde %d, %s engagé %d fois", nom, ronde, p, n)
						}
					}
					if b.byesRonde[ronde] != P%2 {
						t.Fatalf("%s : ronde %d, %d exemptions", nom, ronde, b.byesRonde[ronde])
					}
					// À la ronde r il y a au plus r groupes de points, donc au plus r − 1 flotteurs
					// tant que le champ est assez large pour qu'aucun groupe ne soit reporté entier.
					if P >= 8*rondes && b.croisés[ronde] > ronde-1 {
						t.Fatalf("%s : ronde %d, %d matchs entre points différents", nom, ronde, b.croisés[ronde])
					}
				}
				if P >= 8*rondes && b.revanches != 0 {
					t.Fatalf("%s : %d revanches", nom, b.revanches)
				}
				if len(r.State.Final) != P {
					t.Fatalf("%s : classement de %d joueurs", nom, len(r.State.Final))
				}
				st2, err := tournoi.Replay(r.Journal)
				if err != nil {
					t.Fatalf("%s : rejeu : %v", nom, err)
				}
				x, _ := json.Marshal(r.State.Final)
				y, _ := json.Marshal(st2.Final)
				if string(x) != string(y) || len(st2.Warnings) > 0 {
					t.Fatalf("%s : rejeu différent ou avertissements %v", nom, st2.Warnings)
				}
			}
		}
	}
}

// TestMonradTirage : même graine, mêmes rondes ; deux graines, des rondes différentes.
func TestMonradTirage(t *testing.T) {
	_, a := jouerRondes(t, monrad(6), 32, 1)
	_, b := jouerRondes(t, monrad(6), 32, 1)
	if strings.Join(a.plan, " ") != strings.Join(b.plan, " ") {
		t.Fatal("la même graine donne deux tournois différents")
	}
	// jouerRondes tire le champ selon la graine : on compare les rondes 1, qui ne dépendent que du tirage.
	_, c := jouerRondes(t, monrad(1), 32, 2)
	_, d := jouerRondes(t, monrad(1), 32, 3)
	if strings.Join(c.plan, " ") == strings.Join(d.plan, " ") {
		t.Fatal("deux graines donnent la même ronde 1")
	}
}

// TestMonradSimulation : 32 joueurs, 6 rondes — tout se joue, sans revanche, et la plupart des
// matchs opposent deux joueurs de mêmes points.
func TestMonradSimulation(t *testing.T) {
	croisés, total := 0, 0
	for k := 0; k < 20; k++ {
		r, b := jouerRondes(t, monrad(6), 32, int64(k))
		if r.NMatches != 6*16 {
			t.Fatalf("graine %d : %d matchs", k, r.NMatches)
		}
		if b.revanches != 0 {
			t.Fatalf("graine %d : %d revanches", k, b.revanches)
		}
		for _, n := range b.croisés {
			croisés += n
		}
		total += b.matchs
	}
	if croisés*5 > total {
		t.Errorf("%d matchs sur %d entre points différents", croisés, total)
	}
}

func TestMonradValidate(t *testing.T) {
	c := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 5, Pairing: "wins"}}}
	if err := c.Validate(); err == nil {
		t.Error("pairing wins accepté sur un suisse classique")
	}
	c = monrad(5)
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}
