// Vox stage1 C preamble — base type headers and standard library declarations.
// This file is auto-discovered as part of std/runtime and must sort first
// (the underscore prefix ensures alphabetical ordering before rt_*.c files).
//
// All subsequent .c files in this module may assume these headers are available.

#include <stdint.h>
#include <stdbool.h>
#include <inttypes.h>
#include <stddef.h>
#include <string.h>
#include <limits.h>
#include <math.h>
#include <stdatomic.h>

extern void* malloc(size_t);
extern void* realloc(void*, size_t);
extern void free(void*);
extern void qsort(void*, size_t, size_t, int(*)(const void*, const void*));
extern void exit(int);
extern int printf(const char*, ...);
extern int snprintf(char*, size_t, const char*, ...);
