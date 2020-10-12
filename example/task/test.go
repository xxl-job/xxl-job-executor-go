package task

import (
	"context"
	xxl "github.com/xxl-job/xxl-job-executor-go"
	"log"
)

func Test(cxt context.Context, param *xxl.RunReq) (msg string) {
	log.Println("test one task" + param.ExecutorHandler + " param：" + param.ExecutorParams + " log_id:" + xxl.Int64ToStr(param.LogID))
	return "test done"
}
