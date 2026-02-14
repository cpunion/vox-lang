# Async 模型（D03）

当前状态（Stage2，2026-02）：

1. 关键字已保留：`async`、`await`。
2. `async fn` 语法已接入 parser（AST 可识别 `FuncDecl.is_async`）。
3. 语义层暂未开启：typecheck 会报错 `async fn is deferred (D03)`。
4. `await` 表达式暂未开启：parser 报错 `await expression is deferred (D03)`。
5. trait 方法上的 `async fn` 暂未开启：parser 报错 `async trait method is deferred (D03)`。

这样做的目的：

- 先锁定词法/语法形状，避免后续引入 async 时产生关键字兼容破坏。
- 在尚未具备完整执行模型前，给出稳定且明确的诊断。

## 后续落地顺序

1. Future 表示：确定语言级 `Future[T]` 表示与最小 runtime 契约。
2. lowering：把 `async fn` 降级到状态机/轮询接口。
3. `await` 语义：接入表达式求值与错误传播规则。
4. 借用约束：明确“借用不得跨 `await`”的检查与诊断。

在以上项完成前，D03 仍视为未完成。
