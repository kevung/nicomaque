.. _graphes:

Graphes de matchs
=================

Ce chapitre spécifie ``rules/graphe.go``, commun à tous les formats à graphe (tableaux, consolantes,
dernière chance, grande finale, groupes GSL, poules). Les règles y sont des fonctions pures sur les
types de ``rules`` (``Section``, ``Node``, ``Src``) ; le moteur garde ses propres types
(``Section``, ``GMatch``, ``Src`` de ``state.go``, sérialisés dans l'état) et les convertit dans
``regles.go`` (``toRulesSections``, ``fromRulesSections``).

Types
-----

.. code:: go

   // rules/graphe.go
   const Bye Ref = "BYE" // place vide (exemption)

   type Src struct {
       Player  Ref    `json:"player,omitempty"`  // joueur fixé…
       From    int    `json:"from"`              // …ou indice du nœud d'origine ; -1 pour un joueur fixé
       Section string `json:"section,omitempty"` // section du nœud d'origine ; vide = la même
       Loser   bool   `json:"loser,omitempty"`   // le perdant plutôt que le vainqueur
   }

   type Node struct {
       Key      string `json:"key"`
       Label    Label  `json:"label"`
       Length   int    `json:"length"`
       Src      [2]Src `json:"src"`
       Players  [2]Ref `json:"players"`             // état résolu
       Match    string `json:"match,omitempty"`     // identifiant du match lancé ; vide = pas lancé
       Winner   Ref    `json:"winner,omitempty"`
       Loser    Ref    `json:"loser,omitempty"`
       Done     bool   `json:"done,omitempty"`
       Walkover bool   `json:"walkover,omitempty"`  // terminé sans être joué : exemption ou forfait d'un retiré
       Skipped  bool   `json:"skipped,omitempty"`   // nœud conditionnel non joué (recharge)
       Cond     bool   `json:"cond,omitempty"`      // nœud conditionnel
       CondFrom int    `json:"cond_from,omitempty"` // il existe si le vainqueur de CondFrom…
       CondSide int    `json:"cond_side,omitempty"` // …est le joueur CondSide de ce nœud-là
   }

   type Section struct {
       Name    string  `json:"name"`              // identifiant : main, conso, last, gf, B1G2, poule:A, barrage:A
       Kind    string  `json:"kind"`              // main, conso, last, gf, gsl, se, poule, barrage
       Group   int     `json:"group,omitempty"`
       Block   int     `json:"block,omitempty"`
       Nodes   []Node  `json:"nodes,omitempty"`
       Rounds  [][]int `json:"rounds,omitempty"`  // indices de nœuds, tour par tour
       Players []Ref   `json:"players,omitempty"` // barrage : joueurs à départager
       Spots   int     `json:"spots,omitempty"`   // barrage : places à attribuer
   }

Le ``GMatch`` du moteur a les mêmes champs, à ceci près que l'identifiant du match lancé s'appelle
``MatchID`` (balise ``match_id``) et que ``Done`` est toujours sérialisé ; la ``Section`` du moteur
range ses nœuds dans ``Matches`` (balise ``matches``). Un barrage n'a pas de nœuds : ses joueurs
s'apparient au fil des résultats (voir :ref:`format_poules`).

Accès
-----

.. code:: go

   // graph.go (moteur)
   func (ph *PhaseState) gmatch(section, key string) *GMatch  // nil si introuvable ou key vide
   func (ph *PhaseState) section(name string) *Section        // nil si introuvable

Un ``Match`` joué est relié à son ``GMatch`` par ``(Section, Key)``.

Résolution
----------

.. code:: go

   func Resolve(secs []Section, withdrawn []Ref) []Section       // rules
   func (ph *PhaseState) resolve(withdrawn map[PlayerID]bool)    // enveloppe, regles.go

Fonction centrale : elle propage les résultats dans tous les graphes de la phase **jusqu'au point
fixe**. Elle est idempotente et peut être appelée aussi souvent qu'on veut. ``rules.Resolve``
travaille sur une copie des sections et la renvoie ; l'enveloppe du moteur passe les retirés triés
(``withdrawnRefs``) et recopie ``Players``, ``Winner``, ``Loser``, ``Done``, ``Walkover`` et
``Skipped`` dans ses ``GMatch``, en place.

::

   répéter tant que quelque chose a changé :
     pour chaque section, pour chaque nœud g de la section :

       (a) nœud conditionnel
           si g.Cond :
               from := section.Nodes[g.CondFrom]
               si from.Done et non g.Skipped et non g.Done :
                   si from.Winner != from.Players[g.CondSide] :
                       g.Skipped = true ; g.Done = true ; changé ; passer au nœud suivant

       (b) propagation des places, pour k = 0 puis 1 :
           src := g.Src[k]
           p := ""
           si src.Player != ""      → p = src.Player
           sinon si src.From >= 0 :
               sec := (src.Section == "") ? la section courante : la section nommée src.Section
               si sec existe, src.From est un indice valide, et sec.Nodes[src.From].Done :
                   from := sec.Nodes[src.From]
                   si from.Skipped → passer à la place suivante (place indéterminée)
                   p := from.Loser si src.Loser, sinon from.Winner
           si p != "" et g.Players[k] != p → g.Players[k] = p ; changé

       (c) forfait : nœud non lancé contre un joueur retiré
           si non g.Done, g.Match == "", les deux places connues et différentes de BYE,
              et au moins l'un des deux joueurs est retiré :
                 g.Done = true ; g.Walkover = true
                 g.Winner, g.Loser = Players[0], Players[1]
                 si Players[0] est retiré et Players[1] ne l'est pas → inverser
                 changé ; passer au nœud suivant

       (d) exemptions
           si non g.Done et les deux places connues :
               si les deux sont BYE  → Done, Walkover, Winner = BYE, Loser = BYE
               sinon si l'une est BYE → Done, Walkover, le joueur réel gagne, Loser = BYE

Points d'attention :

- l'ordre ``(a) (b) (c) (d)`` est significatif ;
- ``(c)`` ne s'applique **pas** si le match a déjà été lancé (``Match``/``MatchID`` non vide) : un
  match commencé se conclut par un résultat de forfait, pas par un walkover de graphe. C'est pourquoi
  ``recompute`` relie d'abord chaque place à son match, puis rejoue les résultats ;
- si les deux joueurs sont retirés, l'ordre du tableau tranche (``Players[0]`` gagne) ; le vainqueur
  fictif sera lui-même éliminé par walkover au tour suivant ;
- la propagation d'un ``BYE`` gagnant fait remonter ``BYE`` comme vainqueur d'une paire entièrement
  vide, ce qui vide toute une branche du tableau — comportement voulu.

Sélection des matchs lançables
------------------------------

.. code:: go

   type Slot struct {
       Section int `json:"section"`
       Node    int `json:"node"`
   }

   func Ready(secs []Section) []Slot

Tous les nœuds, dans l'ordre des sections puis des nœuds, tels que : non ``Done``, sans ``Match``,
deux places connues, aucune place à ``BYE``.

.. code:: go

   func ReadyFree(secs []Section, busy []Ref) []Slot

Filtre ``Ready`` : on écarte tout nœud dont un joueur est déjà occupé (``busy`` : les joueurs d'un
match en cours, ``Phase.Busy``) **ou déjà pris par un nœud précédent du même lot** — le premier nœud
qui prend un joueur l'emporte. Le résultat est donc un lot de matchs simultanément lançables, sans
conflit de joueur. Les joueurs occupés dans une autre épreuve ne sont pas écartés ici : leur match
reste proposé et le moteur le retient (``holdElsewhere``, voir :ref:`propositions`).

.. code:: go

   func (s Section) Done() bool      // tous les nœuds de la section sont Done
   func AllDone(secs []Section) bool // idem pour toutes les sections

Constructeurs de sections
-------------------------

Un constructeur ne fait que **poser des** ``Src`` ; il ne consulte jamais l'état. Ajouter un format
de tableau se réduit à écrire un constructeur. Les libellés sont des codes (``rules.Label``,
``rules/codes.go``), jamais des phrases : l'hôte les traduit.

Tableau à élimination simple
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code:: go

   func BracketSection(name, kind string, slots []Ref, lp LengthPlan) Section

La longueur des matchs vient d'un **plan** qui superpose trois sources, de la plus précise à la plus
générale — la liste par tour, la longueur de finale, la longueur par défaut :

.. code:: go

   type LengthPlan struct {
       Default int   `json:"default"`            // longueur courante de la phase (ph.Length)
       Final   int   `json:"final,omitempty"`    // Cfg.FinalLength ; 0 = Default
       ByRound []int `json:"by_round,omitempty"` // Cfg.Lengths, du DERNIER tour vers le premier
   }

::

   lp.At(r, rounds) :
       i := rounds - 1 - r
       si 0 <= i < len(ByRound) et ByRound[i] > 0 → ByRound[i]
       si r == rounds-1 et Final > 0              → Final
       sinon                                      → Default

``LengthPlan{Default: l}`` est le plan d'une longueur unique (mini-tableaux, dernière chance) ;
``BracketSections`` construit celui du tableau principal à partir de ``ph.Length``,
``final_length`` et ``lengths``.

``slots`` a une longueur ``size`` puissance de 2. ``rounds = log2(size)``.

::

   Tour 0 : pour i de 0 à size/2 - 1
       Key    = "<name>.0.<i>"
       Label  = RoundLabel(0, rounds)
       Length = lp.At(0, rounds)
       Src    = { {Player: slots[2i],   From: -1},
                  {Player: slots[2i+1], From: -1} }
   Rounds[0] = les indices de ces nœuds

   Tour r, de 1 à rounds-1 : pour i de 0 à len(Rounds[r-1])/2 - 1
       Key    = "<name>.<r>.<i>"
       Label  = RoundLabel(r, rounds)
       Length = lp.At(r, rounds)
       Src    = { {From: Rounds[r-1][2i]}, {From: Rounds[r-1][2i+1]} }

Libellés de tour :

.. code:: go

   func RoundLabel(r, rounds int) Label

============== ==========================================
``rounds - r`` Libellé
============== ==========================================
1              ``{kind: final}``
2              ``{kind: semi_final}``
3              ``{kind: quarter_final}``
autre          ``{kind: bracket_round, n: r+1}``
============== ==========================================

Le dernier nœud de la section (``Nodes[len-1]``) est toujours la finale : c'est lui qui désigne
le vainqueur de la section.

Sections d'un tableau tiré
~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code:: go

   func BracketSections(slots []Ref, length int, r PhaseRules) []Section

::

   main := BracketSection("main", "main", slots, {length, FinalLength, Lengths})
   si Consolation et len(main.Rounds) >= 2 :
       conso := ConsolationSection("conso", main, length)
       si LastChance et len(conso.Rounds) >= 3 :
           srcs := perdants des nœuds de conso.Rounds[0 .. len-3]   ← toutes sauf les deux dernières rondes
           LastChanceSection("last", srcs, length)
       si Reconciliation :
           section "gf" : nœud "gf.0", {kind: grand_final}, longueur lp.At(0, 1),
                          Src = { vainqueur du dernier nœud de main, vainqueur du dernier nœud de conso }
           si Recharge : nœud "gf.1", {kind: grand_final_recharge}, même longueur,
                          Src = { vainqueur de gf.0, perdant de gf.0 },
                          Cond = vrai, CondFrom = 0, CondSide = 1
                          ← joué seulement si le vainqueur de la consolante a gagné gf.0

Le détail des formats de tableau est dans :ref:`format_tableaux`.

Consolante progressive
~~~~~~~~~~~~~~~~~~~~~~

.. code:: go

   func ConsolationSection(name string, main Section, length int) Section

La consolante (genre ``conso``) recueille les perdants du tableau principal au fur et à mesure. Sa
structure alterne un **tour de fusion** (survivants de la consolante contre les nouveaux perdants du
principal) et un **tour interne** (survivants entre eux). Tous ses matchs ont la longueur ``length``.

::

   rounds := len(main.Rounds)
   si rounds < 2 → section vide

   Tour 0 : pour i = 0, 2, 4, … sur main.Rounds[0]
       Key   = "<name>.0.<i/2>" ; Label = {kind: consolation_round, n: 1}
       Src   = { perdant de main.Rounds[0][i], perdant de main.Rounds[0][i+1] }
   Rounds[0] = ces nœuds ; prev = ces nœuds ; cr = 1

   pour r de 1 à rounds-1 :
       drops := main.Rounds[r]
       fusion : pour i de 0 à len(prev)-1
           d := drops[len(drops)-1-i]           ← ordre INVERSÉ
           Key   = "<name>.<cr>.<i>" ; Label = {kind: consolation_round, n: cr+1}
           Src   = { vainqueur de prev[i], perdant de d (section main) }
       Rounds += ces nœuds ; prev = ces nœuds ; cr++
       si len(prev) == 1 → sortir

       tour interne : pour i = 0, 2, 4, … sur prev
           Key   = "<name>.<cr>.<i/2>" ; Label = {kind: consolation_round, n: cr+1}
           Src   = { vainqueur de prev[i], vainqueur de prev[i+1] }
       Rounds += ces nœuds ; prev = ces nœuds ; cr++

   le dernier nœud construit reçoit le libellé {kind: consolation_final}

L'**ordre inversé** des drops (``drops[len-1-i]``) sert à éviter qu'un joueur retombant du principal
ne retrouve immédiatement l'adversaire qui vient de le battre, ou un joueur de la même moitié de
tableau. Il ne garantit rien au-delà du tour immédiat. Le tirage étant enregistré dans le journal,
cet algorithme peut changer sans casser les journaux existants.

Tableau construit sur des sources (dernière chance)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code:: go

   func LastChanceSection(name string, srcs []Src, length int) Section

::

   size := pow2ceil(len(srcs)) ; au minimum 2
   sec := BracketSection(name, name, `size` places toutes à BYE, LengthPlan{Default: length})
   remplacer, dans l'ordre des nœuds du premier tour :
       d'abord la place 0 de chaque nœud, avec srcs[0], srcs[1], …
       puis la place 1 de chaque nœud, avec les sources restantes
   libellés : nœuds du premier tour → {kind: last_chance}
              autres                → {kind: last_chance, sub: RoundLabel(r, rounds)}

Le genre de la section est son nom (``last``). Remplir toutes les places 0 avant les places 1
**répartit les entrants** : avec 5 sources sur 8 places, on obtient trois joueurs exemptés plutôt
qu'un déséquilibre.

Groupe GSL
~~~~~~~~~~

.. code:: go

   func GSLSection(name string, players []Ref, length int) Section

Le format GSL (groupe de 4 à double élimination, 5 matchs) donne deux qualifiés : le vainqueur du
match des gagnants (0 défaite) et le vainqueur du match décisif (1 défaite). Genre ``gsl`` ; les
sections d'un bloc sont nommées ``B<bloc>G<groupe>`` par ``GSLBlockSections`` (voir
:ref:`format_gsl`).

