package _139_share

import (
	"context"
	"errors"
	"fmt"
	_139 "github.com/OpenListTeam/OpenList/v4/drivers/139"
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	log "github.com/sirupsen/logrus"
	"time"
)

type Yun139Share struct {
	model.Storage
	Addition
}

// fileToObj 转换File到model.Obj（修正utils.ParseSize错误，直接使用int64大小）
func fileToObj(src File) model.Obj {
	// 移除utils.ParseSize，src.Size本身是int64，直接赋值
	modified, _ := time.Parse(time.RFC3339, src.Time)
	return &model.Object{
		ID:       src.ID,
		Name:     src.Name,
		Size:     src.Size, // 直接使用原始int64大小，无需解析
		Modified: modified,
		IsFolder: src.IsDir,
		Path:     src.Path,
	}
}

func (d *Yun139Share) Config() driver.Config {
	return config
}

func (d *Yun139Share) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *Yun139Share) Init(ctx context.Context) error {
	return nil
}

func (d *Yun139Share) Drop(ctx context.Context) error {
	return nil
}

func (d *Yun139Share) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	files, err := d.list(dir.GetID())
	if err != nil {
		log.Warnf("list files error: %v", err)
		return nil, err
	}
	return utils.SliceConvert(files, func(src File) (model.Obj, error) {
		return fileToObj(src), nil
	})
}

func (d *Yun139Share) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	count := op.GetDriverCount("139Yun")
	var err error
	for i := 0; i < count; i++ {
		link, err := d.myLink(ctx, file, args)
		if err == nil {
			return link, nil
		}
	}
	return nil, err
}

func (d *Yun139Share) myLink(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	storage := op.GetFirstDriver("139Yun", idx)
	idx++
	if storage == nil {
		return nil, errors.New("找不到移动云盘帐号")
	}
	yun139 := storage.(*_139.Yun139)
	log.Infof("[%v] 获取移动云盘文件直链 %v %v %v", yun139.ID, file.GetName(), file.GetID(), file.GetSize())
	url, err := d.link(yun139, file.GetID())
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

var _ driver.Driver = (*Yun139Share)(nil)

// 补充缺失的config变量（若meta.go中已定义，此处需确保引用正确）
var config = driver.Config{
	Name:              "Yun139Share",
	DefaultRoot:       "root",
	NoOverwriteUpload: true,
	NoUpload:          true,
}

// 补充缺失的utils.SliceConvert实现（若框架未提供，此处添加兼容实现）
func utilsSliceConvert[T any, R any](src []T, convert func(T) (R, error)) ([]R, error) {
	dst := make([]R, 0, len(src))
	for _, s := range src {
		r, err := convert(s)
		if err != nil {
			return nil, err
		}
		dst = append(dst, r)
	}
	return dst, nil
}

// 兼容框架的utils.SliceConvert调用
var utils = struct {
	SliceConvert func([]File, func(File) (model.Obj, error)) ([]model.Obj, error)
}{
	SliceConvert: func(src []File, convert func(File) (model.Obj, error)) ([]model.Obj, error) {
		return utilsSliceConvert(src, convert)
	},
}
