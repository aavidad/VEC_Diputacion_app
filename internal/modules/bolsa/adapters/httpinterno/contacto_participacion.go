package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type EntradaRegistrarContactoParticipacion struct {
	BolsaRef, ParticipacionRef, LlamamientoRef, OfertaRef, EvidenciaRef, EvidenciaHuellaSHA256, Canal, Resultado, Anotacion, ClaveIdempotencia string
	Instante                                                                                                                                   time.Time
	InstanteServidor                                                                                                                           bool
}

const RutaContactosOferta = "/api/vec/bolsa/ofertas/contactos"

type PreparadorContactoParticipacion interface {
	PrepararSolicitudRegistrarContacto(context.Context, EntradaRegistrarContactoParticipacion) (puertosbolsa.SolicitudRegistrarContactoParticipacion, error)
	PrepararConsultaContactos(context.Context, string, string, string, int) (puertosbolsa.ConsultaContactosParticipacion, error)
	PrepararConsultaContactosBolsa(context.Context, string, string, int) (puertosbolsa.ConsultaContactosBolsa, error)
}
type OperadorContactoParticipacion interface {
	RegistrarContactoParticipacion(context.Context, puertosbolsa.SolicitudRegistrarContactoParticipacion) (puertosbolsa.RegistroContactoParticipacion, error)
	ListarContactosParticipacion(context.Context, puertosbolsa.ConsultaContactosParticipacion) (puertosbolsa.PaginaContactosParticipacion, error)
	ListarContactosBolsa(context.Context, puertosbolsa.ConsultaContactosBolsa) (puertosbolsa.PaginaContactosParticipacion, error)
}
type HandlerContactoParticipacion struct {
	preparador PreparadorContactoParticipacion
	operador   OperadorContactoParticipacion
}

