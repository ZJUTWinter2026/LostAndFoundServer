package repo

import (
	"app/pkg/milvus"
	"context"
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

func (r *VectorRepo) InsertPostVector(ctx context.Context, postID int64, vector []float32) error {
	return milvus.InsertVector(ctx, r.getCollectionName(), postID, vector)
}

func (r *VectorRepo) UpdatePostVector(ctx context.Context, postID int64, vector []float32) error {
	return milvus.UpdateVector(ctx, r.getCollectionName(), postID, vector)
}

func (r *VectorRepo) DeletePostVector(ctx context.Context, postID int64) error {
	return milvus.DeleteVector(ctx, r.getCollectionName(), postID)
}

type VectorSearchResult struct {
	PostID int64
	Score  float32
}

func (r *VectorRepo) SearchSimilarPosts(ctx context.Context, vector []float32, topK int) ([]VectorSearchResult, error) {
	results, err := milvus.SearchSimilarVectors(ctx, r.getCollectionName(), vector, topK)
	if err != nil {
		return nil, err
	}

	var searchResults []VectorSearchResult
	for _, result := range results {
		searchResults = append(searchResults, VectorSearchResult{
			PostID: result.PostID,
			Score:  result.Score,
		})
	}

	return searchResults, nil
}
