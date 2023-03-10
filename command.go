package main

import "C"
import (
	"context"
	"errors"
	"fmt"
	"github.com/cihub/seelog"
	"github.com/yincongcyincong/go12306/action"
	"github.com/yincongcyincong/go12306/module"
	"github.com/yincongcyincong/go12306/utils"
	// "math"
	"strings"
	"time"
)

var trainCache *module.TrainData

func CommandStart() {
	defer func() {
		if err := recover(); err != nil {
			seelog.Error(err)
			seelog.Flush()
		}
	}()

	err := action.GetLoginData()
	if err != nil {
		seelog.Errorf("GetLoginDataRes:%v", err)

		loginType, err := utils.C.Int("login_type", "login_type")
		if err != nil {
			loginType = 0
		}

		if loginType == 0 {
			qrImage, err := action.CreateImage()
			if err != nil {
				seelog.Errorf("创建二维码失败:%v", err)
				return
			}
			qrImage.Image = ""

			err = action.QrLogin(qrImage)
			if err != nil {
				seelog.Errorf("登陆失败:%v", err)
				return
			}
		} else {
			err := utils.LoginByUserNameAndPass()
			if err != nil {
				seelog.Errorf("模拟登陆失败:%v", err)
				return
			}
			err = action.GetLoginData()
			if err != nil {
				seelog.Errorf("模拟登录检测失败:%v", err)

				return
			}
		}
	}

	startCheckLogin()

	// 获取配置参数
	ConfigFile := utils.C

	searchParam := new(module.SearchParam)
	var trainStr, seatStr, passengerStr string
	passengerStr, err = ConfigFile.GetValue("passenger", "name")
	if err != nil {
		panic(err)
	}
	searchParam.TrainDate, err = ConfigFile.GetValue("passenger", "date")
	if err != nil {
		panic(err)
	}
	searchParam.FromStationName, err = ConfigFile.GetValue("passenger", "from")
	if err != nil {
		panic(err)
	}
	searchParam.ToStationName, err = ConfigFile.GetValue("passenger", "to")
	if err != nil {
		panic(err)
	}
	searchParam.FromStation = utils.Station[searchParam.FromStationName]
	searchParam.ToStation = utils.Station[searchParam.ToStationName]
	trainStr, err = ConfigFile.GetValue("passenger", "train")
	if err != nil {
		panic(err)
	}
	seatStr, err = ConfigFile.GetValue("passenger", "seat")
	if err != nil {
		panic(err)
	}
	isNate, err := ConfigFile.GetValue("passenger", "isNate")
	if err != nil {
		panic(err)
	}
	trainMap := utils.GetBoolMap(strings.Split(trainStr, "#"))
	passengerMap := utils.GetBoolMap(strings.Split(passengerStr, "#"))
	seatSlice := strings.Split(seatStr, "#")

	// 查询车次测试
	trains, err := action.GetTrainInfo(searchParam)
	if err != nil {
		seelog.Errorf("查询车站失败:%v", err)
		return
	}
	for _, t := range trains {
		seelog.Infof("车次: %s, 状态: %s, 始发车站: %s, 终点站:%s,  %s: %s, 历时：%s, 二等座: %s, 一等座: %s, 商务座: %s, 软卧: %s, 硬卧: %s，软座: %s，硬座: %s， 无座: %s,",
			t.TrainNo, t.Status, t.FromStationName, t.ToStationName, t.StartTime, t.ArrivalTime, t.DistanceTime, t.SeatInfo["二等座"], t.SeatInfo["一等座"], t.SeatInfo["商务座"], t.SeatInfo["软卧"], t.SeatInfo["硬卧"], t.SeatInfo["软座"], t.SeatInfo["硬座"], t.SeatInfo["无座"])
	}

	// 获取定时抢票配置
	startRunTimeStr, err := ConfigFile.GetValue("cron", "startRunTime")
	OffsetMs, err := ConfigFile.Int("cron", "OffsetMs")
	if err != nil {
		OffsetMs = 0
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return
	}
	startRunTime, err := time.ParseInLocation("2006-01-02 15:04:05", startRunTimeStr, loc)
	if err != nil {
		startRunTime = time.Now()
	}
	offset := time.Duration(OffsetMs)
	startRunTime = startRunTime.Add(offset * time.Millisecond)

	fmt.Printf("开始时间：%v\n", startRunTime.String())
	fmt.Printf("倒计时执行时间：%v\n", startRunTime.Sub(time.Now()).String())
	goStartOffsetMs, err := ConfigFile.Int("cron", "goStartOffsetMs")
	if err != nil {
		goStartOffsetMs = 0
	}

	// 开启抢票
	time.AfterFunc(startRunTime.Sub(time.Now()), func() {
		// 修改多线程
		for i := 0; i < 2; i++ {
			// 梯度开始刷票时间
			time.Sleep(time.Duration(i * goStartOffsetMs))

			ctx, cancelFunc := context.WithCancel(context.Background())
			go func(threadID int) {
				// Search:
				var t *module.TrainData
				var isAfterNate bool
				for i := 0; i < 2; i++ {
					seelog.Infof("第%d线程第%d次循环\n", threadID, i)
					t, isAfterNate, err = getTrainInfo(ctx, searchParam, trainMap, seatSlice, isNate)
					seelog.Infof("刷到票时间%v\n",time.Now())
					if t != nil {
						seelog.Infof("车次: %s, 状态: %s, 始发车站: %s, 终点站:%s,  %s: %s, 历时：%s, 二等座: %s, 一等座: %s, 商务座: %s, 软卧: %s, 硬卧: %s，软座: %s，硬座: %s， 无座: %s, \n sss:%s",
							t.TrainNo, t.Status, t.FromStationName, t.ToStationName, t.StartTime, t.ArrivalTime, t.DistanceTime, t.SeatInfo["二等座"], t.SeatInfo["一等座"], t.SeatInfo["商务座"], t.SeatInfo["软卧"], t.SeatInfo["硬卧"], t.SeatInfo["软座"], t.SeatInfo["硬座"], t.SeatInfo["无座"], t.SecretStr)
						trainCache = t
					}
					if err == success {
						cancelFunc()
						break
					}
					if err == nil {
						cancelFunc()
						break
						// time.Sleep(time.Duration(utils.GetRand(utils.SearchInterval[0], utils.SearchInterval[1])) * time.Millisecond)
					} else {
						time.Sleep(time.Duration(utils.GetRand(utils.SearchInterval[0], utils.SearchInterval[1])) * time.Millisecond)
					}
				}

				if isAfterNate {
					seelog.Info("开始候补", t.TrainNo)
					err = startAfterNate(searchParam, t, passengerMap)
				} else {
					seelog.Info("开始购买", t.TrainNo)
					err = startOrder(searchParam, t, passengerMap)
				}

				// 购买成功加入小黑屋
				if err != nil {
					utils.AddBlackList(t.TrainNo)
					// goto Search
				}

				// 暂时用不上
				if *wxrobot != "" {
					utils.SendWxrootMessage(*wxrobot, fmt.Sprintf("车次：%s 购买成功,  %s 请登陆12306查看，付款", t.TrainNo,passengerStr))
				}
				// goto Reorder

			}(i)
		}
	})

}

