# DocCtl – Föreningsarkiv som Kod

DocCtl är ett plattformsoberoende verksamhetssystem för ideella föreningar som
applicerar "Document as Code" på formell föreningsstyrning. Syftet är att
säkerställa arkivbeständighet och juridisk försegling utan dyra licenskostnader.

**🤖 Transparens angående kodbasen:** För att nå denna MVP har artificiell
intelligens använts för att generera majoriteten av koden. Kärnfokus har legat
på att bevisa konceptet, kryptografin och arkitekturen. Koden fungerar, men
refaktoriseringsbehov finns absolut!

---

## 💡 Varför DocCtl?

Ideell sektor blöder pengar på prenumerationstjänster för e-signering, och
historiska beslut går förlorade i trasiga Word-dokument och raderade
molnmappar. DocCtl löser detta genom att:

* Tvinga fram en "Single Source of Truth" i rena textfiler (Markdown).
* Generera deterministiska PDF/A-2B för långtidsarkivering (via Typst).
* Generera HTML och DOCX för tillgänglighet.
* Ersätta dyra e-signaturer med en decentraliserad, matematisk beviskedja.

---

## 🪆 Säkerhetsmodell: "The Russian Doll"

Dokument förseglas i fyra lager vilket gör manipulation i efterhand praktiskt
omöjlig:

1. **Integritet** — SHA-256-hashning av alla filer i `arkiv/`.
2. **Manifest** — `ATTESTATION.md` binder samman filerna och deras hashar.
3. **Signatur** — `ATTESTATION.md.sig` (EdDSA/GPG) + `PUBLIC_KEY.asc` för verifiering.
4. **Tidsbevis** — `ATTESTATION.md.sig.ots` via OpenTimestamps mot Bitcoins blockkedja *(valfritt)*.

---

## 🔧 Krav

### Obligatoriska
- `go` (1.21+)
- `git`
- `typst`
- `pandoc`

### Valfria
- `opentimestamps-client` (`ots`) — För blockkedjetidsstämpling

---

## 🚀 Kom igång

### 1. Initiera workspace

```bash
./docctl init
```

Skapar `.tooling/` med mallar, `.gitignore` och kör `git init`.

### 2. Starta det interaktiva gränssnittet

```bash
./docctl
```

Vid första körningen guidar systemet dig igenom uppstarten automatiskt.

### Huvudmeny (TUI)

| Val | Funktion |
|---|---|
| ✨ Skapa dokument / möte | Skapa nytt protokoll, styrdokument eller annan grundakt |
| 🏗️ Bygg dokument | Kompilera Markdown → PDF, HTML, DOCX |
| 🔒 Försegla arkiv | Signera och försegla ett byggt dokument |
| ⚙️ Inställningar | Hantera föreningar, nycklar och inställningar |
| ❌ Avsluta | Stäng programmet |

### Typiskt arbetsflöde

```
Skapa (Init) → Redigera Markdown → Bygg (Build) → Försegla (Seal)
```

### Konfigurationsfil

Systemets konfiguration lagras i `.tooling/config.yaml`:

```yaml
foreningar:
  MIFF:
    namn: "Minsta Möjliga Förening"
    org_nummer: "802500-0001"
    organ:
      - styrelsen
      - årsmöte
settings:
  create_zip: true           # Skapa ZIP-arkiv (AIP) vid försegling
  use_open_time_stamps: false # Använd blockkedjetidsstämpling
```

---

## 💻 CLI-kommandon

```bash
# Starta interaktivt gränssnitt
./docctl

# Initiera nytt workspace i nuvarande mapp
./docctl init

# Bygg dokument (konvertera Markdown till PDF/HTML/DOCX)
./docctl build [org-id] [sökväg-till-källor]
# Flaggor:
#   -f, --force   Tvinga ombyggnad även om arkivet är förseglat

# Försegla ett byggt dokument
./docctl seal [sökväg-till-källor]
# Flaggor:
#   -k, --key-file  (obligatorisk) Sökväg till krypterad nyckelfil (.asc)
#   -p, --pin       (obligatorisk) PIN-kod för nyckeln
#   -f, --force     Tvinga omförsegling

# Skapa en ny förening
./docctl org add [id] [namn] [organisationsnummer]

# Återställ standardmallar till inbyggd version
./docctl update-templates
```

---

## 📄 Dokumenttyper

