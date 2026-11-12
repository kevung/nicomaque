package tournoi_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/kevung/nicomaque"
)

func manuel(suite ...tournoi.PhaseConfig) tournoi.Config {
	return tournoi.Config{Name: "Manuel", Phases: append([]tournoi.PhaseConfig{
		{Kind: tournoi.KindManual, Length: 5, Tiebreak: []string{tournoi.TiebreakPointDiff}}}, suite...)}
}

// sansAppariement : une phase manuelle ne propose jamais ni match, ni exemption, ni tirage.
func sansAppariement(t *testing.T, d *pilote) {
	t.Helper()
	for _, a := range d.st.Propose() {
		if a.Kind == tournoi.ActStartMatch || a.Kind == tournoi.ActBye || a.Kind == tournoi.ActDraw {
			t.Fatalf("phase manuelle : le moteur propose %s %s-%s", a.Kind, a.A, a.B)
		}
	}
}

// directeur : pose rondes rondes au hasard — chaque joueur une fois par ronde, le dernier
// exempté en effectif impair — et tire les résultats.
func directeur(t *testing.T, d *pilote, rng *rand.Rand, rondes int) {
	t.Helper()
	for r := 1; r <= rondes; r++ {
		var ids []tournoi.PlayerID
		for _, p := range d.st.Phases[d.st.Current].Entrants {
			if !d.st.Withdrawn[p] {
				ids = append(ids, p)
			}
		}
		rng.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
		for i := 0; i+1 < len(ids); i += 2 {
			ev := d.st.ManualMatchEvent(r, ids[i], ids[i+1], d.now)
			d.must(ev)
			sansAppariement(t, d)
			if rng.Intn(2) == 0 { // des matchs de la ronde se jouent en même temps
				continue
			}
			d.must(tournoi.ResultEvent(ev.MatchID, ids[i+rng.Intn(2)], 5, rng.Intn(5), d.now))
		}
		if len(ids)%2 == 1 {
			d.must(d.st.ManualByeEvent(r, ids[len(ids)-1], d.now))
		}
		for _, id := range d.st.MatchOrder {
			if m := d.st.Matches[id]; m.Status == tournoi.Running {
				w := m.A
				if rng.Intn(2) == 0 {
					w = m.B
				}
				d.must(tournoi.ResultEvent(id, w, 5, rng.Intn(5), d.now))
				sansAppariement(t, d)
			}
		}
	}
}

// TestManuelInvariants : simulation où un directeur tire les matchs au hasard. Le moteur ne
// propose aucun appariement, la clôture est proposée quand rien ne joue, le classement est
// complet et suit les points, et le journal se rejoue à l'identique.
func TestManuelInvariants(t *testing.T) {
	effectifs, graines := []int{2, 3, 5, 8, 13, 32}, 6
	if testing.Short() {
		effectifs, graines = []int{3, 8, 13}, 2
	}
	for _, n := range effectifs {
		for g := 0; g < graines; g++ {
			rng := rand.New(rand.NewSource(int64(n*100 + g)))
			d := piloter(t, manuel(), int64(g), champClubs(n, 0))
			sansAppariement(t, d)
			directeur(t, d, rng, 1+rng.Intn(5))
			var fin *tournoi.Action
			for _, a := range d.st.Propose() {
				if a.Kind == tournoi.ActFinish {
					fin = &a
				}
			}
			if fin == nil {
				t.Fatalf("n=%d graine %d : clôture non proposée : %v", n, g, d.st.Propose())
			}
			d.confirm(*fin)
			ph := d.st.Phases[0]
			if len(d.st.Final) != n || len(d.st.Warnings) > 0 {
				t.Fatalf("n=%d graine %d : classement de %d joueurs, avertissements %v", n, g, len(d.st.Final), d.st.Warnings)
			}
			for i := 1; i < n; i++ {
				a, b := d.st.Final[i-1], d.st.Final[i]
				pa, pb := ph.Wins[a.Player]+ph.Byes[a.Player], ph.Wins[b.Player]+ph.Byes[b.Player]
				if pa < pb || a.Rank > b.Rank {
					t.Fatalf("n=%d graine %d : %s (%d points, rang %d) devant %s (%d points, rang %d)", n, g, a.Player, pa, a.Rank, b.Player, pb, b.Rank)
				}
			}
			d.rejouable()
		}
	}
}

// TestManuelPuisTableau : la phase manuelle passe la main quand le directeur le décide, et la
// suite se joue normalement.
func TestManuelPuisTableau(t *testing.T) {
	for g := int64(0); g < 4; g++ {
		d := piloter(t, manuel(tournoi.PhaseConfig{Kind: tournoi.KindBracket, Length: 5, Entry: "top:4"}), g, champClubs(9, 0))
		directeur(t, d, rand.New(rand.NewSource(g)), 3)
		d.jouer(nil)
		if !d.st.Finished || len(d.st.Phases) != 2 || len(d.st.Phases[1].Entrants) != 4 {
			t.Fatalf("graine %d : fini=%v, phases %d", g, d.st.Finished, len(d.st.Phases))
		}
		if d.st.Final[0].Rank != 1 || d.st.Final[1].Rank == 1 || len(d.st.Final) != 9 {
			t.Fatalf("graine %d : classement %v", g, d.st.Final)
		}
		d.rejouable()
	}
}

