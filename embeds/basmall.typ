#let bas_dokument(
  titel: none,
  datum: none,
  tid: none,
  plats: none,
  ordforande: none,
  sekreterare: none,
  justerare: (),
  bilagor: (),
  org_namn: "",
  org_nummer: "",
  body,
) = {
  set page(
    paper: "a4",
    margin: 2.5cm,
    //header: align(right)[_ #org_namn _],
    footer: context [
      #line(length: 100%, stroke: 0.5pt + luma(150))
      #v(0.1cm)
      #grid(
        columns: (1fr, 1fr),
        align(left)[#text(fill: luma(100), size: 9pt)[#org_namn | Org.nr: #org_nummer]],
        align(right)[#text(
          fill: luma(100),
          size: 9pt,
        )[Sida #counter(page).display() av #counter(page).final().first()]],
      )
    ],
  )
  set text(font: "Linux Libertine", lang: "sv", size: 11pt)

  // -------------------------------------------------------------
  // TABELLER (Vänsterställda och stilrena)
  // -------------------------------------------------------------
  show figure.where(kind: table): set align(left) // Vänsterställer hela tabellen på papperet
  set table(align: left) // Vänsterställer texten inuti tabellens rutor


  // TITEL OCH DATUM
  align(center)[
    #if titel != none { text(size: 20pt, weight: "bold")[#titel] } \
    #v(0.5cm)
  ]
  if datum != none [ #text(size: 12pt)[*Datum:* #datum] \ ]
  if tid != none [ #text(size: 12pt)[*Tid:* #tid] \ ]
  if plats != none [ #text(size: 12pt)[*Plats:* #plats] \ ]

  v(0.5cm)
  // -------------------------------------------------------------
  // AUTONUMRERING AV PARAGRAFER (H2 i Markdown)
  // -------------------------------------------------------------
  let paragraf_counter = counter("paragraf")

  set heading(numbering: (..nums) => {
    let vals = nums.pos()
    if vals.len() > 0 and vals.at(0) == 0 { vals = vals.slice(1) }
    if vals.len() > 0 { "§ " + vals.map(str).join(".") }
  })

  // 2. Vi bestämmer hur Rubrik 2 (H2) ska se ut på papperet (nivå 1 i protokollet)
  show heading.where(level: 2): it => {
    block(width: 100%, inset: (top: 1em, bottom: 0.5em))[
      #text(size: 14pt, weight: "bold")[
        #if it.numbering != none {
          counter(heading).display(it.numbering) // Hämtar det inbyggda numret
          h(0.5em)
        }
        #it.body
      ]
    ]
  }

  // 3. Vi bestämmer hur Rubrik 3 (H3) ska se ut på papperet (nivå 2 i protokollet)
  show heading.where(level: 3): it => {
    block(width: 100%, inset: (top: 0.5em, bottom: 0.2em))[
      #text(size: 12pt, weight: "bold")[
        #if it.numbering != none {
          counter(heading).display(it.numbering)
          h(0.5em)
        }
        #it.body
      ]
    ]
  }


  if bilagor.len() > 0 {
    text(size: 11pt, weight: "bold")[Förteckning över bilagor:]
    v(0.1cm)
    for b in bilagor {
      // _#b_ gör texten kursiv. Backslashet \ gör ett tätt radbyte utan punkt!
      [_#b _ \ ]
    }
    v(1cm) // Skapar lite rymd innan själva protokollet (Mötets öppnande) drar igång
  }
  // -------------------------------------------------------------
  // SKRIV UT SJÄLVA DOKUMENTTEXTEN (Brödtexten)
  // -------------------------------------------------------------
  body


  // -------------------------------------------------------------
  // DYNAMISKA SIGNATURLINJER LÄNGST BAK I DOKUMENTET
  // -------------------------------------------------------------
  let signaturer = ()

  // Skapa instanser i en lista om de finns i Markdown-filen
  if ordforande != none { signaturer.push((roll: "Mötesordförande", namn: ordforande)) }

  if sekreterare != none { signaturer.push((roll: "Mötessekreterare", namn: sekreterare)) }

  // Hantera eventuella justerare (kan vara 0 till jättemånga)
  for j in justerare {
    signaturer.push((roll: "Justerare", namn: j))
  }

  // Rita ut dem om där finns några
  if signaturer.len() > 0 {
    pagebreak()
    text(weight: "bold", size: 14pt)[Signaturer]
    v(3cm)

    // Förbered varje signaturrutas utseende
    let grid_items = signaturer.map(s => {
      block(width: 100%, height: 2.5cm)[
        #line(length: 80%, stroke: 0.5pt)
        #v(0.2cm)
        #text(weight: "bold")[#s.namn]\
        #text(style: "italic", size: 10pt)[#s.roll]
      ]
    })

    // Lägg in dem i ett rutnät, två i bredd
    grid(
      columns: (1fr, 1fr),
      row-gutter: 1cm,
      column-gutter: 2cm,
      ..grid_items
    )
  }
}
