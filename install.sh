#!/usr/bin/env bash
set -Eeuo pipefail

PROJECT_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_ROOT"
umask 077

die() {
    printf 'Fehler: %s\n' "$1" >&2
    exit 1
}

run_root() {
    if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
        "$@"
    elif command -v sudo >/dev/null 2>&1; then
        sudo "$@"
    else
        die "Root-Rechte werden benötigt. Bitte als root starten oder sudo installieren."
    fi
}

compose_available() {
    if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
        return 0
    fi
    command -v docker-compose >/dev/null 2>&1 && docker-compose version >/dev/null 2>&1
}

read_env_value() {
    local key="$1"
    [[ -f "$ENV_PATH" ]] || return 0
    awk -F= -v key="$key" '$1 == key {sub(/^[^=]*=/, ""); gsub(/^"|"$/, ""); print; exit}' "$ENV_PATH"
}

set_env_value() {
    local key="$1"
    local value="$2"
    local temporary_path

    if grep -qE "^${key}=" "$ENV_PATH"; then
        temporary_path="$(mktemp)"
        awk -v key="$key" -v value="$value" '
            BEGIN { updated = 0 }
            $0 ~ ("^" key "=") {
                print key "=" value
                updated = 1
                next
            }
            { print }
            END {
                if (!updated) print key "=" value
            }
        ' "$ENV_PATH" > "$temporary_path"
        chmod 600 "$temporary_path"
        mv "$temporary_path" "$ENV_PATH"
    else
        printf '%s=%s\n' "$key" "$value" >> "$ENV_PATH"
    fi
}

configure_https() {
    local configured_host configured_email
    configured_host="$(read_env_value PUBLIC_HOST)"
    configured_email="$(read_env_value TLS_EMAIL)"

    if [[ -z "${PUBLIC_HOST:-}" ]]; then
        PUBLIC_HOST="${configured_host:-}"
        if [[ -t 0 ]]; then
            if [[ -n "$PUBLIC_HOST" ]]; then
                read -r -p "HTTPS-Domain [${PUBLIC_HOST}]: " entered_host
            else
                read -r -p "HTTPS-Domain: " entered_host
            fi
            PUBLIC_HOST="${entered_host:-$PUBLIC_HOST}"
        fi
    fi
    [[ -n "$PUBLIC_HOST" ]] || die "Bitte eine HTTPS-Domain angeben."
    PUBLIC_HOST="${PUBLIC_HOST#http://}"
    PUBLIC_HOST="${PUBLIC_HOST#https://}"
    PUBLIC_HOST="${PUBLIC_HOST%%/*}"
    [[ "$PUBLIC_HOST" =~ ^[A-Za-z0-9][A-Za-z0-9.-]*$ ]] || die "Die HTTPS-Adresse muss ein Hostname ohne http://, https:// oder Port sein."

    if [[ -z "${TLS_EMAIL:-}" ]]; then
        TLS_EMAIL="${configured_email:-}"
        if [[ -t 0 ]]; then
            while [[ -z "$TLS_EMAIL" ]]; do
                read -r -p "E-Mail für TLS-Zertifikat und Erneuerungswarnungen: " TLS_EMAIL
            done
        fi
    fi
    [[ "$TLS_EMAIL" =~ ^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$ ]] || die "Bitte eine gültige E-Mail-Adresse für das TLS-Zertifikat angeben."

    if [[ ! -f "$ENV_PATH" ]]; then
        install -m 600 /dev/null "$ENV_PATH"
    fi
    set_env_value PUBLIC_HOST "$PUBLIC_HOST"
    set_env_value TLS_EMAIL "$TLS_EMAIL"
    set_env_value BASE_URL "https://${PUBLIC_HOST}"
    set_env_value COOKIE_SECURE true
    set_env_value CORS_ORIGIN "https://${PUBLIC_HOST}"
    chmod 600 "$ENV_PATH"

    if command -v getent >/dev/null 2>&1 && ! getent ahostsv4 "$PUBLIC_HOST" >/dev/null 2>&1; then
        printf 'Hinweis: %s ist aktuell noch nicht per DNS auf diesen Server auflösbar.\n' "$PUBLIC_HOST"
        printf 'Setze zuerst den A-Record auf die Server-IP. Certbot kann das Zertifikat sonst nicht ausstellen.\n'
    fi
}

