package pdf

import "time"

// ... existing code ...

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
}

type DictionaryValue struct {
	Code  string `json:"code,omitempty"`
	Value string `json:"value"`
}

type OrganizationInfo struct {
	ESIA            string `json:"esia"`
	Name            string `json:"name"`
	AddressDocument string `json:"addressDocument"`
}

type IndividualInfo struct {
	ESIA            string  `json:"esia"`
	Name            string  `json:"name"`
	AddressDocument *string `json:"addressDocument"`
}

type RegistryItem struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	InformationDate *string `json:"informationDate"`
	InvNumber       string  `json:"invNumber"`
	Note            *string `json:"note"`
}

type User struct {
	UserType string `json:"userType"`
	OID      string `json:"oid"`
	UserName string `json:"userName"`
	FullName string `json:"fullName"`
}
