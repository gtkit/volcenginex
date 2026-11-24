# volcenginex

`volcenginex` 为译易插件后端提供火山引擎翻译封装。译易的 PDF 和 Office 流程使用文本翻译接口处理本地抽取的文本段。

## 能力矩阵

| 能力 | 官方服务 | 语义 | 计费边界 |
|---|---|---|---|
| 文本翻译、语言检测、用量 | `translate` | 同步 | 机器翻译字符后付费 |
| 图片翻译 | `translate` | 同步 | 机器翻译图片服务 |
| 文档项目、任务、轮询 | `i18n_openapi` | 异步 | 国际化翻译平台套餐/额度 |
| TOS 上传与译文下载 | TOS/HTTP | 传输 | 由对应存储与翻译产品计费 |

## 快速开始

```go
client, err := volcx.New(volcx.Config{AK: "your-ak", SK: "your-sk"})
if err != nil { return err }
defer client.Close()
result, err := client.Text().Translate(ctx, volcx.TextTranslateRequest{
    TargetLanguage: "en",
    TextList: []string{"你好，世界"},
})

// result.CharacterCount is a local rune-based estimate for display or
// pre-deduction. Final billing comes from the Volcengine bill.
```

凭据也可以从 `VOLCENGINE_ACCESS_KEY`、`VOLCENGINE_SECRET_KEY` 和 `VOLCENGINE_SESSION_TOKEN` 读取。完整客户端只创建一个官方 session，文本、图片和文档服务共享连接池。

文本翻译使用 `2025-03-01`。将 `SourceLanguage` 留空即可自动识别源语种。返回的 `Items` 与输入顺序一致，每项包含译文及上游返回的检测语种。

`WithMaxRetries(0)` 禁用官方 SDK 重试，未设置时默认重试三次。文本翻译会校验上游译文条数和内容，只有有效批次计入 `CharacterCount`。如果上游返回数值错误码，可通过 `(*Error).CodeN()` 读取。

## PDF 和 Office 分段翻译

译易在本地解析 PDF、DOCX、XLSX 和 PPTX，按原文档结构提取文本段。把这些文本段按顺序传给 `Text().Translate()` 的 `TextList`，再用 `Items[i].Index` 找回源文本段并回填、排版。长文本块由译易按原有标记边界切段。

Go 库每次向火山发送最多 16 段、合计 5000 个 Unicode 字符。`BatchSize` 可以设置更小的段数上限。空白段保留原索引且跳过发包。单段超过 5000 字符时返回 `ErrInvalidArgument`，整次调用不发包。这个流程只需要机器翻译凭据。

## 已有云文档接口

```text
本地文件 --TOS 预签名 PUT--> 对象 URL
    --DocumentCreate--> 项目 ID
    --DocumentTaskCreate--> 任务 ID
    --轮询 DocumentTaskDetail--> TargetDocURL
    --Download--> 本地译文
```

上传失败返回 `TOSUpload` 错误，创建项目或任务失败返回带请求 ID 的 provider 错误，轮询超时或连续失败返回轮询错误，下载长度不一致返回 `ErrIncompleteDownload`。

## 已知限制

- 云文档接口的 `.doc`、`.xls`、`.ppt`、`.csv` 以及飞书表格/幻灯片类型不支持线上解析，需另存为 `.docx`、`.xlsx` 或 `.pptx`。
- 云文档接口的单任务最多 10 个文档，单文件最多 200 MB，单任务总大小最多 2000 MB。
- 文本接口不提供记忆库和术语库，术语一致性需要业务层处理。
- 文档翻译为异步任务，典型耗时为分钟级，不适合划词实时场景。

## 插件后端代理

浏览器 content script 和 popup 不应接触 AK/SK。服务端代理从环境变量读取凭据，再在 handler 中调用本包。

```go
func translateHandler(client *volcx.Client) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        result, err := client.Text().Translate(r.Context(), volcx.TextTranslateRequest{
            TargetLanguage: "en",
            TextList: []string{"hello"},
        })
        if err != nil { http.Error(w, err.Error(), http.StatusBadGateway); return }
        fmt.Fprint(w, result.Items[0].Translation)
    })
}
```

计费与权限对照见 [`docs/billing.md`](docs/billing.md)，官方未生成接口的兜底说明见 [`docs/escape-hatch.md`](docs/escape-hatch.md)。

## 用量统计

`Text().Translate` 返回的 `CharacterCount` 只用于本地展示或预扣。图片结果的 `ImageCount` 统计图片张数，不统计 OCR 字符。文档全链路结果的 `EstimatedCharacterCount` 只统计调用方提供的内联文本，上传文件由火山内部抽取文本，实际账单可能存在偏差。

后台定时任务可按 Unix 秒时间段调用独立的 `GetUsageCommon` 做对账，该方法不参与单次翻译实时展示。

```go
usage, err := client.Text().GetUsageCommon(ctx, fromUnix, toUnix)
if err != nil { return err }
fmt.Println(len(usage.Points))
```
