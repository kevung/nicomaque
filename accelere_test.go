package tournoi_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/rules"
)

func accéléré(rondes int) tournoi.Config {
	return tournoi.Config{Name: "Suisse accéléré", Phases: []tournoi.PhaseConfig{
		{Kind: tournoi.KindSwissRounds, Length: 7, Rounds: rondes, Seeding: tournoi.SeedingManual, Accelerated: true}}}
}

// rondeAccélérée : un suisse accéléré de n joueurs j01…jn (têtes 1…n), avec de quoi jouer une ronde.
func débutAccéléré(t *testing.T, n, rondes int) (*tournoi.State, func() (map[tournoi.PlayerID]tournoi.PlayerID, tournoi.PlayerID)) {
	t.Helper()
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	st, _, err := tournoi.New(accéléré(rondes), 4, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range joueursNumérotés(n, nil) {
		if err := st.Apply(tournoi.PlayerAddedEvent(p, now)); err != nil {
			t.Fatal(err)
		}
	}
	// jouer : lance la ronde proposée, la meilleure tête gagne chaque match
	jouer := func() (map[tournoi.PlayerID]tournoi.PlayerID, tournoi.PlayerID) {
		t.Helper()
		contre, bye := map[tournoi.PlayerID]tournoi.PlayerID{}, tournoi.PlayerID("")
		for _, a := range st.Propose() {
			e, err := st.EventFromAction(a, now)
			if err != nil {
				t.Fatal(err)
			}
			if err := st.Apply(e); err != nil {
				t.Fatal(err)
			}
			if a.Kind == tournoi.ActBye {
				bye = a.A
				continue
			}
			contre[a.A], contre[a.B] = a.B, a.A
			if err := st.Apply(tournoi.ResultEvent(e.MatchID, min(a.A, a.B), 7, 3, now)); err != nil {
				t.Fatal(err)
			}
		}
		return contre, bye
	}
	return st, jouer
}

func j(i int) tournoi.PlayerID { return tournoi.PlayerID(fmt.Sprintf("j%02d", i)) }

// TestAccelereRonde1 : chaque moitié est pliée séparément ; moitié haute impaire, sa dernière tête
// descend ; moitié basse impaire, sa dernière tête est exemptée.
func TestAccelereRonde1(t *testing.T) {
	_, jouer := débutAccéléré(t, 8, 3)
	contre, bye := jouer()
	for _, pr := range [][2]int{{1, 3}, {2, 4}, {5, 7}, {6, 8}} {
		if contre[j(pr[0])] != j(pr[1]) {
			t.Errorf("8 joueurs : %s contre %s, attendu %s", j(pr[0]), contre[j(pr[0])], j(pr[1]))
		}
	}
	if bye != "" {
		t.Errorf("8 joueurs : %s exempté", bye)
	}
	// 6 joueurs : moitié haute de 3, impaire — la tête 3 descend : 1-2, puis 3-5, 4-6
	_, jouer = débutAccéléré(t, 6, 3)
	contre, bye = jouer()
	if contre[j(1)] != j(2) || contre[j(3)] != j(5) || contre[j(4)] != j(6) || bye != "" {
		t.Errorf("6 joueurs : %v, exempté %q", contre, bye)
	}
	// 7 joueurs : moitié haute 1-3, la tête 3 descend ; moitié basse 3,4,5,6,7 impaire : 7 exemptée
	_, jouer = débutAccéléré(t, 7, 3)
	contre, bye = jouer()
	if contre[j(1)] != j(2) || contre[j(3)] != j(5) || contre[j(4)] != j(6) || bye != j(7) {
		t.Errorf("7 joueurs : %v, exempté %q", contre, bye)
	}
}

// TestAccelerePointVirtuel : à la ronde 2 le point virtuel compte dans l'ordre d'appariement —
// chacun rencontre un joueur de même total (points + point virtuel) : les vainqueurs du haut
// entre eux, et un vainqueur du bas (1) ne rencontre ni un vainqueur du haut (2) ni un perdant du
// bas (0). À la ronde 3 il ne joue plus, et il n'apparaît dans aucun classement.
func TestAccelerePointVirtuel(t *testing.T) {
	st, jouer := débutAccéléré(t, 16, 4)
	jouer() // ronde 1 : la meilleure tête gagne — vainqueurs 1-4 (haut) et 9-12 (bas)
	haut := map[tournoi.PlayerID]bool{}
	for _, e := range st.RulesPhase(0).Entrants {
		if e.TopHalf {
			haut[tournoi.PlayerID(e.Player)] = true
		}
	}
	for i := 1; i <= 16; i++ {
		if haut[j(i)] != (i <= 8) {
			t.Fatalf("moitié haute : %v", haut)
		}
	}
	total := map[tournoi.PlayerID]int{}
	for i := 1; i <= 16; i++ {
		total[j(i)] = st.Phases[0].Wins[j(i)]
		if haut[j(i)] {
			total[j(i)]++
		}
	}
	ronde2, _ := jouer()
	if len(ronde2) != 16 {
		t.Fatalf("ronde 2 : %v", ronde2)
	}
	for a, b := range ronde2 {
		if total[a] != total[b] {
			t.Errorf("ronde 2 : %s (total %d) contre %s (total %d)", a, total[a], b, total[b])
		}
	}
	for i := 1; i <= 4; i++ {
		if adv := ronde2[j(i)]; adv > j(4) {
			t.Errorf("ronde 2 : %s (vainqueur du haut) contre %s", j(i), adv)
		}
	}
	// le classement ne connaît que les points réels
	ph := st.Phases[0]
	rk := st.Ranking()
	for i := 1; i < len(rk); i++ {
		pa, pb := ph.Wins[rk[i-1].Player]+ph.Byes[rk[i-1].Player], ph.Wins[rk[i].Player]+ph.Byes[rk[i].Player]
		if pa < pb || (pa == pb) != (rk[i-1].Rank == rk[i].Rank) {
			t.Fatalf("le point virtuel a fui dans le classement : %v", rk)
		}
	}
	// ronde 3 : plus de point virtuel — chacun contre un joueur de mêmes points réels
	ronde3 := map[tournoi.PlayerID]tournoi.PlayerID{}
	for _, a := range st.Propose() {
		ronde3[a.A] = a.B
	}
	for a, b := range ronde3 {
		if pa, pb := ph.Wins[a], ph.Wins[b]; pa != pb {
			t.Errorf("ronde 3 : %s (%d pts) contre %s (%d pts)", a, pa, b, pb)
		}
	}
}

// TestAccelereMoitieFigee : un retrait après la ronde 1 ne fait franchir la ligne à personne, et
// un retardataire reste en bas.
func TestAccelereMoitieFigee(t *testing.T) {
	st, jouer := débutAccéléré(t, 8, 4)
	jouer()
	now := st.Last
	if err := st.Apply(tournoi.PlayerWithdrawnEvent(j(1), now)); err != nil {
		t.Fatal(err)
	}
	if err := st.Apply(tournoi.PlayerAddedEvent(tournoi.Player{ID: "tard", Name: "Tard", Seed: 1}, now)); err != nil {
		t.Fatal(err)
	}
	for _, e := range st.RulesPhase(0).Entrants {
		want := e.Player == "j01" || e.Player == "j02" || e.Player == "j03" || e.Player == "j04"
		if e.TopHalf != want {
			t.Errorf("%s : moitié haute = %v", e.Player, e.TopHalf)
		}
	}
	if got := rules.TopHalf(*st.RulesPhase(0), []rules.Ref{"j01", "j02", "j03", "j04", "j05"}); len(got) != 2 {
		t.Errorf("effectif impair : la tête du milieu reste en bas, moitié %v", got)
	}
}

// TestAccelereInvariants : ceux du suisse classique, sur la matrice, et la simulation complète de
// 32 × 6 et 31 × 6.
func TestAccelereInvariants(t *testing.T) {
	t.Parallel() // indépendant des autres : la boucle -short garde de la marge
	cfg := func(rondes int) tournoi.Config {
		c := accéléré(rondes)
		c.Phases[0].Seeding = tournoi.SeedingRating // sim.Champ ne numérote personne
		return c
	}
	effectifs, graines := []int{2, 3, 5, 8, 13, 31, 32, 64, 100}, 8
	if testing.Short() {
		effectifs, graines = []int{2, 3, 5, 8, 13, 31, 32}, 3
	}
	for _, P := range effectifs {
		for _, rondes := range []int{1, 4, 6} {
			for k := 0; k < graines; k++ {
				nom := fmt.Sprintf("P=%d rondes=%d graine=%d", P, rondes, k)
				r, b := jouerRondes(t, cfg(rondes), P, int64(k))
				if b.maxRonde != rondes || len(b.parRonde) != rondes {
					t.Fatalf("%s : %d rondes jouées, dernière %d", nom, len(b.parRonde), b.maxRonde)
				}
				for ronde, eng := range b.parRonde {
					if len(eng) != P || b.byesRonde[ronde] != P%2 {
						t.Fatalf("%s : ronde %d, %d engagés, %d exemptions", nom, ronde, len(eng), b.byesRonde[ronde])
					}
				}
				if P >= 2*rondes+2 && b.revanches != 0 {
					t.Fatalf("%s : %d revanches évitables", nom, b.revanches)
				}
				if r.NMatches != rondes*(P/2) || len(r.State.Final) != P {
					t.Fatalf("%s : %d matchs, classement de %d", nom, r.NMatches, len(r.State.Final))
				}
				ph := r.State.Phases[0]
				for i := 1; i < len(r.State.Final); i++ {
					a, c := r.State.Final[i-1], r.State.Final[i]
					pa, pc := ph.Wins[a.Player]+ph.Byes[a.Player], ph.Wins[c.Player]+ph.Byes[c.Player]
					if pa < pc || (pa == pc) != (a.Rank == c.Rank) {
						t.Fatalf("%s : le classement ne suit pas les points réels", nom)
					}
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

func TestAccelereValidate(t *testing.T) {
	refus := map[string]tournoi.PhaseConfig{
		"sans têtes de série": {Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 5, Accelerated: true},
		"avec monrad":         {Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 5, Accelerated: true, Seeding: tournoi.SeedingManual, Pairing: tournoi.PairingMonrad},
		"suisse à vies":       {Kind: tournoi.KindSwissLives, Length: 7, Accelerated: true},
		"tableau":             {Kind: tournoi.KindBracket, Length: 7, Accelerated: true, Seeding: tournoi.SeedingRating},
	}
	for nom, p := range refus {
		c := tournoi.Config{Phases: []tournoi.PhaseConfig{p}}
		if err := c.Validate(); err == nil {
			t.Errorf("%s : configuration acceptée", nom)
		}
	}
}

// TestAccelerePointVirtuelAvecDepartage : avec un départage, l'ordre dans un groupe de points
// ne suffit plus à réunir par hasard les vainqueurs du haut — seule la ronde 2 appariée sur le
// total (points réels + point virtuel) met face à face des joueurs de même total.
func TestAccelerePointVirtuelAvecDepartage(t *testing.T) {
	for _, n := range []int{8, 16, 24, 32} {
		for g := int64(0); g < 6; g++ {
			cfg := accéléré(4)
			cfg.Phases[0].Tiebreak = []string{tournoi.TiebreakPointDiff}
			d := piloter(t, cfg, g, joueursNumérotés(n, nil))
			for i, a := range d.st.Propose() {
				d.confirm(a)
				id := d.st.MatchOrder[len(d.st.MatchOrder)-1]
				w := a.A
				if (int64(i)*5+g*3+int64(n))%4 == 0 {
					w = a.B
				}
				d.must(tournoi.ResultEvent(id, w, 7, i%6, d.now))
			}
			total := map[tournoi.PlayerID]int{}
			for _, e := range d.st.RulesPhase(0).Entrants {
				total[tournoi.PlayerID(e.Player)] = e.Wins
				if e.TopHalf {
					total[tournoi.PlayerID(e.Player)]++
				}
			}
			for _, a := range d.st.Propose() {
				if a.Kind == tournoi.ActStartMatch && total[a.A] != total[a.B] {
					t.Fatalf("n=%d graine %d, ronde 2 : %s (total %d) contre %s (total %d)", n, g, a.A, total[a.A], a.B, total[a.B])
				}
			}
		}
	}
}
