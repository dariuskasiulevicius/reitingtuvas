# Reitingtuvas — naudotojo gidas

## 1. Atsisiuntimas

1. Eikite į projekto **Releases** puslapį GitHub.
2. Atsisiųskite ZIP failą savo sistemai:
   - **Windows**: `reitingtuvas-windows-amd64.zip`
   - **macOS (Intel)**: `reitingtuvas-darwin-amd64.zip`
   - **macOS (Apple Silicon / M1–M4)**: `reitingtuvas-darwin-arm64.zip`
3. Išskleiskite ZIP į pasirinktą aplanką (pvz., Darbalaukis arba Dokumentai).

## 2. Pirmas paleidimas — saugumo įspėjimai

### Windows

Windows SmartScreen gali rodyti įspėjimą „Windows protected your PC":
1. Spauskite **Daugiau informacijos** (More info).
2. Spauskite **Vis tiek paleisti** (Run anyway).

Tai reikia padaryti tik vieną kartą.

### macOS

macOS Gatekeeper blokuos nepasirašytą programą:
1. Du kartus paspauskite `reitingtuvas` — pasirodys klaidos pranešimas.
2. Atidarykite **System Settings → Privacy & Security**.
3. Apačioje matysite pranešimą apie užblokuotą programą — spauskite **Open Anyway** (Vis tiek atidaryti).
4. Patvirtinkite.

Tai reikia padaryti tik vieną kartą.

## 3. Pirmas paleidimas — knygos sukūrimas

Pirmą kartą paleidus programą, ji sukurs failą **reitingtuvas.xlsx** tame pačiame aplanke ir parodys:

```
Sukurta nauja knyga: .../reitingtuvas.xlsx
Užpildykite Nustatymai ir Dalyviai lapus, tada paleiskite iš naujo.
```

Atidarykite `reitingtuvas.xlsx` Excel, Numbers arba LibreOffice programoje.

## 4. Nustatymų pildymas

Atidarykite **Nustatymai** lapą. Užpildykite stulpelį **Reikšmė** šalia kiekvieno rakto:

| Raktas | Ką rašyti | Pavyzdys |
|--------|-----------|----------|
| Šaltinio adresas | Svetainės adresas (gausite iš administratoriaus) | `https://...` |
| Paieškos kelias | Paieškos URL šablonas | (gausite iš administratoriaus) |
| Dalyvio rezultatų kelias | Rezultatų URL šablonas | (gausite iš administratoriaus) |
| Klasės rezultatų kelias | Klasės URL šablonas | (gausite iš administratoriaus) |
| Varžybų kelias | Varžybų URL šablonas | (gausite iš administratoriaus) |
| Dalyvio profilio kelias | Profilio URL šablonas | (gausite iš administratoriaus) |
| Laikotarpis nuo | Laikotarpio pradžia | `2026-01-01` |
| Laikotarpis iki | Laikotarpio pabaiga | `2026-12-31` |
| Pauzė tarp užklausų, ms | Pauzė milisekundėmis (numatytas 200) | `200` |
| Rezultatai galutiniai po, d. | Po kiek dienų rezultatai galutiniai (numatytas 14) | `14` |

**Svarbu:** URL šablonus ir adresą gausite iš administratoriaus. Jų nerasite viešai.

## 5. Dalyvių sąrašas

Atidarykite **Dalyviai** lapą ir užpildykite:

| Pavardė | Vardas | ID | Būsena |
|---------|--------|----|--------|
| Pavardenis | Vardenis | | |
| Kitas | Vardas | | |

- **Pavardė** ir **Vardas** — būtinai su diakritiniais ženklais (ą, č, ę, ė, į, š, ų, ū, ž).
- **ID** — palikite tuščią. Programa surinks automatiškai.
- **Būsena** — palikite tuščią. Programa užpildys.

Jei žinote dalyvio ID (pvz., iš ankstesnio paleidimo), galite jį įrašyti iš karto — tada paieška bus praleista.

## 6. Paleidimas

1. **Uždarykite** `reitingtuvas.xlsx` (programa negali rašyti į atidarytą failą).
2. Du kartus paspauskite `reitingtuvas`.
3. Programa parodys progresą:

```
reitingtuvas v1.0.0
Laikotarpis: 2026-01-01 — 2026-12-31
Dalyviai: 15
Ieškoma dalyvių...
Rasta: 14/15
Renkami rezultatai...
[47/230] Pavyzdžio taurė M21
Surinkta: 187 rezultatų
Ataskaita parašyta į lapą: Ataskaita 2026-10-20 14.30
```

4. Programa paprašys paspausti Enter — tai uždaro langą.

## 7. Rezultatai

Atidarykite `reitingtuvas.xlsx`. Rasite naujus lapus:

### Ataskaita lapas

Kiekviena eilutė = vienas dalyvis × vienos varžybos:

- **Dalyvis** — pavardė ir vardas
- **Varžybų data** — varžybų pradžios data
- **Varžybos** — varžybų pavadinimas
- **Sportas** — sporto šaka
- **Klasė** — amžiaus/lygio grupė
- **Koef.** — varžybų koeficientas (numatytas 1.00)
- **Dalyvio rezultatas** — bendras dalyvio laikas (sudėtos geros dienos)
- **Vieta** — oficiali bendra vieta
- **Lyderio rezultatas** — klasės lyderio laiko suma per visas dienas
- **D1 data, D1 būsena, D1 laikas, D1 vieta, D1 lyderio laikas** — pirmos dienos duomenys
- **D2 ...** — antros dienos duomenys (jei buvo)

Laikai saugomi kaip Excel trukmės (`[h]:mm:ss`) — galite naudoti formules.

### Varžybos lapas

Visos rastos varžybos su koeficientais. **Koeficientas** stulpelį galite keisti — programa jo neperrašys.

### Dienos lapas

Kiekvienos varžybų dienos datos. **Data** stulpelį galite taisyti, jei numatytoji data neteisinga.

## 8. Pakartotinis paleidimas

- Kaskart programa sukuria **naują Ataskaita lapą** (seni lieka).
- Naujos varžybos **pridedamos** į Varžybos lapą su koeficientu 1.00.
- Jūsų pakeisti koeficientai ir datos **nėra perrašomi**.
- Prieš kiekvieną įrašymą sukuriama atsarginė kopija `reitingtuvas.backup.xlsx`.

## 9. Dalyvio neradimas

Jei stulpelyje **Būsena** matote:
- **nerastas** — patikrinkite rašybą, ypač diakritinius ženklus.
- **keli kandidatai: ...** — raskite teisingą ID iš pateiktų variantų ir įrašykite jį rankiniu būdu į **ID** stulpelį.

## 10. Klaidos ir problemos

- **„Uždarykite reitingtuvas.xlsx ir paleiskite iš naujo"** — Excel laiko failą užrakintą. Uždarykite ir bandykite dar kartą.
- **Programa rodo klaidą ir sustoja** — patikrinkite Nustatymai lapą. Dažniausia priežastis: tuščias adresas arba neteisingas datos formatas (reikia `YYYY-MM-DD`).
- **Kai kurie rezultatai trūksta** — programa rodys įspėjimus. Klasės puslapis galėjo būti nepasiekiamas; paleiskite dar kartą — iš talpyklos nereikės atsisiųsti jau turimų.
