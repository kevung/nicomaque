.. _integration:

Intégrer le moteur
==================

Cette page décrit ce que le logiciel hôte doit écrire pour utiliser le moteur. Le détail de
chaque type et de chaque événement est dans la :ref:`référence <modele>`.

Répartition des rôles
---------------------

============================================= =============================================
Le moteur                                     L'hôte
============================================= =============================================
reconstruit l'état à partir du journal        stocke le journal, dans l'ordre, sans le
                                              modifier
propose les actions possibles                 les affiche et recueille la décision du
                                              directeur
convertit une action confirmée en événement   ajoute l'événement au journal
renvoie des codes (libellés, avertissements,  traduit ces codes dans la langue de
notes, raisons d'attente)                     l'utilisateur
ne connaît pas l'heure                        fournit l'heure courante (``ProposeAt``)
ne connaît que son tournoi                    signale les tables et joueurs occupés par
                                              d'autres épreuves (``External``)
============================================= =============================================

Fonctions de base
-----------------

========================================= ==================================================
Appel                                     Rôle
========================================= ==================================================
``New(cfg, graine, now)``                 crée l'état et l'événement ``created``
``Replay(journal)``                       reconstruit l'état complet à partir du journal
``State.Propose()``                       actions proposées, à l'heure du dernier événement
``State.ProposeAt(now)``                  actions proposées à l'heure ``now``
``State.ProposeWith(now, ext)``           idem, avec les tables et joueurs occupés ailleurs
``State.EventFromAction(a, now)``         événement correspondant à une action confirmée
``State.Apply(ev)``                       applique un événement à l'état
========================================= ==================================================

La boucle
---------

.. code:: go

   // 1. Créer le tournoi
   cfg := tournoi.Config{Name: "Open de printemps", Phases: []tournoi.PhaseConfig{
       {Kind: tournoi.KindSwissLives, Length: 7, Target: 16},
       {Kind: tournoi.KindLivesBracket, Length: 9, FinalLength: 11},
   }}
   st, cree, err := tournoi.New(cfg, graine, time.Now())
   journal := tournoi.Journal{cree}

   // 2. Inscrire un joueur
   ev := tournoi.PlayerAddedEvent(joueur, time.Now())
   st.Apply(ev)
   journal = append(journal, ev)

   // 3. Proposer, puis appliquer ce que le directeur confirme
   for _, a := range st.ProposeAt(time.Now()) {
       // afficher l'action ; si le directeur la confirme :
       ev, err := st.EventFromAction(a, time.Now())
       st.Apply(ev)
       journal = append(journal, ev)
   }

   // 4. Saisir un résultat
   ev = tournoi.ResultEvent(idDuMatch, vainqueur, scoreA, scoreB, time.Now())
   st.Apply(ev)
   journal = append(journal, ev)

Le journal se sérialise en JSON (``json.Marshal``). Une reprise après interruption est un
``Replay`` du journal enregistré. Le fichier ``doc.go`` du dépôt contient un exemple complet.

Événements produits par l'hôte
------------------------------

Ces événements ne viennent pas d'une action proposée ; l'hôte les construit directement.

=========================================== ===============================================
Constructeur                                Usage
=========================================== ===============================================
``PlayerAddedEvent``                        inscrire un joueur
``PlayerAddedAtSlotEvent``                  inscrire un retardataire sur une place
                                            d'exemption libre (``State.FreeSlots``)
``PlayerUpdatedEvent``                      corriger la fiche d'un joueur
``PlayerUnavailableEvent``,                 rendre un joueur indisponible jusqu'à une heure
``PlayerUnavailableUntilRoundEvent``        ou une ronde
``PlayerAvailableEvent``                    le rendre disponible
``PlayerWithdrawnEvent``                    retirer un joueur immédiatement
``PlayerWithdrawnAfterCurrentEvent``        le retirer après son match en cours
``ForfeitEvent``                            forfait sur un seul match
``ResultEvent``                             saisir un résultat
``CorrectionEvent``                         corriger un résultat
``CancelEvent``                             annuler un match lancé par erreur
``TableChangedEvent``                       changer la table d'un match
``LengthChangedEvent``                      changer la longueur des matchs d'une phase
``ConfigChangedEvent``                      remplacer la configuration entière
``ReopenedEvent``                           rouvrir un tournoi clos
``NoteEvent``                               consigner une note libre
=========================================== ===============================================

Règles à respecter
------------------

- **Le journal n'est jamais modifié.** Une correction, une annulation, un retrait sont des
  événements ajoutés.
- **Rien de ce qui sort du moteur n'est un texte à afficher.** Libellés de match, notes de
  classement, avertissements et raisons d'attente sont des codes (voir :ref:`journal`). Le
  paquet racine fournit un rendu français (``fr.go``) pour la console et les exemples.
- **Les tirages sont enregistrés.** L'événement ``draw`` contient le placement obtenu ; rejouer
  un journal ne dépend donc pas de l'algorithme de tirage.
- **Le moteur n'a pas d'horloge.** ``Propose()`` utilise l'horodatage du dernier événement ;
  pour les micro-rondes et les pauses, appeler ``ProposeAt`` avec l'heure réelle.

Affichage
---------

Le paquet ``render`` produit des pages HTML autonomes (un fichier, feuille de style embarquée,
aucun script) à partir de l'état et d'un traducteur ``Labeler`` fourni par l'hôte. Voir
:ref:`affichage`.
