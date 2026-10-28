.. _classement:

Classement et prix
==================

Concaténation des phases
------------------------

.. code:: go

   func (s *State) Ranking() []Rank
   func MergeRankings(byPhase [][]Standing) []Standing   // paquet rules

Le classement général se lit **de la dernière phase atteinte à la première** : les joueurs éliminés
plus tôt sont classés derrière. ``Ranking`` calcule ``rules.Rank`` sur l'instantané de chaque phase
ouverte, de la première à la dernière, et les passe à ``rules.MergeRankings`` :

::

   out := [] ; vus := {} ; offset := 0
   pour i de la dernière phase à la première :
       r := byPhase[i]
       pour chaque rang rk de r :
           si rk.Player déjà vu → passer    ← déjà classé par une phase plus tardive
           marquer vu
           out += { Player, Rank: rk.Rank + offset, Note: rk.Note }
       offset += len(r)                             ← len(r), pas le nombre de nouveaux !

   tri stable par Rank croissant
   renumérotation dense en conservant les ex æquo :
       pos := 0 ; prev := -1 ; prevRank := 0
       pour chaque élément :
           si Rank != prev → prev := Rank ; prevRank := pos+1
           Rank := prevRank ; pos++

L'``offset`` s'incrémente de la **taille complète** du classement de phase (y compris les joueurs
déjà vus) : cela garantit qu'aucun joueur d'une phase antérieure ne peut se glisser devant un joueur
d'une phase postérieure, quelle que soit la répartition des ex æquo.

La renumérotation finale produit un classement dense : 1, 2, 2, 4, 5, 5, 5, 8… (rang = position du
premier élément du groupe d'ex æquo).

.. code:: go

   func (s *State) phaseRanking(ph *PhaseState) []Rank   // enveloppe de rules.Rank
   func Rank(p Phase) []Standing                          // paquet rules

``rules.Rank`` dispatche selon le format de la phase :

============================== =================================================================
Kind                           Fonction (paquet ``rules``, ``classement.go``)
============================== =================================================================
``bracket``, ``lives_bracket`` ``rankBracket`` (voir :ref:`format_tableaux`)
``round_robin``                ``rankPools`` (voir :ref:`format_poules`)
``swiss_lives``, ``gsl``       ``rankLives`` (voir :ref:`format_suisse`)
============================== =================================================================

Les notes (``Note``) sont des codes structurés, jamais des phrases : ``winner``, ``finalist``,
``alive``, ``record``, ``forfeit``, ``withdrawn``, ``running``, ``awaiting_draw``, ``unranked``,
``section_exit``, ``section_winner``, ``pool_record``, avec leurs paramètres (``wins``,
``losses``, ``lives``, ``section``, ``qualified``, ``sub``). Leur rendu français (``Note.String``,
``fr.go``) ne sert que la console, la démo, ``render/`` et l'export CSV.

Prix
----

.. code:: go

   func Prizes(ranking []Rank, prizes []float64) map[PlayerID]float64
   func SharePrizes(ranking []Placement, amounts []float64) map[Ref]float64   // paquet rules

Les prix sont indexés par place : ``prizes[0]`` est la dotation du premier, ``prizes[1]`` du
deuxième, etc. Les ex æquo **partagent à parts égales les prix des places qu'ils occupent** :

::

   grouper le classement par Rank
   pour chaque groupe de rang r comptant k joueurs :
       total := somme de prizes[r-1] … prizes[r-2+k]   (les indices hors tableau valent 0)
       chaque joueur du groupe reçoit total / k
   un rang nul ou négatif n'est pas une place du barème : 0

Exemple : deux joueurs ex æquo au rang 3, dotations ``[100, 60, 40, 20, 10]`` → ils se partagent 40
+ 20 = 60, soit 30 chacun ; le joueur suivant est au rang 5 et reçoit 10.

L'unité est libre (montant, pourcentage, part) : le moteur ne fait qu'additionner et diviser. Cette
fonction est le partage entre ex æquo, et rien d'autre ; d'où viennent les montants par place est
décrit dans la section suivante.

Pool, retenue, barème par section
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code:: go

   type PrizeScale struct {
       Percents []float64 `json:"percents,omitempty"` // % du pool distribuable, place par place
       Amounts  []float64 `json:"amounts,omitempty"`  // montants fixes, place par place
   }

   type Retention struct {
       Amount  float64 `json:"amount,omitempty"`
       Percent float64 `json:"percent,omitempty"`
   }

   type PrizePool struct {
       EntryFee  float64               `json:"entry_fee,omitempty"`
       Retention Retention             `json:"retention,omitempty"`
       Sections  map[string]PrizeScale `json:"sections,omitempty"`
   }

   const PrizeSectionAll = "all"   // clé du classement GÉNÉRAL dans Sections

   func (s *State) Pool() float64
   func (s *State) Distributable() float64
   func (s *State) PrizeAmounts(section string) []float64
   func (s *State) SectionPrizes(section string) map[PlayerID]float64
   func (s *State) SectionRanking(section string) []Rank

   // paquet rules (dotation.go)
   func Pool(entryFee float64, entrants int) float64
   func Distributable(pool float64, r Retention) float64
   func PlaceAmounts(scale PrizeScale, distributable float64) []float64

Une affiche de tournoi annonce « 50/30/20 % » et non « 216/130/86 € » : le pool dépend du nombre
d'inscrits, qu'on ne connaît qu'à la clôture des inscriptions.

::

   Pool()          = EntryFee × nombre d'inscrits          (les retirés comptent : ils ont payé)
   Distributable() = Pool − Pool × Retention.Percent/100 − Retention.Amount   (jamais négatif)

``PrizeAmounts(sec)`` (enveloppe de ``rules.PlaceAmounts``) : une section absente de ``Sections``
n'a pas de dotation (``nil``). Si le barème donne des ``Amounts``, ce sont eux, tels quels, ni
arrondis ni ajustés (ils l'emportent sur ``Percents``, que ``Validate`` refuse de toute façon avec
eux). S'il donne des ``Percents``, chaque place vaut ``Distributable × pct/100`` (ramené au
millionième, pour que le bruit binaire ne fasse pas perdre une unité), payée **à l'unité** —
personne ne paie en centimes à une table de tournoi — par la **méthode du plus fort reste**
(Hamilton) : chaque montant est tronqué à l'unité, puis les unités qui manquent pour atteindre la
somme des parts exactes arrondie à l'unité vont, une chacune, aux places de plus grande partie
décimale ; à égalité, à la mieux classée. La somme distribuée égale exactement cette somme arrondie
(le pool distribuable arrondi quand le barème totalise 100 %), aucun prix n'est négatif, chaque
prix est à moins d'une unité de sa part exacte, et un barème décroissant donne des prix
décroissants. Exemple : 4 × 25 % d'un pool de 10 font 2,5 chacun ; les deux unités restantes
vont aux deux premiers, d'où 3/3/2/2.

