package dashboard

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type DashboardRepository struct {
	DB *gorm.DB
}

type DashboardRepositoryI interface {
	GetUserStats(ctx context.Context, start, end time.Time) ([]TimeSeriesStatDto, error)
	GetProductStats(ctx context.Context, start, end time.Time) ([]TimeSeriesStatDto, error)
	GetProductsPerUser(ctx context.Context, start, end time.Time) ([]UserProductStatDto, error)
}

func NewDashboardRepository(db *gorm.DB) *DashboardRepository {
	return &DashboardRepository{DB: db}
}

func (r *DashboardRepository) GetUserStats(ctx context.Context, start, end time.Time) ([]TimeSeriesStatDto, error) {
	results := []TimeSeriesStatDto{}
	err := r.DB.WithContext(ctx).Table("user").
		Select("TO_CHAR(created_at AT TIME ZONE 'America/Sao_Paulo', 'YYYY-MM-DD') AS date, COUNT(*) AS count").
		Where("created_at >= ? AND created_at <= ? AND is_deleted = ?", start, end, false).
		Group("TO_CHAR(created_at AT TIME ZONE 'America/Sao_Paulo', 'YYYY-MM-DD')").
		Order("date ASC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (r *DashboardRepository) GetProductStats(ctx context.Context, start, end time.Time) ([]TimeSeriesStatDto, error) {
	results := []TimeSeriesStatDto{}
	err := r.DB.WithContext(ctx).Table("product").
		Select("TO_CHAR(created_at AT TIME ZONE 'America/Sao_Paulo', 'YYYY-MM-DD') AS date, COUNT(*) AS count").
		Where("created_at >= ? AND created_at <= ? AND is_deleted = ?", start, end, false).
		Group("TO_CHAR(created_at AT TIME ZONE 'America/Sao_Paulo', 'YYYY-MM-DD')").
		Order("date ASC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (r *DashboardRepository) GetProductsPerUser(ctx context.Context, start, end time.Time) ([]UserProductStatDto, error) {
	type rawResult struct {
		UserID   *string `gorm:"column:user_id"`
		UserName *string `gorm:"column:user_name"`
		Count    int     `gorm:"column:count"`
	}

	var rawResults []rawResult
	err := r.DB.WithContext(ctx).Table("product").
		Select("product.id_user AS user_id, \"user\".name AS user_name, COUNT(*) AS count").
		Joins("LEFT JOIN \"user\" ON product.id_user = \"user\".id").
		Where("product.created_at >= ? AND product.created_at <= ? AND product.is_deleted = ?", start, end, false).
		Group("product.id_user, \"user\".name").
		Order("count DESC").
		Scan(&rawResults).Error
	if err != nil {
		return nil, err
	}

	results := []UserProductStatDto{}
	for _, r := range rawResults {
		userName := "Anonymous"
		if r.UserName != nil {
			userName = *r.UserName
		}
		results = append(results, UserProductStatDto{
			UserID:   r.UserID,
			UserName: userName,
			Count:    r.Count,
		})
	}
	return results, nil
}
