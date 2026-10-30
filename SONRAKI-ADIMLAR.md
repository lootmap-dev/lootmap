# SONRAKİ ADIMLAR — Başlangıç Seviyesi İçin Adım Adım Yayın ve İlk Çalıştırma Rehberi

Bu rehber, bilgisayarında geliştirdiğimiz **lootmap** projesini GitHub'a yüklemen, otomatik testleri (CI) görmen, Homebrew (macOS/Linux) ve Scoop (Windows) paket depolarını açman, **v0.1.0** sürümünü yayınlaman ve kendi bilgisayarında **güvenle** ilk kez denemen için hazırlandı.

---

## Bölüm 1: Kendi Bilgisayarında Güvenli İlk Deneme (Hiçbir Dosyayı Değiştirmeden)

`lootmap` varsayılan olarak **salt okunurdur** (yalnızca `lootmap fix --yes` dersen ve onaylarsan dosyaları yedekleyip düzeltir). Yine de ilk denemeyi şu 3 güvenli adımla yap:

### Adım 1.1 — Binary'yi derle
PowerShell'de proje klasöründeyken (`C:\Users\ıbrahım\Desktop\lootmap`) şu komutu çalıştır:

```powershell
go build -o lootmap.exe ./cmd/lootmap
```

### Adım 1.2 — `doctor` komutunu çalıştır (Hiçbir dosyayı okumaz)
Hangi işletim sisteminde olduğunu ve hangi probe'ların bilgisayarında mevcut olduğunu gösterir:

```powershell
.\lootmap.exe doctor
```

### Adım 1.3 — `scan` komutunu çalıştır (Salt okunur tarama)
Bilgisayarındaki açık anahtarları ve ganimet haritasını gösterir. **Hiçbir dosyayı değiştirmez ve hiçbir anahtarın gerçek değerini ekrana basmaz** (yalnızca `ghp_••••••a9Xz` gibi maskeli gösterir):

```powershell
.\lootmap.exe scan
```

İstersen sonucu JSON veya Markdown olarak da görebilirsin:

```powershell
.\lootmap.exe scan --format json
.\lootmap.exe scan --format markdown
```

### Adım 1.4 — `fix --dry-run` ile düzeltme planını önizle (Hiçbir dosyaya dokunmaz)
`--dry-run` bayrağı, `lootmap fix` çalıştırırsan hangi dosyada neyin değişeceğini satır satır listeler ama **hiçbir dosyaya yazmaz**:

```powershell
.\lootmap.exe fix --dry-run
```

> **Not:** Gerçekten düzeltme yapmak istediğinde bile `lootmap` değiştirdiği her dosyanın orijinalini `~/.lootmap/backups/<tarih>/` altına yedekler ve `.\lootmap.exe undo` komutuyla tek saniyede bayt bayt geri alır.

---

## Bölüm 2: `# UNVERIFIED` Olarak İşaretlenen ve Kontrol Etmen Gereken Yollar

`internal/probes/paths.yaml` dosyasında bazı AI araçlarının yolları, resmi belgelerden veya yaygın kurulumlardan alındı ancak gerçek kurulumda doğrulanması için `# UNVERIFIED` yorumuyla işaretlendi. Bilgisayarında bu araçlardan kurulu olan varsa bu klasörlerin var olup olmadığını kontrol et:

### 1. `ai-transcripts` altındaki `# UNVERIFIED` yollar (`internal/probes/paths.yaml`)
- **macOS (`darwin`):**
  - `.gemini/tmp`
  - `Library/Application Support/Windsurf/User/globalStorage`
  - `.codeium/windsurf`
  - `Library/Application Support/Code/User/globalStorage/github.copilot-chat`
- **Linux (`linux`):**
  - `.gemini/tmp`
  - `.config/Windsurf/User/globalStorage`
  - `.codeium/windsurf`
  - `.config/Code/User/globalStorage/github.copilot-chat`