detect_server_ip() {
    local detected_ip=''
    if command -v ip >/dev/null 2>&1; then
        detected_ip="$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{for (i = 1; i <= NF; i += 1) if ($i == "src") { print $(i + 1); exit }}')"
    fi
    if [[ -z "$detected_ip" ]] && command -v hostname >/dev/null 2>&1; then
        detected_ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
    fi
    printf '%s\n' "$detected_ip"
}

read_web_setup_config() {
    local container_id config_copy
    container_id="$(compose "${COMPOSE_ENV_ARGS[@]}" ps -q app | head -n 1)"
    [[ -n "$container_id" ]] || die 'Der App-Container wurde nach dem Web-Setup nicht gefunden.'
    config_copy="$(mktemp)"
    rm -f "$config_copy"
    if ! "${DOCKER[@]}" cp "${container_id}:/app/data/config.json" "$config_copy" >/dev/null; then
        rm -f "$config_copy"
        die 'Die Web-Setup-Konfiguration konnte nicht aus dem Container gelesen werden.'
    fi

    if command -v python3 >/dev/null 2>&1; then
        WEB_BASE_URL="$(python3 - "$config_copy" <<'PY'
import json
import sys

with open(sys.argv[1], encoding='utf-8') as handle:
    config = json.load(handle)
print(config.get('baseUrl', ''))
PY
)"
        WEB_TLS_EMAIL="$(python3 - "$config_copy" <<'PY'
import json
import sys

with open(sys.argv[1], encoding='utf-8') as handle:
    config = json.load(handle)
