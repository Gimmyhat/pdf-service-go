package docxgen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/unidoc/unioffice/color"
	"github.com/unidoc/unioffice/common/license"
	"github.com/unidoc/unioffice/document"
	"github.com/unidoc/unioffice/measurement"
	"github.com/unidoc/unioffice/schema/soo/wml"
)

func init() {
	// Получаем ключ из переменной окружения
	licenseKey := os.Getenv("UNIDOC_LICENSE_API_KEY")
	if licenseKey == "" {
		// Для локальной разработки можно использовать файл с ключом
		// Пробуем найти файл в разных местах
		possiblePaths := []string{
			".unidoc.key",          // текущая директория
			"../../../.unidoc.key", // корень проекта при запуске тестов
			filepath.Join(os.Getenv("GOPATH"), "src/pdf-service-go/.unidoc.key"), // GOPATH
		}

		for _, path := range possiblePaths {
			data, err := os.ReadFile(path)
			if err == nil {
				licenseKey = strings.TrimSpace(string(data))
				break
			}
		}
	}

	if licenseKey == "" {
		fmt.Println("Warning: UniDoc license key not found. Some features may be limited.")
		return
	}

	err := license.SetMeteredKey(licenseKey)
	if err != nil {
		fmt.Printf("Warning: Error loading UniDoc license: %v\n", err)
	}
}

// DocxData представляет структуру данных для шаблона
type DocxData struct {
	Operation                  string         `json:"operation"`
	ID                         string         `json:"id"`
	Email                      string         `json:"email"`
	Phone                      string         `json:"phone"`
	ApplicantType              string         `json:"applicantType"`
	OrganizationInfo           *OrgInfo       `json:"organizationInfo"`
	IndividualInfo             *IndInfo       `json:"individualInfo"`
	PurposeOfGeoInfoAccess     string         `json:"purposeOfGeoInfoAccess"`
	PurposeOfGeoInfoAccessDict *PurposeDict   `json:"purposeOfGeoInfoAccessDictionary"`
	RegistryItems              []RegistryItem `json:"registryItems"`
	CreatedBy                  *UserInfo      `json:"createdBy"`
	VerifiedBy                 *UserInfo      `json:"verifedBy"`
	CreationDate               string         `json:"creationDate"`
	GeoInfoStorageOrganization *OrgDict       `json:"geoInfoStorageOrganization"`
}

type OrgInfo struct {
	Name    string `json:"name"`
	Agent   string `json:"agent"`
	Address string `json:"address"`
}

type IndInfo struct {
	Name string `json:"name"`
	Esia string `json:"esia"`
}

type PurposeDict struct {
	Value string `json:"value"`
}

type RegistryItem struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	InformationDate string `json:"informationDate"`
	InvNumber       string `json:"invNumber"`
	Note            string `json:"note"`
}

type UserInfo struct {
	UserType string `json:"userType"`
	OID      string `json:"oid"`
	UserName string `json:"userName"`
	FullName string `json:"fullName"`
}

type OrgDict struct {
	Code  string `json:"code"`
	Value string `json:"value"`
}

// DocxGenerator представляет генератор DOCX файлов
type DocxGenerator struct {
	templatePath string
}

// NewDocxGenerator создает новый экземпляр генератора DOCX
func NewDocxGenerator(templatePath string) *DocxGenerator {
	return &DocxGenerator{
		templatePath: templatePath,
	}
}

// formatDate форматирует дату из ISO в формат DD.MM.YYYY
func formatDate(dateStr string) string {
	if dateStr == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("02.01.2006")
}

// generateApplicantInfo генерирует информацию о заявителе
func generateApplicantInfo(data *DocxData) string {
	if data.ApplicantType == "ORGANIZATION" && data.OrganizationInfo != nil {
		org := data.OrganizationInfo
		parts := []string{org.Name, org.Address, org.Agent}
		return strings.Join(parts, ", ")
	} else if data.IndividualInfo != nil {
		ind := data.IndividualInfo
		if ind.Esia != "" {
			return fmt.Sprintf("%s (ЕСИА %s)", ind.Name, ind.Esia)
		}
		return ind.Name
	}
	return ""
}

