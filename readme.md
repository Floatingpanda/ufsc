# todo

- activate - payment
- all tutors
- emails
- welcome emails
- sms
- edit my profile

- välja alla knapp avmarkera alla
- implementera "om studiecoach" tab
- lägg till "anställ privat betala enkelt lön" vid signup

- Annonsering
- Mina skickade förfrågningar på hem

[Dummy text för nu: För all information om hur du som studiecoach ska redovisa din inkomst så hänvisar vi till Skatteverket...]
https://skatteverket.se/foretag/drivaforetag/foretagsformer/enskildnaringsverksamhet.4.5c13cb6b1198121ee8580002518.html?q=enskild+verksamhet

- footer pages

- alla studiecoacher
- admin pages

# Pending

- lägga till tid och plats flera tider och plats, blir flera förfrågningar

# Done

- hem Studiecoach
- min profil
- hem elev
- send request page
- signin
- favicon
- location ska vara multiple när man skapar coachkonto

frågor:

- Visa postmark
- trycker man på betalningsikoner
  Hur ska tid fungera, när?
  Hur fungerar betalning, 1st vs perioder?
  När man är på betalningssidan, är det här man ska ange discount code?

  Hur fungerar antal bokningar, går en bokning någonsin ut?
  Har man olika nivåer för olika ämnen?
  Locations, har man en eller flera?

Vart aktiverar man subscription

pages:

---

betalning

Scenario A

1. gå igenom alla steg
2. skapa lesson mot db
3. kolla om användare har aktiverad period
4. Om användaren är aktiverad skicka

Scenario B - Betala för en gång

1. gå igenom alla steg
2. skapa lesson mot db, skicka inte om aktivering saknas
3. Visa val för betalning, användaren väljer 1 gång
4. användaren betalar
5. när betalning gått igenom skicka förfrågan direkt, och visa grattis

Scenario C - Betala för en period

1. gå igenom alla steg
2. skapa lesson mot db, skicka inte om aktivering saknas
3. Visa val för betalning, användaren väljer period
4. användaren betalar
5. när betalning gått igenom skicka senaste öppnas förfrågan direkt, och visa grattis
