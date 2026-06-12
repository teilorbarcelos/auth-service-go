package dashboard

type DashboardStatsResponseDto struct {
	UserCreationStats    []TimeSeriesStatDto    `json:"userCreationStats"`
	ProductCreationStats []TimeSeriesStatDto    `json:"productCreationStats"`
	ProductsPerUser      []UserProductStatDto   `json:"productsPerUser"`
}

type TimeSeriesStatDto struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type UserProductStatDto struct {
	UserID   *string `json:"userId"`
	UserName string  `json:"userName"`
	Count    int     `json:"count"`
}
