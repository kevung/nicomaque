.. _simulation:

Simulation et prévision (``sim``)
=================================

Le paquet ``sim`` joue des tournois fictifs avec le moteur. Il sert à trois choses : les tests
d'invariants, la validation des formats (comparaison avec le simulateur de l'étude) et la prévision
de l'heure de fin d'un tournoi en cours.

Modèle probabiliste
-------------------

.. code:: go

   func PGain(a, b float64, n int) float64

Probabilité que le joueur de PR ``a`` batte celui de PR ``b`` en ``n`` points, formule Elo/FIBS avec
la conversion **3 PR = 100 points Elo** (enveloppe de ``rules.WinProbability``). Le PR est une cote
d'erreurs : plus bas = meilleur.

.. math::  D = \frac{(b-a)\times 100}{3} \qquad P = \frac{1}{1 + 10^{-D\sqrt{n}/2000}} 

Justification (voir :ref:`annexe_etude`) : sur la base BMAB (33 238 matchs uniques), 1 PR
d'écart vaut 2,8 % à 7–11 points (β = 0,036 pour α = 0,5 ; théorie 0,039). L'exposant α de la
longueur n'est pas identifiable avec ces données — la vraisemblance est plate entre 0 et 0,7, α = 1
est exclu — mais la mécanique « erreur cumulée ∝ N, chance ∝ √N » soutient α = 0,5.

.. code:: go

   func Duree(n int, minPerPoint float64, rng *rand.Rand) time.Duration

Durée d'un match de ``n`` points : loi **gamma de forme 8** et de moyenne ``minPerPoint × n``,
obtenue en moyennant huit tirages exponentiels. La forme 8 donne un coefficient de variation
d'environ 35 %, proche de l'observation.

Exécution d'un tournoi
----------------------

.. code:: go

   type Options struct {
       Seed        int64
       MinPerPoint float64          // défaut 8
       Start       time.Time        // défaut 2026-01-01 09:00 UTC
       MaxSteps    int              // défaut 100000
       Hook        func(s *tournoi.State, ev tournoi.Event) error
   }

   type Result struct {
       State    *tournoi.State
       Journal  tournoi.Journal
       Winner   tournoi.PlayerID
       Minutes  float64
       NMatches int
       Steps    int
       Err      error
   }

   func Run(cfg tournoi.Config, players []tournoi.Player, opt Options) Result

Boucle de simulation :

::

   New(cfg, Seed, Start), puis PlayerAddedEvent pour chaque joueur
   tant que non terminé (au plus MaxSteps tours, sinon erreur « simulation sans fin ») :
       acts := ProposeAt(now)
       pour chaque action non-Wait :
           événement := EventFromAction ; Apply ; ajouter au journal ; appeler Hook
           si start_match → enregistrer sa fin prévue (now + Duree(...))
           si draw, repechage, next_phase ou finish → SORTIR de la boucle (l'état a changé)
       si des actions ont été appliquées et la première n'était pas Wait → recommencer
       échéance := le Until le plus récent des Wait encore à venir (micro-rondes)
       si aucun match en cours :
           échéance connue → now := échéance, recommencer
           rien n'a progressé → erreur « blocage »
       si l'échéance précède la fin du premier match → now := échéance, recommencer
       sinon : avancer l'horloge jusqu'à la fin du match le plus proche,
               tirer son vainqueur selon PGain, appliquer ResultEvent

Le score du vainqueur est ``n`` ; celui du perdant est tiré uniformément dans ``[0, n-1]``. Le
résultat porte ``Winner`` (le vainqueur, ``Final[0]``), ``Minutes`` (durée totale depuis
``Start``), ``NMatches`` (matchs lancés) et ``Steps`` ; en cas d'erreur, ``Err`` est rempli et
``State``/``Journal`` sont ceux du moment de l'erreur.

``Hook`` est appelé après chaque événement appliqué : c'est le point d'accroche des tests
d'invariants. Une erreur renvoyée par le hook interrompt la simulation.

.. code:: go

   func Champ(P int, mu, sigma, min, max float64, rng *rand.Rand) []tournoi.Player

Crée ``P`` joueurs de PR tirés dans une loi normale ``N(mu, sigma)`` tronquée à ``[min, max]``,
**triés du meilleur au moins bon** : ``P1`` est toujours le meilleur joueur du champ. Les clubs sont
affectés cycliquement (``Club 0`` à ``Club 4``) pour exercer ``avoid_clubs`` ; les noms sont
``Joueur 1``, ``Joueur 2``… Le champ de référence
de l'étude est ``Champ(64, 6, 2, 2, 10)``.

Prévision de fin
----------------

.. code:: go

   func Forecast(j tournoi.Journal, now time.Time, K int, minPerPoint float64,
                 seed int64) ([]float64, error)

Rejoue le journal, puis simule ``K`` fins de tournoi à partir de ``now`` (la ``k``-ième avec la
graine ``seed + k``). Renvoie les durées restantes en minutes, **triées croissantes** : la médiane
est ``out[K/2]``, le quantile 90 % ``out[9K/10]``.

Particularités :

- ``minPerPoint ≤ 0`` prend ``Config.MinPerPoint`` du tournoi ;
- les PR inconnus (0) sont remplacés par la moyenne des PR connus, ou 6 s'il n'y en a aucun
  (``rules.MeanRating``) ;
- pour un match déjà en cours, la durée restante est un tirage neuf moins le temps déjà écoulé,
  **plancher à 10 % de la durée attendue** (``rules.RemainingDuration``) ;
- le perdant d'un match simulé marque 0 ; chaque fin simulée est bornée à 20 000 tours de boucle ;
- un tournoi déjà terminé renvoie ``[0]``.

Les règles de prévision elles-mêmes vivent dans ``rules`` (``WinProbability``, ``MeanRating``,
``RemainingDuration``, ``EstimatedEnd``, ``AfterBreaks``) ; ``Forecast`` en est l'enveloppe à
journal (voir :ref:`horloge`).

.. code:: go

   func Durations(minutes []float64) []time.Duration
   func EstimatedEnd(j tournoi.Journal, now time.Time, K int, minPerPoint float64,
                     seed int64) (end time.Time, ok bool, err error)

``Durations`` convertit les minutes de ``Forecast`` en ``time.Duration``. ``EstimatedEnd`` donne
l'heure de fin estimée : la médiane des ``K`` fins simulées, reportée au-delà des pauses déclarées
(``Config.Breaks``, via ``rules.EstimatedEnd``). ``ok`` est faux pour un tournoi terminé ou sans
fin simulée.

La prévision est figée par un fichier témoin, ``sim/testdata/prevision.json`` ; ``go test ./sim
-update`` le réécrit après un changement voulu.
