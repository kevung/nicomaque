.. _format_tableaux:

Tableaux (``bracket``, ``lives_bracket``)
=========================================

Les deux formats partagent tout le code ; ils ne diffèrent que par les vies à l'entrée, donc par le
tirage. La règle est dans ``rules`` (``rules/graphes_phase.go`` : ``DrawSlots``, ``SeedOrder``,
``BracketSections``, ``Pair`` ; ``rules/classement.go`` : ``Rank``, ``RankSection``) ;
``phase_bracket.go`` en est l'enveloppe.

- ``bracket`` : tous les entrants ont 1 vie ; le tableau est un tableau classique complété par
  des exemptions.
- ``lives_bracket`` : les entrants arrivent avec les vies héritées de la phase précédente ; un
  joueur à 2 vies occupe une paire entière et est donc **exempt du premier tour**.

Utilitaire
----------

.. code:: go

   func pow2ceil(n int) int   // paquet rules : la plus petite puissance de 2 >= n (1 pour n <= 1)

Tirage des places
-----------------

.. code:: go

   func DrawSlots(entrants []Ref, lives map[Ref]int, seeding string, ratings map[Ref]float64, seed int64) []Ref

Le générateur est construit sur ``seed`` (``Seed.Draw()`` : graine du tournoi, nombre d'événements,
phase).

::

   ids := tri par identifiant puis rng.Shuffle(ids)

   deux := les ids avec lives >= 2   (comptent 2 dans sum)
   une  := les autres                (comptent 1 dans sum)
   size := max(2, pow2ceil(sum))
   si seeding == "rating" → seededSlots(deux, une, ratings, size)   (« Têtes de série »)
   nPairs := size / 2

   ordre := indices pairs croissants (0, 2, 4, …) suivis des indices impairs (1, 3, 5, …)
   k := 0
   pour chaque p de `deux` :
       pairs[ordre[k]] = (p, BYE) ; marquée occupée ; k++

   extra := size - sum          ← nombre d'exemptions supplémentaires à distribuer
   rest  := `une`
   pour chaque paire i non occupée, dans l'ordre croissant :
       a, b := BYE, BYE
       si rest non vide → a := dépiler(rest)
       si extra > 0 et a != BYE → extra--            ← cette paire reçoit une exemption
       sinon si rest non vide   → b := dépiler(rest)
       pairs[i] = (a, b)

   slots := aplatissement des paires : (p0₀, p0₁, p1₀, p1₁, …)

Trois idées :

1. **La taille du tableau est dictée par la somme des vies**, pas par le nombre de joueurs : c'est
   ce qui rend la bascule depuis un suisse exacte quand Σ vies est une puissance de 2.
2. **Les exemptés sont écartés** : en remplissant d'abord une paire sur deux (indices pairs), on
   évite que deux joueurs à 2 vies se retrouvent dans la même moitié de bracket de premier tour.
3. **Les exemptions surnuméraires** (``extra``) vont en seconde position des premières paires
   libres, donc aux joueurs placés en tête de la liste mélangée — c'est-à-dire au hasard.

Têtes de série
~~~~~~~~~~~~~~

Le tirage est aléatoire par défaut. Avec ``seeding: "rating"`` (``PhaseConfig.Seeding``, seule
valeur admise par ``Config.Validate`` hormis le vide, et réservée aux tableaux), le hasard cède la
place au placement classique par cote :

.. code:: go

   func SeedOrder(size int) []int   // paquet rules

::

   order := [1]
   pour n := 1 ; n < size ; n *= 2 :
       chaque place x de order devient les places x et 2n+1-x
   (8 places : 1, 8, 4, 5, 2, 7, 3, 6)

   seededSlots(deux, une, ratings, size) :
       ordre := byStrength(deux) puis byStrength(une)
                ← les joueurs à 2 vies passent devant tous les autres : les exemptions
                  du premier tour vont aux premières têtes
       slots := size places BYE
       pour chaque place, tête := SeedOrder(size)[place] :
           si tête <= len(ordre) → slots[place] := ordre[tête-1]

``byStrength`` trie du meilleur au moins bon par ``Player.Rating`` (le PR : plus bas vaut mieux) ;
une cote nulle ou négative est inconnue et passe derrière toutes les autres ; l'ordre de départ
(et donc entre cotes égales ou inconnues) est celui des identifiants. Les places sans tête de série
sont des exemptions (``BYE``), qui reviennent ainsi aux meilleures têtes.

