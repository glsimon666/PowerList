package _139_share

// ListResp 适配getOutLinkInfoV6接口的响应结构
type ListResp struct {
	Code string `json:"code"`
	Desc string `json:"desc"`
	Data struct {
		Count   int `json:"count"`
		Next    string `json:"next"`
		Folders []struct {
			Name      string `json:"name"`
			Path      string `json:"path"`
			UpdatedAt string `json:"updatedAt"`
		} `json:"folders"`
		Files []File `json:"files"`
	} `json:"data"`
}

// File 文件/文件夹结构体
type File struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	UpdatedAt string `json:"updatedAt"`
	IsDir     bool   `json:"isDir"`
	Time      string `json:"-"` // 用于时间解析的临时字段
	ID        string `json:"id"` // 文件ID，用于获取下载链接
	Size      int64  `json:"size"`
}

// LinkResp 适配IOutLink/getDownloadUrl接口的响应结构
type LinkResp struct {
	Success bool     `json:"success"`
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Data    LinkData `json:"data"`
}

// LinkData 响应中data字段的结构
type LinkData struct {
	CdnDownLoadUrl       string `json:"cdnDownLoadUrl"`
	ContentHash          string `json:"contentHash"`
	ContentHashAlgorithm string `json:"contentHashAlgorithm"`
	DownLoadUrl          string `json:"downLoadUrl"`
}

// GetDownloadUrlReq 下载链接请求体结构
type GetDownloadUrlReq struct {
	LinkID             string `json:"linkID"`
	Account            string `json:"account"`
	CoIDLst            CoIDLst `json:"coIDLst"`
	CommonAccountInfo  CommonAccountInfo `json:"commonAccountInfo"`
}

// CoIDLst 请求体中的coIDLst字段
type CoIDLst struct {
	Item []string `json:"item"`
}

// CommonAccountInfo 请求体中的commonAccountInfo字段
type CommonAccountInfo struct {
	Account     string `json:"account"`
	AccountType int    `json:"accountType"`
}
