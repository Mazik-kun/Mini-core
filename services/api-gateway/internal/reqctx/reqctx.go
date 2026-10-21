// Package reqctx — значения, которые gateway кладёт в context запроса.
// Лежит отдельно от inbound/outbound, чтобы оба слоя могли его читать без циклов.
package reqctx

import (
	"context"

	"github.com/Mazik-kun/mini-core/pkg/authjwt"
)

type key int

const (
	infoKey key = iota
	claimsKey
	tokenKey
)

// Info — изменяемая «карточка» запроса. Создаётся один раз в requestid-middleware,
// а более глубокие слои (auth, errors) дописывают в неё поля, которые потом читает
// logging-middleware (он стоит снаружи и не видит производных контекстов).
type Info struct {
	RequestID string
	UserID    string
	GRPCCode  string
}

func WithInfo(ctx context.Context, i *Info) context.Context {
	return context.WithValue(ctx, infoKey, i)
}

func InfoFrom(ctx context.Context) *Info {
	i, _ := ctx.Value(infoKey).(*Info)
	return i
}

func RequestID(ctx context.Context) string {
	if i := InfoFrom(ctx); i != nil {
		return i.RequestID
	}
	return ""
}

func WithClaims(ctx context.Context, c authjwt.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

func Claims(ctx context.Context) (authjwt.Claims, bool) {
	c, ok := ctx.Value(claimsKey).(authjwt.Claims)
	return c, ok
}

// WithToken кладёт сырой (уже проверенный) JWT — он форвардится downstream как есть.
func WithToken(ctx context.Context, raw string) context.Context {
	return context.WithValue(ctx, tokenKey, raw)
}

func Token(ctx context.Context) string {
	s, _ := ctx.Value(tokenKey).(string)
	return s
}
