package usecase

import (
	"context"
	"demo/config"
	"demo/pkg/log"
	"demo/pkg/store"
	"io"
	"mime/multipart"

	"github.com/minio/minio-go/v7"
)

// LlmUsecase 提供大语言模型服务
type FileUsecase struct {
	l      *log.Logger
	config *config.Config
	minio  *store.Minio
}

func NewFileUsecase(l *log.Logger, c *config.Config, minio *store.Minio) *FileUsecase {
	return &FileUsecase{
		l:      l,
		config: c,
		minio:  minio,
	}
}
func (u *FileUsecase) UploadFile(ctx context.Context, file *multipart.FileHeader) (string, error) {
	// 上传文件到OSS
	// 上传成功后返回文件URL
	f, err := file.Open()
	if err != nil {
		u.l.Logger.Error("open file failed", log.Error(err))
		return "", err
	}
	info, err := u.minio.Client.PutObject(ctx, u.config.Oss.BucketName, file.Filename, f, file.Size, minio.PutObjectOptions{})
	if err != nil {
		u.l.Logger.Error("upload file failed", log.Error(err))
		return "", err
	}
	return info.Key, nil
}

func (u *FileUsecase) UploadFileWithWriter(ctx context.Context, name string, writer io.Reader, size int64) (string, error) {
	// 上传文件到OSS
	// 上传成功后返回文件URL
	info, err := u.minio.Client.PutObject(ctx, u.config.Oss.BucketName, name, writer, size, minio.PutObjectOptions{})
	if err != nil {
		u.l.Logger.Error("upload file failed", log.Error(err))
		return "", err
	}
	return info.Key, nil
}
