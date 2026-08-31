package mpd

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
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

// Test_RootNamespaceDeclarationsPreserved guards the root xmlns declarations.
// encoding/xml drops or mangles them by default, which leaves any prefix used
// further down the manifest unbound.
func Test_RootNamespaceDeclarationsPreserved(t *testing.T) {
	tests := []struct {
		name         string
		rootAttrs    string
		wantDeclared []string
	}{
		{
			name:         "default namespace only",
			rootAttrs:    `xmlns="urn:mpeg:dash:schema:mpd:2011"`,
			wantDeclared: []string{`xmlns="urn:mpeg:dash:schema:mpd:2011"`},
		},
		{
			name:      "default and scte35",
			rootAttrs: `xmlns="urn:mpeg:dash:schema:mpd:2011" xmlns:scte35="https://www.scte.org/schemas/35"`,
			wantDeclared: []string{
				`xmlns="urn:mpeg:dash:schema:mpd:2011"`,
				`xmlns:scte35="https://www.scte.org/schemas/35"`,
			},
		},
		{
			name:      "drm prefixes",
			rootAttrs: `xmlns="urn:mpeg:dash:schema:mpd:2011" xmlns:cenc="urn:mpeg:cenc:2013" xmlns:mspr="urn:microsoft:playready" xmlns:mas="urn:marlin:mas:1-0:services:schemas:mpd"`,
			wantDeclared: []string{
				`xmlns:cenc="urn:mpeg:cenc:2013"`,
				`xmlns:mspr="urn:microsoft:playready"`,
				`xmlns:mas="urn:marlin:mas:1-0:services:schemas:mpd"`,
			},
		},
		{
			name:      "xlink and xsi",
			rootAttrs: `xmlns="urn:mpeg:dash:schema:mpd:2011" xmlns:ns2="http://www.w3.org/1999/xlink" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`,
			wantDeclared: []string{
				`xmlns:ns2="http://www.w3.org/1999/xlink"`,
				`xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<MPD ` + tt.rootAttrs + ` profiles="urn:mpeg:dash:profile:isoff-live:2011" type="dynamic">
  <Period id="p0"/>
</MPD>`)

			m := new(MPD)
			if err := m.Decode(in); err != nil {
				t.Fatalf("Decode: %v", err)
			}

			obtained, err := m.Encode()
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}

			for _, declaration := range tt.wantDeclared {
				assert.Equal(t, 1, strings.Count(string(obtained), declaration),
					"expected %s exactly once in:\n%s", declaration, obtained)
			}
		})
	}
}

// Test_EncodedPrefixesStayBound re-parses everything in testdata after a round
// trip and fails on any element whose prefix has no declaration in scope.
// encoding/xml does not reject unbound prefixes, it reports the raw prefix as
// the namespace, so this walks the tokens rather than relying on a decode error.
func Test_EncodedPrefixesStayBound(t *testing.T) {
	files, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("Failed to read testdata directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".mpd") {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			source, err := os.ReadFile("testdata/" + file.Name())
			if err != nil {
				t.Fatalf("Failed to read file %s: %v", file.Name(), err)
			}

			m := new(MPD)
			if err := m.Decode(source); err != nil {
				t.Fatalf("Decode: %v", err)
			}

			obtained, err := m.Encode()
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}

			assertPrefixesBound(t, obtained)
		})
	}
}

func assertPrefixesBound(t *testing.T, document []byte) {
	t.Helper()

	// One frame per open element, so a declaration goes out of scope with the
	// element that carried it rather than leaking into later siblings.
	scopes := []map[string]bool{{"": true}}
	decoder := xml.NewDecoder(bytes.NewReader(document))

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Fatalf("Failed to re-parse encoded output: %v\n%s", err, document)
		}

		switch element := token.(type) {
		case xml.StartElement:
			// Declarations take effect on the element that carries them.
			frame := map[string]bool{}
			for _, attr := range element.Attr {
				if attr.Name.Space == "xmlns" || attr.Name.Local == "xmlns" {
					frame[attr.Value] = true
				}
			}
			scopes = append(scopes, frame)

			// An unbound prefix is reported verbatim as the namespace, so
			// anything not matching a declaration in scope means the
			// declaration went missing.
			if !inScope(scopes, element.Name.Space) {
				t.Errorf("element %q uses unbound namespace prefix %q\n%s",
					element.Name.Local, element.Name.Space, document)
			}
		case xml.EndElement:
			scopes = scopes[:len(scopes)-1]
		}
	}
}

func inScope(scopes []map[string]bool, namespace string) bool {
	for i := len(scopes) - 1; i >= 0; i-- {
		if scopes[i][namespace] {
			return true
		}
	}

	return false
}

// Test_AncestorNamespaceDeclarationsPreserved covers declarations placed
// somewhere between the root and the Event payload. Event.InnerXML keeps
// prefixed children verbatim, so a declaration dropped from any ancestor
// leaves those prefixes unbound.
func Test_AncestorNamespaceDeclarationsPreserved(t *testing.T) {
	const declaration = ` xmlns:scte35="https://www.scte.org/schemas/35"`
	const payload = `<scte35:SpliceInfoSection tier="4095"><scte35:SpliceInsert spliceEventId="558"/></scte35:SpliceInfoSection>`

	tests := []struct {
		name        string
		mpd         string
		period      string
		eventStream string
		event       string
	}{
		{name: "declared on MPD", mpd: declaration},
		{name: "declared on Period", period: declaration},
		{name: "declared on EventStream", eventStream: declaration},
		{name: "declared on Event", event: declaration},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" profiles="urn:mpeg:dash:profile:isoff-live:2011" type="dynamic"` + tt.mpd + `>
  <Period id="p0"` + tt.period + `>
    <EventStream schemeIdUri="urn:scte:scte35:2013:xml" timescale="90000"` + tt.eventStream + `>
      <Event presentationTime="1" id="1"` + tt.event + `>` + payload + `</Event>
    </EventStream>
  </Period>
</MPD>`)

			m := new(MPD)
			if err := m.Decode(in); err != nil {
				t.Fatalf("Decode: %v", err)
			}

			obtained, err := m.Encode()
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}

			assert.Contains(t, string(obtained), payload, "SCTE-35 payload lost")
			assert.Equal(t, 1, strings.Count(string(obtained), `xmlns:scte35="https://www.scte.org/schemas/35"`),
				"expected the declaration exactly once in:\n%s", obtained)
			assertPrefixesBound(t, obtained)
		})
	}
}
