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

func suisseRondes(rondes int) tournoi.Config {
	return tournoi.Config{Name: "Suisse classique", Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissRounds, Length: 7, Rounds: rondes}}}
}

// bilanRondes : ce qu'une simulation de suisse classique a produit, relevé au fil des événements.
type bilanRondes struct {
	parRonde  map[int]map[tournoi.PlayerID]int // ronde → joueur → engagements (match ou exemption)
	byes      map[tournoi.PlayerID]int
	byesRonde map[int]int
	revanches int
	écarts    int // matchs entre joueurs séparés de plus d'un point
	matchs    int
	maxRonde  int
}

func jouerRondes(t *testing.T, cfg tournoi.Config, P int, graine int64) (sim.Result, *bilanRondes) {
	t.Helper()
	b := &bilanRondes{parRonde: map[int]map[tournoi.PlayerID]int{}, byes: map[tournoi.PlayerID]int{}, byesRonde: map[int]int{}}
	engager := func(r int, p tournoi.PlayerID) {
		if b.parRonde[r] == nil {
			b.parRonde[r] = map[tournoi.PlayerID]int{}
		}
		b.parRonde[r][p]++
		b.maxRonde = max(b.maxRonde, r)
	}
	vus := map[[2]tournoi.PlayerID]bool{}
	hook := func(s *tournoi.State, ev tournoi.Event) error {
		switch ev.Kind {
		case tournoi.EvBye:
			engager(ev.Round, ev.ID)
			b.byes[ev.ID]++
			b.byesRonde[ev.Round]++
		case tournoi.EvMatchStarted:
			if ev.Phase != 0 {
				return nil
			}
			if ev.Round == 0 {
				return fmt.Errorf("match %s sans ronde", ev.MatchID)
			}
			engager(ev.Round, ev.A)
			engager(ev.Round, ev.B)
			k := [2]tournoi.PlayerID{min(ev.A, ev.B), max(ev.A, ev.B)}
			if vus[k] {
				b.revanches++
			}
			vus[k] = true
			ph := s.Phases[0]
			pa, pb := ph.Wins[ev.A]+ph.Byes[ev.A], ph.Wins[ev.B]+ph.Byes[ev.B]
			if pa-pb > 1 || pb-pa > 1 {
				b.écarts++
			}
			b.matchs++
		}
		return nil
	}
	players := sim.Champ(P, 6, 2, 2, 10, rand.New(rand.NewSource(int64(P)*1000+graine)))
	r := sim.Run(cfg, players, sim.Options{Seed: graine, Hook: hook})
	if r.Err != nil {
		t.Fatalf("P=%d graine=%d : %v", P, graine, r.Err)
	}
	return r, b
}

