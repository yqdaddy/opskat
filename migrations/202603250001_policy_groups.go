package migrations

import (
	"github.com/opskat/opskat/internal/model/entity/policy_group_entity"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// migration202603250001 创建权限组表
func migration202603250001() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202603250001",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&policy_group_entity.PolicyGroup{})
		},
	}
}
