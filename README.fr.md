# Go Todo List API

#### [English](README.md) | Français | [Русский](README.ru.md)
### Consultez la [démo en ligne](https://go-todo-list-api.onrender.com/) (mot de passe : `test12345`)

**Veuillez noter que la démo en ligne est hébergée sur un plan gratuit, il va donc falloir un certain temps pour que le serveur se lance lorsque le site est accédé.**

## Aperçu du projet
Ce projet est une API simple de liste de tâches construite en Go. Il fournit un service backend permettant aux utilisateurs de créer, lire,
mettre à jour et supprimer des tâches. L'application utilise des JSON Web Tokens (JWT) pour une authentification sécurisée et SQLite pour un stockage persistant des données. En plus de la gestion basique des tâches, l'API prend également en charge la planification des tâches avec des intervalles de répétition personnalisés.

L'API est construite en utilisant le modèle d'architecture en couches, avec des couches séparées pour l'API. Il y a 4 couches principales :
- **Controller Layer** : Gère les requêtes HTTP entrantes et les dirige vers le gestionnaire approprié situé dans le répertoire `internal/server`.
- **Service Layer** : Contient la logique métier de l'application située dans le répertoire `internal/service`.
- **Repository Layer** : Gère l'interaction avec la base de données située dans le répertoire `internal/storage`.
- **Entities Layer** : Contient les entités de données utilisées par l'application situées dans le répertoire `internal/models`.

En outre, l'application dispose d'une interface web simple pour interagir avec l'API. L'interface web est construite en utilisant HTML, CSS et JavaScript, qui sont minifiés et situés dans le répertoire `web/`.

Il y a un Dockerfile dans le répertoire racine du projet qui peut être utilisé pour créer une image Docker de l'application. Le Dockerfile utilise une construction multi-étapes pour créer une image légère avec le binaire de l'application et les fichiers nécessaires.

## Fonctionnalités
- Gestion des tâches : Créer, lire, mettre à jour et supprimer des tâches.
- Planification des tâches : Planifier des tâches pour des dates futures avec la possibilité de définir un intervalle de répétition personnalisé.
- Authentification : Connexion sécurisée avec JWT.
- Stockage persistant avec SQLite.
- API RESTful construite avec le routeur Chi.

## Dépendances
Le projet utilise les dépendances suivantes :
- **Chi Router** : Routage léger et idiomatique en Go (`github.com/go-chi/chi/v5`)
- **JWT** : Gestion de l'authentification (`github.com/golang-jwt/jwt/v4`)
- **SQLx** : Outils SQL pour Go (`github.com/jmoiron/sqlx`)
- **SQLite3** : Pilote de base de données (`github.com/mattn/go-sqlite3`)
- **Testify** : Utilitaires de test (`github.com/stretchr/testify`)

**Remarque :** Vous avez besoin de la version **1.22.2** de Go ou supérieure pour exécuter l'application.

## Installation
1. Clonez le dépôt :
```bash
git clone https://github.com/antonkazachenko/go-todo-list-api.git
```

2. Accédez au répertoire du projet :
```bash
cd go-todo-list-api
```

3. Installez les dépendances :
```bash
go mod tidy
```
   

## Variables d'environnement
Pour exécuter le serveur localement, vous pouvez configurer les variables d'environnement suivantes :

- `TODO_DBFILE` : Chemin vers le fichier de base de données SQLite (par défaut : `scheduler.db`)
- `TODO_PORT` : Port sur lequel le serveur s'exécutera (par défaut : `7540`)
- `TODO_PASSWORD` : Mot de passe utilisé pour la signature JWT (par défaut : vide)

Vous pouvez définir ces variables d'environnement dans votre shell avant d'exécuter l'application :

```bash
export TODO_DBFILE="your_db_file.db"
export TODO_PORT="your_port_number"
export TODO_PASSWORD="your_password"
```

Si vous ne définissez pas les variables d'environnement, l'application utilisera les valeurs par défaut.

## Utilisation
1. Construisez et exécutez le projet :
```bash
go run main.go
```

2. Accédez à l'API via `http://localhost:PORT/` (Remplacez `PORT` par le port réel spécifié dans votre configuration ou le port par défaut `7540`).

## Points de terminaison de l'API
Voici un aperçu des principaux points de terminaison de l'API :

- **POST /api/task** - Créer une nouvelle tâche.
- **GET /api/tasks** - Obtenir toutes les tâches.
- **GET /api/task** - Obtenir une tâche spécifique.
- **PUT /api/task** - Mettre à jour une tâche spécifique.
- **DELETE /api/task** - Supprimer une tâche spécifique.
- **POST /api/task/done** - Marquer une tâche comme terminée.
- **POST /api/signin** - Connexion utilisateur.

## Authentification
L'authentification dans cette application est gérée à l'aide de JSON Web Tokens (JWT). Après une connexion réussie, un JWT est généré et retourné à l'utilisateur. Cette fonctionnalité peut être vue dans l'onglet réseau des outils de développement du navigateur.

Si vous ne configurez pas la variable d'environnement `TODO_PASSWORD`, l'application n'utilisera pas JWT pour l'authentification.

## Tests
- Le projet utilise Testify pour les tests unitaires.
- Les tests sont situés dans le répertoire `tests/`.
- Exécutez les tests avec :
```bash
go test ./tests
```


## Contribution
Les contributions sont les bienvenues ! N'hésitez pas à soumettre une Pull Request.

## Licence
Ce projet est sous licence MIT. Voir le fichier `LICENSE` pour plus de détails.
