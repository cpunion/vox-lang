#if defined(_WIN32)
extern int _write(int, const void*, unsigned int);
#if defined(_MSC_VER)
#pragma comment(lib, "ws2_32.lib")
#endif
#else
#include <errno.h>
#include <unistd.h>
#include <sched.h>
#endif

#if defined(_MSC_VER)
#  define VOX_NORETURN __declspec(noreturn)
#elif defined(__GNUC__) || defined(__clang__)
#  define VOX_NORETURN __attribute__((noreturn))
#else
#  define VOX_NORETURN
#endif

static VOX_NORETURN void vox_host_panic(const char* msg) {
  if (!msg) msg = "";
  size_t n = strlen(msg);
#ifdef _WIN32
  _write(2, msg, (unsigned int)n);
  _write(2, "\n", 1);
#else
  write(2, msg, n);
  write(2, "\n", 1);
#endif
  exit(1);
}

typedef uint8_t vox_unit;
typedef struct { uint8_t* data; int32_t cap; int32_t elem_size; } vox_vec_data;
typedef struct { vox_vec_data* h; int32_t len; } vox_vec;
typedef struct vox_alloc_node { void* ptr; struct vox_alloc_node* next; } vox_alloc_node;
static vox_alloc_node* vox_alloc_head = NULL;
static bool vox_alloc_cleanup_registered = false;
static void vox_impl_cleanup(void) {
  vox_alloc_node* cur = vox_alloc_head;
  vox_alloc_head = NULL;
  while (cur) {
    vox_alloc_node* next = cur->next;
    if (cur->ptr) free(cur->ptr);
    free(cur);
    cur = next;
  }
}
static void vox_impl_register_cleanup(void) {
  if (vox_alloc_cleanup_registered) return;
  if (atexit(vox_impl_cleanup) != 0) { vox_host_panic("register atexit cleanup failed"); }
  vox_alloc_cleanup_registered = true;
}
static void* vox_impl_malloc(size_t n) {
  if (n == 0) n = 1;
  void* p = malloc(n);
  if (!p) { vox_host_panic("out of memory"); }
  vox_impl_register_cleanup();
  vox_alloc_node* node = (vox_alloc_node*)malloc(sizeof(vox_alloc_node));
  if (!node) { vox_host_panic("out of memory"); }
  node->ptr = p;
  node->next = vox_alloc_head;
  vox_alloc_head = node;
  return p;
}
static void* vox_impl_realloc(void* old_ptr, size_t n) {
  if (!old_ptr) return vox_impl_malloc(n);
  if (n == 0) n = 1;
  void* p = realloc(old_ptr, n);
  if (!p) { vox_host_panic("out of memory"); }
  vox_alloc_node* cur = vox_alloc_head;
  while (cur) {
    if (cur->ptr == old_ptr) {
      cur->ptr = p;
      return p;
    }
    cur = cur->next;
  }
  vox_impl_register_cleanup();
  vox_alloc_node* node = (vox_alloc_node*)malloc(sizeof(vox_alloc_node));
  if (!node) { vox_host_panic("out of memory"); }
  node->ptr = p;
  node->next = vox_alloc_head;
  vox_alloc_head = node;
  return p;
}
static bool vox_impl_forget(void* p) {
  if (!p) return false;
  vox_alloc_node** cur = &vox_alloc_head;
  while (*cur) {
    vox_alloc_node* node = *cur;
    if (node->ptr == p) {
      *cur = node->next;
      free(node);
      return true;
    }
    cur = &node->next;
  }
  return false;
}
static void vox_impl_free(void* p) {
  if (!p) return;
  bool tracked = vox_impl_forget(p);
  if (!tracked) return;
  free(p);
}
static vox_vec_data* vox_vec_data_new(int32_t elem_size) {
  vox_vec_data* h = (vox_vec_data*)vox_impl_malloc(sizeof(vox_vec_data));
  if (!h) { vox_host_panic("out of memory"); }
  h->data = NULL;
  h->cap = 0;
  h->elem_size = elem_size;
  return h;
}
static vox_vec vox_vec_new(int32_t elem_size) {
  vox_vec v; v.h = vox_vec_data_new(elem_size); v.len = 0; return v;
}
static void vox_vec_grow(vox_vec* v, int32_t new_cap) {
  if (!v || !v->h) { vox_host_panic("vec grow invalid vec"); }
  if (new_cap <= v->h->cap) return;
  if (new_cap < 4) new_cap = 4;
  size_t bytes = (size_t)new_cap * (size_t)v->h->elem_size;
  uint8_t* p = (uint8_t*)vox_impl_realloc(v->h->data, bytes);
  if (!p) { vox_host_panic("out of memory"); }
  v->h->data = p;
  v->h->cap = new_cap;
}
static void vox_vec_push(vox_vec* v, const void* elem) {
  if (!v || !v->h || !elem) { vox_host_panic("vec push invalid args"); }
  if (v->len == v->h->cap) { int32_t nc = v->h->cap == 0 ? 4 : v->h->cap * 2; vox_vec_grow(v, nc); }
  memcpy(v->h->data + (size_t)v->len * (size_t)v->h->elem_size, elem, (size_t)v->h->elem_size);
  v->len++;
}
static void vox_vec_insert(vox_vec* v, int32_t idx, const void* elem) {
  if (!v || !v->h || !elem) { vox_host_panic("vec insert invalid args"); }
  if (idx < 0 || idx > v->len) { char buf[96]; snprintf(buf, sizeof(buf), "vec insert index out of bounds: idx=%" PRId32 " len=%" PRId32, idx, v->len); vox_host_panic(buf); }
  if (v->len == v->h->cap) { int32_t nc = v->h->cap == 0 ? 4 : v->h->cap * 2; vox_vec_grow(v, nc); }
  uint8_t* ptr = v->h->data + (size_t)idx * (size_t)v->h->elem_size;
  int32_t tail = v->len - idx;
  if (tail > 0) {
    memmove(ptr + (size_t)v->h->elem_size, ptr, (size_t)tail * (size_t)v->h->elem_size);
  }
  memcpy(ptr, elem, (size_t)v->h->elem_size);
  v->len++;
}
static void vox_vec_set(vox_vec* v, int32_t idx, const void* elem) {
  if (!v || !v->h || !elem) { vox_host_panic("vec set invalid args"); }
  if (idx < 0 || idx >= v->len) { char buf[96]; snprintf(buf, sizeof(buf), "vec set index out of bounds: idx=%" PRId32 " len=%" PRId32, idx, v->len); vox_host_panic(buf); }
  memcpy(v->h->data + (size_t)idx * (size_t)v->h->elem_size, elem, (size_t)v->h->elem_size);
}
static void vox_vec_clear(vox_vec* v) {
  if (!v) return;
  v->len = 0;
}
static void vox_vec_extend(vox_vec* v, const vox_vec* other) {
  if (!v || !v->h || !other || !other->h) return;
  if (other->len <= 0) return;
  if (v->h->elem_size != other->h->elem_size) { vox_host_panic("vec extend elem_size mismatch"); }
  int64_t need64 = (int64_t)v->len + (int64_t)other->len;
  if (need64 > INT32_MAX) { vox_host_panic("vec too large"); }
  int32_t need = (int32_t)need64;
  if (need > v->h->cap) {
    int32_t nc = v->h->cap == 0 ? 4 : v->h->cap;
    while (nc < need) {
      if (nc > INT32_MAX / 2) { nc = need; break; }
      nc = nc * 2;
    }
    vox_vec_grow(v, nc);
  }
  memcpy(v->h->data + (size_t)v->len * (size_t)v->h->elem_size, other->h->data, (size_t)other->len * (size_t)other->h->elem_size);
  v->len = need;
}
static void vox_vec_pop(vox_vec* v, void* out) {
  if (!v || !v->h || !out) { vox_host_panic("vec pop invalid args"); }
  if (v->len <= 0) { vox_host_panic("vec pop on empty vector"); }
  int32_t idx = v->len - 1;
  memcpy(out, v->h->data + (size_t)idx * (size_t)v->h->elem_size, (size_t)v->h->elem_size);
  v->len = idx;
}
static void vox_vec_remove(vox_vec* v, int32_t idx, void* out) {
  if (!v || !v->h || !out) { vox_host_panic("vec remove invalid args"); }
  if (idx < 0 || idx >= v->len) { char buf[96]; snprintf(buf, sizeof(buf), "vec remove index out of bounds: idx=%" PRId32 " len=%" PRId32, idx, v->len); vox_host_panic(buf); }
  uint8_t* ptr = v->h->data + (size_t)idx * (size_t)v->h->elem_size;
  memcpy(out, ptr, (size_t)v->h->elem_size);
  int32_t tail = v->len - idx - 1;
  if (tail > 0) {
    memmove(ptr, ptr + (size_t)v->h->elem_size, (size_t)tail * (size_t)v->h->elem_size);
  }
  v->len = v->len - 1;
}
static int32_t vox_vec_len(const vox_vec* v) { return v ? v->len : 0; }
static bool vox_vec_eq(const vox_vec* a, const vox_vec* b) {
  if (!a || !b) return false;
  if (a->len != b->len) return false;
  if (!a->h || !b->h) return a->len == 0 && b->len == 0;
  if (a->h->elem_size != b->h->elem_size) return false;
  size_t bytes = (size_t)a->len * (size_t)a->h->elem_size;
  if (bytes == 0) return true;
  return memcmp(a->h->data, b->h->data, bytes) == 0;
}
static void vox_vec_get(const vox_vec* v, int32_t idx, void* out) {
  if (!v || !v->h || !out) { vox_host_panic("vec get invalid args"); }
  if (idx < 0 || idx >= v->len) { char buf[96]; snprintf(buf, sizeof(buf), "vec index out of bounds: idx=%" PRId32 " len=%" PRId32, idx, v->len); vox_host_panic(buf); }
  memcpy(out, v->h->data + (size_t)idx * (size_t)v->h->elem_size, (size_t)v->h->elem_size);
}

