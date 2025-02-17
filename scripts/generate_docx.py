#!/usr/bin/env python3
import sys
import json
import os
from docxtpl import DocxTemplate, InlineImage
from docx.shared import Mm
import multiprocessing
from concurrent.futures import ThreadPoolExecutor
import logging
from datetime import datetime
import gc
import json
from logging import StreamHandler
import time
from dateutil import parser
from docx.oxml import OxmlElement
from docx.oxml.ns import qn

class JsonFormatter(logging.Formatter):
    def format(self, record):
        # Преобразуем уровни логирования в формат zap
        level_map = {
            'DEBUG': 'debug',
            'INFO': 'info',
            'WARNING': 'warn',
            'ERROR': 'error',
            'CRITICAL': 'fatal'
        }
        timestamp = time.strftime('%Y-%m-%dT%H:%M:%S.000Z', time.gmtime())
        json_record = {
            "level": level_map.get(record.levelname, record.levelname.lower()),
            "timestamp": timestamp,
            "caller": "python/generate_docx.py",
            "msg": record.getMessage(),
            "logger": "docx_generator"
        }
        if hasattr(record, 'request_id'):
            json_record["request_id"] = record.request_id
        if hasattr(record, 'operation'):
            json_record["operation"] = record.operation
        
        formatted = json.dumps(json_record)
        # Добавляем перевод строки и сбрасываем буфер
        return formatted + "\n"

class FlushingStreamHandler(StreamHandler):
    def emit(self, record):
        super().emit(record)
        self.flush()

def setup_logging():
    # Настраиваем логирование в формате JSON
    logger = logging.getLogger("docx_generator")
    logger.setLevel(logging.INFO)
    
    # Очищаем существующие обработчики
    for handler in logger.handlers[:]:
        logger.removeHandler(handler)
    
    try:
        # Пробуем открыть stdout для Docker
        sys.stdout = open('/proc/1/fd/1', 'w')
    except:
        # Если не получилось, используем стандартный stdout
        pass
    
    # Добавляем обработчик для stdout с JSON форматированием
    handler = FlushingStreamHandler(sys.stdout)
    handler.setFormatter(JsonFormatter())
    logger.addHandler(handler)
    
    # Отключаем propagation, чтобы избежать дублирования
    logger.propagate = False
    
    # Сбрасываем буфер
    sys.stdout.flush()
    
    return logger

# Инициализируем логгер
logger = setup_logging()

# Записываем разделитель для нового запуска
logger.info("New document generation started", extra={'operation': 'START'})

# Глобальный кэш для шаблонов
template_cache = {}

def format_date(date_str, request_id='unknown'):
    """Форматирует дату из строки ISO в формат DD.MM.YYYY"""
    try:
        logger.info(f"Starting date formatting for: {date_str}", extra={'request_id': request_id})
        if not date_str:
            logger.info("Empty date string, returning empty string", extra={'request_id': request_id})
            return ""
        
        logger.info(f"Input date string type: {type(date_str)}", extra={'request_id': request_id})
        if isinstance(date_str, str):
            # Удаляем возможный суффикс г.
            date_str = date_str.replace(" г.", "")
            
            has_timezone = any(x in date_str for x in ['+', '-', 'Z'])
            logger.info(f"Has timezone: {has_timezone}", extra={'request_id': request_id})
            logger.info(f"Normalized date string: {date_str}", extra={'request_id': request_id})
        
        date_obj = parser.parse(date_str)
        logger.info(f"Parsed date: {date_obj}", extra={'request_id': request_id})
        
        formatted = date_obj.strftime("%d.%m.%Y")
        logger.info(f"Formatted result: {formatted}", extra={'request_id': request_id})
        return formatted
    except Exception as e:
        logger.error(f"Error formatting date {date_str}: {e}", extra={'request_id': request_id})
        logger.error(f"Date string type: {type(date_str)}", extra={'request_id': request_id})
        return date_str

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
        logger.error(f"Error generating applicant info: {e}")
        return ''

def get_template(template_path):
    """Получает шаблон из кэша или загружает новый."""
    if template_path not in template_cache:
        template_cache[template_path] = DocxTemplate(template_path)
        # Логируем информацию о шаблоне
        template = template_cache[template_path]
        try:
            # Попытка получить все переменные из шаблона
            variables = template.get_undeclared_template_variables()
            logger.info(f"Template variables found: {variables}")
        except Exception as e:
            logger.error(f"Error analyzing template: {e}")
    return template_cache[template_path]

def add_page_number(paragraph):
    """Добавляет поля номера страницы в параграф"""
    # Текущая страница
    run = paragraph.add_run()
    fldChar1 = OxmlElement('w:fldChar')
    fldChar1.set(qn('w:fldCharType'), 'begin')
    run._r.append(fldChar1)
    
    instrText = OxmlElement('w:instrText')
    instrText.text = "PAGE"
    run._r.append(instrText)
    
    fldChar2 = OxmlElement('w:fldChar')
    fldChar2.set(qn('w:fldCharType'), 'end')
    run._r.append(fldChar2)
    
    # Добавляем текст " из "
    run = paragraph.add_run(" из ")
    
    # Общее количество страниц
    run = paragraph.add_run()
    fldChar3 = OxmlElement('w:fldChar')
    fldChar3.set(qn('w:fldCharType'), 'begin')
    run._r.append(fldChar3)
    
    instrText2 = OxmlElement('w:instrText')
    instrText2.text = "NUMPAGES"
    run._r.append(instrText2)
    
    fldChar4 = OxmlElement('w:fldChar')
    fldChar4.set(qn('w:fldCharType'), 'end')
    run._r.append(fldChar4)

