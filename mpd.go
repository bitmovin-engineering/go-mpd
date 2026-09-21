// Package mpd implements parsing and generating of MPEG-DASH Media Presentation Description (MPD) files.
package mpd

import (
	"bytes"
	"encoding/xml"
	"strings"
)

// http://mpeg.chiariglione.org/standards/mpeg-dash
// https://www.brendanlong.com/the-structure-of-an-mpeg-dash-mpd.html
// http://standards.iso.org/ittf/PubliclyAvailableStandards/MPEG-DASH_schema_files/DASH-MPD.xsd

// MPD represents root XML element.
type MPD struct {
	// Namespaces holds the namespace declarations found on the root element.
	// encoding/xml cannot round-trip these through struct tags: it reports
	// them as attributes in the "xmlns" namespace when decoding and mangles
	// them into "_xmlns:prefix" when encoding, so UnmarshalXML/MarshalXML
	// carry them instead.
	//
	// Losing them corrupts any content that uses a prefix. An scte35: SCTE-35
	// payload kept verbatim in Event.InnerXML is the case that motivated this:
	// without its declaration the prefix is unbound and the manifest is no
	// longer namespace-well-formed.
	Namespaces   []xml.Attr `xml:"-"`
	ForeignAttrs []xml.Attr `xml:"-"`

	SchemaLocation             *string               `xml:"schemaLocation,attr"`
	Type                       *string               `xml:"type,attr"`
	MinimumUpdatePeriod        *string               `xml:"minimumUpdatePeriod,attr"`
	AvailabilityStartTime      *string               `xml:"availabilityStartTime,attr"`
	AvailabilityEndTime        *string               `xml:"availabilityEndTime,attr"`
	MediaPresentationDuration  *string               `xml:"mediaPresentationDuration,attr"`
	MinBufferTime              *string               `xml:"minBufferTime,attr"`
	SuggestedPresentationDelay *string               `xml:"suggestedPresentationDelay,attr"`
	TimeShiftBufferDepth       *string               `xml:"timeShiftBufferDepth,attr"`
	PublishTime                *string               `xml:"publishTime,attr"`
	Profiles                   *string               `xml:"profiles,attr"`
	Id                         *string               `xml:"id,attr"`
	MaxSegmentDuration         *string               `xml:"maxSegmentDuration,attr"`
	MaxSubsegmentDuration      *string               `xml:"maxSubsegmentDuration,attr"`
	BaseURL                    []*BaseURL            `xml:"BaseURL,omitempty"`
	Period                     []*Period             `xml:"Period,omitempty"`
	ProgramInformations        []*ProgramInformation `xml:"ProgramInformation,omitempty"`
	Locations                  []*string             `xml:"Location,omitempty"`
	Metrics                    []*Metrics            `xml:"Metrics,omitempty"`
	EssentialProperties        []*Descriptor         `xml:"EssentialProperty,omitempty"`
	SupplementalProperties     []*Descriptor         `xml:"SupplementalProperty,omitempty"`
	UtcTimings                 []*UtcTiming          `xml:"UTCTiming,omitempty"`
}

// Do not try to use encoding.TextMarshaler and encoding.TextUnmarshaler:
// https://github.com/golang/go/issues/6859#issuecomment-118890463

// Encode generates MPD XML.
// Namespace declarations have to be carried by hand. encoding/xml reports them
// as attributes in the "xmlns" namespace when decoding and mangles them into
// "_xmlns:prefix" when encoding, so struct tags cannot round-trip them.
//
// Every element between the root and Event content needs this, not just the
// root: Event.InnerXML keeps prefixed children verbatim, so a declaration on
// any ancestor that goes missing leaves those prefixes unbound and the
// manifest no longer namespace-well-formed. Each element re-emits exactly the
// declarations it carried, which preserves the original scoping.

