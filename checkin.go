package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

var (
	refresh_Token         string
	pushplus_token        string
	updateAccesssTokenURL = "https://auth.aliyundrive.com/v2/account/token"
	signinURL             = "https://member.aliyundrive.com/v1/activity/sign_in_list"
)

type aliyundrive struct {
	refreshToken string
	accessToken  string
}

func New(refreshToken string) *aliyundrive {
	return &aliyundrive{refreshToken: refreshToken}
}

func (a *aliyundrive) getAccessToken() {
	body := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refresh_Token,
	}
	b := bytes.NewBuffer(nil)
	json.NewEncoder(b).Encode(body)
	rsp, err := http.Post(updateAccesssTokenURL, "application/json", b)
	if err != nil {
		log.Println(err)
		return
	}
	bytersp, _ := io.ReadAll(rsp.Body)
	a.accessToken = gjson.GetBytes(bytersp, "access_token").String()
	a.refreshToken = gjson.GetBytes(bytersp, "refresh_token").String()
	// log.Printf("%#v\n", string(bytersp))
}

type refreshToken struct {
	GrantType    string `json:"grant_Type,omitempty"`
	RefreshToken string `json:"refresh_Token,omitempty"`
	Phone        string
}

type Signrsp struct {
	Success    bool   `json:"success,omitempty"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
	TotalCount string `json:"totalCount,omitempty"`
	NextToken  string `json:"nextToken,omitempty"`
	MaxResults string `json:"maxResults,omitempty"`
	Result     struct {
		Subject           string       `json:"subject,omitempty"`
		Title             string       `json:"title,omitempty"`
		Description       string       `json:"description,omitempty"`
		IsReward          bool         `json:"isReward,omitempty"`
		Blessing          string       `json:"blessing,omitempty"`
		SignInCount       int          `json:"signInCount,omitempty"`
		SignInCover       string       `json:"signInCover,omitempty"`
		SignInRemindCover string       `json:"signInRemindCover,omitempty"`
		RewardCover       string       `json:"rewardCover,omitempty"`
		SignInLogs        []SignInLogs `json:"signInLogs,omitempty"`
	} `json:"result,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}
type SignInLogs struct {
	Day             int    `json:"day,omitempty"`
	Status          string `json:"status,omitempty"`
	Icon            string `json:"icon,omitempty"`
	Notice          string `json:"notice,omitempty"`
	Type            string `json:"type,omitempty"`
	Themes          string `json:"themes,omitempty"`
	CalendarChinese string `json:"calendarChinese,omitempty"`
	CalendarDay     string `json:"calendarDay,omitempty"`
	CalendarMonth   string `json:"calendarMonth,omitempty"`
	Poster          struct {
		Name       string `json:"name,omitempty"`
		Reason     string `json:"reason,omitempty"`
		Background string `json:"background,omitempty"`
		Color      string `json:"color,omitempty"`
		Action     string `json:"action,omitempty"`
	} `json:"poster,omitempty"`
	Reward struct {
		GoodsID     int    `json:"goodsId,omitempty"`
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
		Background  string `json:"background,omitempty"`
		Color       string `json:"color,omitempty"`
		Action      string `json:"action,omitempty"`
		Notice      string `json:"notice,omitempty"`
	} `json:"reward,omitempty"`
	IsReward bool `json:"isReward,omitempty"`
}

func (a aliyundrive) signIn() error {
	req, err := http.NewRequest("POST", signinURL, strings.NewReader("{}"))
	if err != nil {
		log.Printf("签到失败, 错误信息: %v\n", err)
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.accessToken))
	req.Header.Set("Content-Type", "application/json")

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("签到失败, 错误信息: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("签到失败, 错误信息: %v\n", err)
		return err
	}

	if gjson.GetBytes(body, "success").Bool() == true {
		count := gjson.GetBytes(body, "result.signInCount").Int()
		result := gjson.GetBytes(body, fmt.Sprintf("result.signInLogs.%d", count-1))
		ltime, _ := time.LoadLocation("Asia/Shanghai")
		fmt.Println("东八区时间:", time.Now().Local().In(ltime).Format("2006-01-02 15:04:05"))
		log.Printf("签到成功,已签到%d天,本月第%d日\n", count, result.Get("calendarDay").Int())
		if result.Get("isReward").Bool() { //奖励
			log.Printf("%s\n", result.Get("notice").String())
		} else {
			log.Println("无奖励")
		}
		// log.Println(result.String()
	} else {
		log.Println(gjson.GetBytes(body, "message").String())
		return fmt.Errorf("签到失败:%s\n", gjson.GetBytes(body, "message").String())
	}
	return nil
}

func pushplus(content string) {
	v := url.Values{}
	v.Add("token", pushplus_token)
	v.Add("title", "阿里云盘签到")
	v.Add("content", content)
	pushplus_url := "http://www.pushplus.plus/send?" + v.Encode()
	req, _ := http.NewRequest("GET", pushplus_url, nil)
	req.Header.Set("Content-Type", "application/json")
	_, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("发送通知失败,%s", err.Error())
	}
}

func dingding(result string){
	// 构造要发送的消息
	message := struct {
		MsgType string `json:"msgtype"`
		Text struct {
			Content string `json:"content"`
		} `json:"text"`
	}{
		MsgType: "text",
		Text: struct {
			Content string `json:"content"`
		}{
			Content: "HiFiNi" + result,
		},
	}

	// 将消息转换为JSON格式
	messageJson, _ := json.Marshal(message)
	DINGDING_WEBHOOK := os.Getenv("DINGDING_WEBHOOK")
	// 发送HTTP POST请求
	resp, err := http.Post(DINGDING_WEBHOOK,
		"application/json", bytes.NewBuffer(messageJson))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
}

func main() {
	rt := os.Getenv("REFRESH_TOKENS")
	if rt == "" {
		panic("未配置阿里云盘REFRESH_TOKENS")
	} else {
		refresh_Token = rt
	}
	// pptoken := os.Getenv("PUSHPLUS_TOKEN")
	// if pptoken == "" {
	// 	panic("未配置PUSHPLUS_TOKEN")
	// } else {
	// 	pushplus_token = pptoken
	// }

	a := New(refresh_Token)
	a.getAccessToken()
	err := a.signIn()
	if err != nil {
		// pushplus(err.Error())
		dingding(err.Error())
	}
}
