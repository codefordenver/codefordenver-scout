package migrations

import (
	"github.com/codefordenver/codefordenver-scout/models"
	"github.com/jinzhu/gorm"
)

func Migrate(db *gorm.DB) {
	db.AutoMigrate(&models.Brigade{}, &models.File{}, &models.Meeting{}, &models.VolunteerSession{})
}
