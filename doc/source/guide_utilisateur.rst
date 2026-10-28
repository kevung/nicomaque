.. _guide_utilisateur:

Guide du directeur de tournoi
=============================

Cette page suit une journée de tournoi, étape par étape. Pour chaque tâche elle indique ce que
fait le moteur, l'événement écrit au journal et, entre crochets, la commande correspondante de la
console ``tournoi-td`` (voir :ref:`cli`). Elle ne suppose pas de savoir programmer.

Inscrire les joueurs
--------------------

Un joueur a un identifiant, un nom, et facultativement un club et une cote (PR : plus elle est
basse, plus le joueur est fort). Un joueur sans cote n'est pas traité comme un joueur parfait.

- Inscription d'un joueur : événement ``player_added`` [``ajoute <nom> [club] [pr]``].
- Inscription d'une liste : fichier CSV d'une billetterie ou d'un tableur
  [``import <fichier.csv>``] ; la liste se réexporte au même format [``exporte <fichier.csv>``].
  Le format est décrit dans :ref:`joueurs_csv`.
- Correction d'un nom ou d'un club : événement ``player_updated``. Le joueur garde son état.

Lancer les matchs
-----------------

Le moteur propose un match dès que deux joueurs peuvent se rencontrer : deux joueurs, une
longueur en points et une table [``propose``, puis ``ok <n>`` ou ``ok tous``]. Chaque match
confirmé est un événement ``match_started``.

Tables
   Le moteur attribue la plus petite table libre. Une table peut être déclarée **indisponible**
   ou **réservée** ; le moteur la saute. S'il ne reste aucune table libre, la proposition est
   remplacée par une attente avec sa raison. Un match peut changer de table (``table_changed``).

Mode continu
   Au suisse, par défaut, un match est proposé dès que deux joueurs du même groupe sont libres.

Micro-rondes
   Avec ``batch_minutes``, les appariements se font par lots à heure fixe ; le moteur indique
   l'heure du prochain lot. Un joueur ne peut alors plus choisir son adversaire en retardant
   l'annonce de son résultat.

Rondes
   Avec le mode ``rounds``, tous les joueurs jouent la même ronde et la suivante n'est proposée
   qu'une fois la ronde entière terminée.

Saisir les résultats
--------------------

Le résultat est le vainqueur, et le score si on le connaît [``resultat <match> <vainqueur>
[scoreA scoreB]``] : événement ``result``. Chaque résultat libère deux joueurs et produit de
nouvelles propositions.

Gérer les imprévus
------------------

========================================== ==================================================
Situation                                  Ce que fait le moteur
========================================== ==================================================
Joueur arrivé après le tirage              Il énumère les places d'exemption libres
                                           [``places``] et y installe le joueur sans refaire
                                           le tirage [``retardataire <nom> <place>``]. Sans
                                           place libre, le joueur est inscrit et le moteur
                                           indique où il entrera.
Joueur qui s'absente                       Déclaré **indisponible** jusqu'à une heure ou une
                                           ronde : il n'est plus apparié et ne perd ni vie, ni
                                           rang, ni place de tableau.
Joueur qui joue dans une autre épreuve     L'hôte le signale à chaque proposition
de la salle                                (``External.BusyPlayers``) ; rien n'est écrit au
                                           journal.
Joueur qui part                            Retrait immédiat (ses matchs en cours sont perdus
                                           par forfait) [``forfait <joueur>``], ou retrait
                                           après son match en cours.
Joueur absent pour un match                Forfait sur ce match seulement (``ForfeitEvent``) :
                                           il reste dans le tournoi et suit le chemin d'un
                                           perdant.
Qualifié de poule qui part avant le        Le moteur propose de repêcher le suivant de sa
tirage du tableau                          poule (``repechage``). Sans confirmation, la place
                                           devient une exemption.
Match lancé par erreur                     Annulation [``annule <match>``] : la place
                                           redevient libre.
Longueur de match à changer                [``longueur <points>``] : s'applique aux prochains
                                           matchs de la phase.
Configuration à changer                    La configuration entière est remplacée
                                           [``configure <fichier.json>``].
========================================== ==================================================

Corriger une erreur
-------------------

Le journal n'est jamais modifié. Un résultat faux se corrige par un événement
``result_corrected`` [``corrige <match> <vainqueur> [scoreA scoreB]``] ; l'état est recalculé à
partir de tout le journal.

- **Dans un tableau**, si le match corrigé avait des suites, les matchs suivants ont été joués
  par les mauvais joueurs. Le moteur émet un avertissement et propose la réparation : annuler en
  série les matchs incohérents, puis relancer les bons. Rien n'est annulé sans confirmation.
- **Au suisse**, le moteur indique ce que la correction change aux vies (un joueur sans vie qui
  joue encore, un joueur qui retrouve une vie).
- **Après la clôture**, le tournoi peut être rouvert [``rouvre``] : le classement final est
  effacé, la correction appliquée, et le classement recalculé à la clôture suivante.

Pauses
------

Les pauses sont déclarées dans la configuration (``Config.Breaks``). Un match dont la fin
attendue tombe pendant une pause reste proposé, accompagné d'un avertissement.

Classement et dotation
----------------------

Le classement est disponible à tout moment [``classement``]. Les joueurs y sont placés par ce
qu'ils ont atteint : vainqueur, finaliste, éliminé à tel tour, éliminé avec tant de victoires.
Les ex æquo restent ex æquo. Un joueur retiré est classé sur son parcours, avec la note
« retiré ».

La dotation se calcule sur ce classement à partir d'un droit d'entrée, d'une retenue
d'organisation et d'un barème en pourcentages ou en montants. Chaque prix est tronqué à l'unité,
les unités restantes vont aux plus grands restes (à égalité, au mieux classé) : quand le barème
totalise 100 %, la somme distribuée est exactement le pool après retenue. Une consolante peut avoir son propre barème.
Le détail est dans :ref:`classement`.

Pour chaque joueur, le moteur répond aussi à la question « est-ce que je joue ? » : en jeu,
exempté, qualifié, pas encore fixé, éliminé, vainqueur ou retiré (voir :ref:`statut`).

Affichage en salle
------------------

Le paquet ``render`` produit des pages HTML autonomes, lisibles hors ligne : écran de salle
(actions, grille des tables, matchs en cours, arbres, classement) et feuille d'appariements A4
avec une case de score par joueur. La console ``tournoi-td`` régénère ``affichage.html`` après
chaque commande.
