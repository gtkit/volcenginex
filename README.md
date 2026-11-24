# volcenginex

`volcenginex` 是独立的火山引擎翻译 Go 库，提供文本翻译、语种检测、图片翻译、音视频文件翻译、按服务查询翻译用量和可选的云文档任务接口。

## 能力矩阵

| 能力 | 官方服务 | 语义 | 计费边界 |
|---|---|---|---|
| 文本翻译、语言检测 | `translate` | 同步 | 机器翻译字符后付费 |
| 图片翻译 | `translate` | 同步 | 机器翻译图片服务 |
| 音视频文件翻译 | `translate` | 异步 | 机器翻译音视频服务 |
| 翻译用量查询 | `translate` | 同步 | 查询文本、图片或音视频的上游用量 |
| 文档项目、任务、轮询 | `i18n_openapi` | 异步 | 国际化翻译平台套餐/额度 |
| TOS 上传与译文下载 | TOS/HTTP | 传输 | 由对应存储与翻译产品计费 |
| CC-CEDICT 汉英查词 | 本地词库 | 同步 | 无云端请求 |

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

凭据也可以从 `VOLCENGINE_ACCESS_KEY`、`VOLCENGINE_SECRET_KEY` 和 `VOLCENGINE_SESSION_TOKEN` 读取。完整客户端只创建一个官方 session，文本、图片、音视频和文档服务共享连接池。

文本翻译使用 `2025-03-01`。将 `SourceLanguage` 留空即可自动识别源语种。返回的 `Items` 与输入顺序一致，每项包含译文及上游返回的检测语种。

单独检测语种时，调用 `client.Text().Detect(ctx, texts)`。它会跳过空白文本，按每批最多 16 条发送请求，并按输入顺序返回结果。每项的 `Index` 对应 `texts` 中的原始下标。任一批失败时，方法返回错误，结果为 `nil`。

`WithMaxRetries(0)` 禁用官方 SDK 重试，未设置时默认重试三次。文本翻译会校验上游译文条数和内容，只有有效批次计入 `CharacterCount`。如果上游返回数值错误码，可通过 `(*Error).CodeN()` 读取。

`Image().TranslateURL()` 从公网 HTTP/HTTPS 地址下载图片，再发送编码后的字节。下载使用独立的直连客户端，遵循 `WithTimeout` 和 `WithMaxImageBytes`。需要访问私网或自定义代理、鉴权的调用方，可用自己的客户端取得图片字节，再调用 `Image().Translate()`。

## CC-CEDICT 离线汉英查词

从 [MDBG 下载 CC-CEDICT v1 UTF-8 词库](https://www.mdbg.net/chinese/dictionary?lang=en&page=cc-cedict)，选择 `cedict_1_0_ts_utf-8_mdbg.zip`。解压后把文本文件路径传给 `cedict.LoadFile`。`cedict_ts.u8` 和 `cedict_1_0_ts_utf-8_mdbg.txt` 都是常见文件名。词库数据由 MDBG 发布，采用 [CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/)。

```go
import "github.com/gtkit/volcenginex/cedict"

// 在应用启动时加载，后续请求共享这个字典。
dict, err := cedict.LoadFile(dictionaryPath)
if err != nil {
    return err
}
for _, entry := range dict.Lookup("苹果") {
    fmt.Println(entry.Pinyin, entry.Definitions)
}
```

`Lookup` 精确匹配简体或繁体词头，按词库中的顺序返回全部匹配词条及各自的英文释义。首尾空白会被忽略，未命中时返回 `nil`。字典可并发复用，无需火山引擎凭据。查词适用于词库已收录的单词和短语。

## 音视频文件翻译

将火山引擎可访问的音频或视频文件 URL 传给 `Audio().SubmitAudio()`。源语种和目标语种均必填，预签名 URL 的查询参数会原样传给火山引擎。拿到任务 ID 后，通过 `Audio().QueryAudio()` 查询任务状态及字幕。

```go
task, err := client.Audio().SubmitAudio(ctx, volcx.SubmitAudioRequest{
    URI:            "https://media.example.com/sample.mp4",
    SourceLanguage: "en",
    TargetLanguage: "zh",
})
if err != nil { return err }

result, err := client.Audio().QueryAudio(ctx, task.TaskID)
if err != nil { return err }
fmt.Println(result.Status)
for _, subtitle := range result.Subtitles {
    fmt.Println(subtitle.StartTime, subtitle.EndTime, subtitle.Translation)
}
```

查询返回火山引擎的原始任务状态。`created` 表示排队，`success` 表示成功，`download_failed` 等状态表示任务失败。字幕时间值原样返回，上游省略置信度时 `Confidence` 为 `nil`。调用方决定何时再次查询。

提交关闭 SDK 自动重试，以免响应丢失后重复创建任务。查询沿用 `WithMaxRetries` 配置。

## 机器翻译错误

文本翻译、语种检测、图片翻译、音视频任务提交和查询、用量查询接口返回的上游错误可用 `errors.AsType[*volcx.Error](err)` 读取。使用 `Code` 判断错误类型，用 `Message` 获取上游实际文案，用 `RequestID` 排查请求。上游提供数字码时，`CodeN()` 返回数字码和 `true`。

```go
if apiErr, ok := errors.AsType[*volcx.Error](err); ok {
    info, known := volcx.LookupGatewayError(apiErr.Code)
    if known && info.Code == volcx.CodeFlowLimitExceeded {
        // 处理火山引擎限流。
    }
    // apiErr.Message 是本次响应的文案，info.MessageTemplate 是文档中的模板。
}
```

`LookupGatewayError` 支持字符串码和十进制数字码，覆盖[火山引擎列出的 21 条网关错误](https://docs.volcengine.com/docs/machine-translation/interface-error-message?lang=zh)。未知错误码也会随 `*volcx.Error` 返回。启用批次部分成功时，从 `result.Errors` 读取失败批次的错误。

## PDF 和 Office 分段翻译

调用方解析 PDF、DOCX、XLSX 或 PPTX，按原文档结构抽取文本段。将文本段按顺序传给 `Text().Translate()` 的 `TextList`，然后按返回项的 `Index` 回填译文。调用方可按原文的标记边界切分长文本块，并根据文档布局排版。

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

`Client.GetUsage(ctx, service, from, to)` 使用 `UsageServiceText`、`UsageServiceImage` 或 `UsageServiceAudio` 选择查询服务。`from` 和 `to` 是 `time.Time`，按北京时间转换为日期，查询包含首尾两天。

```go
for _, service := range []volcx.UsageService{volcx.UsageServiceImage, volcx.UsageServiceAudio} {
    usage, err := client.GetUsage(ctx, service, from, to)
    if err != nil { return err }
    fmt.Println(service, usage.Points)
}
```

`Usage.Points` 保留上游的 `Timestamp`、`Value` 和顺序。图片及音视频的 `Value` 单位和取整规则待线上实测。

文本用量查询也使用 `Client.GetUsage`，将服务指定为 `UsageServiceText`。完整调用见 [可运行示例](example/usage/main.go)。

兼容入口 `Text().Usage()` 查询北京时间含当天的最近 30 个自然日。`Text().GetUsageCommon(ctx, from, to)` 接收闭区间 Unix 秒，按北京时间查询覆盖日期的文本翻译用量。
