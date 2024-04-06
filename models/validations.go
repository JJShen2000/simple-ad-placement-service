package models

import (
    "github.com/go-playground/validator/v10"
	"github.com/biter777/countries"
)

var validate *validator.Validate

func init() {
    validate = validator.New()
    validate.RegisterValidation("validCountryCode", validCountryCodeValidator)
}

// custom validation function to validate country code
func validCountryCodeValidator(fl validator.FieldLevel) bool {
    code := fl.Field().String()
    return countries.ByName(code) != countries.Unknown
}

func GetValidate() *validator.Validate {
	return validate
}
