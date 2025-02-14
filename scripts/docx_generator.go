package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

// Структуры для JSON остаются теми же, что в вашем коде
type Request struct {
	Operation                  string        `json:"operation"`
	ID                         string        `json:"id"`
	Email                      string        `json:"email"`
	Phone                      string        `json:"phone"`
	ApplicantType              string        `json:"applicantType"`
	OrganizationInfo           *Organization `json:"organizationInfo"`
	IndividualInfo             *Individual   `json:"individualInfo"`
	PurposeOfGeoInfoAccess     string        `json:"purposeOfGeoInfoAccess"`
	PurposeOfGeoInfoAccessDict Dict          `json:"purposeOfGeoInfoAccessDictionary"`
	RegistryItems              []Item        `json:"registryItems"`
	CreationDate               string        `json:"creationDate"`
	GeoInfoStorageOrganization Dict          `json:"geoInfoStorageOrganization"`
}

type Organization struct {
	Name    string `json:"name"`
	Agent   string `json:"agent"`
	Address string `json:"address"`
}

type Individual struct {
	ESIA string `json:"esia"`
	Name string `json:"name"`
}

type Dict struct {
	Value string `json:"value"`
}

type Item struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	InformationDate string `json:"informationDate"`
	InvNumber       string `json:"invNumber"`
	Note            string `json:"note"`
}

func formatDate(dateStr string) string {
	if dateStr == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, strings.Replace(dateStr, "Z", "+00:00", 1))
	if err != nil {
		log.Printf("Ошибка форматирования даты %s: %v", dateStr, err)
		return dateStr
	}
	return t.Format("02.01.2006")
}

func generateApplicantInfo(data Request) (string, map[string]interface{}) {
	result := make(map[string]interface{})

	if data.ApplicantType == "ORGANIZATION" {
		if data.OrganizationInfo != nil {
			name := data.OrganizationInfo.Name
			address := data.OrganizationInfo.Address
			agent := data.OrganizationInfo.Agent

			result["applicant_name"] = name
			result["applicant_agent"] = agent
			result["is_organization"] = true

			parts := []string{name, address, agent}
			var nonEmpty []string
			for _, part := range parts {
				if part != "" {
					nonEmpty = append(nonEmpty, part)
				}
			}
			return strings.Join(nonEmpty, ", "), result
		}
	} else { // INDIVIDUAL
		if data.IndividualInfo != nil {
			name := data.IndividualInfo.Name
			esia := data.IndividualInfo.ESIA

			result["applicant_name"] = fmt.Sprintf("физическое лицо %s", name)
			result["applicant_agent"] = ""
			result["is_organization"] = false

			if esia != "" {
				return fmt.Sprintf("%s (ЕСИА %s)", name, esia), result
			}
			return name, result
		}
	}

	return "", result
}

// DocxProcessor обрабатывает DOCX файлы
type DocxProcessor struct {
	templatePath string
	files        map[string][]byte
}

// NewDocxProcessor создает новый процессор DOCX
func NewDocxProcessor(templatePath string) (*DocxProcessor, error) {
	// Открываем DOCX как ZIP архив
	reader, err := zip.OpenReader(templatePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия шаблона: %v", err)
	}
	defer reader.Close()

	// Читаем все файлы из архива
	files := make(map[string][]byte)
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("ошибка открытия файла %s: %v", file.Name, err)
		}

		data, err := ioutil.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения файла %s: %v", file.Name, err)
		}

		files[file.Name] = data
	}

	return &DocxProcessor{
		templatePath: templatePath,
		files:        files,
	}, nil
}

