package tournoi_test

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/sim"
)

// Une consolante alimentée par les perdants d'UNE ronde, jouée à côté de la phase principale.

func consoSuisse() tournoi.Config {
	return tournoi.Config{Name: "Suisse, consolante de la ronde 1, tableau", Phases: []tournoi.PhaseConfig{
		{Kind: tournoi.KindSwissRounds, Length: 5, Rounds: 3},
		{Kind: tournoi.KindBracket, Length: 3, Name: "Consolante", ConsolationOf: &tournoi.ConsolationSource{Phase: 0, Round: 1}},
		{Kind: tournoi.KindBracket, Length: 7, Entry: "top:4", Name: "Tableau"},
	}}
}

func consoTableau() tournoi.Config {
	return tournoi.Config{Name: "Tableau, consolante suisse du premier tour", Phases: []tournoi.PhaseConfig{
		{Kind: tournoi.KindBracket, Length: 7},
		{Kind: tournoi.KindSwissRounds, Length: 3, Rounds: 2, Name: "Consolante", ConsolationOf: &tournoi.ConsolationSource{Phase: 0, Round: 1}},
	}}
}

func TestConsolanteValidate(t *testing.T) {
	src := func(p, r int) *tournoi.ConsolationSource { return &tournoi.ConsolationSource{Phase: p, Round: r} }
	suisse := tournoi.PhaseConfig{Kind: tournoi.KindSwissRounds, Length: 5, Rounds: 3}
	tableau := tournoi.PhaseConfig{Kind: tournoi.KindBracket, Length: 5}
	conso := func(k string, s *tournoi.ConsolationSource) tournoi.PhaseConfig {
		p := tournoi.PhaseConfig{Kind: k, Length: 3, ConsolationOf: s}
		if k == tournoi.KindSwissRounds {
			p.Rounds = 2
		}
		return p
	}
	refus := map[string][]tournoi.PhaseConfig{
		"source_posterieure":  {suisse, conso(tournoi.KindBracket, src(1, 1))},
		"premiere_phase":      {conso(tournoi.KindBracket, src(0, 1))},
		"ronde_nulle":         {suisse, conso(tournoi.KindBracket, src(0, 0))},
		"ronde_au_dela":       {suisse, conso(tournoi.KindBracket, src(0, 4))},
		"format_conso":        {suisse, conso(tournoi.KindSwissLives, src(0, 1))},
		"source_poules":       {{Kind: tournoi.KindRoundRobin, Length: 5}, conso(tournoi.KindBracket, src(0, 1))},
		"source_fil_de_l_eau": {{Kind: tournoi.KindSwissRounds, Length: 5, Rounds: 3, Mode: "continuous"}, conso(tournoi.KindBracket, src(0, 1))},
		"source_laterale":     {suisse, conso(tournoi.KindBracket, src(0, 1)), conso(tournoi.KindBracket, src(1, 1))},
		"principale_entre":    {suisse, tableau, conso(tournoi.KindBracket, src(0, 1))},
		"entree":              {suisse, func() tournoi.PhaseConfig { p := conso(tournoi.KindBracket, src(0, 1)); p.Entry = "all"; return p }()},
	}
	for nom, phases := range refus {
		c := tournoi.Config{Phases: phases}
		err := c.Validate()
		if err == nil {
			t.Errorf("%s : configuration acceptée", nom)
			continue
		}
		if !strings.Contains(err.Error(), "consolation_of") {
			t.Errorf("%s : refus sans nommer consolation_of : %v", nom, err)
		}
	}
	for _, c := range []tournoi.Config{consoSuisse(), consoTableau(),
		{Phases: []tournoi.PhaseConfig{suisse, conso(tournoi.KindBracket, src(0, 1)), conso(tournoi.KindBracket, src(0, 2)), tableau}}} {
		if err := c.Validate(); err != nil {
			t.Errorf("%s : %v", c.Name, err)
		}
	}
}

