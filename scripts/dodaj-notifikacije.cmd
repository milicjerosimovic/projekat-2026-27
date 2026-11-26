@echo off
if "%~1"=="" goto usage
docker compose exec -T cassandra cqlsh -e "INSERT INTO notifications.notifications (user_id, created_at, id, type, message) VALUES ('%~1', toTimestamp(now()), uuid(), 'SHARE', 'Marko je podelio fajl izvestaj.pdf sa tobom');"
docker compose exec -T cassandra cqlsh -e "INSERT INTO notifications.notifications (user_id, created_at, id, type, message) VALUES ('%~1', toTimestamp(now()), uuid(), 'NEW_FILE', 'Dodata nova datoteka u folder Projekat');"
docker compose exec -T cassandra cqlsh -e "INSERT INTO notifications.notifications (user_id, created_at, id, type, message) VALUES ('%~1', toTimestamp(now()), uuid(), 'NEW_VERSION', 'Nova verzija datoteke slika.png');"
echo Gotovo. Osvezi stranicu sa notifikacijama.
exit /b 0
:usage
echo Upotreba: scripts\dodaj-notifikacije.cmd ID_KORISNIKA
exit /b 1