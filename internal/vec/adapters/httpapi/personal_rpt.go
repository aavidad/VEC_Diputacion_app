package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
	personalmodule "vec-diputacion-granada/internal/modules/personal"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
)

// CatalogoPersonal es el puerto minimo que necesita esta frontera HTTP. La
// raiz de composicion puede inyectar cualquier implementacion compatible sin
// que el adaptador elija memoria, fichero o una futura base de datos.
type CatalogoPersonal interface {
	ListPositions(context.Context, personaldomain.RPTPositionFilter) (personaldomain.RPTPositionPage, error)
	GetPosition(context.Context, string) (personaldomain.RPTPosition, error)
	UpsertPosition(context.Context, personaldomain.RPTPosition) (personaldomain.RPTPosition, error)
	DeletePosition(context.Context, string) (bool, error)
	ImportPositions(context.Context, personaldomain.RPTImportCommand) (personaldomain.RPTImportReceipt, error)
	Stats(context.Context) (personaldomain.CatalogStats, error)
	ListCatalogEntries(context.Context) ([]personaldomain.CatalogEntry, error)
}

type ConsultaCategoriasProfesionales interface {
	ListarVigentes(context.Context) (personalports.CatalogoCategoriasProfesionalesConsultable, error)
}

func (h *Handler) handlePersonalRPTPositions(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	if !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	if !h.requirePermission(w, principal, personalmodule.PermissionPositionRead) {
		return
	}
	if !h.requirePersonalCatalog(w) {
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
		if !h.requirePersonalCatalog(w) {
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
		if !h.requirePersonalCatalog(w) {
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
		if !h.requirePersonalCatalog(w) {
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
	if !h.requirePersonalCatalog(w) {
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
	w.Header().Set("Cache-Control", "no-store")
	if !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	if !h.requirePermission(w, principal, personalmodule.PermissionPositionRead) {
		return
	}
	if !h.requirePersonalCatalog(w) {
		return
	}
	stats, err := h.personalCatalog.Stats(r.Context())
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	catalogo, err := h.listarCategoriasProfesionales(r.Context())
	if err != nil {
		h.writeError(w, http.StatusServiceUnavailable, "catalogo_categorias_profesionales_no_disponible")
		return
	}
	stats.Categories = len(catalogo.Categorias)
	stats.CategoriesByArea = make(map[string]int)
	for _, categoria := range catalogo.Categorias {
		stats.CategoriesByArea[categoria.Area]++
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"stats": stats})
}

func (h *Handler) handlePersonalCategories(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	w.Header().Set("Cache-Control", "no-store")
	switch r.Method {
	case http.MethodGet:
		if !h.requirePermission(w, principal, personalmodule.PermissionPositionRead) {
			return
		}
		filtro, err := filtroCategoriasProfesionalesDesdePeticion(r)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "filtro_categorias_profesionales_invalido")
			return
		}
		catalogo, err := h.listarCategoriasProfesionales(r.Context())
		if err != nil {
			h.writeError(w, http.StatusServiceUnavailable, "catalogo_categorias_profesionales_no_disponible")
			return
		}
		page := paginarCategoriasProfesionales(catalogo, filtro)
		w.Header().Set("Cache-Control", "no-store")
		h.writeJSON(w, http.StatusOK, map[string]any{"categories": page})
	case http.MethodPost:
		if !h.requirePermission(w, principal, personalmodule.PermissionPositionManage) {
			return
		}
		h.writeCatalogoGobernadoConflict(w)
	default:
		w.Header().Set("Allow", "GET, POST")
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handlePersonalCategory(w http.ResponseWriter, r *http.Request, principal domain.Principal, path string) {
	w.Header().Set("Cache-Control", "no-store")
	slug := strings.TrimSpace(strings.TrimPrefix(path, "/personal/categories/"))
	if slug == "" {
		h.writeError(w, http.StatusNotFound, "categoria_profesional_no_encontrada")
		return
	}
	switch r.Method {
	case http.MethodGet:
		if !h.requirePermission(w, principal, personalmodule.PermissionPositionRead) {
			return
		}
		catalogo, err := h.listarCategoriasProfesionales(r.Context())
		if err != nil {
			h.writeError(w, http.StatusServiceUnavailable, "catalogo_categorias_profesionales_no_disponible")
			return
		}
		for _, categoria := range catalogo.Categorias {
			if categoria.Clave == slug {
				w.Header().Set("Cache-Control", "no-store")
				h.writeJSON(w, http.StatusOK, map[string]any{
					"category": proyectarCategoriaProfesionalHTTP(categoria, catalogo.Fuente.Demostracion),
					"catalogo": catalogo.Referencia,
					"fuente":   catalogo.Fuente,
				})
				return
			}
		}
		h.writeError(w, http.StatusNotFound, "categoria_profesional_no_encontrada")
	case http.MethodPut:
		if !h.requirePermission(w, principal, personalmodule.PermissionPositionManage) {
			return
		}
		h.writeCatalogoGobernadoConflict(w)
	case http.MethodDelete:
		if !h.requirePermission(w, principal, personalmodule.PermissionPositionManage) {
			return
		}
		h.writeCatalogoGobernadoConflict(w)
	default:
		w.Header().Set("Allow", "GET, PUT, DELETE")
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

const (
	limiteCategoriasProfesionalesPorDefecto = 100
	limiteMaximoCategoriasProfesionales     = 500
)

type filtroCategoriasProfesionalesHTTP struct {
	consulta string
	area     string
	limite   int
	desde    int
}

type categoriaProfesionalHTTP struct {
	Catalogo     string `json:"catalog"`
	Clave        string `json:"clave"`
	Slug         string `json:"slug"`
	Etiqueta     string `json:"etiqueta"`
	Nombre       string `json:"name"`
	Descripcion  string `json:"descripcion,omitempty"`
	Orden        int    `json:"orden"`
	Area         string `json:"area"`
	AreaEtiqueta string `json:"area_etiqueta"`
	Fuente       string `json:"source"`
	Modulo       string `json:"module_key"`
	Estado       string `json:"state"`
	Uso          string `json:"usage"`
}

type paginaCategoriasProfesionalesHTTP struct {
	Items    []categoriaProfesionalHTTP                              `json:"items"`
	Total    int                                                     `json:"total"`
	Limit    int                                                     `json:"limit"`
	Offset   int                                                     `json:"offset"`
	Catalogo personalports.ReferenciaCatalogoCategoriasProfesionales `json:"catalogo"`
	Fuente   personalports.FuenteCategoriasProfesionalesConsultable  `json:"fuente"`
}

func (h *Handler) listarCategoriasProfesionales(ctx context.Context) (personalports.CatalogoCategoriasProfesionalesConsultable, error) {
	if h == nil || dependenciaHTTPNula(h.categoriasProfesionales) {
		return personalports.CatalogoCategoriasProfesionalesConsultable{}, personalports.ErrCatalogoCategoriasProfesionalesNoDisponible
	}
	return h.categoriasProfesionales.ListarVigentes(ctx)
}

func (h *Handler) writeCatalogoGobernadoConflict(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusConflict)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   "catalogo_gobernado_requiere_borrador",
		"message": "Las categorías publicadas no se modifican directamente; el cambio requiere una nueva versión en borrador y su flujo de aprobación.",
	})
}

func filtroCategoriasProfesionalesDesdePeticion(r *http.Request) (filtroCategoriasProfesionalesHTTP, error) {
	if r == nil || r.URL == nil {
		return filtroCategoriasProfesionalesHTTP{}, errors.New("filtro invalido")
	}
	valores, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return filtroCategoriasProfesionalesHTTP{}, errors.New("filtro invalido")
	}
	for clave, lista := range valores {
		if len(lista) != 1 || (clave != "q" && clave != "area" && clave != "limit" && clave != "offset") {
			return filtroCategoriasProfesionalesHTTP{}, errors.New("filtro invalido")
		}
	}
	consulta := strings.TrimSpace(valores.Get("q"))
	area := strings.TrimSpace(valores.Get("area"))
	if !utf8.ValidString(consulta) || !utf8.ValidString(area) || utf8.RuneCountInString(consulta) > 100 ||
		utf8.RuneCountInString(area) > 80 || !areaCategoriaProfesionalValida(area) {
		return filtroCategoriasProfesionalesHTTP{}, errors.New("filtro invalido")
	}
	limite, err := enteroConsultaCategorias(valores.Get("limit"))
	if err != nil {
		return filtroCategoriasProfesionalesHTTP{}, err
	}
	if limite <= 0 || limite > limiteMaximoCategoriasProfesionales {
		limite = limiteCategoriasProfesionalesPorDefecto
	}
	desde, err := enteroConsultaCategorias(valores.Get("offset"))
	if err != nil {
		return filtroCategoriasProfesionalesHTTP{}, err
	}
	if desde < 0 {
		desde = 0
	}
	return filtroCategoriasProfesionalesHTTP{consulta: consulta, area: area, limite: limite, desde: desde}, nil
}

func areaCategoriaProfesionalValida(area string) bool {
	if area == "" {
		return true
	}
	for indice, caracter := range area {
		if indice == 0 {
			if caracter >= 'a' && caracter <= 'z' {
				continue
			}
			return false
		}
		if (caracter >= 'a' && caracter <= 'z') || caracter == '_' || (caracter >= '0' && caracter <= '9') {
			continue
		}
		return false
	}
	return true
}

func enteroConsultaCategorias(valor string) (int, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return 0, nil
	}
	numero, err := strconv.Atoi(valor)
	if err != nil {
		return 0, errors.New("filtro invalido")
	}
	return numero, nil
}

func paginarCategoriasProfesionales(
	catalogo personalports.CatalogoCategoriasProfesionalesConsultable,
	filtro filtroCategoriasProfesionalesHTTP,
) paginaCategoriasProfesionalesHTTP {
	filtradas := make([]categoriaProfesionalHTTP, 0, len(catalogo.Categorias))
	consulta := textoBusquedaCategoriaProfesional(filtro.consulta)
	for _, categoria := range catalogo.Categorias {
		if filtro.area != "" && categoria.Area != filtro.area {
			continue
		}
		if consulta != "" {
			texto := textoBusquedaCategoriaProfesional(strings.Join([]string{
				categoria.Clave, categoria.Etiqueta, categoria.Descripcion, categoria.AreaEtiqueta,
			}, " "))
			if !strings.Contains(texto, consulta) {
				continue
			}
		}
		filtradas = append(filtradas, proyectarCategoriaProfesionalHTTP(categoria, catalogo.Fuente.Demostracion))
	}
	total := len(filtradas)
	desde := filtro.desde
	if desde > total {
		desde = total
	}
	hasta := desde + filtro.limite
	if hasta > total {
		hasta = total
	}
	items := append([]categoriaProfesionalHTTP(nil), filtradas[desde:hasta]...)
	return paginaCategoriasProfesionalesHTTP{
		Items: items, Total: total, Limit: filtro.limite, Offset: filtro.desde,
		Catalogo: catalogo.Referencia, Fuente: catalogo.Fuente,
	}
}

func proyectarCategoriaProfesionalHTTP(
	categoria personalports.CategoriaProfesionalConsultable,
	demostracion bool,
) categoriaProfesionalHTTP {
	estado := "Vigente"
	if demostracion {
		estado = "Demostración pendiente de validación RRHH"
	}
	return categoriaProfesionalHTTP{
		Catalogo: "categoria_profesional", Clave: categoria.Clave, Slug: categoria.Clave,
		Etiqueta: categoria.Etiqueta, Nombre: categoria.Etiqueta, Descripcion: categoria.Descripcion,
		Orden: categoria.Orden, Area: categoria.Area, AreaEtiqueta: categoria.AreaEtiqueta,
		Fuente: "catalogo_gobernado_vec", Modulo: personalmodule.ModuleID, Estado: estado,
		Uso: "Bolsa, RPT, certificados y demás módulos autorizados.",
	}
}

func textoBusquedaCategoriaProfesional(valor string) string {
	valor = norm.NFD.String(strings.ToLower(strings.TrimSpace(valor)))
	var salida strings.Builder
	salida.Grow(len(valor))
	for _, caracter := range valor {
		if unicode.Is(unicode.Mn, caracter) {
			continue
		}
		salida.WriteRune(caracter)
	}
	return salida.String()
}

func (h *Handler) handlePersonalCatalogs(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	if !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	if !h.requirePermission(w, principal, personalmodule.PermissionPositionRead) {
		return
	}
	if !h.requirePersonalCatalog(w) {
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

func (h *Handler) requirePersonalCatalog(w http.ResponseWriter) bool {
	if h != nil && !dependenciaHTTPNula(h.personalCatalog) {
		return true
	}
	h.writeError(w, http.StatusServiceUnavailable, "catalogo de Personal no disponible")
	return false
}

func dependenciaHTTPNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
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
