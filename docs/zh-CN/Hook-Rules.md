# 钩子匹配规则

钩子的匹配规则包含一些逻辑性处理和签名校验方法。

## 支持设置的规则列表

* [与逻辑](#and)
* [或逻辑](#or)
* [非逻辑](#not)
* [组合使用](#multi-level)
* [匹配逻辑](#match)
  * [数值匹配](#match-value)
  * [正则匹配](#match-regex)
  * [请求内容 hmac-sha1 签名校验](#match-payload-hmac-sha1)
  * [请求内容 hmac-sha256 签名校验](#match-payload-hmac-sha256)
  * [请求内容 hmac-sha512 签名校验](#match-payload-hmac-sha512)
  * [IP 白名单](#match-whitelisted-ip-range)
  * [scalr 签名校验](#match-scalr-signature)

## And

当且仅当，所有子规则结果都为 `true`，才会执行钩子。

```json
{
"and":
  [
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
    },
    {
      "match":
      {
        "type": "regex",
        "regex": ".*",
        "parameter":
        {
          "source": "payload",
          "name": "repository.owner.name"
        }
      }
    }
  ]
}
```

## OR

当任何子规则结果为 `true` 时，才会执行钩子。

```json
{
"or":
  [
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
    },
    {
      "match":
      {
        "type": "value",
        "value": "refs/heads/development",
        "parameter":
        {
          "source": "payload",
          "name": "ref"
        }
      }
    }
  ]
}
```

## Not

当且仅当子规则结果都为 `false` 时，才会执行钩子。

```json
{
"not":
  {
    "match":
    {
      "type": "value",
      "value": "refs/heads/development",
      "parameter":
      {
        "source": "payload",
        "name": "ref"
      }
    }
  }
}
```

## Multi-level

```json
{
    "and": [
    {
        "match": {
            "parameter": {
                "source": "header",
                "name": "X-Hub-Signature"
            },
            "type": "payload-hmac-sha1",
            "secret": "mysecret"
        }
    },
    {
        "or": [
        {
            "match":
            {
                "parameter":
                {
                    "source": "payload",
                    "name": "ref"
                },
                "type": "value",
                "value": "refs/heads/master"
            }
        },
        {
            "match":
            {
                "parameter":
                {
                    "source": "header",
                    "name": "X-GitHub-Event"
                },
                "type": "value",
                "value": "ping"
            }
        }
        ]
    }
    ]
}
```

## Match

当且仅当 `parameter` 字段中的数值满足 `type` 指定规则时，才会执行钩子。

*注意* 匹配规则中的 `数值类型` 和 `布尔类型` 的值需要使用引号引起来，作为字符串传递。

### Match value

```json
{
  "match":
  {
    "type": "value",
    "value": "refs/heads/development",
    "parameter":
    {
      "source": "payload",
      "name": "ref"
    }
  }
}
```

### Match regex

正则表达式的语法，可以参考 [Golang Regexp Syntax](http://golang.org/pkg/regexp/syntax/)

```json
{
  "match":
  {
    "type": "regex",
    "regex": ".*",
    "parameter":
    {
      "source": "payload",
      "name": "ref"
    }
  }
}
```

### Match payload-hmac-sha1

使用 SHA1 哈希和指定的的 *secret* 字段验证提交数据的 HMAC 签名有效：

```json
{
  "match":
  {
    "type": "payload-hmac-sha1",
    "secret": "yoursecret",
    "parameter":
    {
      "source": "header",
      "name": "X-Hub-Signature"
    }
  }
}
```

注意，你可以使用逗号分隔字符串，来传递多个签名。程序将尝试匹配所有的签名，找到任意一项匹配内容。

```yaml
X-Hub-Signature: sha1=the-first-signature,sha1=the-second-signature
```

### Match payload-hmac-sha256

使用 SHA256 哈希和指定的的 *secret* 字段验证提交数据的 HMAC 签名有效：

```json
{
  "match":
  {
    "type": "payload-hmac-sha256",
    "secret": "yoursecret",
    "parameter":
    {
      "source": "header",
      "name": "X-Signature"
    }
  }
}
```

注意，你可以使用逗号分隔字符串，来传递多个签名。程序将尝试匹配所有的签名，找到任意一项匹配内容。

```yaml
X-Hub-Signature: sha256=the-first-signature,sha256=the-second-signature
```

### Match payload-hmac-sha512

使用 SHA512 哈希和指定的的 *secret* 字段验证提交数据的 HMAC 签名有效：

```json
{
  "match":
  {
    "type": "payload-hmac-sha512",
    "secret": "yoursecret",
    "parameter":
    {
      "source": "header",
      "name": "X-Signature"
    }
  }
}
```

注意，你可以使用逗号分隔字符串，来传递多个签名。程序将尝试匹配所有的签名，找到任意一项匹配内容。

```yaml
X-Hub-Signature: sha512=the-first-signature,sha512=the-second-signature
```

### Match Whitelisted IP range

支持使用 IPv4 或 IPv6 格式的地址搭配[CIDR表示法](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing#CIDR_blocks)，来表达有效的 IP 范围。

如果想要匹配单个 IP 地址，请在 IP 后添加 `/32`。

```json
{
  "match":
  {
    "type": "ip-whitelist",
    "ip-range": "192.168.0.1/24"
  }
}
```

### Match scalr-signature

验证是否是有效的 scalr 签名，以及请求是在五分钟内收到的未过期请求。你可以在 Scalr 中为每一个 WebHook URL 生成唯一的签名密钥。

因为校验方法和时间相关，请确保你的 Scalr 和 WebHook 服务器都设置了 NTP 服务，时间一致。

```json
{
  "match":
  {
    "type": "scalr-signature",
    "secret": "Scalr-provided signing key"
  }
}
```
