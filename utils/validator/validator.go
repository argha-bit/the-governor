package validator

import (
	"regexp"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

type Validator struct {
	Validator *validator.Validate
	trans     ut.Translator
}

type ValidationError struct {
	Err    error
	Msg    string
	Fields map[string]string
}

func NewValidator() *Validator {
	localTrans := en.New()
	validate := validator.New()
	universalTranslator := ut.New(localTrans, localTrans)
	translator, _ := universalTranslator.GetTranslator("en") // not catching found becuase en will be found by default

	validatorObj := &Validator{
		Validator: validate,
		trans:     translator,
	}
	setUpValidations(validatorObj)
	setUpRegisteredTranslation(validatorObj)
	return validatorObj
}

func setUpValidations(validatorObj *Validator) {
	validatorObj.Validator.RegisterValidation("checkValidGrade", validatorObj.checkValidGrade)
	validatorObj.Validator.RegisterValidation("checkValidGradeUpdate", validatorObj.checkValidGradeUpdate)
}
func setUpRegisteredTranslation(validatorObj *Validator) {
	registerTranslation(validatorObj.Validator, validatorObj.trans, "checkValidGrade", "invalid class value,please enter valid grade. e.g Grade 1, Grade 2")
	registerTranslation(validatorObj.Validator, validatorObj.trans, "checkValidGradeUpdate", "invalid class value,please enter valid grade. e.g Grade 1, Grade 2")

}
func (v *Validator) checkValidGradeUpdate(fl validator.FieldLevel) bool {
	class, ok := fl.Field().Interface().(string)
	if !ok {
		return true
	}
	r := regexp.MustCompile(`^Grade\s(1[0-2]|[1-9])$`)
	return r.MatchString(class)
}

func (v *Validator) checkValidGrade(fl validator.FieldLevel) bool {
	class, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}
	r := regexp.MustCompile(`^Grade\s(1[0-2]|[1-9])$`)
	return r.MatchString(class)

}
func registerTranslation(v *validator.Validate, trans ut.Translator, tag, message string) {
	v.RegisterTranslation(tag, trans, func(ut ut.Translator) error {
		return ut.Add(tag, message, true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T(tag, fe.Field(), fe.Param())
		return t
	},
	)
}

func (v *Validator) Validate(i interface{}) error {
	err := v.Validator.Struct(i)
	if err == nil {
		return nil
	}
	return v.NewValidationError(err)
}

// overriding Error method for struct so it can be passed as an error
func (v *ValidationError) Error() string {
	return v.Msg
}

// structurizes generated error in to validation error
func (v *Validator) NewValidationError(err error) *ValidationError {
	switch err.(type) {
	case validator.ValidationErrors:
		return v.createValidationError(err.(validator.ValidationErrors))
	default:
		return &ValidationError{
			Err: err,
			Msg: err.Error(),
		}
	}

}

// creates validation error when generated and structures them
func (v *Validator) createValidationError(errs validator.ValidationErrors) *ValidationError {

	customFieldErrs := map[string]string{}
	msg := ""
	for _, e := range errs {
		customFieldErrs[strings.ToLower(e.Field())] = strings.ToLower(e.Translate(v.trans))
		msg = customFieldErrs[strings.ToLower(e.Field())]
	}
	return &ValidationError{
		Err:    errs,
		Msg:    msg,
		Fields: customFieldErrs,
	}
}
