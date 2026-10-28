.. _propositions:

Moteur de propositions
======================

``Propose``
-----------

.. code:: go

   func (s *State) Propose() []Action                // = ProposeAt(s.Last)
   func (s *State) ProposeAt(now time.Time) []Action // = ProposeWith(now, External{})
   func (s *State) ProposeWith(now time.Time, ext External) []Action

   type External struct {
       BusyTables  []int      `json:"busy_tables,omitempty"`  // tables occupées hors de ce tournoi
       BusyPlayers []PlayerID `json:"busy_players,omitempty"` // joueurs de ce tournoi qui jouent ailleurs
   }

C'est le point d'entrée principal du moteur (``engine.go``). Il ne modifie pas l'état : il
travaille sur une copie superficielle de ``State``, réglée à l'heure ``now`` (dont dépendent les
indisponibilités à échéance), si bien qu'un hôte peut l'appeler sous verrou de lecture. Hors du
temps et de ``External``, il ne dépend que du journal déjà appliqué.

``External`` est ce que l'hôte sait de la salle et que le journal de ce tournoi ignore : les tables
où joue, en ce moment, **une autre épreuve** dirigée dans la même salle. Ce n'est pas un événement :
l'occupation par l'extérieur change à chaque match de l'autre épreuve, elle n'appartient pas à
l'histoire de celle-ci, et le rejeu n'en a pas besoin — le ``match_started`` confirmé porte la table
effectivement prise. Les tirages et les passages de phase n'en dépendent pas.

``BusyPlayers`` dit quels joueurs de **ce** tournoi jouent en ce moment dans une autre épreuve de la
salle — un joueur inscrit à deux épreuves y a deux ``PlayerID``, et c'est l'hôte qui les relie. Un
identifiant inconnu est ignoré. Ces joueurs sont écartés des appariements dynamiques (suisse,
barrages de poule) et la file porte, au suisse, une attente par joueur en vie (``wait``, raison
``player_busy``, ``A`` = le joueur) ; dans un graphe, leur match reste proposé mais **retenu** —
raison ``player_busy``, sans table. Une indisponibilité (``player_unavailable``), écrite au journal,
l'emporte dans la raison. Rien n'est écrit : le TD qui lance quand même le match à la main est obéi,
et le rejeu l'ignore.

::

   si s.Current < 0 ou s.Finished → nil

   ph := phase courante
   selon ph.Cfg.Kind :
       swiss_lives :
           si micro-rondes actives, now < échéance et la phase n'est pas finie :
               → withAbsences([ Wait, raison waiting_batch, Until = échéance ])
           acts := proposeSwiss(ph)
       gsl                      → acts := proposeGSL(ph)
       lives_bracket, bracket   → acts := proposeBracket(ph)
       round_robin              → acts := proposeRR(ph)

   rep := proposeRepair(ph) + proposeSwissRepair(ph)     ← réparations, en tête de liste
   acts := rep + acts

   si acts est vide :
       si runningInPhase(ph) > 0 :
           → withAbsences([ Wait, raison matches_running ])
       si phaseDone(ph) :
           si une phase suivante existe → proposeRepechages(ph)   ← vide hors poules
                                          + [ NextPhase, Label = {kind: phase, text: nom de la phase suivante} ]
           sinon                        → [ Finish ]
       → withAbsences([ Wait, raison no_pairing ])

   si ph n'est ni tirée ni commencée et n'est pas la première :
       acts := proposeRepechages(phase précédente) + acts   ← passage fait, tirage pas encore
   si la phase n'est pas un suisse : holdAbsent(acts)      ← match d'un absent : raison player_unavailable
                                     holdElsewhere(acts)   ← joueur dans ext.BusyPlayers : raison player_busy
   assignTables(acts, ext.BusyTables)                      ← une action retenue n'a pas de table
   flagBreaks(acts, now)                                   ← Warn = ends_in_break
   → withAbsences(acts)

