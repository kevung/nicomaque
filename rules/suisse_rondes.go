package rules

import (
	"math/rand"
	"sort"
)

// ---- Suisse classique : nombre de rondes fixe, sans vies ----
//
// Une variante NOMMÉE, à part du suisse à vies : personne n'est éliminé, la phase finit après
// Rules.Rounds rondes, et le classement est celui des points (victoires + exemptions). Sans
// départage, les ex æquo le restent — y compris en tête.

// swissRoundsBudget borne une recherche en profondeur ; au-delà, on passe à l'exempté ou au
// nombre de revanches suivant, pour qu'un grand champ ne bloque jamais.
const swissRoundsBudget = 200000

// points : victoires plus exemptions — une exemption vaut une victoire.
func (e Entrant) points() int { return e.Wins + e.Byes }

func (v *vue) roundsDone() bool {
	if v.Running > 0 {
		return false
	}
	return v.Round >= v.Rules.Rounds || len(v.alive()) <= 1
}

// maxCritères : au plus trois critères de départage, après les points.
const maxCritères = 3

// clé : ce qui ordonne un joueur — ses points, puis la valeur de chaque critère de départage
// configuré, dans l'ordre. Deux clés égales sont deux ex æquo.
type clé [1 + maxCritères]int

func (a clé) devant(b clé) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

// buchholz : la somme des points des adversaires rencontrés, une fois par rencontre. Une
// exemption n'apporte aucun adversaire.
func (v *vue) buchholz(e Entrant) int {
	n := 0
	for _, o := range e.Played {
		n += v.entrant(o).points()
	}
	return n
}

// TiebreakValue : la valeur du critère de départage c pour l'entrant e de la phase p.
func TiebreakValue(p Phase, e Entrant, c string) int { return voir(&p).critère(e, c) }

func (v *vue) critère(e Entrant, c string) int {
	switch c {
	case TiebreakPointDiff:
		return e.PointsFor - e.PointsAgainst
	case TiebreakPointsScored:
		return e.PointsFor
	case TiebreakBuchholz:
		return v.buchholz(e)
	}
	return 0
}

func (v *vue) clé(p Ref) clé {
	e := v.entrant(p)
	k := clé{e.points()}
	for i, c := range v.Rules.Tiebreak {
		if i >= maxCritères {
			break
		}
		k[i+1] = v.critère(e, c)
	}
	return k
}

// ordreClassement : les joueurs par points décroissants, puis par départage s'il y en a un. À
// égalité complète l'ordre est tiré au sort (après le tri par identifiant) : rien ne distingue
// deux ex æquo.
func (v *vue) ordreClassement(players []Ref, rng *rand.Rand) []Ref {
	g := sortedRefs(players)
	rng.Shuffle(len(g), func(i, j int) { g[i], g[j] = g[j], g[i] })
	if v.Rules.Seeding != "" {
		g = v.têtes(players) // des têtes de série : à égalité complète, la meilleure tête d'abord
	}
	clés := make(map[Ref]clé, len(g))
	for _, p := range g {
		clés[p] = v.clé(p)
	}
	sort.SliceStable(g, func(i, j int) bool { return clés[g[i]].devant(clés[g[j]]) })
	return g
}

