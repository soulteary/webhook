# 钩子定义

我们可以在 JSON 或者 YAML 文件中定义钩子对象。每一个有效的钩子对象都必须包含 `id` 和 `execute-command` 属性，其他属性都是可选项。

## 钩子属性

* `id` - 钩子的 ID。用于创建 HTTP 地址，如：`http://yourserver:port/hooks/your-hook-id`。
* `execute-command` - 钩子地址在被访问时，对应的执行命令。  
* `command-working-directory` - 指定执行脚本时使用的工作目录。
* `response-message` - 将返回给钩子调用方的字符串。
* `response-headers`- 将在 HTTP 响应中返回的响应头数据，格式为 `{"name":"X-Example-Header","value":"it works"}`。
* `success-http-response-code` - 调用成功后，返回的 HTTP 状态码。
* `incoming-payload-content-type` - 设置传入HTTP请求的 `Content-Type`，例如：`application/json`。
* `http-methods` - 允许的 HTTP 请求方法，可以设置为 `POST` 或 `GET` 等。
* `include-command-output-in-response` - 布尔值（`true`/`false`），是否应该等待脚本程序执行完毕，并将原始程序输出返回给调用方。如果程序执行失败，将会返回 `HTTP 500 程序内部错误` 的状态信息，通常会返回 `HTTP 200 OK`。
* `include-command-out-in-response-on-error` - 布尔值（`true`/`false`），当命令执行失败时，是否将命令中的 `stdout` 和 `stderror` 返回给调用方。
* `pass-arguments-to-command` - 将指定参数设置在 JSON 字符串中，并传递给要调用程序的参数中，你可以访问[请求值设置][Request-Values]文档，来了解详细的内容。例如，我们可以传递一个字符串内容，格式为：`{"source":"string","name":"value"}`
* `parse-parameters-as-json` - 将指定参数设置在 JSON 字符串中，使用规则和`pass-arguments-to-command` 一致。
* `pass-environment-to-command` - 将指定的参数设置为环境变量，并传递给调用程序的参数中。如果没有指定 `"envname"`字段，那么程序将采用 "HOOK_argumentname" （`argumentname` 具体请求参数名）变量名称，否则将使用 `"envname"` 字段作为名称。在[请求值设置][Request-Values]文档中可以了解更多细节。例如，如果要将静态字符串值传递给命令,可以将其指定为 `{"source":"string","envname":"SOMETHING","name":"value"}`。
* `pass-file-to-command` - 指定要传递给命令的文件列表。传递给命令的内容将在序列化处理并存储在临时文件中（并行调用程序，将发生文件的覆盖）。如果你想在脚本中使用环境变量的方法访问文件名称，可以参考 `pass-environment-to-command`。如果你定义了 `command-working-directory`，将会作为文件的保存目录。如果额外设置了 `base64decode` 为 true，那么程序将会对接收到的二进制数据先进行 Base64 解码，再进行文件保存。默认情况下，这些文件将会在 WebHook 程序退出后被删除。更多信息可以查阅[请求值设置][Request-Values]文档。
* `trigger-rule` - 配置钩子的具体触发规则，访问[钩子规则](Hook-Rules.md)文档，来查看详细内容。
* `trigger-rule-mismatch-http-response-code` - 设置在不满足触发规则时返回给调用方的 HTTP 状态码。
* `trigger-signature-soft-failures` - 设置是否允许忽略钩子触发过程中的签名验证处理结果，默认情况下，如果签名校验失败，那么会被视为程序执行出错。

## 示例

更复杂的例子，可以查看[示例](Hook-Examples.md)文档。


[Request-Values]: ./Referencing-Request-Values.md
