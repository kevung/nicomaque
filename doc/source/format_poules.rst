.. _format_poules:

Poules (``round_robin``)
========================

Principe
--------

Les entrants sont répartis en poules ; chaque poule est une toutes rondes (table de Berger). Les
``qualifiers`` premiers de chaque poule passent à la phase suivante. Les égalités à la place
qualificative ne sont **pas** départagées arithmétiquement : elles déclenchent un **barrage** à 2
vies entre les joueurs concernés.

Les défauts sont posés par ``Config.Validate`` : ``group_size`` vaut 4, ``qualifiers`` vaut 2, et
``qualifiers`` doit être strictement inférieur à ``group_size``. La règle est dans ``rules``
(``rules/graphes_phase.go`` : ``Pair``, ``PoolSections``, ``PoolWins``, ``PoolTie``,
``BarrageSection``, ``PoolQualified`` ; ``rules/classement.go`` : ``Rank``) ; ``phase_rr.go`` en
est l'enveloppe.

Tirage des poules
-----------------

::

   si non Drawn :
       n := len(Entrants) ; si n < 2 → nil
       rng := générateur(seed.Draw())
       ids := tri par identifiant puis rng.Shuffle(ids)
       ng := ceil(n / GroupSize)   (au minimum 1)
       répartir en « serpentin simple » : groups[i mod ng] += ids[i]
       → une action draw, Label = draw_pools{N: ng}, Draw = { Groups }   (sans Lives)

La distribution ``i mod ng`` équilibre les tailles à un joueur près.

.. code:: go

   func (s *State) applyRRDraw(ph *PhaseState, section string, d *Draw) error

Avec ``section`` vide (tirage des poules) :

::

   ph.Sections += PoolSections(d.Groups, ph.Length) :
       pour chaque groupe i :
           sec := PoolSection(PoolSectionName(i), g, length) ; sec.Group = i+1
   ph.Drawn = true ; ph.resolve(s.Withdrawn)

Les poules sont des sections de genre ``poule`` dont le nom est un **identifiant** : ``poule:A``,
``poule:B``, … (``rules.PoolSectionName(i)`` = ``"poule:" + ('A'+i)``). Les clés de leurs nœuds
sont ``poule:A.<ronde>.<i>`` (``poule:A.0.0``, …) et leurs libellés ``pool_round{N: ronde}``
(table de Berger). Au-delà de 26 poules, le caractère produit sort de l'alphabet : limite connue,
sans conséquence fonctionnelle.

Décompte et détection des égalités
----------------------------------

.. code:: go

   func PoolWins(sec Section) map[Ref]int   // paquet rules ; rrWins en est l'enveloppe