// UnmarshalXML decodes the MPD and keeps its namespace declarations.
func (m *MPD) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// plain drops the methods so DecodeElement does not recurse.
	type plain MPD

	if err := d.DecodeElement((*plain)(m), &start); err != nil {
		return err
	}

	m.Namespaces = namespaceDeclarations(start.Attr)
	foreign, bindings := foreignAttributes(start.Attr, m.Namespaces)
	m.ForeignAttrs = foreign
	m.Namespaces = append(m.Namespaces, bindings...)

	// ForeignAttrs already carries a qualified schemaLocation.
	if hasForeignAttr(m.ForeignAttrs, "schemaLocation") {
		m.SchemaLocation = nil
	}

	return nil
}

// MarshalXML encodes the MPD and restores its namespace declarations and
// qualified attributes.
func (m *MPD) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	type plain MPD

	start = withNamespaces(start, m.Namespaces)

	return e.EncodeElement((*plain)(m), withNamespaces(start, m.ForeignAttrs))
}

// UnmarshalXML decodes the Period and keeps its namespace declarations.
func (p *Period) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type plain Period

	if err := d.DecodeElement((*plain)(p), &start); err != nil {
		return err
	}

	p.Namespaces = namespaceDeclarations(start.Attr)

	return nil
}

// MarshalXML encodes the Period and restores its namespace declarations.
func (p *Period) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	type plain Period

	return e.EncodeElement((*plain)(p), withNamespaces(start, p.Namespaces))
}

// UnmarshalXML decodes the EventStream and keeps its namespace declarations.
func (s *EventStream) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type plain EventStream

	if err := d.DecodeElement((*plain)(s), &start); err != nil {
		return err
	}

	s.Namespaces = namespaceDeclarations(start.Attr)

	return nil
}

// MarshalXML encodes the EventStream and restores its namespace declarations.
func (s *EventStream) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	type plain EventStream

	return e.EncodeElement((*plain)(s), withNamespaces(start, s.Namespaces))
}

// UnmarshalXML decodes the Event and keeps its namespace declarations, which
// are often the ones its InnerXML payload depends on.
func (v *Event) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type plain Event

	if err := d.DecodeElement((*plain)(v), &start); err != nil {
		return err
	}

	v.Namespaces = namespaceDeclarations(start.Attr)

	return nil
}

// MarshalXML encodes the Event and restores its namespace declarations.
func (v *Event) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	type plain Event

	return e.EncodeElement((*plain)(v), withNamespaces(start, v.Namespaces))
}

// UnmarshalXML decodes the ContentProtection and keeps its namespace
// declarations and qualified attributes.
func (c *ContentProtection) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type plain ContentProtection

	if err := d.DecodeElement((*plain)(c), &start); err != nil {
		return err
	}

	c.Namespaces = namespaceDeclarations(start.Attr)
	foreign, bindings := foreignAttributes(start.Attr, c.Namespaces)
	c.ForeignAttrs = foreign
	c.Namespaces = append(c.Namespaces, bindings...)

	// ForeignAttrs already carries a qualified default_KID.
	if hasForeignAttr(c.ForeignAttrs, "default_KID") {
		c.DefaultKID = nil
	}

	return nil
}

// MarshalXML encodes the ContentProtection and restores its namespace
// declarations and qualified attributes.
func (c *ContentProtection) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	type plain ContentProtection

	start = withNamespaces(start, c.Namespaces)

	return e.EncodeElement((*plain)(c), withNamespaces(start, c.ForeignAttrs))
}

// withNamespaces adds the declarations to a start element. The element name is
// left alone: EventStream, for one, is encoded as both EventStream and
// InbandEventStream depending on the field it came from.
func withNamespaces(start xml.StartElement, declarations []xml.Attr) xml.StartElement {
	start.Attr = append(append([]xml.Attr(nil), start.Attr...), declarations...)

	return start
}