var success = errors.New("success")

func getTrainInfo(ctx context.Context, searchParam *module.SearchParam, trainMap map[string]bool, seatSlice []string, isNate string) (*module.TrainData, bool, error) {
	// 如果在晚上11点到早上5点之间，停止抢票，只自动登陆
	waitToOrder()

	var err error
	searchParam.SeatType = ""
	var trainData *module.TrainData
	select {
	case <-ctx.Done():
		return nil, false, success

	default:

	}
	trains, err := action.GetTrainInfo(searchParam)
	if err != nil {
		seelog.Errorf("查询车站失败:%v", err)
		return nil, false, err
	}

	for _, t := range trains {
		// 在选中的，但是不在小黑屋里面
		if utils.InBlackList(t.TrainNo) {
			seelog.Info(t.TrainNo, "在小黑屋，需等待60s")
			continue
		}

		if trainMap[t.TrainNo] {
			for _, s := range seatSlice {
				if t.SeatInfo[s] != "" && t.SeatInfo[s] != "无" {
					trainData = t
					searchParam.SeatType = utils.OrderSeatType[s]
					seelog.Infof("%s %s 数量: %s", t.TrainNo, s, t.SeatInfo[s])
					return trainData, false, nil
				}
				seelog.Infof("%s %s 数量: %s", t.TrainNo, s, t.SeatInfo[s])
			}
		}
	}

	// 判断是否可以候补
	if (isNate == "1" || isNate == "是") && searchParam.SeatType == "" {
		for _, t := range trains {
			// 在选中的，但是不在小黑屋里面
			if utils.InBlackList(t.TrainNo) {
				seelog.Info(t.TrainNo, "在小黑屋，需等待60s")
				continue
			}

			if trainMap[t.TrainNo] {
				for _, s := range seatSlice {
					if t.SeatInfo[s] == "无" && t.IsCanNate == "1" {
						trainData = t
						searchParam.SeatType = utils.OrderSeatType[s]
						return trainData, true, nil
					}
				}
			}
		}
	}

	if trainData == nil || searchParam.SeatType == "" {
		seelog.Info("暂无车票可以购买")
		return nil, false, errors.New("暂无车票可以购买")
	}

	return trainData, false, nil
}

