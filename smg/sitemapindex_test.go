package smg

import (
	"encoding/xml"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	baseURL     = "https://www.example.com"
	n           = 5
	letterBytes = "////abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var (
	lenLetters = len(letterBytes)
)

type SitemapIndexXml struct {
	XMLName xml.Name `xml:"sitemapindex"`
	Sitemaps    []Loc   `xml:"sitemap"`
}

type Loc struct {
	Loc     string   `xml:"loc"`
	LasMod  string   `xml:"lastmod"`
}

// TestCompleteAction tests the whole sitemap-generator module with a semi-basic usage
func TestCompleteAction(t *testing.T) {
	routes := buildRoutes(10, 40, 10)
	path := t.TempDir()

	smi := NewSitemapIndex(true)
	smi.SetCompress(false)
	smi.SetHostname(baseURL)
	smi.SetSitemapIndexName("test_sitemap_index")
	smi.SetOutputPath(path)
	now := time.Now().UTC()

	// Testing a list of named sitemaps
	names := []string{"test_sitemap1", "test_sitemap2", "test_sitemap3", "test_sitemap4", "test_sitemap5"}
	for _, name := range names {
		sm := smi.NewSitemap()
		sm.SetName(name)
		sm.SetLastMod(&now)
		for _, route := range routes {
			err := sm.Add(&SitemapLoc{
				Loc:        route,
				LastMod:    &now,
				ChangeFreq: Always,
				Priority:   0.4,
			})
			if err != nil {
				t.Fatal("Unable to add SitemapLoc:", name, err)
			}
		}
	}
	// -----------------------------

	// Testing another one with autogenerated name:
	smSixth := smi.NewSitemap()
	for _, route := range routes {
		err := smSixth.Add(&SitemapLoc{
			Loc:        route,
			LastMod:    &now,
			ChangeFreq: Daily,
			Priority:   0.8,
		})
		if err != nil {
			t.Fatal("Unable to add 6th SitemapLoc:", err)
		}
	}
	// -----------------------------

	indexFilename, err := smi.Save()
	if err != nil {
		t.Fatal("Unable to Save SitemapIndex:", err)
	}

	err = smi.PingSearchEngines()
	if err != nil {
		t.Fatal("Unable to Ping search engines:", err)
	}

	smi.SetCompress(true)
	indexCompressedFilename, err := smi.Save()
	if err != nil {
		t.Fatal("Unable to Save Compressed SitemapIndex:", err)
	}
	// -----------------------------------------------------------------

	// Checking the sitemap_index file, compressed file:
	// Compressed files;
	assertOutputFile(t, path, indexCompressedFilename)
	// Plain files:
	assertOutputFile(t, path, indexFilename)
	// -----------------------------------------------------------------

	// Checking 5 named output files
	for _, name := range names {
		// Compressed files;
		assertOutputFile(t, path, name+fileGzExt)
		// Plain files:
		assertOutputFile(t, path, name+fileExt)
	}
	// -----------------------------------------------------------------

	// Checking the 6th sitemap which was no-name:
	// Compressed files;
	assertOutputFile(t, path, "sitemap6"+fileGzExt)
	// Plain files:
	assertOutputFile(t, path, "sitemap6"+fileExt)
}

// TestLargeURLSetSitemap tests another one with 100001 items to be split to three files
func TestLargeURLSetSitemap(t *testing.T) {
	path := t.TempDir()

	smi := NewSitemapIndex(true)
	smi.SetCompress(false)
	smi.SetHostname(baseURL)
	smi.SetOutputPath(path)
	now := time.Now().UTC()

	smLarge := smi.NewSitemap()
	smLarge.SetName("fake_name_which_will_be_changed")
	moreRoutes := buildRoutes(100001, 40, 10)
	for _, route := range moreRoutes {
		err := smLarge.Add(&SitemapLoc{
			Loc:        route,
			LastMod:    &now,
			ChangeFreq: Hourly,
			Priority:   1,
		})
		if err != nil {
			t.Fatal("Unable to add large SitemapLoc:", err)
		}
	}
	// Testing changing Name after building a large sitemap which is split into several files
	smLarge.SetName("large")
	assertURLsCount(t, smLarge)

	indexFilename, err := smi.Save()
	if err != nil {
		t.Fatal("Unable to Save SitemapIndex:", err)
	}

	assertOutputFile(t, path, indexFilename)

	// Checking the larger sitemap which was no-name, file no. 1:
	assertOutputFile(t, path, "large"+fileExt)
	//  file no. 2:
	assertOutputFile(t, path, "large1"+fileExt)
	//  file no. 3:
	assertOutputFile(t, path, "large2"+fileExt)
}

// TestBigSizeSitemap test another one with long urls which makes file bigger than 50MG
// it must be split to two files
func TestBigSizeSitemap(t *testing.T) {
	path := t.TempDir()

	smi := NewSitemapIndex(true)
	smi.SetCompress(false)
	smi.SetHostname(baseURL)
	smi.SetOutputPath(path)
	now := time.Now().UTC()

	smBig := smi.NewSitemap()
	smBig.SetName("big")
	bigRoutes := buildRoutes(20000, 4000, 1000)
	for _, route := range bigRoutes {
		err := smBig.Add(&SitemapLoc{
			Loc:        route,
			LastMod:    &now,
			ChangeFreq: Hourly,
			Priority:   1,
		})
		if err != nil {
			t.Fatal("Unable to add large SitemapLoc:", err)
		}
	}

	indexFilename, err := smi.Save()
	if err != nil {
		t.Fatal("Unable to Save SitemapIndex:", err)
	}

	assertOutputFile(t, path, indexFilename)

	assertOutputFile(t, path, "big"+fileExt)
	// no. 2:
	assertOutputFile(t, path, "big1"+fileExt)
}

