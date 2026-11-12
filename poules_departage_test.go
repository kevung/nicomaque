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
// barrage ne reste que pour une égalité complète —, et une dernière phase de poules à barrage final
// (final_barrage) a toujours un seul vainqueur, qu'il y ait une poule ou plusieurs, un départage ou non.
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
					ph := tournoi.PhaseConfig{Kind: tournoi.KindRoundRobin, Length: 5, Groups: ng, Qualifiers: q, FinalBarrage: true}
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
		ph := tournoi.PhaseConfig{Kind: tournoi.KindRoundRobin, Length: 5, Groups: 1, Qualifiers: 1, FinalBarrage: true}
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
			tournoi.PhaseConfig{Kind: tournoi.KindRoundRobin, Length: 5, Groups: 4, Qualifiers: 1, FinalBarrage: true}, 16, k)
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

// TestBarrageFinalCorrectionAvertit : une correction de poule passée après le tirage du barrage
// final change les premiers de poule ; le barrage reste celui du tirage, et le TD en est averti.
func TestBarrageFinalCorrectionAvertit(t *testing.T) {
	cfg := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindRoundRobin, Length: 5, GroupSize: 4, Qualifiers: 1, FinalBarrage: true}}}
	d := piloter(t, cfg, 1, champClubs(8, 0))
	final := func() *tournoi.Section {
		for _, sec := range d.st.Phases[0].Sections {
			if sec.Name == "barrage:final" {
				return sec
			}
		}
		return nil
	}
	périmé := func() bool {
		for _, w := range d.st.Warnings {
			if w.Code == tournoi.WarnBarrageStale && w.Section == "barrage:final" {
				return true
			}
		}
		return false
	}
	d.jouer(func() bool { return final() != nil })
	if final() == nil || périmé() {
		t.Fatalf("barrage final attendu, sans avertissement : %v", d.st.Warnings)
	}
	x := final().Players[0]
	for _, id := range d.st.MatchOrder {
		if m := d.st.Matches[id]; m.Winner == x {
			o := m.A
			if o == x {
				o = m.B
			}
			d.must(tournoi.CorrectionEvent(id, o, m.Length, 0, d.now))
		}
	}
	if !périmé() {
		t.Fatalf("pas d'avertissement barrage_stale après la correction : %v", d.st.Warnings)
	}
	d.rejouable()
}

// TestUnSeulQualifie : une phase d'un seul entrant ne bloque pas le tournoi — elle est refusée
// à la validation quand la configuration le dit (top:1), et son entrant est le vainqueur quand
// cela arrive en cours.
func TestUnSeulQualifie(t *testing.T) {
	refusée := tournoi.Config{Phases: []tournoi.PhaseConfig{
		{Kind: tournoi.KindSwissRounds, Length: 5, Rounds: 3}, {Kind: tournoi.KindBracket, Length: 5, Entry: "top:1"}}}
	if refusée.Validate() == nil {
		t.Error("entry top:1 accepté")
	}
	cfg := tournoi.Config{Phases: []tournoi.PhaseConfig{
		{Kind: tournoi.KindRoundRobin, Length: 5, Groups: 1, Qualifiers: 1}, {Kind: tournoi.KindBracket, Length: 5}}}
	d := piloter(t, cfg, 2, champClubs(5, 0))
	d.jouer(nil)
	if !d.st.Finished {
		t.Fatalf("tournoi bloqué : %v", d.st.Propose())
	}
	if n := len(d.st.Final); n != 5 || d.st.Final[0].Rank != 1 || d.st.Final[1].Rank == 1 {
		t.Fatalf("classement final : %v", d.st.Final)
	}
	if s := d.st.StatusOf(d.st.Final[0].Player); s.Kind != tournoi.StatusWinner {
		t.Errorf("statut du vainqueur : %+v", s)
	}
	d.rejouable()
}
