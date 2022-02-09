package smg

import (
	"encoding/xml"
	"fmt"
	"github.com/jinzhu/copier"
	"io"
	"path/filepath"
	"time"
)

// ChangeFreq todo
type ChangeFreq string

// these consts! todo
const (
	Always  ChangeFreq = "always"
	Hourly  ChangeFreq = "hourly"
	Daily   ChangeFreq = "daily"
	Weekly  ChangeFreq = "weekly"
	Monthly ChangeFreq = "monthly"
	Yearly  ChangeFreq = "yearly"
	Never   ChangeFreq = "never"

	FileExt      string = ".xml"
	FileGzExt    string = ".xml.gz"
	MaxFileSize  int64  = 52428800
	MaxURLsCount int    = 50000
)

// Sitemap todo
type Sitemap struct {
	XMLName     xml.Name         `xml:"urlset"`
	Xmlns       string           `xml:"xmlns,attr"`
	Locs        []*SitemapLoc    `xml:"url" copier:"-"`
	PrettyPrint bool             `xml:"-"`
	Compress    bool             `xml:"-"`
	Name        string           `xml:"-"`
	Hostname    string           `xml:"-"`
	OutputPath  string           `xml:"-"`
	SitemapLoc  *SitemapIndexLoc `xml:"-"`
	NextSitemap *Sitemap         `xml:"-" copier:"-"`
	fileNum     int
	content     []byte
}

// NewSitemap returns a new Sitemap.
func NewSitemap() *Sitemap {
	t := time.Now().UTC()

	return &Sitemap{
		Xmlns:    "http://www.sitemaps.org/schemas/sitemap/0.9",
		Locs:     make([]*SitemapLoc, 0),
		Compress: true,
		SitemapLoc: &SitemapIndexLoc{
			LastMod: &t,
		},
	}
}

// Add adds an URL to a Sitemap.
// in case of exceeding the Sitemaps.org limits, splits the Sitemap into several Sitemap instances using a Linked list
func (s *Sitemap) Add(u *SitemapLoc) error {
	if s.NextSitemap != nil {
		s.NextSitemap.Add(u)
		return nil
	}

	if len(s.Locs) >= MaxURLsCount {
		s.buildNextSitemap()
		s.NextSitemap.Add(u)
		return nil
	}

	s.Locs = append(s.Locs, u)

	if n, err := s.CountXMLBytes(); err == nil && n >= MaxFileSize {
		s.Locs = s.Locs[:len(s.Locs)-1]
		s.buildNextSitemap()
		s.NextSitemap.Add(u)
		return nil
	} else if err != nil {
		return err
	}
	u.Loc = filepath.Join(s.Hostname, u.Loc)
	return nil
}

// buildNextSitemap builds a new Sitemap instance based on current one and connects to it via NextSitemap
func (s *Sitemap) buildNextSitemap() {
	s.NextSitemap = NewSitemap()
	copier.Copy(s.NextSitemap, s)
	s.NextSitemap.fileNum = s.fileNum + 1
}

// CountXMLBytes counts the number of bytes after encoding the XML sitemap to be able to split large files.
func (s *Sitemap) CountXMLBytes() (n int64, err error) {
	nilWriter := &JustCounterWriter{}
	_, err = nilWriter.Write([]byte(xml.Header))
	if err != nil {
		return 0, err
	}

	en := xml.NewEncoder(nilWriter)
	if s.PrettyPrint {
		en.Indent("", "  ")
	}
	err = en.Encode(s)
	_, err = nilWriter.Write([]byte{'\n'})
	return nilWriter.Count(), err
}

// WriteTo writes XML encoded sitemap to given io.Writer.
// Implements io.WriterTo interface.
func (s *Sitemap) WriteTo(writer io.Writer) (int64, error) {
	headerCount, err := writer.Write([]byte(xml.Header))
	if err != nil {
		return 0, err
	}
	en := xml.NewEncoder(writer)
	if s.PrettyPrint {
		en.Indent("", "  ")
	}
	err = en.Encode(s)
	if err != nil {
		return 0, err
	}

	bodyCount, err := writer.Write([]byte{'\n'})
	if err != nil {
		return 0, err
	}
	return int64(headerCount + bodyCount), err
}

// SetName sets the Name of Sitemap output xml file
// It must be without ".xml" extension
func (s *Sitemap) SetName(name string) {
	s.Name = name
}

// SetHostname sets the Hostname of Sitemap urls which be prepended to all URLs.
// Note: you do not have to call SetHostname in case you are building Sitemap using SitemapIndex.NewSitemap
// but you can set a separate Hostname for a specific Sitemap using SetHostname,
// else the SitemapIndex.SetHostname does this action for all Sitemaps of the entire SitemapIndex.
func (s *Sitemap) SetHostname(hostname string) {
	s.Hostname = hostname
}

// SetOutputPath sets the OutputPath of Sitemap which will be used to save the xml file.
// Note: you do not have to call SetOutputPath in case you are building Sitemap using SitemapIndex.NewSitemap
// but you can set a separate OutputPath for a specific Sitemap using SetOutputPath,
// else the SitemapIndex.SetOutputPath does this action for all Sitemaps of the entire SitemapIndex.
func (s *Sitemap) SetOutputPath(outputPath string) {
	s.OutputPath = outputPath
}

// SetLastMod sets the LastMod if this Sitemap which will be used in it's URL in SitemapIndex
func (s *Sitemap) SetLastMod(lastMod *time.Time) {
	s.SitemapLoc.LastMod = lastMod
}

// SetCompress sets the Compress option to be either enabled or disabled for Sitemap
// When Compress is enabled, the output file is compressed using gzip with .xml.gz extension.
func (s *Sitemap) SetCompress(compress bool) {
	s.Compress = compress
}

// SetPrettyPrint sets the PrettyPrint option to be either enabled or disabled for
// Sitemap. When PrettyPrint is enabled, the output file is easy to read and is
// recommended to be set to false for production use.
func (s *Sitemap) SetPrettyPrint(prettyPrint bool) {
	s.PrettyPrint = prettyPrint
}

// Save makes the OutputPath in case of absence and saves the Sitemap into OutputPath using it's Name.
// it returns the filename.
func (s *Sitemap) Save() (filenames []string, err error) {
	err = checkAndMakeDir(s.OutputPath)
	if err != nil {
		return
	}

	// Appends the fileNum at the end of filename in case of more than 0 (it is extended Sitemap)
	var filename string
	if s.fileNum > 0 {
		filename = fmt.Sprintf("%s%d", s.Name, s.fileNum)
	} else {
		filename = s.Name
	}

	if s.Compress {
		filename += FileGzExt
	} else {
		filename += FileExt
	}

	_, err = writeToFile(s, filename, s.OutputPath, s.Compress)

	if s.NextSitemap != nil {
		filenames, err = s.NextSitemap.Save()
		if err != nil {
			return nil, err
		}
	}
	return append(filenames, filename), nil
}
