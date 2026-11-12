package tournoi_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/rules"
	"github.com/kevung/nicomaque/sim"
)

// joueursNumérotés : n joueurs j1…jn inscrits dans cet ordre, la tête de série i au joueur
// d'indice perm[i-1] ; sans perm, ji porte la tête i.
func joueursNumérotés(n int, perm []int) []tournoi.Player {
	out := make([]tournoi.Player, n)
	for i := range out {
		out[i] = tournoi.Player{ID: tournoi.PlayerID(fmt.Sprintf("j%02d", i+1)), Name: fmt.Sprintf("J%d", i+1), Seed: i + 1, Rating: 6}
	}
	for tête, i := range perm {
		out[i].Seed = tête + 1
	}
	return out
}

// TestTetesSaisiesPlacement : la table de placement d'un tableau — 1 contre la dernière, les
// exemptions aux meilleures, les joueurs sans numéro derrière dans l'ordre d'inscription.
func TestTetesSaisiesPlacement(t *testing.T) {
	cas := []struct {
		nom   string
		seeds []int // tête de série de chaque inscrit, dans l'ordre d'inscription
		want  [][2]int
	}{ // 0 = exemption ; les paires du premier tour, en indices d'inscription (1…n)
		{"4", []int{1, 2, 3, 4}, [][2]int{{1, 4}, {2, 3}}},
		{"8", []int{1, 2, 3, 4, 5, 6, 7, 8}, [][2]int{{1, 8}, {4, 5}, {2, 7}, {3, 6}}},
		{"6 : exemptions aux têtes 1 et 2", []int{1, 2, 3, 4, 5, 6}, [][2]int{{1, 0}, {4, 5}, {2, 0}, {3, 6}}},
		{"numéros dans le désordre", []int{4, 3, 2, 1}, [][2]int{{4, 1}, {3, 2}}},
		{"sans numéro derrière, ordre d'inscription", []int{0, 2, 0, 1}, [][2]int{{4, 3}, {2, 1}}},
		{"même numéro : ordre d'inscription", []int{1, 1, 2, 2}, [][2]int{{1, 4}, {2, 3}}},
		{"aucun numéro : ordre d'inscription", []int{0, 0, 0, 0}, [][2]int{{1, 4}, {2, 3}}},
	}
	for _, c := range cas {
		var ordre []rules.Ref
		seeds, lives := map[rules.Ref]int{}, map[rules.Ref]int{}
		num := map[rules.Ref]int{rules.Bye: 0}
		for i, s := range c.seeds {
			p := rules.Ref(fmt.Sprintf("j%02d", i+1))
			ordre, seeds[p], lives[p], num[p] = append(ordre, p), s, 1, i+1
		}
		slots := rules.SeedSlots(ordre, lives, seeds)
		got := map[[2]int]bool{}
		for i := 0; i+1 < len(slots); i += 2 {
			a, b := num[slots[i]], num[slots[i+1]]
			got[[2]int{a, b}], got[[2]int{b, a}] = true, true
		}
		for _, w := range c.want {
			if !got[w] {
				t.Errorf("%s : paire %v absente de %v", c.nom, w, slots)
			}
		}
	}
	// 13 joueurs dans un tableau de 16 : les trois exemptions aux têtes 1, 2 et 3
	var ordre []rules.Ref
	seeds, lives := map[rules.Ref]int{}, map[rules.Ref]int{}
	for i := 1; i <= 13; i++ {
		p := rules.Ref(fmt.Sprintf("j%02d", i))
		ordre, seeds[p], lives[p] = append(ordre, p), i, 1
	}
	slots := rules.SeedSlots(ordre, lives, seeds)
	exemptés := map[rules.Ref]bool{}
	for i := 0; i+1 < len(slots); i += 2 {
		if slots[i+1] == rules.Bye {
			exemptés[slots[i]] = true
		}
		if slots[i] == rules.Bye {
			exemptés[slots[i+1]] = true
		}
	}
	if len(slots) != 16 || len(exemptés) != 3 || !exemptés["j01"] || !exemptés["j02"] || !exemptés["j03"] {
		t.Errorf("13 joueurs : exemptés %v dans %v", exemptés, slots)
	}
}

