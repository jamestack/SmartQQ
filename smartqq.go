package smartqq

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
)

type QClient struct {
	IsLogin      bool
	HttpClient   *Client
	onQRChange   func(*QClient, []byte)
	onLogined    func(*QClient)
	onMessage    func(*QClient, QMessage)
	parameter    map[string]string
	lastSendTime int64
	//	sendLock     sync.Mutex
}

type QMessage struct {
	Poll_type string
	From_uin  int
	Send_uin  int
	To_uin    int
	Msg_id    int
	Content   string
	Retcode   int
	Time      int
}

func (qc *QClient) OnQRChange(fun func(*QClient, []byte)) {
	qc.onQRChange = fun
}

func (qc *QClient) OnLogined(fun func(*QClient)) {
	qc.onLogined = fun
}

func (qc *QClient) OnMessage(fun func(*QClient, QMessage)) {
	qc.onMessage = fun
}

func (qc *QClient) SendToQQ(from_uin int, message string) error {
	qc.sendMsg("to", from_uin, message)
	return nil
}

func (qc *QClient) SendToGroup(from_uin int, message string) error {
	qc.sendMsg("group_uin", from_uin, message)
	return nil
}

func (qc *QClient) SendToDiscuss(from_uin int, message string) error {
	qc.sendMsg("did", from_uin, message)
	return nil
}

func (qc *QClient) pollMessage() {
	ierr := 0
	client := qc.HttpClient
	for {
		if ierr > 5 {
			return
		}
		client.Header["Origin"] = "http://d1.web2.qq.com"
		client.Header["Referer"] = "http://d1.web2.qq.com/proxy.html?v=20151105001&callback=1&id=2"
		resp_poll, err := client.Post("http://d1.web2.qq.com/channel/poll2", `r={"ptwebqq":"`+qc.parameter["ptwebqq"]+`","clientid":53999199,"psessionid":"`+qc.parameter["psessionid"]+`","key":""}`)
		//Timeout与容错处理,如果连续出错超过5次则返回错误
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				ierr = 0
				continue
			} else {
				fmt.Println("Poll Error,", err.Error())
				ierr++
				continue
			}
		}
		recode, err := parseMessage(qc, resp_poll.Body)
		if err != nil {
			if recode == 103 {
				fmt.Println("PollMessage Err,code 103,请登录网页版SmartQQ(http://w.qq.com)然后退出，方可恢复。")
				return
			} else {
				fmt.Println("PollMessage Err,code:", recode, ",err:", err)
			}
		}
	}
}

func parseMessage(qc *QClient, json_str string) (int, error) {
	sj, err := simplejson.NewJson([]byte(json_str))
	if err != nil {
		return -1, err
	}
	retcode, err := sj.Get("retcode").Int()
	if err != nil {
		return -1, err
	}
	if retcode != 0 {
		errmsg, err := sj.Get("errmsg").String()
		if err != nil {
			fmt.Println(err)
			return retcode, err
		}
		//		fmt.Println("Retcode:", retcode, ",errmsg:", errmsg)
		return retcode, errors.New("errmsg:" + errmsg)
	}
	poll_type, err := sj.Get("result").GetIndex(0).Get("poll_type").String()
	if err != nil {
		return -1, err
	}
	sj_value := sj.Get("result").GetIndex(0).Get("value")
	sj_content := sj_value.Get("content")
	sj_content_arr, err := sj_content.Array()
	if err != nil {
		return -1, err
	}
	len_content := len(sj_content_arr)
	content := ""
	for i := 0; i < len_content; i++ {
		if msg_part, err := sj_content.GetIndex(i).String(); err == nil {
			content += msg_part
		}
	}
	from_uin, err := sj_value.Get("from_uin").Int()
	if err != nil {
		return -1, err
	}
	send_uin := 0
	if poll_type == "group_message" || poll_type == "discu_message" {
		send_uin, err = sj_value.Get("send_uin").Int()
	}
	to_uin, err := sj_value.Get("to_uin").Int()
	if err != nil {
		return -1, err
	}
	msg_id, err := sj_value.Get("msg_id").Int()
	if err != nil {
		return -1, err
	}
	time, err := sj_value.Get("time").Int()
	if err != nil {
		return -1, err
	}
	qm := QMessage{
		Poll_type: poll_type,
		From_uin:  from_uin,
		Send_uin:  send_uin,
		To_uin:    to_uin,
		Msg_id:    msg_id,
		Content:   content,
		Time:      time,
	}
	qc.onMessage(qc, qm)
	return retcode, nil
}

