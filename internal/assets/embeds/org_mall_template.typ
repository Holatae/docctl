// Importera fabrikens moduler
#import "../../.tooling/mallar/modul_protokoll.typ": layout_protokoll
#import "../../.tooling/mallar/modul_styrdokument.typ": layout_styrdokument

// Pandoc anropar denna funktion med all data
#let org_dokument(
  typ: "protokoll", // Fallback om yaml saknar typ
  titel: none,
  datum: none,
  tid: none,
  plats: none,
  version: none,
  antagen: none,
  ordforande: none,
  sekreterare: none,
  justerare: (),
  parter: (),
  bilagor: (),
  body,
) = {
  let org_namn = "{{ORG_NAMN}}"
  let org_nummer = "{{ORG_NUMMER}}"

  // ============================================
  // ROUTERN (Växelväxlaren)
  // ============================================

  if typ == "protokoll" {
    // 1. Specifik routing för Protokoll
    layout_protokoll(
      titel: titel,
      datum: datum,
      tid: tid,
      plats: plats,
      ordforande: ordforande,
      sekreterare: sekreterare,
      justerare: justerare,
      bilagor: bilagor,
      org_namn: org_namn,
      org_nummer: org_nummer,
      body,
    )
  } else if typ == "avtal" {
    // 2. (Har du en modul_avtal.typ lägger du den här!)
  } else {
    // ============================================
    // 3. FALLBACK-KROCKKUDDEN!
    // ============================================
    // Hit kommer "policy", "reglemente", "arbetsordning", "test" och allt annat
    // systemet inte känner igen specifikt. De skrivs ut som ett snyggt Styrdokument!

    layout_styrdokument(
      titel: titel,
      version: version,
      antagen: antagen,
      bilagor: bilagor,
      org_namn: org_namn,
      org_nummer: org_nummer,
      body,
    )
  }
}

