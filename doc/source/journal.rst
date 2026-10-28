.. _journal:

Journal et événements
=====================

Le catalogue des codes
----------------------

Rien de ce qui sort du moteur n'est destiné à être affiché tel quel. Un logiciel hôte affiche un
tournoi dans la langue de son utilisateur ; le moteur ne peut donc figer aucune langue, et surtout
pas dans le journal, qui est conservé pour toujours.

Cinq familles de codes, toutes dans ``codes.go``, et les statuts de joueur (``statut.go``). Les
valeurs des libellés, des notes et des noms de section sont définies une fois dans
``rules/codes.go`` ; ``codes.go`` les reprend sous ses propres types :

+---------------------+--------------------------------------------------------------------------------------------------------------------+-------------------------------------+
| Famille             | Type                                                                                                               | Où elle apparaît                    |
+=====================+====================================================================================================================+=====================================+
| Libellés            | ``Label{Kind, N, Losses, Match, Section, Text, Players, Spots, Sub}``                                              | ``Event.Label``, ``Match.Label``,   |
|                     |                                                                                                                    | ``Action.Label``, ``GMatch.Label``  |
+---------------------+--------------------------------------------------------------------------------------------------------------------+-------------------------------------+
| Notes de classement | ``Note{Kind, Wins, Losses, Lives, Section, Qualified, Sub}``                                                       | ``Rank.Note``                       |
+---------------------+--------------------------------------------------------------------------------------------------------------------+-------------------------------------+
| Avertissements      | ``Warning{Code, Match, Section, Label, A, B, ExpectedA, ExpectedB, Length, ScoreA, ScoreB, Player, Table, Other}`` | ``State.Warnings``, ``Action.Warn`` |
|                     |                                                                                                                    | (le code seul)                      |
+---------------------+--------------------------------------------------------------------------------------------------------------------+-------------------------------------+
| Informations        | ``Info{Code, Player, Phase, Section, Label}``                                                                      | ``State.Infos``                     |
+---------------------+--------------------------------------------------------------------------------------------------------------------+-------------------------------------+
| Statuts             | ``PlayerStatus{Player, Kind, Phase, Section, Round, Label}``                                                       | ``State.Statuses()``,               |
|                     |                                                                                                                    | ``State.StatusOf(p)``               |
+---------------------+--------------------------------------------------------------------------------------------------------------------+-------------------------------------+
| Raisons d'attente   | ``ReasonCode``                                                                                                     | ``Action.Reason``                   |
+---------------------+--------------------------------------------------------------------------------------------------------------------+-------------------------------------+

``Label.Sub`` compose un libellé dans un autre (« Bloc 2, groupe B : match décisif »).
``Label.Text`` est le **seul** champ texte admis, et il ne contient que ce que le TD a lui-même
saisi — le nom d'une phase, la lettre d'une poule.

Les valeurs des codes entrent dans le journal : les changer casse les journaux existants. C'est la
raison d'être de ``Event.Version``.

Les **noms de section** (``Section.Name``, ``Match.Section``, ``Event.Section``) sont eux aussi des
identifiants et non des libellés : ``main``, ``conso``, ``last``, ``gf``, ``poule:A``,
``barrage:A``, et les groupes GSL (``B1G2``). Ils voyagent dans le journal et dans la base de
l'hôte. Les poules sont nommées ``poule:A``, ``poule:B``… (``rules.PoolSectionName``), le
barrage d'une poule ``barrage:`` suivi de la même lettre (``rules.BarrageSectionName``), les
groupes GSL ``B<bloc>G<groupe>``.

Un test du dépôt (``codes_test.go``) refuse toute chaîne accentuée ou contenant un espace qui
sortirait du moteur — propositions, classement, avertissements, informations, journal — hors les
champs que le TD a saisis (``text``, ``name``, ``club``). Un autre vérifie que chaque code a un
rendu français dans ``fr.go``, et qu'aucun ne retombe sur sa valeur brute.

Structure d'un événement
------------------------