Construction du tableau
-----------------------

.. code:: go

   func (s *State) buildBracket(ph *PhaseState, slots []PlayerID)
   func BracketSections(slots []Ref, length int, r PhaseRules) []Section   // paquet rules

::

   lp   := LengthPlan{Default: length, Final: FinalLength, ByRound: Lengths}
   main := BracketSection("main", "main", slots, lp)
   Sections := [main]

   si Consolation et len(main.Rounds) >= 2 :
       conso := ConsolationSection("conso", main, length)
       Sections += conso

       si LastChance et len(conso.Rounds) >= 3 :
           srcs := pour r de 0 à len(conso.Rounds)-3 :
                       les perdants de tous les matchs de conso.Rounds[r]
           Sections += LastChanceSection("last", srcs, length)

       si Reconciliation :
           gf := section "gf" (genre gf)
           gf.0  grand_final : vainqueur du dernier match de main
                               contre vainqueur du dernier match de conso
                 longueur = lp.At(0, 1) — la longueur de finale
           si Recharge :
           gf.1  grand_final_recharge : vainqueur de gf.0 contre perdant de gf.0
                     même longueur ; Cond = vrai, CondFrom = 0, CondSide = 1
           Rounds = [[0]] (+ [[1]] si recharge)
           Sections += gf

   ph.Drawn = true
   ph.resolve(s.Withdrawn)

``length`` est la longueur courante de la phase (``ph.Length``) : consolante et dernière chance
n'ont pas de plan de longueurs par tour. ``Config.Validate`` refuse ``recharge`` sans
``reconciliation`` et ``reconciliation`` sans ``consolation``.

La **dernière chance** ne recueille que les perdants des tours de consolante **sauf les deux
derniers** : perdre en finale ou en demi-finale de consolante ne donne pas droit à un troisième
tableau.

La **recharge** est le mécanisme de la vraie double élimination : ``CondSide = 1`` signifie que le
match n'existe que si le vainqueur de la grande finale est le joueur venu de la place 1,
c'est-à-dire le vainqueur de la consolante (qui a déjà une défaite). Le vainqueur du principal,
invaincu, doit donc être battu deux fois. Si c'est lui qui gagne la première grande finale,
``resolve`` marque ``gf.1`` comme ``Skipped`` (donc ``Done``) et le tournoi se termine.

Application du tirage
---------------------

.. code:: go

   func (s *State) applyDraw(ph *PhaseState, section string, d *Draw) error

Pour ``bracket`` et ``lives_bracket`` :

::

   si len(d.Slots) < 2 ou len(d.Slots) n'est pas une puissance de 2
       → erreur « le nombre de places doit être une puissance de 2 »
   buildBracket(ph, d.Slots)

Proposition
-----------

.. code:: go

   func (s *State) proposeBracket(ph *PhaseState) []Action   // enveloppe de rules.Pair

::

   si non Drawn :
       si len(Entrants) < 2 → nil
       slots := DrawSlots(Entrants, vies, Seeding, cotes, seed.Draw())
       → une action draw
           Label = draw_bracket{N: len(slots)}, Sub = phase{Text: nom de la phase}
           Draw  = { Slots: slots, Lives: vies des entrants }

   sinon :
       pour chaque match de readyFree(Sections, Busy) :
           Label := libellé du nœud
           si la section est de genre "main" et qu'il existe plus d'une section
                → Label := main_draw, Sub = libellé du nœud
           → action start_match { Section, Key, Label, A, B, Length du nœud }

Les libellés des nœuds sont des codes : ``final``, ``semi_final``, ``quarter_final``, puis
``bracket_round{N}`` au principal ; ``consolation_round{N}`` et ``consolation_final`` en
consolante ; ``last_chance`` (avec le libellé du tour en ``Sub``) en dernière chance ;
``grand_final`` et ``grand_final_recharge`` en grande finale.

Toutes les sections sont proposées en parallèle : consolante, dernière chance et principal avancent
simultanément dès que les places sont connues et les joueurs libres.

Classement d'un tableau
-----------------------

.. code:: go

   func Rank(p Phase) []Standing   // paquet rules ; rankBracket pour bracket et lives_bracket

Priorité des sections, par genre (plus haut = mieux classé) :

