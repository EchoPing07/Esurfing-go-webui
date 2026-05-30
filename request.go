package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"net/http"
)

// NewGetRequest 创建带认证头的 GET 请求
func (c *Client) NewGetRequest(url string) (request *http.Request, err error) {
	req, err := http.NewRequestWithContext(c.Ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", UserAgentAndroid)
	req.Header.Set("Accept", "text/html,text/xml,application/xhtml+xml,application/x-javascript,*/*")
	req.Header.Set("Client-ID", c.ClientID.String())
	req.Header.Set("Connection", "keep-alive")
	if c.SchoolID != "" {
		req.Header.Set("CDC-SchoolId", c.SchoolID)
	}
	if c.Domain != "" {
		req.Header.Set("CDC-Domain", c.Domain)
	}
	if c.Area != "" {
		req.Header.Set("CDC-Area", c.Area)
	}

	return req, nil
}

// NewPostRequest 创建带认证头的 POST 请求
func (c *Client) NewPostRequest(url string, data []byte) (request *http.Request, err error) {
	md5Hex := md5.Sum(data)

	req, err := http.NewRequestWithContext(c.Ctx, http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgentAndroid)
	req.Header.Set("Accept", "text/html,text/xml,application/xhtml+xml,application/x-javascript,*/*")
	req.Header.Set("Client-ID", c.ClientID.String())
	req.Header.Set("CDC-Checksum", hex.EncodeToString(md5Hex[:]))
	req.Header.Set("Algo-ID", c.AlgoID)
	return req, nil
}

// NewPostRequestWithCustomCtx 创建带自定义上下文的 POST 请求（用于超时控制）
func (c *Client) NewPostRequestWithCustomCtx(ctx context.Context, url string, data []byte) (request *http.Request, err error) {
	md5Hex := md5.Sum(data)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgentAndroid)
	req.Header.Set("Accept", "text/html,text/xml,application/xhtml+xml,application/x-javascript,*/*")
	req.Header.Set("Client-ID", c.ClientID.String())
	req.Header.Set("CDC-Checksum", hex.EncodeToString(md5Hex[:]))
	req.Header.Set("Algo-ID", c.AlgoID)
	return req, nil
}