// pairClassicRound : la ronde r du suisse classique. Dans l'ordre : le moins de revanches
// possible ; l'exemption au plus bas classé parmi ceux qui en ont eu le moins (jamais une
// seconde tant qu'un joueur n'en a pas eu) ; chacun contre le
// plus proche encore libre. Un glouton ne suffit pas : il apparie proprement la tête et accule
// les derniers à une revanche qu'un autre choix plus haut évitait.
func (v *vue) pairClassicRound(r int, players []Ref, rng *rand.Rand) []Pairing {
	if len(players) == 0 || r > v.Rules.Rounds {
		return nil
	}
	lbl := Label{Kind: LabelRound, N: r}
	var pairs [][2]Ref
	var bye Ref
	if r == 1 && v.Rules.Seeding != "" {
		pairs, bye = plier(v.têtes(players))
	} else if v.Rules.Pairing == PairingMonrad {
		pairs, bye = v.pairMonrad(players, rng)
	} else {
		ordre := v.ordreClassement(players, rng)
		pairs, bye = v.pairNearest(ordre, len(ordre)%2 == 1)
	}
	acts := make([]Pairing, 0, len(pairs)+1)
	for _, pr := range pairs {
		acts = append(acts, Pairing{Kind: PairMatch, Label: lbl, Round: r, A: pr[0], B: pr[1], Length: v.Length})
	}
	if bye != "" {
		acts = append(acts, Pairing{Kind: PairBye, Label: lbl, Round: r, A: bye})
	}
	return acts
}

// seeds : la tête de série saisie de chaque entrant.
func (v *vue) seeds() map[Ref]int {
	out := make(map[Ref]int, len(v.Entrants))
	for _, e := range v.Entrants {
		out[e.Player] = e.Seed
	}
	return out
}

// têtes : les joueurs dans l'ordre des têtes de série — saisies (SeedingManual) ou par cote
// (SeedingRating) —, ceux qui n'en ont pas derrière, dans l'ordre des entrants.
func (v *vue) têtes(players []Ref) []Ref {
	dedans := refSet(players)
	var ordre []Ref
	for _, e := range v.Entrants {
		if dedans[e.Player] {
			ordre = append(ordre, e.Player)
		}
	}
	if v.Rules.Seeding == SeedingRating {
		ratings := make(map[Ref]float64, len(v.Entrants))
		for _, e := range v.Entrants {
			ratings[e.Player] = e.Rating
		}
		return byStrength(ordre, ratings)
	}
	return bySeed(ordre, v.seeds())
}

// plier : la ronde 1 d'un suisse à têtes de série — la moitié haute contre la moitié basse, la
// tête 1 contre la tête n/2 + 1. Effectif impair : la dernière tête est exemptée.
func plier(ordre []Ref) (pairs [][2]Ref, bye Ref) {
	n := len(ordre) / 2
	for i := 0; i < n; i++ {
		pairs = append(pairs, [2]Ref{ordre[i], ordre[i+n]})
	}
	if len(ordre)%2 == 1 {
		bye = ordre[len(ordre)-1]
	}
	return pairs, bye
}

// pairMonrad : le Monrad. L'adversaire est TIRÉ AU SORT dans le groupe de mêmes points, au lieu
// d'être le plus proche au classement — le départage, s'il y en a un, n'y joue aucun rôle. Les
// groupes sont traités du meilleur au moins bon, sans aucune revanche à l'intérieur d'un groupe.
// Groupe impair : un joueur flotte vers le groupe suivant (on essaie du dernier tiré au premier).
// Groupe impossible à apparier sans revanche : il rejoint entier le suivant. Le dernier groupe
// s'apparie comme un suisse (pairNearest) : le moins de revanches, l'exemption en bas.
//
// Avec Rules.MonradWiden, si le dernier groupe ne s'apparie qu'au prix d'une revanche, on
// recommence en lui adjoignant le groupe d'au-dessus, puis le suivant : une revanche qu'un groupe
// plus large évite n'est pas jouée. Tous les groupes réunis, c'est l'appariement du suisse, au
// minimum de revanches. Sans l'option, la revanche du dernier groupe est jouée.
func (v *vue) pairMonrad(players []Ref, rng *rand.Rand) (pairs [][2]Ref, bye Ref) {
	g := sortedRefs(players)
	rng.Shuffle(len(g), func(i, j int) { g[i], g[j] = g[j], g[i] })
	sort.SliceStable(g, func(i, j int) bool { return v.entrant(g[i]).points() > v.entrant(g[j]).points() })
	var bornes []int // début de chaque groupe de points
	for i := range g {
		if i == 0 || v.entrant(g[i]).points() != v.entrant(g[i-1]).points() {
			bornes = append(bornes, i)
		}
	}
	if !v.Rules.MonradWiden {
		return v.monradAvecQueue(g, bornes)
	}
	moins := 1 << 30
	for queue := 1; queue <= len(bornes); queue++ {
		got, exempté := v.monradAvecQueue(g, bornes[:len(bornes)-queue+1])
		n := 0
		for _, pr := range got {
			if v.met(pr[0], pr[1]) {
				n++
			}
		}
		if n < moins {
			pairs, bye, moins = got, exempté, n
		}
		if n == 0 {
			break
		}
	}
	return pairs, bye
}

