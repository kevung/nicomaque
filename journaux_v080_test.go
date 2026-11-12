package tournoi_test

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	tournoi "github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/rules"
)

// Un tournoi commencé sous v0.8.0 se poursuit à l'identique : ses journaux, rejoués pas à pas,
// rendent les mêmes propositions, les mêmes statuts, le même classement, le même reste à jouer
// et les mêmes avertissements. Les empreintes du fichier témoin viennent de v0.8.0 et ne se
// régénèrent pas ; un champ ajouté à Action, à PlayerStatus ou à Warning doit donc être omis
// quand il est vide. Les journaux « perturbe/ » portent retraits, corrections et annulations.
//
// Seul écart voulu : la phase d'un seul entrant (journaux « top1 »), où v0.8.0 attendait sans
// fin (no_pairing) — le moteur y propose maintenant la clôture.
type pasV080 struct {
	P string `json:"p"` // empreinte de Propose
	S string `json:"s"` // empreinte de Statuses
	R string `json:"r"` // RemainingMatches de la phase courante, « min-max »
	K string `json:"k"` // empreinte de Ranking
	W string `json:"w"` // empreinte de Warnings, vide sans avertissement
}

type temoinV080 struct {
	Name    string          `json:"name"`
	Journal tournoi.Journal `json:"journal"`
	Pas     []pasV080       `json:"pas"`
}

func empreinteV080(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:6])
}

func TestJournauxV080(t *testing.T) {
	chemin := "testdata/journaux_v080.json.gz"
	// NICOMAQUE_V080 : un jeu plus large, hors dépôt, pour une mesure.
	if c := os.Getenv("NICOMAQUE_V080"); c != "" {
		chemin = c
	}
	f, err := os.Open(chemin)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var témoins []temoinV080
	if err := json.NewDecoder(z).Decode(&témoins); err != nil {
		t.Fatal(err)
	}
	pas, seuls, écarts := 0, 0, map[string]int{}
	for _, tm := range témoins {
		s, err := tournoi.Replay(tm.Journal[:1])
		if err != nil {
			t.Fatalf("%s : %v", tm.Name, err)
		}
		for i := range tm.Journal {
			if i > 0 {
				if err := s.Apply(tm.Journal[i]); err != nil {
					t.Fatalf("%s, pas %d : %v", tm.Name, i, err)
				}
			}
			pas++
			acts := s.Propose()
			var reste string
			if ph := s.RulesPhase(s.Current); ph != nil {
				rm := rules.RemainingMatches(*ph)
				reste = fmt.Sprintf("%d-%d", rm.Min, rm.Max)
			}
			avert := ""
			if len(s.Warnings) > 0 {
				avert = empreinteV080(s.Warnings)
			}
			mêmes := map[string]bool{
				"Propose":          empreinteV080(acts) == tm.Pas[i].P,
				"Statuses":         empreinteV080(s.Statuses()) == tm.Pas[i].S,
				"Ranking":          empreinteV080(s.Ranking()) == tm.Pas[i].K,
				"RemainingMatches": reste == tm.Pas[i].R,
				"Warnings":         avert == tm.Pas[i].W,
			}
			if ph := s.Phases[s.Current]; !s.Finished && ph.Index > 0 && !ph.Drawn && len(ph.Entrants) == 1 {
				seuls++
				if len(acts) != 1 || acts[0].Kind != tournoi.ActFinish {
					b, _ := json.Marshal(acts)
					t.Errorf("%s, pas %d : phase d'un seul entrant, clôture attendue ; propose %s", tm.Name, i, b)
				}
				delete(mêmes, "Propose")
				delete(mêmes, "Statuses")
			}
			for quoi, ok := range mêmes {
				if ok {
					continue
				}
				if écarts[quoi]++; écarts[quoi] <= 3 {
					b, _ := json.Marshal(acts)
					t.Errorf("%s, pas %d (%s) : %s diffère de v0.8.0 (reste %s, attendu %s) ; propose %s",
						tm.Name, i, tm.Journal[i].Kind, quoi, reste, tm.Pas[i].R, b)
				}
			}
		}
	}
	if chemin == "testdata/journaux_v080.json.gz" && seuls == 0 {
		t.Error("aucun pas de phase d'un seul entrant : le témoin top1 manque")
	}
	t.Logf("%d journaux, %d pas (dont %d en phase d'un seul entrant) ; écarts : Propose %d, Statuses %d, Ranking %d, RemainingMatches %d, Warnings %d",
		len(témoins), pas, seuls, écarts["Propose"], écarts["Statuses"], écarts["Ranking"], écarts["RemainingMatches"], écarts["Warnings"])
}
