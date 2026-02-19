# Async 模型（D03）

本章定义 Vox 的 async/await 设计方向。

结论：

1. 语言核心采用 **pull 模型**（Future + poll）。
2. push 模型作为边界适配能力（例如 UI 事件流、SSE、WebSocket）。
3. lowering 采用状态机（不是纯 CPS）。

## 1. 当前已落地

1. 关键字保留：`async`、`await`。
2. `async fn` 语法已接入 parser，AST 有 `FuncDecl.is_async`。
3. 语义已完整接入（D03 主体已完成）：
   - `async fn`（无 `await`）已进入正常 typecheck/codegen 管线（当前行为仍等价同步函数）
   - `await` 表达式已解析为 AST 节点（`ExprNode.Await`），并已接入 typecheck/irgen：仅允许在 `async fn` 中使用；推荐表面语法为 `e.await`（同时保留前缀 `await e` 兼容）。当前阶段支持通过 frame 状态机保留进度（frame.state + frame.a0/a1/...），`Pending => return Pending`，`Ready(v) => v`。
   - `trait async fn` 已支持（含 default body），并通过“隐式关联类型投影” desugar：对每个 async 方法自动引入 `type __async$<method>`，并把方法返回类型改写为 `Self.__async$<method>`；实现侧由编译器自动把该关联类型绑定到 lowering 后的 frame 类型。
4. async 入口与测试可运行（最小执行器，v0）：
   - 当构建可执行文件启用 driver main 时：若用户定义 `async fn main() -> T`，编译器在编译期生成一个同步 `fn main() -> T` wrapper。
   - 当构建测试二进制启用 test main 时：若发现 `async fn test_*() -> ()`，编译器为该 test 生成一个同步 wrapper 并交给测试运行器调用。
   - wrapper 内部使用轮询 `poll` 直到 `Ready`；`Pending` 分支优先使用 runtime 注入路径：若存在 `default_runtime()`，优先调用 `park_until_wake_with(rt, iter, cx)`，再回退 `park_with(rt, iter, cx)`、`pending_wait_with(rt, iter, cx)`；若不存在 runtime 注入路径，则回退 `park_until_wake(iter, cx)`、`park(iter, cx)`、`pending_wait(iter, cx)`、`spin_wait(iter)`，最后 pure continue。
   - 取消轮询同样优先 runtime 注入路径：`cancel_requested_with(rt, cx)`，其次回退 `cancel_requested(cx)`。
   - 取消分支支持可选 frame 重绑定钩子：优先 `cancel_drop_state_with(rt, cx, state, f)`，其次 `cancel_drop_with(rt, cx, f)`，再其次 `cancel_drop_state(cx, state, f)`、`cancel_drop(cx, f)`（用于宿主按 Future/frame 类型 + state 做更细粒度释放或重置）。
   - 取消资源清理支持可选钩子：优先 `cancel_cleanup_with(rt, cx)`，其次 `cancel_cleanup(cx)`（若存在则在取消返回前执行）。
   - 取消结果传播支持可选钩子：
     - 优先 `cancel_return_with(rt, cx)`，其次 `cancel_return(cx)`；
     - 若两者都不存在，回退为默认可恢复返回（`()` 或返回类型默认值）。

## 2. 为什么核心选 pull

1. 与 Rust-like 语义一致，心智模型稳定。
2. backpressure 自然：只有被 poll 才推进。
3. lowering 可控：`async fn -> Future state machine`。
4. 便于静态规则：后续可做“借用不得跨 await”。

## 3. 核心抽象（std/async）

```vox
enum Poll[T] { Pending, Ready(T) }

struct Waker { token: i64 }
struct Context { waker: Waker }

trait Future {
  type Output;
  fn poll(x: &mut Self, cx: &Context) -> Poll[Self.Output];
}

trait Runtime {
  fn pending_wait(rt: Self, i: i32, c: Context) -> ();
  fn park_until_wake(rt: Self, i: i32, c: Context) -> bool;
  fn cancel_requested(rt: Self, c: Context) -> bool;
}

struct ReadyPoll { ready: bool, token: i64 }
struct ReadyQueue { tokens: Vec[i64], head: i32 }
trait EventSource {
  fn wait(src: Self, timeout_ms: i32, c: Context) -> ReadyPoll;
}

fn wake(c: Context) -> ();
fn default_runtime() -> EventRuntime;
fn park_until_wake(i: i32, c: Context) -> bool;
fn park(i: i32, c: Context) -> ();
fn pending_wait(i: i32, c: Context) -> (); // 兼容别名，默认转到 park
fn cancel_requested(c: Context) -> bool;
fn cancel_drop_state_with[R, F](rt: R, c: Context, state: i32, f: F) -> F; // 可选钩子
fn cancel_drop_with[R, F](rt: R, c: Context, f: F) -> F; // 可选钩子
fn cancel_drop_state[F](c: Context, state: i32, f: F) -> F; // 可选钩子
fn cancel_drop[F](c: Context, f: F) -> F; // 可选钩子
fn cancel_cleanup_with[R](rt: R, c: Context) -> (); // 可选钩子
fn cancel_cleanup(c: Context) -> (); // 可选钩子
fn cancel_return_with[R, T](rt: R, c: Context) -> T; // 可选钩子
fn cancel_return[T](c: Context) -> T; // 可选钩子；不提供时回退默认返回
```

说明：

