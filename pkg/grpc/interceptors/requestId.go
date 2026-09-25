package interceptors

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ctxKey — приватный тип для ключа контекста.
// Используем пустую структуру, а не строку, чтобы никто извне
// не мог создать такой же ключ и случайно перезаписать наше значение.
type ctxKey struct{}

// requestIDKey — единственный экземпляр нашего ключа.
// Через него кладём и достаём request-id из контекста.
var requestIDKey = ctxKey{}

// RequestIDFromContext достаёт request-id из контекста.
// Возвращает пустую строку, если request-id там нет
// (например, код вызван вне gRPC-запроса).
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// RequestID возвращает unary-интерцептор, который:
//  1. читает x-request-id из входящих метаданных, если клиент его прислал;
//  2. иначе генерирует новый UUID v7;
//  3. кладёт его в контекст, чтобы весь код ниже мог достать;
//  4. добавляет его в исходящие метаданные, чтобы клиент увидел его в ответе.
func RequestID() grpc.UnaryServerInterceptor {
	// Фабрика: возвращаем замыкание с сигнатурой grpc.UnaryServerInterceptor.
	// Логгер тут не нужен, поэтому фабрика без параметров.
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Пытаемся достать метаданные из входящего контекста.
		// metadata.FromIncomingContext возвращает (metadata.MD, bool).
		// ok == false, если клиент вообще ничего не прислал.
		var requestID string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			// md.Get("x-request-id") возвращает []string,
			// потому что один заголовок может быть передан несколько раз.
			if values := md.Get("x-request-id"); len(values) > 0 {
				requestID = values[0]
			}
		}

		// Если клиент не прислал — генерируем свой.
		// uuid.NewV7() возвращает (uuid.UUID, error).
		// Ошибка тут практически невозможна, но на всякий случай
		// есть fallback на v4, который ошибок не возвращает.
		if requestID == "" {
			if id, err := uuid.NewV7(); err == nil {
				requestID = id.String()
			} else {
				requestID = uuid.NewString()
			}
		}

		// Кладём request-id в контекст с нашим приватным ключом.
		// Теперь любой код ниже может достать его через RequestIDFromContext.
		// Переменная ctx затеняется — дальше в функции используется новый контекст.
		ctx = context.WithValue(ctx, requestIDKey, requestID)

		// Добавляем x-request-id в исходящие метаданные.
		// Клиент увидит его в response headers.
		// metadata.Pairs принимает пары (ключ, значение).
		// Ошибку игнорируем: если не получилось поставить заголовок — не критично.
		_ = grpc.SetHeader(ctx, metadata.Pairs("x-request-id", requestID))

		// Передаём управление дальше по цепочке интерцепторов.
		// handler — это либо следующий интерцептор, либо сам метод сервиса.
		return handler(ctx, req)
	}
}