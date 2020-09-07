package task

import (
	"context"
	"fmt"
	xxl "github.com/xxl-job/go-client"
)

func Test(cxt context.Context, param *xxl.RunReq) (msg string) {
	fmt.Println("test one task" + param.ExecutorHandler + " param：" + param.ExecutorParams + "log_id" + xxl.Int64ToStr(param.LogID))
	return "test done"
}
