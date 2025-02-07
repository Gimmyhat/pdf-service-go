from docxtpl import DocxTemplate
import json
import sys
from datetime import datetime

def generate_docx(template_path, data_path, output_path):
    # Загружаем шаблон
    doc = DocxTemplate(template_path)
    
    # Читаем данные
    with open(data_path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    
    # Подготавливаем контекст
    context = {
        'id': data['id'],
        'geoInfoStorageOrganization': data['geoInfoStorageOrganization'],
        'phone': data['phone'],
        'email': data['email'],
        'purposeOfGeoInfoAccessDictionary': data['purposeOfGeoInfoAccessDictionary'],
        'creationDate': datetime.fromisoformat(data['creationDate'].replace('Z', '+00:00')).strftime('%d.%m.%Y'),
        'registryItems': data['registryItems']
    }
    
    if data['applicantType'] == 'INDIVIDUAL' and data['individualInfo']:
        context['applicant_name'] = data['individualInfo']['name']
    elif data['organizationInfo']:
        context['applicant_name'] = data['organizationInfo']['name']
    
    # Рендерим документ
    doc.render(context)
    
    # Сохраняем результат
    doc.save(output_path)

if __name__ == '__main__':
    if len(sys.argv) != 4:
        print('Usage: python generate_docx.py template_path data_path output_path')
        sys.exit(1)
    
    template_path = sys.argv[1]
    data_path = sys.argv[2]
    output_path = sys.argv[3]
    
    generate_docx(template_path, data_path, output_path) 