print(config.get('adminEmail', ''))
PY
)"
    else
        WEB_BASE_URL="$(sed -n 's/.*"baseUrl"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$config_copy" | head -n 1)"
        WEB_TLS_EMAIL="$(sed -n 's/.*"adminEmail"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$config_copy" | head -n 1)"
    fi
    rm -f "$config_copy"

    WEB_BASE_URL="${WEB_BASE_URL%/}"
    if [[ "$WEB_BASE_URL" =~ ^https://([A-Za-z0-9][A-Za-z0-9.-]*)$ ]]; then
        PUBLIC_HOST="${BASH_REMATCH[1]}"
    else
        die 'Bitte im Web-Setup eine endgültige HTTPS-Adresse ohne Pfad eintragen.'
    fi
    TLS_EMAIL="$WEB_TLS_EMAIL"
    [[ "$TLS_EMAIL" =~ ^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$ ]] || die 'Die Admin-E-Mail aus dem Web-Setup ist keine gültige TLS-E-Mail-Adresse.'
}

open_temporary_setup_port() {
    if command -v ufw >/dev/null 2>&1 && run_root ufw status 2>/dev/null | grep -q 'Status: active'; then
        if ! run_root ufw status 2>/dev/null | grep -Eq '8080/tcp'; then
            run_root ufw allow 8080/tcp comment 'ShareX temporary setup' >/dev/null || true
            TEMPORARY_UFW_RULE=true
        fi
    fi
}

close_temporary_setup_port() {
    if [[ "${TEMPORARY_UFW_RULE:-false}" == true ]] && command -v ufw >/dev/null 2>&1; then
        run_root ufw delete allow 8080/tcp >/dev/null || true
        TEMPORARY_UFW_RULE=false
    fi
}
trap close_temporary_setup_port EXIT

finish_web_setup_https() {
    local setup_status
    printf '\nWeb-Setup abgeschlossen. Übernehme jetzt Domain und HTTPS-Konfiguration...\n'
    read_web_setup_config
    configure_https
    set_env_value BIND_ADDRESS '127.0.0.1'
    COMPOSE_ENV_ARGS=(--env-file .env)
    start_nginx
    nginx_configure
    compose "${COMPOSE_ENV_ARGS[@]}" config >/dev/null || die 'Die HTTPS-Compose-Konfiguration ist ungültig.'
    compose "${COMPOSE_ENV_ARGS[@]}" up --build -d || die 'Der HTTPS-Neustart von Docker Compose ist fehlgeschlagen.'
    issue_tls_certificate
    nginx_harden_site
    harden_host
    close_temporary_setup_port
    setup_status='https'
    printf 'HTTPS ist eingerichtet für %s.\n' "$PUBLIC_HOST"
}

run_web_setup_mode() {
    local server_ip temporary_url setup_status
    server_ip="$(detect_server_ip)"
    [[ "$server_ip" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]] || die 'Die öffentliche Server-IP konnte nicht automatisch ermittelt werden.'

    if [[ ! -f "$ENV_PATH" ]]; then
        install -m 600 /dev/null "$ENV_PATH"
    fi
    set_env_value BIND_ADDRESS '0.0.0.0'
    set_env_value PORT '8080'
    set_env_value BASE_URL "http://${server_ip}:8080"
    set_env_value COOKIE_SECURE false
    set_env_value CORS_ORIGIN "http://${server_ip}:8080"
    chmod 600 "$ENV_PATH"
    COMPOSE_ENV_ARGS=(--env-file .env)

    compose "${COMPOSE_ENV_ARGS[@]}" config >/dev/null || die 'Die temporäre HTTP-Compose-Konfiguration ist ungültig.'
    open_temporary_setup_port
    printf '\nStarte temporären HTTP-Setup-Zugriff...\n'
    compose "${COMPOSE_ENV_ARGS[@]}" up --build -d || die 'Der temporäre HTTP-Container konnte nicht gestartet werden.'
    temporary_url="http://${server_ip}:8080/setup"
    printf '\nÖffne jetzt im Browser:\n%s\n' "$temporary_url"
    printf 'Trage dort die endgültige HTTPS-Adresse deiner Domain ein. Die Admin-E-Mail wird für das TLS-Zertifikat verwendet.\n'
    printf 'Der Installer wartet auf den Abschluss des Web-Setups...\n'

    while true; do
        setup_status="$(curl -fsS --max-time 5 http://127.0.0.1:8080/api/setup/status 2>/dev/null || true)"
        if [[ "$setup_status" == *'"configured":true'* ]]; then
            break
        fi
        sleep 3
    done
    finish_web_setup_https
}

nginx_body_size() {
    # High reverse-proxy ceiling; the Go app still enforces per-user / instance limits.
    printf '12g'
}

nginx_configure() {
    local temporary_path app_port body_size
    app_port="$(read_env_value PORT)"
    app_port="${app_port:-8080}"
    body_size="$(nginx_body_size)"
    temporary_path="$(mktemp)"
    cat > "$temporary_path" <<EOF
server {
    listen 80;
    listen [::]:80;
    server_name ${PUBLIC_HOST};

    client_max_body_size ${body_size};
    proxy_read_timeout 300s;
    proxy_send_timeout 300s;

    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;
    add_header Referrer-Policy "no-referrer" always;
    add_header Permissions-Policy "camera=(), microphone=(), geolocation=()" always;

    location ^~ /.well-known/acme-challenge/ {
        root /var/www/sharex-uploader;
        default_type "text/plain";
    }

    location / {
        proxy_pass http://127.0.0.1:${app_port};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_buffering off;
    }
}
EOF
    run_root install -d -m 755 /var/www/sharex-uploader
    run_root install -m 0644 "$temporary_path" /etc/nginx/sites-available/sharex-uploader
    rm -f "$temporary_path"
    if [[ -d /etc/nginx/sites-enabled ]]; then
        run_root ln -sfn /etc/nginx/sites-available/sharex-uploader /etc/nginx/sites-enabled/sharex-uploader
        if [[ -e /etc/nginx/sites-enabled/default ]]; then
            run_root rm -f /etc/nginx/sites-enabled/default
        fi
    fi
    run_root nginx -t || die 'Die Nginx-Konfiguration ist ungültig.'
    if command -v systemctl >/dev/null 2>&1; then
        run_root systemctl reload nginx || run_root systemctl restart nginx
    else
        run_root service nginx reload || run_root service nginx restart
    fi
}

# After Certbot rewrites the site file, re-apply body size + security headers safely.
nginx_harden_site() {
    local site_path body_size temporary_path
    site_path=/etc/nginx/sites-available/sharex-uploader
    [[ -f "$site_path" ]] || return 0
    body_size="$(nginx_body_size)"
    temporary_path="$(mktemp)"

    # Normalize upload limit everywhere in the site file.
    sed -E "s/client_max_body_size[[:space:]]+[^;]+;/client_max_body_size ${body_size};/g" "$site_path" > "$temporary_path"
    if ! grep -q 'client_max_body_size' "$temporary_path"; then
        sed -i -E "0,/server_name .*/s//&\n\n    client_max_body_size ${body_size};/" "$temporary_path" || true
    fi

    # Add security headers once if missing.
    if ! grep -q 'X-Content-Type-Options' "$temporary_path"; then
        sed -i -E "0,/server_name .*/s||&\n\n    add_header X-Content-Type-Options \"nosniff\" always;\n    add_header X-Frame-Options \"DENY\" always;\n    add_header Referrer-Policy \"no-referrer\" always;\n    add_header Permissions-Policy \"camera=(), microphone=(), geolocation=()\" always;\n    add_header Strict-Transport-Security \"max-age=31536000; includeSubDomains\" always;|" "$temporary_path" || true
    fi

    if run_root nginx -t -c /etc/nginx/nginx.conf >/dev/null 2>&1; then
        :
    fi
    run_root install -m 0644 "$temporary_path" "$site_path"
    rm -f "$temporary_path"
    if run_root nginx -t; then
        if command -v systemctl >/dev/null 2>&1; then
            run_root systemctl reload nginx || true
        else
            run_root service nginx reload || true
        fi
    else
        printf 'Hinweis: nginx_harden_site konnte die Site nicht neu laden (nginx -t fehlgeschlagen).\n' >&2
    fi
}

harden_host() {
    # Optional host hardening for fresh installs (safe defaults, keeps SSH password auth).
    if ! command -v apt-get >/dev/null 2>&1; then
        return 0
    fi
    printf '\nHärte Host-Sicherheit (UFW, fail2ban, unattended-upgrades)...\n'
    run_root env DEBIAN_FRONTEND=noninteractive apt-get install -y ufw fail2ban unattended-upgrades >/dev/null || true

    if command -v ufw >/dev/null 2>&1; then
        run_root ufw default deny incoming || true
        run_root ufw default allow outgoing || true
        run_root ufw allow 22/tcp || true
        run_root ufw allow 80/tcp || true
        run_root ufw allow 443/tcp || true
        run_root ufw --force enable || true
    fi

    if [[ -d /etc/fail2ban ]]; then
        run_root tee /etc/fail2ban/jail.d/sshd.local >/dev/null <<'EOF'
[sshd]
enabled = true
port = ssh
filter = sshd
logpath = /var/log/auth.log
maxretry = 5
bantime = 1h
findtime = 15m
EOF
        if command -v systemctl >/dev/null 2>&1; then
            run_root systemctl enable --now fail2ban || true
            run_root systemctl restart fail2ban || true
        fi
    fi

    run_root tee /etc/apt/apt.conf.d/20auto-upgrades >/dev/null <<'EOF'
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
APT::Periodic::AutocleanInterval "7";
EOF
}

issue_tls_certificate() {
    printf '\nFordere TLS-Zertifikat für %s an...\n' "$PUBLIC_HOST"
    run_root certbot --nginx --non-interactive --agree-tos --no-eff-email \
        --email "$TLS_EMAIL" --redirect --keep-until-expiring -d "$PUBLIC_HOST" \
        || die 'Certbot konnte kein Zertifikat ausstellen. Prüfe DNS sowie die Freigabe von Port 80.'
}

install_host_dependencies() {
    if command -v docker >/dev/null 2>&1 && compose_available \
        && command -v curl >/dev/null 2>&1 \
        && command -v openssl >/dev/null 2>&1 \
        && command -v nginx >/dev/null 2>&1 \
        && command -v certbot >/dev/null 2>&1; then
        return
    fi

    local package_manager=''
    if command -v apt-get >/dev/null 2>&1; then
        package_manager=apt
    elif command -v dnf >/dev/null 2>&1; then
        package_manager=dnf
    elif command -v yum >/dev/null 2>&1; then
        package_manager=yum
    elif command -v pacman >/dev/null 2>&1; then
        package_manager=pacman
    elif command -v apk >/dev/null 2>&1; then
        package_manager=apk
    elif command -v zypper >/dev/null 2>&1; then
        package_manager=zypper
    else
        die "Kein unterstützter Paketmanager gefunden. Installiere Docker Engine, Docker Compose, curl und openssl manuell."
    fi

    printf 'Installiere fehlende Linux-Abhängigkeiten über %s...\n' "$package_manager"
    case "$package_manager" in
        apt)
            run_root apt-get update
            run_root env DEBIAN_FRONTEND=noninteractive apt-get install -y \
                ca-certificates curl openssl docker.io nginx certbot python3-certbot-nginx
            if ! compose_available; then
                if ! run_root env DEBIAN_FRONTEND=noninteractive apt-get install -y docker-compose-v2; then
                    if ! run_root env DEBIAN_FRONTEND=noninteractive apt-get install -y docker-compose-plugin; then
                        run_root env DEBIAN_FRONTEND=noninteractive apt-get install -y docker-compose
                    fi
                fi
            fi
            ;;
        dnf)
            run_root dnf install -y ca-certificates curl openssl docker nginx certbot python3-certbot-nginx
            if ! compose_available; then
                if ! run_root dnf install -y docker-compose-plugin; then
                    run_root dnf install -y docker-compose
                fi
            fi
            ;;
        yum)
            run_root yum install -y ca-certificates curl openssl docker nginx certbot python3-certbot-nginx
            if ! compose_available; then
                if ! run_root yum install -y docker-compose-plugin; then
                    run_root yum install -y docker-compose
                fi
            fi
            ;;
        pacman)
            run_root pacman -Sy --needed --noconfirm ca-certificates curl openssl docker docker-compose nginx certbot certbot-nginx
            ;;
        apk)
            run_root apk add --no-cache ca-certificates curl openssl docker docker-cli-compose nginx certbot certbot-nginx
            ;;
        zypper)
            run_root zypper --non-interactive install ca-certificates curl openssl docker docker-compose nginx certbot python3-certbot-nginx
            ;;
    esac
}

