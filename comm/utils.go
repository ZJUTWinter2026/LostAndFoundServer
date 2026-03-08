package comm

import (
	"app/comm/enum"
	"app/dao/repo"
	"errors"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
)

func CheckSysAdmin(ctx *gin.Context) kit.Code {
	adminID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return CodeNotLoggedIn
	}

	urp := repo.NewUserRepo()
	user, err := urp.FindById(ctx, adminID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return CodeAdminPermissionDenied
	}
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询用户失败")
		return CodeServerError
	}
	if user == nil || user.Usertype != enum.UserTypeSystemAdmin {
		return CodeAdminPermissionDenied
	}
	return CodeOK
}

func CheckAdminPermission(ctx *gin.Context) kit.Code {
	adminID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return CodeNotLoggedIn
	}

	urp := repo.NewUserRepo()
	user, err := urp.FindById(ctx, adminID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return CodeAdminPermissionDenied
	}
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询用户失败")
		return CodeServerError
	}
	if user == nil || (user.Usertype != enum.UserTypeAdmin && user.Usertype != enum.UserTypeSystemAdmin) {
		return CodeAdminPermissionDenied
	}
	return CodeOK
}
