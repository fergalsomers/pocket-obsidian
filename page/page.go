package page

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	nurl "net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/flytam/filenamify"
	readability "github.com/go-shiori/go-readability"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v3"
)

type Page struct {
	Title     string   `yaml:"title"`
	Url       string   `yaml:"url"`
	TimeAdded int64    `yaml:"time_added"`
	Tags      []string `yaml:"tags"`
	Read      bool     `yaml:"read"`
}

func (p *Page) YamlBytes() []byte {
	yamlData, err := yaml.Marshal(p)
	if err != nil {
		panic(err)
	}

	return yamlData
}

func (p *Page) YamlString() string {
	return string(p.YamlBytes())
}

func ReadPageYamlBytes(b []byte) (*Page, error) {
	var page Page
	if err := yaml.Unmarshal(b, &page); err != nil {
		return nil, fmt.Errorf("error unmarshalling YAML: %w", err)
	}
	return &page, nil
}

func RecordToPage(record []string, markRead bool, mandatoryTags []string) (*Page, error) {
	timeAdded, err := strconv.ParseInt(record[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing time_added [%s]: %w", record[2], err)
	}
	page := Page{
		Title:     record[0],
		Url:       record[1],
		TimeAdded: timeAdded,
		Tags:      append(mandatoryTags, strings.Split(record[3], "|")...),
		Read:      record[4] != "unread" || markRead,
	}
	return &page, nil
}

func (p *Page) GetHTML() ([]byte, string, error) {
	return GetHTML(p.Url)
}

func GetHTML(url string) ([]byte, string, error) {
	// Attempt to retrieve the HTML content
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)

	if err != nil {
		// handle error (could be timeout)
		return nil, "", fmt.Errorf("error fetching URL %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("error retrieving URL code: %d, %s", resp.StatusCode, url)
	}

	contentType := resp.Header.Get("content-type")
	if !strings.HasPrefix(contentType, "text/html") {
		// this is not HTML - don't bother downloading.
		return nil, contentType, nil
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, contentType, fmt.Errorf("error parsing content from URL %s: %w", url, err)
	}

	return content, contentType, nil
}

func ToMarkdown(input []byte) ([]byte, error) {
	// Convert HTML to Markdown
	return htmltomarkdown.ConvertReader(bytes.NewReader(input))
}

type ClippingMetadata struct {
	Title       string   `yaml:"title"`
	Source      string   `yaml:"source"`
	Author      []string `yaml:"author"`
	Published   string   `yaml:"published"`
	Created     string   `yaml:"created"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	Read        bool     `yaml:"read"`
}

func (c *ClippingMetadata) YamlBytes() []byte {
	yamlData, err := yaml.Marshal(c)
	if err != nil {
		panic(err)
	}

	return yamlData
}

func NewClipping(p *Page, markdownContent []byte) *Clipping {

	unixTimeUTC := time.Unix(p.TimeAdded, 0)
	dateAdded := unixTimeUTC.Format(time.DateOnly)

	return &Clipping{
		Metadata: ClippingMetadata{
			Title:   p.Title,
			Source:  p.Url,
			Created: dateAdded,
			Tags:    p.Tags,
			Read:    p.Read,
			Author:  []string{}, // Placeholder for author, can be populated later
		},
		MarkdownContent: markdownContent,
	}
}

type Clipping struct {
	Metadata        ClippingMetadata
	MarkdownContent []byte
}

func ReadClippingMetadataYamlBytes(b []byte) (*ClippingMetadata, error) {
	var clippingMetadata ClippingMetadata
	if err := yaml.Unmarshal(b, &clippingMetadata); err != nil {
		return nil, fmt.Errorf("error unmarshalling YAML: %w", err)
	}
	return &clippingMetadata, nil
}

const (
	Delimiter = "---\n"
)

var ByteDelimiter = []byte(Delimiter)

func (c *Clipping) Write(w io.Writer) error {
	if _, err := w.Write(ByteDelimiter); err != nil {
		return err
	}
	if _, err := w.Write(c.Metadata.YamlBytes()); err != nil {
		return err
	}
	if _, err := w.Write(ByteDelimiter); err != nil {
		return err
	}
	if _, err := w.Write(c.MarkdownContent); err != nil {
		return err
	}
	return nil
}

func ReadClipping(r io.Reader) (*Clipping, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading clipping: %w", err)
	}

	parts := bytes.Split(b, ByteDelimiter)
	if len(parts) != 3 {
		return nil, fmt.Errorf("clipping does not contain enough parts, expected at least 2, got %d", len(parts))
	}

	c, err := ReadClippingMetadataYamlBytes(parts[1])
	if err != nil {
		return nil, fmt.Errorf("error reading YAML page: %w", err)
	}
	return &Clipping{
		Metadata:        *c,
		MarkdownContent: parts[2],
	}, nil
}

const (
	mediumTitlePocket = "A story from"
	medium404desc     = "On Medium, anyone can share insightful perspectives"
)

func (c *Clipping) Decorate(a *Article) {
	if a.Description != "" && c.Metadata.Description == "" {
		c.Metadata.Description = a.Description
	}
	if a.Published != "" && c.Metadata.Published == "" {
		c.Metadata.Published = a.Published
	}
	if a.Authors != nil && len(c.Metadata.Author) != 0 {
		c.Metadata.Author = a.Authors
	}
	if a.Title != "" {
		c.Metadata.Title = a.Title
	}

	md, err := htmltomarkdown.ConvertString(a.Content)
	if err == nil {
		c.MarkdownContent = []byte(md)
	} else {
		// parse the markdown from the retrieved article / readability content
		md, err := htmltomarkdown.ConvertNode(a.Node)
		if err != nil {
			c.MarkdownContent = md
		} else {
			// fallback to trying to parse the markdown directly from the web-page
			c.MarkdownContent = []byte("Unable to retrieve Markdown content")
		}
	}
}

type Article struct {
	Title       string
	Description string
	Published   string
	Authors     []string
	Content     string
	Node        *html.Node
}

func (a Article) String() string {
	return fmt.Sprintf("{Article: description=\"%s\", authors=%v, published=\"%s\"  }", a.Description, a.Authors, a.Published)
}

func (a *Article) IsEmpty() bool {
	return a.Description == "" && a.Published == "" && len(a.Authors) == 0
}

func (a *Article) IsFull() bool {
	return a.Description != "" && a.Published != "" && a.Authors != nil
}

func ExtractArticleFromContent(url string) (*Article, error) {
	content, _, err := GetHTML(url)
	if err != nil {
		return nil, err
	}
	if content == nil {
		return nil, nil
	}

	node, err := html.Parse(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %w", err)
	}
	article, err := getArticleMetadataFromNode(node, url)
	if err != nil {
		log.Printf("error extracting metadata from HTML: %v", err)
	}

	if strings.HasPrefix(article.Description, medium404desc) {
		return nil, fmt.Errorf("error Medium article doesn't exist %s", url)
	}
	return article, nil
}

func ParseDateFromDateTimeString(s string) (string, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t.Format(time.DateOnly), nil
	}
	t, err = time.Parse(time.DateOnly, s)
	if err == nil {
		return t.Format(time.DateOnly), nil
	}
	return "", err
}

// Use Readability to extract metadata

func getArticleMetadataFromNode(node *html.Node, url string) (*Article, error) {
	u, err := nurl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("error unnable to parse URL: %s - %v", url, err)
	}
	article, err := readability.FromDocument(node, u)
	if err != nil {
		return nil, err
	}
	a := Article{
		Authors:     []string{article.Byline},
		Title:       article.Title,
		Description: article.Excerpt,
		Content:     article.Content,
		Node:        node,
	}
	if article.PublishedTime != nil {
		a.Published = article.PublishedTime.Format(time.DateOnly)
	}
	return &a, nil
}

func getArticleMetadataFromReader(r io.Reader, url string) (*Article, error) {
	node, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %w", err)
	}
	return getArticleMetadataFromNode(node, url)
}

func CleanFilename(s string) string {
	s1, err := filenamify.Filenamify(s, filenamify.Options{})
	if err == nil {
		return s1
	} else {
		return s
	}
}

// ReccordToClipping
// Convert a CSV record to a Clippping and write it to the outputDir
// Also uses the clipping to create (cleaned) filename
// Pocket does not always get titles correct and processing can generate a better title.
func RecordToClipping(outputDir string, record []string, markRead bool, clippingTags []string) (*Clipping, error) {
	p, err := RecordToPage(record, markRead, clippingTags)
	if err != nil {
		log.Fatalf("Error converting record to page: %v", err)
	}

	c := NewClipping(p, nil)
	article, err := ExtractArticleFromContent(c.Metadata.Source)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve page %v", err)
	} else {
		if article != nil {
			c.Decorate(article)
		}
	}

	outputFile := filepath.Join(outputDir, CleanFilename(fmt.Sprintf("%s.md", c.Metadata.Title)))
	file, err := os.Create(outputFile)
	if err != nil {
		return nil, fmt.Errorf("error creating file %s: %v", outputFile, err)
	}
	defer file.Close()

	err = c.Write(file)
	if err != nil {
		return nil, fmt.Errorf("error writing clipping to file %s: %v", outputFile, err)
	}
	return c, nil
}