// TestSitemapIndexSave tests that on SitemapIndex.Save(), function produces a proper URL path to the sitemap
func TestSitemapIndexSave(t *testing.T) {
	path := "./tmp/sitemap_test"
	testLocation := "/test"
	testSitemapName := "test_sitemap_1"

	smi := NewSitemapIndex(true)
	smi.SetCompress(false)
	smi.SetHostname(baseURL)
	smi.SetSitemapIndexName("test_sitemap_index")
	smi.SetOutputPath(path)
	now := time.Now().UTC()

	sm := smi.NewSitemap()
	sm.SetName(testSitemapName)
	sm.SetLastMod(&now)

	err := sm.Add(&SitemapLoc{
		Loc:        testLocation,
		LastMod:    &now,
		ChangeFreq: Always,
		Priority:   0.4,
	})
	if err != nil {
		t.Fatal("Unable to add SitemapLoc test_sitemap_1: ", err)
	}

	expectedUrl := fmt.Sprintf("%s/%s.xml", baseURL, testSitemapName)
	sitemapFilepath, err := smi.Save()
	if err != nil {
		t.Fatal("Unable to Save Sitemap:", err)
	}
	xmlFile, err := os.Open(fmt.Sprintf("%s/%s", path, sitemapFilepath))
	if err != nil {
		t.Fatal("Unable to open file:", err)
	}
	defer xmlFile.Close()
	byteValue, _ := io.ReadAll(xmlFile)
	var sitemapIndex SitemapIndexXml
	err = xml.Unmarshal(byteValue, &sitemapIndex)
	if err != nil {
		t.Fatal("Unable to unmarhsall sitemap byte array into xml: ", err)
	}
	actualUrl := sitemapIndex.Sitemaps[0].Loc
	if actualUrl != expectedUrl {
		t.Fatal(fmt.Sprintf("URL Mismatch: \nActual: %s\nExpected: %s", actualUrl, expectedUrl))
	}
}

// TestSitemapIndexSaveWithServerURI tests that on SitemapIndex.Save(), function produces a proper URL path to the sitemap
func TestSitemapIndexSaveWithServerURI(t *testing.T) {
	path := "./tmp/sitemap_test"
	testLocation := "/test"
	testServerURI := "/server/"
	testSitemapName := "test_sitemap_1"

	smi := NewSitemapIndex(true)
	smi.SetCompress(false)
	smi.SetHostname(baseURL)
	smi.SetSitemapIndexName("test_sitemap_index")
	smi.SetOutputPath(path)
	smi.SetServerURI(testServerURI)
	now := time.Now().UTC()

	sm := smi.NewSitemap()
	sm.SetName(testSitemapName)
	sm.SetLastMod(&now)

	err := sm.Add(&SitemapLoc{
		Loc:        testLocation,
		LastMod:    &now,
		ChangeFreq: Always,
		Priority:   0.4,
	})
	if err != nil {
		t.Fatal("Unable to add SitemapLoc test_sitemap_1: ", err)
	}

	expectedUrl := fmt.Sprintf("%s%s%s.xml", baseURL, testServerURI, testSitemapName)
	sitemapFilepath, err := smi.Save()
	if err != nil {
		t.Fatal("Unable to Save Sitemap:", err)
	}
	xmlFile, err := os.Open(fmt.Sprintf("%s/%s", path, sitemapFilepath))
	if err != nil {
		t.Fatal("Unable to open file:", err)
	}
	defer xmlFile.Close()
	byteValue, _ := io.ReadAll(xmlFile)
	var sitemapIndex SitemapIndexXml
	err = xml.Unmarshal(byteValue, &sitemapIndex)
	if err != nil {
		t.Fatal("Unable to unmarhsall sitemap byte array into xml: ", err)
	}
	actualUrl := sitemapIndex.Sitemaps[0].Loc
	if actualUrl != expectedUrl {
		t.Fatal(fmt.Sprintf("URL Mismatch: \nActual: %s\nExpected: %s", actualUrl, expectedUrl))
	}
}

func assertOutputFile(t *testing.T, path, name string) {
	f, err := os.Stat(filepath.Join(path, name))
	if os.IsNotExist(err) || f.IsDir() {
		t.Fatal("File does not exist or is directory:", name, err)
	}
	if f.Size() == 0 {
		t.Fatal("Zero size:", name)
	} else if f.Size() > int64(maxFileSize) {
		t.Fatal("Size is more than limits:", name, f.Size())
	}
}

func assertURLsCount(t *testing.T, sm *Sitemap) {
	if sm.GetURLsCount() > maxURLsCount {
		t.Fatal("URLsCount is more than limits:", sm.Name, sm.GetURLsCount())
	}
}

func buildRoutes(n, l, s int) []string {
	rand.Seed(time.Now().UnixNano())

	routes := make([]string, n)
	for i := range routes {
		routes[i] = randString(rand.Intn(l) + s)
	}
	return routes
}

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(lenLetters)]
	}
	return string(b)
}
