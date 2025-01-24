package handlex

import (
	"github.com/go-playground/validator/v10"
	"mime/multipart"
	"path/filepath"
	"slices"
	"strings"
)

type RequestValidator interface {
	Validate(data any) error
	RegisterValidation(tag string, fn validator.Func, callValidationEvenIfNull ...bool) error
}

type requestValidator struct {
	validate *validator.Validate
}

func NewRequestValidator() RequestValidator {
	validate := validator.New()

	// Register default custom validation
	_ = validate.RegisterValidation("allow-file-extensions", ValidateAllowFileExtensions)
	_ = validate.RegisterValidation("allow-file-mime-types", ValidateAllowFileMimeTypes)

	return &requestValidator{
		validate: validate,
	}
}

func (r *requestValidator) RegisterValidation(tag string, fn validator.Func, callValidationEvenIfNull ...bool) error {
	return r.validate.RegisterValidation(tag, fn, callValidationEvenIfNull...)
}

func (r *requestValidator) Validate(data any) error {
	return r.validate.Struct(data)
}

func ValidateAllowFileExtensions(fl validator.FieldLevel) bool {
	fileHeader, ok := fl.Field().Interface().(multipart.FileHeader)
	if !ok {
		return false
	}

	tag := fl.Param()
	allowedFileExtensions := strings.Split(tag, ":")

	fileExtension := filepath.Ext(fileHeader.Filename)
	return slices.Contains(allowedFileExtensions, fileExtension)
}

func ValidateAllowFileMimeTypes(fl validator.FieldLevel) bool {
	fileHeader, ok := fl.Field().Interface().(multipart.FileHeader)
	if !ok {
		return false
	}

	tag := fl.Param()
	allowedTypes := strings.Split(tag, ":")

	mimeType, err := GetFileMimeType(&fileHeader)
	if err != nil {
		return false
	}

	return slices.Contains(allowedTypes, mimeType)
}
