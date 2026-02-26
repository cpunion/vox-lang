// Thin wrapper: @ffi_import generates void* return which conflicts with
// char* getenv() in stdlib.h (included transitively via windows.h on MinGW).
extern char* getenv(const char*);
const char* vox_impl_getenv(const char* key) {
  const char* v = getenv(key);
  return v ? v : "";
}

// Legacy stubs: the bootstrap compiler's generated main() still assigns
// vox__argc and vox__argv. Keep these so generated C compiles until
// bootstrap is bumped to a version that no longer emits them.
static int vox__argc = 0;
static char** vox__argv = NULL;

// Minimal argv helper: dereferences argv[i] from an intptr_t argv handle.
// State (argc/argv) is now stored in Vox static mut variables; this only
// does the pointer dereference that @ptr_read[rawptr] would do once the
// bootstrap compiler supports that intrinsic.
const char* vox_impl_argv_get(intptr_t argv, int32_t i) {
  char** p = (char**)argv;
  if (!p || i < 0) return "";
  return p[i] ? p[i] : "";
}

#ifndef _WIN32
// fcntl is variadic in system headers; keep a thin wrapper to avoid
// conflicting extern declarations between non-variadic FFI and variadic header.
extern int fcntl(int, int, ...);
int32_t vox_impl_fcntl3(int32_t fd, int32_t cmd, int32_t arg) {
  return fcntl(fd, cmd, arg);
}
#endif
// Forward declarations needed by Vox-generated FFI extern declarations.
struct sockaddr;
struct addrinfo;

#ifdef _WIN32
typedef uintptr_t SOCKET;
extern int __stdcall connect(SOCKET, const struct sockaddr*, int);
extern int __stdcall bind(SOCKET, const struct sockaddr*, int);
extern SOCKET __stdcall accept(SOCKET, struct sockaddr*, int*);
extern int __stdcall getaddrinfo(const char*, const char*, const struct addrinfo*, struct addrinfo**);
extern void __stdcall freeaddrinfo(struct addrinfo*);
extern int __stdcall ioctlsocket(SOCKET, long, unsigned long*);
extern int __stdcall setsockopt(SOCKET, int, int, const char*, int);
extern SOCKET __stdcall socket(int, int, int);
extern int __stdcall listen(SOCKET, int);
int32_t vox_impl_sock_connect(int32_t fd, void* addr, uint32_t len) {
  return connect((SOCKET)(intptr_t)fd, (const struct sockaddr*)addr, (int)len);
}
int32_t vox_impl_sock_bind(int32_t fd, void* addr, uint32_t len) {
  return bind((SOCKET)(intptr_t)fd, (const struct sockaddr*)addr, (int)len);
}
int32_t vox_impl_sock_accept(int32_t fd, void* addr, void* len_ptr) {
  return (int32_t)accept((SOCKET)(intptr_t)fd, (struct sockaddr*)addr, (int*)len_ptr);
}
int32_t vox_impl_getaddrinfo(const char* node, const char* service, void* hints, void* res_out) {
  return getaddrinfo(node, service, (const struct addrinfo*)hints, (struct addrinfo**)res_out);
}
void vox_impl_freeaddrinfo(void* res) {
  freeaddrinfo((struct addrinfo*)res);
}
int32_t vox_impl_fcntl3(int32_t fd, int32_t cmd, int32_t arg) {
  unsigned long mode = (unsigned long)arg;
  return ioctlsocket((SOCKET)(intptr_t)fd, (long)cmd, &mode);
}
int32_t vox_impl_setsockopt(int32_t fd, int32_t level, int32_t name, void* val, uint32_t len) {
  return setsockopt((SOCKET)(intptr_t)fd, level, name, (const char*)val, (int)len);
}
#endif

#if defined(_WIN32)
intptr_t vox_impl_create_iocp(intptr_t file, intptr_t existing, uintptr_t key, uint32_t threads) {
  return (intptr_t)CreateIoCompletionPort((HANDLE)file, (HANDLE)existing, (ULONG_PTR)key, (DWORD)threads);
}
int32_t vox_impl_post_iocp(intptr_t iocp, uint32_t bytes, uintptr_t key, void* overlapped) {
  return PostQueuedCompletionStatus((HANDLE)iocp, (DWORD)bytes, (ULONG_PTR)key, (LPOVERLAPPED)overlapped) != 0 ? 1 : 0;
}
int32_t vox_impl_get_iocp(intptr_t iocp, void* bytes, void* key, void* overlapped, uint32_t timeout) {
  return GetQueuedCompletionStatus((HANDLE)iocp, (LPDWORD)bytes, (PULONG_PTR)key, (LPOVERLAPPED*)overlapped, (DWORD)timeout) != 0 ? 1 : 0;
}
int32_t vox_impl_close_handle(intptr_t h) {
  return CloseHandle((HANDLE)h) != 0 ? 1 : 0;
}
bool vox_impl_iocp_wait_ms(intptr_t iocp, int32_t timeout_ms) {
  DWORD bytes = 0;
  ULONG_PTR key = 0;
  OVERLAPPED* ov = NULL;
  return GetQueuedCompletionStatus((HANDLE)iocp, &bytes, &key, &ov, (DWORD)timeout_ms) != 0;
}
int32_t vox_impl_win_socket(int32_t domain, int32_t ty, int32_t proto) {
  return (int32_t)socket(domain, ty, proto);
}
int32_t vox_impl_win_listen(int32_t fd, int32_t backlog) {
  return listen((SOCKET)fd, backlog);
}
#endif

// Windows-only thin wrappers for Winsock functions that need SOCKET type
// casting or __stdcall calling convention.
#if defined(_WIN32)
typedef struct { unsigned short wVersion; unsigned short wHighVersion; char szDescription[257]; char szSystemStatus[129]; unsigned short iMaxSockets; unsigned short iMaxUdpDg; char* lpVendorInfo; } VOX_WSADATA;
extern int __stdcall WSAStartup(unsigned short, VOX_WSADATA*);
extern int __stdcall recv(SOCKET, char*, int, int);
extern int __stdcall closesocket(SOCKET);
extern int __stdcall select(int, void*, void*, void*, void*);

void vox_impl_wsa_init(void) {
  static int inited = 0;
  if (inited) return;
  VOX_WSADATA wsa;
  if (WSAStartup(0x0202, &wsa) != 0) { vox_host_panic("wsa startup failed"); }
  inited = 1;
}

int32_t vox_impl_win_recv(int32_t fd, void* buf, int32_t max_n, int32_t flags) {
  return recv((SOCKET)(intptr_t)fd, (char*)buf, max_n, flags);
}

int32_t vox_impl_win_closesocket(int32_t fd) {
  return closesocket((SOCKET)(intptr_t)fd);
}

bool vox_impl_win_sock_poll(int32_t fd, bool want_write, int32_t timeout_ms) {
  if (timeout_ms < 0) timeout_ms = 0;
  char fds[520];
  memset(fds, 0, 520);
  *(uint32_t*)fds = 1;
  *(SOCKET*)(fds + 8) = (SOCKET)(intptr_t)fd;
  int32_t tv[2];
  tv[0] = timeout_ms / 1000;
  tv[1] = (timeout_ms % 1000) * 1000;
  int n = select(0, want_write ? NULL : fds, want_write ? fds : NULL, NULL, tv);
  return n > 0;
}
#endif

