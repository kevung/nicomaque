package tournoi

import (
	"encoding/json"
	"fmt"

	"github.com/kevung/nicomaque/rules"
)

// Kinds de phases.
const (
	KindSwissLives   = "swiss_lives"   // suisse à vies (continu ou par rondes)
	KindLivesBracket = "lives_bracket" // tableau à exemptions : 2 vies = exempt du 1er tour
	KindGSL          = "gsl"           // blocs de groupes GSL (2 vies)
	KindBracket      = "bracket"       // élimination simple, consolante, dernière chance, double élimination
	KindRoundRobin   = "round_robin"   // poules toutes rondes puis qualification (barrage si égalité)
	// KindSwissRounds : le suisse classique, à nombre de rondes FIXE et sans vies. Une variante
	// nommée, à part du suisse à vies que l'étude retient : personne n'est éliminé, et sans
	// départage ses ex æquo le restent, y compris en tête.
	KindSwissRounds = "swiss_rounds"
)

// PairingMonrad : sur un suisse classique, l'adversaire est tiré au sort dans le groupe de mêmes
// points au lieu d'être le plus proche au classement.
const PairingMonrad = rules.PairingMonrad

// Critères de départage (PhaseConfig.Tiebreak).
const (
	TiebreakPointDiff    = rules.TiebreakPointDiff
	TiebreakPointsScored = rules.TiebreakPointsScored
	TiebreakBuchholz     = rules.TiebreakBuchholz
)

// swissRoundsLives : dans un suisse classique personne n'est éliminé. Chacun porte un nombre de
// vies qu'aucun nombre de rondes n'épuise — un artifice de la mécanique partagée avec le suisse
// à vies, pas un réglage : il ne dépend pas de Rounds, qui peut changer en cours de tournoi.
const swissRoundsLives = 1 << 20

// Config décrit un tournoi : une suite de phases.
type Config struct {
	Name        string        `json:"name"`
	Phases      []PhaseConfig `json:"phases"`
	MinPerPoint float64       `json:"min_per_point,omitempty"` // durée moyenne d'un point (minutes), défaut 8
	Tables      Tables        `json:"tables,omitempty"`        // les tables de la salle (tables.go)
	// Prizes : droit d'entrée, retenue d'organisation et barème par section (prizes.go). La
	// forme ancienne — une simple liste de montants — reste lue et devient le barème du
	// classement général.
	Prizes PrizePool `json:"prizes,omitempty"`
	// Breaks : les pauses programmées de la journée (repas, remise des prix). Rien n'est
	// bloqué : une proposition dont le match rencontrerait une pause porte l'avertissement
	// WarnEndsInBreak, et le directeur décide (horaires.go).
	Breaks []TimeRange `json:"breaks,omitempty"`
}

