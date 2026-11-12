package tournoi_test

import (
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/rules"
	"github.com/kevung/nicomaque/sim"
)

var tousCritères = []string{tournoi.TiebreakPointDiff, tournoi.TiebreakPointsScored, tournoi.TiebreakBuchholz}

func suisseDépartagé(rondes int, critères ...string) tournoi.Config {
	return tournoi.Config{Name: "Suisse classique départagé", Phases: []tournoi.PhaseConfig{
		{Kind: tournoi.KindSwissRounds, Length: 7, Rounds: rondes, Tiebreak: critères}}}
}

// TestDepartageRegle : la règle, sur un instantané écrit à la main.
func TestDepartageRegle(t *testing.T) {
	// a, b, c ont 2 points ; d en a 0. Écarts : a +6, b +6, c +2. Marqués : a 14, b 12.
	ph := rules.Phase{Format: rules.FormatSwissRounds, Length: 7,
		Rules: rules.PhaseRules{Rounds: 2, Length: 7, Mode: rules.ModeRounds},
		Entrants: []rules.Entrant{
			{Player: "d", Lives: 99, Losses: 2, PointsFor: 4, PointsAgainst: 14, Played: []rules.Ref{"a", "b"}},
			{Player: "c", Lives: 99, Wins: 1, Byes: 1, PointsFor: 7, PointsAgainst: 5, Played: []rules.Ref{"e"}},
			{Player: "b", Lives: 99, Wins: 2, PointsFor: 12, PointsAgainst: 6, Played: []rules.Ref{"d", "e"}},
			{Player: "a", Lives: 99, Wins: 2, PointsFor: 14, PointsAgainst: 8, Played: []rules.Ref{"d", "c"}},
			{Player: "e", Lives: 99, Losses: 2, PointsFor: 9, PointsAgainst: 13, Played: []rules.Ref{"c", "b"}},
		}}
	rangs := func(critères ...string) map[rules.Ref]int {
		p := ph
		p.Rules.Tiebreak = critères
		out := map[rules.Ref]int{}
		for _, s := range rules.Rank(p) {
			out[s.Player] = s.Rank
		}
		return out
	}
	attendu := func(nom string, got map[rules.Ref]int, want map[rules.Ref]int) {
		t.Helper()
		for p, r := range want {
			if got[p] != r {
				t.Errorf("%s : %s rang %d, attendu %d (%v)", nom, p, got[p], r, got)
			}
		}
	}
	attendu("sans départage", rangs(), map[rules.Ref]int{"a": 1, "b": 1, "c": 1, "d": 4, "e": 4})
	// l'écart sépare c, pas a de b : ils restent ex æquo, sans tri caché
	attendu("écart", rangs(rules.TiebreakPointDiff), map[rules.Ref]int{"a": 1, "b": 1, "c": 3, "e": 4, "d": 5})
	attendu("écart puis marqués", rangs(rules.TiebreakPointDiff, rules.TiebreakPointsScored), map[rules.Ref]int{"a": 1, "b": 2, "c": 3})
	// Buchholz : a a joué d (0) et c (2) = 2 ; b a joué d et e = 0 ; c a joué e = 0 (l'exemption n'apporte rien)
	attendu("buchholz", rangs(rules.TiebreakBuchholz), map[rules.Ref]int{"a": 1, "b": 2, "c": 2})
	// le départage n'ordonne que des ex æquo : d (écart −10) reste derrière c malgré tout critère
	attendu("jamais devant les points", rangs(rules.TiebreakPointsScored), map[rules.Ref]int{"a": 1, "b": 2, "c": 3, "e": 4, "d": 5})
	if v := rules.TiebreakValue(ph, ph.Entrants[3], rules.TiebreakBuchholz); v != 2 {
		t.Errorf("Buchholz de a : %d, 2 attendu", v)
	}
}

