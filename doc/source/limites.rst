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
- **Suisse classique** (``swiss_rounds``) : sans ``tiebreak``, les joueurs à égalité de points
  restent ex æquo, y compris en tête.
- **Consolante et dernière chance** : pas d'option de saut (perdants du premier tour seulement,
  entrée jusqu'au tour *k*, consolante à tirage avec heure limite d'entrée).
- **Revanches en consolante** : l'ordre inversé d'entrée évite la revanche immédiate, pas les
  suivantes ; leur fréquence n'est pas mesurée.
- **Poules en dernière phase** : sans ``final_barrage``, plusieurs poules donnent un premier par
  poule, et une poule unique ne désigne un vainqueur que si ``qualifiers`` vaut 1 ou si son
  premier est seul en tête. Un tableau d'un seul qualifié ne se tire pas.
- **Barrage tiré puis poule corrigée** : un barrage (de poule ou final) n'est pas recalculé. Si
  une correction de résultat de poule, passée après son tirage, change les joueurs que la poule
  désigne, le barrage se joue entre ceux du tirage et l'avertissement ``barrage_stale`` le
  signale ; au directeur d'annuler ses matchs s'il veut le refaire. Pour le barrage final,
  l'avertissement se trompe dans deux cas, tous deux après un retrait postérieur au tirage
  suivi de corrections. Il manque quand un joueur qui n'a pas été tiré se retire après le
  tirage, puis qu'une correction le ferait passer premier de sa poule : le moteur le prend
  pour un retiré d'avant le tirage. Il se lève à tort quand le tiré d'une poule et un autre
  joueur de cette poule se retirent après le tirage, puis que des corrections font passer
  premiers des joueurs retirés : le tirage reste juste.
- **Ronde de demi-exemptés seuls** : si, après des retraits, tous les joueurs restants d'une
  ronde de suisse classique sont demi-exemptés, rien ne s'apparie et le moteur attend
  (``no_pairing``) ; retirer une demi-exemption débloque la ronde.
- **Entrée** ``top:N`` **et ex æquo** : ``top:N`` coupe dans l'ordre du classement de la phase précédente,
  sans départager des ex æquo à la frontière ; sans ``tiebreak``, deux joueurs à égalité peuvent
  se trouver de part et d'autre de la coupe. ``top:1`` est admis : la phase n'a qu'un entrant,
  qui en est le vainqueur sans jouer.
- **Plus de rondes que d'adversaires** : un suisse classique dont ``rounds`` dépasse le nombre
  d'adversaires possibles (effectif moins un) reste jouable, par revanches, mais ``Propose``
  y ralentit nettement. ``Validate`` ne borne pas ``rounds`` : l'effectif n'est pas connu à la
  configuration.
- **Suisse classique au fil de l'eau** : après la ronde 1, le reste d'une ronde appariée en
  entier se recalcule à chaque proposition ; il peut changer quand un match est confirmé. Un
  joueur indisponible garde ses parties à jouer : la phase ne finit pas tant qu'il n'est ni
  revenu ni retiré. ``batch_minutes`` ne s'y applique pas. L'appariement glouton — tant qu'un
  joueur du même compte joue encore — n'évite ni club ni pays ; seule la ronde 1 et les restes
  de ronde appariés en entier les évitent.
- **Remplacement** : la place d'un joueur retiré ne se reprend pas (``player_replaced`` est
  refusé) ; le remplaçant s'inscrit alors comme un retardataire.
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
