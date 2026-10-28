.. _cli:

Outils en ligne de commande
===========================

Deux binaires accompagnent la bibliothèque. Ils servent à éprouver le moteur ; ce ne sont pas
des logiciels de tournoi.

Configurations nommées
----------------------

Les deux outils acceptent les mêmes noms de configuration (``tournoi-td`` en a un de plus,
``suisse_rondes``) :

==================== ========================================================================
Nom                  Phases
==================== ========================================================================
``suisse``           suisse 2 vies continu, matchs en 7
``suisse_rondes``    suisse 2 vies par rondes (``tournoi-td`` seulement)
``suisse_tableau``   suisse 2 vies, ``target: 16``, puis tableau à vies (9 points, finale en
                     11)
``gsl``              blocs GSL, ``target: 16``, puis tableau à vies
``elim``             élimination simple (9 points, finale en 11)
``conso``            tableau principal, consolante et dernière chance
``double``           double élimination (consolante, réconciliation, recharge)
``poules``           poules de 4 (2 qualifiés, matchs en 5), puis tableau
==================== ========================================================================

``cmd/tournoi-demo``
--------------------

Simule un tournoi complet et écrit ses sorties dans un dossier.

.. code:: bash

   go run ./cmd/tournoi-demo -format suisse_tableau -joueurs 32 -etapes -sortie demo
   go run ./cmd/tournoi-demo -rejouer demo/journal.json

============== =========================================================================
Option         Rôle
============== =========================================================================
``-format``    configuration nommée (défaut ``suisse_tableau``)
``-joueurs``   effectif (défaut 32)
``-graine``    graine (défaut 1)
``-sortie``    dossier de sortie (défaut ``demo``)
``-etapes``    écrire aussi les pages d'affichage à 15 %, 40 %, 70 % et 100 % du journal
``-rejouer``   rejouer un journal et afficher l'état, les actions proposées et les
               avertissements
============== =========================================================================

Fichiers produits : ``journal.json``, ``affichage.html``, ``classement.csv``, un SVG par section,
``affichage_mi_tournoi.html`` (rejeu des deux tiers du journal) et, avec ``-etapes``, les pages
``etapeK_*``. La console affiche aussi une prévision de fin (``Forecast`` sur 50 tirages) et la
durée effectivement simulée.

``cmd/tournoi-td``
------------------

Console interactive de direction de tournoi. Le journal est réécrit après chaque commande et
``affichage.html`` régénéré à côté ; relancer la commande sur le même fichier reprend le tournoi.

.. code:: bash

   go run ./cmd/tournoi-td -journal montournoi.json -format suisse_tableau

============== =========================================================================
Option         Rôle
============== =========================================================================
``-journal``   fichier journal, créé s'il n'existe pas (défaut ``tournoi.json``)
``-format``    configuration nommée d'un nouveau tournoi (défaut ``suisse_tableau``)
``-config``    configuration JSON d'un nouveau tournoi (prioritaire sur ``-format``)
``-nom``       nom du tournoi
============== =========================================================================

.. list-table::
   :header-rows: 1
   :widths: 45 55

   * - Commande
     - Effet
   * - ``ajoute <nom> [club] [pr]``
     - inscrire un joueur (identifiant = nom en minuscules)
   * - ``import <fichier.csv>``
     - inscrire les joueurs d'un fichier CSV
   * - ``exporte <fichier.csv>``
     - écrire les joueurs inscrits dans un CSV que ``import`` relit
   * - ``places``
     - lister les places d'exemption libres pour un retardataire
   * - ``retardataire <nom> <place>``
     - inscrire un joueur sur une place libre, sans refaire le tirage
   * - ``propose``
     - afficher les actions proposées, numérotées
   * - ``ok [n | tous]``
     - confirmer une proposition, ou toutes
   * - ``resultat <match> <vainqueur> [scoreA scoreB]``
     - saisir un résultat
   * - ``corrige <match> <vainqueur> [scoreA scoreB]``
     - corriger un résultat déjà saisi
   * - ``annule <match>``
     - annuler un match lancé par erreur
   * - ``forfait <joueur>``
     - retirer un joueur du tournoi
   * - ``longueur <points>``
     - changer la longueur des prochains matchs de la phase en cours
   * - ``configure <fichier.json>``
     - remplacer la configuration entière
   * - ``rouvre``
     - rouvrir un tournoi clos
   * - ``matchs``, ``vies``, ``classement``, ``etat``
     - consulter les matchs, les vies, le classement, l'état
   * - ``simule``
     - jouer au hasard tous les matchs en cours
   * - ``aide``, ``quitte``
     - afficher l'aide, quitter

Un joueur se désigne par son identifiant ou par une partie de son nom (erreur si plusieurs
joueurs correspondent). ``ok tous`` s'interrompt après une action ``draw``, ``next_phase`` ou
``finish`` : l'état a changé et les propositions suivantes ne sont plus valables.

Exemple de session :

.. code:: text

   ajoute Alice Paris 4.2
   ajoute Bob Lyon 5.1
   propose
   ok tous
   resultat M1 alice 7 3
   classement