// PhaseConfig paramètre une phase. Les champs inutiles pour un type sont ignorés.
type PhaseConfig struct {
	Kind        string `json:"kind"`
	Name        string `json:"name,omitempty"`
	Lives       int    `json:"lives,omitempty"`        // swiss_lives : nombre de vies (défaut 2)
	Rounds      int    `json:"rounds,omitempty"`       // swiss_rounds : nombre de rondes (obligatoire)
	Length      int    `json:"length"`                 // longueur des matchs (points)
	FinalLength int    `json:"final_length,omitempty"` // longueur de la finale (tableaux), 0 = Length
	// Lengths donne la longueur tour par tour d'un tableau, DU DERNIER TOUR VERS LE PREMIER
	// (« 15, 13, 11, 9 » = finale en 15, demies en 13, quarts en 11, reste en 9). L'ordre est
	// celui dans lequel un organisateur annonce son tournoi, et il ne dépend pas de la taille
	// du tableau : la même liste sert un tableau de 16 et un tableau de 64, où elle allonge les
	// quatre derniers tours. Une liste plus courte que le tableau retombe sur Length.
	Lengths []int `json:"lengths,omitempty"`
	// LengthLate allonge les matchs de la fin d'un suisse, quand il ne reste plus que
	// LateThreshold joueurs en vie ou moins. Les deux vont ensemble : l'un sans l'autre ne fait
	// rien. Un changement de longueur décidé à la main par le TD (EvLengthChanged) l'emporte.
	LengthLate    int `json:"length_late,omitempty"`
	LateThreshold int `json:"late_threshold,omitempty"`
	// BatchMinutes : micro-rondes d'un suisse continu. Les joueurs libres attendent l'échéance,
	// puis tous ceux d'un même groupe de défaites sont appariés d'un coup. 0 = appariement au
	// fil de l'eau. L'échéance est dérivée du journal (horaires.go), jamais stockée à part.
	BatchMinutes int    `json:"batch_minutes,omitempty"`
	Mode         string `json:"mode,omitempty"`    // swiss_lives : "continuous" (défaut) ou "rounds"
	Pairing      string `json:"pairing,omitempty"` // swiss_lives : "random" (défaut) ou "wins"
	// MonradWiden : Monrad seulement. Quand le dernier groupe de points ne s'apparie qu'au prix
	// d'une revanche, il s'élargit au groupe d'au-dessus, puis au suivant, tant que cela en évite
	// une. Éteint par défaut : le Monrad joue alors la revanche du dernier groupe.
	MonradWiden bool `json:"monrad_widen,omitempty"`
	AvoidClubs  bool `json:"avoid_clubs,omitempty"` // éviter les joueurs du même club quand c'est possible
	// AvoidCountries : éviter les joueurs du même pays (Player.Country) quand c'est possible,
	// comme avoid_clubs. AvoidRounds : sur un suisse classique, ces préférences ne valent que
	// jusqu'à cette ronde (0 = toutes). AvoidExcept : clubs sortis de la comparaison — le club
	// hôte, que tout le monde partage.
	AvoidCountries bool     `json:"avoid_countries,omitempty"`
	AvoidRounds    int      `json:"avoid_rounds,omitempty"`
	AvoidExcept    []string `json:"avoid_except,omitempty"`
	AllowRematch   bool     `json:"allow_rematch,omitempty"`
	Target         int      `json:"target,omitempty"`         // swiss_lives / gsl : figer quand la somme des vies vaut Target (puissance de 2)
	Consolation    bool     `json:"consolation,omitempty"`    // bracket : consolante progressive (perdants du tableau principal)
	LastChance     bool     `json:"last_chance,omitempty"`    // bracket : dernière chance (perdants de la consolante)
	Reconciliation bool     `json:"reconciliation,omitempty"` // bracket : le vainqueur de la consolante joue le vainqueur du principal (double élimination)
	Recharge       bool     `json:"recharge,omitempty"`       // bracket : en double élimination, le vainqueur du principal doit être battu deux fois
	GroupSize      int      `json:"group_size,omitempty"`     // round_robin : taille des poules (défaut 4)
	// Groups : round_robin, le NOMBRE de poules, à la place de leur taille (group_size) — une
	// seule poule fait un toutes-rondes complet. Jamais plus d'une poule pour deux joueurs.
	Groups     int `json:"groups,omitempty"`
	Qualifiers int `json:"qualifiers,omitempty"` // round_robin : qualifiés par poule (défaut 2)
	// FinalBarrage : round_robin en DERNIÈRE phase, les premiers de poule (ou les co-premiers
	// d'une poule unique) se départagent sur le tapis par un barrage final, qui désigne UN
	// vainqueur. Éteint par défaut : chaque poule garde son premier, et la phase se clôt à la fin
	// des poules et de leurs barrages de qualification.
	FinalBarrage bool   `json:"final_barrage,omitempty"`
	Entry        string `json:"entry,omitempty"` // "survivors" (défaut : joueurs encore en vie de la phase précédente), "all", "top:N"
	// Seeding place les joueurs d'un tableau par cote au lieu de les tirer au hasard :
	// "" (défaut) ou SeedingRating. Le défaut VIDE est un choix de conception et non un oubli —
	// l'étude conclut « pas de têtes de série protégées », c'est la culture actuelle du
	// backgammon. Voir seeding.go.
	Seeding string `json:"seeding,omitempty"`
	// Tiebreak : départage ARITHMÉTIQUE des ex æquo d'un suisse classique ou d'une poule (à la
	// place du barrage, qui ne reste que pour une égalité complète), critères appliqués
	// dans l'ordre après les points : TiebreakPointDiff, TiebreakPointsScored, TiebreakBuchholz.
	// Le défaut VIDE est un choix de conception et non un oubli — l'étude conclut « aucun
	// départage » : à égalité on reste ex æquo, ou on se départage sur le tapis. Ce champ sépare
	// des joueurs au même parcours sur leurs points de match et la force de leurs adversaires.
	Tiebreak []string `json:"tiebreak,omitempty"`
	// Accelerated : suisse classique accéléré (méthode de Bakou). La moitié haute du champ — les
	// premières têtes de série des joueurs de la ronde 1 — reçoit un point VIRTUEL aux rondes 1
	// et 2, pour l'appariement seulement : il n'apparaît dans aucun classement. Exige seeding,
	// que l'étude écarte par défaut : sans ordre de têtes de série, il n'y a pas de moitié haute.
	Accelerated bool `json:"accelerated,omitempty"`
}

