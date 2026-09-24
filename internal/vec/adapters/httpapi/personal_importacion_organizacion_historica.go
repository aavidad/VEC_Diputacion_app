package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	RutaPrepararImportacionOrganizacion  = "/api/vec/personal/organizacion-historica/importaciones/preparar"
	RutaConciliarImportacionOrganizacion = "/api/vec/personal/organizacion-historica/importaciones/conciliar"
	RutaPublicarImportacionOrganizacion  = "/api/vec/personal/organizacion-historica/importaciones/publicar"
	// El dominio limita el material canónico a 4 MiB. El transporte admite
	// margen para espacios y escapado JSON antes de esa comprobación.
	maximoCuerpoImportacionOrganizacion = 8 << 20
)

var (
	ErrHandlerImportacionOrganizacionInvalido = errors.New("personal http: importacion de organizacion no disponible")
	errEntradaImportacionOrganizacion         = errors.New("personal http: entrada de importacion invalida")
	patronClaveHTTPImportacionOrganizacion    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

// El resolutor obtiene actor, perfil activo y organismo de autoridades del
// servidor. Una cabecera o el cuerpo JSON no pueden establecerlos.
type AutoridadContextoImportacionOrganizacion interface {
	ResolverContextoImportacionOrganizacion(context.Context) (vecdomain.ContextoActor, string, error)
}

type OperadorImportacionOrganizacion interface {
	Preparar(context.Context, personaldomain.SolicitudImportacionOrganizacion) (personalports.ReciboImportacionOrganizacion, error)
	Conciliar(context.Context, personaldomain.SolicitudImportacionOrganizacion) (personalports.ReciboImportacionOrganizacion, error)
	Publicar(context.Context, personaldomain.SolicitudImportacionOrganizacion) (personalports.ReciboImportacionOrganizacion, error)
}

// La composición adapta este puerto a la autoridad común de auditoría.
// Sin registrador no se construye el manejador, y un fallo al registrar
// una denegación impide devolverla como resultado confirmado.
type AuditorDenegacionImportacionOrganizacion interface {
	RegistrarDenegacionImportacionOrganizacion(context.Context, DenegacionImportacionOrganizacion) error
}

type DenegacionImportacionOrganizacion struct {
	CorrelacionRef string
	Motivo         string
	Ruta           string
	ActorRef       string
}

type handlerImportacionOrganizacion struct {
	autoridad AutoridadContextoImportacionOrganizacion
	operador  OperadorImportacionOrganizacion
	auditoria AuditorDenegacionImportacionOrganizacion
}

var _ OperadorImportacionOrganizacion = (*personalapp.ServicioImportacionOrganizacion)(nil)

func NewHandlerImportacionOrganizacionPersonal(a AutoridadContextoImportacionOrganizacion, o OperadorImportacionOrganizacion, auditor AuditorDenegacionImportacionOrganizacion) (http.Handler, error) {
	if dependenciaHTTPNula(a) || dependenciaHTTPNula(o) || dependenciaHTTPNula(auditor) {
		return nil, ErrHandlerImportacionOrganizacionInvalido
	}
	return &handlerImportacionOrganizacion{a, o, auditor}, nil
}

type entradaImportacionOrganizacion struct {
	LoteRef          string                                            `json:"lote_ref"`
	RevisionEsperada int64                                             `json:"revision_esperada"`
	Manifiesto       personaldomain.ManifiestoImportacionOrganizacion  `json:"manifiesto"`
	Hechos           []personaldomain.HechoImportacionOrganizacion     `json:"hechos"`
	Decisiones       []personaldomain.DecisionConciliacionOrganizacion `json:"decisiones"`
}

func (h *handlerImportacionOrganizacion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || dependenciaHTTPNula(h.autoridad) || dependenciaHTTPNula(h.operador) || dependenciaHTTPNula(h.auditoria) {
		responderImportacionOrganizacion(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	fase, rutaValida := faseRutaImportacionOrganizacion(r)
	if !rutaValida {
		responderImportacionOrganizacion(w, http.StatusNotFound, "recurso_no_encontrado", nil)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderImportacionOrganizacion(w, http.StatusMethodNotAllowed, "metodo_no_permitido", nil)
		return
	}
	entrada, clave, err := leerEntradaImportacionOrganizacion(w, r)
	if err != nil {
		responderImportacionOrganizacion(w, http.StatusBadRequest, "peticion_no_valida", nil)
		return
	}
	actor, organismo, err := h.autoridad.ResolverContextoImportacionOrganizacion(r.Context())
	if err != nil {
		switch {
		case errors.Is(err, ErrAutenticacionRutaExactaRequerida), errors.Is(err, vecdomain.ErrContextoActorNoResuelto):
			h.denegar(w, r.Context(), http.StatusUnauthorized, "autenticacion_requerida", r.URL.Path, "")
		case errors.Is(err, ErrAccesoRutaExactaDenegado), errors.Is(err, vecdomain.ErrAutorizacionDenegada), errors.Is(err, vecdomain.ErrPermissionDenied):
			h.denegar(w, r.Context(), http.StatusForbidden, "acceso_denegado", r.URL.Path, "")
		default:
			responderImportacionOrganizacion(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		}
		return
	}
	if actor.Validar() != nil || organismo == "" {
		responderImportacionOrganizacion(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	entrada.Manifiesto.OrganismoRef = organismo
	for i := range entrada.Hechos {
		entrada.Hechos[i].OrganismoRef = organismo
	}
	solicitud := personaldomain.SolicitudImportacionOrganizacion{
		Fase: fase, LoteRef: entrada.LoteRef, RevisionEsperada: entrada.RevisionEsperada,
		ClaveIdempotencia: clave, CorrelacionRef: correlacionImportacionOrganizacion(clave),
		Manifiesto: entrada.Manifiesto, Hechos: entrada.Hechos, Decisiones: entrada.Decisiones, Actor: actor,
	}
	if fase == personaldomain.FasePublicarOrganizacion {
		solicitud.RevisorActorRef = actor.Principal.ID
	}
	if solicitud.Validar() != nil {
		responderImportacionOrganizacion(w, http.StatusBadRequest, "peticion_no_valida", nil)
		return
	}
	var recibo personalports.ReciboImportacionOrganizacion
	switch fase {
	case personaldomain.FasePrepararOrganizacion:
		recibo, err = h.operador.Preparar(r.Context(), solicitud)
	case personaldomain.FaseConciliarOrganizacion:
		recibo, err = h.operador.Conciliar(r.Context(), solicitud)
	case personaldomain.FasePublicarOrganizacion:
		recibo, err = h.operador.Publicar(r.Context(), solicitud)
	}
	if err != nil {
		switch {
		case errors.Is(err, personaldomain.ErrImportacionOrganizacionInvalida):
			responderImportacionOrganizacion(w, http.StatusBadRequest, "peticion_no_valida", nil)
		case errors.Is(err, personaldomain.ErrImportacionOrganizacionDenegada):
			h.denegar(w, r.Context(), http.StatusForbidden, "acceso_denegado", r.URL.Path, actor.Principal.ID)
		case errors.Is(err, personaldomain.ErrImportacionOrganizacionConflicto):
			responderImportacionOrganizacion(w, http.StatusConflict, "conflicto", nil)
		default:
			responderImportacionOrganizacion(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		}
		return
	}
	if !reciboImportacionHTTPValido(recibo, solicitud) {
		responderImportacionOrganizacion(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	estado := http.StatusCreated
	if recibo.Replay {
		estado = http.StatusOK
	}
	responderImportacionOrganizacion(w, estado, "", map[string]any{"data": recibo})
}

func faseRutaImportacionOrganizacion(r *http.Request) (personaldomain.FaseImportacionOrganizacion, bool) {
	if !peticionRutaExactaCanonica(r) || r.URL.RawQuery != "" {
		return "", false
	}
	switch r.URL.Path {
	case RutaPrepararImportacionOrganizacion:
		return personaldomain.FasePrepararOrganizacion, true
	case RutaConciliarImportacionOrganizacion:
		return personaldomain.FaseConciliarOrganizacion, true
	case RutaPublicarImportacionOrganizacion:
		return personaldomain.FasePublicarOrganizacion, true
	default:
		return "", false
	}
}

func leerEntradaImportacionOrganizacion(w http.ResponseWriter, r *http.Request) (entradaImportacionOrganizacion, string, error) {
	var entrada entradaImportacionOrganizacion
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength == 0 || r.ContentLength > maximoCuerpoImportacionOrganizacion ||
		len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 || cabeceraOrganizacionHistoricaPresente(r.Header, "Cookie") ||
		cabeceraOrganizacionHistoricaPresente(r.Header, "Proxy-Authorization") || cabeceraOrganizacionHistoricaPresente(r.Header, "Content-Encoding") ||
		!cabeceraImportacionOrganizacionExacta(r.Header, "Content-Type", "application/json") {
		return entrada, "", errEntradaImportacionOrganizacion
	}
	clave, ok := cabeceraImportacionOrganizacionUnica(r.Header, "Idempotency-Key")
	if !ok || !patronClaveHTTPImportacionOrganizacion.MatchString(clave) {
		return entrada, "", errEntradaImportacionOrganizacion
	}
	cuerpo, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoCuerpoImportacionOrganizacion+1))
	if err != nil || len(cuerpo) == 0 || len(cuerpo) > maximoCuerpoImportacionOrganizacion || !utf8.Valid(cuerpo) || validarJSONImportacionOrganizacion(cuerpo) != nil {
		return entrada, "", errEntradaImportacionOrganizacion
	}
	dec := json.NewDecoder(bytes.NewReader(cuerpo))
	dec.DisallowUnknownFields()
	if dec.Decode(&entrada) != nil || dec.Decode(&struct{}{}) != io.EOF {
		return entrada, "", errEntradaImportacionOrganizacion
	}
	return entrada, clave, nil
}

func cabeceraImportacionOrganizacionUnica(h http.Header, nombre string) (string, bool) {
	var valores []string
	for k, vs := range h {
		if strings.EqualFold(k, nombre) {
			valores = append(valores, vs...)
		}
	}
	returnValor := ""
	if len(valores) == 1 {
		returnValor = valores[0]
	}
	return returnValor, len(valores) == 1 && returnValor != "" && returnValor == strings.TrimSpace(returnValor)
}
func cabeceraImportacionOrganizacionExacta(h http.Header, nombre, esperado string) bool {
	valor, ok := cabeceraImportacionOrganizacionUnica(h, nombre)
	return ok && valor == esperado
}

// Recorre también los objetos anidados: JSON válido y claves únicas, sin
// campos de identidad, ámbito, fase o correlación controlados por el cliente.
func validarJSONImportacionOrganizacion(cuerpo []byte) error {
	d := json.NewDecoder(bytes.NewReader(cuerpo))
	if err := validarValorJSONImportacionOrganizacion(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errEntradaImportacionOrganizacion
	}
	return nil
}
func validarValorJSONImportacionOrganizacion(d *json.Decoder, profundidad int) error {
	if profundidad > 24 {
		return errEntradaImportacionOrganizacion
	}
	t, err := d.Token()
	if err != nil {
		return errEntradaImportacionOrganizacion
	}
	inicio, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch inicio {
	case '{':
		vistas := make(map[string]struct{})
		for d.More() {
			k, err := d.Token()
			if err != nil {
				return errEntradaImportacionOrganizacion
			}
			clave, ok := k.(string)
			if !ok {
				return errEntradaImportacionOrganizacion
			}
			if _, repetida := vistas[clave]; repetida || !claveCanonicaImportacionOrganizacion(clave) || claveProhibidaImportacionOrganizacion(clave) {
				return errEntradaImportacionOrganizacion
			}
			vistas[clave] = struct{}{}
			if err := validarValorJSONImportacionOrganizacion(d, profundidad+1); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := validarValorJSONImportacionOrganizacion(d, profundidad+1); err != nil {
				return err
			}
		}
	default:
		return errEntradaImportacionOrganizacion
	}
	fin, err := d.Token()
	if err != nil || fin != json.Delim('}') && fin != json.Delim(']') {
		return errEntradaImportacionOrganizacion
	}
	if inicio == '{' && fin != json.Delim('}') || inicio == '[' && fin != json.Delim(']') {
		return errEntradaImportacionOrganizacion
	}
	return nil
}
func claveProhibidaImportacionOrganizacion(k string) bool {
	switch k {
	case "organismo_ref", "actor", "perfil_activo_ref", "fase", "clave_idempotencia", "correlacion_ref", "revisor_actor_ref":
		return true
	default:
		return false
	}
}
func claveCanonicaImportacionOrganizacion(k string) bool {
	if k == "" || len(k) > 80 {
		return false
	}
	for _, c := range k {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
}
func correlacionImportacionOrganizacion(clave string) string {
	h := sha256.Sum256([]byte(clave))
	return "corr_" + hex.EncodeToString(h[:16])
}

func reciboImportacionHTTPValido(r personalports.ReciboImportacionOrganizacion, s personaldomain.SolicitudImportacionOrganizacion) bool {
	material, err := personaldomain.NuevoMaterialImportacionOrganizacion(s)
	if err != nil {
		return false
	}
	if r.ReciboRef == "" || r.LoteRef == "" || r.Fase != s.Fase || r.ClaveIdempotencia != s.ClaveIdempotencia || r.ActorRef != s.Actor.Principal.ID ||
		r.RevisionAnterior != s.RevisionEsperada || r.RevisionNueva != s.RevisionEsperada+1 || r.AuditoriaRef == "" || r.DecisionRef == "" ||
		r.MaterialHuellaSHA256 != material.HuellaSHA256() || r.FuenteHuellaSHA256 != s.Manifiesto.FuenteHuellaSHA256 ||
		!personaldomain.InstanteImportacionValido(r.RegistradoEn) {
		return false
	}
	if s.Fase != personaldomain.FasePrepararOrganizacion && r.LoteRef != s.LoteRef {
		return false
	}
	switch s.Fase {
	case personaldomain.FasePrepararOrganizacion:
		return r.Estado == "preparacion_no_autoritativa"
	case personaldomain.FaseConciliarOrganizacion:
		return r.Estado == "conciliacion_pendiente" || r.Estado == "conciliada"
	case personaldomain.FasePublicarOrganizacion:
		return r.Estado == "publicada"
	default:
		return false
	}
}

func (h *handlerImportacionOrganizacion) denegar(w http.ResponseWriter, ctx context.Context, estado int, codigo, ruta, actor string) {
	orden := DenegacionImportacionOrganizacion{CorrelacionRef: nuevaCorrelacionRutaExacta(), Motivo: codigo, Ruta: ruta, ActorRef: actor}
	ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoMaximoAuditoriaFronteraRutaExacta)
	defer cancelar()
	if err := h.auditoria.RegistrarDenegacionImportacionOrganizacion(ctxAuditoria, orden); err != nil {
		responderImportacionOrganizacion(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	responderImportacionOrganizacion(w, estado, codigo, nil)
}

func responderImportacionOrganizacion(w http.ResponseWriter, estado int, codigo string, datos any) {
	for _, k := range []string{"Set-Cookie", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials", "Location", "Content-Encoding"} {
		w.Header().Del(k)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	if datos == nil {
		datos = map[string]any{"error": map[string]string{"codigo": codigo, "clave_i18n": "api.personal.organizacion_historica.importacion.error." + codigo}}
	}
	contenido, err := json.Marshal(datos)
	if err != nil {
		estado = http.StatusServiceUnavailable
		contenido = []byte(`{"error":{"codigo":"servicio_no_disponible"}}`)
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(estado)
	_, _ = w.Write(contenido)
}
