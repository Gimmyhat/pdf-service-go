#!/usr/bin/env python3
import sys
import json
import os
from docxtpl import DocxTemplate
import multiprocessing
from concurrent.futures import ThreadPoolExecutor
import logging
from datetime import datetime
import gc

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Глобальный кэш для шаблонов
template_cache = {}

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

def get_template(template_path):
    """Получает шаблон из кэша или загружает новый."""
    if template_path not in template_cache:
        template_cache[template_path] = DocxTemplate(template_path)
    return template_cache[template_path]

def process_template(template_path, data, output_path):
    try:
        # Оптимальное количество потоков
        cpu_count = min(multiprocessing.cpu_count(), 4)
        
        with ThreadPoolExecutor(max_workers=cpu_count) as executor:
            futures = []
            
            # Форматируем даты асинхронно
            if 'creationDate' in data:
                futures.append(('creationDate', executor.submit(format_date, data['creationDate'])))
            
            if 'registryItems' in data and isinstance(data['registryItems'], list):
                for i, item in enumerate(data['registryItems']):
                    if 'informationDate' in item and item['informationDate']:
                        futures.append((f'registry_{i}', executor.submit(format_date, item['informationDate'])))

            # Генерируем информацию о заявителе асинхронно
            futures.append(('applicant_info', executor.submit(generate_applicant_info, data)))

            # Собираем результаты
            for key, future in futures:
                try:
                    result = future.result()
                    if key == 'creationDate':
                        data['creationDate'] = result
                    elif key == 'applicant_info':
                        data['applicant_info'] = result
                    elif key.startswith('registry_'):
                        idx = int(key.split('_')[1])
                        data['registryItems'][idx]['informationDate'] = result
                except Exception as e:
                    logger.error(f"Error processing {key}: {e}")

        # Используем кэшированный шаблон
        doc = get_template(template_path)
        doc.render(data)
        doc.save(output_path)

        # Очищаем память
        gc.collect()
        
        return True
    except Exception as e:
        logger.error(f"Error processing template: {e}")
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