// monradAvecQueue : les groupes qui commencent aux bornes données, le dernier allant jusqu'au
// bout de g et s'appariant comme un suisse.
func (v *vue) monradAvecQueue(g []Ref, bornes []int) (pairs [][2]Ref, bye Ref) {
	var report []Ref // flotteur, ou groupe entier reporté
	for b, i := range bornes {
		if b == len(bornes)-1 { // le dernier groupe
			groupe := append(append([]Ref{}, report...), g[i:]...)
			reste, exempté := v.pairNearest(groupe, len(groupe)%2 == 1)
			return append(pairs, reste...), exempté
		}
		groupe := append(append([]Ref{}, report...), g[i:bornes[b+1]]...)
		report = nil
		if len(groupe)%2 == 0 {
			if got, ok := v.sansRevanche(groupe); ok {
				pairs = append(pairs, got...)
			} else {
				report = groupe
			}
			continue
		}
		report = groupe
		for k := len(groupe) - 1; k >= 0; k-- {
			reste := append(append([]Ref{}, groupe[:k]...), groupe[k+1:]...)
			if got, ok := v.sansRevanche(reste); ok {
				pairs, report = append(pairs, got...), []Ref{groupe[k]}
				break
			}
		}
	}
	return pairs, ""
}

// sansRevanche apparie un groupe sans aucune revanche, les rencontres de club évitées quand
// c'est possible (avoid_clubs).
func (v *vue) sansRevanche(groupe []Ref) ([][2]Ref, bool) {
	passes := []bool{false}
	if v.Rules.AvoidClubs {
		passes = []bool{true, false}
	}
	for _, sansClub := range passes {
		budget := swissRoundsBudget
		if got, ok := v.chercher(groupe, 0, sansClub, &budget); ok {
			return got, true
		}
	}
	return nil, false
}

// pairNearest apparie ordre (déjà classé). exempter : un joueur est mis de côté, que l'appelant
// exempte ou laisse attendre.
func (v *vue) pairNearest(ordre []Ref, exempter bool) (pairs [][2]Ref, bye Ref) {
	candidats := []Ref{""}
	if exempter {
		// Seuls ceux qui ont eu le moins d'exemptions sont candidats, du bas du classement vers
		// le haut : mieux vaut une revanche qu'une seconde exemption au même joueur.
		moins := 1 << 30
		for _, p := range ordre {
			moins = min(moins, v.entrant(p).Byes)
		}
		candidats = candidats[:0]
		for i := len(ordre) - 1; i >= 0; i-- {
			if v.entrant(ordre[i]).Byes == moins {
				candidats = append(candidats, ordre[i])
			}
		}
	}
	// Le club est une préférence : on l'essaie d'abord, à nombre de revanches égal, et on y
	// renonce plutôt que de créer une revanche.
	passes := []bool{false}
	if v.Rules.AvoidClubs {
		passes = []bool{true, false}
	}
	for limite := 0; limite <= len(ordre)/2; limite++ {
		for _, sansClub := range passes {
			for _, c := range candidats {
				reste := make([]Ref, 0, len(ordre))
				for _, p := range ordre {
					if p != c {
						reste = append(reste, p)
					}
				}
				budget := swissRoundsBudget
				if got, ok := v.chercher(reste, limite, sansClub, &budget); ok {
					return got, c
				}
			}
		}
	}
	return nil, "" // inatteignable : à limite n/2, la première branche réussit
}