.. code:: go

   const JournalVersion = 2 // version écrite dans chaque nouvel événement

   type EventKind string

   const (
       EvCreated         EventKind = "created"
       EvPlayerAdded     EventKind = "player_added"
       EvPlayerWithdrawn EventKind = "player_withdrawn"
       EvPlayerUpdated   EventKind = "player_updated"
       EvPlayerUnavailable EventKind = "player_unavailable"
       EvPlayerAvailable   EventKind = "player_available"
       EvMatchStarted    EventKind = "match_started"
       EvResult          EventKind = "result"
       EvResultCorrected EventKind = "result_corrected"
       EvMatchCancelled  EventKind = "match_cancelled"
       EvBye             EventKind = "bye"
       EvDraw            EventKind = "draw"
       EvNextPhase       EventKind = "next_phase"
       EvRepechage       EventKind = "repechage"
       EvLengthChanged   EventKind = "length_changed"
       EvConfigChanged   EventKind = "config_changed"
       EvReopened        EventKind = "reopened"
       EvTableChanged    EventKind = "table_changed"
       EvFinished        EventKind = "finished"
       EvNote            EventKind = "note"
   )

   type Event struct {
       Seq          int       `json:"seq"`
       Version      int       `json:"version,omitempty"` // format du journal ; 0 = avant les codes
       Kind         EventKind `json:"kind"`
       Time         time.Time `json:"time"`
       Config       *Config   `json:"config,omitempty"`
       Seed         int64     `json:"seed,omitempty"`
       Player       *Player   `json:"player,omitempty"`
       ID           PlayerID  `json:"player_id,omitempty"`
       MatchID      MatchID   `json:"match_id,omitempty"`
       Phase        int       `json:"phase,omitempty"`
       Section      string    `json:"section,omitempty"`
       Label        Label     `json:"label,omitempty"`   // un CODE, jamais une phrase
       Round        int       `json:"round,omitempty"`
       Key          string    `json:"key,omitempty"`
       A            PlayerID  `json:"a,omitempty"`
       B            PlayerID  `json:"b,omitempty"`
       Length       int       `json:"length,omitempty"`
       Table        int       `json:"table,omitempty"`
       Winner       PlayerID  `json:"winner,omitempty"`
       ScoreA       int       `json:"score_a,omitempty"`
       ScoreB       int       `json:"score_b,omitempty"`
       Forfeit      bool      `json:"forfeit,omitempty"`
       AfterCurrent bool      `json:"after_current,omitempty"` // retrait différé
       Draw         *Draw     `json:"draw,omitempty"`
       Text         string    `json:"text,omitempty"`
       Slot         string    `json:"slot,omitempty"` // retardataire : place d'exemption prise
       Until        *time.Time `json:"until,omitempty"` // player_unavailable : heure de retour
   }

   type Journal []Event

Un événement est une structure « plate » : tous les types partagent les mêmes champs, les champs
inutiles restant vides. C'est un choix délibéré — il rend le journal lisible et évite la
désérialisation polymorphe.

``Version`` est le format du journal (``JournalVersion`` = 2). Un événement sans ce champ
(version 0) vient d'un journal antérieur aux codes structurés et est **converti à la lecture** :
``Replay`` applique ``Event.upgraded()`` à chaque événement, qui transforme un libellé texte
« Ronde k » en ``Label{Kind: round, N: k}`` et renseigne ``Round`` (voir
:ref:`annexe_tests` pour la compatibilité ascendante). La version 2 ne change que la règle de
``config_changed`` (voir « Journaux anciens » à la section ``config_changed``). Les constructeurs d'événements de
``events.go`` posent la version : un ``Event`` écrit à la main sans version serait relu comme un
journal ancien.

``Label`` est un **code**, pas une chaîne (voir « Le catalogue des codes » ci-dessus). À la
lecture, un libellé JSON qui est une chaîne (journaux de version 0) est accepté et rangé tel quel
dans ``Label.Text``.

