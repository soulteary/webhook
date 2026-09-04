#!/bin/sh
set -eu

printf '%s\n' \
  "OwlMail webhook event" \
  "  event: ${OWLMAIL_EVENT:-}" \
  "  email id: ${OWLMAIL_EMAIL_ID:-}" \
  "  delivery id: ${OWLMAIL_DELIVERY_ID:-}" \
  "  title: ${OWLMAIL_TITLE:-}" \
  "  from: ${OWLMAIL_FROM:-}" \
  "  to: ${OWLMAIL_TO:-}" \
  "  received: ${OWLMAIL_RECEIVED_AT:-}" \
  "  message: ${OWLMAIL_MESSAGE:-}"
