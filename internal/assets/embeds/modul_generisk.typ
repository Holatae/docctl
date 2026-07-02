#import "karna.typ": pappers_motor

// Generisk layout: inga formkrav (ingen versionsruta, ingen signaturssida).
// Används när typ-fältet i YAML-frontmatter inte är "protokoll" eller "styrdokument".
#let layout_generisk(
  titel: none,
  org_namn: "",
  org_nummer: "",
  body,
) = {
  let generisk_innehall = [

    // Numrering: 1. / 1.1 / 1.1.1 — nollstrippning hanterar att titeln inte är ett Typst-heading
    #set heading(numbering: (..nums) => {
      let vals = nums.pos()
      if vals.len() > 0 and vals.at(0) == 0 { vals = vals.slice(1) }
      if vals.len() > 0 { vals.map(str).join(".") }
    })

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
    generisk_innehall,
  )
}
