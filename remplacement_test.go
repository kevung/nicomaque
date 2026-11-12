package tournoi_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/kevung/nicomaque"
)

func configsRemplacement() map[string]tournoi.Config {
	return map[string]tournoi.Config{
		"tableau":   {Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindBracket, Length: 5, Seeding: tournoi.SeedingManual, Consolation: true}}},
		"poules":    {Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindRoundRobin, Length: 5, Groups: 2, Qualifiers: 2, Seeding: tournoi.SeedingManual}, {Kind: tournoi.KindBracket, Length: 5}}},
		"classique": {Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissRounds, Length: 5, Rounds: 3, Seeding: tournoi.SeedingManual, Tiebreak: []string{tournoi.TiebreakBuchholz}}, {Kind: tournoi.KindBracket, Length: 5, Entry: "top:4"}}},
		"vies":      {Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissLives, Length: 5, Target: 4}, {Kind: tournoi.KindLivesBracket, Length: 5}}},
		"gsl":       {Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindGSL, Length: 5}}},
	}
}

func aJoué(st *tournoi.State, p tournoi.PlayerID) bool {
	for _, id := range st.MatchOrder {
		if m := st.Matches[id]; m.Status != tournoi.Cancelled && (m.A == p || m.B == p) {
			return true
		}
	}
	return false
}

// TestRemplacementInvariants : simulation avec des remplacements et des échanges tirés avant la
// ronde 1 (tirage fait ou non) et en cours de premier tour. Un remplacement n'est accepté que
// pour un joueur qui n'a pas joué ; l'ancien disparaît du classement, le remplaçant y figure ;
// le tournoi se termine avec un classement complet et le journal se rejoue à l'identique.
func TestRemplacementInvariants(t *testing.T) {
	effectifs, graines := []int{4, 7, 8, 13, 16}, 6
	if testing.Short() {
		effectifs, graines = []int{7, 8}, 2
	}
	for nom, cfg := range configsRemplacement() {
		for _, n := range effectifs {
			for g := 0; g < graines; g++ {
				rng := rand.New(rand.NewSource(int64(n*100 + g)))
				d := piloter(t, cfg, int64(g), champClubs(n, 3))
				// g pair : avant le tirage ; g impair : tirage confirmé et quelques matchs lancés.
				for k := 0; k < g%2*(1+rng.Intn(3)); k++ {
					if acts := d.st.Propose(); len(acts) > 0 && acts[0].Kind != tournoi.ActWait {
						d.confirm(acts[0])
					}
				}
				partis := map[tournoi.PlayerID]bool{}
				for k := 0; k < 4; k++ {
					old := d.st.Order[rng.Intn(len(d.st.Order))]
					if rng.Intn(3) == 0 {
						autre := d.st.Order[rng.Intn(len(d.st.Order))]
						err := d.apply(tournoi.PlayersSwappedEvent(old, autre, d.now))
						if (err == nil) != (old != autre && !aJoué(d.st, old) && !aJoué(d.st, autre)) {
							t.Fatalf("%s n=%d g=%d : échange %s-%s : %v", nom, n, g, old, autre, err)
						}
						continue
					}
					neuf := tournoi.Player{ID: tournoi.PlayerID(fmt.Sprintf("r%d", k)), Name: "Remplaçant", Rating: 5}
					joué := aJoué(d.st, old)
					err := d.apply(tournoi.PlayerReplacedEvent(old, neuf, d.now))
					if (err == nil) == joué {
						t.Fatalf("%s n=%d g=%d : remplacement de %s (a joué : %v) : %v", nom, n, g, old, joué, err)
					}
					if err == nil {
						partis[old] = true
					}
				}
				d.jouer(nil)
				if !d.st.Finished || len(d.st.Final) != n || len(d.st.Warnings) > 0 {
					t.Fatalf("%s n=%d g=%d : fini=%v, %d classés, avertissements %v, propose %v", nom, n, g, d.st.Finished, len(d.st.Final), d.st.Warnings, d.st.Propose())
				}
				vus := map[tournoi.PlayerID]bool{}
				for _, r := range d.st.Final {
					if partis[r.Player] || vus[r.Player] || d.st.Players[r.Player] == nil {
						t.Fatalf("%s n=%d g=%d : %s au classement (remplacé, en double ou inconnu)", nom, n, g, r.Player)
					}
					vus[r.Player] = true
				}
				d.rejouable()
			}
		}
	}
}

