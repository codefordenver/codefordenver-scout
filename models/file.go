package models

type File struct {
	ID        int `gorm:"primary_key"`
	BrigadeID int `gorm:"type:int REFERENCES brigades(id);not null"`
	Brigade   Brigade `gorm:"foreignkey:BrigadeID"`
	Name      string `gorm:"not null"`
	URL       string `gorm:"not null"`
}
