# Changelog

## [Unreleased]

- 新增火山引擎文本、图片和文档翻译 SDK 封装。
- 新增 TOS 预签名上传、译文下载、轮询和统一错误模型。
- 模块路径按需求约定使用 `github.com/gtkit/volcenginex`。
- 文本翻译错误可通过 `CodeN()` 读取上游数值错误码。
- 修复 `WithMaxRetries(0)` 未关闭 SDK 重试的问题。
- 文本翻译拒绝缺项、多项和空白译文，失败批次不计入字符数。
- 文本翻译按每批最多 16 段、5000 个 Unicode 字符发送，保留分段文本的原始顺序和索引。
