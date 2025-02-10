#!/usr/bin/env python3
import sys
import json
import os
from docxtpl import DocxTemplate
import multiprocessing
from concurrent.futures import ThreadPoolExecutor
import logging
from datetime import datetime

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def generate_applicant_info(data):
    """Генерирует информацию о заявителе на основе данных запроса."""
    applicant_type = data.get('applicantType')
    
    if applicant_type == 'ORGANIZATION':
        org_info = data.get('organizationInfo', {})
        if org_info:
            info = org_info.get('name', '')
            address = org_info.get('address')
            agent = org_info.get('agent')
            
            if address:
                info += f", {address}"
            if agent:
                info += f", Представитель: {agent}"
            return info
            
    elif applicant_type == 'INDIVIDUAL':
        ind_info = data.get('individualInfo', {})
        if ind_info:
            info = ind_info.get('name', '')
            esia = ind_info.get('esia')
            
            if esia:
                info += f" (ЕСИА {esia})"
            return info
            
    return ''

def process_template(template_path, data, output_path):
    try:
        doc = DocxTemplate(template_path)
        doc.render(data)
        doc.save(output_path)
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

    # Проверяем флаг параллельной обработки
    parallel_processing = os.environ.get('DOCX_PARALLEL_PROCESSING', '').lower() == 'true'
    
    if parallel_processing:
        # Используем ThreadPoolExecutor для параллельной обработки
        max_workers = min(multiprocessing.cpu_count(), 4)  # Ограничиваем максимальное количество потоков
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            future = executor.submit(process_template, template_path, data, output_path)
            if not future.result():
                sys.exit(1)
    else:
        # Последовательная обработка
        if not process_template(template_path, data, output_path):
            sys.exit(1)

if __name__ == '__main__':
    main() 