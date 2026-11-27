package sim_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/sim"
)

// Une salle de N tables borne la simulation : jamais plus de N matchs en cours, ce qui attend
// une table attend (waiting_table) au lieu d'être lancé.
func TestRun_TablesBoundRunningMatches(t *testing.T) {
	formats := map[string][]tournoi.PhaseConfig{
		"swiss_rounds": {{Kind: tournoi.KindSwissRounds, Rounds: 4, Length: 5}},
		"bracket":      {{Kind: tournoi.KindBracket, Length: 5}},
		"round_robin":  {{Kind: tournoi.KindRoundRobin, GroupSize: 4, Qualifiers: 1, Length: 5}, {Kind: tournoi.KindBracket, Length: 5}},
	}
	for nom, phases := range formats {
		cfg := tournoi.Config{Phases: phases, Tables: tournoi.Tables{Count: 3}}
		players := sim.Champ(16, 6, 2, 2, 10, rand.New(rand.NewSource(1)))
		pic := 0
		r := sim.Run(cfg, players, sim.Options{Seed: 1, Hook: func(s *tournoi.State, _ tournoi.Event) error {
			pic = max(pic, len(s.Running()))
			return nil
		}})
		if r.Err != nil {
			t.Fatalf("%s : %v", nom, r.Err)
		}
		if pic > 3 {
			t.Errorf("%s : %d matchs en cours sur 3 tables", nom, pic)
		}
	}
}

// La prévision joue la même salle : une seule table allonge la fin restante.
func TestForecast_TablesLengthenTheEnd(t *testing.T) {
	players := sim.Champ(16, 6, 2, 2, 10, rand.New(rand.NewSource(2)))
	début := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	moyenne := func(tables int) float64 {
		cfg := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissRounds, Rounds: 4, Length: 5}}, Tables: tournoi.Tables{Count: tables}}
		st, ev, err := tournoi.New(cfg, 1, début)
		if err != nil {
			t.Fatal(err)
		}
		j := tournoi.Journal{ev}
		for _, p := range players {
			e := tournoi.PlayerAddedEvent(p, début)
			if err := st.Apply(e); err != nil {
				t.Fatal(err)
			}
			j = append(j, e)
		}
		d, err := sim.Forecast(j, début, 8, 8, 1)
		if err != nil {
			t.Fatal(err)
		}
		s := 0.0
		for _, x := range d {
			s += x
		}
		return s / float64(len(d))
	}
	if une, libre := moyenne(1), moyenne(0); une <= 2*libre {
		t.Errorf("une table : %.0f min, salle illimitée : %.0f min ; attendu bien plus long", une, libre)
	}
}
