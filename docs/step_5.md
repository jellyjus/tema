Шаг 5. Расчёт edge и интерпретация сигнала
🎯 Цель

На основе:

    ожидаемой вероятности (expected)

    рыночной вероятности (market)

получить:

👉 mispricing (edge)
👉 и превратить его в понятный сигнал (что делать)
1. Входные данные

Для каждого рынка:

expected_probability (из Шага 4)  
market_probability (из market_prices)

2. Расчёт edge
   Формула:

edge = expected - market

Интерпретация:

edge > 0 → рынок недооценивает → BUY YES  
edge < 0 → рынок переоценивает → BUY NO

Пример:

expected = 0.65  
market = 0.50

edge = +0.15

👉 сигнал: BUY YES
3. Абсолютное значение (важно)

Для фильтрации:

abs_edge = |edge|

4. Порог сигнала (threshold)
   Ввести параметр:

THRESHOLD = 0.10–0.15

Условие:

if abs(edge) >= THRESHOLD:
signal = True
else:
ignore

5. Направление сделки

if edge > 0:
direction = "BUY YES"
else:
direction = "BUY NO"

6. Сила сигнала (опционально, но полезно)

Можно ввести:

signal_strength = abs(edge)

Или градацию:

0.10–0.15 → слабый  
0.15–0.25 → средний
>0.25 → сильный

7. Минимальные фильтры (обязательно)

Чтобы не ловить мусор:
7.1 Ликвидность

if volume < MIN_VOLUME:
skip

7.2 Наличие данных

если нет expected → пропуск  
если нет market → пропуск

8. Формирование сигнала (структура)

Создаётся объект:

market_id  
market_probability  
expected_probability  
edge  
abs_edge  
direction  
timestamp

9. Псевдокод

for market in markets:

    expected = get_expected(market.id)
    market_price = get_market_price(market.id)

    if expected is None or market_price is None:
        continue

    edge = expected - market_price
    abs_edge = abs(edge)

    if abs_edge < THRESHOLD:
        continue

    if edge > 0:
        direction = "BUY YES"
    else:
        direction = "BUY NO"

    create_signal(
        market_id=market.id,
        expected=expected,
        market=market_price,
        edge=edge,
        abs_edge=abs_edge,
        direction=direction
    )

10. Что НЕ делать на этом этапе

❌ не усложнять формулу edge
❌ не добавлять AI
❌ не делать динамический threshold
❌ не учитывать поведение (это уже следующий слой)
11. Definition of Done

Шаг выполнен, если:

✅ считается edge для рынков
✅ есть фильтрация по threshold
✅ определяется направление
✅ создаются сигналы
🔚 Коротко для программиста

    Нужно реализовать расчёт разницы между expected и рыночной вероятностью и генерировать сигнал, если отклонение превышает заданный порог.