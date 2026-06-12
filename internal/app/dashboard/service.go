package dashboard

import (
	"context"
	"time"
)

type DashboardService struct {
	Repo DashboardRepositoryI
}

type DashboardServiceI interface {
	GetStats(ctx context.Context, start, end time.Time) (*DashboardStatsResponseDto, error)
}

func NewDashboardService(repo DashboardRepositoryI) *DashboardService {
	return &DashboardService{Repo: repo}
}

func (s *DashboardService) GetStats(ctx context.Context, start, end time.Time) (*DashboardStatsResponseDto, error) {
	userStats, err := s.Repo.GetUserStats(ctx, start, end)
	if err != nil {
		return nil, err
	}

	productStats, err := s.Repo.GetProductStats(ctx, start, end)
	if err != nil {
		return nil, err
	}

	productsPerUser, err := s.Repo.GetProductsPerUser(ctx, start, end)
	if err != nil {
		return nil, err
	}

	return &DashboardStatsResponseDto{
		UserCreationStats:    userStats,
		ProductCreationStats: productStats,
		ProductsPerUser:      productsPerUser,
	}, nil
}
