package mpd

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"

	"github.com/pschlump/xml-diff/xmllib"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func XmlClean(xmlString string, cfg xmllib.CfgType) string {
	xmlReader := strings.NewReader(xmlString)
	cleanXmlLeft, err := xmllib.ConvertXML(xmlReader, cfg)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	return cleanXmlLeft.String()
}

func Test_UnmarshalMarshalAllFiles(t *testing.T) {
	files, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("Failed to read testdata directory: %v", err)
	}

	for _, file := range files {

		if !file.IsDir() && strings.HasSuffix(file.Name(), ".mpd") {
			t.Run(file.Name(), func(t *testing.T) {
				expected, err := os.ReadFile("testdata/" + file.Name())
				if err != nil {
					t.Fatalf("Failed to read file %s: %v", file.Name(), err)
				}

				mpd := new(MPD)
				err = mpd.Decode(expected)
				if err != nil {
					assert.Fail(t, "Error decoding MPD", err)
				}

				obtained, err := mpd.Encode()
				if err != nil {
					assert.Fail(t, "Error encoding MPD", err)
				}

				cleanXmlLeft := XmlClean(string(expected), xmllib.CfgType{})
				cleanXmlRight := XmlClean(string(obtained), xmllib.CfgType{})

				dmp := diffmatchpatch.New()
				diffs := dmp.DiffMain(cleanXmlLeft, cleanXmlRight, false)
				if len(diffs) > 1 {
					// 1, because diff equal is always there
					t.Fatalf("%d Differences found:\n%s", len(diffs), dmp.DiffPrettyText(diffs))
				}
			})
		}
	}
}
