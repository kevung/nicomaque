package tournoi_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/rules"
	"github.com/kevung/nicomaque/sim"
)

// TestEquipesPremiereRonde : à la ronde 1, sur la matrice (équipes de 2 à 4, effectifs pairs et
// impairs, avec un joueur sans équipe), avoid_teams supprime toute paire de coéquipiers ; sans
// l'option, il y en a, sinon le cas ne serait pas couvert.
func TestEquipesPremiereRonde(t *testing.T) {
	variantes := map[string]func(*rules.PhaseRules){
		"":         func(*rules.PhaseRules) {},
		"têtes":    func(r *rules.PhaseRules) { r.Seeding = "rating" },
		"accéléré": func(r *rules.PhaseRules) { r.Accelerated = true },
		"monrad":   func(r *rules.PhaseRules) { r.Pairing = rules.PairingMonrad },
	}
	for _, format := range []rules.Format{rules.FormatSwissRounds, rules.FormatSwissLives} {
		for nomV, variante := range variantes {
			if format == rules.FormatSwissLives && nomV != "" {
				continue
			}
			for taille := 2; taille <= 4; taille++ {
				for n := 7; n <= 12; n++ {
					phase := func(évite bool) rules.Phase {
						r := rules.PhaseRules{AvoidTeams: évite, Length: 7, Lives: 2, Mode: rules.ModeRounds}
						if format == rules.FormatSwissRounds {
							r.Rounds = 3
							variante(&r)
						}
						p := rules.Phase{Format: format, Length: 7, Rules: r}
						for i := 0; i < n; i++ {
							id := rules.Ref(fmt.Sprintf("p%d", i))
							équipe := fmt.Sprintf("E%d", i/taille)
							if i == n-1 {
								équipe = "" // sans équipe
							}
							p.Entrants = append(p.Entrants, rules.Entrant{Player: id, Lives: 2, Team: équipe, Rating: float64(i + 1)})
							p.Free = append(p.Free, id)
						}
						return p
					}
					compte := func(ph rules.Phase) (coéquipiers int) {
						équipe := map[rules.Ref]string{}
						for _, e := range ph.Entrants {
							équipe[e.Player] = e.Team
						}
						for g := int64(0); g < 12; g++ {
							joués := 0
							for _, pr := range rules.Pair(ph, rules.Seed{Base: g}) {
								if pr.Kind != rules.PairMatch {
									continue
								}
								joués++
								if équipe[pr.A] != "" && équipe[pr.A] == équipe[pr.B] {
									coéquipiers++
								}
							}
							if joués != n/2 {
								t.Fatalf("%s, équipes de %d, %d joueurs : %d matchs proposés", format, taille, n, joués)
							}
						}
						return coéquipiers
					}
					if c := compte(phase(true)); c != 0 {
						t.Errorf("%s %s, équipes de %d, %d joueurs : %d paires de coéquipiers", format, nomV, taille, n, c)
					}
					if taille == 4 && nomV != "têtes" && compte(phase(false)) == 0 {
						t.Errorf("%s %s, %d joueurs : aucun coéquipier apparié sans l'option", format, nomV, n)
					}
				}
			}
		}
	}
}

// équipesChamp : P joueurs répartis en P/taille équipes.
func équipesChamp(P, taille int, graine int64) []tournoi.Player {
	pl := sim.Champ(P, 6, 2, 2, 10, rand.New(rand.NewSource(graine)))
	for i := range pl {
		pl[i].Team = fmt.Sprintf("E%d", i%(P/taille))
	}
	return pl
}

