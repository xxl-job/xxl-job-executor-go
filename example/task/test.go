package task

import (
	"context"
	xxl "github.com/xxl-job/xxl-job-executor-go"
	"time"
)

func Test(cxt context.Context, param *xxl.RunReq) (msg string) {
	logger := param.GetFileLogger()
	logger.Logf("test one task" + param.ExecutorHandler + " param：" + param.ExecutorParams + " log_id:" + xxl.Int64ToStr(param.LogID))
	time.Sleep(60 * time.Second)
	logger.Logf("测试日志 %s", "xxx")
	return "test done"
}
