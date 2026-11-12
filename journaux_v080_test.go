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
// rendent les mêmes propositions, les mêmes statuts, le même classement et le même reste à jouer. Le fichier témoin
// vient de v0.8.0 et ne se régénère pas ; il ne garde que des empreintes, donc un champ ajouté
// à Action ou à PlayerStatus doit être omis quand il est vide.
type pasV080 struct {
	P string `json:"p"` // empreinte de Propose
	S string `json:"s"` // empreinte de Statuses
	R string `json:"r"` // RemainingMatches de la phase courante, « min-max »
	K string `json:"k"` // empreinte de Ranking
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
	pas, écarts := 0, map[string]int{}
	for _, tm := range témoins {
		for i := range tm.Journal {
			s, err := tournoi.Replay(tm.Journal[:i+1])
			if err != nil {
				t.Fatalf("%s, pas %d : %v", tm.Name, i, err)
			}
			pas++
			acts := s.Propose()
			var reste string
			if ph := s.RulesPhase(s.Current); ph != nil {
				rm := rules.RemainingMatches(*ph)
				reste = fmt.Sprintf("%d-%d", rm.Min, rm.Max)
			}
			for quoi, ok := range map[string]bool{
				"Propose":          empreinteV080(acts) == tm.Pas[i].P,
				"Statuses":         empreinteV080(s.Statuses()) == tm.Pas[i].S,
				"Ranking":          empreinteV080(s.Ranking()) == tm.Pas[i].K,
				"RemainingMatches": reste == tm.Pas[i].R,
			} {
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
	t.Logf("%d journaux, %d pas ; écarts : Propose %d, Statuses %d, Ranking %d, RemainingMatches %d",
		len(témoins), pas, écarts["Propose"], écarts["Statuses"], écarts["Ranking"], écarts["RemainingMatches"])
}
