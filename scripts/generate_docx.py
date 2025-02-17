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

def format_date(date_str, request_id='unknown'):
    """Форматирует дату из строки ISO в формат DD.MM.YYYY"""
    try:
        if not date_str:
            return ""
        
        if isinstance(date_str, str):
            date_str = date_str.replace(" г.", "")
        
        date_obj = parser.parse(date_str)
        return date_obj.strftime("%d.%m.%Y")
    except Exception as e:
        logger.error(f"Error formatting date {date_str}: {e}", extra={'request_id': request_id})
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

def process_template(template_path, data, output_path):
    try:
        request_id = data.get('requestId', 'unknown')
        logger.info(f"Starting template processing", extra={'request_id': request_id, 'operation': 'CREATE'})
        
        # Форматируем даты
        if 'creationDate' in data:
            data['creationDate_original'] = data['creationDate']
            data['creationDate'] = format_date(data['creationDate'], request_id)
        
        if 'registryItems' in data and isinstance(data['registryItems'], list):
            for item in data['registryItems']:
                if 'informationDate' in item and item['informationDate']:
                    item['informationDate'] = format_date(item['informationDate'], request_id)

        # Генерируем информацию о заявителе
        data['applicant_info'] = generate_applicant_info(data)

        # Загружаем и рендерим шаблон
        doc = DocxTemplate(template_path)
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

        return True
    except Exception as e:
        logger.error(f"Error processing template: {e}", extra={'request_id': request_id})
        return False

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