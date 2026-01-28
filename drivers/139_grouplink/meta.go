package _139_grouplink

import (
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
)

// 原有代码完全保留（Addition/Config/结构体等）
type Addition struct {
	ShareId  string `json:"shareId" required:"true"`
	SharePwd string `json:"sharePwd" required:"true"`
	RootID   string `json:"rootId" default:"root"`
}

var config = driver.Config{
	Name:        "139GroupLink",
	OnlyProxy:   true,
	NoUpload:    true,
	NoOverwriteUpload: true,
	DefaultRoot: "root",
}

type Yun139GroupLink struct {
	Storage model.Storage
	Addition
	UserDomainId string // 已保存的分享者用户ID
}

// 原有方法完全保留
func (d *Yun139GroupLink) GetAddition() driver.Additional { return &d.Addition }
func (d *Yun139GroupLink) Config() driver.Config { return config }
func (d *Yun139GroupLink) GetStorage() *model.Storage { return &d.Storage }
func (d *Yun139GroupLink) SetStorage(s model.Storage) { d.Storage = s }
func (d *Yun139GroupLink) GetRootId() string { return d.RootID }

// ---------------------- 新增核心方法：实现框架GetDownloadUrl接口 ----------------------
// 方法签名必须严格匹配框架要求，框架下载时会自动调用该方法
func (d *Yun139GroupLink) GetDownloadUrl(fileId string) (string, error) {
	// 内部直接复用我们已写好的专属下载函数，逻辑完全保留
	return d.getDownloadUrl(fileId)
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &Yun139GroupLink{}
	})
}
