.. _format_gsl:

Blocs GSL (``gsl``)
===================

Principe
--------

Le format GSL est un suisse à 2 vies dont **l'appariement est figé par blocs** : au lieu d'apparier
au fil de l'eau (manipulable par l'heure d'annonce d'un résultat), on constitue des groupes de 4 (ou
3, ou 2) joueurs et on joue le bloc entier avant d'en constituer un nouveau.

- Les joueurs à **0 défaite** sont répartis en groupes GSL (double élimination interne à 4 :
  ouverture, gagnants, perdants, décisif ; section de genre ``gsl``).
- Les joueurs à **1 défaite** (une seule vie) sont répartis en mini-tableaux à élimination directe
  (section de genre ``se``).
- ``Lives`` est forcé à 2 par ``Config.Validate`` ; ``Target``, s'il est donné, doit être une
  puissance de 2.

C'est la parade anti-manipulation recommandée par l'étude (:ref:`annexe_etude`), et elle a
l'avantage de se caler naturellement sur les créneaux horaires (un bloc par créneau, repas compris).

La règle est dans ``rules`` (``rules/graphes_phase.go`` : ``Pair``, ``Partition``,
``GSLBlockSections``) ; ``phase_gsl.go`` en est l'enveloppe (``proposeGSL``, ``applyGSLDraw``).

Partition d'une liste en groupes
--------------------------------

.. code:: go

   func Partition(g []Ref) [][]Ref   // paquet rules

La liste (**déjà mélangée** par l'appelant) est découpée en groupes de 4 autant que possible, le
reste étant absorbé selon ``n mod 4`` :

=========== ============================================
``n mod 4`` Tailles produites
=========== ============================================
0           ``4, 4, …, 4``
3           ``4, …, 4, 3``
2           ``4, …, 4, 3, 3`` si ``n ≥ 4`` ; sinon ``2``
1           ``4, …, 4, 3, 2`` si ``n ≥ 4`` ; sinon ``1``
=========== ============================================

Les groupes sont ensuite découpés dans l'ordre de la liste (``g[k:k+taille]``).

Un groupe de taille 1 (``n == 1``) n'est pas transformé en section : le joueur passe le bloc sans
jouer, sans perdre de vie ni gagner de bye. Le groupe garde néanmoins son numéro.

Fin de bloc, fin de phase
-------------------------

.. code:: go

   func (v *vue) gslBlockDone() bool   // paquet rules

Toutes les sections dont ``Block == Round`` (le bloc courant) sont terminées (``Section.Done``).

.. code:: go

   func (v *vue) gslDone() bool        // paquet rules ; rules.PhaseDone pour le format gsl

::

   si Running > 0 ou non gslBlockDone   → faux
   si Target > 0 et Σ vies restantes <= Target → vrai
   → len(alive) <= 1

Proposition
-----------

.. code:: go

   func (v *vue) pairGSL(seed Seed) []Pairing   // paquet rules, appelé par rules.Pair

::

   si gslDone → nil

   si le bloc courant n'est pas terminé :
       → pour chaque match de readyFree(Sections, Busy) dont la section a Block == Round :
             proposition match (action start_match)
                 Section = nom de la section, Key = clé du nœud
                 Label   = in_block{N: bloc, Section: nom de la section}, Sub = libellé du nœud
                 Length  = longueur du nœud

   si Running > 0 → nil        ← finale ou matchs hors bloc en cours

   nouveau bloc :
       rng := générateur(seed.Draw())
       g0 := joueurs en vie à 0 défaite
       g1 := joueurs en vie à 1 défaite ou plus

       si len(g0) == 1 et len(g1) == 1 :
           → proposition match, Label = final, sans section ni clé, Length = longueur de la phase
       sinon :
           groupes := []
           pour g dans [g0, g1] :
               g := tri par identifiant puis rng.Shuffle(g)
               groupes += Partition(g)
           → proposition draw, Label = draw_block{N: Round+1, Players: nombre de groupes},
                               Groups = groupes

L'enveloppe (``State.actions``) ajoute au tirage ``Draw.Lives`` : les vies d'entrée des joueurs
de la phase, à titre d'information.

**La finale GSL est un match libre** (sans section) opposant le dernier joueur invaincu au dernier
joueur à une défaite. Si l'invaincu gagne, l'autre atteint 2 défaites et la phase se termine. S'il
perd, les deux joueurs ont une défaite chacun, ``alive`` vaut toujours 2, et le moteur repropose une
finale : c'est une **recharge implicite** exactement équivalente à celle de la double élimination.
C'est la raison pour laquelle un GSL à 64 joueurs produit 126 **ou** 127 matchs.

Application du tirage
---------------------

.. code:: go

   func (s *State) applyGSLDraw(ph *PhaseState, d *Draw) error
   func GSLBlockSections(block int, groups [][]Ref, losses map[Ref]int, length int) []Section  // rules

::

   ph.Round++ ; ph.Drawn = true
   ph.Sections += GSLBlockSections(ph.Round, d.Groups, défaites de la phase, ph.Length) :
       pour chaque groupe g d'indice i :
           si len(g) < 2 → passer
           nom := "B<bloc>G<i+1>"
           si losses[g[0]] == 0 → sec := GSLSection(nom, g, length)   (genre gsl)
           sinon                → sec := SESection(nom, g, length)    (genre se)
           sec.Block, sec.Group = bloc, i+1
   ph.resolve(s.Withdrawn)

Le type de section est déterminé par le **premier joueur du groupe** : c'est correct puisque la
partition est faite séparément sur ``g0`` et ``g1``, un groupe est donc homogène en nombre de
défaites.

Les libellés des nœuds sont des codes : ``opening``, ``winners_match``, ``losers_match``,
``decider`` dans un groupe de 4 ; ``opening``, ``winners_match``, ``decider`` dans un groupe de 3 ;
``single_match`` dans un groupe de 2 ; ``single_elimination`` dans un mini-tableau.

Le classement d'une phase GSL est celui d'une phase à vies (``rules.Rank``, comme le suisse, voir
:ref:`format_suisse`).
