.. _configuration:

Configuration
=============

Structures
----------

.. code:: go

   type Config struct {
       Name        string        `json:"name"`
       Phases      []PhaseConfig `json:"phases"`
       MinPerPoint float64       `json:"min_per_point,omitempty"` // minutes/point ; défaut 8
       Tables      Tables        `json:"tables,omitempty"`        // tables de la salle
       Prizes      PrizePool     `json:"prizes,omitempty"`        // dotation
       Breaks      []TimeRange   `json:"breaks,omitempty"`        // pauses programmées
   }

   type Tables struct {      // Count = 0 : illimité (numérotation sans borne)
       Count       int         `json:"count,omitempty"`
       Unavailable []int       `json:"unavailable,omitempty"` // plateau cassé, table retirée
       Reserved    []TableRule `json:"reserved,omitempty"`    // retransmission, table d'honneur
   }

   type TableRule struct {   // réserve une table à une section et/ou une phase
       Table     int    `json:"table"`
       Section   string `json:"section,omitempty"` // identifiant de section (main, conso, poule:A…)
       Phase     int    `json:"phase,omitempty"`   // index de phase ; 0 = toutes
       AllPhases bool   `json:"all_phases,omitempty"`
   }

   type PrizePool struct {
       EntryFee  float64               `json:"entry_fee,omitempty"` // droit d'entrée par inscrit
       Retention Retention             `json:"retention,omitempty"`
       Sections  map[string]PrizeScale `json:"sections,omitempty"`  // clé : "all", "main", "conso"…
   }

   type Retention struct {   // le pourcentage s'applique au pool brut, puis le montant s'en retranche
       Amount  float64 `json:"amount,omitempty"`
       Percent float64 `json:"percent,omitempty"`
   }

   type PrizeScale struct {  // l'un OU l'autre, place par place
       Percents []float64 `json:"percents,omitempty"` // % du pool distribuable
       Amounts  []float64 `json:"amounts,omitempty"`  // montants fixes
   }

   type TimeRange struct {   // Start incluse, End exclue
       Start time.Time `json:"start"`
       End   time.Time `json:"end"`
   }

   type PhaseConfig struct {
       Kind           string `json:"kind"`
       Name           string `json:"name,omitempty"`
       Lives          int    `json:"lives,omitempty"`
       Length         int    `json:"length"`
       FinalLength    int    `json:"final_length,omitempty"`
       Lengths        []int  `json:"lengths,omitempty"`
       LengthLate     int    `json:"length_late,omitempty"`
       LateThreshold  int    `json:"late_threshold,omitempty"`
       Mode           string `json:"mode,omitempty"`
       Pairing        string `json:"pairing,omitempty"`
       AvoidClubs     bool   `json:"avoid_clubs,omitempty"`
       AllowRematch   bool   `json:"allow_rematch,omitempty"`
       Target         int    `json:"target,omitempty"`
       Consolation    bool   `json:"consolation,omitempty"`
       LastChance     bool   `json:"last_chance,omitempty"`
       Reconciliation bool   `json:"reconciliation,omitempty"`
       Recharge       bool   `json:"recharge,omitempty"`
       GroupSize      int    `json:"group_size,omitempty"`
       Qualifiers     int    `json:"qualifiers,omitempty"`
       Entry          string `json:"entry,omitempty"`
       BatchMinutes   int    `json:"batch_minutes,omitempty"`
       Seeding        string `json:"seeding,omitempty"`
   }

Deux formes anciennes restent lisibles : ``"tables": 12`` (un nombre) devient
``Tables{Count: 12}``, et ``"prizes": [100, 60, 40]`` (une liste de montants) devient la dotation de
la clé ``"all"`` (``PrizeSectionAll``, le classement général). Une table réservée n'accueille que la
section et la phase pour lesquelles elle l'est ; ``Tables.AvailableFor(n, section, phase)`` le dit.
La dotation (pool = droit d'entrée × inscrits, retenue, arrondi à l'unité par le plus fort reste,
partage entre ex æquo) est décrite dans :ref:`classement`.

Signification des champs de phase
---------------------------------

