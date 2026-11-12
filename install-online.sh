#!/usr/bin/env bash
set -Eeuo pipefail

umask 077

REPOSITORY_URL="${SHAREX_REPOSITORY_URL:-https://github.com/BySlayX/Datei-Uploader-ShareX.git}"
REPOSITORY_BRANCH="${SHAREX_REPOSITORY_BRANCH:-main}"
INSTALL_DIR="${SHAREX_INSTALL_DIR:-/opt/sharex-uploader}"
PARENT_DIR="$(dirname -- "$INSTALL_DIR")"
STAGING_DIR=""

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

[[ "${EUID:-$(id -u)}" -eq 0 ]] || die 'Bitte mit sudo oder als root starten.'

case "$INSTALL_DIR" in
    /opt/*|/var/www/*) ;;
    *) die "Unsicherer Installationspfad: $INSTALL_DIR" ;;
esac

if [[ -e "$INSTALL_DIR" ]]; then
    die "Der Installationsordner existiert bereits: $INSTALL_DIR. Eine bestehende Installation wird nicht überschrieben."
fi

command -v mktemp >/dev/null 2>&1 || die 'mktemp wurde nicht gefunden.'
command -v install >/dev/null 2>&1 || die 'install wurde nicht gefunden.'

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

[[ -f "$STAGING_DIR/repository/install.sh" ]] || die 'install.sh wurde im Repository nicht gefunden.'
[[ -f "$STAGING_DIR/repository/docker-compose.yml" ]] || die 'docker-compose.yml wurde im Repository nicht gefunden.'
[[ -f "$STAGING_DIR/repository/Dockerfile" ]] || die 'Dockerfile wurde im Repository nicht gefunden.'

chmod 0755 "$STAGING_DIR/repository/install.sh"
mv -- "$STAGING_DIR/repository" "$INSTALL_DIR"
STAGING_DIR=""

printf 'Projekt installiert unter %s\n' "$INSTALL_DIR"
printf 'Starte jetzt den Server-Installer …\n\n'
cd -- "$INSTALL_DIR"
if [[ -r /dev/tty ]]; then
    exec bash "$INSTALL_DIR/install.sh" </dev/tty
fi
exec bash "$INSTALL_DIR/install.sh"
