package docx

import (
	"time"
)

// Template представляет DOCX шаблон с плейсхолдерами
type Template struct {
	Path string
	// Добавим кэширование скомпилированного шаблона позже
}

// Document представляет структуру документа
type Document struct {
	ID                     string         `json:"id"`
	Operation              string         `json:"operation"`
	CreationDate           time.Time      `json:"creationDate"`
	ApplicantType          string         `json:"applicantType"`
	OrganizationInfo       *Organization  `json:"organizationInfo,omitempty"`
	IndividualInfo         *Individual    `json:"individualInfo,omitempty"`
	RegistryItems          []RegistryItem `json:"registryItems"`
	PurposeOfGeoInfoAccess string         `json:"purposeOfGeoInfoAccess"`
}

// Organization представляет информацию об организации
type Organization struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Agent   string `json:"agent"`
}

// Individual представляет информацию о физическом лице
type Individual struct {
	Name string `json:"name"`
	Esia string `json:"esia,omitempty"`
}

// RegistryItem представляет элемент реестра
type RegistryItem struct {
	ID              string    `json:"id"`
	InformationDate time.Time `json:"informationDate"`
	Description     string    `json:"description"`
}

// TemplateData представляет данные для заполнения шаблона
type TemplateData struct {
	CreationDate           string         `json:"creationDate"`
	ApplicantInfo          string         `json:"applicant_info"`
	ApplicantName          string         `json:"applicant_name"`
	ApplicantAgent         string         `json:"applicant_agent"`
	IsOrganization         bool           `json:"is_organization"`
	PurposeOfGeoInfoAccess string         `json:"purposeOfGeoInfoAccess"`
	RegistryItems          []RegistryItem `json:"registryItems"`
}

// Validator интерфейс для валидации документа
type Validator interface {
	Validate() error
}

// Generator интерфейс для генерации DOCX файла
type Generator interface {
	Generate(doc *Document, outputPath string) error
}
