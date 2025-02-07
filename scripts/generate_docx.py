from docxtpl import DocxTemplate
import json
import sys
from datetime import datetime

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
        'registryItems': data['registryItems'],
        'applicant_info': generate_applicant_info(data)
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