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
	// 分页计算：从0开始，左闭右闭区间，贴合接口规范
	start := page * size
	end := (page + 1) * size - 1

	reqBody := GetOutLinkInfoReq{
		LinkID: y.ShareId,
		Passwd: y.SharePwd,
		PCaID:  pCaID, // 现在传的是有效pCaId，非空
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

// list 获取分享文件列表（分页）【核心修改：根目录先探活获取真实pCaId，禁止传空】
func (y *Yun139GroupLink) list(pCaID string) ([]File, error) {
	var actualID string
	files := make([]File, 0)
	page := 0 // 初始值0不变

	// ---------------------- 核心修复：根目录处理逻辑，禁止传空pCaId ----------------------
	if pCaID == "" || pCaID == "root" {
		// 根目录场景：先做一次探活调用（page=0，传临时占位符"root"），获取接口返回的真实根pCaId
		probeResp, err := y.getShareInfo("root", 0)
		if err != nil {
			return nil, fmt.Errorf("根目录探活获取pCaId失败：%v", err)
		}
		// 校验接口返回的根pCaId是否有效，避免接口返回空
		if probeResp.Data.PcaId == "" {
			return nil, errors.New("接口未返回有效根目录pCaId")
		}
		actualID = probeResp.Data.PcaId // 用接口返回的真实pCaId作为后续查询的ID
		log.Debugf("根目录探活成功，获取真实pCaId：%s", actualID)

		// 把探活调用的第一页数据直接加入文件列表，避免重复查询
		for _, asset := range probeResp.Data.AssetsList {
			file := fileToObj(asset)
			files = append(files, file)
		}
		// 探活后如果有下一页，page直接置1，继续查询
		if probeResp.Data.NextPageCursor == nil || probeResp.Data.NextPageCursor == "" {
			log.Debugf("根目录仅1页数据，共%d个文件", len(files))
			return files, nil
		}
		page = 1
	} else {
		// 非根目录场景：保留原有逻辑，直接用传入的pCaID
		actualID = pCaID
	}
	// ---------------------- 根目录修复结束 ----------------------

	// 非根目录分页查询 / 根目录后续页查询（通用逻辑，无修改）
	for {
		res, err := y.getShareInfo(actualID, page)
		if err != nil {
			return nil, fmt.Errorf("分页查询失败（page=%d）：%v", page, err)
		}

		for _, asset := range res.Data.AssetsList {
			file := fileToObj(asset)
			files = append(files, file)
		}

		// 无下一页则终止
		if res.Data.NextPageCursor == nil || res.Data.NextPageCursor == "" {
			break
		}
		page++
	}

	log.Debugf("获取到%d个文件", len(files))
	return files, nil
}
