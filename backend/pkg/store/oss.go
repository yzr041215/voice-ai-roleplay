package store

import (
	"context"
	"demo/config"
	"fmt"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Minio struct {
	Client *minio.Client
	// 如果多处用到 bucket 可再存一个默认 bucket
}

func NewMinioStore(c *config.Config) *Minio {
	// 初始化 MinIO 客户端
	client, err := minio.New(c.Oss.EndPoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.Oss.AccessKey, c.Oss.SecretKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("minio new client: %v", err)
	}

	// 可选：确保 bucket 存在（不存在就创建）
	ctx := context.Background()
	bucketName := c.Oss.BucketName // 与业务对齐
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		log.Fatalf("bucket check: %v", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			log.Fatalf("make bucket: %v", err)
		}
		log.Printf("bucket %s created\n", bucketName)
	}
	publicReadPolicy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
		  {
			"Effect": "Allow",
			"Principal": {"AWS": "*"},
			"Action": ["s3:GetObject"],
			"Resource": ["arn:aws:s3:::%s/*"]
		  }
		]
	  }`, bucketName)
	if err := client.SetBucketPolicy(ctx, bucketName, publicReadPolicy); err != nil {
		log.Fatalf("set policy: %v", err)
	}
	log.Println("mybucket is now public-read")

	return &Minio{Client: client}
}
