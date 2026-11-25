# Changelog

## [Unreleased]

- 新增 `dictionary` 子包，以只读 SQLite 接入 ECDICT Ultimate，和 CC-CEDICT 提供统一的中英离线查词，支持显式语种、自动分流和混合输入错误。
- 新增 `cedict` 子包，支持从调用方提供的 CC-CEDICT v1 文本离线查询简繁中文词语的英文释义。
- 移除 `Client.TOS()`、`Client.Document()`、`WithTOS`、`WithPollPolicy`、`WithProgressHook` 及相关类型和错误。这是破坏性变更。已有结果 URL 可继续用 `Client.Download` 下载，文档文本段可通过 `Text().Translate` 翻译。
- 新增火山引擎文本、图片翻译 SDK 封装、译文下载和统一错误模型。
- 模块路径按需求约定使用 `github.com/gtkit/volcenginex`。
- 新增机器翻译 21 个网关错误码常量及文档模板查询。文本翻译、语种检测、图片翻译和文本用量错误可通过 `CodeN()` 读取上游数值错误码。
- 修复 `WithMaxRetries(0)` 未关闭 SDK 重试的问题。
- 文本翻译拒绝缺项、多项和空白译文，失败批次不计入字符数。
- 文本翻译按每批最多 16 段、5000 个 Unicode 字符发送，保留分段文本的原始顺序和索引。
- 修复文本用量查询的日期、服务参数和响应解码，保留 `GetUsageCommon()` 的 Unix 秒入参。
- `Image().TranslateURL()` 先下载公网图片，再将字节提交给火山图片翻译。
- 新增 `Audio().SubmitAudio()` 和 `Audio().QueryAudio()`，支持提交音视频文件翻译任务，并查询任务状态和带时间的字幕。
- 新增 `Client.GetUsage()`，支持按服务查询文本、图片和音视频翻译用量。现有文本用量接口保持兼容，图片和音视频用量保留上游原值，单位待实测。