``withAbsences`` ne fait rien hors d'un suisse ; dans un suisse, elle ajoute une attente
``player_unavailable`` par joueur en vie indisponible (``Until`` ou ``Round`` disent quand il
revient), puis une attente ``player_busy`` par joueur en vie occupé ailleurs, ni absent ni en match
ici. Un joueur n'a qu'une attente.

Les raisons d'attente sont des codes (``ReasonCode``, ``codes.go``) : ``matches_running``,
``no_pairing``, ``waiting_batch``, ``waiting_table``, ``player_unavailable``, ``player_busy``. Le
libellé de ``next_phase`` est un ``Label`` de genre ``phase`` dont ``Text`` porte le nom de phase
saisi par l'organisateur (ou le nom par défaut de ``PhaseName``).

L'ordre du repli est important : **attendre la fin des matchs en cours a priorité sur la clôture de
la phase**. Le cas ``no_pairing`` ne devrait pas se produire dans un tournoi sain ; il signale une
impasse (par exemple deux joueurs libres qui se sont déjà rencontrés et ``allow_rematch``
désactivé — situation que les replis du suisse évitent, voir :ref:`format_suisse`).

Fin de phase
------------

.. code:: go

   func (s *State) phaseDone(ph *PhaseState) bool // = rules.PhaseDone(s.bilan(ph))

============================== ========================================================
Kind                           Condition (``rules.PhaseDone``)
============================== ========================================================
``swiss_lives``                ``swissDone`` (voir :ref:`format_suisse`)
``gsl``                        ``gslDone`` (voir :ref:`format_gsl`)
``round_robin``                ``Drawn && AllDone(Sections) && barragesDone``
``bracket``, ``lives_bracket`` ``Drawn && AllDone(Sections)``
============================== ========================================================

``rules.AllDone`` : tous les nœuds de toutes les sections sont ``Done`` (un match sauté compte comme
fait). Une phase de tableau non encore tirée n'est jamais « faite ». ``s.bilan(ph)`` est
l'instantané ``rules.Phase`` de la phase, construit par ``regles.go`` sans les adversaires ni les
joueurs occupés.

Matchs en cours
---------------

.. code:: go

   func (s *State) Running() []*Match                  // tous les matchs de statut Running, dans MatchOrder
   func (s *State) runningInPhase(ph *PhaseState) int  // leur nombre dans une phase donnée
   func (s *State) busy(p PlayerID) bool               // p a-t-il un match en cours ?

Réparation d'un graphe désaccordé
---------------------------------

Corriger un résultat de tableau après coup recalcule l'état et lève l'avertissement
``bracket_wrong_players`` ; le directeur se trouve alors devant un arbre faux, avec les matchs
suivants déjà joués par les mauvaises personnes. ``proposeRepair`` (``reparation.go``) émet la
réparation comme des propositions ORDINAIRES, en tête de liste :

::

   1. les places dont le match (non annulé) n'a pas été joué par les deux joueurs que la place attendait
   2. + tout ce qui en descend (point fixe sur les Src, sections traversées comprises)
   3. → une action cancel_match par place (Section, Key, Label, Match, A, B),
        triées par indice de section décroissant, puis par indice de match décroissant

Un demi-finaliste faux fait une finale fausse, même si la finale oppose bien les deux joueurs que le
graphe attendait : elle les attendait pour de mauvaises raisons. D'où l'annulation **en série**. Les
annulations sont proposées du plus profond au moins profond — on défait le plus récent d'abord,
comme sur le papier.

Les relances correctes ne sont pas fabriquées : une fois les annulations confirmées, les places
redeviennent libres et sont proposées comme n'importe quel match prêt, à l'appel suivant. Tant que
le TD n'a rien confirmé, il n'y a rien à relancer.

**Rien n'est appliqué d'office.** Ne rien confirmer laisse l'état tel quel, avertissement compris :
un directeur peut préférer laisser le tableau tel qu'il a été joué et le noter à la main.

