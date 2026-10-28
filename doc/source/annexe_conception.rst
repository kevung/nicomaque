.. _annexe_conception:

Décisions de conception
=======================

Cette page énonce les décisions qui gouvernent le moteur et leur justification. Les chiffres
cités viennent de :ref:`annexe_etude`.

Décisions techniques
--------------------

Journal d'événements, état dérivé
   La seule donnée persistante est le journal ; l'état est une fonction pure du journal
   (``State = Replay(Journal)``). Conséquences : une reprise après panne est un ``Replay`` ; un
   préfixe du journal donne l'état à un instant passé, propositions comprises ; deux exécutions
   du même journal donnent le même état.

Le journal n'est jamais modifié
   Une erreur se corrige par un événement ajouté :

   ======================================== ===============================
   Situation                                Événement
   ======================================== ===============================
   mauvais vainqueur saisi                  ``result_corrected``
   match lancé par erreur                   ``match_cancelled``
   joueur qui abandonne                     ``player_withdrawn``
   fiche de joueur erronée                  ``player_updated``
   joueur absent pour un temps              ``player_unavailable``,
                                            ``player_available``
   configuration à changer                  ``config_changed``
   tournoi clos à corriger                  ``reopened``
   ======================================== ===============================

   Ces événements déclenchent un recalcul complet de la comptabilité des phases, suivi d'un
   contrôle de cohérence qui remplit ``State.Warnings`` (par exemple : un match de tableau joué
   par des joueurs que le tableau n'attendait pas). Le contrôle est aussi fait à chaque
   ``match_started`` et ``result``. Quand un graphe est désaccordé, le moteur propose
   l'annulation en série des matchs incohérents ; il n'annule rien de lui-même.

Le moteur propose, le directeur décide
   Le moteur expose ``Propose``, ``EventFromAction`` et ``Apply``. Aucune action n'est appliquée
   d'office ; chaque confirmation produit exactement un événement. Une décision douteuse du
   directeur (deux matchs sur une table, un match dont la fin tombe pendant une pause) est
   enregistrée et accompagnée d'un avertissement, pas refusée.

Déterminisme
   ``Propose`` ne dépend que du journal. Le générateur pseudo-aléatoire d'une proposition est
   initialisé par ``graine·1000003 + nombre d'événements·7919 + indice de phase`` (une ronde de
   suisse utilise une dérivation propre à la ronde, voir :ref:`format_suisse`). Deux appels
   successifs sans événement intermédiaire renvoient la même proposition. Toute itération sur
   une map dont l'ordre influence une proposition passe par un tri stable.

Tirages enregistrés
   Tout tirage (placement dans un tableau, composition des groupes) est écrit dans l'événement
   ``draw``. Rejouer un journal ne fait pas appel à l'algorithme de tirage : celui-ci peut
   évoluer sans invalider les journaux existants.

Codes, pas de phrases
   Libellés de match, notes de classement, avertissements et raisons d'attente sont des codes
   structurés ; l'hôte les traduit dans la langue de son utilisateur. Les noms de section sont des
   identifiants (``main``, ``conso``, ``poule:A``). Un test refuse toute chaîne accentuée ou à
   espaces sortant du moteur, hors les champs saisis par le directeur.

Graphes de matchs comme données
   Tableaux, groupes GSL, poules et consolantes sont des ``Section`` de ``GMatch`` dont chaque
   place est soit un joueur, soit le vainqueur ou le perdant d'un autre match. Une fonction unique
   propage les résultats jusqu'au point fixe. Ajouter un format de graphe revient à écrire un
   constructeur de section.

Règles en fonctions pures
   Les règles (appariement, classement, dotation, prévision, feuille d'appariements) sont dans le
   paquet ``rules``, sans état, sans entrée-sortie ni horloge. Le paquet racine construit un
   instantané de l'état et appelle ces fonctions : chaque règle a une seule implémentation,
   réutilisable par un autre logiciel.

Bibliothèque, pas service
   Le moteur s'exécute dans le processus de l'hôte, sans réseau, sans base de données et sans
   dépendance externe.

Décisions de format
-------------------

Suisse à vies plutôt qu'élimination directe
   En élimination directe, une seule défaite élimine. À 7 points, un joueur meilleur de 3 PR ne
   gagne qu'environ 58 % du temps. Le suisse à 2 vies admet une défaite et produit le même nombre
   de matchs qu'une double élimination (2N − 2 pour N joueurs, plus une éventuelle recharge),
   sans contrainte d'effectif.

Continu par défaut
   Un suisse synchrone fait attendre les joueurs 10 à 15 fois plus. Le mode par rondes reste
   disponible : plus lisible, doublable sur papier, non manipulable. Une ronde plus grande que la
   salle se joue en plusieurs vagues, avec les mêmes appariements, et la ronde suivante n'arrive
   qu'une fois la ronde entière jouée. Les micro-rondes (``batch_minutes``) apparient par lots à
   heure fixe.

Appariement dans le groupe de même nombre de défaites
   Pas de score ni de classement intermédiaire. Priorités : un joueur déjà exempté est apparié en
   premier ; pas de revanche sauf faute d'autre solution ; éviter deux joueurs du même club
   (option). Quand aucun appariement régulier n'est possible, le moteur propose un match de repli
   (finale ou match croisé) : un tournoi se termine toujours.

Bascule vers un tableau
   Chaque match consomme une vie. Quand la somme des vies restantes atteint une puissance de 2, le
   tableau est exact : un joueur à 2 vies occupe deux places (exempt du premier tour), un joueur à
   1 vie en occupe une. Un tableau de 32 places pour 20 joueurs est donc normal : sa taille vient
   de la somme des vies.

Aucun départage
   Un départage arithmétique (Buchholz, différence de points, cote) sépare deux joueurs qui ont
   le même parcours à partir d'informations qui ne mesurent pas leur performance ; au backgammon,
   la variance d'un match rend ces critères peu informatifs. Une égalité reste un ex æquo (avec
   partage des prix des places concernées) ou se règle par un barrage joué quand une place
   qualificative est en jeu.

Tirage aléatoire par défaut
   C'est la pratique du backgammon, et protéger les meilleurs ajoute un avantage structurel à des
   joueurs que le format favorise déjà, sur la base d'une cote bruitée. Les têtes de série
   existent en option (``seeding: "rating"``), désactivée par défaut.

Pas de finale à handicap
   Le format récompense le parcours ; il ne le corrige pas.

Repêchage proposé d'un qualifié de poule retiré
   Quand un qualifié se retire **avant le tirage** du tableau, le moteur propose sa place au
   suivant de sa poule (le joueur non qualifié et non retiré qui a le plus de victoires). Le
   directeur confirme (événement ``repechage``) ou passe à la phase suivante, et la place devient
   une exemption. Le repêchage n'est pas appliqué d'office : un règlement local ou un repêché
   absent peuvent faire préférer l'exemption, et les journaux existants se rejouent à
   l'identique. Un suivant lui-même retiré est sauté ; entre suivants ex æquo, le moteur propose
   chacun. **Après le tirage**, il n'y a plus de repêchage : le retiré perd son match par forfait.

Retraits classés sur le parcours
   Un joueur retiré est classé comme s'il avait été éliminé à l'instant du retrait (par ses
   victoires au suisse et en GSL, par la sortie atteinte en tableau), ex æquo avec les éliminés
   du même niveau, avec la note ``withdrawn``.

Longueurs de match
   La répartition des longueurs entre les tours est presque neutre. Le moteur accepte une longueur
   par phase, une longueur de finale et une longueur par tour ; les allonger est un choix
   d'organisateur.
