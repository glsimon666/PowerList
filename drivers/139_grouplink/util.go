package _139_grouplink

import (
	"encoding/json"
	"errors"
	"sync/atomic"
	log "github.com/sirupsen/logrus"
	_139 "github.com/OpenListTeam/OpenList/v4/drivers/139"
	"github.com/OpenListTeam/OpenList/v4/drivers/base"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"net/http"
)

const apiBase = "https://share-kd-njs.yun.139.com/yun-share/general/IOutLink/"
var idx int32 = 0

// ---------------------- 新增：grouplink专属下载接口 结构体 ----------------------
// GetDownloadUrlReq getDownloadUrl接口请求体（贴合Scheme，精简）
type GetDownloadUrlReq struct {
	LinkID  string `json:"linkID"`  // 分组分享ID（grouplink的ShareId）
	CoIDLst CoIDLst `json:"coIDLst"`// 文件ID列表
}
// CoIDLst 139生态通用的文件ID列表结构
type CoIDLst struct {
	Item []string `json:"item"` // 文件ID数组，单个文件也传数组
}

// GetDownloadUrlResp getDownloadUrl接口响应体（贴合Scheme，精简）
type GetDownloadUrlResp struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		DownloadURL string `json:"downloadURL"` // 核心高速下载直链
	} `json:"data"`
}
// ---------------------- 新增结束 ----------------------

// httpPost 封装POST请求（保留auth参数，鉴权开关）
func (y *Yun139GroupLink) httpPost(pathname string, data interface{}, auth bool) ([]byte, error) {
	u := apiBase + pathname
	req := base.RestyClient.R()

	// 请求头：贴合抓包，保留必选头，x-share-channel=0102（抓包固定值）
	req.SetHeaders(map[string]string{
		"Content-Type":     "application/json;charset=utf-8",
		"Referer":          "https://yun.139.com/",
		"User-Agent":       "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0",
		"Origin":           "https://yun.139.com",
		"x-share-channel":  "0102", // 抓包固定值，接口硬要求
		"hcy-cool-flag":    "1",
	})

	// 鉴权逻辑：复用原有，auth=true时自动添加139Yun的Authorization
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

// ---------------------- 新增：grouplink专属下载接口调用方法 ----------------------
// getDownloadUrl 调用抓包的专属下载接口，带鉴权，返回高速直链
func (y *Yun139GroupLink) getDownloadUrl(fid string) (string, error) {
	// 1. 构造请求体（贴合Scheme，linkID=当前分组的ShareId）
	req := GetDownloadUrlReq{
		LinkID: y.ShareId,
		CoIDLst: CoIDLst{
			Item: []string{fid}, // 单个文件ID，按139规范传数组
		},
	}

	// 2. 调用接口：auth=true（自动添加Authorization鉴权头），无加解密
	respBody, err := y.httpPost("getDownloadUrl", req, true)
	if err != nil {
		return "", fmt.Errorf("下载接口请求失败：%v", err)
	}

	// 3. 解析明文响应体（无解密，直接解析）
	var resp GetDownloadUrlResp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("下载响应解析失败：%v，body：%s", err, string(respBody))
	}

	// 4. 判断是否成功（和list接口一致的判断逻辑，减少冗余）
	if !resp.Success || resp.Code != "0000" {
		return "", fmt.Errorf("下载接口返回错误：%s（码：%s）", resp.Message, resp.Code)
	}

	// 5. 校验直链是否有效
	if resp.Data.DownloadURL == "" {
		return "", errors.New("下载接口未返回有效高速直链")
	}

	log.Debugf("grouplink专属接口获取高速直链成功：%s", resp.Data.DownloadURL)
	return resp.Data.DownloadURL, nil
}
// ---------------------- 新增结束 ----------------------

// getShareInfo 调用getOutLinkInfo接口获取分享信息【无鉴权】
func (y *Yun139GroupLink) getShareInfo(pCaID string, page int) (GetOutLinkInfoResp, error) {
	var resp GetOutLinkInfoResp
	size := 200
	start := page*size + 1
	end := (page + 1) * size

	reqBody := GetOutLinkInfoReq{
		LinkID: y.ShareId,
		Passwd: y.SharePwd,
		PCaID:  pCaID,
		BNum:   start,
		ENum:   end,
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
	page := 0

	for {
		res, err := y.getShareInfo(actualID, page)
		if err != nil {
			return nil, err
		}

		for _, asset := range res.Data.AssetsList {
			file := fileToObj(asset)
			files = append(files, file)
		}

		if res.Data.NextPageCursor == nil || res.Data.NextPageCursor == "" {
			break
		}
		page++
	}

	log.Debugf("获取到%d个文件", len(files))
	return files, nil
}
