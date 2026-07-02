#let pappers_motor(
  titel: none,
  org_namn: "",
  org_nummer: "",
  body,
) = {
  set page(
    paper: "a4",
    margin: 2.5cm,
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

  set text(font: ("Linux Libertine", "Libertinus Serif", "Georgia", "Times New Roman"), lang: "sv", size: 11pt)

  // Tabeller vänsterställs
  show figure.where(kind: table): set align(left)
  set table(align: left)

  // TITEL
  if titel != none {
    align(center)[
      #text(size: 20pt, weight: "bold")[#titel] \
      #v(0.5cm)
    ]
  }

  // SPOTTA UT INNEHÅLLET (Som formateras av modulerna ovanför)
  body
}