// namespaceDeclarations picks the xmlns declarations out of a start element's
// attributes and rewrites them into a form the encoder emits verbatim: writing
// the whole "xmlns:prefix" as the attribute's local name sidesteps encoding/xml
// rewriting the prefix.
func namespaceDeclarations(attrs []xml.Attr) []xml.Attr {
	var declarations []xml.Attr

	for _, attr := range attrs {
		switch {
		case attr.Name.Space == "xmlns":
			// xmlns:prefix="uri"
			declarations = append(declarations, xml.Attr{
				Name:  xml.Name{Local: "xmlns:" + attr.Name.Local},
				Value: attr.Value,
			})
		case attr.Name.Space == "" && attr.Name.Local == "xmlns":
			// xmlns="uri"
			declarations = append(declarations, xml.Attr{
				Name:  xml.Name{Local: "xmlns"},
				Value: attr.Value,
			})
		}
	}

	return declarations
}

// conventionalPrefixes maps a namespace to the prefix DASH manifests write it
// with, used when the declaration is not on the element carrying the attribute.
var conventionalPrefixes = map[string]string{
	"urn:mpeg:cenc:2013":                        "cenc",
	"urn:microsoft:playready":                   "mspr",
	"urn:marlin:mas:1-0:services:schemas:mpd":   "mas",
	"http://www.w3.org/2001/XMLSchema-instance": "xsi",
	"http://www.w3.org/1999/xlink":              "ns2",
	"https://www.scte.org/schemas/35":           "scte35",
}

// foreignAttributes picks the namespace-qualified attributes out of a start
// element's attributes, rewritten the way namespaceDeclarations rewrites
// declarations. Without this a tag such as `xml:"cenc:default_KID,attr"` never
// matches and the unprefixed field catches the value, dropping the prefix.
//
// The second return holds the declarations needed to bind any prefix taken
// from conventionalPrefixes. The manifest may declare that namespace under a
// different prefix, which would leave the one written here unbound.
func foreignAttributes(attrs []xml.Attr, declarations []xml.Attr) ([]xml.Attr, []xml.Attr) {
	var foreign, bindings []xml.Attr

	for _, attr := range attrs {
		if attr.Name.Space == "" || attr.Name.Space == "xmlns" {
			continue
		}

		prefix, needsBinding := prefixFor(attr.Name.Space, declarations)
		if prefix == "" {
			continue
		}

		foreign = append(foreign, xml.Attr{
			Name:  xml.Name{Local: prefix + ":" + attr.Name.Local},
			Value: attr.Value,
		})

		if needsBinding && !declaresPrefix(bindings, prefix) {
			bindings = append(bindings, xml.Attr{
				Name:  xml.Name{Local: "xmlns:" + prefix},
				Value: attr.Name.Space,
			})
		}
	}

	return foreign, bindings
}

// prefixFor recovers the prefix an attribute was written with, which the
// decoder replaces with the namespace it resolved to. The second return says
// whether a declaration has to be added to bind the prefix.
//
// A prefix that was already unbound in the source is reported by the decoder
// as itself rather than a namespace, and is returned unchanged: there is no
// namespace to bind it to, and the output stays faithful to the input.
func prefixFor(namespace string, declarations []xml.Attr) (string, bool) {
	for _, declaration := range declarations {
		if declaration.Value == namespace {
			if prefix, found := strings.CutPrefix(declaration.Name.Local, "xmlns:"); found {
				return prefix, false
			}
		}
	}

	if prefix, known := conventionalPrefixes[namespace]; known {
		return prefix, true
	}

	if !strings.ContainsAny(namespace, ":/") {
		return namespace, false
	}

	return "", false
}

// declaresPrefix reports whether these declarations already bind the prefix.
func declaresPrefix(declarations []xml.Attr, prefix string) bool {
	for _, declaration := range declarations {
		if declaration.Name.Local == "xmlns:"+prefix {
			return true
		}
	}

	return false
}

// hasForeignAttr reports whether a qualified attribute with this local name is
// already carried.
func hasForeignAttr(foreign []xml.Attr, local string) bool {
	for _, attr := range foreign {
		if strings.HasSuffix(attr.Name.Local, ":"+local) {
			return true
		}
	}

	return false
}

