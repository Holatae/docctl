// Router: väljer layoutmodul baserat på typ-fältet i YAML-frontmatter.
//
// typ: protokoll    → layout_protokoll  (§-numrering, signaturssida)
// typ: styrdokument → layout_styrdokument (versionsruta, decimalrubrik)
// (allt annat)      → layout_generisk   (ren layout, inga formkrav)

#import "modul_protokoll.typ": layout_protokoll
#import "modul_styrdokument.typ": layout_styrdokument
#import "modul_generisk.typ": layout_generisk

#let org_dokument(
  typ: "protokoll",
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
  org_namn: "",
  org_nummer: "",
  body,
) = {
  if typ == "protokoll" {
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
  } else if typ == "styrdokument" {
    layout_styrdokument(
      titel: titel,
      version: version,
      antagen: antagen,
      bilagor: bilagor,
      org_namn: org_namn,
      org_nummer: org_nummer,
      body,
    )
  } else {
    layout_generisk(
      titel: titel,
      org_namn: org_namn,
      org_nummer: org_nummer,
      body,
    )
  }
}
