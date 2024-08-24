package elasticsearch

/*
First, create an index template:

doc := map[string]interface{}{
    "id": "123",
    "content": "Sample content",
    "timestamp": time.Now(),
}
err := searchIndex.IndexRecord("myindex", "123", doc, time.Now())
if err != nil {
    log.Fatal(err)
}

When indexing documents, include a timestamp and specify the date:
doc := map[string]interface{}{
    "id": "123",
    "content": "Sample content",
    "timestamp": time.Now(),
}
err := searchIndex.IndexRecord("myindex", "123", doc, time.Now())
if err != nil {
    log.Fatal(err)
}

To search across a date range:
startDate := time.Now().AddDate(0, 0, -7) // 7 days ago
endDate := time.Now()
results, err := searchIndex.SearchDateRange("myindex", startDate, endDate, map[string]string{"content": "sample"})
if err != nil {
    log.Fatal(err)
}
*/
