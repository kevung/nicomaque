.. _annexe_tests:

Tests et invariants
===================

Le moteur est validé par simulation : il joue des tournois complets et vérifie, après chaque
événement, des propriétés qui doivent toujours être vraies. Il n'a pas encore dirigé de tournoi
réel ; ce que la simulation ne couvre pas est listé dans :ref:`limites`.

Nature des tests
----------------

Les tests **ne sont pas des tests unitaires** : ce sont des **tests d'invariants par simulation**.
Un nouveau format ou une nouvelle option s'ajoute dans la liste ``configs()`` et se trouve
automatiquement couvert par toutes les vérifications.

.. code:: bash

   go test -short ./...                 # la boucle de travail, ≈ 2 s : matrice réduite
   go test ./...                        # tout : matrice complète et TestParity, plusieurs minutes
   go test -run TestFormatsInvariants ./
   go test -run '^$' -fuzz FuzzApply -fuzztime 60s ./   # fuzzing (aussi en intégration continue)
   go vet ./... && gofmt -l .

Invariants vérifiés
-------------------

``TestFormatsInvariants`` joue toutes les configurations de ``configs()`` × 8 effectifs (2, 3, 5,
8, 13, 32, 64, 100) × 8 graines, et vérifie après chaque événement puis à la fin. En mode
``-short`` la matrice est réduite à six effectifs (2, 3, 5, 8, 13, 32) et trois graines, ce qui la
ramène sous les trois secondes : c'est la boucle qu'on lance à chaque sauvegarde, la matrice entière
tournant en intégration continue.

1. **Terminaison** : aucune simulation ne dépasse ``MaxSteps``, aucun blocage.
2. **Vainqueur unique** : ``Final`` existe et contient exactement un joueur au rang 1.
3. **Classement complet** : ``len(Final) == P`` — tous les inscrits sont classés.
4. **Rejeu identique** : ``Replay(journal)`` produit exactement le même ``Final`` (comparaison
   JSON).
5. **Aucun avertissement** après rejeu : ``len(Warnings) == 0``.
6. **Aucun joueur éliminé ne joue** : dans une phase à vies, à l'instant où un match démarre,
   ``Losses[p] < Lives[p]`` pour les deux joueurs.
7. **Pas de match entre groupes différents** : dans une phase à vies, ``Losses[A] == Losses[B]``,
   sauf pour les matchs de repli (libellés ``final`` et ``crossed``).
8. **Peu de rematchs** : en suisse, le nombre de secondes rencontres reste sous
   ``graines × (Lives + 3)`` dès que ``P ≥ 8``.

