package account

import (
	"app/comm"
	"app/comm/enum"
	"app/dao/model"
	"app/dao/repo"
	"reflect"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/ndb"
	"github.com/zjutjh/mygo/swagger"
)

func ListHandler() gin.HandlerFunc {
	api := ListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfList).Pointer()).Name()] = api
	return hfList
}

type ListApi struct {
	Info     struct{} `name:"获取账号列表" desc:"获取账号列表"`
	Request  ListApiRequest
	Response ListApiResponse
}

type ListApiRequest struct {
	Query struct {
		UID      int64  `form:"uid" binding:"omitempty" desc:"学号/工号"`
		UserType string `form:"user_type" binding:"omitempty" desc:"用户类型"`
		Page     int    `form:"page" binding:"omitempty,min=1" desc:"页码"`
		PageSize int    `form:"page_size" binding:"omitempty,min=1,max=50" desc:"每页数量"`
	}
}

type ListApiResponse struct {
	Total    int64         `json:"total" desc:"总数"`
	Page     int           `json:"page" desc:"页码"`
	PageSize int           `json:"page_size" desc:"每页数量"`
	List     []AccountItem `json:"list" desc:"账号列表"`
}

type AccountItem struct {
	ID            int64      `json:"id"`
	UID           int64      `json:"uid"`
	UserType      string     `json:"user_type"`
	FirstLogin    bool       `json:"first_login"`
	DisabledUntil *time.Time `json:"disabled_until,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (a *ListApi) Run(ctx *gin.Context) kit.Code {
	if code := checkSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	req := a.Request.Query
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	db := ndb.Pick().WithContext(ctx).Model(&model.User{})
	if req.UID > 0 {
		db = db.Where("uid = ?", req.UID)
	}
	if req.UserType != "" {
		db = db.Where("usertype = ?", req.UserType)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return comm.CodeDatabaseError
	}

	var users []*model.User
	offset := (page - 1) * pageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return comm.CodeDatabaseError
	}

	list := make([]AccountItem, 0, len(users))
	for _, u := range users {
		item := AccountItem{
			ID:         u.ID,
			UID:        u.UID,
			UserType:   u.Usertype,
			FirstLogin: u.FirstLogin,
			CreatedAt:  u.CreatedAt,
		}
		if !u.DisabledUntil.IsZero() {
			item.DisabledUntil = &u.DisabledUntil
		}
		list = append(list, item)
	}

	a.Response = ListApiResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     list,
	}
	return comm.CodeOK
}

func (a *ListApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindQuery(&a.Request.Query)
}

func hfList(ctx *gin.Context) {
	api := &ListApi{}
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

func checkSysAdmin(ctx *gin.Context) kit.Code {
	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	adminID := cast.ToInt64(id)

	urp := repo.NewUserRepo()
	user, err := urp.FindById(ctx, adminID)
	if err != nil {
		return comm.CodeDatabaseError
	}
	if user == nil || user.Usertype != enum.UserTypeSystemAdmin {
		return comm.CodeAdminPermissionDenied
	}
	return comm.CodeOK
}
