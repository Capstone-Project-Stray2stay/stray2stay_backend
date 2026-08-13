package port

import "mime/multipart"

type ImageUploader interface {
	UploadImages(files []*multipart.FileHeader, folder string) ([]string, error)
}