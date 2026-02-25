static int vox__argc = 0;
static char** vox__argv = NULL;

extern char* getenv(const char*);
const char* vox_impl_getenv(const char* key) {
  if (!key) key = "";
  const char* v = getenv(key);
  if (!v) return "";
  return v;
}

vox_vec vox_impl_args(void) {
  vox_vec v = vox_vec_new((int32_t)sizeof(const char*));
  for (int i = 1; i < vox__argc; i++) {
    const char* s = vox__argv[i];
    vox_vec_push(&v, &s);
  }
  return v;
}

const char* vox_impl_exe_path(void) {
  if (!vox__argv || vox__argc <= 0 || !vox__argv[0]) return "";
  return vox__argv[0];
}

void* vox_impl_alloc_buf(int32_t size) {
  if (size < 0) size = 0;
  char* p = (char*)vox_impl_malloc((size_t)size + 1);
  if (!p) { vox_host_panic("out of memory"); }
  p[size] = '\0';
  return (void*)p;
}


// Platform I/O wrappers (avoid C type conflicts with system headers).
// Direct @ffi_import to kevent/epoll_ctl/epoll_wait generates void* externs
// that conflict with struct pointer params in system headers included
// transitively by c_rt_core (e.g. <sys/event.h> on macOS, <sys/epoll.h> on Linux).
#if defined(__APPLE__) || defined(__FreeBSD__) || defined(__OpenBSD__) || defined(__NetBSD__)
int32_t vox_impl_kevent(int32_t kq, void* changelist, int32_t nchanges, void* eventlist, int32_t nevents, void* timeout) {
  return kevent(kq, changelist, nchanges, eventlist, nevents, timeout);
}
#endif
#if defined(__linux__)
int32_t vox_impl_epoll_ctl(int32_t epfd, int32_t op, int32_t fd, void* event) {
  return epoll_ctl(epfd, op, fd, event);
}
int32_t vox_impl_epoll_wait(int32_t epfd, void* events, int32_t maxevents, int32_t timeout) {
  return epoll_wait(epfd, events, maxevents, timeout);
}
#endif

// === Event loop: Vox-driven wake table + platform poller ===
// Forward declarations for atomic functions used by el_init.
intptr_t vox_impl_atomic_i64_new(int64_t init);
intptr_t vox_impl_atomic_i32_new(int32_t init);
#define VOX_EL_SLOTS 256
static intptr_t vox_el_token_h[VOX_EL_SLOTS];
static intptr_t vox_el_pending_h[VOX_EL_SLOTS];
static bool vox_el_inited = false;

#if defined(__linux__)
static int vox_el_epfd = -1;
static int vox_el_efd = -1;
#elif defined(__APPLE__) || defined(__FreeBSD__) || defined(__OpenBSD__) || defined(__NetBSD__)
static int vox_el_kq = -1;
#elif defined(_WIN32)
static HANDLE vox_el_iocp = NULL;
#endif

static void vox_el_init_poller(void) {
#if defined(__linux__)
  if (vox_el_epfd >= 0) return;
  vox_el_efd = eventfd(0, VOX_EFD_NONBLOCK | VOX_EFD_CLOEXEC);
  if (vox_el_efd < 0) { vox_host_panic("el: eventfd failed"); }
  vox_el_epfd = epoll_create1(VOX_EPOLL_CLOEXEC);
  if (vox_el_epfd < 0) { vox_host_panic("el: epoll_create1 failed"); }
  char ev[12]; memset(ev, 0, 12);
  *(uint32_t*)ev = VOX_EPOLLIN;
  *(uint64_t*)(ev + 4) = (uint64_t)(uint32_t)vox_el_efd;
  if (epoll_ctl(vox_el_epfd, VOX_EPOLL_CTL_ADD, vox_el_efd, ev) != 0) {
    vox_host_panic("el: epoll_ctl failed");
  }
#elif defined(__APPLE__) || defined(__FreeBSD__) || defined(__OpenBSD__) || defined(__NetBSD__)
  if (vox_el_kq >= 0) return;
  vox_el_kq = kqueue();
  if (vox_el_kq < 0) { vox_host_panic("el: kqueue failed"); }
  char ev[VOX_KEVENT_SZ];
  vox_kev_set(ev, 1, VOX_EVFILT_USER, VOX_EV_ADD | VOX_EV_CLEAR, 0, 0, NULL);
  if (kevent(vox_el_kq, ev, 1, NULL, 0, NULL) != 0) {
    vox_host_panic("el: kqueue register failed");
  }
#elif defined(_WIN32)
  if (vox_el_iocp) return;
  vox_el_iocp = CreateIoCompletionPort(INVALID_HANDLE_VALUE, NULL, 0, 0);
  if (!vox_el_iocp) { vox_host_panic("el: IOCP failed"); }
#endif
}

