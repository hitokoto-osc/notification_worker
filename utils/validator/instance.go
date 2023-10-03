package validator

import (
	"context"
	"github.com/bytedance/sonic"
	"github.com/cockroachdb/errors"
	lib "github.com/go-playground/validator/v10"
	"github.com/hitokoto-osc/notification-worker/consts"
)

var validator *lib.Validate

func init() {
	validator = lib.New(lib.WithRequiredStructEnabled())
	validator.RegisterCustomTypeFunc(validateHitokotoTypeValue, consts.HitokotoTypeAnime)
	validator.RegisterCustomTypeFunc(validatePollMethodValue, consts.PollMethodApprove)
	validator.RegisterCustomTypeFunc(validatePollStatusValue, consts.PollStatusUnknown)
}

// GetValidator get validator
func GetValidator() *lib.Validate {
	return validator
}

// ValidateStruct validate struct
// It is a shortcut for: validator.Struct(s)
func ValidateStruct(s any) error {
	return validator.Struct(s)
}

// UnmarshalV unmarshal and validate
// It is a shortcut for: sonic.Unmarshal(s, &t); validator.StructCtx(ctx, t)
func UnmarshalV[T any](ctx context.Context, s []byte) (*T, error) {
	var t T
	err := sonic.Unmarshal(s, &t)
	if err != nil {
		return nil, err
	}
	err = validator.StructCtx(ctx, t)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func IsValidationError(err error) bool {
	var validationErrors lib.ValidationErrors
	return errors.Is(err, &validationErrors)
}
