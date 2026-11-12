.. _format_manuel:

Appariement manuel (``manual``)
===============================

Principe
--------

Dans une phase ``manual`` le moteur ne propose **aucun appariement** : le directeur pose chaque
match et chaque exemption, ronde par ronde, et décide de la fin de la phase. Personne n'est
éliminé ; le classement est celui des **points** — une victoire vaut un point, une exemption
aussi —, les ex æquo le restent, et ``tiebreak`` les départage comme au suisse classique
(:ref:`format_suisse_rondes`).

Poser une ronde
---------------

.. code:: go

   func (s *State) ManualMatchEvent(round int, a, b PlayerID, now time.Time) Event
   func (s *State) ManualByeEvent(round int, p PlayerID, now time.Time) Event
   func (s *State) ManualByeCancelledEvent(round int, p PlayerID, now time.Time) Event

``ManualMatchEvent`` rend un ``match_started`` et ``ManualByeEvent`` un ``bye``, tous deux dans la
ronde ``round`` de la phase courante. Une ronde s'ouvre en y posant un premier match ou une
première exemption : ``round`` vaut de 1 à la dernière ronde ouverte plus un. Une ronde peut
rester partielle, et une ronde antérieure peut encore être complétée.

``Apply`` ne refuse que ce qui rendrait la ronde incohérente :

- un match contre soi-même ;
- un joueur qui n'est pas dans la phase, ou qui est retiré ;
- un joueur déjà placé dans la ronde — par un match non annulé ou par une exemption ;
- une ronde qui laisserait un trou dans la numérotation.

Un match posé par erreur s'annule (``match_cancelled``) : ses deux joueurs redeviennent libres
dans la ronde. Une exemption posée par erreur se retire de même (``ManualByeCancelledEvent``,
événement ``bye_cancelled``) tant que la phase n'est pas finie : le joueur redevient libre dans
la ronde, qui reste ouverte même vide.

Propositions et fin de phase
----------------------------

``Propose`` rend l'attente ``manual`` et, dès qu'aucun match de la phase ne joue, le passage à
la phase suivante (``next_phase``) ou la clôture (``finish``) — proposés, jamais imposés : c'est
le directeur qui dit quand la phase est finie, y compris sans avoir posé aucun match. ``Apply``
refuse un ``next_phase`` envoyé pendant qu'un match de la phase joue.
``rules.PhaseDone`` reste faux et ``rules.RemainingMatches`` rend zéro : le moteur ne sait pas ce
que le directeur posera encore.

La phase suivante reçoit ses entrants comme après un suisse classique (``survivors`` : tous les
joueurs non retirés ; ``top:N`` : les premiers du classement).
