package models

import "time"

type User struct {
	BaseModel
	Name      string  `gorm:"type:varchar(255);not null" json:"name"`
	Phone     *string `gorm:"type:varchar(15)" json:"phone"`
	Email     string  `gorm:"type:varchar(255);unique;not null" json:"email"`
	CognitoID *string `gorm:"type:varchar(255);unique" json:"cognito_id"`
	Active    bool    `gorm:"default:true" json:"active"`
	Document  *string `gorm:"type:varchar(20)" json:"document"`
	Avatar    *string `gorm:"type:varchar(255)" json:"avatar"`
	IDAuth    *string `gorm:"type:varchar(40);unique" json:"id_auth"`
	IDRole    string  `gorm:"type:varchar(40);not null" json:"id_role"`

	Auth *Auth `gorm:"foreignKey:IDAuth" json:"Auth,omitempty" swaggerignore:"true"`
	Role *Role `gorm:"foreignKey:IDRole" json:"Role,omitempty"`
}

type Auth struct {
	BaseModel
	Password                  *string    `gorm:"type:varchar(255)" json:"-"`
	RequestPasswordToken      *string    `gorm:"type:varchar(255)" json:"request_password_token"`
	RequestPasswordExpiration *time.Time `json:"request_password_expiration"`
	Retries                   int        `gorm:"default:0" json:"retries"`
	FirstAccess               bool       `gorm:"default:true" json:"first_access"`
	Active                    bool       `gorm:"default:true" json:"active"`
	SessionVersion            int        `gorm:"default:0" json:"session_version"`

	User *User `gorm:"foreignKey:IDAuth" json:"user,omitempty" swaggerignore:"true"`
}

type Role struct {
	BaseModel
	Name        string        `gorm:"type:varchar(255);not null" json:"name"`
	Description string        `gorm:"type:varchar(255);not null" json:"description"`
	Active      bool          `gorm:"default:true" json:"active"`
	Users       []User        `gorm:"foreignKey:IDRole" json:"Users,omitempty" swaggerignore:"true"`
	RoleFeature []RoleFeature `gorm:"foreignKey:IDRole" json:"RoleFeature,omitempty"`
}

type Feature struct {
	BaseModel
	Name        string        `gorm:"type:varchar(255);not null" json:"name"`
	Description string        `gorm:"not null" json:"description"`
	Active      bool          `gorm:"default:true" json:"active"`
	RoleFeature []RoleFeature `gorm:"foreignKey:IDFeature" json:"RoleFeature,omitempty" swaggerignore:"true"`
}

type RoleFeature struct {
	IDRole    string `gorm:"type:varchar(40);primaryKey" json:"id_role"`
	IDFeature string `gorm:"type:varchar(40);primaryKey" json:"id_feature"`
	Create    bool   `gorm:"default:false" json:"create"`
	View      bool   `gorm:"default:false" json:"view"`
	Activate  bool   `gorm:"default:false" json:"activate"`
	Delete    bool   `gorm:"default:false" json:"delete"`

	Role    Role    `gorm:"foreignKey:IDRole;references:ID" json:"role" swaggerignore:"true"`
	Feature Feature `gorm:"foreignKey:IDFeature;references:ID" json:"feature" swaggerignore:"true"`
}