start_docker() {
    if docker info >/dev/null 2>&1; then
        return
    fi

    if command -v systemctl >/dev/null 2>&1; then
        run_root systemctl enable --now docker || die "Der Docker-Dienst konnte nicht gestartet werden."
    elif command -v service >/dev/null 2>&1; then
        run_root service docker start || die "Der Docker-Dienst konnte nicht gestartet werden."
    fi
}

start_nginx() {
    if command -v systemctl >/dev/null 2>&1; then
        run_root systemctl enable --now nginx || die 'Der Nginx-Dienst konnte nicht gestartet werden.'
    elif command -v service >/dev/null 2>&1; then
        run_root service nginx start || die 'Der Nginx-Dienst konnte nicht gestartet werden.'
    fi
}

install_host_dependencies
start_docker

if docker info >/dev/null 2>&1; then
    DOCKER=(docker)
elif [[ "${EUID:-$(id -u)}" -ne 0 ]] && command -v sudo >/dev/null 2>&1 && sudo docker info >/dev/null 2>&1; then
    DOCKER=(sudo docker)
else
    die "Docker ist nicht erreichbar. Prüfe den Docker-Dienst und ob dein Benutzer Zugriff auf /var/run/docker.sock hat."
fi

if "${DOCKER[@]}" compose version >/dev/null 2>&1; then
    COMPOSE=("${DOCKER[@]}" compose)
