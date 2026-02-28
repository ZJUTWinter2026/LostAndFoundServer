package milvus

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"github.com/zjutjh/mygo/nlog"
)

var milvusClient client.Client

func GetClient() client.Client {
	return milvusClient
}

func InitClient(address string) error {
	if address == "" {
		nlog.Pick().Info("Milvus地址未配置，跳过初始化")
		return nil
	}

	c, err := client.NewClient(context.Background(), client.Config{
		Address: address,
	})
	if err != nil {
		return fmt.Errorf("连接Milvus失败: %w", err)
	}

	milvusClient = c
	nlog.Pick().Info("Milvus连接成功")
	return nil
}

func CloseClient() {
	if milvusClient != nil {
		milvusClient.Close()
	}
}

func CreateCollectionIfNotExist(ctx context.Context, collectionName string, dimension int) error {
	if milvusClient == nil {
		return fmt.Errorf("Milvus客户端未初始化")
	}

	has, err := milvusClient.HasCollection(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("检查Collection失败: %w", err)
	}

	if has {
		return nil
	}

	schema := &entity.Schema{
		CollectionName: collectionName,
		Description:    "Lost and found post vectors",
		AutoID:         false,
		Fields: []*entity.Field{
			{
				Name:       "post_id",
				DataType:   entity.FieldTypeInt64,
				PrimaryKey: true,
				AutoID:     false,
			},
			{
				Name:     "vector",
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": fmt.Sprintf("%d", dimension),
				},
			},
		},
	}

	err = milvusClient.CreateCollection(ctx, schema, entity.DefaultShardNumber)
	if err != nil {
		return fmt.Errorf("创建Collection失败: %w", err)
	}

	idx, err := entity.NewIndexAUTOINDEX(entity.MetricType(entity.L2))
	if err != nil {
		return fmt.Errorf("创建索引配置失败: %w", err)
	}

	err = milvusClient.CreateIndex(ctx, collectionName, "vector", idx, false)
	if err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	nlog.Pick().Infof("Collection %s 创建成功", collectionName)
	return nil
}

func LoadCollection(ctx context.Context, collectionName string) error {
	if milvusClient == nil {
		return fmt.Errorf("Milvus客户端未初始化")
	}

	return milvusClient.LoadCollection(ctx, collectionName, false)
}
