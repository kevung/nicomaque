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
la fait suivre d'un tableau (``entry: "top:N"``).

La règle vit dans ``rules/suisse_rondes.go`` ; l'enveloppe est celle du suisse à vies en mode
``rounds`` (``phase_swiss.go``) : ronde ouverte reproposée à l'identique tant qu'elle n'est pas
entièrement lancée, orphelins réappariés entre eux, absences, retardataires (voir
:ref:`format_suisse`, « Mode par rondes »).

``Validate`` exige ``rounds`` > 0, pose ``mode`` à ``rounds`` et refuse sur ce format ``target``,
``length_late``, ``pairing``, ``batch_minutes``, un autre ``mode`` et un ``lives`` saisi.
``rounds`` est refusé sur tout autre format.

Appariement d'une ronde
-----------------------

.. code:: go

   func PairSwissRound(p Phase, r int, players []Ref, seed Seed) (plan []Pairing, normal bool)

::

   ordre := joueurs triés par identifiant, mélangés (graine de la ronde), puis triés (stable)
            par points décroissants                     ← points = Wins + Byes
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

À points égaux, l'ordre est tiré au sort : sans départage, rien ne distingue deux ex æquo. Le
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

Fin de la phase
---------------

La phase est finie quand la ronde ``rounds`` est entièrement lancée et tous ses matchs terminés,
ou quand il reste au plus un joueur non retiré. ``RemainingMatches`` compte les matchs en cours
plus une ronde complète des joueurs en lice par ronde à venir.

Les joueurs ne portent pas de vies qui aient un sens : la mécanique partagée avec le suisse à
vies leur en donne un nombre qu'aucun nombre de rondes n'épuise. Vers la phase suivante,
``survivors`` admet tous les non-retirés, chacun avec les vies d'entrée de cette phase.

Classement
----------

::

   pour chaque entrant, dans l'ordre des entrants :
       score := Wins + Byes
       note  := {kind: record, wins, losses, byes}      ← kind: withdrawn pour un retiré
   trier (stable) par score décroissant ; même score, même rang

Un retiré garde son parcours et son rang parmi les joueurs au même nombre de points.
