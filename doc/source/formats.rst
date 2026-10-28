.. _formats:

Formats
=======

Un tournoi est une suite de **phases** (``Config.Phases``). Chaque phase a un format (``Kind``) ;
les survivants d'une phase entrent dans la suivante, avec leurs vies restantes quand la phase
suivante est à vies.

.. list-table::
   :header-rows: 1
   :widths: 18 22 60

   * - ``Kind``
     - Format
     - Principe
   * - ``swiss_lives``
     - Suisse à vies
     - Chaque joueur a *L* vies (2 par défaut) et en perd une par défaite. Appariement dans le
       groupe de même nombre de défaites, en continu, en micro-rondes ou par rondes. Option
       ``target`` : arrêt quand la somme des vies restantes atteint une puissance de 2.
   * - ``lives_bracket``
     - Tableau à vies
     - Élimination directe où un joueur arrivé avec 2 vies est exempt du premier tour. Suite
       naturelle d'un suisse arrêté par ``target``.
   * - ``bracket``
     - Tableau
     - Élimination simple ; options consolante, dernière chance, réconciliation et recharge
       (double élimination).
   * - ``gsl``
     - Blocs GSL
     - Groupes de 4 (ou 3, ou 2) à double élimination interne ; l'appariement est figé par les
       résultats du bloc, pas par l'heure.
   * - ``round_robin``
     - Poules
     - Poules toutes rondes (table de Berger), *k* qualifiés par poule ; une égalité à une place
       qualificative se règle par barrage à 2 vies.

Les chapitres suivants décrivent chaque format jusqu'au niveau de l'algorithme :

.. toctree::
   :maxdepth: 1

   format_suisse
   format_tableaux
   format_gsl
   format_poules

Longueurs de match
------------------

Chaque phase a une longueur (``Length``) et, pour les tableaux, une longueur de finale
(``FinalLength``) ou une liste de longueurs par tour (``Lengths``), lue du dernier tour vers le
premier : ``15, 13, 11, 9`` donne une finale en 15 points, des demi-finales en 13, des quarts en
11 et les tours précédents en 9. Voir :ref:`configuration`.

Têtes de série
--------------

Par défaut, le tirage d'un tableau est entièrement aléatoire. L'option ``seeding: "rating"``
place les joueurs par cote selon la construction classique (1 contre 16, 2 contre 15…), de sorte
que les deux premiers ne peuvent se rencontrer qu'en finale. La justification du défaut est dans
:ref:`annexe_conception`.

Comparaison
-----------

Mesures sur 1 500 tournois simulés de 64 joueurs, matchs en 7 points (détail et incertitudes dans
:ref:`annexe_etude`). « P(meilleur) » est la probabilité que le joueur le plus fort gagne le
tournoi ; la durée est le temps de salle avec un nombre de tables illimité.

=================================== ============= ======== =========
Format                              P(meilleur)   Matchs   Durée
=================================== ============= ======== =========
Élimination simple                  3,8 %         63,0     7 h 00
Suisse 2 vies, continu              3,5 %         126,5    10 h 20
Suisse 2 vies, par rondes           3,7 %         126,5    14 h 12
Suisse 2 vies → tableau de 16       4,2 %         124,3    9 h 45
Blocs GSL                           4,5 %         126,5    14 h 25
Double élimination avec recharge    5,7 %         126,5    13 h 35
=================================== ============= ======== =========

Choisir un format
-----------------

======================================================= ========================================
Objectif                                                Configuration
======================================================= ========================================
Meilleur compromis qualité / durée (format recommandé)  Suisse 2 vies continu, ``target: 16``,
                                                        puis ``lives_bracket``
Durée la plus courte                                    ``bracket`` (élimination simple)
Aucun joueur éliminé après un seul match                Suisse 2 vies, ou ``bracket`` avec
                                                        consolante
Programme connu à l'avance, appariement non             ``gsl``
manipulable
Petit effectif, beaucoup de matchs                      ``round_robin``
Vainqueur battu deux fois au moins                      ``bracket`` avec réconciliation et
                                                        recharge
======================================================= ========================================

Les configurations nommées des outils ``tournoi-demo`` et ``tournoi-td`` (``suisse``,
``suisse_rondes``, ``suisse_tableau``, ``gsl``, ``elim``, ``conso``, ``double``, ``poules``)
correspondent à ces combinaisons ; voir :ref:`cli`.
