package _139_grouplink

import (
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
)

// GetOutLinkInfoReq 接口请求体【核心修复：PCaID的json tag】
type GetOutLinkInfoReq struct {
	LinkID  string `json:"linkId"`  // 正确：和接口一致（小写l）
	Passwd  string `json:"passwd"`  // 正确：和接口一致
	PCaID   string `json:"pCaId"`   // 致命修复：从pCaID→pCaId（最后一位d小写），和接口完全匹配
	BNum    int    `json:"bNum"`    // 正确：和接口一致
	ENum    int    `json:"eNum"`    // 正确：和接口一致
}

// 以下所有代码完全不变，保留原有实现
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
		PcaId          string      `json:"pCaId"`          // 接口返回的pCaId（和请求体一致）
	} `json:"data"`
}

type Assets struct {
	AssetsId      string      `json:"assetsId"`
	AssetsName    string      `json:"assetsName"`
	Category      int         `json:"category"`
	CoType        int         `json:"coType"`
	CoSuffix      string      `json:"coSuffix"`
	CoSize        int64       `json:"coSize"`
	UdTime        string      `json:"udTime"`
	ThumbnailURL  string      `json:"thumbnailURL"`
	BthumbnailURL string      `json:"bthumbnailURL"`
	PresentURL    string      `json:"presentURL"`
	Path          string      `json:"path"`
	IsDir         bool        `json:"-"`
	Time          time.Time   `json:"-"`
}

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

type File struct {
	Name      string
	Path      string
	Size      int64
	ID        string
	IsDirFlag bool
	Time      time.Time
	URL       string
}

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

func (f File) GetHash() utils.HashInfo {
	return utils.HashInfo{}
}

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
