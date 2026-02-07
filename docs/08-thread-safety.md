# 线程安全（Deferred）

本章暂缓。初步方向是沿用 Rust 的 `Send`/`Sync` 模型与自动推导，并要求跨线程共享与可变性必须通过同步原语（`Mutex`/`Atomic` 等）。