def process_template(template_path, data, output_path):
    try:
        # Получаем request_id из данных
        request_id = data.get('requestId', 'unknown')
        logger.info(f"Starting template processing", extra={'request_id': request_id, 'operation': 'CREATE'})
        
        # Оптимальное количество потоков
        cpu_count = min(multiprocessing.cpu_count(), 4)
        
        # Сохраняем исходную дату для сравнения
        original_date = data.get('creationDate')
        logger.info(f"Original date from request: {original_date}", extra={'request_id': request_id})
        logger.info(f"Initial data structure: {json.dumps(data, indent=2, ensure_ascii=False)}", extra={'request_id': request_id})
        logger.info(f"Template path: {template_path}", extra={'request_id': request_id})
        
        with ThreadPoolExecutor(max_workers=cpu_count) as executor:
            futures = []
            
            # Форматируем даты асинхронно
            if 'creationDate' in data:
                logger.info(f"Found creationDate in data: {data['creationDate']}", extra={'request_id': request_id})
                logger.info(f"Date value type before formatting: {type(data['creationDate'])}", extra={'request_id': request_id})
                futures.append(('creationDate', executor.submit(format_date, data['creationDate'], request_id)))
            else:
                logger.warning("No creationDate found in data", extra={'request_id': request_id})
            
            if 'registryItems' in data and isinstance(data['registryItems'], list):
                logger.info(f"Processing {len(data['registryItems'])} registry items", extra={'request_id': request_id})
                for i, item in enumerate(data['registryItems']):
                    if 'informationDate' in item and item['informationDate']:
                        futures.append((f'registry_{i}', executor.submit(format_date, item['informationDate'], request_id)))

            # Генерируем информацию о заявителе асинхронно
            futures.append(('applicant_info', executor.submit(generate_applicant_info, data)))

            # Собираем результаты
            for key, future in futures:
                try:
                    result = future.result()
                    if key == 'creationDate':
                        logger.info(f"Processing creationDate result: {result}", extra={'request_id': request_id})
                        logger.info(f"Result type: {type(result)}", extra={'request_id': request_id})
                        data['creationDate_original'] = original_date
                        data['creationDate'] = result
                        logger.info(f"Updated data with formatted date: {data['creationDate']}", extra={'request_id': request_id})
                        logger.info(f"Date value type after formatting: {type(data['creationDate'])}", extra={'request_id': request_id})
                    elif key == 'applicant_info':
                        data['applicant_info'] = result
                    elif key.startswith('registry_'):
                        idx = int(key.split('_')[1])
                        data['registryItems'][idx]['informationDate'] = result
                except Exception as e:
                    logger.error(f"Error processing {key}: {e}", extra={'request_id': request_id})
                    logger.error(f"Exception details: {str(e)}", extra={'request_id': request_id})

        logger.info("Template variables before rendering:", extra={'request_id': request_id})
        doc = get_template(template_path)
        
        try:
            variables = doc.get_undeclared_template_variables()
            logger.info(f"Template expects variables: {variables}", extra={'request_id': request_id})
            # Проверяем, какие переменные шаблон ожидает для даты
            date_vars = [var for var in variables if 'date' in var.lower()]
            logger.info(f"Date-related template variables: {date_vars}", extra={'request_id': request_id})
        except Exception as e:
            logger.error(f"Error getting template variables: {e}", extra={'request_id': request_id})

        logger.info(f"Final data before rendering: {json.dumps(data, indent=2, ensure_ascii=False)}", extra={'request_id': request_id})
        logger.info(f"Final creationDate value: {data.get('creationDate')}", extra={'request_id': request_id})
        logger.info(f"Final creationDate type: {type(data.get('creationDate'))}", extra={'request_id': request_id})
        
        # Рендерим шаблон
        doc.render(data)
        
        # Добавляем нумерацию страниц в футер
        section = doc.docx.sections[0]
        footer = section.footer
        paragraph = footer.paragraphs[0] if footer.paragraphs else footer.add_paragraph()
        paragraph.alignment = 2  # По правому краю
        add_page_number(paragraph)
        
        # Сохраняем документ
        doc.save(output_path)
        logger.info(f"Document saved to: {output_path}", extra={'request_id': request_id})

        # Очищаем память
        gc.collect()
        
        return True
    except Exception as e:
        logger.error(f"Error processing template: {e}", extra={'request_id': request_id})
        logger.error(f"Exception details: {str(e)}", extra={'request_id': request_id})
        return False

def main():
    if len(sys.argv) != 4:
        print("Usage: generate_docx.py <template_path> <data_path> <output_path>")
        sys.exit(1)

    template_path = sys.argv[1]
    data_path = sys.argv[2]
    output_path = sys.argv[3]

    try:
        with open(data_path, 'r', encoding='utf-8') as f:
            data = json.load(f)
    except Exception as e:
        logger.error(f"Error reading data file: {e}")
        sys.exit(1)

    if not process_template(template_path, data, output_path):
        sys.exit(1)

if __name__ == "__main__":
    main() 