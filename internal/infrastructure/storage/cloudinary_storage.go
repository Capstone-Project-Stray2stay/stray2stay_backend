package storage

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"strconv"
	"strings"

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

// DeleteImage removes the asset behind a Cloudinary secure URL, deriving its
// public ID from the URL since callers only ever persist the URL, not the ID.
func (c *CloudinaryUploader) DeleteImage(imageURL string) error {
	publicID, err := publicIDFromURL(imageURL)
	if err != nil {
		return err
	}

	result, err := c.cld.Upload.Destroy(context.Background(), uploader.DestroyParams{
		PublicID:     publicID,
		ResourceType: "image",
	})
	if err != nil {
		return err
	}

	// Destroy reports a mismatched/already-gone public ID as a 200 response
	// with Result "not found" rather than a Go error, so a bad extraction
	// above would otherwise look like a silent success.
	if result.Result != "ok" {
		return fmt.Errorf("cloudinary destroy for public id %q returned %q", publicID, result.Result)
	}
	return nil
}

// publicIDFromURL extracts the public ID (including any folder prefix) from a
// Cloudinary delivery URL, e.g. ".../upload/v1700000000/users/abc123.jpg" ->
// "users/abc123".
func publicIDFromURL(imageURL string) (string, error) {
	const marker = "/upload/"
	idx := strings.Index(imageURL, marker)
	if idx == -1 {
		return "", errors.New("not a cloudinary url")
	}

	rest := imageURL[idx+len(marker):]
	if slash := strings.Index(rest, "/"); slash != -1 {
		if version := rest[:slash]; len(version) > 1 && version[0] == 'v' {
			if _, err := strconv.Atoi(version[1:]); err == nil {
				rest = rest[slash+1:]
			}
		}
	}

	if dot := strings.LastIndex(rest, "."); dot != -1 {
		rest = rest[:dot]
	}

	if rest == "" {
		return "", errors.New("could not determine public id")
	}
	return rest, nil
}