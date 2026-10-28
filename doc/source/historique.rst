.. _historique:

Historique des versions
=======================

Les versions suivent la numérotation sémantique. Une sortie du moteur qui change à entrée égale
(proposition, classement) donne une version mineure. Le format du journal est versionné
indépendamment (champ ``version`` de chaque événement, voir :ref:`annexe_tests`).

0.8.0 — 28 octobre 2026
-----------------------

- Le projet est repris sous ``github.com/kevung/nicomaque``. Le code est celui de 0.7.0 ; seul
  le chemin de module change.
- L'ancien module ``github.com/PileOfCells/backgammon-tournoi`` est déprécié à partir de sa
  version 0.7.1, qui renvoie au nouveau.
- La documentation est réorganisée et publiée en français et en anglais.

0.7.0 — 22 octobre 2026
-----------------------

- Appariements, classement et matchs restants en fonctions pures du paquet ``rules`` (``Pair``,
  ``Rank``, ``RemainingMatches``), sans changement de règle.
- Dotation répartie à l'unité par la méthode du plus fort reste : avec un barème à 100 %, la
  somme distribuée égale exactement le pool après retenue.

0.6.0 — 22 octobre 2026
-----------------------

- Paquet ``rules`` : dotation, prévision de fin et feuille d'appariements en fonctions pures,
  sans changement de règle.

0.5.1 et 0.5.2 — 5 octobre 2026
-------------------------------

- Licence MIT ; Nicolas Harmand et Kévin Unger titulaires du droit d'auteur.

0.5.0 — 4 octobre 2026
----------------------

- Statut de chaque joueur (``State.Statuses``) : en jeu, exempté, qualifié, pas encore fixé,
  éliminé, vainqueur, retiré.
- Repêchage proposé d'un qualifié de poule retiré avant le tirage (événement ``repechage``).

0.4.1 — 26 septembre 2026
-------------------------

- Un joueur retiré d'un tableau est classé avec les perdants du match qui l'attendait.

0.4.0 — 26 septembre 2026
-------------------------

- ``External.BusyPlayers`` : un joueur occupé dans une autre épreuve de la salle n'est pas
  apparié, sans rien écrire au journal.

0.3.0 — 26 septembre 2026
-------------------------

- Indisponibilité d'un joueur sans retrait (``player_unavailable``, ``player_available``).
- Un joueur retiré est classé sur son parcours, avec la note ``withdrawn``.
- Correction de fiche sans toucher à l'état (``player_updated``).
- Table partagée signalée (``table_shared``) ; tables occupées par une autre épreuve
  (``External.BusyTables``).
- Au suisse, une correction qui élimine un joueur en match ou lui rend une vie est signalée.
- Mode par rondes : une ronde reste ouverte jusqu'à ce que tous ses joueurs y soient engagés.
- Les options de construction d'un tableau tiré sont figées (journal en version 2).

0.2.1 — 7 septembre 2026
------------------------

- Rendu : la table n'est affichée qu'une fois attribuée.

0.2.0 — 7 septembre 2026
------------------------

- Configuration modifiable en cours de tournoi (``config_changed``) et réouverture d'un tournoi
  clos (``reopened``).
- Micro-rondes (``batch_minutes``) et pauses programmées (``Config.Breaks``).
- Réparation proposée d'un graphe désaccordé après une correction.
- Dotation : droit d'entrée, retenue, barème par section.
- Têtes de série en option (``seeding: "rating"``), désactivées par défaut.
- Retardataires installés sur une place d'exemption libre.
- Longueurs de match par tour ; fin de suisse allongée.
- Export CSV des joueurs.
- Rendu traduisible (``Labeler``) et pages autonomes ; feuille d'appariements imprimable.
- Fuzzing d'``Apply`` et du rejeu de journal ; lecture des journaux en version 0.

0.1.0 — 7 septembre 2026
------------------------

- Première version publiée : journal d'événements, suisse à vies, tableaux, blocs GSL, poules,
  rendu, simulation, console de direction et démonstration.
- Codes structurés à la place des textes, journal versionné.
- Tables indisponibles, réservées et déplaçables.
- Score facultatif, forfait sur un seul match, remarque, retrait différé.
