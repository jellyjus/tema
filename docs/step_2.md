🔥 Шаг 2. Структура хранения и схема БД
🎯 Цель

Создать структуру данных, которая:

    хранит рынки и их историю

    позволяет быстро считать модель

    легко расширяется дальше

1. Общий подход

На MVP:

👉 простая реляционная БД (рекомендация: PostgreSQL или SQLite)

Важно:

    не усложнять

    но сразу заложить масштабируемость

2. Основные таблицы
   📊 2.1 Таблица markets (справочник рынков)

Хранит уникальные рынки

id (string / primary key)
title (text)
created_at (timestamp)

📈 2.2 Таблица market_prices (история цен)

Ключевая таблица

id (auto)
market_id (foreign key → markets.id)
probability (float 0–1)
volume (float)
timestamp (timestamp)

Почему отдельно?

👉 чтобы хранить историю, а не перезаписывать
🔗 2.3 Таблица relations (связи между рынками)

Основа модели

id
target_market_id (куда влияет)
source_market_id (что влияет)
weight (float 0–1)
relation_type (positive / negative)

🧠 2.4 Таблица signals (результат модели)

Пока можно заложить сразу

id
market_id
expected_probability
market_probability
edge
adjusted_edge
direction (YES / NO)
timestamp

📊 2.5 Таблица market_behavior (поведение рынка)

Для будущего (можно MVP-lite)

id
market_id
price_change (float)
volume_change (float)
volatility (float)
sentiment_score (float)
timestamp

3. Связи между таблицами

markets 1 → N market_prices  
markets 1 → N relations (как source и target)  
markets 1 → N signals  
markets 1 → N market_behavior

4. Индексы (важно)

Обязательно:

market_prices.market_id  
market_prices.timestamp

relations.target_market_id  
relations.source_market_id

5. Что важно учесть
   5.1 История — критична

Нельзя хранить только текущую цену
👉 нужна динамика
5.2 Нормализация

    probability всегда 0–1

    единый формат времени

5.3 Простота

Не добавлять:
❌ сложные типы
❌ лишние таблицы
❌ агрегации на уровне БД
6. Минимальный сценарий работы

1. получили данные
2. записали в market_prices
3. (если новый рынок) → добавили в markets

7. Пример записи
   markets:

id: "btc_up"
title: "Will BTC rise this month?"

market_prices:

market_id: "btc_up"
probability: 0.62
volume: 12000
timestamp: 2026-04-24 12:00

8. Definition of Done

Шаг выполнен, если:

✅ есть таблицы
✅ данные сохраняются
✅ можно получить:

    последнюю цену

    историю цен

🔚 Коротко для программиста

    Нужно спроектировать БД с разделением:
    — рынки (справочник)
    — история цен
    — связи
    — сигналы

    Ключевое: хранить историю и обеспечить быстрый доступ к последним данным.