//msg_id加密算法
var msg_num int64 = time.Now().Unix() % 1E4 * 1E4

func (qc *QClient) sendMsg(sendType string, toUin int, msg string) {
	msg_num++

	//为goroutine加锁，限制两次发送消息的时间必须大于2秒 ps:后来索性去除goroutine了
	//	qc.sendLock.Lock()
	//	if time.Now().Unix()-qc.lastSendTime < 3 {
	//		time.Sleep(2 * time.Second)
	//	}
	//	qc.lastSendTime = time.Now().Unix()
	//	qc.sendLock.Unlock()

	qc.HttpClient.Header["Origin"] = "http://d1.web2.qq.com"
	qc.HttpClient.Header["Referer"] = "http://d1.web2.qq.com/proxy.html?v=20151105001&callback=1&id=2"
	send_data := `{"` + sendType + `":` + fmt.Sprintf("%d", toUin) + `,"content":"[\"` + msg + `\",[\"font\",{\"name\":\"宋体\",\"size\":10,\"style\":[0,0,0],\"color\":\"000000\"}]]","face":528,"clientid":53999199,"msg_id":` + fmt.Sprint(msg_num) + `,"psessionid":"` + qc.parameter["psessionid"] + `"}`
	send_url := ""
	switch sendType {
	case "to":
		send_url = "http://d1.web2.qq.com/channel/send_buddy_msg2"
	case "group_uin":
		send_url = "http://d1.web2.qq.com/channel/send_qun_msg2"
	case "did":
		send_url = "http://d1.web2.qq.com/channel/send_discu_msg2"
	default:
		return
	}
	resp_send, err := qc.HttpClient.Post(send_url, "r="+send_data)
	//	fmt.Println(send_data)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	result, err := simplejson.NewJson([]byte(resp_send.Body))
	if err == nil {
		if retcode, err := result.Get("retcode").Int(); err == nil && retcode == 100001 {
			qc.sendMsg(sendType, toUin, msg)
		}
	}
	//	fmt.Println(resp_send.Body)
}

