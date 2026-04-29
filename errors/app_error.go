package errors

type Code string

const (
	BadRequest          Code = "BAD_REQUEST"           // 400
	Unauthorized        Code = "UNAUTHORIZED"          // 401
	PaymentRequired     Code = "PAYMENT_REQUIRED"      // 402
	Forbidden           Code = "FORBIDDEN"             // 403
	NotFound            Code = "NOT_FOUND"             // 404
	Conflict            Code = "CONFLICT"              // 409
	InternalServerError Code = "INTERNAL_SERVER_ERROR" // 500
	ServiceUnavailable  Code = "SERVICE_UNAVAILABLE"   // 503
)

type AppError struct {
	Code    Code
	Message string
	Meta    any
}

func (e *AppError) Error() string {
	return e.Message
}

func Wrap(code Code, message string, meta any) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Meta:    meta,
	}
}

func NewInternalServerError(message string) *AppError {
	return &AppError{Code: InternalServerError, Message: message}
}

func NewBadRequest(message string) *AppError {
	return Wrap(BadRequest, message, nil)
}

func NewBadRequestValidation(message string, meta any) *AppError {
	return &AppError{
		Code:    BadRequest,
		Message: message,
		Meta:    meta,
	}
}

func NewNotFound(message string) *AppError {
	return Wrap(NotFound, message, nil)
}

func NewConflict(message string) *AppError {
	return Wrap(Conflict, message, nil)
}

func NewForbidden(message string) *AppError {
	return Wrap(Forbidden, message, nil)
}

func NewUnauthorized(message string) *AppError {
	return Wrap(Unauthorized, message, nil)
}

func NewValidationError(message string) *AppError {
	return Wrap(BadRequest, message, nil)
}

func NewPaymentRequired(message string) *AppError {
	return Wrap(PaymentRequired, message, nil)
}

func NewServiceUnavailable(message string) *AppError {
	return Wrap(ServiceUnavailable, message, nil)
}
