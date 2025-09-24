package store

import (
	"demo/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioStore struct {
	DB *minio.Client
}

func NewMinioStore(c *config.Config) *MinioStore {
	client, err := minio.New("minio:9000", &minio.Options{
		Creds:  credentials.NewStaticV4(c.Minio.AccessKey, c.Minio.SecretKey, ""),
		Secure: true,
	})
	if err != nil {
		panic(err)
	}

	return &MinioStore{
		DB: client,
	}
}
