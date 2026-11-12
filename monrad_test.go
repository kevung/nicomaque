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
// deux joueurs de mêmes points, sauf flotteur (au plus un par groupe de points). Sans
// monrad_widen, le dernier groupe peut jouer une revanche ; avec, un champ large n'en joue aucune.
func TestMonradInvariants(t *testing.T) {
	t.Parallel() // indépendant des autres : la boucle -short garde de la marge
	for _, élargi := range []bool{false, true} {
		monradInvariants(t, élargi)
	}
}

func monradInvariants(t *testing.T, élargi bool) {
	effectifs, graines := []int{2, 3, 5, 8, 13, 32, 64, 100}, 8
	if testing.Short() {
		effectifs, graines = []int{2, 3, 5, 8, 13, 32}, 3
	}
	for _, P := range effectifs {
		for _, rondes := range []int{1, 4, 7} {
			for k := 0; k < graines; k++ {
				nom := fmt.Sprintf("P=%d rondes=%d graine=%d élargi=%v", P, rondes, k, élargi)
				cfg := monrad(rondes)
				cfg.Phases[0].MonradWiden = élargi
				r, b := jouerRondes(t, cfg, P, int64(k))
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
				if élargi && P >= 8*rondes && b.revanches != 0 {
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

// TestMonradSimulation : 32 joueurs, 6 rondes — tout se joue et la plupart des matchs opposent
// deux joueurs de mêmes points. Les seules revanches sont celles du dernier groupe : rares sans
// monrad_widen, absentes avec.
func TestMonradSimulation(t *testing.T) {
	t.Parallel() // indépendant des autres : la boucle -short garde de la marge
	for _, élargi := range []bool{false, true} {
		croisés, revanches, total := 0, 0, 0
		graines := 20
		if testing.Short() {
			graines = 6
		}
		for k := 0; k < graines; k++ {
			c := monrad(6)
			c.Phases[0].MonradWiden = élargi
			r, b := jouerRondes(t, c, 32, int64(k))
			if r.NMatches != 6*16 {
				t.Fatalf("graine %d : %d matchs", k, r.NMatches)
			}
			for _, n := range b.croisés {
				croisés += n
			}
			revanches += b.revanches
			total += b.matchs
		}
		if croisés*5 > total {
			t.Errorf("élargi=%v : %d matchs sur %d entre points différents", élargi, croisés, total)
		}
		if élargi && revanches != 0 {
			t.Errorf("monrad_widen : %d revanches", revanches)
		}
		if revanches*50 > total {
			t.Errorf("élargi=%v : %d revanches sur %d matchs", élargi, revanches, total)
		}
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

// TestMonradElargi : monrad_widen évite des revanches que le Monrad sans l'option joue dans son
// dernier groupe, et n'en ajoute jamais sur l'ensemble des tirages.
func TestMonradElargi(t *testing.T) {
	strict, large := 0, 0
	for k := 0; k < 30; k++ {
		_, a := jouerRondes(t, monrad(6), 8, int64(k))
		c := monrad(6)
		c.Phases[0].MonradWiden = true
		_, b := jouerRondes(t, c, 8, int64(k))
		strict, large = strict+a.revanches, large+b.revanches
	}
	if strict == 0 {
		t.Fatal("le Monrad sans monrad_widen ne joue aucune revanche : le cas n'est pas couvert")
	}
	if large >= strict {
		t.Errorf("monrad_widen : %d revanches, %d sans l'option", large, strict)
	}
	c := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 5, MonradWiden: true}}}
	if err := c.Validate(); err == nil {
		t.Error("monrad_widen accepté sans pairing monrad")
	}
}