func (qc *QClient) Run() {
	client := Client{
		IsKeepCookie: true,
		Header: map[string]string{
			"Host":                      "d1.web2.qq.com",
			"Connection":                "keep-alive",
			"Cache-Control":             "max-age=0",
			"Accept":                    "*/*",
			"Upgrade-Insecure-Requests": "1",
			"User-Agent":                "Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/49.0.2623.87 Safari/537.36",
			"DNT":                       "1",
			"Accept-Encoding":           "gzip, deflate",
			"Accept-Language":           "zh-CN,zh;q=0.8",
		},
		Cookies: map[string]string{
			"pgv_info": "ssid=s3883232818",
			"pgv_id":   "2677229032",
		},
	}

	if qc.parameter == nil {
		qc.parameter = map[string]string{}
	}

	qc.HttpClient = &client

	client.Get("https://ui.ptlogin2.qq.com/cgi-bin/login?daid=164&target=self&style=16&mibao_css=m_webqq&appid=501004106&enable_qlogin=0&no_verifyimg=1&s_url=http%3A%2F%2Fw.qq.com%2Fproxy.html&f_url=loginerroralert&strong_login=1&login_state=10&t=20131024001")
	resp_image, err := client.Get("https://ssl.ptlogin2.qq.com/ptqrshow?appid=501004106&e=0&l=M&s=5&d=72&v=4&t=0.1")
	if err != nil {
		fmt.Println(err)
		return
	}
	//回调：OnQRChange()
	qc.onQRChange(qc, []byte(resp_image.Body))

	regexp_image_state := regexp.MustCompile(`ptuiCB\(\'(\d+)\'`)

validate_login:
	for {
		resp_image_state, err := client.Get("https://ssl.ptlogin2.qq.com/ptqrlogin?webqq_type=10&remember_uin=1&login2qq=1&aid=501004106&u1=http%3A%2F%2Fw.qq.com%2Fproxy.html%3Flogin2qq%3D1%26webqq_type%3D10&ptredirect=0&ptlang=2052&daid=164&from_ui=1&pttype=1&dumy=&fp=loginerroralert&action=0-0-157510&mibao_css=m_webqq&t=1&g=1&js_type=0&js_ver=10143&login_sig=&pt_randsalt=0")
		if err != nil {
			fmt.Println(err)
			return
		}
		switch code := regexp_image_state.FindAllStringSubmatch(resp_image_state.Body, -1)[0][1]; code {
		case "65":
			fmt.Println("二维码已失效.")
			resp_image, err = client.Get("https://ssl.ptlogin2.qq.com/ptqrshow?appid=501004106&e=0&l=M&s=5&d=72&v=4&t=0.1")
			if err != nil {
				fmt.Println(err)
				return
			}
			//回调：OnQRChange()
			qc.onQRChange(qc, []byte(resp_image.Body))
		case "66":
			fmt.Println("二维码未失效.")
		case "67":
			fmt.Println("二维码正在验证..")
		case "0":
			fmt.Println("二维码验证成功！")
			sig_link := ""
			if reg_sig := regexp.MustCompile(`ptuiCB\(\'0\',\'0\',\'([^\']+)\'`).FindAllStringSubmatch(resp_image_state.Body, -1); len(reg_sig) == 1 {
				sig_link = reg_sig[0][1]
			} else {
				fmt.Println("Check Sig Err:")
				return
			}
			resp_check_sig, err := client.Get(sig_link)
			if resp_check_sig.StatusCode != 302 {
				fmt.Println("Get Err:", err.Error())
				return
			}
			break validate_login
		default:
			fmt.Println("未知状态(" + code + ")")
			return
		}
		time.Sleep(time.Second)
	}

	//获取ptwebqq
	client.Get("http://s.web2.qq.com/proxy.html?v=20130916001&callback=1&id=1")
	//获取vfwebqq
	client.Header["Referer"] = "http://s.web2.qq.com/proxy.html?v=20130916001&callback=1&id=1"
	resp_vfwebqq, err := client.Get("http://s.web2.qq.com/api/getvfwebqq?ptwebqq=" + client.Cookies["ptwebqq"] + "&clientid=53999199&psessionid=&t=0.1")
	if err != nil {
		fmt.Println(err)
		return
	}
	sp_vfweb, err := simplejson.NewJson([]byte(resp_vfwebqq.Body))
	if vf_code_int, _ := sp_vfweb.Get("retcode").Int(); vf_code_int != 0 {
		return
	}
	vfwebqq, err := sp_vfweb.Get("result").Get("vfwebqq").String()
	if err != nil {
		return
	}
	qc.parameter["vfwebqq"] = vfwebqq

	//获取psessionid
	client.Header["Referer"] = "http://d1.web2.qq.com/proxy.html?v=20151105001&callback=1&id=2"
	resp_psession, err := client.Post("http://d1.web2.qq.com/channel/login2", `r={"ptwebqq":"`+client.Cookies["ptwebqq"]+`","clientid":53999199,"psessionid":"","status":"online"}`)
	if err != nil {
		fmt.Println(err)
		return
	}
	if psessionid := regexp.MustCompile(`\"psessionid\"\:\"([^\"]+)\"`).FindAllStringSubmatch(resp_psession.Body, -1); len(psessionid) == 1 {
		qc.parameter["psessionid"] = psessionid[0][1]
	} else {
		return
	}

	if uin := regexp.MustCompile(`\"uin\"\:([\d]+),\"`).FindAllStringSubmatch(resp_psession.Body, -1); len(uin) == 1 {
		qc.parameter["uin"] = uin[0][1]
	} else {
		return
	}

	//回调：qc.OnLogined()
	if qc.onLogined != nil {
		qc.onLogined(qc)
	}
	//开始轮训新消息
	if qc.onMessage != nil {
		qc.pollMessage()
	}

}
