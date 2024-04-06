# 钩子示例

我们可以在 JSON 或者 YAML 文件中定义钩子对象，具体定义可参考[钩子定义]文档。

🌱 此页面仍在持续建议，欢迎贡献你的力量。

## 目录

* [Incoming Github webhook](#incoming-github-webhook)
* [Incoming Bitbucket webhook](#incoming-bitbucket-webhook)
* [Incoming Gitlab webhook](#incoming-gitlab-webhook)
* [Incoming Gogs webhook](#incoming-gogs-webhook)
* [Incoming Gitea webhook](#incoming-gitea-webhook)
* [Slack slash command](#slack-slash-command)
* [A simple webhook with a secret key in GET query](#a-simple-webhook-with-a-secret-key-in-get-query)
* [JIRA Webhooks](#jira-webhooks)
* [Pass File-to-command sample](#pass-file-to-command-sample)
* [Incoming Scalr Webhook](#incoming-scalr-webhook)
* [Travis CI webhook](#travis-ci-webhook)
* [XML Payload](#xml-payload)
* [Multipart Form Data](#multipart-form-data)
* [Pass string arguments to command](#pass-string-arguments-to-command)
* [Receive Synology DSM notifications](#receive-synology-notifications)

## Incoming Github webhook

```json
[
  {
    "id": "webhook",
    "execute-command": "/home/adnan/redeploy-go-webhook.sh",
    "command-working-directory": "/home/adnan/go",
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
            "secret": "mysecret",
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

## Incoming Bitbucket webhook


Bitbucket 不会将任何密钥传递回 WebHook。[根据该产品文档](https://support.atlassian.com/organization-administration/docs/ip-addresses-and-domains-for-atlassian-cloud-products/#Outgoing-Connections)，为了确保调用发起方是 Bitbucket，我们需要设置一组 IP 白名单：

```json
[
  {
    "id": "webhook",
    "execute-command": "/home/adnan/redeploy-go-webhook.sh",
    "command-working-directory": "/home/adnan/go",
    "pass-arguments-to-command":
    [
      {
        "source": "payload",
        "name": "actor.username"
      }
    ],
    "trigger-rule":
    {
      "or":
      [
        { "match": { "type": "ip-whitelist", "ip-range": "13.52.5.96/28" } },
        { "match": { "type": "ip-whitelist", "ip-range": "13.236.8.224/28" } },
        { "match": { "type": "ip-whitelist", "ip-range": "18.136.214.96/28" } },
        { "match": { "type": "ip-whitelist", "ip-range": "18.184.99.224/28" } },
        { "match": { "type": "ip-whitelist", "ip-range": "18.234.32.224/28" } },
        { "match": { "type": "ip-whitelist", "ip-range": "18.246.31.224/28" } },
        { "match": { "type": "ip-whitelist", "ip-range": "52.215.192.224/28" } },
        { "match": { "type": "ip-whitelist", "ip-range": "104.192.137.240/28" } },
        { "match": { "type": "ip-whitelist", "ip-range": "104.192.138.240/28" } },
        { "match": { "type": "ip-whitelist", "ip-range": "104.192.140.240/28" } },
        { "match": { "type": "ip-whitelist", "ip-range": "104.192.142.240/28" } },
        { "match": { "type": "ip-whitelist", "ip-range": "104.192.143.240/28" } },
        { "match": { "type": "ip-whitelist", "ip-range": "185.166.143.240/28" } },
        { "match": { "type": "ip-whitelist", "ip-range": "185.166.142.240/28" } }
      ]
    }
  }
]
```

## Incoming GitLab Webhook

GitLab 提供了多种事件类型支持。可以参考下面的文档 [gitlab-ce/integrations/webhooks](https://gitlab.com/gitlab-org/gitlab-ce/blob/master/doc/user/project/integrations/webhooks.md) 来进行设置。

通过配置 `payload` 和钩子规则，来访问请求体中的数据：

```json
[
  {
    "id": "redeploy-webhook",
    "execute-command": "/home/adnan/redeploy-go-webhook.sh",
    "command-working-directory": "/home/adnan/go",
    "pass-arguments-to-command":
    [
      {
        "source": "payload",
        "name": "user_name"
      }
    ],
    "response-message": "Executing redeploy script",
    "trigger-rule":
    {
      "match":
      {
        "type": "value",
        "value": "<YOUR-GENERATED-TOKEN>",
        "parameter":
        {
          "source": "header",
          "name": "X-Gitlab-Token"
        }
      }
    }
  }
]
```

## Incoming Gogs webhook

```json
[
  {
    "id": "webhook",
    "execute-command": "/home/adnan/redeploy-go-webhook.sh",
    "command-working-directory": "/home/adnan/go",
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
            "type": "payload-hmac-sha256",
            "secret": "mysecret",
            "parameter":
            {
              "source": "header",
              "name": "X-Gogs-Signature"
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

## Incoming Gitea webhook

```json
[
  {
    "id": "webhook",
    "execute-command": "/home/adnan/redeploy-go-webhook.sh",
    "command-working-directory": "/home/adnan/go",
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
            "type": "value",
            "value": "mysecret",
            "parameter":
            {
              "source": "payload",
              "name": "secret"
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

## Slack slash command

```json
[
  {
    "id": "redeploy-webhook",
    "execute-command": "/home/adnan/redeploy-go-webhook.sh",
    "command-working-directory": "/home/adnan/go",
    "response-message": "Executing redeploy script",
    "trigger-rule":
    {
      "match":
      {
        "type": "value",
        "value": "<YOUR-GENERATED-TOKEN>",
        "parameter":
        {
          "source": "payload",
          "name": "token"
        }
      }
    }
  }
]
```

## A simple webhook with a secret key in GET query

__因为安全性比较低，不推荐在生产环境使用__

`example.com:9000/hooks/simple-one` - 将不会被调用
`example.com:9000/hooks/simple-one?token=42` - 将会被调用

```json
[
  {
    "id": "simple-one",
    "execute-command": "/path/to/command.sh",
    "response-message": "Executing simple webhook...",
    "trigger-rule":
    {
      "match":
      {
        "type": "value",
        "value": "42",
        "parameter":
        {
          "source": "url",
          "name": "token"
        }
      }
    }
  }
]
```

## JIRA Webhooks

[来自网友 @perfecto25 的教程](https://sites.google.com/site/mrxpalmeiras/more/jira-webhooks)

## Pass File-to-command sample

### Webhook configuration

```json
[
  {
    "id": "test-file-webhook",
    "execute-command": "/bin/ls",
    "command-working-directory": "/tmp",
    "pass-file-to-command":
    [
      {
        "source": "payload",
        "name": "binary",
        "envname": "ENV_VARIABLE", // to use $ENV_VARIABLE in execute-command
                                   // if not defined, $HOOK_BINARY will be provided
        "base64decode": true,      // defaults to false
      }
    ],
    "include-command-output-in-response": true
  }
]
```

### Sample client usage 

将下面的内容保存为 `testRequest.json` 文件：

```json
{"binary":"iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAGXRFWHRTb2Z0d2FyZQBBZG9iZSBJbWFnZVJlYWR5ccllPAAAA2lpVFh0WE1MOmNvbS5hZG9iZS54bXAAAAAAADw/eHBhY2tldCBiZWdpbj0i77u/IiBpZD0iVzVNME1wQ2VoaUh6cmVTek5UY3prYzlkIj8+IDx4OnhtcG1ldGEgeG1sbnM6eD0iYWRvYmU6bnM6bWV0YS8iIHg6eG1wdGs9IkFkb2JlIFhNUCBDb3JlIDUuMC1jMDYwIDYxLjEzNDc3NywgMjAxMC8wMi8xMi0xNzozMjowMCAgICAgICAgIj4gPHJkZjpSREYgeG1sbnM6cmRmPSJodHRwOi8vd3d3LnczLm9yZy8xOTk5LzAyLzIyLXJkZi1zeW50YXgtbnMjIj4gPHJkZjpEZXNjcmlwdGlvbiByZGY6YWJvdXQ9IiIgeG1sbnM6eG1wUmlnaHRzPSJodHRwOi8vbnMuYWRvYmUuY29tL3hhcC8xLjAvcmlnaHRzLyIgeG1sbnM6eG1wTU09Imh0dHA6Ly9ucy5hZG9iZS5jb20veGFwLzEuMC9tbS8iIHhtbG5zOnN0UmVmPSJodHRwOi8vbnMuYWRvYmUuY29tL3hhcC8xLjAvc1R5cGUvUmVzb3VyY2VSZWYjIiB4bWxuczp4bXA9Imh0dHA6Ly9ucy5hZG9iZS5jb20veGFwLzEuMC8iIHhtcFJpZ2h0czpNYXJrZWQ9IkZhbHNlIiB4bXBNTTpEb2N1bWVudElEPSJ4bXAuZGlkOjEzMTA4RDI0QzMxQjExRTBCMzYzRjY1QUQ1Njc4QzFBIiB4bXBNTTpJbnN0YW5jZUlEPSJ4bXAuaWlkOjEzMTA4RDIzQzMxQjExRTBCMzYzRjY1QUQ1Njc4QzFBIiB4bXA6Q3JlYXRvclRvb2w9IkFkb2JlIFBob3Rvc2hvcCBDUzMgV2luZG93cyI+IDx4bXBNTTpEZXJpdmVkRnJvbSBzdFJlZjppbnN0YW5jZUlEPSJ1dWlkOkFDMUYyRTgzMzI0QURGMTFBQUI4QzUzOTBEODVCNUIzIiBzdFJlZjpkb2N1bWVudElEPSJ1dWlkOkM5RDM0OTY2NEEzQ0REMTFCMDhBQkJCQ0ZGMTcyMTU2Ii8+IDwvcmRmOkRlc2NyaXB0aW9uPiA8L3JkZjpSREY+IDwveDp4bXBtZXRhPiA8P3hwYWNrZXQgZW5kPSJyIj8+IBFgEwAAAmJJREFUeNqkk89rE1EQx2d/NNq0xcYYayPYJDWC9ODBsKIgAREjBmvEg2cvHnr05KHQ9iB49SL+/BMEfxBQKHgwCEbTNNIYaqgaoanFJi+rcXezye4689jYkIMIDnx47837zrx583YFx3Hgf0xA6/dJyAkkgUy4vgryAnmNWH9L4EVmotFoKplMHgoGg6PkrFarjXQ6/bFcLj/G5W1E+3NaX4KZeDx+dX5+7kg4HBlmrC6JoiDFYrGhROLM/mp1Y6JSqdCd3/SW0GUqEAjkl5ZyHTSHKBQKnO6a9khD2m5cr91IJBJ1VVWdiM/n6LruNJtNDs3JR3ukIW03SHTHi8iVsbG9I51OG1bW16HVasHQZopDc/JZVgdIQ1o3BmTkEnJXURS/KIpgGAYPkCQJPi0u8uzDKQN0XQPbtgE1MmrHs9nsfSqAEjxCNtHxZHLy4G4smUQgyzL4LzOegDGGp1ucVqsNqKVrpJCM7F4hg6iaZvhqtZrg8XjA4xnAU3XeKLqWaRImoIZeQXVjQO5pYp4xNVirsR1erxer2O4yfa227WCwhtWoJmn7m0h270NxmemFW4706zMm8GCgxBGEASCfhnukIW03iFdQnOPz0LNKp3362JqQzSw4u2LXBe+Bs3xD+/oc1NxN55RiC9fOme0LEQiRf2rBzaKEeJJ37ZWTVunBeGN2WmQjg/DeLTVP89nzAive2dMwlo9bpFVC2xWMZr+A720FVn88fAUb3wDMOjyN7YNc6TvUSHQ4AH6TOUdLL7em68UtWPsJqxgTpgeiLu1EBt1R+Me/mF7CQPTfAgwAGxY2vOTrR3oAAAAASUVORK5CYII="}
```

使用 `curl` 来执行携带上面数据的访问请求：

```bash
#!/bin/bash
curl -H "Content-Type:application/json" -X POST -d @testRequest.json \
http://localhost:9000/hooks/test-file-webhook
```

或者，你也可以使用 [jpmens/jo](https://github.com/jpmens/jo) 这个工具，在一行命令中展开 JSON 的数据，传递给 WebHook：

```bash
jo binary=%filename.zip | curl -H "Content-Type:application/json" -X POST -d @- \
http://localhost:9000/hooks/test-file-webhook
```

## Incoming Scalr Webhook

Scalr 根据具体事件(如主机启动、宕机) 向我们配置的 WebHook URL 端点发起请求调用，告知事件的发生。

Scalr 会为每个配置的钩子地址分配一个唯一的签名密钥。

你可以参考这个文档来了解[如何在 Scalr 中配置网络钩子](https://scalr-wiki.atlassian.net/wiki/spaces/docs/pages/6193173/Webhooks。想要使用 Scalr，我们需要配置钩子匹配规则，将匹配类型设置为 `"scalr-signature"`。

[来自网友 @hassanbabaie 的教程]

```json
[
    {
        "id": "redeploy-webhook",
        "execute-command": "/home/adnan/redeploy-go-webhook.sh",
        "command-working-directory": "/home/adnan/go",
        "include-command-output-in-response": true,
        "trigger-rule": 
		{
            "match": 
			{
                "type": "scalr-signature",
                "secret": "Scalr-provided signing key"
            }
        },
        "pass-environment-to-command": 
		[
            {
                "envname": "EVENT_NAME",
                "source": "payload",
                "name": "eventName"
            },
            {
                "envname": "SERVER_HOSTNAME",
                "source": "payload",
                "name": "data.SCALR_SERVER_HOSTNAME"
            }
        ]
    }
]

```

## Travis CI webhook

Travis 会以 `payload=<JSON_STRING>` 的形式向 WebHook 发送消息。所以，我们需要解析 JSON 中的数据：

```json
[
  {
    "id": "deploy",
    "execute-command": "/root/my-server/deployment.sh",
    "command-working-directory": "/root/my-server",
    "parse-parameters-as-json": [
      {
        "source": "payload",
        "name": "payload"
      }
    ],
    "trigger-rule":
    {
      "and":
      [
        {
          "match":
          {
            "type": "value",
            "value": "passed",
            "parameter": {
              "name": "payload.state",
              "source": "payload"
            }
          }
        },
        {
          "match":
          {
            "type": "value",
            "value": "master",
            "parameter": {
              "name": "payload.branch",
              "source": "payload"
            }
          }
        }
      ]
    }
  }
]
```

## JSON Array Payload

如果 JSON 内容是一个数组而非对象，WebHook 将解析该数据并将其放入一个名为 `root.` 的对象中。

所以，当我们访问具体数据时，需要使用 `root.` 开头的语法来访问数据。

```json
[
  {
    "email": "example@test.com",
    "timestamp": 1513299569,
    "smtp-id": "<14c5d75ce93.dfd.64b469@ismtpd-555>",
    "event": "processed",
    "category": "cat facts",
    "sg_event_id": "sg_event_id",
    "sg_message_id": "sg_message_id"
  },
  {
    "email": "example@test.com",
    "timestamp": 1513299569,
    "smtp-id": "<14c5d75ce93.dfd.64b469@ismtpd-555>",
    "event": "deferred",
    "category": "cat facts",
    "sg_event_id": "sg_event_id",
    "sg_message_id": "sg_message_id",
    "response": "400 try again later",
    "attempt": "5"
  }
]
```

访问数组中的第二个元素，我们可以这样：

```json
[
  {
    "id": "sendgrid",
    "execute-command": "{{ .Hookecho }}",
    "trigger-rule": {
      "match": {
        "type": "value",
        "parameter": {
          "source": "payload",
          "name": "root.1.event"
        },
        "value": "deferred"
      }
    }
  }
]
```

## XML Payload

假设我们要处理的 XML 数据如下：

```xml
<app>
  <users>
    <user id="1" name="Jeff" />
    <user id="2" name="Sally" />
  </users>
  <messages>
    <message id="1" from_user="1" to_user="2">Hello!!</message>
  </messages>
</app>
```

```json
[
  {
    "id": "deploy",
    "execute-command": "/root/my-server/deployment.sh",
    "command-working-directory": "/root/my-server",
    "trigger-rule": {
      "and": [
        {
          "match": {
            "type": "value",
            "parameter": {
              "source": "payload",
              "name": "app.users.user.0.-name"
            },
            "value": "Jeff"
          }
        },
        {
          "match": {
            "type": "value",
            "parameter": {
              "source": "payload",
              "name": "app.messages.message.#text"
            },
            "value": "Hello!!"
          }
        },
      ],
    }
  }
]
```

## Multipart Form Data

下面是 [Plex Media Server webhook](https://support.plex.tv/articles/115002267687-webhooks/) 的数据示例。

Plex Media Server 在调用 WebHook 时，将发送两类数据：payload 和 thumb，我们只需要关心 payload 部分。

```json
[
  {
    "id": "plex",
    "execute-command": "play-command.sh",
    "parse-parameters-as-json": [
      {
        "source": "payload",
        "name": "payload"
      }
    ],
    "trigger-rule":
    {
      "match":
      {
        "type": "value",
        "parameter": {
          "source": "payload",
          "name": "payload.event"
        },
        "value": "media.play"
      }
    }
  }
]
```

包含多个部分的表单数据体，每个部分都将有一个 `Content-Disposition` 头，例如：

```
Content-Disposition: form-data; name="payload"
Content-Disposition: form-data; name="thumb"; filename="thumb.jpg"
```

我们根据 `Content-Disposition` 值中的 `name` 属性来区分不同部分。

## Pass string arguments to command

想要将简单的字符串作为参数传递给需要执行的命令，需要使用 `string` 参数。

下面的例子中，我们将在传递数据 `pusher.email` 之前，将两个字符串 (`-e` 和 `123123`) 传递给 `execute-command`：

```json
[
  {
    "id": "webhook",
    "execute-command": "/home/adnan/redeploy-go-webhook.sh",
    "command-working-directory": "/home/adnan/go",
    "pass-arguments-to-command":
    [
      {
        "source": "string",
        "name": "-e"
      },
      {
        "source": "string",
        "name": "123123"
      },
      {
        "source": "payload",
        "name": "pusher.email"
      }
    ]
  }
]
```

## Receive Synology DSM notifications

我们可以通过 WebHook 安全的接收来自群晖的推送通知。

尽管 DSM 7.x 中引入的 Webhook 功能似乎因为功能不完整，存在使用问题。但是我们可以使用 Synology SMS 通知服务来完成 WebHook 请求调用。想要要在 DSM 上配置 SMS 通知，可以参考下面的文档 [ryancurrah/synology-notifications](https://github.com/ryancurrah/synology-notifications)。使用这个方案，我们将可以在 WebHook 中接受任何来自群会发送的通知内容。

在设置的过程中，我们需要指定一个 `api_key`，你可以随便生成一个 32 个字符长度的内容，来启用身份验证机制，来保护 WebHook 的执行。

除此之外，我们还可以在群晖的“控制面板 - 通知 - 规则”中配置想要接收的通知类型。

```json
[
  {
    "id": "synology",
    "execute-command": "do-something.sh",
    "command-working-directory": "/opt/webhook-linux-amd64/synology",
    "response-message": "Request accepted",
    "pass-arguments-to-command":
    [
      {
        "source": "payload",
        "name": "message"
      }
    ],
    "trigger-rule":
    {
      "match":
      {
        "type": "value",
        "value": "PUT_YOUR_API_KEY_HERE",
        "parameter":
        {
          "source": "header",
          "name": "api_key"
        }
      }
    }
  }
]
```

[Hook-Definition]: ./Hook-Definition.md
