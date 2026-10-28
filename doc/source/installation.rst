.. _installation:

Installation
============

Prérequis
---------

- Go 1.22 ou plus récent.
- Aucune autre dépendance : le module n'utilise que la bibliothèque standard.

Ajouter le module à un projet
-----------------------------

.. code:: bash

   go get github.com/kevung/nicomaque@latest

.. code:: go

   import tournoi "github.com/kevung/nicomaque"

Le chemin de module est ``github.com/kevung/nicomaque`` depuis la version 0.8.0. Les versions
0.1.0 à 0.7.1 ont été publiées sous ``github.com/PileOfCells/backgammon-tournoi`` ; ce module est
déprécié et renvoie au nouveau (voir :ref:`historique`).

Paquets
-------

=========================================== ===================================================
Paquet                                      Contenu
=========================================== ===================================================
``github.com/kevung/nicomaque``             le moteur : configuration, journal, état,
                                            propositions, classement, dotation
``github.com/kevung/nicomaque/rules``       les règles en fonctions pures (appariement,
                                            classement, dotation, prévision), sans état
``github.com/kevung/nicomaque/render``      pages HTML et SVG autonomes (voir :ref:`affichage`)
``github.com/kevung/nicomaque/sim``         simulation de tournois et prévision de fin
                                            (voir :ref:`simulation`)
``github.com/kevung/nicomaque/players``     import et export de joueurs en CSV
                                            (voir :ref:`joueurs_csv`)
``cmd/tournoi-demo``, ``cmd/tournoi-td``    outils en ligne de commande (voir :ref:`cli`)
=========================================== ===================================================

Essayer sans écrire de code
---------------------------

Depuis un clone du dépôt :

.. code:: bash

   git clone https://github.com/kevung/nicomaque
   cd nicomaque

   # simuler un tournoi complet de 32 joueurs et écrire ses pages dans demo/
   go run ./cmd/tournoi-demo -format suisse_tableau -joueurs 32 -etapes -sortie demo

   # diriger un tournoi à la main, dans une console
   go run ./cmd/tournoi-td -journal montournoi.json -format suisse_tableau

Le dossier ``exemples/`` du dépôt contient les sorties de ``tournoi-demo -etapes`` pour sept
formats ; ``exemples/index.html`` en est la galerie.

Vérifier le module
------------------

.. code:: bash

   go test -short ./...   # matrice réduite, quelques secondes
   go test ./...          # matrice complète et campagne TestParity, plusieurs minutes

Le contenu des tests est décrit dans :ref:`annexe_tests`.

Construire cette documentation
------------------------------

.. code:: bash

   python3 -m venv .venv-doc
   .venv-doc/bin/pip install -r doc/requirements.txt
   .venv-doc/bin/python doc/build.py              # français et anglais, dans doc/build/site/
   .venv-doc/bin/python doc/build.py --langue fr  # une seule langue
