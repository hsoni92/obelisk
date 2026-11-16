# Shared install/uninstall path defaults (source only; no side effects).
# shellcheck shell=bash
OBELISK_ROOT="${OBELISK_ROOT:-/opt/obelisk}"
OBELISK_CONFIG="${OBELISK_CONFIG:-/etc/obelisk/obelisk.yaml}"
OBELISK_NAMESPACE="${OBELISK_NAMESPACE:-obelisk}"
OBELISK_CLI_PATH="${OBELISK_CLI_PATH:-/usr/local/bin/obelisk}"
OBELISK_INSTALL_LOG="${OBELISK_INSTALL_LOG:-/var/log/obelisk-install.log}"
OBELISK_UNINSTALL_LOG="${OBELISK_UNINSTALL_LOG:-/var/log/obelisk-uninstall.log}"
OBELISK_LIB_DIR="${OBELISK_LIB_DIR:-/var/lib/obelisk}"
OBELISK_ETC_DIR="${OBELISK_ETC_DIR:-/etc/obelisk}"
OBELISK_UNINSTALL_META="${OBELISK_UNINSTALL_META:-${OBELISK_LIB_DIR}/.uninstall-meta.json}"
OBELISK_KUBECONFIG_DEFAULT="${KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"