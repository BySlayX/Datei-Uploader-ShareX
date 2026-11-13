#!/usr/bin/env bash
set -Eeuo pipefail

umask 077

REPOSITORY_URL="${SHAREX_REPOSITORY_URL:-https://github.com/BySlayX/Datei-Uploader-ShareX.git}"
REPOSITORY_BRANCH="${SHAREX_REPOSITORY_BRANCH:-main}"
INSTALL_DIR="${SHAREX_INSTALL_DIR:-/opt/sharex-uploader}"
PARENT_DIR="$(dirname -- "$INSTALL_DIR")"
STAGING_DIR=""
UPDATE_MODE=false

die() {
    printf 'Fehler: %s\n' "$1" >&2
    exit 1
}

cleanup() {
    if [[ -n "$STAGING_DIR" && -d "$STAGING_DIR" ]]; then
        rm -rf -- "$STAGING_DIR"
    fi
}
trap cleanup EXIT

start_project_installer() {
    printf 'Starte den Server-Installer aus %s …\n\n' "$INSTALL_DIR"
    cd -- "$INSTALL_DIR"
    if [[ "$UPDATE_MODE" == true ]]; then
        if [[ -t 0 || -t 1 || -t 2 ]]; then
            exec env SHAREX_UPDATE_ONLY=true bash "$INSTALL_DIR/install.sh" </dev/tty
        fi
        exec env SHAREX_UPDATE_ONLY=true bash "$INSTALL_DIR/install.sh"
    fi
    if [[ -f "$INSTALL_DIR/.env" ]] && grep -q '^PUBLIC_HOST=' "$INSTALL_DIR/.env" && grep -q '^BASE_URL=https://' "$INSTALL_DIR/.env"; then
        if [[ -t 0 || -t 1 || -t 2 ]]; then
            exec bash "$INSTALL_DIR/install.sh" </dev/tty
        fi
        exec bash "$INSTALL_DIR/install.sh"
    fi
    if [[ -t 0 || -t 1 || -t 2 ]]; then
        exec env SHAREX_WEB_SETUP=1 bash "$INSTALL_DIR/install.sh" </dev/tty
    fi
    exec env SHAREX_WEB_SETUP=1 bash "$INSTALL_DIR/install.sh"
}

validate_repository() {
    local repository_path="$1"
    [[ -f "$repository_path/install.sh" ]] || die 'install.sh wurde im Repository nicht gefunden.'
    [[ -f "$repository_path/docker-compose.yml" ]] || die 'docker-compose.yml wurde im Repository nicht gefunden.'
    [[ -f "$repository_path/Dockerfile" ]] || die 'Dockerfile wurde im Repository nicht gefunden.'
    [[ -f "$repository_path/backend/main.go" ]] || die 'backend/main.go wurde nicht gefunden. Bitte die Ordnerstruktur des Repositorys erhalten.'
    [[ -f "$repository_path/frontend/app.jsx" ]] || die 'frontend/app.jsx wurde nicht gefunden. Bitte die Ordnerstruktur des Repositorys erhalten.'
    [[ -f "$repository_path/frontend/package.json" ]] || die 'frontend/package.json wurde nicht gefunden. Bitte die Ordnerstruktur des Repositorys erhalten.'
}

download_repository() {
    install -d -m 0755 "$PARENT_DIR"
    STAGING_DIR="$(mktemp -d "${INSTALL_DIR}.bootstrap.XXXXXX")"

    if command -v git >/dev/null 2>&1; then
        git clone --depth 1 --branch "$REPOSITORY_BRANCH" "$REPOSITORY_URL" "$STAGING_DIR/repository"
    else
        command -v curl >/dev/null 2>&1 || die 'Git oder curl wird benötigt.'
        command -v tar >/dev/null 2>&1 || die 'Git oder tar wird benötigt.'
        archive_url="${REPOSITORY_URL%.git}/archive/refs/heads/${REPOSITORY_BRANCH}.tar.gz"
        install -d -m 0755 "$STAGING_DIR/repository"
        curl --fail --location --silent --show-error "$archive_url" | tar --extract --gzip --strip-components=1 --directory "$STAGING_DIR/repository"
    fi

    validate_repository "$STAGING_DIR/repository"
}

