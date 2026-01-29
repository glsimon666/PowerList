package _139_grouplink

import (
	"encoding/json"
	"errors"
	"sync/atomic" // 原子操作，多账号轮询协程安全
	log "github.com/sirupsen/logrus"
	_139 "github.com/OpenListTeam/OpenList/v4/drivers/139"
	"github.com/OpenListTeam/OpenList/v4/drivers/base"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"net/http"
)

// 接口基础地址
const apiBase = "https://share-kd-njs.yun.139.com/yun-share/general/IOutLink/"

// 原子化idx，和139share一致（多账号轮询、协程安全）
var idx int32 = 0

// httpPost 封装POST请求（保留auth参数，鉴权开关）
func (y *Yun139GroupLink) httpPost(pathname string, data interface{}, auth bool) ([]byte, error) {
	u := apiBase + pathname
	req := base.RestyClient.R()

	// 设置请求头（匹配接口要求）
	req.SetHeaders(map[string]string{
		"Content-Type":     "application/json;charset=utf-8",
		"Referer":          "https://yun.139.com/",
		"User-Agent":       "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0",
		"Origin":           "https://yun.139.com",
		"x-share-channel":  "0102",
		"hcy-cool-flag":    "1",
		"x-deviceinfo":     "||3|12.27.0|chrome|131.0.0.0|5c7c68368f048245e1ce47f1c0f8f2d0||windows 10|1536X695|zh-CN|||",
	})

	// 鉴权逻辑：完全参照139share（仅auth=true时添加）
	if auth {
		driver := op.GetFirstDriver("139Yun", int(atomic.LoadInt32(&idx)%int32(op.GetDriverCount("139Yun"))))
		if driver != nil {
			yun139 := driver.(*_139.Yun139)
			req.SetHeader("Authorization", "Basic "+yun139.Authorization)
		} else {
			log.Warn("未找到139Yun驱动，无法添加Authorization鉴权头")
		}
	}

	// 序列化请求体
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req.SetBody(jsonData)

	// 执行请求
	res, err := req.Execute(http.MethodPost, u)
	if err != nil {
		log.Warnf("HTTP请求失败: %v", err)
		return nil, err
	}

	return res.Body(), nil
}

// getShareInfo 调用getOutLinkInfo接口获取分享信息【无鉴权，和139share一致】
func (y *Yun139GroupLink) getShareInfo(pCaID string, page int) (GetOutLinkInfoResp, error) {
	var resp GetOutLinkInfoResp
	size := 200 // 每页条数
	start := page*size + 1
	end := (page + 1) * size

	// 构造请求体
	reqBody := GetOutLinkInfoReq{
		LinkID: y.ShareId,
		Passwd: y.SharePwd,
		PCaID:  pCaID,
		BNum:   start,
		ENum:   end,
	}

	// 调用接口：无鉴权（false），和139share一致
	body, err := y.httpPost("getOutLinkInfo", reqBody, false)
	if err != nil {
		return resp, err
	}

	// 解析响应
	if err := json.Unmarshal(body, &resp); err != nil {
		log.Warnf("响应解析失败: %v, body: %s", err, string(body))
		return resp, err
	}

	// 校验响应码
	if !resp.Success || resp.Code != "0000" {
		return resp, errors.New(resp.Message)
	}

	return resp, nil
}

// list 获取分享文件列表（分页）
func (y *Yun139GroupLink) list(pCaID string) ([]File, error) {
	actualID := pCaID
	if pCaID == "" || pCaID == "root" {
		actualID = "" // 根目录传空
	}

	files := make([]File, 0)
	page := 0

	for {
		// 调用接口获取分页数据
		res, err := y.getShareInfo(actualID, page)
		if err != nil {
			return nil, err
		}

		// 解析资产列表为File
		for _, asset := range res.Data.AssetsList {
			file := fileToObj(asset)
			files = append(files, file)
		}

		// 无下一页则终止（nextPageCursor为空）
		if res.Data.NextPageCursor == nil || res.Data.NextPageCursor == "" {
			break
		}
		page++
	}

	log.Debugf("获取到%d个文件", len(files))
	return files, nil
}

