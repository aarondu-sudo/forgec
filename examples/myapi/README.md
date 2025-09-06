# myapi example

Commands:

- go generate ./... — generates exports.go and forgec.h
- go build -buildmode=c-shared -o dist/libmyapi.so . — Linux/macOS
- go build -buildmode=c-shared -o dist/myapi.dll . — Windows

Usage from C (snippet):

```
#include "forgec.h"
#include <stdio.h>
int main(){
    int32_t out = 0;
    int32_t rc = PM_Add(3,4,&out);
    if(rc!=0){
        const char* msg = capi_last_error_json();
        printf("error: %s\n", msg);
        capi_free((void*)msg);
        return 1;
    }
    printf("PM_Add(3,4)=%d\n", (int)out);
    return 0;
}
```

