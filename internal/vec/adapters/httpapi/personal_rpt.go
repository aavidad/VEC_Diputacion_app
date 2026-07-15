package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	personalmodule "vec-diputacion-granada/internal/modules/personal"
	personalfile "vec-diputacion-granada/internal/modules/personal/adapters/file"
	personalmemory "vec-diputacion-granada/internal/modules/personal/adapters/memory"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
)

func newWorkspacePersonalCatalogService(catalogPath string) (*personalapp.CatalogService, error) {
	var store personalports.CatalogStore = personalmemory.NewCatalogStore()
	if strings.TrimSpace(catalogPath) != "" {
		durable, err := personalfile.NewCatalogStore(catalogPath)
		if err != nil {
			return nil, err
		}
		store = durable
	}
	return personalapp.NewCatalogService(store)
}

func (h *Handler) handlePersonalRPTPositions(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	if !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	if !h.requirePermission(w, principal, personalmodule.PermissionPositionRead) {
		return
	}
	page, err := h.personalCatalog.ListPositions(r.Context(), personalPositionFilterFromRequest(r))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"positions": page})
}

func (h *Handler) handlePersonalRPTPosition(w http.ResponseWriter, r *http.Request, principal domain.Principal, path string) {
	code := strings.TrimSpace(strings.TrimPrefix(path, "/personal/rpt/positions/"))
	if code == "" {
		h.writeError(w, http.StatusNotFound, "personal rpt position not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		if !h.requirePermission(w, principal, personalmodule.PermissionPositionRead) {
			return
		}
		position, err := h.personalCatalog.GetPosition(r.Context(), code)
		if err != nil {
			if errors.Is(err, personalapp.ErrRPTPositionNotFound) {
				if fallback, ok := h.findPersonalRPTPositionByOfficialCode(r.Context(), code); ok {
					h.writeJSON(w, http.StatusOK, map[string]any{"position": fallback})
					return
				}
			}
			status := http.StatusBadRequest
			if errors.Is(err, personalapp.ErrRPTPositionNotFound) {
				status = http.StatusNotFound
			}
			h.writeError(w, status, err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{"position": position})
	case http.MethodPut:
		if !h.requirePermission(w, principal, personalmodule.PermissionPositionManage) {
			return
		}
		var position personaldomain.RPTPosition
		if err := json.NewDecoder(r.Body).Decode(&position); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid rpt position json")
			return
		}
		position.Code = code
		stored, err := h.personalCatalog.UpsertPosition(r.Context(), position)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		receipt, err := h.recordPersonalCatalogAudit(r.Context(), principal, "personal.rpt.position.upsert", code)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{"position": stored, "receipt": receipt})
	case http.MethodDelete:
		if !h.requirePermission(w, principal, personalmodule.PermissionPositionManage) {
			return
		}
		deleted, err := h.personalCatalog.DeletePosition(r.Context(), code)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if !deleted {
			h.writeError(w, http.StatusNotFound, personalapp.ErrRPTPositionNotFound.Error())
			return
		}
		receipt, err := h.recordPersonalCatalogAudit(r.Context(), principal, "personal.rpt.position.delete", code)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "code": code, "receipt": receipt})
	default:
		w.Header().Set("Allow", "GET, PUT, DELETE")
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) findPersonalRPTPositionByOfficialCode(ctx context.Context, code string) (personaldomain.RPTPosition, bool) {
	page, err := h.personalCatalog.ListPositions(ctx, personaldomain.RPTPositionFilter{Query: code, Limit: 2000})
	if err != nil {
		return personaldomain.RPTPosition{}, false
	}
	for _, position := range page.Items {
		if strings.EqualFold(position.Code, code) || strings.EqualFold(rptOfficialCode(position), code) {
			return position, true
		}
	}
	return personaldomain.RPTPosition{}, false
}

