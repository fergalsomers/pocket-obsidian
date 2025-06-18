package page

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	_ "embed"
)

//go:embed testdata/meta_html.html
var sampleHTTML []byte

func TestPage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "testcsv suite")
}

type testContentDownloader struct {
	returnCodes map[string][]byte
}

func (t *testContentDownloader) Get(url string) ([]byte, string, error) {
	if t.returnCodes == nil {
		return nil, "", fmt.Errorf("%d : Not Found", http.StatusNotFound)
	}

	content := t.returnCodes[url]
	if content == nil {
		return nil, "", fmt.Errorf("%d : Not Found", http.StatusNotFound)
	}

	return content, "text/html", nil
}

var _ = Describe("PageTest", Ordered, func() {

	Context("PageTest", func() {

		It("Should create a page struct", func() {
			page, err := RecordToPage([]string{"Test Title", "http://example.com", "1746041473", "test|example", "unread"}, false, []string{})
			Expect(err).To(BeNil(), "Failed to create page from record")
			Expect(page.Title).To(Equal("Test Title"), "Expected title 'Test Title', got '%s'", page.Title)
			Expect(page.Url).To(Equal("http://example.com"), "Expected URL 'http://example.com', got '%s'", page.Url)
			Expect(page.TimeAdded).To(Equal(int64(1746041473)), "Expected timeAdded 1746041473, got %d", page.TimeAdded)
			Expect(page.Tags).To(Equal([]string{"test", "example"}), "Expected tags ['test,example'], got %v", page.Tags)
			Expect(page.Read).To(BeFalse(), "Expected read status to be false, got %v", page.Read)
		})

		It("Should create a page struct with markRead true", func() {
			page, err := RecordToPage([]string{"Test Title", "http://example.com", "1746041473", "test|example", "unread"}, true, []string{})
			Expect(err).To(BeNil(), "Failed to create page from record")
			Expect(page.Title).To(Equal("Test Title"), "Expected title 'Test Title', got '%s'", page.Title)
			Expect(page.Url).To(Equal("http://example.com"), "Expected URL 'http://example.com', got '%s'", page.Url)
			Expect(page.TimeAdded).To(Equal(int64(1746041473)), "Expected timeAdded 1746041473, got %d", page.TimeAdded)
			Expect(page.Tags).To(Equal([]string{"test", "example"}), "Expected tags ['test,example'], got %v", page.Tags)
			Expect(page.Read).To(BeTrue(), "Expected read status to be false, got %v", page.Read)
		})

		It("Should create a page struct with clippings tag inserted", func() {
			page, err := RecordToPage([]string{"Test Title", "http://example.com", "1746041473", "test|example", "unread"}, true, []string{"clippings"})
			Expect(err).To(BeNil(), "Failed to create page from record")
			Expect(page.Title).To(Equal("Test Title"), "Expected title 'Test Title', got '%s'", page.Title)
			Expect(page.Url).To(Equal("http://example.com"), "Expected URL 'http://example.com', got '%s'", page.Url)
			Expect(page.TimeAdded).To(Equal(int64(1746041473)), "Expected timeAdded 1746041473, got %d", page.TimeAdded)
			Expect(page.Tags).To(Equal([]string{"clippings", "test", "example"}), "Expected tags ['test,example'], got %v", page.Tags)
			Expect(page.Read).To(BeTrue(), "Expected read status to be false, got %v", page.Read)
		})

		It("Should write page to string", func() {
			page := Page{
				Title:     "Test Title",
				Url:       "http://example.com",
				TimeAdded: 1633036800,
				Tags:      []string{"test", "example"},
				Read:      false,
			}
			yamlString := page.YamlString()
			Expect(yamlString).To(ContainSubstring("title: Test Title"), "Expected YAML to contain title 'Test Title'")
			Expect(yamlString).To(ContainSubstring("url: http://example.com"), "Expected YAML to contain URL 'http://example.com'")
			Expect(yamlString).To(ContainSubstring("time_added: 1633036800"), "Expected YAML to contain time_added '1633036800'")
			Expect(yamlString).To(ContainSubstring("tags:\n    - test\n    - example"), "Expected YAML to contain tags 'test' and 'example'")
			Expect(yamlString).To(ContainSubstring("read: false"), "Expected YAML to contain read status 'false'")
		})

		It("Should read YAML bytes to page struct", func() {
			yamlData := []byte(`
title: Test Title
url: http://example.com
time_added: 1633036800
tags:
  - test
  - example
read: false
`)
			page, err := ReadPageYamlBytes(yamlData)
			Expect(err).To(BeNil(), "Failed to read YAML bytes to page struct")
			Expect(page.Title).To(Equal("Test Title"), "Expected title 'Test Title', got '%s'", page.Title)
			Expect(page.Url).To(Equal("http://example.com"), "Expected URL 'http://example.com', got '%s'", page.Url)
			Expect(page.TimeAdded).To(Equal(int64(1633036800)), "Expected timeAdded 1633036800, got %d", page.TimeAdded)
			Expect(page.Tags).To(Equal([]string{"test", "example"}), "Expected tags ['test', 'example'], got %v", page.Tags)
			Expect(page.Read).To(BeFalse(), "Expected read status to be false, got %v", page.Read)
		})

		It("Should convert HTML to Markdown", func() {
			htmlContent := `<h1>Test Title</h1><p>This is a test paragraph.</p>`

			markdown, err := ToMarkdown([]byte(htmlContent))
			Expect(err).To(BeNil(), "Failed to convert HTML to Markdown")
			log.Println("Markdown Output:", markdown)
			Expect(markdown).To(ContainSubstring("# Test Title"), "Expected Markdown to contain '# Test Title'")
			Expect(markdown).To(ContainSubstring("This is a test paragraph."), "Expected Markdown to contain 'This is a test paragraph.'")
		})

		It("Should write Clipping", func() {
			page := Page{
				Title:     "Test Title",
				Url:       "http://example.com",
				TimeAdded: 1633036800,
				Tags:      []string{"test", "example"},
				Read:      false,
			}
			htmlContent := `<h1>Test Title</h1><p>This is a test paragraph.</p>`
			markdown, err := ToMarkdown([]byte(htmlContent))
			Expect(err).To(BeNil(), "Failed to convert HTML to Markdown")

			c := NewClipping(&page, markdown)
			var b bytes.Buffer
			err = c.Write(&b)
			Expect(err).To(BeNil(), "Failed to write clipping")
		})

		It("Should read Clipping", func() {
			yamlData := []byte(`
---
title: Test Title
source: http://example.com
created: 2021-09-30
tags:
  - test
  - example
read: false
---
# Test Title
This is a test paragraph.
`)
			clipping, err := ReadClipping(bytes.NewReader(yamlData))
			dateAdded := time.Unix(int64(1633036800), 0).Format(time.DateOnly)

			Expect(err).To(BeNil(), "Failed to read clipping from YAML data")
			Expect(clipping.Metadata.Title).To(Equal("Test Title"), "Expected clipping title 'Test Title', got '%s'", clipping.Metadata.Title)
			Expect(clipping.Metadata.Source).To(Equal("http://example.com"), "Expected clipping URL 'http://example.com', got '%s'", clipping.Metadata.Source)
			Expect(clipping.Metadata.Created).To(Equal(dateAdded), "Expected clipping time_added %s, got %d", dateAdded, clipping.Metadata.Created)
			Expect(clipping.Metadata.Tags).To(Equal([]string{"test", "example"}), "Expected clipping tags ['test', 'example'], got %v", clipping.Metadata.Tags)
			Expect(clipping.Metadata.Read).To(BeFalse(), "Expected clipping read status to be false, got %v", clipping.Metadata.Read)
			Expect(clipping.MarkdownContent).To(ContainSubstring("# Test Title"), "Expected clipping Markdown to contain '# Test Title'")
			Expect(clipping.MarkdownContent).To(ContainSubstring("This is a test paragraph."), "Expected clipping Markdown to contain 'This is a test paragraph.'")
		})

		It("Should be able to read a written clipping", func() {
			page := Page{
				Title:     "Test Title",
				Url:       "http://example.com",
				TimeAdded: 1633036800,
				Tags:      []string{"test", "example"},
				Read:      false,
			}
			htmlContent := `<h1>Test Title</h1><p>This is a test paragraph.</p>`
			markdown, err := ToMarkdown([]byte(htmlContent))
			Expect(err).To(BeNil(), "Failed to convert HTML to Markdown")

			c := NewClipping(&page, markdown)
			var b bytes.Buffer
			err = c.Write(&b)
			Expect(err).To(BeNil(), "Failed to write clipping")

			// now read it back in
			c2, err := ReadClipping(bytes.NewReader(b.Bytes()))
			Expect(err).To(BeNil(), "Failed to read clipping from written data")
			Expect(c2.Metadata).To(Equal(c.Metadata), "Expected clipping page read from written data to be equal to original clipping")
			Expect(c2.MarkdownContent).To(Equal(c.MarkdownContent), "Expected clipping Markdown content read from written data to be equal to original clipping")
		})

	})

	Context("Metadata Tests", func() {

		It("Should parse metadata content", func() {
			fp := filepath.Join("testdata", "meta_html.html")
			file, err := os.Open(fp)
			Expect(err).To(BeNil(), "Failed to open test HTML file")
			defer file.Close()

			r := bufio.NewReader(file)
			article, err := getArticleMetadataFromReader(r, "http://example.org")
			Expect(err).To(BeNil())
			Expect(article.Description).NotTo(BeEmpty())
			Expect(article.Published).NotTo(BeEmpty())
			Expect(article.Published).To(Equal("2024-01-11"))
		})
	})

	It("Should parse dates with standard datetime", func() {
		s, err := ParseDateFromDateTimeString("2024-01-11T10:48:12-08:00")
		Expect(err).To(BeNil())
		Expect(s).To(Equal("2024-01-11"))
	})

	It("Should parse dates with time offset", func() {
		s, err := ParseDateFromDateTimeString("2024-09-26T12:15:14.000+00:00")
		Expect(err).To(BeNil())
		Expect(s).To(Equal("2024-09-26"))
	})

	It("Should parse dates with date", func() {
		s, err := ParseDateFromDateTimeString("2024-09-26")
		Expect(err).To(BeNil())
		Expect(s).To(Equal("2024-09-26"))
	})

	It("Should merge article contet", func() {
		c := Clipping{
			Metadata: ClippingMetadata{
				Title: "http:something",
			},
		}
		a := Article{
			Title:       "A title",
			Description: "A description",
			Authors:     []string{"fred"},
			Published:   "2020-01-15",
		}
		c.Decorate(&a)
		Expect(c.Metadata.Title).To(Equal(a.Title))
		Expect(c.Metadata.Description).To(Equal(a.Description))
		Expect(c.Metadata.Published).To(Equal(a.Published))
	})

	It("Should merge article contet", func() {
		c := Clipping{
			Metadata: ClippingMetadata{
				Title:       "original",
				Description: "original",
				Published:   "original",
			},
		}
		a := Article{
			Title:       "A title",
			Description: "A description",
			Authors:     []string{"fred"},
			Published:   "2020-01-15",
		}
		c.Decorate(&a)
		Expect(c.Metadata.Title).To(Equal("A title")) // this one is alway set
		Expect(c.Metadata.Description).To(Equal("original"))
		Expect(c.Metadata.Published).To(Equal("original"))
	})

	It("Should clean filenames", func() {
		orig := "akka/stream-design.rst at wip-stream-design-docs · akka/akka · GitHub"
		s := cleanFilename(orig)
		Expect(s).NotTo(Equal(orig))
	})

	It("Should test retrieve content from non- existent URL", func() {
		newstack := "https://thenewstack.io/why-kubernetes-needs-to-be-dumbed-down-for-devops/"
		r := &testContentDownloader{
			returnCodes: nil,
		}

		nonExistingURL := "http://nowhere.com"
		content, meta, err := r.Get(nonExistingURL)
		Expect(err).To(Not(BeNil()))
		Expect(meta).To((BeEmpty()), "Expected meta to be nil for non-existing URL")
		Expect(content).To(BeNil(), "Expected content to be nil for non-existing URL")

		r = &testContentDownloader{
			returnCodes: map[string][]byte{
				newstack: sampleHTTML,
			},
		}

		content, meta, err = r.Get(nonExistingURL)
		Expect(err).To(Not(BeNil()))
		Expect(meta).To((BeEmpty()), "Expected meta to be nil for non-existing URL")
		Expect(content).To(BeNil(), "Expected content to be nil for non-existing URL")
	})

	It("Should retrieve content from URL", func() {

		newstack := "https://thenewstack.io/why-kubernetes-needs-to-be-dumbed-down-for-devops/"
		r := &testContentDownloader{
			returnCodes: map[string][]byte{
				newstack: sampleHTTML,
			},
		}

		content, _, err := r.Get(newstack)
		Expect(err).To(BeNil())
		article, err := getArticleMetadataFromReader(bytes.NewReader(content), newstack)
		Expect(err).To(BeNil())
		Expect(article.Content).NotTo(BeEmpty())
		c := &Clipping{}
		c.Decorate(article)
		Expect(c.MarkdownContent).NotTo(BeNil())
	})

	It("Should extract content from a testDownlaoder", func() {
		newstack := "https://thenewstack.io/why-kubernetes-needs-to-be-dumbed-down-for-devops/"
		r := &testContentDownloader{
			returnCodes: map[string][]byte{
				newstack: sampleHTTML,
			},
		}
		article, error := ExtractArticleFromContent(r, newstack)
		log.Printf("Title %s", article.Title)
		log.Printf("%v", article)
		Expect(error).To(BeNil(), "Failed to extract article from content")
		Expect(article).NotTo(BeNil(), "Expected article to be not nil")
		Expect(article.Title).To(Equal("Using Istio Traffic Management on Amazon EKS to Enhance User Experience | Amazon Web Services"), "Expected article title to match")
	})

})
