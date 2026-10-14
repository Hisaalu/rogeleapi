package storage

type PresignRequest struct {
	FileName    string
	ContentType string
	Size        int64
	Prefix      string
}

type PresignResult struct {
	UploadURL string `json:"upload_url"`
	FileKey   string `json:"file_key"`
	PublicURL string `json:"public_url"`
	ExpiresIn int    `json:"expires_in"`
}
