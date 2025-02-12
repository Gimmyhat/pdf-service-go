package docx

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"strings"
)

// DocxGenerator реализует интерфейс Generator
type DocxGenerator struct {
	templatePath string
	cache        map[string]*Template
}

// NewGenerator создает новый генератор DOCX файлов
func NewGenerator(templatePath string) *DocxGenerator {
	return &DocxGenerator{
		templatePath: templatePath,
		cache:        make(map[string]*Template),
	}
}

// Generate генерирует DOCX файл на основе документа
func (g *DocxGenerator) Generate(doc *Document, outputPath string) error {
	// Сначала валидируем документ
	validator := NewDocumentValidator(doc)
	if err := validator.Validate(); err != nil {
		return fmt.Errorf("document validation failed: %w", err)
	}

	// Подготавливаем данные для шаблона
	data := g.prepareTemplateData(doc)

	// Открываем шаблон
	template, err := g.openTemplate()
	if err != nil {
		return fmt.Errorf("failed to open template: %w", err)
	}

	// Заменяем плейсхолдеры в шаблоне
	if err := g.fillTemplate(template, data, outputPath); err != nil {
		return fmt.Errorf("failed to fill template: %w", err)
	}

	return nil
}

// prepareTemplateData подготавливает данные для заполнения шаблона
func (g *DocxGenerator) prepareTemplateData(doc *Document) *TemplateData {
	data := &TemplateData{
		CreationDate:           doc.CreationDate.Format("02.01.2006"),
		PurposeOfGeoInfoAccess: doc.PurposeOfGeoInfoAccess,
		RegistryItems:          doc.RegistryItems,
	}

	// Формируем информацию о заявителе
	switch doc.ApplicantType {
	case "ORGANIZATION":
		org := doc.OrganizationInfo
		data.ApplicantInfo = fmt.Sprintf("%s, %s, %s", org.Name, org.Address, org.Agent)
		data.ApplicantName = org.Name
		data.ApplicantAgent = org.Agent
		data.IsOrganization = true
	case "INDIVIDUAL":
		ind := doc.IndividualInfo
		data.ApplicantInfo = ind.Name
		if ind.Esia != "" {
			data.ApplicantInfo += fmt.Sprintf(" (ЕСИА %s)", ind.Esia)
		}
		data.ApplicantName = fmt.Sprintf("физическое лицо %s", ind.Name)
		data.IsOrganization = false
	}

	return data
}

// openTemplate открывает DOCX шаблон
func (g *DocxGenerator) openTemplate() (*zip.ReadCloser, error) {
	return zip.OpenReader(g.templatePath)
}

// fillTemplate заполняет шаблон данными
func (g *DocxGenerator) fillTemplate(template *zip.ReadCloser, data *TemplateData, outputPath string) error {
	defer template.Close()

	// Создаем новый zip файл для результата
	output, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer output.Close()

	writer := zip.NewWriter(output)
	defer writer.Close()

	// Копируем все файлы из шаблона, заменяя плейсхолдеры в document.xml
	for _, file := range template.File {
		reader, err := file.Open()
		if err != nil {
			return err
		}
		defer reader.Close()

		// Создаем новый файл в выходном архиве
		writer, err := writer.Create(file.Name)
		if err != nil {
			return err
		}

		// Если это document.xml, заменяем плейсхолдеры
		if strings.HasSuffix(file.Name, "document.xml") {
			content, err := io.ReadAll(reader)
			if err != nil {
				return err
			}

			// Заменяем плейсхолдеры
			replaced := g.replacePlaceholders(string(content), data)
			if _, err := writer.Write([]byte(replaced)); err != nil {
				return err
			}
		} else {
			// Иначе просто копируем файл
			if _, err := io.Copy(writer, reader); err != nil {
				return err
			}
		}
	}

	return nil
}

// replacePlaceholders заменяет плейсхолдеры в XML документе
func (g *DocxGenerator) replacePlaceholders(content string, data *TemplateData) string {
	replacements := map[string]string{
		"{{creationDate}}":           data.CreationDate,
		"{{applicant_info}}":         data.ApplicantInfo,
		"{{applicant_name}}":         data.ApplicantName,
		"{{applicant_agent}}":        data.ApplicantAgent,
		"{{purposeOfGeoInfoAccess}}": data.PurposeOfGeoInfoAccess,
	}

	for key, value := range replacements {
		content = strings.ReplaceAll(content, key, value)
	}

	return content
}
