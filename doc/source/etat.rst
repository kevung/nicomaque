.. _etat:

Reconstruction de l'état
========================

``Replay`` et ``New``
---------------------

.. code:: go

   func Replay(j Journal) (*State, error)
   func New(cfg Config, seed int64, now time.Time) (*State, Event, error)

``Replay`` part d'un état neuf :

::

   State{Players: {}, Withdrawn: {}, WithdrawAfter: {}, Matches: {}, Current: -1}

puis applique les événements dans l'ordre, chacun passé par ``Event.upgraded()`` : un événement de
version 0 (libellé en texte) est converti en codes avant d'être appliqué, si bien que le moteur ne
voit jamais de libellé texte. À la première erreur, ``Replay`` renvoie l'état partiel **et** une
erreur enrichie : ``événement <i> (<kind>) : <erreur>``.

``New`` valide la configuration, construit l'événement ``created`` (``Version: JournalVersion``,
soit 2) et l'applique ; elle renvoie l'état, l'événement (que l'hôte doit ajouter en tête de
journal) et l'erreur éventuelle.

Après chaque événement appliqué, ``Apply`` rafraîchit ``State.Infos`` (``refreshInfos``),
incrémente ``NEvents`` et avance ``Last`` si l'horodatage de l'événement est plus récent ; les
branches qui sortent tôt (retrait différé) passent par le même chemin (``seen``), pour ne pas
fausser le générateur aléatoire, qui dépend de ``NEvents``.

Entrée dans une phase
---------------------

.. code:: go

   func (s *State) enter(ph *PhaseState, p PlayerID, lives int)

Ajoute ``p`` à ``ph.Entrants`` (sans doublon) et fixe ``ph.Lives[p] = lives``. Si ``p`` est déjà
entrant, **rien n'est modifié** — y compris ses vies.

Un inscrit entre dans la première phase à son ``player_added`` si la phase 0 n'est pas tirée et
qu'elle est un suisse, ou qu'aucun match n'y est lancé ; il reçoit alors ``livesFor(cfg)``. Un
retardataire placé sur une place d'exemption d'un tableau déjà tiré (``player_added`` avec
``slot``) prend la place ``BYE`` de la ``Src``, entre avec **une** vie, et ``recompute`` est
appelé.

.. code:: go

   func livesFor(cfg PhaseConfig) int

=============== ===============
Kind            Vies à l'entrée
=============== ===============
``swiss_lives`` ``cfg.Lives``
``gsl``         2
tout autre      1
=============== ===============

Passage d'une phase à la suivante
---------------------------------

.. code:: go

   func (s *State) enterFrom(next, prev *PhaseState)

::

   selon next.Cfg.Entry :

     "all" :
         pour chaque p de s.Order dans l'ordre d'inscription, non retiré :
             enter(next, p, livesFor(next.Cfg))

     "top:N" :
         r := phaseRanking(prev)
         pour chaque rang rk de r (dans l'ordre du classement) :
             si rk.Rank <= N et p non retiré → enter(next, p, livesFor(next.Cfg))

     "survivors" (défaut) :
         pour chaque p de survivors(prev), non retiré :
             enter(next, p, entryLives(next, prev, p))

   entryLives(next, prev, p) :
         Entry ∉ {"survivors", ""}    → livesFor(next.Cfg)
         next.Kind ∈ {lives_bracket, gsl, swiss_lives} → remainingLives(prev, p)
         sinon → livesFor(next.Cfg)

Le cas ``survivors`` est la branche par défaut : toute valeur d'``Entry`` autre que ``all`` et
``top:N`` y tombe.

Après des poules, ``survivors(prev)`` = ``rrQualified(prev)`` (``rules.PoolQualified``),
**repêchés compris** : un qualifié retiré n'entre pas, et son repêché (s'il y en a un) entre à sa
place.

C'est ici que se joue la **bascule à vies** : un joueur qui sort d'un suisse à 2 vies avec ses deux
vies intactes entre dans un ``lives_bracket`` avec 2 vies, ce qui lui vaudra une **exemption du
premier tour**. Un joueur qui n'a plus qu'une vie entre au premier tour.

``top:N`` est lu par ``Sscanf("%d")`` ; un suffixe illisible donne ``N = 0``, donc une phase vide.
Le préfixe est reconnu par ``len(Entry) > 4 && Entry[:4] == "top:"``.

Vies restantes, joueurs en vie, survivants
------------------------------------------

.. code:: go

   func (s *State) remainingLives(ph *PhaseState, p PlayerID) int

