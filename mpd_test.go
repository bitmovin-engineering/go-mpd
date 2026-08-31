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

// Test_EventInnerXMLRoundTrip guards the Event element content against being
// dropped. EventType permits arbitrary element content, so anything other than
// a verbatim round trip silently corrupts SCTE-35 signalling.
func Test_EventInnerXMLRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "scte35 xml payload",
			content: `<scte35:SpliceInfoSection xmlns:scte35="https://www.scte.org/schemas/35" tier="4095"><scte35:SpliceInsert spliceEventId="558" outOfNetworkIndicator="true"><scte35:Program><scte35:SpliceTime ptsTime="5190214887"/></scte35:Program></scte35:SpliceInsert></scte35:SpliceInfoSection>`,
		},
		{
			name:    "scte35 binary payload",
			content: `<Signal xmlns="urn:scte:scte35:2013:xml"><Binary>/DAlAAAAAAAAAP/wFAUAAAABf+/+AA==</Binary></Signal>`,
		},
		{
			name:    "base64 chardata",
			content: `SGVsbG8gJiB3b3JsZA==`,
		},
		{
			name:    "escaped chardata",
			content: `text with &amp; escaped &lt;chars&gt;`,
		},
		{
			name:    "empty",
			content: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" profiles="urn:mpeg:dash:profile:isoff-live:2011" type="dynamic">
  <Period id="p0">
    <EventStream schemeIdUri="urn:scte:scte35:2013:xml" timescale="90000">
      <Event presentationTime="40814816400" duration="10800000" id="103571352">` + tt.content + `</Event>
    </EventStream>
  </Period>
</MPD>`)

			m := new(MPD)
			if err := m.Decode(in); err != nil {
				t.Fatalf("Decode: %v", err)
			}
			assert.Equal(t, tt.content, soleEventContent(t, m), "Event content lost on decode")

			obtained, err := m.Encode()
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}

			// Re-decode rather than substring match: Contains is a no-op for
			// the empty case, and would not notice the payload being escaped
			// into chardata on the way out.
			reDecoded := new(MPD)
			if err := reDecoded.Decode(obtained); err != nil {
				t.Fatalf("Decode of encoded output: %v\n%s", err, obtained)
			}
			assert.Equal(t, tt.content, soleEventContent(t, reDecoded), "Event content lost on encode:\n%s", obtained)
		})
	}
}

// soleEventContent returns the InnerXML of the single Event the test manifests
// carry, failing rather than panicking when the shape is not what we expect.
func soleEventContent(t *testing.T, m *MPD) string {
	t.Helper()

	if len(m.Period) != 1 {
		t.Fatalf("expected 1 Period, got %d", len(m.Period))
	}
	if len(m.Period[0].EventStreams) != 1 {
		t.Fatalf("expected 1 EventStream, got %d", len(m.Period[0].EventStreams))
	}

	events := m.Period[0].EventStreams[0].Events
	if len(events) != 1 {
		t.Fatalf("expected 1 Event, got %d", len(events))
	}

	return events[0].InnerXML
}