// La chaîne principale saute la phase latérale : le passage va de la phase 0 à la phase 2, et
// le classement général ignore la consolante.
func TestConsolanteChainePrincipale(t *testing.T) {
	players := sim.Champ(16, 6, 2, 2, 10, rand.New(rand.NewSource(7)))
	r := sim.Run(consoSuisse(), players, sim.Options{Seed: 3})
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	passages := 0
	for _, ev := range r.Journal {
		if ev.Kind == tournoi.EvNextPhase {
			passages++
		}
	}
	if passages != 1 || r.State.Current != 2 {
		t.Fatalf("%d passage(s), phase courante %d ; attendu un passage vers la phase 2", passages, r.State.Current)
	}
	if !r.State.Phases[1].Side {
		t.Fatalf("la phase 1 n'est pas marquée latérale")
	}
	if len(r.State.Final) != 16 || countRank1(r.State.Final) != 1 {
		t.Fatalf("classement final : %d joueurs, %d premiers", len(r.State.Final), countRank1(r.State.Final))
	}
}

// Une phase latérale ne s'ajoute ni ne se retire au milieu des phases déjà créées : les
// événements écrits portent leurs indices. Après elles, la configuration reste libre.
func TestConsolanteReconfiguration(t *testing.T) {
	cfg := tournoi.Config{Phases: []tournoi.PhaseConfig{
		{Kind: tournoi.KindSwissRounds, Length: 5, Rounds: 2},
		{Kind: tournoi.KindBracket, Length: 7, Entry: "top:2"}}}
	players := sim.Champ(8, 6, 2, 2, 10, rand.New(rand.NewSource(1)))
	r := sim.Run(cfg, players, sim.Options{Seed: 1})
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	var j tournoi.Journal
	for _, ev := range r.Journal {
		if ev.Kind == tournoi.EvFinished {
			break
		}
		j = append(j, ev)
	}
	st, err := tournoi.Replay(j)
	if err != nil {
		t.Fatal(err)
	}
	insérée := tournoi.Config{Phases: []tournoi.PhaseConfig{cfg.Phases[0],
		{Kind: tournoi.KindBracket, Length: 3, ConsolationOf: &tournoi.ConsolationSource{Phase: 0, Round: 1}},
		cfg.Phases[1]}}
	ev := tournoi.Event{Version: tournoi.JournalVersion, Kind: tournoi.EvConfigChanged, Time: st.Last, Config: &insérée}
	var refus *tournoi.ConfigRefusal
	if err := st.Apply(ev); !errors.As(err, &refus) || refus.Field != "consolation_of" || refus.Phase != 1 {
		t.Fatalf("consolante insérée avant une phase commencée : %v", err)
	}
	// Avant le passage : la phase 1 n'existe pas encore, l'insertion est admise.
	var avant tournoi.Journal
	for _, ev := range j {
		if ev.Kind == tournoi.EvNextPhase {
			break
		}
		avant = append(avant, ev)
	}
	st, err = tournoi.Replay(avant)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Apply(ev); err != nil {
		t.Fatalf("consolante insérée après la phase en cours : %v", err)
	}
}

// perdantsDeRonde : les perdants des matchs joués de la ronde r de la phase 0 (suisse), dans
// l'ordre du journal.
func perdantsDeRonde(j tournoi.Journal, r int) []tournoi.PlayerID {
	ronde := map[tournoi.MatchID]tournoi.Event{}
	var out []tournoi.PlayerID
	for _, ev := range j {
		switch ev.Kind {
		case tournoi.EvMatchStarted:
			if ev.Phase == 0 && ev.Round == r {
				ronde[ev.MatchID] = ev
			}
		case tournoi.EvResult:
			if m, ok := ronde[ev.MatchID]; ok && !ev.Forfeit {
				if ev.Winner == m.A {
					out = append(out, m.B)
				} else {
					out = append(out, m.A)
				}
			}
		}
	}
	return out
}

// sansLaterale : le journal privé de tout ce qui touche la phase latérale k.
func sansLaterale(j tournoi.Journal, k int) tournoi.Journal {
	retirés := map[tournoi.MatchID]bool{}
	var out tournoi.Journal
	for _, ev := range j {
		switch ev.Kind {
		case tournoi.EvFinished:
			continue // la consolante n'étant pas jouée, la clôture serait refusée
		case tournoi.EvSidePhaseOpened, tournoi.EvDraw, tournoi.EvBye:
			if ev.Phase == k {
				continue
			}
		case tournoi.EvMatchStarted:
			if ev.Phase == k {
				retirés[ev.MatchID] = true
				continue
			}
		case tournoi.EvResult:
			if retirés[ev.MatchID] {
				continue
			}
		}
		out = append(out, ev)
	}
	return out
}

