package register

import (
	"app/comm"
	"app/comm/enum"
	"app/dao/model"
	"app/dao/repo"
	"app/pkg/llm"
	"app/pkg/milvus"
	"app/register/generate"
	"context"
	"fmt"

	"github.com/zjutjh/mygo/config"
	"github.com/zjutjh/mygo/feishu"
	"github.com/zjutjh/mygo/foundation/kernel"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/ndb"
	"github.com/zjutjh/mygo/nesty"
	"github.com/zjutjh/mygo/nlog"
	"golang.org/x/crypto/bcrypt"
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

		// 业务引导器
		BizConfBoot(),
		initAgentServices,
		initDefaultAdmin,
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

// initDefaultAdmin 初始化默认系统管理员
func initDefaultAdmin() error {
	db := ndb.Pick()
	var count int64
	if err := db.Model(&model.User{}).Where("usertype = ?", enum.UserTypeSystemAdmin).Count(&count).Error; err != nil {
		return fmt.Errorf("查询管理员数量失败: %w", err)
	}

	if count > 0 {
		return nil
	}

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	admin := &model.User{
		Username:   "root",
		Name:       "系统管理员",
		IDCard:     "",
		Password:   string(hashedPwd),
		Usertype:   enum.UserTypeSystemAdmin,
		FirstLogin: false,
	}

	if err := db.Create(admin).Error; err != nil {
		return fmt.Errorf("创建默认管理员失败: %w", err)
	}

	nlog.Pick().Info("默认系统管理员已创建")
	return nil
}

func initAgentServices() error {
	if !comm.BizConf.Agent.Enable {
		nlog.Pick().Info("Agent功能已禁用，跳过初始化")
		return nil
	}

	cfg := comm.BizConf.Agent

	if cfg.LLM.Model != "" {
		llm.GetChatModel()
		nlog.Pick().Info("LLM模型初始化完成")
	}

	if cfg.VisionLLM.Model != "" {
		llm.GetVisionModel()
		nlog.Pick().Info("VisionLLM模型初始化完成")
	}

	if cfg.Embedding.Model != "" {
		llm.GetEmbeddingModel()
		nlog.Pick().Info("Embedding模型初始化完成")
	}

	if cfg.Milvus.Address != "" {
		err := milvus.InitClient(cfg.Milvus.Address)
		if err != nil {
			nlog.Pick().WithError(err).Warn("Milvus初始化失败")
		} else {
			nlog.Pick().Info("Milvus初始化完成")

			// 确保 Collection 存在并已加载
			collectionName := cfg.Milvus.Collection
			if collectionName == "" {
				collectionName = "lost_and_found"
			}

			// 将配置的 collection 名同步给 VectorRepo
			repo.SetDefaultVectorCollection(collectionName)
			dimension := cfg.Embedding.Dimension
			if dimension <= 0 {
				dimension = 1536
			}

			bgCtx := context.Background()
			if err := milvus.CreateCollectionIfNotExist(bgCtx, collectionName, dimension); err != nil {
				nlog.Pick().WithError(err).Warn("Milvus Collection 创建失败")
			} else {
				nlog.Pick().Infof("Milvus Collection [%s] 就绪", collectionName)
				if err := milvus.LoadCollection(bgCtx, collectionName); err != nil {
					nlog.Pick().WithError(err).Warn("Milvus Collection 加载失败")
				} else {
					nlog.Pick().Infof("Milvus Collection [%s] 已加载", collectionName)
				}
			}
		}
	}

	return nil
}
