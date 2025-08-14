#!/usr/bin/env python3
import sys
import json
import os
from docxtpl import DocxTemplate
import logging
from datetime import datetime
from pathlib import Path
import requests
import PyPDF2
from docx import Document
from docx.shared import Inches
import re # Добавляем импорт модуля регулярных выражений
import time
import traceback

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Глобальный шаблон
TEMPLATE = None

def init_app(template_path):
    """Инициализация приложения."""
    global TEMPLATE
    logger.info("Initializing template from %s", template_path)
    try:
        TEMPLATE = DocxTemplate(template_path)
        logger.info("Template initialized successfully")
    except Exception as e:
        logger.error("Failed to initialize template: %s", e)
        raise

def format_date(date_str):
    """Форматирует дату из строки ISO в формат DD.MM.YYYY.
       Если строка содержит только год (YYYY), возвращает год как есть.
    """
    if not date_str:
        return ""

    if isinstance(date_str, str):
        # Убираем пробелы в начале и конце
        date_str = date_str.strip()

        # Проверяем, является ли строка просто годом (YYYY)
        if re.fullmatch(r"\d{4}", date_str):
            logger.info(f"Detected year-only format: '{date_str}'. Returning as is.")
            return date_str
        
        # Если это не год, пытаемся обработать как ISO дату
        try:
            # Удаляем миллисекунды, если они есть
            if '.' in date_str:
                date_str = date_str.split('.')[0]
            
            # Добавляем Z (UTC), только если строка выглядит как дата/время без таймзоны
            # и не является просто годом (проверка выше уже была)
            if 'T' in date_str and 'Z' not in date_str and '+' not in date_str and '-' not in date_str.split('T')[-1]:
                 # Добавляем Z только если есть 'T' и нет явной таймзоны
                 logger.debug(f"Adding Z to date string: {date_str}")
                 date_str += 'Z'
            
            # Заменяем Z на +00:00 для fromisoformat
            if date_str.endswith('Z'):
                 parse_str = date_str[:-1] + '+00:00'
            else:
                 parse_str = date_str

            logger.debug(f"Attempting to parse as ISO: {parse_str}")
            date_obj = datetime.fromisoformat(parse_str)
            formatted_date = date_obj.strftime("%d.%m.%Y")
            logger.info(f"Formatted date '{date_str}' to '{formatted_date}'")
            return formatted_date
        except ValueError as e:
            # Если парсинг не удался, возвращаем исходную строку (без добавленного Z)
            logger.error(f"Could not parse date string '{date_str}' as ISO: {e}. Returning original string.")
            # Важно: возвращаем оригинальную строку до добавления Z, если она была добавлена
            # Найдем оригинальную строку до модификаций внутри try
            # Проще всего просто вернуть исходный аргумент функции в случае ошибки
            # Но так как мы модифицируем date_str, нужна исходная копия
            # Переделаем: будем работать с копией для парсинга
            logger.warning(f"Returning original unparsed string: {date_str}") # Логируем что возвращаем оригинал
            # Вернемся к исходной идее - если не парсится, возвращаем как есть
            # Найдем способ вернуть строку до добавления Z
            # Просто вернем исходный date_str на момент ошибки
            return date_str # Возвращаем строку в том виде, в каком она вызвала ошибку
        except Exception as e:
            logger.error(f"Unexpected error formatting date '{date_str}': {e}", exc_info=True)
            return date_str # Возвращаем строку в том виде, в каком она вызвала ошибку
    else:
        # Если это не строка, пытаемся преобразовать и обработать
        try:
            date_obj = datetime.fromisoformat(str(date_str))
            return date_obj.strftime("%d.%m.%Y")
        except Exception as e:
            logger.error(f"Error formatting non-string date {date_str}: {e}")
            return str(date_str)

def generate_applicant_info(data):
    """Генерирует информацию о заявителе на основе данных запроса."""
    try:
        applicant_type = data.get('applicantType')
        
        if applicant_type == 'ORGANIZATION':
            org_info = data.get('organizationInfo', {})
            if org_info:
                name = org_info.get('name', '')
                address = org_info.get('address', '')
                agent = org_info.get('agent', '')
                
                data['applicant_name'] = name
                data['applicant_agent'] = agent
                data['is_organization'] = True
                
                return f"{name}, {address}, {agent}".rstrip(', ')
                
        else:  # INDIVIDUAL
            ind_info = data.get('individualInfo', {})
            if ind_info:
                name = ind_info.get('name', '')
                esia = ind_info.get('esia', '')
                esia_suffix = f" (ЕСИА {esia})" if esia else ''
                
                data['applicant_name'] = f"физическое лицо {name}"
                data['applicant_agent'] = ''
                data['is_organization'] = False
                
                return f"{name}{esia_suffix}"
                
        return ''
    except Exception as e:
        logger.error(f"Error generating applicant info: {e}", exc_info=True)
        return ''

def process_dates(data):
    """Обрабатывает все даты в данных."""
    if 'creationDate' in data:
        data['creationDate'] = format_date(data['creationDate'])
    
    if 'registryItems' in data and isinstance(data['registryItems'], list):
        for item in data['registryItems']:
            if 'informationDate' in item:
                item['informationDate'] = format_date(item['informationDate'])

