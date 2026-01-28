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

// ---------------------- grouplink专属下载接口 结构体【终极修复：完全对齐抓包】 ----------------------
// GetDownloadUrlReq 下载接口请求体：和抓包一字不差，3个必传参数
type GetDownloadUrlReq struct {
	UserDomainId string `json:"userDomainId"` // 分享者用户ID，探活获取
	LinkId       string `json:"linkId"`       // 分组链接ID，小写l，和抓包一致
	AssetsId     string `json:"assetsId"`     // 文件ID，和抓包一致，无嵌套
}
// GetDownloadUrlResp 下载接口响应体：核心解析downLoadUrl，和抓包一字不差
type GetDownloadUrlResp struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		DownLoadUrl   string `json:"downLoadUrl"`   // 核心高速直链，和抓包完全一致
		CdnDownLoadUrl string `json:"cdnDownLoadUrl"`// 备用CDN直链，可选解析
	} `json:"data"`
}
// ---------------------- 结构体修复结束 ----------------------

// httpPost 封装POST请求（保留auth参数，鉴权开关）【完全保留，一字不改】
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

// getDownloadUrl 专属下载接口【终极修复：对齐抓包请求/响应，保留所有正确逻辑】
func (y *Yun139GroupLink) getDownloadUrl(fid string) (string, error) {
	// 校验userDomainId是否有效（探活时已获取，为空则直接报错）
	if y.UserDomainId == "" {
		return "", errors.New("userDomainId未初始化，请先执行根目录探活")
	}
	// 构造抓包级合法请求体，3个参数完全对齐
	req := GetDownloadUrlReq{
		UserDomainId: y.UserDomainId, // 探活保存的分享者用户ID
		LinkId:       y.ShareId,      // 分组链接ID
		AssetsId:     fid,            // 单个文件ID，直接传入，无嵌套
	}

	// 原有请求逻辑完全保留（auth=true带鉴权，正确）
	respBody, err := y.httpPost("getDownloadUrl", req, true)
	if err != nil {
		return "", fmt.Errorf("下载接口请求失败：%v", err)
	}

	// 原有解析逻辑保留，仅修改字段名匹配
	var resp GetDownloadUrlResp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("下载响应解析失败：%v，body：%s", err, string(respBody))
	}

	// 原有接口错误判断逻辑保留（和列表接口一致，正确）
	if !resp.Success || resp.Code != "0000" {
		return "", fmt.Errorf("下载接口返回错误：%s（码：%s）", resp.Message, resp.Code)
	}

	// 核心：解析抓包的downLoadUrl，而非原错误的DownloadURL
	if resp.Data.DownLoadUrl == "" {
		return "", errors.New("下载接口未返回有效高速直链")
	}

	log.Debugf("grouplink专属接口获取高速直链成功：%s", resp.Data.DownLoadUrl)
	return resp.Data.DownLoadUrl, nil
}

// getShareInfo 调用getOutLinkInfo【完全保留上一轮修复后的代码，一字不改】
func (y *Yun139GroupLink) getShareInfo(pCaID string, page int) (GetOutLinkInfoResp, error) {
	var resp GetOutLinkInfoResp
	// 构造抓包的固定参数，完全对齐接口要求
	reqBody := GetOutLinkInfoReq{
		LinkId:         y.ShareId,
		Passwd:         y.SharePwd,
		CaSrt:          0,          // 抓包固定值
		CoSrt:          0,          // 抓包固定值
		SrtDr:          1,          // 抓包固定值
		PageNum:        page + 1,   // 框架page从0开始 → 接口PageNum从1开始
		PCaId:          pCaID,      // 目录ID，根目录传空
		PageSize:       100,        // 接口最大支持100
		NextPageCursor: nil,        // 抓包固定值null
	}

	// 以下请求逻辑【完全保留】
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

// list 获取文件列表【仅新增1行：保存userDomainId，其余完全保留上一轮修复代码】
func (y *Yun139GroupLink) list(pCaID string) ([]File, error) {
	var actualID string
	files := make([]File, 0)
	page := 0 // 初始值0不变

	if pCaID == "" || pCaID == "root" {
		// 根目录场景：传空PCaId探活，获取接口返回的真实根pCaId
		probeResp, err := y.getShareInfo("", 0)
		if err != nil {
			return nil, fmt.Errorf("根目录探活获取pCaId失败：%v", err)
		}
		// 校验接口返回的根pCaId是否有效
		if probeResp.Data.PcaId == "" {
			return nil, errors.New("接口未返回有效根目录pCaId")
		}
		actualID = probeResp.Data.PcaId // 用接口返回的真实pCaId作为后续查询的ID
		// ---------------------- 新增1行：保存userDomainId，从outLink.ownerUserId获取 ----------------------
		y.UserDomainId = probeResp.Data.OutLink.OwnerUserId
		// ----------------------------------------------------------------------------------------
		log.Debugf("根目录探活成功，获取真实pCaId：%s，userDomainId：%s", actualID, y.UserDomainId)

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

	// 非根目录分页查询 / 根目录后续页查询（通用逻辑，完全保留）
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
