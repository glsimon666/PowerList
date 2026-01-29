package _139_grouplink

import (
	"context"
	"encoding/json" // 新增：解决undefined: json
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

// List 列出分享文件（实现Driver接口）
func (d *Yun139GroupLink) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	// 获取文件列表
	files, err := d.list(dir.GetID())
	if err != nil {
		log.Warnf("获取文件列表失败: %v", err)
		return nil, err
	}

	// 转换为PowerList标准model.Obj
	return utils.SliceConvert(files, func(src File) (model.Obj, error) {
		return src, nil
	})
}

// Link 获取文件直链（模仿139share的多账号重试逻辑）
func (d *Yun139GroupLink) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	// 转换文件对象（修复后可正常断言）
	f, ok := file.(File)
	if !ok {
		return nil, errors.New("文件格式错误")
	}

	// 优先使用资产自带的PresentURL
	if f.URL != "" {
		exp := 15 * time.Minute // 直链过期时间（默认15分钟）
		return &model.Link{
			URL:         f.URL,
			Expiration:  &exp,
			Concurrency: 5, // 默认并发数
			PartSize:    10 * utils.MB, // 默认分块大小
		}, nil
	}

	// 备用逻辑：复用139云盘账号获取直链（同139share）
	count := op.GetDriverCount("139Yun")
	if count == 0 {
		return nil, errors.New("未配置139Yun账号，无法获取高速下载链接")
	}

	var lastErr error
	// 轮询所有139Yun账号重试
	for i := 0; i < count; i++ {
		link, err := d.myLink(ctx, file)
		if err == nil {
			return link, nil
		}
		lastErr = err
		atomic.AddInt32(&idx, 1) // 原子自增，轮询下一个账号
	}
	return nil, fmt.Errorf("所有%d个139Yun账号均获取直链失败：%v", count, lastErr)
}

// myLink 复用139Yun驱动获取直链
func (d *Yun139GroupLink) myLink(ctx context.Context, file model.Obj) (*model.Link, error) {
	driverIdx := int(atomic.LoadInt32(&idx) % int32(op.GetDriverCount("139Yun")))
	storage := op.GetFirstDriver("139Yun", driverIdx)
	if storage == nil {
		return nil, errors.New("找不到139云盘账号")
	}

	yun139 := storage.(*_139.Yun139)
	log.Infof("[%v] 获取139分组链接文件直链 %v %v", yun139.ID, file.GetName(), file.GetID())

	// 调用139Yun驱动的直链逻辑
	url, err := yun139.GetDownloadUrl(ctx, file.GetID())
	if err != nil {
		return nil, err
	}

	exp := 15 * time.Minute
	return &model.Link{
		URL:         url + fmt.Sprintf("#storageId=%d", yun139.ID),
		Expiration:  &exp,
		Concurrency: yun139.Concurrency,
		PartSize:    yun139.ChunkSize,
	}, nil
}

