package csv

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCSV(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "testcsv suite")
}

var _ = Describe("CSVTest", Ordered, func() {

	Context("RouteTest", func() {

		It("should parse a CSV file", func() {
			testFile := filepath.Join("testdata", "test.csv")
			records, err := ReadCSV(testFile)
			Expect(err).To(BeNil(), "Failed to read CSV file")
			Expect(records).ToNot(BeEmpty(), "No records found in the CSV file")
			expectedHeader := []string{"title", "url", "time_added", "tags", "status"} // title,url,time_added,tags,status
			Expect(records[0]).To(Equal(expectedHeader), "Expected header %v, got %v", expectedHeader, records[0])
			Expect(len(records)).To(Equal(6), "Expected 5 records, got %d", len(records))
		})
	})
})
