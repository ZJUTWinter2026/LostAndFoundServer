package user

import (
	"app/comm"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"
)

// UploadHandler API router注册点
func UploadHandler() gin.HandlerFunc {
	api := UploadApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfUpload).Pointer()).Name()] = api
	return hfUpload
}

type UploadApi struct {
	Info     struct{}          `name:"上传图片" desc:"上传图片"`
	Request  UploadApiRequest  // API请求参数 (Body/Header/Body/Body)
	Response UploadApiResponse // API响应数据 (Body中的Data部分)
}

type UploadApiRequest struct{}

type UploadApiResponse struct {
	Urls []string `json:"urls" desc:"图片访问地址"`
}

// Run Api业务逻辑执行点
func (u *UploadApi) Run(ctx *gin.Context) kit.Code {
	uploadDir, baseURL, maxSize := uploadConfig()

	// 限制上传大小: 默认10MB，不可关闭
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxSize)

	files, err := collectFiles(ctx)
	if err != nil {
		return comm.CodeParameterInvalid
	}

	if len(files) == 0 {
		return comm.CodeParameterInvalid
	}

	dateDir := time.Now().Format("20060102")
	saveDir := filepath.Join(uploadDir, dateDir)
	err = os.MkdirAll(saveDir, 0o755)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("创建上传目录失败")
		return comm.CodeServerError
	}

	urls := make([]string, 0, len(files))
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		name := uuid.NewString() + ext
		savePath := filepath.Join(saveDir, name)
		err = ctx.SaveUploadedFile(file, savePath)
		if err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Warn("保存上传文件失败")
			return comm.CodeServerError
		}
		// 拼接访问URL: baseURL + uploadDir + dateDir + name
		// 例如: http://127.0.0.1:8000/uploads/20231027/uuid.jpg
		fileURL := strings.TrimRight(baseURL, "/") + "/" + uploadDir + "/" + dateDir + "/" + name
		urls = append(urls, fileURL)
	}

	u.Response = UploadApiResponse{Urls: urls}
	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (u *UploadApi) Init(ctx *gin.Context) (err error) {
	return nil
}

// hfUpload API执行入口
func hfUpload(ctx *gin.Context) {
	api := &UploadApi{}
	err := api.Init(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("参数绑定校验错误")
		reply.Fail(ctx, comm.CodeParameterInvalid)
		return
	}
	code := api.Run(ctx)
	if !ctx.IsAborted() {
		if code == comm.CodeOK {
			reply.Success(ctx, api.Response)
		} else {
			reply.Fail(ctx, code)
		}
	}
}

func collectFiles(ctx *gin.Context) ([]*multipart.FileHeader, error) {
	form, err := ctx.MultipartForm()
	if err != nil {
		return nil, err
	}
	files := form.File["files"]
	if len(files) == 0 {
		if file, err := ctx.FormFile("file"); err == nil {
			files = append(files, file)
		}
	}
	return files, nil
}

func uploadConfig() (string, string, int64) {
	return comm.BizConf.Upload.Dir, comm.BizConf.Upload.BaseURL, comm.BizConf.Upload.MaxSizeMB * humanize.MiByte
}
