package contextx

import "context"

type key string

const (
	RequestIDKey      key = "request_id"
	CorrelationIDKey  key = "correlation_id"
	SubjectUUIDKey    key = "subject_uuid"
	RoleCodeKey       key = "role_code"
	PermissionCodesKey key = "permission_codes"
	AuthTypeKey       key = "auth_type"
)

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, RequestIDKey, id)
}

func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, id)
}

func WithSubjectUUID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, SubjectUUIDKey, id)
}

func WithRoleCode(ctx context.Context, code string) context.Context {
	return context.WithValue(ctx, RoleCodeKey, code)
}

func WithPermissionCodes(ctx context.Context, codes []string) context.Context {
	return context.WithValue(ctx, PermissionCodesKey, codes)
}

func WithAuthType(ctx context.Context, authType string) context.Context {
	return context.WithValue(ctx, AuthTypeKey, authType)
}

func GetRequestID(ctx context.Context) string {
	return getString(ctx, RequestIDKey)
}

func GetCorrelationID(ctx context.Context) string {
	return getString(ctx, CorrelationIDKey)
}

func GetSubjectUUID(ctx context.Context) string {
	return getString(ctx, SubjectUUIDKey)
}

func GetRoleCode(ctx context.Context) string {
	return getString(ctx, RoleCodeKey)
}

func GetPermissionCodes(ctx context.Context) []string {
	if v, ok := ctx.Value(PermissionCodesKey).([]string); ok {
		return v
	}
	return nil
}

func GetAuthType(ctx context.Context) string {
	return getString(ctx, AuthTypeKey)
}

func getString(ctx context.Context, k key) string {
	if v, ok := ctx.Value(k).(string); ok {
		return v
	}
	return ""
}
