package models

type {{.Name}} struct {
	BaseModel
	Name   string `gorm:"type:varchar(255);not null" json:"name"`
	Active bool   `gorm:"default:true" json:"active"`
}