elif command -v docker-compose >/dev/null 2>&1 && docker-compose version >/dev/null 2>&1; then
    if [[ "${DOCKER[0]}" == sudo ]]; then
        COMPOSE=(sudo docker-compose)
    else
        COMPOSE=(docker-compose)
    fi
else
    die "Docker Compose wurde nicht gefunden. Bitte das Compose-Plugin installieren."
fi

compose() {
    "${COMPOSE[@]}" "$@"
}

ENV_PATH="$PROJECT_ROOT/.env"
COMPOSE_ENV_ARGS=()
if [[ "${SHAREX_DIRECT_HTTPS:-false}" == true ]]; then
    SETUP_MODE='https'
elif [[ "${SHAREX_WEB_SETUP:-}" == true ]]; then
    SETUP_MODE='web'
elif [[ -f "$ENV_PATH" ]] && grep -q '^PUBLIC_HOST=' "$ENV_PATH" && grep -q '^BASE_URL=https://' "$ENV_PATH"; then
    SETUP_MODE='https'
else
    SETUP_MODE='web'
fi

if [[ "$SETUP_MODE" == web ]]; then
    run_web_setup_mode
else
    start_nginx
    printf '\nHTTPS-Einrichtung\n'
    printf 'Der Installer verwendet Nginx und Certbot. Port 80 wird auf HTTPS umgeleitet, Port 443 erhält automatisch ein Zertifikat.\n'
    configure_https
    printf '.env wurde für https://%s vorbereitet.\n' "$PUBLIC_HOST"
    COMPOSE_ENV_ARGS=(--env-file .env)

    compose "${COMPOSE_ENV_ARGS[@]}" config >/dev/null || die 'Die .env- oder Docker-Compose-Konfiguration ist ungültig.'
    nginx_configure

    printf '\nBaue und starte ShareX Uploader...\n'
    compose "${COMPOSE_ENV_ARGS[@]}" up --build -d || die 'Docker Compose konnte die Anwendung nicht starten.'

    issue_tls_certificate
    nginx_harden_site
    harden_host
