# DocCtl – Föreningsarkiv som Kod

*(An English version of this README will be added at a later stage.)*

**🤖 Transparens angående kodbasen:** För att nå denna MVP (Version 1.0) har
artificiell intelligens använts för att generera majoriteten av koden.
Kärnfokus har legat på att bevisa konceptet, kryptografin och arkitekturen.
Koden fungerar, men refaktoriseringsbehov finns absolut!

---

DocCtl är ett plattformsoberoende verksamhetssystem för ideella föreningar som
applicerar "Document as Code" på formell föreningsstyrning. Syftet är att lösa
formsprängning, säkerställa arkivbeständighet och erbjuda juridisk försegling
utan dyra licenskostnader.

## 💡 Varför DocCtl?

Ideell sektor blöder pengar på prenumerationstjänster för e-signering, och
historiska beslut går förlorade i trasiga Word-dokument och raderade
molnmappar. DocCtl löser detta genom att:

* Tvinga fram en "Single Source of Truth" i rena textfiler (Markdown).
* Generera deterministiska PDF/A för långtidsarkivering (via Typst).
* Generera filer i olika format för tillgänglighetssyfte, exemeplvis HTML och DOCX.
* Ersätta dyra e-signaturer med en decentraliserad matematisk beviskedja.

## 🪆 Säkerhetsmodell: "The Russian Doll"

Dokument förseglas i fyra lager, vilket gör manipulation i efterhand praktiskt
omöjlig:

1. **Integritet:** SHA-256 hashning av PDF och bilagor.
2. **Manifest:** Ett textdokument som binder samman filerna.
3. **Signatur:** Arkivariens kryptografiska underskrift via GPG.
4. **Tidsbevis:** Tidsstämpling via OpenTimestamps mot Bitcoins blockkedja.

## 🚀 Kom igång (För utvecklare/Tidiga testare)

### Krav / Beroenden

För att hela kedjan (från bygge till försegling) ska fungera behöver följande
program finnas installerade på ditt system:

1. `go`
2. `git`
3. `opentimestamps-client`
4. `gnupg` (GPG)
5. `typst`
6. `pandoc`

### Exempelanvändning

Du kommer åt systemets terminalgränssnitt (TUI) genom att helt enkelt köra
binärfilen i terminalen:

```bash
./docctl
```
