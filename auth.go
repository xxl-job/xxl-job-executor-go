package xxl

import (
	"net/http"
	"net/url"
	"sync"
)

type Auth interface {
	Login(addr string) ([]*http.Cookie, error)
}

type AuthImpl struct {
	UserName string
	Password string

	mu      sync.Mutex
	cookies []*http.Cookie
}

func NewAuth(userName string, password string) *AuthImpl {
	return &AuthImpl{UserName: userName, Password: password}
}

const (
	loginUrl = "/login"
)

// Login 登录
func (a *AuthImpl) Login(addr string) ([]*http.Cookie, error) {
	if len(a.cookies) == 0 {

		a.mu.Lock()
		defer a.mu.Unlock()

		if len(a.cookies) > 0 {
			return a.cookies, nil
		}

		values := url.Values{}
		values.Add("userName", a.UserName)
		values.Add("password", a.Password)

		resp, err := http.PostForm(addr+loginUrl, values)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()
		a.cookies = resp.Cookies()
	}

	return a.cookies, nil
}