1. `Pending` 表示当前不能继续，需要由 waker 驱动下一次 poll。
2. `Ready(T)` 表示完成，返回结果。
3. `Context/Waker` 定义最小执行器接口契约；具体调度策略由 runtime/宿主决定。
4. `Runtime` trait 定义“Pending 时如何等待/让出 + 取消轮询”的 runtime 分层接口，当前标准库提供：
   - `EventRuntime`（基于 `__wake_wait(token, timeout_ms)` 的超时等待/唤醒消费，作为 `default_runtime()`）
   - `SpinRuntime`（`yield_now` 兼容路径）
   并暴露 `default_runtime()` + `*_with(rt, ...)` 供宿主注入自定义 runtime。
5. `EventSource + ReadyQueue` 提供“事件源抽象 + 多源就绪队列”基线：
   - `EventSource.wait(...)` 统一“单次等待 -> ReadyPoll”接口；
   - `ReadyQueue` 提供 token 队列（push/pop）作为多源事件汇聚结构；
   - `drain_ready_once(...)` 提供“多 context 单轮扫描并入队”基线，避免宿主重复拼接轮询样板；
   - 现阶段先用于接口收敛，后续由平台后端（epoll/kqueue/IOCP）填充具体事件源实现。

## 4. lowering 设计（D03-3 目标）

`async fn f(args) -> T` 语义改写为“返回 future 值”：

1. 编译器生成 frame 结构（局部变量 + 子 future + state tag）。
2. 生成对应 `poll(frame, cx) -> Poll[T]`：
   - 从 `state` 恢复执行点。
   - 遇到 `e.await`：先确保子 future 已初始化，再 poll。
   - 子 future `Pending`：保存 state，返回 `Pending`。
   - 子 future `Ready(v)`：恢复并继续执行。
   - 函数返回：`Ready(ret)`。

## 5. await 的类型/语义规则（当前实现）

1. `e.await`（或兼容语法 `await e`）可用于：
   - Poll-shaped 枚举 `{ Pending, Ready(T) }`
   - 或实现了 `std/async::Future` 的类型（要求其 `poll(...) -> Poll[Output]`）
2. `e.await` 的表达式类型为 `T`（或 `Future::Output`）。
3. lowering 语义：`Ready(v) => v`；`Pending` 时从 enclosing `poll` 返回 `Pending`，并通过 async frame（state + aN 字段）保留进度。
4. `await` 只能出现在 async 上下文（`async fn`，`async` block 后续引入）。
5. 当前 lowering 支持 `await` 出现在嵌套的语句块里（`if`/`while` 的 body 内），编译器会把控制流拆分成状态机分支/回边。
6. `await` 已支持一般表达式控制流场景：
   - `block` / `if` / `match` / `try` 表达式内部。
   - 宏调用参数内部。
7. `async fn` 支持局部变量 shadowing（同名 `let`）；async 归一化会先做词法作用域唯一化，再进入状态机/capture lowering。
8. 无生命周期标注的约束：`async fn` 会返回一个可逃逸的 Future/frame 值，因此其 **参数/输出类型中禁止出现非 `&'static` 的借用**（包括嵌套在 `Vec[...]`、struct/enum 字段中的借用）。否则借用将随 frame 逃逸，无法在类型系统中表达其有效期。

## 6. push 与 pull 的转换

核心规则：pull 是语言内核，push 在边界适配。

### 6.1 push -> pull

典型做法：

1. push 源写入队列。
2. push 源触发 waker。
3. pull 侧 future 在 poll 中消费队列，空则 `Pending`。

### 6.2 pull -> push

典型做法：

1. 执行器驱动 future poll。
2. `Ready(v)` 时调用 push 回调/下游 sink。
3. `Pending` 时让出执行权，等待唤醒。

`std/async` 中保留 `Sink` 作为最小边界契约：

```vox
trait Sink {
  type Item;
  fn push(x: &mut Self, v: Self.Item) -> bool;
  fn close(x: &mut Self) -> ();
}
```

## 7. 取消与 drop

约定：future 被 drop 视为取消。

1. 编译器生成的 frame 在 drop 时释放已初始化字段。
2. 必须保证“未初始化字段不 drop”。
3. 取消语义保持幂等（重复取消不出错）。
4. 当前 v1 已落地取消轮询与传播钩子：生成的 async entry/test wrapper 会在 `Pending` 路径查询取消钩子（优先 `cancel_requested_with(rt, cx)`，其次 `cancel_requested(cx)`）；命中后先执行可选 frame 重绑定（优先 `cancel_drop_state_with/cancel_drop_with`，回退 `cancel_drop_state/cancel_drop`），再执行可选清理（`cancel_cleanup_with/cancel_cleanup`），最后执行可选返回传播（`cancel_return_with/cancel_return`），否则回退默认返回（不 panic）。

## 8. 与借用规则的关系（D03-4 目标）

当前规则（已落地）：

1. 非 static 借用不得跨 `await` 存活。
2. 违反时报静态错误，并在诊断文本中同时给出借用创建点与跨越的 `await` 点。

这与 Vox “无用户生命周期标注”目标一致：

- 用户不写 `'a`。
- 编译器内部完成跨挂起点的借用安全检查。

## 9. 当前剩余工作

1. runtime/executor 体验继续增强（当前已支持 `default_runtime + *_with` 注入，默认 runtime 已切到 wake-token 超时等待基线，且有 `EventSource + ReadyQueue + drain_ready_once` 统一接口；后续补真正的 epoll/kqueue/IOCP 事件源实现与接线）。
2. drop/cancel 语义继续细化与验证（当前已支持“可恢复返回 + 可选清理/传播钩子”基线；后续补更细粒度资源回收策略）。
