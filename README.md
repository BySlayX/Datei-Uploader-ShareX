# ShareX Uploader

Ein selbst gehosteter File-Host für Browser-Uploads und ShareX. Die Dateien bleiben auf deinem eigenen Server.

## Das kann ShareX Uploader

- Dateien per Drag-and-drop oder über ShareX hochladen
- Uploads in einer Galerie suchen, ansehen, herunterladen und löschen
- Dateien mit Vorschau und teilbaren Links verwalten
- Benutzer und individuelle Upload-Limits verwalten
- SQLite oder MySQL/MariaDB verwenden
- Passwort-Reset und Sicherheitsmails per SMTP versenden
- Systemstatus, Statistiken, API-Keys und Branding im Admin-Bereich verwalten
- HTTPS mit Nginx und Certbot einrichten

## Installation

Auf einem Linux-Server diesen Befehl ausführen:

```bash
curl -fsSL https://raw.githubusercontent.com/BySlayX/Datei-Uploader-ShareX/main/install-online.sh | sudo bash
```

Der Installer lädt das Projekt nach `/opt/sharex-uploader`, richtet Docker, Nginx und HTTPS ein und startet anschließend den Browser-Setup.

## Web-Setup

Nach der Installation die angezeigte URL im Browser öffnen und den Setup-Assistenten abschließen.

Der Assistent:

1. richtet die Datenbank ein,
2. erstellt den ersten Admin,
3. erzeugt die ShareX- und Admin-API-Keys,
4. sperrt die Setup-Seite nach erfolgreicher Einrichtung.

Danach kann die automatisch erzeugte ShareX-Konfiguration heruntergeladen und in ShareX importiert werden.

Die ShareX- und Admin-API-Keys werden aus Sicherheitsgründen nur einmal angezeigt. Speichere sie direkt sicher und teile sie niemals öffentlich.
