#!/usr/bin/env python3
import sys
from datetime import datetime

def format_date(date_str):
    """Форматирует дату из строки ISO в формат DD.MM.YYYY"""
    try:
        print(f"Starting date formatting for: {date_str}")
        if not date_str:
            print("Empty date string, returning empty string")
            return ""
        
        print(f"Input date string type: {type(date_str)}")
        if isinstance(date_str, str):
            has_timezone = any(x in date_str for x in ['+', '-', 'Z'])
            print(f"Date string has timezone: {has_timezone}")
            # Удаляем возможный суффикс г.
            date_str = date_str.replace(" г.", "")
        
        date_obj = datetime.fromisoformat(date_str.replace('Z', '+00:00'))
        print(f"Parsed date object: {date_obj}")
        
        formatted = date_obj.strftime("%d.%m.%Y")
        print(f"Formatted date result: {formatted}")
        return formatted
    except Exception as e:
        print(f"Error formatting date {date_str}: {e}", file=sys.stderr)
        print(f"Date string type: {type(date_str)}", file=sys.stderr)
        return date_str

# Тестовые данные
test_dates = [
    "2025-01-29T10:08:39.725+03:00",  # Дата из первого запроса
    "2025-02-13T17:10:17.02749+03:00", # Дата из второго запроса
    "2025-02-13T17:10:17.02749+03:00 г.", # Дата с суффиксом
    "2025-02-13T17:10:17Z",           # Дата с Z
    "2025-02-13",                     # Только дата
    "",                               # Пустая строка
    None,                             # None
]

if __name__ == "__main__":
    print("Testing date formatting:")
    print("-" * 50)
    sys.stdout.flush()

    for date in test_dates:
        try:
            print(f"\nInput date: {date}")
            print(f"Type: {type(date)}")
            result = format_date(date)
            print(f"Result: {result}")
            print("-" * 50)
            sys.stdout.flush()
        except Exception as e:
            print(f"Error testing date {date}: {e}", file=sys.stderr)
            sys.stderr.flush() 