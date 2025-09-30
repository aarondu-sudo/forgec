package main

/*
#include <stdlib.h>
#include <stdint.h>
*/
import "C"

import (
	p "example.com/myapi/internal"
	"example.com/myapi/sentrywrap"
	"unsafe"
)

//export capi_free
func capi_free(p unsafe.Pointer) { C.free(p) }

//export capi_last_error_json
func capi_last_error_json() *C.char {
	s := sentrywrap.LastErrorJSON()
	return C.CString(s)
}

//export PM_Add
func PM_Add(a C.int32_t, b C.int32_t, out *C.int32_t) C.int32_t {
	var errno C.int32_t = 0
	sentrywrap.RecoverAndReport(func() {
		res, err := p.Add(int32(a), int32(b))
		if err != nil {
			errno = 1
			sentrywrap.SetLastError(err)
			return
		}
		if out != nil {
			*out = C.int32_t(res)
		}
	})
	return errno
}

//export PM_Minus
func PM_Minus(a C.int32_t, b C.int32_t, out *C.int32_t) C.int32_t {
	var errno C.int32_t = 0
	sentrywrap.RecoverAndReport(func() {
		res, err := p.Minus(int32(a), int32(b))
		if err != nil {
			errno = 1
			sentrywrap.SetLastError(err)
			return
		}
		if out != nil {
			*out = C.int32_t(res)
		}
	})
	return errno
}

//export PM_NewCloudSave
func PM_NewCloudSave(appId C.int64_t) C.int32_t {
	var errno C.int32_t = 0
	sentrywrap.RecoverAndReport(func() {
		err := p.NewCloudSave(int64(appId))
		if err != nil {
			errno = 1
			sentrywrap.SetLastError(err)
			return
		}
	})
	return errno
}

func main() {}
