package tournoi_test

import (
	"errors"
	"math/rand"
	"strings"
	"testing"

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