+--------------------+--------------------------------+--------------------------------------------+
| Champ              | Formats concernés              | Signification                              |
+====================+================================+============================================+
| ``kind``           | tous                           | ``swiss_lives``, ``lives_bracket``,        |
|                    |                                | ``gsl``, ``bracket``, ``round_robin``      |
+--------------------+--------------------------------+--------------------------------------------+
| ``name``           | tous                           | Nom saisi par le TD ; vide = nom par       |
|                    |                                | défaut du format (``PhaseName``)           |
+--------------------+--------------------------------+--------------------------------------------+
| ``lives``          | ``swiss_lives``                | Nombre de vies (défaut 2). Forcé à 2 pour  |
|                    |                                | ``gsl``                                    |
+--------------------+--------------------------------+--------------------------------------------+
| ``length``         | tous                           | Longueur des matchs en points              |
|                    |                                | (obligatoire, > 0)                         |
+--------------------+--------------------------------+--------------------------------------------+
| ``final_length``   | tableaux                       | Longueur du dernier tour (0 = ``length``)  |
+--------------------+--------------------------------+--------------------------------------------+
| ``lengths``        | tableaux                       | Longueur tour par tour, **du dernier tour  |
|                    |                                | vers le premier** (``[15,13,11,9]`` =      |
|                    |                                | finale 15, demies 13, quarts 11, reste 9). |
|                    |                                | Plus précis que ``final_length`` et        |
|                    |                                | l'emporte sur lui ; une liste plus courte  |
|                    |                                | que le tableau retombe sur ``length``      |
+--------------------+--------------------------------+--------------------------------------------+
| ``length_late``    | ``swiss_lives``                | Longueur des matchs de fin de phase, quand |
|                    |                                | il reste au plus ``late_threshold``        |
|                    |                                | joueurs en vie. Sans effet sans seuil      |
|                    |                                | (refusé par ``Validate``) ; un             |
|                    |                                | ``length_changed`` du TD l'emporte         |
+--------------------+--------------------------------+--------------------------------------------+
| ``late_threshold`` | ``swiss_lives``                | Nombre de joueurs en vie à partir duquel   |
|                    |                                | ``length_late`` s'applique                 |
+--------------------+--------------------------------+--------------------------------------------+
| ``mode``           | ``swiss_lives``                | ``continuous`` (défaut) ou ``rounds``      |
+--------------------+--------------------------------+--------------------------------------------+
| ``batch_minutes``  | ``swiss_lives`` continu        | Micro-rondes : les joueurs libres          |
|                    |                                | attendent l'échéance du prochain lot, puis |
|                    |                                | tous ceux d'un même groupe de défaites     |
|                    |                                | sont appariés d'un coup. 0 = appariement   |
|                    |                                | au fil de l'eau. Refusé en mode ``rounds`` |
|                    |                                | — une ronde EST un lot                     |
+--------------------+--------------------------------+--------------------------------------------+
| ``pairing``        | ``swiss_lives``                | ``random`` (défaut) ou ``wins`` (apparier  |
|                    |                                | d'abord les joueurs les plus victorieux du |
|                    |                                | groupe)                                    |
+--------------------+--------------------------------+--------------------------------------------+
| ``avoid_clubs``    | ``swiss_lives``                | Éviter les rencontres entre joueurs d'un   |
|                    |                                | même club quand c'est possible             |
+--------------------+--------------------------------+--------------------------------------------+
| ``allow_rematch``  | ``swiss_lives``                | Autoriser une seconde rencontre entre deux |
|                    |                                | mêmes joueurs                              |
+--------------------+--------------------------------+--------------------------------------------+
| ``target``         | ``swiss_lives``, ``gsl``       | Figer la phase quand Σ vies ≤ ``target``   |
|                    |                                | (puissance de 2 ; suisse : 2 vies exigées) |
+--------------------+--------------------------------+--------------------------------------------+
| ``consolation``    | ``bracket``, ``lives_bracket`` | Consolante progressive alimentée par les   |
|                    |                                | perdants du principal                      |
+--------------------+--------------------------------+--------------------------------------------+
| ``last_chance``    | idem                           | Dernière chance alimentée par les perdants |
|                    |                                | de la consolante                           |
+--------------------+--------------------------------+--------------------------------------------+
| ``reconciliation`` | idem                           | Le vainqueur de la consolante affronte     |
|                    |                                | celui du principal (grande finale)         |
+--------------------+--------------------------------+--------------------------------------------+
| ``recharge``       | idem                           | Double élimination vraie : le vainqueur du |
|                    |                                | principal doit être battu deux fois        |
+--------------------+--------------------------------+--------------------------------------------+
| ``group_size``     | ``round_robin``                | Taille des poules (défaut 4)               |
+--------------------+--------------------------------+--------------------------------------------+
| ``qualifiers``     | ``round_robin``                | Qualifiés par poule (défaut 2)             |
+--------------------+--------------------------------+--------------------------------------------+
| ``entry``          | toutes sauf la première        | ``survivors`` (défaut : les survivants,    |
|                    |                                | avec leurs vies restantes si la phase est  |
|                    |                                | à vies), ``all``, ``top:N``                |
+--------------------+--------------------------------+--------------------------------------------+
| ``seeding``        | ``bracket``, ``lives_bracket`` | ``""`` (défaut) : tirage intégralement     |
|                    |                                | aléatoire. ``"rating"`` : placement        |
|                    |                                | classique par cote d'entrée (1 contre 16,  |
|                    |                                | 2 contre 15… ; ``rules.SeedOrder``), les   |
|                    |                                | joueurs à 2 vies d'abord. **Le défaut vide |
|                    |                                | est un choix de conception**, pas un       |
|                    |                                | oubli : l'étude conclut « pas de têtes de  |
|                    |                                | série protégées », c'est la culture du     |
|                    |                                | backgammon. Refusé sur un format sans      |
|                    |                                | tirage de tableau                          |
+--------------------+--------------------------------+--------------------------------------------+

