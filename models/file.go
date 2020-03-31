package models

type File struct {
	ID        int `gorm:"PRIMARY_KEY"`
	BrigadeID int `gorm:"type:int REFERENCES brigades(id);NOT NULL;"`
	Brigade   Brigade
	Name      string `gorm:"NOT NULL"`
	URL       string `gorm:"NOT NULL"`
}
