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

// TestUnSeulQualifie : une phase d'un seul entrant ne bloque pas le tournoi — son entrant en
// est le vainqueur, que la configuration le dise (top:1, admis : un journal v0.8.0 peut le
// porter) ou que cela arrive en cours.
func TestUnSeulQualifie(t *testing.T) {
	for nom, cfg := range map[string]tournoi.Config{
		"top:1": {Phases: []tournoi.PhaseConfig{
			{Kind: tournoi.KindSwissRounds, Length: 5, Rounds: 3}, {Kind: tournoi.KindBracket, Length: 5, Entry: "top:1"}}},
		"un qualifié": {Phases: []tournoi.PhaseConfig{
			{Kind: tournoi.KindRoundRobin, Length: 5, Groups: 1, Qualifiers: 1}, {Kind: tournoi.KindBracket, Length: 5}}},
	} {
		t.Run(nom, func(t *testing.T) { unSeulQualifie(t, cfg) })
	}
}

func unSeulQualifie(t *testing.T, cfg tournoi.Config) {
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

// TestBarrageFinalRetraitNAvertitPas : le retrait d'un finaliste, barrage final tiré, ne change
// pas les premiers de poule — aucun résultat n'est corrigé, donc aucun avertissement, ni au
// retrait ni jusqu'à la clôture.
func TestBarrageFinalRetraitNAvertitPas(t *testing.T) {
	cfg := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindRoundRobin, Length: 5, Groups: 2, Qualifiers: 1, FinalBarrage: true}}}
	for graine := int64(0); graine < 4; graine++ {
		for i := 0; i < 2; i++ {
			d := piloter(t, cfg, graine, champClubs(8, 0))
			var final *tournoi.Section
			d.jouer(func() bool {
				for _, sec := range d.st.Phases[0].Sections {
					if sec.Name == "barrage:final" {
						final = sec
					}
				}
				return final != nil
			})
			if final == nil || len(final.Players) != 2 {
				t.Fatalf("graine %d : barrage final à deux attendu", graine)
			}
			d.must(tournoi.PlayerWithdrawnEvent(final.Players[i], d.now))
			if len(d.st.Warnings) != 0 {
				t.Fatalf("graine %d, retrait du finaliste %d : %v", graine, i, d.st.Warnings)
			}
			d.jouer(nil)
			if !d.st.Finished || len(d.st.Warnings) != 0 {
				t.Fatalf("graine %d, retrait du finaliste %d : fini %v, avertissements %v", graine, i, d.st.Finished, d.st.Warnings)
			}
			d.rejouable()
		}
		// Retrait d'un premier de poule AVANT le tirage : le suivant de sa poule est tiré à
		// bon droit, sans avertissement non plus.
		d := piloter(t, cfg, graine, champClubs(8, 0))
		tirage := func() bool {
			for _, a := range d.st.Propose() {
				if a.Kind == tournoi.ActDraw && a.Section == "barrage:final" {
					return true
				}
			}
			return false
		}
		d.jouer(tirage)
		if !tirage() {
			t.Fatalf("graine %d : tirage du barrage final attendu", graine)
		}
		d.must(tournoi.PlayerWithdrawnEvent(d.st.Ranking()[0].Player, d.now))
		d.jouer(nil)
		if !d.st.Finished || len(d.st.Warnings) != 0 {
			t.Fatalf("graine %d, retrait avant le tirage : fini %v, avertissements %v", graine, d.st.Finished, d.st.Warnings)
		}
		d.rejouable()
	}
}

// barrageFinal pilote deux poules de quatre à un qualifié jusqu'au tirage du barrage final
// PROPOSÉ, et rend le pilote avec les joueurs de chaque poule, du premier au dernier.
func barrageFinal(t *testing.T, graine int64) (*pilote, map[string][]tournoi.PlayerID) {
	t.Helper()
	cfg := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindRoundRobin, Length: 5, GroupSize: 4, Qualifiers: 1, FinalBarrage: true}}}
	d := piloter(t, cfg, graine, champClubs(8, 0))
	d.jouer(func() bool { return tirageFinalProposé(d) })
	if !tirageFinalProposé(d) {
		t.Fatalf("graine %d : tirage du barrage final attendu", graine)
	}
	poule := map[tournoi.PlayerID]string{}
	for _, m := range d.st.Matches {
		poule[m.A], poule[m.B] = m.Section, m.Section
	}
	poules := map[string][]tournoi.PlayerID{}
	for _, l := range d.st.Ranking() {
		poules[poule[l.Player]] = append(poules[poule[l.Player]], l.Player)
	}
	return d, poules
}

func tirageFinalProposé(d *pilote) bool {
	for _, a := range d.st.Propose() {
		if a.Kind == tournoi.ActDraw && a.Section == "barrage:final" {
			return true
		}
	}
	return false
}

func (d *pilote) finalTiré() *tournoi.Section {
	for _, sec := range d.st.Phases[0].Sections {
		if sec.Name == "barrage:final" {
			return sec
		}
	}
	return nil
}

func (d *pilote) barragePérimé() bool {
	for _, w := range d.st.Warnings {
		if w.Code == tournoi.WarnBarrageStale {
			return true
		}
	}
	return false
}

// corrige donne à gagnant le match de poule entre a et b.
func (d *pilote) corrige(a, b, gagnant tournoi.PlayerID) {
	d.t.Helper()
	for _, id := range d.st.MatchOrder {
		if m := d.st.Matches[id]; m.A == a && m.B == b || m.A == b && m.B == a {
			d.must(tournoi.CorrectionEvent(id, gagnant, m.Length, 0, d.now))
			return
		}
	}
	d.t.Fatalf("pas de match entre %s et %s", a, b)
}