static int32_t vox_str_len(const char* s) {
  if (!s) return 0;
  size_t n = strlen(s);
  if (n > INT32_MAX) { vox_host_panic("string too long"); }
  return (int32_t)n;
}
static int32_t vox_str_byte_at(const char* s, int32_t idx) {
  int32_t n = vox_str_len(s);
  if (idx < 0 || idx >= n) { vox_host_panic("string index out of bounds"); }
  return (int32_t)(uint8_t)s[idx];
}

static const char* vox_str_slice(const char* s, int32_t start, int32_t end) {
  int32_t n = vox_str_len(s);
  if (start < 0 || end < start || end > n) { vox_host_panic("string slice out of bounds"); }
  int32_t m = end - start;
  char* out = (char*)vox_impl_malloc((size_t)m + 1);
  if (!out) { vox_host_panic("out of memory"); }
  memcpy(out, s + start, (size_t)m);
  out[m] = '\0';
  return out;
}

static const char* vox_str_concat(const char* a, const char* b) {
  if (!a) a = "";
  if (!b) b = "";
  size_t na = strlen(a);
  size_t nb = strlen(b);
  if (na + nb + 1 > SIZE_MAX) { vox_host_panic("string too long"); }
  char* out = (char*)vox_impl_malloc(na + nb + 1);
  if (!out) { vox_host_panic("out of memory"); }
  memcpy(out, a, na);
  memcpy(out + na, b, nb);
  out[na + nb] = '\0';
  return out;
}

