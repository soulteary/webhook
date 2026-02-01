# Webhook Config UI

在网页中快速生成可调用的 webhook 配置（YAML/JSON 片段）及调用 URL、curl 示例，便于复制到 `hooks.yaml` / `hooks.json` 或直接调用。

## 运行方式

```bash
# 使用默认端口 9080
go run ./cmd/config-ui

# 指定端口
go run ./cmd/config-ui -port 9080

# 或通过环境变量
PORT=9080 go run ./cmd/config-ui
```

编译为独立二进制后：

```bash
go build -o webhook-config-ui ./cmd/config-ui
./webhook-config-ui -port 9080
```

## 使用说明

1. 浏览器打开 `http://localhost:9080`（或你指定的端口）。
2. 填写表单：
   - **必填**：Hook ID、执行命令
   - **可选**：工作目录、响应消息、**Webhook 服务地址**（如 `http://localhost:9000`，用于生成正确的调用 URL）、HTTP 方法、成功状态码、是否返回命令输出等
   - **可选（高级）**：响应头、传递参数/环境变量、触发规则、请求 Content-Type（均为 JSON 格式；若填写则需为合法 JSON，否则生成接口会返回 400 及错误说明）
3. 可点击「加载示例」快速填充一份示例（id、执行命令、Webhook 地址等），再点「生成」试跑。
4. 点击「生成」后，页面会展示：
   - 调用 URL（如 `http://localhost:9000/hooks/my-hook`）
   - curl 示例
   - YAML 与 JSON 配置片段
5. 可复制或下载 YAML/JSON 片段，粘贴到 webhook 的 `hooks.yaml` / `hooks.json` 中使用。可选区块（响应头、传递参数、触发规则等）默认折叠，点击「可选」展开。YAML/JSON 结果块可折叠以节省空间。成功生成后会记住 Hook ID 与 Webhook 服务地址（localStorage），下次打开页面时自动回填（若为空）。

## 与 webhook 同机部署

config-ui 为独立服务，与 webhook 主程序分开运行。若需同机部署：

- webhook 主服务：例如 `:9000`（hooks 端点 `/hooks/:id`）
- config-ui：例如 `:9080`（仅提供配置生成页）

可编写 `docker-compose.yml` 或 systemd 单元，分别启动两个进程；生成结果中的「调用 URL」需与 webhook 实际监听地址一致（可在生成后手动替换 host/port）。

## 技术说明

- 静态资源与页面配置通过 `embed` 嵌入，单二进制即可运行。
- 页面配置来自 `config/page.yaml`（i18n 与表单结构）。
- 生成 API：`POST /api/generate`，请求体为表单对应 JSON，响应为 `{ "yaml", "json", "callUrl", "curlExample" }`；错误时返回 `{ "error": "..." }` 及 4xx 状态码。

## 故障排查

- **页面空白**：若仅见白屏，多为模板字段与结构体不一致（如模板用了 `{{.Id}}` 而结构体为 `ID`）。当前模板已使用 `{{.ID}}`，重新构建并运行即可。
- **表单无法提交 / 复制无效**：确认浏览器控制台无 JS 报错；检查 `/static/js/app.js` 是否正常加载（Network 面板）。
- **生成接口 400**：查看响应体中的 `error` 字段，多为「id / execute-command 未填」或「可选 JSON 格式错误」。
