package admin

import (
	"app/comm"
	"app/dao/repo"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"
)

func ExportDataHandler() gin.HandlerFunc {
	api := ExportDataApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfExportData).Pointer()).Name()] = api
	return hfExportData
}

type ExportDataApi struct {
	Info     struct{} `name:"导出系统数据" desc:"导出系统数据到Excel文件"`
	Request  ExportDataApiRequest
	Response ExportDataApiResponse
}

type ExportDataApiRequest struct {
	Query struct{}
}

type ExportDataApiResponse struct {
	Url string `json:"url" desc:"Excel文件下载地址"`
}

func (e *ExportDataApi) Run(ctx *gin.Context) kit.Code {
	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	f := excelize.NewFile()
	defer f.Close()

	_ = f.DeleteSheet("Sheet1")

	if err := e.writeUserSheet(ctx, f); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("写入用户Sheet失败")
		return comm.CodeServerError
	}

	if err := e.writePostSheet(ctx, f); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("写入发布记录Sheet失败")
		return comm.CodeServerError
	}

	if err := e.writeClaimSheet(ctx, f); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("写入认领申请Sheet失败")
		return comm.CodeServerError
	}

	if err := e.writeFeedbackSheet(ctx, f); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("写入投诉反馈Sheet失败")
		return comm.CodeServerError
	}

	if err := e.writeAnnouncementSheet(ctx, f); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("写入公告通知Sheet失败")
		return comm.CodeServerError
	}

	if err := e.writeAuditLogSheet(ctx, f); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("写入审计日志Sheet失败")
		return comm.CodeServerError
	}

	if err := e.writeSystemConfigSheet(ctx, f); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("写入系统配置Sheet失败")
		return comm.CodeServerError
	}

	uploadDir := comm.BizConf.Upload.Dir
	exportDir := filepath.Join(uploadDir, "export")
	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("创建导出目录失败")
		return comm.CodeServerError
	}

	filename := "export_" + time.Now().Format("20060102150405") + "_" + uuid.NewString()[:8] + ".xlsx"
	filePath := filepath.Join(exportDir, filename)

	if err := f.SaveAs(filePath); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("保存Excel文件失败")
		return comm.CodeServerError
	}

	baseURL := comm.BizConf.Upload.BaseURL
	fileURL := baseURL + "/" + uploadDir + "/export/" + filename

	e.Response = ExportDataApiResponse{Url: fileURL}
	return comm.CodeOK
}

func (e *ExportDataApi) Init(ctx *gin.Context) error {
	return nil
}

func hfExportData(ctx *gin.Context) {
	api := &ExportDataApi{}
	err := api.Init(ctx)
	if err != nil {
		reply.Fail(ctx, comm.CodeParameterInvalid)
		return
	}
	code := api.Run(ctx)
	if code == comm.CodeOK {
		reply.Success(ctx, api.Response)
	} else {
		reply.Fail(ctx, code)
	}
}

func setCell(f *excelize.File, sheet string, col, row int, value interface{}) {
	cell, _ := excelize.CoordinatesToCellName(col, row)
	_ = f.SetCellValue(sheet, cell, value)
}

