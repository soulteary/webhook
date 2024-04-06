#!/bin/sh

if [ -n "$TEXT" ]; then
    curl -X POST -H "Content-Type: application/json" --data {\"msg_type\":\"text\",\"content\":{\"text\":\"$TEXT\"}} \
        https://open.feishu.cn/open-apis/bot/v2/hook/6dca9854-381a-4bb9-a87b-33a222833e04
    echo "Send message successfully".
else
  echo "TEXT is empty"
fi
