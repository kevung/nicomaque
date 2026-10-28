.. _index:

Nicomaque
=========

Nicomaque est un moteur de tournoi de backgammon : une bibliothèque Go qu'un logiciel hôte
intègre dans son propre processus pour diriger un tournoi. Il tient les inscriptions, propose les
appariements et les tables, tient les tableaux et produit le classement et la dotation.

- **Langage** : Go 1.22, bibliothèque standard seule, aucune dépendance externe.
- **Module** : ``github.com/kevung/nicomaque`` ; le paquet s'appelle ``tournoi``.
- **Licence** : MIT.
- **Formats** : suisse à vies, tableaux (élimination simple, consolante, dernière chance, double
  élimination), blocs GSL, poules toutes rondes, enchaînés en phases.
- **Statut** : validé par simulation ; le moteur n'a pas encore dirigé de tournoi réel (voir
  :ref:`limites`).

Fonctionnement en quatre points :

1. L'hôte conserve un **journal** : la liste ordonnée des événements du tournoi, en JSON.
2. Le moteur reconstruit l'état du tournoi à partir de ce journal (``Replay``).
3. Le moteur **propose** des actions (lancer un match, donner un bye, tirer un tableau, passer à
   la phase suivante, clore) ; il n'en applique aucune de lui-même.
4. Le directeur de tournoi confirme ; chaque confirmation devient un événement ajouté au journal.

Le moteur n'a ni interface, ni stockage, ni texte affichable : tout ce qu'il renvoie est un code
structuré que l'hôte traduit. Il ne départage pas : une égalité reste un ex æquo, ou se joue en
barrage.

Ce qui a changé d'une version à l'autre est dans l':ref:`historique`.

.. toctree::
   :maxdepth: 1
   :caption: Prise en main

   installation
   guide_utilisateur
   integration

.. toctree::
   :maxdepth: 2
   :caption: Formats

   formats

.. toctree::
   :maxdepth: 1
   :caption: Référence du moteur

   modele
   configuration
   journal
   etat
   propositions
   graphes
   classement
   statut
   horloge

.. toctree::
   :maxdepth: 1
   :caption: Paquets et outils

   affichage
   simulation
   joueurs_csv
   cli
   api_reference

.. toctree::
   :maxdepth: 1
   :caption: Annexes

   glossaire
   faq
   annexe_conception
   annexe_etude
   annexe_tests
   limites
   historique
   a_propos
