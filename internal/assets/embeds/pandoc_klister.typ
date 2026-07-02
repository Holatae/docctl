// Pandoc-mall: $variabel$-syntaxen är Pandoc (ersätts vid konvertering),
// resten är vanlig Typst-kod. Filen ska aldrig kompileras direkt med typst.
//
// YAML-fält som stöds i källdokumentets frontmatter:
//   typ, title, datum, tid, plats, version, antagen,
//   ordforande, sekreterare, justerare (lista), parter (lista), bilagor (lista)

#import "$org_mall$": org_namn, org_nummer
#import "/.tooling/mallar/router.typ": org_dokument

#show: doc => org_dokument(
  org_namn: org_namn,
  org_nummer: org_nummer,
  typ: $if(typ)$"$typ$"$else$"protokoll"$endif$,
  titel: $if(title)$[$title$]$else$none$endif$,
  datum: $if(datum)$[$datum$]$else$none$endif$,
  tid: $if(tid)$[$tid$]$else$none$endif$,
  plats: $if(plats)$[$plats$]$else$none$endif$,
  version: $if(version)$[$version$]$else$none$endif$,
  antagen: $if(antagen)$[$antagen$]$else$none$endif$,
  ordforande: $if(ordforande)$[$ordforande$]$else$none$endif$,
  sekreterare: $if(sekreterare)$[$sekreterare$]$else$none$endif$,
  justerare: ($for(justerare)$[$justerare$],$endfor$),
  parter: ($for(parter)$[$parter$],$endfor$),
  bilagor: ($for(bilagor)$[$bilagor$],$endfor$),
  doc
)

$body$
