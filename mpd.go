// Package mpd implements parsing and generating of MPEG-DASH Media Presentation Description (MPD) files.
package mpd

import (
	"bytes"
	"encoding/xml"
	"github.com/unki2aut/go-xsd-types"
)

// http://mpeg.chiariglione.org/standards/mpeg-dash
// https://www.brendanlong.com/the-structure-of-an-mpeg-dash-mpd.html
// http://standards.iso.org/ittf/PubliclyAvailableStandards/MPEG-DASH_schema_files/DASH-MPD.xsd

// MPD represents root XML element.
type MPD struct {
	XMLNS                      *string               `xml:"xmlns,attr"`
	XMLNSXSI                   *string               `xml:"xmlns:xsi,attr"`
	Ns2                        *string               `xml:"ns2,attr"`
	Xsi                        *string               `xml:"xsi,attr"`
	XsiSchemaLocation          *string               `xml:"xsi:schemaLocation,attr"`
	SchemaLocation             *string               `xml:"schemaLocation,attr"`
	Type                       *string               `xml:"type,attr"`
	MinimumUpdatePeriod        *xsd.Duration         `xml:"minimumUpdatePeriod,attr"`
	AvailabilityStartTime      *xsd.DateTime         `xml:"availabilityStartTime,attr"`
	AvailabilityEndTime        *xsd.DateTime         `xml:"availabilityEndTime,attr"`
	MediaPresentationDuration  *xsd.Duration         `xml:"mediaPresentationDuration,attr"`
	MinBufferTime              *xsd.Duration         `xml:"minBufferTime,attr"`
	SuggestedPresentationDelay *xsd.Duration         `xml:"suggestedPresentationDelay,attr"`
	TimeShiftBufferDepth       *xsd.Duration         `xml:"timeShiftBufferDepth,attr"`
	PublishTime                *xsd.DateTime         `xml:"publishTime,attr"`
	Profiles                   *string               `xml:"profiles,attr"`
	Id                         *string               `xml:"id,attr"`
	MaxSegmentDuration         *xsd.Duration         `xml:"maxSegmentDuration,attr"`
	MaxSubsegmentDuration      *xsd.Duration         `xml:"maxSubsegmentDuration,attr"`
	BaseURL                    []*BaseURL            `xml:"BaseURL,omitempty"`
	Period                     []*Period             `xml:"Period,omitempty"`
	ProgramInformations        []*ProgramInformation `xml:"ProgramInformation,omitempty"`
	Locations                  []*string             `xml:"Location,omitempty"`
	Metrics                    []*Metrics            `xml:"Metrics,omitempty"`
	EssentialProperties        []*Descriptor         `xml:"EssentialProperty,omitempty"`
	SupplementalProperties     []*Descriptor         `xml:"SupplementalProperty,omitempty"`
	UtcTimings                 []*UtcTiming          `xml:"UTCTiming,omitempty"`
	XmlnsCenc                  *string               `xml:"xmlns:cenc,attr"`
	Cenc                       *string               `xml:"cenc,attr"`
	Mspr                       *string               `xml:"mspr,attr"`
	XmlnsMspr                  *string               `xml:"xmlns:mspr,attr"`
	Mas                        *string               `xml:"mas,attr"`
	XmlnsMas                   *string               `xml:"xmlns:mas,attr"`
}

// Do not try to use encoding.TextMarshaler and encoding.TextUnmarshaler:
// https://github.com/golang/go/issues/6859#issuecomment-118890463

// Encode generates MPD XML.
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
	Value            string  `xml:",chardata"`
	PresentationTime *uint64 `xml:"presentationTime,attr"`
	Duration         *uint64 `xml:"duration,attr"`
	ID               *uint64 `xml:"id,attr"`
	MessageData      *string `xml:"messageData,attr"`
}

type EventStream struct {
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
	ID                 *string       `xml:"id,attr"`
	Start              *xsd.Duration `xml:"start,attr"`
	Duration           *xsd.Duration `xml:"duration,attr"`
	Href               *string       `xml:"href,attr"`
	Actuate            *string       `xml:"actuate,attr"`
	BitStreamSwitching *bool         `xml:"bitstreamSwitching,attr"`
	Label              *string       `xml:"label,attr"`
	BitmovinCustomXml  *string       `xml:"bitmovinCustomXml,attr"`

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
	StartTime *xsd.Duration `xml:"starttime,attr"`
	Duration  *xsd.Duration `xml:"duration,attr"`
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
	ID    *int64  `xml:"id,attr"`
	Lang  *string `xml:"lang,attr"`
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

	Labels                     []*Label             `xml:"label,omitempty"`
	FramePackings              []*Descriptor        `xml:"framePacking,omitempty"`
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
	SchemeIDURI         *string             `xml:"schemeIdUri,attr"`
	Value               *string             `xml:"value,attr"`
	Cenc                *string             `xml:"cenc,attr"`
	CencPSSH            *string             `xml:"cenc:pssh,attr"`
	CencDefaultKID      *string             `xml:"cenc:default_KID,attr"`
	DefaultKID          *string             `xml:"default_KID,attr"`
	CencPsshBody        *string             `xml:"cenc:pssh,omitempty"`
	PsshBody            *Pssh               `xml:"pssh,omitempty"`
	Pro                 *Pro                `xml:"pro,omitempty"`
	MsprPro             *string             `xml:"mspr:pro,omitempty"`
	MarlinContentIds    []*MarlinContentIds `xml:"MarlinContentIds,omitempty"`
	MasMarlinContentIds []*MarlinContentIds `xml:"mas:MarlinContentIds,omitempty"`
}

type MarlinContentIds struct {
	MarlinContentId    *MarlinContentId `xml:"MarlinContentId,omitempty"`
	MasMarlinContentId *MarlinContentId `xml:"mas:MarlinContentId,omitempty"`
}

type MarlinContentId struct {
	Value string `xml:",chardata"`
}

type Pssh struct {
	Value string  `xml:",chardata"`
	Cenc  *string `xml:"cenc,attr"`
}

type Pro struct {
	Value string  `xml:",chardata"`
	Mspr  *string `xml:"mspr,attr"`
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
