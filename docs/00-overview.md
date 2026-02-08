# Vox 语言设计概览

> 本文档用于锁定 Vox 的“总体框架”。细节规范按主题拆分到其它文档，并可迭代修改。

## 设计目标

1. Rust-like 的系统语言体验：静态强类型、RAII、零成本抽象、C/WASM 互操作。
2. 强编译期执行：用普通代码在编译期求值、反射与生成（类似 Zig 的 `comptime`）。
3. 简化内存模型：不暴露用户生命周期标注；避免悬垂引用；不引入全自动 GC 作为核心前提。
4. 默认不可变：`let` 默认不可变，显式 `mut` 才可变。

## 非目标（当前阶段）

1. effect/IO/资源系统：**暂不设计**（后续再引入，避免过早绑定整体语义）。
2. 并发/async：保留方向，但不作为本阶段规范核心。

## 核心设计决策（已定）

| 领域 | 决策 |
|---|---|
| 语法风格 | Rust-like（花括号、`fn/let/match` 等） |
| 类型系统 | 静态强类型 + 类型推断 + 泛型 + trait |
| 错误处理 | `Result[T, E]` + `Option[T]` + `?` + `try {}` |
| 编译期执行 | `comptime`（纯、可反射、可生成） |
| 宏 | AST 宏（普通 `fn` 返回 AST，`quote`/`$` 插值；通过 `name!(...)` 在展开阶段执行） |
| 平台 | Linux/macOS/Windows 原生 + 可嵌入（C ABI）+ WASM 原生 + JS 互操作 |
| 内存管理 | RAII + 所有权；引用语义与 Rust 对齐命名；**借用引用为“临时引用”**（见内存规范） |
| 允许的生命周期标注 | 仅允许 `&'static T`（无其它 `'a`） |

## 语义总览

### 1) 值与所有权（Rust 命名）

- `T`：拥有值（move 语义 + RAII drop）。`String/Vec` 也属于 `T`，只是内部持有堆资源。
- `Box[T]`：唯一堆所有权（move-only）。
- `Arc[T]` / `Weak[T]`：共享堆所有权（引用计数）。
- `&T` / `&mut T`：借用引用（**Vox 中为临时引用**，不能长期保存/返回/捕获；见 `docs/07-memory-management.md`）。
- `&'static T`：允许长期保存（唯一允许的显式 lifetime 标注）。

### 2) 字符串与切片

- 字面量：`&'static str`（可按 `&str` 使用）
- 拥有字符串：`String`
- 临时子串：`&str`（不能逃逸）
- 可长期保存的子串：标准库类型 `StrView { owner: Arc[str], lo, hi }`（草案）

### 3) 编译期执行（`comptime`）

Vox 允许在编译期解释执行一段受限的程序，目标包括：

- 计算常量与 const 泛型参数
- 类型反射（查询字段/布局/类型类别等）
- 受控的代码生成（配合 AST 宏）

细节见 `docs/04-generics-comptime.md` 与 `docs/05-comptime-detailed.md`。

### 4) 表达式块（Block Expression）

在表达式位置允许使用“块表达式”，用于在一个表达式里写多条语句并以最后一个表达式作为结果：

```vox
let x: i32 = {
  let a: i32 = 40;
  a + 2
};

let y: i32 = if cond {
  let t: i32 = 1;
  t + 1
} else {
  0
};

let z: i32 = match v {
  E.A(n) => {
    let m: i32 = n + 1;
    m
  },
  E.None => 0,
};
```

## 文档结构

规范拆分按编号组织，参考 `docs/README.md`。
