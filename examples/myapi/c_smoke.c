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