func (m *MPD) Encode() ([]byte, error) {
	x := new(bytes.Buffer)
	e := xml.NewEncoder(x)
	e.Indent("", "  ")
	err := e.Encode(m)
	if err != nil {
		return nil, err
	}

	return x.Bytes(), nil
}

// Decode parses MPD XML.
func (m *MPD) Decode(b []byte) error {
	return xml.Unmarshal(b, m)
}

type UrlType struct {
	Value     string  `xml:",chardata"`
	SourceURL *string `xml:"sourceURL,attr"`
	Range     *string `xml:"range,attr"`
}

type ProgramInformation struct {
	Title              string  `xml:"Title,omitempty"`
	Source             string  `xml:"Source,omitempty"`
	Copyright          string  `xml:"Copyright,omitempty"`
	Lang               *string `xml:"lang,attr,omitempty"`
	MoreInformationURL *string `xml:"moreInformationURL,attr,omitempty"`
}

type SegmentBase struct {
	Initialization           *UrlType `xml:"Initialization,omitempty"`
	RepresentationIndex      *UrlType `xml:"RepresentationIndex,omitempty"`
	Timescale                *uint64  `xml:"timescale,attr,omitempty"`
	PresentationTimeOffset   *uint64  `xml:"presentationTimeOffset,attr,omitempty"`
	IndexRange               *string  `xml:"indexRange,attr,omitempty"`
	IndexRangeExact          *bool    `xml:"indexRangeExact,attr,omitempty"`
	AvailabilityTimeOffset   *float64 `xml:"availabilityTimeOffset,attr,omitempty"`
	AvailabilityTimeComplete *bool    `xml:"availabilityTimeComplete,attr,omitempty"`
}

type SegmentURL struct {
	Media      *string `xml:"media,attr"`
	MediaRange *string `xml:"mediaRange,attr"`
	Index      *string `xml:"index,attr"`
	IndexRange *string `xml:"indexRange,attr"`
}

type SegmentList struct {
	SegmentBase
	SegmentURLs []*SegmentURL `xml:"SegmentURL,omitempty"`
	Href        *string       `xml:"href,attr"`
	Actuate     *string       `xml:"actuate,attr"`
}

type Event struct {
	// Namespaces holds the xmlns declarations carried by this element. A
	// prefix used inside InnerXML may be declared here rather than higher up,
	// and dropping the declaration would leave it unbound. See MPD.Namespaces.
	Namespaces []xml.Attr `xml:"-"`

	// InnerXML holds the element content verbatim. EventType allows arbitrary
	// element content (xs:any), so this cannot be a chardata field: SCTE-35
	// payloads such as scte35:SpliceInfoSection are child elements and a
	// chardata field would silently drop them on decode.
	InnerXML         string  `xml:",innerxml"`
	PresentationTime *uint64 `xml:"presentationTime,attr"`
	Duration         *uint64 `xml:"duration,attr"`
	ID               *uint64 `xml:"id,attr"`
	MessageData      *string `xml:"messageData,attr"`
}

type EventStream struct {
	// Namespaces holds the xmlns declarations carried by this element, which
	// may be the ones an Event payload below it relies on. See MPD.Namespaces.
	Namespaces []xml.Attr `xml:"-"`

	Events      []*Event `xml:"Event,omitempty"`
	Href        *string  `xml:"href,attr"`
	Actuate     *string  `xml:"actuate,attr"`
	SchemeIDURI *string  `xml:"schemeIdUri,attr"`
	Value       *string  `xml:"value,attr"`
	Timescale   *uint64  `xml:"timescale,attr"`
	MessageData *string  `xml:"messageData,attr"`
}

type Subset struct {
	Contains []int64 `xml:"contains,attr"`
	ID       *string `xml:"id,attr"`
}