====================== ========
Section                ``prio``
====================== ========
``gf`` (grande finale) 4
``main``, ``se``       3
``conso``              2
``last``               1
====================== ========

::

   si non ph.Drawn :
       → tous les entrants au rang 1, note awaiting_draw

   initialiser score[p] = -1, note = unranked pour chaque entrant

   (1) SORTIES — pour chaque section, chaque tour r, chaque match g de ce tour :
       ignorer si Skipped
       si non Done : chaque joueur RETIRÉ placé dans g y prend sa sortie (il le perdra par forfait)
       ignorer si Walkover et que le perdant n'est pas retiré (exemption par BYE)
       candidat := prio[genre]*1000 + r         ← pour le PERDANT
       remplacer score[g.Loser] si :
           score actuel < 0
           OU prio[genre] < (score actuel)/1000     ← sortie en section moins prioritaire
           OU (prio identique ET candidat > score actuel)← sortie plus profonde
       note := section_exit{Section: nom de la section, Sub: libellé du match}

   (2) VAINQUEURS DE SECTION — pour chaque section non vide :
       last := dernier match de la section
       (pour "gf" : le dernier match Done et non Skipped, en partant de la fin)
       si last.Done, non Skipped, et Winner != BYE :
           candidat := prio[genre]*1000 + 999
           remplacer si score < 0, ou candidat > score, ou section == "gf"
           note := section_winner{Section: nom de la section}
           si section == "gf" : le perdant reçoit prio(gf)*1000+998, note finalist

   (3) RETIRÉS ET NON CLASSÉS :
       si retiré → note := withdrawn{Section, Sub de sa sortie s'il en a une}
                   (son score, lui, est celui de son parcours)
       si score < 0 :
           score := 0
           si la note est restée unranked → note running, score := 5000

   tri stable par score décroissant (ordre des entrants entre égaux), rangs avec ex æquo
   (même score → même rang, le rang suivant saute d'autant)

La règle (1) est contre-intuitive mais essentielle : **un joueur est classé par sa dernière
sortie**, c'est-à-dire par la section de plus faible priorité où il a perdu. Un joueur battu au
premier tour du principal puis en finale de consolante est classé sur sa performance en consolante,
pas sur son élimination du principal.

Un joueur **retiré** est classé comme s'il perdait chacun de ses matchs restants : le match qu'il
perd par forfait (walkover résolu contre lui) est une sortie comme une autre, et la règle (1)
retient la dernière. Il est donc **ex æquo avec les perdants du match qui l'attendait**. Un joueur
de ``lives_bracket`` qui gagne un match puis part est classé avec les perdants du tour de consolante
où il redescend, et non dernier ; une demi-finaliste qui part pendant sa demi-finale, dans un
tableau à consolante, est classée avec la perdante du tour de consolante qu'elle aurait joué — et
non devant la gagnante de la consolante, qui a joué deux matchs de plus. Tant que l'adversaire de ce
match n'est pas connu, le match qui l'attend tient lieu de sortie : le retiré n'est jamais « en
cours ». Retiré sans avoir joué, il est classé avec les perdants de son premier match.

Le score 5000 attribué aux joueurs en cours (``running``) les place **devant tout le monde** : c'est voulu,
puisque le classement intermédiaire doit montrer les joueurs encore en course en tête.

La note ``section_exit`` dit où un joueur est **sorti d'une section**, pas qu'il est éliminé : le
perdant du principal reversé en consolante la porte, avec la section ``main``, tant que sa
consolante n'est pas jouée ; il n'est pas non plus ``running`` au sens de la règle (3). C'est une
note de classement — elle situe le joueur — et non une réponse à « joue-t-il encore ? ». Cette
réponse est le statut (``State.Statuses``, voir :ref:`statut`), qui le dit en jeu, en
consolante.

Le classement PROPRE d'une section (``SectionRanking``, sur lequel se répartit sa dotation) est
décrit dans :ref:`classement`.

Survivants d'un tableau
-----------------------

.. code:: go

   func (s *State) bracketSurvivors(ph *PhaseState) []PlayerID

::

   si non ph.Drawn                → tous les entrants
   r := rules.Rank(instantané de la phase)
   si rules.AllDone(Sections)     → [ r[0].Player ]        ← le vainqueur seul
   sinon                          → tous les joueurs dont la note est running