// replacePlaceholder заменяет плейсхолдер в параграфе
func replacePlaceholder(p document.Paragraph, placeholder, value string) {
	// Собираем все runs в параграфе
	runs := p.Runs()
	if len(runs) == 0 {
		return
	}

	// Собираем весь текст параграфа для проверки
	var paragraphText strings.Builder
	for _, r := range runs {
		paragraphText.WriteString(r.Text())
	}
	fullText := paragraphText.String()

	// Если плейсхолдер найден в тексте параграфа
	if strings.Contains(fullText, placeholder) {
		fmt.Printf("Found placeholder '%s' in text: '%s'\n", placeholder, fullText)

		// Если плейсхолдер полностью содержится в одном run
		for _, r := range runs {
			runText := r.Text()
			if strings.Contains(runText, placeholder) {
				newText := strings.ReplaceAll(runText, placeholder, value)
				fmt.Printf("Replacing in run: '%s' -> '%s'\n", placeholder, value)
				r.ClearContent()
				r.AddText(newText)
				return
			}
		}

		// Если плейсхолдер разбит между несколькими runs
		// Находим начало и конец плейсхолдера
		var startRun, endRun int
		var startFound bool
		placeholderParts := []byte(placeholder)
		currentPart := 0

		for i, r := range runs {
			runText := r.Text()
			for j := 0; j < len(runText); j++ {
				if currentPart < len(placeholderParts) && runText[j] == placeholderParts[currentPart] {
					if !startFound {
						startRun = i
						startFound = true
					}
					currentPart++
					if currentPart == len(placeholderParts) {
						endRun = i
						// Заменяем плейсхолдер
						if startRun == endRun {
							runs[startRun].ClearContent()
							runs[startRun].AddText(value)
						} else {
							// Очищаем все runs между start и end
							for k := startRun; k <= endRun; k++ {
								runs[k].ClearContent()
							}
							// Добавляем значение в первый run
							runs[startRun].AddText(value)
						}
						return
					}
				} else {
					currentPart = 0
					startFound = false
				}
			}
		}
	}
}