// TestDepartagePlafondEtForfait : ce que le moteur relève des matchs — score plafonné à la
// longueur, forfait compté longueur–0, exemption sans points ni adversaire, correction recalculée.
func TestDepartagePlafondEtForfait(t *testing.T) {
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	st, _, err := tournoi.New(suisseDépartagé(1, tousCritères...), 1, now)
	if err != nil {
		t.Fatal(err)
	}
	appliquer := func(e tournoi.Event) {
		t.Helper()
		if err := st.Apply(e); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []tournoi.PlayerID{"a", "b", "c", "d", "e"} {
		appliquer(tournoi.PlayerAddedEvent(tournoi.Player{ID: id, Name: string(id)}, now))
	}
	var matchs []tournoi.Event
	var exempté tournoi.PlayerID
	for _, a := range st.Propose() {
		e, _ := st.EventFromAction(a, now)
		appliquer(e)
		if a.Kind == tournoi.ActStartMatch {
			matchs = append(matchs, e)
		} else {
			exempté = a.A
		}
	}
	if len(matchs) != 2 || exempté == "" {
		t.Fatalf("%d matchs, exempté %q", len(matchs), exempté)
	}
	appliquer(tournoi.ResultEvent(matchs[0].MatchID, matchs[0].A, 9, 3, now)) // 9-3 en 7 points
	appliquer(tournoi.ForfeitEvent(matchs[1].MatchID, matchs[1].B, now))
	notes := func() map[tournoi.PlayerID]tournoi.Note {
		out := map[tournoi.PlayerID]tournoi.Note{}
		for _, r := range st.Ranking() {
			out[r.Player] = r.Note
		}
		return out
	}
	n := notes()
	if g := n[matchs[0].A]; g.PointDiff != 4 || g.PointsScored != 7 {
		t.Errorf("vainqueur 9-3 en 7 : %+v, attendu écart +4 et 7 marqués", g)
	}
	if g := n[matchs[0].B]; g.PointDiff != -4 || g.PointsScored != 3 {
		t.Errorf("perdant 9-3 en 7 : %+v", g)
	}
	if g := n[matchs[1].B]; g.PointDiff != 7 || g.PointsScored != 7 {
		t.Errorf("vainqueur par forfait : %+v, attendu 7-0", g)
	}
	if g := n[exempté]; g.Byes != 1 || g.PointDiff != 0 || g.PointsScored != 0 || g.Buchholz != 0 {
		t.Errorf("exempté : %+v", g)
	}
	appliquer(tournoi.CorrectionEvent(matchs[0].MatchID, matchs[0].B, 5, 7, now))
	if g := notes()[matchs[0].B]; g.Wins != 1 || g.PointDiff != 2 || g.PointsScored != 7 {
		t.Errorf("après correction : %+v", g)
	}
}

// TestDepartageInvariants : sur la matrice, le départage n'ordonne que des ex æquo — les points
// ne changent pas, l'ordre des points non plus —, le rejeu est identique, et sans départage la
// même graine donne le même tournoi qu'avant.
func TestDepartageInvariants(t *testing.T) {
	effectifs, graines := []int{2, 3, 5, 8, 13, 32, 64, 100}, 8
	if testing.Short() {
		effectifs, graines = []int{2, 3, 5, 8, 13, 32}, 3
	}
	for _, P := range effectifs {
		for k := 0; k < graines; k++ {
			players := sim.Champ(P, 6, 2, 2, 10, rand.New(rand.NewSource(int64(P*1000+k))))
			r := sim.Run(suisseDépartagé(5, tousCritères...), players, sim.Options{Seed: int64(k)})
			if r.Err != nil {
				t.Fatalf("P=%d k=%d : %v", P, k, r.Err)
			}
			if len(r.State.Final) != P {
				t.Fatalf("P=%d k=%d : classement de %d joueurs", P, k, len(r.State.Final))
			}
			ph := r.State.Phases[0]
			for i := 1; i < len(r.State.Final); i++ {
				a, c := r.State.Final[i-1], r.State.Final[i]
				pa, pc := ph.Wins[a.Player]+ph.Byes[a.Player], ph.Wins[c.Player]+ph.Byes[c.Player]
				if pa < pc {
					t.Fatalf("P=%d k=%d : %s (%d pts) devant %s (%d pts)", P, k, a.Player, pa, c.Player, pc)
				}
				ka := [3]int{a.Note.PointDiff, a.Note.PointsScored, a.Note.Buchholz}
				kc := [3]int{c.Note.PointDiff, c.Note.PointsScored, c.Note.Buchholz}
				if (a.Rank == c.Rank) != (pa == pc && ka == kc) {
					t.Fatalf("P=%d k=%d : rangs %d et %d pour %v/%v à %d et %d points", P, k, a.Rank, c.Rank, ka, kc, pa, pc)
				}
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

// TestDepartageSimulation : 32 joueurs, 4 rondes — une de moins qu'il n'en faut pour isoler un
// invaincu. Sans départage la tête est presque toujours partagée ; avec les trois critères, un
// seul premier dans plus de 95 % des graines.
func TestDepartageSimulation(t *testing.T) {
	if testing.Short() {
		t.Skip("long")
	}
	const essais = 40
	seuls := map[bool]int{}
	for _, avec := range []bool{false, true} {
		cfg := suisseDépartagé(4)
		if avec {
			cfg = suisseDépartagé(4, tousCritères...)
		}
		for k := 0; k < essais; k++ {
			players := sim.Champ(32, 6, 2, 2, 10, rand.New(rand.NewSource(int64(k))))
			r := sim.Run(cfg, players, sim.Options{Seed: int64(k)})
			if r.Err != nil {
				t.Fatal(r.Err)
			}
			if countRank1(r.State.Final) == 1 {
				seuls[avec]++
			}
		}
	}
	if seuls[true]*100 < essais*95 {
		t.Errorf("avec départage : un seul premier dans %d tournois sur %d", seuls[true], essais)
	}
	if seuls[false]*2 > essais {
		t.Errorf("sans départage : un seul premier dans %d tournois sur %d — la mesure ne prouve rien", seuls[false], essais)
	}
}

func TestDepartageValidate(t *testing.T) {
	refus := map[string]tournoi.PhaseConfig{
		"critère inconnu": {Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 5, Tiebreak: []string{"sonneborn"}},
		"critère répété":  {Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 5, Tiebreak: []string{"buchholz", "buchholz"}},
		"suisse à vies":   {Kind: tournoi.KindSwissLives, Length: 7, Tiebreak: []string{"buchholz"}},
		"tableau":         {Kind: tournoi.KindBracket, Length: 7, Tiebreak: []string{"buchholz"}},
		"poules":          {Kind: tournoi.KindRoundRobin, Length: 7, Tiebreak: []string{"buchholz"}},
	}
	for nom, p := range refus {
		c := tournoi.Config{Phases: []tournoi.PhaseConfig{p}}
		if err := c.Validate(); err == nil {
			t.Errorf("%s : configuration acceptée", nom)
		}
	}
}
