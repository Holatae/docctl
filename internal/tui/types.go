package tui

type DocumentType string

const (
	DocumentTypeProtokoll    DocumentType = "protokoll"
	DocumentTypeStyrdokument DocumentType = "styrdokument"
	DocumentTypeOther        DocumentType = "other"
)

type SourceFormat string

const (
	SourceFormatMarkdown SourceFormat = "markdown"
	SourceFormatFODT     SourceFormat = "fodt"
)