Validation et valeurs par défaut
--------------------------------

``Config.Validate() error`` **modifie la configuration en place** pour y écrire les défauts, et
renvoie une erreur au premier problème. Elle est appelée par ``New``, par ``ParseConfig``, par
``CheckConfig`` et par ``Apply`` sur ``created`` et ``config_changed`` : une configuration invalide
ne peut donc pas entrer dans un journal.

Algorithme :

::

   si len(Phases) == 0            → erreur « config : au moins une phase »
   si MinPerPoint <= 0            → MinPerPoint = 8
   Prizes.validate() :
       EntryFee < 0, Retention.Percent ∉ [0, 100], Retention.Amount < 0 → erreur
       une section avec Percents ET Amounts                             → erreur
       un pourcentage ou un montant négatif                             → erreur
       somme de tous les pourcentages (toutes sections) > 100          → erreur
   pour chaque pause b : si b.End n'est pas après b.Start → erreur

   pour chaque phase p d'indice i :
       si p.Length <= 0           → erreur « phase i : longueur de match manquante »
       si un élément de p.Lengths <= 0              → erreur
       si p.LengthLate > 0 et p.LateThreshold <= 0  → erreur « length_late sans late_threshold »

       selon p.Kind :
         swiss_lives :
             si p.Lives <= 0                       → p.Lives = 2
             si p.Mode == ""                       → p.Mode = "continuous"
             si p.Mode ∉ {continuous, rounds}      → erreur « mode inconnu »
             si p.Pairing == ""                    → p.Pairing = "random"
             si p.Target != 0 et Target n'est pas une puissance de 2 → erreur
             si p.Target != 0 et p.Lives != 2      → erreur
                   « la bascule vers un tableau à vies suppose 2 vies »
             si p.BatchMinutes < 0                 → erreur
             si p.BatchMinutes > 0 et p.Mode == "rounds" → erreur
                   « une ronde EST un lot »
         lives_bracket, bracket :
             si p.Seeding ∉ {"", "rating"}         → erreur « seeding inconnu »
             si p.Recharge et non p.Reconciliation → erreur « recharge sans reconciliation »
             si p.Reconciliation et non p.Consolation → erreur « reconciliation sans consolante »
         gsl :
             p.Lives = 2   (forcé)
             si p.Target != 0 et Target n'est pas une puissance de 2 → erreur
         round_robin :
             si p.GroupSize <= 0   → p.GroupSize = 4
             si p.Qualifiers <= 0  → p.Qualifiers = 2
             si p.Qualifiers >= p.GroupSize → erreur « qualifiers doit être < group_size »
         défaut : erreur « type inconnu »

       si p.Entry == ""  → p.Entry = "survivors"
       si p.BatchMinutes != 0 et p.Kind != swiss_lives → erreur
       si p.Seeding != "" et p.Kind ∉ {bracket, lives_bracket} → erreur

``Entry`` n'est pas vérifiée : une valeur inconnue fait entrer les survivants, mais avec les vies
par défaut du format (pas leurs vies restantes), et un ``top:N`` illisible donne une phase vide
(voir :ref:`etat`).

