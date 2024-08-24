package elasticsearch

/*
doc := map[string]interface{}{
    "id": "123",
    "content": "Sample content",
    "timestamp": time.Now(),
}
err := searchIndex.IndexRecord("myindex", "123", doc, time.Now())
if err != nil {
    log.Fatal(err)
}


    FileSize:         321092,
    // ... other field values ...
}

err = searchIndex.IndexRecord("diagnoses", "unique-id-2", diagnosisRecord)
if err != nil {
    log.Fatal(err)
}


// SearchDateRange and other methods remain the same...

// Example usage
func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	searchIndex, err := NewSearchIndex(logger, "https://your-opensearch-endpoint.amazonaws.com")
	if err != nil {
		logger.Fatal("Failed to create SearchIndex", zap.Error(err))
	}

	// Example 1: Treatment Record
	treatmentRecord := struct {
		Certified   bool     `json:"certified"`
		CertifiedBy []string `json:"certifiedBy"`
		CreatedAt   string   `json:"createdAt"`
		Cure        string   `json:"cure"`
		Description string   `json:"description"`
		Image       string   `json:"image"`
		Name        string   `json:"name"`
		Price       string   `json:"price"`
		Type        string   `json:"type"`
	}{
		Certified:   true,
		CertifiedBy: []string{"BBH"},
		CreatedAt:   "1724482468775",
		Cure:        "typhoid",
		Description: "A treatment for colon cancer",
		Image:       "image from s3",
		Name:        "stm-mmm",
		Price:       "237",
		Type:        "medical",
	}

	err = searchIndex.IndexRecord("treatments", "unique-id-1", treatmentRecord)
	if err != nil {
		logger.Fatal("Failed to index treatment record", zap.Error(err))
	}

	// Example 2: Diagnosis Record
	diagnosisRecord := struct {
		DiagnosisSummary string    `json:"diagnosis_summary"`
		FileName         string    `json:"file_name"`
		FileSize         int       `json:"file_size"`
		FileType         string    `json:"file_type"`
		ID               int       `json:"id"`
		UploadPathS3     string    `json:"upload_path_s3"`
		UploadTime       time.Time `json:"upload_time"`
		UserID           string    `json:"user_id"`
	}{
		DiagnosisSummary: "check how I am feeling",
		FileName:         "carbon (17).png",
		FileSize:         321092,
		FileType:         "image",
		ID:               0,
		UploadPathS3:     "https://stm-streamex-dev-eu-west-2-services-object-store.s3.eu-west-2.amazonaws.com/diagnosis/diagnosis/d672a2a4-80d1-70c7-968d-2aeb7b04d917/2024-08-24/image/carbon (17).png",
		UploadTime:       time.Now(),
		UserID:           "d672a2a4-80d1-70c7-968d-2aeb7b04d917",
	}

	err = searchIndex.IndexRecord("diagnoses", "unique-id-2", diagnosisRecord)
	if err != nil {
		logger.Fatal("Failed to index diagnosis record", zap.Error(err))
	}

	logger.Info("Records indexed successfully")
}
*/

// Example Bulk indexing - I can bulk index from kafka topic, to speed up CPU time more business logic and
// deal with internally needed data movements separately

/*
// Example usage in main
func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	searchIndex, err := NewSearchIndex(logger, "https://your-opensearch-endpoint.amazonaws.com")
	if err != nil {
		logger.Fatal("Failed to create SearchIndex", zap.Error(err))
	}

	// Example: Bulk indexing treatment records
	treatmentRecords := []interface{}{
		struct {
			Certified   bool     `json:"certified"`
			CertifiedBy []string `json:"certifiedBy"`
			CreatedAt   string   `json:"createdAt"`
			Cure        string   `json:"cure"`
			Description string   `json:"description"`
			Image       string   `json:"image"`
			Name        string   `json:"name"`
			Price       string   `json:"price"`
			Type        string   `json:"type"`
		}{
			Certified:   true,
			CertifiedBy: []string{"BBH"},
			CreatedAt:   "1724482468775",
			Cure:        "typhoid",
			Description: "A treatment for typhoid",
			Image:       "image1.jpg",
			Name:        "stm-typhoid",
			Price:       "237",
			Type:        "medical",
		},
		struct {
			Certified   bool     `json:"certified"`
			CertifiedBy []string `json:"certifiedBy"`
			CreatedAt   string   `json:"createdAt"`
			Cure        string   `json:"cure"`
			Description string   `json:"description"`
			Image       string   `json:"image"`
			Name        string   `json:"name"`
			Price       string   `json:"price"`
			Type        string   `json:"type"`
		}{
			Certified:   false,
			CertifiedBy: []string{"NHS"},
			CreatedAt:   "1724482468780",
			Cure:        "malaria",
			Description: "A treatment for malaria",
			Image:       "image2.jpg",
			Name:        "stm-malaria",
			Price:       "300",
			Type:        "medical",
		},
	}

	err = searchIndex.BulkIndex("treatments", treatmentRecords)
	if err != nil {
		logger.Fatal("Failed to bulk index treatment records", zap.Error(err))
	}

	// Example: Bulk indexing diagnosis records
	diagnosisRecords := []interface{}{
		struct {
			DiagnosisSummary string    `json:"diagnosis_summary"`
			FileName         string    `json:"file_name"`
			FileSize         int       `json:"file_size"`
			FileType         string    `json:"file_type"`
			ID               int       `json:"id"`
			UploadPathS3     string    `json:"upload_path_s3"`
			UploadTime       time.Time `json:"upload_time"`
			UserID           string    `json:"user_id"`
		}{
			DiagnosisSummary: "check how I am feeling",
			FileName:         "carbon (17).png",
			FileSize:         321092,
			FileType:         "image",
			ID:               0,
			UploadPathS3:     "https://example.com/path1",
			UploadTime:       time.Now(),
			UserID:           "user1",
		},
		struct {
			DiagnosisSummary string    `json:"diagnosis_summary"`
			FileName         string    `json:"file_name"`
			FileSize         int       `json:"file_size"`
			FileType         string    `json:"file_type"`
			ID               int       `json:"id"`
			UploadPathS3     string    `json:"upload_path_s3"`
			UploadTime       time.Time `json:"upload_time"`
			UserID           string    `json:"user_id"`
		}{
			DiagnosisSummary: "follow-up examination",
			FileName:         "scan.jpg",
			FileSize:         500000,
			FileType:         "image",
			ID:               1,
			UploadPathS3:     "https://example.com/path2",
			UploadTime:       time.Now().Add(time.Hour),
			UserID:           "user2",
		},
	}

	err = searchIndex.BulkIndex("diagnoses", diagnosisRecords)
	if err != nil {
		logger.Fatal("Failed to bulk index diagnosis records", zap.Error(err))
	}

	logger.Info("Bulk indexing completed successfully")
}
*/
