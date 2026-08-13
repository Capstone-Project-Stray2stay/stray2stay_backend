package storage

import (
	"context"
	"mime/multipart"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type CloudinaryUploader struct {
	cld *cloudinary.Cloudinary
}

func NewCloudinaryUploader(cloudName, apiKey, apiSecret string) (*CloudinaryUploader, error) {
	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, err
	}
	return &CloudinaryUploader{cld: cld}, nil
}

func (c *CloudinaryUploader) UploadImages(files []*multipart.FileHeader, folder string) ([]string, error) {
	urls := make([]string, 0, len(files))

	for _, fh := range files {
		file, err := fh.Open()
		if err != nil {
			return nil, err
		}

		result, err := c.cld.Upload.Upload(context.Background(), file, uploader.UploadParams{
			Folder: folder,
		})
		file.Close()

		if err != nil {
			return nil, err
		}

		urls = append(urls, result.SecureURL)
	}

	return urls, nil
}