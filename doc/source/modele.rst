.. _modele:

Modèle de données
=================

Tous les types de cette page appartiennent au paquet ``tournoi``.

Identifiants et joueurs
-----------------------

.. code:: go

   type PlayerID string          // fourni par l'hôte ; unique dans un tournoi
   const BYE PlayerID = "BYE"    // place vide réservée dans un tableau

   type Player struct {
       ID     PlayerID `json:"id"`
       Name   string   `json:"name"`
       Club   string   `json:"club,omitempty"`
       Rating float64  `json:"rating,omitempty"` // PR (plus bas = meilleur) ; 0 = inconnu
   }

``BYE`` est une valeur réservée : aucun joueur inscrit ne peut porter cet identifiant. Le moteur ne
le vérifie pas (``player_added`` ne refuse qu'un identifiant vide) ; c'est à l'hôte de l'interdire.

Le ``Rating`` n'intervient ni dans l'appariement d'un suisse, d'un bloc GSL ou d'une poule, ni dans
le classement. Il sert dans trois cas seulement :

- le tirage d'un tableau dont la phase a ``seeding: "rating"`` (placement classique par cote,
  ``rules.SeedOrder``) ; par défaut (``seeding`` vide), le tirage est aléatoire et ignore la cote,
  conformément au principe « pas de têtes de série protégées » ;
- la simulation (``sim.Run``) et la prévision de fin (``sim.Forecast``), où une cote inconnue est
  remplacée par la moyenne des cotes connues, ou 6 (``rules.MeanRating``) ;
- l'affichage.

Match
-----

.. code:: go

   type MatchID string           // attribué par le moteur : "M1", "M2", …

   type MatchStatus string
   const (
       Running   MatchStatus = "running"
       Finished  MatchStatus = "finished"
       Cancelled MatchStatus = "cancelled"
   )

   type Match struct {
       ID      MatchID     `json:"id"`
       Phase   int         `json:"phase"`             // indice de la phase (0-based)
       Section string      `json:"section,omitempty"` // identifiant : "main", "conso", "poule:A", "B1G3"…
       Label   Label       `json:"label,omitempty"`   // libellé structuré (codes.go)
       Key     string      `json:"key,omitempty"`     // clé du GMatch dans sa section
       Round   int         `json:"round,omitempty"`   // ronde suisse (0 = sans objet)
       A       PlayerID    `json:"a"`
       B       PlayerID    `json:"b"`
       Length  int         `json:"length"`            // en points
       Table   int         `json:"table,omitempty"`
       Status  MatchStatus `json:"status"`
       Start   time.Time   `json:"start"`
       End     time.Time   `json:"end,omitempty"`
       Winner  PlayerID    `json:"winner,omitempty"`
       ScoreA  int         `json:"score_a,omitempty"`
       ScoreB  int         `json:"score_b,omitempty"`
       Forfeit bool        `json:"forfeit,omitempty"`
   }

Méthodes :

- ``Loser() PlayerID`` : renvoie ``B`` si ``Winner == A``, sinon ``A``. **N'est valide que pour un
  match terminé** ; sur un match non terminé elle renvoie ``A`` (puisque ``Winner`` est vide).
- ``Has(p PlayerID) bool`` : ``A == p || B == p``.

Un match est relié à son emplacement dans un graphe par le couple (``Section``, ``Key``).
``Key`` est vide pour un match libre, apparié dynamiquement hors graphe : suisse (continu ou par
rondes), finale GSL et barrage. ``Section`` est vide aussi pour le suisse et la finale GSL ; un
match de barrage porte le nom de sa section (``barrage:A``), sans ``Key``.

``Section`` est un **identifiant**, jamais un texte à afficher : ``main``, ``conso``, ``last``,
``gf`` (tableaux), ``B<bloc>G<groupe>`` (GSL), ``poule:A``, ``poule:B``… (poules),
``barrage:A``… (barrage de la poule A). ``Label`` est un libellé **structuré** (``Label{Kind, N,
Losses, Match, Section, Text, Players, Spots, Sub}``, ``codes.go``) que l'hôte traduit ; seul
``Text`` porte du texte, et seulement ce que le TD a saisi (nom de phase).

Rang
----

.. code:: go

   type Rank struct {
       Player PlayerID `json:"player"`
       Rank   int      `json:"rank"`         // 1 = premier ; les ex æquo partagent le rang
       Note   Note     `json:"note,omitempty"` // note structurée (codes.go)
   }

   type Note struct {
       Kind      NoteKind `json:"kind,omitempty"`
       Wins      int      `json:"wins,omitempty"`
       Losses    int      `json:"losses,omitempty"`
       Lives     int      `json:"lives,omitempty"`
       Section   string   `json:"section,omitempty"`
       Qualified bool     `json:"qualified,omitempty"`
       Sub       *Label   `json:"sub,omitempty"`
   }

