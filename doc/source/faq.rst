.. _faq:

Questions fréquentes
====================

Questions de joueurs et de directeurs de tournoi. Les justifications détaillées sont dans
:ref:`annexe_conception` ; les chiffres dans :ref:`annexe_etude`.

Classement
----------

Un joueur est parti en cours de tournoi — pourquoi n'est-il pas dernier ?
   Son retrait n'efface pas ce qu'il a joué. Il est classé comme éliminé à l'instant du
   retrait : au suisse, par ses victoires ; dans un tableau, comme perdant du match qui
   l'attendait ; dans une poule, par ses victoires, sans être qualifié. Il est ex æquo avec les
   éliminés du même niveau, avec la note « retiré ». Le mot « forfait » désigne seulement le
   match qu'il n'a pas joué.

Pourquoi deux joueurs ex æquo ne sont-ils pas départagés ?
   Ils ont le même parcours. Les départager demanderait d'utiliser la force de leurs adversaires,
   qu'ils n'ont pas choisis, ou la marge de leurs victoires, qui au backgammon dépend fortement du
   videau et du hasard. Si une qualification est en jeu, la place se joue en barrage ; sinon, les
   prix des places concernées sont partagés à parts égales.

Le classement dit que j'ai été sorti du tableau — suis-je éliminé ?
   Pas forcément. Le classement situe chaque joueur (« principal, premier tour » dit où il a
   perdu), mais un joueur battu au premier tour du principal peut jouer la consolante. Le statut
   (voir :ref:`statut`) répond à « est-ce que je joue ? » : en jeu, exempté, qualifié, pas encore
   fixé, éliminé, vainqueur ou retiré.

Formats
-------

Pourquoi le meilleur joueur ne gagne-t-il que 4 à 9 % du temps ?
   Sur 64 joueurs, les formats mesurés donnent 3,5 à 5,7 % ; un vainqueur tiré au sort donnerait
   1,6 %. Le format multiplie cette probabilité par 2 à 3,5. Aucun format raisonnable ne fait
   beaucoup mieux à durée égale : il faudrait des matchs beaucoup plus longs et plusieurs jours
   de jeu.

Pourquoi un joueur qui a perdu peut-il gagner le tournoi ?
   C'est le principe d'un format à deux vies. À 7 points, un joueur meilleur de 3 PR gagne
   environ 58 % du temps : perdre un match est courant, et éliminer dès la première défaite
   augmente la part du hasard dans le résultat.

Pourquoi 2 vies plutôt que 3 ?
   À budget de temps constant, 3 vies n'est jamais le meilleur choix : la troisième vie coûte plus
   de temps qu'elle n'apporte de justesse. Le moteur accepte un nombre de vies quelconque.

Le jeu en continu n'avantage-t-il pas ceux qui jouent vite ?
   Le continu réduit l'attente de 174 à 12 minutes par joueur sur un tournoi de 16 heures. Il
   permet en revanche à un joueur de retarder l'annonce de son résultat pour influencer son
   appariement. Les parades sont les micro-rondes (``batch_minutes``), les blocs GSL et le mode
   par rondes ; sur 64 joueurs, GSL et rondes demandent environ 14 h de salle contre 10 h 20 pour
   le continu, pour le même nombre de matchs.

Pourquoi le tableau final a-t-il 32 places alors que nous sommes 20 ?
   Sa taille vient de la somme des vies restantes, pas du nombre de joueurs. Un joueur encore
   invaincu occupe deux places : il est exempt du premier tour.

Pourquoi les meilleurs joueurs ne sont-ils pas protégés dans le tableau ?
   Le tirage aléatoire est le défaut, conformément à la pratique du backgammon. Les têtes de série
   existent en option (``seeding: "rating"``) pour l'organisateur qui les veut.

Pourquoi ai-je un bye alors qu'un autre joueur en a déjà eu un ?
   Un joueur déjà exempté est apparié en priorité, ce qui évite un second bye tant que d'autres
   n'en ont pas eu. Cette règle s'applique à l'intérieur d'un groupe de même nombre de défaites ;
   l'équité entre groupes différents n'est pas garantie (voir :ref:`limites`).

J'ai perdu contre X au premier tour et je le retrouve en consolante. Est-ce normal ?
   Les perdants du principal entrent dans la consolante dans l'ordre inverse, ce qui évite la
   revanche immédiate, pas les suivantes. La fréquence de ces revanches n'est pas encore mesurée.

Déroulement
-----------

Un joueur doit s'absenter — faut-il le retirer ?
   Non : on le déclare indisponible jusqu'à une heure, jusqu'à une ronde ou jusqu'à nouvel
   ordre. Il n'est plus apparié pendant ce temps et ne perd ni vie, ni rang, ni place de tableau.
   Dans un tableau, son match reste affiché mais retenu. Un joueur qui dispute un match d'une
   autre épreuve de la salle est signalé par l'hôte (``External.BusyPlayers``) ; rien n'est écrit
   au journal.

Peut-on inscrire un joueur en retard ?
   Au suisse, oui, tant que la phase n'est pas figée : il entre avec toutes ses vies. Dans un
   tableau ou des poules déjà tirés, le moteur liste les places d'exemption libres
   (``State.FreeSlots``) et y installe le joueur sans refaire le tirage. Sans place libre, le
   joueur est inscrit et le moteur indique la phase ou la section où il entrera.

Un qualifié de ma poule part avant le tableau — qui prend sa place ?
   Le suivant de sa poule, si le directeur le confirme : le moteur propose le repêchage avant le
   passage au tableau, ou avant le tirage si le passage est fait. Entre suivants ex æquo, il
   propose chacun. Si le directeur passe directement au tableau, la place devient une exemption.
   Après le tirage, le retiré perd son premier match par forfait.

Le directeur s'est trompé de vainqueur. Faut-il tout refaire ?
   Non. Il saisit une correction, ajoutée au journal ; l'erreur reste visible. Le moteur recalcule
   tout, signale ce qui n'est plus cohérent (un match de tableau joué par le mauvais joueur) et
   propose d'annuler en série les matchs concernés. Le directeur confirme ou fait autrement.

Et si l'ordinateur s'éteint en plein tournoi ?
   Le journal relu redonne l'état exact, matchs en cours et tirages compris ; les tests le
   vérifient sur toute la matrice de simulation. Cela suppose que l'hôte enregistre chaque
   événement au moment où il est appliqué.

Combien de temps dure un tournoi ?
   Un match dure en moyenne 8 minutes par point ; on compte 11 minutes par point pour planifier une
   ronde. Un format à 2 vies produit environ 2 matchs par joueur (126 pour 64 joueurs) ;
   l'élimination simple en produit un de moins que le nombre de joueurs (63 pour 64). Sur 64
   joueurs en 7 points : environ 7 h pour l'élimination simple, 10 h pour le suisse continu, 14 h
   pour le suisse par rondes. En cours de tournoi, le moteur fournit une prévision de fin
   (médiane et délai à 90 %) par simulation des matchs restants.

Intégration
-----------

Pourquoi une bibliothèque plutôt qu'un logiciel complet ?
   La logique du tournoi se vérifie ainsi indépendamment de tout affichage — les tests jouent des
   milliers de tournois sans écran — et elle est réutilisable par n'importe quel logiciel hôte.
   blunderDB et gammonGo l'utilisent.

Le moteur affiche-t-il du texte dans une langue donnée ?
   Non. Il renvoie des codes ; l'hôte les traduit. Un rendu français est fourni pour la console,
   les démonstrations et le paquet ``render``.
