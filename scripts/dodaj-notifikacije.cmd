@echo off
if "%~1"=="" goto usage
docker compose exec -T cassandra cqlsh -e "INSERT INTO notifications.notifications (user_id, created_at, id, type, message) VALUES ('%~1', toTimestamp(now()), uuid(), 'SHARE', 'Probna notifikacija: podeljena je datoteka sa tobom');"
docker compose exec -T cassandra cqlsh -e "INSERT INTO notifications.notifications (user_id, created_at, id, type, message) VALUES ('%~1', toTimestamp(now()), uuid(), 'NEW_FILE', 'Probna notifikacija: dodata je nova datoteka u deljeni folder');"
docker compose exec -T cassandra cqlsh -e "INSERT INTO notifications.notifications (user_id, created_at, id, type, message) VALUES ('%~1', toTimestamp(now()), uuid(), 'NEW_VERSION', 'Probna notifikacija: otpremljena je nova verzija deljene datoteke');"
echo Gotovo. Osvezi stranicu sa notifikacijama.
exit /b 0
:usage
echo Upotreba: scripts\dodaj-notifikacije.cmd ID_KORISNIKA
exit /b 1