// safeFormatTime 安全格式化可空 *time.Time，nil 时返回空字符串
func safeFormatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func (e *ExportDataApi) writeUserSheet(ctx context.Context, f *excelize.File) error {
	sheet := "用户表"
	_, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}

	headers := []string{"ID", "用户名", "姓名", "身份证号", "用户类型", "校区", "首次登录", "禁用截止时间", "创建时间", "更新时间"}
	for i, header := range headers {
		setCell(f, sheet, i+1, 1, header)
	}

	urp := repo.NewUserRepo()
	users, err := urp.ListAll(ctx)
	if err != nil {
		return err
	}

	for i, user := range users {
		row := i + 2
		setCell(f, sheet, 1, row, user.ID)
		setCell(f, sheet, 2, row, user.Username)
		setCell(f, sheet, 3, row, user.Name)
		setCell(f, sheet, 4, row, user.IDCard)
		setCell(f, sheet, 5, row, user.Usertype)
		setCell(f, sheet, 6, row, user.Campus)
		setCell(f, sheet, 7, row, user.FirstLogin)
		setCell(f, sheet, 8, row, safeFormatTime(user.DisabledUntil))
		setCell(f, sheet, 9, row, user.CreatedAt.Format("2006-01-02 15:04:05"))
		setCell(f, sheet, 10, row, user.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func (e *ExportDataApi) writePostSheet(ctx context.Context, f *excelize.File) error {
	sheet := "发布记录"
	_, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}

	headers := []string{"ID", "发布者ID", "发布类型", "物品名称", "物品类型", "校区", "地点", "存放地点", "事件时间", "物品特征", "联系人", "联系电话", "是否有悬赏", "悬赏说明", "状态", "取消原因", "驳回原因", "认领人数", "归档处理方式", "处理时间", "创建时间", "更新时间"}
	for i, header := range headers {
		setCell(f, sheet, i+1, 1, header)
	}

	prp := repo.NewPostRepo()
	posts, err := prp.ListAll(ctx)
	if err != nil {
		return err
	}

	for i, post := range posts {
		row := i + 2
		setCell(f, sheet, 1, row, post.ID)
		setCell(f, sheet, 2, row, post.PublisherID)
		setCell(f, sheet, 3, row, post.PublishType)
		setCell(f, sheet, 4, row, post.ItemName)
		setCell(f, sheet, 5, row, post.ItemType)
		setCell(f, sheet, 6, row, post.Campus)
		setCell(f, sheet, 7, row, post.Location)
		setCell(f, sheet, 8, row, post.StorageLocation)
		setCell(f, sheet, 9, row, post.EventTime.Format("2006-01-02 15:04:05"))
		setCell(f, sheet, 10, row, post.Features)
		setCell(f, sheet, 11, row, post.ContactName)
		setCell(f, sheet, 12, row, post.ContactPhone)
		setCell(f, sheet, 13, row, post.HasReward)
		setCell(f, sheet, 14, row, post.RewardDescription)
		setCell(f, sheet, 15, row, post.Status)
		setCell(f, sheet, 16, row, post.CancelReason)
		setCell(f, sheet, 17, row, post.RejectReason)
		setCell(f, sheet, 18, row, post.ClaimCount)
		setCell(f, sheet, 19, row, post.ArchiveMethod)
		setCell(f, sheet, 20, row, safeFormatTime(post.ProcessedAt))
		setCell(f, sheet, 21, row, post.CreatedAt.Format("2006-01-02 15:04:05"))
		setCell(f, sheet, 22, row, post.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func (e *ExportDataApi) writeClaimSheet(ctx context.Context, f *excelize.File) error {
	sheet := "认领申请"
	_, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}

	headers := []string{"ID", "发布记录ID", "认领者ID", "补充说明", "状态", "审核人ID", "审核时间", "创建时间", "更新时间"}
	for i, header := range headers {
		setCell(f, sheet, i+1, 1, header)
	}

	crp := repo.NewClaimRepo()
	claims, err := crp.ListAll(ctx)
	if err != nil {
		return err
	}

	for i, claim := range claims {
		row := i + 2
		setCell(f, sheet, 1, row, claim.ID)
		setCell(f, sheet, 2, row, claim.PostID)
		setCell(f, sheet, 3, row, claim.ClaimantID)
		setCell(f, sheet, 4, row, claim.Description)
		setCell(f, sheet, 5, row, claim.Status)
		setCell(f, sheet, 6, row, claim.ReviewedBy)
		setCell(f, sheet, 7, row, safeFormatTime(claim.ReviewedAt))
		setCell(f, sheet, 8, row, claim.CreatedAt.Format("2006-01-02 15:04:05"))
		setCell(f, sheet, 9, row, claim.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func (e *ExportDataApi) writeFeedbackSheet(ctx context.Context, f *excelize.File) error {
	sheet := "投诉反馈"
	_, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}

	headers := []string{"ID", "物品ID", "投诉者ID", "投诉类型", "详细说明", "是否已处理", "处理人ID", "处理时间", "创建时间", "更新时间"}
	for i, header := range headers {
		setCell(f, sheet, i+1, 1, header)
	}

	frp := repo.NewFeedbackRepo()
	feedbacks, err := frp.ListAllData(ctx)
	if err != nil {
		return err
	}

	for i, feedback := range feedbacks {
		row := i + 2
		setCell(f, sheet, 1, row, feedback.ID)
		setCell(f, sheet, 2, row, feedback.PostID)
		setCell(f, sheet, 3, row, feedback.ReporterID)
		setCell(f, sheet, 4, row, feedback.Type)
		setCell(f, sheet, 5, row, feedback.Description)
		setCell(f, sheet, 6, row, feedback.Processed)
		setCell(f, sheet, 7, row, feedback.ProcessedBy)
		setCell(f, sheet, 8, row, safeFormatTime(feedback.ProcessedAt))
		setCell(f, sheet, 9, row, feedback.CreatedAt.Format("2006-01-02 15:04:05"))
		setCell(f, sheet, 10, row, feedback.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func (e *ExportDataApi) writeAnnouncementSheet(ctx context.Context, f *excelize.File) error {
	sheet := "公告通知"
	_, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}

	headers := []string{"ID", "标题", "内容", "类型", "校区", "状态", "发布者ID", "目标用户ID", "审核人ID", "审核时间", "创建时间", "更新时间"}
	for i, header := range headers {
		setCell(f, sheet, i+1, 1, header)
	}

	arr := repo.NewAnnouncementRepo()
	announcements, err := arr.ListAllData(ctx)
	if err != nil {
		return err
	}

	for i, ann := range announcements {
		row := i + 2
		setCell(f, sheet, 1, row, ann.ID)
		setCell(f, sheet, 2, row, ann.Title)
		setCell(f, sheet, 3, row, ann.Content)
		setCell(f, sheet, 4, row, ann.Type)
		setCell(f, sheet, 5, row, ann.Campus)
		setCell(f, sheet, 6, row, ann.Status)
		setCell(f, sheet, 7, row, ann.PublisherID)
		setCell(f, sheet, 8, row, ann.TargetUserID)
		setCell(f, sheet, 9, row, ann.ReviewedBy)
		setCell(f, sheet, 10, row, safeFormatTime(ann.ReviewedAt))
		setCell(f, sheet, 11, row, ann.CreatedAt.Format("2006-01-02 15:04:05"))
		setCell(f, sheet, 12, row, ann.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func (e *ExportDataApi) writeAuditLogSheet(ctx context.Context, f *excelize.File) error {
	sheet := "审计日志"
	_, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}

	headers := []string{"ID", "管理员ID", "操作类型", "理由", "发布信息ID", "旧状态", "新状态", "创建时间", "更新时间"}
	for i, header := range headers {
		setCell(f, sheet, i+1, 1, header)
	}

	alr := repo.NewAuditLogRepo()
	logs, err := alr.ListAll(ctx)
	if err != nil {
		return err
	}

	for i, log := range logs {
		row := i + 2
		setCell(f, sheet, 1, row, log.ID)
		setCell(f, sheet, 2, row, log.AdminID)
		setCell(f, sheet, 3, row, log.ActionType)
		setCell(f, sheet, 4, row, log.Reason)
		setCell(f, sheet, 5, row, log.PostID)
		setCell(f, sheet, 6, row, log.OldStatus)
		setCell(f, sheet, 7, row, log.NewStatus)
		setCell(f, sheet, 8, row, log.CreatedAt.Format("2006-01-02 15:04:05"))
		setCell(f, sheet, 9, row, log.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func (e *ExportDataApi) writeSystemConfigSheet(ctx context.Context, f *excelize.File) error {
	sheet := "系统配置"
	_, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}

	headers := []string{"ID", "配置键名", "配置值", "描述", "创建时间", "更新时间"}
	for i, header := range headers {
		setCell(f, sheet, i+1, 1, header)
	}

	scr := repo.NewSystemConfigRepo()
	configs, err := scr.ListAll(ctx)
	if err != nil {
		return err
	}

	for i, config := range configs {
		row := i + 2
		setCell(f, sheet, 1, row, config.ID)
		setCell(f, sheet, 2, row, config.ConfigKey)
		setCell(f, sheet, 3, row, config.ConfigValue)
		setCell(f, sheet, 4, row, config.Description)
		setCell(f, sheet, 5, row, config.CreatedAt.Format("2006-01-02 15:04:05"))
		setCell(f, sheet, 6, row, config.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}
