package repo

import (
	"app/pkg/milvus"
	"context"
	"fmt"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// defaultVectorCollection 是全局默认的 Milvus Collection 名称，
// 在应用启动时由 register/boot.go 调用 SetDefaultVectorCollection 设置。
var defaultVectorCollection = "lost_and_found"

// SetDefaultVectorCollection 设置默认 Milvus Collection 名称（应在启动阶段调用）
func SetDefaultVectorCollection(name string) {
	if name != "" {
		defaultVectorCollection = name
	}
}

type VectorRepo struct {
	collectionName string
}

func NewVectorRepo() *VectorRepo {
	// 使用包级默认值，该值在启动时从配置读取
	return &VectorRepo{collectionName: defaultVectorCollection}
}

func (r *VectorRepo) SetCollectionName(name string) {
	r.collectionName = name
}

func (r *VectorRepo) getCollectionName() string {
	if r.collectionName != "" {
		return r.collectionName
	}
	return "lost_and_found"
}

func (r *VectorRepo) getClient() client.Client {
	return milvus.GetClient()
}

func float64ToFloat32(in []float64) []float32 {
	out := make([]float32, len(in))
	for i, v := range in {
		out[i] = float32(v)
	}
	return out
}

func (r *VectorRepo) Insert(ctx context.Context, postID int64, vector []float64) error {
	getClient := r.getClient()
	if getClient == nil {
		return fmt.Errorf("Milvus客户端未初始化")
	}

	postIDColumn := entity.NewColumnInt64("post_id", []int64{postID})
	vectorColumn := entity.NewColumnFloatVector("vector", len(vector), [][]float32{float64ToFloat32(vector)})

	_, err := getClient.Insert(ctx, r.getCollectionName(), "", postIDColumn, vectorColumn)
	if err != nil {
		return fmt.Errorf("插入向量失败: %w", err)
	}

	return nil
}

func (r *VectorRepo) Delete(ctx context.Context, postID int64) error {
	getClient := r.getClient()
	if getClient == nil {
		return fmt.Errorf("Milvus客户端未初始化")
	}

	err := getClient.DeleteByPks(ctx, r.getCollectionName(), "", entity.NewColumnInt64("post_id", []int64{postID}))
	if err != nil {
		return fmt.Errorf("删除向量失败: %w", err)
	}

	return nil
}

func (r *VectorRepo) Update(ctx context.Context, postID int64, vector []float64) error {
	// 先 Insert 再 Delete：若 Insert 失败则旧数据仍在，避免向量永久丢失
	if err := r.Insert(ctx, postID, vector); err != nil {
		return err
	}
	// 删除旧条目（忽略删除失败，数据库中顶多有一条重复记录，不影响搜索正确性）
	_ = r.Delete(ctx, postID)
	return nil
}

type VectorSearchResult struct {
	PostID int64
	Score  float32
}

func (r *VectorRepo) Search(ctx context.Context, vector []float64, topK int) ([]VectorSearchResult, error) {
	getClient := r.getClient()
	if getClient == nil {
		return nil, fmt.Errorf("Milvus客户端未初始化")
	}

	sp, err := entity.NewIndexAUTOINDEXSearchParam(10)
	if err != nil {
		return nil, fmt.Errorf("创建搜索参数失败: %w", err)
	}

	results, err := getClient.Search(
		ctx,
		r.getCollectionName(),
		[]string{},
		"",
		[]string{"post_id"},
		[]entity.Vector{entity.FloatVector(float64ToFloat32(vector))},
		"vector",
		entity.L2,
		topK,
		sp,
	)
	if err != nil {
		return nil, fmt.Errorf("搜索向量失败: %w", err)
	}

	var searchResults []VectorSearchResult
	for _, result := range results {
		postIDs, ok := result.Fields.GetColumn("post_id").(*entity.ColumnInt64)
		if !ok {
			continue
		}

		for i := 0; i < result.ResultCount; i++ {
			postID, err := postIDs.ValueByIdx(i)
			if err != nil {
				continue
			}
			searchResults = append(searchResults, VectorSearchResult{
				PostID: postID,
				Score:  result.Scores[i],
			})
		}
	}

	return searchResults, nil
}
