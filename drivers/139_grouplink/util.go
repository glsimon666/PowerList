package _139_grouplink

import (
	"encoding/json"
	"errors"
	"sync/atomic" // 新增：原子操作，和139share一致
	log "github.com/sirupsen/logrus"
	_139 "github.com/OpenListTeam/OpenList/v4/drivers/139" // 139Yun驱动依赖
	"github.com/OpenListTeam/OpenList/v4/drivers/base"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"net/http"
	"time"
)

// 接口基础地址
const apiBase = "https://share-kd-njs.yun.139.com/yun-share/general/IOutLink/"

// 新增：原子化idx，和139share完全一致（多账号轮询、协程安全）
var idx int32 = 0

// httpPost 封装POST请求（保留auth参数，鉴权开关）
func (y *Yun139GroupLink) httpPost(pathname string, data interface{}, auth bool) ([]byte, error) {
	u := apiBase + pathname
	req := base.RestyClient.R()

	// 固定请求头（匹配139云盘接口要求）
	req.SetHeaders(map[string]string{
		"Content-Type":    "application/json;charset=utf-8",
		"Referer":         "https://yun.139.com/",
		"User-Agent":      "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0",
		"Origin":          "https://yun.139.com",
		"x-share-channel": "0102",
	})

	// 鉴权逻辑：完全参照139share（仅auth=true时添加）
	if auth {
		driver := op.GetFirstDriver("139Yun", int(atomic.LoadInt32(&idx)%int32(op.GetDriverCount("139Yun"))))
		if driver != nil {
			yun139 := driver.(*_139.Yun139)
			req.SetHeader("Authorization", "Basic "+yun139.Authorization)
			log.Debugf("已为请求添加139Yun鉴权头，账号：%s", yun139.Account)
		} else {
			log.Warn("未找到配置的139Yun账号，无法添加鉴权头，将无法获取高速下载链接")
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
		log.Warnf("HTTP请求失败: %v, url: %s", err, u)
		return nil, err
	}

	return res.Body(), nil
}

// getShareInfo 调用getOutLinkInfo接口获取分享信息【无鉴权，和139share一致】
func (y *Yun139GroupLink) getShareInfo(pCaID string, page int) (GetOutLinkInfoResp, error) {
	var resp GetOutLinkInfoResp
	size := 200 // 每页条数，和139share一致
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

	// 调用接口：最后一个参数设为false【无鉴权，和139share完全一致】
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

// list 获取分享文件列表（分页）【无鉴权，基础能力】
func (y *Yun139GroupLink) list(pCaID string) ([]File, error) {
	actualID := pCaID
	if pCaID == "" || pCaID == "root" {
		actualID = "" // 根目录传空
	}

	files := make([]File, 0)
	page := 0

	for {
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

	log.Debugf("获取到%d个分享文件（无鉴权）", len(files))
	return files, nil
}

