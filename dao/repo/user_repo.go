package repo

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/zjutjh/mygo/ndb"
	"gorm.io/gorm"

	"app/dao/model"
	"app/dao/query"
)

type UserRepo struct {
	query *query.Query
}

func NewUserRepo() *UserRepo {
	return &UserRepo{
		query: query.Use(ndb.Pick()),
	}
}

// FindById 根据ID查询用户
func (r *UserRepo) FindById(ctx context.Context, id int64) (*model.User, error) {
	u := r.query.User
	record, err := u.WithContext(ctx).Where(u.ID.Eq(id)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

// FindByUid 根据Uid查找用户
func (r *UserRepo) FindByUid(ctx context.Context, uid int64) (*model.User, error) {
	u := r.query.User
	record, err := u.WithContext(ctx).Where(u.UID.Eq(uid)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, uid int64, password string) error {
	u := r.query.User
	_, err := u.WithContext(ctx).Where(u.UID.Eq(uid)).Update(u.Password, password)
	return err
}

func (r *UserRepo) EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func (r *UserRepo) JoinURL(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		trim := strings.Trim(part, "/")
		if trim != "" {
			cleaned = append(cleaned, trim)
		}
	}
	return "/" + strings.Join(cleaned, "/")
}

func (r *UserRepo) UpdateFirstLogin(ctx context.Context, id int64) error {
	u := r.query.User
	_, err := u.WithContext(ctx).Where(u.ID.Eq(id)).Update(u.FirstLogin, false)
	return err
}