// Period represents XSD's PeriodType.
type Period struct {
	// Namespaces holds the xmlns declarations carried by this element, which
	// may be the ones an Event payload below it relies on. See MPD.Namespaces.
	Namespaces []xml.Attr `xml:"-"`

	ID                 *string `xml:"id,attr"`
	Start              *string `xml:"start,attr"`
	Duration           *string `xml:"duration,attr"`
	Href               *string `xml:"href,attr"`
	Actuate            *string `xml:"actuate,attr"`
	BitStreamSwitching *bool   `xml:"bitstreamSwitching,attr"`
	Label              *string `xml:"label,attr"`
	BitmovinCustomXml  *string `xml:"bitmovinCustomXml,attr"`

	AdaptationSets         []*AdaptationSet `xml:"AdaptationSet,omitempty"`
	BaseURL                []*BaseURL       `xml:"BaseURL,omitempty"`
	SegmentBase            *SegmentBase     `xml:"SegmentBase,omitempty"`
	SegmentList            *SegmentList     `xml:"SegmentList,omitempty"`
	SegmentTemplate        *SegmentTemplate `xml:"SegmentTemplate,omitempty"`
	AssetIdentifier        *Descriptor      `xml:"AssetIdentifier,omitempty"`
	EventStreams           []*EventStream   `xml:"EventStream,omitempty"`
	Subsets                []*Subset        `xml:"Subset,omitempty"`
	SupplementalProperties []*Descriptor    `xml:"SupplementalProperty,omitempty"`
}

type Metrics struct {
	Reportings []*Descriptor `xml:"Reporting,omitempty"`
	Ranges     []*Range      `xml:"Range,omitempty"`
	Metrics    string        `xml:"metrics,attr"`
}

type Range struct {
	StartTime *string `xml:"starttime,attr"`
	Duration  *string `xml:"duration,attr"`
}

// BaseURL represents XSD's BaseURLType.
type BaseURL struct {
	Value                    string  `xml:",chardata"`
	ServiceLocation          *string `xml:"serviceLocation,attr"`
	ByteRange                *string `xml:"byteRange,attr"`
	AvailabilityTimeOffset   *uint64 `xml:"availabilityTimeOffset,attr"`
	AvailabilityTimeComplete *bool   `xml:"availabilityTimeComplete,attr"`
}

type Label struct {
	Value string  `xml:",chardata"`
	ID    *int64  `xml:"id,attr,omitempty"`
	Lang  *string `xml:"lang,attr,omitempty"`
}

// ContentComponent represents XSD's ContentComponentType.
type ContentComponent struct {
	Accessibilities []*Descriptor `xml:"Accessibility,omitempty"`
	Roles           []*Descriptor `xml:"Role,omitempty"`
	Ratings         []*Descriptor `xml:"Rating,omitempty"`
	Viewpoints      []*Descriptor `xml:"Viewpoint,omitempty"`
	ID              *int64        `xml:"id,attr,omitempty"`
	Lang            *string       `xml:"lang,attr,omitempty"`
	ContentType     *string       `xml:"contentType,attr,omitempty"`
	Par             *string       `xml:"par,attr,omitempty"`
}