void vox_impl_el_init(void) {
  if (vox_el_inited) return;
  for (int i = 0; i < VOX_EL_SLOTS; i++) {
    vox_el_token_h[i] = vox_impl_atomic_i64_new(0);
    vox_el_pending_h[i] = vox_impl_atomic_i32_new(0);
  }
  vox_el_init_poller();
  vox_el_inited = true;
}

int32_t vox_impl_el_n_slots(void) { return VOX_EL_SLOTS; }
intptr_t vox_impl_el_token_handle(int32_t i) { return vox_el_token_h[i]; }
intptr_t vox_impl_el_pending_handle(int32_t i) { return vox_el_pending_h[i]; }

void vox_impl_el_poller_wake(void) {
#if defined(__linux__)
  uint64_t one = 1;
  ssize_t n = write(vox_el_efd, &one, sizeof(one));
  (void)n;
#elif defined(__APPLE__) || defined(__FreeBSD__) || defined(__OpenBSD__) || defined(__NetBSD__)
  char ev[VOX_KEVENT_SZ];
  vox_kev_set(ev, 1, VOX_EVFILT_USER, 0, VOX_NOTE_TRIGGER, 0, NULL);
  kevent(vox_el_kq, ev, 1, NULL, 0, NULL);
#elif defined(_WIN32)
  if (vox_el_iocp) PostQueuedCompletionStatus(vox_el_iocp, 1, (ULONG_PTR)1, NULL);
#endif
}

void vox_impl_el_poller_wait(int32_t timeout_ms) {
  if (timeout_ms < 0) timeout_ms = 0;
#if defined(__linux__)
  char ev[12];
  int n = epoll_wait(vox_el_epfd, ev, 1, timeout_ms);
  if (n > 0) { uint64_t v = 0; while (read(vox_el_efd, &v, sizeof(v)) > 0) {} }
#elif defined(__APPLE__) || defined(__FreeBSD__) || defined(__OpenBSD__) || defined(__NetBSD__)
  struct timespec ts;
  ts.tv_sec = timeout_ms / 1000;
  ts.tv_nsec = (long)(timeout_ms % 1000) * 1000000L;
  char ev[VOX_KEVENT_SZ];
  kevent(vox_el_kq, NULL, 0, ev, 1, &ts);
#elif defined(_WIN32)
  DWORD bytes = 0; ULONG_PTR key = 0; OVERLAPPED* ov = NULL;
  GetQueuedCompletionStatus(vox_el_iocp, &bytes, &key, &ov, (DWORD)timeout_ms);
#elif defined(__EMSCRIPTEN__)
  sched_yield();
#else
  struct timespec ts;
  ts.tv_sec = timeout_ms / 1000;
  ts.tv_nsec = (long)(timeout_ms % 1000) * 1000000L;
  nanosleep(&ts, NULL);
#endif
}

