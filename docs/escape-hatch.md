# 官方 SDK 逃生通道

本包当前覆盖的文本和图片接口使用官方 SDK 已生成的强类型方法，文档接口使用官方 SDK 的请求处理管线和包内私有线协议结构。

官方 SDK 尚未生成某个新 Action 时，调用方可以直接使用 `github.com/volcengine/volcengine-go-sdk/volcengine/universal` 的 `DoCall` 或 `DoCallWithType`。该路径由调用方自行承担 Action、版本、字段和返回结构维护，本包不会用它替代已有强类型方法。

SDK 内部 JSON codec 由官方实现固定。`gtkit/json/v2` 只用于本包自有 DTO 和 `httpc` 编解码器，二者不能通过 monkey patch 互换。
