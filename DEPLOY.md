# 🚀 Guide de Déploiement

Guide complet pour publier et distribuer Épurer.

---

## 📋 Prérequis

- [x] Repository GitHub créé et public
- [x] Code pushé sur GitHub
- [x] Tag v1.0.0 créé
- [ ] GitHub Personal Access Token (pour GoReleaser, optionnel)

---

## 🎯 Méthode 1 : Publication Manuelle (Recommandée pour v1.0.0)

### Étape 1 : Vérifier que le repo est public

```bash
# Vérifier sur GitHub que le repo est bien PUBLIC
# https://github.com/0SansNom/epurer/settings
```

### Étape 2 : Compiler les binaries

```bash
cd /Users/0SansNom/Downloads/epurer

# Nettoyer
make clean

# Compiler pour Apple Silicon (ARM64)
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o epurer ./cmd/epurer
tar -czf epurer_1.0.0_darwin_arm64.tar.gz epurer README.md LICENSE
rm epurer

# Compiler pour Intel (AMD64)
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o epurer ./cmd/epurer
tar -czf epurer_1.0.0_darwin_amd64.tar.gz epurer README.md LICENSE
rm epurer

# Générer les checksums
shasum -a 256 *.tar.gz > checksums.txt
```

### Étape 3 : Créer la Release GitHub

1. Aller sur : https://github.com/0SansNom/epurer/releases/new

2. **Configuration** :
   - Choose a tag : `v1.0.0` (existant)
   - Release title : `v1.0.0 - Initial Release`
   - Target : `main`

3. **Description** (copier-coller) :

```markdown
# Épurer v1.0.0 🧹

**Intelligent cache cleanup for macOS developers**

First stable release! Clean your development caches and reclaim disk space safely.

## ✨ Features

- 🔍 **Smart Detection** - Automatically detects installed development tools
- 🚀 **Concurrent Scanning** - Fast parallel filesystem operations
- 🛡️ **Three Safety Levels** - Conservative, Standard, Aggressive
- 📊 **Beautiful CLI** - Colored tables and progress bars
- 💾 **Dry Run Mode** - Preview before deleting
- 🎯 **Domain Filtering** - Target specific technology stacks

## 🎯 Supported Technologies

- **Frontend**: Node.js, npm, yarn, pnpm, Vite, Webpack, Parcel, Next.js
- **Backend**: Python, Java, Go, Rust, PHP, Ruby, Maven, Gradle
- **Mobile**: Xcode, Android Studio, Flutter, CocoaPods
- **DevOps**: Docker, Kubernetes, Terraform, Helm
- **Data/ML**: Conda, Jupyter, TensorFlow, PyTorch, Hugging Face
- **System**: Caches, logs, Homebrew, trash

## 📦 Installation

### Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/0SansNom/epurer/main/install.sh | bash
```

### Manual Installation

**Apple Silicon (M1/M2/M3):**
```bash
curl -LO https://github.com/0SansNom/epurer/releases/download/v1.0.0/epurer_1.0.0_darwin_arm64.tar.gz
tar -xzf epurer_1.0.0_darwin_arm64.tar.gz
sudo mv epurer /usr/local/bin/
sudo chmod +x /usr/local/bin/epurer
```

**Intel Macs:**
```bash
curl -LO https://github.com/0SansNom/epurer/releases/download/v1.0.0/epurer_1.0.0_darwin_amd64.tar.gz
tar -xzf epurer_1.0.0_darwin_amd64.tar.gz
sudo mv epurer /usr/local/bin/
sudo chmod +x /usr/local/bin/epurer
```

## 🚀 Quick Start

```bash
epurer detect      # Detect installed tools
epurer report      # Generate cleanup report
epurer clean       # Clean with default settings
epurer smart       # Smart automatic cleanup
```

## 📚 Documentation

