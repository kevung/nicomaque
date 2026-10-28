.. _glossaire:

Glossaire
=========

.. glossary::

   Action
      Proposition du moteur au directeur de tournoi (``Action``) : lancer un match, donner un
      bye, tirer un tableau, passer à la phase suivante, clore, attendre. Une action confirmée
      devient un événement.

   Barrage
      Mini-tournoi à 2 vies entre joueurs à égalité pour une place qualificative de poule.

   Bascule
      Arrêt d'un suisse à vies quand la somme des vies restantes atteint une puissance de 2
      (option ``target``), pour enchaîner sur un tableau à vies.

   Bye, exemption
      Place vide d'un tableau (``BYE``), ou ronde sautée par un joueur au suisse par rondes.

   Code
      Valeur structurée renvoyée par le moteur à la place d'un texte : libellé de match
      (``Label``), note de classement (``Note``), avertissement (``Warning``), raison d'attente
      (``ReasonCode``). L'hôte les traduit.

   Événement
      Fait consigné au journal (``Event``), avec un type (``kind``), un numéro d'ordre (``seq``),
      un horodatage et une version.

   Format
      Type d'une phase (``Kind``) : ``swiss_lives``, ``lives_bracket``, ``gsl``, ``bracket``,
      ``round_robin``.

   GMatch
      Case d'un graphe de matchs : deux places, une longueur, un résultat éventuel. Un
      ``GMatch`` n'est pas encore un match joué.

   Journal
      Liste ordonnée des événements d'un tournoi (``Journal``, tableau JSON). Seule donnée
      persistante ; l'hôte la stocke.

   Match
      Match réellement lancé : deux joueurs, une table, des horodatages. Relié à son ``GMatch``
      par le couple (section, clé).

   Phase
      Étape du tournoi, d'un format donné. Les phases s'enchaînent dans l'ordre de
      ``Config.Phases``.

   PR
      *Performance Rating* : mesure de la qualité de jeu au backgammon ; plus elle est basse,
      meilleur est le joueur (2 : très fort, 10 : débutant). Le moteur l'appelle ``rating``.

   Repêchage
      Remplacement d'un qualifié de poule retiré avant le tirage du tableau par le suivant de sa
      poule, proposé par le moteur et confirmé par le directeur (événement ``repechage``).

   Section
      Graphe de matchs nommé à l'intérieur d'une phase : tableau principal, consolante, groupe
      GSL, poule, barrage. Son nom est un identifiant, pas un libellé.

   TD
      Directeur de tournoi.

   Tirage
      Placement des joueurs dans un tableau ou composition des groupes. Le résultat est écrit
      dans l'événement ``draw``.

   Vie
      Droit à une défaite. Un joueur sans vie restante est éliminé de la phase.

   Walkover
      Victoire acquise sans jouer : contre une exemption, ou contre un joueur retiré.

Nomenclature des identifiants
-----------------------------

============================= ================================= ================================
Objet                         Forme                             Exemple
============================= ================================= ================================
Match réel                    ``M<n>``                          ``M17``
Section de tableau            ``main``, ``conso``, ``last``,    ``conso``
                              ``gf``
Section de groupe GSL         ``B<bloc>G<groupe>``              ``B2G3``
Section de poule              ``poule:<lettre>``                ``poule:C``
Section de barrage            ``barrage:<lettre>``              ``barrage:C``
``GMatch`` de tableau         ``<section>.<tour>.<indice>``     ``main.2.1``, ``conso.3.0``
``GMatch`` de grande finale   ``gf.<n>``                        ``gf.0``
``GMatch`` de groupe GSL      ``<section>.<n>``                 ``B2G3.4``
``GMatch`` de poule           ``<section>.<ronde>.<indice>``    ``poule:A.1.0``
============================= ================================= ================================
