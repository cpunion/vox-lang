typedef struct { _Atomic int32_t value; } vox_atomic_i32;

intptr_t vox_impl_atomic_i32_new(int32_t init) {
  vox_atomic_i32* a = (vox_atomic_i32*)vox_impl_malloc(sizeof(vox_atomic_i32));
  if (!a) { vox_host_panic("out of memory"); }
  atomic_store_explicit(&a->value, init, memory_order_seq_cst);
  return (intptr_t)a;
}

int32_t vox_impl_atomic_i32_load(intptr_t h) {
  vox_atomic_i32* a = (vox_atomic_i32*)(intptr_t)h;
  return atomic_load_explicit(&a->value, memory_order_seq_cst);
}

void vox_impl_atomic_i32_store(intptr_t h, int32_t v) {
  vox_atomic_i32* a = (vox_atomic_i32*)(intptr_t)h;
  atomic_store_explicit(&a->value, v, memory_order_seq_cst);
}

int32_t vox_impl_atomic_i32_fetch_add(intptr_t h, int32_t delta) {
  vox_atomic_i32* a = (vox_atomic_i32*)(intptr_t)h;
  return atomic_fetch_add_explicit(&a->value, delta, memory_order_seq_cst);
}

int32_t vox_impl_atomic_i32_swap(intptr_t h, int32_t v) {
  vox_atomic_i32* a = (vox_atomic_i32*)(intptr_t)h;
  return atomic_exchange_explicit(&a->value, v, memory_order_seq_cst);
}

bool vox_impl_atomic_i32_cas(intptr_t h, int32_t expected, int32_t desired) {
  vox_atomic_i32* a = (vox_atomic_i32*)(intptr_t)h;
  return atomic_compare_exchange_strong_explicit(&a->value, &expected, desired, memory_order_seq_cst, memory_order_seq_cst);
}

void vox_impl_atomic_i32_drop(intptr_t h) {
  vox_atomic_i32* a = (vox_atomic_i32*)(intptr_t)h;
  if (!a) return;
  vox_impl_free(a);
}

typedef struct { _Atomic int64_t value; } vox_atomic_i64;

intptr_t vox_impl_atomic_i64_new(int64_t init) {
  vox_atomic_i64* a = (vox_atomic_i64*)vox_impl_malloc(sizeof(vox_atomic_i64));
  if (!a) { vox_host_panic("out of memory"); }
  atomic_store_explicit(&a->value, init, memory_order_seq_cst);
  return (intptr_t)a;
}

int64_t vox_impl_atomic_i64_load(intptr_t h) {
  vox_atomic_i64* a = (vox_atomic_i64*)(intptr_t)h;
  return atomic_load_explicit(&a->value, memory_order_seq_cst);
}

void vox_impl_atomic_i64_store(intptr_t h, int64_t v) {
  vox_atomic_i64* a = (vox_atomic_i64*)(intptr_t)h;
  atomic_store_explicit(&a->value, v, memory_order_seq_cst);
}

int64_t vox_impl_atomic_i64_fetch_add(intptr_t h, int64_t delta) {
  vox_atomic_i64* a = (vox_atomic_i64*)(intptr_t)h;
  return atomic_fetch_add_explicit(&a->value, delta, memory_order_seq_cst);
}

int64_t vox_impl_atomic_i64_swap(intptr_t h, int64_t v) {
  vox_atomic_i64* a = (vox_atomic_i64*)(intptr_t)h;
  return atomic_exchange_explicit(&a->value, v, memory_order_seq_cst);
}

bool vox_impl_atomic_i64_cas(intptr_t h, int64_t expected, int64_t desired) {
  vox_atomic_i64* a = (vox_atomic_i64*)(intptr_t)h;
  return atomic_compare_exchange_strong_explicit(&a->value, &expected, desired, memory_order_seq_cst, memory_order_seq_cst);
}

void vox_impl_atomic_i64_drop(intptr_t h) {
  vox_atomic_i64* a = (vox_atomic_i64*)(intptr_t)h;
  if (!a) return;
  vox_impl_free(a);
}

