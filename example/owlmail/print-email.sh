#!/bin/sh
set -eu

printf '%s\n' \
  "OwlMail webhook event" \
  "  event: ${OWLMAIL_EVENT:-}" \
  "  id: ${OWLMAIL_EMAIL_ID:-}" \
  "  title: ${OWLMAIL_TITLE:-}" \
  "  from: ${OWLMAIL_FROM:-}" \
  "  to: ${OWLMAIL_TO:-}" \
  "  received: ${OWLMAIL_RECEIVED_AT:-}" \
  "  message: ${OWLMAIL_MESSAGE:-}"
