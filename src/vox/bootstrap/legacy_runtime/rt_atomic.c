#ifndef VOX_RUNTIME_ATOMIC
#define VOX_RUNTIME_ATOMIC

int32_t rt_atomic_i32_load(void* p) { return atomic_load_explicit((_Atomic int32_t*)p, memory_order_seq_cst); }
void rt_atomic_i32_store(void* p, int32_t v) { atomic_store_explicit((_Atomic int32_t*)p, v, memory_order_seq_cst); }
bool rt_atomic_i32_cas(void* p, int32_t expected, int32_t desired) { return atomic_compare_exchange_strong_explicit((_Atomic int32_t*)p, &expected, desired, memory_order_seq_cst, memory_order_seq_cst); }
int32_t rt_atomic_i32_fetch_add(void* p, int32_t delta) { return atomic_fetch_add_explicit((_Atomic int32_t*)p, delta, memory_order_seq_cst); }
int64_t rt_atomic_i64_load(void* p) { return atomic_load_explicit((_Atomic int64_t*)p, memory_order_seq_cst); }
void rt_atomic_i64_store(void* p, int64_t v) { atomic_store_explicit((_Atomic int64_t*)p, v, memory_order_seq_cst); }
bool rt_atomic_i64_cas(void* p, int64_t expected, int64_t desired) { return atomic_compare_exchange_strong_explicit((_Atomic int64_t*)p, &expected, desired, memory_order_seq_cst, memory_order_seq_cst); }
int64_t rt_atomic_i64_fetch_add(void* p, int64_t delta) { return atomic_fetch_add_explicit((_Atomic int64_t*)p, delta, memory_order_seq_cst); }

#endif /* VOX_RUNTIME_ATOMIC */