// AdaptationSet represents XSD's AdaptationSetType.
type AdaptationSet struct {
	ID               *string `xml:"id,attr,omitempty"`
	ContentType      *string `xml:"contentType,attr,omitempty"`
	MimeType         *string `xml:"mimeType,attr,omitempty"`
	SegmentAlignment *string `xml:"segmentAlignment,attr,omitempty"`

	Href         *string `xml:"href,attr,omitempty"`
	Actuate      *string `xml:"actuate,attr,omitempty"`
	Group        *int64  `xml:"group,attr,omitempty"`
	Lang         *string `xml:"lang,attr,omitempty"`
	Par          *string `xml:"par,attr,omitempty"`
	MinBandwidth *int64  `xml:"minBandwidth,attr,omitempty"`
	MaxBandwidth *int64  `xml:"maxBandwidth,attr,omitempty"`
	MinWidth     *int64  `xml:"minWidth,attr,omitempty"`
	MaxWidth     *int64  `xml:"maxWidth,attr,omitempty"`
	MinHeight    *int64  `xml:"minHeight,attr,omitempty"`
	MaxHeight    *int64  `xml:"maxHeight,attr,omitempty"`
	MinFrameRate *string `xml:"minFrameRate,attr,omitempty"`
	MaxFrameRate *string `xml:"maxFrameRate,attr,omitempty"`

	Labels                     []*Label             `xml:"Label,omitempty"`
	FramePackings              []*Descriptor        `xml:"FramePacking,omitempty"`
	AudioChannelConfigurations []*Descriptor        `xml:"AudioChannelConfiguration,omitempty"`
	ContentProtections         []*ContentProtection `xml:"ContentProtection,omitempty"`
	EssentialProperties        []*Descriptor        `xml:"EssentialProperty,omitempty"`
	SupplementalProperties     []*Descriptor        `xml:"SupplementalProperty,omitempty"`
	InbandEventStreams         []*EventStream       `xml:"InbandEventStream,omitempty"`

	Accessibilities   []*Descriptor       `xml:"Accessibility,omitempty"`
	Roles             []*Descriptor       `xml:"Role,omitempty"`
	Ratings           []*Descriptor       `xml:"Rating,omitempty"`
	Viewpoints        []*Descriptor       `xml:"Viewpoint,omitempty"`
	ContentComponents []*ContentComponent `xml:"ContentComponent,omitempty"`
	BaseURLs          []*BaseURL          `xml:"BaseURL,omitempty"`
	SegmentBase       *SegmentBase        `xml:"SegmentBase,omitempty"`
	SegmentList       *SegmentList        `xml:"SegmentList,omitempty"`
	SegmentTemplate   *SegmentTemplate    `xml:"SegmentTemplate,omitempty"`
	Representations   []*Representation   `xml:"Representation,omitempty"`

	SubsegmentAlignment     *string `xml:"subsegmentAlignment,attr,omitempty"`
	SubsegmentStartsWithSAP *int64  `xml:"subsegmentStartsWithSAP,attr,omitempty"`
	BitstreamSwitching      *bool   `xml:"bitstreamSwitching,attr,omitempty"`

	Profiles *string `xml:"profiles,attr"`
	Width    *uint64 `xml:"width,attr"`
	Height   *uint64 `xml:"height,attr"`
	// Sample Aspect Ratio: aspect ratio of one pixel (typically 1:1)
	SAR               *string  `xml:"sar,attr"`
	FrameRate         *string  `xml:"frameRate,attr"`
	AudioSamplingRate *string  `xml:"audioSamplingRate,attr"`
	SegmentProfiles   *string  `xml:"segmentProfiles,attr"`
	Codecs            *string  `xml:"codecs,attr"`
	MaximumSapPeriod  *float64 `xml:"maximumSAPPeriod,attr"`
	StartWithSap      *uint64  `xml:"startWithSAP,attr"`
	MaxPlayoutRate    *float64 `xml:"maxPlayoutRate,attr"`
	CodingDependency  *bool    `xml:"codingDependency,attr"`
	ScanType          *string  `xml:"scanType,attr"`
}

type UtcTiming struct {
	ID          *string `xml:"id,attr"`
	Value       *string `xml:"value,attr"`
	SchemeIDURI *string `xml:"schemeIdUri,attr"`
}

type SubRepresentation struct {
	ID               *string `xml:"id,attr,omitempty"`
	Level            *int64  `xml:"level,attr,omitempty"`
	DependencyLevel  *string `xml:"dependencyLevel,attr,omitempty"`
	Bandwidth        *int64  `xml:"bandwidth,attr,omitempty"`
	ContentComponent *string `xml:"contentComponent,attr,omitempty"`
}

