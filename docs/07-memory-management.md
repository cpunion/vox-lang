# 内存管理（Vox：无用户生命周期标注）

> 说明：本章的“借用不可逃逸”是目标语义。Stage2 当前是过渡实现，`&T` / `&mut T` / `&'static T` / `&'static mut T` 在类型检查阶段先擦除为 `T`，因此尚未强制这些借用位置约束。

## 核心原则

1. RAII：离开作用域自动 `drop`。
2. 所有权：`T` 是拥有值，默认 move 语义（`Copy` 类型按位复制）。
3. 借用引用：`&T` / `&mut T` **仅用于临时借用**，避免引入通用 lifetime 系统。
4. 唯一允许的显式 lifetime：`'static`（`&'static T` 可长期保存）。

## 值、Box、Arc

```vox
fn takes(v: String) { /* v 拥有 */ }

fn demo() {
  let s = String::from("hi");
  takes(s);        // move
}
```

```vox
let b: Box[Node] = Box::new(Node { ... });   // 唯一堆所有权
let a: Arc[Node] = Arc::new(Node { ... });   // 共享堆所有权
let w: Weak[Node] = Arc::downgrade(&a);
```

## 临时借用（Vox 的关键差异，目标语义）

### 规则

`&T` / `&mut T` 不能“逃逸”：

- 不能作为结构体字段类型
- 不能作为函数返回类型
- 不能被闭包长期捕获（闭包/生成器若要持有数据，必须持有 `T/Box/Arc/...`）
- 不能存入容器（例如 `Vec[&T]`）

允许：

- 作为参数类型（`fn f(x: &T)`）
- 作为局部临时绑定（`let r = &x;`），但该值不得逃逸上述位置

这条规则的目标是：在不允许用户写 `'a` 的前提下，彻底消除“引用生命周期关系”在 API 设计中的外显。

### 示例：允许的借用

```vox
fn len(s: &str) -> usize { s.len() }

fn demo() {
  let s = String::from("hello");
  let n = len(&s);
}
```

### 示例：禁止的引用字段

```vox
struct Parser {
  // error: non-static borrowed reference is not storable
  input: &str,
}
```

## `'static`（唯一允许的长期引用）

```vox
const NAME: &'static str = "vox";

struct Config {
  app_name: &'static str, // OK
}

fn default_name() -> &'static str { "vox" }
```

## 可长期保存的子串/子切片（标准库草案）

为替代“返回/存储 `&str`”，标准库提供拥有型视图：

```vox
struct StrView {
  owner: Arc[str],
  lo: u32,
  hi: u32,
}

impl StrView {
  fn as_str(&self) -> &str { /* 临时借用 */ }
  fn sub(&self, lo: u32, hi: u32) -> StrView { /* range 叠加 */ }
}
```

同理当前标准库也提供拥有型 `Slice[T]`（`owner: Vec[T] + lo/hi`）用于长期保存子切片。

当前实现注记（stage1/stage2）：

- 标准库 `std/string` 已落地 `StrView`，当前字段模型为 `owner: String + lo/hi`（拥有型视图）。
- 标准库 `std/collections` 已落地 `Slice[T]`，当前字段模型为 `owner: Vec[T] + lo/hi`（拥有型视图）。
- 语法级 `&T` / `&mut T` / `&'static T` / `&'static mut T` 已在 stage2 接入（当前过渡语义等价 `T`）；`Arc[str]` 仍是后续演进方向。长期保存子串仍建议使用 `StrView`。
- Stage2 C 后端运行时已加入“分配跟踪 + 进程退出清理（`atexit`）”机制，用于回收运行期动态分配的字符串/容器 backing storage；这不是语言级 `drop`，也不改变现有所有权语义。

## Unsafe

`unsafe` 仍然存在，用于：

- 裸指针解引用
- FFI
- 自定义容器/性能关键代码中的不变量维护

Safe Vox 的目标是：不允许出现悬垂引用这一类“生命周期相关 UB”。