// La consolante de la ronde 1 s'ouvre quand la ronde est soldée, reçoit ses perdants, se joue
// pendant la suite du suisse et va au bout. Personne ne manque une ronde du suisse pour
// autant, et la phase principale est exactement celle qu'on aurait jouée sans consolante.
func TestConsolanteSuisseEnParallele(t *testing.T) {
	for _, P := range []int{3, 5, 8, 16} {
		for k := int64(0); k < 3; k++ {
			players := sim.Champ(P, 6, 2, 2, 10, rand.New(rand.NewSource(int64(P)*10+k)))
			r := sim.Run(consoSuisse(), players, sim.Options{Seed: k})
			if r.Err != nil {
				t.Fatalf("P=%d k=%d : %v", P, k, r.Err)
			}
			var ouvertures []tournoi.Event
			parallèle := false
			passé := false
			for _, ev := range r.Journal {
				switch {
				case ev.Kind == tournoi.EvSidePhaseOpened:
					ouvertures = append(ouvertures, ev)
				case ev.Kind == tournoi.EvNextPhase:
					passé = true
				case ev.Kind == tournoi.EvMatchStarted && ev.Phase == 1 && !passé:
					parallèle = true
				}
			}
			perdants := perdantsDeRonde(r.Journal, 1)
			if len(perdants) < 2 {
				if len(ouvertures) != 0 {
					t.Fatalf("P=%d : consolante ouverte avec %d perdant(s)", P, len(perdants))
				}
				continue
			}
			if len(ouvertures) != 1 || ouvertures[0].Phase != 1 || ouvertures[0].Draw == nil {
				t.Fatalf("P=%d k=%d : %d ouverture(s) de consolante", P, k, len(ouvertures))
			}
			if got := ouvertures[0].Draw.Groups[0]; !mêmesJoueurs(got, perdants) {
				t.Fatalf("P=%d k=%d : entrants %v, perdants de la ronde 1 %v", P, k, got, perdants)
			}
			if P >= 8 && !parallèle {
				t.Errorf("P=%d k=%d : aucun match de consolante avant la fin du suisse", P, k)
			}
			conso := r.State.SectionRanking(tournoi.SidePhaseSection(1))
			if len(conso) != len(perdants) || countRank1(conso) != 1 {
				t.Fatalf("P=%d k=%d : classement de consolante %v", P, k, conso)
			}
			if len(r.State.Final) != P {
				t.Fatalf("P=%d : classement général de %d joueurs", P, len(r.State.Final))
			}
			parRonde := map[tournoi.PlayerID]map[int]int{}
			for _, ev := range r.Journal {
				if (ev.Kind == tournoi.EvMatchStarted || ev.Kind == tournoi.EvBye) && ev.Phase == 0 {
					for _, p := range []tournoi.PlayerID{ev.A, ev.B, ev.ID} {
						if p == "" {
							continue
						}
						if parRonde[p] == nil {
							parRonde[p] = map[int]int{}
						}
						parRonde[p][ev.Round]++
					}
				}
			}
			for _, p := range players {
				for ronde := 1; ronde <= 3; ronde++ {
					if parRonde[p.ID][ronde] != 1 {
						t.Fatalf("P=%d k=%d : %s engagé %d fois dans la ronde %d du suisse", P, k, p.ID, parRonde[p.ID][ronde], ronde)
					}
				}
			}
			rejoué, err := tournoi.Replay(r.Journal)
			if err != nil || len(rejoué.Warnings) > 0 {
				t.Fatalf("P=%d k=%d : rejeu %v, avertissements %v", P, k, err, rejoué.Warnings)
			}
			seul, err := tournoi.Replay(sansLaterale(r.Journal, 1))
			if err != nil {
				t.Fatalf("P=%d k=%d : rejeu sans la consolante : %v", P, k, err)
			}
			a, b := fmt.Sprint(r.State.Final), fmt.Sprint(seul.Ranking())
			if a != b {
				t.Fatalf("P=%d k=%d : la consolante change le classement général\n%s\n%s", P, k, a, b)
			}
		}
	}
}