// TestManuelRefus : ce que la phase refuse — et rien d'autre.
func TestManuelRefus(t *testing.T) {
	d := piloter(t, manuel(), 1, champClubs(6, 0))
	p := func(i int) tournoi.PlayerID { return tournoi.PlayerID(fmt.Sprintf("p%02d", i)) }
	refusé := func(quoi string, ev tournoi.Event) {
		t.Helper()
		if d.st.Apply(ev) == nil {
			t.Fatalf("%s : accepté", quoi)
		}
	}
	refusé("contre soi-même", d.st.ManualMatchEvent(1, p(0), p(0), d.now))
	refusé("ronde 0", d.st.ManualMatchEvent(0, p(0), p(1), d.now))
	refusé("ronde 2 avant la ronde 1", d.st.ManualMatchEvent(2, p(0), p(1), d.now))
	refusé("joueur inconnu", d.st.ManualMatchEvent(1, p(0), "zz", d.now))
	m := d.st.ManualMatchEvent(1, p(0), p(1), d.now)
	d.must(m)
	refusé("deux fois dans la ronde", d.st.ManualMatchEvent(1, p(2), p(1), d.now))
	d.must(d.st.ManualByeEvent(1, p(2), d.now))
	refusé("exempté puis apparié", d.st.ManualMatchEvent(1, p(2), p(3), d.now))
	refusé("exempté deux fois", d.st.ManualByeEvent(1, p(2), d.now))
	// Une exemption posée par erreur se retire : le joueur redevient libre dans la ronde.
	refusé("exemption absente", d.st.ManualByeCancelledEvent(1, p(3), d.now))
	refusé("exemption d'une autre ronde", d.st.ManualByeCancelledEvent(2, p(2), d.now))
	d.must(d.st.ManualByeCancelledEvent(1, p(2), d.now))
	if n := d.st.Phases[0].Byes[p(2)]; n != 0 || len(d.st.Phases[0].ByeRounds[p(2)]) != 0 {
		t.Fatalf("exemption retirée : il en reste %d (%v)", n, d.st.Phases[0].ByeRounds[p(2)])
	}
	refusé("exemption retirée deux fois", d.st.ManualByeCancelledEvent(1, p(2), d.now))
	d.must(d.st.ManualMatchEvent(1, p(2), p(4), d.now))
	d.must(tournoi.CancelEvent(d.st.MatchOrder[len(d.st.MatchOrder)-1], d.now))
	// Tant qu'un match joue : ni passage ni clôture proposés, et un passage envoyé quand même
	// est refusé.
	if acts := d.st.Propose(); len(acts) != 1 || acts[0].Kind != tournoi.ActWait {
		t.Fatalf("match en cours : %v", acts)
	}
	// Une ronde partielle est permise, et un match annulé libère ses joueurs dans la ronde.
	d.must(tournoi.CancelEvent(m.MatchID, d.now))
	d.must(d.st.ManualMatchEvent(1, p(1), p(3), d.now))
	d.must(tournoi.PlayerWithdrawnEvent(p(5), d.now))
	refusé("joueur retiré", d.st.ManualMatchEvent(1, p(4), p(5), d.now))
	d.must(d.st.ManualMatchEvent(2, p(0), p(4), d.now))
	d.rejouable()
	// Un tournoi sans phase manuelle ne change pas : un match posé à la main y reste accepté,
	// et une exemption ne s'y retire pas.
	autre := piloter(t, tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissLives, Length: 5}}}, 1, champClubs(4, 0))
	autre.must(autre.st.ManualMatchEvent(7, p(0), p(1), autre.now))
	if autre.st.Apply(autre.st.ManualByeCancelledEvent(1, p(2), autre.now)) == nil {
		t.Error("exemption retirée hors d'une phase manuelle")
	}
	// Deux phases : un joueur du tournoi qui n'est pas entré dans la phase manuelle n'y joue
	// pas, et le passage de phase attend la fin des matchs.
	deux := manuel(tournoi.PhaseConfig{Kind: tournoi.KindManual, Length: 5, Entry: "top:2"})
	e := piloter(t, deux, 1, champClubs(4, 0))
	e.must(e.st.ManualMatchEvent(1, p(0), p(1), e.now))
	suivante := tournoi.Event{Version: tournoi.JournalVersion, Kind: tournoi.EvNextPhase, Time: e.now}
	if e.st.Apply(suivante) == nil {
		t.Fatal("next_phase accepté pendant un match")
	}
	e.must(tournoi.ResultEvent(e.st.MatchOrder[0], p(0), 5, 0, e.now))
	e.must(suivante)
	dehors := 0
	for i := 0; i < 4; i++ {
		entré := false
		for _, x := range e.st.Phases[1].Entrants {
			entré = entré || x == p(i)
		}
		if !entré {
			dehors++
			autreJoueur := e.st.Phases[1].Entrants[0]
			if e.st.Apply(e.st.ManualMatchEvent(1, p(i), autreJoueur, e.now)) == nil {
				t.Errorf("joueur %s hors de la phase : match accepté", p(i))
			}
			if e.st.Apply(e.st.ManualByeEvent(1, p(i), e.now)) == nil {
				t.Errorf("joueur %s hors de la phase : exemption acceptée", p(i))
			}
		}
	}
	if dehors == 0 {
		t.Fatal("top:2 sur quatre joueurs : aucun joueur hors de la phase")
	}
	// Une phase sans aucun match se clôt quand même.
	vide := piloter(t, manuel(), 1, champClubs(3, 0))
	vide.jouer(nil)
	if !vide.st.Finished || len(vide.st.Final) != 3 {
		t.Fatalf("phase vide : fini=%v, classement %v", vide.st.Finished, vide.st.Final)
	}
	c := manuel()
	c.Phases[0].Lives = 2
	if c.Validate() == nil {
		t.Error("lives accepté sur une phase manuelle")
	}
}
