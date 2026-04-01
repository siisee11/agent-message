#!/bin/sh
set -eu

WEB_PUSH_STATE_DIR="${WEB_PUSH_STATE_DIR:-/var/lib/agent-message/web-push}"
WEB_PUSH_STATE_FILE="${WEB_PUSH_STATE_DIR}/web-push.env"

load_saved_web_push_config() {
  if [ -f "${WEB_PUSH_STATE_FILE}" ]; then
    set -a
    . "${WEB_PUSH_STATE_FILE}"
    set +a
  fi
}

save_web_push_config() {
  mkdir -p "${WEB_PUSH_STATE_DIR}"
  cat >"${WEB_PUSH_STATE_FILE}" <<EOF
WEB_PUSH_VAPID_PUBLIC_KEY=${WEB_PUSH_VAPID_PUBLIC_KEY}
WEB_PUSH_VAPID_PRIVATE_KEY=${WEB_PUSH_VAPID_PRIVATE_KEY}
WEB_PUSH_SUBJECT=${WEB_PUSH_SUBJECT}
EOF
  chmod 600 "${WEB_PUSH_STATE_FILE}"
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
  eval "$(
    node <<'NODE'
const { generateKeyPairSync } = require('node:crypto')

function shEscape(value) {
  return `'${String(value).replace(/'/g, `'\\''`)}'`
}

const { privateKey, publicKey } = generateKeyPairSync('ec', { namedCurve: 'prime256v1' })
const privateJWK = privateKey.export({ format: 'jwk' })
const publicJWK = publicKey.export({ format: 'jwk' })
const x = Buffer.from(publicJWK.x, 'base64url')
const y = Buffer.from(publicJWK.y, 'base64url')
const publicKeyBytes = Buffer.concat([Buffer.from([0x04]), x, y])

console.log(`WEB_PUSH_VAPID_PUBLIC_KEY=${shEscape(publicKeyBytes.toString('base64url'))}`)
console.log(`WEB_PUSH_VAPID_PRIVATE_KEY=${shEscape(privateJWK.d)}`)
NODE
  )"
  export WEB_PUSH_VAPID_PUBLIC_KEY WEB_PUSH_VAPID_PRIVATE_KEY
}

if [ -z "${WEB_PUSH_VAPID_PUBLIC_KEY:-}" ] && [ -z "${WEB_PUSH_VAPID_PRIVATE_KEY:-}" ] && [ -z "${WEB_PUSH_SUBJECT:-}" ]; then
  load_saved_web_push_config
fi

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

exec /usr/local/bin/agent-message-server