// TestSuisseRondesInvariants : le suisse classique sur effectifs × rondes × graines.
func TestSuisseRondesInvariants(t *testing.T) {
	effectifs, graines := []int{2, 3, 5, 8, 13, 32, 64, 100}, 8
	if testing.Short() {
		effectifs, graines = []int{2, 3, 5, 8, 13, 32}, 3
	}
	for _, P := range effectifs {
		for _, rondes := range []int{1, 4, 7} {
			for k := 0; k < graines; k++ {
				nom := fmt.Sprintf("P=%d rondes=%d graine=%d", P, rondes, k)
				r, b := jouerRondes(t, suisseRondes(rondes), P, int64(k))
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
				}
				// pas de seconde exemption tant qu'un joueur n'en a pas eu
				lo, hi := 1<<30, 0
				for _, p := range r.State.Order {
					lo, hi = min(lo, b.byes[p]), max(hi, b.byes[p])
				}
				if hi-lo > 1 {
					t.Fatalf("%s : exemptions inégales (de %d à %d)", nom, lo, hi)
				}
				// aucune revanche tant que le champ le permet largement
				if P >= 2*rondes+2 && b.revanches != 0 {
					t.Fatalf("%s : %d revanches évitables", nom, b.revanches)
				}
				if len(r.State.Final) != P {
					t.Fatalf("%s : classement de %d joueurs", nom, len(r.State.Final))
				}
				// le classement suit les points, et les ex æquo le restent
				ph := r.State.Phases[0]
				for i := 1; i < len(r.State.Final); i++ {
					a, c := r.State.Final[i-1], r.State.Final[i]
					pa, pc := ph.Wins[a.Player]+ph.Byes[a.Player], ph.Wins[c.Player]+ph.Byes[c.Player]
					if pa < pc || (pa == pc) != (a.Rank == c.Rank) {
						t.Fatalf("%s : classement incohérent entre %s (%d pts, rang %d) et %s (%d pts, rang %d)", nom, a.Player, pa, a.Rank, c.Player, pc, c.Rank)
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

// TestSuisseRondesSimulation : 16 et 15 joueurs sur 4 rondes — aucune revanche, et des
// adversaires proches au classement.
func TestSuisseRondesSimulation(t *testing.T) {
	for _, P := range []int{16, 15} {
		loin, total := 0, 0
		for k := 0; k < 20; k++ {
			r, b := jouerRondes(t, suisseRondes(4), P, int64(k))
			if b.revanches != 0 {
				t.Fatalf("P=%d graine=%d : %d revanches", P, k, b.revanches)
			}
			if want := 4 * (P / 2); r.NMatches != want {
				t.Fatalf("P=%d : %d matchs, %d attendus", P, r.NMatches, want)
			}
			loin, total = loin+b.écarts, total+b.matchs
		}
		if loin*20 > total { // plus de 5 % de matchs à plus d'un point d'écart
			t.Errorf("P=%d : %d matchs sur %d à plus d'un point d'écart", P, loin, total)
		}
	}
}

// TestSuisseRondesRevanchesMinimales : plus de rondes que d'adversaires — les revanches sont
// inévitables, et la phase va quand même au bout.
func TestSuisseRondesRevanchesMinimales(t *testing.T) {
	r, b := jouerRondes(t, suisseRondes(5), 4, 1)
	if r.NMatches != 10 {
		t.Fatalf("%d matchs, 10 attendus", r.NMatches)
	}
	if b.revanches != 4 { // 6 paires distinctes pour 10 matchs
		t.Fatalf("%d revanches, 4 attendues", b.revanches)
	}
}

// TestSuisseRondesRetrait : un retiré après la ronde 2 n'est plus apparié, garde son parcours au
// classement, et la phase va au bout avec les autres.
func TestSuisseRondesRetrait(t *testing.T) {
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	st, _, err := tournoi.New(suisseRondes(4), 7, now)
	if err != nil {
		t.Fatal(err)
	}
	appliquer := func(e tournoi.Event) {
		t.Helper()
		if err := st.Apply(e); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range sim.Champ(8, 6, 2, 2, 10, rand.New(rand.NewSource(7))) {
		appliquer(tournoi.PlayerAddedEvent(p, now))
	}
	const retiré = tournoi.PlayerID("P3")
	parti := false
	for pas := 0; !st.Finished && pas < 500; pas++ {
		now = now.Add(time.Minute)
		if !parti && st.Phases[0].Round == 2 && len(st.Propose()) > 0 && st.Propose()[0].Round == 3 {
			appliquer(tournoi.PlayerWithdrawnEvent(retiré, now))
			parti = true
		}
		for _, a := range st.Propose() {
			if a.Kind == tournoi.ActWait {
				continue
			}
			if parti && (a.A == retiré || a.B == retiré) {
				t.Fatalf("%s proposé après son retrait : %v", retiré, a)
			}
			e, err := st.EventFromAction(a, now)
			if err != nil {
				t.Fatal(err)
			}
			appliquer(e)
			if a.Kind == tournoi.ActStartMatch {
				appliquer(tournoi.ResultEvent(e.MatchID, a.A, 7, 3, now))
			}
			if a.Kind == tournoi.ActFinish {
				break
			}
		}
	}
	if !st.Finished || !parti {
		t.Fatalf("fini=%v retiré=%v", st.Finished, parti)
	}
	if got := st.Phases[0].Round; got != 4 {
		t.Fatalf("dernière ronde %d, 4 attendue", got)
	}
	for _, rk := range st.Final {
		if rk.Player == retiré {
			if rk.Note.Kind != tournoi.NoteWithdrawn || rk.Note.Wins+rk.Note.Losses+rk.Note.Byes != 2 {
				t.Fatalf("note du retiré : %+v", rk.Note)
			}
			return
		}
	}
	t.Fatalf("%s absent du classement", retiré)
}

// TestSuisseRondesClubs : avoid_clubs est une préférence — suivie quand un appariement sans
// rencontre de club existe, abandonnée plutôt que de créer une revanche.
func TestSuisseRondesClubs(t *testing.T) {
	e := func(id, club string, adv ...rules.Ref) rules.Entrant {
		return rules.Entrant{Player: rules.Ref(id), Lives: 99, Club: club, Opponents: adv}
	}
	ph := rules.Phase{Format: rules.FormatSwissRounds, Length: 7,
		Rules:    rules.PhaseRules{Rounds: 3, Length: 7, Mode: rules.ModeRounds, AvoidClubs: true},
		Entrants: []rules.Entrant{e("a", "X"), e("b", "X"), e("c", "Y"), e("d", "Y")},
		Free:     []rules.Ref{"a", "b", "c", "d"}}
	for g := int64(0); g < 20; g++ {
		for _, pr := range rules.Pair(ph, rules.Seed{Base: g}) {
			if (pr.A == "a" && pr.B == "b") || (pr.A == "b" && pr.B == "a") || (pr.A == "c" && pr.B == "d") || (pr.A == "d" && pr.B == "c") {
				t.Fatalf("graine %d : %s contre %s, même club", g, pr.A, pr.B)
			}
		}
	}
	// a et b ont déjà joué c et d : seule reste la rencontre de club, préférée à une revanche.
	ph.Entrants = []rules.Entrant{e("a", "X", "c", "d"), e("b", "X", "c", "d"), e("c", "Y", "a", "b"), e("d", "Y", "a", "b")}
	ph.Round = 2
	prs := rules.Pair(ph, rules.Seed{Base: 1})
	if len(prs) != 2 {
		t.Fatalf("%d appariements, 2 attendus", len(prs))
	}
	for _, pr := range prs {
		if (pr.A == "a") != (pr.B == "b") && (pr.A == "b") != (pr.B == "a") {
			t.Fatalf("revanche %s contre %s plutôt qu'une rencontre de club", pr.A, pr.B)
		}
	}
}

func TestSuisseRondesValidate(t *testing.T) {
	refus := map[string]tournoi.PhaseConfig{
		"sans rondes":    {Kind: tournoi.KindSwissRounds, Length: 7},
		"bascule":        {Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 5, Target: 8},
		"vies":           {Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 5, Lives: 2},
		"continu":        {Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 5, Mode: "continuous"},
		"micro-rondes":   {Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 5, BatchMinutes: 10},
		"rondes à vies":  {Kind: tournoi.KindSwissLives, Length: 7, Rounds: 5},
		"rondes tableau": {Kind: tournoi.KindBracket, Length: 7, Rounds: 5},
	}
	for nom, p := range refus {
		c := tournoi.Config{Phases: []tournoi.PhaseConfig{p}}
		if err := c.Validate(); err == nil {
			t.Errorf("%s : configuration acceptée", nom)
		}
	}
	c := suisseRondes(5)
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	// une configuration validée se revalide (elle voyage dans le journal)
	if err := c.Validate(); err != nil {
		t.Fatalf("seconde validation : %v", err)
	}
}

// débutRondes : un suisse classique de P joueurs, prêt pour la ronde 1.
func débutRondes(t *testing.T, P, rondes int) (*tournoi.State, func(tournoi.Event), time.Time) {
	t.Helper()
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	st, _, err := tournoi.New(suisseRondes(rondes), 3, now)
	if err != nil {
		t.Fatal(err)
	}
	appliquer := func(e tournoi.Event) {
		t.Helper()
		if err := st.Apply(e); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range sim.Champ(P, 6, 2, 2, 10, rand.New(rand.NewSource(3))) {
		appliquer(tournoi.PlayerAddedEvent(p, now))
	}
	return st, appliquer, now
}

// TestSuisseRondesDerniereRondeEnDeuxVagues : la dernière ronde lancée en deux vagues (salle
// trop petite) ne clôt pas la phase quand la première vague est finie.
func TestSuisseRondesDerniereRondeEnDeuxVagues(t *testing.T) {
	st, appliquer, now := débutRondes(t, 8, 1)
	acts := st.Propose()
	if len(acts) != 4 {
		t.Fatalf("%d propositions, 4 attendues", len(acts))
	}
	for _, a := range acts[:2] {
		e, _ := st.EventFromAction(a, now)
		appliquer(e)
		appliquer(tournoi.ResultEvent(e.MatchID, a.A, 7, 2, now))
	}
	reste := st.Propose()
	if len(reste) != 2 {
		t.Fatalf("après la première vague : %v", reste)
	}
	for i, a := range reste {
		if a.Kind != tournoi.ActStartMatch || a.A != acts[2+i].A || a.B != acts[2+i].B {
			t.Fatalf("la seconde vague a changé : %v, attendu %v", a, acts[2+i])
		}
		e, _ := st.EventFromAction(a, now)
		appliquer(e)
		appliquer(tournoi.ResultEvent(e.MatchID, a.A, 7, 2, now))
	}
	if fin := st.Propose(); len(fin) != 1 || fin[0].Kind != tournoi.ActFinish {
		t.Fatalf("ronde finie : %v", fin)
	}
}

// TestSuisseRondesRetardataire : un inscrit après la ronde 1 entre à zéro point et joue la suite.
func TestSuisseRondesRetardataire(t *testing.T) {
	st, appliquer, now := débutRondes(t, 8, 2)
	for _, a := range st.Propose() {
		e, _ := st.EventFromAction(a, now)
		appliquer(e)
		appliquer(tournoi.ResultEvent(e.MatchID, a.A, 7, 2, now))
	}
	appliquer(tournoi.PlayerAddedEvent(tournoi.Player{ID: "tard", Name: "Tard"}, now))
	matchs, byes, vu := 0, 0, false
	for _, a := range st.Propose() {
		if a.Round != 2 {
			t.Fatalf("ronde %d proposée", a.Round)
		}
		switch a.Kind {
		case tournoi.ActStartMatch:
			matchs++
		case tournoi.ActBye:
			byes++
		}
		vu = vu || a.A == "tard" || a.B == "tard"
	}
	if matchs != 4 || byes != 1 || !vu {
		t.Fatalf("ronde 2 : %d matchs, %d exemptions, retardataire apparié=%v", matchs, byes, vu)
	}
	if n := len(st.Ranking()); n != 9 {
		t.Fatalf("classement de %d joueurs", n)
	}
}
