package task

import (
	"context"
	"fmt"
	xxl "github.com/xxl-job/go-client"
)

func Test(cxt context.Context, param *xxl.RunReq) {
	fmt.Println("test one task" + param.ExecutorHandler + " param：" + param.ExecutorParams)
}
