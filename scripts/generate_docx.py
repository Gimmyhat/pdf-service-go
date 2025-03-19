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
        logger.error("Error formatting date %s: %s", date_str, e)
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
        logger.error("Error generating applicant info: %s", e)
        return ''

def process_dates(data):
    """Обрабатывает все даты в данных."""
    if 'creationDate' in data:
        data['creationDate'] = format_date(data['creationDate'])
    
    if 'registryItems' in data and isinstance(data['registryItems'], list):
        for item in data['registryItems']:
            if 'informationDate' in item:
                item['informationDate'] = format_date(item['informationDate'])

def count_pdf_pages(docx_path):
    """Конвертирует DOCX в PDF и считает страницы."""
    try:
        # URL для Gotenberg из переменной окружения или по умолчанию
        gotenberg_url = os.getenv('GOTENBERG_URL', 'http://localhost:3000')
        endpoint = f"{gotenberg_url}/forms/libreoffice/convert"

        logger.info(f"Converting {docx_path} to PDF using Gotenberg at {endpoint}")
        logger.info(f"DOCX file size: {os.path.getsize(docx_path)} bytes")
        
        # Готовим файл для отправки
        with open(docx_path, 'rb') as f:
            files = {'file': (os.path.basename(docx_path), f, 'application/vnd.openxmlformats-officedocument.wordprocessingml.document')}
            
            # Отправляем запрос в Gotenberg
            logger.info("Sending request to Gotenberg...")
            response = requests.post(endpoint, files=files)
            logger.info(f"Gotenberg response status: {response.status_code}")
            
            if response.status_code == 200:
                # Сохраняем PDF во временный файл
                pdf_path = str(Path(docx_path).with_suffix('.pdf'))
                with open(pdf_path, 'wb') as f:
                    f.write(response.content)
                
                pdf_size = os.path.getsize(pdf_path)
                logger.info(f"Generated PDF size: {pdf_size} bytes")
                
                # Читаем количество страниц из PDF
                with open(pdf_path, 'rb') as f:
                    pdf = PyPDF2.PdfReader(f)
                    num_pages = len(pdf.pages)
                    logger.info(f"PDF page count: {num_pages}")
                    
                    # Проверяем размер первой и последней страницы
                    first_page = pdf.pages[0]
                    last_page = pdf.pages[-1]
                    logger.info(f"First page size: {first_page.mediabox}")
                    logger.info(f"Last page size: {last_page.mediabox}")
                
                # Удаляем временный PDF файл
                os.remove(pdf_path)
                return num_pages
            else:
                logger.error(f"Gotenberg conversion failed: {response.status_code}")
                if response.content:
                    logger.error(f"Gotenberg error: {response.content.decode('utf-8', errors='ignore')}")
                return None
    except Exception as e:
        logger.error(f"Error converting to PDF: {e}", exc_info=True)
        return None

def count_docx_pages(docx_path):
    """Подсчет страниц в DOCX файле напрямую."""
    try:
        doc = Document(docx_path)
        
        # Получаем размер страницы из секций документа
        section = doc.sections[0]
        page_height = section.page_height
        page_margin_top = section.top_margin
        page_margin_bottom = section.bottom_margin
        available_height = page_height - page_margin_top - page_margin_bottom
        
        total_height = 0
        for paragraph in doc.paragraphs:
            # Учитываем высоту каждого параграфа
            if paragraph.runs:  # Если параграф не пустой
                total_height += paragraph.runs[0].font.size or 12  # Размер шрифта в твипах
        
        # Учитываем таблицы
        for table in doc.tables:
            for row in table.rows:
                total_height += row.height or Inches(0.3)  # Примерная высота строки
        
        # Конвертируем твипы в дюймы и считаем страницы
        total_pages = int((total_height / 20) / (available_height / 914400) + 0.5)  # 914400 твипов = 1 дюйм
        
        logger.info(f"DOCX direct page count: {total_pages}")
        return max(1, total_pages)  # Минимум 1 страница
        
    except Exception as e:
        logger.error(f"Error counting DOCX pages directly: {e}", exc_info=True)
        return None

def process_template(data, output_path):
    """Обработка шаблона и сохранение результата."""
    global TEMPLATE
    
    if not TEMPLATE:
        logger.error("Template not initialized")
        raise ValueError("Template not initialized")
        
    try:
        # Обрабатываем даты и генерируем информацию о заявителе
        process_dates(data)
        data['applicant_info'] = generate_applicant_info(data)
        
        # Первый рендеринг с временным значением num_pages
        data['num_pages'] = 0
        logger.info("First render with num_pages = 0")
        TEMPLATE.render(data)
        TEMPLATE.save(output_path)
        
        # Пробуем получить количество страниц напрямую из DOCX
        pages = count_docx_pages(output_path)
        
        # Если не получилось, используем метод через PDF
        if pages is None:
            logger.info("Falling back to PDF page counting method")
            pages = count_pdf_pages(output_path)
            
        if pages is not None:
            # Вычитаем 1 страницу (сопроводительная записка)
            data['num_pages'] = pages - 1
            logger.info(f"Second render with page count (excluding cover): {pages-1}")
            TEMPLATE.render(data)
            TEMPLATE.save(output_path)
            logger.info("Document generated successfully")
            return True
        else:
            logger.error("Could not determine page count")
            return False
            
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
        # Инициализируем шаблон при старте
        init_app(template_path)

        # Читаем данные
        with open(data_path, 'r', encoding='utf-8') as f:
            data = json.load(f)
    except Exception as e:
        logger.error("Error during initialization: %s", e)
        sys.exit(1)

    if not process_template(data, output_path):
        sys.exit(1)

if __name__ == "__main__":
    main() 