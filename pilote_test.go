package tournoi_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/kevung/nicomaque"
)

// pilote : un directeur de tournoi de test — il applique des événements à la main, confirme des
// propositions et tient le journal, pour les scénarios qu'une simulation ne produit pas.
type pilote struct {
	t   *testing.T
	st  *tournoi.State
	j   tournoi.Journal
	now time.Time
}

func piloter(t *testing.T, cfg tournoi.Config, graine int64, joueurs []tournoi.Player) *pilote {
	t.Helper()
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	st, ev, err := tournoi.New(cfg, graine, now)
	if err != nil {
		t.Fatal(err)
	}
	d := &pilote{t: t, st: st, j: tournoi.Journal{ev}, now: now}
	for _, p := range joueurs {
		d.must(tournoi.PlayerAddedEvent(p, now))
	}
	return d
}

func (d *pilote) apply(ev tournoi.Event) error {
	d.now = d.now.Add(time.Minute)
	if err := d.st.Apply(ev); err != nil {
		return err
	}
	d.j = append(d.j, ev)
	return nil
}

func (d *pilote) must(ev tournoi.Event) {
	d.t.Helper()
	if err := d.apply(ev); err != nil {
		d.t.Fatal(err)
	}
}

func (d *pilote) confirm(a tournoi.Action) {
	d.t.Helper()
	ev, err := d.st.EventFromAction(a, d.now)
	if err != nil {
		d.t.Fatal(err)
	}
	d.must(ev)
}

// jouer : confirme la première proposition et fait gagner le plus petit identifiant, jusqu'à ce
// que stop rende vrai, que la clôture soit proposée (stop non nil) ou que rien n'avance.
func (d *pilote) jouer(stop func() bool) {
	d.t.Helper()
	for n := 0; n < 2000 && !d.st.Finished; n++ {
		if stop != nil && stop() {
			return
		}
		av := false
		for _, a := range d.st.Propose() {
			if a.Kind == tournoi.ActWait {
				continue
			}
			if stop != nil && a.Kind == tournoi.ActFinish {
				return
			}
			d.confirm(a)
			av = true
			break
		}
		for _, id := range d.st.MatchOrder {
			if m := d.st.Matches[id]; m.Status == tournoi.Running {
				d.must(tournoi.ResultEvent(id, min(m.A, m.B), m.Length, 1, d.now))
				av = true
			}
		}
		if !av {
			return
		}
	}
}

// rejouable : le journal du pilote se rejoue au même classement, sans avertissement nouveau.
func (d *pilote) rejouable() {
	d.t.Helper()
	st, err := tournoi.Replay(d.j)
	if err != nil {
		d.t.Fatalf("rejeu : %v", err)
	}
	if fmt.Sprint(st.Ranking()) != fmt.Sprint(d.st.Ranking()) || len(st.Warnings) != len(d.st.Warnings) {
		d.t.Fatalf("rejeu différent : %v / %v", st.Warnings, d.st.Warnings)
	}
}

func champClubs(n, clubs int) []tournoi.Player {
	var out []tournoi.Player
	for i := 0; i < n; i++ {
		p := tournoi.Player{ID: tournoi.PlayerID(fmt.Sprintf("p%02d", i)), Name: fmt.Sprintf("J%d", i), Seed: i + 1}
		if clubs > 0 {
			p.Club = fmt.Sprintf("C%d", i%clubs)
		}
		out = append(out, p)
	}
	return out
}