int64_t vox_impl_now_ns(void) {
#if defined(_WIN32)
  return (int64_t)GetTickCount64() * (int64_t)1000000;
#elif defined(__EMSCRIPTEN__)
  clock_t c = clock();
  if (c == (clock_t)-1) return 0;
  return (int64_t)c * (int64_t)1000000000 / (int64_t)CLOCKS_PER_SEC;
#else
  struct timespec ts;
  if (clock_gettime(CLOCK_MONOTONIC, &ts) != 0) {
    if (timespec_get(&ts, TIME_UTC) == 0) return 0;
  }
  return (int64_t)ts.tv_sec * (int64_t)1000000000 + (int64_t)ts.tv_nsec;
#endif
}


#ifndef _WIN32
struct sockaddr;
struct addrinfo;
extern int socket(int, int, int);
extern int connect(int, const struct sockaddr*, unsigned int);
extern int bind(int, const struct sockaddr*, unsigned int);
extern int accept(int, struct sockaddr*, unsigned int*);
extern int getaddrinfo(const char*, const char*, const struct addrinfo*, struct addrinfo**);
extern void freeaddrinfo(struct addrinfo*);
extern int setsockopt(int, int, int, const void*, unsigned int);
extern int fcntl(int, int, ...);
int32_t vox_impl_sock_connect(int32_t fd, void* addr, uint32_t len) {
  return connect(fd, (const struct sockaddr*)addr, len);
}
int32_t vox_impl_sock_bind(int32_t fd, void* addr, uint32_t len) {
  return bind(fd, (const struct sockaddr*)addr, len);
}
int32_t vox_impl_sock_accept(int32_t fd, void* addr, void* len_ptr) {
  return accept(fd, (struct sockaddr*)addr, (unsigned int*)len_ptr);
}
int32_t vox_impl_getaddrinfo(const char* node, const char* service, void* hints, void* res_out) {
  return getaddrinfo(node, service, (const struct addrinfo*)hints, (struct addrinfo**)res_out);
}
void vox_impl_freeaddrinfo(void* res) {
  freeaddrinfo((struct addrinfo*)res);
}
int32_t vox_impl_fcntl3(int32_t fd, int32_t cmd, int32_t arg) {
  return fcntl(fd, cmd, arg);
}
int32_t vox_impl_setsockopt(int32_t fd, int32_t level, int32_t name, void* val, uint32_t len) {
  return setsockopt(fd, level, name, (const void*)val, len);
}
#endif
#ifdef _WIN32
typedef uintptr_t SOCKET;
struct sockaddr;
struct addrinfo;
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

#if defined(_WIN32)
struct addrinfo { int ai_flags; int ai_family; int ai_socktype; int ai_protocol; size_t ai_addrlen; char* ai_canonname; struct sockaddr* ai_addr; struct addrinfo* ai_next; };
typedef struct { unsigned short wVersion; unsigned short wHighVersion; char szDescription[257]; char szSystemStatus[129]; unsigned short iMaxSockets; unsigned short iMaxUdpDg; char* lpVendorInfo; } VOX_WSADATA;
extern int __stdcall WSAStartup(unsigned short, VOX_WSADATA*);
extern int __stdcall recv(SOCKET, char*, int, int);
extern int __stdcall closesocket(SOCKET);
extern int __stdcall select(int, void*, void*, void*, void*);
static int vox_tcp_win_inited = 0;
static void vox_tcp_win_ensure_init(void) {
  if (vox_tcp_win_inited) return;
  VOX_WSADATA wsa;
  int rc = WSAStartup(0x0202, &wsa);
  if (rc != 0) { vox_host_panic("tcp wsa startup failed"); }
  vox_tcp_win_inited = 1;
}

