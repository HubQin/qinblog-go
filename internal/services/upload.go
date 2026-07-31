package services

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"

	"github.com/qin/qinblog/internal/config"
	"github.com/qin/qinblog/internal/support"
)

// Upload 图片上传服务（对应 App\Handlers\ImageUploader）
type uploadService struct{}

var Upload = &uploadService{}

var allowedImageExt = map[string]bool{"png": true, "jpg": true, "gif": true, "jpeg": true}

// SaveImage 保存上传图片，folder 为业务目录（如 posts），maxWidth>0 时限制最大宽度。
// 存储路径：{UploadPath}/images/{folder}/YYYYMM/时间戳_随机10.ext，返回 /storage/... 访问路径
func (s *uploadService) SaveImage(file *multipart.FileHeader, folder string, maxWidth int) (string, error) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Filename), "."))
	if ext == "" {
		ext = "png"
	}
	if !allowedImageExt[ext] {
		return "", errors.New("不支持的图片格式")
	}

	folderName := filepath.Join("images", folder, time.Now().Format("200601"))
	dir := filepath.Join(config.C.UploadPath, folderName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%d_%s.%s", time.Now().Unix(), support.RandomString(10), ext)
	fullPath := filepath.Join(dir, filename)

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return "", err
	}
	dst.Close()

	// gif 动图不做处理，其余图片限制最大宽度（等价 intervention/image 的 resize）
	if maxWidth > 0 && ext != "gif" {
		if img, err := imaging.Open(fullPath, imaging.AutoOrientation(true)); err == nil && img.Bounds().Dx() > maxWidth {
			resized := imaging.Resize(img, maxWidth, 0, imaging.Lanczos)
			_ = imaging.Save(resized, fullPath)
		}
	}

	// 等价 Storage::url()
	return "/storage/" + filepath.ToSlash(filepath.Join(folderName, filename)), nil
}
