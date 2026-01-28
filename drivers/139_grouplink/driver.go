package _139_grouplink

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	_139 "github.com/OpenListTeam/OpenList/v4/drivers/139"
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
	log "github.com/sirupsen/logrus"
	"time"
)

var _ driver.Driver = (*Yun139GroupLink)(nil)

// Init 初始化驱动（空实现）
func (d *Yun139GroupLink) Init(ctx context.Context) error {
	return nil
}

// Drop 销毁驱动（空实现）
func (d *Yun139GroupLink) Drop(ctx context.Context) error {
	return nil
}

// List 列出分享文件（实现Driver接口）
func (d *Yun139GroupLink) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	files, err := d.list(dir.GetID())
	if err != nil {
		log.Warnf("获取文件列表失败: %v", err)
		return nil, err
	}

	return utils.SliceConvert(files, func(src File) (model.Obj, error) {
		return src, nil
	})
}

// Link 获取文件直链（多账号轮询+优先PresentURL）
func (d *Yun139GroupLink) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	f, ok := file.(File)
	if !ok {
		return nil, errors.New("文件格式错误")
	}

	// 优先使用资产自带的PresentURL（无鉴权兜底）
	if f.URL != "" {
		exp := 15 * time.Minute
		return &model.Link{
			URL:         f.URL,
			Expiration:  &exp,
			Concurrency: 5,
			PartSize:    10 * utils.MB,
		}, nil
	}

	// 多账号轮询重试（原有逻辑不变）
	count := op.GetDriverCount("139Yun")
	if count == 0 {
		return nil, errors.New("未配置139Yun账号，无法获取高速下载链接")
	}

	var lastErr error
	for i := 0; i < count; i++ {
		link, err := d.myLink(ctx, f)
		if err == nil {
			return link, nil
		}
		lastErr = err
		atomic.AddInt32(&idx, 1)
	}
	return nil, fmt.Errorf("所有%d个139Yun账号均获取直链失败：%v", count, lastErr)
}

// myLink 单账号获取高速直链（核心修改：改用grouplink专属getDownloadUrl）
func (d *Yun139GroupLink) myLink(ctx context.Context, f File) (*model.Link, error) {
	// 仅保留账号日志，无需再获取139Yun实例（鉴权头由httpPost自动添加）
	driverIdx := int(atomic.LoadInt32(&idx) % int32(op.GetDriverCount("139Yun")))
	storage := op.GetFirstDriver("139Yun", driverIdx)
	if storage == nil {
		return nil, errors.New("找不到139云盘账号")
	}
	yun139 := storage.(*_139.Yun139)
	log.Infof("[139Yun-%d] 为grouplink文件获取高速直链：%s（ID：%s）", yun139.ID, f.Name, f.ID)

	// ---------------------- 核心修改：调用grouplink专属下载接口 ----------------------
	url, err := d.getDownloadUrl(f.ID)
	if err != nil {
		return nil, err
	}
	// ---------------------- 修改结束 ----------------------

	// 组装Link对象（原有逻辑不变，复用139Yun的并发/分块配置）
	exp := 15 * time.Minute
	return &model.Link{
		URL:         url + fmt.Sprintf("#storageId=%d", yun139.ID),
		Expiration:  &exp,
		Concurrency: yun139.Concurrency, // 复用139Yun账号的并发数
		PartSize:    yun139.ChunkSize,   // 复用139Yun账号的分块大小
	}, nil
}
