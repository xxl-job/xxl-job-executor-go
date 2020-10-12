package task

import (
	"context"
	xxl "github.com/xxl-job/xxl-job-executor-go"
	"log"
	"time"
)

func Test2(cxt context.Context, param *xxl.RunReq) (msg string) {
	num := 1
	for {

		select {
		case <-cxt.Done():
			log.Println("task" + param.ExecutorHandler + "被手动终止")
			return
		default:
			num++
			time.Sleep(10 * time.Second)
			log.Println("test one task"+param.ExecutorHandler+" param："+param.ExecutorParams+"执行行", num)
			if num > 10 {
				log.Println("test one task" + param.ExecutorHandler + " param：" + param.ExecutorParams + "执行完毕！")
				return
			}
		}
	}

}