// TestRemplacementPlace : le remplaçant prend la place telle quelle — tête de série, place du
// tirage, match qui l'attend — ; un échange permute deux places.
func TestRemplacementPlace(t *testing.T) {
	cfg := configsRemplacement()["tableau"]
	d := piloter(t, cfg, 3, champClubs(8, 0))
	d.confirm(d.st.Propose()[0]) // tirage
	contre := func(p tournoi.PlayerID) tournoi.PlayerID {
		for _, a := range d.st.Propose() {
			if a.Kind == tournoi.ActStartMatch && a.A == p {
				return a.B
			} else if a.Kind == tournoi.ActStartMatch && a.B == p {
				return a.A
			}
		}
		t.Fatalf("%s sans match proposé", p)
		return ""
	}
	adv := contre("p00")
	d.must(tournoi.PlayerReplacedEvent("p00", tournoi.Player{ID: "neuf", Name: "Neuf", Seed: 99}, d.now))
	if contre("neuf") != adv || d.st.Players["p00"] != nil || d.st.Players["neuf"].Seed != 1 || d.st.Order[0] != "neuf" {
		t.Fatalf("le remplaçant n'a pas la place de p00 : contre %s (attendu %s), fiche %+v", contre("neuf"), adv, d.st.Players["neuf"])
	}
	// p02 échange sa place avec un joueur qu'il ne rencontre pas au premier tour.
	y, sy := tournoi.PlayerID("p05"), 6
	if contre("p02") == y {
		y, sy = "p06", 7
	}
	a, b := contre("p02"), contre(y)
	d.must(tournoi.PlayersSwappedEvent("p02", y, d.now))
	if contre("p02") != b || contre(y) != a || d.st.Players["p02"].Seed != sy || d.st.Players[y].Seed != 3 || d.st.Players["p02"].Name != "J2" {
		t.Fatalf("échange : p02 contre %s (attendu %s), %s contre %s (attendu %s)", contre("p02"), b, y, contre(y), a)
	}
	d.jouer(nil)
	d.rejouable()
}

// TestRemplacementRefus : ce qui interdit de changer le joueur d'une place.
func TestRemplacementRefus(t *testing.T) {
	cfg := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissRounds, Length: 5, Rounds: 3}}}
	d := piloter(t, cfg, 1, champClubs(6, 0))
	neuf := tournoi.Player{ID: "neuf", Name: "Neuf"}
	refusé := func(quoi string, ev tournoi.Event) {
		t.Helper()
		if d.st.Apply(ev) == nil {
			t.Fatalf("%s : accepté", quoi)
		}
	}
	refusé("ancien inconnu", tournoi.PlayerReplacedEvent("zz", neuf, d.now))
	refusé("remplaçant déjà inscrit", tournoi.PlayerReplacedEvent("p00", tournoi.Player{ID: "p01", Name: "X"}, d.now))
	refusé("remplaçant sans identifiant", tournoi.PlayerReplacedEvent("p00", tournoi.Player{Name: "X"}, d.now))
	refusé("remplaçant nommé comme l'exemption", tournoi.PlayerReplacedEvent("p00", tournoi.Player{ID: tournoi.BYE, Name: "X"}, d.now))
	refusé("inscrit nommé comme l'exemption", tournoi.PlayerAddedEvent(tournoi.Player{ID: tournoi.BYE, Name: "X"}, d.now))
	refusé("échange avec soi-même", tournoi.PlayersSwappedEvent("p00", "p00", d.now))
	d.must(tournoi.HalfByeEvent("p05", 0, 1, d.now))
	refusé("demi-exemption annoncée", tournoi.PlayerReplacedEvent("p05", neuf, d.now))
	refusé("échange d'un demi-exempté", tournoi.PlayersSwappedEvent("p05", "p00", d.now))
	d.must(tournoi.HalfByeCancelledEvent("p05", 0, 1, d.now))
	d.must(tournoi.PlayerWithdrawnEvent("p04", d.now))
	refusé("retiré", tournoi.PlayerReplacedEvent("p04", neuf, d.now))
	var lancé tournoi.Action
	for _, a := range d.st.Propose() {
		if a.Kind == tournoi.ActStartMatch {
			lancé = a
			break
		}
	}
	d.confirm(lancé)
	refusé("match lancé", tournoi.PlayerReplacedEvent(lancé.A, neuf, d.now))
	refusé("échange après un match lancé", tournoi.PlayersSwappedEvent(lancé.B, "p05", d.now))
	// Un match annulé n'a rien fait jouer ; il suit la place dans un échange, comme le rang
	// d'inscription.
	annulé := d.st.MatchOrder[0]
	d.must(tournoi.CancelEvent(annulé, d.now))
	rang := func(p tournoi.PlayerID) int {
		for i, x := range d.st.Order {
			if x == p {
				return i
			}
		}
		return -1
	}
	autre := tournoi.PlayerID("p05")
	if autre == lancé.A || autre == lancé.B {
		autre = "p03"
	}
	ra, rb := rang(lancé.B), rang(autre)
	d.must(tournoi.PlayersSwappedEvent(lancé.B, autre, d.now))
	if m := d.st.Matches[annulé]; m.B != autre || m.A != lancé.A {
		t.Fatalf("échange : le match annulé porte %s et %s, attendu %s et %s", m.A, m.B, lancé.A, autre)
	}
	if rang(lancé.B) != rb || rang(autre) != ra {
		t.Fatalf("échange : rangs d'inscription %d et %d, attendus %d et %d", rang(lancé.B), rang(autre), rb, ra)
	}
	// L'indisponibilité est au joueur : le remplaçant n'en hérite pas.
	d.must(tournoi.PlayerUnavailableEvent(lancé.A, time.Time{}, d.now))
	if _, ok := d.st.Unavailable[lancé.A]; !ok {
		t.Fatalf("%s devrait être indisponible", lancé.A)
	}
	d.must(tournoi.PlayerReplacedEvent(lancé.A, neuf, d.now))
	if len(d.st.Unavailable) != 0 {
		t.Fatalf("le remplaçant hérite de l'indisponibilité : %v", d.st.Unavailable)
	}
	d.jouer(nil)
	d.rejouable()
}