// ReplaceContentControls заменяет содержимое элементов управления контентом
func (d *DocxProcessor) ReplaceContentControls(replacements map[string]interface{}) error {
	docXml, ok := d.files["word/document.xml"]
	if !ok {
		return fmt.Errorf("не найден файл document.xml")
	}

	content := string(docXml)
	fmt.Println("Заменяем следующие плейсхолдеры:")

	// Обработка условного блока для представителя юридического лица
	isOrg, hasIsOrg := replacements["is_organization"].(bool)
	agent, hasAgent := replacements["applicant_agent"].(string)
	if hasIsOrg && hasAgent {
		if isOrg && agent != "" {
			// Если это организация и есть представитель, добавляем информацию о представителе
			replacements["organization_agent_info"] = agent
		} else {
			// Если это не организация или нет представителя, помечаем для полного удаления
			replacements["organization_agent_info"] = ""
		}
	}

	// Обработка SDT контролов
	sdtPattern := `<w:sdt>.*?<w:tag w:val="([^"]+)".*?<w:sdtContent>(.*?)</w:sdtContent>.*?</w:sdt>`
	re := regexp.MustCompile(sdtPattern)

	// Находим все SDT контролы
	matches := re.FindAllStringSubmatch(content, -1)
	fmt.Printf("Найдено %d SDT контролов\n", len(matches))
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		tag := match[1]
		originalContent := match[2]
		fullMatch := match[0] // Полное совпадение SDT контрола

		// Ищем значение для замены
		value, exists := replacements[tag]
		if !exists {
			fmt.Printf("! Не найдено значение для тега '%s' в replacements\n", tag)
			fmt.Println("Доступные ключи в replacements:")
			for k := range replacements {
				fmt.Printf("- %s\n", k)
			}
			continue
		}

		var strValue string
		switch v := value.(type) {
		case string:
			strValue = v
		case int:
			strValue = fmt.Sprintf("%d", v)
		case bool:
			strValue = fmt.Sprintf("%v", v)
		case Dict:
			strValue = v.Value
		default:
			strValue = fmt.Sprintf("%v", v)
		}

		if strValue == "" {
			// Если значение пустое, полностью удаляем SDT контрол
			content = strings.Replace(content, fullMatch, "", 1)
			continue
		}

		// Сохраняем форматирование текста
		formattedValue := preserveFormatting(originalContent, strValue)

		// Заменяем содержимое SDT
		newContent := fmt.Sprintf(`<w:sdt><w:sdtPr><w:tag w:val="%s"/></w:sdtPr><w:sdtContent>%s</w:sdtContent></w:sdt>`,
			tag, formattedValue)
		content = strings.Replace(content, match[0], newContent, 1)
		fmt.Printf("- Заменили %s на %s\n", tag, strValue)
	}

	// Обрабатываем таблицу с registryItems
	if items, ok := replacements["registryItems"].([]Item); ok && len(items) > 0 {
		fmt.Println("Начинаем создание таблицы registryItems...")

		// Создаем новую таблицу
		var tableContent strings.Builder
		tableContent.WriteString(`<w:p/>
		<w:tbl>
			<w:tblPr>
				<w:tblStyle w:val="TableGrid"/>
				<w:tblW w:w="10000" w:type="dxa"/>
				<w:tblBorders>
					<w:top w:val="single" w:sz="4" w:space="0" w:color="auto"/>
					<w:left w:val="single" w:sz="4" w:space="0" w:color="auto"/>
					<w:bottom w:val="single" w:sz="4" w:space="0" w:color="auto"/>
					<w:right w:val="single" w:sz="4" w:space="0" w:color="auto"/>
					<w:insideH w:val="single" w:sz="4" w:space="0" w:color="auto"/>
					<w:insideV w:val="single" w:sz="4" w:space="0" w:color="auto"/>
				</w:tblBorders>
			</w:tblPr>
			<w:tblGrid>
				<w:gridCol w:w="1000"/>
				<w:gridCol w:w="1500"/>
				<w:gridCol w:w="4000"/>
				<w:gridCol w:w="1200"/>
				<w:gridCol w:w="1200"/>
				<w:gridCol w:w="1100"/>
			</w:tblGrid>`)

		// Добавляем заголовок таблицы
		tableContent.WriteString(`<w:tr>
			<w:tc>
				<w:tcPr><w:tcW w:w="1000" w:type="dxa"/></w:tcPr>
				<w:p><w:r><w:t>№</w:t></w:r></w:p>
			</w:tc>
			<w:tc>
				<w:tcPr><w:tcW w:w="1500" w:type="dxa"/></w:tcPr>
				<w:p><w:r><w:t>Инв. номер</w:t></w:r></w:p>
			</w:tc>
			<w:tc>
				<w:tcPr><w:tcW w:w="4000" w:type="dxa"/></w:tcPr>
				<w:p><w:r><w:t>Название</w:t></w:r></w:p>
			</w:tc>
			<w:tc>
				<w:tcPr><w:tcW w:w="1200" w:type="dxa"/></w:tcPr>
				<w:p><w:r><w:t>Дата</w:t></w:r></w:p>
			</w:tc>
			<w:tc>
				<w:tcPr><w:tcW w:w="1200" w:type="dxa"/></w:tcPr>
				<w:p><w:r><w:t>Реестровый номер</w:t></w:r></w:p>
			</w:tc>
			<w:tc>
				<w:tcPr><w:tcW w:w="1100" w:type="dxa"/></w:tcPr>
				<w:p><w:r><w:t>Примечание</w:t></w:r></w:p>
			</w:tc>
		</w:tr>`)

		// Добавляем строки с данными
		for i, item := range items {
			invNumber := getValueOrDefault(item.InvNumber, "-")
			name := getValueOrDefault(item.Name, "-")
			date := getValueOrDefault(formatDate(item.InformationDate), "-")
			note := getValueOrDefault(item.Note, "-")

			tableContent.WriteString(fmt.Sprintf(`<w:tr>
				<w:tc>
					<w:tcPr><w:tcW w:w="1000" w:type="dxa"/></w:tcPr>
					<w:p><w:r><w:t>%d</w:t></w:r></w:p>
				</w:tc>
				<w:tc>
					<w:tcPr><w:tcW w:w="1500" w:type="dxa"/></w:tcPr>
					<w:p><w:r><w:t>%s</w:t></w:r></w:p>
				</w:tc>
				<w:tc>
					<w:tcPr><w:tcW w:w="4000" w:type="dxa"/></w:tcPr>
					<w:p><w:r><w:t>%s</w:t></w:r></w:p>
				</w:tc>
				<w:tc>
					<w:tcPr><w:tcW w:w="1200" w:type="dxa"/></w:tcPr>
					<w:p><w:r><w:t>%s</w:t></w:r></w:p>
				</w:tc>
				<w:tc>
					<w:tcPr><w:tcW w:w="1200" w:type="dxa"/></w:tcPr>
					<w:p><w:r><w:t>%d</w:t></w:r></w:p>
				</w:tc>
				<w:tc>
					<w:tcPr><w:tcW w:w="1100" w:type="dxa"/></w:tcPr>
					<w:p><w:r><w:t>%s</w:t></w:r></w:p>
				</w:tc>
			</w:tr>`, i+1, invNumber, name, date, item.ID, note))
		}

		tableContent.WriteString("</w:tbl><w:p/>")

		// Находим место для вставки таблицы (перед закрывающим тегом body)
		bodyEndIndex := strings.LastIndex(content, "</w:body>")
		if bodyEndIndex != -1 {
			// Вставляем таблицу перед закрывающим тегом body
			content = content[:bodyEndIndex] + tableContent.String() + content[bodyEndIndex:]
			fmt.Printf("Создана таблица с %d строками\n", len(items))
		} else {
			fmt.Println("Не найдено место для вставки таблицы")
		}
	}

	d.files["word/document.xml"] = []byte(content)
	return nil
}