Groupe de **4** joueurs ``a, b, c, d`` — clés ``<name>.0`` à ``<name>.4`` :

= =================== =========================
# Libellé             Places
= =================== =========================
0 ``opening``         ``a`` — ``b``
1 ``opening``         ``c`` — ``d``
2 ``winners_match``   vainqueur 0 — vainqueur 1
3 ``losers_match``    perdant 0 — perdant 1
4 ``decider``         perdant 2 — vainqueur 3
= =================== =========================

``Rounds = [[0,1], [2,3], [4]]``.

Groupe de **3** joueurs ``a, b, c`` :

= =================== =====================
# Libellé             Places
= =================== =====================
0 ``opening``         ``a`` — ``b``
1 ``winners_match``   vainqueur 0 — ``c``
2 ``decider``         perdant 0 — perdant 1
= =================== =====================

``Rounds = [[0], [1], [2]]``.

Groupe de **2** : un seul nœud, libellé ``single_match``, ``Rounds = [[0]]``.

Groupe de taille autre : section sans nœud (``GSLBlockSections`` ne crée pas de section pour un
groupe de moins de deux joueurs).

Mini-tableau à élimination directe
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code:: go

   func SESection(name string, players []Ref, length int) Section

Pour un groupe de 2 à 4 joueurs **à une vie** (blocs GSL des joueurs déjà battus une fois) :

