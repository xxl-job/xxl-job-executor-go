package xxl

import (
	"fmt"
	"log"
)

//应用日志
type LogFunc func(req LogReq, res *LogRes) []byte

//系统日志
type Logger interface {
	Infof(format string, a ...interface{})
	Errorf(format string, a ...interface{})
	Fatalf(format string, a ...interface{})
}

type logger struct {
}

func (l *logger) Infof(format string, a ...interface{}) {
	fmt.Println(fmt.Sprintf(format, a...))
}

func (l *logger) Errorf(format string, a ...interface{}) {
	log.Println(fmt.Sprintf(format, a...))
}

func (l *logger) Fatalf(format string, a ...interface{}) {
	log.Fatalf(format, a...)
}