Enveloppe de ``rules.RemainingLives(lives, losses, withdrawn)`` :

::

   si p est retiré → 0
   l := ph.Lives[p] - ph.Losses[p] ; si l < 0 → 0

.. code:: go

   func (s *State) alive(ph *PhaseState) []PlayerID

Les entrants (dans l'ordre de ``Entrants``) dont ``remainingLives > 0``.

.. code:: go

   func (s *State) survivors(ph *PhaseState) []PlayerID

+-----------------------------------+--------------------------------------------------------------+
| Kind de la phase quittée          | Survivants                                                   |
+===================================+==============================================================+
| ``round_robin``                   | ``rrQualified(ph)`` — les qualifiés des poules, barrages     |
|                                   | inclus                                                       |
+-----------------------------------+--------------------------------------------------------------+
| ``bracket``, ``lives_bracket``    | ``bracketSurvivors(ph)`` — les entrants si le tableau n'est  |
|                                   | pas tiré ; le premier du classement si toutes les sections   |
|                                   | sont finies ; sinon les joueurs de note ``running``          |
+-----------------------------------+--------------------------------------------------------------+
| autres (``swiss_lives``, ``gsl``) | ``alive(ph)``                                                |
+-----------------------------------+--------------------------------------------------------------+

Comptabilité d'un résultat
--------------------------

.. code:: go

   func (s *State) onResult(m *Match)

::

   ph := phaseOf(m.Phase) ; si absente → ne rien faire
   loser := m.Loser()
   ph.Wins[m.Winner]++
   ph.Losses[loser]++
   si remainingLives(ph, loser) == 0 et ph.Lives[loser] > 0 :
       ph.ElimOrder += loser
   si un GMatch (m.Section, m.Key) existe :
       g.Done, g.Winner, g.Loser = true, m.Winner, loser
       ph.resolve(s.Withdrawn)

La garde ``ph.Lives[loser] > 0`` évite d'enregistrer comme « éliminé » un joueur qui n'était pas
entrant de la phase (cas pathologique après correction).

``ElimOrder`` sert au classement des phases à vies : le **dernier éliminé** quand il ne reste qu'un
joueur en vie est le finaliste.

Recalcul complet
----------------

.. code:: go

   func (s *State) recompute()

Déclenché par ``player_withdrawn`` (immédiat), ``result_corrected``, ``match_cancelled``,
``config_changed``, par ``player_added`` sur une place d'exemption (``slot``), et par ``result``
quand un retrait différé prend effet (l'un des deux joueurs du match vient de passer dans
``Withdrawn``).

::

   1. Pour chaque phase :
          Losses, Wins, Opponents ← vides
          ElimOrder ← nil
          pour chaque GMatch de chaque section :
              Done, Winner, Loser, Walkover, Skipped ← false, "", "", false, false
              MatchID ← ""
              Players[k] ← Src[k].Player  (donc "" pour une place dérivée d'un autre match)

   2. Pour chaque match dans l'ordre de MatchOrder :
          ph := phaseOf(m.Phase) ; si absente ou m.Status == Cancelled → passer
          ph.Opponents[m.A] += m.B ; ph.Opponents[m.B] += m.A
          si un GMatch (m.Section, m.Key) existe → son MatchID = m.ID
      Puis, dans le même ordre : si m.Status == Finished → onResult(m)

   3. Pour chaque phase : resolve(s.Withdrawn)

   4. s.Warnings = check()

Ce qui **n'est pas** réinitialisé : ``Byes`` et ``ByeRounds`` (un bye reste acquis),
``Entrants``, ``Lives``, ``Round``, ``Roster``, ``Drawn``, ``Repechages`` et ``Sections``
(structure).

``GMatch.MatchID`` repart vide à l'étape 1 et n'est reposé à l'étape 2 que pour les matchs non
annulés : une place dont le match vient d'être annulé redevient lançable, au lieu de rester bloquée
sur l'identifiant d'un match qui n'existe plus.

L'étape 2 fait **deux passes** : relier toutes les places à leur match, puis rejouer les résultats.
En une seule passe, ``resolve`` — appelée par ``onResult`` — verrait une place encore sans match
dont un joueur est retiré, et la déclarerait gagnée par forfait (``Walkover``) alors que le match a
été joué avant le retrait ; le classement ignorerait ce match, et une demi-finaliste retirée
finirait dernière.

Les **places dérivées** (``GMatch.Players[k]`` dont la source est un autre match) repartent vides à
l'étape 1 et sont refaites par ``resolve`` : un match qui cesse d'être joué — correction,
annulation, ou place d'exemption prise par un retardataire — ne laisse pas derrière lui le joueur
qu'il avait fait avancer. Les places **fixes** (``Src[k].Player``,
un joueur ou un ``BYE``) sont restaurées telles quelles.

Contrôle de cohérence
---------------------

.. code:: go

   func (s *State) check() []Warning

Les avertissements sont des ``Warning`` : un code et ses paramètres, jamais une phrase (le rendu
français est ``Warning.String()``, ``fr.go``).

Pour chaque match non annulé rattaché à un ``GMatch`` dont les deux places sont connues : si
l'ensemble ``{m.A, m.B}`` diffère de ``{g.Players[0], g.Players[1]}``, l'avertissement
``bracket_wrong_players`` (``Match``, ``Section``, ``Label``, ``A``, ``B``, ``ExpectedA`` =
``g.Players[0]``, ``ExpectedB`` = ``g.Players[1]``).

Pour chaque match terminé de longueur ``> 0`` dont un score dépasse la longueur annoncée,
l'avertissement ``score_over_length`` (``Match``, ``Length``, ``ScoreA``, ``ScoreB``).

Dans les phases ``swiss_lives``, deux avertissements disent ce qu'une correction (ou une annulation)
a changé à la vie d'un joueur — un suisse n'a pas de graphe, donc pas de ``bracket_wrong_players`` :

- ``correction_eliminates_running`` (``Match``, ``Player``, ``A``, ``B``, ``Label``) : un match **en
  cours** dont un joueur n'a plus aucune vie. Cela n'arrive pas dans un tournoi mené normalement —
  on ne lance pas un joueur éliminé — ; c'est la trace d'une correction d'un résultat antérieur qui
  l'a éliminé pendant qu'il jouait (un joueur retiré n'est pas signalé). L'annulation du match est
  proposée (``cancel_match``, ``reparation.go`` ; voir :ref:`propositions`). L'avertissement tombe
  quand le match finit ou est annulé.
- ``correction_revives`` (``Player``) : un joueur que la correction a rendu à la vie
  (``State.Revived``). Il est de nouveau appariable, et le moteur le dit jusqu'à ce qu'il rejoue —
  ou qu'une autre correction l'élimine de nouveau. ``Revived`` est tenu par ``noteRevived``, qui
  compare les joueurs en vie dans les phases suisses avant et après ``result_corrected`` ou
  ``match_cancelled`` ; un ``match_started`` en retire ses deux joueurs. Les avertissements sont
  émis dans l'ordre des identifiants (``sortedIDs``).

Et pour chaque table qui porte **plusieurs matchs en cours**, l'avertissement ``table_shared``, une
fois par match arrivé en second dans ``MatchOrder``, avec ``Table``, ``Match`` (ce match), ``A``,
``B`` et ``Other`` (le premier occupant). Il est dérivé de l'état : il disparaît quand l'un des deux matchs
se termine, est annulé ou change de table. Un match sans table (``Table = 0``) ne partage rien. Le
score est **enregistré** malgré tout : pendant un tournoi, c'est la parole du TD qui fait foi ; le
moteur le signale, il ne le refuse pas.

``check`` est rappelée après **tout événement qui touche un match** : ``result``, ``match_started``
et ``table_changed`` directement, les autres (``result_corrected``, ``match_cancelled``,
``player_withdrawn``, ``config_changed``) par ``recompute``, qui la termine (puis de nouveau par
``noteRevived`` après une correction ou une annulation). Un avertissement qui
n'apparaîtrait qu'après une correction sans rapport ne servirait à rien : le TD doit le voir au
moment où il peut encore agir. ``length_changed`` ne touche que les matchs FUTURS d'une phase et ne
change donc rien à ce que ``check`` regarde.

Elle parcourt les **graphes** et non les matchs — chaque place connaît son match par ``MatchID`` —
parce que chercher, pour chaque match, sa place dans toutes les sections serait quadratique, et
``check`` tourne à chaque résultat. L'ordre des avertissements est : pour chaque match dans l'ordre
de ``MatchOrder`` (celui que le TD lit), ``bracket_wrong_players`` puis ``score_over_length`` ;
ensuite ``correction_eliminates_running`` et ``correction_revives`` ; enfin ``table_shared``.

``State.Warnings`` **doit être vide** dans un tournoi mené normalement ; c'est un invariant vérifié
par les tests. Sa non-vacuité signale au TD qu'une correction a désynchronisé un tableau, ou qu'un
score est incohérent.
