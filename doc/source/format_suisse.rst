.. _format_suisse:

Suisse à vies (``swiss_lives``)
===============================

Principe
--------

Un suisse à *L* vies est une **double élimination sans tableau** (pour *L* = 2) : chaque joueur est
éliminé après *L* défaites, et l'appariement se fait toujours **à l'intérieur du groupe des joueurs
ayant le même nombre de défaites**. Tout effectif est admis, la fin est toujours résoluble sans
départage, et il n'y a pas de graphe à construire. ``lives`` vaut 2 par défaut.

La règle vit dans ``rules/suisse.go`` (``rules.Pair``, ``PairSwissRound``, ``PairSwissFallback``,
``PairSwissOrphans``, ``SwissLength``) ; ``phase_swiss.go`` en est l'enveloppe : il construit
l'instantané ``rules.Phase`` (``regles.go``), calcule les joueurs appariables et tient la ronde
ouverte.

Deux modes (``mode``) :

- ``continuous`` (défaut, posé par ``Validate``) : dès que deux joueurs du même groupe sont libres,
  un match est proposé. C'est le mode recommandé — l'étude montre que l'attente est le premier
  gaspillage d'un tournoi (10 à 15 fois plus d'attente en synchrone qu'en continu, voir
  :ref:`annexe_etude`). Les micro-rondes (``batch_minutes``) y regroupent les appariements par lots
  (voir :ref:`propositions`).
- ``rounds`` : tous les matchs d'une ronde sont proposés ensemble, byes compris, et la ronde
  suivante n'est proposée qu'une fois la ronde entièrement engagée et tous ses matchs terminés.
  ``batch_minutes`` y est refusé : une ronde est déjà un lot.

``Validate`` refuse un autre mode, un ``target`` qui n'est pas une puissance de 2, un ``target``
avec ``lives`` ≠ 2, et ``length_late`` sans ``late_threshold``.

Bascule vers un tableau (``target``)
------------------------------------

.. code:: go

   func (s *State) sumLives(ph *PhaseState) int    // Σ vies restantes des entrants (rules.LivesLeft)
   func (s *State) swissDone(ph *PhaseState) bool  // = rules.PhaseDone(s.bilan(ph))

::

   vies restantes d'un joueur : max(Lives − Losses, 0), et 0 s'il est retiré (rules.RemainingLives)

   swissDone :
       si Running > 0                           → faux
       si Target > 0 et Σ vies <= Target        → vrai
       → len(alive) <= 1

L'idée : chaque match consomme exactement une vie. Quand la somme des vies restantes atteint une
puissance de 2, on peut basculer sur un tableau à exemptions parfaitement équilibré — un joueur à 2
vies occupe deux places du tableau (il est exempt du premier tour), un joueur à 1 vie en occupe une.
C'est pourquoi ``Validate`` impose ``Lives == 2`` avec ``Target``. Pour ne pas « dépasser » la
cible, l'appariement est borné par un **budget**, décrit ci-dessous pour chacun des deux modes.

Sans ``Target``, la phase va jusqu'à son terme naturel : un seul joueur en vie.

Joueurs libres et historique
----------------------------

.. code:: go

   func (s *State) free(ph *PhaseState, r int) []PlayerID // phase_swiss.go
   func (v *vue) met(a, b Ref) bool                       // rules : b figure dans les adversaires de a
   func (v *vue) sameClub(a, b Ref) bool                  // rules

