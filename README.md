# volcenginex

`volcenginex` 为译易插件后端提供火山引擎翻译封装。文本/图片翻译和文档翻译使用两套独立产品服务，共享账号凭据但拥有独立权限和账单。

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
```

凭据也可以从 `VOLCENGINE_ACCESS_KEY`、`VOLCENGINE_SECRET_KEY` 和 `VOLCENGINE_SESSION_TOKEN` 读取。完整客户端只创建一个官方 session，文本、图片和文档服务共享连接池。

## 文档翻译全链路

```text
本地文件 --TOS 预签名 PUT--> 对象 URL
    --DocumentCreate--> 项目 ID
    --DocumentTaskCreate--> 任务 ID
    --轮询 DocumentTaskDetail--> TargetDocURL
    --Download--> 本地译文
```

上传失败返回 `TOSUpload` 错误，创建项目或任务失败返回带请求 ID 的 provider 错误，轮询超时或连续失败返回轮询错误，下载长度不一致返回 `ErrIncompleteDownload`。

## 已知限制

- `.doc`、`.xls`、`.ppt`、`.csv` 以及飞书表格/幻灯片类型不支持线上解析，需另存为 `.docx`、`.xlsx` 或 `.pptx`。
- 单任务最多 10 个文档，单文件最多 200 MB，单任务总大小最多 2000 MB。
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
