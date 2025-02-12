package docx

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrEmptyID              = errors.New("id is required")
	ErrEmptyApplicantType   = errors.New("applicant type is required")
	ErrInvalidApplicantType = errors.New("invalid applicant type")
	ErrMissingOrganization  = errors.New("organization info is required for organization applicant")
	ErrMissingIndividual    = errors.New("individual info is required for individual applicant")
	ErrNoRegistryItems      = errors.New("at least one registry item is required")
	ErrEmptyPurpose         = errors.New("purpose of geo info access is required")
)

// DocumentValidator реализует интерфейс Validator для Document
type DocumentValidator struct {
	doc *Document
}

// NewDocumentValidator создает новый валидатор для документа
func NewDocumentValidator(doc *Document) *DocumentValidator {
	return &DocumentValidator{doc: doc}
}

// Validate проверяет корректность документа
func (v *DocumentValidator) Validate() error {
	var errors []string

	// Проверяем обязательные поля
	if v.doc.ID == "" {
		errors = append(errors, ErrEmptyID.Error())
	}

	if v.doc.ApplicantType == "" {
		errors = append(errors, ErrEmptyApplicantType.Error())
	} else {
		// Проверяем тип заявителя и наличие соответствующей информации
		switch v.doc.ApplicantType {
		case "ORGANIZATION":
			if v.doc.OrganizationInfo == nil {
				errors = append(errors, ErrMissingOrganization.Error())
			} else if err := v.validateOrganization(v.doc.OrganizationInfo); err != nil {
				errors = append(errors, err.Error())
			}
		case "INDIVIDUAL":
			if v.doc.IndividualInfo == nil {
				errors = append(errors, ErrMissingIndividual.Error())
			} else if err := v.validateIndividual(v.doc.IndividualInfo); err != nil {
				errors = append(errors, err.Error())
			}
		default:
			errors = append(errors, ErrInvalidApplicantType.Error())
		}
	}

	if len(v.doc.RegistryItems) == 0 {
		errors = append(errors, ErrNoRegistryItems.Error())
	} else {
		for i, item := range v.doc.RegistryItems {
			if err := v.validateRegistryItem(&item); err != nil {
				errors = append(errors, fmt.Sprintf("registry item %d: %s", i+1, err.Error()))
			}
		}
	}

	if v.doc.PurposeOfGeoInfoAccess == "" {
		errors = append(errors, ErrEmptyPurpose.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateOrganization проверяет информацию об организации
func (v *DocumentValidator) validateOrganization(org *Organization) error {
	var errors []string

	if org.Name == "" {
		errors = append(errors, "organization name is required")
	}
	if org.Address == "" {
		errors = append(errors, "organization address is required")
	}
	if org.Agent == "" {
		errors = append(errors, "organization agent is required")
	}

	if len(errors) > 0 {
		return fmt.Errorf("invalid organization info: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateIndividual проверяет информацию о физическом лице
func (v *DocumentValidator) validateIndividual(ind *Individual) error {
	if ind.Name == "" {
		return errors.New("individual name is required")
	}
	return nil
}

// validateRegistryItem проверяет элемент реестра
func (v *DocumentValidator) validateRegistryItem(item *RegistryItem) error {
	var errors []string

	if item.ID == "" {
		errors = append(errors, "registry item id is required")
	}
	if item.Description == "" {
		errors = append(errors, "registry item description is required")
	}
	if item.InformationDate.IsZero() {
		errors = append(errors, "registry item information date is required")
	}

	if len(errors) > 0 {
		return fmt.Errorf("invalid registry item: %s", strings.Join(errors, "; "))
	}

	return nil
}
