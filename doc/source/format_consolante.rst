.. _format_consolante:

Consolante en parallèle (``consolation_of``)
============================================

Principe
--------

Une phase qui porte ``consolation_of`` est une **phase latérale** : elle reçoit les perdants
d'**une seule ronde** d'une phase antérieure et se joue **en même temps** que la suite du
tournoi. La chaîne principale ne la voit pas : le passage de phase la saute, ``State.Current``
ne la désigne jamais, et le classement général l'ignore. Elle a son propre classement, et sa
propre dotation.

.. code:: json

   {"phases": [
     {"kind": "swiss_rounds", "rounds": 5, "length": 7},
     {"kind": "bracket", "length": 5, "name": "Consolante",
      "consolation_of": {"phase": 0, "round": 1}},
     {"kind": "bracket", "length": 9, "entry": "top:8"}]}

Ici les perdants de la ronde 1 du suisse jouent un tableau de consolante pendant les rondes 2 à
5 ; à la fin du suisse, le passage va directement à la phase 2.

Configuration
-------------

.. code:: go

   type ConsolationSource struct {
       Phase int `json:"phase"` // indice de la phase source
       Round int `json:"round"` // ronde du suisse, ou tour du tableau principal, à partir de 1
   }

   PhaseConfig.ConsolationOf *ConsolationSource `json:"consolation_of,omitempty"`

``Validate`` refuse, en nommant ``consolation_of`` :

- une consolante d'un autre format que ``bracket`` ou ``swiss_rounds`` par rondes ;
- une source qui ne la précède pas, une source elle-même latérale, ou une phase principale
  entre la source et la consolante — les phases latérales d'une source forment un bloc qui la
  suit directement ;
- une source autre qu'un ``bracket`` (``round`` est alors le tour de la section ``main``) ou un
  ``swiss_rounds`` par rondes (``round`` ≤ ``rounds``) ;
- ``round`` < 1, et un ``entry`` autre que le défaut : les entrants sont les perdants.

Ouverture
---------

La ronde source est **soldée** quand tous ses appelés y sont engagés et que tous ses matchs sont
finis (suisse), ou quand tous les matchs du tour sont résolus (tableau). Ses perdants sont ceux
d'un match **joué** : ni une exemption, ni un forfait, ni un retiré. S'ils sont au moins deux,
``Propose`` ajoute l'action ``open_side_phase`` (``Phase`` : la consolante, ``Draw.Groups[0]`` : ses
entrants) ; confirmée, elle devient l'événement ``side_phase_opened``, qui matérialise les entrants
comme un tirage. Rien ne s'ouvre d'office ; avec moins de deux perdants, la consolante ne s'ouvre
pas, et ne retient rien.

Propositions
------------

``Propose`` rend les propositions de la phase principale, puis celles de chaque consolante ouverte
et non finie, puis les ouvertures possibles, par indice de phase. Chaque consolante est proposée
par le fichier de son format (``Action.Phase`` la désigne) :

- **un joueur n'a jamais deux matchs à la fois**. Un joueur qui joue dans une autre phase, ou que
  réclame déjà une proposition de la principale, compte comme occupé : un graphe garde son match
  proposé, retenu (``player_busy``, sans table), un suisse ne l'apparie pas ;
- **la phase principale a la priorité** : un joueur qu'elle propose n'est pas proposé ailleurs ;
- un suisse par rondes **n'ouvre pas de ronde** tant qu'un de ses joueurs joue dans une autre
  phase (une attente ``player_busy`` par joueur) : il ne manque aucune ronde ;
- les tables d'une proposition sont prises pour les suivantes.

La phase principale finie, le passage de phase reste proposé ; la clôture (``finish``) attend que
chaque consolante ouverte soit finie et qu'aucune ne reste à ouvrir.

Cette attente vaut pour tout format de dernière phase, phase manuelle comprise ; ``Apply`` refuse
de même un ``finished`` tant qu'une consolante reste à ouvrir ou à finir.

Classement
----------

``Ranking()`` ignore les phases latérales. Le classement propre d'une consolante est
``SectionRanking(SidePhaseSection(i))``, soit la section ``phase:<i>`` ; ``Prizes.Sections``
l'adresse sous ce nom. ``SectionRanking("main")`` et les autres noms de section ne cherchent
jamais dans une phase latérale.

Correction de la ronde source
-----------------------------

Une correction après l'ouverture peut changer les perdants. La consolante n'est **pas refaite** :
ses entrants sont dans le journal. ``State.Warnings`` porte ``consolation_not_loser`` pour un
entrant qui n'a plus perdu, ``consolation_missing`` pour un nouveau perdant resté dehors
(``Section`` : ``phase:<i>``, ``Player``) ; une seconde correction qui rétablit les perdants les
fait tomber.

Reconfiguration
---------------

``config_changed`` refuse d'ajouter, de retirer ou de déplacer une phase latérale parmi les phases
déjà créées (``Field`` : ``consolation_of``) : les événements écrits portent leurs indices. Au-delà,
la configuration reste libre.
