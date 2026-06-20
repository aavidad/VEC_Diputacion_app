package domain

import (
	"errors"
	"strings"
)

var (
	ErrRPTPositionInvalid          = errors.New("personal rpt position invalid")
	ErrProfessionalCategoryInvalid = errors.New("personal professional category invalid")
	ErrCatalogEntryInvalid         = errors.New("personal catalog entry invalid")
)

type RPTPosition struct {
	Code               string `json:"code"`
	Name               string `json:"name"`
	Dot                int    `json:"dot"`
	Type               string `json:"type"`
	Administration     string `json:"administration"`
	Provision          string `json:"provision"`
	Group              string `json:"group"`
	Area               string `json:"area"`
	Scale              string `json:"scale"`
	CategoryCode       string `json:"category_code"`
	CategorySlug       string `json:"category_slug"`
	Delegation         string `json:"delegation,omitempty"`
	CenterCode         string `json:"center_code,omitempty"`
	CenterName         string `json:"center_name,omitempty"`
	DestinationLevel   int    `json:"destination_level"`
	SpecificKind       string `json:"specific_kind,omitempty"`
	AnnualAmountCents  int64  `json:"annual_amount_cents,omitempty"`
	GCPCTLevel         string `json:"gcp_ct_level,omitempty"`
	SpecificComplement string `json:"specific_complement"`
	GeoDispersion      string `json:"geo_dispersion"`
	Telework           string `json:"telework"`
	Coverage           string `json:"coverage"`
	State              string `json:"state"`
	Source             string `json:"source"`
	Requirements       string `json:"requirements,omitempty"`
	Observations       string `json:"observations"`
	Page               int    `json:"page,omitempty"`
	Raw                string `json:"raw,omitempty"`
}

func (p RPTPosition) Validate() error {
	if strings.TrimSpace(p.Code) == "" || strings.TrimSpace(p.Name) == "" {
		return ErrRPTPositionInvalid
	}
	if p.Dot < 0 || p.DestinationLevel < 0 || p.AnnualAmountCents < 0 {
		return ErrRPTPositionInvalid
	}
	return nil
}

type ProfessionalCategory struct {
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	Area       string `json:"area"`
	Source     string `json:"source"`
	SourcePath string `json:"source_path"`
	ModuleKey  string `json:"module_key"`
	State      string `json:"state"`
	Usage      string `json:"usage"`
}

func (c ProfessionalCategory) Validate() error {
	if strings.TrimSpace(c.Slug) == "" || strings.TrimSpace(c.Name) == "" {
		return ErrProfessionalCategoryInvalid
	}
	return nil
}

type CatalogEntry struct {
	Catalog   string `json:"catalog"`
	Code      string `json:"code"`
	Label     string `json:"label"`
	Source    string `json:"source"`
	ModuleKey string `json:"module_key"`
	State     string `json:"state"`
	Usage     string `json:"usage"`
}

func (e CatalogEntry) Validate() error {
	if strings.TrimSpace(e.Catalog) == "" || strings.TrimSpace(e.Code) == "" || strings.TrimSpace(e.Label) == "" {
		return ErrCatalogEntryInvalid
	}
	return nil
}

type CategoryAlias struct {
	Alias        string `json:"alias"`
	CategorySlug string `json:"category_slug"`
	Source       string `json:"source"`
}

type CategoryRule struct {
	ID             string  `json:"id"`
	Label          string  `json:"label"`
	Section        string  `json:"section"`
	PointsPerMonth float64 `json:"points_per_month"`
	Source         string  `json:"source"`
}

type RPTPositionFilter struct {
	Query      string `json:"query,omitempty"`
	Group      string `json:"group,omitempty"`
	CenterCode string `json:"center_code,omitempty"`
	Provision  string `json:"provision,omitempty"`
	State      string `json:"state,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
}

type RPTPositionPage struct {
	Items  []RPTPosition `json:"items"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

type ProfessionalCategoryFilter struct {
	Query  string `json:"query,omitempty"`
	Area   string `json:"area,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type ProfessionalCategoryPage struct {
	Items  []ProfessionalCategory `json:"items"`
	Total  int                    `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}

type RPTImportCommand struct {
	Source    string        `json:"source"`
	Version   string        `json:"version"`
	Replace   bool          `json:"replace"`
	Positions []RPTPosition `json:"positions"`
}

type RPTImportReceipt struct {
	Source   string `json:"source"`
	Version  string `json:"version"`
	Imported int    `json:"imported"`
	Replaced bool   `json:"replaced"`
}

type CatalogStats struct {
	Positions        int            `json:"positions"`
	Categories       int            `json:"categories"`
	CatalogEntries   int            `json:"catalog_entries"`
	PositionsByGroup map[string]int `json:"positions_by_group"`
	CategoriesByArea map[string]int `json:"categories_by_area"`
	PendingLegend    int            `json:"pending_legend"`
}