// TestBarrageFinalEgaliteApresPromotion : le premier d'une poule se retire avant le tirage,
// son suivant est tiré ; des corrections mettent ensuite les trois restants de la poule à
// égalité. Le barrage final est périmé — le suivant promu n'excuse pas l'égalité née après lui.
func TestBarrageFinalEgaliteApresPromotion(t *testing.T) {
	d, poules := barrageFinal(t, 1)
	a := poules["poule:A"]
	d.must(tournoi.PlayerWithdrawnEvent(a[0], d.now))
	d.jouer(func() bool { return d.finalTiré() != nil })
	if d.finalTiré() == nil || d.barragePérimé() {
		t.Fatalf("barrage final attendu, sans avertissement : %v", d.st.Warnings)
	}
	x, z, dernier := a[1], a[2], a[3]
	if v := map[tournoi.PlayerID]int{}; true {
		for _, m := range d.st.Matches {
			if m.Section == "poule:A" {
				v[m.Winner]++
			}
		}
		if v[x] != 2 || v[z] != 1 || v[dernier] != 0 {
			t.Fatalf("poule A inattendue : %v", v)
		}
	}
	d.corrige(x, z, z)
	d.corrige(z, dernier, dernier)
	if !d.barragePérimé() {
		t.Fatalf("trois ex æquo en tête de poule A, un seul tiré : barrage_stale attendu, %v", d.st.Warnings)
	}
	d.rejouable()
}

// TestBarrageFinalRetraitsMelesNAvertitPas : chaque poule se juge seule. Un retrait avant le
// tirage dans l'une, un retrait après le tirage dans l'autre (ou dans la même) : aucun
// résultat n'est corrigé, aucun avertissement.
func TestBarrageFinalRetraitsMelesNAvertitPas(t *testing.T) {
	for graine := int64(0); graine < 4; graine++ {
		for _, mêmePoule := range []bool{false, true} {
			d, poules := barrageFinal(t, graine)
			a, b := poules["poule:A"], poules["poule:B"]
			d.must(tournoi.PlayerWithdrawnEvent(a[0], d.now))
			d.jouer(func() bool { return d.finalTiré() != nil })
			if d.finalTiré() == nil || len(d.st.Warnings) != 0 {
				t.Fatalf("graine %d : barrage final attendu, sans avertissement : %v", graine, d.st.Warnings)
			}
			après := b[0]
			if mêmePoule {
				après = a[1]
			}
			d.must(tournoi.PlayerWithdrawnEvent(après, d.now))
			if len(d.st.Warnings) != 0 {
				t.Fatalf("graine %d, même poule %v, au second retrait : %v", graine, mêmePoule, d.st.Warnings)
			}
			d.jouer(nil)
			if !d.st.Finished || len(d.st.Warnings) != 0 {
				t.Fatalf("graine %d, même poule %v : fini %v, avertissements %v", graine, mêmePoule, d.st.Finished, d.st.Warnings)
			}
			d.rejouable()
		}
	}
}

// TestBarrageDePouleCorrectionAvertit : un barrage de poule est tiré entre les ex æquo de la
// frontière ; une correction qui sort l'un d'eux de l'égalité le périme.
func TestBarrageDePouleCorrectionAvertit(t *testing.T) {
	cfg := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindRoundRobin, Length: 5, GroupSize: 4, Qualifiers: 2}}}
	vus := 0
	for graine := int64(0); graine < 12; graine++ {
		d := piloter(t, cfg, graine, champClubs(8, 0))
		var barrage *tournoi.Section
		rng := rand.New(rand.NewSource(graine))
		for pas := 0; pas < 200 && barrage == nil && !d.st.Finished; pas++ {
			for _, a := range d.st.Propose() {
				if a.Kind != tournoi.ActWait && a.Kind != tournoi.ActFinish {
					d.confirm(a)
					break
				}
			}
			for _, id := range d.st.MatchOrder {
				if m := d.st.Matches[id]; m.Status == tournoi.Running {
					d.must(tournoi.ResultEvent(id, []tournoi.PlayerID{m.A, m.B}[rng.Intn(2)], m.Length, 0, d.now))
				}
			}
			for _, sec := range d.st.Phases[0].Sections {
				if strings.HasPrefix(sec.Name, "barrage:") {
					barrage = sec
				}
			}
		}
		if barrage == nil {
			continue
		}
		vus++
		if d.barragePérimé() {
			t.Fatalf("graine %d : barrage tout juste tiré, déjà périmé : %v", graine, d.st.Warnings)
		}
		exAequo := map[tournoi.PlayerID]bool{}
		for _, p := range barrage.Players {
			exAequo[p] = true
		}
		// Un match de poule entre un ex æquo et un joueur de sa poule hors de l'égalité.
		corrigé := false
		for _, id := range d.st.MatchOrder {
			m := d.st.Matches[id]
			if "barrage:"+strings.TrimPrefix(m.Section, "poule:") != barrage.Name || exAequo[m.A] == exAequo[m.B] {
				continue
			}
			perdant := m.A
			if perdant == m.Winner {
				perdant = m.B
			}
			d.must(tournoi.CorrectionEvent(id, perdant, m.Length, 0, d.now))
			corrigé = true
			break
		}
		if !corrigé || !d.barragePérimé() {
			t.Fatalf("graine %d : correction %v, barrage_stale attendu : %v", graine, corrigé, d.st.Warnings)
		}
		d.rejouable()
	}
	if vus == 0 {
		t.Fatal("aucun barrage de poule en douze graines")
	}
}
