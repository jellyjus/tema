# tema — Документация

Детектор сигналов prediction-маркетов. Получает данные Polymarket, строит многофакторные вероятностные модели на основе пользовательских связей между рынками, рассчитывает edge и генерирует сигналы с размером позиции и трекингом P&L.

## Содержание

- [Запуск](#запуск)
- [Переменные окружения](#переменные-окружения)
- [Архитектура и поток данных](#архитектура-и-поток-данных)
- [Доменные концепции](#доменные-концепции)
- [HTTP API](#http-api)
- [Веб-интерфейс](#веб-интерфейс)
- [Схема базы данных](#схема-базы-данных)
- [Алгоритмы](#алгоритмы)
- [Жизненный цикл сделки](#жизненный-цикл-сделки)
- [Деплой за прокси](#деплой-за-прокси)
- [Структура проекта](#структура-проекта)

---

## Запуск

```bash
# Требования: Go 1.26+, PostgreSQL 14+

# Создать базу
createdb tema

# Запуск (DB креды по умолчанию)
go run cmd/tema/main.go

# Или с кастомными переменными
DATABASE_URL="postgres://user:pass@localhost:5432/tema?sslmode=disable" \
BANKROLL=5000 RISK_K=0.3 go run cmd/tema/main.go

# Сборка
go build -o tema ./cmd/tema/main.go
```

Сервис автоматически создаёт таблицы при старте (миграции через `db.Migrate()`). Демон каждые `FETCH_INTERVAL` секунд получает данные с Polymarket и пересчитывает сигналы.

---

## Переменные окружения

| Переменная               | По умолчанию                                                        | Описание                                       |
|--------------------------|---------------------------------------------------------------------|-------------------------------------------------|
| `DATABASE_URL`           | `postgres://postgres:pass@localhost:5432/tema?sslmode=disable`       | Строка подключения к PostgreSQL                |
| `FETCH_INTERVAL`         | `60`                                                                 | Секунд между обновлениями данных с Polymarket   |
| `PORT`                   | `8080`                                                               | Порт HTTP-сервера                                |
| `SIGNAL_THRESHOLD`      | `0.01`                                                               | Минимальный `|adjusted_edge|` для сигнала         |
| `MIN_VOLUME`             | `0`                                                                  | Минимальный объём рынка для включения в сигналы  |
| `PRICE_CHANGE_THRESHOLD` | `0.05`                                                               | Порог изменения цены для флага «crowd»          |
| `BANKROLL`               | `1000`                                                               | Общий капитал для расчёта размера ставки ($)      |
| `RISK_K`                 | `0.5`                                                                | Коэффициент риска для расчёта размера ставки      |
| `BASE_PATH`              | `/tema`                                                              | URL-префикс для фронтенда (прокси снимает его)  |

---

## Архитектура и поток данных

```
Polymarket API
      │
      ▼
  Fetcher ──► Markets + Prices (DB)
      │
      ▼
  Relations (user-defined) ──► Modeler ──► Expected Probability
                                              │
                                              ▼
                                        Signaler ──► Edge + Direction
                                              │
                                              ▼
                                      Behavior Analyzer ──► Adjusted Edge
                                              │
                                              ▼
                                         Sizer ──► Bet Size
                                              │
                              ┌───────────────┼───────────────┐
                              ▼                               ▼
                         Signals (DB)                    Trades (DB)
                              │
                              ▼
                     Dashboard (Vue 3)
```

Каждый цикл:
1. Fetcher получает активные рынки с Polymarket
2. Цены записываются в БД (append-only)
3. Из связей (relations) вычисляется ожидаемая вероятность для каждого целевого рынка
4. Edge = expected − market probability
5. Поведенческий анализ даёт confidence-множитель (crowd = 1.1, neutral = 1.0)
6. Adjusted edge = edge × confidence
7. Фильтрация по порогу и объёму
8. Расчёт размера ставки (sizer)
9. Сигналы сохраняются (с очисткой предыдущих), сделки создаются автоматически

---

## Доменные концепции

### Рынок (Market)
Событие на Polymarket. Вероятность = цена YES (0–1). Например: «Будет ли Трамп президентом?» с probability = 0.62.

### Связь (Relation)
Направленная причинно-следственная связь `источник → цель` с типом и весом:
- **positive**: рост вероятности источника → рост вероятности цели
- **negative**: рост вероятности источника → снижение вероятности цели
- **weight** (0–1): сила связи

Пример: «FDA одобряет препарат X» (positive, 0.8) → «Акции компании Y вырастут»

### Ожидаемая вероятность (Expected Probability)
```
expected = Σ(prob_positive × weight_positive) + Σ((1 - prob_negative) × weight_negative)
           ──────────────────────────────────────────────────────────────────────
                              Σ(all weights)
```
Клэмпится в [0, 1].

### Edge
```
edge = expected_probability − market_probability
```
Положительный → BUY YES, отрицательный → BUY NO. Порог: `|adjusted_edge| ≥ SIGNAL_THRESHOLD`.

### Adjusted Edge
```
adjusted_edge = edge × confidence
```
- `confidence = 1.0` — нейтральное поведение рынка
- `confidence = 1.1` — crowd (цена резко изменилась с ростом объёма)

### Размер ставки (Bet Size)
```
bet = bankroll × k × |adjusted_edge|

Если probability < 0.1 или > 0.9:
    bet *= 0.5    (скидка за экстремальную вероятность)

bet = clamp(bet, bankroll × 1%, bankroll × 5%)

Если Σ(all bets) > bankroll × 25%:
    пропорциональное уменьшение всех ставок
```

### Сила сигнала
| Adjusted Edge | Сила    |
|---------------|---------|
| ≥ 0.25        | strong  |
| 0.15 – 0.25   | medium  |
| < 0.15        | weak    |

### Сделка (Trade)
Paper trade — отслеживание без реальных ставок:
- `open`: автоматически создаётся из сигнала
- `closed`: закрывается пользователем с указанием exit_price

---

## HTTP API

Бекенд обслуживает роуты без префикса. Прокси (например, nginx) снимает `BASE_PATH` и прокидывает запросы.

### Рынки

| Метод | Путь          | Описание                          |
|-------|---------------|-----------------------------------|
| GET   | `/api/markets`| Список всех сохранённых рынков     |

**Пример ответа:**
```json
[{"id": "abc123", "title": "Will X happen?", "created_at": "2025-04-28T12:00:00Z"}]
```

### Связи

| Метод  | Путь               | Описание                                |
|--------|--------------------|-----------------------------------------|
| GET    | `/api/relations`   | Список всех связей (с названиями рынков) |
| POST   | `/api/relations`   | Создать связь                            |
| DELETE | `/api/relations/{id}` | Удалить связь                         |

**POST body:**
```json
{
  "source_market_id": "abc123",
  "target_market_id": "def456",
  "relation_type": "positive",
  "weight": 0.8
}
```

**GET ответ:**
```json
[{
  "id": 1,
  "source_market_id": "abc123",
  "source_market_title": "Will X happen?",
  "target_market_id": "def456",
  "target_market_title": "Will Y happen?",
  "relation_type": "positive",
  "weight": 0.8
}]
```

### Цены

| Метод | Путь                | Описание                           |
|-------|---------------------|-------------------------------------|
| GET   | `/api/prices/latest`| Последняя цена по каждому рынку     |

### Сигналы

| Метод | Путь                 | Описание                             |
|-------|----------------------|---------------------------------------|
| GET   | `/api/signals?limit=50` | Список последних сигналов (с bet_size) |

**Пример ответа:**
```json
[{
  "id": 42,
  "market_id": "abc123",
  "title": "Will X happen?",
  "market_probability": 0.45,
  "expected_probability": 0.62,
  "edge": 0.17,
  "adjusted_edge": 0.187,
  "direction": "BUY YES",
  "behavior": "neutral",
  "bet_size": 93.5,
  "timestamp": "2025-04-28 12:00:05"
}]
```

> Сигналы пересоздаются каждый fetch-цикл (старые удаляются, новые вставляются).

### Сделки

| Метод | Путь                      | Описание                                         |
|-------|---------------------------|--------------------------------------------------|
| GET   | `/api/trades?limit=100`   | Список сделок (с названиями рынков)              |
| POST  | `/api/trades`              | Открыть сделку                                    |
| POST  | `/api/trades/{id}/close`   | Закрыть сделку, указав `exit_price` (0–1)         |
| GET   | `/api/trades/metrics`      | Агрегированные метрики: PnL, ROI, win rate        |

**POST /api/trades body:**
```json
{
  "market_id": "abc123",
  "direction": "BUY YES",
  "entry_price": 0.45,
  "bet_size": 93.5
}
```

**POST /api/trades/{id}/close body:**
```json
{"exit_price": 0.85}
```

> При закрытии PnL рассчитывается автоматически: выиграл → `bet × (1 − entry)`, проиграл → `−bet × entry`.

**GET /api/trades/metrics ответ:**
```json
{
  "total_pnl": 127.50,
  "roi": 0.142,
  "win_rate": 0.65,
  "total_trades": 10,
  "wins": 6,
  "losses": 4
}
```

### Dashboard

| Метод | Путь | Описание                    |
|-------|------|------------------------------|
| GET   | `/`  | HTML-дашборд (Vue 3, no build)|

---

## Веб-интерфейс

Дашборд доступен по адресу `http://{host}:{port}{BASE_PATH}/`. Четыре вкладки:

### 1. Сигналы

Таблица с текущими сигналами, обновляется по кнопке «Обновить»:

| Столбец      | Описание                                        |
|--------------|--------------------------------------------------|
| Рынок        | Название рынка (обрезанное, полный текст в tip) |
| Маркет.       | Рыночная вероятность                             |
| Ожид.        | Ожидаемая вероятность (из модели)                |
| Edge          | `expected − market`                              |
| Adj.         | Adjusted edge (с confidence-множителем)          |
| Напр.         | BUY YES / BUY NO (цветные теги)                 |
| Ставка        | Размер ставки в $ (из sizer)                     |
| Сила          | weak / medium / strong (цветовая кодировка)     |
| Повед.        | neutral / crowd (цветной тег)                    |
| Время         | Timestamp сигнала                                 |

Сигналы отсортированы по `|adjusted_edge|` по убыванию.

### 2. Связи

CRUD-интерфейс для управления причинно-следственными связями:

- **Форма добавления**: поля Источник, Цель (с автодополнением по названиям рынков из datalist), Тип (positive/negative), Вес (0.01–1.0)
- **Таблица связей**: ID, Названия источника и цели, Тип (цветной тег), Вес, Кнопка удаления (✕)
- Кнопка «Обновить» для обновления списка

### 3. Сделки

Полный трекинг P&L:

**Метрики (карточки сверху):**
| Метрика   | Описание                            |
|-----------|--------------------------------------|
| PnL       | Суммарный PnL в $ (зелёный/красный) |
| ROI       | Return on investment в %             |
| Win Rate  | Процент выигрышных сделок             |
| Всего     | Общее количество сделок               |
| Win/Loss  | Соотношение выигрышей и проигрышей    |

**Таблица сделок:**

| Столбец | Описание                                             |
|---------|-------------------------------------------------------|
| Рынок   | Название рынка                                        |
| Напр.   | BUY YES / BUY NO                                      |
| Вход    | Цена входа в центах                                   |
| Выход   | Цена закрытия в центах (или — если открыта)            |
| Ставка  | Размер ставки в $                                     |
| PnL     | Прибыль/убыток в $ (цветной)                          |
| Статус  | open (голубой) / closed (серый)                         |
| Открыт  | Время открытия                                         |
|         | Кнопка «Закрыть» с полем для exit_price (для open)     |

**Закрытие сделки**: вводите `exit_price` (0–1), где 1 = событие произошло, 0 = нет. PnL рассчитывается автоматически.

### 4. Рынки

Список всех рынков, загруженных из Polymarket:

| Столбец | Описание            |
|---------|----------------------|
| ID      | Обрезанный ID рынка |
| Название| Полное название     |
| Создан  | Дата добавления     |

---

## Схема базы данных

Миграции выполняются автоматически при старте (`db.Migrate()`).

### markets
```sql
id TEXT PRIMARY KEY
title TEXT NOT NULL
created_at TIMESTAMPTZ NOT NULL DEFAULT now()
```

### market_prices (append-only time-series)
```sql
id BIGSERIAL PRIMARY KEY
market_id TEXT REFERENCES markets(id)
probability DOUBLE PRECISION CHECK (probability >= 0 AND probability <= 1)
volume DOUBLE PRECISION DEFAULT 0
timestamp TIMESTAMPTZ DEFAULT now()
-- Индексы: market_id, timestamp
```

### relations
```sql
id BIGSERIAL PRIMARY KEY
source_market_id TEXT REFERENCES markets(id)
target_market_id TEXT REFERENCES markets(id)
relation_type TEXT CHECK (relation_type IN ('positive', 'negative'))
weight DOUBLE PRECISION CHECK (weight > 0 AND weight <= 1)
UNIQUE(source_market_id, target_market_id)
-- Индексы: target_market_id, source_market_id
```

### signals (пересоздаются каждый цикл)
```sql
id BIGSERIAL PRIMARY KEY
market_id TEXT REFERENCES markets(id)
market_probability DOUBLE PRECISION
expected_probability DOUBLE PRECISION
edge DOUBLE PRECISION
adjusted_edge DOUBLE PRECISION DEFAULT 0
direction TEXT CHECK (direction IN ('BUY YES', 'BUY NO'))
behavior TEXT DEFAULT 'neutral' CHECK (behavior IN ('crowd', 'neutral'))
bet_size DOUBLE PRECISION DEFAULT 0
timestamp TIMESTAMPTZ DEFAULT now()
```

### market_behavior
```sql
id BIGSERIAL PRIMARY KEY
market_id TEXT REFERENCES markets(id)
price_change DOUBLE PRECISION DEFAULT 0
volume_change DOUBLE PRECISION DEFAULT 0
volatility DOUBLE PRECISION DEFAULT 0
sentiment_score DOUBLE PRECISION DEFAULT 0
timestamp TIMESTAMPTZ DEFAULT now()
```

### trades
```sql
id BIGSERIAL PRIMARY KEY
market_id TEXT REFERENCES markets(id)
direction TEXT CHECK (direction IN ('BUY YES', 'BUY NO'))
entry_price DOUBLE PRECISION NOT NULL
exit_price DOUBLE PRECISION          -- NULL для открытых
bet_size DOUBLE PRECISION NOT NULL
pnl DOUBLE PRECISION                 -- NULL для открытых
status TEXT DEFAULT 'open' CHECK (status IN ('open', 'closed'))
timestamp_open TIMESTAMPTZ DEFAULT now()
timestamp_close TIMESTAMPTZ          -- NULL для открытых
```

---

## Алгоритмы

### Ожидаемая вероятность (modeler)

Для каждого целевого рынка собираются все входящие связи:

```
expected_num = Σ(source_prob × weight)           для positive
             + Σ((1 − source_prob) × weight)      для negative

expected_den = Σ(weights)

expected = clamp(expected_num / expected_den, 0, 1)
```

### Поведенческий анализ (behavior)

Сравниваются текущие и предыдущие цены:
```
price_change = current_prob − previous_prob
volume_change = current_volume − previous_volume

Если |price_change| > threshold И volume_change > 0:
    behavior = crowd,    confidence = 1.1
Иначе:
    behavior = neutral,  confidence = 1.0
```

### Расчёт размера ставки (sizer)

```
bet = bankroll × k × |adjusted_edge|

Если probability < 0.1 или > 0.9:
    bet *= 0.5

bet = max(bet, bankroll × 1%)
bet = min(bet, bankroll × 5%)

Если Σ(all bets) > bankroll × 25%:
    все ставки пропорционально уменьшаются
```

### Расчёт PnL (при закрытии сделки)

```
BUY YES:
    если exit_price >= 0.5 (событие произошло):
        PnL =  bet_size × (1 − entry_price)
    иначе:
        PnL = −bet_size × entry_price

BUY NO:
    если exit_price < 0.5 (событие НЕ произошло):
        PnL =  bet_size × (1 − entry_price)
    иначе:
        PnL = −bet_size × entry_price
```

---

## Жизненный цикл сделки

1. **Сигнал сгенерирован** → автоматическое создание `open` trade с `entry_price = market_probability`, `bet_size` из sizer
2. **Дедупликация** — один открытый trade на market (`HasOpenTrade` проверка)
3. **Пользователь закрывает** через UI — вводит `exit_price` (0–1, разрешение события)
4. **PnL автоматически рассчитывается** при закрытии
5. **Метрики** — агрегированные PnL, ROI, win rate доступны на дашборде и через API

> Сделки — paper trades, реальные ставки на Polymarket НЕ совершаются.

---

## Деплой за прокси

Сервис предназначен для работы за reverse proxy (nginx, Caddy и т.д.) с префиксом пути.

```nginx
# nginx пример
location /tema/ {
    proxy_pass http://127.0.0.1:8080/;
}
```

Фронтенд получает `BASE_PATH` из env и подставляет его во все API-вызовы. Бекенд обслуживает роуты без префикса (`/api/...`, `/`), прокси снимает `/tema`.

- `BASE_PATH=/tema` (default) → фронтенд шлёт запросы на `/tema/api/...`
- `BASE_PATH=` (пустая строка) → фронтенд шлёт запросы на `/api/...` (без префикса)

---

## Структура проекта

```
cmd/tema/main.go            — entrypoint: daemon + HTTP server + signal pipeline
internal/
  behavior/behavior.go      — crowd detection, confidence multiplier
  config/config.go           — env config (DATABASE_URL, FETCH_INTERVAL, PORT, SIGNAL_THRESHOLD, MIN_VOLUME, PRICE_CHANGE_THRESHOLD, BANKROLL, RISK_K, BASE_PATH)
  db/migrate.go              — schema migrations (6 таблиц + ALTER for bet_size)
  db/store.go                — все DB операции
  fetcher/fetcher.go         — Polymarket Gamma API клиент
  model/model.go             — domain types (Market, MarketPrice, Relation, RelationInput, Signal, Trade, TradeStatus, SignalDirection)
  modeler/modeler.go          — expected probability calculation
  server/server.go           — HTTP API handlers + //go:embed index.html + base path injection
  server/index.html           — Vue 3 dashboard (4 вкладки, CDN, no build step)
  signaler/signaler.go       — edge, direction, threshold filtering, strength tiers
  sizer/sizer.go             — position sizing (bankroll, k-factor, min/max, exposure cap)
```

### Зависимости

- `github.com/HuakunShen/polymarket-kit/go-client/gamma` — Polymarket data fetching (read-only, без авторизации)
- `github.com/jackc/pgx/v5/pgxpool` — PostgreSQL драйвер