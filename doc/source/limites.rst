.. _limites:

Limites connues
===============

Ce que le moteur ne fait pas, ou fait avec une restriction, dans la version publiée. La liste de
travail détaillée est dans ``docs/RESTE_A_FAIRE.md`` du dépôt.

Validation
----------

- Le moteur n'a pas encore dirigé de tournoi réel. La simulation couvre le déroulement
  mathématique d'un tournoi, pas les situations de salle. Un premier usage est recommandé sur un
  tournoi de club de 16 à 32 joueurs, en doublant sur papier.
- L'interface du directeur et la persistance du journal sont de la responsabilité du logiciel
  hôte : la reprise après panne suppose que l'hôte enregistre chaque événement au fil de l'eau.

Règles
------

- **Byes entre groupes** : en mode par rondes, la règle « pas de second bye tant que d'autres
  n'en ont pas eu » s'applique à l'intérieur d'un groupe de même nombre de défaites, pas entre
  groupes.
- **Consolante et dernière chance** : pas d'option de saut (perdants du premier tour seulement,
  entrée jusqu'au tour *k*, consolante à tirage avec heure limite d'entrée).
- **Revanches en consolante** : l'ordre inversé d'entrée évite la revanche immédiate, pas les
  suivantes ; leur fréquence n'est pas mesurée.
- **Poules** : l'égalité à trois pour deux places, réglée par barrage, n'a pas été éprouvée en
  conditions réelles (durée du barrage).
- **Classement des places non gagnantes** : les règles (par tour atteint en tableau, par
  victoires au suisse, ex æquo partagés) restent à valider avec la FFBG.

Affichage
---------

- Pas d'écran joueur (« votre prochain match, votre table ») dans le paquet ``render``.

Modèle de l'étude
-----------------

- L'effet de la longueur de match (exposant α) n'est pas fixé par les données disponibles.
- Le modèle de durée est une moyenne par point, indépendante du niveau et de la longueur.
- Le simulateur de l'étude et ``sim`` diffèrent de 1 à 2 écarts types sur les probabilités de la
  double élimination et du GSL ; la cause n'est pas identifiée.
