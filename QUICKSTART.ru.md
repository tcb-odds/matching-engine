# Быстрый старт - Подписка на события

## Запуск за 30 секунд

```bash
# Запустить весь стек
docker-compose up

# В консоли увидите:
# - matching-engine запущен на :9000
# - subscriber-client подключился и слушает события
# - grpcui доступен на http://localhost:8080
```

Готово! Теперь:
- Откройте `http://localhost:8080` в браузере (gRPC UI)
- Выберите метод `Process`
- Отправьте ордер и увидите события в логах `subscriber-client`

## Тестирование подписки

### Через grpcurl

```bash
# Terminal 1: Подписаться на события
grpcurl -plaintext localhost:9000 \
  tcb.matching_engine.Engine/SubscribeToOrderUpdates

# Terminal 2: Отправить ордер на покупку
grpcurl -plaintext -d '{
  "ID":"buy-1",
  "Type":"buy",
  "Amount":"10",
  "Price":"50000",
  "Pair":"BTC/USDT"
}' localhost:9000 tcb.matching_engine.Engine/Process

# Terminal 3: Отправить ордер на продажу (сопоставится!)
grpcurl -plaintext -d '{
  "ID":"sell-1",
  "Type":"sell",
  "Amount":"10",
  "Price":"49000",
  "Pair":"BTC/USDT"
}' localhost:9000 tcb.matching_engine.Engine/Process

# В Terminal 1 увидите 2 события "matched"
```

### Через веб-интерфейс (gRPC UI)

1. Откройте: `http://localhost:8080` (grpcui уже запущен вместе с остальными сервисами)
2. В списке методов найдите `SubscribeToOrderUpdates`
3. Нажмите `Invoke` (откроется streaming connection)
4. В другой вкладке отправьте ордера через `Process`
5. Увидите события в реальном времени

## Что получаем в событиях?

### Событие "matched" (сопоставление)

```json
{
  "event_type": "matched",
  "timestamp": 1730044800,
  "pair": "BTC/USDT",
  "order": {
    "ID": "buy-1",
    "Type": "buy",
    "Amount": "10",
    "Price": "50000",
    "FilledAmount": "10"
  },
  "matched_with": {
    "ID": "sell-1",
    "Type": "sell",
    "Amount": "10",
    "Price": "49000"
  },
  "trade_amount": "10",
  "execution_price": "50000"
}
```

### Событие "partially_filled" (частичное исполнение)

```json
{
  "event_type": "partially_filled",
  "timestamp": 1730044801,
  "pair": "BTC/USDT",
  "order": {
    "ID": "buy-1",
    "Amount": "7",        // осталось
    "FilledAmount": "3"   // уже исполнено
  }
}
```

### Событие "cancelled" (отмена)

```json
{
  "event_type": "cancelled",
  "timestamp": 1730044802,
  "pair": "BTC/USDT",
  "order": {
    "ID": "buy-1",
    "Amount": "7",
    "FilledAmount": "3"
  }
}
```

## Сценарий полного теста

### Шаг 1: Запустить все

```bash
docker-compose up
```

### Шаг 2: Подключиться к логам клиента

```bash
docker logs -f subscriber-client
```

### Шаг 3: Отправить серию ордеров

```bash
# Ордер 1: Buy
grpcurl -plaintext -d '{"ID":"1","Type":"buy","Amount":"10","Price":"50000","Pair":"BTC/USDT"}' \
  localhost:9000 tcb.matching_engine.Engine/Process

# Ордер 2: Sell (частичное сопоставление)
grpcurl -plaintext -d '{"ID":"2","Type":"sell","Amount":"5","Price":"49000","Pair":"BTC/USDT"}' \
  localhost:9000 tcb.matching_engine.Engine/Process

# Ордер 3: Отмена оставшегося
grpcurl -plaintext -d '{"ID":"1","Pair":"BTC/USDT"}' \
  localhost:9000 tcb.matching_engine.Engine/Cancel
```

### Шаг 4: Смотрим события в логах

```
EVENT RECEIVED:
  Type: matched
  Order ID: 2
  Filled Amount: 5/5

EVENT RECEIVED:
  Type: partially_filled
  Order ID: 1
  Filled Amount: 5/10

EVENT RECEIVED:
  Type: cancelled
  Order ID: 1
  Filled Amount: 5/10
```

## Кастомизация клиента

### Фильтрация по парам

```go
for {
    event, _ := stream.Recv()
    
    // Обрабатываем только BTC/USDT
    if event.Pair == "BTC/USDT" {
        handleEvent(event)
    }
}
```

### Фильтрация по типу события

```go
switch event.EventType {
case "matched":
    log.Printf("Matched: %s", event.Order.ID)
case "partially_filled":
    log.Printf("Partial: %s", event.Order.ID)
case "cancelled":
    log.Printf("Cancelled: %s", event.Order.ID)
}
```

### Подсчет статистики

```go
stats := map[string]int{"matched": 0, "partially_filled": 0, "cancelled": 0}

for {
    event, _ := stream.Recv()
    stats[event.EventType]++
    
    log.Printf("Stats: matched=%d, partial=%d, cancelled=%d",
        stats["matched"], stats["partially_filled"], stats["cancelled"])
}
```

## Troubleshooting

### "connection refused"

```bash
# Проверить что сервер запущен
docker ps | grep matching-engine

# Проверить порт
netstat -an | grep 9000
```

### "stream terminated"

- Нормально! Перезапустите клиента - он реконнектится
- Сервер был перезапущен - все стримы закрылись

### Не видно событий

```bash
# Проверить логи сервера
docker logs matching-engine

# Должно быть:
# New subscriber connected: <uuid> (total: 1)
```

### Docker Compose не запускается

```bash
# Проверить версию
docker-compose --version

# Пересобрать образы
docker-compose build --no-cache
docker-compose up
```

## Дальше

- Полная документация: [SUBSCRIPTION.md](SUBSCRIPTION.md)
- Русский README: [README.ru.md](README.ru.md)
- Proto API: [matching-engine.proto](pkg/proto/matching-engine.proto)

## Полезные команды

```bash
# Запустить только сервер
docker-compose up matching-engine

# Запустить в фоне
docker-compose up -d

# Остановить все
docker-compose down

# Посмотреть логи
docker-compose logs -f matching-engine
docker-compose logs -f subscriber-client

# Пересобрать после изменений
docker-compose up --build

# Очистить все
docker-compose down -v
docker system prune -a
```

## Готово

Теперь вы можете:
- Подписываться на события ордеров
- Получать real-time обновления
- Фильтровать события по парам
- Интегрировать в свой сервис

Вопросы? Смотрите [SUBSCRIPTION.md](SUBSCRIPTION.md) для детальной информации.

