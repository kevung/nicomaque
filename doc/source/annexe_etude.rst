.. _annexe_etude:

Étude des formats
=================

Le choix des formats repose sur une étude par simulation menée en septembre 2026 avec un
simulateur Python (non publié), puis vérifiée avec le paquet ``sim`` du dépôt.

Critères
--------

Par ordre de priorité :

1. maximiser les chances des meilleurs joueurs **à durée donnée** ;
2. un format pilotable par un directeur de tournoi, y compris sur papier ;
3. aucun match inutile : chaque match élimine, qualifie ou classe ;
4. aucun départage arithmétique ;
5. tout effectif accepté (13, 47, 100 joueurs).

Modèle
------

Probabilité de gain
   Formule Elo/FIBS avec la conversion 3 PR = 100 points Elo :

   .. math::

      D = \frac{(b-a)\times 100}{3} \qquad P = \frac{1}{1 + 10^{-D\sqrt{N}/2000}}

   où *a* et *b* sont les PR des deux joueurs et *N* la longueur du match en points.

Validation
   Sur la base BMAB (33 238 matchs uniques), 1 PR d'écart vaut 2,8 % de chances de gain sur un
   match en 7 à 11 points (β = 0,036 à α = 0,5 ; valeur théorique 0,039).

Effet de la longueur
   L'exposant α de *N* n'est pas identifiable avec ces données : la vraisemblance est plate
   entre 0 et 0,7, et α = 1 est exclu. Le PR du match donne α ≈ 0,23, intervalle de confiance
   [0 ; 0,53]. La valeur retenue, α = 0,5, s'appuie sur la mécanique du jeu (erreur cumulée
   proportionnelle à *N*, chance proportionnelle à √N), vérifiée sur les données. C'est la
   principale incertitude du modèle.

Durée
   Environ 8 minutes par point en moyenne ; 11 minutes par point pour planifier une ronde,
   attente comprise.

Résultats
---------

Grilles à budget de temps constant, 32 et 64 joueurs, tournois de 8 h et de 16 h :

1. **Un suisse à vies est une double élimination sans tableau** : chaque défaite compte pareil,
   tout effectif est admis, la fin est toujours résoluble sans départage.
2. **L'attente est le premier poste de perte** : un suisse synchrone fait attendre 10 à 15 fois
   plus qu'un suisse continu (174 minutes par joueur contre 12 sur 16 h), soit environ deux
   matchs.
3. **Le continu est manipulable** par l'heure d'annonce d'un résultat. Parades : blocs GSL
   (appariement figé par les résultats) ou appariement par lots.
4. **La bascule « somme des vies = 2^k »** (2 vies = exemption du premier tour) est dans le
   groupe de tête, que la longueur compte ou non. L'élimination simple longue est la meilleure
   si √N est exact, la pire sinon. 3 vies et le format HSBT ne sont jamais gagnants ; le HSBT
   écarte l'information des rondes du samedi (environ 50 % du champ qualifié).
5. **La répartition des longueurs entre les tours est presque neutre** : allonger la finale ne
   change pratiquement pas le résultat.
6. **Le format joue sur un facteur 2 à 3** : une fin de semaine fait gagner le meilleur joueur
   4 à 9 % du temps (64 ou 32 joueurs), contre 1,6 à 3 % si le vainqueur était tiré au sort.

Format recommandé
-----------------

Suisse 2 vies en continu (appariement aléatoire dans le groupe de même nombre de défaites, sans
revanche, bye au groupe le plus bas), arrêt quand la somme des vies vaut 16 ou 32, puis tableau à
exemptions avec des matchs plus longs. Variante non manipulable : blocs GSL calés sur les repas.

Choix qui en découlent : pas de finale à handicap ; tirage aléatoire des tableaux par défaut, les
têtes de série restant une option (``seeding: "rating"``) ; un joueur retiré est classé sur son
parcours, pas en queue de classement.

Chiffres de référence du moteur
-------------------------------

Mesures de ``go test -run TestParity -v ./`` : 1 500 tournois simulés de 64 joueurs par format,
matchs en 7 points, PR tirés selon une loi normale de moyenne 6 et d'écart-type 2, bornée entre
2 et 10. P(meilleur) est la probabilité que le joueur le plus fort gagne ; P(top 4) celle qu'un
des quatre plus forts gagne. La durée est le temps de salle simulé avec un nombre de tables
illimité.

=================================== ============= ========== ======== =========
Format                              P(meilleur)   P(top 4)   Matchs   Durée
=================================== ============= ========== ======== =========
Élimination simple                  3,8 %         14,6 %     63,0     7 h 00
Suisse 2 vies, continu              3,5 %         16,5 %     126,5    10 h 20
Suisse 2 vies, par rondes           3,7 %         15,0 %     126,5    14 h 12
Suisse 2 vies → tableau de 16       4,2 %         15,5 %     124,3    9 h 45
Blocs GSL                           4,5 %         15,3 %     126,5    14 h 25
Double élimination avec recharge    5,7 %         18,0 %     126,5    13 h 35
Vainqueur tiré au sort              1,6 %         6,3 %      —        —
=================================== ============= ========== ======== =========

Incertitude : environ ±0,5 point sur P(meilleur) et ±1 point sur P(top 4).

Lectures :

- **Coût de l'attente** : le suisse 2 vies joue le même nombre de matchs en continu et par
  rondes (126,5), et dure 3 h 52 de plus par rondes.
- **Format recommandé** : le suisse 2 vies suivi d'un tableau de 16 a une qualité comparable au
  reste de la famille à deux vies pour la durée la plus courte de cette famille (9 h 45 contre
  13 à 14 h).
- **Écarts de qualité** : sur P(top 4), la famille à deux vies (15,0 à 18,0 %) devance
  l'élimination simple (14,6 %) au-delà de l'incertitude pour la double élimination et le suisse
  continu. Sur P(meilleur), seule la double élimination se détache ; les autres écarts sont de
  l'ordre du bruit de mesure.
- **Mode par rondes** : son générateur a changé en septembre 2026 (un tirage par ronde, pour
  que les appariements d'une ronde lancée en plusieurs vagues restent les mêmes). Sa valeur de
  P(meilleur) est passée de 4,9 à 3,7 % ; 9 000 tournois supplémentaires ramènent l'écart au
  bruit de mesure (4,3 % contre 4,6 %).

Le simulateur de l'étude et ``sim`` donnent les mêmes nombres de matchs et les mêmes durées,
mais des probabilités qui diffèrent de 1 à 2 écarts types sur la double élimination et le GSL ;
la cause n'est pas identifiée (voir :ref:`limites`). Ces chiffres sont des ordres de grandeur,
pas des valeurs au dixième de point.

Ces mesures sont à refaire après tout changement de l'algorithme d'appariement.
