package pdf

import (
	"errors"
	"time"
)

// Определяем пользовательские ошибки
var (
	ErrTemplateNotFound = errors.New("template file not found")
)

// ... existing code ...

// Определение отсутствующих типов
type OrganizationInfo struct {
	Name    string `json:"name"`
	INN     string `json:"inn"`
	OGRN    string `json:"ogrn"`
	Address string `json:"address"`
	Agent   string `json:"agent"`
}

type IndividualInfo struct {
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	MiddleName      string `json:"middleName"`
	Document        string `json:"document"`
	PhoneNumber     string `json:"phoneNumber"`
	Email           string `json:"email"`
	Name            string `json:"name"`
	Esia            string `json:"esia"`
	AddressDocument string `json:"addressDocument"`
}

type RegistryItem struct {
	ID                  interface{} `json:"id"`
	Name                string      `json:"name"`
	Description         string      `json:"description"`
	InvNumber           string      `json:"invNumber"`           // Добавим поле инвентарного номера, если оно используется в шаблоне
	GeoInfoCarrierTypes string      `json:"geoInfoCarrierTypes"` // Добавляем недостающее поле
	InformationDate     string      `json:"informationDate"`     // Добавляем недостающее поле
}

type User struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

type DocxRequest struct {
	Operation                  string            `json:"operation"`
	ID                         string            `json:"id"`
	Email                      string            `json:"email"`
	Phone                      string            `json:"phone"`
	ApplicantType              string            `json:"applicantType"`
	OrganizationInfo           *OrganizationInfo `json:"organizationInfo"`
	IndividualInfo             *IndividualInfo   `json:"individualInfo"`
	PurposeOfGeoInfoAccess     string            `json:"purposeOfGeoInfoAccess"`
	PurposeOfGeoInfoAccessDict DictionaryValue   `json:"purposeOfGeoInfoAccessDictionary"`
	RegistryItems              []RegistryItem    `json:"registryItems"`
	CreatedBy                  User              `json:"createdBy"`
	VerifiedBy                 *User             `json:"verifiedBy"`
	CreationDate               time.Time         `json:"creationDate"`
	GeoInfoStorageOrganization DictionaryValue   `json:"geoInfoStorageOrganization"`
	Pages                      int               `json:"pages"`   // Количество страниц в документе
	IsDraft                    bool              `json:"isDraft"` // Флаг, указывающий что это черновик для подсчета страниц
	Status                     string            `json:"status"`  // Статус документа
}

type DictionaryValue struct {
	Code  string `json:"code,omitempty"`
	Value string `json:"value"`
}