func (p *PhaseConfig) validateTiebreak(i int) error {
	vus := map[string]bool{}
	for _, c := range p.Tiebreak {
		if c != TiebreakPointDiff && c != TiebreakPointsScored && c != TiebreakBuchholz {
			return fmt.Errorf("phase %d : critère de départage %q inconnu (%q, %q, %q)", i, c, TiebreakPointDiff, TiebreakPointsScored, TiebreakBuchholz)
		}
		if vus[c] {
			return fmt.Errorf("phase %d : critère de départage %q répété", i, c)
		}
		vus[c] = true
	}
	return nil
}

// Validate vérifie la cohérence d'une configuration et remplit les défauts.
func (c *Config) Validate() error {
	if len(c.Phases) == 0 {
		return fmt.Errorf("config : au moins une phase")
	}
	if c.MinPerPoint <= 0 {
		c.MinPerPoint = 8
	}
	if err := c.Prizes.validate(); err != nil {
		return err
	}
	for i, b := range c.Breaks {
		if !b.End.After(b.Start) {
			return fmt.Errorf("pause %d : elle finit avant de commencer (%s → %s)", i, b.Start, b.End)
		}
	}
	for i := range c.Phases {
		p := &c.Phases[i]
		if p.Length <= 0 {
			return fmt.Errorf("phase %d : longueur de match manquante", i)
		}
		for k, l := range p.Lengths {
			if l <= 0 {
				return fmt.Errorf("phase %d : lengths[%d] = %d, une longueur de match est positive", i, k, l)
			}
		}
		if p.LengthLate > 0 && p.LateThreshold <= 0 {
			return fmt.Errorf("phase %d : length_late sans late_threshold — à partir de combien de joueurs ?", i)
		}
		switch p.Kind {
		case KindSwissLives:
			if p.Lives <= 0 {
				p.Lives = 2
			}
			if p.Mode == "" {
				p.Mode = "continuous"
			}
			if p.Mode != "continuous" && p.Mode != "rounds" {
				return fmt.Errorf("phase %d : mode %q inconnu", i, p.Mode)
			}
			if p.Pairing == "" {
				p.Pairing = "random"
			}
			if p.Target != 0 && (p.Target&(p.Target-1)) != 0 {
				return fmt.Errorf("phase %d : target doit être une puissance de 2", i)
			}
			if p.Target != 0 && p.Lives != 2 {
				return fmt.Errorf("phase %d : la bascule vers un tableau à vies suppose 2 vies", i)
			}
			if p.BatchMinutes < 0 {
				return fmt.Errorf("phase %d : batch_minutes négatif (%d)", i, p.BatchMinutes)
			}
			if p.BatchMinutes > 0 && p.Mode == "rounds" {
				return fmt.Errorf("phase %d : batch_minutes en mode rondes — une ronde EST un lot", i)
			}
		case KindSwissRounds:
			if p.Rounds <= 0 {
				return fmt.Errorf("phase %d : swiss_rounds sans rounds — combien de rondes ?", i)
			}
			if p.Target != 0 || p.LengthLate != 0 {
				return fmt.Errorf("phase %d : target et length_late sont ceux du suisse à vies, pas de %q", i, p.Kind)
			}
			if p.Pairing != "" && p.Pairing != PairingMonrad {
				return fmt.Errorf("phase %d : pairing %q inconnu pour %q (vide ou %q)", i, p.Pairing, p.Kind, PairingMonrad)
			}
			if p.MonradWiden && p.Pairing != PairingMonrad {
				return fmt.Errorf("phase %d : monrad_widen sans pairing %q", i, PairingMonrad)
			}
			if p.Mode != "" && p.Mode != "rounds" {
				return fmt.Errorf("phase %d : mode %q inconnu pour %q (rondes seulement)", i, p.Mode, p.Kind)
			}
			if p.Lives != 0 && p.Lives != swissRoundsLives {
				return fmt.Errorf("phase %d : lives ne se règle pas sur %q — personne n'y est éliminé", i, p.Kind)
			}
			if p.Seeding != "" && p.Seeding != SeedingRating && p.Seeding != SeedingManual {
				return fmt.Errorf("phase %d : seeding %q inconnu (vide, %q ou %q)", i, p.Seeding, SeedingRating, SeedingManual)
			}
			if p.Accelerated && p.Seeding == "" {
				return fmt.Errorf("phase %d : accelerated sans seeding — la moitié haute se lit dans les têtes de série", i)
			}
			if p.Accelerated && p.Pairing == PairingMonrad {
				return fmt.Errorf("phase %d : accelerated et pairing %q s'excluent", i, PairingMonrad)
			}
			p.Mode, p.Lives = "rounds", swissRoundsLives
			if err := p.validateTiebreak(i); err != nil {
				return err
			}
		case KindLivesBracket, KindBracket:
			if p.Seeding != "" && p.Seeding != SeedingRating && p.Seeding != SeedingManual {
				return fmt.Errorf("phase %d : seeding %q inconnu (vide, %q ou %q)", i, p.Seeding, SeedingRating, SeedingManual)
			}
			if p.Recharge && !p.Reconciliation {
				return fmt.Errorf("phase %d : recharge sans reconciliation", i)
			}
			if p.Reconciliation && !p.Consolation {
				return fmt.Errorf("phase %d : reconciliation sans consolante", i)
			}
		case KindGSL:
			p.Lives = 2
			if p.Target != 0 && (p.Target&(p.Target-1)) != 0 {
				return fmt.Errorf("phase %d : target doit être une puissance de 2", i)
			}
		case KindRoundRobin:
			if p.Groups < 0 {
				return fmt.Errorf("phase %d : groups négatif (%d)", i, p.Groups)
			}
			if p.Groups > 0 && p.GroupSize > 0 {
				return fmt.Errorf("phase %d : groups et group_size s'excluent — le nombre de poules OU leur taille", i)
			}
			if p.Groups == 0 && p.GroupSize <= 0 {
				p.GroupSize = 4
			}
			if p.Qualifiers <= 0 {
				p.Qualifiers = 2
			}
			if p.Groups == 0 && p.Qualifiers >= p.GroupSize {
				return fmt.Errorf("phase %d : qualifiers doit être < group_size", i)
			}
			if p.Seeding != "" && p.Seeding != SeedingRating && p.Seeding != SeedingManual {
				return fmt.Errorf("phase %d : seeding %q inconnu (vide, %q ou %q)", i, p.Seeding, SeedingRating, SeedingManual)
			}
			if err := p.validateTiebreak(i); err != nil {
				return err
			}
		default:
			return fmt.Errorf("phase %d : type %q inconnu", i, p.Kind)
		}
		if p.Entry == "" {
			p.Entry = "survivors"
		}
		if p.Rounds != 0 && p.Kind != KindSwissRounds {
			return fmt.Errorf("phase %d : rounds ne s'applique qu'à %q, pas à %q", i, KindSwissRounds, p.Kind)
		}
		if p.Accelerated && p.Kind != KindSwissRounds {
			return fmt.Errorf("phase %d : accelerated ne s'applique qu'à %q, pas à %q", i, KindSwissRounds, p.Kind)
		}
		if len(p.Tiebreak) != 0 && p.Kind != KindSwissRounds && p.Kind != KindRoundRobin {
			return fmt.Errorf("phase %d : tiebreak ne s'applique qu'à %q et %q, pas à %q", i, KindSwissRounds, KindRoundRobin, p.Kind)
		}
		if p.AvoidRounds < 0 || p.AvoidRounds != 0 && (p.Kind != KindSwissRounds || !p.AvoidClubs && !p.AvoidCountries) {
			return fmt.Errorf("phase %d : avoid_rounds borne avoid_clubs ou avoid_countries sur %q", i, KindSwissRounds)
		}
		if len(p.AvoidExcept) != 0 && !p.AvoidClubs {
			return fmt.Errorf("phase %d : avoid_except sans avoid_clubs", i)
		}
		if p.AvoidCountries && p.Kind != KindSwissRounds && p.Kind != KindSwissLives {
			return fmt.Errorf("phase %d : avoid_countries ne s'applique qu'aux suisses, pas à %q", i, p.Kind)
		}
		if p.BatchMinutes != 0 && p.Kind != KindSwissLives {
			return fmt.Errorf("phase %d : batch_minutes ne s'applique qu'au suisse continu, pas à %q", i, p.Kind)
		}
		if p.FinalBarrage && p.Kind != KindRoundRobin {
			return fmt.Errorf("phase %d : final_barrage ne vaut que pour round_robin", i)
		}
		if p.Groups != 0 && p.Kind != KindRoundRobin {
			return fmt.Errorf("phase %d : groups ne s'applique qu'à %q, pas à %q", i, KindRoundRobin, p.Kind)
		}
		if p.Seeding != "" && p.Kind != KindBracket && p.Kind != KindLivesBracket && p.Kind != KindSwissRounds && p.Kind != KindRoundRobin {
			return fmt.Errorf("phase %d : seeding ne s'applique qu'aux tableaux, au suisse classique et aux poules, pas à %q", i, p.Kind)
		}
	}
	return nil
}

// ParseConfig lit une configuration JSON et la valide.
func ParseConfig(b []byte) (*Config, error) {
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}
