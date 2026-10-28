.. _horloge:

Horloge et prévisions
=====================

.. code:: go

   func (s *State) Expected(n int) time.Duration

``Config.MinPerPoint × n`` minutes (``rules.ExpectedDuration``). Avec le défaut de 8 min/point, un match en 7 points est attendu
en 56 minutes. L'étude recommande de compter **11 min/point pour planifier une ronde** (attente
comprise) et 8 min/point pour l'espérance d'un match.

.. code:: go

   func (s *State) SlowMatches(now time.Time, facteur float64) []*Match

Les matchs en cours (``Running()``, dans l'ordre de lancement) dont la durée écoulée
``now - Start`` dépasse strictement ``facteur × Expected(Length)`` (``rules.IsSlow``). Le tableau
des matchs en cours de ``render`` applique le même critère avec un facteur de 1,5 et marque la
ligne (classe CSS ``lent``).

.. code:: go

   type Clock struct {
       Played   int           `json:"played"`
       Running  int           `json:"running"`
       Elapsed  time.Duration `json:"elapsed"`
       AvgMatch time.Duration `json:"avg_match"`
       AvgPerPt time.Duration `json:"avg_per_pt"`
   }

   func (s *State) ClockAt(now, start time.Time) Clock

Enveloppe de ``rules.PaceAt(s.RulesMatches(), now, start)`` (``rules.Pace`` a les mêmes champs) :

::

   Elapsed := now - start
   pour chaque match :
     Finished, non Forfeit, End renseignée → Played++ ; total += End-Start ; pts += Length
     Running → Running++
     (Cancelled : ignoré)
   AvgMatch := total / Played      (si Played > 0)
   AvgPerPt := total / pts         (si pts > 0)

Les matchs gagnés par forfait sont exclus des moyennes : ils fausseraient la mesure.

Autres pièces du temps
----------------------

.. code:: go

   func (s *State) RulesMatches() []rules.Match   // matchs dans l'ordre de lancement, libellés compris
   func (c Config) BreakIntervals() []rules.Interval // Config.Breaks au format de rules

Ces deux fonctions donnent aux règles pures de ``rules/prevision.go`` leur entrée :

- ``rules.PlayingTime(matches, now)`` : union des intervalles ``[Start, End)`` des matchs non
  annulés, jusqu'à ``now`` (un match en cours ou sans fin compte jusqu'à ``now``) ; les trous où
  aucune table ne jouait ne comptent pas ;
- ``rules.DayOfPlay(start, now)`` : jour calendaire de ``now`` compté depuis celui de ``start``
  (1 = le même jour), 0 si ``start`` est nul ;
- ``rules.AfterBreaks(now, play, breaks)`` : l'heure où se termine ``play`` posé à partir de
  ``now``, pauses sautées (une pause en cours retarde le départ, une pause rencontrée ajoute sa
  longueur ; finir exactement au début d'une pause, c'est finir avant elle) ;
- ``rules.EstimatedEnd(now, remaining, breaks)`` : la médiane des durées restantes simulées (la
  plus haute des deux du milieu pour un nombre pair), posée après ``now`` par ``AfterBreaks`` ;
  ``ok`` est faux sans aucune durée.

Les durées restantes viennent de ``sim.Forecast`` (voir :ref:`simulation`).

Le temps dans les propositions
------------------------------

Le moteur n'a pas d'horloge : ``ProposeAt(now)`` reçoit l'heure de l'hôte, ``Propose()`` prend
``State.Last`` (horodatage du dernier événement). Deux règles seulement en dépendent :

- **micro-rondes** (``batch_minutes`` d'un suisse continu) : l'échéance du prochain lot est le
  départ du dernier match non annulé lancé dans la phase, plus ``batch_minutes`` ; elle n'est
  stockée nulle part. Le premier lot ne s'attend pas. Avant l'échéance, le moteur propose
  ``wait`` avec ``Reason = waiting_batch`` et ``Until`` = l'échéance ;
- **pauses** (``Config.Breaks``, ``TimeRange`` à ``Start`` incluse et ``End`` exclue) : une
  proposition ``start_match`` dont l'intervalle ``[now, now + Expected(Length)]`` rencontre une
  pause porte ``Warn = ends_in_break``. Rien n'est bloqué : le TD décide.

Voir :ref:`propositions`.
