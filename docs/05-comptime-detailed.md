# Comptime 详解（草案）

## 确定性边界

编译期执行要求可复现：

- 允许：控制流、递归、循环、数组/结构体操作、纯计算、受控反射。
- 禁止（默认）：文件 IO、网络、系统时间、随机数、进程环境等非确定性来源。

后续若需要“受控编译期 IO”，应通过显式的构建系统接口完成（不在语言核心规范中隐式开放）。

## comptime 可执行性：推导 + 可选注解（已定）

Vox 不要求在函数上写 `comptime` 关键字。编译器在分析阶段推导“该函数是否可在编译期执行”。

执行规则（草案）：

- 任何 `fn` 在满足“确定性边界”的前提下，都可能被编译期解释器执行。
- 当某次调用发生在编译期上下文（例如 `const` 初始化、`comptime {}`、宏展开）时：
  - 若被调用函数可 comptime 执行：在编译期执行。
  - 否则：编译错误。

可选注解：

```vox
@comptime
fn crc_table() -> [u32; 256] { ... }
```

含义：要求该函数必须满足 comptime 约束；若包含不允许的操作（IO/随机/系统时间等）则直接报错。

## 反射内建（Stage1 已实现子集）

当前已实现：

```vox
@size_of(T) -> usize
@align_of(T) -> usize
@type_name(T) -> String
@field_count(T) -> usize
@field_name(T, I) -> String
@is_integer(T) -> bool
@is_float(T) -> bool
@is_struct(T) -> bool
@is_enum(T) -> bool
```

语法与约束（当前）：

- 调用形态固定为 `@name(Type)`（单一类型参数）。
- 可用于普通表达式与 `const` 初始化（均会折叠为常量）。
- `@size_of/@align_of` 当前按 Stage1 C 后端目标布局模型计算。
- `@type_name` 返回编译器的类型显示字符串。
- `@field_count` 当前支持 `struct/enum`（分别返回字段数/variant 数）。
- `@field_name` 当前支持 `struct/enum`（分别返回字段名/variant 名），`I` 为 `usize` const 实参。
- `@is_*` 返回类型分类判定（当前要求 `T` 为 concrete type）。

暂未实现（保留方向）：

```vox
@field_type(T, index: usize) -> type
```

## comptime 报错

建议提供：

```vox
@compile_error(msg: &'static str) -> !
```

## 资源限制

编译器可对 comptime 执行施加上限（步数/时间/内存），超限时报错并提示可能的无限递归/爆炸展开。