// TestTetesSaisiesTableau : de bout en bout — le tirage du journal porte le placement, la tête 1
// rencontre la dernière, et le rejeu est identique sur effectifs × graines.
func TestTetesSaisiesTableau(t *testing.T) {
	cfg := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindBracket, Length: 7, Consolation: true, Seeding: tournoi.SeedingManual}}}
	for _, P := range []int{2, 3, 5, 8, 13, 32} {
		for k := int64(0); k < 3; k++ {
			perm := rand.New(rand.NewSource(k)).Perm(P)
			joueurs := joueursNumérotés(P, perm)
			r := sim.Run(cfg, joueurs, sim.Options{Seed: k})
			if r.Err != nil {
				t.Fatalf("P=%d k=%d : %v", P, k, r.Err)
			}
			var slots []tournoi.PlayerID
			for _, ev := range r.Journal {
				if ev.Kind == tournoi.EvDraw {
					slots = ev.Draw.Slots
				}
			}
			tête1 := joueurs[perm[0]].ID
			adversaire := tournoi.BYE
			if P == len(slots) {
				adversaire = joueurs[perm[P-1]].ID
			}
			trouvé := false
			for i := 0; i+1 < len(slots); i += 2 {
				if (slots[i] == tête1 && slots[i+1] == adversaire) || (slots[i+1] == tête1 && slots[i] == adversaire) {
					trouvé = true
				}
			}
			if !trouvé {
				t.Fatalf("P=%d k=%d : la tête 1 (%s) ne rencontre pas %s dans %v", P, k, tête1, adversaire, slots)
			}
			if countRank1(r.State.Final) != 1 || len(r.State.Final) != P {
				t.Fatalf("P=%d k=%d : classement %v", P, k, r.State.Final)
			}
			st2, err := tournoi.Replay(r.Journal)
			if err != nil {
				t.Fatal(err)
			}
			x, _ := json.Marshal(r.State.Final)
			y, _ := json.Marshal(st2.Final)
			if string(x) != string(y) || len(st2.Warnings) > 0 {
				t.Fatalf("P=%d k=%d : rejeu différent ou avertissements %v", P, k, st2.Warnings)
			}
		}
	}
}

// TestTetesSaisiesRonde1 : la ronde 1 d'un suisse classique à têtes de série est PLIÉE — la
// moitié haute contre la moitié basse —, et la dernière tête est exemptée en effectif impair.
func TestTetesSaisiesRonde1(t *testing.T) {
	ronde1 := func(n int, seeding string, joueurs []tournoi.Player) (map[tournoi.PlayerID]tournoi.PlayerID, tournoi.PlayerID) {
		t.Helper()
		now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
		st, _, err := tournoi.New(tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 3, Seeding: seeding}}}, 5, now)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range joueurs[:n] {
			if err := st.Apply(tournoi.PlayerAddedEvent(p, now)); err != nil {
				t.Fatal(err)
			}
		}
		contre, bye := map[tournoi.PlayerID]tournoi.PlayerID{}, tournoi.PlayerID("")
		for _, a := range st.Propose() {
			if a.Kind == tournoi.ActBye {
				bye = a.A
			} else {
				contre[a.A], contre[a.B] = a.B, a.A
			}
		}
		return contre, bye
	}
	contre, bye := ronde1(8, tournoi.SeedingManual, joueursNumérotés(8, nil))
	for i := 1; i <= 4; i++ {
		a, b := tournoi.PlayerID(fmt.Sprintf("j%02d", i)), tournoi.PlayerID(fmt.Sprintf("j%02d", i+4))
		if contre[a] != b {
			t.Errorf("8 joueurs : %s contre %s, attendu %s", a, contre[a], b)
		}
	}
	if bye != "" {
		t.Errorf("8 joueurs : %s exempté", bye)
	}
	contre, bye = ronde1(7, tournoi.SeedingManual, joueursNumérotés(7, nil))
	if contre["j01"] != "j04" || contre["j03"] != "j06" || bye != "j07" {
		t.Errorf("7 joueurs : %v, exempté %s", contre, bye)
	}
	// par cote : le meilleur PR (le plus bas) est la tête 1
	parCote := joueursNumérotés(4, nil)
	for i := range parCote {
		parCote[i].Seed, parCote[i].Rating = 0, float64(10-i) // j04 le meilleur
	}
	contre, _ = ronde1(4, tournoi.SeedingRating, parCote)
	if contre["j04"] != "j02" || contre["j03"] != "j01" {
		t.Errorf("par cote : %v", contre)
	}
}

func TestTetesSaisiesValidate(t *testing.T) {
	for _, k := range []string{tournoi.KindSwissLives, tournoi.KindGSL, tournoi.KindRoundRobin} {
		c := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: k, Length: 7, Seeding: tournoi.SeedingManual}}}
		if err := c.Validate(); err == nil {
			t.Errorf("seeding seed accepté sur %s", k)
		}
	}
	c := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 3, Seeding: "elo"}}}
	if err := c.Validate(); err == nil {
		t.Error("seeding inconnu accepté sur un suisse classique")
	}
}
