package dto

type PresignUploadRequest struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Prefix      string `json:"prefix,omitempty"`
}

type PresignUploadResponse struct {
	UploadURL string `json:"upload_url"`
	FileKey   string `json:"file_key"`
	PublicURL string `json:"public_url"`
	ExpiresIn int    `json:"expires_in"`
}
