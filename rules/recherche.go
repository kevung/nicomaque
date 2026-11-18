package rules

// Search compte les pas de la recherche d'appariement du suisse par rondes, et la borne quand
// l'appelant le demande.
//
// Un pas est une paire examinée : une case des tables de rencontres et de clubs qu'une tentative
// construit, ou une paire essayée par la recherche en profondeur. Le compte ne dépend que de
// l'instantané et de la graine : la même entrée dépense les mêmes pas sur toute machine, si bien
// qu'un budget épuisé est un verdict reproductible, pas une affaire d'horloge.
//
// Le compteur est à l'appelant, qui en choisit la portée : un compteur neuf par proposition
// borne un appariement, un compteur partagé borne une suite de propositions (une simulation
// entière). Épuisé, il le reste : les tentatives suivantes échouent sans rien examiner, et
// l'appariement obtenu n'a aucun sens — l'appelant le jette (voir Exhausted).
//
// Ce budget est distinct de swissRoundsBudget, qui fait partie de la règle : une tentative qui
// dépasse ce plafond est tenue pour infructueuse et l'appariement passe au candidat suivant,
// sans erreur. Search ne change aucun appariement ; il dit seulement quand l'appelant n'a pas
// voulu attendre la fin de la recherche.
type Search struct {
	// Limit : les pas accordés. 0 ou moins : sans limite, le compteur ne fait que compter.
	Limit int64
	// Used : les pas dépensés, cumulés d'un appel à l'autre.
	Used int64
}

// Exhausted : la recherche a dépassé le budget accordé. Faux sur un compteur nil.
func (s *Search) Exhausted() bool { return s != nil && s.Limit > 0 && s.Used > s.Limit }

// spend dépense n pas ; faux si le budget est épuisé. Toujours vrai sur un compteur nil.
func (s *Search) spend(n int) bool {
	if s == nil {
		return true
	}
	s.Used += int64(n)
	return !s.Exhausted()
}
