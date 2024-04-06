# 请求内容设置

程序支持设置四种类型的请求内容：

- HTTP 请求头
- HTTP 查询参数
- HTTP 请求参数
- HTTP 请求体内容

## HTTP 请求头

```json
{
  "source": "header",
  "name": "Header-Name"
}
```

## HTTP 查询参数

```json
{
  "source": "url",
  "name": "parameter-name"
}
```

## HTTP 请求参数

```json
{
  "source": "request",
  "name": "method"
}
```

```json
{
  "source":"request",
  "name":"remote-addr"
}
```

## HTTP 请求体内容（JSON / XML / 表单内容）

```json
{
  "source": "payload",
  "name": "parameter-name"
}
```

### JSON

我们可以使用类似下面的方式来设置 JSON 请求数据。

```json
{
  "commits": [
    {
      "commit": {
        "id": 1
      }
    },
    {  
      "commit": {
        "id": 2
      }
    }
  ]
}
```

如果我们想在程序中获得 “第一个 commit 提交” 数据的 ID，可以这样：

```json
{
  "source": "payload", 
  "name": "commits.0.commit.id"
}
```

如果 JSON 中包含我们访问的 Key，例如 `{ "commits.0.commit.id": "value", ... }`。那么程序将会优先访问这个数据，而非展开具体的对象，解析其中的数据。

### XML有效负载

使用 XML 作为数据类似上面的 JSON 数据使用，但是相对更复杂一些。以下面的 XML 数据为例：

```xml
<app>
  <users>
    <user id="1" name="Li Lei" />
    <user id="2" name="Han Meimei" />
  </users>
  <messages>
    <message id="1" from_user="1" to_user="2">Nice To Meet U!!</message>
  </messages>
</app>
```

如果我们想要访问 `user` 元素，我们需要将其转换为数组，而不能使用 JSON 中的访问方式。

在 XML 中，`app.users.user.0.name` 将得到 `Li Lei`；因为只有一个 `message` 元素，所以解析的时候不会作为数组处理。`app.messages.message.id` 的结果是 `1`。如果想要访问 `message` 的标签文本，那么我们需要使用 `app.messages.message.#text`。

## 环境变量使用

当我们想使用 `envname` 属性，来设置命令使用的环境变量名称时：

```json
{
   "source":"url",
   "name":"q",
   "envname":"QUERY"
}
```

上面的例子中，我们设置了一个环境变量 `QUERY`，对应的是请求 WebHook 的查询字符串中的查询参数 `q`。

## 特殊情况

如果你想不对 JSON 进行解析，将完整的 JSON 传递给命令，可以使用下面的方法：

```json
{
  "source": "entire-payload"
}
```

类似的，HTTP 请求头可以用下面的方法：

```json 
{
  "source": "entire-headers"
}
```

查询变量，可以使用下面的方：

```json
{
  "source": "entire-query"  
}
```
