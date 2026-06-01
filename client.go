package main

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// Client 天翼校园网认证客户端
type Client struct {
	Config          *Config
	Log             *log.Logger
	HttpClient      *http.Client
	Ctx             context.Context
	Cancel          context.CancelFunc
	cipher          Cipher
	heartBeatTicker *time.Ticker
	lastHeartbeatInterval int

	UserIP     string
	AcIP       string
	Domain     string
	Area       string
	SchoolID   string
	ClientID   uuid.UUID
	Hostname   string
	MacAddress string
	Ticket     string
	AlgoID     string

	IndexUrl    string
	TicketUrl   string
	AuthUrl     string
	KeepUrl     string
	TermUrl     string
	RedirectUrl string

	OnStatusChange func(status string)
	OnAuthSuccess  func(userIP string)
	OnHeartbeat    func(interval int)
}

// NewClient 创建认证客户端
func NewClient(config *Config) (*Client, error) {
	if config.Username == "" || config.Password == "" {
		return nil, errors.New("username or password is empty")
	}

	transport, err := NewHttpTransport(config)
	if err != nil {
		return nil, errors.New(fmt.Errorf("failed to create transport: %w", err).Error())
	}

	ctx, cancel := context.WithCancel(context.Background())

	rid := GenerateRandomString(5)

	if config.BindInterface == "" {
		config.BindInterface = "sys_default"
	}
	if config.CheckInterval <= 0 {
		config.CheckInterval = 10000
	}
	if config.RetryInterval == 0 {
		config.RetryInterval = 10000
	}
	if config.RetryInterval < 0 {
		config.RetryInterval = math.MaxInt32
	}

	cl := &Client{
		Config: config,
		Ctx:    ctx,
		Cancel: cancel,
		HttpClient: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: transport,
		},
		AlgoID: "00000000-0000-0000-0000-000000000000",
		Log: log.New(
			os.Stdout,
			"["+rid+"][user:"+config.Username+" bind_device:"+config.BindInterface+"] ",
			log.LstdFlags|log.Lmsgprefix,
		),
		heartBeatTicker: time.NewTicker(time.Duration(math.MaxInt32)),
	}

	return cl, nil
}

// Start 启动客户端主循环，定期检测网络状态并发送心跳
func (c *Client) Start() {
	c.Log.Println("client start")
	defer c.heartBeatTicker.Stop()
	defer c.Logout()

	if err := c.CheckNetwork(); err != nil {
		c.Log.Printf("Network check failed:%v", err)
	}

	ticker := time.NewTicker(time.Millisecond * time.Duration(c.Config.CheckInterval))
	defer ticker.Stop()

	for {
		select {
		case <-c.Ctx.Done():
			c.Log.Println("client context cancel")
			return
		case <-ticker.C:
			if err := c.CheckNetwork(); err != nil {
				c.Log.Printf("Network check failed:%v", err)
			}
		case <-c.heartBeatTicker.C:
			err := c.SendHeartbeat()
			if err != nil {
				c.Log.Printf("send heartbeat error: %v", err)
			} else {
				c.Log.Println("send heartbeat")
				if c.OnStatusChange != nil {
					c.OnStatusChange("online")
				}
				if c.OnHeartbeat != nil {
					c.OnHeartbeat(c.lastHeartbeatInterval)
				}
			}
		}
	}
}

// SendHeartbeat 发送心跳包，维持在线状态
func (c *Client) SendHeartbeat() error {
	stateXML, err := c.GenerateStateXML()
	if err != nil {
		return errors.New(err.Error())
	}

	decrypted, err := c.PostXML(c.KeepUrl, stateXML)
	if err != nil {
		return errors.New(err.Error())
	}

	var stateResp StateResponse
	if err := xml.Unmarshal(decrypted, &stateResp); err != nil {
		return errors.New(err.Error())
	}

	interval, err := strconv.Atoi(stateResp.Interval)
	if err != nil {
		return errors.New(err.Error())
	}

	c.heartBeatTicker.Reset(time.Duration(interval) * time.Second)
	c.lastHeartbeatInterval = interval
	return nil
}

// Logout 注销并断开连接
func (c *Client) Logout() {
	request, err := c.NewGetRequest("http://connect.rom.miui.com/generate_204")
	if err != nil {
		if c.OnStatusChange != nil {
			c.OnStatusChange("offline")
		}
		return
	}
	resp, err := c.HttpClient.Do(request)
	if err == nil && resp != nil && resp.StatusCode == http.StatusNoContent && c.cipher != nil {
		stateXML, _ := c.GenerateStateXML()
		_, _ = c.PostXMLWithTimeout(c.TermUrl, stateXML)
		c.Log.Println("log out request sent")
	}
	if c.OnStatusChange != nil {
		c.OnStatusChange("offline")
	}
}

// CheckNetwork 检测网络状态，未认证时自动触发认证流程
func (c *Client) CheckNetwork() error {
	request, err := c.NewGetRequest("http://connect.rom.miui.com/generate_204")
	if err != nil {
		return errors.New(err.Error())
	}

	resp, err := c.HttpClient.Do(request)
	if err != nil {
		return errors.New(err.Error())
	}
	if resp == nil {
		return errors.New("nil response")
	}
	defer func(Body io.ReadCloser) {
		if Body != nil {
			_ = Body.Close()
		}
	}(resp.Body)

	switch resp.StatusCode {
	case http.StatusNoContent:
		if c.OnStatusChange != nil {
			c.OnStatusChange("online")
		}
		return nil

	case http.StatusFound:
		c.heartBeatTicker.Reset(time.Duration(math.MaxInt32))
		c.Log.Println("auth required")
		if c.OnStatusChange != nil {
			c.OnStatusChange("auth")
		}
		return c.HandleRedirect(resp)

	default:
		return errors.New(fmt.Sprintf("unexpected status code: %d", resp.StatusCode))
	}
}

// HandleRedirect 处理 302 重定向，执行认证
func (c *Client) HandleRedirect(resp *http.Response) error {
	if err := c.Auth(resp.Header.Get("Location")); err != nil {
		c.Log.Printf("auth failed: %v", err)
		if c.OnStatusChange != nil {
			c.OnStatusChange("offline")
		}
		return nil
	}

	c.Log.Println("auth finished")
	if c.OnStatusChange != nil {
		c.OnStatusChange("online")
	}
	if c.OnAuthSuccess != nil {
		c.OnAuthSuccess(c.UserIP)
	}
	return nil
}
