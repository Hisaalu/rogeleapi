package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	"github.com/Hisaalu/rogele/internal/domain"
)

const (
	PresignTTL     = 15 * time.Minute
	MaxFileSize    = 5 * 1024 * 1024 // 5 MB
	MaxPathSegment = 100
)

var allowedMIME = map[string]struct{}{
	// documents
	"application/pdf":    {},
	"application/msword": {},
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   {},
	"application/vnd.ms-powerpoint":                                             {},
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": {},
	"application/vnd.ms-excel":                                                  {},
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         {},
	"text/plain": {},
	"text/csv":   {},
	// images
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
	"image/gif":  {},
	// video
	"video/mp4":       {},
	"video/quicktime": {},
	"video/webm":      {},
	// audio
	"audio/mpeg":  {},
	"audio/mp4":   {},
	"audio/x-m4a": {},
}

func IsAllowedMIME(contentType string) bool {
	_, ok := allowedMIME[strings.ToLower(contentType)]
	return ok
}

func (c *R2Client) PresignUpload(ctx context.Context, in PresignRequest) (*PresignResult, error) {
	if in.FileName == "" {
		return nil, fmt.Errorf("%w: file name is required", domain.ErrInvalidInput)
	}
	if in.Size <= 0 {
		return nil, fmt.Errorf("%w: file size must be greater than zero", domain.ErrInvalidInput)
	}
	if in.Size > MaxFileSize {
		return nil, fmt.Errorf("%w: file exceeds the %d MB limit", domain.ErrInvalidInput, MaxFileSize/1024/1024)
	}
	if in.ContentType == "" {
		return nil, fmt.Errorf("%w: content type is required", domain.ErrInvalidInput)
	}
	if !IsAllowedMIME(in.ContentType) {
		return nil, fmt.Errorf("%w: file type %s is not allowed", domain.ErrInvalidInput, in.ContentType)
	}

	prefix := sanitizeSegment(in.Prefix)
	if prefix == "" {
		prefix = "uploads"
	}
	ext := strings.ToLower(filepath.Ext(in.FileName))
	key := fmt.Sprintf("%s/%s%s", prefix, uuid.NewString(), ext)

	presigned, err := c.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(in.ContentType),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = PresignTTL
	})
	if err != nil {
		return nil, fmt.Errorf("presign: %w", err)
	}

	return &PresignResult{
		UploadURL: presigned.URL,
		FileKey:   key,
		PublicURL: c.PublicURL(key),
		ExpiresIn: int(PresignTTL.Seconds()),
	}, nil
}

func sanitizeSegment(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > MaxPathSegment {
		s = s[:MaxPathSegment]
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		}
	}
	return b.String()
}
