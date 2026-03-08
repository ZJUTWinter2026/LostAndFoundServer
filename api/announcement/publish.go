package announcement

import (
	"app/comm"
	"app/comm/enum"
	"app/dao/model"
	"app/dao/repo"
	"errors"
	"reflect"
	"runtime"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func PublishHandler() gin.HandlerFunc {
	api := PublishApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfPublish).Pointer()).Name()] = api
	return hfPublish
}

type PublishApi struct {
	Info     struct{} `name:"发布公告" desc:"系统管理员发布全局公告"`
	Request  PublishApiRequest
	Response PublishApiResponse
}

type PublishApiRequest struct {
	Body struct {
		Title        string `json:"title" binding:"required,max=100" desc:"标题"`
		Content      string `json:"content" binding:"required,max=5000" desc:"内容"`
		Type         string `json:"type" binding:"required,oneof=SYSTEM REGION" desc:"类型 SYSTEM系统公告/REGION区域公告"`
		Campus       string `json:"campus" binding:"omitempty,oneof=ZHAO_HUI PING_FENG MO_GAN_SHAN" desc:"所属校区: ZHAO_HUI, PING_FENG, MO_GAN_SHAN, 仅REGION类型有效"`
		TargetUserID int64  `json:"target_user_id" desc:"目标用户ID, 0表示全局公告/系统公告, 非0表示针对特定用户, 仅超管可用"`
	}
}

type PublishApiResponse struct {
	ID int64 `json:"id" desc:"公告ID"`
}

func (a *PublishApi) Run(ctx *gin.Context) kit.Code {
	req := a.Request.Body

	publisherID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	urp := repo.NewUserRepo()
	user, err := urp.FindById(ctx, publisherID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return comm.CodeAdminPermissionDenied
	}
	if err != nil {
		return comm.CodeServerError
	}
	if user == nil || (user.Usertype != enum.UserTypeAdmin && user.Usertype != enum.UserTypeSystemAdmin) {
		return comm.CodeAdminPermissionDenied
	}

	if strings.TrimSpace(req.Title) == "" {
		return comm.CodeParameterInvalid
	}
	if utf8.RuneCountInString(req.Content) > 5000 {
		return comm.CodeParameterInvalid
	}

	isSysAdmin := user.Usertype == enum.UserTypeSystemAdmin

	if !isSysAdmin {
		if req.Type != enum.AnnouncementTypeRegion {
			return comm.CodeParameterInvalid
		}
		if user.Campus == "" {
			return comm.CodeParameterInvalid
		}
	}

	announcement := &model.Announcement{
		Title:        strings.TrimSpace(req.Title),
		Content:      req.Content,
		Type:         req.Type,
		PublisherID:  publisherID,
		TargetUserID: req.TargetUserID,
	}

	if req.Type == enum.AnnouncementTypeRegion {
		if isSysAdmin {
			announcement.Campus = req.Campus
		} else {
			announcement.Campus = user.Campus
		}
	}

	if isSysAdmin {
		announcement.Status = enum.AnnouncementStatusApproved
	} else {
		announcement.Status = enum.AnnouncementStatusPending
	}

	arr := repo.NewAnnouncementRepo()
	err = arr.Create(ctx, announcement)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("发布公告失败")
		return comm.CodeServerError
	}

	a.Response = PublishApiResponse{ID: announcement.ID}
	return comm.CodeOK
}

func (a *PublishApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfPublish(ctx *gin.Context) {
	api := &PublishApi{}
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
