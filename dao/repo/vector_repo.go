package repo

import (
	"app/pkg/milvus"
	"context"
	"fmt"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

type VectorRepo struct {
	collectionName string
}

func NewVectorRepo() *VectorRepo {
	return &VectorRepo{}
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
	client := r.getClient()
	if client == nil {
		return fmt.Errorf("Milvus客户端未初始化")
	}

	postIDColumn := entity.NewColumnInt64("post_id", []int64{postID})
	vectorColumn := entity.NewColumnFloatVector("vector", len(vector), [][]float32{float64ToFloat32(vector)})

	_, err := client.Insert(ctx, r.getCollectionName(), "", postIDColumn, vectorColumn)
	if err != nil {
		return fmt.Errorf("插入向量失败: %w", err)
	}

	return nil
}

func (r *VectorRepo) Delete(ctx context.Context, postID int64) error {
	client := r.getClient()
	if client == nil {
		return fmt.Errorf("Milvus客户端未初始化")
	}

	err := client.DeleteByPks(ctx, r.getCollectionName(), "", entity.NewColumnInt64("post_id", []int64{postID}))
	if err != nil {
		return fmt.Errorf("删除向量失败: %w", err)
	}

	return nil
}

func (r *VectorRepo) Update(ctx context.Context, postID int64, vector []float64) error {
	if err := r.Delete(ctx, postID); err != nil {
		return err
	}
	return r.Insert(ctx, postID, vector)
}

type VectorSearchResult struct {
	PostID int64
	Score  float32
}

func (r *VectorRepo) Search(ctx context.Context, vector []float64, topK int) ([]VectorSearchResult, error) {
	client := r.getClient()
	if client == nil {
		return nil, fmt.Errorf("Milvus客户端未初始化")
	}

	sp, err := entity.NewIndexAUTOINDEXSearchParam(10)
	if err != nil {
		return nil, fmt.Errorf("创建搜索参数失败: %w", err)
	}

	results, err := client.Search(
		ctx,
		r.getCollectionName(),
		[]string{},
		"",
		[]string{"post_id"},
		[]entity.Vector{entity.FloatVector(float64ToFloat32(vector))},
		"vector",
		entity.MetricType(entity.AUTOINDEX),
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
