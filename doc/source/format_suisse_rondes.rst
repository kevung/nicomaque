.. _format_suisse_rondes:

Suisse classique (``swiss_rounds``)
===================================

Principe
--------

Le suisse classique se joue en un **nombre de rondes fixe** (``rounds``, obligatoire) : personne
n'est éliminé, chaque joueur présent joue chaque ronde, et le classement est celui des **points** —
une victoire vaut un point, une exemption aussi.

C'est une **variante nommée**, à part du suisse à vies (:ref:`format_suisse`) que l'étude retient.
Elle existe parce que des organisateurs la jouent, et elle a un prix : sans départage, les joueurs
à égalité de points restent ex æquo, **y compris en tête**. Un tournoi qui veut un vainqueur unique
la fait suivre d'un tableau (``entry: "top:N"``), ou configure un départage (``tiebreak``).

La règle vit dans ``rules/suisse_rondes.go`` ; l'enveloppe est celle du suisse à vies en mode
``rounds`` (``phase_swiss.go``) : ronde ouverte reproposée à l'identique tant qu'elle n'est pas
entièrement lancée, orphelins réappariés entre eux, absences, retardataires (voir
:ref:`format_suisse`, « Mode par rondes »).

``Validate`` exige ``rounds`` > 0, pose ``mode`` à ``rounds`` et refuse sur ce format ``target``,
``length_late``, ``batch_minutes``, un autre ``mode``, un ``lives`` saisi et un ``pairing`` autre
que ``monrad``.
``rounds`` est refusé sur tout autre format.

Appariement d'une ronde
-----------------------

.. code:: go

   func PairSwissRound(p Phase, r int, players []Ref, seed Seed) (plan []Pairing, normal bool)

::

   ordre := joueurs triés par identifiant, mélangés (graine de la ronde), puis triés (stable)
            par points décroissants                     ← points = Wins + Byes
            puis par critères de départage, s'il y en a (voir « Départage »)
   si l'effectif est impair :
       candidats à l'exemption := ceux qui ont eu le MOINS d'exemptions, du dernier au premier
   pour limite de 0 à n/2 :                              ← nombre de revanches admis
       pour sansClub dans (vrai, faux) si avoid_clubs, sinon (faux) :
           pour chaque candidat c (ou aucun) :
               chercher un appariement de ordre − c avec au plus limite revanches
               trouvé → rendre les paires, et c exempté

Trois priorités, dans cet ordre : **le moins de revanches possible** ; l'exemption au **plus bas
classé parmi ceux qui en ont eu le moins** (jamais une seconde tant qu'un joueur n'en a pas eu) ;
chacun contre **le plus proche encore libre** au classement.

À égalité de points — et de départage, s'il y en a un —, l'ordre est tiré au sort : rien ne
distingue deux ex æquo. Le
tirage dépend de la graine du tournoi, de la phase et du numéro de ronde seulement
(``Seed.Round``) : une ronde ouverte se recalcule à l'identique.

La recherche (``chercher``) est en profondeur d'abord, avec retour arrière : le premier joueur
libre essaie ses adversaires du plus proche au plus lointain. Un appariement glouton ne suffit
pas — il apparie proprement la tête et accule les derniers à une revanche qu'un autre choix plus
haut évitait. Chaque tentative est bornée à 200 000 essais, après quoi la recherche passe au
candidat ou à la limite suivante : un grand champ ne bloque jamais.

``avoid_clubs`` est une **préférence** : à nombre de revanches égal, un appariement sans rencontre
de club est essayé d'abord ; il n'est jamais payé d'une revanche.

Un joueur seul à apparier (les autres absents ou retirés) est exempté.

Têtes de série (``seeding``)
----------------------------

Sans ``seeding``, la ronde 1 est tirée au sort. Avec ``seeding: "seed"`` (têtes de série saisies,
``Player.Seed``) ou ``"rating"`` (par cote), la **ronde 1 est pliée** : la moitié haute contre la
moitié basse, la tête 1 contre la tête *n*/2 + 1 ; en effectif impair, la dernière tête est
exemptée. Aux rondes suivantes, deux joueurs à égalité complète sont pris dans l'ordre des têtes
de série au lieu d'être tirés au sort. Le défaut reste le tirage : les têtes de série sont une
option nommée (voir :ref:`annexe_conception`).

Accélération (``accelerated``)
------------------------------

L'accélération donne un **point virtuel** à la moitié haute du champ aux rondes 1 et 2, pour
l'appariement seulement : il n'apparaît dans aucun classement. Elle exige ``seeding`` — sans ordre
de têtes de série il n'y a pas de moitié haute — et ``Validate`` la refuse avec ``pairing:
"monrad"``.

