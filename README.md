# SmartQQ
SmartQQ Robot By Golang.
##简介
1.基于 Smart QQ（Web QQ） 的 Api 封装，你可以用这个 Api 制作属于自己的 QQ 机器人！<be/>
2.提供的接口包括接收QQ好友、QQ群、QQ讨论组消息，并可以主动或被动的发送消息给好友、群、讨论组。<br/>
3.用数组或其他方式保存QClient对象，可以实现批量QQ登录及收发消息。
4.更多有趣的用途请自行脑洞吧。
##依赖
1.因为SmartQQ登录验证及收发消息接口较为复杂，所以使用的是自己封装的Http-Client包（已经集成在本包中），稍后会开源出来。<br/>
2.另外因为qq返回的json有点小复杂，所以使用了bitly的SimpleJson包来解析QQ返回的json字符串，请手动go get github.com/bitly/go-simplejson
##进度
1.当前版本只提供了登录验证、收发消息的接口。<br/>
2.未来会逐步实现查询QQ好友列表、聊天记录等接口。
##不足
1.SmartQQ不支持收发图片、语音、视屏、及附件。<br/>
2.SmartQQ接口不够稳定，有时候发送消息会返回失败但实际上是发送成功了的，有时候会发送失败但是返回成功...万恶的TX<br/>
3.截止2016年4月7日接口可用。
##使用方法
先get source
```cmd
go get github.com/bitly/go-simplejson
go get github.com/JamesWone/SmartQQ
```
然后，直接见Demo吧!
##Demo
```go
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/JamesWone/SmartQQ"
)

//使用自己封装的Http-Client包
var client_turing smartqq.Client = smartqq.Client{
	IsKeepCookie: true,
	Timeout:      5,
}

//调用图灵机器人Api
func getResponseByTuringRobot(request string) string {
	resp_turing, err := client_turing.Post("http://www.niurenqushi.com/app/simsimi/ajax.aspx", "txt="+request)
	if err != nil {
		return ""
	}
	return resp_turing.Body
}

func main() {
	//初始化一个QClient
	client := smartqq.QClient{}
	//当二维码图片变动后触发
	client.OnQRChange(func(qc *smartqq.QClient, image_bin []byte) {
		//将二维码保存至当前目录，打开手机QQ扫描二维码后即可登录成功
		fmt.Println("正在保存二维码图片.")
		file_image, err := os.OpenFile("v.jpg", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file_image.Close()
		if _, err := file_image.Write(image_bin); err != nil {
			fmt.Println(err)
			return
		}
	})
	//当登录成功后触发
	client.OnLogined(func(qc *smartqq.QClient) {
		fmt.Println("登录成功了！")
	})
	//当收到消息后触发
	client.OnMessage(func(qc *smartqq.QClient, qm smartqq.QMessage) {
		fmt.Println("收到新消息了：")
		fmt.Println(qm)
		content := qm.Content
		if strings.Contains(qm.Content, "@ai") {
			content = strings.Replace(qm.Content, "@ai", "", 1)
			switch qm.Poll_type {
			//QQ好友消息
			case "message":
				//发送给QQ好友
				qc.SendToQQ(qm.From_uin, getResponseByTuringRobot(content)+"\n(by:ai)")
			//QQ群消息
			case "group_message":
				//发送给QQ群
				qc.SendToGroup(qm.From_uin, getResponseByTuringRobot(content)+"\n(by:ai)")
			//讨论组消息
			case "discu_message":
				//发送给讨论组
				qc.SendToDiscuss(qm.From_uin, getResponseByTuringRobot(content)+"\n(by:ai)")
			}
		}
	})
	fmt.Println("开始登录.")
	//开始登录，并自动收发消息
	client.Run()
}
```
