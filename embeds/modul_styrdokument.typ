#import "karna.typ": pappers_motor

#let layout_styrdokument(
  titel: none,
  version: none,
  antagen: none,
  bilagor: (),
  org_namn: "",
  org_nummer: "",
  body,
) = {
  let styrdokument_innehall = [

    // En snygg versionstabell högst upp istället för datum/plats!
    #align(center)[
      #block(fill: luma(240), inset: 10pt, radius: 4pt, width: 80%)[
        #grid(
          columns: (1fr, 1fr),
          align(left)[*Version:* #if version != none { version } else { "Utkast" }],
          align(right)[*Beslutat:* #if antagen != none { antagen } else { "Ej antagen" }],
        )
      ]
      #v(1.5cm)
    ]

    // Standard numrering (1.1 istället för § 1.1)
    #set heading(numbering: "1.1")
    #show heading.where(level: 2): it => block(width: 100%, inset: (top: 1em, bottom: 0.5em))[
      #text(size: 14pt, weight: "bold")[#if it.numbering != none {
          counter(heading).display(it.numbering)
          h(0.5em)
        } #it.body]
    ]

    #body
  ]

  pappers_motor(
    titel: titel,
    org_namn: org_namn,
    org_nummer: org_nummer,
    styrdokument_innehall,
  )
}
