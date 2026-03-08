package controllers

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	common "gitlab.com/anti_metter/switching_common"
	a "gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	"gitlab.com/bot3342545/il-dashboard/services"
)

type WASendController struct {
}

func (o *WASendController) Init() {
	gocom.POST("/api/v1/instant-link/wa-send/message", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.sendMessage)
	gocom.POST("/api/v1/instant-link/wa-send/bulk", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.sendBulkMessage)
	gocom.POST("/api/v1/instant-link/wa-send/upload", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.uploadFile)
	gocom.GET("/api/v1/instant-link/wa-send/:fileType/:filename", o.getImage)
}

func (o *WASendController) sendMessage(ctx gocom.Context) error {
	req := dtos.WASendMessageReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetWASendSvc().SendMessage(req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *WASendController) sendBulkMessage(ctx gocom.Context) error {
	req := dtos.WASendBulkMessageReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetWASendSvc().SendBulkMessage(req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *WASendController) uploadFile(ctx gocom.Context) error {
	// Get file from form

	logger.Infof("[WASendController uploadFile] start")

	file, formErr := ctx.FormFile("file")
	if formErr != nil {
		logger.Errorf("[WASendController uploadFile] Failed to get file from form: %v", formErr)
		return ctx.SendError(common.ERR_INVALID_REQUEST)
	}
	if file == nil {
		logger.Info("[WASendController uploadFile] file not exists")
		return ctx.SendError(common.ERR_INVALID_REQUEST)
	}

	ret, err := services.GetWASendSvc().UploadFile(file, a.Get(ctx))
	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *WASendController) getImage(ctx gocom.Context) error {
	filename := ctx.Param("filename")
	fileType := ctx.Param("fileType")

	data, err := services.GetWASendSvc().GetFiles(fileType, filename, a.Get(ctx))
	if err != nil {
		return ctx.SendError(err)
	}

	// Determine content type based on file extension
	ext := filepath.Ext(filename)
	contentType := "application/octet-stream" // default
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".mp4":
		contentType = "video/mp4"
	case ".mov":
		contentType = "video/quicktime"
	case ".pdf":
		contentType = "application/pdf"
	}

	ctx.SetHeader("Content-Type", contentType)
	ctx.SetHeader("Cache-Control", "public, max-age=31536000") // Cache 1 year

	// Write to temp file in dir folder and serve directly
	// This avoids the issue with SendFileBytes creating temp files with random names
	dirPath := "./dir"
	os.MkdirAll(dirPath, 0755) // Ensure dir exists

	tempFilePath := filepath.Join(dirPath, filename)
	if err := os.WriteFile(tempFilePath, data, 0644); err != nil {
		logger.Errorf("[WASendController getImage] Failed to write temp file: %v", err)
		return ctx.SendError(common.ERR_INTERNAL_SERVER_ERROR)
	}

	// Clean up temp file after serving
	defer func() {
		if err := os.Remove(tempFilePath); err != nil {
			logger.Warnf("[WASendController getImage] Failed to remove temp file: %v", err)
		}
	}()

	return ctx.SendFile(tempFilePath, filename)
}

//---------------------------------------

var waSendController *WASendController
var waSendControllerOnce sync.Once

func GetWASendController() *WASendController {
	if waSendController == nil {
		waSendControllerOnce.Do(func() {
			waSendController = &WASendController{}
		})
	}

	return waSendController
}
