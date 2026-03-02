package account

import (
	"app/comm"
	"app/dao/model"
	"reflect"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
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
		Username string `form:"username" desc:"用户名"`
		UserType string `form:"user_type" desc:"用户类型"`
		Page     int    `form:"page" binding:"required,min=1" desc:"页码"`
		PageSize int    `form:"page_size" binding:"required,min=1,max=50" desc:"每页数量"`
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
	Username      string     `json:"username"`
	Name          string     `json:"name"`
	UserType      string     `json:"user_type"`
	FirstLogin    bool       `json:"first_login"`
	DisabledUntil *time.Time `json:"disabled_until,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (a *ListApi) Run(ctx *gin.Context) kit.Code {
	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
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
	if req.Username != "" {
		db = db.Where("username = ?", req.Username)
	}
	if req.UserType != "" {
		db = db.Where("usertype = ?", req.UserType)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return comm.CodeServerError
	}

	var users []*model.User
	offset := (page - 1) * pageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return comm.CodeServerError
	}

	list := make([]AccountItem, 0, len(users))
	for _, u := range users {
		item := AccountItem{
			ID:         u.ID,
			Username:   u.Username,
			Name:       u.Name,
			UserType:   u.Usertype,
			FirstLogin: u.FirstLogin,
			CreatedAt:  u.CreatedAt,
		}
		if u.DisabledUntil != nil {
			item.DisabledUntil = u.DisabledUntil
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