static bool vox_str_starts_with(const char* s, const char* pre) {
  if (!s) s = "";
  if (!pre) pre = "";
  size_t ns = strlen(s);
  size_t np = strlen(pre);
  if (np > ns) return false;
  return memcmp(s, pre, np) == 0;
}

static bool vox_str_ends_with(const char* s, const char* suf) {
  if (!s) s = "";
  if (!suf) suf = "";
  size_t ns = strlen(s);
  size_t nf = strlen(suf);
  if (nf > ns) return false;
  return memcmp(s + (ns - nf), suf, nf) == 0;
}

static bool vox_str_contains(const char* s, const char* needle) {
  if (!s) s = "";
  if (!needle) needle = "";
  return strstr(s, needle) != NULL;
}

static int32_t vox_str_index_of(const char* s, const char* needle) {
  if (!s) s = "";
  if (!needle) needle = "";
  if (needle[0] == '\0') return 0;
  const char* p = strstr(s, needle);
  if (!p) return -1;
  size_t idx = (size_t)(p - s);
  if (idx > (size_t)INT32_MAX) { vox_host_panic("string index overflow"); }
  return (int32_t)idx;
}

static int32_t vox_str_last_index_of(const char* s, const char* needle) {
  if (!s) s = "";
  if (!needle) needle = "";
  int32_t ns = vox_str_len(s);
  int32_t nn = vox_str_len(needle);
  if (nn == 0) return ns;
  if (ns < nn) return -1;
  int32_t last = -1;
  for (int32_t i = 0; i <= ns - nn; i++) {
    if (memcmp(s + i, needle, (size_t)nn) == 0) last = i;
  }
  return last;
}

