// Package contract contains typed shapes for durable evaluation inspection fixtures.
package contract

// DatasetInspectionRevision is the immutable dataset revision identity returned by inspection routes.
type DatasetInspectionRevision struct {
	ID                            string   `json:"id"`
	CorpusID                      string   `json:"corpusId"`
	DatasetKey                    string   `json:"datasetKey"`
	SemanticRevision              string   `json:"semanticRevision"`
	Jurisdiction                  string   `json:"jurisdiction"`
	ManifestSHA256                string   `json:"manifestSha256"`
	JSONLSHA256                   string   `json:"jsonlSha256"`
	ContentSHA256                 string   `json:"contentSha256"`
	DeclaredSnapshotDate          string   `json:"declaredSnapshotDate"`
	QueryLanguages                []string `json:"queryLanguages"`
	AuthoritativeEvidenceLanguage string   `json:"authoritativeEvidenceLanguage"`
}

// DatasetInspectionReview is the latest immutable review state exposed by inspection routes.
type DatasetInspectionReview struct {
	Decision         string `json:"decision"`
	PublicationState string `json:"publicationState"`
	ReviewedAt       string `json:"reviewedAt"`
}

// DatasetCatalogResponse is the catalog projection for one dataset revision.
type DatasetCatalogResponse struct {
	Revision  DatasetInspectionRevision `json:"revision"`
	Available bool                      `json:"available"`
	Review    *DatasetInspectionReview  `json:"review"`
}

// DatasetSourceResponse is the safe manifest authority and corpus-binding projection.
type DatasetSourceResponse struct {
	ID               string  `json:"id"`
	SourceAlias      string  `json:"sourceAlias"`
	Title            string  `json:"title"`
	OfficialURL      string  `json:"officialUrl"`
	IssuingAuthority string  `json:"issuingAuthority"`
	DocumentType     string  `json:"documentType"`
	AuthorityRole    string  `json:"authorityRole"`
	CorpusSourceID   *string `json:"corpusSourceId"`
	Bound            bool    `json:"bound"`
}

// DatasetStarterResponse is safe starter-case metadata without case content.
type DatasetStarterResponse struct {
	ID             string `json:"id"`
	CaseID         string `json:"caseId"`
	Rank           int    `json:"rank"`
	QueryLanguage  string `json:"queryLanguage"`
	ReviewEligible bool   `json:"reviewEligible"`
}

// DatasetDetailResponse extends the catalog identity with maintainer-only detail metadata.
type DatasetDetailResponse struct {
	DatasetCatalogResponse
	Sources  []DatasetSourceResponse  `json:"sources"`
	Starters []DatasetStarterResponse `json:"starters"`
}

// DatasetMissingRequirement is one safe, bounded incompatibility diagnostic.
type DatasetMissingRequirement struct {
	SourceAlias string `json:"sourceAlias"`
	Locator     string `json:"locator"`
	Reason      string `json:"reason"`
}

// DatasetPreflightResponse confirms compatibility for immutable selected identities.
type DatasetPreflightResponse struct {
	DatasetRevisionID   string                      `json:"datasetRevisionId"`
	CorpusID            string                      `json:"corpusId"`
	SnapshotID          string                      `json:"snapshotId"`
	Compatible          bool                        `json:"compatible"`
	MissingRequirements []DatasetMissingRequirement `json:"missingRequirements"`
}

// DatasetPreflightErrorResponse is the public error envelope for a rejected preflight.
type DatasetPreflightErrorResponse struct {
	Error struct {
		Code                string                      `json:"code"`
		Message             string                      `json:"message"`
		RequestID           string                      `json:"requestId"`
		MissingRequirements []DatasetMissingRequirement `json:"missingRequirements"`
	} `json:"error"`
}
