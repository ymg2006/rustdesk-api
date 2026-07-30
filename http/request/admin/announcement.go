package admin

type AnnouncementCreateReq struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
	Status  *int   `json:"status"`
}

type AnnouncementUpdateReq struct {
	ID      uint   `json:"id" binding:"required"`
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
	Status  int    `json:"status"`
}
