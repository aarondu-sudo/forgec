#pragma once

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

int32_t PM_Add(int32_t a, int32_t b, int32_t* out);
int32_t PM_Minus(int32_t a, int32_t b, int32_t* out);

const char* capi_last_error_json(void);
void capi_free(void* p);

#ifdef __cplusplus
}
#endif
