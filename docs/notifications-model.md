# Notifications servis: model podataka (zahtev 2.1)

## Odgovornost servisa

Notifications servis čuva notifikacije korisnika i vraća ih klijentu. Podatke čuva u wide-column bazi (Apache Cassandra).

## Entitet: Notification

| Atribut | Tip | Opis |
|---|---|---|
| user_id | text | ID korisnika kome je notifikacija namenjena (ključ particije) |
| created_at | timestamp | Vreme nastanka notifikacije (sortira se od najnovije) |
| id | uuid | Jedinstveni identifikator notifikacije |
| type | text | Vrsta: SHARE, NEW_FILE ili NEW_VERSION |
| message | text | Tekst notifikacije koji se prikazuje korisniku |

Primarni ključ: (user_id), created_at, id.

## Zašto ovakav model

Najčešći upit je "sve notifikacije jednog korisnika, od najnovije". Zato je user_id ključ particije (sve notifikacije jednog korisnika su na istom mestu), a created_at klasterišući ključ sa opadajućim redosledom, pa baza vraća podatke već sortirane.

## Tipovi notifikacija

| Tip | Kada nastaje |
|---|---|
| SHARE | Neko je sa korisnikom podelio datoteku ili folder |
| NEW_FILE | U folder podeljen sa korisnikom dodata je nova datoteka |
| NEW_VERSION | Otpremljena je nova verzija datoteke podeljene sa korisnikom |

## API

| Metoda | Putanja | Opis |
|---|---|---|
| GET | /notifications?userId=... | Vraća notifikacije korisnika (JSON), najnovije prvo |
| GET | /health | Provera da servis radi |

Klijent pristupa servisu preko API gateway-a: GET /api/notifications?userId=...

## Stilovi komunikacije

| Od | Ka | Stil | Svrha |
|---|---|---|---|
| Klijentska aplikacija | API gateway | Sinhrono (REST/HTTP) | Zahtev za prikaz notifikacija |
| API gateway | Notifications | Sinhrono (REST/HTTP) | Prosleđivanje zahteva |
| Notifications | Cassandra | Sinhrono (CQL) | Čitanje i upis notifikacija |
| Sharing, Files | Notifications | Asinhrono (događaji preko message brokera) | Obaveštavanje o deljenju i novoj verziji (planirano za ocenu 8) |

Za ocenu 6 baza se popunjava ručno, a asinhrono generisanje notifikacija dodaje se kasnije.