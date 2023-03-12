package utils

import (
    "bytes"
    "context"
    "fmt"
    "log"
    "math/rand"
    "strings"
    "time"

    "github.com/chromedp/cdproto/cdp"
    "github.com/chromedp/cdproto/dom"
    "github.com/chromedp/cdproto/input"
    "github.com/chromedp/cdproto/network"
    "github.com/chromedp/cdproto/page"
    "github.com/chromedp/cdproto/runtime"
    "github.com/chromedp/chromedp"
)

func LoginByUserNameAndPass() (err error) {

    ua:="Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36"
    options := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.Flag("headless", true), // debug使用
        chromedp.Flag("blink-settings", "imagesEnabled=false"), //不显示图片
        chromedp.UserAgent(ua), //自定义ua
    )
    allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), options...)
    defer cancel()
    // create context
    ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
    defer cancel()
    // create a timeout
    ctx, cancel = context.WithTimeout(ctx, 30000*time.Second)
    defer cancel()


    user, err := C.GetValue("login_info", "user")
    if err!=nil {
        panic("no user")
    }
    pass, err := C.GetValue("login_info", "pass")
    if err!=nil {
        panic("no pass")
    }
    cookiesVal := ""

    //cookie
    err = chromedp.Run(ctx,
        // //设置webdriver检测反爬
        chromedp.ActionFunc(func(cxt context.Context) error {
            _, err := page.AddScriptToEvaluateOnNewDocument("Object.defineProperty(navigator, 'webdriver', { get: () => false, });").Do(cxt)
            return err
        }),
        //登录链接
        chromedp.Navigate(`https://kyfw.12306.cn/otn/resources/login.html`),
        //等待页面元素加载完成
        // chromedp.WaitVisible(`#nc_1_n1z`),
        chromedp.Sleep(time.Second*1),

        // 账号
        chromedp.SetValue(`#J-userName`, user, chromedp.ByID),
        // 密码
        chromedp.SetValue(`#J-password`, pass, chromedp.ByID),
        chromedp.Click(`#J-login`, chromedp.ByID),
        // 模拟滑动验证
        chromedp.QueryAfter("#nc_1_n1z", func(fctx context.Context, id runtime.ExecutionContextID, node ...*cdp.Node) error {
            n:=node[0]
            fmt.Println(n)

            return MouseDragNode(n, fctx)
        }),
        chromedp.Sleep(time.Millisecond*300),
        // //点击登录
        // chromedp.Click(`.content1-s3-p4`, chromedp.ByQuery),
        // chromedp.Sleep(time.Millisecond*300),
        // //获取cookie
        chromedp.ActionFunc(func(cctx context.Context) error {
            for i:=0; i<1; i++ {
                cookes,err:=network.GetAllCookies().Do(cctx)
                if err!=nil {
                    return err
                }
                var cookieStr bytes.Buffer
                for _, v := range cookes {
                    cookieStr.WriteString(v.Name + "=" + v.Value + ";")
                }
                cookiesVal = cookieStr.String()
                fmt.Println(cookiesVal)
                if strings.Contains(cookiesVal,"acw_tc") {
                    break
                }
                time.Sleep(time.Millisecond*500)
            }
            return nil
        }),
    )

    AddCookieStr([]string{cookiesVal})

    SugarLogger.Infof("cookie\n %v \n", cookie.cookie)
    return err
}


//模拟滑动
func MouseDragNode(n *cdp.Node, cxt context.Context) error {
    boxes, err := dom.GetContentQuads().WithNodeID(n.NodeID).Do(cxt)
    if err!=nil {
        return err
    }
    if len(boxes) == 0 {
        return chromedp.ErrInvalidDimensions
    }
    content := boxes[0]
    c := len(content)
    if c%2 != 0 || c < 1 {
        return chromedp.ErrInvalidDimensions
    }
    var x, y float64
    for i := 0; i < c; i += 2 {
        x += content[i]
        y += content[i+1]
    }
    x /= float64(c / 2)
    y /= float64(c / 2)
    p := &input.DispatchMouseEventParams{
        Type:       input.MousePressed,
        X:          x,
        Y:          y,
        Button:     input.Left,
        ClickCount: 1,
    }
    // 鼠标左键按下
    if err := p.Do(cxt); err != nil {
        return err
    }
    // 拖动
    p.Type = input.MouseMoved
    max   := 900.0
    for {
        if p.X > max {
            break
        }
        rt := rand.Intn(150)+50
        // rt := rand.Intn(200)
        chromedp.Run(cxt, chromedp.Sleep(time.Millisecond * time.Duration(rt)))
        x := rand.Intn(10) + 15
        y := rand.Intn(5)
        p.X = p.X + float64(x)
        p.Y = p.Y + float64(y)
        //fmt.Println("X坐标：",p.X)
        if err := p.Do(cxt); err != nil {
            return err
        }
    }
    // 鼠标松开
    p.Type = input.MouseReleased
    return p.Do(cxt)
}
