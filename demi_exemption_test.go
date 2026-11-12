package tournoi_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/sim"
)

// suisseDemi : un suisse classique dirigé à la main, où un tirage pose des demi-exemptions avant
// chaque ronde (une chance sur dix par joueur et par ronde à venir). Rend l'état, le journal et
// les demi-exemptions posées.
func suisseDemi(t *testing.T, P, rondes int, graine int64, taux float64) (*tournoi.State, tournoi.Journal, map[tournoi.PlayerID][]int) {
	t.Helper()
	rng := rand.New(rand.NewSource(graine))
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	cfg := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissRounds, Length: 5, Rounds: rondes,
		Tiebreak: []string{tournoi.TiebreakBuchholz}}}}
	st, ev, err := tournoi.New(cfg, graine, now)
	if err != nil {
		t.Fatal(err)
	}
	journal := tournoi.Journal{ev}
	appliquer := func(ev tournoi.Event) error {
		if err := st.Apply(ev); err != nil {
			return err
		}
		journal = append(journal, ev)
		return nil
	}
	joueurs := sim.Champ(P, 6, 2, 2, 10, rng)
	for _, p := range joueurs {
		if err := appliquer(tournoi.PlayerAddedEvent(p, now)); err != nil {
			t.Fatal(err)
		}
	}
	posées := map[tournoi.PlayerID][]int{}
	engagés := map[int]map[tournoi.PlayerID]bool{}
	dernière := 0
	for étapes := 0; !st.Finished && étapes < 2000; étapes++ {
		now = now.Add(time.Minute)
		acts := st.Propose()
		// Avant qu'une ronde s'ouvre : des demi-exemptions pour elle.
		if r := st.Phases[0].Round + 1; r > dernière && r <= rondes && len(acts) > 0 && acts[0].Round == r {
			dernière = r
			posé := false
			for _, p := range joueurs {
				if rng.Float64() < taux && appliquer(tournoi.HalfByeEvent(p.ID, 0, r, now)) == nil {
					posées[p.ID] = append(posées[p.ID], r)
					posé = true
				}
			}
			if posé {
				continue
			}
		}
		avancé := false
		for _, a := range acts {
			if a.Kind == tournoi.ActWait {
				continue
			}
			ev, err := st.EventFromAction(a, now)
			if err != nil {
				t.Fatal(err)
			}
			if err := appliquer(ev); err != nil {
				t.Fatal(err)
			}
			avancé = true
			if a.Kind == tournoi.ActStartMatch || a.Kind == tournoi.ActBye {
				if engagés[a.Round] == nil {
					engagés[a.Round] = map[tournoi.PlayerID]bool{}
				}
				for _, p := range []tournoi.PlayerID{a.A, a.B} {
					if p == "" {
						continue
					}
					if engagés[a.Round][p] {
						t.Fatalf("ronde %d : %s engagé deux fois", a.Round, p)
					}
					engagés[a.Round][p] = true
				}
			}
			if a.Kind == tournoi.ActStartMatch {
				g, sa, sb := a.A, a.Length, rng.Intn(a.Length)
				if rng.Intn(2) == 0 {
					g, sa, sb = a.B, sb, sa
				}
				if err := appliquer(tournoi.ResultEvent(ev.MatchID, g, sa, sb, now)); err != nil {
					t.Fatal(err)
				}
			} else {
				break
			}
		}
		if !avancé {
			break
		}
	}
	if !st.Finished {
		t.Fatalf("P=%d graine=%d : le tournoi ne se termine pas (ronde %d)", P, graine, st.Phases[0].Round)
	}
	for p, rs := range posées {
		for _, r := range rs {
			if engagés[r][p] {
				t.Fatalf("%s est engagé à la ronde %d malgré sa demi-exemption", p, r)
			}
		}
	}
	return st, journal, posées
}

// TestDemiExemptionInvariants : un demi-exempté n'est ni apparié ni exempté à sa ronde, il
// reçoit un demi-point, le tournoi va au bout et se rejoue à l'identique.
func TestDemiExemptionInvariants(t *testing.T) {
	effectifs, graines := []int{3, 5, 8, 13, 32}, 6
	if testing.Short() {
		effectifs, graines = []int{5, 8, 13}, 2
	}
	total := 0
	for _, P := range effectifs {
		for k := int64(0); k < int64(graines); k++ {
			nom := fmt.Sprintf("P=%d graine=%d", P, k)
			st, journal, posées := suisseDemi(t, P, 5, k, 0.1)
			ph := st.Phases[0]
			for _, p := range ph.Entrants {
				total += len(posées[p])
				// Chaque ronde est jouée, exemptée ou demi-exemptée.
				joués := ph.Wins[p] + ph.Losses[p] + ph.Byes[p] + len(posées[p])
				if joués != 5 {
					t.Fatalf("%s : %s a %d rondes (%d demi-exemptions)", nom, p, joués, len(posées[p]))
				}
			}
			for _, s := range st.Final {
				if s.Note.HalfByes != len(posées[s.Player]) {
					t.Fatalf("%s : %s noté %d demi-exemptions, %d posées", nom, s.Player, s.Note.HalfByes, len(posées[s.Player]))
				}
			}
			// Le classement suit les demi-points.
			demi := func(s tournoi.Rank) int { return 2*(s.Note.Wins+s.Note.Byes) + s.Note.HalfByes }
			for i := 1; i < len(st.Final); i++ {
				if demi(st.Final[i]) > demi(st.Final[i-1]) {
					t.Fatalf("%s : %s (%d demi-points) derrière %s (%d)", nom, st.Final[i].Player, demi(st.Final[i]), st.Final[i-1].Player, demi(st.Final[i-1]))
				}
			}
			st2, err := tournoi.Replay(journal)
			if err != nil {
				t.Fatalf("%s : rejeu : %v", nom, err)
			}
			x, _ := json.Marshal(st.Final)
			y, _ := json.Marshal(st2.Final)
			if string(x) != string(y) || len(st2.Warnings) > 0 {
				t.Fatalf("%s : rejeu différent ou avertissements %v", nom, st2.Warnings)
			}
		}
	}
	if total == 0 {
		t.Fatal("aucune demi-exemption posée : le cas n'est pas couvert")
	}
}

