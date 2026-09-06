# 发布指南

本文是维护者发布 WebHook 稳定版本的操作手册。发布只能由一次不可变的语义化
版本 Tag 推送触发。不要手工创建 GitHub Release，也不要从 Actions 页面手工
运行 Release 工作流。

## 7.3.0 发布范围

7.3.0 包含 7.2.0 之后的以下变更：

- 移除重复的 `webhook-config-ui` Release 二进制；统一使用唯一的
  `webhook` 二进制和 `webhook -config-ui`；
- 通过 `getSecret` 和 `trimSpace` 增加受约束的 Docker Secret 模板支持，
  不开放任意文件系统读取；
- 将示例、容器标签、完整性校验命令和迁移下载版本更新到 7.3.0；
- 将 Release CI 限制为只接受 Tag，并把预检、发布、证明、验证和文档发布拆为
  可安全重试的 Job。

## 发布不变量

- Tag 必须是没有 `v` 前缀的 `MAJOR.MINOR.PATCH`。
- 被打 Tag 的提交已经进入 `main`，且该提交的所有必需检查均通过。
- 每个版本只打一次 Tag；绝不移动、删除或复用已经发布的 Tag。
- GitHub Release 和两个镜像仓库的镜像均由 Release 工作流创建。
- `preflight` 在使用发布凭据或产生发布副作用前执行全部构建检查。`publish` 只能
  执行一次；`attest`、`verify` 或 `documentation` 可以独立重试，不会再次运行
  GoReleaser。

## 发布前提

准备 Tag 前必须：

1. 合并本次发布的全部 PR，包括版本和文档准备 PR。
2. 等待最终 `main` 提交上的 Tests、容器 Smoke Test、Security Scan、CodeQL、
   Documentation 和 Benchmarks 全部通过。
3. 确认 Actions 可以使用 Docker Hub 凭据及 GitHub 仓库、Package 权限。
4. 本地只需要安装 Git。Go、Python、MkDocs、GoReleaser、Syft、Cosign 和
   GitHub CLI 均由 GitHub Actions 提供，不再是本地发布依赖。

7.3.0 选定的 `main` 提交必须同时包含 PR #177 和 PR #178。

## 无误发布的准确顺序

在已经同步、工作区干净的 `main` 中只执行一条命令：

```bash
./scripts/release.sh 7.3.0
```

该命令先执行快速、只依赖 Git 的本地预检，显示准确的提交 SHA，然后要求输入
`7.3.0` 确认，最后创建并推送一枚 Annotated Tag。非 `main`、工作区不干净、
本地与 `origin/main` 不一致、版本格式错误、文档版本未更新或同名 Tag 已存在时
都会停止。

推送前不要创建 Draft 或空的 GitHub Release，因为该操作由 GoReleaser 负责。
脚本只会推送 `refs/tags/7.3.0`，不会批量推送其他本地 Tag。

Tag 推送后会按以下顺序执行：

| 顺序 | Workflow/Job | 副作用 | 重试规则 |
|---|---|---|---|
| 1 | Release / `preflight` | 验证 Tag 和 `main` 祖先关系，执行 Race Test、严格文档构建和 GoReleaser 配置检查 | 没有发布副作用，可安全重试 |
| 2 | Release / `publish` | 构建 GitHub Release、压缩包、SBOM 和已签名多架构镜像 | 只执行一次 |
| 3 | Release / `attest` | 下载已发布资产并生成 GitHub Provenance | 只重试该 Job |
| 4 | Release / `verify` | 验证资产、Sigstore Bundle、Provenance、镜像 UID 和镜像签名 | 只重试该 Job |
| 5 | Release / `documentation` | 构建 `7.3.0`、更新 `latest` 并设置文档默认版本 | 只重试该 Job |

必须等待 Release 工作流完成；看到 GitHub Release 已出现不代表发布已经完成。

```bash
RELEASE_VERSION=7.3.0
RELEASE_RUN_ID="$(gh run list --workflow build.yml --branch "$RELEASE_VERSION" --limit 1 --json databaseId --jq '.[0].databaseId')"
gh run watch "$RELEASE_RUN_ID" --exit-status
```

## 发布后验证

Release 工作流通过后执行：

```bash
gh release view "$RELEASE_VERSION" --repo soulteary/webhook \
  --json tagName,isDraft,isPrerelease,url

mkdir -p "release-$RELEASE_VERSION"
gh release download "$RELEASE_VERSION" --repo soulteary/webhook \
  --dir "release-$RELEASE_VERSION"
find "release-$RELEASE_VERSION" -maxdepth 1 -type f -printf '%f\n' | sort

test -z "$(find "release-$RELEASE_VERSION" -name '*webhook-config-ui*' -print -quit)"

for variant in core runner extended; do
  image="ghcr.io/soulteary/webhook:${variant}-${RELEASE_VERSION}"
  docker pull "$image"
  test "$(docker image inspect --format '{{ .Config.User }}' "$image")" = "65532:65532"
  docker run --rm "$image" -version | grep "$RELEASE_VERSION"
done
```

还需要确认 `https://soulteary.github.io/webhook/7.3.0/` 和 `latest` 文档地址均可
正常访问。Release 工作流已经自动验证 Checksum Sigstore Bundle、GitHub
Attestation、镜像签名及三类运行时镜像。

## 失败处理

不要立即点击 **Re-run all jobs**。

| 失败位置 | 安全操作 |
|---|---|
| 本地预检或确认 | 修复或更新工作区后重新运行命令；此时还没有 Release |
| CI `preflight` | 修复 `main`、准备新版本并推送新 Tag；此时尚未开始发布 |
| `publish`，且 **Run GoReleaser** 尚未开始 | 重试失败的 `publish` Job |
| `publish`，且 **Run GoReleaser** 已经开始 | 不要重试；先检查 GitHub Release 和两个镜像仓库是否存在部分不可变产物 |
| `attest` | 重试失败 Job；已经完成的 `publish` 不会重跑 |
| `verify` | 重试失败 Job，或只重试 `verify` |
| `documentation` | 只重试失败的 `documentation` Job |

如果 GoReleaser 已经开始，并且任意带版本号的 Release 资产或镜像已经存在，
不要删除 Tag 后复用版本。保留现场，在 `main` 修复原因后发布下一个 Patch 版本
（例如 7.3.1），避免同一个不可变版本对应不同的二进制内容。

如果本地 Tag 已创建但推送失败，先用 `git show 7.3.0` 检查，再只重试
`git push origin refs/tags/7.3.0`。不要重新运行 `release.sh`，它会被重复 Tag
保护主动终止。