intptr_t vox_impl_tcp_connect(const char* host, int32_t port) {
  vox_tcp_win_ensure_init();
  if (!host) host = "";
  if (port <= 0 || port > 65535) { vox_host_panic("invalid tcp port"); }
  char port_buf[16];
  snprintf(port_buf, sizeof(port_buf), "%" PRId32, port);
  struct addrinfo hints;
  memset(&hints, 0, sizeof(hints));
  hints.ai_family = 0;
  hints.ai_socktype = 1;
  hints.ai_protocol = 6;
  struct addrinfo* res = NULL;
  int rc = getaddrinfo(host, port_buf, &hints, &res);
  if (rc != 0 || !res) { vox_host_panic("tcp connect resolve failed"); }
  SOCKET fd = (SOCKET)(~(uintptr_t)0);
  for (struct addrinfo* p = res; p != NULL; p = p->ai_next) {
    fd = socket(p->ai_family, p->ai_socktype, p->ai_protocol);
    if (fd == (SOCKET)(~(uintptr_t)0)) continue;
    if (connect(fd, p->ai_addr, (int)p->ai_addrlen) == 0) { break; }
    closesocket(fd);
    fd = (SOCKET)(~(uintptr_t)0);
  }
  freeaddrinfo(res);
  if (fd == (SOCKET)(~(uintptr_t)0)) { vox_host_panic("tcp connect failed"); }
  return (intptr_t)(uintptr_t)fd;
}

const char* vox_impl_tcp_recv(intptr_t h, int32_t max_n) {
  if (h < 0) { vox_host_panic("invalid tcp handle"); }
  SOCKET fd = (SOCKET)(uintptr_t)h;
  if (max_n <= 0) return "";
  char* buf = (char*)vox_impl_malloc((size_t)max_n + 1);
  if (!buf) { vox_host_panic("out of memory"); }
  int n = recv(fd, buf, max_n, 0);
  if (n < 0) { vox_host_panic("tcp recv failed"); }
  buf[n] = '\0';
  return buf;
}

void vox_impl_tcp_close(intptr_t h) {
  if (h < 0) return;
  closesocket((SOCKET)(uintptr_t)h);
}

static bool vox_tcp_wait_select(SOCKET fd, bool want_write, int32_t timeout_ms) {
  if (timeout_ms < 0) timeout_ms = 0;
  char fds[520];
  memset(fds, 0, 520);
  *(uint32_t*)fds = 1;
  *(SOCKET*)(fds + 8) = fd;
  int32_t tv[2];
  tv[0] = timeout_ms / 1000;
  tv[1] = (timeout_ms % 1000) * 1000;
  int n = select(0, want_write ? NULL : fds, want_write ? fds : NULL, NULL, tv);
  return n > 0;
}

bool vox_impl_tcp_wait_read(intptr_t h, int32_t timeout_ms) {
  if (h < 0) { vox_host_panic("invalid tcp handle"); }
  return vox_tcp_wait_select((SOCKET)(uintptr_t)h, false, timeout_ms);
}

