# Matching Engine - Быстрый движок сопоставления ордеров

Высокопроизводительный движок сопоставления ордеров (Order Matching Engine) на Go с поддержкой **real-time подписки** на события через gRPC Streaming.

## Возможности

- **Быстрое сопоставление** - использует Splay Trees для O(log n) операций
- **Real-time подписки** - gRPC Server-Side Streaming (HTTP/2)
- **Частичное исполнение** - поддержка частично исполненных ордеров
- **Лимитные и рыночные ордера** - оба типа поддерживаются
- **Thread-safe** - безопасная работа в многопоточной среде
- **Docker ready** - готовые образы и docker-compose

## Установка

### Требования

- Docker и Docker Compose

## Быстрый старт

```bash
# Запустить весь стек (сервер + тестовый клиент + grpcui)
docker-compose up

# Или только сервер
docker-compose up matching-engine

# В фоне
docker-compose up -d
```

Сервисы:
- **Matching Engine**: `localhost:9000` (gRPC/HTTP2)
- **gRPC UI**: `http://localhost:8080` (веб-интерфейс для тестирования)

## Подписка на события

### Что можно получать?

- **matched** - ордера сопоставлены
- **partially_filled** - ордер частично исполнен
- **cancelled** - ордер отменен

### Пример клиента (Go)

```go
package main

import (
    "context"
    "log"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    proto "github.com/tcb-odds/matching-engine/pkg/proto"
)

func main() {
    // Подключение
    conn, _ := grpc.Dial("localhost:9000", 
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    defer conn.Close()

    client := proto.NewEngineClient(conn)

    // Подписка на ВСЕ события
    stream, _ := client.SubscribeToOrderUpdates(
        context.Background(), 
        &proto.SubscribeRequest{},
    )

    // Получение событий
    for {
        event, err := stream.Recv()
        if err != nil {
            break
        }

        log.Printf("[%s] %s - Pair: %s, Order: %s", 
            event.EventType, 
            event.Order.ID,
            event.Pair,
            event.Order.FilledAmount,
        )
    }
}
```

### Запуск тестового клиента

```bash
docker-compose up subscriber-client
```

## API Методы

### 1. Process - Лимитный ордер

```bash
grpcurl -plaintext -d '{
  "ID":"order-1",
  "Type":"buy",
  "Amount":"10",
  "Price":"50000",
  "Pair":"BTC/USDT"
}' localhost:9000 tcb.matching_engine.Engine/Process
```

### 2. ProcessMarket - Рыночный ордер

```bash
grpcurl -plaintext -d '{
  "ID":"order-2",
  "Type":"sell",
  "Amount":"5",
  "Price":"0",
  "Pair":"BTC/USDT"
}' localhost:9000 tcb.matching_engine.Engine/ProcessMarket
```

### 3. Cancel - Отмена ордера

```bash
grpcurl -plaintext -d '{
  "ID":"order-1",
  "Pair":"BTC/USDT"
}' localhost:9000 tcb.matching_engine.Engine/Cancel
```

### 4. FetchBook - Получить книгу ордеров

```bash
grpcurl -plaintext -d '{
  "pair":"BTC/USDT",
  "limit":10
}' localhost:9000 tcb.matching_engine.Engine/FetchBook
```

### 5. SubscribeToOrderUpdates - Подписка на события

```bash
grpcurl -plaintext \
  localhost:9000 \
  tcb.matching_engine.Engine/SubscribeToOrderUpdates
```

## Структура проекта

```
matching-engine/
├── cmd/
│   └── main.go                 # Точка входа
├── internal/
│   └── app/
│       ├── engine/             # Логика matching engine
│       │   ├── order_book.go   # Книга ордеров
│       │   ├── order.go        # Структура ордера
│       │   ├── process_*.go    # Логика сопоставления
│       │   └── *_test.go       # Тесты
│       ├── server/             # gRPC сервер
│       │   ├── engine.go       # Обработчики RPC
│       │   └── subscription_manager.go  # Менеджер подписок
│       └── util/               # Утилиты (BigDecimal)
├── pkg/
│   └── proto/                  # Proto файлы и генерированный код
│       ├── matching-engine.proto
│       ├── matching-engine.pb.go
│       └── matching-engine_grpc.pb.go
├── examples/
│   └── subscriber_client.go    # Пример клиента
├── docker-compose.yaml         # Docker Compose конфигурация
├── Dockerfile                  # Dockerfile для сервера
├── Dockerfile.client           # Dockerfile для клиента
├── SUBSCRIPTION.md             # Подробная документация по подпискам
└── README.ru.md               # Этот файл
```

## Как работает сопоставление?

### Пример: Полное сопоставление

```
Шаг 1: Process(Buy 10 BTC @ 50000)
  → Добавлен в BuyTree
  Книга: BUY: 50000 → 10 BTC

Шаг 2: Process(Sell 10 BTC @ 49000)
  → Находит Buy @ 50000 (цена подходит!)
  → Сопоставление: 10 BTC @ 50000
  
События подписчикам:
  1. matched: Buy order (полностью)
  2. matched: Sell order (полностью)
  
Книга: (пусто)
```

### Пример: Частичное исполнение

```
Шаг 1: Process(Buy 10 BTC @ 50000)
  Книга: BUY: 50000 → 10 BTC

Шаг 2: Process(Sell 3 BTC @ 49000)
  → Сопоставление: 3 BTC @ 50000
  
События подписчикам:
  1. matched: Sell order (полностью, 3 BTC)
  2. partially_filled: Buy order (осталось 7 BTC)
  
Книга: BUY: 50000 → 7 BTC
```

## HTTP/2 и gRPC Streaming

Проект использует **HTTP/2** для gRPC streaming, что обеспечивает:

- **Мультиплексирование** - несколько стримов на одном соединении
- **Бинарный протокол** - эффективная передача данных
- **Server Push** - сервер может отправлять данные без запроса
- **Низкая latency** - < 1ms от события до клиента

### Требования к клиентам

- Поддержка HTTP/2 (все современные gRPC библиотеки)
- Для браузеров: gRPC-Web с прокси (envoy)

## Производительность

- **Latency**: < 1ms на операцию сопоставления
- **Throughput**: > 10,000 ордеров/сек
- **Подписчики**: Неограниченно (ограничено RAM)
- **Overhead на подписчика**: ~100KB RAM

## Документация

- **[SUBSCRIPTION.md](SUBSCRIPTION.md)** - Полная документация по подпискам
- **[Proto файлы](pkg/proto/matching-engine.proto)** - API определения
- **[Примеры](examples/)** - Примеры клиентов

## Changelog

### v1.1.0 (2025-10-27) - Подписки
- Добавлен gRPC streaming для подписки на события
- Добавлено поле `FilledAmount` в Order
- Реализован SubscriptionManager
- Docker Compose конфигурация
- Документация на русском

### v1.0.0 (Initial)
- Базовый matching engine
- Лимитные и рыночные ордера
- Частичное исполнение
- gRPC API

## Лицензия

Этот проект распространяется под лицензией MIT.

## Ссылки

- [gRPC Documentation](https://grpc.io/docs/)
- [Protocol Buffers](https://protobuf.dev/)
- [HTTP/2 Specification](https://http2.github.io/)

---

**Автор:** TCB Odds Team  
**Email:** support@tcb-odds.com  
**Версия:** 1.1.0

