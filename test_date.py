from datetime import datetime

def format_date(date_str):
    """Форматирует дату из строки ISO в формат DD.MM.YYYY."""
    try:
        if not date_str:
            return ""
        date_obj = datetime.fromisoformat(date_str.replace('Z', '+00:00'))
        return date_obj.strftime("%d.%m.%Y")
    except Exception as e:
        print(f"Error formatting date {date_str}: {e}")
        return date_str

# Тестовые даты
dates = [
    "2025-02-13T17:10:17.02749+03:00",  # creationDate из запроса
    None,  # пустая дата
    "",    # пустая строка
    "2025-02-13",  # только дата
    "2025-02-13T17:10:17Z",  # с Z в конце
]

print("Testing date formatting:")
for date in dates:
    result = format_date(date)
    print(f"Input:  {date}")
    print(f"Output: {result}")
    print("-" * 50) 