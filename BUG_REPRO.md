# Bug Reproduction

## 包的性质

当前 test_model_fix 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。要复现原始缺陷，必须检出下面固定的 parent SHA；不要在当前修复结果源码上期待重新出现修复前失败。生成系统使用的可信验证补丁和完整验证日志仅在本地留存，不提交到结果分支。

## 问题现象

航班状态准备批量推送时，路由服务偶尔会停顿几百毫秒；就在这段时间关闭订阅的客户端，会让推送协程恢复后直接 panic，仍在线的运营看板也收不到后续状态。单条推送和没有中途退订的批次都正常。请修复批量分发与退订之间的并发协调，断开的订阅应被安全跳过，其他订阅仍按顺序收到整个批次，并确保竞态检测通过。

## 含 Bug 版本

- 仓库：VanceMichael/go-label-25
- 仓库地址：https://github.com/VanceMichael/go-label-25.git
- parent SHA：2d875f4305343bec587ea4c95fb509950ba2ea0d

## 复现步骤

```bash
git clone -- https://github.com/VanceMichael/go-label-25.git bug-repro
cd bug-repro
git checkout --detach 2d875f4305343bec587ea4c95fb509950ba2ea0d
go test ./internal/stream -run ^TestPublishBatchSurvivesUnsubscribeDuringPreparation$ -count=1
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/stream -run ^TestPublishBatchSurvivesUnsubscribeDuringPreparation$ -count=1
--- FAIL: TestPublishBatchSurvivesUnsubscribeDuringPreparation (0.01s)
    batch_test.go:46: publish batch panic: send on closed channel
FAIL
FAIL	github.com/VanceMichael/go-base-airbridge/internal/stream	0.036s
FAIL

```

stderr：

```text
(empty)
```

### linux/arm64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/stream -run ^TestPublishBatchSurvivesUnsubscribeDuringPreparation$ -count=1
--- FAIL: TestPublishBatchSurvivesUnsubscribeDuringPreparation (0.00s)
    batch_test.go:46: publish batch panic: send on closed channel
FAIL
FAIL	github.com/VanceMichael/go-base-airbridge/internal/stream	0.003s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

路由准备期间退订并关闭一个订阅通道后，PublishBatch 必须正常结束而不能出现 send on closed channel，已退订对象不再接收消息，仍在线订阅者按 loaded、departed 的顺序收到完整批次；TestPublishBatchSurvivesUnsubscribeDuringPreparation 在 -race 下由红转绿，stream 原有行为、仓库其余测试与 go build ./... 保持通过，不得用 recover 吞掉 panic、停止健康订阅投递或修改测试时序来绕过关闭通道竞争。
