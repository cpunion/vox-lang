# 模块与包（草案）

## 目标

1. 目录即模块（减少声明样板）。
2. 默认私有，`pub` 显式公开。
3. 导入语法直接、可读。

## 模块模型

源码根目录默认 `src/`，目录结构定义模块路径：

```
project/
  src/
    main.vox
    utils/
      lib.vox
      io.vox
```

## 导入

导入路径使用字符串字面量：

```vox
import "utils"
import { read_file } from "utils/io"
import "utils" as u
```

导入后，模块名（或别名）作为命名空间，通过 `.` 访问其中的符号：

```vox
import "utils"
utils.helper()

import "utils" as u
u.helper()
```

类型位置同样支持 `module.Type`：

```vox
import "utils" as u
fn f(p: u.Point) -> i32 { return p.x; }
```

规则（当前决策）：

- 使用 `pkg.name(...)` 形式访问依赖包符号时，必须在同一文件中先写 `import "pkg"`（或 `import "pkg" as alias` 后用 `alias.name(...)`）。

## 导出

```vox
fn internal() {}

pub fn public_api() {}

pub struct Api {
  pub name: String,
  cache: Cache, // 私有字段
}
```

## 包（草案）

- 包清单文件：`vox.toml`（字段与语义待定）
- `src/main.vox` 作为可执行入口，`src/lib.vox` 作为库入口（可同时存在）
