#!/usr/bin/env python3
"""Construit la documentation de Nicomaque en français et en anglais.

    python doc/build.py              # les deux langues, dans doc/build/site/
    python doc/build.py --langue fr  # une seule, pour aller vite

Le français est la source (doc/source/*.rst) ; l'anglais est le catalogue gettext
doc/source/locale/en. Une chaîne non traduite retombe sur le français.

Le site produit a une page racine qui redirige vers le français et un dossier par
langue ; c'est ce dossier que .github/workflows/pages.yml publie.
"""

import argparse
import pathlib
import shutil
import subprocess
import sys

DOC = pathlib.Path(__file__).resolve().parent
SOURCE = DOC / "source"
BUILD = DOC / "build"
SITE = BUILD / "site"

# Garder en accord avec LANGUAGES dans source/conf.py.
LANG = {"fr": "Français", "en": "English"}

# Avertissements qui signifient une page mal rendue, et pas seulement imparfaite :
# balisage cassé, renvoi perdu par une traduction. Ils font échouer la construction.
BLOCKING = ("[docutils]", "[i18n.inconsistent_references]")


def construire(code):
    avertissements = BUILD / f"warnings-{code}.txt"
    subprocess.run([sys.executable, "-m", "sphinx", "-b", "html", "-q",
                    "-D", f"language={code}", "-w", str(avertissements),
                    "-d", str(BUILD / "doctrees" / code),
                    str(SOURCE), str(SITE / code)], check=True)
    texte = avertissements.read_text(encoding="utf-8")
    return [l for l in texte.splitlines()
            if "WARNING" in l and l.rstrip().endswith(BLOCKING)]


def racine():
    """La racine du site redirige vers le français et liste les langues."""
    liens = "\n".join(f'<li><a href="{c}/" hreflang="{c}">{n}</a></li>' for c, n in LANG.items())
    (SITE / "index.html").write_text(f"""<!doctype html>
<html lang="fr"><head><meta charset="utf-8">
<meta http-equiv="refresh" content="0; url=fr/">
<title>Nicomaque</title>
</head><body>
<h1>Nicomaque</h1>
<ul>
{liens}
</ul>
</body></html>
""", encoding="utf-8")
    (SITE / ".nojekyll").write_text("", encoding="utf-8")


def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--langue", action="append", choices=list(LANG),
                    help="ne construire que cette langue")
    args = ap.parse_args()

    shutil.rmtree(SITE, ignore_errors=True)
    SITE.mkdir(parents=True)
    bloquants = {}
    for code in args.langue or list(LANG):
        print(f"  {code}")
        trouves = construire(code)
        if trouves:
            bloquants[code] = trouves
    racine()
    if bloquants:
        for code, lignes in bloquants.items():
            print(f"\n{code} : {len(lignes)} avertissement(s) bloquant(s)")
            for l in lignes:
                print("  " + l)
        sys.exit("balisage cassé ou renvoi perdu (voir ci-dessus) : corriger le .rst ou le msgstr")
    print(f"documentation construite dans {SITE}")


if __name__ == "__main__":
    main()
