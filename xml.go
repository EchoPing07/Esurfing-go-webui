package main

import (
	"context"
	"encoding/xml"
	"io"
	"time"
)

// UserAgentAndroid 模拟安卓客户端的 User-Agent
const (
	UserAgentAndroid = "CCTP/android64_vpn/2093"
)

// TicketRequest 获取票据的请求体
type TicketRequest struct {
	XMLName   xml.Name `xml:"request"`
	Text      string   `xml:",chardata"`
	UserAgent string   `xml:"user-agent"`
	ClientID  string   `xml:"client-id"`
	LocalTime string   `xml:"local-time"`
	HostName  string   `xml:"host-name"`
	Ipv4      string   `xml:"ipv4"`
	Ipv6      string   `xml:"ipv6"`
	Mac       string   `xml:"mac"`
	Ostag     string   `xml:"ostag"`
	Gwip      string   `xml:"gwip"`
}

// TicketResponse 获取票据的响应体
type TicketResponse struct {
	XMLName xml.Name `xml:"response"`
	Text    string   `xml:",chardata"`
	Ticket  string   `xml:"ticket"`
	Expire  string   `xml:"expire"`
}

// LoginRequest 登录请求体
type LoginRequest struct {
	XMLName   xml.Name `xml:"request"`
	Text      string   `xml:",chardata"`
	UserAgent string   `xml:"user-agent"`
	ClientID  string   `xml:"client-id"`
	Ticket    string   `xml:"ticket"`
	LocalTime string   `xml:"local-time"`
	Userid    string   `xml:"userid"`
	Passwd    string   `xml:"passwd"`
}

// LoginResponse 登录响应体
type LoginResponse struct {
	XMLName    xml.Name `xml:"response"`
	Text       string   `xml:",chardata"`
	Userid     string   `xml:"userid"`
	KeepRetry  string   `xml:"keep-retry"`
	KeepURL    string   `xml:"keep-url"`
	TermURL    string   `xml:"term-url"`
	UserConfig struct {
		Text            string `xml:",chardata"`
		AgainstInterval string `xml:"against-interval"`
	} `xml:"user-config"`
	DomainConfig string `xml:"domain-config"`
}

// State 心跳状态请求体
type State struct {
	XMLName   xml.Name `xml:"request"`
	Text      string   `xml:",chardata"`
	UserAgent string   `xml:"user-agent"`
	ClientID  string   `xml:"client-id"`
	LocalTime string   `xml:"local-time"`
	HostName  string   `xml:"host-name"`
	Ipv4      string   `xml:"ipv4"`
	Ticket    string   `xml:"ticket"`
	Ipv6      string   `xml:"ipv6"`
	Mac       string   `xml:"mac"`
	Ostag     string   `xml:"ostag"`
}

// StateResponse 心跳响应体
type StateResponse struct {
	XMLName  xml.Name `xml:"response"`
	Text     string   `xml:",chardata"`
	Interval string   `xml:"interval"`
	Level    string   `xml:"level"`
}

// EConfig 服务器配置（票据URL、认证URL等）
type EConfig struct {
	XMLName   xml.Name `xml:"config"`
	Text      string   `xml:",chardata"`
	TicketURL string   `xml:"ticket-url"`
	AuthURL string   `xml:"auth-url"`
}

// GenerateGetTicketXML 生成获取票据的XML请求
func (c *Client) GenerateGetTicketXML() ([]byte, error) {
	tr := TicketRequest{
		UserAgent: UserAgentAndroid,
		ClientID:  c.ClientID.String(),
		LocalTime: time.Now().Format(time.DateTime),
		HostName:  c.Hostname,
		Ipv4:      c.UserIP,
		Mac:       c.MacAddress,
		Ostag:     c.Hostname,
		Gwip:      c.AcIP,
	}
	out, err := xml.Marshal(tr)
	if err != nil {
		return nil, err
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>"), out...), nil
}

// GenerateStateXML 生成心跳状态XML请求
func (c *Client) GenerateStateXML() ([]byte, error) {
	s := &State{
		UserAgent: UserAgentAndroid,
		ClientID:  c.ClientID.String(),
		LocalTime: time.Now().Format(time.DateTime),
		HostName:  c.Hostname,
		Ipv4:      c.UserIP,
		Ticket:    c.Ticket,
		Mac:       c.MacAddress,
		Ostag:     c.Hostname,
	}
	bytes, err := xml.Marshal(s)
	if err != nil {
		return nil, err
	}

	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>"), bytes...), nil
}

// GenerateLoginXML 生成登录XML请求
func (c *Client) GenerateLoginXML() ([]byte, error) {
	lr := &LoginRequest{
		UserAgent: UserAgentAndroid,
		ClientID:  c.ClientID.String(),
		Ticket:    c.Ticket,
		LocalTime: time.Now().Format(time.DateTime),
		Userid:    c.Config.Username,
		Passwd:    c.Config.Password,
	}

	bytes, err := xml.Marshal(lr)
	if err != nil {
		return nil, err
	}

	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>"), bytes...), nil
}

// PostXML 发送加密XML请求并解密响应
func (c *Client) PostXML(url string, data []byte) ([]byte, error) {
	encXML, err := c.cipher.Encrypt(data)
	if err != nil {
		return nil, err
	}

	req, err := c.NewPostRequest(url, encXML)
	if err != nil {
		return nil, err
	}

	response, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)

	data, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return c.cipher.Decrypt(data)
}

// PostXMLWithTimeout 发送加密XML请求（带3秒超时，用于注销等操作）
func (c *Client) PostXMLWithTimeout(url string, data []byte) ([]byte, error) {
	// 设置3秒超时，避免程序退出时阻塞
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*3))
	defer cancel()
	encXML, err := c.cipher.Encrypt(data)
	if err != nil {
		return nil, err
	}

	req, err := c.NewPostRequestWithCustomCtx(ctx, url, encXML)
	if err != nil {
		return nil, err
	}

	response, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)

	data, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return c.cipher.Decrypt(data)
}
