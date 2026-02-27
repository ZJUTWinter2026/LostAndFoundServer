package account

import (
	"app/comm"
	"app/comm/enum"
	"app/dao/model"
	"app/dao/repo"
	"reflect"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/ndb"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func SendNotificationHandler() gin.HandlerFunc {
	api := SendNotificationApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfSendNotification).Pointer()).Name()] = api
	return hfSendNotification
}

type SendNotificationApi struct {
	Info     struct{}                  `name:"发送系统通知" desc:"系统管理员发送系统通知"`
	Request  SendNotificationApiRequest
	Response SendNotificationApiResponse
}

type SendNotificationApiRequest struct {
	Body struct {
		UserID    *int64 `json:"user_id" desc:"指定用户ID(为空则发送全体通知)"`
		Title     string `json:"title" binding:"required,max=100" desc:"通知标题"`
		Content   string `json:"content" binding:"required,max=1000" desc:"通知内容"`
		IsGlobal  bool   `json:"is_global" desc:"是否为全体通知"`
	}
}

type SendNotificationApiResponse struct {
	ID int64 `json:"id" desc:"通知ID"`
}

func (a *SendNotificationApi) Run(ctx *gin.Context) kit.Code {
	publisherID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	urp := repo.NewUserRepo()
	user, err := urp.FindById(ctx, publisherID)
	if err != nil {
		return comm.CodeServerError
	}
	if user == nil || user.Usertype != enum.UserTypeSystemAdmin {
		return comm.CodeAdminPermissionDenied
	}

	req := a.Request.Body

	if strings.TrimSpace(req.Title) == "" {
		return comm.CodeParameterInvalid
	}
	if utf8.RuneCountInString(req.Content) > 1000 {
		return comm.CodeParameterInvalid
	}

	db := ndb.Pick().WithContext(ctx)

	announcement := &model.Announcement{
		Title:       strings.TrimSpace(req.Title),
		Content:     req.Content,
		Type:        enum.AnnouncementTypeSystem,
		Status:      enum.AnnouncementStatusApproved,
		PublisherID: publisherID,
	}

	if !req.IsGlobal && req.UserID != nil {
		var targetUser model.User
		if err := db.First(&targetUser, *req.UserID).Error; err != nil {
			return comm.CodeDataNotFound
		}
	}

	if err := db.Create(announcement).Error; err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("发送系统通知失败")
		return comm.CodeServerError
	}

	a.Response = SendNotificationApiResponse{ID: announcement.ID}
	return comm.CodeOK
}

func (a *SendNotificationApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfSendNotification(ctx *gin.Context) {
	api := &SendNotificationApi{}
	if err := api.Init(ctx); err != nil {
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
