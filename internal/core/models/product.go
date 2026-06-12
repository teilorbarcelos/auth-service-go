package models

type Product struct {
	BaseModel
	Name        string  `gorm:"type:varchar(255);not null" json:"name"`
	SKU         string  `gorm:"type:varchar(100);unique;not null" json:"sku"`
	Category    string  `gorm:"type:varchar(100);not null" json:"category"`
	Price       float64 `gorm:"type:decimal(10,2);not null" json:"price"`
	Stock       int     `gorm:"default:0" json:"stock"`
	Description string  `gorm:"type:text" json:"description"`
	Active      bool    `gorm:"default:true" json:"active"`
	IDUser      *string `gorm:"column:id_user;type:varchar(40)" json:"id_user,omitempty"`
	User        *User   `gorm:"foreignKey:IDUser" json:"user,omitempty"`
}
