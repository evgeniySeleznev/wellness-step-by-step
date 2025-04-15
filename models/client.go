package models

import "gorm.io/gorm"

type Client struct {
	gorm.Model
	FullName           string `gorm:"not null"`
	Phone              string `gorm:"not null"`
	Email              string `gorm:"not null;unique"`
	AdvertisingChannel string `gorm:"not null"`
	SpecialistID       uint
	MeetingPlace       string `gorm:"not null"`
	Occupation         string `gorm:"not null"`
	Gender             string `gorm:"not null"`
	Age                int    `gorm:"not null"`
	ReasonForVisit     string `gorm:"not null"`
	SpecialistNotes    string
}
