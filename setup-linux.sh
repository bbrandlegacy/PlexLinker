#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "${EUID}" -eq 0 ]]; then
  echo "Run this installer as the Linux account that will use PlexLinker, not as root." >&2
  exit 1
fi

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
architecture="$(uname -m)"
case "${architecture}" in
  x86_64|amd64) bundled_name="plexlinker-linux-amd64" ;;
  aarch64|arm64) bundled_name="plexlinker-linux-arm64" ;;
  *) echo "Unsupported Linux architecture: ${architecture}" >&2; exit 1 ;;
esac
source_binary="${PLEXLINKER_SOURCE_EXE:-${script_dir}/${bundled_name}}"
if [[ ! -f "${source_binary}" ]]; then
  echo "PlexLinker binary not found: ${source_binary}" >&2
  exit 1
fi

echo "PlexLinker Linux setup"
echo "  Source: ${source_binary}"
echo "  Install: /usr/local/bin/plexlinker"
echo "  User: ${USER}"
echo "  Persistence: systemd user lingering plus PlexLinker aria2 service"
echo "The sudo password is handled only by sudo and is never read or stored by this script."

if [[ "${PLEXLINKER_SETUP_CONFIRMED:-0}" != "1" ]]; then
  read -r -p "Proceed with these changes? [y/N]: " answer
  case "${answer,,}" in
    y|yes) ;;
    *) echo "Setup cancelled."; exit 1 ;;
  esac
fi

sudo -v
keepalive_pid=""
cleanup() {
  if [[ -n "${keepalive_pid}" ]]; then
    kill "${keepalive_pid}" 2>/dev/null || true
  fi
  sudo -k
}
trap cleanup EXIT
while true; do
  sudo -n true 2>/dev/null || exit
  sleep 45
done &
keepalive_pid=$!

if ! command -v aria2c >/dev/null 2>&1; then
  if command -v apt-get >/dev/null 2>&1; then
    sudo apt-get update
    sudo apt-get install --yes aria2
  elif command -v dnf >/dev/null 2>&1; then
    sudo dnf install --assumeyes aria2
  elif command -v yum >/dev/null 2>&1; then
    sudo yum install --assumeyes aria2
  elif command -v zypper >/dev/null 2>&1; then
    sudo zypper --non-interactive install aria2
  elif command -v pacman >/dev/null 2>&1; then
    sudo pacman --sync --needed --noconfirm aria2
  else
    echo "No supported package manager found. Install aria2 and rerun setup." >&2
    exit 1
  fi
fi

sudo install -m 0755 "${source_binary}" /usr/local/bin/plexlinker
if command -v loginctl >/dev/null 2>&1 && [[ -d /run/systemd/system ]]; then
  sudo loginctl enable-linger "${USER}"
else
  echo "systemd-logind is unavailable; PlexLinker will use aria2 daemon fallback mode."
fi

config_root="${XDG_CONFIG_HOME:-${HOME}/.config}/PlexLinker"
mkdir -p "${config_root}"
touch "${config_root}/setup-complete"
chmod 600 "${config_root}/setup-complete"

command -v aria2c >/dev/null
/usr/local/bin/plexlinker --version
echo "Setup complete. Run: plexlinker"
