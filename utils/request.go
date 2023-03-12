package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/satori/go.uuid"

	"github.com/yincongcyincong/go12306/module"
)

var (
	client  *http.Client
	cdnMap  = make(map[string]*http.Client)
	cdnLock = sync.Mutex{}
)

func GetClient() *http.Client {
	if client == nil {
		client = &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 1 * time.Minute,
				}).DialContext,
				TLSHandshakeTimeout:   5 * time.Second,
				ResponseHeaderTimeout: 5 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				MaxIdleConnsPerHost:   50,
				IdleConnTimeout:       10 * time.Second,
			},

		}
	}

	return client

}

func GetClientDo(req *http.Request) ([]byte,error) {

	if client == nil {
		client = &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 1 * time.Minute,
				}).DialContext,
				TLSHandshakeTimeout:   5 * time.Second,
				ResponseHeaderTimeout: 5 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				MaxIdleConnsPerHost:   50,
				IdleConnTimeout:       10 * time.Second,
			},

		}
	}

	reqID:="UUID"
	u1:= uuid.NewV4()
	reqID=u1.String()

	now:=time.Now()
	SugarLogger.Debugf("send req url:%s,reqID :%s ,header:%v", req.URL,reqID,req.Header)
	resp, err := client.Do(req)

	if err != nil || resp == nil {
		SugarLogger.Errorf("url: %s,reqID :%s,err: %v ,resp: %s", req.URL, reqID,err, resp)
		return nil, err
	}
	defer resp.Body.Close()
	setCookies := resp.Header.Values("Set-Cookie")
	AddCookieStr(setCookies)

	res, err := ioutil.ReadAll(resp.Body)
	SugarLogger.Debugf("url: %s ,reqID :%s ,resp: %s ,timeout :%s", req.URL,reqID,string(res),time.Since(now))

	return res, err
}

func GetCdnClient(cdn string) *http.Client {
	cdnLock.Lock()
	defer cdnLock.Unlock()
	if _, ok := cdnMap[cdn]; !ok {

		cdnMap[cdn] = &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					addr = cdn + ":443"
					return (&net.Dialer{
						Timeout:   1 * time.Second,
						KeepAlive: 1 * time.Minute,
					}).DialContext(ctx, network, addr)
				},
				TLSHandshakeTimeout:   1 * time.Second,
				ResponseHeaderTimeout: 1 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				MaxIdleConnsPerHost:   20,
				IdleConnTimeout:       10 * time.Second,
			},
		}
	}

	c := cdnMap[cdn]
	return c
}

func Request(data string, cookieStr, url string, res interface{}, headers map[string]string) error {

	req, err := http.NewRequest("POST", url, strings.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", cookieStr)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Host", "kyfw.12306.cn")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", "https://kyfw.12306.cn")
	req.Header.Set("Referer", "https://kyfw.12306.cn/otn/view/index.html")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	respBody, err := GetClientDo(req)

	err = json.Unmarshal(respBody, res)
	if err != nil {
		SugarLogger.Errorf("json unmarshal fail: %v, %v, %v", err, string(respBody), url)
		return err
	}

	return nil
}

func RequestGet(cookieStr, url string, res interface{}, headers map[string]string) error {

	req, err := http.NewRequest("GET", url, strings.NewReader(""))
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", cookieStr)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Host", "kyfw.12306.cn")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", "https://kyfw.12306.cn")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	respBody, err := GetClientDo(req)

	err = json.Unmarshal(respBody, res)
	if err != nil {
		return err
	}

	return nil
}

func RequestGetWithoutJson(cookieStr, url string, headers map[string]string) ([]byte, error) {

	req, err := http.NewRequest("GET", url, strings.NewReader(""))
	if err != nil {
		return []byte{}, err
	}
	req.Header.Set("Cookie", cookieStr)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Host", "kyfw.12306.cn")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", "https://kyfw.12306.cn")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	respBody, err := GetClientDo(req)
	if err != nil {
		return []byte{}, err
	}

	return respBody, nil
}

func RequestGetWithCDN(cookieStr, url string, res interface{}, headers map[string]string, cdn string) error {

	req, err := http.NewRequest("GET", url, strings.NewReader(""))
	if err != nil {
		return err
	}

	req.Header.Set("Cookie", cookieStr)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Host", "kyfw.12306.cn")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", "https://kyfw.12306.cn")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	SugarLogger.Infof("url\n %v \n", url)
	// SugarLogger.Infof("header \n %v",req.Header)
	// SugarLogger.Infof("cookie\n %v \n", cookie.cookie)

	now := time.Now()
	var  resp *http.Response
	if OpenCND==0{
		http_client := &http.Client{}
		resp, err = http_client.Do(req)
	}else {
		resp, err = GetCdnClient(cdn).Do(req)
	}

	SugarLogger.Infof("URL= %v timecoset %v \n",url,time.Since(now))
	if err != nil ||resp==nil {
		return fmt.Errorf("%s ：%s",err.Error(),"resp is nuil")
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	SugarLogger.Debugf("url: %v, response: %v, cdn: %s", url, string(respBody), cdn)

	if res != nil {
		err = json.Unmarshal(respBody, res)
		if err != nil {
			return fmt.Errorf("%s ：%s",err.Error(),"json Unmarshal resp failed")
		}
	}

	// 添加cookie
	setCookies := resp.Header.Values("Set-Cookie")
	AddCookieStr(setCookies)

	return nil
}

func EncodeParam(r *http.Request, param interface{}) error {
	respBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(respBody, param)
	if err != nil {
		return err
	}

	return nil
}

func HTTPSuccResp(w http.ResponseWriter, data interface{}) {
	w.WriteHeader(http.StatusOK)
	resp := &module.CommonResp{
		Code:    0,
		Message: "success",
		Data:    data,
	}
	respJson, _ := json.Marshal(resp)
	fmt.Fprint(w, string(respJson))
}

func HTTPFailResp(w http.ResponseWriter, statusCode, code int, message string, data interface{}) {
	w.WriteHeader(statusCode)
	resp := &module.CommonResp{
		Code:    code,
		Message: message,
		Data:    data,
	}
	respJson, _ := json.Marshal(resp)
	fmt.Fprint(w, string(respJson))
}
