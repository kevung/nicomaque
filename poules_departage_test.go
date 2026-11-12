package tournoi_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/sim"
)

// barrages : le nombre de sections de barrage tirées dans le journal.
func barrages(j tournoi.Journal) int {
	n := 0
	for _, ev := range j {
		if ev.Kind == tournoi.EvDraw && strings.HasPrefix(ev.Section, "barrage:") {
			n++
		}
	}
	return n
}

// jouerPoules : un tournoi de poules en dernière phase, rejoué, avec ses invariants — un seul
// premier, classement complet, rejeu identique et sans avertissement.
func jouerPoules(t *testing.T, nom string, ph tournoi.PhaseConfig, P int, k int64) sim.Result {
	t.Helper()
	joueurs := sim.Champ(P, 6, 2, 2, 10, rand.New(rand.NewSource(int64(P)*100+k)))
	r := sim.Run(tournoi.Config{Phases: []tournoi.PhaseConfig{ph}}, joueurs, sim.Options{Seed: k})
	if r.Err != nil {
		t.Fatalf("%s : %v", nom, r.Err)
	}
	if len(r.State.Final) != P || countRank1(r.State.Final) != 1 {
		t.Fatalf("%s : classement de %d joueurs, %d premiers", nom, len(r.State.Final), countRank1(r.State.Final))
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
	return r
}

// TestPoulesDepartageInvariants : avec tiebreak, la poule est classée par le départage — un
// barrage ne reste que pour une égalité complète —, et une dernière phase de poules a toujours
// un seul vainqueur, qu'il y ait une poule ou plusieurs, un départage ou non.
func TestPoulesDepartageInvariants(t *testing.T) {
	effectifs, graines := []int{2, 3, 5, 8, 13, 32}, 4
	if testing.Short() {
		effectifs, graines = []int{2, 5, 8}, 2
	}
	for _, P := range effectifs {
		for _, ng := range []int{1, 2, 3} {
			for _, q := range []int{1, 2} {
				for k := 0; k < graines; k++ {
					nom := fmt.Sprintf("P=%d poules=%d qualifiés=%d graine=%d", P, ng, q, k)
					ph := tournoi.PhaseConfig{Kind: tournoi.KindRoundRobin, Length: 5, Groups: ng, Qualifiers: q}
					jouerPoules(t, nom+" sans départage", ph, P, int64(k))
					ph.Tiebreak = tousCritères
					jouerPoules(t, nom+" avec départage", ph, P, int64(k))
				}
			}
		}
	}
}

// TestPoulesDepartageSimulation : une poule unique de 8, 40 tirages. Sans départage, le premier
// sort souvent d'un barrage ; avec, presque jamais, et le classement n'a presque plus d'ex æquo.
func TestPoulesDepartageSimulation(t *testing.T) {
	if testing.Short() {
		t.Skip("40 poules de 8, deux fois")
	}
	sans, avec, exAequo := 0, 0, 0
	for k := int64(0); k < 40; k++ {
		ph := tournoi.PhaseConfig{Kind: tournoi.KindRoundRobin, Length: 5, Groups: 1, Qualifiers: 1}
		sans += barrages(jouerPoules(t, "sans", ph, 8, k).Journal)
		ph.Tiebreak = tousCritères
		r := jouerPoules(t, "avec", ph, 8, k)
		avec += barrages(r.Journal)
		rangs := map[int]bool{}
		for _, st := range r.State.Final {
			rangs[st.Rank] = true
		}
		exAequo += 8 - len(rangs)
	}
	if sans == 0 {
		t.Fatal("aucun barrage sans départage : le cas n'est pas couvert")
	}
	if avec*4 > sans {
		t.Errorf("%d barrages avec départage, %d sans", avec, sans)
	}
	if exAequo > 8 {
		t.Errorf("%d ex æquo restants sur 40 poules de 8", exAequo)
	}
}

// TestPoulesFinales : plusieurs poules en dernière phase. Les premiers de poule se départagent
// sur le tapis, par le barrage final, et lui seul désigne le vainqueur.
func TestPoulesFinales(t *testing.T) {
	for k := int64(0); k < 4; k++ {
		r := jouerPoules(t, fmt.Sprintf("graine %d", k),
			tournoi.PhaseConfig{Kind: tournoi.KindRoundRobin, Length: 5, Groups: 4, Qualifiers: 1}, 16, k)
		final := false
		for _, ev := range r.Journal {
			if ev.Kind == tournoi.EvDraw && ev.Section == "barrage:final" {
				final = true
				if len(ev.Draw.Groups) != 1 || len(ev.Draw.Groups[0]) != 4 {
					t.Fatalf("graine %d : barrage final entre %v", k, ev.Draw.Groups)
				}
			}
		}
		if !final {
			t.Fatalf("graine %d : pas de barrage final entre les quatre premiers de poule", k)
		}
	}
}

func TestPoulesDepartageValidate(t *testing.T) {
	c := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindRoundRobin, Length: 5, Tiebreak: []string{"elo"}}}}
	if err := c.Validate(); err == nil {
		t.Error("critère inconnu accepté sur des poules")
	}
	c = tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindRoundRobin, Length: 5, Tiebreak: tousCritères}}}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}
