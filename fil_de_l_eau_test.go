package tournoi_test

import (
	"encoding/json"
	"fmt"
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
				fin, exempté := false, false
				for _, a := range acts {
					if exempté && (a.Kind == tournoi.ActStartMatch || a.Kind == tournoi.ActBye) {
						t.Fatalf("n=%d g=%d : une exemption proposée n'arrête pas la proposition : %v", n, g, acts)
					}
					exempté = exempté || a.Kind == tournoi.ActBye
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

// TestFilDeLEauRefus : pas de demi-exemption au fil de l'eau — il n'y a pas de ronde à manquer.
func TestFilDeLEauRefus(t *testing.T) {
	d := piloter(t, filDeLEau(3), 1, champClubs(6, 0))
	if d.st.Apply(tournoi.HalfByeEvent("p05", 0, 2, d.now)) == nil {
		t.Fatal("demi-exemption acceptée au fil de l'eau")
	}
	classique := filDeLEau(3)
	classique.Phases[0].Mode = ""
	d = piloter(t, classique, 1, champClubs(6, 0))
	d.must(tournoi.HalfByeEvent("p05", 0, 2, d.now)) // le refus tient bien au mode
}

// filChevauché : sept joueurs, quatre rondes, des rondes qui se chevauchent — au retour, le
// reste de la ronde 3 (trois joueurs) s'apparie en entier alors que quatre joueurs sont déjà
// libres pour la ronde 4. Rend la dernière proposition ; faux si la graine ne construit pas le cas.
func filChevauché(t *testing.T, cfg tournoi.Config, g int64, clubs int) ([]tournoi.Action, bool) {
	d := piloter(t, cfg, g, champClubs(7, clubs))
	lancer := func(r int) {
		for _, a := range d.st.Propose() {
			if (a.Kind == tournoi.ActStartMatch || a.Kind == tournoi.ActBye) && a.Round == r {
				d.confirm(a)
			}
		}
	}
	enCours := func(r int) (ids []tournoi.MatchID) {
		for _, id := range d.st.MatchOrder {
			if m := d.st.Matches[id]; m.Status == tournoi.Running && m.Round == r {
				ids = append(ids, id)
			}
		}
		return ids
	}
	finir := func(id tournoi.MatchID) { d.must(tournoi.ResultEvent(id, d.st.Matches[id].A, 5, 0, d.now)) }
	lancer(1)
	for _, id := range enCours(1) {
		finir(id)
	}
	lancer(2)
	r2 := enCours(2)
	if len(r2) != 3 {
		return nil, false
	}
	finir(r2[0])
	lancer(3)
	finir(r2[1])
	lancer(3)
	for _, id := range enCours(3) {
		finir(id)
	}
	libres := false // des joueurs libres pour la ronde 4, qu'on ne lance pas
	for _, a := range d.st.Propose() {
		libres = libres || a.Kind == tournoi.ActStartMatch && a.Round == 4
	}
	finir(r2[2])
	return d.st.Propose(), libres
}

// TestFilDeLEauExemptionArrete : une exemption proposée arrête la proposition — les joueurs
// libres de la ronde suivante attendent, l'exempté va les rejoindre.
func TestFilDeLEauExemptionArrete(t *testing.T) {
	atteints := 0
	for g := int64(0); g < 10; g++ {
		acts, ok := filChevauché(t, filDeLEau(4), g, 0)
		exemption, ronde4 := false, 0
		for _, a := range acts {
			exemption = exemption || a.Kind == tournoi.ActBye && a.Round == 3
			if a.Kind == tournoi.ActStartMatch && a.Round == 4 {
				ronde4++
			}
		}
		if !ok || !exemption {
			continue
		}
		atteints++
		if ronde4 != 0 {
			t.Fatalf("graine %d : exemption proposée en ronde 3, et %d match(s) de ronde 4 avec elle : %v", g, ronde4, acts)
		}
	}
	if atteints == 0 {
		t.Fatal("aucune graine ne construit le cas : le test ne prouve rien")
	}
}

// TestFilDeLEauEvitementBorne : avoid_rounds borne l'évitement à la ronde APPARIÉE, pas à la
// dernière ronde ouverte. Même histoire jusque-là ; le reste de la ronde 3, apparié en entier
// quand la ronde 4 pourrait déjà s'ouvrir, évite les clubs si la borne est 3 et pas si elle est 2.
func TestFilDeLEauEvitementBorne(t *testing.T) {
	for g := int64(0); g < 40; g++ {
		var vues [2]string
		for i, borne := range []int{2, 3} {
			cfg := filDeLEau(4)
			cfg.Phases[0].AvoidClubs, cfg.Phases[0].AvoidRounds = true, borne
			acts, _ := filChevauché(t, cfg, g, 2)
			for _, a := range acts {
				if a.Round == 3 {
					vues[i] += fmt.Sprintf("%s %s-%s ", a.Kind, a.A, a.B)
				}
			}
		}
		if vues[0] != "" && vues[0] != vues[1] {
			return
		}
	}
	t.Error("le reste de la ronde 3 s'apparie de même borné à 2 ou à 3 rondes : avoid_rounds ignore la ronde appariée")
}

// TestFilDeLEauEvitementBorneCompte : dans les rondes que la borne couvre, le fil de l'eau
// évite les clubs autant que l'évitement permanent — même quand ces rondes s'apparient alors
// qu'une ronde hors borne est déjà ouverte. Les rondes se chevauchent : tout ce qui est
// proposé est lancé, les résultats tombent un par un dans un ordre tiré.
func TestFilDeLEauEvitementBorneCompte(t *testing.T) {
	t.Parallel()
	const borne = 2
	mêmeClub := func(n int, g int64, rondesÉvitées int) (mêmes, matchs int) {
		cfg := filDeLEau(4)
		cfg.Phases[0].AvoidClubs, cfg.Phases[0].AvoidRounds = true, rondesÉvitées
		d := piloter(t, cfg, g, champClubs(n, 3))
		rng := rand.New(rand.NewSource(g*100 + int64(n)))
		for pas := 0; pas < 2000 && !d.st.Finished; pas++ {
			for _, a := range d.st.Propose() {
				if a.Kind == tournoi.ActStartMatch || a.Kind == tournoi.ActBye {
					d.confirm(a)
				}
			}
			var enCours []tournoi.MatchID
			for _, id := range d.st.MatchOrder {
				if d.st.Matches[id].Status == tournoi.Running {
					enCours = append(enCours, id)
				}
			}
			if len(enCours) == 0 {
				break
			}
			m := d.st.Matches[enCours[rng.Intn(len(enCours))]]
			gagnant := m.A
			if rng.Intn(2) == 0 {
				gagnant = m.B
			}
			d.must(tournoi.ResultEvent(m.ID, gagnant, m.Length, 0, d.now))
		}
		for _, id := range d.st.MatchOrder {
			if m := d.st.Matches[id]; m.Status != tournoi.Cancelled && m.Round <= borne {
				matchs++
				if d.st.Players[m.A].Club == d.st.Players[m.B].Club {
					mêmes++
				}
			}
		}
		return mêmes, matchs
	}
	for _, n := range []int{7, 12} {
		var bornés, permanents, matchs int
		for g := int64(0); g < 40; g++ {
			b, mb := mêmeClub(n, g, borne)
			p, mp := mêmeClub(n, g, 0)
			if mb != mp || mb != borne*(n/2) {
				t.Fatalf("%d joueurs, graine %d : %d et %d matchs en rondes 1-%d", n, g, mb, mp, borne)
			}
			bornés, permanents, matchs = bornés+b, permanents+p, matchs+mb
		}
		t.Logf("%d joueurs : %d rencontres de même club sur %d en rondes 1-%d, borné ; %d en évitement permanent", n, bornés, matchs, borne, permanents)
		if bornés != permanents {
			t.Errorf("%d joueurs : %d rencontres de même club en rondes 1-%d avec avoid_rounds %d, %d en évitement permanent",
				n, bornés, borne, borne, permanents)
		}
	}
}
