package account

import (
	"app/comm"
	"app/dao/model"
	"reflect"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/ndb"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"
	"golang.org/x/crypto/bcrypt"
)

func CreateHandler() gin.HandlerFunc {
	api := CreateApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfCreate).Pointer()).Name()] = api
	return hfCreate
}

type CreateApi struct {
	Info     struct{} `name:"新增账号" desc:"系统管理员新增账号"`
	Request  CreateApiRequest
	Response CreateApiResponse
}

type CreateApiRequest struct {
	Body struct {
		Name     string `json:"name" binding:"required,max=10" desc:"姓名"`
		UID      int64  `json:"uid" binding:"required" desc:"学号/工号"`
		IDCard   string `json:"id_card" binding:"required,len=18" desc:"身份证号"`
		Password string `json:"password" binding:"min=6,max=18" desc:"密码(可选,学生默认身份证后六位)"`
		UserType string `json:"user_type" binding:"required,oneof=STUDENT ADMIN SYSTEM_ADMIN" desc:"用户类型"`
	}
}

type CreateApiResponse struct {
	ID int64 `json:"id" desc:"用户ID"`
}

func (a *CreateApi) Run(ctx *gin.Context) kit.Code {
	if code := checkSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	req := a.Request.Body
	db := ndb.Pick().WithContext(ctx)

	var existingUser model.User
	if err := db.Where("uid = ?", req.UID).First(&existingUser).Error; err == nil {
		return comm.CodeDataConflict
	}

	password := req.Password
	if password == "" {
		if len(req.IDCard) >= 6 {
			password = req.IDCard[len(req.IDCard)-6:]
		} else {
			return comm.CodeParameterInvalid
		}
	}

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("密码加密失败")
		return comm.CodeHashError
	}

	user := &model.User{
		UID:        req.UID,
		Name:       strings.TrimSpace(req.Name),
		IDCard:     req.IDCard,
		Password:   string(hashedPwd),
		Usertype:   req.UserType,
		FirstLogin: true,
	}

	if err := db.Create(user).Error; err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("创建用户失败")
		return comm.CodeServerError
	}

	a.Response = CreateApiResponse{ID: user.ID}
	return comm.CodeOK
}

func (a *CreateApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfCreate(ctx *gin.Context) {
	api := &CreateApi{}
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
