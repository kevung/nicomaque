# Configuration Sphinx de la documentation de Nicomaque.
#
# Le français est la langue source ; l'anglais est un catalogue gettext dans
# locale/en/LC_MESSAGES. Construction : `python doc/build.py` (voir doc/CLAUDE.md).

project = 'Nicomaque'
copyright = '2026, Nicolas Harmand, Kévin Unger'
author = 'Nicolas Harmand, Kévin Unger'
# Dernière étiquette du module ; à relever à chaque version publiée.
release = '0.8.0'

extensions = [
        'sphinx_rtd_theme',
        ]

# Les langues publiées. Garder en accord avec LANG dans doc/build.py.
LANGUAGES = [
        ("fr", "Français"),
        ("en", "English"),
        ]

language = 'fr'
templates_path = ['_templates']
locale_dirs = ['locale/']
gettext_compact = False  # un catalogue par page : les diffs restent lisibles
gettext_uuid = False

exclude_patterns = []

# Les blocs « :: » sont du pseudo-code : pas de coloration par défaut.
highlight_language = 'none'

html_theme = "sphinx_rtd_theme"
html_static_path = ['_static']
html_css_files = [
        'theme_overrides.css',
        'custom.css',
        ]
html_show_sphinx = False
html_show_sourcelink = False
html_copy_source = False
html_title = "Nicomaque"
html_baseurl = "https://kevung.github.io/nicomaque/"

html_context = {
        'languages': [[code, name, "../" + code] for code, name in LANGUAGES]
        }