func NuevoHandlerContactoParticipacion(p PreparadorContactoParticipacion, o OperadorContactoParticipacion) (http.Handler, error) {
	if p == nil || o == nil {
		return nil, errors.New("bolsa http interno: contactos no disponibles")
	}
	return &HandlerContactoParticipacion{p, o}, nil
}
func (h *HandlerContactoParticipacion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r != nil && r.URL != nil && r.URL.Path == RutaContactosOferta && r.URL.RawPath == "" {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			responderContacto(w, 405, map[string]any{"error": map[string]string{"codigo": "metodo_no_permitido"}})
			return
		}
		h.listarOferta(w, r)
		return
	}
	bolsa, participacion, ok := ReferenciasRutaContactosParticipacion(r)
	if !ok {
		responderContacto(w, 404, map[string]any{"error": map[string]string{"codigo": "recurso_no_encontrado"}})
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.listar(w, r, bolsa, participacion)
	case http.MethodPost:
		h.registrar(w, r, bolsa, participacion)
	default:
		w.Header().Set("Allow", "GET, POST")
		responderContacto(w, 405, map[string]any{"error": map[string]string{"codigo": "metodo_no_permitido"}})
	}
}
func (h *HandlerContactoParticipacion) registrar(w http.ResponseWriter, r *http.Request, bolsa, participacion string) {
	if r.URL.RawQuery != "" || r.Body == nil || r.Body == http.NoBody || r.ContentLength <= 0 || r.ContentLength > 4096 || len(r.TransferEncoding) != 0 || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" {
		responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	clave := r.Header.Get("Idempotency-Key")
	if len(r.Header.Values("Idempotency-Key")) != 1 || clave == "" || strings.TrimSpace(clave) != clave || len(clave) > 256 {
		responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	var c struct {
		Canal                 string          `json:"canal"`
		Instante              json.RawMessage `json:"instante"`
		Resultado             string          `json:"resultado"`
		Anotacion             string          `json:"anotacion"`
		LlamamientoRef        string          `json:"llamamiento_ref"`
		OfertaRef             string          `json:"oferta_ref"`
		EvidenciaRef          string          `json:"evidencia_ref"`
		EvidenciaHuellaSHA256 string          `json:"evidencia_huella_sha256"`
	}
	d := json.NewDecoder(io.LimitReader(r.Body, 4097))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(&struct{}{}) != io.EOF {
		responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	servidor := len(c.Instante) == 0 && c.Canal == dominiobolsa.CanalContactoTelefono && c.LlamamientoRef != "" && c.OfertaRef == ""
	if servidor {
		preparador, ok := h.preparador.(interface{ SoportaRegistroTelefonoServidor() bool })
		if !ok || !preparador.SoportaRegistroTelefonoServidor() {
			responderContacto(w, 503, map[string]any{"error": map[string]string{"codigo": "servicio_no_disponible"}})
			return
		}
	}
	var instante time.Time
	if !servidor && (len(c.Instante) == 0 || json.Unmarshal(c.Instante, &instante) != nil || instante.IsZero()) {
		responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	s, err := h.preparador.PrepararSolicitudRegistrarContacto(r.Context(), EntradaRegistrarContactoParticipacion{BolsaRef: bolsa, ParticipacionRef: participacion, LlamamientoRef: c.LlamamientoRef, OfertaRef: c.OfertaRef, EvidenciaRef: c.EvidenciaRef, EvidenciaHuellaSHA256: c.EvidenciaHuellaSHA256, Canal: c.Canal, Resultado: c.Resultado, Anotacion: c.Anotacion, ClaveIdempotencia: clave, Instante: instante, InstanteServidor: servidor})
	if err != nil {
		responderErrorContacto(w, err)
		return
	}
	out, err := h.operador.RegistrarContactoParticipacion(r.Context(), s)
	if err != nil {
		responderErrorContacto(w, err)
		return
	}
	estado := http.StatusCreated
	if out.Reutilizado {
		estado = http.StatusOK
	}
	datos := salidaContacto(out.Contacto, out.ReciboRef, out.Reutilizado)
	if out.Intentos != nil {
		datos["intentos"] = salidaEstadoIntentos(*out.Intentos)
	}
	responderContacto(w, estado, map[string]any{"data": datos})
}
func (h *HandlerContactoParticipacion) listar(w http.ResponseWriter, r *http.Request, bolsa, participacion string) {
	if r.ContentLength != 0 || r.Header.Get("Accept") != "application/json" {
		responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	v, e := url.ParseQuery(r.URL.RawQuery)
	llamamiento, conLlamamiento := v["llamamiento_ref"]
	oferta, conOferta := v["oferta_ref"]
	if e != nil || len(v) > 4 || (conLlamamiento && !referenciaLlamamientoConsultaValida(llamamiento)) || (conOferta && !referenciaOfertaContactoValida(oferta)) || (conOferta && conLlamamiento) {
		responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	for clave, lista := range v {
		if (clave != "llamamiento_ref" && clave != "oferta_ref" && clave != "cursor" && clave != "limite") || len(lista) != 1 {
			responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
			return
		}
	}
	cursor := v.Get("cursor")
	limite := 20
	if conLlamamiento {
		limite = limiteHistoricoIntentos
	}
	if x := v.Get("limite"); x != "" {
		limite, e = strconv.Atoi(x)
	}
	if e != nil || limite < 1 || limite > 100 {
		responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	consulta, e := h.preparador.PrepararConsultaContactos(r.Context(), bolsa, participacion, cursor, limite)
	if e != nil {
		responderErrorContacto(w, e)
		return
	}
	if conOferta {
		consulta.OfertaRef = oferta[0]
	}
	p, e := h.operador.ListarContactosParticipacion(r.Context(), consulta)
	if e != nil {
		publicarAcuseConsultaContactos(w, e)
		responderErrorContacto(w, e)
		return
	}
	items := make([]map[string]any, 0, len(p.Contactos))
	for _, c := range p.Contactos {
		items = append(items, salidaContactoLeido(c))
	}
	datos := map[string]any{"esquema": "vec.bolsa.rrhh.contactos.v1", "contactos": items, "cursor_siguiente": valorNulo(p.CursorSiguiente), "hay_mas": len(p.Contactos) == limite}
	if conLlamamiento {
		intentos, err := h.estadoIntentos(r.Context(), llamamiento[0], p.Contactos, cursor == "" && len(p.Contactos) < limite)
		if err != nil {
			responderErrorContacto(w, err)
			return
		}
		datos["intentos"] = intentos
		if p.RegistroTelefonoDisponible {
			if preparador, ok := h.preparador.(interface{ SoportaRegistroTelefonoServidor() bool }); ok && preparador.SoportaRegistroTelefonoServidor() {
				intentos["registro_telefono"] = map[string]any{"esquema": "vec.bolsa.registro_telefono.v1", "resultados": []string{dominiobolsa.ResultadoContactoContactado, dominiobolsa.ResultadoContactoNoContesta, dominiobolsa.ResultadoContactoComunica, dominiobolsa.ResultadoContactoNumeroErroneo, dominiobolsa.ResultadoContactoAcepta, dominiobolsa.ResultadoContactoRechaza, dominiobolsa.ResultadoContactoAplazado, dominiobolsa.ResultadoContactoBuzon}, "instante_servidor": true, "anotacion_opcional": true}
			}
		}
	}
	responderContacto(w, 200, map[string]any{"data": datos})
}

func referenciaOfertaContactoValida(v []string) bool {
	if len(v) != 1 || len(v[0]) != len("oferta:")+64 || !strings.HasPrefix(v[0], "oferta:") {
		return false
	}
	for _, r := range strings.TrimPrefix(v[0], "oferta:") {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func (h *HandlerContactoParticipacion) listarOferta(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength != 0 || r.Header.Get("Accept") != "application/json" {
		responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	v, err := url.ParseQuery(r.URL.RawQuery)
	bolsa, oferta := v["bolsa_ref"], v["oferta_ref"]
	if err != nil || len(v) < 2 || len(v) > 4 || len(bolsa) != 1 || bolsa[0] == "" || len(bolsa[0]) > 256 || !referenciaOfertaContactoValida(oferta) {
		responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	for clave, lista := range v {
		if (clave != "bolsa_ref" && clave != "oferta_ref" && clave != "cursor" && clave != "limite") || len(lista) != 1 {
			responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
			return
		}
	}
	limite := 50
	if v.Has("limite") {
		limite, err = strconv.Atoi(v.Get("limite"))
	}
	if err != nil || limite < 1 || limite > 100 || len(v.Get("cursor")) > 256 {
		responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	q, err := h.preparador.PrepararConsultaContactosBolsa(r.Context(), bolsa[0], v.Get("cursor"), limite)
	if err != nil {
		responderErrorContacto(w, err)
		return
	}
	q.OfertaRef = oferta[0]
	p, err := h.operador.ListarContactosBolsa(r.Context(), q)
	if err != nil {
		publicarAcuseConsultaContactos(w, err)
		responderErrorContacto(w, err)
		return
	}
	items := make([]map[string]any, 0, len(p.Contactos))
	for _, c := range p.Contactos {
		items = append(items, salidaContactoLeido(c))
	}
	responderContacto(w, 200, map[string]any{"data": map[string]any{"esquema": "vec.bolsa.rrhh.contactos-oferta.v1", "bolsa_ref": bolsa[0], "oferta_ref": oferta[0], "contactos": items, "cursor_siguiente": valorNulo(p.CursorSiguiente), "hay_mas": len(p.Contactos) == limite}})
}
func ReferenciasRutaContactosParticipacion(r *http.Request) (string, string, bool) {
	if r == nil || r.URL == nil || r.URL.RawPath != "" || strings.Contains(r.URL.EscapedPath(), "%") {
		return "", "", false
	}
	s := strings.Split(strings.TrimPrefix(r.URL.Path, RutaBolsasGestion+"/"), "/")
	if len(s) != 4 || s[0] == "" || s[1] != "candidatos" || s[2] == "" || s[3] != "contactos" {
		return "", "", false
	}
	return s[0], s[2], true
}
func salidaContacto(c dominiobolsa.ContactoParticipacion, recibo string, reutilizado bool) map[string]any {
	datos := salidaContactoLeido(c)
	datos["recibo_ref"] = valorNulo(recibo)
	datos["reutilizado"] = reutilizado
	return datos
}
func salidaContactoLeido(c dominiobolsa.ContactoParticipacion) map[string]any {
	return map[string]any{"contacto_ref": c.ContactoRef, "participacion_ref": c.ParticipacionRef, "llamamiento_ref": valorNulo(c.LlamamientoRef), "oferta_ref": valorNulo(c.OfertaRef), "evidencia_ref": valorNulo(c.EvidenciaRef), "evidencia_huella_sha256": valorNulo(c.EvidenciaHuellaSHA256), "canal": c.Canal, "instante": c.Instante.UTC().Format(time.RFC3339Nano), "actor_ref": c.Actor, "resultado": c.Resultado, "anotacion": c.Anotacion}
}
func valorNulo(v string) any {
	if v == "" {
		return nil
	}
	return v
}
func responderErrorContacto(w http.ResponseWriter, err error) {
	if codigo := codigoErrorIntento(err); codigo != "" {
		responderContacto(w, 409, map[string]any{"error": map[string]string{"codigo": codigo}})
		return
	}
	switch {
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada), errors.Is(err, dominiovec.ErrPermissionDenied):
		responderContacto(w, 403, map[string]any{"error": map[string]string{"codigo": "acceso_denegado"}})
	case errors.Is(err, dominiobolsa.ErrContactoParticipacionInvalido):
		responderContacto(w, 409, map[string]any{"error": map[string]string{"codigo": "contacto_en_conflicto"}})
	case errors.Is(err, puertosbolsa.ErrContactoParticipacionNoEncontrado):
		responderContacto(w, 404, map[string]any{"error": map[string]string{"codigo": "recurso_no_encontrado"}})
	default:
		responderContacto(w, 503, map[string]any{"error": map[string]string{"codigo": "servicio_no_disponible"}})
	}
}

// publicarAcuseConsultaContactos solo expone referencias de un intento AD169
// cuyo acuse ya validó la aplicación; nunca datos del contacto.
func publicarAcuseConsultaContactos(w http.ResponseWriter, err error) {
	if acuse, confirmado := application.AcuseConsultaContactosFallida(err); confirmado {
		w.Header().Set("X-Audit-Ref", acuse.AuditoriaRef)
		w.Header().Set("X-Correlation-Ref", acuse.CorrelacionRef)
	}
}
func responderContacto(w http.ResponseWriter, estado int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(v)
}