def process_template(data, output_path, timings=None):
    """Обработка шаблона и сохранение результата."""
    global TEMPLATE
    
    if not TEMPLATE:
        logger.error("Template not initialized")
        raise ValueError("Template not initialized")
        
    try:
        # Проверяем, является ли это черновиком для подсчета страниц
        is_draft = data.get('isDraft', False)
        if is_draft:
            logger.info("Processing DRAFT document for page counting")
        else:
            logger.info("Processing FINAL document with page count")
        
        # Обрабатываем даты и генерируем информацию о заявителе
        t_pd_start = time.time()
        process_dates(data)
        if timings is not None:
            timings.append({"stage": "process_dates", "ms": round((time.time() - t_pd_start) * 1000, 2)})
        t_ai_start = time.time()
        data['applicant_info'] = generate_applicant_info(data)
        if timings is not None:
            timings.append({"stage": "generate_applicant_info", "ms": round((time.time() - t_ai_start) * 1000, 2)})
        
        # Создаем укороченный ID без текста ЕФГИ
        if 'id' in data:
            # Используем ID
            full_id = data.get('id', '')
            short_id = full_id
            
            # Если ID начинается с "ЕФГИ-", берем часть строки с 5-го символа
            if full_id and full_id.startswith("ЕФГИ-"):
                short_id = full_id[5:]  # Получаем строку начиная с 5-го символа (после "ЕФГИ-")
                logger.info(f"Extracted short ID: '{short_id}' from full ID: '{full_id}'")
            else:
                logger.info(f"ID '{full_id}' does not start with 'ЕФГИ-', keeping original")
            
            data['short_id'] = short_id
        
        # Устанавливаем количество страниц для отображения
        # Для черновика устанавливаем заглушку, для финала - используем счетчик
        if is_draft:
            # В черновике просто используем заглушку, т.к. этот документ только для подсчета
            data['display_pages'] = "[Подсчет страниц...]"
            logger.info(f"Using placeholder for page count in draft document")
        else:
            # В финальном документе используем реальное количество страниц
            page_count = data.get('pages', 0)
            
            # Вычитаем еще одну страницу для отображения (сопроводительная записка)
            display_count = max(1, page_count - 1)
            
            # Логика отображения: 
            # Если 1 страница - "на 1 листе"
            # Для остальных случаев - "на X листах"
            if display_count == 1:
                data['display_pages'] = "на 1 листе"
            else:
                data['display_pages'] = f"на {display_count} листах"
            
            logger.info(f"Setting display pages to: '{data['display_pages']}', actual page count: {page_count}, display count: {display_count}")
        
        # Логируем все переменные для отладки
        logger.info("Template variables:")
        logger.info(f"id: {data.get('id', 'NOT FOUND')}")
        logger.info(f"short_id: {data.get('short_id', 'NOT FOUND')}")
        logger.info(f"creationDate: {data.get('creationDate', 'NOT FOUND')}")
        logger.info(f"isDraft: {is_draft}")
        logger.info(f"pages: {data.get('pages', 0)}")
        logger.info(f"display_pages: {data.get('display_pages', 'NOT SET')}")
        logger.info(f"status: {data.get('status', 'NOT SET')}")
        
        try:
            # Рендерим документ
            t_render_start = time.time()
            TEMPLATE.render(data)
            render_ms = round((time.time() - t_render_start) * 1000, 2)
            if timings is not None:
                timings.append({"stage": "render", "ms": render_ms})

            t_save_start = time.time()
            TEMPLATE.save(output_path)
            save_ms = round((time.time() - t_save_start) * 1000, 2)
            if timings is not None:
                timings.append({"stage": "save", "ms": save_ms})
            logger.info("Document generated successfully")
            return True
        except Exception as e:
            logger.error(f"Error rendering template: {e}", exc_info=True)
            return False
    except Exception as e:
        logger.error(f"Error processing template: {e}", exc_info=True)
        return False

def main():
    if len(sys.argv) != 4:
        print("Usage: generate_docx.py <template_path> <data_path> <output_path>")
        sys.exit(1)

    template_path = sys.argv[1]
    data_path = sys.argv[2]
    output_path = sys.argv[3]

    try:
        # Инициализируем шаблон при старте
        t0 = time.time()
        init_app(template_path)
        init_ms = round((time.time() - t0) * 1000, 2)

        # Читаем данные
        t_read_start = time.time()
        with open(data_path, 'r', encoding='utf-8') as f:
            data = json.load(f)
        read_ms = round((time.time() - t_read_start) * 1000, 2)
    except Exception as e:
        logger.error("Error during initialization: %s", e)
        sys.exit(1)

    timings = []
    timings.append({"stage": "init_app", "ms": init_ms})
    timings.append({"stage": "read_input", "ms": read_ms})

    t_proc_start = time.time()
    ok = process_template(data, output_path, timings)
    total_ms = round((time.time() - t0) * 1000, 2)

    # Пишем тайминги в stdout и в файл рядом с выходным DOCX
    try:
        summary = {
            "total_ms": total_ms,
            "stages": timings
        }
        logger.info("DOCX generation timings: %s", json.dumps(summary, ensure_ascii=False))
        timings_path = output_path + ".timings.json"
        with open(timings_path, 'w', encoding='utf-8') as tf:
            json.dump(summary, tf, ensure_ascii=False, indent=2)
        print(f"TIMINGS_FILE={timings_path}")
    except Exception:
        traceback.print_exc()

    if not ok:
        sys.exit(1)

if __name__ == "__main__":
    main() 