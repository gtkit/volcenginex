# 火山引擎翻译 SDK 实施计划

需求来源是 `/Users/xiaozhaofu/Downloads/volcengine-translate-sdk-prompt.md`。本计划只覆盖当前 OpenSpec change `volcengine-translate-sdk`，模块路径按需求文档采用 `github.com/gtkit/volcenginex`。

目标是实现可编译、可测试的 Go 1.27 SDK，复用官方火山 SDK 的强类型客户端，提供文本、图片、文档异步翻译、TOS 上传和译文下载能力。非目标包括修改官方 SDK、实现 universal 主路径、实现服务端代理和计费系统。

## 已核实事实

- 当前仓库只有 `go.mod`、`version.go`、Makefile 和规则文件。
- 当前 `go.mod` 是 `github.com/gtkit/go-volcengine`，需求文档要求 `github.com/gtkit/volcenginex`。
- `volcengine-go-sdk v1.0.146` 实际不包含目标服务包，已核实并采用包含 `translate20250301` 和 `i18nopenapi` 的 `v1.2.53`。
- 本次其他依赖 API 已从本地模块缓存和源码核对。
- 需求文档指定官方服务、字段映射、请求 ID、限制值、轮询策略和 HTTP/JSON 传输分工。

## 实施步骤

1. 核实依赖版本和官方类型，更新 module 与依赖清单。验证 `go mod tidy`、`go build ./...`。
2. 实现配置、Option、共享 session、子服务、错误和日志适配。验证凭据、默认值、上下文和错误映射。
3. 实现文本与图片 DTO、转换、分批并发和输入校验。验证 httptest 请求字段、空白项、顺序和大小限制。
4. 实现文档 DTO、限制表、项目/任务操作和轮询。验证字段映射、GET query、截止时间、状态机、退避、取消和进度。
5. 实现 TOS 预签名桥接、上传、JSON codec 和下载。验证原始字节、流式大小、临时文件和原子替换。
6. 添加 examples、README、CHANGELOG、billing 与 escape-hatch 文档，并补齐测试和基准。
7. 执行 OpenSpec strict validate、功能测试、构建，并在获准后执行受控门禁。

## 风险、回滚与验收

- 依赖 API 差异通过源码核对和编译测试处理，不能凭提示词猜测。
- 若实现失败，回滚当前 change 文件和模块声明即可，不触碰外部服务或官方依赖源码。
- 验收包含请求方法与字段、错误 `errors.Is/As`、上下文取消、轮询终止、TOS 原始 PUT、下载长度校验、race 与覆盖率证据。

<!-- harness-contract:v1 -->
```json
{
  "schema_version": 1,
  "stage": "implementation",
  "change": "volcengine-translate-sdk",
  "allowed_files": [
    {"path": "docs/PLAN.md", "kind": "document"},
    {"path": "go.mod", "kind": "code"},
    {"path": "go.sum", "kind": "code"},
    {"path": ".harness/error-journal.md", "kind": "document"},
    {"path": "version.go", "kind": "code"},
    {"path": "doc.go", "kind": "code"},
    {"path": "option.go", "kind": "code"},
    {"path": "client.go", "kind": "code"},
    {"path": "errors.go", "kind": "code"},
    {"path": "log.go", "kind": "code"},
    {"path": "dto.go", "kind": "code"},
    {"path": "convert.go", "kind": "code"},
    {"path": "text.go", "kind": "code"},
    {"path": "image.go", "kind": "code"},
    {"path": "document.go", "kind": "code"},
    {"path": "document_poll.go", "kind": "code"},
    {"path": "doc_limits.go", "kind": "code"},
    {"path": "tos.go", "kind": "code"},
    {"path": "download.go", "kind": "code"},
    {"path": "internal/jsoncodec/codec.go", "kind": "code"},
    {"path": "internal/trace/trace.go", "kind": "code"},
    {"path": "text_test.go", "kind": "code"},
    {"path": "client_test.go", "kind": "code"},
    {"path": "convert_test.go", "kind": "code"},
    {"path": "errors_test.go", "kind": "code"},
    {"path": "misc_test.go", "kind": "code"},
    {"path": "image_test.go", "kind": "code"},
    {"path": "document_test.go", "kind": "code"},
    {"path": "download_test.go", "kind": "code"},
    {"path": "tos_test.go", "kind": "code"},
    {"path": "example_test.go", "kind": "code"},
    {"path": "benchmark_test.go", "kind": "code"},
    {"path": "example/text/main.go", "kind": "code"},
    {"path": "example/image/main.go", "kind": "code"},
    {"path": "example/document/main.go", "kind": "code"},
    {"path": "example/usage/main.go", "kind": "code"},
    {"path": "README.md", "kind": "document"},
    {"path": "CHANGELOG.md", "kind": "document"},
    {"path": "LICENSE", "kind": "document"},
    {"path": "docs/billing.md", "kind": "document"},
    {"path": "docs/escape-hatch.md", "kind": "document"}
  ],
  "required_context": [
    {"kind": "rules", "path": "AGENTS.md", "name": "project-rules"},
    {"kind": "guide", "path": ".harness/guides/workflow-contract.md", "name": "workflow-contract"},
    {"kind": "guide", "path": ".harness/guides/pkg-structure.md", "name": "package-structure"},
    {"kind": "guide", "path": ".harness/guides/pkg-errors.md", "name": "package-errors"},
    {"kind": "guide", "path": ".harness/guides/pkg-testing.md", "name": "package-testing"},
    {"kind": "skill", "path": "/Users/xiaozhaofu/.agents/skills/use-modern-go/SKILL.md", "name": "use-modern-go"},
    {"kind": "skill", "path": "/Users/xiaozhaofu/.agents/skills/go-error-handling/SKILL.md", "name": "go-error-handling"},
    {"kind": "skill", "path": "/Users/xiaozhaofu/.agents/skills/go-testing/SKILL.md", "name": "go-testing"}
  ],
  "checks": [
    {"id": "build", "kind": "functional", "argv": ["go", "build", "./..."], "timeout_seconds": 300},
    {"id": "unit-test", "kind": "functional", "argv": ["go", "test", "./..."], "timeout_seconds": 300}
  ],
  "regressions": []
}
```
