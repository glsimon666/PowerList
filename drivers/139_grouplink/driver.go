package _139_grouplink

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic" // 新增：原子操作
	_139 "github.com/OpenListTeam/OpenList/v4/drivers/139"
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
	log "github.com/sirupsen/logrus"
	"time"
)

// 确保实现Driver接口
var _ driver.Driver = (*Yun139GroupLink)(nil)

// Init 初始化驱动（空实现）
func (d *Yun139GroupLink) Init(ctx context.Context) error {
	return nil
}

// Drop 销毁驱动（空实现）
func (d *Yun139GroupLink) Drop(ctx context.Context) error {
	return nil
}

// List 列出分享文件【无鉴权，和139share一致】
func (d *Yun139GroupLink) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	files, err := d.list(dir.GetID())
	if err != nil {
		log.Warnf("获取分享文件列表失败: %v", err)
		return nil, err
	}

	// 转换为PowerList标准model.Obj
	return utils.SliceConvert(files, func(src File) (model.Obj, error) {
		return src, nil
	})
}

// Link 获取文件高速下载直链【核心鉴权逻辑，完全参照139share】
func (d *Yun139GroupLink) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	// 转换文件对象
	f, ok := file.(File)
	if !ok {
		return nil, errors.New("文件格式错误，无法获取直链")
	}

	// 先尝试复用接口返回的PresentURL（无鉴权兜底）
	if f.URL != "" {
		log.Debugf("使用接口原生PresentURL，文件：%s", f.Name)
		exp := 15 * time.Minute
		return &model.Link{
			URL:         f.URL,
			Expiration:  &exp,
			Concurrency: 5,
			PartSize:    10 * utils.MB,
		}, nil
	}

	// 核心：多账号轮询重试（完全参照139share）
	count := op.GetDriverCount("139Yun")
	if count == 0 {
		return nil, errors.New("未配置139Yun账号，无法获取高速下载链接，请先配置139Yun驱动")
	}

	// 轮询所有139Yun账号，直到获取直链成功
	var lastErr error
	for i := 0; i < count; i++ {
		link, err := d.myLink(ctx, f)
		if err == nil {
			return link, nil
		}
		lastErr = err
		atomic.AddInt32(&idx, 1) // 原子自增，轮询下一个账号
		log.Warnf("当前139Yun账号获取直链失败，重试下一个：%v", err)
	}

	return nil, fmt.Errorf("所有%d个139Yun账号均获取直链失败，最后错误：%v", count, lastErr)
}

// myLink 单账号尝试获取高速下载直链【鉴权核心，参照139share】
func (d *Yun139GroupLink) myLink(ctx context.Context, f File) (*model.Link, error) {
	// 1. 获取当前轮询的139Yun账号（和139share一致）
	driverIdx := int(atomic.LoadInt32(&idx) % int32(op.GetDriverCount("139Yun")))
	storage := op.GetFirstDriver("139Yun", driverIdx)
	if storage == nil {
		return nil, errors.New("未找到139Yun账号")
	}
	yun139 := storage.(*_139.Yun139)
	log.Infof("[139GroupLink] 使用139Yun账号[%d]获取高速直链：%s（ID：%s）", driverIdx, f.Name, f.ID)

	// 2. 构造139云盘下载接口请求体（参照139share的dlFromOutLinkV3）
	type dlReq struct {
		LinkID string   `json:"linkID"`  // 分组分享ID
		CoIDLst []string `json:"coIDLst"` // 文件ID列表
	}
	reqBody := dlReq{
		LinkID: d.ShareId,
		CoIDLst: []string{f.ID},
	}

	// 3. 调用下载接口：auth=true 开启鉴权（高速下载核心）
	respBody, err := d.httpPost("dlFromOutLinkV3", reqBody, true)
	if err != nil {
		return nil, fmt.Errorf("下载接口请求失败：%v", err)
	}

	// 4. 解析下载接口响应（适配139云盘返回格式，和139share一致）
	var dlResp struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			DownloadURL string `json:"downloadURL"` // 高速下载直链
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &dlResp); err != nil {
		return nil, fmt.Errorf("下载响应解析失败：%v，body：%s", err, string(respBody))
	}
	if !dlResp.Success || dlResp.Code != "0000" {
		return nil, fmt.Errorf("下载接口返回错误：%s（%s）", dlResp.Message, dlResp.Code)
	}
	if dlResp.Data.DownloadURL == "" {
		return nil, errors.New("下载接口未返回有效直链")
	}

	// 5. 组装PowerList标准Link对象（复用139Yun账号的并发/分块配置，高速下载）
	exp := 15 * time.Minute // 直链过期时间，和139share一致
	return &model.Link{
		URL:         dlResp.Data.DownloadURL + fmt.Sprintf("#storageId=%d", yun139.ID),
		Expiration:  &exp,
		Concurrency: yun139.Concurrency, // 复用139Yun的并发数
		PartSize:    yun139.ChunkSize,   // 复用139Yun的分块大小
	}, nil
}

