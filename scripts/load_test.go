package main

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

// Тестовые наборы данных с разным количеством записей
var testSizes = []int{10, 100, 500, 1000, 5000}

// Количество итераций для каждого теста
const iterations = 5

func generateTestData(size int) Request {
	items := make([]Item, size)
	for i := 0; i < size; i++ {
		items[i] = Item{
			ID:              i + 1,
			Name:            fmt.Sprintf("Test Document %d", i+1),
			InformationDate: "2025-02-14T00:00:00Z",
			InvNumber:       fmt.Sprintf("INV-%d", i+1),
			Note:            fmt.Sprintf("Note %d", i+1),
		}
	}

	return Request{
		Operation:     "TEST",
		ID:            "TEST-ID",
		Email:         "test@test.com",
		Phone:         "123-456-789",
		ApplicantType: "ORGANIZATION",
		OrganizationInfo: &Organization{
			Name:    "Test Organization",
			Agent:   "Test Agent",
			Address: "Test Address",
		},
		PurposeOfGeoInfoAccess: "Test Purpose",
		PurposeOfGeoInfoAccessDict: Dict{
			Value: "Test Purpose Dict",
		},
		RegistryItems: items,
		CreationDate:  "2025-02-14T00:00:00Z",
		GeoInfoStorageOrganization: Dict{
			Value: "Test Storage",
		},
	}
}

func BenchmarkDocxGeneration(b *testing.B) {
	templatePath := "../test_docx/result.docx"

	for _, size := range testSizes {
		b.Run(fmt.Sprintf("Items_%d", size), func(b *testing.B) {
			testData := generateTestData(size)

			// Прогрев
			processTestDocument(testData, templatePath)

			var totalTime time.Duration
			var maxMemory uint64

			for i := 0; i < iterations; i++ {
				// Очищаем память перед каждым тестом
				runtime.GC()

				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				beforeAlloc := m.Alloc

				start := time.Now()
				processTestDocument(testData, templatePath)
				elapsed := time.Since(start)

				runtime.ReadMemStats(&m)
				memUsed := m.Alloc - beforeAlloc
				if memUsed > maxMemory {
					maxMemory = memUsed
				}

				totalTime += elapsed

				fmt.Printf("Size: %d, Iteration: %d, Time: %v, Memory: %v MB\n",
					size, i+1, elapsed, float64(memUsed)/1024/1024)
			}

			avgTime := totalTime / time.Duration(iterations)
			fmt.Printf("\nResults for %d items:\n", size)
			fmt.Printf("Average time: %v\n", avgTime)
			fmt.Printf("Max memory usage: %.2f MB\n\n", float64(maxMemory)/1024/1024)
		})
	}
}

func processTestDocument(req Request, templatePath string) {
	processor, err := NewDocxProcessor(templatePath)
	if err != nil {
		panic(err)
	}

	applicantInfo, applicantData := generateApplicantInfo(req)

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

	if len(req.RegistryItems) > 0 {
		replacements["registry_item_name"] = req.RegistryItems[0].Name
		replacements["registry_item_inv_number"] = req.RegistryItems[0].InvNumber
		replacements["registry_item_date"] = formatDate(req.RegistryItems[0].InformationDate)
		replacements["registry_item_id"] = fmt.Sprintf("%d", req.RegistryItems[0].ID)
		replacements["registry_item_note"] = getValueOrDefault(req.RegistryItems[0].Note, "-")
	}

	for k, v := range applicantData {
		replacements[k] = v
	}

	if err := processor.ReplaceContentControls(replacements); err != nil {
		panic(err)
	}

	outputPath := fmt.Sprintf("test_result_%d.docx", len(req.RegistryItems))
	if err := processor.Save(outputPath); err != nil {
		panic(err)
	}
}
