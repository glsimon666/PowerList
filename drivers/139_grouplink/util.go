package _139_grouplink

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	log "github.com/sirupsen/logrus"
	_139 "github.com/OpenListTeam/OpenList/v4/drivers/139"
	"github.com/OpenListTeam/OpenList/v4/drivers/base"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"net/http"
)

const apiBase = "https://share-kd-njs.yun.139.com/yun-share/general/IOutLink/"
var idx int32 = 0

// ---------------------- grouplink专属下载接口 结构体 ----------------------
type GetDownloadUrlReq struct {
	LinkID  string `json:"linkID"`
	CoIDLst CoIDLst `json:"coIDLst"`
}
type CoIDLst struct {
	Item []string `json:"item"`
}
type GetDownloadUrlResp struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		DownloadURL string `json:"downloadURL"`
	} `json:"data"`
}
// ---------------------- 结构体结束 ----------------------

// httpPost 封装POST请求（保留auth参数，鉴权开关）
func (y *Yun139GroupLink) httpPost(pathname string, data interface{}, auth bool) ([]byte, error) {
	u := apiBase + pathname
	req := base.RestyClient.R()

	req.SetHeaders(map[string]string{
		"Content-Type":     "application/json;charset=utf-8",
		"Referer":          "https://yun.139.com/",
		"User-Agent":       "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0",
		"Origin":           "https://yun.139.com",
		"x-share-channel":  "0102", // 抓包固定值
		"hcy-cool-flag":    "1",
	})

	if auth {
		driverIdx := int(atomic.LoadInt32(&idx) % int32(op.GetDriverCount("139Yun")))
		driver := op.GetFirstDriver("139Yun", driverIdx)
		if driver != nil {
			yun139 := driver.(*_139.Yun139)
			req.SetHeader("Authorization", "Basic "+yun139.Authorization)
		} else {
			log.Warn("未找到139Yun驱动，无法添加Authorization鉴权头")
		}
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req.SetBody(jsonData)

	res, err := req.Execute(http.MethodPost, u)
	if err != nil {
		log.Warnf("HTTP请求失败: %v, url: %s", err, u)
		return nil, err
	}

	return res.Body(), nil
}

// getDownloadUrl 调用抓包的专属下载接口，带鉴权，返回高速直链
func (y *Yun139GroupLink) getDownloadUrl(fid string) (string, error) {
	req := GetDownloadUrlReq{
		LinkID: y.ShareId,
		CoIDLst: CoIDLst{
			Item: []string{fid},
		},
	}

	respBody, err := y.httpPost("getDownloadUrl", req, true)
	if err != nil {
		return "", fmt.Errorf("下载接口请求失败：%v", err)
	}

	var resp GetDownloadUrlResp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("下载响应解析失败：%v，body：%s", err, string(respBody))
	}

	if !resp.Success || resp.Code != "0000" {
		return "", fmt.Errorf("下载接口返回错误：%s（码：%s）", resp.Message, resp.Code)
	}

	if resp.Data.DownloadURL == "" {
		return "", errors.New("下载接口未返回有效高速直链")
	}

	log.Debugf("grouplink专属接口获取高速直链成功：%s", resp.Data.DownloadURL)
	return resp.Data.DownloadURL, nil
}

// getShareInfo 调用getOutLinkInfo接口获取分享信息【无鉴权】
func (y *Yun139GroupLink) getShareInfo(pCaID string, page int) (GetOutLinkInfoResp, error) {
	var resp GetOutLinkInfoResp
	size := 200 // 每页条数不变
	// ---------------------- 核心修复：2行分页计算，去掉+1，从0开始 ----------------------
	start := page * size // page=0时start=0（符合接口要求），page=1时start=200，以此类推
	end := (page + 1) * size - 1 // 配套调整end，让区间为[0,199]、[200,399]，贴合接口分页规范
	// ---------------------- 修复结束 ----------------------

	reqBody := GetOutLinkInfoReq{
		LinkID: y.ShareId,
		Passwd: y.SharePwd,
		PCaID:  pCaID,
		BNum:   start, // 传修正后的起始值
		ENum:   end,   // 传修正后的结束值
	}

	body, err := y.httpPost("getOutLinkInfo", reqBody, false)
	if err != nil {
		return resp, err
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		log.Warnf("响应解析失败: %v, body: %s", err, string(body))
		return resp, err
	}

	if !resp.Success || resp.Code != "0000" {
		return resp, errors.New(resp.Message)
	}

	return resp, nil
}

// list 获取分享文件列表（分页）
func (y *Yun139GroupLink) list(pCaID string) ([]File, error) {
	actualID := pCaID
	if pCaID == "" || pCaID == "root" {
		actualID = ""
	}

	files := make([]File, 0)
	page := 0 // 初始值0不变

	for {
		res, err := y.getShareInfo(actualID, page)
		if err != nil {
			return nil, err
		}

		for _, asset := range res.Data.AssetsList {
			file := fileToObj(asset)
			files = append(files, file)
		}

		// 分页终止条件不变，接口返回空游标则停止
		if res.Data.NextPageCursor == nil || res.Data.NextPageCursor == "" {
			break
		}
		page++
	}

	log.Debugf("获取到%d个文件", len(files))
	return files, nil
}
