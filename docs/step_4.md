Шаг 4. Реализация расчёта expected probability
🎯 Цель

Реализовать механизм, который:

    на основе заданных связей рассчитывает “ожидаемую вероятность” (expected) для каждого рынка

1. Входные данные

Система должна использовать:
из БД:

    markets — список рынков

    market_prices — последняя цена (probability)

    relations — связи

Для каждого target-рынка:

target_market_id
→ список всех source_market_id
→ их веса и типы

2. Получение актуальной цены

Для каждого рынка:

👉 берём последнюю запись из market_prices

SELECT probability
FROM market_prices
WHERE market_id = X
ORDER BY timestamp DESC
LIMIT 1

3. Логика расчёта expected
   Базовая формула:

expected = Σ (contribution_i)

Где contribution считается так:
🟢 Positive связь:

contribution = prob_source × weight

🔴 Negative связь:

contribution = (1 - prob_source) × weight

4. Полный алгоритм

Для каждого target-рынка:

relations = get_relations(target_market_id)

expected = 0

for rel in relations:

    prob_source = get_latest_price(rel.source_market_id)

    if rel.type == "positive":
        contribution = prob_source * rel.weight
    else:
        contribution = (1 - prob_source) * rel.weight

    expected += contribution

5. Нормализация (важно)

Если сумма весов ≠ 1:
Вариант 1 (предпочтительно):

expected = expected / sum_of_weights

Вариант 2:

заранее контролировать веса (вручную)
6. Граничные условия

Обязательно:

expected ∈ [0, 1]

Если выходит за границы:

expected = max(0, min(1, expected))

7. Минимальные проверки

Перед расчётом:

    есть ли связи?

    есть ли цены у source рынков?

Если нет связей:

👉 пропускаем рынок
Если нет данных по source:

👉 пропускаем конкретную связь
8. Оптимизация (простая, но важная)

Чтобы не делать лишние запросы:

👉 заранее загрузить все последние цены в память

prices = {market_id: probability}

9. Результат шага

Для каждого рынка получаем:

target_market_id  
expected_probability

10. Сохранение результата (опционально)

Можно сразу писать:

signals (или отдельную таблицу intermediate)

или держать в памяти до следующего шага
11. Definition of Done

Шаг выполнен, если:

✅ для каждого рынка считается expected
✅ учитываются все связи
✅ работает с несколькими факторами
✅ корректно обрабатываются positive/negative
🔚 Коротко для программиста

    Нужно реализовать расчёт expected probability как сумму вкладов от всех связанных рынков с учётом веса и типа связи.

💡 Важно для тебя

Вот здесь модель впервые становится:

данные → логика → число

👉 это и есть “ядро”