static const char* vox_i32_to_string(int32_t v) {
  char buf[32];
  int n = snprintf(buf, sizeof(buf), "%" PRId32, v);
  if (n < 0 || (size_t)n >= sizeof(buf)) { vox_host_panic("format failed or buffer too small"); }
  char* out = (char*)vox_impl_malloc((size_t)n + 1);
  if (!out) { vox_host_panic("out of memory"); }
  memcpy(out, buf, (size_t)n + 1);
  return out;
}

static const char* vox_i64_to_string(int64_t v) {
  char buf[32];
  int n = snprintf(buf, sizeof(buf), "%" PRId64, v);
  if (n < 0 || (size_t)n >= sizeof(buf)) { vox_host_panic("format failed or buffer too small"); }
  char* out = (char*)vox_impl_malloc((size_t)n + 1);
  if (!out) { vox_host_panic("out of memory"); }
  memcpy(out, buf, (size_t)n + 1);
  return out;
}

static const char* vox_u64_to_string(uint64_t v) {
  char buf[32];
  int n = snprintf(buf, sizeof(buf), "%" PRIu64, v);
  if (n < 0 || (size_t)n >= sizeof(buf)) { vox_host_panic("format failed or buffer too small"); }
  char* out = (char*)vox_impl_malloc((size_t)n + 1);
  if (!out) { vox_host_panic("out of memory"); }
  memcpy(out, buf, (size_t)n + 1);
  return out;
}

static const char* vox_isize_to_string(intptr_t v) {
  char buf[32];
  int n = snprintf(buf, sizeof(buf), "%" PRIdPTR, v);
  if (n < 0 || (size_t)n >= sizeof(buf)) { vox_host_panic("format failed or buffer too small"); }
  char* out = (char*)vox_impl_malloc((size_t)n + 1);
  if (!out) { vox_host_panic("out of memory"); }
  memcpy(out, buf, (size_t)n + 1);
  return out;
}

static const char* vox_usize_to_string(uintptr_t v) {
  char buf[32];
  int n = snprintf(buf, sizeof(buf), "%" PRIuPTR, v);
  if (n < 0 || (size_t)n >= sizeof(buf)) { vox_host_panic("format failed or buffer too small"); }
  char* out = (char*)vox_impl_malloc((size_t)n + 1);
  if (!out) { vox_host_panic("out of memory"); }
  memcpy(out, buf, (size_t)n + 1);
  return out;
}