- **Windows (`windows`):**
  - `.gemini/tmp` (PowerShell'de kontrol: `Test-Path "$env:USERPROFILE\.gemini\tmp"`)
  - `AppData/Roaming/Windsurf/User/globalStorage` (Kontrol: `Test-Path "$env:APPDATA\Windsurf\User\globalStorage"`)
  - `.codeium/windsurf` (Kontrol: `Test-Path "$env:USERPROFILE\.codeium\windsurf"`)
  - `AppData/Roaming/Code/User/globalStorage/github.copilot-chat` (Kontrol: `Test-Path "$env:APPDATA\Code\User\globalStorage\github.copilot-chat"`)

### 2. `ai-config` altındaki `# UNVERIFIED` yollar (`internal/probes/paths.yaml`)
- **macOS / Linux / Windows ortak:**
  - `.codex/auth.json` (Kontrol: `Test-Path "$env:USERPROFILE\.codex\auth.json"`)
  - `.codeium/windsurf/mcp_config.json` (Kontrol: `Test-Path "$env:USERPROFILE\.codeium\windsurf\mcp_config.json"`)

Doğruladığın satırların yanındaki `# UNVERIFIED` yorumunu silebilirsin.

---

## Bölüm 3: GitHub Organizasyonu ve Repo Oluşturma (`lootmap-dev/lootmap`)

### Adım 3.1 — GitHub'da `lootmap-dev` organizasyonunu aç
1. Tarayıcıda [https://github.com/account/organizations/new](https://github.com/account/organizations/new) adresine git.
2. **Free** (Ücretsiz) planı seç.
3. Organizasyon adına **`lootmap-dev`** yaz ve oluştur.

### Adım 3.2 — `lootmap` reposunu oluştur ve kodu gönder (Push)
Eğer bilgisayarında GitHub CLI (`gh`) kuruluysa PowerShell'de şu komutları çalıştır:

```powershell
git branch -M main
gh repo create lootmap-dev/lootmap --public --description "See your laptop the way malware does. Then fix it." --source=. --remote=origin --push
```

**Eğer `gh` kurulu değilse (Tarayıcıdan):**
1. [https://github.com/organizations/lootmap-dev/repositories/new](https://github.com/organizations/lootmap-dev/repositories/new) adresine git.
2. Repository name kısmına **`lootmap`** yaz, **Public** seç, *"Add a README"* kutularını **işaretleme** (boş bırak) ve **Create repository** düğmesine bas.
3. PowerShell'de şu komutları çalıştır:

```powershell
git branch -M main
git remote add origin https://github.com/lootmap-dev/lootmap.git
git push -u origin main
```

### Adım 3.3 — CI (Otomatik Testler) Ekranını Kontrol Et
1. Tarayıcıda `https://github.com/lootmap-dev/lootmap/actions` adresine git.
2. **`ci`** iş akışının (Ubuntu, macOS, Windows testleri, Linux ağsız sandbox, `golangci-lint`, `govulncheck` ve GoReleaser snapshot) yeşil tik (✔) aldığını gör.

---

## Bölüm 4: Homebrew ve Scoop Paket Depolarını Hazırlama

GoReleaser yeni bir sürüm yayınladığında macOS/Linux için `homebrew-tap`, Windows için `scoop-bucket` depolarına otomatik olarak kurulum dosyası yazar.

### Adım 4.1 — İki boş repo oluştur
Tarayıcıdan (veya `gh` ile) `lootmap-dev` organizasyonu altında şu iki **Public** repoyu (**Add a README file** kutusunu işaretleyerek, yani içinde `main` dalı hazır olarak) oluştur:
1. **`lootmap-dev/homebrew-tap`**
2. **`lootmap-dev/scoop-bucket`**

(`gh` kullanıyorsan):
```powershell
gh repo create lootmap-dev/homebrew-tap --public --add-readme
gh repo create lootmap-dev/scoop-bucket --public --add-readme
```

### Adım 4.2 — GoReleaser için Personal Access Token (PAT) ekle
1. GitHub'da **Settings → Developer settings → Personal access tokens → Fine-grained tokens → Generate new token** sayfasına git.
2. **Resource owner** olarak `lootmap-dev` seç.
3. **Only select repositories** diyerek `homebrew-tap` ve `scoop-bucket` repolarını seç.
4. **Repository permissions → Contents** iznini **Read and write** yap ve token'ı üret.
5. Şimdi `https://github.com/lootmap-dev/lootmap/settings/secrets/actions` sayfasına git ve **New repository secret** ile şu iki secret'ı ekle (ikisine de aynı token'ı yapıştırabilirsin):
   - `HOMEBREW_TAP_GITHUB_TOKEN`
   - `SCOOP_BUCKET_GITHUB_TOKEN`

---

## Bölüm 5: `v0.1.0` Sürümünü Etiketleme ve Yayınlama (Release)

Her şey yeşil olduğunda ilk resmi sürümü çıkarmak için yalnızca bir Git etiketi (tag) gönderirsin; GitHub Actions gerisini (6 platform için derleme, SBOM, Cosign imzası, SLSA provenance, Homebrew ve Scoop güncellemesi) otomatik yapar:

```powershell
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

Ardından:
1. `https://github.com/lootmap-dev/lootmap/actions` sayfasında **`release`** iş akışının tamamlanmasını izle.
2. `https://github.com/lootmap-dev/lootmap/releases` sayfasında imzalı `v0.1.0` sürümünün yayınlandığını gör!
