.. _api_reference:

Récapitulatif de l'API
======================

Le module s'importe sous le nom ``tournoi`` :

.. code:: go

   import tournoi "github.com/kevung/nicomaque"

La liste ci-dessous est celle de ``go doc -all`` sur le paquet racine ; les paquets annexes suivent
en fin de page.

Types
-----

Joueurs et matchs : ``PlayerID``, ``Player``, ``MatchID``, ``MatchStatus``, ``Match``, ``Rank``,
``Absence``.

Actions et journal : ``ActionKind``, ``Action``, ``Draw``, ``EventKind``, ``Event``, ``Journal``,
``External``.

Configuration : ``Config``, ``PhaseConfig``, ``Tables``, ``TableRule``, ``TimeRange``,
``PrizePool``, ``PrizeScale``, ``Retention``, ``ConfigRefusal``.

État : ``State``, ``PhaseState``, ``Section``, ``GMatch``, ``Src``, ``Slot``, ``Clock``,
``PlayerStatus``, ``StatusKind``.

Codes (``codes.go`` ; aucun texte destiné à l'affichage ne sort du moteur, voir :ref:`affichage`) :
``LabelKind``, ``Label``, ``NoteKind``, ``Note``, ``WarningCode``, ``Warning``, ``InfoCode``,
``Info``, ``ReasonCode``. Les noms de section (``Section.Name``, ``Match.Section``) sont des
identifiants — ``main``, ``conso``, ``last``, ``gf``, ``poule:A``, ``barrage:A``… — et non des
libellés.

Constantes
----------

- ``BYE`` (``PlayerID`` réservé à une place vide) ;
- ``MatchStatus`` : ``Running``, ``Finished``, ``Cancelled`` ;
- ``ActionKind`` : ``ActStartMatch``, ``ActBye``, ``ActDraw``, ``ActNextPhase``, ``ActFinish``,
  ``ActWait``, ``ActCancelMatch`` (annulation proposée d'un match devenu incohérent),
  ``ActRepechage`` (repêchage proposé d'un suivant de poule) ;
- ``EventKind`` : ``EvCreated``, ``EvPlayerAdded``, ``EvPlayerWithdrawn``, ``EvPlayerUpdated``,
  ``EvPlayerUnavailable``, ``EvPlayerAvailable``, ``EvMatchStarted``, ``EvResult``,
  ``EvResultCorrected``, ``EvMatchCancelled``, ``EvBye``, ``EvDraw``, ``EvNextPhase``,
  ``EvRepechage``, ``EvLengthChanged``, ``EvConfigChanged``, ``EvReopened``, ``EvTableChanged``,
  ``EvFinished``, ``EvNote`` ;
- formats (chaînes non typées, ``PhaseConfig.Kind``) : ``KindSwissLives``, ``KindLivesBracket``,
  ``KindGSL``, ``KindBracket``, ``KindRoundRobin`` ;
- raisons d'un ``ConfigRefusal`` : ``RefusalRemoved``, ``RefusalFinished``, ``RefusalDrawn``,
  ``RefusalStarted`` ;
- ``JournalVersion`` (= 2, posée dans chaque événement par les constructeurs ; un journal
  ``Version: 0`` est converti à la lecture) ;
- ``PrizeSectionAll`` (= ``"all"``, clé de la dotation du classement général) ;
- ``SeedingRating`` (``PhaseConfig.Seeding`` : placement par cote, même valeur que
  ``rules.SeedingRating``) ;
- les codes ``Label*`` (``LabelNone``, ``LabelRound``, ``LabelFinal``, ``LabelInSection``,
  ``LabelBarrage``, ``LabelPoolName``, ``LabelSectionMain``…), ``Note*`` (dont ``NoteWithdrawn`` :
  retiré, classé sur son parcours ; ``NoteForfeit`` : match perdu par forfait), ``Warn*``,
  ``Info*`` (``InfoEntersAt``, ``InfoNoEntry``), ``Reason*`` et ``Status*``.

Fonctions et méthodes
---------------------

.. code:: go

   // cycle de vie
   func New(cfg Config, seed int64, now time.Time) (*State, Event, error)
   func Replay(j Journal) (*State, error)
   func (s *State) Apply(ev Event) error
   func (s *State) Step(evs ...Event) ([]Action, error) // Apply de chaque événement, puis Propose

   // boucle du TD
   func (s *State) Propose() []Action              // = ProposeAt(s.Last)
   func (s *State) ProposeAt(now time.Time) []Action // = ProposeWith(now, External{})
   func (s *State) ProposeWith(now time.Time, ext External) []Action // tables et joueurs pris par une autre épreuve
   func (s *State) EventFromAction(a Action, now time.Time) (Event, error)
   func (s *State) CheckConfig(next Config) error  // prévisualisation ; refus = *ConfigRefusal

   // constructeurs d'événements (posent Kind, Time et Version)
   func PlayerAddedEvent(p Player, now time.Time) Event
   func PlayerAddedAtSlotEvent(p Player, slot Slot, now time.Time) Event // retardataire sur une place d'exemption
   func PlayerUpdatedEvent(p Player, now time.Time) Event
   func PlayerWithdrawnEvent(id PlayerID, now time.Time) Event
   func PlayerWithdrawnAfterCurrentEvent(id PlayerID, now time.Time) Event // finit son match en cours
   func PlayerUnavailableEvent(id PlayerID, until time.Time, now time.Time) Event
   func PlayerUnavailableUntilRoundEvent(id PlayerID, round int, now time.Time) Event
   func PlayerAvailableEvent(id PlayerID, now time.Time) Event
   func ResultEvent(id MatchID, winner PlayerID, scoreA, scoreB int, now time.Time) Event
   func ForfeitEvent(id MatchID, winner PlayerID, now time.Time) Event
   func CorrectionEvent(id MatchID, winner PlayerID, scoreA, scoreB int, now time.Time) Event
   func CancelEvent(id MatchID, now time.Time) Event
   func TableChangedEvent(id MatchID, table int, now time.Time) Event
   func LengthChangedEvent(phase, length int, now time.Time) Event
   func ConfigChangedEvent(cfg Config, now time.Time) Event // la configuration ENTIÈRE
   func RepechageEvent(phase int, section string, withdrawn, replacement PlayerID, now time.Time) Event
   func ReopenedEvent(now time.Time) Event
   func NoteEvent(text string, now time.Time) Event
   func (e Event) WithNote(text string) Event // remarque attachée à un résultat

   // consultation
   func (s *State) Running() []*Match
   func (s *State) Ranking() []Rank
   func (s *State) SectionRanking(name string) []Rank
   func (s *State) FreeSlots() []Slot          // places d'exemption libres (retardataires)
   func (s *State) Statuses() []PlayerStatus   // « est-ce que je joue ? », un statut par inscrit
   func (s *State) StatusOf(p PlayerID) PlayerStatus
   func (s *State) StandingsCSV() []byte       // une section dotée par bloc
   func (s *State) Pool() float64
   func (s *State) Distributable() float64
   func (s *State) PrizeAmounts(section string) []float64
   func (s *State) SectionPrizes(section string) map[PlayerID]float64
   func (s *State) Expected(n int) time.Duration
   func (s *State) SlowMatches(now time.Time, facteur float64) []*Match
   func (s *State) ClockAt(now time.Time, start time.Time) Clock
   func Prizes(ranking []Rank, prizes []float64) map[PlayerID]float64

   // passerelles vers les règles pures (rules)
   func (s *State) RulesMatches() []rules.Match   // entrée de rules.Batches, PairingSheet, PaceAt…
   func (s *State) RulesPhase(i int) *rules.Phase // entrée de rules.Pair, Rank, RemainingMatches ; nil si absente
   func (c Config) BreakIntervals() []rules.Interval

   // configuration
   func ParseConfig(b []byte) (*Config, error)
   func (c *Config) Validate() error
   func (t Tables) AvailableFor(n int, section string, phase int) bool
   func (p PrizePool) Empty() bool
   func (p PrizePool) SectionNames() []string
   func (r *ConfigRefusal) Error() string

   // sérialisation
   func (j Journal) Bytes() ([]byte, error)
   func ParseJournal(b []byte) (Journal, error)
   func (l *Label) UnmarshalJSON(b []byte) error     // accepte aussi le libellé-chaîne des journaux v0
   func (p *PrizePool) UnmarshalJSON(b []byte) error // accepte aussi une simple liste de montants
   func (t *Tables) UnmarshalJSON(b []byte) error    // accepte aussi un simple nombre de tables

   // rendu français des codes (fr.go) — la console, la démo et render ; un hôte
   // multilingue traduit les codes lui-même (render.Labeler) et ignore ces fonctions
   func (l Label) String() string
   func (n Note) String() string
   func (w Warning) String() string
   func (i Info) String() string
   func (p PlayerStatus) String() string
   func (r ReasonCode) String() string
   func PhaseName(p PhaseConfig) string
   func SectionName(name string) string

   // divers
   func (l Label) Empty() bool
   func (m *Match) Loser() PlayerID
   func (m *Match) Has(p PlayerID) bool
   func (a Action) String() string

Les règles elles-mêmes (appariements, tirages, résolution des graphes, classement, dotation,
prévision) ne sont pas dans le paquet racine : elles vivent, exportées, dans ``rules``, et les
fonctions du paquet racine en sont des enveloppes. Le reste du paquet racine est non exporté. Un
journal ne dépend pas de ces règles : les tirages sont matérialisés dans ``EvDraw``, si bien qu'un
changement d'algorithme ne casse pas les journaux existants (voir :ref:`journal`).

Paquets annexes
---------------

``rules`` (``github.com/kevung/nicomaque/rules``)
   Règles en fonctions pures, sans état, sans E/S ni horloge, sur ses propres types (``Ref``,
   ``Phase``, ``PhaseRules``, ``Seed``, ``Pairing``, ``Section``, ``Node``, ``Src``, ``Slot``,
   ``Match``, ``Label``, ``Note``, ``Standing``, ``Placement``, ``PrizeScale``, ``Retention``,
   ``Interval``, ``Pace``, ``Remaining``, ``Sheet``, ``LengthPlan``…). Appariements : ``Pair``,
   ``PairSwissRound``, ``PairSwissOrphans``, ``PairSwissFallback`` ; graphes : ``BracketSection(s)``,
   ``ConsolationSection``, ``LastChanceSection``, ``GSLSection``, ``GSLBlockSections``,
   ``SESection``, ``PoolSection(s)``, ``BarrageSection``, ``Resolve``, ``Ready``, ``ReadyFree``,
   ``AllDone``, ``PhaseDone`` ; tirages : ``DrawSlots``, ``SeedOrder``, ``Partition`` ; classement :
   ``Rank``, ``RankSection``, ``MergeRankings``, ``PoolQualified``, ``PoolTie``, ``PoolWins`` ;
   dotation : ``Pool``, ``Distributable``, ``PlaceAmounts``, ``SharePrizes`` ; prévision et
   horloge : ``RemainingMatches``, ``PaceAt``, ``ExpectedDuration``, ``IsSlow``, ``PlayingTime``,
   ``DayOfPlay``, ``EstimatedEnd``, ``AfterBreaks``, ``WinProbability``, ``MeanRating``,
   ``RemainingDuration`` ; feuille d'appariements : ``Batches``, ``PairingSheet``.

``render`` (``github.com/kevung/nicomaque/render``)
   ``Labeler``, ``French()``, ``Term``, ``Renderer`` (``New``, ``BracketBoardSVG``,
   ``LivesBoard``, ``TableGrid``, ``RunningTable``, ``ActionsList``, ``StandingsTable``,
   ``PairingSheet``, ``Page``), ``Batches``, ``DefaultStyle``, ``DefaultCredit``. Voir
   :ref:`affichage`.

``sim`` (``github.com/kevung/nicomaque/sim``)
   ``PGain``, ``Duree``, ``Options``, ``Result``, ``Run``, ``Champ``, ``Forecast``,
   ``Durations``, ``EstimatedEnd``. Voir :ref:`simulation`.

``players`` (``github.com/kevung/nicomaque/players``)
   ``FromCSV``, ``ToCSV``. Voir :ref:`joueurs_csv`.
