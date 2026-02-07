# 平台支持（草案）

目标平台：

- Linux / macOS / Windows（native）
- WASM（浏览器/Node）
- 嵌入：生成 `staticlib` / `cdylib` 风格产物，提供 C ABI

构建目标三元组、链接器配置、交叉编译细节 deferred（将与 `vox build --target ...` 选项一起定义）。