La note est un **code** et ses paramètres, jamais une phrase : ``winner``, ``finalist``,
``alive`` (``Lives``), ``record`` (``Wins``, ``Losses``), ``forfeit``, ``withdrawn``, ``running``,
``awaiting_draw``, ``unranked``, ``section_exit`` (``Section`` + ``Sub``), ``section_winner``,
``pool_record`` (``Section``, ``Wins``, ``Qualified``) — valeurs exactes dans ``rules/codes.go``.
Le rendu français (``Note.String()``, ``fr.go``) ne sert que la console, la démo et ``render``.
Voir :ref:`classement`.

Action
------

.. code:: go

   type ActionKind string
   const (
       ActStartMatch  ActionKind = "start_match"  // lancer un match
       ActBye         ActionKind = "bye"          // donner un bye (ronde synchrone)
       ActDraw        ActionKind = "draw"         // effectuer un tirage (le tirage est joint)
       ActNextPhase   ActionKind = "next_phase"   // passer à la phase suivante
       ActCancelMatch ActionKind = "cancel_match" // annuler un match devenu incohérent
       ActRepechage   ActionKind = "repechage"    // repêcher B à la place du qualifié de poule A, retiré
       ActFinish      ActionKind = "finish"       // clore le tournoi
       ActWait        ActionKind = "wait"         // rien à faire : attendre
   )

   type Action struct {
       Kind    ActionKind  `json:"kind"`
       Phase   int         `json:"phase"`
       Section string      `json:"section,omitempty"`
       Label   Label       `json:"label,omitempty"`
       Round   int         `json:"round,omitempty"`
       Key     string      `json:"key,omitempty"`
       Match   MatchID     `json:"match,omitempty"`  // cancel_match : le match à annuler
       A       PlayerID    `json:"a,omitempty"`
       B       PlayerID    `json:"b,omitempty"`
       Length  int         `json:"length,omitempty"`
       Table   int         `json:"table,omitempty"`
       Draw    *Draw       `json:"draw,omitempty"`
       Reason  ReasonCode  `json:"reason,omitempty"` // ActWait : pourquoi
       Until   time.Time   `json:"until,omitempty"`  // waiting_batch : échéance du prochain lot
       Warn    WarningCode `json:"warn,omitempty"`   // ce que le moteur remarque sans rien bloquer
   }

``Label``, ``Reason`` (``matches_running``, ``no_pairing``, ``waiting_batch``, ``waiting_table``,
``player_unavailable``, ``player_busy``) et ``Warn`` sont des codes. ``Action.String()`` (rendu
français, pour la console) produit une phrase destinée au TD :

================ =====================================================================
Kind             Phrase
================ =====================================================================
``start_match``  ``Lancer <Label> : <A> contre <B> en <Length> points, table <Table>``,
                 suivi de « — <avertissement> » si ``Warn`` est posé
