package xxl

import (
	"encoding/json"
	"strconv"
)

//int64 to str
func Int64ToStr(i int64) string {
	return strconv.FormatInt(i, 10)
}

//str to int64
func StrToInt64(str string) int64 {
	i, _ := strconv.ParseInt(str, 10, 64)
	return i
}

//执行任务回调
func returnCall(req *RunReq, code int64) []byte {
	msg := ""
	if code != 200 {
		msg = "Task not registered"
	}
	data := call{
		&callElement{
			LogID:      req.LogID,
			LogDateTim: req.LogDateTime,
			ExecuteResult: &ExecuteResult{
				Code: code,
				Msg:  msg,
			},
		},
	}
	str, _ := json.Marshal(data)
	return str
}

//杀死任务返回
func returnKill(req *killReq, code int64) []byte {
	msg := ""
	if code != 200 {
		msg = "Task kill err"
	}
	data := res{
		Code: code,
		Msg:  msg,
	}
	str, _ := json.Marshal(data)
	return str
}

//日志返回
func returnLog(req *logReq, code int64) []byte {
	msg := "nil"
	if code != 200 {
		msg = "log err"
	}
	data := &logRes{Code: code, Msg: msg, Content: logResContent{
		FromLineNum: req.FromLineNum,
		ToLineNum:   0,
		LogContent:  "Please view the log server",
		//IsEnd:       line < 5,
		IsEnd: true,
	}}
	str, _ := json.Marshal(data)
	return str
}

//通用返回
func returnGeneral() []byte {
	data := &res{
		Code:200,
		Msg:"",
	}
	str, _ := json.Marshal(data)
	return str
}