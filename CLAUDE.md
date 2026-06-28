# DocCtl — Kodbase-dokumentation

## Vad projektet är

Go-baserat CLI/TUI-verktyg för att hantera föreningsdokument enligt "Document as Code"-principen. Målgrupp: svenska ideella föreningar. Språk i UI och felmeddelanden: svenska.

## Build & test

```bash
go build ./...          # Kompilera
go test ./...           # Kör alla tester
go run main.go          # Kör utan att bygga binär
```

Externalaberoenden som måste finnas på systemet för full funktionalitet: `typst`, `pandoc`, `git`, `gpg`. OpenTimestamps (`ots`) är valfritt.

## Paketarkitektur

| Paket | Ansvar |
|---|---|
| `internal/app` | Kärnlogik: build-pipeline, försegling, konfiguration, hjälpfunktioner |
| `internal/tui` | Terminal UI — Bubble Tea state machine + Huh-formulär |
| `internal/keys` | EdDSA-nyckelgenerering, listning och upplåsning (ProtonMail/go-crypto) |
| `internal/assets` | Inbäddade Typst-mallar via `embed.FS` |

## Viktiga datastrukturer

- `Config`, `Association`, `Settings` — `internal/app/config.go`
- `KeyMeta` — `internal/keys/keys.go`
- `mainModel` (Bubble Tea-modell) + `appState` (iota-konstanter) — `internal/tui/tui.go`
- `DocumentType` + konstanter (`DocumentTypeProtokoll`, `DocumentTypeStyrdokument`, `DocumentTypeOther`) — `internal/tui/types.go`

## TUI state machine

`mainModel.Update()` i `tui.go` delegerar till en `updateXxx`-funktion per state. Alla states listas som `appState`-konstanter (iota) längst upp i `tui.go`. Formulär skapas i `forms.go` och återanvänds via `mainModel.activeForm`.

Fel i TUI-flödet sätts i `mainModel.errMsg` och returnerar `tea.Quit`. `StartTUI()` kontrollerar `errMsg` på den returnerade modellen och returnerar det som ett Go-fel.

## Dokumentpipeline

```
Markdown (YAML-frontmatter)
    ↓  pandoc --template pandoc_klister.typ
Typst-kod
    ↓  typst compile
PDF/A-2B   →  arkiv/{namn}.pdf
pandoc     →  arkiv/{namn}.html
pandoc     →  arkiv/{namn}.docx
```

Implementeras i `internal/app/build.go` (`DoBuild`). Tillfälliga Typst-filer namnges `*_temp.typ` och städas bort efter bygget.

## Förseglingskedja

Implementeras i `internal/app/seal.go` (`DoSeal`):
1. SHA-256-hashning av alla filer i `arkiv/`
2. Skriver `ATTESTATION.md` (manifest)
3. OpenPGP-signerar manifestet (via go-crypto, inget gpg-beroende) → `ATTESTATION.md.sig` + `PUBLIC_KEY.asc`
4. Valfritt: `ots stamp ATTESTATION.md.sig` → `ATTESTATION.md.sig.ots`
5. Valfritt: ZIP-arkiv (`{namn}_Arkivpaket.zip`) om `settings.create_zip: true`

## Filplatser

| Fil/mapp | Syfte |
|---|---|
| `.tooling/config.yaml` | Föreningar och inställningar, skapas vid `init` |
| `.tooling/mallar/` | Typst-mallar, kopieras från inbäddad `embed.FS` vid `init` |
| `.keys/{OrgID}_{FP}.asc` | Krypterade privata nycklar, aldrig i git |
| `{OrgID}/mallar/{OrgID}.typ` | Organisationsspecifik Typst-mall, skapas vid `org add` |
| `**/källor/` | Källfiler (Markdown + bilagor) |
| `**/arkiv/` | Byggda och förseglade filer |

## Inbäddade mallar

Lagras i `internal/assets/embeds/` och bäddas in i binären via `//go:embed`. Kopieras till `.tooling/mallar/` vid `init` och kan återställas med `docctl update-templates`.

| Mall | Funktion |
|---|---|
| `pandoc_klister.typ` | Brygga Pandoc→Typst, mappar YAML-frontmatter till Typst-variabler |
| `router.typ` | Väljer layout-modul baserat på `typ`-fältet |
| `karna.typ` | Bassidemall (A4, marginaler, sidfot med org-info, typsnitt) |
| `modul_protokoll.typ` | Layout för protokoll (§-numrering, underskriftssida) |
| `modul_styrdokument.typ` | Layout för styrdokument (versionstabell, 1.1-numrering) |
| `org_mall_template.typ` | Mall för org-specifik fil, placeholders: `{{ORG_NAMN}}`, `{{ORG_NUMMER}}` |
