package xxl

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Auth interface {
	Login() ([]*http.Cookie, error)
}

type AuthImpl struct {
	Addr     string
	UserName string
	Password string

	mu        sync.Mutex
	closeChan chan struct{}
	cookies   []*http.Cookie
}

func NewAuth(userName, password, addr string, interval time.Duration) (Auth, func(), error) {
	auth := &AuthImpl{
		UserName:  userName,
		Password:  password,
		Addr:      addr,
		closeChan: make(chan struct{}),
	}

	auth.CronReplaceCookie(interval)

	return auth, func() {
		auth.closeChan <- struct{}{}
	}, nil
}

const loginUrl = "/login"

var LoginErr = errors.New("login failed")

// Login 登录
func (a *AuthImpl) Login() ([]*http.Cookie, error) {
	if len(a.cookies) == 0 {

		a.mu.Lock()
		defer a.mu.Unlock()

		if len(a.cookies) > 0 {
			return a.cookies, nil
		}

		cookies, err := a.login()

		if err != nil {
			return nil, err
		}

		a.cookies = cookies
	}

	return a.cookies, nil
}

func (a *AuthImpl) CronReplaceCookie(interval time.Duration) {

	go func(t time.Duration) {
		ticker := time.NewTicker(t)

	ReplaceXxlJonCookieLoop:
		for {
			select {
			case <-ticker.C:
				cookies, err := a.login()
				if err != nil {
					continue
				}

				a.cookies = cookies
			case <-a.closeChan:
				break ReplaceXxlJonCookieLoop
			}
		}
	}(interval)
}

func (a *AuthImpl) login() ([]*http.Cookie, error) {
	values := url.Values{}
	values.Add("userName", a.UserName)
	values.Add("password", a.Password)

	resp, err := http.PostForm(a.Addr+loginUrl, values)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	all, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response := &res{}
	if err = json.Unmarshal(all, response); err != nil {
		return nil, err
	}

	if response.Code == FailureCode {
		return nil, LoginErr
	}

	cookies := resp.Cookies()

	return cookies, nil
}
