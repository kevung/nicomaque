package tournoi_test

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/sim"
)

// suisseClubs : un suisse par rondes où un club dépasse la moitié du champ, avec avoid_clubs —
// la recherche n'y trouve souvent aucune ronde sans revanche ni rencontre de club, et épuise
// chaque tentative.
func suisseClubs(n, rondes, club int) (tournoi.Config, []tournoi.Player) {
	cfg := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissRounds, Rounds: rondes, Length: 5, AvoidClubs: true}}}
	players := sim.Champ(n, 6, 2, 2, 10, rand.New(rand.NewSource(int64(n))))
	for i := range players {
		if i < club {
			players[i].Club = "A"
		} else {
			players[i].Club = fmt.Sprintf("C%d", i)
		}
	}
	return cfg, players
}

// Sans limite, le compteur ne change rien : le même journal qu'une simulation qui ne le
// fixe pas, et les pas sont comptés quand même.
func TestSearchBudget_UnlimitedChangesNothing(t *testing.T) {
	cfg, players := suisseClubs(13, 9, 8)
	libre := sim.Run(cfg, players, sim.Options{Seed: 3})
	compté := sim.Run(cfg, players, sim.Options{Seed: 3, Search: &tournoi.SearchBudget{Limit: 1 << 60}})
	if libre.Err != nil || compté.Err != nil {
		t.Fatalf("erreurs : %v / %v", libre.Err, compté.Err)
	}
	if !reflect.DeepEqual(libre.Journal, compté.Journal) {
		t.Fatal("un budget non atteint a changé le journal")
	}
	if libre.SearchSteps == 0 || libre.SearchSteps != compté.SearchSteps {
		t.Fatalf("pas comptés : %d sans budget, %d avec", libre.SearchSteps, compté.SearchSteps)
	}
}

// Le compte est déterministe : même entrée, même graine, mêmes pas — donc, sous un budget, le
// même verdict au même endroit. Et l'épuisement se distingue d'un appariement impossible.
func TestSearchBudget_ExhaustionIsTypedAndReproducible(t *testing.T) {
	cfg, players := suisseClubs(37, 17, 20)
	limite := int64(5_000_000)
	var premiers []sim.Result
	for range 2 {
		r := sim.Run(cfg, players, sim.Options{Seed: 1, Search: &tournoi.SearchBudget{Limit: limite}})
		premiers = append(premiers, r)
	}
	r := premiers[0]
	if !errors.Is(r.Err, tournoi.ErrSearchBudget) || errors.Is(r.Err, sim.ErrStalled) {
		t.Fatalf("erreur %v, attendu ErrSearchBudget seul", r.Err)
	}
	var e *tournoi.SearchBudgetError
	if !errors.As(r.Err, &e) || e.Limit != limite || e.Used <= limite || e.Used != r.SearchSteps {
		t.Fatalf("erreur typée %+v, pas %d", e, r.SearchSteps)
	}
	if r.SearchSteps != premiers[1].SearchSteps || len(r.Journal) != len(premiers[1].Journal) {
		t.Fatalf("deux exécutions : %d pas / %d événements, puis %d / %d",
			r.SearchSteps, len(r.Journal), premiers[1].SearchSteps, len(premiers[1].Journal))
	}
}

// Un budget épuisé ne rend aucune action : un appariement interrompu n'en est pas un. Un
// compteur neuf par proposition borne un appariement, non le tournoi.
func TestProposeSearch_ExhaustedProposesNothing(t *testing.T) {
	cfg, players := suisseClubs(16, 5, 9)
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	st, ev, err := tournoi.New(cfg, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	_ = ev
	for _, p := range players {
		if err := st.Apply(tournoi.PlayerAddedEvent(p, now)); err != nil {
			t.Fatal(err)
		}
	}
	// Le tirage de la ronde 1 ne cherche rien ; on l'applique pour atteindre une recherche.
	for range 4 {
		acts, err := st.ProposeSearch(now, tournoi.External{}, &tournoi.SearchBudget{})
		if err != nil {
			t.Fatal(err)
		}
		if len(acts) == 0 || acts[0].Kind == tournoi.ActStartMatch {
			break
		}
		e, err := st.EventFromAction(acts[0], now)
		if err != nil {
			t.Fatal(err)
		}
		if err := st.Apply(e); err != nil {
			t.Fatal(err)
		}
	}
	compte := &tournoi.SearchBudget{}
	attendu, err := st.ProposeSearch(now, tournoi.External{}, compte)
	if err != nil || len(attendu) == 0 || compte.Used == 0 {
		t.Fatalf("proposition libre : %d actions, %d pas, %v", len(attendu), compte.Used, err)
	}
	acts, err := st.ProposeSearch(now, tournoi.External{}, &tournoi.SearchBudget{Limit: compte.Used - 1})
	if !errors.Is(err, tournoi.ErrSearchBudget) || acts != nil {
		t.Fatalf("un pas de moins : %d actions, %v", len(acts), err)
	}
	acts, err = st.ProposeSearch(now, tournoi.External{}, &tournoi.SearchBudget{Limit: compte.Used})
	if err != nil || !reflect.DeepEqual(acts, attendu) {
		t.Fatalf("budget juste : %v", err)
	}
	if !reflect.DeepEqual(st.ProposeAt(now), attendu) {
		t.Fatal("ProposeAt diffère d'une proposition sous budget suffisant")
	}
}

// Une ronde entièrement lancée ne se recherche plus : tant que ses matchs se jouent, une
// proposition ne dépense aucun pas.
func TestProposeSearch_LaunchedRoundSearchesNothing(t *testing.T) {
	cfg, players := suisseClubs(40, 7, 0)
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	st, _, err := tournoi.New(cfg, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range players {
		if err := st.Apply(tournoi.PlayerAddedEvent(p, now)); err != nil {
			t.Fatal(err)
		}
	}
	acts, err := st.ProposeSearch(now, tournoi.External{}, &tournoi.SearchBudget{})
	if err != nil || len(acts) == 0 {
		t.Fatalf("ronde 1 : %d actions, %v", len(acts), err)
	}
	for _, a := range acts {
		e, err := st.EventFromAction(a, now)
		if err != nil {
			t.Fatal(err)
		}
		if err := st.Apply(e); err != nil {
			t.Fatal(err)
		}
	}
	compte := &tournoi.SearchBudget{}
	if acts, err := st.ProposeSearch(now, tournoi.External{}, compte); err != nil || compte.Used != 0 {
		t.Fatalf("ronde lancée : %d actions, %d pas, %v ; attendu aucun pas", len(acts), compte.Used, err)
	}
}
