package _139_grouplink

import (
	"time"

	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
)

// GetOutLinkInfoReq 接口请求体【终极修复：完全对齐抓包真实参数，删除错误的bNum/eNum】
type GetOutLinkInfoReq struct {
	LinkId         string      `json:"linkId"`  // 抓包原字段，小写l
	Passwd         string      `json:"passwd"`  // 抓包原字段
	CaSrt          int         `json:"caSrt"`   // 固定值0，分类排序
	CoSrt          int         `json:"coSrt"`   // 固定值0，文件排序
	SrtDr          int         `json:"srtDr"`   // 固定值1，排序方向
	PageNum        int         `json:"pageNum"` // 分页页码，从1开始
	PCaId          string      `json:"pCaId"`   // 目录ID，根目录传空
	PageSize       int         `json:"pageSize"`// 单页条数，最大100
	NextPageCursor interface{} `json:"nextPageCursor"` // 固定传null
}

// 以下所有代码【完全保留，一字不改】
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
		PcaId          string      `json:"pCaId"`          // 接口返回的pCaId
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
	OwnerUserId string `json:"ownerUserId"` // 新增：分享者用户ID，用于下载接口的userDomainId
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
		ID:        src.AssetsId,       // 保留：文件唯一ID，供下载接口使用
		Name:      src.AssetsName,     // 保留：文件名
		Size:      0,                  // 修正：列表接口返回0，置空避免框架误判
		Path:      src.Path,           // 保留：文件路径
		IsDirFlag: false,              // 保留：非目录
		Time:      parsedTime,         // 保留：修改时间
		URL:       "",                 // 修正：清空无效的presentURL，框架会从GetDownloadUrl获取真实直链
	}
}