// Source tableau, consolante suisse : les perdants du premier tour (pas les exemptés).
func TestConsolanteTableauSource(t *testing.T) {
	for _, P := range []int{6, 8, 16} {
		players := sim.Champ(P, 6, 2, 2, 10, rand.New(rand.NewSource(int64(P))))
		r := sim.Run(consoTableau(), players, sim.Options{Seed: 2})
		if r.Err != nil {
			t.Fatalf("P=%d : %v", P, r.Err)
		}
		var entrants []tournoi.PlayerID
		for _, ev := range r.Journal {
			if ev.Kind == tournoi.EvSidePhaseOpened {
				entrants = ev.Draw.Groups[0]
			}
		}
		attendus := P - (P+1)/2 // un perdant par match joué du premier tour
		if P == 6 {
			attendus = 2 // tableau de 8 : deux exemptions, deux matchs joués
		}
		if len(entrants) != attendus {
			t.Fatalf("P=%d : %d entrants en consolante, attendu %d", P, len(entrants), attendus)
		}
		if countRank1(r.State.Final) != 1 || len(r.State.Final) != P {
			t.Fatalf("P=%d : classement général %v", P, r.State.Final)
		}
		if len(r.State.SectionRanking(tournoi.SidePhaseSection(1))) != attendus {
			t.Fatalf("P=%d : classement de consolante incomplet", P)
		}
	}
}

