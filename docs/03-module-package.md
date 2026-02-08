# 模块与包（草案）

## 目标

1. 目录即模块（减少声明样板）。
2. 默认私有，`pub` 显式公开。
3. 导入语法直接、可读。

## 模块模型

源码根目录默认 `src/`，**目录结构定义模块路径**；同一目录下的多个 `.vox` 文件属于同一个模块（类似 Go 的“一个目录一个包”）。

```
project/
  src/
    main.vox
    utils/
      fs.vox    // 与其他文件同属 utils 模块
      math.vox  // 与其他文件同属 utils 模块
      more.vox  // 与其他文件同属 utils 模块
    utils/io/
      file.vox  // 子模块 utils/io
```

模块路径规则（当前决策）：

- `src/` 目录本身是根模块；`src/*.vox` 都属于根模块（文件名不形成模块名）。
- `src/<dir>/*.vox` 属于模块 `<dir>`。
- `src/<dir>/<subdir>/*.vox` 属于模块 `<dir>/<subdir>`（以目录为边界）。

## 导入

导入路径使用字符串字面量：

```vox
import "utils"
import "utils/io"
import { read_file } from "utils/io"
import "utils" as u

// 显式命名空间（用于消歧义）：
import "pkg:dep" as dep   // 依赖包 dep（来自 vox.toml [dependencies]）
import "mod:dep" as dep   // 本地模块 dep（来自 src/dep/**）
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

命名导入也可以导入函数/类型（Stage0 最小子集）：

```vox
import { read_file, Point } from "utils/io"
fn main() -> i32 {
  let p: Point = Point { x: 1, y: 2 };
  return p.x;
}
```

规则（当前决策）：

- 使用 `pkg.name(...)` 形式访问依赖包符号时，必须在同一文件中先写 `import "pkg"`（或 `import "pkg" as alias` 后用 `alias.name(...)`）。
- Import 默认按“同包本地模块优先，其次依赖包”的策略解析；若出现歧义（同名本地模块与依赖包同时存在），必须用 `pkg:` / `mod:` 显式指定。

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
- `src/main.vox` 作为可执行入口，`src/` 下的其他 `.vox` 文件共同构成包的源码（库入口不依赖特定文件名）
