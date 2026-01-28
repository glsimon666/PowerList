package _139_grouplink

import (
	"time"

	"github.com/OpenListTeam/OpenList/v4/pkg/utils" // 新增：导入utils包，适配HashInfo类型
)

// GetOutLinkInfoReq 接口请求体
type GetOutLinkInfoReq struct {
	LinkID  string `json:"linkId"`  // 分享链接ID（如2sNZX0hQL8q9m）
	Passwd  string `json:"passwd"`  // 分享密码（如qmo9）
	PCaID   string `json:"pCaID"`   // 目录ID，根目录为空或root
	BNum    int    `json:"bNum"`    // 起始条数
	ENum    int    `json:"eNum"`    // 结束条数
}

// GetOutLinkInfoResp 接口响应体
type GetOutLinkInfoResp struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		NodNum         interface{} `json:"nodNum"`
		AssetsList     []Assets    `json:"assetsList"` // 文件/资源列表
		IsCreator      string      `json:"isCreator"`
		OutLink        OutLink     `json:"outLink"`
		NextPageCursor interface{} `json:"nextPageCursor"` // 分页游标
		PcaId          string      `json:"pcaId"`
	} `json:"data"`
}

// Assets 资源项（文件）
type Assets struct {
	AssetsId      string      `json:"assetsId"`   // 文件ID
	AssetsName    string      `json:"assetsName"` // 文件名
	Category      int         `json:"category"`
	CoType        int         `json:"coType"`
	CoSuffix      string      `json:"coSuffix"`   // 文件后缀
	CoSize        int64       `json:"coSize"`     // 文件大小（字节）
	UdTime        string      `json:"udTime"`     // 更新时间（20250203164249）
	ThumbnailURL  string      `json:"thumbnailURL"`
	BthumbnailURL string      `json:"bthumbnailURL"`
	PresentURL    string      `json:"presentURL"` // 播放/下载链接
	Path          string      `json:"path"`       // 文件路径
	IsDir         bool        `json:"-"`          // 是否为文件夹（139分组链接暂未返回文件夹，默认false）
	Time          time.Time   `json:"-"`          // 解析后的时间
}

// OutLink 分享链接基础信息
type OutLink struct {
	LinkId     string `json:"linkId"`
	LinkCode   string `json:"linkCode"`
	ChannelId  string `json:"channelId"`
	Passwd     string `json:"passwd"`
	Url        string `json:"url"`
	LkName     string `json:"lkName"`
	CtTime     string `json:"ctTime"`
	LastUdTime string `json:"lastUdTime"`
}

// File 适配PowerList model.Obj的文件结构体
type File struct {
	Name      string
	Path      string
	Size      int64
	ID        string
	IsDirFlag bool // 解决与IsDir()方法的命名冲突
	Time      time.Time
	URL       string // 播放/下载链接
}

// 实现model.Obj接口的必要方法
func (f File) GetID() string {
	return f.ID
}

func (f File) GetName() string {
	return f.Name
}

func (f File) GetSize() int64 {
	return f.Size
}

func (f File) GetPath() string {
	return f.Path
}

func (f File) IsDir() bool {
	return f.IsDirFlag
}

func (f File) ModTime() time.Time {
	return f.Time
}

func (f File) CreateTime() time.Time {
	return f.Time
}

// 修改：返回utils.HashInfo结构体（而非string），139grouplink无hash，返回空结构体兜底
func (f File) GetHash() utils.HashInfo {
	return utils.HashInfo{}
}

// fileToObj 转换为PowerList标准model.Obj
func fileToObj(src Assets) File {
	parsedTime, _ := time.Parse("20060102150405", src.UdTime)
	return File{
		ID:        src.AssetsId,
		Name:      src.AssetsName,
		Size:      src.CoSize,
		Path:      src.Path,
		IsDirFlag: false,
		Time:      parsedTime,
		URL:       src.PresentURL,
	}
}
