package register

import (
	"app/comm"
	"app/comm/enum"
	"app/dao/model"
	"app/register/generate"
	"fmt"
	"time"

	"github.com/zjutjh/mygo/jwt"
	"golang.org/x/crypto/bcrypt"

	"github.com/zjutjh/mygo/config"
	"github.com/zjutjh/mygo/feishu"
	"github.com/zjutjh/mygo/foundation/kernel"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/ndb"
	"github.com/zjutjh/mygo/nesty"
	"github.com/zjutjh/mygo/nlog"
)

func Boot() kernel.BootList {
	return kernel.BootList{
		// 基础引导器
		feishu.Boot(),   // 飞书Bot (消息提醒)
		nlog.Boot(),     // 业务日志
		generate.Boot(), // 导入生成代码

		// Client引导器
		ndb.Boot(), // DB
		// nedis.Boot(), // Redis
		nesty.Boot(), // HTTP Client
		jwt.Boot[string](),

		// 业务引导器
		BizConfBoot(),
		AppBoot(),
	}
}

// BizConfBoot 初始化应用业务配置引导器
func BizConfBoot() func() error {
	return func() error {
		err := config.Pick().UnmarshalKey("biz", &comm.BizConf)
		if err != nil {
			return fmt.Errorf("%w: 解析应用业务配置错误: %w", kit.ErrDataUnmarshal, err)
		}
		return nil
	}
}

// AppBoot 应用定制引导器
func AppBoot() func() error {
	return initDefaultAdmin
}

// initDefaultAdmin 初始化默认系统管理员
func initDefaultAdmin() error {
	db := ndb.Pick()
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("查询用户数量失败: %w", err)
	}

	if count > 0 {
		return nil
	}

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	admin := &model.User{
		Username:      "root",
		Name:          "系统管理员",
		IDCard:        "",
		Password:      string(hashedPwd),
		Usertype:      enum.UserTypeSystemAdmin,
		FirstLogin:    false,
		DisabledUntil: time.Now(),
	}

	if err := db.Create(admin).Error; err != nil {
		return fmt.Errorf("创建默认管理员失败: %w", err)
	}

	nlog.Pick().Info("默认系统管理员已创建")
	return nil
}