``free(ph, r)`` : les joueurs en vie (dans l'ordre des entrants), privés de ceux qui ont un match en
cours, de ceux qui sont indisponibles pour la ronde ``r`` (0 hors rondes ; voir :ref:`etat`) et de
ceux qui jouent dans une autre épreuve de la salle (``External.BusyPlayers``). Le moteur le passe
dans ``rules.Phase.Free``.

``met`` lit ``Entrant.Opponents`` (adversaires des matchs non annulés, avec répétitions).
``sameClub`` : les deux clubs sont non vides et identiques.

Appariement d'un groupe
-----------------------

.. code:: go

   func (v *vue) pairGroup(g []Ref, rng *rand.Rand, allowRematch bool) (pairs [][2]Ref, rest []Ref)

C'est le cœur de l'appariement suisse (``rules/suisse.go``). Glouton, sur une liste dont l'ordre a
été construit en plusieurs passes :

::

   1. g := sortedRefs(g)                      ← ordre déterministe de départ
   2. rng.Shuffle(g)                          ← mélange aléatoire
   3. si pairing == "wins" :
          tri stable décroissant sur Wins     ← les plus victorieux en tête
   4. tri stable décroissant sur Byes         ← ceux qui ont déjà eu un bye d'abord

   5. libres := g
      tant que len(libres) >= 2 :
          a := libres[0] ; retirer a de libres
          passes := [faux]                    ← une seule passe
          si avoid_clubs → passes := [vrai, faux]   ← d'abord en évitant les clubs
          trouvé := -1
          pour exigerClub dans passes :
              pour chaque c de libres, dans l'ordre :
                  si non allowRematch et non allow_rematch et met(a, c) → passer
                  si exigerClub et sameClub(a, c)                       → passer
                  trouvé := index de c ; sortir
              si trouvé >= 0 → sortir
          si trouvé < 0 : rest += a ; continuer
          pairs += (a, libres[trouvé]) ; retirer libres[trouvé]

   6. rest += ce qui reste dans libres        ← au plus un joueur en fin de boucle

Propriétés :

- **le tri par byes est le dernier appliqué**, donc le plus fort : un joueur qui a déjà été exempté
  passe en tête de liste et sera apparié en priorité. C'est la règle « pas de second bye tant que
  d'autres n'en ont pas eu », appliquée **à l'intérieur d'un groupe** ; l'équité entre groupes
  différents n'est pas assurée ;
- ``pairing: "wins"`` ne fait qu'ordonner : il ne calcule aucun score de type Buchholz. Il rapproche
  les joueurs de même palmarès dans un même groupe de défaites. Le défaut est ``"random"`` ;
- ``avoid_clubs`` est une préférence, pas une contrainte : si aucun adversaire d'un autre club n'est
  disponible, la seconde passe accepte le même club ;
- l'interdiction de rematch est en revanche une **contrainte dure**, sauf ``allow_rematch`` ou appel
  de repli (``allowRematch = vrai``).

Longueur des matchs
-------------------

.. code:: go

   func SwissLength(p Phase) int // rules ; enveloppe : (s *State) swissLength(ph)

::

   si length_late > 0, Length == Rules.Length et len(alive) <= late_threshold → length_late
   sinon                                                                      → Length

``Length`` est la longueur courante de la phase. La fin d'un suisse se joue ainsi en matchs plus
longs. Un ``length_changed`` posé à la main par le TD (``Length ≠ length`` configurée) l'emporte :
c'est lui qui dirige.

Proposition
-----------

.. code:: go

   func (s *State) proposeSwiss(ph *PhaseState) []Action // phase_swiss.go

::

   si swissDone(ph) → nil
   r := 0
   si Mode == "rounds" :
       si ph.Round > 0 et roundRest(ph, ph.Round) non vide → roundRest(ph, ph.Round)
       si runningInPhase(ph) > 0 → nil                   ← on attend la fin de la ronde
       r := ph.Round + 1
   → actions(rules.Pair(snapshot(ph, free(ph, r)), seed()))

``rules.Pair`` dispatche vers ``pairContinuous(générateur(seed.Draw()))`` en mode continu, et en
mode par rondes vers ``pairRound(Round+1, Free, générateur(seed.Round(Round+1)))``, avec repli
``rematchOrFinal(Free, générateur(seed.Draw()))`` quand la ronde ne trouve aucune paire et qu'au
moins deux joueurs sont libres. Les graines sont décrites dans :ref:`propositions`.

Mode continu
~~~~~~~~~~~~

.. code:: go

   func (v *vue) pairContinuous(rng *rand.Rand) []Pairing

::

   L := Lives ; free := Free

   budget := +∞
   si Target > 0 :
       budget := Σ vies − Running − Target           ← matchs encore lançables
       si budget <= 0 → nil                           ← la phase est figée

   pour l de 0 à L-1, tant que budget > 0 :
       g := les joueurs de free ayant exactement l défaites
       pairs, _ := pairGroup(g, rng, faux)
       pour chaque paire (a, b), tant que budget > 0 :
           start_match
               Label  = {kind: swiss_group, losses: Losses[a], match: Wins[a]+Losses[a]+1}
               A, B   = la paire ; Length = SwissLength
           budget--

   si aucune action, aucun match en cours, et au moins 2 joueurs libres → REPLI

Le **budget** garantit qu'on ne « dépasse » jamais la cible : avec Σ vies = 18 et ``target`` 16, deux
matchs au plus peuvent être en cours ou lancés, même si six joueurs sont libres.

**Repli** (quand aucun appariement régulier n'est possible et qu'aucun match ne tourne) :

::

   1. pour l de 0 à L-1 :
          pairs := pairGroup(groupe l, rng, allowRematch = VRAI)
          si une paire existe → une seule action, Label = {kind: rematch}
   2. sinon :
          fr := free trié par identifiant, puis stablement par défaites croissantes (leastLosses)
          Label = {kind: final} si len(fr) == 2, sinon {kind: crossed}
          une action opposant fr[0] et fr[1]

Ce repli est ce qui permet de **toujours conclure** : quand il ne reste que deux joueurs qui se sont
déjà rencontrés, ou deux joueurs de groupes différents (par exemple un à 0 défaite et un à 1 défaite
en fin de suisse à 2 vies), le moteur propose quand même le match. Les libellés ``final`` et
``crossed`` sont d'ailleurs reconnus par les tests d'invariants (``sim_test.go``) comme les seules
exceptions autorisées à la règle « pas de match entre groupes différents ».

Mode par rondes
~~~~~~~~~~~~~~~

Une ronde reste **ouverte** tant que l'un des joueurs appelés n'y est pas engagé. Une ronde de 12
matchs dans une salle de 6 tables se lance en deux vagues ; un bye se confirme parfois avant les
matchs. Tant que la ronde est ouverte, ses appariements non lancés restent proposés — les mêmes —,
et la ronde suivante ne vient que lorsque la ronde est entièrement engagée et ses matchs terminés.

.. code:: go

   func PairSwissRound(p Phase, r int, players []Ref, seed Seed) (plan []Pairing, normal bool)

``PairSwissRound`` (``pairRound``, générateur ``seed.Round(r)``) est l'appariement de la ronde ``r``
sur l'instantané ``p`` :

::

   budget := ∞ ; si Target > 0 : budget := Σ vies − Target ; si budget ≤ 0 → (nil, normal)
   pour l de 0 à Lives-1, tant que budget > 0 :
       g := joueurs à l défaites (dans p)
       pairs, rest := pairGroup(g, rng, faux)
       pour chaque paire, tant que budget > 0 →
           start_match, Label = {kind: round, n: r}, Round = r, Length = SwissLength ; budget--
       restes += rest
   si aucune action → (nil, non normal)
   si budget ≤ 0 → (actions, normal)              ← ronde tronquée par la bascule : pas de bye
   pour chaque joueur de restes → bye, Label = {kind: round, n: r}, Round = r

Le budget d'une ronde ne retranche pas les matchs en cours : une ronde n'est appariée qu'une fois
la précédente terminée. La DERNIÈRE ronde d'une phase à bascule est ainsi tronquée au nombre exact
de matchs qui reste à jouer, et ne donne aucun bye : un bye enregistré fausserait l'ordre
d'appariement d'une ronde qui n'aura pas lieu.

**Les mêmes appariements**, parce qu'ils sont *recalculés à l'identique* plutôt que stockés. Le plan
d'une ronde ne dépend que de trois choses :

- l'état **d'avant la ronde** : ``beforeRound(ph, r)`` est la phase dont on a retiré les victoires,
  défaites et adversaires des matchs de ronde ``r`` (non annulés), et les byes de ronde ``r``
  (``PhaseState.ByeRounds``) ;
- les **joueurs appelés** : ``PhaseState.Roster``, relevé par ``Apply`` (``openRound``) au premier
  événement de la ronde (``match_started`` ou ``bye`` dont ``Round`` dépasse ``ph.Round``), avant de
  l'appliquer — exactement les joueurs libres (``free(ph, r)``) qu'un ``Propose`` de cet instant
  appariait ;
- un **générateur propre à la ronde**, ``rules.Seed.Round(r)`` :

.. code:: go

   func (s Seed) Round(r int) int64 {
       return s.Base*1000003 + int64(s.Phase)*7919 + int64(r)*104729
   }

Il ne dépend pas du nombre d'événements : sans cela, confirmer le premier match d'une ronde
redistribuerait les suivants.

.. code:: go

   func (s *State) roundRest(ph *PhaseState, r int) []Action // phase_swiss.go

``roundRest(ph, R)`` renvoie ``nil`` si ``Roster`` est vide ; sinon il recalcule le plan
``PairSwissRound(snapshot(beforeRound(ph, R)), R, ph.Roster, seed)`` (``nil`` si le plan n'est pas
normal) et garde, dans l'ordre du plan :

- chaque ``start_match`` dont les deux joueurs sont **libres** : ni engagés dans la ronde (un match
  non annulé de ronde ``R``, ou un bye de ronde ``R``), ni éliminés ou retirés, ni en match, ni
  indisponibles pour la ronde, ni occupés dans une autre épreuve. Sa longueur est celle
  d'aujourd'hui (``swissLength``), un ``length_changed`` ayant pu passer ;
- chaque ``bye`` dont le joueur est libre.

Les **orphelins** — joueurs libres du plan, non repris, prévus contre quelqu'un qui a joué autre
chose (un match lancé à la main) ou qui est parti — sont appariés entre eux par
``PairSwissOrphans`` (groupe par groupe de défaites sur l'instantané d'avant la ronde,
``pairGroup`` sans rematch, générateur ``seed.Round(R)``, ``Label = {kind: round, n: R}``), dans le
budget de la bascule s'il y en a une (``Σ vies − matchs en cours − Target − matchs déjà gardés``).
Un orphelin seul attend la ronde suivante. Un joueur inscrit pendant la ronde n'est pas dans
``Roster`` : il entre à la ronde suivante.

Limites assumées : un joueur retiré en cours de ronde, dans une phase à bascule, change le budget et
peut donc changer les appariements non lancés ; la ronde reste une ronde (personne n'y joue deux
fois).

Le repli du mode par rondes :

.. code:: go

   func PairSwissFallback(p Phase, free []Ref, seed Seed) []Pairing // rematchOrFinal, générateur seed.Draw()

::

   1. pour l de 0 à Lives-1 : la première paire avec allowRematch = VRAI → Label = {kind: rematch}
   2. sinon : fr trié par identifiant puis par défaites croissantes → une action fr[0] contre fr[1],
      Label = {kind: final}, quel que soit le nombre de joueurs libres

Les actions du repli ne portent pas de numéro de ronde (``Round = 0``) : elles ne comptent dans
aucune ronde.

**Le compteur de rondes** ``ph.Round`` n'est pas incrémenté par le moteur : il suit le champ
``Event.Round`` des événements ``match_started`` et ``bye`` (un journal de version 0 est converti à
la lecture par ``Event.upgraded``, qui relit le numéro dans l'ancien libellé « Ronde k », voir
:ref:`journal`). Dès que le TD confirme la première action d'une ronde, ``ph.Round`` prend son
numéro et ``Roster`` est relevé. ``Match.Round`` garde la ronde de chaque match.

Absences et réparation
----------------------

Un suisse n'apparie pas un joueur indisponible ni un joueur occupé dans une autre épreuve : il est
absent de ``free`` et la file porte une attente par joueur (``player_unavailable`` ou
``player_busy``). Une correction qui élimine un joueur en plein match
(``correction_eliminates_running``) fait proposer l'annulation de ce match en tête de liste. Voir
:ref:`propositions`.

Classement d'une phase à vies
-----------------------------

.. code:: go

   func Rank(p Phase) []Standing // rules/classement.go : rankLives ; enveloppe : (s *State) phaseRanking(ph)

Utilisé pour ``swiss_lives`` **et** ``gsl`` (``rules.Rank`` y retombe pour tout format qui n'est ni
un tableau ni des poules).

::

   alive := alive(p)
   pour chaque entrant e, dans l'ordre des entrants :
       score := Wins
       note  := {kind: record, wins: Wins, losses: Losses}
       si vies restantes > 0 :
           score += 1000 + vies restantes
           note  := {kind: alive, lives: vies restantes}
           si len(alive) == 1 → note := {kind: winner}
       si e est retiré :
           score := Wins
           note  := {kind: withdrawn, wins, losses, lives: max(Lives − Losses, 0)}

   si len(alive) == 1 et Eliminated non vide :
       le DERNIER éliminé reçoit score 999 et la note {kind: finalist}

   tri stable par score décroissant
   rang := 1 ; pour chaque i : si score[i] != score[i-1] → rang := i+1

Lecture des scores :

+-----------------------------+--------------------------------------------------------------------+
| Score                       | Signification                                                      |
+=============================+====================================================================+
| ``1000 + vies + victoires`` | joueur encore en vie ; plus il a de vies et de victoires, mieux il |
|                             | est classé                                                         |
+-----------------------------+--------------------------------------------------------------------+
| ``999``                     | finaliste (dernier éliminé quand le vainqueur est connu)           |
+-----------------------------+--------------------------------------------------------------------+
| ``victoires``               | éliminé, ou retiré ; départagé par le nombre de victoires au       |
|                             | moment de l'élimination (ou du retrait), ex æquo conservés         |
+-----------------------------+--------------------------------------------------------------------+

C'est la traduction exacte du principe « pas de départage » : deux joueurs éliminés avec le même
nombre de victoires sont **ex æquo**, définitivement. Les notes sont des codes (``rules.Note``) que
l'hôte traduit.

**Un joueur retiré est classé sur son parcours**, comme s'il avait été éliminé à l'instant de son
retrait : par ses victoires, **ex æquo** avec les éliminés qui en ont autant, derrière tous les
joueurs encore en vie. Ses vies restantes ne comptent pas — il ne les jouera pas — mais la note les
porte, avec ses victoires et ses défaites (``withdrawn``). Le parti pris est celui des éliminés :
pas de départage, et un retrait n'efface pas ce qui a été joué. Le meilleur joueur d'un suisse de
cinq jours qui part le mercredi avec huit victoires est classé parmi les éliminés à huit victoires.
Le « forfait » ne désigne que le match abandonné (``Match.Forfeit``) et, dans le classement propre
d'une section, le match perdu par forfait (note ``forfeit``). Voir :ref:`classement`.
