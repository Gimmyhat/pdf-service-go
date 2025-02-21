#!/usr/bin/env python3
import sys
import json
import os
from docxtpl import DocxTemplate
import logging
from datetime import datetime

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Глобальный шаблон
TEMPLATE = None

def init_template(template_path):
    """Инициализация глобального шаблона."""
    global TEMPLATE
    if TEMPLATE is None:
        logger.info("Initializing template from %s", template_path)
        TEMPLATE = DocxTemplate(template_path)
    return TEMPLATE

def format_date(date_str):
    """Форматирует дату из строки ISO в формат DD.MM.YYYY."""
    try:
        if not date_str:
            return ""
        if isinstance(date_str, str):
            # Удаляем миллисекунды, если они есть
            if '.' in date_str:
                date_str = date_str.split('.')[0]
            # Добавляем UTC если нет временной зоны
            if 'Z' not in date_str and '+' not in date_str and '-' not in date_str:
                date_str += 'Z'
            date_obj = datetime.fromisoformat(date_str.replace('Z', '+00:00'))
        else:
            date_obj = datetime.fromisoformat(str(date_str))
        return date_obj.strftime("%d.%m.%Y")
    except Exception as e:
        logger.error(f"Error formatting date {date_str}: {e}")
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
        logger.error(f"Error generating applicant info: {e}")
        return ''

def process_dates(data):
    """Обрабатывает все даты в данных."""
    if 'creationDate' in data:
        data['creationDate'] = format_date(data['creationDate'])
    
    if 'registryItems' in data and isinstance(data['registryItems'], list):
        for item in data['registryItems']:
            if 'informationDate' in item:
                item['informationDate'] = format_date(item['informationDate'])

def process_template(template_path, data, output_path):
    """Обрабатывает шаблон и генерирует документ."""
    try:
        logger.info("Starting template processing")
        
        # Получаем инициализированный шаблон
        doc = init_template(template_path)
        
        # Обрабатываем даты
        process_dates(data)
        
        # Генерируем информацию о заявителе
        data['applicant_info'] = generate_applicant_info(data)
        
        # Рендерим документ
        logger.info("Rendering document")
        doc.render(data)
        
        # Сохраняем результат
        logger.info("Saving document to %s", output_path)
        doc.save(output_path)
        
        return True
    except Exception as e:
        logger.error("Error processing template: %s", e)
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