::

   4 joueurs → slots = [p0, p1, p2, p3]
   3 joueurs → slots = [p0, p1, p2, BYE]
   2 joueurs → slots = [p0, p1]
   autre     → section vide (genre se)
   sec := BracketSection(name, "se", slots, LengthPlan{Default: length})
   tous les libellés ← {kind: single_elimination}

Poule toutes rondes (table de Berger)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code:: go

   func PoolSection(name string, players []Ref, length int) Section

::

   ids := players ; n := len(ids)
   si n est impair → ajouter BYE ; n++

   pour r de 0 à n-2 :
       pour i de 0 à n/2 - 1 :
           a, b := ids[i], ids[n-1-i]
           si a == BYE ou b == BYE → passer (pas de nœud)
           Key   = "<name>.<r>.<i>"
           Label = {kind: pool_round, n: r+1}
           Src   = { {Player: a}, {Player: b} }
       Rounds += les indices produits
       rotation : ids := [ids[0], ids[n-1], ids[1], ids[2], …, ids[n-2]]

Le premier élément reste fixe, les autres tournent d'un cran : c'est la table de Berger classique.
Chaque joueur rencontre chaque autre exactement une fois, en ``n-1`` rondes. Genre ``poule`` ;
``PoolSections`` nomme les poules ``poule:A``, ``poule:B``… (``PoolSectionName``) et pose
``Group = i+1``.

Les places d'une poule sont **toutes fixées** (``Src.Player``) : les matchs sont donc tous lançables
dès le tirage, sous la seule contrainte qu'un joueur ne joue qu'un match à la fois. C'est ce qui
donne aux poules leur souplesse d'ordonnancement (voir :ref:`format_poules`).
