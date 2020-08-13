package task

import (
	"context"
	xxl "github.com/xxl-job/go-client"
)

func Panic(cxt context.Context, param *xxl.RunReq) {
	panic("test panic")
}