``Seq`` est un numéro d'ordre facultatif renseigné par l'hôte (les binaires du dépôt écrivent
``Seq = len(journal)`` avant l'ajout). Le moteur **ne le lit pas** : l'ordre du tableau fait foi.

``Time`` est l'horodatage. Le moteur l'utilise pour :

- l'horodatage de début et de fin des matchs ;
- la mise à jour de ``State.Last`` (le maximum des horodatages vus), qui sert d'heure à
  ``Propose()`` ;
- l'heure de l'état pendant l'application de l'événement (``s.clock = ev.Time``), à laquelle
  une indisponibilité à échéance (``Until``) est comparée.

Il n'exige pas la monotonie.

Champs utilisés par type
------------------------

+------------------------+-------------------------------------------------------------------------+
| Kind                   | Champs lus                                                              |
+========================+=========================================================================+
| ``created``            | ``Config`` (obligatoire), ``Seed``                                      |
+------------------------+-------------------------------------------------------------------------+
| ``player_added``       | ``Player`` (obligatoire, ``ID`` non vide) ; retardataire : ``Phase``,   |
|                        | ``Section``, ``Slot``                                                   |
+------------------------+-------------------------------------------------------------------------+
| ``player_withdrawn``   | ``ID``, ``AfterCurrent``                                                |
+------------------------+-------------------------------------------------------------------------+
| ``player_updated``     | ``Player`` (obligatoire, ``ID`` d'un joueur inscrit)                    |
+------------------------+-------------------------------------------------------------------------+
| ``player_unavailable`` | ``ID`` ; ``Until`` (heure de retour) ou ``Round`` (ronde de retour), ou |
|                        | aucun des deux                                                          |
+------------------------+-------------------------------------------------------------------------+
| ``player_available``   | ``ID``                                                                  |
+------------------------+-------------------------------------------------------------------------+
| ``match_started``      | ``MatchID``, ``Phase``, ``Section``, ``Label``, ``Round``, ``Key``,     |
|                        | ``A``, ``B``, ``Length``, ``Table``                                     |
+------------------------+-------------------------------------------------------------------------+
| ``result``             | ``MatchID``, ``Winner``, ``ScoreA``, ``ScoreB``, ``Forfeit`` ;          |
|                        | ``Text`` (remarque facultative, ``WithNote``, conservée au journal)     |
+------------------------+-------------------------------------------------------------------------+
| ``result_corrected``   | idem                                                                    |
+------------------------+-------------------------------------------------------------------------+
| ``match_cancelled``    | ``MatchID``                                                             |
+------------------------+-------------------------------------------------------------------------+
| ``bye``                | ``Phase``, ``ID``, ``Round``, ``Label``                                 |
+------------------------+-------------------------------------------------------------------------+
| ``draw``               | ``Phase``, ``Section`` (barrage), ``Draw`` (obligatoire), ``Label``     |
+------------------------+-------------------------------------------------------------------------+
| ``next_phase``         | —                                                                       |
+------------------------+-------------------------------------------------------------------------+
| ``repechage``          | ``Phase`` (la phase de poules), ``Section`` (la poule), ``A`` (le       |
|                        | qualifié retiré), ``B`` (le repêché)                                    |
+------------------------+-------------------------------------------------------------------------+
| ``length_changed``     | ``Phase``, ``Length``                                                   |
+------------------------+-------------------------------------------------------------------------+
| ``config_changed``     | ``Config`` (obligatoire, la configuration **entière**), ``Version``     |
+------------------------+-------------------------------------------------------------------------+
| ``reopened``           | —                                                                       |
+------------------------+-------------------------------------------------------------------------+
| ``table_changed``      | ``MatchID``, ``Table``                                                  |
+------------------------+-------------------------------------------------------------------------+
| ``finished``           | —                                                                       |
+------------------------+-------------------------------------------------------------------------+
| ``note``               | ``Text``                                                                |
+------------------------+-------------------------------------------------------------------------+

Sémantique de ``Apply``
-----------------------

``func (s *State) Apply(ev Event) error`` applique un événement. **Première règle** : si
``ev.Kind != EvCreated`` et ``s.Current < 0``, erreur « tournoi non créé ». Aucun événement ne peut
précéder la création.

Avant le traitement, ``s.clock = ev.Time``. Un ``Kind`` inconnu est une erreur
(« événement inconnu »).

À la fin de tout traitement réussi, et pour **tous** les types y compris ``note`` :

::

   si Kind ∈ {match_started, result, table_changed} → s.Warnings = check()
   refreshInfos()
   s.NEvents++
   si ev.Time > s.Last → s.Last = ev.Time

``check()`` est rappelée sur ces trois types parce qu'ils changent ce qu'elle regarde (les
joueurs qui occupent une place de graphe, un score au regard de la longueur, deux matchs sur une
même table) sans passer par ``recompute()``, qui l'appelle déjà. ``refreshInfos()`` recalcule
``State.Infos``.

``NEvents`` est donc le nombre d'événements appliqués avec succès ; il entre dans la graine du
générateur, ce qui garantit qu'une nouvelle proposition suit toujours un tirage différent.

``created``
~~~~~~~~~~~

::

   si ev.Config == nil                → erreur « created sans config »
   cfg := *ev.Config ; cfg.Validate() → propage l'erreur éventuelle
   s.Config, s.Seed = copie profonde de cfg, ev.Seed
   s.Phases = [ newPhaseState(0, cfg.Phases[0]) ]
   s.Current = 0

Un second ``created`` dans un journal écraserait la configuration ; le moteur ne l'interdit pas
explicitement, mais l'hôte ne doit jamais le produire.

``player_added``
~~~~~~~~~~~~~~~~

::

   si ev.Player == nil ou ev.Player.ID == "" → erreur « joueur sans identifiant »
   si l'identifiant est inconnu → l'ajouter à s.Order
   s.Players[id] = copie de *ev.Player     (réinscription = mise à jour de la fiche)
   supprimer id de s.Withdrawn             (une réinscription annule un forfait)

   ph := phase courante
   si ev.Slot != "" :
       takeSlot(phaseOf(ev.Phase), ev.Section, ev.Slot, id)   ← erreur si la place n'est plus libre
   sinon si ph.Index == 0 et non ph.Drawn et (ph.Kind == swiss_lives ou non ph.Started) :
       enter(ph, id, livesFor(ph.Cfg))

**Retardataires.** Trois chemins, et un seul interdit : refaire le tirage.

1. Le tirage n'a pas eu lieu (ou la phase est un suisse non figé) : le joueur entre tout de suite,
   avec toutes ses vies.
2. ``ev.Slot`` désigne une **place d'exemption libre** d'un tableau déjà tiré (``State.FreeSlots``
   les énumère, ``PlayerAddedAtSlotEvent`` construit l'événement) : il l'occupe là où elle est, avec
   une vie — il ne bénéficie pas de l'exemption qu'il prend. Une place dont le match a été lancé, ou
   dont le tour a commencé, ou qui n'existe pas, est **refusée** : ``Apply`` renvoie une erreur.
3. Sinon il est enregistré (``Players``, ``Order``) sans entrer dans aucune phase, et
   ``State.Infos`` porte un code disant où il entrera (``enters_at``) ou que rien ne l'admet
   (``no_entry``).

``player_updated``
~~~~~~~~~~~~~~~~~~

::

   si ev.Player == nil ou ev.Player.ID == "" → erreur « joueur sans identifiant »
   si l'identifiant est inconnu              → erreur (une inscription est un player_added)
   s.Players[id] = copie de *ev.Player

La **correction d'une fiche** — nom, club, cote — et rien d'autre. La fiche est entière, comme la
configuration de ``config_changed`` : ce qui est écrit est le résultat. L'état du joueur n'est pas
touché : un retiré reste retiré, ses vies, ses victoires, ses matchs et sa place dans les phases
sont intacts.

Ce n'est **pas une réinscription**, et c'est ce qui la distingue de ``player_added`` sur un
identifiant connu, qui met la fiche à jour **et** annule un retrait : corriger le club d'une
joueuse retirée par ``player_added`` la remettrait en jeu.
``player_added`` garde son sens pour les journaux existants — c'est aussi le chemin du retour d'un
joueur retiré.

``player_unavailable`` et ``player_available``
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

::

   si ID inconnu → erreur
   player_available   : supprimer s.Unavailable[ID]
   player_unavailable :
       a := Absence{Until: *ev.Until (si présent), Round: ev.Round, Phase: s.Current}
       si Until et Round sont tous deux renseignés → erreur (une heure OU une ronde)
       si Round > 0 et la phase courante n'est pas un suisse (swiss_lives) en mode "rounds" → erreur
       s.Unavailable[ID] = a                 (remplace une absence précédente)

Un joueur **indisponible** n'est plus apparié, et ne perd rien : ni vie, ni victoire, ni rang, ni
place dans un tableau. C'est le joueur qui ne revient que dimanche à 14 h, celui qui dîne, celui qui
joue en ce moment dans une autre épreuve de la salle. Les autres moyens ne conviennent pas : le
retirer puis le réinscrire le classe « forfait » pendant l'absence, ne rien faire le laisse
apparier, et un forfait lui coûte une vie.

.. code:: go

   type Absence struct {
       Until time.Time `json:"until,omitempty"` // heure de retour ; zéro = pas d'échéance horaire
       Round int       `json:"round,omitempty"` // suisse par rondes : ronde de retour
       Phase int       `json:"phase"`           // phase où elle a été déclarée (Round n'y vaut que là)
   }

L'absence tombe :

- à l'heure ``Until`` (comparée à l'heure de la proposition, ``ProposeAt(now)``, ou de l'événement
  en cours d'application) ;
- pour une absence « jusqu'à la ronde ``Round`` », quand la ronde appariée atteint ``Round``, ou
  au passage à une autre phase ;
- sans échéance, à ``player_available``, qui lève aussi toute absence avant son terme.

Effets, tous dérivés :

- **suisse** : ``free(ph, r)`` écarte les absents — en continu comme par rondes (la ronde ``r`` est
  celle qu'on apparie). Un absent n'est pas dans le ``Roster`` de la ronde : la ronde se termine
  sans lui, **sans bye**. La file porte une attente par absent en vie (``wait``, raison
  ``player_unavailable``, ``A`` = le joueur, ``Until`` ou ``Round`` = son retour) ;
- **graphes** (tableaux, GSL, poules, barrages) : la place est fixée, le match d'un absent reste
  donc proposé, mais **retenu** — raison ``player_unavailable``, sans table. Le TD peut le lancer
  quand même.

``player_withdrawn``
~~~~~~~~~~~~~~~~~~~~

::

   si le joueur est inconnu → erreur
   si ev.AfterCurrent et busy(id) :          (retrait différé)
       s.WithdrawAfter[id] = true ; fin (l'événement compte, NEvents++)
   supprimer id de s.WithdrawAfter
   s.Withdrawn[id] = true
   pour chaque match en cours où il joue (ordre de MatchOrder) :
       Status = Finished ; Forfeit = true ; End = ev.Time
       Winner = l'autre joueur
       onResult(match)
   recompute()

Le retrait est **général** : le joueur quitte le tournoi. Ses matchs de graphe non encore lancés
sont perdus par walkover lors de la résolution (voir :ref:`graphes`).

**Retrait différé** (``AfterCurrent``, ``PlayerWithdrawnAfterCurrentEvent``) : le joueur qui doit
partir finit le match qu'il joue. Il est inscrit dans ``WithdrawAfter`` ; occupé, il n'est plus
apparié ; ``promoteDeferred()``, appelée après chaque résultat, le fait passer dans ``Withdrawn``
dès qu'il n'a plus de match en cours. S'il n'a pas de match en cours, le retrait est immédiat.

Le **forfait d'un seul match**, sans quitter le tournoi, n'est pas un retrait : c'est un
``result`` avec ``Forfeit = true`` (``ForfeitEvent``), après lequel le perdant suit le chemin d'un
perdant ordinaire.

Un qualifié de poule qui se retire avant le tirage de la phase suivante ouvre un repêchage
(proposé, voir :ref:`format_poules`).

``match_started``
~~~~~~~~~~~~~~~~~

Validations, dans l'ordre :

::

   si MatchID vide ou déjà présent → erreur « identifiant invalide ou déjà utilisé »
   si A == B, ou A vide, ou B vide → erreur « joueurs invalides »
   pour p ∈ {A, B} :
       si p inconnu de s.Players → erreur « joueur inconnu »
       si busy(p)               → erreur « a déjà un match en cours »

``busy(p)`` : il existe un match de statut ``Running`` où ``p`` joue. **Un joueur ne peut jamais
avoir deux matchs en cours.** C'est l'invariant central de l'ordonnancement.

Puis :

::

   ph := phaseOf(ev.Phase)   (si elle existe)
       openRound(ph, ev.Round)   ← suisse par rondes, nouvelle ronde : relever ph.Roster
   créer Match{ID, Phase, Section, Label, Key, Round, A, B, Length, Table,
               Status: Running, Start: ev.Time}
   s.Matches[ID] = m ; s.MatchOrder += ID ; s.nextID++
   supprimer A et B de s.Revived

   ph (si elle existe) :
       ph.Started = true
       si ph.Kind == swiss_lives et ev.Round > ph.Round → ph.Round = ev.Round
       ph.Opponents[A] += B ; ph.Opponents[B] += A
       si un GMatch (Section, Key) existe → son MatchID = ID
   (puis check(), comme pour tout match_started)

``ev.Round`` (relu du libellé ``"Ronde %d"`` pour un journal de version 0, voir ``Event.upgraded``)
est le seul mécanisme qui fait avancer le compteur de rondes du suisse synchrone. ``openRound``
n'agit que dans un ``swiss_lives`` en mode ``rounds`` et pour ``ev.Round > ph.Round`` : au premier
événement d'une ronde (match ou bye) et avant de l'appliquer, il relève les joueurs libres de la
phase (``free(ph, r)``) ; ce sont les joueurs appelés à la ronde (voir :ref:`format_suisse`).

``nextMatchID()`` renvoie ``"M" + (nextID+1)``. Comme ``nextID`` n'est incrémenté que par
``match_started``, les identifiants sont consécutifs à partir de ``M1``, sans trou, y compris après
un rejeu partiel.

``result`` et ``result_corrected``
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

::

   m := s.Matches[MatchID] ; si absent → erreur « match inconnu »
   si Kind == result et m.Status == Finished
        → erreur « match déjà terminé (utiliser result_corrected) »
   si Winner ∉ {m.A, m.B} → erreur « vainqueur absent du match »

   m.Status = Finished ; m.Winner, m.ScoreA, m.ScoreB, m.Forfeit = ev.…
   si m.End est nul OU Kind == result → m.End = ev.Time

   si Kind == result_corrected → avant := aliveInSwiss() ; recompute() ; noteRevived(avant)
   sinon                       → onResult(m)

   promoteDeferred()          (un retrait différé prend effet si le joueur est libre)
   si m.A ou m.B est maintenant retiré → recompute()

``aliveInSwiss`` relève les joueurs en vie de chaque phase ``swiss_lives``. ``noteRevived`` compare
avant et après : un joueur que la correction rend à la vie entre dans ``State.Revived``, un joueur
qu'elle élimine en sort ; les avertissements sont recalculés (voir :ref:`etat`, « Contrôle de
cohérence »). ``match_started`` retire ses deux joueurs de ``Revived``.

Une correction sur un match **annulé** le ramène à l'état ``Finished`` : c'est le moyen de revenir
sur une annulation.

Noter que ``result_corrected`` sur un match jamais terminé fonctionne aussi : il vaut alors saisie
initiale suivie d'un recalcul complet (plus coûteux, mais correct).

``match_cancelled``
~~~~~~~~~~~~~~~~~~~

::

   m := s.Matches[MatchID] ; si absent → erreur
   m.Status = Cancelled ; m.End = ev.Time
   avant := aliveInSwiss() ; recompute() ; noteRevived(avant)

Le match reste dans ``MatchOrder`` et dans ``Matches`` : le journal ne perd rien. Il est simplement
ignoré par tous les calculs (comptabilité, adversaires rencontrés, graphes).

``bye``
~~~~~~~

::

   ph := phaseOf(ev.Phase) ; si absente → erreur « phase inconnue »
   openRound(ph, ev.Round)
   ph.Byes[ev.ID]++
   si ev.Round > 0 → ajouter ev.Round à la liste ph.ByeRounds[ev.ID]
   si ph.Kind == swiss_lives et ev.Round > ph.Round → ph.Round = ev.Round

Un bye ne compte ni victoire ni défaite. Il **appartient à sa ronde** : le confirmer seul ne clôt
pas la ronde, ses matchs restent proposés. Il est mémorisé pour éviter d'en donner deux au même
joueur (voir :ref:`format_suisse`).

``draw``
~~~~~~~~

::

   ph := phaseOf(ev.Phase) ; si absente → erreur
   si ev.Draw == nil       → erreur « draw sans tirage »
   applyDraw(ph, ev.Section, ev.Draw)

``applyDraw`` dispatche selon le format (voir :ref:`formats`) :

+--------------------------------+-----------------------------------------------------------------+
| Kind                           | Traitement                                                      |
+================================+=================================================================+
| ``bracket``, ``lives_bracket`` | vérifie que ``len(Slots)`` est une puissance de 2 ≥ 2, puis     |
|                                | ``buildBracket``                                                |
+--------------------------------+-----------------------------------------------------------------+
| ``gsl``                        | ``applyGSLDraw`` : nouveau bloc de groupes                      |
+--------------------------------+-----------------------------------------------------------------+
| ``round_robin``                | ``applyRRDraw`` : poules ``poule:A``, ``poule:B``… à partir de  |
|                                | ``Groups`` (``Section`` vide), ou barrage (``Section =``        |
|                                | ``"barrage:<lettre>"``, un seul groupe dans ``Groups``, poule   |
|                                | ``poule:<lettre>`` existante)                                   |
+--------------------------------+-----------------------------------------------------------------+
| autre                          | erreur « la phase n'a pas de tirage »                           |
+--------------------------------+-----------------------------------------------------------------+

``next_phase``
~~~~~~~~~~~~~~

::

   ph := phase courante
   si ph == nil ou s.Current+1 >= len(Config.Phases) → erreur « pas de phase suivante »
   ph.Done = true
   next := newPhaseState(s.Current+1, Config.Phases[s.Current+1])
   s.Phases += next ; s.Current++
   enterFrom(next, ph)

``repechage``
~~~~~~~~~~~~~

::

   ph := Phases[ev.Phase] ; erreur si absente ou si ce n'est pas une phase de poules
   erreur si repêchage fermé (voir :ref:`format_poules`) :
       pas de phase après les poules, phase suivante en entrée « all »,
       poules ou barrages non terminés, phase suivante déjà tirée ou commencée
   sec := la poule ev.Section (``poule:<lettre>``) ; erreur si inconnue
   erreur si A n'est pas, dans sec, un qualifié (rrQualified) retiré
   erreur si B n'est pas, dans sec, un joueur ni qualifié ni retiré
   ph.Repechages[A] = B
   si la phase suivante est déjà ouverte (passage fait, tirage pas encore) :
       retirer A de ses entrants ; enter(next, B, entryLives(next, ph, B))

B n'est pas forcément le joueur proposé : le TD peut choisir un autre joueur non qualifié de la
poule (entre ex æquo, typiquement). Sans ``repechage`` confirmé, la place du retiré devient une
exemption du tableau suivant ; un journal sans ``repechage`` se rejoue donc à l'identique.

``length_changed``
~~~~~~~~~~~~~~~~~~

::

   ph := phaseOf(ev.Phase) ; si absente → erreur
   ph.Length = ev.Length

Change la longueur des **matchs futurs** de la phase. Les matchs déjà lancés gardent leur longueur ;
les ``GMatch`` déjà construits (tableau tiré) gardent la leur aussi, puisqu'elle a été figée à la
construction du graphe. Le champ n'a donc d'effet immédiat que sur les formats à appariement
dynamique (suisse, barrages) et sur les graphes construits **après** le changement (blocs GSL
suivants). ``config_changed`` peut aussi changer une longueur ; ``length_changed`` reste lu pour
les journaux existants.

``config_changed``
~~~~~~~~~~~~~~~~~~

::

   si ev.Config == nil → erreur
   cfg := copie profonde de *ev.Config ; cfg.Validate() → erreur éventuelle
   acceptConfig(cfg, ev.Version) → erreur éventuelle (un *ConfigRefusal)
   setConfig(cfg) ; recompute()

L'événement porte la configuration **entière**, et non le champ à changer : ce qui est écrit dans le
journal est le résultat, pas l'instruction qui y mène — le même choix que pour ``draw``. Un journal
se relit alors sans connaître la règle de composition des retouches successives.

``acceptConfig`` refuse trois choses, et seulement trois :

- une configuration qui a **moins de phases** que le tournoi n'en a ouvertes (une phase ouverte ne
  se retire pas) ;
- un changement de ``Kind`` sur une phase **terminée** (indice < ``Current``), **tirée** ou
  **commencée** (raison testée dans cet ordre) ;
- sur un tableau (``bracket``, ``lives_bracket``) **déjà tiré**, un changement de l'une des options
  qui ne servent qu'à construire le graphe au tirage : ``consolation``, ``last_chance``,
  ``reconciliation``, ``recharge``, ``seeding``. Le graphe est fait ; les changer ne pourrait plus
  rien produire : sans ce refus, cocher la consolante d'un tableau tiré serait accepté, écrit dans
  la configuration, et sans effet.

Le refus est un ``*ConfigRefusal``, qui nomme la phase, le champ et la raison par des codes :

.. code:: go

   type ConfigRefusal struct {
       Phase  int    `json:"phase"`            // la phase qui bloque (removed : len(next.Phases))
       Field  string `json:"field"`            // champ JSON de PhaseConfig : "kind", "consolation"… ;
                                               // "phases" pour un retrait
       Reason string `json:"reason"`           // RefusalRemoved "removed", RefusalFinished "finished",
                                               // RefusalDrawn "drawn", RefusalStarted "started"
       From   string `json:"from,omitempty"`   // kind : l'ancien format
       To     string `json:"to,omitempty"`     // kind : le nouveau format
       Opened int    `json:"opened,omitempty"` // removed : le nombre de phases déjà ouvertes
   }

``Error()`` en donne le rendu français (console, démo) ; un hôte multilingue traduit les codes.

**Journaux anciens.** Le troisième refus n'existe qu'à partir de la **version 2** du journal. Un
``config_changed`` de version 0 ou 1 qui changeait ces options après le tirage a été joué ainsi —
accepté, sans effet — et se rejoue ainsi : ``acceptConfig`` reçoit la version de l'événement et
n'applique le verrou du tirage qu'à partir de 2.

.. code:: go

   func (s *State) CheckConfig(next Config) error

``CheckConfig`` est la **prévisualisation** : ``Validate`` puis ``acceptConfig`` sur une copie, aux
règles de la version courante, sans rien appliquer. C'est ce qu'un formulaire de réglages appelle
pour griser ou expliquer un champ avant d'écrire l'événement.

Tout le reste est accepté, y compris ce que le moteur ne peut pas juger : la bascule (``target``),
les longueurs à venir, les tables, les pauses, la dotation, et une phase **ajoutée après** la phase
courante.

``setConfig`` répercute la nouvelle configuration sur les ``PhaseState`` déjà ouverts. La longueur
courante d'une phase (``PhaseState.Length``) ne suit la configuration que si la configuration l'a
**effectivement changée** : sinon, une retouche qui ne touche qu'à la bascule effacerait le
``length_changed`` que le TD venait de saisir à la main. Quand le ``Kind`` d'une phase non commencée
change, sa longueur courante reprend celle de la configuration et les vies de ses entrants sont
recalculées (``livesFor`` : ``Lives`` pour un suisse, 2 pour un GSL, 1 sinon).

``reopened``
~~~~~~~~~~~~

::

   si !s.Finished → erreur « le tournoi n'est pas clos »
   s.Finished = false ; s.Final = nil

Rouvre un tournoi clos, parce qu'un résultat était faux. Le classement final figé est effacé et sera
recalculé à la clôture suivante ; le journal, lui, garde tout — la clôture, la réouverture, la
correction et la nouvelle clôture sont quatre événements.

``table_changed``
~~~~~~~~~~~~~~~~~

::

   m := s.Matches[MatchID] ; si absent → erreur « match inconnu »
   m.Table = ev.Table
   check()

Le déplacement est **toujours accepté**, y compris vers une table qui porte déjà un match en cours :
le moteur avertit, il ne bloque pas. ``check`` lève alors ``table_shared`` (voir :ref:`etat`), qui tombe de lui-même dès que l'un des deux matchs se termine ou repart ailleurs.
Refuser aurait laissé le TD sans recours dans la salle pleine où il en a justement besoin — un
plateau cassé, quatorze tables sur quatorze occupées, deux joueurs qui finissent sur un coin de
table en attendant.

``finished``
~~~~~~~~~~~~

::

   s.Finished = true
   s.Final = s.Ranking()

Le classement est **figé** dans ``Final``. ``StandingsCSV`` et les fonctions de rendu utilisent
``Final`` s'il existe, sinon recalculent ``Ranking()``.

``note``
~~~~~~~~

Sans effet, sinon l'incrément de ``NEvents`` (ce qui décale le générateur aléatoire — une note
change donc le prochain tirage : c'est sans conséquence sur la validité, mais à savoir).

Construction d'événements
-------------------------

.. code:: go

   func (s *State) EventFromAction(a Action, now time.Time) (Event, error)
   func ResultEvent(id MatchID, winner PlayerID, scoreA, scoreB int, now time.Time) Event

   // constructeurs d'événements (ils posent Version = JournalVersion ; ne jamais écrire un
   // Event littéral sans version, il serait relu comme un journal ancien)
   func PlayerAddedEvent(p Player, now time.Time) Event
   func PlayerAddedAtSlotEvent(p Player, slot Slot, now time.Time) Event
   func PlayerUpdatedEvent(p Player, now time.Time) Event          // correction de fiche
   func PlayerUnavailableEvent(id PlayerID, until time.Time, now time.Time) Event // until nul = jusqu'à nouvel ordre
   func PlayerUnavailableUntilRoundEvent(id PlayerID, round int, now time.Time) Event
   func PlayerAvailableEvent(id PlayerID, now time.Time) Event
   func PlayerWithdrawnEvent(id PlayerID, now time.Time) Event
   func PlayerWithdrawnAfterCurrentEvent(id PlayerID, now time.Time) Event
   func ForfeitEvent(id MatchID, winner PlayerID, now time.Time) Event
   func RepechageEvent(phase int, section string, withdrawn, replacement PlayerID, now time.Time) Event
   func CorrectionEvent(id MatchID, winner PlayerID, scoreA, scoreB int, now time.Time) Event
   func CancelEvent(id MatchID, now time.Time) Event
   func LengthChangedEvent(phase, length int, now time.Time) Event
   func ConfigChangedEvent(cfg Config, now time.Time) Event
   func ReopenedEvent(now time.Time) Event
   func TableChangedEvent(id MatchID, table int, now time.Time) Event
   func NoteEvent(text string, now time.Time) Event
   func (e Event) WithNote(text string) Event

``EventFromAction`` traduit une action confirmée en événement. Base commune :
``Event{Version: JournalVersion, Time: now, Phase: a.Phase, Section: a.Section, Label: a.Label,
Round: a.Round, Key: a.Key}``, puis :

+------------------+-------------------------------------------------------------------------------+
| Action           | Événement produit                                                             |
+==================+===============================================================================+
| ``start_match``  | ``match_started`` avec ``MatchID = s.nextMatchID()``, ``A``, ``B``,           |
|                  | ``Length``, ``Table``                                                         |
+------------------+-------------------------------------------------------------------------------+
| ``bye``          | ``bye`` avec ``ID = a.A``                                                     |
+------------------+-------------------------------------------------------------------------------+
| ``draw``         | ``draw`` avec ``Draw = a.Draw``                                               |
+------------------+-------------------------------------------------------------------------------+
| ``cancel_match`` | ``match_cancelled`` avec ``MatchID = a.Match``                                |
+------------------+-------------------------------------------------------------------------------+
| ``next_phase``   | ``next_phase``                                                                |
+------------------+-------------------------------------------------------------------------------+
| ``repechage``    | ``repechage`` avec ``A = a.A`` (le retiré), ``B = a.B`` (le repêché)          |
+------------------+-------------------------------------------------------------------------------+
| ``finish``       | ``finished``                                                                  |
+------------------+-------------------------------------------------------------------------------+
| autre (``wait``) | erreur « action sans événement associé »                                      |
+------------------+-------------------------------------------------------------------------------+

**Attention** : l'identifiant de match est attribué au moment de l'appel. Si le TD confirme
plusieurs actions ``start_match`` d'un même lot, il doit appeler ``EventFromAction`` puis ``Apply``
**pour chacune, dans l'ordre**, sans quoi deux matchs recevraient le même identifiant et le second
``Apply`` échouerait.

``ResultEvent`` construit ``{Version, Kind: result, MatchID, Winner, ScoreA, ScoreB, Time}``.
``ForfeitEvent`` est le même événement avec ``Forfeit = true`` et sans score ;
``CorrectionEvent`` a les champs de ``ResultEvent`` avec ``Kind = result_corrected``.
``WithNote`` attache une remarque (``Text``) à un événement de résultat.

Boucle d'intégration côté hôte
------------------------------

.. code:: go

   cfg := tournoi.Config{Name: "Open", Phases: []tournoi.PhaseConfig{
       {Kind: tournoi.KindSwissLives, Length: 7, Target: 16},
       {Kind: tournoi.KindLivesBracket, Length: 9, FinalLength: 11},
   }}
   st, created, err := tournoi.New(cfg, seed, time.Now())
   journal := tournoi.Journal{created}

   // inscriptions
   ev := tournoi.PlayerAddedEvent(p, time.Now())
   st.Apply(ev); journal = append(journal, ev)

   // boucle du TD
   for _, a := range st.Propose() {         // « lancer P3 contre P8 en 7 points, table 4 »
       ev, _ := st.EventFromAction(a, time.Now())   // le TD confirme
       st.Apply(ev); journal = append(journal, ev)
   }

   // résultat saisi par le TD
   ev = tournoi.ResultEvent("M12", "P3", 7, 4, time.Now())
   st.Apply(ev); journal = append(journal, ev)

``func (s *State) Step(evs ...Event) ([]Action, error)`` applique une suite d'événements puis
renvoie ``Propose()`` (il s'arrête à la première erreur) ; c'est un raccourci, pas une primitive.
L'intégration complète est décrite dans :ref:`integration`.

**Règle d'or de l'hôte** : un événement n'est ajouté au journal que si ``Apply`` a réussi. Un
``Apply`` en échec laisse l'état inchangé pour les validations en tête de traitement ; pour les
traitements qui modifient l'état avant de pouvoir échouer (``player_added`` sur une place
refusée : le joueur est déjà enregistré quand ``takeSlot`` échoue), l'hôte doit repartir d'un
``Replay``.

Exemple de journal
------------------

Extrait d'un journal produit par ``tournoi-demo -format suisse_tableau`` (champs ``config`` et
horodatages abrégés) :

.. code:: json

   [
    {"seq": 0, "version": 2, "kind": "created", "time": "2026-01-01T09:00:00Z",
     "config": {"name": "Suisse 2 vies puis tableau à vies",
                "phases": [{"kind": "swiss_lives", "lives": 2, "length": 7, "mode": "continuous",
                            "pairing": "random", "target": 16, "entry": "survivors"},
                           {"kind": "lives_bracket", "length": 9, "final_length": 11,
                            "entry": "survivors"}],
                "min_per_point": 8},
     "seed": 1, "label": {}},
    {"seq": 1, "version": 2, "kind": "player_added", "time": "2026-01-01T09:00:00Z",
     "player": {"id": "P1", "name": "Joueur 1", "club": "Club 0", "rating": 3.5}, "label": {}},
    {"seq": 7, "version": 2, "kind": "next_phase", "time": "2026-01-01T09:00:00Z",
     "label": {"kind": "phase", "text": "Tableau final"}},
    {"seq": 8, "version": 2, "kind": "draw", "time": "2026-01-01T09:00:00Z", "phase": 1,
     "label": {"kind": "draw_bracket", "n": 16, "sub": {"kind": "phase", "text": "Tableau final"}},
     "draw": {"slots": ["P1", "BYE", "P3", "BYE", "P4", "BYE", "P2", "BYE",
                        "P6", "BYE", "BYE", "BYE", "P5", "BYE", "BYE", "BYE"],
              "lives": {"P1": 2, "P2": 2, "P3": 2, "P4": 2, "P5": 2, "P6": 2}}},
    {"seq": 9, "version": 2, "kind": "match_started", "time": "2026-01-01T09:00:00Z",
     "match_id": "M1", "phase": 1, "section": "main", "key": "main.1.0",
     "label": {"kind": "quarter_final"}, "a": "P1", "b": "P3", "length": 9, "table": 1},
    {"seq": 12, "version": 2, "kind": "result", "time": "2026-01-01T09:32:17Z",
     "match_id": "M1", "label": {}, "winner": "P1", "score_a": 9, "score_b": 7},
    {"seq": 19, "version": 2, "kind": "finished", "time": "2026-01-01T11:20:29Z",
     "phase": 1, "label": {}}
   ]
