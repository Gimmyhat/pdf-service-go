package docx

import (
	"strings"
	"testing"
	"time"
)

func TestDocumentValidator_Validate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		doc     *Document
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid organization document",
			doc: &Document{
				ID:            "test-id",
				Operation:     "test-operation",
				CreationDate:  now,
				ApplicantType: "ORGANIZATION",
				OrganizationInfo: &Organization{
					Name:    "Test Org",
					Address: "Test Address",
					Agent:   "Test Agent",
				},
				RegistryItems: []RegistryItem{
					{
						ID:              "item-1",
						InformationDate: now,
						Description:     "Test Item",
					},
				},
				PurposeOfGeoInfoAccess: "Test Purpose",
			},
			wantErr: false,
		},
		{
			name: "valid individual document",
			doc: &Document{
				ID:            "test-id",
				Operation:     "test-operation",
				CreationDate:  now,
				ApplicantType: "INDIVIDUAL",
				IndividualInfo: &Individual{
					Name: "Test Person",
					Esia: "1234567890",
				},
				RegistryItems: []RegistryItem{
					{
						ID:              "item-1",
						InformationDate: now,
						Description:     "Test Item",
					},
				},
				PurposeOfGeoInfoAccess: "Test Purpose",
			},
			wantErr: false,
		},
		{
			name: "empty id",
			doc: &Document{
				Operation:     "test-operation",
				CreationDate:  now,
				ApplicantType: "INDIVIDUAL",
				IndividualInfo: &Individual{
					Name: "Test Person",
				},
				RegistryItems: []RegistryItem{
					{
						ID:              "item-1",
						InformationDate: now,
						Description:     "Test Item",
					},
				},
				PurposeOfGeoInfoAccess: "Test Purpose",
			},
			wantErr: true,
			errMsg:  "id is required",
		},
		{
			name: "missing organization info",
			doc: &Document{
				ID:            "test-id",
				Operation:     "test-operation",
				CreationDate:  now,
				ApplicantType: "ORGANIZATION",
				RegistryItems: []RegistryItem{
					{
						ID:              "item-1",
						InformationDate: now,
						Description:     "Test Item",
					},
				},
				PurposeOfGeoInfoAccess: "Test Purpose",
			},
			wantErr: true,
			errMsg:  "organization info is required",
		},
		{
			name: "missing individual info",
			doc: &Document{
				ID:            "test-id",
				Operation:     "test-operation",
				CreationDate:  now,
				ApplicantType: "INDIVIDUAL",
				RegistryItems: []RegistryItem{
					{
						ID:              "item-1",
						InformationDate: now,
						Description:     "Test Item",
					},
				},
				PurposeOfGeoInfoAccess: "Test Purpose",
			},
			wantErr: true,
			errMsg:  "individual info is required",
		},
		{
			name: "invalid applicant type",
			doc: &Document{
				ID:            "test-id",
				Operation:     "test-operation",
				CreationDate:  now,
				ApplicantType: "INVALID",
				RegistryItems: []RegistryItem{
					{
						ID:              "item-1",
						InformationDate: now,
						Description:     "Test Item",
					},
				},
				PurposeOfGeoInfoAccess: "Test Purpose",
			},
			wantErr: true,
			errMsg:  "invalid applicant type",
		},
		{
			name: "no registry items",
			doc: &Document{
				ID:            "test-id",
				Operation:     "test-operation",
				CreationDate:  now,
				ApplicantType: "INDIVIDUAL",
				IndividualInfo: &Individual{
					Name: "Test Person",
				},
				PurposeOfGeoInfoAccess: "Test Purpose",
			},
			wantErr: true,
			errMsg:  "at least one registry item is required",
		},
		{
			name: "empty purpose",
			doc: &Document{
				ID:            "test-id",
				Operation:     "test-operation",
				CreationDate:  now,
				ApplicantType: "INDIVIDUAL",
				IndividualInfo: &Individual{
					Name: "Test Person",
				},
				RegistryItems: []RegistryItem{
					{
						ID:              "item-1",
						InformationDate: now,
						Description:     "Test Item",
					},
				},
			},
			wantErr: true,
			errMsg:  "purpose of geo info access is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewDocumentValidator(tt.doc)
			err := v.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("DocumentValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("DocumentValidator.Validate() error message = %v, want to contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