Le test de puissance de deux est ``Target & (Target-1) == 0`` (vrai aussi pour 0, d'où la garde
``Target != 0``).

``Validate`` ne remplit **pas** ``Name`` : un hôte multilingue fabrique son propre nom. Le nom par
défaut français est donné par ``PhaseName(p PhaseConfig) string`` (``fr.go``), qui renvoie
``p.Name`` s'il est saisi, sinon :

=================================== =======================
Kind                                Nom
=================================== =======================
``swiss_lives``                     ``Suisse <Lives> vies``
``lives_bracket``                   ``Tableau final``
``gsl``                             ``Blocs GSL``
``bracket`` avec ``reconciliation`` ``Double élimination``
``bracket`` sinon                   ``Tableau``
``round_robin``                     ``Poules``
(autre)                             ``p.Kind``
=================================== =======================

**Note d'implémentation importante** : ``Validate`` étant idempotente et appliquée au rejeu, la
configuration écrite dans l'événement ``created`` peut être partielle ; le rejeu produira la même
configuration complétée. Une implémentation qui changerait les défauts changerait donc le sens des
journaux anciens : les défauts font partie du contrat.

Modification en cours de tournoi
--------------------------------

L'événement ``config_changed`` porte la configuration **entière**, validée puis comparée à l'état
(``acceptConfig``, ``reconfig.go``). ``State.CheckConfig(next Config) error`` fait le même contrôle
sans rien appliquer (prévisualisation d'un formulaire de réglages). Un refus est un
``*ConfigRefusal`` :

.. code:: go

   type ConfigRefusal struct {
       Phase  int    `json:"phase"`
       Field  string `json:"field"`            // champ JSON de PhaseConfig ; "phases" pour un retrait
       Reason string `json:"reason"`           // removed | finished | drawn | started
       From   string `json:"from,omitempty"`   // kind : l'ancien format
       To     string `json:"to,omitempty"`     // kind : le nouveau format
       Opened int    `json:"opened,omitempty"` // removed : nombre de phases déjà ouvertes
   }

Règles :

- moins de phases que de phases déjà ouvertes → ``removed`` ;
- ``kind`` changé pour une phase ouverte : ``finished`` si elle est passée, ``drawn`` si elle est
  tirée, ``started`` si des matchs y sont lancés ; sinon accepté (les entrants reçoivent les vies du
  nouveau format, et la longueur courante celle de la nouvelle configuration) ;
- tableau (``bracket``, ``lives_bracket``) déjà tiré : ``consolation``, ``last_chance``,
  ``reconciliation``, ``recharge`` et ``seeding`` sont figés (``drawn``) — à partir de la version 2
  du journal seulement ; un ``config_changed`` de version antérieure est accepté, sans effet ;
- tout le reste est accepté (bascule, longueurs, dotation, tables, pauses). La longueur courante
  d'une phase ouverte ne suit la configuration que si ``length`` a effectivement changé, pour ne pas
  effacer un ``length_changed`` saisi par le TD.

Voir :ref:`journal`.

Configurations types
--------------------

Chacun des deux binaires (``cmd/tournoi-demo``, ``cmd/tournoi-td``) a sa propre table ``formats`` de
configurations nommées, à garder cohérentes ; ``suisse_rondes`` n'existe que dans ``tournoi-td`` :

================== =============================================================================
Nom                Phases
================== =============================================================================
``suisse``         ``swiss_lives`` 7 points
``suisse_rondes``  ``swiss_lives`` 7 points, ``mode: rounds`` (``tournoi-td`` seulement)
``suisse_tableau`` ``swiss_lives`` 7 pts ``target: 16``, puis ``lives_bracket`` 9 pts, finale 11
``gsl``            ``gsl`` 7 pts ``target: 16``, puis ``lives_bracket`` 9 pts
``elim``           ``bracket`` 9 pts, finale 11
``conso``          ``bracket`` 9 pts, ``consolation`` + ``last_chance``
``double``         ``bracket`` 7 pts, ``consolation`` + ``reconciliation`` + ``recharge``
``poules``         ``round_robin`` 5 pts (poules de 4, 2 qualifiés), puis ``bracket`` 9 pts
================== =============================================================================

**Format recommandé par l'étude** : ``suisse_tableau`` — suisse 2 vies en continu, appariement
aléatoire dans le groupe de même nombre de défaites, sans rematch, bascule à Σ vies = 16 ou 32, puis
tableau à exemptions avec des matchs plus longs.
