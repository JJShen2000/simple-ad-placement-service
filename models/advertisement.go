package models

import (
	"time"
)

type Advertisement struct {
	ID         int          `db:"id"`
	Title      string       `db:"title" json:"title" validate:"required,max=255"`
	StartAt    time.Time    `db:"start_at" json:"startAt" validate:"required"`
	EndAt      time.Time    `db:"end_at"  json:"endAt" validate:"required,gtfield=StartAt"`
	Conditions []Conditions `db:"created_at" json:"conditions" validate:"omitempty"`
}

type Conditions struct {
	AgeStart int      `db:"age_start" json:"ageStart" validate:"omitempty,min=1,max=100,default=1"`
	AgeEnd   int      `db:"age_end" json:"ageEnd" validate:"omitempty,min=1,max=100,default=100"`
	Gender   []string `db:"gender" json:"gender" validate:"omitempty,max=2,dive,oneof=M F"`
	Country  []string `db:"country" json:"country" validate:"omitempty,dive,validCountryCode"`
	Platform []string `db:"platform" json:"platform" validate:"omitempty,dive,oneof=android ios web"`
}