// side_phase_opened est vérifié comme un tirage : phase latérale, entrants connus, une fois.
func TestConsolanteOuvertureRefusee(t *testing.T) {
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	st, _, err := tournoi.New(consoSuisse(), 1, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []tournoi.PlayerID{"a", "b", "c", "d"} {
		if err := st.Apply(tournoi.PlayerAddedEvent(tournoi.Player{ID: id, Name: string(id)}, now)); err != nil {
			t.Fatal(err)
		}
	}
	ouvre := func(phase int, ids ...tournoi.PlayerID) error {
		return st.Apply(tournoi.Event{Version: tournoi.JournalVersion, Kind: tournoi.EvSidePhaseOpened, Time: now,
			Phase: phase, Draw: &tournoi.Draw{Groups: [][]tournoi.PlayerID{ids}}})
	}
	for nom, err := range map[string]error{
		"principale": ouvre(2, "a", "b"),
		"inconnue":   ouvre(7, "a", "b"),
		"inconnu":    ouvre(1, "a", "z"),
		"doublon":    ouvre(1, "a", "a"),
		"seul":       ouvre(1, "a"),
	} {
		if err == nil {
			t.Errorf("%s : ouverture acceptée", nom)
		}
	}
	if err := ouvre(1, "a", "b"); err != nil {
		t.Fatal(err)
	}
	if err := ouvre(1, "c", "d"); err == nil {
		t.Errorf("seconde ouverture acceptée")
	}
	if st.Current != 0 || !st.Phases[1].Opened || len(st.Phases[1].Entrants) != 2 {
		t.Fatalf("état après ouverture : courante %d, phases %d", st.Current, len(st.Phases))
	}
}

func mêmesJoueurs(a, b []tournoi.PlayerID) bool {
	if len(a) != len(b) {
		return false
	}
	vus := map[tournoi.PlayerID]int{}
	for _, p := range a {
		vus[p]++
	}
	for _, p := range b {
		vus[p]--
	}
	for _, n := range vus {
		if n != 0 {
			return false
		}
	}
	return true
}

// Une correction de la ronde source après l'ouverture : la consolante n'est pas refaite, le TD
// est prévenu des deux joueurs concernés ; corriger à nouveau fait tomber l'avertissement.
func TestConsolanteCorrectionSource(t *testing.T) {
	players := sim.Champ(8, 6, 2, 2, 10, rand.New(rand.NewSource(3)))
	r := sim.Run(consoSuisse(), players, sim.Options{Seed: 1})
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	var m tournoi.Event
	for _, ev := range r.Journal {
		if ev.Kind == tournoi.EvMatchStarted && ev.Phase == 0 && ev.Round == 1 {
			m = ev
			break
		}
	}
	st := r.State
	gagnant := st.Matches[m.MatchID].Winner
	perdant := m.A
	if perdant == gagnant {
		perdant = m.B
	}
	if err := st.Apply(tournoi.CorrectionEvent(m.MatchID, perdant, 5, 3, st.Last)); err != nil {
		t.Fatal(err)
	}
	codes := map[tournoi.WarningCode]tournoi.PlayerID{}
	for _, w := range st.Warnings {
		if w.Section == tournoi.SidePhaseSection(1) {
			codes[w.Code] = w.Player
		}
	}
	if codes[tournoi.WarnConsolationNotLoser] != perdant || codes[tournoi.WarnConsolationMissing] != gagnant || len(codes) != 2 {
		t.Fatalf("avertissements après correction : %v", st.Warnings)
	}
	if err := st.Apply(tournoi.CorrectionEvent(m.MatchID, gagnant, 5, 3, st.Last)); err != nil {
		t.Fatal(err)
	}
	for _, w := range st.Warnings {
		if w.Section == tournoi.SidePhaseSection(1) {
			t.Fatalf("avertissement resté après la seconde correction : %v", w)
		}
	}
}

// La clôture attend la consolante, quelle que soit la dernière phase principale et quelle que
// soit la ronde source — y compris la dernière, qui ne se solde qu'à la fin de la phase.
func TestConsolanteClotureAttend(t *testing.T) {
	dernières := map[string]tournoi.PhaseConfig{
		"manuelle": {Kind: tournoi.KindManual, Length: 5, Entry: "top:4"},
		"tableau":  {Kind: tournoi.KindBracket, Length: 7, Entry: "top:4"},
		"poules":   {Kind: tournoi.KindRoundRobin, Length: 5, Groups: 1, Qualifiers: 1, FinalBarrage: true, Entry: "top:4"},
		"aucune":   {},
	}
	for nom, fin := range dernières {
		for _, ronde := range []int{1, 3} {
			phases := []tournoi.PhaseConfig{
				{Kind: tournoi.KindSwissRounds, Length: 5, Rounds: 3},
				{Kind: tournoi.KindBracket, Length: 3, ConsolationOf: &tournoi.ConsolationSource{Phase: 0, Round: ronde}}}
			if fin.Kind != "" {
				phases = append(phases, fin)
			}
			players := sim.Champ(8, 6, 2, 2, 10, rand.New(rand.NewSource(int64(ronde))))
			r := sim.Run(tournoi.Config{Phases: phases}, players, sim.Options{Seed: 1})
			if r.Err != nil {
				t.Fatalf("%s ronde %d : %v", nom, ronde, r.Err)
			}
			conso := r.State.SectionRanking(tournoi.SidePhaseSection(1))
			if len(conso) != 4 || countRank1(conso) != 1 {
				t.Fatalf("%s ronde %d : clos avec la consolante inachevée : %v", nom, ronde, conso)
			}
		}
	}
	// Un finished confirmé à la main avant la fin de la consolante est refusé.
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	st, _, _ := tournoi.New(consoSuisse(), 1, now)
	for _, id := range []tournoi.PlayerID{"a", "b", "c", "d"} {
		_ = st.Apply(tournoi.PlayerAddedEvent(tournoi.Player{ID: id, Name: string(id)}, now))
	}
	if err := st.Apply(tournoi.Event{Version: tournoi.JournalVersion, Kind: tournoi.EvSidePhaseOpened, Time: now,
		Phase: 1, Draw: &tournoi.Draw{Groups: [][]tournoi.PlayerID{{"a", "b"}}}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Apply(tournoi.Event{Version: tournoi.JournalVersion, Kind: tournoi.EvFinished, Time: now}); err == nil {
		t.Fatalf("clôture acceptée pendant une consolante ouverte")
	}
}

// Changer le format d'une consolante ouverte, la principale passée au-delà : refusé parce
// qu'elle a commencé, pas parce qu'elle serait finie.
func TestConsolanteReconfigurationMotif(t *testing.T) {
	r := sim.Run(consoSuisse(), sim.Champ(8, 6, 2, 2, 10, rand.New(rand.NewSource(5))), sim.Options{Seed: 1})
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	cfg := consoSuisse()
	cfg.Phases[1].Kind, cfg.Phases[1].Rounds = tournoi.KindSwissRounds, 2
	var refus *tournoi.ConfigRefusal
	err := r.State.Apply(tournoi.ConfigChangedEvent(cfg, r.State.Last))
	if !errors.As(err, &refus) || refus.Phase != 1 || refus.Reason == tournoi.RefusalFinished {
		t.Fatalf("refus : %v", err)
	}
}
