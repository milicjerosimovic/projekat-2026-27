# projekat-2026-27
## Pokretanje

Potrebni programi: Docker Desktop (mora biti pokrenut) i Git.

1. Kloniraj repozitorijum:
   ```
   git clone https://github.com/milicjerosimovic/projekat-2026-27.git
   cd projekat-2026-27
   ```
2. Napravi fajl sa podešavanjima iz šablona:

   Windows (cmd):
   ```
   copy .env.example .env

3. Pokreni sve:
   ```
   docker compose up --build
   ```
4. Otvori **https://localhost:3443**.
5. Zaustavljanje: `Ctrl + C`, pa `docker compose down`.

   
 ## Adrese


|---|---|
| Prijava i registracija | https://localhost:3443 |
| Files (folderi i otpremanje) | https://localhost:3443/files-ui/ |
| Notifikacije | https://localhost:3443/notifications-ui/ |
| MailHog (imejlovi i kodovi za prijavu) | http://localhost:8025 |

Adresa `http://localhost:3000` automatski preusmerava na HTTPS.

Files i Notifikacije se otvaraju tek **posle prijave** (u istom browseru). Servisi i baze nisu otvoreni direktno ka računaru, već samo kroz nginx i gateway (`https://localhost:3443/api/...`).
## Upozorenje browsera (Advanced)

Aplikacija radi preko HTTPS-a sa razvojnim sertifikatom koji se pravi automatski pri prvom pokretanju. Zato će browser upozoriti da veza nije privatna. To je očekivano.

1. Klikni **Advanced** (**Napredno**).
2. Klikni **Proceed to localhost** (**Nastavi na localhost**).

Ako treba novi sertifikat, obriši folder `certs` i ponovo pokreni `docker compose up --build`.

scripts\dodaj-notifikacije.cmd ID_KORISNIKA
