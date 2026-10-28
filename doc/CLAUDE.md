# doc — règles

Sphinx, thème Read the Docs, deux langues : le français est la source (`source/*.rst`), l'anglais
est le catalogue gettext `source/locale/en/LC_MESSAGES/`. Construction :
`.venv-doc/bin/python doc/build.py` (`--langue fr` pour une seule) ; publication sur GitHub Pages
à chaque poussée sur `main` (`.github/workflows/pages.yml`).

- **Un `.rst` modifié part avec son `.po` dans le même commit** : `scripts/doc-po-update.sh`,
  remplir les `msgstr` vides ou fuzzy avec `scripts/po-fill.py <traductions.json>`, puis
  `scripts/doc-i18n-check.sh` doit finir sur « all translations complete ». Ne jamais lancer
  sphinx-intl à la main (des chemins absolus réécrivent tous les `#:`) ni réécrire un `.po` avec
  msgcat, Babel ou polib (ils reformatent tout le fichier et noient le vrai changement).
- **Ne jamais lire un `.po` en entier** : `grep -n` le msgid, puis lire la plage.
- **Présent, version publiée seulement** : pas de « bientôt », pas d'historique dans les pages de
  référence. Le travail restant vit dans `docs/RESTE_A_FAIRE.md` ; ce qui manque aujourd'hui
  dans `limites.rst` ; ce qui a changé dans `historique.rst`.
- **Le code fait foi.** La référence (`modele`, `configuration`, `journal`, `etat`,
  `propositions`, `graphes`, `format_*`, `classement`, `statut`, `horloge`, paquets annexes) est
  la spécification du moteur : la mettre à jour dans le même commit que le comportement.
- **Le guide montre une tâche, la référence décrit un mécanisme** : une description détaillée
  dans `guide_utilisateur.rst` devient un `:ref:` vers la page de référence.
- Les chiffres de `annexe_etude.rst` (« Chiffres de référence ») et de `formats.rst`
  (« Comparaison ») viennent de `go test -run TestParity -v ./` : les refaire si l'appariement
  change.
- Une nouvelle page s'ajoute à un `toctree` de `index.rst` (ou de `formats.rst`) et commence par
  `.. _<nom>:`.
- Ne jamais commettre un `.mo`.