``SectionPrizes(sec)`` applique ``Prizes`` aux montants de ``PrizeAmounts(sec)`` sur le classement
de la section : ``SectionRanking(sec)`` pour une section, et pour ``all`` le classement figé à la
clôture (``State.Final``) s'il existe, sinon ``Ranking()``.

``Sections`` est indexée par identifiant de section (``main``, ``conso``, ``last``, ``gf``,
``poule:A``\ …), plus la clé réservée ``all`` pour le classement général — celui que ``Ranking()``
renvoie, et le seul qui existe dans un tournoi sans tableau.

``SectionRanking(sec)`` est le classement PROPRE d'une section : ses joueurs, par tour atteint dans
cette section, le vainqueur en tête. Il diffère du classement général, qui mélange les sections par
priorité (un vainqueur de consolante passe derrière un demi-finaliste du principal). Une section de
poule se classe par nombre de victoires, faute de tour atteint (ici, à la différence de
``PoolWins``, une victoire par forfait compte) ; un barrage, qui n'a pas de graphe, n'a pas de
classement de section (liste vide). Dans les autres sections, le perdant d'un match prend la
profondeur de ce tour (note ``section_exit``, ou ``forfeit`` s'il l'a perdu par forfait), le
vainqueur du dernier match joué la profondeur maximale (note ``section_winner``), et un joueur sans
sortie aussi (note ``running``). Un retiré dont le prochain match attend encore son adversaire est
classé au tour de ce match (note ``withdrawn``), et non ``running``. Entre ex æquo, l'ordre est
celui de première apparition dans le graphe. C'est ``rules.RankSection`` ; l'enveloppe cherche la
section de la dernière phase vers la première, la plus récente qui porte ce nom comptant.

``Config.Validate`` refuse une dotation incohérente : pourcentages ET montants dans le même barème,
pourcentages (toutes sections confondues) totalisant plus de 100 % du pool, retenue hors
[0, 100] %, droit d'entrée, retenue, pourcentage ou montant négatif. Une
dotation fausse ne se voit qu'au moment de payer, devant les joueurs.

La forme liste (``"prizes": [100, 60, 40]``, une simple liste de montants) est également lue
(``PrizePool.UnmarshalJSON``) et devient le barème en montants du classement général
(``Sections["all"].Amounts``) : les journaux qui l'emploient se relisent sans être réécrits.

Export CSV
----------

.. code:: go

   func (s *State) StandingsCSV() []byte

Séparateur ``;``, en-tête ``section;rang;id;nom;club;note;prix``. La première colonne dit à quel
classement la ligne appartient : ``all`` pour le classement général (``State.Final``, figé à la
clôture, s'il existe, sinon ``Ranking()``), toujours écrit, même sans dotation ; puis l'identifiant
de chaque **section dotée**, dans l'ordre alphabétique, une section dont ``SectionRanking`` est vide
étant omise. Les
blocs sont contigus : un tableur les lit comme des tableaux séparés, un hôte qui n'en veut qu'un
filtre sur la colonne. Les sections apparaissent parce qu'elles ont une dotation — sortir le
classement de chaque groupe GSL d'un tournoi de cent joueurs noierait la feuille affichée au mur. Le
prix est formaté avec deux décimales (``0.00`` sans dotation). La colonne ``note`` est le rendu
français de la note (``Note.String``). Un joueur absent de ``Players`` est écrit avec son identifiant
en guise de nom et un club vide.
