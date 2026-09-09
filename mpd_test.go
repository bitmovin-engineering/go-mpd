package mpd

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

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

// assertPrefixesBound fails when the document uses a namespace prefix with no
// declaration in scope, on an element or on an attribute.
//
// This walks RawToken rather than Token deliberately. Token resolves a bound
// prefix to its URI but leaves an unbound one verbatim, so a check keyed on the
// resolved value cannot tell the two apart: in <root xmlns:a="b"><b:child/></root>
// the declared URI and the undeclared prefix are both "b" and the document
// looks fine. RawToken reports prefixes as written, so declarations can be
// tracked by prefix name instead.
func assertPrefixesBound(t *testing.T, document []byte) {
	t.Helper()

	unbound, err := unboundPrefixes(document)
	if err != nil {
		t.Fatalf("Failed to re-parse encoded output: %v\n%s", err, document)
	}

	for _, problem := range unbound {
		t.Errorf("%s\n%s", problem, document)
	}
}

// unboundPrefixes reports every namespace prefix used without a declaration in
// scope, on an element or on an attribute.
//
// This walks RawToken rather than Token deliberately. Token resolves a bound
// prefix to its URI but leaves an unbound one verbatim, so a check keyed on the
// resolved value cannot tell the two apart: in <root xmlns:a="b"><b:child/></root>
// the declared URI and the undeclared prefix are both "b" and the document
// looks fine. RawToken reports prefixes as written, so declarations can be
// tracked by prefix name instead.
func unboundPrefixes(document []byte) ([]string, error) {
	var unbound []string

	// One frame per open element, so a declaration goes out of scope with the
	// element that carried it rather than leaking into later siblings. The
	// empty prefix needs no declaration and xml is bound implicitly.
	scopes := []map[string]bool{{"": true, "xml": true}}
	decoder := xml.NewDecoder(bytes.NewReader(document))

	for {
		token, err := decoder.RawToken()
		if err == io.EOF {
			return unbound, nil
		}
		if err != nil {
			return nil, err
		}

		switch element := token.(type) {
		case xml.StartElement:
			// Declarations take effect on the element that carries them.
			scopes = append(scopes, declaredPrefixes(element.Attr))
			unbound = append(unbound, elementUnboundPrefixes(scopes, element)...)
		case xml.EndElement:
			scopes = popScope(scopes)
		}
	}
}

func elementUnboundPrefixes(scopes []map[string]bool, element xml.StartElement) []string {
	var unbound []string

	if !prefixInScope(scopes, element.Name.Space) {
		unbound = append(unbound, fmt.Sprintf("element %q uses unbound namespace prefix %q",
			element.Name.Local, element.Name.Space))
	}

	for _, attr := range element.Attr {
		if isNamespaceDeclaration(attr) {
			continue
		}

		if !prefixInScope(scopes, attr.Name.Space) {
			unbound = append(unbound, fmt.Sprintf("attribute %q on element %q uses unbound namespace prefix %q",
				attr.Name.Local, element.Name.Local, attr.Name.Space))
		}
	}

	return unbound
}

// declaredPrefixes returns the prefixes an element declares. RawToken reports
// xmlns:prefix="uri" as an attribute named prefix in the xmlns namespace.
func declaredPrefixes(attrs []xml.Attr) map[string]bool {
	declared := map[string]bool{}

	for _, attr := range attrs {
		if attr.Name.Space == "xmlns" {
			declared[attr.Name.Local] = true
		}
	}

	return declared
}

func isNamespaceDeclaration(attr xml.Attr) bool {
	return attr.Name.Space == "xmlns" || (attr.Name.Space == "" && attr.Name.Local == "xmlns")
}

func prefixInScope(scopes []map[string]bool, prefix string) bool {
	for i := len(scopes) - 1; i >= 0; i-- {
		if scopes[i][prefix] {
			return true
		}
	}

	return false
}

// popScope keeps the root frame, since RawToken does not check that start and
// end elements match.
func popScope(scopes []map[string]bool) []map[string]bool {
	if len(scopes) == 1 {
		return scopes
	}

	return scopes[:len(scopes)-1]
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

// Test_UnboundPrefixes covers the guard the namespace tests lean on. It has to
// agree with a namespace-aware parser, including on documents where the
// resolved-URI shortcut would be fooled.
func Test_UnboundPrefixes(t *testing.T) {
	tests := []struct {
		name     string
		document string
		want     int
	}{
		{
			name:     "no prefixes at all",
			document: `<MPD xmlns="urn:mpeg:dash:schema:mpd:2011"><Period id="p0"/></MPD>`,
		},
		{
			name:     "declared and used",
			document: `<MPD xmlns="urn:d" xmlns:scte35="urn:s"><Period scte35:x="1"><scte35:Signal/></Period></MPD>`,
		},
		{
			name:     "xml prefix needs no declaration",
			document: `<MPD xmlns="urn:d"><Period xml:lang="en"/></MPD>`,
		},
		{
			name:     "unbound element prefix",
			document: `<MPD xmlns="urn:d"><scte35:Signal/></MPD>`,
			want:     1,
		},
		{
			name:     "unbound attribute prefix",
			document: `<MPD xmlns="urn:d"><Period xlink:href="x"/></MPD>`,
			want:     1,
		},
		{
			name: "prefix matching a declared uri",
			// Token would resolve the declaration to "b" and report the
			// undeclared prefix as "b" too, so the two become
			// indistinguishable. RawToken keeps them apart.
			document: `<MPD xmlns:a="b"><b:child/></MPD>`,
			want:     1,
		},
		{
			name:     "declaration does not leak to a sibling",
			document: `<MPD xmlns="urn:d"><Period xmlns:a="urn:a"><a:X/></Period><Period><a:Y/></Period></MPD>`,
			want:     1,
		},
		{
			name:     "declaration reaches a descendant",
			document: `<MPD xmlns="urn:d" xmlns:a="urn:a"><Period><EventStream><a:X/></EventStream></Period></MPD>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unbound, err := unboundPrefixes([]byte(tt.document))
			if err != nil {
				t.Fatalf("unboundPrefixes: %v", err)
			}

			assert.Len(t, unbound, tt.want, "got %v", unbound)
		})
	}
}