// Вспомогательные функции
func preserveFormatting(original, newValue string) string {
	// Проверяем, содержит ли оригинал ячейку таблицы
	if strings.Contains(original, "<w:tc>") {
		// Извлекаем форматирование ячейки
		tcPr := regexp.MustCompile(`<w:tcPr>.*?</w:tcPr>`).FindString(original)
		pPr := regexp.MustCompile(`<w:pPr>.*?</w:pPr>`).FindString(original)

		// Создаем новую ячейку с тем же форматированием
		return fmt.Sprintf(`<w:tc>%s<w:p>%s<w:r><w:t>%s</w:t></w:r></w:p></w:tc>`,
			tcPr, pPr, newValue)
	}

	// Извлекаем всю информацию о форматировании из оригинального контента
	runStart := regexp.MustCompile(`<w:r[^>]*>.*?<w:t[^>]*>`).FindString(original)
	runEnd := regexp.MustCompile(`</w:t>.*?</w:r>`).FindString(original)

	if runStart == "" || runEnd == "" {
		// Если форматирование не найдено, ищем хотя бы базовые теги
		rPrMatch := regexp.MustCompile(`<w:rPr>.*?</w:rPr>`).FindString(original)
		if rPrMatch != "" {
			// Если нашли информацию о форматировании, используем её
			return fmt.Sprintf("<w:r>%s<w:t>%s</w:t></w:r>", rPrMatch, newValue)
		}
		// Если совсем ничего не нашли, используем базовое форматирование
		return fmt.Sprintf("<w:r><w:t>%s</w:t></w:r>", newValue)
	}

	// Разбиваем значение на строки, если оно содержит переносы
	lines := strings.Split(newValue, "\n")
	if len(lines) == 1 {
		// Если нет переносов строк, просто вставляем значение
		return runStart + newValue + runEnd
	}

	// Если есть переносы строк, создаем несколько прогонов с тем же форматированием
	var result strings.Builder
	for i, line := range lines {
		if i > 0 {
			// Добавляем разрыв строки между прогонами
			result.WriteString("<w:br/>")
		}
		result.WriteString(runStart + line + runEnd)
	}
	return result.String()
}

func getValueOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// Save сохраняет документ
func (d *DocxProcessor) Save(outputPath string) error {
	// Создаем новый ZIP архив
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// Сначала читаем оригинальный файл для получения информации о файлах
	reader, err := zip.OpenReader(d.templatePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия шаблона при сохранении: %v", err)
	}
	defer reader.Close()

	// Выводим список файлов в оригинальном архиве
	fmt.Println("Файлы в оригинальном архиве:")
	for _, file := range reader.File {
		fmt.Printf("- %s (размер: %d)\n", file.Name, file.UncompressedSize64)
	}

	// Копируем все файлы из оригинального архива, заменяя только измененные
	for _, file := range reader.File {
		// Создаем новый файл в выходном архиве с теми же параметрами
		header := file.FileHeader
		writer, err := w.CreateHeader(&header)
		if err != nil {
			return fmt.Errorf("ошибка создания файла %s: %v", file.Name, err)
		}

		// Если это файл, который мы изменили - записываем измененную версию
		if data, ok := d.files[file.Name]; ok {
			fmt.Printf("Записываем измененный файл %s (размер: %d)\n", file.Name, len(data))
			if _, err := writer.Write(data); err != nil {
				return fmt.Errorf("ошибка записи измененного файла %s: %v", file.Name, err)
			}
			continue
		}

		// Иначе копируем оригинальный файл
		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("ошибка открытия файла %s: %v", file.Name, err)
		}

		written, err := io.Copy(writer, rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("ошибка копирования файла %s: %v", file.Name, err)
		}
		fmt.Printf("Скопирован файл %s (размер: %d)\n", file.Name, written)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("ошибка закрытия архива: %v", err)
	}

	// Записываем результат в файл
	data := buf.Bytes()
	fmt.Printf("Общий размер архива: %d байт\n", len(data))
	return ioutil.WriteFile(outputPath, data, 0644)
}

func main() {
	// Проверяем аргументы командной строки
	args := os.Args[1:]
	if len(args) < 1 {
		log.Println("Использование: go run scripts/docx_generator.go <путь к JSON файлу> [путь к шаблону]")
		log.Println("По умолчанию: JSON файл = test/exm2.json, шаблон = internal/domain/pdf/templates/template.docx")
		jsonPath := "test/exm2.json"
		templatePath := "internal/domain/pdf/templates/template.docx"
		processDocument(jsonPath, templatePath)
		return
	}

	// Получаем путь к JSON файлу из аргументов
	jsonPath := args[0]

	// Получаем путь к шаблону (опционально)
	templatePath := "internal/domain/pdf/templates/template.docx"
	if len(args) > 1 {
		templatePath = args[1]
	}

	processDocument(jsonPath, templatePath)
}

func processDocument(jsonPath, templatePath string) {
	// Загрузка JSON данных
	jsonData, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		log.Fatalf("Ошибка чтения JSON файла: %v", err)
	}

	var req Request
	if err := json.Unmarshal(jsonData, &req); err != nil {
		log.Fatalf("Ошибка разбора JSON: %v", err)
	}

	// Формирование информации о заявителе
	applicantInfo, applicantData := generateApplicantInfo(req)

	// Создание процессора
	processor, err := NewDocxProcessor(templatePath)
	if err != nil {
		log.Fatalf("Ошибка создания процессора: %v", err)
	}

	// Подготовка замен для шаблона
	replacements := map[string]interface{}{
		"id":                                     req.ID,
		"email":                                  req.Email,
		"phone":                                  req.Phone,
		"applicant_info":                         applicantInfo,
		"geoInfoStorageOrganization.value":       req.GeoInfoStorageOrganization.Value,
		"purposeOfGeoInfoAccessDictionary.value": req.PurposeOfGeoInfoAccessDict.Value,
		"creationDate":                           formatDate(req.CreationDate),
		"registry_pages":                         len(req.RegistryItems),
		"registryItems":                          req.RegistryItems,
	}

	// Добавляем данные из первого элемента registryItems, если он существует
	if len(req.RegistryItems) > 0 {
		// Добавляем значения как простой текст
		replacements["registry_item_name"] = req.RegistryItems[0].Name
		replacements["registry_item_inv_number"] = req.RegistryItems[0].InvNumber
		replacements["registry_item_date"] = formatDate(req.RegistryItems[0].InformationDate)
		replacements["registry_item_id"] = fmt.Sprintf("%d", req.RegistryItems[0].ID)
		replacements["registry_item_note"] = getValueOrDefault(req.RegistryItems[0].Note, "-")
	}

	// Добавляем данные о заявителе
	for k, v := range applicantData {
		replacements[k] = v
	}

	// Замена меток
	if err := processor.ReplaceContentControls(replacements); err != nil {
		log.Fatalf("Ошибка замены меток: %v", err)
	}

	// Генерируем уникальное имя файла на основе текущего времени
	timestamp := time.Now().Format("20060102_150405")
	outputPath := fmt.Sprintf("result_%s.docx", timestamp)

	// Сохранение результата
	if err := processor.Save(outputPath); err != nil {
		log.Fatalf("Ошибка сохранения документа: %v", err)
	}

	fmt.Printf("Документ успешно сохранен как %s!\n", outputPath)
}
