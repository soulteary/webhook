# 配置模版

使用 `-template` [CLI 参数][CLI-ENV] 时，可以将配置文件解析为 Go 模板。除了支持[Go 模板内置的函数和特性][Go-Template]，程序还提供：

- `getenv`：注入环境变量；
- `getenvRequired`：环境变量未设置或为空时拒绝加载配置；
- `getSecret`：从专用 Secret 目录读取一个具名文件；
- `trimSpace`：显式移除模板值首尾的空白，包括文件末尾的换行符。

认证密钥应使用 `getenvRequired` 或 `getSecret`，确保配置缺失时安全失败。

## Secret 文件

`getSecret` 只接受单个文件名，不接受任意路径。默认从 `/run/secrets`
读取，与 Docker 和 Docker Swarm 的 Secret 挂载方式一致。如果 Secret
挂载在其他位置，可以把 `WEBHOOK_SECRET_DIR` 设置为相应的绝对路径。

该函数会拒绝绝对路径、目录分隔符、路径穿越、解析到 Secret 目录外的
文件、非普通文件，以及大于 1 MiB 的文件。函数原样保留文件内容；如果
Secret 文件末尾带换行符，应显式使用 `trimSpace`。

```json
{
  "source": "string",
  "envname": "AUTHORIZATION",
  "name": "Bearer {{ getSecret "webhook-token" | trimSpace | js }}"
}
```

项目不会提供通用的 `cat` 模板函数。把读取范围限制在运维人员控制的
专用目录内，可以避免 Hook 模板变成任意文件读取器。

## 使用示例

在下面的 JSON 示例文件中（YAML 同理），使用了 `payload-hmac-sha1` 匹配规则来选择性执行钩子程序。其中 HMAC 密钥使用 `getenvRequired` 函数从环境变量中获取。

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
            "secret": "{{ getenvRequired "XXXTEST_SECRET" | js }}",
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

[CLI-ENV]: ./Webhook-Parameters.md
[Go-Template]: https://golang.org/pkg/text/template/