// TestEquipesSimulation : un tournoi complet. Avec avoid_teams, aucun match entre coéquipiers
// hors du secours d'un suisse à vies bloqué, et aucune revanche sur un suisse classique ; le classement par
// équipes additionne exactement les points du tournoi : Σ points = 2 × (matchs + exemptions)
// quand tout le monde a une équipe, Σ différences = 0 ; les rencontres regroupent chaque match
// terminé une fois.
func TestEquipesSimulation(t *testing.T) {
	graines := int64(6)
	if testing.Short() {
		graines = 2
	}
	cfgs := map[string]tournoi.PhaseConfig{
		"rondes": {Kind: tournoi.KindSwissRounds, Length: 5, Rounds: 5, AvoidTeams: true},
		"vies":   {Kind: tournoi.KindSwissLives, Length: 5, Lives: 2, Mode: "rounds", AvoidTeams: true},
	}
	for nom, pc := range cfgs {
		for k := int64(0); k < graines; k++ {
			for _, évite := range []bool{true, false} {
				pc.AvoidTeams = évite
				coéquipiers, revanches := 0, 0
				vus := map[[2]tournoi.PlayerID]bool{}
				hook := func(s *tournoi.State, ev tournoi.Event) error {
					if ev.Kind != tournoi.EvMatchStarted {
						return nil
					}
					// Le secours d'un suisse à vies qui ne trouve plus aucune paire (revanche,
					// « finale ») est le seul cas où rien d'autre n'existe.
					secours := ev.Label.Kind == tournoi.LabelRematch || ev.Label.Kind == tournoi.LabelFinal
					if ta := s.Players[ev.A].Team; ta != "" && ta == s.Players[ev.B].Team && !secours {
						coéquipiers++
					}
					p := [2]tournoi.PlayerID{min(ev.A, ev.B), max(ev.A, ev.B)}
					if vus[p] {
						revanches++
					}
					vus[p] = true
					return nil
				}
				pl := équipesChamp(33, 3, k)
				r := sim.Run(tournoi.Config{Phases: []tournoi.PhaseConfig{pc}}, pl, sim.Options{Seed: k, Hook: hook})
				if r.Err != nil {
					t.Fatalf("%s graine %d : %v", nom, k, r.Err)
				}
				if !évite {
					if coéquipiers == 0 {
						t.Errorf("%s graine %d : aucun coéquipier sans l'option, le cas n'est pas couvert", nom, k)
					}
					continue
				}
				if coéquipiers != 0 || nom == "rondes" && revanches != 0 {
					t.Errorf("%s graine %d : %d matchs de coéquipiers, %d revanches", nom, k, coéquipiers, revanches)
				}
				vérifierClassementEquipes(t, r.State, fmt.Sprintf("%s graine %d", nom, k))
			}
		}
	}
}

func vérifierClassementEquipes(t *testing.T, s *tournoi.State, cas string) {
	t.Helper()
	var points, diff, membres int
	for _, l := range s.TeamRanking() {
		points, diff, membres = points+l.Points, diff+l.Diff, membres+l.Members
	}
	matchs, exemptions := 0, 0
	for _, id := range s.MatchOrder {
		if s.Matches[id].Status == tournoi.Finished {
			matchs++
		}
	}
	for _, ph := range s.Phases {
		for _, b := range ph.Byes {
			exemptions += b
		}
	}
	if membres != len(s.Players) || points != 2*(matchs+exemptions) || diff != 0 {
		t.Errorf("%s : %d membres, %d demi-points, différence %d ; attendus %d, %d, 0",
			cas, membres, points, diff, len(s.Players), 2*(matchs+exemptions))
	}
	rencontres := 0
	for a := 0; a < len(s.Players)/3; a++ {
		for b := a; b < len(s.Players)/3; b++ {
			rencontres += len(s.TeamMeeting(fmt.Sprintf("E%d", a), fmt.Sprintf("E%d", b)))
		}
	}
	if rencontres != matchs {
		t.Errorf("%s : %d matchs regroupés en rencontres, %d terminés", cas, rencontres, matchs)
	}
}

// TestEquipesClassementDore : table dorée de RankTeams — somme par équipe, ordre points, puis
// différence, puis points marqués ; égalité complète = ex æquo ; un apport sans équipe ignoré.
func TestEquipesClassementDore(t *testing.T) {
	got := rules.RankTeams([]rules.TeamTally{
		{Team: "B", Points: 4, PointsFor: 10, PointsAgainst: 6},
		{Team: "A", Points: 2, PointsFor: 7, PointsAgainst: 9},
		{Team: "C", Points: 6, PointsFor: 12, PointsAgainst: 8},
		{Team: "", Points: 40, PointsFor: 90},
		{Team: "A", Points: 4, PointsFor: 5, PointsAgainst: 3},
		{Team: "D", Points: 6, PointsFor: 12, PointsAgainst: 8},
		{Team: "E", Points: 6, PointsFor: 11, PointsAgainst: 7},
		{Team: "F", Points: 6, PointsFor: 9, PointsAgainst: 9},
	})
	want := []rules.TeamStanding{
		{Team: "C", Rank: 1, Members: 1, Points: 6, Diff: 4, PointsFor: 12},
		{Team: "D", Rank: 1, Members: 1, Points: 6, Diff: 4, PointsFor: 12},
		{Team: "E", Rank: 3, Members: 1, Points: 6, Diff: 4, PointsFor: 11},
		{Team: "A", Rank: 4, Members: 2, Points: 6, Diff: 0, PointsFor: 12},
		{Team: "F", Rank: 5, Members: 1, Points: 6, Diff: 0, PointsFor: 9},
		{Team: "B", Rank: 6, Members: 1, Points: 4, Diff: 4, PointsFor: 10},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RankTeams :\n got %+v\nwant %+v", got, want)
	}
}