repository_revision() {
    local repository_path="$1"
    git -C "$repository_path" rev-parse HEAD 2>/dev/null || true
}

installed_revision() {
    if [[ -f "$INSTALL_DIR/.sharex-revision" ]]; then
        tr -d '[:space:]' < "$INSTALL_DIR/.sharex-revision"
    else
        repository_revision "$INSTALL_DIR"
    fi
}

write_revision_marker() {
    local revision="$1"
    [[ -n "$revision" ]] || return 0
    printf '%s\n' "$revision" > "$INSTALL_DIR/.sharex-revision"
    chmod 600 "$INSTALL_DIR/.sharex-revision"
}

update_project_files() {
    local item item_name
    shopt -s dotglob nullglob
    for item in "$STAGING_DIR/repository"/*; do
        item_name="${item##*/}"
        case "$item_name" in
            .env|.git|data|uploads)
                continue
                ;;
        esac
        cp -a -- "$item" "$INSTALL_DIR/"
    done
    shopt -u dotglob nullglob
    chmod 0755 "$INSTALL_DIR/install.sh"
}

[[ "${EUID:-$(id -u)}" -eq 0 ]] || die 'Bitte mit sudo oder als root starten.'

case "$INSTALL_DIR" in
    /opt/*|/var/www/*) ;;
    *) die "Unsicherer Installationspfad: $INSTALL_DIR" ;;
esac

command -v mktemp >/dev/null 2>&1 || die 'mktemp wurde nicht gefunden.'
command -v install >/dev/null 2>&1 || die 'install wurde nicht gefunden.'

if [[ -e "$INSTALL_DIR" ]]; then
    [[ -d "$INSTALL_DIR" ]] || die "Der Installationspfad ist kein Verzeichnis: $INSTALL_DIR"
    [[ -f "$INSTALL_DIR/install.sh" ]] || die "Die vorhandene Projektkopie ist unvollständig: install.sh fehlt."
    [[ -f "$INSTALL_DIR/docker-compose.yml" ]] || die "Die vorhandene Projektkopie ist unvollständig: docker-compose.yml fehlt."
    [[ -f "$INSTALL_DIR/Dockerfile" ]] || die "Die vorhandene Projektkopie ist unvollständig: Dockerfile fehlt."
    printf 'Vorhandene Projektkopie erkannt. Suche nach Updates in GitHub …\n'
    current_revision="$(installed_revision)"
    download_repository
    remote_revision="$(repository_revision "$STAGING_DIR/repository")"

    if [[ -n "$current_revision" && -n "$remote_revision" && "$current_revision" == "$remote_revision" ]]; then
        rm -rf -- "$STAGING_DIR"
        STAGING_DIR=""
        printf 'Die Installation ist bereits aktuell (%s).\n' "${current_revision:0:12}"
        exit 0
    fi

    printf 'Neues Update gefunden. Aktualisiere nur den Projektcode; .env, Datenbank und Uploads bleiben erhalten …\n'
    update_project_files
    write_revision_marker "$remote_revision"
    rm -rf -- "$STAGING_DIR"
    STAGING_DIR=""
    UPDATE_MODE=true
    printf 'Projektcode wurde aktualisiert.\n'
    start_project_installer
fi

download_repository

chmod 0755 "$STAGING_DIR/repository/install.sh"
mv -- "$STAGING_DIR/repository" "$INSTALL_DIR"
STAGING_DIR=""

write_revision_marker "$(repository_revision "$INSTALL_DIR")"

printf 'Projekt installiert unter %s\n' "$INSTALL_DIR"
start_project_installer
