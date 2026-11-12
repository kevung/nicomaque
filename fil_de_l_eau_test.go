package tournoi_test

import (
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/sim"
)

func filDeLEau(rondes int) tournoi.Config {
	return tournoi.Config{Name: "Suisse au fil de l'eau", Phases: []tournoi.PhaseConfig{
		{Kind: tournoi.KindSwissRounds, Length: 5, Rounds: rondes, Mode: "continuous", Tiebreak: []string{tournoi.TiebreakBuchholz}}}}
}

// TestFilDeLEauInvariants : un directeur confirme tout ce qui est proposé et saisit les
// résultats un par un, dans un ordre tiré au hasard — les rondes se chevauchent. Un match
// apparie deux joueurs au même nombre de parties, dans la ronde de ce nombre plus un ; aucune
// revanche tant qu'un joueur du même compte joue encore ; personne ne joue plus de Rounds
// parties et chacun les joue toutes ; Propose ne dépend que du journal.
func TestFilDeLEauInvariants(t *testing.T) {
	effectifs, graines := []int{4, 5, 8, 9, 16, 17, 32}, 6
	if testing.Short() {
		effectifs, graines = []int{5, 8, 17}, 2
	}
	for _, n := range effectifs {
		for g := 0; g < graines; g++ {
			rondes := 3 + g%3
			if rondes > n-1 {
				rondes = n - 1
			}
			rng := rand.New(rand.NewSource(int64(n*100 + g)))
			d := piloter(t, filDeLEau(rondes), int64(g), champClubs(n, 0))
			ph := d.st.Phases[0]
			parties := func(p tournoi.PlayerID) int { return ph.Wins[p] + ph.Losses[p] + ph.Byes[p] }
			vus := map[[2]tournoi.PlayerID]bool{}
			retrait := g%3 == 2 // un retrait en cours de route
			for pas := 0; pas < 5000 && !d.st.Finished; pas++ {
				acts := d.st.Propose()
				if st2, err := tournoi.Replay(d.j); err != nil {
					t.Fatal(err)
				} else if a, _ := json.Marshal(acts); true {
					if b, _ := json.Marshal(st2.Propose()); string(a) != string(b) {
						t.Fatalf("n=%d g=%d : Propose dépend d'autre chose que du journal\n %s\n %s", n, g, a, b)
					}
				}
				// qui joue encore sa k-ième partie, avant de rien confirmer
				attendus := map[int]bool{}
				for _, id := range d.st.MatchOrder {
					if m := d.st.Matches[id]; m.Status == tournoi.Running {
						attendus[parties(m.A)+1], attendus[parties(m.B)+1] = true, true
					}
				}
				fin := false
				for _, a := range acts {
					switch a.Kind {
					case tournoi.ActStartMatch:
						ka, kb := parties(a.A), parties(a.B)
						if ka != kb || a.Round != ka+1 || ka >= rondes {
							t.Fatalf("n=%d g=%d : %s (%d parties) contre %s (%d parties) en ronde %d", n, g, a.A, ka, a.B, kb, a.Round)
						}
						paire := [2]tournoi.PlayerID{min(a.A, a.B), max(a.A, a.B)}
						if vus[paire] && attendus[ka] {
							t.Fatalf("n=%d g=%d : revanche %v alors qu'un joueur à %d parties joue encore", n, g, paire, ka)
						}
						vus[paire] = true
						d.confirm(a)
					case tournoi.ActBye:
						if a.Round != parties(a.A)+1 || attendus[parties(a.A)] {
							t.Fatalf("n=%d g=%d : exemption de %s en ronde %d (%d parties, attendus %v)", n, g, a.A, a.Round, parties(a.A), attendus)
						}
						d.confirm(a)
					case tournoi.ActFinish:
						fin = true
					}
				}
				var enCours []tournoi.MatchID
				for _, id := range d.st.MatchOrder {
					if d.st.Matches[id].Status == tournoi.Running {
						enCours = append(enCours, id)
					}
				}
				if len(enCours) == 0 {
					if !fin {
						if len(d.st.Propose()) > 0 && d.st.Propose()[0].Kind != tournoi.ActWait {
							continue // une exemption vient d'être confirmée : la suite est proposée
						}
						t.Fatalf("n=%d g=%d : bloqué : %v", n, g, d.st.Propose())
					}
					for _, a := range d.st.Propose() {
						if a.Kind == tournoi.ActFinish {
							d.confirm(a)
						}
					}
					continue
				}
				m := d.st.Matches[enCours[rng.Intn(len(enCours))]]
				w := m.A
				if rng.Intn(2) == 0 {
					w = m.B
				}
				d.must(tournoi.ResultEvent(m.ID, w, 5, rng.Intn(5), d.now))
				if retrait && len(d.st.MatchOrder) > n {
					retrait = false
					d.must(tournoi.PlayerWithdrawnEvent(d.st.Order[rng.Intn(n)], d.now))
				}
			}
			if !d.st.Finished || len(d.st.Final) != n || len(d.st.Warnings) > 0 {
				t.Fatalf("n=%d g=%d : fini=%v, %d classés, avertissements %v", n, g, d.st.Finished, len(d.st.Final), d.st.Warnings)
			}
			for _, p := range d.st.Order {
				if k := parties(p); k > rondes || !d.st.Withdrawn[p] && k != rondes {
					t.Fatalf("n=%d g=%d : %s a joué %d parties sur %d", n, g, p, k, rondes)
				}
			}
			d.rejouable()
		}
	}
}

// TestFilDeLEauSimulation : le tournoi simulé se termine, avec autant de matchs que par rondes,
// et ne dure pas plus longtemps — on n'y attend pas la fin d'une ronde.
func TestFilDeLEauSimulation(t *testing.T) {
	graines := 8
	if testing.Short() {
		graines = 3
	}
	var fil, rondes float64
	for g := 0; g < graines; g++ {
		joueurs := sim.Champ(32, 6, 2, 2, 10, rand.New(rand.NewSource(int64(g))))
		a := sim.Run(filDeLEau(5), joueurs, sim.Options{Seed: int64(g)})
		cfg := filDeLEau(5)
		cfg.Phases[0].Mode = ""
		b := sim.Run(cfg, joueurs, sim.Options{Seed: int64(g)})
		if a.Err != nil || b.Err != nil || !a.State.Finished || a.NMatches != b.NMatches {
			t.Fatalf("graine %d : %v / %v, %d matchs au fil de l'eau, %d par rondes", g, a.Err, b.Err, a.NMatches, b.NMatches)
		}
		fil, rondes = fil+a.Minutes, rondes+b.Minutes
	}
	t.Logf("durée moyenne : %.0f min au fil de l'eau, %.0f min par rondes", fil/float64(graines), rondes/float64(graines))
	if fil > rondes {
		t.Errorf("le fil de l'eau dure plus longtemps (%.0f min) que les rondes (%.0f min)", fil/float64(graines), rondes/float64(graines))
	}
}