func TestEquipesValidate(t *testing.T) {
	for _, k := range []string{tournoi.KindBracket, tournoi.KindRoundRobin, tournoi.KindGSL} {
		c := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: k, Length: 7, AvoidTeams: true}}}
		if err := c.Validate(); err == nil {
			t.Errorf("avoid_teams accepté sur %s", k)
		}
	}
	c := tournoi.Config{Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissRounds, Length: 7, Rounds: 5, AvoidTeams: true}}}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

// TestEquipesDemiExemptions : une demi-exemption apporte un demi-point à l'équipe.
func TestEquipesDemiExemptions(t *testing.T) {
	st, _, posées := suisseDemi(t, 13, 5, 1, 0.2)
	demi := 0
	for _, r := range posées {
		demi += len(r)
	}
	if demi == 0 {
		t.Fatal("aucune demi-exemption posée : le cas n'est pas couvert")
	}
	for id, pl := range st.Players {
		pl.Team = "E" + string(id[len(id)-1])
	}
	ph := st.Phases[0]
	attendu := 0
	for _, p := range ph.Entrants {
		attendu += 2*(ph.Wins[p]+ph.Byes[p]) + len(posées[p])
	}
	points := 0
	for _, l := range st.TeamRanking() {
		points += l.Points
	}
	if points != attendu {
		t.Errorf("%d demi-points d'équipes, %d attendus (%d demi-exemptions)", points, attendu, demi)
	}
}

// TestEquipesPliageGarde : sur une ronde 1 pliée par les têtes, avoid_teams ne touche que la
// paire de coéquipiers, qui échange son second joueur avec l'échiquier voisin.
func TestEquipesPliageGarde(t *testing.T) {
	p := rules.Phase{Format: rules.FormatSwissRounds, Length: 7,
		Rules: rules.PhaseRules{AvoidTeams: true, Length: 7, Rounds: 3, Mode: rules.ModeRounds, Seeding: "rating"}}
	for i := 1; i <= 8; i++ {
		id := rules.Ref(fmt.Sprintf("p%d", i))
		équipe := id
		if i == 5 {
			équipe = "p1"
		}
		p.Entrants = append(p.Entrants, rules.Entrant{Player: id, Lives: 2, Rating: float64(i), Team: string(équipe)})
		p.Free = append(p.Free, id)
	}
	var got [][2]rules.Ref
	for _, pr := range rules.Pair(p, rules.Seed{Base: 1}) {
		got = append(got, [2]rules.Ref{pr.A, pr.B})
	}
	want := [][2]rules.Ref{{"p1", "p6"}, {"p2", "p5"}, {"p3", "p7"}, {"p4", "p8"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ronde 1 pliée : %v, attendu %v", got, want)
	}
}

// TestEquipesPetitsChamps : 2 à 5 joueurs tous coéquipiers — rien d'autre n'existe, le tournoi
// se joue quand même jusqu'au bout, sans panique.
func TestEquipesPetitsChamps(t *testing.T) {
	for _, pc := range []tournoi.PhaseConfig{
		{Kind: tournoi.KindSwissRounds, Length: 3, Rounds: 3, AvoidTeams: true},
		{Kind: tournoi.KindSwissRounds, Length: 3, Rounds: 2, AvoidTeams: true, Seeding: "rating"},
		{Kind: tournoi.KindSwissLives, Length: 3, Lives: 2, Mode: "rounds", AvoidTeams: true},
		{Kind: tournoi.KindSwissLives, Length: 3, Lives: 2, AvoidTeams: true},
	} {
		for P := 2; P <= 5; P++ {
			pl := sim.Champ(P, 6, 2, 2, 10, rand.New(rand.NewSource(int64(P))))
			for i := range pl {
				pl[i].Team = "E"
			}
			r := sim.Run(tournoi.Config{Phases: []tournoi.PhaseConfig{pc}}, pl, sim.Options{Seed: int64(P)})
			if r.Err != nil || len(r.State.Final) != P {
				t.Errorf("%s %s, %d joueurs : %v, %d classés", pc.Kind, pc.Mode, P, r.Err, len(r.State.Final))
			}
		}
	}
}
