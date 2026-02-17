# Vox Pkgs

`pkgs/` 类似 crates 目录，放可复用包。

当前包含：

- `pkgs/openai`：OpenAI chat/completions 请求封装（curl + JSON 最小实现）
- `pkgs/genai`：provider/model 规格解析（如 `openai:gpt-5-nano`）
- `pkgs/llm`：统一入口 `llm.new("provider:model")` + `llm.complete(...)`
- `pkgs/mathlib`：基础示例库

后续可以继续把更多通用包集中到这里。
