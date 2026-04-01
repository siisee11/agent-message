#!/bin/sh
set -eu

APP_USER="${APP_USER:-app}"
UPLOAD_DIR="${UPLOAD_DIR:-/var/lib/agent-message/uploads}"
WEB_PUSH_STATE_DIR="${WEB_PUSH_STATE_DIR:-/var/lib/agent-message/web-push}"
WEB_PUSH_STATE_FILE="${WEB_PUSH_STATE_DIR}/web-push.env"

running_as_root() {
  [ "$(id -u)" -eq 0 ]
}

ensure_runtime_dir() {
  dir_path="$1"
  mkdir -p "${dir_path}"
  if running_as_root; then
    chown -R "${APP_USER}:${APP_USER}" "${dir_path}"
  fi
}

load_saved_web_push_config() {
  if [ -f "${WEB_PUSH_STATE_FILE}" ]; then
    set -a
    . "${WEB_PUSH_STATE_FILE}"
    set +a
  fi
}

save_web_push_config() {
  ensure_runtime_dir "${WEB_PUSH_STATE_DIR}"
  cat >"${WEB_PUSH_STATE_FILE}" <<EOF
WEB_PUSH_VAPID_PUBLIC_KEY=${WEB_PUSH_VAPID_PUBLIC_KEY}
WEB_PUSH_VAPID_PRIVATE_KEY=${WEB_PUSH_VAPID_PRIVATE_KEY}
WEB_PUSH_SUBJECT=${WEB_PUSH_SUBJECT}
EOF
  chmod 600 "${WEB_PUSH_STATE_FILE}"
  if running_as_root; then
    chown "${APP_USER}:${APP_USER}" "${WEB_PUSH_STATE_FILE}"
  fi
}

set_default_web_push_subject() {
  if [ -n "${WEB_PUSH_SUBJECT:-}" ]; then
    return
  fi
  if [ -n "${APP_HOSTNAME:-}" ]; then
    WEB_PUSH_SUBJECT="https://${APP_HOSTNAME}"
    export WEB_PUSH_SUBJECT
  fi
}

generate_web_push_keys() {
  node_script="$(mktemp)"
  cat >"${node_script}" <<'NODE'
const { generateKeyPairSync } = require('node:crypto')

function shEscape(value) {
  return "'" + String(value).replace(/'/g, "'\\''") + "'"
}

const { privateKey, publicKey } = generateKeyPairSync('ec', { namedCurve: 'prime256v1' })
const privateJWK = privateKey.export({ format: 'jwk' })
const publicJWK = publicKey.export({ format: 'jwk' })
const x = Buffer.from(publicJWK.x, 'base64url')
const y = Buffer.from(publicJWK.y, 'base64url')
const publicKeyBytes = Buffer.concat([Buffer.from([0x04]), x, y])

console.log("WEB_PUSH_VAPID_PUBLIC_KEY=" + shEscape(publicKeyBytes.toString('base64url')))
console.log("WEB_PUSH_VAPID_PRIVATE_KEY=" + shEscape(privateJWK.d))
NODE
  generated_keys="$(node "${node_script}")"
  rm -f "${node_script}"
  eval "${generated_keys}"
  export WEB_PUSH_VAPID_PUBLIC_KEY WEB_PUSH_VAPID_PRIVATE_KEY
}

if [ -z "${WEB_PUSH_VAPID_PUBLIC_KEY:-}" ] && [ -z "${WEB_PUSH_VAPID_PRIVATE_KEY:-}" ] && [ -z "${WEB_PUSH_SUBJECT:-}" ]; then
  load_saved_web_push_config
fi

ensure_runtime_dir "${UPLOAD_DIR}"
ensure_runtime_dir "${WEB_PUSH_STATE_DIR}"

set_default_web_push_subject

if [ -z "${WEB_PUSH_VAPID_PUBLIC_KEY:-}" ] && [ -z "${WEB_PUSH_VAPID_PRIVATE_KEY:-}" ] && [ "${AGENT_AUTO_GENERATE_WEB_PUSH:-0}" = "1" ]; then
  generate_web_push_keys
  set_default_web_push_subject
  if [ -z "${WEB_PUSH_SUBJECT:-}" ]; then
    echo "WEB_PUSH_SUBJECT or APP_HOSTNAME is required when auto-generating web push keys." >&2
    exit 1
  fi
  save_web_push_config
fi

if [ -n "${WEB_PUSH_VAPID_PUBLIC_KEY:-}" ] || [ -n "${WEB_PUSH_VAPID_PRIVATE_KEY:-}" ] || [ -n "${WEB_PUSH_SUBJECT:-}" ]; then
  if [ -z "${WEB_PUSH_VAPID_PUBLIC_KEY:-}" ] || [ -z "${WEB_PUSH_VAPID_PRIVATE_KEY:-}" ] || [ -z "${WEB_PUSH_SUBJECT:-}" ]; then
    echo "WEB_PUSH_VAPID_PUBLIC_KEY, WEB_PUSH_VAPID_PRIVATE_KEY, and WEB_PUSH_SUBJECT must be set together." >&2
    exit 1
  fi
fi

if running_as_root; then
  exec su-exec "${APP_USER}:${APP_USER}" /usr/local/bin/agent-message-server
fi

exec /usr/local/bin/agent-message-server