func startOrder(searchParam *module.SearchParam, trainData *module.TrainData, passengerMap map[string]bool) error {
	// err := action.GetLoginData()
	// if err != nil {
	// 	seelog.Errorf("自动登陆失败：%v", err)
	// 	return err
	// }
	//
	// err = action.CheckUser()
	// if err != nil {
	// 	seelog.Errorf("检查用户状态失败：%v", err)
	// 	return err
	// }

	err := action.SubmitOrder(trainData, searchParam)
	if err != nil {
		seelog.Errorf("提交订单失败：%v", err)
		return err
	}

	submitToken, err := action.GetRepeatSubmitToken()
	seelog.Infof("submitToken=%v",submitToken)
	if err != nil {
		seelog.Errorf("获取提交数据失败：%v", err)
		return err
	}

	passengers, err := action.GetPassengers(submitToken)

	if err != nil {
		seelog.Errorf("获取乘客失败：%v", err)
		return err
	}
	// go
	buyPassengers := make([]*module.Passenger, 0)
	for _, p := range passengers.Data.NormalPassengers {
		if passengerMap[p.Alias] {
			buyPassengers = append(buyPassengers, p)
		}
	}
	seelog.Infof("buyPassengers=%v",buyPassengers)

	err = action.CheckOrder(buyPassengers, submitToken, searchParam)
	if err != nil {
		seelog.Errorf("检查订单失败：%v", err)
		return err
	}

	// err = action.GetQueueCount(submitToken, searchParam)
	// if err != nil {
	// 	seelog.Errorf("获取排队数失败：%v", err)
	// 	return err
	// }

	err = action.ConfirmQueue(buyPassengers, submitToken, searchParam)
	if err != nil {
		seelog.Errorf("提交订单失败：%v", err)
		return err
	}
	seelog.Infof("提交订单时间%v\n",time.Now())

	var orderWaitRes *module.OrderWaitRes
	for i := 0; i < 20; i++ {
		orderWaitRes, err = action.OrderWait(submitToken)
		if err != nil {
			time.Sleep(7 * time.Second)
			continue
		}
		if orderWaitRes != nil {
			seelog.Info("等待数据：%v", *orderWaitRes)
		}

		if orderWaitRes.Data.OrderId != "" {
			break
		}
	}

	if orderWaitRes != nil {
		err = action.OrderResult(submitToken, orderWaitRes.Data.OrderId)
		if err != nil {
			seelog.Errorf("获取订单状态失败：%v", err)
		}
	}

	if orderWaitRes == nil || orderWaitRes.Data.OrderId == "" {
		seelog.Infof("购买成功")
		return nil
	}

	seelog.Infof("购买成功，订单号：%s", orderWaitRes.Data.OrderId)
	return nil
}

func startAfterNate(searchParam *module.SearchParam, trainData *module.TrainData, passengerMap map[string]bool) error {
	err := action.GetLoginData()
	if err != nil {
		seelog.Errorf("自动登陆失败：%v", err)
		return err
	}

	err = action.AfterNateChechFace(trainData, searchParam)
	if err != nil {
		seelog.Errorf("人脸验证失败：%v", err)
		return err
	}

	_, err = action.AfterNateSuccRate(trainData, searchParam)
	if err != nil {
		seelog.Errorf("获取候补成功率失败：%v", err)
		return err
	}

	err = action.CheckUser()
	if err != nil {
		seelog.Errorf("检查用户状态失败：%v", err)
		return err
	}

	err = action.AfterNateSubmitOrder(trainData, searchParam)
	if err != nil {
		seelog.Errorf("提交候补订单失败：%v", err)
		return err
	}

	submitToken := &module.SubmitToken{
		Token: "",
	}
	passengers, err := action.GetPassengers(submitToken)
	if err != nil {
		seelog.Errorf("获取乘客失败：%v", err)
		return err
	}
	buyPassengers := make([]*module.Passenger, 0)
	for _, p := range passengers.Data.NormalPassengers {
		if passengerMap[p.Alias] {
			buyPassengers = append(buyPassengers, p)
		}
	}

	err = action.PassengerInit()
	if err != nil {
		seelog.Errorf("初始化乘客信息失败：%v", err)
		return err
	}

	err = action.AfterNateGetQueueNum()
	if err != nil {
		seelog.Errorf("获取候补排队信息失败：%v", err)
		return err
	}

	confirmRes, err := action.AfterNateConfirmHB(buyPassengers, searchParam, trainData)
	if err != nil {
		seelog.Errorf("提交订单失败：%v", err)
		return err
	}

	seelog.Infof("候补成功，订单号：%s", confirmRes.Data.ReserveNo)
	return nil
}

func waitToOrder() {
	if time.Now().Hour() >= 23 || time.Now().Hour() <= 4 {
		seelog.Infof("时间在23 点 ～5点暂时不买票")

		for {
			if 5 <= time.Now().Hour() && time.Now().Hour() < 23 {
				break
			}

			time.Sleep(1 * time.Minute)
		}
	}
}

func startCheckLogin() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				seelog.Error(err)
				seelog.Flush()
			}
		}()

		timer := time.NewTicker(2 * time.Minute)
		alTimer := time.NewTicker(10 * time.Minute)
		time.Sleep(2 * time.Minute)
		for {
			select {
			case <-timer.C:
				if !action.CheckLogin() {
					seelog.Errorf("登陆状态为未登陆")
				} else {
					seelog.Info("登陆状态为登陆中")
				}
			case <-alTimer.C:
				err := action.GetLoginData()
				if err != nil {
					seelog.Errorf("自动登陆失败：%v", err)
				}
			}
		}
	}()
}
