typedef struct vox_sync_handle_node { intptr_t h; struct vox_sync_handle_node* next; } vox_sync_handle_node;
static vox_sync_handle_node* vox_sync_handles = NULL;
static void vox_sync_handle_add(intptr_t h) {
  vox_sync_handle_node* n = (vox_sync_handle_node*)vox_impl_malloc(sizeof(vox_sync_handle_node));
  if (!n) { vox_host_panic("out of memory"); }
  n->h = h;
  n->next = vox_sync_handles;
  vox_sync_handles = n;
}
static bool vox_sync_handle_live(intptr_t h) {
  vox_sync_handle_node* n = vox_sync_handles;
  while (n) {
    if (n->h == h) return true;
    n = n->next;
  }
  return false;
}
static bool vox_sync_handle_remove(intptr_t h) {
  vox_sync_handle_node** cur = &vox_sync_handles;
  while (*cur) {
    vox_sync_handle_node* n = *cur;
    if (n->h == h) {
      *cur = n->next;
      vox_impl_free(n);
      return true;
    }
    cur = &n->next;
  }
  return false;
}

typedef struct { _Atomic int32_t value; } vox_atomic_i32;

intptr_t vox_impl_atomic_i32_new(int32_t init) {
  vox_atomic_i32* a = (vox_atomic_i32*)vox_impl_malloc(sizeof(vox_atomic_i32));
  if (!a) { vox_host_panic("out of memory"); }
  atomic_store_explicit(&a->value, init, memory_order_seq_cst);
  intptr_t h = (intptr_t)a;
  vox_sync_handle_add(h);
  return h;
}

int32_t vox_impl_atomic_i32_load(intptr_t h) {
  vox_atomic_i32* a = (vox_atomic_i32*)(intptr_t)h;
  if (!vox_sync_handle_live(h)) { vox_host_panic("invalid or dropped atomic handle"); }
  return atomic_load_explicit(&a->value, memory_order_seq_cst);
}

void vox_impl_atomic_i32_store(intptr_t h, int32_t v) {
  vox_atomic_i32* a = (vox_atomic_i32*)(intptr_t)h;
  if (!vox_sync_handle_live(h)) { vox_host_panic("invalid or dropped atomic handle"); }
  atomic_store_explicit(&a->value, v, memory_order_seq_cst);
}

int32_t vox_impl_atomic_i32_fetch_add(intptr_t h, int32_t delta) {
  vox_atomic_i32* a = (vox_atomic_i32*)(intptr_t)h;
  if (!vox_sync_handle_live(h)) { vox_host_panic("invalid or dropped atomic handle"); }
  return atomic_fetch_add_explicit(&a->value, delta, memory_order_seq_cst);
}

int32_t vox_impl_atomic_i32_swap(intptr_t h, int32_t v) {
  vox_atomic_i32* a = (vox_atomic_i32*)(intptr_t)h;
  if (!vox_sync_handle_live(h)) { vox_host_panic("invalid or dropped atomic handle"); }
  return atomic_exchange_explicit(&a->value, v, memory_order_seq_cst);
}

bool vox_impl_atomic_i32_cas(intptr_t h, int32_t expected, int32_t desired) {
  vox_atomic_i32* a = (vox_atomic_i32*)(intptr_t)h;
  if (!vox_sync_handle_live(h)) { vox_host_panic("invalid or dropped atomic handle"); }
  return atomic_compare_exchange_strong_explicit(&a->value, &expected, desired, memory_order_seq_cst, memory_order_seq_cst);
}

void vox_impl_atomic_i32_drop(intptr_t h) {
  if (!vox_sync_handle_remove(h)) return;
  vox_atomic_i32* a = (vox_atomic_i32*)(intptr_t)h;
  if (!a) return;
  vox_impl_free(a);
}

typedef struct { _Atomic int64_t value; } vox_atomic_i64;

intptr_t vox_impl_atomic_i64_new(int64_t init) {
  vox_atomic_i64* a = (vox_atomic_i64*)vox_impl_malloc(sizeof(vox_atomic_i64));
  if (!a) { vox_host_panic("out of memory"); }
  atomic_store_explicit(&a->value, init, memory_order_seq_cst);
  intptr_t h = (intptr_t)a;
  vox_sync_handle_add(h);
  return h;
}

int64_t vox_impl_atomic_i64_load(intptr_t h) {
  vox_atomic_i64* a = (vox_atomic_i64*)(intptr_t)h;
  if (!vox_sync_handle_live(h)) { vox_host_panic("invalid or dropped atomic handle"); }
  return atomic_load_explicit(&a->value, memory_order_seq_cst);
}

void vox_impl_atomic_i64_store(intptr_t h, int64_t v) {
  vox_atomic_i64* a = (vox_atomic_i64*)(intptr_t)h;
  if (!vox_sync_handle_live(h)) { vox_host_panic("invalid or dropped atomic handle"); }
  atomic_store_explicit(&a->value, v, memory_order_seq_cst);
}

int64_t vox_impl_atomic_i64_fetch_add(intptr_t h, int64_t delta) {
  vox_atomic_i64* a = (vox_atomic_i64*)(intptr_t)h;
  if (!vox_sync_handle_live(h)) { vox_host_panic("invalid or dropped atomic handle"); }
  return atomic_fetch_add_explicit(&a->value, delta, memory_order_seq_cst);
}

int64_t vox_impl_atomic_i64_swap(intptr_t h, int64_t v) {
  vox_atomic_i64* a = (vox_atomic_i64*)(intptr_t)h;
  if (!vox_sync_handle_live(h)) { vox_host_panic("invalid or dropped atomic handle"); }
  return atomic_exchange_explicit(&a->value, v, memory_order_seq_cst);
}

bool vox_impl_atomic_i64_cas(intptr_t h, int64_t expected, int64_t desired) {
  vox_atomic_i64* a = (vox_atomic_i64*)(intptr_t)h;
  if (!vox_sync_handle_live(h)) { vox_host_panic("invalid or dropped atomic handle"); }
  return atomic_compare_exchange_strong_explicit(&a->value, &expected, desired, memory_order_seq_cst, memory_order_seq_cst);
}

void vox_impl_atomic_i64_drop(intptr_t h) {
  if (!vox_sync_handle_remove(h)) return;
  vox_atomic_i64* a = (vox_atomic_i64*)(intptr_t)h;
  if (!a) return;
  vox_impl_free(a);
}