// Generate генерирует DOCX файл из шаблона и данных
func (g *DocxGenerator) Generate(jsonData []byte, outputPath string) error {
	// Парсим JSON данные
	var data DocxData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("error parsing JSON: %w", err)
	}

	// Открываем шаблон
	doc, err := document.Open(g.templatePath)
	if err != nil {
		return fmt.Errorf("error opening template: %w", err)
	}

	// Форматируем дату создания
	formattedDate := formatDate(data.CreationDate)

	// Генерируем информацию о заявителе
	applicantInfo := generateApplicantInfo(&data)
	isOrganization := data.ApplicantType == "ORGANIZATION"
	var applicantAgent string
	if isOrganization && data.OrganizationInfo != nil {
		applicantAgent = data.OrganizationInfo.Agent
	}

	// Подготавливаем все значения для замены
	replacements := []struct {
		placeholder string
		value       string
	}{
		{"{{ id }}", data.ID},
		{"{{id}}", data.ID},
		{"{{ creationDate }}", formattedDate},
		{"{{creationDate}}", formattedDate},
		{"{{ applicant_info }}", applicantInfo},
		{"{{applicant_info}}", applicantInfo},
		{"{{ applicant_name }}", applicantInfo}, // для совместимости
		{"{{applicant_name}}", applicantInfo},   // для совместимости
		{"{{ email }}", data.Email},
		{"{{email}}", data.Email},
		{"{{ phone }}", data.Phone},
		{"{{phone}}", data.Phone},
		{"{{ geoInfoStorageOrganization.value }}", data.GeoInfoStorageOrganization.Value},
		{"{{geoInfoStorageOrganization.value}}", data.GeoInfoStorageOrganization.Value},
		{"{{ purposeOfGeoInfoAccessDictionary.value }}", data.PurposeOfGeoInfoAccessDict.Value},
		{"{{purposeOfGeoInfoAccessDictionary.value}}", data.PurposeOfGeoInfoAccessDict.Value},
	}

	// Заменяем плейсхолдеры в документе
	for _, p := range doc.Paragraphs() {
		// Получаем полный текст параграфа
		var paragraphText strings.Builder
		for _, r := range p.Runs() {
			paragraphText.WriteString(r.Text())
		}
		fullText := paragraphText.String()

		// Обрабатываем условную конструкцию для организации
		if strings.Contains(fullText, "{% if is_organization %}") {
			runs := p.Runs()
			if isOrganization {
				// Находим и заменяем плейсхолдер applicant_agent
				var newText string
				if strings.Contains(fullText, "{{ applicant_agent }}") {
					newText = fmt.Sprintf("Представитель юридического лица: %s", applicantAgent)
				} else if strings.Contains(fullText, "{{applicant_agent}}") {
					newText = fmt.Sprintf("Представитель юридического лица: %s", applicantAgent)
				}

				// Очищаем все runs в параграфе
				for _, r := range runs {
					r.ClearContent()
				}
				// Добавляем новый текст в первый run
				if len(runs) > 0 {
					runs[0].AddText(newText)
				}
			} else {
				// Если не организация, очищаем содержимое всех runs в параграфе
				for _, r := range runs {
					r.ClearContent()
				}
			}
			continue
		}

		// Заменяем остальные плейсхолдеры
		for _, r := range replacements {
			replacePlaceholder(p, r.placeholder, r.value)
		}
	}

	// Создаем новый футер
	fmt.Println("Creating new footer...")
	footer := doc.AddFooter()

	// Добавляем параграф с горизонтальной линией
	linePara := footer.AddParagraph()
	linePara.Properties().SetAlignment(wml.ST_JcLeft)
	lineRun := linePara.AddRun()
	line := strings.Repeat("_", 150)
	lineRun.AddText(line)

	// Добавляем параграф для основного текста
	mainPara := footer.AddParagraph()
	mainPara.Properties().SetAlignment(wml.ST_JcLeft)
	mainRun := mainPara.AddRun()
	footerText := "ЕФГИ Заявка на предоставление в пользование геологической информации о недрах № " + data.ID +
		". Документ сформирован с использованием ФГИС «ЕФГИ» " + formattedDate
	mainRun.AddText(footerText)

	// Добавляем параграф для номеров страниц
	pagePara := footer.AddParagraph()
	pagePara.Properties().SetAlignment(wml.ST_JcRight)
	pageRun := pagePara.AddRun()
	pageRun.AddText("Стр. ")
	pageRun.AddField(document.FieldCurrentPage)
	pageRun.AddText(" из ")
	pageRun.AddField(document.FieldNumberOfPages)

	// Устанавливаем футер для всего документа
	doc.BodySection().SetFooter(footer, wml.ST_HdrFtrDefault)

	// Добавляем таблицу с реестровыми записями
	if len(data.RegistryItems) > 0 {
		table := doc.AddTable()
		borders := table.Properties().Borders()
		borders.SetAll(wml.ST_BorderSingle, color.Black, 1*measurement.Point)

		// Устанавливаем ширину таблицы
		width := 100.0 // процентов
		table.Properties().SetWidthPercent(width)

		// Добавляем заголовки таблицы
		headerRow := table.AddRow()
		headers := []string{"№ п/п", "Инвентарный номер объекта учета в каталоге", "Название документа, авторы", "Год выпуска документа", "Реестровый номер", "Примечание"}
		widths := []float64{5, 15, 40, 10, 15, 15} // проценты

		for i, header := range headers {
			cell := headerRow.AddCell()
			cell.Properties().SetWidthPercent(widths[i])

			para := cell.AddParagraph()
			para.Properties().SetAlignment(wml.ST_JcCenter)

			run := para.AddRun()
			run.Properties().SetBold(true)
			run.AddText(header)

			// Добавляем серый фон для заголовков
			cell.Properties().SetShading(wml.ST_ShdSolid, color.Gray, color.Auto)
		}

		// Добавляем данные
		for i, item := range data.RegistryItems {
			if item.Name == "" && item.Note == "Документ недоступен" {
				continue // Пропускаем недоступные документы
			}

			row := table.AddRow()

			// Номер
			numCell := row.AddCell()
			numCell.Properties().SetWidthPercent(widths[0])
			numPara := numCell.AddParagraph()
			numPara.Properties().SetAlignment(wml.ST_JcCenter)
			numRun := numPara.AddRun()
			numRun.AddText(fmt.Sprintf("%d", i+1))

			// Инв. номер
			invCell := row.AddCell()
			invCell.Properties().SetWidthPercent(widths[1])
			invPara := invCell.AddParagraph()
			invPara.Properties().SetAlignment(wml.ST_JcCenter)
			invRun := invPara.AddRun()
			invRun.AddText(item.InvNumber)

			// Наименование
			nameCell := row.AddCell()
			nameCell.Properties().SetWidthPercent(widths[2])
			namePara := nameCell.AddParagraph()
			namePara.Properties().SetAlignment(wml.ST_JcLeft)
			nameRun := namePara.AddRun()
			nameRun.AddText(item.Name)

			// Год выпуска
			yearCell := row.AddCell()
			yearCell.Properties().SetWidthPercent(widths[3])
			yearPara := yearCell.AddParagraph()
			yearPara.Properties().SetAlignment(wml.ST_JcCenter)
			yearRun := yearPara.AddRun()
			if item.InformationDate != "" {
				yearRun.AddText(formatDate(item.InformationDate))
			}

			// Реестровый номер (используем ID)
			regCell := row.AddCell()
			regCell.Properties().SetWidthPercent(widths[4])
			regPara := regCell.AddParagraph()
			regPara.Properties().SetAlignment(wml.ST_JcCenter)
			regRun := regPara.AddRun()
			regRun.AddText(fmt.Sprintf("%d", item.ID))

			// Примечание
			noteCell := row.AddCell()
			noteCell.Properties().SetWidthPercent(widths[5])
			notePara := noteCell.AddParagraph()
			notePara.Properties().SetAlignment(wml.ST_JcLeft)
			noteRun := notePara.AddRun()
			noteRun.AddText(item.Note)
		}
	}

	// Сохраняем документ
	return doc.SaveToFile(outputPath)
}
