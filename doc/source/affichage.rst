.. _affichage:

Affichage (``render``)
======================

Fonctions **pures** produisant du HTML et du SVG. Rien n'y est écrit en dur dans une langue : le
moteur ne produit que des codes, et ce paquet les traduit par un ``Labeler`` que l'hôte fournit — y
compris ses propres mots (« Joueur », « Table », « libre »), qui sont eux aussi des codes
(``Term``). ``French()`` est fourni pour la console, la démo et les tests.

.. code:: go

   type Renderer struct {
       L      Labeler
       Style  string // vide = DefaultStyle
       Credit string // vide = DefaultCredit
       Lang   string // attribut lang de la page ; vide = "fr"
   }
   func New(l Labeler) *Renderer

   type Labeler interface {
       Label(tournoi.Label) string
       Note(tournoi.Note) string
       Reason(tournoi.ReasonCode) string
       Warn(tournoi.WarningCode) string
       SectionName(string) string
       PhaseName(tournoi.PhaseConfig) string
       Term(t Term, count int) string // count : formes au pluriel (« 3 joueurs »)
   }
   func French() Labeler

   type Term string // TermPlayer, TermTable, TermFree, TermBye, TermUnknown…
   const DefaultStyle  = `…` // feuille sobre, lisible de loin, correcte à l'impression (A4)
   const DefaultCredit = `Nicomaque — moteur de tournoi de Nicolas Harmand`

   func Batches(st *tournoi.State) [][]*tournoi.Match

Le contrat du ``Labeler`` : aucune méthode ne renvoie de code brut. ``French()`` délègue aux méthodes
``String()`` du paquet racine et à ``tournoi.SectionName``/``tournoi.PhaseName`` pour les codes du
moteur, et porte les mots du rendu ; c'est le seul endroit du paquet où une phrase est écrite.
L'hôte remplace ``Style`` pour retrouver sa charte : les classes CSS (``lent``, ``alerte``,
``libre``, ``occupee``, ``hs``, ``case``, ``credit``…) sont donc un contrat.

+-----------------------------+--------------------------------------------------------------------+
| Méthode                     | Produit                                                            |
+=============================+====================================================================+
| ``BracketBoardSVG(st, ph)`` | **Toutes** les sections en graphe d'une phase dans un seul SVG,    |
|                             | côte à côte, avec les descentes d'une section à l'autre            |
+-----------------------------+--------------------------------------------------------------------+
| ``LivesBoard(st, ph, now)`` | Colonnes de joueurs par nombre de défaites : état, temps           |
|                             | d'attente, adversaires déjà rencontrés                             |
+-----------------------------+--------------------------------------------------------------------+
| ``TableGrid(st, now)``      | Une case par table : son match et sa durée, ou libre,              |
|                             | indisponible, réservée                                             |
+-----------------------------+--------------------------------------------------------------------+
| ``RunningTable(st, now)``   | Les matchs en cours, triés par table : table, libellé, joueurs,    |
|                             | longueur, durée (classe ``lent`` au-delà de 1,5 × ``Expected``)    |
+-----------------------------+--------------------------------------------------------------------+
| ``ActionsList(st, acts)``   | La liste ordonnée des actions proposées ; une action qui porte un  |
|                             | avertissement reçoit la classe ``alerte`` et le texte du ``Warn``  |
+-----------------------------+--------------------------------------------------------------------+
| ``StandingsTable(st)``      | Le classement final (``Final``) ou courant (``Ranking``) avec      |
|                             | club, note et prix du classement général (``PrizeSectionAll``)     |
+-----------------------------+--------------------------------------------------------------------+
| ``PairingSheet(st, round)`` | La feuille d'appariements d'un lot, imprimable A4, une case de     |
|                             | score par joueur — page autonome                                   |
+-----------------------------+--------------------------------------------------------------------+
| ``Page(st, acts, now)``     | La page d'écran de salle, autonome, rafraîchie toutes les 30 s par |
|                             | une méta (pas de script)                                           |
+-----------------------------+--------------------------------------------------------------------+

``Page`` assemble, pour la phase courante : les actions à faire, la grille des tables (dès qu'une
table est connue), les matchs en cours, le tableau des vies (phases ``swiss_lives`` et ``gsl``
seulement) et le graphe des sections ; puis le classement. Dans ``ActionsList``, la table d'un
lancement n'est nommée que si elle est attribuée (``Table > 0``) ; une attente affiche sa raison
(``Reason``) et, le cas échéant, son échéance ``Until`` (« 15:04 »).

**Une page produite est autonome** : un seul fichier, feuille de style embarquée, aucune ressource
externe — pas de police distante, pas de script, pas d'image liée. Elle doit s'ouvrir hors ligne,
depuis une clé USB, sur l'ordinateur de la salle. Un crédit (``Credit``) figure en pied de page, et
l'attribut ``lang`` de la page vaut ``Lang``. ``PairingSheet`` et ``Page`` renvoient une page
complète ; les autres méthodes, un fragment HTML (ou SVG) à insérer dans celle de l'hôte.

**Les lots.** ``Batches(st)`` groupe les matchs de la phase courante par instant de départ. Un lot
est ce qu'un directeur appelle une ronde dans un suisse par rondes, un bloc dans un GSL, un tour
dans un tableau, une micro-ronde dans un suisse à ``batch_minutes``. Le moteur n'a pas de mot pour
cela et n'en a pas besoin : l'instant de départ le dit. ``PairingSheet(st, n)`` imprime le n-ième ;
``0`` ou un numéro au-delà du dernier donne le plus récent. Le découpage en lots et l'ordre des
lignes viennent de ``rules.Batches`` et ``rules.PairingSheet`` (``render`` n'en est que le rendu,
voir :ref:`api_reference`).

**Mise en page du SVG.** Une colonne par tour (``Section.Rounds``), boîte de 190 × 44 px, écart horizontal
60 px, vertical 12 px, et 40 px entre deux sections. Chaque section occupe sa propre bande de
colonnes, si bien que deux arbres ne peuvent pas se chevaucher. L'ordonnée d'un match est la
**moyenne des ordonnées de ses matchs sources de la même section** ; les chevauchements sont ensuite
corrigés en décalant vers le bas (nécessaire pour les groupes GSL, où le match des gagnants et celui
des perdants partagent des sources). Les liaisons sont tracées **sous** les boîtes ; celles qui
traversent une section — les descentes du principal vers la consolante — sont en pointillé et d'une
autre couleur. Une paire d'exemptions (``BYE`` contre ``BYE``) n'est pas dessinée. Le fond d'une
boîte est blanc (match à venir), jaune pâle (lancé) ou vert pâle (terminé) ; le vainqueur est en
gras, le score affiché une fois le match terminé, et le libellé et la longueur du match figurent
au-dessus de la boîte. Le titre de chaque section passe par ``Labeler.SectionName`` (les noms de
section sont des identifiants : ``main``, ``conso``, ``poule:A``, ``barrage:A``…).

Le nom affiché d'un joueur est ``Player.Name``, ou l'identifiant à défaut — un nom est saisi par le
TD et ne se traduit pas. ``BYE`` et une place inconnue passent, eux, par le ``Labeler``. Tout texte
issu des données passe par ``html.EscapeString``.

Les rendus sont figés par des **fichiers témoins** dans ``render/testdata/`` : toute modification du
rendu fait échouer les tests, et ``go test ./render -update`` les régénère une fois le changement
relu.