Compte les victoires **dans la section** : chaque joueur figurant en ``Src[k].Player`` est
initialisé à 0 (pour qu'un joueur sans victoire apparaisse), puis chaque match ``Done`` et non
``Walkover`` ajoute une victoire à son vainqueur. Les walkovers (exemptions, forfaits) ne comptent
donc pas.

.. code:: go

   func PoolTie(sec Section, qualifiers int) (tied []Ref, spots int)   // paquet rules

::

   w   := PoolWins(sec)
   ids := tri par identifiant (clés de w) puis tri stable décroissant sur w
   q   := qualifiers
   si q >= len(ids)                → pas d'égalité (toute la poule est qualifiée)
   si w[ids[q-1]] != w[ids[q]]     → pas d'égalité (la coupure est nette)

   v := w[ids[q-1]]                ← le nombre de victoires litigieux
   spots := q
   pour chaque p :
       si w[p] > v  → spots--      ← qualifié d'office, il consomme une place
       si w[p] == v → tied += p

Résultat : ``tied`` sont les joueurs à égalité sur la ligne de coupure, ``spots`` le nombre de
places qu'ils doivent se partager.

Barrage
-------

Un barrage est une ``Section`` d'un genre particulier : elle n'a **pas de graphe** (``Matches``
vide) ; ses matchs sont appariés dynamiquement, comme un suisse à 2 vies, jusqu'à ce qu'il ne reste
que ``Spots`` joueurs.

.. code:: go

   func (s *State) applyRRDraw(...)   // avec section = "barrage:<lettre>"
   func BarrageSection(pool Section, tied []Ref, qualifiers int) Section   // paquet rules

::

   si len(d.Groups) != 1 → erreur « un seul groupe attendu »
   poule := ph.section(PoolOfBarrage(section))     ← "barrage:A" → "poule:A"
   si absente → erreur « poule inconnue »
   _, spots := PoolTie(poule, Qualifiers)          ← les places sont RECALCULÉES, pas relues du tirage
   ph.Sections += Section{Name: section, Kind: "barrage", Players: d.Groups[0], Spots: spots}

Le barrage de ``poule:A`` s'appelle ``barrage:A`` (``rules.BarrageSectionName``). L'instantané
de la phase (``rules.Phase.Barrages``) porte les matchs lancés et non annulés des sections de
barrage, dans l'ordre de ``MatchOrder``.

.. code:: go

   func (v *vue) barrageLosses(name string) (losses map[Ref]int, running int)   // paquet rules

Pour les matchs de barrage de section ``name`` non annulés : compte les ``Running`` et incrémente
``losses[perdant]`` pour les autres.

.. code:: go

   func (v *vue) barrageAlive(bs *Section) []Ref  // moins de 2 défaites, dans l'ordre de bs.Players

.. code:: go

   func (v *vue) pairBarrage(bs *Section, busy, ailleurs map[Ref]bool, seed Seed) []Pairing

::

   losses, running := barrageLosses(bs.Name)
   alive  := barrageAlive(bs)
   budget := len(alive) - running - bs.Spots
   si budget <= 0 → nil                     ← le barrage a rendu son verdict

   rng  := générateur(seed.Draw())
   free := alive privés des joueurs occupés (Busy) et de ceux qui jouent ailleurs (Elsewhere)
   pour l de 0 à 1, tant que budget > 0 :
       g := free ayant exactement l défaites de barrage
       g := tri par identifiant puis rng.Shuffle(g)
       apparier g[0]-g[1], g[2]-g[3], … tant que budget > 0
           action start_match { Section: bs.Name, Label: barrage{Section: bs.Name},
                                Length: longueur de la phase }

   si aucune action, running == 0 et au moins 2 joueurs libres :
       fr := free trié par identifiant puis (stable) par défaites croissantes
       → une action croisée fr[0] contre fr[1], Label = barrage_crossed{Section: bs.Name}

Même logique de budget que le suisse : on ne lance pas un match qui ferait descendre le nombre de
survivants en dessous du nombre de places.

.. code:: go

   func (v *vue) barragesDone() bool   // paquet rules

Pour chaque poule ayant une égalité : le barrage doit exister **et** ``len(barrageAlive) <= Spots``.
La phase est finie (``rules.PhaseDone``) quand elle est tirée, que tous les nœuds de ses sections
sont terminés et que ``barragesDone`` est vrai.

Proposition
-----------

.. code:: go

   func (s *State) proposeRR(ph *PhaseState) []Action   // enveloppe de rules.Pair (pairPools)

::

   si non Drawn → l'action de tirage des poules (section « Tirage des poules »)

   acts := pour chaque match de readyFree(Sections, Busy) :
               action start_match { Section, Key, A, B, Length,
                                    Label = in_section{Section}, Sub = libellé du nœud }

   pour chaque section de genre "poule" dont TOUS les matchs sont faits :
       tied, spots := PoolTie(sec, Qualifiers)
       si pas d'égalité → passer
       si la section barrage:<lettre> n'existe pas encore :
           acts += action draw
               Section = "barrage:<lettre>"
               Label   = draw_barrage{Section: "poule:<lettre>", Players: len(tied), Spots: spots}
               Draw    = { Groups: [tied] }
       sinon :
           acts += pairBarrage(barrage)

Les poules qui ont fini avancent donc vers leur barrage pendant que les autres jouent encore.

Qualifiés et classement
-----------------------

.. code:: go

   func PoolQualified(p Phase) []Ref   // paquet rules ; rrQualified en est l'enveloppe
                                       // = repecher(qualifiedBase())

::

   qualifiedBase — pour chaque poule :
       w   := PoolWins ; ids := tri par identifiant puis tri stable décroissant sur w
       tied, spots := PoolTie
       si pas d'égalité :
           → les `Qualifiers` premiers de ids
       sinon :
           v := w[tied[0]]
           → tous ceux qui ont strictement plus de v victoires
           + si un barrage existe et len(barrageAlive) <= spots → les survivants du barrage

   repecher — pour chaque qualifié p de la base :
       tant que p est retiré et ph.Repechages[p] existe → p := ph.Repechages[p]   (en chaîne)
       garder p (une seule fois)

Un repêchage ne tient qu'autant que le retrait : un qualifié réinscrit avant le tirage retrouve sa
place, et son repêché la perd.

Repêchage d'un qualifié retiré
------------------------------------

**Règle.** Un qualifié de poule qui se retire **avant le tirage** de la phase suivante laisse sa
place au **suivant de sa poule**, si le TD le confirme. Le moteur propose (``ActRepechage``,
confirmée en événement ``repechage``), il n'impose pas : passer à la phase suivante, ou tirer, sans
confirmer laisse la place vide, et elle devient une exemption du tableau. La règle vit dans
``repechage.go``.

::

   proposeRepechages(ph) :
       vide si ph n'est pas une phase de poules, si elle n'a pas de phase suivante, si celle-ci
       prend tous les joueurs (« all »), si les poules ou les barrages ne sont pas finis, ou si la
       phase suivante est déjà tirée ou commencée
       qual := rrQualified(ph)
       pour chaque poule, dans l'ordre :
           gone := le premier joueur de la poule (ordre des identifiants) qualifié et retiré ;
               aucun → poule suivante
           cands := joueurs de la poule ni dans qual ni retirés, au plus grand nombre de victoires
           pour chaque c de cands (ordre des identifiants) :
               Action{repechage, Phase, Section = poule, A = gone, B = c,
                      Label{repechage, Section = poule, Players = len(cands)}}

Cas limites :

+------------------------------------------------+------------------------------------------------+
| Cas                                            | Réponse du moteur                              |
+================================================+================================================+
| Le suivant est lui-même retiré                 | il est sauté : le candidat est le suivant      |
|                                                | **non retiré**                                 |
+------------------------------------------------+------------------------------------------------+
| Plus personne dans la poule (tous qualifiés ou | aucune proposition ; la place devient une      |
| retirés)                                       | exemption                                      |
+------------------------------------------------+------------------------------------------------+
| Le repêché se retire à son tour avant le       | il est un qualifié retiré comme un autre :     |
| tirage                                         | nouveau repêchage proposé                      |
+------------------------------------------------+------------------------------------------------+
| Deux qualifiés de la même poule se retirent    | un repêchage à la fois : le second est proposé |
|                                                | une fois le premier confirmé                   |
+------------------------------------------------+------------------------------------------------+
| Suivants ex æquo (même nombre de victoires)    | une proposition par candidat,                  |
|                                                | ``Label.Players`` = leur nombre ; le TD        |
|                                                | choisit (aucun départage)                      |
+------------------------------------------------+------------------------------------------------+
| Retrait après le passage de phase, avant le    | le repêchage reste proposé, devant le tirage ; |
| tirage                                         | le repêché remplace le retiré parmi les        |
|                                                | entrants                                       |
+------------------------------------------------+------------------------------------------------+
| Retrait après le tirage (ou après un premier   | pas de repêchage : le retiré perd son match    |
| match de la phase suivante)                    | par forfait (walkover), comme tout retrait de  |
|                                                | tableau ; ``repechage`` est refusé             |
+------------------------------------------------+------------------------------------------------+
| Phase suivante en entrée ``all``               | sans objet (tout le monde entre déjà) ; ni     |
|                                                | proposé ni accepté                             |
+------------------------------------------------+------------------------------------------------+

Le repêchage est proposé avec l'action de passage de phase (avant elle), puis, une fois le passage
confirmé et tant que la phase suivante n'est ni tirée ni commencée, devant ses propositions. À
l'application, le moteur vérifie à nouveau ces conditions, que ``A`` est un qualifié retiré de la
poule et ``B`` un joueur de la poule ni qualifié ni retiré ; si la phase suivante est déjà ouverte,
``B`` remplace ``A`` parmi ses entrants.

Le repêché est **qualifié** au classement de la poule (``Note.Qualified``, bonus de 100) ; le retiré
garde la note ``withdrawn``, jamais qualifié.

.. code:: go

   func Rank(p Phase) []Standing   // paquet rules ; rankPools pour round_robin

::

   qual := ensemble des PoolQualified
   pour chaque joueur d'une poule :
       score := victoires dans sa poule (PoolWins)
       note  := pool_record{Section: "poule:X", Wins: n}
       si retiré   → note := withdrawn{Section: "poule:X", Wins: n} ; jamais qualifié
       sinon, si qualifié → score += 100 ; note.Qualified := vrai

   ids := Entrants triés par identifiant, puis stablement par score décroissant
   rangs avec ex æquo (même score → même rang)

Le bonus de 100 place tous les qualifiés devant tous les non-qualifiés, quelles que soient les
poules — ce qui est correct puisque les poules ne sont pas comparables entre elles. Un retiré n'en
bénéficie pas : il est classé sur ses victoires, ex æquo avec les non-qualifiés qui en ont autant.
