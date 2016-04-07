package smartqq

import (
	"compress/gzip"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	IsKeepCookie bool
	Timeout      int
	Cookies      map[string]string
	Header       map[string]string
}

type Response struct {
	StatusCode int
	Body       string
	Header     *http.Header
	Cookies    []*http.Cookie
}

func (client *Client) request(method string, url string, data string) (response Response, err error) {
	//初始化client
	if client.Cookies == nil {
		client.Cookies = map[string]string{}
	}
	if client.Header == nil {
		client.Header = map[string]string{}
	}
	if client.Timeout == 0 {
		client.Timeout = 30
	}
	//初始化http.Client
	var DefaultTransport http.RoundTripper = &http.Transport{
		Dial: func(netw, addr string) (net.Conn, error) {
			conn, err := net.DialTimeout(netw, addr, time.Duration(client.Timeout)*time.Second)
			if err != nil {
				return nil, err
			}
			conn.SetDeadline(time.Now().Add(time.Duration(client.Timeout) * time.Second))
			return conn, nil
		},
		ResponseHeaderTimeout: time.Duration(client.Timeout) * time.Second,
	}

	request_get, err := http.NewRequest(method, url, strings.NewReader(data))
	if err != nil {
		return response, err
	}

	//设置请求的Header
	for k, _ := range client.Header {
		request_get.Header.Set(k, client.Header[k])
	}
	//解析Cookie map数据为字符串
	if cookiestr, ok := client.Header["Cookie"]; ok {
		cookiestr = strings.Replace(cookiestr, " ", "", 10)
		cookiestr = strings.Replace(cookiestr, "\t", "", 10)
		cookiestr = strings.Replace(cookiestr, "\n", "", 10)
		cookie_item := strings.Split(cookiestr, ";")
		for k, _ := range cookie_item {
			cookie_item_sp := strings.Split(cookie_item[k], "=")
			if len(cookie_item_sp) == 2 {
				client.Cookies[cookie_item_sp[0]] = cookie_item_sp[1]
			}
		}
	}
	cookie_str := ""
	for k, _ := range client.Cookies {
		cookie_str += k + "=" + client.Cookies[k] + "; "
	}
	//设置请求的Cookie
	request_get.Header.Set("Cookie", cookie_str)

	//防止因为没有Content-Type，而导致提交POST数据失败
	if method == "POST" {
		if content_type, ok := client.Header["Content-Type"]; !ok || content_type == "" {
			request_get.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		}
	}

	//发送请求
	response_get, err := DefaultTransport.RoundTrip(request_get)

	//检查请求错误
	if err != nil {
		return response, errors.New("Get Error," + err.Error())
	}

	response.StatusCode = response_get.StatusCode

	//释放response.body对象，防止内存泄露
	defer response_get.Body.Close()

	//如果IsKeepCookie为true则保存Cookie状态
	if client.IsKeepCookie == true {
		response_cookie := response_get.Cookies()
		for k, _ := range response_cookie {
			if response_cookie[k].Value != "" {
				client.Cookies[response_cookie[k].Name] = response_cookie[k].Value
			}
		}
	}

	var body_bin []byte
	switch response_get.Header.Get("Content-Encoding") {
	case "gzip":
		reader, _ := gzip.NewReader(response_get.Body)
		body_bin, _ = ioutil.ReadAll(reader)
	default:
		body_bin, _ = ioutil.ReadAll(response_get.Body)
	}

	//返回数据
	response.Body = string(body_bin)
	response.Header = &response_get.Header
	response.Cookies = response_get.Cookies()

	return response, nil
}

func (client *Client) Get(url string) (Response, error) {
	return client.request("GET", url, "")
}

func (client *Client) Post(url string, data string) (Response, error) {
	return client.request("POST", url, data)
}
