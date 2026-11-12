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

// ordreClassement : les joueurs par points décroissants. À points égaux l'ordre est tiré au
// sort (après le tri par identifiant) : sans départage, rien ne distingue deux ex æquo.
func (v *vue) ordreClassement(players []Ref, rng *rand.Rand) []Ref {
	g := sortedRefs(players)
	rng.Shuffle(len(g), func(i, j int) { g[i], g[j] = g[j], g[i] })
	sort.SliceStable(g, func(i, j int) bool { return v.entrant(g[i]).points() > v.entrant(g[j]).points() })
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
	ordre := v.ordreClassement(players, rng)
	lbl := Label{Kind: LabelRound, N: r}
	pairs, bye := v.pairNearest(ordre, len(ordre)%2 == 1)
	acts := make([]Pairing, 0, len(pairs)+1)
	for _, pr := range pairs {
		acts = append(acts, Pairing{Kind: PairMatch, Label: lbl, Round: r, A: pr[0], B: pr[1], Length: v.Length})
	}
	if bye != "" {
		acts = append(acts, Pairing{Kind: PairBye, Label: lbl, Round: r, A: bye})
	}
	return acts
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

// rankRounds : le classement d'un suisse classique — par points, ex æquo conservés. Un retiré
// est classé sur son parcours, avec la note NoteWithdrawn.
func (v *vue) rankRounds() []Standing {
	ps := make([]Ref, len(v.Entrants))
	pts := map[Ref]int{}
	for i, e := range v.Entrants {
		ps[i] = e.Player
		pts[e.Player] = e.points()
	}
	sort.SliceStable(ps, func(i, j int) bool { return pts[ps[i]] > pts[ps[j]] })
	scores, notes := make([]int, len(ps)), make([]Note, len(ps))
	for i, p := range ps {
		e := v.entrant(p)
		scores[i] = pts[p]
		notes[i] = Note{Kind: NoteRecord, Wins: e.Wins, Losses: e.Losses, Byes: e.Byes}
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
