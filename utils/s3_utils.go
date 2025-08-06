package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// GetPublicAvatarURL 获取公有头像 URL
func GetPublicAvatarURL(filename string) (string, error) {
	avatarURL := fmt.Sprintf("%s/%s/%s", config.GConf.S3Config.Endpoint, config.GConf.S3Config.Bucket, filename)
	return avatarURL, nil
}

// GetPresignedImageURL 获取预签名图片 URL
func GetPresignedImageURL(filename string) (string, error) {
	cfg, err := awsConfig.LoadDefaultConfig(
		context.Background(),
		awsConfig.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     config.GConf.S3Config.AccessKeyID,
				SecretAccessKey: config.GConf.S3Config.SecretAccessKey,
			},
		}),
		awsConfig.WithRegion(config.GConf.S3Config.Region),
	)
	if err != nil {
		return "", fmt.Errorf("load aws config failed: %w", err)
	}

	endpoint := config.GConf.S3Config.Endpoint

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = &endpoint
	})
	presignClient := s3.NewPresignClient(s3Client)

	req, err := presignClient.PresignGetObject(
		context.Background(),
		&s3.GetObjectInput{
			Bucket: aws.String(config.GConf.S3Config.Bucket),
			Key:    aws.String(filename),
		},
		s3.WithPresignExpires(time.Duration(config.GConf.S3Config.PresignExpire)*time.Second), //URL过期时间
	)
	if err != nil {
		return "", fmt.Errorf("presign get object failed: %w", err)
	}

	// 替换域名
	url := req.URL
	internalURL := fmt.Sprintf("http://%s.%s", config.GConf.S3Config.Bucket, strings.TrimPrefix(config.GConf.S3Config.Endpoint, "http://"))
	url = strings.Replace(url, internalURL, config.GConf.S3Config.Domain, 1)

	url = strings.Replace(url, "https://", "http://", 1)

	return url, nil
}

// ListAvatarFiles 获取所有以avatar开头的文件列表
func ListAvatarFiles() ([]string, error) {
	cfg, err := awsConfig.LoadDefaultConfig(
		context.Background(),
		awsConfig.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     config.GConf.S3Config.AccessKeyID,
				SecretAccessKey: config.GConf.S3Config.SecretAccessKey,
			},
		}),
		awsConfig.WithRegion(config.GConf.S3Config.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config failed: %w", err)
	}

	endpoint := config.GConf.S3Config.Endpoint

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = &endpoint
	})

	var avatarFiles []string
	var continuationToken *string

	for {
		input := &s3.ListObjectsV2Input{
			Bucket: aws.String(config.GConf.S3Config.Bucket),
			Prefix: aws.String("avatar"), // 只列出以avatar开头的文件
		}
		if continuationToken != nil {
			input.ContinuationToken = continuationToken
		}

		result, err := s3Client.ListObjectsV2(context.Background(), input)
		if err != nil {
			return nil, fmt.Errorf("list objects failed: %w", err)
		}

		for _, object := range result.Contents {
			if object.Key != nil {
				avatarFiles = append(avatarFiles, *object.Key)
			}
		}

		if !*result.IsTruncated {
			break
		}
		continuationToken = result.NextContinuationToken
	}

	return avatarFiles, nil
}

// GetAvatarURLs 获取所有头像的URL列表
func GetAvatarURLs() ([]string, error) {
	avatarFiles, err := ListAvatarFiles()
	if err != nil {
		return nil, err
	}

	var avatarURLs []string
	for _, filename := range avatarFiles {
		// 使用预签名URL以支持私有空间访问
		url, err := GetPresignedImageURL(filename)
		if err != nil {
			continue // 跳过有问题的文件
		}
		avatarURLs = append(avatarURLs, url)
	}

	return avatarURLs, nil
}

// GetBackgroundFilenameFromAvatarFilename 根据头像文件名构造对应的背景文件名
func GetBackgroundFilenameFromAvatarFilename(avatarFilename string) string {
	return strings.Replace(avatarFilename, "avatar_", "bg_", 1)
}
