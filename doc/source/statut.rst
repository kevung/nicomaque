.. _statut:

Statut d'un joueur
==================

.. code:: go

   type StatusKind string
   type PlayerStatus struct {
       Player  PlayerID   `json:"player"`
       Kind    StatusKind `json:"kind"`
       Phase   int        `json:"phase,omitempty"`
       Section string     `json:"section,omitempty"`
       Round   int        `json:"round,omitempty"`
       Label   Label      `json:"label,omitempty"`
   }
   func (s *State) Statuses() []PlayerStatus       // un par inscrit, ordre d'inscription
   func (s *State) StatusOf(p PlayerID) PlayerStatus // identifiant inconnu → not_entered

La réponse à « est-ce que je joue ? », que le classement ne donne pas : une note situe un joueur,
elle ne dit pas s'il a encore un match (voir la note ``section_exit`` dans :ref:`classement`). Le
statut est **dérivé** de l'état, comme ``State.Infos`` : il ne dépend que du journal, ni de l'heure
ni d'une proposition non confirmée — sauf le repêchage, dont les candidats proposés restent indécis
tant que le TD n'a pas tranché. ``Statuses`` est vide (``nil``) avant la création du tournoi.

``Phase`` est la phase dont parle le statut : en cours (``playing``, ``bye``), à venir
(``qualified``, ``undecided``), dernière où le joueur est entré (``eliminated``, ``withdrawn``,
``winner``).

+-----------------+---------------------------------------+---------------------------------------+
| ``Kind``        | Sens                                  | Champs                                |
+=================+=======================================+=======================================+
| ``playing``     | encore en course dans la phase en     | ``Phase`` ; tableau : ``Section``,    |
|                 | cours                                 | ``Round``, ``Label`` du prochain      |
|                 |                                       | match                                 |
+-----------------+---------------------------------------+---------------------------------------+
| ``bye``         | exempté : ne joue pas maintenant,     | tableau : ``Section``, ``Round``      |
|                 | entre plus tard                       | (tour, à partir de 1), ``Label`` du   |
|                 |                                       | match où il entre ; suisse par rondes |
|                 |                                       | : ``Round`` = ronde du retour,        |
|                 |                                       | ``Label`` = ``round``                 |
+-----------------+---------------------------------------+---------------------------------------+
| ``qualified``   | entre dans la phase ``Phase``         | ``Phase``                             |
+-----------------+---------------------------------------+---------------------------------------+
| ``undecided``   | son sort dans la phase ``Phase``      | ``Phase``                             |
|                 | n'est pas fixé                        |                                       |
+-----------------+---------------------------------------+---------------------------------------+
| ``eliminated``  | plus aucun match dans le tournoi      | ``Phase`` = dernière phase où il est  |
|                 |                                       | entré                                 |
+-----------------+---------------------------------------+---------------------------------------+
| ``winner``      | premier, la dernière phase finie ou   | ``Phase``                             |
|                 | le tournoi clos                       |                                       |
+-----------------+---------------------------------------+---------------------------------------+
| ``withdrawn``   | retiré ; prime sur tout le reste      | ``Phase`` = dernière phase où il est  |
|                 |                                       | entré                                 |
+-----------------+---------------------------------------+---------------------------------------+
| ``not_entered`` | inscrit, engagé dans aucune phase     | —                                     |
|                 | (``State.Infos`` dit où il entrera)   |                                       |
+-----------------+---------------------------------------+---------------------------------------+

Règles, dans l'ordre où elles s'appliquent :

::

   retiré                                   → withdrawn
   tournoi clos ou dernière phase finie, rang 1 (State.Final s'il est figé, sinon Ranking())
                                            → winner
   tournoi clos, autres                     → not_entered s'il n'est entré nulle part, sinon eliminated
   phase en cours finie, une phase suivante → passage simulé : enterFrom sur une phase neuve
                                              (exactement ce que fera next_phase)
       entrant simulé                       → qualified (Phase = suivante)
       candidat d'un repêchage proposé      → undecided (Phase = suivante)
       non engagé                           → not_entered
       sinon                                → eliminated, sauf phase d'entrée « all » au-delà
                                              de la suivante (→ qualified pour celle-là)
   engagé dans aucune phase                 → not_entered
   entrant d'une phase antérieure seulement → eliminated, sauf phase à venir d'entrée « all »
   phase k > 0 ni tirée ni commencée        → qualified (Phase = k) ;
                                              candidats d'un repêchage de k-1 → undecided
   suisse, GSL : plus de vie                → parcours fini (ci-dessous)
   suisse par rondes : bye de la ronde en cours (ByeRounds), pas en match
                                            → bye, Round = ronde + 1, Label = {round, N: ronde + 1}
   tableau non tiré                         → playing
   tableau tiré : un match du graphe non joué l'attend (le plus petit tour, puis l'ordre des
   sections ; Section, Round = tour + 1, Label du match)
       aucune victoire ni défaite dans la phase, un match franchi sans le jouer (exemption
       ou forfait d'un retiré), ce match pas lancé
                                            → bye
       sinon                                → playing
   tableau tiré, aucun match ne l'attend
       vainqueur du tableau (grande finale jouée, ou finale du principal sans grande finale)
                                            → qualified pour la suivante, sinon winner
       sinon                                → parcours fini
   sinon                                    → playing

   parcours fini, phase non finie :
       pas de phase suivante                → eliminated
       phase suivante en « top:N »          → undecided (le rang ne se fixe qu'à la fin)
       sinon                                → eliminated, sauf phase à venir d'entrée « all »

Un **repêché** est qualifié dès que le repêchage est confirmé, puisque ``rrQualified`` le
compte ; le retiré qu'il remplace est ``withdrawn``. Un candidat proposé mais pas confirmé est
``undecided`` ; si le TD passe outre et tire le tableau, le repêchage est clos et le candidat est
éliminé. Une poule ne connaît pas d'élimination en cours de phase : ses joueurs sont ``playing``
jusqu'à la fin des poules et des barrages.

Le passage simulé est la raison d'être de cette fonction dans le moteur : la liste des qualifiés est
celle que ``next_phase`` produira (un test le vérifie à chaque passage de toute la matrice), sans
que l'hôte ait à rejouer le journal ni à recoder ``enterFrom``.