``TestStatutsInvariants`` rejoue la même matrice (plus une entrée ``top:4`` et une entrée ``all``)
et vérifie les statuts après chaque événement : un statut par inscrit ; ``withdrawn`` exactement
pour les retirés ; un joueur annoncé ``eliminated`` ne démarre plus aucun match, un ``bye`` ou un
``eliminated`` n'est pas en train de jouer ; les ``qualified`` annoncés avant ``next_phase`` sont
exactement les entrants qu'il produit ; à la clôture, ``winner`` exactement pour les rangs 1. Les
feuilles T1 et T2 de la simulation blunderDB 2026-10 (``testdata/simulation-2026-10/``) et l'épreuve
A de T3 (repêchage d'un qualifié de poule) sont rejouées avec les mêmes contrôles.

Fuzzing
-------

Deux cibles gardent ``Apply`` contre ce qu'aucun test écrit à la main ne cherche :

- ``FuzzApply`` applique des suites d'événements tirées au hasard. Les octets ne sont pas du JSON
  mais un **plan** : chaque triplet choisit un type d'événement et ses paramètres parmi ce que
  l'état contient. Du JSON aléatoire n'atteindrait presque jamais un événement valide, et ne
  testerait donc que le décodeur.
- ``FuzzReplayJournal`` rejoue des octets quelconques présentés comme un journal — le fichier que
  l'hôte a stocké et qui revient tronqué, réordonné ou corrompu.

Dans les deux cas, **une erreur est une réponse acceptable** ; un ``panic`` ou un état incohérent
n'en est pas une. L'état est vérifié après chaque application : autant de matchs que d'identifiants
ordonnés et aucun doublon, aucun match d'un joueur contre lui-même, un vainqueur qui appartient à
son match, aucun entrant inconnu, aucun compteur négatif, aucun joueur deux fois au classement, et
``Propose`` qui ne panique pas.

Compatibilité ascendante
------------------------

+---------+----------------------------------------------------------------------------------------+
| Version | Ce qui change à la lecture                                                             |
+=========+========================================================================================+
| 0       | libellés en texte français, convertis en codes par ``Event.upgraded`` ; ronde relue du |
|         | libellé                                                                                |
+---------+----------------------------------------------------------------------------------------+
| 1       | codes structurés ; ``config_changed`` peut changer les options de construction d'un    |
|         | tableau tiré (sans effet)                                                              |
+---------+----------------------------------------------------------------------------------------+
| 2       | ``config_changed`` ne le peut plus : refus ``drawn``                                   |
+---------+----------------------------------------------------------------------------------------+

``testdata/journal_v0.json`` est un journal complet écrit dans la forme d'avant les codes structurés
— aucun champ ``version``, aucun champ ``round``, un ``label`` qui était une chaîne française. Le
test le rejoue et compare le classement **place par place**. Cette fixture ne se régénère pas : elle
vaut par le fait qu'elle n'a pas bougé.

``TestMatchCounts`` fixe les comptes de matchs sur 64 joueurs :

================================ ===============
Format                           Matchs attendus
================================ ===============
Suisse 2 vies continu            126 ou 127
Élimination simple               63
Double élimination avec recharge 126 ou 127
Blocs GSL                        126 ou 127
================================ ===============

(126 = 2 × 64 − 2 ; le 127ᵉ est la recharge, jouée une fois sur deux environ.)

``TestCorrection`` vérifie qu'après l'inversion du premier résultat d'un tournoi par
``result_corrected``, la somme des défaites de la phase égale le nombre de matchs terminés —
c'est-à-dire que ``recompute`` a bien tout reconstruit.

``TestWithdrawnAfterDraw`` vérifie qu'un joueur retiré **après** le tirage d'un tableau n'est plus
jamais proposé, que ses matchs non lancés sont perdus par walkover, que le tableau se termine et
qu'aucun avertissement n'est produit.

``TestParity`` (long, sauté en mode court) mesure sur 1 500 tournois de 64 joueurs par format : P(le
meilleur gagne), P(un des quatre meilleurs gagne), le nombre moyen de matchs et la durée moyenne.
Ces valeurs sont comparées à celles de :ref:`annexe_etude` ; **tout changement d'algorithme
d'appariement est validé par ce test**. Ordre de grandeur attendu : une fin de semaine ne fait
gagner le meilleur joueur que 4 à 9 % du temps (64 ou 32 joueurs), contre 1,6 à 3 % au hasard.

Fichiers dorés
--------------

Les sorties des règles sont figées dans des fichiers de référence, rejoués à chaque exécution
contre le paquet ``rules`` et contre les enveloppes du paquet racine :

========================================= ==================================================
Fichier                                   Contenu
========================================= ==================================================
``testdata/trace_moteur.json``            propositions et classements à chaque pas des
                                          simulations (``TestDoresTrace``)
``rules/testdata/appariements.json``      appariements
``rules/testdata/dotation.json``          dotation
``rules/testdata/feuille.json``           feuille d'appariements
``rules/testdata/fin.json``,              prévision de fin et horloge
``rules/testdata/horloge.json``
``sim/testdata/prevision.json``           prévision par simulation
``render/testdata/*``                     pages HTML et SVG témoins
========================================= ==================================================

Une sortie qui change à entrée égale est une version mineure. Les fichiers se régénèrent avec
``go test -run TestDores . -args -update`` et ``go test ./render ./sim -update``.

Intégration continue
--------------------

``.github/workflows/ci.yml`` s'exécute à chaque poussée sur ``main``, à chaque proposition de
fusion et chaque lundi :

- ``tests`` : la suite complète, sans ``-short``, donc avec ``TestParity`` ;
- ``fuzz`` : ``FuzzApply`` et ``FuzzReplayJournal``, 60 s chacune (300 s le lundi) ;
- ``exemples`` : régénère la galerie ``exemples/`` et échoue si celle du dépôt n'est plus à jour.

Invariants structurels
----------------------

À tout instant :

- un joueur a **au plus un match en cours** ;
- ``MatchOrder`` est en bijection avec les clés de ``Matches``, dans l'ordre de lancement ;
- les identifiants de matchs sont ``M1``, ``M2``, … consécutifs sans trou ;
- ``sum(Losses)`` sur une phase = nombre de matchs terminés (non annulés) de cette phase ;
- ``remainingLives(p) = Lives[p] − Losses[p]``, borné à 0 ;
- un ``GMatch`` ``Done`` a ``Winner`` et ``Loser`` renseignés, sauf s'il est ``Skipped`` ;
- ``resolve`` est idempotente : l'appeler deux fois de suite ne change rien ;
- ``Replay(journal)`` est une fonction pure du journal.
