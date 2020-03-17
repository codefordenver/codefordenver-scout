package models

type File struct {
	ID        int `gorm:"AUTO_INCREMENT"`
	BrigadeID int `gorm:"not null"`
	Brigade   Brigade
	Name      string `gorm:"not null"`
	URL       string `gorm:"not null"`
}
