.. _joueurs_csv:

Import et export de joueurs (``players``)
=========================================

.. code:: go

   func FromCSV(b []byte) ([]tournoi.Player, error)

Import souple destiné aux exports HelloAsso, FFBG ou tableur :

- BOM UTF-8 retiré ;
- séparateur **détecté** sur la première ligne parmi ``;``, ``,`` et tabulation (celui qui apparaît
  le plus souvent, dans cet ordre à égalité ; ``,`` si aucun n'apparaît) ;
- lecture tolérante (``FieldsPerRecord = -1``, ``LazyQuotes``) ;
- un fichier vide est une erreur (``csv vide``) ;
- colonnes reconnues sur l'en-tête, en minuscules et sans espaces aux extrémités : un en-tête
  **égal** à l'un des mots-clés l'emporte, sinon le premier en-tête qui en **contient** un : ``id`` ;
  nom (``nom complet``, ``name``, ``nom``) ; prénom (``prénom``, ``prenom``, ``first``) ; nom de
  famille (``nom de famille``, ``last``) ; ``club`` ; PR (``cote``, ``pr``, ``rating``, ``elo``).
  La priorité à l'égalité exacte évite que « prénom » soit pris pour la colonne ``pr`` d'un fichier
  qui a une vraie colonne de cote ;
- si ni colonne de nom ni colonne de prénom : erreur ;
- le nom est la colonne « nom », sauf si elle est vide, ou si prénom et nom de famille existent
  tous deux et que le prénom est renseigné — auquel cas c'est la concaténation prénom + nom de
  famille ;
- une ligne sans nom est ignorée ;
- l'identifiant est la colonne ``id`` si elle est renseignée, sinon un **slug** du nom ; en cas de
  collision, l'identifiant devient le slug du nom suivi de ``-2``, ``-3``\ … ;
- le PR accepte la virgule décimale ; une valeur illisible laisse ``Rating`` à 0.

Le slug conserve les minuscules ASCII, les chiffres et les caractères non-ASCII (donc les lettres
accentuées), remplace espace, tiret, souligné et apostrophe par un tiret, supprime le reste, puis
élague les tirets aux extrémités.

.. code:: go

   func ToCSV(ps []tournoi.Player) []byte

Export dans le format que ``FromCSV`` relit : séparateur point-virgule, colonnes ``nom``, ``club``,
``cote``. C'est l'annuaire de l'hôte, qu'on exporte pour le réimporter d'un tournoi à l'autre et
qu'un organisateur ouvre dans un tableur. Une cote inconnue (0) sort en cellule vide, et non en
« 0 » qui se lirait comme un joueur parfait ; elle revient à 0 à la relecture. Une colonne ``id`` n'est ajoutée en tête que si elle est **nécessaire** : quand au moins un
identifiant diffère de celui que ``FromCSV`` dériverait du nom (slug, avec les mêmes suffixes de
collision), pour qu'un identifiant venu d'ailleurs ne se perde pas.
