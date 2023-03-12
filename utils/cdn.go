package utils

import (
	"sync"
	"sync/atomic"
)

type AvailableCDN struct {
	cdns     []string
	currency int
	lock     sync.Mutex
	wg       sync.WaitGroup
	idx      int64
}

var availableCDN = &AvailableCDN{
	cdns:     make([]string, 0),
	currency: 10,
	lock:     sync.Mutex{},
	wg:       sync.WaitGroup{},
	idx:      0,
}

func InitAvailableCDN() {
		defer func() {
			if err := recover(); err != nil {
				SugarLogger.Error(err)
				SugarLogger.Sync()
			}
		}()

		if OpenCND==0{
			return
		}
		num := (len(CDNs) / availableCDN.currency) + 1
		for i := 0; i < num; i++ {
			availableCDN.wg.Add(1)
			var tmpCDNs []string
			if i != num-1 {
				tmpCDNs = CDNs[i*availableCDN.currency : (i+1)*availableCDN.currency]
			} else {
				tmpCDNs = CDNs[i*availableCDN.currency:]
			}

			go func(cdns []string) {
				defer func() {
					if err := recover(); err != nil {
						SugarLogger.Error(err)
						SugarLogger.Sync()
					}
				}()

				defer availableCDN.wg.Done()
				for _, cdn := range cdns {
					err := RequestGetWithCDN(GetCookieStr(), "https://kyfw.12306.cn/otn/dynamicJs/omseuuq", nil, nil, cdn)
					// err := RequestGetWithCDN(GetCookieStr(), "https://www.baidu.com", nil, nil, cdn)
					if err != nil {
						SugarLogger.Infof("cdn %s 请求失败", cdn)
						continue
					}

					availableCDN.lock.Lock()
					availableCDN.cdns = append(availableCDN.cdns, cdn)
					availableCDN.lock.Unlock()

				}

			}(tmpCDNs)
		}

		availableCDN.wg.Wait()
		SugarLogger.Infof("可用cdn数量为: %d", len(availableCDN.cdns))
		SugarLogger.Infof("可用cdn数量: %v", availableCDN.cdns)


}

func GetCdn() string {
	if len(availableCDN.cdns)==0{
		return ""
	}
	cdn := availableCDN.cdns[int(availableCDN.idx)%len(availableCDN.cdns)]
	atomic.AddInt64(&availableCDN.idx, 1)
	return cdn
}