bool vox_impl_tcp_wait_write(intptr_t h, int32_t timeout_ms) {
  if (h < 0) { vox_host_panic("invalid tcp handle"); }
  return vox_tcp_wait_select((SOCKET)(uintptr_t)h, true, timeout_ms);
}
#else
#if defined(__APPLE__)
struct addrinfo { int ai_flags; int ai_family; int ai_socktype; int ai_protocol; unsigned int ai_addrlen; char* ai_canonname; struct sockaddr* ai_addr; struct addrinfo* ai_next; };
#else
struct addrinfo { int ai_flags; int ai_family; int ai_socktype; int ai_protocol; unsigned int ai_addrlen; struct sockaddr* ai_addr; char* ai_canonname; struct addrinfo* ai_next; };
#endif
extern ssize_t recv(int, void*, size_t, int);
static bool vox_tcp_wait_unix_fd(int fd, bool want_write, int32_t timeout_ms) {
  if (fd < 0) { vox_host_panic("invalid tcp handle"); }
  if (timeout_ms < 0) timeout_ms = 0;
#if defined(__linux__)
  int ep = epoll_create1(VOX_EPOLL_CLOEXEC);
  if (ep < 0) { vox_host_panic("tcp epoll create failed"); }
  char ev[12];
  memset(ev, 0, 12);
  *(uint32_t*)ev = want_write ? (VOX_EPOLLOUT | 0x8 | 0x10) : (VOX_EPOLLIN | 0x8 | 0x10);
  *(uint64_t*)(ev + 4) = 1;
  if (epoll_ctl(ep, VOX_EPOLL_CTL_ADD, fd, ev) != 0) { close(ep); vox_host_panic("tcp epoll ctl failed"); }
  int n = epoll_wait(ep, ev, 1, timeout_ms);
  close(ep);
  return n > 0;
#elif defined(__APPLE__) || defined(__FreeBSD__) || defined(__OpenBSD__) || defined(__NetBSD__)
  int kq = kqueue();
  if (kq < 0) { vox_host_panic("tcp kqueue create failed"); }
  char ch[VOX_KEVENT_SZ];
  int16_t filt = want_write ? VOX_EVFILT_WRITE : VOX_EVFILT_READ;
  vox_kev_set(ch, (uintptr_t)fd, filt, VOX_EV_ADD | VOX_EV_ENABLE | VOX_EV_ONESHOT, 0, 0, NULL);
  if (kevent(kq, ch, 1, NULL, 0, NULL) != 0) { close(kq); vox_host_panic("tcp kqueue register failed"); }
  struct timespec ts;
  ts.tv_sec = timeout_ms / 1000;
  ts.tv_nsec = (long)(timeout_ms % 1000) * 1000000L;
  char ev[VOX_KEVENT_SZ];
  int n = kevent(kq, NULL, 0, ev, 1, &ts);
  close(kq);
  return n > 0;
#else
  (void)fd; (void)want_write; (void)timeout_ms;
  sched_yield();
  return true;
#endif
}

bool vox_impl_tcp_wait_read(intptr_t h, int32_t timeout_ms) {
  return vox_tcp_wait_unix_fd((int)h, false, timeout_ms);
}

bool vox_impl_tcp_wait_write(intptr_t h, int32_t timeout_ms) {
  return vox_tcp_wait_unix_fd((int)h, true, timeout_ms);
}

intptr_t vox_impl_tcp_connect(const char* host, int32_t port) {
  if (!host) host = "";
  if (port <= 0 || port > 65535) { vox_host_panic("invalid tcp port"); }
  char port_buf[16];
  snprintf(port_buf, sizeof(port_buf), "%" PRId32, port);
  struct addrinfo hints;
  memset(&hints, 0, sizeof(hints));
  hints.ai_family = 0;
  hints.ai_socktype = 1;
  struct addrinfo* res = NULL;
  int rc = getaddrinfo(host, port_buf, &hints, &res);
  if (rc != 0 || !res) { vox_host_panic("tcp connect resolve failed"); }
  int fd = -1;
  for (struct addrinfo* p = res; p != NULL; p = p->ai_next) {
    fd = socket(p->ai_family, p->ai_socktype, p->ai_protocol);
    if (fd < 0) continue;
    if (connect(fd, p->ai_addr, p->ai_addrlen) == 0) { break; }
    close(fd);
    fd = -1;
  }
  freeaddrinfo(res);
  if (fd < 0) { vox_host_panic("tcp connect failed"); }
  return (intptr_t)fd;
}

const char* vox_impl_tcp_recv(intptr_t h, int32_t max_n) {
  int fd = (int)h;
  if (fd < 0) { vox_host_panic("invalid tcp handle"); }
  if (max_n <= 0) return "";
  char* buf = (char*)vox_impl_malloc((size_t)max_n + 1);
  if (!buf) { vox_host_panic("out of memory"); }
  ssize_t n = recv(fd, buf, (size_t)max_n, 0);
  if (n < 0) { vox_host_panic("tcp recv failed"); }
  buf[n] = '\0';
  return buf;
}

void vox_impl_tcp_close(intptr_t h) {
  int fd = (int)h;
  if (fd < 0) return;
  close(fd);
}
#endif