**Dans un suisse**, il n'y a pas de graphe à réparer, mais une correction peut éliminer un joueur en
plein match (``correction_eliminates_running``). ``proposeSwissRepair`` propose alors, en tête de
liste avec les autres réparations, un ``cancel_match`` par match en cours concerné (avec son
``Round``). Le TD peut aussi laisser finir le match : son adversaire perdra peut-être une vie contre
un joueur éliminé, et c'est sa décision.

Corollaire dans ``recompute`` (``state.go``) : ``GMatch.MatchID`` repart **vide** au recalcul, et
n'est reposé que pour les matchs non annulés ; les places dérivées d'un autre match repartent elles
aussi vides (``Players[k] = Src[k].Player``). Sans cela, une place dont le match vient d'être annulé
garderait son identifiant, ``rules.Ready`` l'écarterait, et le tableau resterait bloqué à l'endroit
même qu'on veut rejouer.

Micro-rondes et pauses
----------------------

Le moteur n'a pas d'horloge. Le temps entre par ``ProposeAt(now)`` — l'hôte donne la sienne — ou, à
défaut, par l'horodatage du dernier événement du journal (``Propose()`` = ``ProposeAt(s.Last)``).
Seul ce qui dépend du temps change avec ``now`` : les micro-rondes, les pauses et les
indisponibilités à échéance. Le reste — appariements, tirages, passage de phase — ne dépend que du
journal, et le rejeu n'est donc pas affecté : ce qui est rejoué, ce sont les événements, pas les
propositions.

Micro-rondes (``batch_minutes``)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

L'appariement au fil de l'eau rend le suisse continu manipulable par l'heure d'annonce d'un
résultat : celui qui finit à 14 h 03 choisit son adversaire en annonçant à 14 h 04 ou à 14 h 20.
``PhaseConfig.BatchMinutes`` (``batch_minutes``) ferme cette porte : les joueurs libres attendent
l'échéance, puis tous ceux d'un même groupe de défaites sont appariés d'un coup.

::

   batchDeadline(ph) :
       si batch_minutes <= 0         → pas de micro-rondes
       dernier := départ (Match.Start) le plus tardif des matchs non annulés de la phase
       si aucun                      → pas d'échéance : le premier lot part tout de suite
       échéance := dernier + batch_minutes

   si échéance existe, now < échéance et non swissDone(ph) :
       → une seule action {wait, reason: waiting_batch, until: échéance}
         (plus les attentes par joueur de withAbsences)
   sinon                         → appariement ordinaire du suisse

L'horodatage du dernier lot **n'est stocké nulle part** : c'est le départ du dernier match lancé
dans la phase, un lot étant exactement cela — un paquet de matchs lancés ensemble. Le déduire évite
un champ d'état de plus à tenir juste après une correction ou une annulation.

``Action.Until`` porte l'échéance pour que l'hôte affiche un compte à rebours. ``Validate`` refuse
``batch_minutes`` négatif, en mode ``rounds`` (une ronde est déjà un lot) et hors ``swiss_lives``.

.. _pauses-configbreaks:

Pauses (``Config.Breaks``)
~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code:: go

   type TimeRange struct {
       Start time.Time `json:"start"` // incluse
       End   time.Time `json:"end"`   // exclue
   }

``Validate`` refuse une pause qui finit avant de commencer. ``flagBreaks`` (``horaires.go``) :

::

   pour chaque action start_match de longueur > 0 :
       fin := now + Expected(Length)
       si now < pause.End et fin > pause.Start pour une pause → Action.Warn = ends_in_break

``Expected(n)`` est la durée attendue d'un match de *n* points (``rules.ExpectedDuration``, à partir
de ``Config.MinPerPoint``, voir :ref:`horloge`).

