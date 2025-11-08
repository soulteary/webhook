# 配置模版

当我们使用 `-template` [CLI 参数][CLI-ENV] 时，可以将启用将配置文件解析为 Go 模版的功能。除了支持[Go 模板内置的函数和特性][Go-Template] 之外，程序还额外提供了一个 `getenv` 的模板函数，能够将各种环境变量注入到配置文件中。

## 使用示例

在下面的 JSON 示例文件中（YAML同理），使用了 `payload-hmac-sha1` 匹配规则来选择性执行钩子程序。其中 HMAC 密钥使用了 `getenv` 函数来从环境变量中获取。

除此之外，还通过了 Go 模版内置的 `js` 和管道传书语法来确保输出的结果是 `JavaScript / JSON` 字符串。

```json
[
  {
    "id": "webhook",
    "execute-command": "/home/adnan/redeploy-go-webhook.sh",
    "command-working-directory": "/home/adnan/go",
    "response-message": "I got the payload!",
    "response-headers":
    [
      {
        "name": "Access-Control-Allow-Origin",
        "value": "*"
      }
    ],
    "pass-arguments-to-command":
    [
      {
        "source": "payload",
        "name": "head_commit.id"
      },
      {
        "source": "payload",
        "name": "pusher.name"
      },
      {
        "source": "payload",
        "name": "pusher.email"
      }
    ],
    "trigger-rule":
    {
      "and":
      [
        {
          "match":
          {
            "type": "payload-hmac-sha1",
            "secret": "{{ getenv "XXXTEST_SECRET" | js }}",
            "parameter":
            {
              "source": "header",
              "name": "X-Hub-Signature"
            }
          }
        },
        {
          "match":
          {
            "type": "value",
            "value": "refs/heads/master",
            "parameter":
            {
              "source": "payload",
              "name": "ref"
            }
          }
        }
      ]
    }
  }
]

```

[CLI-ENV]: ./CLI-ENV.md
[Go-Template]: https://golang.org/pkg/text/template/