fi

if [[ -f "$ENV_PATH" ]]; then
    base_url="$(awk -F= '/^BASE_URL=/{sub(/^BASE_URL=/, ""); gsub(/^"|"$/, ""); print; exit}' .env)"
    display_url="$base_url"
else
    base_url="https://${PUBLIC_HOST}"
    display_url="$base_url"
fi
base_url="${base_url:-https://${PUBLIC_HOST}}"
display_url="${display_url:-$base_url}"

if command -v curl >/dev/null 2>&1; then
    ready=false
    setup_required=false
    for _ in $(seq 1 60); do
        if curl -fsS --max-time 2 "${base_url%/}/api/health" >/dev/null 2>&1; then
            ready=true
            break
        fi
        if setup_status="$(curl -fsS --max-time 2 "${base_url%/}/api/setup/status" 2>/dev/null)" && [[ "$setup_status" == *'"configured":false'* ]]; then
            setup_required=true
            break
        fi
        sleep 2
    done

    if [[ "$setup_required" == true ]]; then
        printf '\nInstallation vorbereitet. Öffne jetzt den Browser-Installer:\n'
        printf '%s/setup\n' "${display_url%/}"
        printf 'Dort werden die Datenbank und der Admin-Zugang eingerichtet.\n'
    elif [[ "$ready" == true ]]; then
        printf '\nInstallation abgeschlossen.\n'
        printf 'App: %s\n' "$base_url"
        printf 'Die ShareX- und Admin-API-Schlüssel werden nur im Web-Setup einmalig angezeigt. Sichere sie dort und teile sie niemals öffentlich.\n'
        printf 'Die ShareX-Konfiguration kann danach im Browser heruntergeladen werden.\n'
    else
        printf '\nContainer wurden gestartet, aber der Health-Check antwortet noch nicht.\n'
        printf 'Status: docker compose --env-file .env ps\n'
        printf 'Logs:   docker compose --env-file .env logs app\n'
    fi
else
    printf '\nContainer wurden gestartet. curl ist nicht installiert, daher wurde kein Health-Check ausgeführt.\n'
    printf 'Status: docker compose --env-file .env ps\n'
fi