``bye``          ``Bye pour <A> (<Label>)``
``draw``         ``Tirage : <Label>``
``next_phase``   ``Passer à la phase suivante : <Label>``
``cancel_match`` ``Annuler <Match> : <A> contre <B> (<Label>), devenu incohérent``
``repechage``    ``<Label> : <B> à la place de <A>, retiré``
``finish``       ``Clore le tournoi``
(autre)          ``Attendre : <Reason>`` ; avec ``waiting_batch`` et ``Until`` :
                 ``Attendre : <Reason> (jusqu'à HH:MM)``
================ =====================================================================

Tirage
------

.. code:: go

   type Draw struct {
       Slots  []PlayerID       `json:"slots,omitempty"`  // places du 1er tour (BYE = vide)
       Groups [][]PlayerID     `json:"groups,omitempty"` // groupes (GSL, poules, barrage)
       Lives  map[PlayerID]int `json:"lives,omitempty"`  // vies à l'entrée (information)
   }

``Slots`` a **toujours une longueur puissance de 2 ≥ 2** ; la paire *i* du premier tour est
``(Slots[2i], Slots[2i+1])``.

``Lives`` est purement informatif (affichage, audit) : il n'est pas relu au rejeu.

Le tirage étant **matérialisé** dans l'événement, l'algorithme qui le produit (``rules.DrawSlots``,
placement par cote avec ``seeding: "rating"``, composition des groupes) peut changer sans casser un
journal existant : le rejeu relit les places, il ne refait pas le tirage.

État
----

.. code:: go

   type State struct {
       Config     Config               `json:"config"`
       Seed       int64                `json:"seed"`
       Players    map[PlayerID]*Player `json:"players"`
       Order      []PlayerID           `json:"order"`     // ordre d'inscription
       Withdrawn  map[PlayerID]bool    `json:"withdrawn,omitempty"`
       WithdrawAfter map[PlayerID]bool `json:"withdraw_after,omitempty"` // retraits différés
       Matches    map[MatchID]*Match   `json:"matches"`
       MatchOrder []MatchID            `json:"match_order"` // ordre de lancement
       Phases     []*PhaseState        `json:"phases"`   // uniquement les phases atteintes
       Current    int                  `json:"current"`  // phase en cours ; -1 = néant
       Finished   bool                 `json:"finished"`
       Final      []Rank               `json:"final,omitempty"` // figé à la clôture, effacé par reopened
       NEvents    int                  `json:"n_events"`
       Last       time.Time            `json:"last"`     // horodatage du dernier événement
       Warnings   []Warning            `json:"warnings,omitempty"` // incohérences (codes)
       Infos      []Info               `json:"infos,omitempty"`    // inscrits qui ne jouent nulle part, et où ils entreront
       Revived    map[PlayerID]bool    `json:"revived,omitempty"`  // rendus à la vie par une correction, jusqu'à ce qu'ils rejouent
       Unavailable map[PlayerID]Absence `json:"unavailable,omitempty"` // indisponibilités déclarées
       nextID     int                  // interne : compteur de matchs
       clock      time.Time            // interne : heure de la proposition ou de l'événement appliqué
       ailleurs   map[PlayerID]bool    // interne : joueurs occupés dans une autre épreuve (ProposeWith)
   }

   type Absence struct {
       Until time.Time `json:"until,omitempty"` // heure de retour ; zéro = pas d'échéance horaire
       Round int       `json:"round,omitempty"` // suisse par rondes : ronde de retour
       Phase int       `json:"phase"`           // phase où l'absence a été déclarée
   }

``WithdrawAfter`` tient les **retraits différés** (``player_withdrawn`` avec ``after_current``) : le
joueur finit le match qu'il joue, n'est plus apparié, et passe dans ``Withdrawn`` dès que ce match
est terminé. Un retrait différé sans match en cours est un retrait immédiat.

``Unavailable`` tient les indisponibilités (``player_unavailable``), levées par
``player_available``, par l'heure ``Until`` ou par la ronde de retour ``Round``.

``Warnings`` et ``Infos`` sont des listes de codes (``Warning{Code, Match, Section, Label, A, B,
ExpectedA, ExpectedB, Length, ScoreA, ScoreB, Player, Table, Other}``, ``Info{Code, Player, Phase,
Section, Label}``) dérivées de l'état ; voir :ref:`etat`.

``Phases`` ne contient que les phases **atteintes** : à la création, une seule (``Phases[0]``) ; une
phase est ajoutée à chaque ``next_phase``. ``Config.Phases`` contient la liste complète prévue.

.. code:: go

   type PhaseState struct {
       Index     int                     `json:"index"`
       Cfg       PhaseConfig             `json:"cfg"`
       Entrants  []PlayerID              `json:"entrants"`  // ordre d'entrée dans la phase
       Lives     map[PlayerID]int        `json:"lives"`     // vies à l'entrée
       Losses    map[PlayerID]int        `json:"losses"`
       Wins      map[PlayerID]int        `json:"wins"`
       Byes      map[PlayerID]int        `json:"byes"`
       Opponents map[PlayerID][]PlayerID `json:"opponents"` // adversaires déjà rencontrés
       ElimOrder []PlayerID              `json:"elim_order"`// ordre d'élimination
       Round     int                     `json:"round"`     // ronde synchrone / bloc GSL
       Roster    []PlayerID              `json:"roster,omitempty"`     // suisse par rondes : appelés à la ronde Round
       ByeRounds map[PlayerID][]int      `json:"bye_rounds,omitempty"` // rondes où le joueur a eu un bye
       Drawn     bool                    `json:"drawn"`     // le tirage a eu lieu
       Done      bool                    `json:"done"`
       Sections  []*Section              `json:"sections,omitempty"`
       Length    int                     `json:"length"`    // longueur courante des matchs
       Started   bool                    `json:"started"`   // au moins un match lancé
       Repechages map[PlayerID]PlayerID  `json:"repechages,omitempty"` // poules : qualifié retiré → repêché
   }

Un ``PhaseState`` neuf est créé par ``newPhaseState(i, cfg)`` : ``Lives``, ``Losses``, ``Wins``,
``Byes``, ``ByeRounds`` et ``Opponents`` initialisées vides, ``Length = cfg.Length``, tout le reste à
zéro (``Repechages`` reste ``nil`` jusqu'au premier repêchage).

Graphe de matchs
----------------

.. code:: go

   type Section struct {
       Name    string     `json:"name"`             // identifiant : "main", "conso", "B2G3", "poule:A", "barrage:A"…
       Kind    string     `json:"kind"`         // main|conso|last|gf|gsl|se|poule|barrage
       Group   int        `json:"group,omitempty"`  // numéro de groupe/poule
       Block   int        `json:"block,omitempty"`  // GSL : numéro de bloc
       Matches []GMatch   `json:"matches"`
       Rounds  [][]int    `json:"rounds,omitempty"` // indices de Matches par tour
       Players []PlayerID `json:"players,omitempty"`// barrage : joueurs concernés
       Spots   int        `json:"spots,omitempty"`  // barrage : places à attribuer
   }

   type GMatch struct {
       Key      string      `json:"key"`      // unique dans la section
       Label    Label       `json:"label,omitempty"`  // libellé structuré
       Length   int         `json:"length"`
       Src      [2]Src      `json:"src"`      // origine des deux places
       Players  [2]PlayerID `json:"players"`  // places résolues ("" = inconnue)
       MatchID  MatchID     `json:"match_id,omitempty"` // match réel associé
       Winner   PlayerID    `json:"winner,omitempty"`
       Loser    PlayerID    `json:"loser,omitempty"`
       Done     bool        `json:"done"`
       Walkover bool        `json:"walkover,omitempty"` // gagné sans jouer (BYE, forfait)
       Skipped  bool        `json:"skipped,omitempty"`  // conditionnel non joué (recharge)
       CondFrom int         `json:"cond_from,omitempty"`
       CondSide int         `json:"cond_side,omitempty"`
       Cond     bool        `json:"cond,omitempty"`
   }

   type Src struct {
       Player  PlayerID `json:"player,omitempty"`  // place fixée
       From    int      `json:"from"`        // sinon : indice du match source ; -1 = aucun
       Section string   `json:"section,omitempty"` // section source ("" = la même)
       Loser   bool     `json:"loser,omitempty"` // prendre le perdant au lieu du vainqueur
   }

Règles :

- une ``Src`` est **soit** ``Player`` non vide avec ``From = -1`` (place fixée), **soit**
  ``From ≥ 0`` (place héritée) ; ``From = -1`` avec ``Player`` vide signifie « place définitivement
  vide », ce qu'aucun constructeur ne produit (les places vides sont représentées par
  ``Player: BYE``) ;
- ``Section`` vide signifie « le match *From* de la section courante » ;
- une section est **acyclique** : un ``GMatch`` ne référence que des matchs d'indice inférieur dans
  sa propre section, ou des matchs d'une section construite avant elle.

**Match conditionnel** (``Cond``) : le match n'existe que si le vainqueur du match ``CondFrom``
(même section) est le joueur qui occupait la place ``CondSide`` de ce match. Sert exclusivement à la
recharge de la double élimination (``gf.1``, ``Cond: true, CondFrom: 0, CondSide: 1``) : la seconde
grande finale n'a lieu que si le vainqueur de la consolante (place 1 de ``gf.0``) a gagné la
première. Sinon le match est marqué ``Skipped``.

Le moteur garde ses propres ``Section``/``GMatch`` (sérialisés dans l'état) ; les constructeurs et
la résolution vivent dans ``rules/graphe.go`` (``rules.Section``, ``rules.Node``), et ``regles.go``
convertit dans les deux sens. Voir :ref:`graphes`.

Sérialisation
-------------

- ``Journal.Bytes() ([]byte, error)`` : ``json.MarshalIndent(j, "", " ")`` — un tableau JSON.
- ``ParseJournal([]byte) (Journal, error)``.
- ``ParseConfig([]byte) (*Config, error)`` : désérialise puis **valide** (voir :ref:`configuration`).

``State`` est sérialisable en JSON (champs exportés) mais **n'est pas** un format d'échange : il est
toujours reconstruit depuis le journal. Les champs privés (``nextID``, ``clock``, ``ailleurs``) ne
sont pas sérialisés ; ``nextID`` est recalculé par le rejeu (incrémenté à chaque ``match_started``),
et ``nextMatchID()`` donne ``"M<nextID+1>"``.

``ParseJournal`` ne convertit rien : c'est ``Replay`` qui applique ``Event.upgraded()`` à chaque
événement, pour qu'un journal de version 0 (libellés en texte) soit lu en codes. Voir
:ref:`journal`.