Alla dokument skrivs i **Markdown med YAML-frontmatter** och lagras under `källor/`.

### Protokoll

Lagras i: `{OrgID}/Årsakter/{år}/{organ}/{datum}/källor/protokoll.md`

```markdown
---
typ: protokoll
title: Protokoll styrelsen
datum: 2025-05-05
tid: 18:00
plats: Föreningslokalen
ordforande: Namn Namnsson
sekreterare: Namn Namnsson
justerare:
  - Justerare 1
  - Justerare 2
---

## Mötets öppnande
```

### Styrdokument / Policy

Lagras i: `{OrgID}/Grundakter/Styrdokument/{kategori}/{namn}/källor/document.md`

```markdown
---
typ: policy
title: Kommunikationspolicy
version: 1.0
antagen: 2025-01-15 av Styrelsen
---

## 1. Syfte
```

### Andra grundakter

Lagras i: `{OrgID}/Grundakter/{kategori}/{namn}/källor/document.md`

Används för t.ex. avtal, motioner och propositioner. Samma frontmatter-format som styrdokument.

---

## 🗂️ Katalogstruktur

> *Notering: Kortnamnet som ID (t.ex. `MIFF`) kräver att varje förening har en unik förkortning. UUID som organisationsidentifierare är en planerad framtida förbättring.*

```
{workspace}/
├── .tooling/
│   ├── config.yaml              # Föreningar och inställningar
│   └── mallar/                  # Typst-mallar (återställs med update-templates)
│       ├── karna.typ
│       ├── router.typ
│       ├── pandoc_klister.typ
│       ├── modul_protokoll.typ
│       └── modul_styrdokument.typ
│
├── .keys/                       # Signeringsnycklar (aldrig i git)
│   └── {OrgID}_{FP}.asc
│
├── {OrgID}/
│   ├── mallar/
│   │   └── {OrgID}.typ          # Organisationsspecifik mall (anpassningsbar)
│   │
│   ├── Årsakter/
│   │   └── {år}/
│   │       └── {organ}/
│   │           └── {datum}/
│   │               ├── källor/
│   │               │   ├── protokoll.md
│   │               │   └── bilagor/     # Valfria bilagor
│   │               └── arkiv/           # Byggt och förseglat
│   │                   ├── protokoll.pdf
│   │                   ├── protokoll.html
│   │                   ├── protokoll.docx
│   │                   ├── ATTESTATION.md
│   │                   ├── ATTESTATION.md.sig
│   │                   ├── PUBLIC_KEY.asc
│   │                   └── {datum}_Arkivpaket.zip  # Om create_zip: true
│   │
│   └── Grundakter/
│       ├── Styrdokument/
│       │   └── {kategori}/
│       │       └── {namn}/
│       │           ├── källor/
│       │           └── arkiv/
│       └── {kategori}/          # Egna kategorier (Avtal, Motioner, osv.)
│           └── {namn}/
│               ├── källor/
│               └── arkiv/
```

---

## 🔑 Nyckelhantering

Signeringsnycklar genereras och hanteras via TUI:

**Inställningar → Nyckelhantering → Skapa ny signeringsnyckel**

- Nycklar är EdDSA-nycklar krypterade med en PIN
- Lagras som ASCII-armored PGP-filer i `.keys/`
- `.keys/`-mappen är undantagen från git via `.gitignore`
- Publika nyckeln exporteras automatiskt som `PUBLIC_KEY.asc` vid försegling

---

## 👩‍💻 För utvecklare

### Bygga

```bash
go build ./...
```

### Tester

```bash
go test ./...
```

### Paketöversikt

| Paket | Ansvar |
|---|---|
| `internal/app` | Kärnlogik: build-pipeline, försegling, config, hjälpfunktioner |
| `internal/tui` | Terminal UI — state machine (Bubble Tea) + formulär (Huh) |
| `internal/keys` | EdDSA-nyckelgenerering, listning och upplåsning (ProtonMail/go-crypto) |
| `internal/assets` | Inbäddade Typst-mallar via `embed.FS` |

### Dokumentpipeline

```
Markdown (YAML-frontmatter)
    ↓ pandoc (med pandoc_klister.typ)
Typst-kod
    ↓ typst compile
PDF/A-2B  →  arkiv/{namn}.pdf
    (+ pandoc separat för HTML och DOCX)
```
