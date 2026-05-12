package image

import "time"

// Generated 表示系统已经生成并入库的一张图片。
type Generated struct {
	ID        int64
	AssetID   string
	ImageURL  string
	Provider  string
	Prompt    string
	CreatedAt time.Time
}

// Reference 表示用户上传或远端同步得到的参考素材图片。
type Reference struct {
	ID        int64
	AssetID   string
	ImageURL  string
	Name      string
	CreatedAt time.Time
}