// Representation represents XSD's RepresentationType.
type Representation struct {
	ID                         *string              `xml:"id,attr"`
	Bandwidth                  *uint64              `xml:"bandwidth,attr"`
	Width                      *uint64              `xml:"width,attr"`
	Height                     *uint64              `xml:"height,attr"`
	Codecs                     *string              `xml:"codecs,attr"`
	FramePackings              []*Descriptor        `xml:"framePacking,omitempty"`
	AudioChannelConfigurations []*Descriptor        `xml:"AudioChannelConfiguration,omitempty"`
	InbandEventStreams         []*EventStream       `xml:"InbandEventStream,omitempty"`
	BaseURL                    []*BaseURL           `xml:"BaseURL,omitempty"`
	SubRepresentations         []*SubRepresentation `xml:"SubRepresentation,omitempty"`
	SegmentBase                *SegmentBase         `xml:"SegmentBase,omitempty"`
	EssentialProperties        []*Descriptor        `xml:"EssentialProperty,omitempty"`
	SupplementalProperties     []*Descriptor        `xml:"SupplementalProperty,omitempty"`
	SegmentList                *SegmentList         `xml:"SegmentList,omitempty"`
	ContentProtections         []*ContentProtection `xml:"ContentProtection,omitempty"`
	SegmentTemplate            *SegmentTemplate     `xml:"SegmentTemplate,omitempty"`

	FrameRate              *string  `xml:"frameRate,attr"`
	AudioSamplingRate      *string  `xml:"audioSamplingRate,attr"`
	SAR                    *string  `xml:"sar,attr"`
	ScanType               *string  `xml:"scanType,attr"`
	QualityRanking         *uint64  `xml:"qualityRanking,attr"`
	DependencyId           *string  `xml:"dependencyId,attr"`
	MediaStreamStructureId *string  `xml:"mediaStreamStructureId,attr"`
	Profiles               *string  `xml:"profiles,attr"`
	MimeType               *string  `xml:"mimeType,attr"`
	SegmentProfiles        *string  `xml:"segmentProfiles,attr"`
	MaximumSapPeriod       *float64 `xml:"maximumSAPPeriod,attr"`
	StartWithSap           *uint64  `xml:"startWithSAP,attr"`
	MaxPlayoutRate         *float64 `xml:"maxPlayoutRate,attr"`
	CodingDependency       *bool    `xml:"codingDependency,attr"`
}

type ContentProtection struct {
	// Namespaces and ForeignAttrs carry xmlns:cenc and cenc:default_KID. See
	// MPD.Namespaces. InnerXML keeps cenc:pssh and mspr:pro children verbatim,
	// as Event.InnerXML does for SCTE-35 payloads.
	Namespaces   []xml.Attr `xml:"-"`
	ForeignAttrs []xml.Attr `xml:"-"`
	InnerXML     string     `xml:",innerxml"`

	SchemeIDURI *string `xml:"schemeIdUri,attr"`
	Value       *string `xml:"value,attr"`
	DefaultKID  *string `xml:"default_KID,attr"`
}

// Descriptor represents XSD's DescriptorType.
type Descriptor struct {
	SchemeIDURI *string `xml:"schemeIdUri,attr"`
	Value       *string `xml:"value,attr"`
}

// SegmentTemplate represents XSD's SegmentTemplateType.
type SegmentTemplate struct {
	Duration               *uint64          `xml:"duration,attr"`
	Timescale              *uint64          `xml:"timescale,attr"`
	Media                  *string          `xml:"media,attr"`
	Initialization         *string          `xml:"initialization,attr"`
	StartNumber            *uint64          `xml:"startNumber,attr"`
	PresentationTimeOffset *uint64          `xml:"presentationTimeOffset,attr"`
	SegmentTimeline        *SegmentTimeline `xml:"SegmentTimeline,omitempty"`
}

// SegmentTimeline represents XSD's SegmentTimelineType.
type SegmentTimeline struct {
	S []*SegmentTimelineS `xml:"S"`
}

// SegmentTimelineS represents XSD's SegmentTimelineType's inner S elements.
type SegmentTimelineS struct {
	T *uint64 `xml:"t,attr"`
	D uint64  `xml:"d,attr"`
	R *int64  `xml:"r,attr"`
}