static const char* vox_f32_to_string(float v) {
  char buf[64];
  int n = snprintf(buf, sizeof(buf), "%.9g", (double)v);
  if (n < 0 || (size_t)n >= sizeof(buf)) { vox_host_panic("format failed or buffer too small"); }
  char* out = (char*)vox_impl_malloc((size_t)n + 1);
  if (!out) { vox_host_panic("out of memory"); }
  memcpy(out, buf, (size_t)n + 1);
  return out;
}

static const char* vox_f64_to_string(double v) {
  char buf[64];
  int n = snprintf(buf, sizeof(buf), "%.17g", v);
  if (n < 0 || (size_t)n >= sizeof(buf)) { vox_host_panic("format failed or buffer too small"); }
  char* out = (char*)vox_impl_malloc((size_t)n + 1);
  if (!out) { vox_host_panic("out of memory"); }
  memcpy(out, buf, (size_t)n + 1);
  return out;
}

static const char* vox_bool_to_string(bool v) {
  return v ? "true" : "false";
}

static const char* vox_str_escape_c(const char* s) {
  if (!s) s = "";
  size_t n = strlen(s);
  // First pass: compute output length.
  size_t out_n = 0;
  for (size_t i = 0; i < n; i++) {
    unsigned char ch = (unsigned char)s[i];
    switch (ch) {
    case '\\':
    case '"':
    case '\n':
    case '\r':
    case '\t':
      out_n += 2;
      break;
    default:
      if (ch >= 0x20 && ch <= 0x7e) out_n += 1; else out_n += 4; // \\xHH
      break;
    }
  }
  char* out = (char*)vox_impl_malloc(out_n + 1);
  if (!out) { vox_host_panic("out of memory"); }
  size_t j = 0;
  for (size_t i = 0; i < n; i++) {
    unsigned char ch = (unsigned char)s[i];
    switch (ch) {
    case '\\': out[j++] = '\\'; out[j++] = '\\'; break;
    case '"': out[j++] = '\\'; out[j++] = '"'; break;
    case '\n': out[j++] = '\\'; out[j++] = 'n'; break;
    case '\r': out[j++] = '\\'; out[j++] = 'r'; break;
    case '\t': out[j++] = '\\'; out[j++] = 't'; break;
    default:
      if (ch >= 0x20 && ch <= 0x7e) { out[j++] = (char)ch; }
      else {
        static const char* hex = "0123456789abcdef";
        out[j++] = '\\'; out[j++] = 'x';
        out[j++] = hex[(ch >> 4) & 0xf];
        out[j++] = hex[ch & 0xf];
      }
      break;
    }
  }
  out[j] = '\0';
  return out;
}

static const char* vox_vec_str_join(const vox_vec* v, const char* sep) {
  if (!sep) sep = "";
  if (!v || !v->h || v->h->elem_size != (int32_t)sizeof(const char*)) { vox_host_panic("vec_str_join expects Vec[String]"); }
  int32_t n = v->len;
  const char* const* items = (const char* const*)v->h->data;
  size_t sep_n = strlen(sep);
  size_t total = 0;
  for (int32_t i = 0; i < n; i++) {
    const char* s = items[i];
    if (!s) s = "";
    total += strlen(s);
    if (i + 1 < n) total += sep_n;
  }
  if (total + 1 > SIZE_MAX) { vox_host_panic("string too long"); }
  char* out = (char*)vox_impl_malloc(total + 1);
  if (!out) { vox_host_panic("out of memory"); }
  size_t j = 0;
  for (int32_t i = 0; i < n; i++) {
    const char* s = items[i];
    if (!s) s = "";
    size_t m = strlen(s);
    memcpy(out + j, s, m);
    j += m;
    if (i + 1 < n && sep_n != 0) {
      memcpy(out + j, sep, sep_n);
      j += sep_n;
    }
  }
  out[j] = '\0';
  return out;
}