func rptOfficialCode(position personaldomain.RPTPosition) string {
	for _, part := range strings.Split(position.Observations, "|") {
		label, value, ok := strings.Cut(strings.TrimSpace(part), ":")
		if ok && strings.EqualFold(strings.TrimSpace(label), "Codigo RPT oficial") {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (h *Handler) handlePersonalRPTImports(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	if !h.requireMethod(w, r, http.MethodPost) {
		return
	}
	if !h.requirePermission(w, principal, personalmodule.PermissionPositionManage) {
		return
	}
	var cmd personaldomain.RPTImportCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid rpt import json")
		return
	}
	receipt, err := h.personalCatalog.ImportPositions(r.Context(), cmd)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	audit, err := h.recordPersonalCatalogAudit(r.Context(), principal, "personal.rpt.import", receipt.Source)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.writeJSON(w, http.StatusCreated, map[string]any{"import": receipt, "receipt": audit})
}

func (h *Handler) handlePersonalRPTStats(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	if !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	if !h.requirePermission(w, principal, personalmodule.PermissionPositionRead) {
		return
	}
	stats, err := h.personalCatalog.Stats(r.Context())
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"stats": stats})
}

func (h *Handler) handlePersonalCategories(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	switch r.Method {
	case http.MethodGet:
		if !h.requirePermission(w, principal, personalmodule.PermissionPositionRead) {
			return
		}
		page, err := h.personalCatalog.ListCategories(r.Context(), personalCategoryFilterFromRequest(r))
		if err != nil {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{"categories": page})
	case http.MethodPost:
		if !h.requirePermission(w, principal, personalmodule.PermissionPositionManage) {
			return
		}
		var category personaldomain.ProfessionalCategory
		if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid professional category json")
			return
		}
		if err := h.personalCatalog.UpsertCategory(r.Context(), category); err != nil {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		stored, err := h.personalCatalog.GetCategory(r.Context(), category.Slug)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		receipt, err := h.recordPersonalCatalogAudit(r.Context(), principal, "personal.category.upsert", stored.Slug)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.writeJSON(w, http.StatusCreated, map[string]any{"category": stored, "receipt": receipt})
	default:
		w.Header().Set("Allow", "GET, POST")
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handlePersonalCategory(w http.ResponseWriter, r *http.Request, principal domain.Principal, path string) {
	slug := strings.TrimSpace(strings.TrimPrefix(path, "/personal/categories/"))
	if slug == "" {
		h.writeError(w, http.StatusNotFound, personalapp.ErrCategoryNotFound.Error())
		return
	}
	switch r.Method {
	case http.MethodGet:
		if !h.requirePermission(w, principal, personalmodule.PermissionPositionRead) {
			return
		}
		category, err := h.personalCatalog.GetCategory(r.Context(), slug)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, personalapp.ErrCategoryNotFound) {
				status = http.StatusNotFound
			}
			h.writeError(w, status, err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{"category": category})
	case http.MethodPut:
		if !h.requirePermission(w, principal, personalmodule.PermissionPositionManage) {
			return
		}
		var category personaldomain.ProfessionalCategory
		if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid professional category json")
			return
		}
		category.Slug = slug
		if err := h.personalCatalog.UpsertCategory(r.Context(), category); err != nil {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		stored, err := h.personalCatalog.GetCategory(r.Context(), slug)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		receipt, err := h.recordPersonalCatalogAudit(r.Context(), principal, "personal.category.upsert", slug)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{"category": stored, "receipt": receipt})
	case http.MethodDelete:
		if !h.requirePermission(w, principal, personalmodule.PermissionPositionManage) {
			return
		}
		deleted, err := h.personalCatalog.DeleteCategory(r.Context(), slug)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if !deleted {
			h.writeError(w, http.StatusNotFound, personalapp.ErrCategoryNotFound.Error())
			return
		}
		receipt, err := h.recordPersonalCatalogAudit(r.Context(), principal, "personal.category.delete", slug)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "slug": slug, "receipt": receipt})
	default:
		w.Header().Set("Allow", "GET, PUT, DELETE")
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handlePersonalCatalogs(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	if !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	if !h.requirePermission(w, principal, personalmodule.PermissionPositionRead) {
		return
	}
	entries, err := h.personalCatalog.ListCatalogEntries(r.Context())
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"catalogs": entries})
}

