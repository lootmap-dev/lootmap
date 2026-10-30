# lootmap — Proje Şartnamesi

Oct 30, 2026 · @ibo

## Neden bu proje

Araştırma, ilk fikrin (vibe-coding tarayıcısı) artık çok kalabalık olduğunu, gerçek boşluğun ise geliştiricinin kendi bilgisayarında olduğunu gösterdi.

**Dolu alanlar (girmiyoruz)**

- **Vibe-coding tarayıcıları:** "vibeaudit" adı npm'de ve birkaç üründe zaten kullanılıyor. Aynı işi yapan en az 6 açık kaynak araç var: [kelp](https://github.com/maskjelly/kelp), [vibeshield](https://github.com/Anujsuthar004/vibeshield-security), [rlsgate](https://github.com/GerardoRdz96/rlsgate), [vibeproof](https://github.com/humora2504/vibeproof), [vibescan](https://github.com/Armur-Ai/vibescan), [vibe-guard](https://devjosef.github.io/vibe-guard).
- **MCP tarayıcıları:** Snyk ve Cisco gibi büyük şirketlerin açık kaynak araçları alanı dolduruyor ([karşılaştırma](https://reverbico.com/blog/top-mcp-security-tools/)).

**Boşluk: geliştirici bilgisayarındaki ganimet**

- Shai-Hulud npm solucanı kuruluş anında çalışıp `~/.aws/credentials`, `.npmrc`, GitHub ve SSH anahtarlarını topladı; bunun için TruffleHog'u kurbanların kendi ev dizininde çalıştırdı ([MITRE ATT&CK](https://attack.mitre.org/software/S9008/)). İkinci dalgada üç günde 25.000'den fazla geliştirici etkilendi ([Hackmag](https://hackmag.com/news/shai-hulud-2-0)).
- 2026'daki Bitwarden CLI ele geçirmesi aynı dosyaların yanında AI araçlarının ve MCP'nin konfigürasyon dosyalarını da hedef aldı ([JFrog](https://research.jfrog.com/post/bitwarden-cli-hijack/)).
- AI kodlama ajanları okudukları her şeyi diske düz metin olarak kaydediyor; Claude Code oturum kayıtları `~/.claude/projects/` altında varsayılan olarak 30 gün tutuluyor ([eSEOspace](https://eseospace.com/blog/claude-code-secrets-management/)). Bir geliştirici kendi bilgisayarında 62.311 olası gizli anahtar bulduğunu yazdı ([DEV](https://dev.to/beydemirfurkan/i-scanned-my-own-laptop-for-leaked-secrets-and-found-62311-of-them-3gp6)).

**Mevcut araçlar ve eksikleri**

| Araç | Ne yapıyor | Eksik kalan |
| --- | --- | --- |
| [bagel](https://boostsecurityio.github.io/bagel) (Boost Security, GPL-3.0) | İş istasyonu envanteri, JSON rapor | Kurumsal envanter odaklı; düzeltme yok, "ne kaybederim" analizi yok, kötü amaçlı paket kontrolü yok |
| [agentsweep](https://pypi.org/project/agentsweep/) | AI ajan geçmişinde anahtar bulur ve maskeler | Yalnız AI geçmişi; `.aws`, SSH, shell geçmişi yok |
| [ghosttype](https://github.com/xfreed0m/ghosttype) | AI sohbet geçmişinden anahtar çıkarır | Red team aracı; TruffleHog gerektirir, ağırlıklı macOS |
| TruffleHog / Gitleaks | Genel gizli anahtar taraması | Bağlam yok, çok gürültülü, düzeltme yok; saldırganların kullandığı araç |

**Sonuç:** Tamamen boş alan neredeyse yok. Ama "bilgisayarımda bir zararlı çalışsa ne kaybederim, nasıl kapatırım?" sorusunu tek komutla, güzel ve güvenli şekilde cevaplayan bir araç yok. lootmap bu boşluğu hedefliyor.

## Özet

lootmap, bilgisayarınıza bir infostealer ya da Shai-Hulud benzeri bir solucan bulaşsa neleri çalacağını 30 saniyede gösterir ve tek komutla kapatır: `lootmap scan`, ardından `lootmap fix`.

- **Tek cümlelik vaat:** "See your laptop the way malware does. Then fix it."
- **Hedef kitle:** npm/pip ile paket kuran, AI kodlama ajanı kullanan, bilgisayarında bulut, GitHub ve npm anahtarları duran her geliştirici. İkinci kitle: küçük ekiplerdeki güvenlik sorumluları (takımın bilgisayarlarını standart bir kontrolden geçirmek için).
- **Fark yaratan üç şey:**
  1. **Ganimet haritası:** bulguları dosya listesi olarak değil, sonuç olarak gösterir: "Saldırgan şunlara erişir: AWS hesabın (kalıcı anahtar), GitHub (repo yazma), npm (12 paketi yayınlama), 3 production veritabanı."
  2. **Düzeltme:** her bulgu için yedekli, geri alınabilir düzeltme: AI kayıtlarını maskeleme, dosya izinleri, npm install script'lerini kapatma, anahtarı yenileme rehberi.
  3. **Güvenilir araç:** Go ile tek binary, çalışma zamanı bağımlılığı yok, varsayılan olarak ağ erişimi yok, anahtar değerlerini asla yazdırmaz ve saklamaz, imzalı sürümler.
- **Neden paylaşılır:** sonuç ekranı şok edici ve kişiseldir ("laptop'umda 214 anahtar varmış"). İnsanlar ekran görüntüsünü paylaşır, bu da organik yayılmayı sağlar.
- **Başarı ölçütleri:** v0.1 6 hafta içinde; ortalama bir geliştirici bilgisayarında tarama 30 saniyenin altında; gerçek makinelerde manuel incelemeyle yanlış alarm oranı %5'in altında; macOS, Linux ve Windows'ta çalışır.
- **İsim:** "lootmap" npm'de boş (30 Ekim 2026 kontrolü) ve web aramasında bu isimde bir güvenlik aracı çıkmadı. GitHub kullanıcı adı (`lootmap` org), Homebrew formula adı ve alan adı kod yazılmadan önce elle kontrol edilip alınmalı.

## Kapsam

v0.1 üç soruya cevap verir: ne açıkta, bunlarla ne kaybederim, nasıl kapatırım.

**v0.1 — MVP (6 hafta)**

- 12 probe (bkz. Probe ve kontrol seti): bulut, git/GitHub, npm/pip, SSH, shell geçmişi, dağınık `.env` dosyaları, 6 AI ajanının kayıtları ve konfigürasyonları, MCP konfigürasyonları, Docker, kubeconfig.
- Ganimet haritası raporu (terminal), JSON ve Markdown çıktısı, 0–100 skor.
- `lootmap fix`: güvenli düzeltmeler, her zaman önizleme + yedek + `lootmap undo`.
- `lootmap check-packages`: bilgisayardaki tüm lockfile'larda, global npm paketlerinde ve npm önbelleğinde bilinen kötü amaçlı paket sürümlerini arar (OSV `MAL-` kayıtları, çevrimdışı veri paketi).
- macOS, Linux, Windows; Homebrew, Scoop, `curl | sh` yerine imzalı indirme ve `go install`.

**v0.2–v0.3**

- `--verify` (isteğe bağlı, ağ gerektirir): anahtarın hala geçerli olup olmadığını ve yetkilerini sağlayıcıya salt-okunur bir çağrıyla öğrenir (örn. GitHub token kapsamları, npm'de yayınlanabilen paketler, AWS `GetCallerIdentity`).
- `lootmap guard`: arka planda AI ajan kayıtlarını izler, yeni yazılan anahtarları anında maskeler.
- Anahtarı kasaya taşıma: `.env` değerlerini 1Password / macOS Keychain referanslarına çevirme.
- Takım modu: imzalı, değer içermeyen JSON raporlarını toplayıp tek sayfada gösterme.
- Tarayıcı profilleri ve kripto cüzdanları için yalnız **varlık** tespiti (içerik asla okunmaz).

**Kapsam dışı**

- Tarayıcı şifrelerini çözmek, keychain içeriğini okumak: bu aracı bir infostealer'a çevirir.
- Başkasının bilgisayarını uzaktan tarama, herhangi bir veri gönderme.
- Antivirüs / EDR işi (çalışan zararlıyı bulma); lootmap önleyici bir hijyen aracıdır.

## Teknoloji ve mimari

Araç Go ile yazılır ve tek binary olarak dağıtılır. Bunun nedeni basit: anahtarlarınızı okuyan bir aracın kendisi bir tedarik zinciri riski olmamalı. Yüzlerce npm bağımlılığı olan bir "güvenlik aracı", Shai-Hulud'un yayıldığı yolun ta kendisi olurdu.

| Katman | Seçim | Neden |
| --- | --- | --- |
| Dil | Go 1.23+ | Tek binary, üç işletim sistemi, hızlı, bağımlılık yok |
| CLI | `spf13/cobra` | Standart, alt komutlar |
| Terminal arayüzü | `charmbracelet/lipgloss` | Paylaşılabilir, şık rapor ekranı |
| Gizli anahtar desenleri | Kendi desen tablomuz (YAML, binary'e gömülü) + entropi kontrolü | Kontrol bizde; Gitleaks kuralları lisansına uygun şekilde referans alınır |
| Kötü amaçlı paket verisi | OSV `MAL-` kayıtlarından üretilen, imzalı, gömülü veri paketi; `lootmap update-db` ile yenilenir | Varsayılan çevrimdışı çalışma |
| Build / sürüm | GoReleaser, cosign imzası, SLSA provenance, SBOM | Aracın kendisine güvenilebilmesi |
| Test / lint | `go test`, `golangci-lint`, `govulncheck` | Standart Go kalitesi |
| Lisans | Apache-2.0 | Kurumların rahatça kullanabilmesi (bagel GPL-3.0) |

**Temel kavramlar**

- **Probe:** bir yerin nasıl okunacağını bilen modül (örn. `~/.aws/credentials` INI formatı, Claude Code JSONL kayıtları, Cursor SQLite veritabanı). Her probe hangi işletim sisteminde hangi yollara bakacağını bilir.
- **Detector:** bir metin parçasında anahtar türünü tanıyan desen (GitHub PAT, AWS, npm, OpenAI, Anthropic, Stripe, PEM, veritabanı URL'si vb.).
- **Loot (ganimet):** bulgunun "saldırgan bununla ne yapar" karşılığı. Örn. `npm_` token = "paketlerini yayınlayabilir", kalıcı AWS anahtarı = "hesabına erişir, süresiz". Haritalama `loot/catalog.yaml` dosyasında tutulur.
- **Fixer:** bir bulgu türünü güvenle düzelten modül; her zaman önce plan üretir, sonra uygular ve yedek alır.

**Akış:** keşif (hangi araçlar kurulu) → probe'lar paralel çalışır → detector'lar metni tarar → bulgular parmak izine göre tekilleştirilir (aynı anahtar 40 dosyada olsa bile tek "ganimet" olarak görünür, yerleri listelenir) → loot kataloğu ile etki hesaplanır → rapor ve skor.

&#91;embedded content: lootmap akışı · 7 adım\]

Üst sıradaki her adım salt okunurdur; diske yazan tek adım, kullanıcının onayladığı düzeltmedir.

**Değer asla saklanmaz:** bellekte bulunan anahtar yalnız SHA-256 parmak izine ve maskeli önizlemeye (`ghp_••••••a9Xz`) çevrilir; ham değer hiçbir çıktıya, log'a veya önbelleğe yazılmaz.

## Probe ve kontrol seti

v0.1, zararlıların gerçekten okuduğu yerlere bakan 12 probe ile çıkar. Liste, Shai-Hulud ve Bitwarden CLI saldırılarının hedef listesinden türetildi.

| Probe | Nereye bakar (macOS/Linux; Windows karşılıkları kodda) | Ne bulur | Şiddet | Otomatik düzeltme |
| --- | --- | --- | --- | --- |
| `ai-transcripts` | `~/.claude/projects/**/*.jsonl`, `~/.codex/sessions/`, Gemini CLI, Cursor, Windsurf, Copilot Chat geçmişleri | Sohbet ve araç çıktısına sızmış anahtarlar | Kritik | Var: yedekli maskeleme (`[REDACTED:github-pat]`) |
| `ai-config` | `~/.claude.json`, `claude_desktop_config.json`, `~/.cursor/mcp.json`, Windsurf ve VS Code MCP dosyaları | MCP sunucu tanımlarında düz metin token'lar | Yüksek | Kısmi: izinleri 600 yapma; kasaya taşıma rehberi |
| `cloud` | `~/.aws/credentials`, `~/.aws/config`, `~/.config/gcloud/`, `~/.azure/` | Kalıcı AWS erişim anahtarları (SSO yerine), GCP servis hesabı JSON'ları | Kritik | Rehber: SSO'ya geçiş, anahtarı devre dışı bırakma |
| `git` | `~/.git-credentials`, `git config credential.helper`, `~/.config/gh/hosts.yml` | `store` helper ile düz metin şifreler, klasik PAT'ler | Kritik | Var: helper'ı osxkeychain / libsecret / manager'a çevirme |
| `npm` | `~/.npmrc`, proje `.npmrc`'leri, `~/.pypirc` | Yayın token'ları; `ignore-scripts` kapalı olması | Kritik | Var: `ignore-scripts=true` önerisi; token için rehber |
| `ssh` | `~/.ssh/` | Parolasız özel anahtarlar, `ForwardAgent yes`, gevşek izinler | Yüksek | Var: izinler; parola ekleme komutu önerisi |
| `shell-history` | bash, zsh, fish, PowerShell geçmişleri | Komut satırına yazılmış token ve şifreler | Yüksek | Var: yedekli maskeleme; `HISTIGNORE` önerisi |
| `shell-rc` | `~/.bashrc`, `~/.zshrc`, `~/.profile`, `config.fish` | `export OPENAI_API_KEY=...` gibi kalıcı anahtarlar | Yüksek | Rehber: kasa referansına taşıma |
| `dotenv` | Geliştirme klasörleri (varsayılan: `~/code`, `~/projects`, `~/dev`, `~/src`, `~/Documents/GitHub`; config ile değişir), derinlik sınırlı | Dağınık `.env*` dosyalarında production anahtarları ve veritabanı URL'leri | Yüksek | Rehber |
| `containers` | `~/.docker/config.json`, `~/.kube/config` | `credsStore` olmadan base64 kayıtlı registry şifreleri; kubeconfig içindeki token ve istemci anahtarları | Yüksek | Kısmi: izinler |
| `iac` | `~/.terraform.d/credentials.tfrc.json` ve benzerleri | Terraform Cloud ve benzeri token'lar | Orta | Rehber |
| `packages` | Geliştirme klasörlerindeki lockfile'lar, global npm kökü, npm önbelleği | OSV'de `MAL-` olarak işaretli paket sürümleri | Kritik | Rehber: olay müdahale adımları |

**Kurallar**

- Her probe yalnız okur; düzeltme ayrı komuttur.
- Yollar her araç için gerçek kurulumlarla doğrulanmadan koda girmez; AI araçlarının dosya düzeni sürümle değiştiği için yollar `probes/paths.yaml` dosyasında tutulur ve katılımcılar günceller.
- Bir dosya 50 MB'tan büyükse parça parça akış olarak okunur; ikili dosyalar atlanır.
- `.env.example`, test fixture'ları, `xxx`/`changeme` gibi yer tutucular ve çevre değişkeni referansları (`${TOKEN}`) bulgu sayılmaz.

## CLI tasarımı ve çıktı

Raporun ilk ekranı dosya listesi değil, "ne kaybedersin" özetidir; ayrıntılar altında gelir.

| Komut | Ne yapar |
| --- | --- |
| `lootmap scan` | Tam tarama, ganimet haritası raporu |
| `lootmap scan --probe ai-transcripts,npm` | Yalnız seçili probe'lar |
| `lootmap fix` | Düzeltme planını gösterir, onay alır, yedekleyip uygular |
| `lootmap fix --dry-run` | Yalnız planı gösterir |
| `lootmap undo [id]` | Son ya da belirli düzeltmeyi yedekten geri alır |
| `lootmap check-packages` | Kötü amaçlı paket kontrolü |
| `lootmap update-db` | OSV veri paketini indirir ve imzasını doğrular (tek ağ kullanan komut) |
| `lootmap explain <bulgu-id>` | Bulgunun nedenini ve düzeltme adımlarını anlatır |
| `lootmap doctor` | Hangi araçların kurulu olduğunu ve hangi probe'ların çalışacağını gösterir |

**Bayraklar:** `--format pretty|json|markdown` · `--output <dosya>` · `--roots <yol,yol>` (geliştirme klasörleri) · `--min-severity` · `--no-color` · `--strict` (bulgu varsa çıkış kodu 1).

**Hedef ekran (İngilizce olacak; okunabilirlik için Türkçe)**

```text
lootmap v0.1.0 · macOS 15 · 12 probe · 18.402 dosya · 21 sn · ağ erişimi: YOK

 BU BİLGİSAYARDA BİR ZARARLI ÇALIŞSA ŞUNLARI ALIR:

  ● AWS hesabı           kalıcı erişim anahtarı · süresiz       ~/.aws/credentials
  ● GitHub              klasik PAT · 3 yerde                 ~/.git-credentials +2
  ● npm                 yayın token'ı · paketlerini yayınlar   ~/.npmrc
  ● OpenAI, Anthropic   7 anahtar · AI ajan kayıtlarında      ~/.claude/projects (41 dosya)
  ● Postgres            3 bağlantı adresi · şifre içeriyor     4 .env dosyası
  ○ SSH                 parolasız anahtar                    ~/.ssh/id_ed25519

 Skor: 20/100 (F) · 6 kritik · 4 yüksek · 2 orta

 9 bulgu otomatik düzeltilebilir → lootmap fix
 5 anahtar yenilenmeli (rotate) → lootmap explain --rotate
```

**Skor:** 100 − (kritik ganimet × 10 + yüksek × 4 + orta × 2 + düşük × 1); not A ≥ 90, B ≥ 75, C ≥ 60, D ≥ 40, F < 40, en az 0. Skor dosya sayısına değil, tekilleştirilmiş **ganimete** göre hesaplanır; aynı anahtar 40 dosyada olsa bile bir kez sayılır.

**JSON çıktısı (şema özeti)**

```json
{
  "tool": "lootmap",
  "version": "0.1.0",
  "host": { "os": "darwin", "hostnameHash": "sha256:…" },
  "score": 20,
  "loot": [
    {
      "id": "L-7f3a",
      "kind": "github-pat-classic",
      "impact": "GitHub hesabına yazma erişimi",
      "severity": "critical",
      "fingerprint": "sha256:a1b2…",
      "preview": "ghp_••••••a9Xz",
      "locations": [
        { "probe": "git", "path": "~/.git-credentials", "line": 1 },
        { "probe": "ai-transcripts", "path": "~/.claude/projects/app/3c1e.jsonl", "line": 812 }
      ],
      "fixes": ["redact-transcripts", "git-credential-helper"],
      "rotate": "https://github.com/settings/tokens"
    }
  ]
}
```

JSON hiçbir zaman ham değer, tam hostname veya kullanıcı adı içermez; yollar `~` ile kısaltılır. Böylece rapor bir güvenlik ekibiyle güvenle paylaşılabilir.

## Düzeltme motoru ve güvenlik ilkeleri

`lootmap fix` hiçbir şeyi sormadan değiştirmez ve yaptığı her şey geri alınabilir.

**Düzeltme akışı**

1. Plan: hangi dosyada ne değişecek, satır satır önizleme.
2. Onay: her düzeltme grubu için ayrı onay (`--yes` ile toplu).
3. Yedek: değişen her dosya `~/.lootmap/backups/<zaman>/` altına izinleri 600 olarak kopyalanır. Yedekler maskelenmemiş veri içerdiği için 7 gün sonra otomatik silinir ve bu kullanıcıya söylenir.
4. Uygulama: dosya geçici kopyaya yazılır, sonra atömik olarak yer değiştirilir; yarım kalmış dosya oluşmaz.
5. Doğrulama: düzeltilen probe yeniden çalıştırılır ve bulgunun kaybolduğu gösterilir.

**v0.1 otomatik düzeltmeleri:** AI kayıtlarında ve shell geçmişinde maskeleme · kimlik bilgisi dosyalarının izinlerini 600 yapma · git `credential.helper store`'u işletim sisteminin kasasına çevirme · `~/.npmrc`'ye `ignore-scripts=true` ekleme (uyarıyla: bazı paketlerin kurulum script'i gerekir) · SSH `ForwardAgent` kapatma.

**Asla otomatik yapılmayanlar:** anahtar silmek veya yenilemek (rotate). Bunlar kullanıcının hesabında yapılır; lootmap doğrudan ilgili ayar sayfasının bağlantısını ve adımları verir.

**Canlı oturum koruması:** son 60 saniyede değişmiş AI kayıt dosyalarına ve ilgili ajan süreci çalışırken o ajanın kayıtlarına dokunulmaz (`--force` hariç).

**Aracın kendi güvenlik ilkeleri (kodda zorunlu)**

- Varsayılan olarak hiçbir ağ bağlantısı yok. Ağ yalnız `update-db` ve ileride `--verify` ile, açıkça istenince kullanılır. Güncelleme kontrolü, telemetri, analitik yok.
- Ham anahtar değeri bellekte en kısa süre tutulur; hiçbir çıktıya, log'a, panik mesajına yazılmaz.
- Tarayıcı şifre veritabanları, keychain içeriği ve cüzdan dosyaları okunmaz.
- `sudo` / yönetici yetkisi istemez; yalnız kullanıcının kendi dosyalarına bakar.
- Sürümler cosign ile imzalıdır, SBOM ve SLSA provenance eklenir; README'de imza doğrulama komutu bulunur.
- `SECURITY.md` ile sorumlu ifşa süreci ve tehdit modeli (`THREAT-MODEL.md`) yayımlanır.

## Proje yapısı ve arayüzler

Yeni bir AI aracı çıktığında katılımcı yalnız bir probe dosyası ve bir YAML satırı ekleyerek destek getirebilmeli.

```text
lootmap/
├─ cmd/lootmap/main.go
├─ internal/
│  ├─ cli/            # cobra komutları: scan, fix, undo, check-packages, explain, doctor
│  ├─ engine/         # probe'ları paralel çalıştırma, tekilleştirme, skor
│  ├─ probes/         # her probe ayrı dosya: aitranscripts.go, cloud.go, git.go …
│  │  └─ paths.yaml   # işletim sistemi bazında yollar
│  ├─ detect/         # detector'lar + patterns.yaml + entropi
│  ├─ loot/           # catalog.yaml: anahtar türü → etki cümlesi, şiddet, rotate bağlantısı
│  ├─ fix/            # fixer'lar, yedekleme, atömik yazma, undo
│  ├─ osv/            # gömülü MAL veri paketi, lockfile ayrıştırıcıları
│  ├─ secret/         # maskeleme, parmak izi; ham değeri taşıyan tek tür
│  └─ report/         # pretty (lipgloss), json, markdown
├─ testdata/homes/    # sahte ev dizinleri: mac-dev, linux-dev, windows-dev, clean
├─ docs/              # probe ve fix belgeleri, tehdit modeli
├─ .goreleaser.yaml
└─ README.md · SECURITY.md · THREAT-MODEL.md · CONTRIBUTING.md · LICENSE
```

**Temel arayüzler**

```go
type Probe interface {
    ID() string                         // "ai-transcripts"
    Available(env Env) bool              // araç kurulu mu, yol var mı
    Scan(ctx context.Context, env Env, emit func(Chunk)) error
}

// Chunk: taranacak metin parçası ve nereden geldiği.
type Chunk struct {
    Probe string
    Path  string // ~ ile kısaltılmış
    Line  int
    Text  string
}

type Detector interface {
    ID() string                         // "github-pat-classic"
    Find(c Chunk) []secret.Match         // Match ham değeri dışa açmaz
}

type Fixer interface {
    ID() string                         // "redact-transcripts"
    Plan(f []Finding) (Plan, error)     // değişiklikleri önizler, dosyaya dokunmaz
    Apply(p Plan, b Backup) error       // yedekler, atömik yazar
}
```

**Kod standartları**

- `Env` arayüzü ev dizinini ve işletim sistemini soyutlar; testler gerçek ev dizinine asla dokunmaz.
- `secret.Value` türünün `String()`, `MarshalJSON()` ve `Format()` metotları her zaman maskeli çıktı verir; böylece yanlışlıkla log'a yazmak imkânsız hale gelir.
- Doğrudan bağımlılık sayısı 10'un altında tutulur; `net/http` yalnız `osv` paketinde import edilebilir (CI'da kontrol edilir).

## Test stratejisi

Bu araç insanların en hassas dosyalarına dokunduğu için iki şeyi kanıtlamak zorunda: doğru bulur ve hiçbir şeyi bozmaz.

- **Sahte ev dizinleri:** `testdata/homes/` altında macOS, Linux ve Windows düzeninde, sahte ama gerçekçi anahtarlarla dolu ev dizinleri. Her probe için beklenen bulgular golden dosyalarda tutulur. Bir de `clean` ev dizini var: burada sıfır bulgu çıkmalı.
- **Sahte anahtarlar:** test anahtarları çalışma anında üretilir; repoya gerçek formatta anahtar commit edilmez (GitHub push protection'ı tetiklememek için).
- **Yanlış alarm testleri:** yer tutucular, `${VAR}` referansları, belgelerdeki örnek anahtarlar (AWS'in resmi `AKIAIOSFODNN7EXAMPLE` örneği gibi), base64 görseller, `.env.example`. Hiçbiri bulgu üretmemeli.
- **Düzeltme testleri:** her fixer için plan → uygula → doğrula → undo döngüsü; undo sonrası dosya bayt bayt orijinaliyle aynı olmalı. Yarıda kesilen yazma simülasyonunda orijinal dosya bozulmamalı.
- **Sızıntı testi:** tüm test anahtarları tarama sonrası stdout, stderr, JSON, Markdown ve log çıktılarında aranır; tek bir eşleşme testi patlatır.
- **Ağ testi:** `scan`, `fix` ve `check-packages` ağ erişimi kapalı bir sandbox'ta çalıştırılır; herhangi bir bağlantı denemesi testi patlatır.
- **Fuzz:** JSONL, INI, YAML ayrıştırıcıları Go'nun yerleşik fuzz testleriyle; bozuk dosya aracı çökertmemeli.
- **Performans:** 2 GB AI kaydı ve 20.000 dosya içeren sahte ev dizini 30 saniyenin altında taranmalı.
- **Gerçek makine doğrulaması:** lansmandan önce en az 10 gönüllünün gerçek bilgisayarında çalıştırılır; yalnız değer içermeyen JSON paylaşılır ve yanlış alarmlar elle sınıflandırılır.
- **CI:** macOS, Ubuntu, Windows matrisi; `go test -race`, `golangci-lint`, `govulncheck`, GoReleaser snapshot build'i.

## README ve lansman

Lansmanı bir sonraki büyük tedarik zinciri olayına hazır bekletmek, yıldız kazanmanın en güçlü yoludur. Bu tür olaylar son bir yılda birkaç ayda bir tekrarlandı ve her seferinde insanlar "ben etkilendim mi?" diye araç aranıyor.

**README sırası (İngilizce)**

1. Tek cümle: "See your laptop the way malware does. Then fix it."
2. 15 saniyelik GIF: `lootmap scan` çalışıyor, ganimet haritası açılıyor, `lootmap fix` sonrası skor F'den B'ye çıkıyor.
3. Kurulum: `brew install lootmap/tap/lootmap`, Scoop, imzalı binary, `go install`. İmza doğrulama komutu hemen altında.
4. "Why": Shai-Hulud ve Bitwarden CLI saldırılarının ne topladığını kaynaklı olarak anlatan 3 satır.
5. Gizlilik kutusu: ağ yok, telemetri yok, değer saklanmaz, sudo yok. Bu kutu güvenin temelidir.
6. Neye bakar tablosu (12 probe) ve desteklenen AI araçları rozet listesi.
7. bagel, TruffleHog ve agentsweep ile dürüst kıyaslama.
8. "Add a new AI tool in 20 minutes" katılım rehberi.

**GitHub topics:** `security`, `secrets`, `infostealer`, `supply-chain-security`, `developer-tools`, `ai-agents`, `claude-code`, `cursor`, `npm`, `cli`, `golang`.

**Lansman planı**

1. **Hazırlık (lansmandan 2 hafta önce):** 10 gönüllünün değer içermeyen raporlarından anonim istatistik çıkar (örn. "10 geliştiricinin 8'inde AI kayıtlarında canlı anahtar vardı"). Blog yazısını bu sayılarla hazırla.
2. **Lansman günü (Salı veya Çarşamba, Türkiye saatiyle 16:00–17:00):** Show HN başlığı önerisi: "Show HN: lootmap – see what an infostealer would take from your laptop". Aynı gün r/netsec, r/programming, r/ClaudeAI, r/cursor, X, Bluesky, Mastodon infosec. İlk 6 saat tüm yorumlara cevap.
3. **Olay anı planı:** bir sonraki npm/PyPI saldırısında 24 saat içinde ilgili paketleri veri paketine ekle, `lootmap check-packages` ile "etkilendim mi?" kontrolünü duyur. Bu an, araç için en büyük büyüme fırsatıdır.
4. **İlk 8 hafta:** iki haftada bir sürüm; her yeni AI aracı desteği ayrı duyuru; `good first issue` olarak 15 probe/detector fikri.
5. **Güven ve görünürlük:** OpenSSF Scorecard rozeti, awesome-security listelerine PR, güvenlik bültenlerine (örn. tl;dr sec) tanıtım e-postası.

## Antigravity ile uygulama adımları

Bu belgeyi Markdown olarak dışa aktarıp repo köküne `SPEC.md` adıyla koyun. Antigravity'ye her seferinde tek bir aşama verin ve kabul kriterleri geçmeden ilerlemeyin.

**İlk mesaj (bir kez, aynen yapıştırın)**

```text
Read SPEC.md fully. It is the source of truth for this project (lootmap, Go).
We build it phase by phase. Rules for every phase:
- Implement ONLY the phase I name. Do not start later phases.
- Tests first (go test, table-driven, golden files), then code. All must pass.
- NEVER read or modify my real home directory. Tests use testdata/homes only,
  through the Env interface.
- No network calls anywhere except internal/osv update code.
- Raw secret values must only live in internal/secret.Value, which always
  prints masked. Never log, print or serialize a raw secret.
- Never commit real-format secrets; generate fake ones at test time.
- Ask me before adding any dependency not listed in SPEC.md.
- At the end of each phase show: files changed, test output, how to try it.
Start with Phase 0.
```

**Aşamalar ve kabul kriterleri**

1. **Aşama 0 — İskelet:** Go modülü, cobra CLI, `Env` soyutlaması, golangci-lint, CI matrisi, GoReleaser snapshot. *Kabul:* üç işletim sisteminde `lootmap --help` ve `lootmap doctor` çalışır.
2. **Aşama 1 — Çekirdek:** `secret` paketi (maskeleme, parmak izi), detector'lar (`patterns.yaml`), engine (paralel probe, tekilleştirme), JSON raporu. *Kabul:* sızıntı testi geçer; aynı anahtar 3 dosyada tek ganimet olarak raporlanır.
3. **Aşama 2 — Klasik probe'lar:** `cloud`, `git`, `npm`, `ssh`, `shell-history`, `shell-rc`. *Kabul:* golden testler geçer; `clean` ev dizininde sıfır bulgu.
4. **Aşama 3 — AI probe'ları:** `ai-transcripts` (Claude Code ve Codex ile başla, sonra Gemini CLI, Cursor, Windsurf, Copilot Chat) ve `ai-config`. Her aracın gerçek dosya düzenini önce benimle doğrula. *Kabul:* 2 GB'lık sahte kayıt 30 saniyenin altında taranır.
5. **Aşama 4 — Kalan probe'lar:** `dotenv`, `containers`, `iac`. *Kabul:* derinlik sınırı ve `--roots` çalışır; `node_modules` ve `.git` atlanır.
6. **Aşama 5 — Ganimet haritası ve rapor:** `loot/catalog.yaml`, skor, lipgloss ekranı, Markdown çıktısı, `explain`. *Kabul:* çıktı "CLI tasarımı" bölümündeki hedef ekrana benzer; 80 sütunda bozulmaz.
7. **Aşama 6 — Düzeltme motoru:** yedek, atömik yazma, `fix`, `fix --dry-run`, `undo`, canlı oturum koruması, v0.1 fixer'ları. *Kabul:* her fixer için undo sonrası dosya bayt bayt aynı.
8. **Aşama 7 — Kötü amaçlı paket kontrolü:** OSV veri paketi üretme script'i, imza doğrulama, lockfile ayrıştırıcıları (npm, pnpm, yarn), `check-packages`, `update-db`. *Kabul:* bilinen bir `MAL-` sürümü içeren sahte lockfile yakalanır.
9. **Aşama 8 — Dağıtım:** GoReleaser + cosign + SBOM + SLSA, Homebrew tap, Scoop, README, SECURITY.md, THREAT-MODEL.md. *Kabul:* temiz bir Mac'te `brew install` ile kurulup çalışır; imza doğrulama komutu README'deki gibi çalışır.

**Dikkat edilecekler**

- Antigravity'yi kendi gerçek bilgisayarınızda bu repo üzerinde çalıştırırken, ajanın testleri gerçek ev dizininizde çalıştırmadığından emin olun. İlk gerçek taramayı kendiniz, aşama 6 bittikten sonra ve önce `--dry-run` ile yapın.
- Ajanlar testi geçirmek için testi gevşetmeye meyillidir. Özellikle sızıntı ve ağ testlerinde yapılan değişiklikleri her zaman kendiniz okuyun.
- AI araçlarının dosya yollarını ajanın hafızasından değil, gerçek kurulumdan doğrulayın.
- Her aşama sonunda commit atın.

## Kaynaklar

- [MITRE ATT&CK — Shai-Hulud (S9008)](https://attack.mitre.org/software/S9008/)
- [Hackmag — Shai-Hulud 2.0](https://hackmag.com/news/shai-hulud-2-0)
- [JFrog — Bitwarden CLI hijack](https://research.jfrog.com/post/bitwarden-cli-hijack/)
- [Netwrix — AI kodlama asistanlarının kimlik bilgisi saklama riskleri](https://netwrix.com/en/resources/blog/ai-coding-assistant-credential-storage-risks/)
- [Security Boulevard — AI coding agents are leaking credentials](https://securityboulevard.com/2026/09/ai-coding-agents-are-leaking-credentials-cursor-claude-code-copilot-and-mcp/)
- [eSEOspace — Claude Code secrets management](https://eseospace.com/blog/claude-code-secrets-management/)
- [DEV — I scanned my own laptop and found 62,311 secrets](https://dev.to/beydemirfurkan/i-scanned-my-own-laptop-for-leaked-secrets-and-found-62311-of-them-3gp6)
- [OWASP DevSecOps — Developer workstation security](https://owasp.github.io/DevSecOpsGuideline/2-Process/2-2-Develop/2-2-4-Developer-Workstation-and-Dev-Container-Security/)
- [Zeltser — Securing API keys on your workstation](https://zeltser.com/securing-api-keys-on-your-workstation)
- Rakipler: [bagel](https://boostsecurityio.github.io/bagel) · [agentsweep](https://pypi.org/project/agentsweep/) · [ghosttype](https://github.com/xfreed0m/ghosttype) · [kelp](https://github.com/maskjelly/kelp) · [vibeshield](https://github.com/Anujsuthar004/vibeshield-security) · [rlsgate](https://github.com/GerardoRdz96/rlsgate) · [vibeproof](https://github.com/humora2504/vibeproof) · [vibescan](https://github.com/Armur-Ai/vibescan)
- ["vibeaudit" npm paketi (isim çakışması)](https://www.pkgstats.com/pkg:@jackdog668/vibeaudit)