- **Moitié haute** (``rules.TopHalf``) : les ⌊*n*/2⌋ premières têtes de série des joueurs de la
  ronde 1 ; en effectif impair, la tête du milieu reste en bas. Elle est **figée à la ronde 1** :
  le moteur la relit des joueurs engagés à cette ronde, si bien qu'un retrait ou un retardataire ne
  fait franchir la ligne à personne.
- **Ronde 1** : chaque moitié est pliée séparément. Moitié haute impaire : sa dernière tête
  descend en tête de la moitié basse ; moitié basse impaire : sa dernière tête est exemptée.
- **Ronde 2** : l'ordre du classement est retrié (stable) par points + point virtuel, puis
  apparié comme toute ronde.
- **Rondes suivantes** : plus de point virtuel.

Monrad (``pairing: "monrad"``)
------------------------------

Le Monrad garde les rondes, les points et le classement du suisse classique ; seul change le choix
de l'adversaire, **tiré au sort dans le groupe de mêmes points** au lieu d'être le plus proche au
classement. Le départage, s'il y en a un, n'y joue aucun rôle.

::

   g := joueurs triés par identifiant, mélangés (graine de la ronde), triés (stable) par points
   pour chaque groupe de mêmes points, du meilleur au moins bon, sauf le dernier :
       groupe := report + groupe
       pair   : apparier sans aucune revanche ; impossible → report := groupe entier
       impair : retirer un joueur, du dernier tiré au premier, jusqu'à apparier le reste sans
                revanche → report := ce joueur (le flotteur) ; impossible → report := groupe entier
   dernier groupe (+ report) : appariement du suisse — le moins de revanches, l'exemption au
                plus bas parmi ceux qui en ont eu le moins

Un match oppose donc deux joueurs de mêmes points, sauf un flotteur par groupe impair.

Le dernier groupe peut donc jouer une revanche qu'un groupe plus large aurait évitée. Avec
``monrad_widen: true``, la ronde est alors refaite en lui adjoignant le groupe d'au-dessus, puis le
suivant, jusqu'à ne plus avoir de revanche ; tous les groupes réunis, c'est l'appariement du suisse.
L'option est éteinte par défaut.

Le tirage vient de ``Seed.Round`` : il se rejoue à l'identique, et deux graines donnent deux
tournois.

Fin de la phase
---------------

La phase est finie quand la ronde ``rounds`` est entièrement lancée et tous ses matchs terminés,
ou quand il reste au plus un joueur non retiré. ``RemainingMatches`` compte les matchs en cours
plus une ronde complète des joueurs en lice par ronde à venir.

Les joueurs ne portent pas de vies qui aient un sens : la mécanique partagée avec le suisse à
vies leur en donne un nombre qu'aucun nombre de rondes n'épuise. Vers la phase suivante,
``survivors`` admet tous les non-retirés, chacun avec les vies d'entrée de cette phase.

Départage (``tiebreak``)
------------------------

``tiebreak`` est une liste de critères, appliqués dans l'ordre **après les points**. Elle est
**vide par défaut**, et ce défaut est un choix de conception : le moteur ne départage pas (voir
:ref:`annexe_conception`). La remplir, c'est accepter de séparer deux joueurs au même parcours sur
leurs points de match et sur la force de leurs adversaires.

.. list-table::
   :header-rows: 1
   :widths: 24 76

   * - Critère
     - Valeur
   * - ``point_diff``
     - Points de match marqués moins points encaissés.
   * - ``points_scored``
     - Points de match marqués.
   * - ``buchholz``
     - Somme des points des adversaires rencontrés, une fois par rencontre.

Seuls les matchs **terminés** comptent. Les points de match sont **plafonnés à la longueur du
match** : 9-3 dans un match en 7 points compte 7-3. Un forfait compte longueur–0. Un résultat saisi
sans score ne compte que pour l'adversaire rencontré. Une exemption n'apporte ni points de match ni
adversaire.

Le départage n'ordonne que des ex æquo : il ne passe jamais un joueur devant un autre qui a plus
de points. Deux joueurs égaux sur les points et sur tous les critères **restent ex æquo** — il n'y
a pas de tri caché par identifiant. L'ordre d'appariement d'une ronde suit le même classement.

``Validate`` refuse un critère inconnu ou répété, et ``tiebreak`` sur tout autre format.

Classement
----------

::

   pour chaque entrant, dans l'ordre des entrants :
       clé  := (Wins + Byes, puis la valeur de chaque critère de tiebreak)
       note := {kind: record, wins, losses, byes,       ← kind: withdrawn pour un retiré
                point_diff, points_scored, buchholz}    ← les critères configurés seulement
   trier (stable) par clé décroissante ; même clé, même rang

Un retiré garde son parcours et son rang parmi les joueurs au même nombre de points.