- [README](https://github.com/0SansNom/epurer#readme) - Complete guide
- [CONTRIBUTING](https://github.com/0SansNom/epurer/blob/main/CONTRIBUTING.md) - How to contribute

## 📝 What's New

- Initial release with full feature set
- Support for 40+ development tools
- Intelligent cleanup with safety levels
- Beautiful CLI interface

## 📝 License

MIT License
```

4. **Attacher les fichiers** :
   - Glisser-déposer :
     - `epurer_1.0.0_darwin_arm64.tar.gz`
     - `epurer_1.0.0_darwin_amd64.tar.gz`
     - `checksums.txt`

5. **Options** :
   - ✅ Set as the latest release
   - ❌ Set as a pre-release

6. **Publier** : Cliquer sur "Publish release"

### Étape 4 : Tester l'installation

```bash
# Attendre 1-2 minutes que GitHub propage les fichiers

# Tester le script d'installation
curl -fsSL https://raw.githubusercontent.com/0SansNom/epurer/main/install.sh | bash

# Vérifier
epurer --version
epurer detect
```

---

## ⚡ Méthode 2 : Automatisation avec GoReleaser (Pour versions futures)

### Setup Initial (Une seule fois)

```bash
# Installer GoReleaser
brew install goreleaser

# Créer un GitHub Token
# https://github.com/settings/tokens
# Permissions : repo, workflow

# Sauvegarder le token
export GITHUB_TOKEN="ghp_votre_token_ici"
# Ou ajouter dans ~/.zshrc :
# export GITHUB_TOKEN="ghp_votre_token_ici"
```

### Publier une nouvelle version

```bash
# 1. Faire les modifications du code
# ...

# 2. Commit et tag
git add .
git commit -m "Add new feature X"
git tag -a v1.1.0 -m "Release v1.1.0 - Add feature X"

# 3. Push
git push origin main
git push origin v1.1.0

# 4. Release avec GoReleaser
goreleaser release --clean
```

**GoReleaser fera automatiquement** :
- ✅ Compiler pour ARM64 et AMD64
- ✅ Créer les archives .tar.gz
- ✅ Générer les checksums
- ✅ Créer la release GitHub
- ✅ Upload les binaries
- ✅ Générer le changelog

---

## 🍺 Méthode 3 : Distribution via Homebrew (Avancé)

### Créer un Homebrew Tap

1. **Créer un nouveau repository** : `homebrew-tap`

2. **Créer la Formula** : `Formula/epurer.rb`

```ruby
class Epurer < Formula
  desc "Intelligent developer cache cleaner for macOS"
  homepage "https://github.com/0SansNom/epurer"
  version "1.0.0"
  license "MIT"

  if Hardware::CPU.intel?
    url "https://github.com/0SansNom/epurer/releases/download/v1.0.0/epurer_1.0.0_darwin_amd64.tar.gz"
    sha256 "SHA256_AMD64_ICI"  # Copier depuis checksums.txt
  elsif Hardware::CPU.arm?
    url "https://github.com/0SansNom/epurer/releases/download/v1.0.0/epurer_1.0.0_darwin_arm64.tar.gz"
    sha256 "SHA256_ARM64_ICI"  # Copier depuis checksums.txt
  end

  def install
    bin.install "epurer"
  end

  test do
    system "#{bin}/epurer", "--version"
  end
end
```

3. **Utilisation** :

```bash
# Installation
brew tap 0SansNom/tap
brew install epurer

# Mise à jour
brew upgrade epurer
```

---

## 📝 Checklist Complète de Déploiement

### Avant la release

- [ ] Tests passent : `make test`
- [ ] Binary compile : `make build`
- [ ] Version mise à jour dans CHANGELOG.md
- [ ] README à jour
- [ ] Commit et tag créés

### Publication

- [ ] Binaries compilés (ARM64 + AMD64)
- [ ] Archives .tar.gz créées
- [ ] Checksums générés
- [ ] Release GitHub créée
- [ ] Fichiers uploadés
- [ ] Release publiée (pas pre-release)

### Après publication

- [ ] Installation testée
- [ ] Script install.sh fonctionne
- [ ] Badge de version ajouté au README
- [ ] Annonce (Reddit, Twitter, etc.)

---

## 🔄 Workflow pour Versions Futures

### Version Patch (v1.0.1 - Bug fixes)

```bash
# Fix le bug
git add .
git commit -m "Fix: description du bug"

# Tag
git tag -a v1.0.1 -m "Release v1.0.1 - Bug fixes"

# Push
git push origin main --tags

# Release (manuelle ou GoReleaser)
```

### Version Minor (v1.1.0 - Nouvelles features)

```bash
# Ajouter la feature
git add .
git commit -m "feat: nouvelle feature"

# Update CHANGELOG.md
git add CHANGELOG.md
git commit -m "docs: update changelog for v1.1.0"

# Tag
git tag -a v1.1.0 -m "Release v1.1.0 - New features"

# Push et release
git push origin main --tags
goreleaser release --clean  # ou manuelle
```

### Version Major (v2.0.0 - Breaking changes)

Même processus, mais :
- Documenter les breaking changes
- Fournir un guide de migration
- Communiquer largement

---

## 🆘 Dépannage

### Le script install.sh retourne 404

**Cause** : Repository privé ou fichier pas pushé

**Solution** :
```bash
# Vérifier que le repo est public
# Vérifier que install.sh est présent
git ls-tree -r HEAD --name-only | grep install.sh

# Si absent, l'ajouter
git add install.sh
git commit -m "Add install script"
git push
```

### GoReleaser échoue

**Cause** : Token GitHub manquant ou invalide

**Solution** :
```bash
# Vérifier le token
echo $GITHUB_TOKEN

# Créer un nouveau token si nécessaire
# https://github.com/settings/tokens

# Utiliser --token directement
goreleaser release --clean --token="ghp_votre_token"
```

### Binary trop gros

**Solution** : Utiliser UPX pour compresser
```bash
brew install upx
upx --best epurer
```

---

## 📊 Monitoring Post-Release

### GitHub Insights

- ⭐ Stars
- 👁️ Watchers
- 🍴 Forks
- 📥 Downloads (releases)

### Métriques à suivre

```bash
# Nombre de downloads
curl -s https://api.github.com/repos/0SansNom/epurer/releases/latest \
  | grep download_count
```

---

## 🎯 Bonnes Pratiques

1. **Versioning** : Suivre [Semantic Versioning](https://semver.org)
   - MAJOR.MINOR.PATCH
   - v1.0.0 → v1.0.1 (patch)
   - v1.0.0 → v1.1.0 (minor)
   - v1.0.0 → v2.0.0 (major)

2. **Changelog** : Toujours documenter les changements

3. **Tests** : Tester avant chaque release

4. **Communication** : Annoncer les nouvelles versions

5. **Support** : Répondre aux issues rapidement

---

## 🚀 Promotion du Projet

### Plateformes

- **Reddit** : r/golang, r/macapps, r/programming
- **Hacker News** : https://news.ycombinator.com/submit
- **Product Hunt** : https://www.producthunt.com/
- **Dev.to** : Écrire un article
- **Twitter/X** : Annoncer la release

### Template d'Annonce

```
🧹 Épurer v1.0.0 is out!

Intelligent cache cleanup for macOS developers.

✨ Features:
- Auto-detects 40+ dev tools
- 3 safety levels
- Beautiful CLI
- Can free 50+ GB!

⬇️ Install:
brew tap 0SansNom/tap && brew install epurer

🔗 https://github.com/0SansNom/epurer
```

---

## ✅ Résumé

**Pour v1.0.0** : Utilise **Méthode 1** (Manuelle)
**Pour v1.1.0+** : Utilise **Méthode 2** (GoReleaser)
**Optionnel** : Ajoute **Méthode 3** (Homebrew Tap)

Le projet est maintenant prêt à être partagé avec le monde ! 🌍