**Rien n'est bloqué.** Le directeur sait des choses que le moteur ignore — que ces deux-là mangeront
après, que la pause est indicative, que la salle ferme de toute façon. On le lui dit, il décide ;
c'est la règle de tous les avertissements du moteur.

L'avertissement tombe dès que le match **rencontre** la pause, et pas seulement s'il s'y termine :
un match lancé à midi moins dix et attendu pour 13 h 30 fait manquer le repas tout autant.

Attribution des tables
----------------------

.. code:: go

   func (s *State) assignTables(acts []Action, dehors []int)

::

   used := { table de chaque match en cours, si > 0 } ∪ { tables de dehors, si > 0 }
   pour chaque action du lot, dans l'ordre :
       si ce n'est pas un start_match, si Table est déjà renseignée,
          ou si Reason est déjà posée (action retenue) → passer
       t := la plus petite table t ≥ 1 telle que
                non used[t] et Config.Tables.AvailableFor(t, Section, Phase)
            (Count > 0 : t ≤ Count ;
             Count = 0, salle illimitée : t ≤ |used| + |Unavailable| + |Reserved| + 1)
       si aucune : Action.Reason = waiting_table ; passer
       Action.Table = t ; used[t] = true

``AvailableFor`` (``tables.go``) écarte les tables hors service (``Tables.Unavailable``), celles
au-delà de ``Count`` et celles qui sont réservées (``Tables.Reserved``, des ``TableRule``) à une
autre section ou à une autre phase. ``dehors`` est ``External.BusyTables`` : les tables où joue une
autre épreuve de la salle. La table proposée est donc toujours **la plus petite libre**. Une action
qui n'en trouve pas garde ``Table = 0`` et porte la raison ``waiting_table`` : elle reste
confirmable (le moteur ne bloque pas), le TD saisit une table ou attend qu'une se libère. Voir
:ref:`configuration` pour la forme de ``Config.Tables``.

Générateur pseudo-aléatoire
---------------------------

.. code:: go

   // rules/phase.go
   type Seed struct {
       Base   int64 `json:"base"`   // graine du tournoi (State.Seed)
       Events int   `json:"events"` // nombre d'événements appliqués (State.NEvents)
       Phase  int   `json:"phase"`  // phase courante (State.Current)
   }

   func (s Seed) Draw() int64 { return s.Base*1000003 + int64(s.Events)*7919 + int64(s.Phase) }
   func (s Seed) Round(r int) int64 {
       return s.Base*1000003 + int64(s.Phase)*7919 + int64(r)*104729
   }

   func generateur(graine int64) *rand.Rand { return rand.New(rand.NewSource(graine)) }

Le moteur construit la graine par ``State.seed()`` (``regles.go``) et la passe à ``rules.Pair`` ;
le générateur est **recréé à chaque appel** et ne dépend que de la graine, du nombre d'événements et
de la phase (``Seed.Draw``). Une ronde de suisse par rondes a le sien, ``Seed.Round(r)``, qui ne
dépend que de la graine, de la phase et du numéro de ronde (voir :ref:`format_suisse`). Toute
implémentation visant la compatibilité binaire avec la référence doit reproduire le générateur
``math/rand`` de Go (source *Additive Lagged Fibonacci* de la bibliothèque standard) et sa méthode
``Shuffle``. **Une implémentation dans un autre langage ne pourra pas reproduire les propositions à
l'identique** ; elle reproduira en revanche exactement les états et les classements à partir d'un
journal donné, puisque les tirages y sont matérialisés. C'est la propriété qui compte.

Utilitaires d'ordre
-------------------

.. code:: go

   func sortedIDs(ids []PlayerID) []PlayerID // engine.go ; rules.sortedRefs est son pendant

Renvoie une **copie** triée par ordre croissant de l'identifiant (comparaison de chaînes). C'est le
préalable obligatoire à tout mélange aléatoire ou tri partiel : sans lui, l'ordre d'entrée
dépendrait de l'itération d'une map, et les propositions ne seraient plus déterministes.
