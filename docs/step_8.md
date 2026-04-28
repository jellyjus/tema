Шаг 8. Position Sizing (размер ставки)
🎯 Цель

Определить:

    сколько ставить на каждый сигнал, а не просто “входить или нет”

1. Входные данные

Для каждого сигнала:

adjusted_edge  
market_probability  
signal_strength

    глобально:

total_bankroll (общий капитал)

2. Базовый принцип

   Чем выше edge → тем больше ставка
   Чем ниже edge → тем меньше ставка

3. Простейшая модель (MVP)
   Фиксированная доля от банка

bet_size = bankroll × k × abs(adjusted_edge)

Где:

k = коэффициент риска (например 0.5)

Пример:

bankroll = $1000  
adjusted_edge = 0.15

bet = 1000 × 0.5 × 0.15 = $75

4. Ограничения (обязательно)
   4.1 Максимальный размер ставки

max_bet = 5% от bankroll

4.2 Минимальный размер

min_bet = 1% от bankroll

Итог:

bet = bankroll * k * abs(adjusted_edge)

bet = min(bet, max_bet)
bet = max(bet, min_bet)

5. Учёт вероятности (важный нюанс)

Рынки с вероятностью:

0.05 или 0.95 → рискованные

Ограничение:

if market_probability < 0.1 or market_probability > 0.9:
bet *= 0.5

6. Учёт количества сигналов

Если сигналов много:

👉 нельзя ставить везде по максимуму
Ввод:

max_total_exposure = 20–30% bankroll

Логика:

    считаем все ставки

    если превышает лимит → пропорционально уменьшаем

7. Приоритизация

Если сигналов больше, чем бюджет:

сортируем по adjusted_edge  
берём топ-N

8. Псевдокод

signals = sort_by_edge(signals)

for signal in signals:

    bet = bankroll * k * abs(signal.adjusted_edge)

    if signal.market_probability < 0.1 or > 0.9:
        bet *= 0.5

    bet = clamp(bet, min_bet, max_bet)

    assign_bet(signal, bet)

9. Что НЕ делать в MVP

❌ Kelly criterion (пока рано)
❌ сложные risk-модели
❌ корреляции между рынками
❌ динамическое управление портфелем
10. Definition of Done

Шаг выполнен, если:

✅ для каждого сигнала считается размер ставки
✅ есть ограничения (min/max)
✅ учитывается bankroll
✅ есть контроль общего риска
🔚 Коротко для программиста

    Нужно реализовать простую функцию, которая рассчитывает размер ставки на основе adjusted_edge и ограничивает риск через лимиты.