// rt_fs.c — thin C helpers for portable directory traversal.
// Walk algorithm lives in Vox (std/fs/file_common.vox).
// Uses C struct access for dirent portability across platforms/architectures.

#include <dirent.h>
#include <sys/stat.h>

void* rt_fs_opendir(const char* path) { return (void*)opendir(path); }
void* rt_fs_readdir(void* dir) { return (void*)readdir((DIR*)dir); }
int32_t rt_fs_closedir(void* dir) { return closedir((DIR*)dir); }

const char* rt_fs_dirent_name(void* ent) {
  return ((struct dirent*)ent)->d_name;
}

int32_t rt_fs_dirent_is_dir(const char* full_path) {
  struct stat st;
  if (stat(full_path, &st) == 0 && S_ISDIR(st.st_mode)) return 1;
  return 0;
}
