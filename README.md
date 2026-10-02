# component-base
Scheme, typing, encoding, decoding, and conversion packages for FangcunMont project.

## 消息职责退役

下一版本不再提供 `pkg/messaging`、`pkg/event`、`pkg/eventcatalog`、
`pkg/eventcodec`、`pkg/eventmessaging`、`pkg/outbox` 和 `pkg/outboxcore`。
NSQ 发布订阅、消息封装、重试及 Outbox 共性职责由
[reliable-messaging](https://github.com/FangcunMount/reliable-messaging) 承接；
首期 SDK 仅支持 NSQ，旧 RabbitMQ 入口在本次退役，不承诺替代支持。
业务事件、事务边界、业务幂等和失败处置仍由服务负责。

这是删除公开 API 的不兼容变更，计划以 `v0.7.0` 发布。升级前须清理
运行代码及测试的旧包导入，并通过固定旧版本的消息格式和恢复契约验证。
IAM 和 QS 的新链路使用可靠消息 SDK；其他调用者不能仅修改模块版本。
原数据库事务和宿主持久记录不会因更换模块自动消失。

旧实现仍可从固定 `v0.6.11` 获取。历史互通及回退验证应使用固定旧版本的
独立模块或二进制，不能将旧算法复制回服务，也不能将当前 SDK 对端当作旧版本。
本仓库不提供空实现或静默成功的兼容入口。
