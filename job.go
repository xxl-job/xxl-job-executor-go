package xxl

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type JobImpl struct {
	addr string
	auth Auth
}

func NewJob(addr string, auth Auth) Job {
	return &JobImpl{
		addr: addr,
		auth: auth,
	}
}

type Job interface {
	// Add 添加任务
	Add(ctx context.Context, req *AddJob) (int64, error)
	// QueryJobList 查询JobInfo
	QueryJobList(ctx context.Context, req *QueryJob) (*JobList, error)
	// RemoveJob 移除任务
	RemoveJob(ctx context.Context, req *DeleteJob) error
	// RunJob 运行任务
	RunJob(ctx context.Context, jobId int64) error
}

const (
	addJobUrl    = "/jobinfo/add"
	queryJobUrl  = "/jobinfo/pageList"
	removeJobUrl = "/jobinfo/remove"
	runJobUrl    = "/jobinfo/start"
)

var (
	AddJobErr   = errors.New("add job failed")
	QueryJobErr = errors.New("query job failed")
	DelJobErr   = errors.New("delete job failed")
	RunJobErr   = errors.New("run job failed")
)

// Add 添加任务
func (j *JobImpl) Add(ctx context.Context, req *AddJob) (int64, error) {

	values := url.Values{}
	values.Add("jobGroup", Int64ToStr(req.JobGroup))
	values.Add("jobDesc", req.JobDesc)
	values.Add("author", req.Author)
	values.Add("scheduleType", req.ScheduleType)
	values.Add("triggerStatus", Int64ToStr(req.TriggerStatus))
	values.Add("alarmEmail", req.AlarmEmail)
	values.Add("scheduleConf", req.ScheduleConf)
	values.Add("schedule_conf_CRON", req.ScheduleConfCron)
	values.Add("cronGen_display", req.CronGenDisplay)
	values.Add("schedule_conf_FIX_DELAY", req.ScheduleConfFixDelay)
	values.Add("schedule_conf_FIX_RATE", req.ScheduleConfFixRate)
	values.Add("glueType", req.GlueType)
	values.Add("executorHandler", req.ExecutorHandler)
	values.Add("executorParam", req.ExecutorParam)
	values.Add("executorRouteStrategy", req.ExecutorRouteStrategy)
	values.Add("childJobId", req.ChildJobId)
	values.Add("misfireStrategy", req.MisfireStrategy)
	values.Add("executorBlockStrategy", req.ExecutorBlockStrategy)
	values.Add("executorTimeout", Int64ToStr(req.ExecutorTimeout))
	values.Add("executorFailRetryCount", Int64ToStr(req.ExecutorFailRetryCount))
	values.Add("glueRemark", req.GlueRemark)
	values.Add("glueSource", req.GlueSource)

	reader := strings.NewReader(values.Encode())

	client := &http.Client{}

	request, err := http.NewRequest(http.MethodPost, j.addr+addJobUrl, reader)

	if err != nil {
		return 0, err
	}

	cookies, err := j.auth.Login()
	if err != nil {
		return 0, err
	}
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(request)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, AddJobErr
	}

	all, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	response := &AddJobResp{}
	if err = json.Unmarshal(all, response); err != nil {
		return 0, err
	}

	if response.Code != SuccessCode {
		return 0, AddJobErr
	}

	idStr := response.Content
	i, err := StrToInt64(idStr)
	if err != nil {
		return 0, err
	}

	return i, nil
}

// QueryJobList 查询JobInfo
func (j *JobImpl) QueryJobList(ctx context.Context, req *QueryJob) (*JobList, error) {
	values := url.Values{}
	values.Add("jobGroup", Int64ToStr(req.JobGroup))
	values.Add("jobId", Int64ToStr(req.JobId))
	values.Add("triggerStatus", Int64ToStr(int64(req.TriggerStatus)))
	values.Add("start", Int64ToStr(int64(req.Start)))
	values.Add("length", Int64ToStr(int64(req.Length)))

	reader := strings.NewReader(values.Encode())

	client := &http.Client{}

	request, err := http.NewRequest(http.MethodPost, j.addr+queryJobUrl, reader)

	if err != nil {
		return nil, err
	}

	cookies, err := j.auth.Login()
	if err != nil {
		return nil, err
	}
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, QueryJobErr
	}

	readAll, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	jobList := &JobList{}
	if err = json.Unmarshal(readAll, jobList); err != nil {
		return nil, err
	}

	return jobList, nil
}

// RemoveJob 移除任务
func (j *JobImpl) RemoveJob(ctx context.Context, req *DeleteJob) error {
	values := url.Values{}
	values.Add("id", Int64ToStr(req.Id))

	reader := strings.NewReader(values.Encode())

	client := &http.Client{}

	request, err := http.NewRequest(http.MethodPost, j.addr+removeJobUrl, reader)

	if err != nil {
		return err
	}

	cookies, err := j.auth.Login()
	if err != nil {
		return err
	}

	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(request)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return DelJobErr
	}

	all, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	response := &res{}
	if err = json.Unmarshal(all, response); err != nil {
		return err
	}

	if response.Code != SuccessCode {
		return DelJobErr
	}

	return nil
}

func (j *JobImpl) RunJob(ctx context.Context, jobId int64) error {
	values := url.Values{}
	values.Add("id", Int64ToStr(jobId))

	reader := strings.NewReader(values.Encode())

	client := &http.Client{}

	request, err := http.NewRequest(http.MethodPost, j.addr+runJobUrl, reader)

	if err != nil {
		return err
	}

	cookies, err := j.auth.Login()
	if err != nil {
		return err
	}

	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(request)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return RunJobErr
	}

	all, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	response := &res{}
	if err = json.Unmarshal(all, response); err != nil {
		return err
	}

	if response.Code != SuccessCode {
		return RunJobErr
	}

	return nil
}
