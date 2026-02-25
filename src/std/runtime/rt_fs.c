static bool vox_str_has_suffix(const char* s, const char* suf) {
  if (!s || !suf) return false;
  size_t n = strlen(s);
  size_t m = strlen(suf);
  if (m > n) return false;
  return memcmp(s + (n - m), suf, m) == 0;
}

static char* vox_path_join2(const char* a, const char* b) {
  if (!a) a = "";
  if (!b) b = "";
  size_t na = strlen(a);
  size_t nb = strlen(b);
  bool slash = (na != 0 && a[na - 1] != '/');
  size_t n = na + (slash ? 1 : 0) + nb;
  char* out = (char*)vox_impl_malloc(n + 1);
  if (!out) { vox_host_panic("out of memory"); }
  memcpy(out, a, na);
  size_t j = na;
  if (slash) out[j++] = '/';
  memcpy(out + j, b, nb);
  out[n] = '\0';
  return out;
}

static void vox_walk_dir_suffix(vox_vec* out, const char* root, const char* rel, const char* suffix) {
  char* full = rel && rel[0] ? vox_path_join2(root, rel) : vox_path_join2(root, "");
  DIR* d = opendir(full);
  if (!d) { vox_impl_free(full); return; }
  struct dirent* ent;
  while ((ent = readdir(d)) != NULL) {
    const char* name = ent->d_name;
    if (!name || name[0] == '\0') continue;
    if (strcmp(name, ".") == 0 || strcmp(name, "..") == 0) continue;
    char* child_rel = rel && rel[0] ? vox_path_join2(rel, name) : vox_path_join2("", name);
    char* child_full = vox_path_join2(root, child_rel);
    struct stat st;
    if (stat(child_full, &st) == 0 && S_ISDIR(st.st_mode)) {
      vox_walk_dir_suffix(out, root, child_rel, suffix);
      vox_impl_free(child_full);
      vox_impl_free(child_rel);
      continue;
    }
    if (vox_str_has_suffix(child_rel, suffix)) {
      const char* s = child_rel;
      vox_vec_push(out, &s);
      vox_impl_free(child_full);
      continue;
    }
    vox_impl_free(child_full);
    vox_impl_free(child_rel);
  }
  closedir(d);
  vox_impl_free(full);
}

static int vox_cmp_str_ptr(const void* a, const void* b) {
  const char* sa = *(const char**)a;
  const char* sb = *(const char**)b;
  return strcmp(sa, sb);
}

static void vox_vec_sort_strings(vox_vec* v) {
  if (v->len <= 1) return;
  qsort(v->h->data, (size_t)v->len, (size_t)v->h->elem_size, vox_cmp_str_ptr);
}

vox_vec vox_impl_walk_vox_files(const char* root) {
  if (!root || root[0] == '\0') root = ".";
  vox_vec out = vox_vec_new((int32_t)sizeof(const char*));
  vox_walk_dir_suffix(&out, root, "src", ".vox");
  vox_walk_dir_suffix(&out, root, "tests", ".vox");
  vox_vec_sort_strings(&out);
  return out;
}

vox_vec vox_impl_walk_c_files(const char* root) {
  if (!root || root[0] == '\0') root = ".";
  vox_vec out = vox_vec_new((int32_t)sizeof(const char*));
  vox_walk_dir_suffix(&out, root, "src", ".c");
  vox_vec_sort_strings(&out);
  return out;
}