// chercher apparie reste, dans l'ordre, avec au plus limite revanches, en profondeur d'abord.
// sansClub : aucune paire de même club.
func (v *vue) chercher(reste []Ref, limite int, sansClub bool, budget *int) ([][2]Ref, bool) {
	n := len(reste)
	vus, exclus := make([][]bool, n), make([][]bool, n)
	for i := range vus {
		vus[i], exclus[i] = make([]bool, n), make([]bool, n)
		for j := range vus[i] {
			vus[i][j] = v.met(reste[i], reste[j])
			exclus[i][j] = sansClub && v.sameClub(reste[i], reste[j])
		}
	}
	pris := make([]bool, n)
	out := make([][2]Ref, 0, n/2)
	var rec func(dépensé int) bool
	rec = func(dépensé int) bool {
		i := 0
		for i < n && pris[i] {
			i++
		}
		if i == n {
			return true
		}
		pris[i] = true
		for j := i + 1; j < n; j++ {
			if pris[j] || exclus[i][j] {
				continue
			}
			if *budget--; *budget < 0 {
				break
			}
			coût := 0
			if vus[i][j] {
				coût = 1
			}
			if dépensé+coût > limite {
				continue
			}
			pris[j] = true
			out = append(out, [2]Ref{reste[i], reste[j]})
			if rec(dépensé + coût) {
				return true
			}
			out = out[:len(out)-1]
			pris[j] = false
		}
		pris[i] = false
		return false
	}
	if !rec(0) {
		return nil, false
	}
	return out, true
}

// rankRounds : le classement d'un suisse classique — par points, puis par départage s'il y en a
// un ; les ex æquo qui résistent le restent. Un retiré est classé sur son parcours, avec la note
// NoteWithdrawn.
func (v *vue) rankRounds() []Standing {
	ps := make([]Ref, len(v.Entrants))
	pts := map[Ref]clé{}
	for i, e := range v.Entrants {
		ps[i] = e.Player
		pts[e.Player] = v.clé(e.Player)
	}
	sort.SliceStable(ps, func(i, j int) bool { return pts[ps[i]].devant(pts[ps[j]]) })
	scores, notes := make([]clé, len(ps)), make([]Note, len(ps))
	for i, p := range ps {
		e := v.entrant(p)
		scores[i] = pts[p]
		notes[i] = Note{Kind: NoteRecord, Wins: e.Wins, Losses: e.Losses, Byes: e.Byes}
		for _, c := range v.Rules.Tiebreak {
			switch c {
			case TiebreakPointDiff:
				notes[i].PointDiff = v.critère(e, c)
			case TiebreakPointsScored:
				notes[i].PointsScored = v.critère(e, c)
			case TiebreakBuchholz:
				notes[i].Buchholz = v.critère(e, c)
			}
		}
		if v.withdrawn[p] {
			notes[i].Kind = NoteWithdrawn
		}
	}
	return rangs(ps, scores, notes)
}

// resteRondes : les matchs encore à jouer d'un suisse classique — ceux en cours, plus une ronde
// complète des joueurs en lice par ronde à venir. Min suppose la ronde ouverte entièrement lancée.
func (v *vue) resteRondes() Remaining {
	n := len(v.alive()) / 2
	àVenir := max(v.Rules.Rounds-v.Round, 0) * n
	hi := àVenir + v.Running
	if v.Round > 0 && v.Round <= v.Rules.Rounds {
		hi = àVenir + max(v.Running, n) // la ronde ouverte peut ne pas être entièrement lancée
	}
	return Remaining{àVenir + v.Running, hi}
}
