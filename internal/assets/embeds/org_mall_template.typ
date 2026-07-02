// Organisations-specifik fil — skapas automatiskt vid "org add".
// Säker att redigera: skrivs aldrig över av "docctl update-templates".
//
// Lägg till egna Typst-variabler eller funktioner nedan för att anpassa
// utseendet på din förenings dokument. Tänk på att karna.typ och layout-
// modulerna också behöver uppdateras för att använda eventuella nya variabler.
//
// Exempel på vad som kan läggas till:
//   #let huvud_typsnitt = "Helvetica"
//   #let accent_farg    = rgb("#004B87")

#let org_namn   = "{{ORG_NAMN}}"
#let org_nummer = "{{ORG_NUMMER}}"
