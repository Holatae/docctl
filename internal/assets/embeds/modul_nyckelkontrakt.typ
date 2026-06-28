#import "karna.typ": pappers_motor

#show: pappers_motor.with(
  titel: "Nyckeldeklaration – Ensidigt åtagande för digital signering",
  org_namn: "{{ORG_NAMN}}",
  org_nummer: "{{ORG_NUMMER}}",
)

Undertecknad utfärdar härmed denna deklaration för att tjäna som ett varaktigt och bindande bevis
gällande kopplingen mellan undertecknads fysiska identitet och den kryptografiska nyckel som anges
nedan. Syftet med denna deklaration är att fastställa att nyckelns signaturer fullt ut ska
jämställas med undertecknads fysiska namnteckning inom ramen för angiven verksamhet. Denna handling
är utformad för att kunna bifogas digitalt som ett fristående rotbevis (bevispaket) för att uppfylla
kraven på identifiering för en Avancerad Elektronisk Signatur (AES) enligt Europaparlamentets och
rådets förordning (EU) nr 910/2014 (eIDAS-förordningen).

#v(0.8cm)

#table(
  columns: (auto, 1fr),
  stroke: none,
  fill: (_, row) => if calc.odd(row) { luma(240) } else { white },
  inset: (x: 0.6em, y: 0.5em),
  [*Nyckelinnehavare*], [{{NAMN}}],
  [*Roll/Kontext*],     [{{ETIKETT}}],
  [*E-post*],           [{{EMAIL}}],
  [*Fingeravtryck*],       [#raw("{{FINGERAVTRYCK}}")],
  [*Utfärdat datum*],      [{{SKAPAD}}],
  [*Giltig till*],         [{{GILTIG_TILL}}],
  [*Publik nyckelkatalog*],[#link("{{NYCKEL_KATALOG}}")],
)

#v(1cm)

== Deklaration och bindande åtagande

Genom min fysiska underskrift av detta dokument förklarar jag mig bunden till följande:

#set enum(tight: false, indent: 0.5em)

+ *Unik koppling och exklusiv kontroll:* Den digitala nyckeln med fingeravtrycket
  #raw("{{FINGERAVTRYCK}}") är unikt knuten till min identitet. Jag bekräftar att den privata
  nyckeln förvaras säkert under min strikta och exklusiva kontroll (skyddad via PIN-kod,
  lösenfras eller hårdvarutoken).

+ *Tillskrivning och ansvar:* Jag åtar mig att alla signaturer genererade med denna nyckel kan
  tillskrivas mig som upphovsman och utgör bevis för mitt samtycke till innehållet i berörda
  handlingar. Åtagandet gäller handlingar, protokoll, beslut och dokument som rör *{{ORG_NAMN}}*.

+ *Bevisbörda gentemot tredje part:* Jag är införstådd med att denna deklaration i skannat
  utförande kommer att distribueras tillsammans med de digitalt signerade dokumenten. Syftet är
  att ge extern tredje part (mottagare av dokumenten) tillförlitligt bevis på att jag är
  upphovsman till signaturen.

+ *Verifiering av tidpunkt:* För att säkerställa signaturernas giltighet och hantera bevisbördan
  avseende när en signering har skett, godtar jag att signaturerna tidsstämplas digitalt via en
  oberoende tredjepartslösning (exempelvis OpenTimestamps).

+ *Återkallande och giltighet:* Åtagandet gäller till och med det datum som anges i tabellen
  ovan under "Giltig till", eller tills dess att mitt återkallningscertifikat har distribuerats
  och bekräftats — beroende på vilket som inträffar först. Om jag misstänker att min privata
  nyckel har komprometterats förbinder jag mig att omedelbart publicera återkallningscertifikatet.
  Aktuell nyckelstatus kan verifieras på #link("{{NYCKEL_KATALOG}}"). Giltigheten bedöms utifrån
  certifikatets skapandedatum: signaturer vars skapandedatum föregår återkallningscertifikatets
  tidsstämpel förblir bindande.

#v(1.2cm)

== Underskrift

_Utfärdat som en ensidig, bindande viljeförklaring undertecknad på papper. Originalet förvaras i
säkert arkiv, och digitala kopior får fritt bifogas signerade filer för verifiering av min identitet._

#v(2.5cm)

#grid(
  columns: (1fr, 1fr),
  column-gutter: 2cm,
  row-gutter: 1.5cm,
  [
    #line(length: 100%, stroke: 0.5pt)
    #v(0.2cm)
    *{{NAMN}}*, Deklarant/Nyckelinnehavare
  ],
  [
    #line(length: 100%, stroke: 0.5pt)
    #v(0.2cm)
    Ort och datum
  ],
)

#v(0.8cm)
#text(size: 10pt)[Avser kontext: *{{ORG_NAMN}}* | Org.nr: {{ORG_NUMMER}}]
