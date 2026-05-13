# Refactoring Roadmap

## Phase 1: Fondations — domaine & config
*Bloque tout le reste. À faire avant d'écrire une seule ligne de sync ou de TUI.*

### Renommer le binaire & module Go
*   **Description**: Choisir le nom final (ex. `pm` ou `tkt`), mettre à jour `go.mod`, tous les imports, le Makefile et les workflows CI.
*   **Estimation**: ~1h, mécanique

### Nettoyer le modèle de domaine
*   **Description**: Purger `wikifiedDescription`, `CommandResult` et tout champ YouTrack-spécifique de `internal/model`. Ajouter `ProviderRef` (provider, key, ref) sur `LocalIssue` en remplacement de `remote_id`/`remote_db_id`.
*   **Estimation**: ~2h

### Refondre la config — tout par projet
*   **Description**: Supprimer `~/.config/ytcli/config.yml` et la config globale utilisateur. Créer `.pmrc.yml` (versionné, remplace `.ytcli.yml`) avec la section `provider`. Le token va dans `~/.config/pm/credentials.yml` pointé par `.pmrc.yml`, pas stocké dedans.
*   **Estimation**: ~3h

### Définir l'interface `RemoteProvider`
*   **Description**: Créer `internal/provider/provider.go` avec l'interface Go : `Auth`, `FetchIssues`, `PushOp`, `FetchSchema`, `Ping`. Définir les types génériques `RemoteIssue`, `Query`, `QueueOp` indépendants de tout provider.
*   **Estimation**: ~3h

### Migrer le schema SQLite
*   **Description**: Migration v2 : remplacer `remote_id`/`remote_db_id` par `provider_name`, `provider_key`, `provider_ref`. Ajouter colonne `provider_name` sur `sync_queue`. Mettre à jour toutes les fonctions `store/`.
*   **Estimation**: ~4h

---

## Phase 2: Provider YouTrack — adapter modulaire
*Dépend de Phase 1*

### Déplacer le client API YouTrack
*   **Description**: Migrer `internal/api/` vers `internal/provider/youtrack/`. Le package implémente `RemoteProvider`. Toute la logique YouTrack (commands API, champs custom, wikifiedDescription) reste ici — elle ne remonte plus dans le domaine.
*   **Estimation**: ~4h

### Mapper `RemoteIssue` ↔ `model.Issue`
*   **Description**: Écrire le mapper YouTrack→domaine dans l'adapter. Extraire State, Priority et Assignee des `customFields` ici, pas dans le sync engine. Le sync engine ne voit que des `model.Issue` propres.
*   **Estimation**: ~3h

### Refaire l'authentification
*   **Description**: Auth YouTrack via token permanent (et OAuth optionnel). Le flow est déclenché par `pm remote auth [provider]`, pas `ytcli auth login`. Credentials stockés via keyring, fallback fichier — mais pointé par `.pmrc.yml`, pas cherché globalement.
*   **Estimation**: ~2h

### Migrer les tests API existants
*   **Description**: Adapter les tests de `internal/api/api_test.go` au nouveau package. Conserver la couverture — elle est bonne.
*   **Estimation**: ~2h

---

## Phase 3: Sync engine — découplé & robuste
*Dépend de Phase 1 & 2*

### Réécrire le `SyncManager` sur l'interface
*   **Description**: `sync.Manager` reçoit un `RemoteProvider`, pas un `*service.Service` YouTrack. Pull et Push passent exclusivement par l'interface. Zéro import YouTrack dans `internal/sync/`.
*   **Estimation**: ~4h

### Erreurs structurées & retry intelligent
*   **Description**: Créer `internal/errs/` : types `AuthError`, `NetworkError`, `ValidationError`, `ConflictError` avec contexte (provider, opération, id). Messages d'erreur human-readable et machine-readable distincts. Backoff exponentiel sur `NetworkError`, pas sur `AuthError`.
*   **Estimation**: ~3h

### Gestion des conflits explicite
*   **Description**: Implémenter `ConflictResolver` avec stratégies pluggables : `local-wins`, `remote-wins`, `manual` (défaut). Stocker les conflits en DB avec diff JSON pour résolution dans le TUI.
*   **Estimation**: ~4h

### Workflow state machine
*   **Description**: Valider les transitions d'état contre le workflow du projet (`SchemaConfig`) avant d'écrire en DB. Retourner `ValidationError` si la transition est invalide. Permettre de bypasser avec `--force`.
*   **Estimation**: ~3h

---

## Phase 4: CLI — UX refondée
*Dépend de Phase 1, 2, 3*

### Restructurer l'arbre de commandes
*   **Description**: Supprimer les commandes YouTrack-spécifiques du root (`cmd`, `issues` vs `list`). Nouvelle structure : `pm init`, `pm add`, `pm ls`, `pm show`, `pm edit`, `pm done`, `pm sync`, `pm remote [add|auth|pull|push]`, `pm config`.
*   **Estimation**: ~4h

### Erreurs human-readable en CLI
*   **Description**: Chaque `handleError` affiche : ce qui a échoué, pourquoi (depuis le type d'erreur), et quoi faire (hint contextuel). Ex. `AuthError` → "Token expiré — relancez : pm remote auth youtrack". Plus de stack traces nues.
*   **Estimation**: ~3h

### `pm remote` — sous-commande provider
*   **Description**: Regrouper toutes les interactions provider : `pm remote add youtrack https://…`, `pm remote auth`, `pm remote pull`, `pm remote push`, `pm remote status`. La gestion multi-provider est claire et extensible.
*   **Estimation**: ~3h

### Sortie JSON & exit codes — audit complet
*   **Description**: Vérifier que chaque commande respecte les exit codes (0-4) et le format JSON défini dans `AGENTS.md`. Ajouter les cas manquants. Mettre à jour `AGENTS.md` avec le nouveau nom de commande.
*   **Estimation**: ~2h

---

## Phase 5: TUI — interface principale
*Dépend de Phase 1, 3, 4*

### Vue liste + détail (split)
*   **Description**: Panel gauche : liste scrollable avec filtre live (`/`). Panel droit : détail de l'issue sélectionnée avec description rendue en markdown. Navigation vim (`j/k`, `gg/G`). Indicateur de sync dans la status bar.
*   **Estimation**: ~8h

### Command palette (`:`)
*   **Description**: Fuzzy search sur toutes les actions disponibles : `:state done`, `:assign @me`, `:tag bug`, `:sync`. Remplace le paradigme "connaître la commande par cœur" — découverte naturelle.
*   **Estimation**: ~5h

### Vue board kanban (`b`)
*   **Description**: Colonnes = états du workflow. Drag-and-drop via clavier (`m` pour déplacer). Affiche le nombre d'issues par colonne. Bascule liste/board sans perdre le filtre actif.
*   **Estimation**: ~6h

### Édition inline & formulaires
*   **Description**: Appuyer `e` sur une issue ouvre un formulaire `huh` pour éditer résumé, état, priorité, assignee. Appuyer `c` pour ajouter un commentaire. Sauvegarde locale immédiate, sync queued.
*   **Estimation**: ~4h

### Résolution de conflits interactive
*   **Description**: Quand `pm remote pull` détecte un conflit, le TUI présente un diff côte-à-côte (local vs remote) et demande quelle version garder ou comment merger. Intégré dans la status bar avec notification.
*   **Estimation**: ~5h
