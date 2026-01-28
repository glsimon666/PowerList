package _139_grouplink

import (
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
)

// Addition 驱动附加配置（必填参数）
type Addition struct {
	ShareId  string `json:"shareId" required:"true"`  // 分组分享ID（linkId）
	SharePwd string `json:"sharePwd"`                 // 分享密码（passwd）
	RootID   string `json:"rootId" default:"root"`    // 根目录ID
}

// 驱动固定配置
var config = driver.Config{
	Name:        "139GroupLink",       // 驱动名称
	OnlyProxy:   true,                 // 仅代理模式
	NoUpload:    true,                 // 禁止上传
	NoOverwriteUpload: true,           // 禁止覆盖上传
	DefaultRoot: "root",               // 默认根目录
}

// Yun139GroupLink 驱动核心结构体
type Yun139GroupLink struct {
	Storage model.Storage
	Addition
}

// GetAddition 返回驱动附加配置
func (d *Yun139GroupLink) GetAddition() driver.Additional {
	return &d.Addition
}

// Config 返回驱动配置
func (d *Yun139GroupLink) Config() driver.Config {
	return config
}

// GetStorage 返回*model.Storage（指针类型），匹配driver.Driver接口要求
func (d *Yun139GroupLink) GetStorage() *model.Storage {
	return &d.Storage
}

// SetStorage 参数改为值类型model.Storage（接口要求），直接赋值即可
func (d *Yun139GroupLink) SetStorage(s model.Storage) {
	d.Storage = s
}

// 新增：实现IRootId根目录接口（框架挂载必需），返回配置的RootID（默认root）
func (d *Yun139GroupLink) GetRootId() string {
	return d.RootID
}

// 初始化注册驱动
func init() {
	op.RegisterDriver(func() driver.Driver {
		return &Yun139GroupLink{}
	})
}