func (h *Handler) requirePermission(w http.ResponseWriter, principal domain.Principal, permission string) bool {
	if principal.HasPermission(permission) {
		return true
	}
	h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
	return false
}

func (h *Handler) recordPersonalCatalogAudit(ctx context.Context, principal domain.Principal, action, subjectRef string) (domain.AuditEntry, error) {
	if h.internal == nil {
		return domain.AuditEntry{}, domain.ErrPermissionDenied
	}
	authorized, err := application.NewAuthorizedAuditCommand(application.AuditCommand{
		Principal:  principal,
		Action:     action,
		ModuleID:   personalmodule.ModuleID,
		SubjectRef: subjectRef,
		Result:     "accepted",
		Metadata: map[string]string{
			"receipt_type": "personal.catalog",
			"source":       "httpapi",
			"at":           time.Now().UTC().Format(time.RFC3339),
		},
	}, personalmodule.PermissionPositionManage, "")
	if err != nil {
		return domain.AuditEntry{}, err
	}
	receipt, err := h.internal.RecordAudit(ctx, authorized)
	if err != nil {
		return domain.AuditEntry{}, err
	}
	return receipt.Entry(), nil
}

func personalPositionFilterFromRequest(r *http.Request) personaldomain.RPTPositionFilter {
	query := r.URL.Query()
	return personaldomain.RPTPositionFilter{
		Query:      query.Get("q"),
		Group:      query.Get("group"),
		CenterCode: query.Get("center_code"),
		Provision:  query.Get("provision"),
		State:      query.Get("state"),
		Limit:      intQuery(query.Get("limit")),
		Offset:     intQuery(query.Get("offset")),
	}
}

func personalCategoryFilterFromRequest(r *http.Request) personaldomain.ProfessionalCategoryFilter {
	query := r.URL.Query()
	return personaldomain.ProfessionalCategoryFilter{
		Query:  query.Get("q"),
		Area:   query.Get("area"),
		Limit:  intQuery(query.Get("limit")),
		Offset: intQuery(query.Get("offset")),
	}
}

func intQuery(value string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(value))
	return n
}

func rptPositionFromMap(raw map[string]any) personaldomain.RPTPosition {
	return personaldomain.RPTPosition{
		Code:               asString(raw["code"]),
		Name:               asString(raw["name"]),
		Dot:                1,
		Type:               asString(raw["tp"]),
		Administration:     asString(raw["ad"]),
		Provision:          asString(raw["fp"]),
		Group:              asString(raw["group"]),
		Area:               asString(raw["area"]),
		Scale:              asString(raw["scale"]),
		CategoryCode:       asString(raw["category"]),
		DestinationLevel:   asInt(raw["cd"]),
		SpecificComplement: asString(raw["ce"]),
		GeoDispersion:      asString(raw["dg"]),
		Telework:           asString(raw["ta"]),
		Coverage:           asString(raw["coverage"]),
		State:              asString(raw["state"]),
		Source:             "workspace_seed_nominas_rpt",
	}
}

func professionalCategoryFromMap(raw map[string]any) personaldomain.ProfessionalCategory {
	return personaldomain.ProfessionalCategory{
		Slug:       asString(raw["slug"]),
		Name:       asString(raw["name"]),
		Area:       asString(raw["area"]),
		Source:     asString(raw["source"]),
		SourcePath: asString(raw["source_path"]),
		ModuleKey:  asString(raw["module_key"]),
		State:      asString(raw["state"]),
		Usage:      asString(raw["usage"]),
	}
}

func catalogEntryFromMap(raw map[string]any) personaldomain.CatalogEntry {
	return personaldomain.CatalogEntry{
		Catalog:   asString(raw["catalog"]),
		Code:      asString(raw["code"]),
		Label:     asString(raw["label"]),
		Source:    asString(raw["source"]),
		ModuleKey: asString(raw["module_key"]),
		State:     asString(raw["state"]),
		Usage:     asString(raw["usage"]),
	}
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func asInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(typed))
		return n
	default:
		return 0
	}
}
