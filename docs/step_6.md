Шаг 6. Поведенческий слой (корректировка сигнала)
🎯 Цель

Улучшить качество сигналов за счёт понимания:

    движение цены вызвано толпой или более “осмысленными” действиями

1. Входные данные

Используем из market_prices:

текущая цена  
предыдущая цена  
объём  
предыдущий объём

2. Что нужно посчитать
   2.1 Изменение цены

price_change = current_price - previous_price

2.2 Изменение объёма

volume_change = current_volume - previous_volume

2.3 “Резкость” движения (упрощённо)

sharp_move = abs(price_change) > PRICE_THRESHOLD

Пример:

PRICE_THRESHOLD = 0.05 (5%)

3. Классификация поведения (MVP-логика)

Очень простая:

if sharp_move and volume_change > 0:
behavior = "crowd"
else:
behavior = "neutral"

Интерпретация:

    crowd → эмоциональное движение

    neutral → нет явного сигнала

(пока НЕ пытаемся ловить “умные деньги” — это позже)
4. Введение коэффициента confidence

Теперь добавляем:

confidence

Логика:

if behavior == "crowd":
confidence = 1.1
else:
confidence = 1.0

5. Корректировка edge
   Формула:

adjusted_edge = edge × confidence

Пример:

edge = 0.12  
behavior = crowd

adjusted_edge = 0.12 × 1.1 = 0.132

6. Обновление фильтра

Теперь проверяем:

abs(adjusted_edge) ≥ THRESHOLD

👉 важно: используем adjusted_edge, а не обычный edge
7. Обновление сигнала

В сигнал добавляем:

behavior (crowd / neutral)  
confidence  
adjusted_edge

Итоговая структура сигнала:

market_id  
market_probability  
expected_probability  
edge  
adjusted_edge  
direction  
behavior  
timestamp

8. Псевдокод

for market in markets:

    edge = get_edge(market)

    price_change = get_price_change(market)
    volume_change = get_volume_change(market)

    if abs(price_change) > PRICE_THRESHOLD and volume_change > 0:
        confidence = 1.1
        behavior = "crowd"
    else:
        confidence = 1.0
        behavior = "neutral"

    adjusted_edge = edge * confidence

    if abs(adjusted_edge) < THRESHOLD:
        continue

    generate_signal(
        market_id=market.id,
        edge=edge,
        adjusted_edge=adjusted_edge,
        behavior=behavior
    )

9. Что важно НЕ делать

❌ не усложнять поведенку
❌ не пытаться точно определять “умные деньги”
❌ не вводить сложные метрики

👉 MVP = простая эвристика
10. Definition of Done

Шаг выполнен, если:

✅ считается изменение цены и объёма
✅ определяется поведение (crowd / neutral)
✅ применяется коэффициент
✅ используется adjusted_edge
🔚 Коротко для программиста

    Нужно добавить простой анализ изменения цены и объёма и на его основе корректировать силу сигнала через коэффициент.