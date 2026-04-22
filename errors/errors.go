package errors

var (
	ErrBadRequest = &AppError{
		Code:    "BAD_REQUEST",
		Message: "Bad request",
	}

	ErrUnauthorized = &AppError{
		Code:    "UNAUTHORIZED",
		Message: "Unauthorized",
	}

	ErrForbidden = &AppError{
		Code:    "FORBIDDEN",
		Message: "Forbidden",
	}

	ErrNotFound = &AppError{
		Code:    "NOT_FOUND",
		Message: "Resource not found",
	}

	ErrConflict = &AppError{
		Code:    "CONFLICT",
		Message: "Conflict",
	}

	ErrInternal = &AppError{
		Code:    "INTERNAL_ERROR",
		Message: "Internal server error",
	}
)