// TestDemiExemptionRegles : posée avant la ronde seulement, une fois, sur un suisse classique ;
// retirée par un événement, jamais en modifiant le journal.
func TestDemiExemptionRegles(t *testing.T) {
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	cfg := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissRounds, Length: 5, Rounds: 3}}}
	st, _, err := tournoi.New(cfg, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 4; i++ {
		id := tournoi.PlayerID(fmt.Sprintf("j%d", i))
		if err := st.Apply(tournoi.PlayerAddedEvent(tournoi.Player{ID: id, Name: string(id)}, now)); err != nil {
			t.Fatal(err)
		}
	}
	refus := map[string]tournoi.Event{
		"ronde 0":         tournoi.HalfByeEvent("j1", 0, 0, now),
		"ronde trop loin": tournoi.HalfByeEvent("j1", 0, 4, now),
		"joueur inconnu":  tournoi.HalfByeEvent("zz", 0, 1, now),
		"phase inconnue":  tournoi.HalfByeEvent("j1", 3, 1, now),
		"rien à retirer":  tournoi.HalfByeCancelledEvent("j1", 0, 1, now),
	}
	for nom, ev := range refus {
		if err := st.Apply(ev); err == nil {
			t.Errorf("%s : accepté", nom)
		}
	}
	if err := st.Apply(tournoi.HalfByeEvent("j1", 0, 1, now)); err != nil {
		t.Fatal(err)
	}
	if err := st.Apply(tournoi.HalfByeEvent("j1", 0, 1, now)); err == nil {
		t.Error("demi-exemption posée deux fois")
	}
	if err := st.Apply(tournoi.HalfByeCancelledEvent("j1", 0, 1, now)); err != nil {
		t.Fatal(err)
	}
	if err := st.Apply(tournoi.HalfByeEvent("j2", 0, 1, now)); err != nil {
		t.Fatal(err)
	}
	// j2 demi-exempté : trois joueurs à apparier, un match et une exemption, sans j2.
	acts := st.Propose()
	for _, a := range acts {
		if a.A == "j2" || a.B == "j2" {
			t.Fatalf("j2 proposé à la ronde 1 : %+v", a)
		}
	}
	ev, err := st.EventFromAction(acts[0], now)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Apply(ev); err != nil {
		t.Fatal(err)
	}
	// La ronde 1 est ouverte : on n'y touche plus.
	if err := st.Apply(tournoi.HalfByeEvent("j3", 0, 1, now)); err == nil {
		t.Error("demi-exemption posée sur une ronde ouverte")
	}
	if err := st.Apply(tournoi.HalfByeCancelledEvent("j2", 0, 1, now)); err == nil {
		t.Error("demi-exemption retirée d'une ronde ouverte")
	}
	// Hors suisse classique : refusée.
	st, _, _ = tournoi.New(tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissLives, Length: 5, Mode: "rounds"}}}, 1, now)
	_ = st.Apply(tournoi.PlayerAddedEvent(tournoi.Player{ID: "j1", Name: "j1"}, now))
	if err := st.Apply(tournoi.HalfByeEvent("j1", 0, 1, now)); err == nil {
		t.Error("demi-exemption acceptée sur un suisse à vies")
	}
}

// TestDemiExemptionStatut : pendant sa ronde, un demi-exempté est exempt, pas « en jeu ».
func TestDemiExemptionStatut(t *testing.T) {
	cfg := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissRounds, Length: 5, Rounds: 3}}}
	d := piloter(t, cfg, 1, champClubs(6, 0))
	d.must(tournoi.HalfByeEvent("p00", 0, 1, d.now))
	d.confirm(d.st.Propose()[0])
	if s := d.st.StatusOf("p00"); s.Kind != tournoi.StatusBye || s.Round != 2 {
		t.Fatalf("demi-exempté pendant la ronde 1 : %+v", s)
	}
}
