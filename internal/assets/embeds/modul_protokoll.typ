#import "karna.typ": pappers_motor

#let layout_protokoll(
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
  // 1. Förbered brödtexten med Protokollens specialregler
  let protokoll_innehall = [

    // Metadata under titeln
    #align(center)[
      #if datum != none [ #text(size: 12pt)[*Datum:* #datum] \ ]
      #if tid != none [ #text(size: 12pt)[*Tid:* #tid] \ ]
      #if plats != none [ #text(size: 12pt)[*Plats:* #plats] \ ]
      #v(1cm)
    ]

    // Bilageförteckning
    #if bilagor.len() > 0 {
      text(size: 11pt, weight: "bold")[Förteckning över bilagor:]
      v(0.1cm)
      for b in bilagor [ _#b _\ ]
      v(1cm)
    }

    // Autonumrering (Med §-tecken!)
    #set heading(numbering: (..nums) => {
      let vals = nums.pos()
      if vals.len() > 0 and vals.at(0) == 0 { vals = vals.slice(1) }
      if vals.len() > 0 { "§ " + vals.map(str).join(".") }
    })

    #show heading.where(level: 2): it => block(width: 100%, inset: (top: 1em, bottom: 0.5em))[
      #text(size: 14pt, weight: "bold")[#if it.numbering != none {
          counter(heading).display(it.numbering)
          h(0.5em)
        } #it.body]
    ]

    // Släpp in själva markdown-texten
    #body

    // Signatursidan
    #let signaturer = ()
    #if ordforande != none { signaturer.push((roll: "Mötesordförande", namn: ordforande)) }
    #if sekreterare != none { signaturer.push((roll: "Mötessekreterare", namn: sekreterare)) }
    #for j in justerare { signaturer.push((roll: "Justerare", namn: j)) }

    #if signaturer.len() > 0 {
      pagebreak()
      text(weight: "bold", size: 14pt)[Signaturer]
      v(3cm)
      let grid_items = signaturer.map(s => block(width: 100%, height: 2.5cm)[
        #line(length: 80%, stroke: 0.5pt) #v(0.2cm) #text(weight: "bold")[#s.namn]\ #text(
          style: "italic",
          size: 10pt,
        )[#s.roll]
      ])
      grid(columns: (1fr, 1fr), row-gutter: 1cm, column-gutter: 2cm, ..grid_items)
    }
  ]

  // 2. Skicka alltihop till kärnan för utskrift!
  pappers_motor(
    titel: titel,
    org_namn: org_namn,
    org_nummer: org_nummer,
    protokoll_innehall,
  )
}
