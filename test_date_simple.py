from datetime import datetime
from dateutil import parser

def format_date(date_str):
    """Форматирует дату из строки ISO в формат DD.MM.YYYY"""
    try:
        print(f"Input date: {date_str}")
        print(f"Input type: {type(date_str)}")
        
        if not date_str:
            print("Empty date string")
            return ""
        
        if isinstance(date_str, str):
            # Удаляем возможный суффикс г.
            date_str = date_str.replace(" г.", "")
            has_timezone = any(x in date_str for x in ['+', '-', 'Z'])
            print(f"Has timezone: {has_timezone}")
            print(f"Cleaned date string: {date_str}")
        
        date_obj = parser.parse(date_str)
        print(f"Parsed date: {date_obj}")
        
        formatted = date_obj.strftime("%d.%m.%Y")
        print(f"Formatted result: {formatted}")
        return formatted
    except Exception as e:
        print(f"Error: {str(e)}")
        return date_str

# Тестовые даты
test_dates = [
    "2025-02-13T17:10:17.02749+03:00",  # Наша проблемная дата
    "2025-02-13T17:10:17Z",             # С UTC
    "2025-02-13",                       # Только дата
    "2025-02-13 г.",                    # С суффиксом
    "",                                 # Пустая строка
    None                                # None
]

if __name__ == "__main__":
    print("Testing date formatting:")
    print("-" * 50)
    
    for date in test_dates:
        print("\nTesting date:", date)
        result = format_date(date)
        print(f"Final result: {result}")
        print("-" * 50) 