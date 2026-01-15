package auth

import "context"

type ctxKey int

const uidCtxKey ctxKey = iota

func contextWithUID(ctx context.Context, uid int64) context.Context {
	return context.WithValue(ctx, uidCtxKey, uid)
}

func ExtractUIDFromContext(ctx context.Context) (int64, bool) {
	uid, ok := ctx.Value(uidCtxKey).(int64)
	return uid, ok
}
