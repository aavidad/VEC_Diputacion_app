package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/application"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	accionConsultarDocumentosBolsa = "lectura_bolsas_rrhh_consultar"
	accionDescargarReciboBolsa     = "lectura_bolsas_rrhh_descargar_recibo"
)

type claveCapturaNominalDocumentosBolsa struct{}

// La captura sólo existe dentro de una petición y se rellena una vez tras
// validar la sesión real. Conserva el actor histórico para auditar el resultado
// de una lectura que termina cuando la conexión HTTP ya se ha cancelado.
type capturaNominalDocumentosBolsa struct {
	mu         sync.Mutex
	soporte    *soporteAltaContratacionTemporalDesarrollo
	sello      *selloConsultasContratacionTemporalDesarrollo
	actorID    string
	certSHA256 string
	ruta       string
	metodo     string
	bolsaRef   string
	accion     string
	contexto   ctports.ContextoAutorizacionAltaV3
	lista      bool
}

func (c *capturaNominalDocumentosBolsa) guardar(s *soporteAltaContratacionTemporalDesarrollo,
	capacidad capacidadConsultaContratacionTemporalDesarrollo, bolsaRef, accion string,
	contexto ctports.ContextoAutorizacionAltaV3,
) error {
	bolsaRuta, _, descarga, rutaValida := referenciasDocumentosBolsaRRHH(capacidad.ruta)
	if c == nil || s == nil || !soporteDocumentoCoincideConCapacidad(s, capacidad) ||
		!rutaValida || bolsaRuta != bolsaRef ||
		(descarga != (accion == accionDescargarReciboBolsa)) ||
		(accion != accionConsultarDocumentosBolsa && accion != accionDescargarReciboBolsa) ||
		(capacidad.metodo != http.MethodGet && capacidad.metodo != http.MethodHead) ||
		contexto.Resultado.Validar() != nil || contexto.Vinculo.ValidarPara(contexto.Resultado) != nil {
		return vecdomain.ErrAutorizacionDenegada
	}
	resultado, err := contexto.Resultado.Clonar()
	if err != nil {
		return vecdomain.ErrAutorizacionDenegada
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lista {
		return bolsaapp.ErrConsultaDocumentosBolsaNoDisponible
	}
	c.soporte, c.sello, c.actorID, c.certSHA256 = s, capacidad.sello, capacidad.principal.ID, capacidad.principal.Attributes["certificate_sha256"]
	c.ruta, c.metodo, c.bolsaRef, c.accion = capacidad.ruta, capacidad.metodo, bolsaRef, accion
	c.contexto = ctports.ContextoAutorizacionAltaV3{Vinculo: contexto.Vinculo, Resultado: resultado}
	c.lista = true
	return nil
}

// El auditor usa esta copia ya validada, sin volver a resolver una sesión que
// pudo caducar después del acceso. El contexto de la petición conserva sus
// valores incluso cuando Err() indica cancelación.
func contextoNominalAuditadoDocumentosBolsa(ctx context.Context, bolsaRef, accion string) (ctports.ContextoAutorizacionAltaV3, error) {
	vacia := ctports.ContextoAutorizacionAltaV3{}
	if ctx == nil {
		return vacia, bolsaapp.ErrConsultaDocumentosBolsaNoDisponible
	}
	c, ok := ctx.Value(claveCapturaNominalDocumentosBolsa{}).(*capturaNominalDocumentosBolsa)
	capacidad, tieneCapacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	if !ok || c == nil || !tieneCapacidad {
		return vacia, bolsaapp.ErrConsultaDocumentosBolsaNoDisponible
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.lista || c.soporte == nil || c.bolsaRef != bolsaRef || c.accion != accion ||
		c.ruta != capacidad.ruta || c.metodo != capacidad.metodo || c.sello != capacidad.sello ||
		c.actorID != capacidad.principal.ID || c.certSHA256 != capacidad.principal.Attributes["certificate_sha256"] ||
		!soporteDocumentoCoincideConCapacidad(c.soporte, capacidad) {
		return vacia, bolsaapp.ErrConsultaDocumentosBolsaNoDisponible
	}
	resultado, err := c.contexto.Resultado.Clonar()
	if err != nil || c.contexto.Vinculo.ValidarPara(resultado) != nil {
		return vacia, bolsaapp.ErrConsultaDocumentosBolsaNoDisponible
	}
	return ctports.ContextoAutorizacionAltaV3{Vinculo: c.contexto.Vinculo, Resultado: resultado}, nil
}

// El registro documental de Bolsa reutiliza exactamente la autoridad RRHH de
// su lista actual. La sesión nominal de ese canal se revalida y registra; la
// auditoría de éxito es una dependencia obligatoria, previa a la respuesta.
func nuevoControlLecturaDocumentosBolsa(resolver func(context.Context) (*soporteAltaContratacionTemporalDesarrollo, error),
	auditar func(context.Context, string, string, error) error) *controlLecturaDocumentosBolsa {
	if resolver == nil || auditar == nil {
		return nil
	}
	return &controlLecturaDocumentosBolsa{Autorizar: func(ctx context.Context, bolsaRef, accion string) error {
		s, err := resolver(ctx)
		if err != nil {
			return err
		}
		if s == nil {
			return bolsaapp.ErrConsultaDocumentosBolsaNoDisponible
		}
		return s.autorizarDocumentosBolsa(ctx, bolsaRef, accion)
	}, Auditar: auditar}
}

// El soporte procede de la composición RRHH ya publicada: el del alta, que
// es también el que usa la técnica. Solo se elige por la capacidad mTLS
// sellada; el identificador de la URL no selecciona actor.
func resolverSoporteDocumentosBolsa(base *soporteAltaContratacionTemporalDesarrollo,
) func(context.Context) (*soporteAltaContratacionTemporalDesarrollo, error) {
	return func(ctx context.Context) (*soporteAltaContratacionTemporalDesarrollo, error) {
		if ctx == nil || ctx.Err() != nil {
			return nil, vecdomain.ErrAutorizacionDenegada
		}
		c, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
		if !ok || c.sello == nil || !rutaDocumentosBolsaRRHHDesarrollo(c.ruta) ||
			(c.metodo != http.MethodGet && c.metodo != http.MethodHead) ||
			!principalContratacionTemporalDesarrolloValido(c.principal) {
			return nil, vecdomain.ErrAutorizacionDenegada
		}
		if soporteDocumentoCoincideConCapacidad(base, c) {
			return base, nil
		}
		return nil, vecdomain.ErrAutorizacionDenegada
	}
}

func soporteDocumentoCoincideConCapacidad(s *soporteAltaContratacionTemporalDesarrollo,
	c capacidadConsultaContratacionTemporalDesarrollo) bool {
	return s != nil && s.sello == c.sello && s.principalID == c.principal.ID &&
		s.certificadoSHA256 == c.principal.Attributes["certificate_sha256"]
}

func (s *soporteAltaContratacionTemporalDesarrollo) autorizarDocumentosBolsa(ctx context.Context, bolsaRef, accion string) error {
	if s == nil || ctx == nil || ctx.Err() != nil || !referenciaBolsaDocumentoAuditable(bolsaRef) ||
		(accion != accionConsultarDocumentosBolsa && accion != accionDescargarReciboBolsa) {
		return vecdomain.ErrAutorizacionDenegada
	}
	capacidad, valida := s.capacidadValida(ctx)
	bolsaRuta, _, descarga, rutaValida := referenciasDocumentosBolsaRRHH(capacidad.ruta)
	if !valida || !rutaValida || bolsaRuta != bolsaRef ||
		(descarga != (accion == accionDescargarReciboBolsa)) ||
		(capacidad.metodo != http.MethodGet && capacidad.metodo != http.MethodHead) ||
		!principalContratacionTemporalDesarrolloValido(capacidad.principal) {
		return vecdomain.ErrAutorizacionDenegada
	}
	instante := time.Now().UTC().Truncate(time.Microsecond)
	if capacidad.certificadoVerificadoEn.IsZero() || capacidad.certificadoVerificadoEn.After(instante) ||
		!instante.Before(capacidad.certificadoValidoHasta) {
		return vecdomain.ErrAutorizacionDenegada
	}
	contexto, err := s.contextoOperativoDesarrollo(ctx)
	if errors.Is(err, ctports.ErrConsultaRRHHNoDisponible) {
		return bolsaapp.ErrConsultaDocumentosBolsaNoDisponible
	}
	if err != nil || contexto.Resultado.Validar() != nil || contexto.Vinculo.ValidarPara(contexto.Resultado) != nil {
		return vecdomain.ErrAutorizacionDenegada
	}
	captura, ok := ctx.Value(claveCapturaNominalDocumentosBolsa{}).(*capturaNominalDocumentosBolsa)
	if !ok || captura.guardar(s, capacidad, bolsaRef, accion, contexto) != nil {
		return bolsaapp.ErrConsultaDocumentosBolsaNoDisponible
	}
	return nil
}

// El control se compone desde la autoridad nominal de Bolsa. Autorizar se
// ejecuta antes de consultar el repositorio y Auditar antes de escribir HTTP.
// Si falta cualquiera, la ruta queda cerrada.
type controlLecturaDocumentosBolsa struct {
	Autorizar func(context.Context, string, string) error
	Auditar   func(context.Context, string, string, error) error
}

func rutaDocumentosBolsaRRHHDesarrollo(ruta string) bool {
	_, _, _, ok := referenciasDocumentosBolsaRRHH(ruta)
	return ok
}

func referenciasDocumentosBolsaRRHH(ruta string) (bolsaRef, documentoRef string, descarga bool, ok bool) {
	if !strings.HasPrefix(ruta, rutaBolsasRRHHDesarrollo+"/") {
		return "", "", false, false
	}
	partes := strings.Split(strings.TrimPrefix(ruta, rutaBolsasRRHHDesarrollo+"/"), "/")
	if len(partes) == 2 && partes[1] == "documentos" && referenciaBolsaDocumentoAuditable(partes[0]) {
		return partes[0], "", false, true
	}
	if len(partes) == 4 && partes[1] == "documentos" && partes[3] == "recibo" &&
		referenciaBolsaDocumentoAuditable(partes[0]) && referenciaDocumentoRutaBolsaValida(partes[2]) {
		return partes[0], partes[2], true, true
	}
	return "", "", false, false
}

func referenciaDocumentoRutaBolsaValida(ref string) bool {
	if ref == "" || len(ref) > 512 || strings.Contains(ref, "..") {
		return false
	}
	for _, c := range ref {
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') &&
			c != ':' && c != '.' && c != '_' && c != '~' && c != '-' {
			return false
		}
	}
	return true
}

func referenciaBolsaDocumentoAuditable(ref string) bool {
	if !strings.HasPrefix(ref, "bolsa:") || len(ref) > 200 || strings.Contains(ref, "..") ||
		len(ref) <= len("bolsa:") {
		return false
	}
	for _, c := range ref {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') &&
			c != ':' && c != '.' && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func consultaDocumentosBolsaRRHH(cruda, bolsaRef string) (bolsaapp.ConsultaDocumentosBolsa, error) {
	q := bolsaapp.ConsultaDocumentosBolsa{BolsaRef: bolsaRef, Limite: 50}
	if len(cruda) > 1024 {
		return q, bolsaapp.ErrConsultaDocumentosBolsaInvalida
	}
	v, err := url.ParseQuery(cruda)
	if err != nil {
		return q, errors.Join(bolsaapp.ErrConsultaDocumentosBolsaInvalida, err)
	}
	if len(v) > 4 {
		return q, bolsaapp.ErrConsultaDocumentosBolsaInvalida
	}
	for clave, valores := range v {
		if len(valores) != 1 || valores[0] == "" || valores[0] != strings.TrimSpace(valores[0]) {
			return q, bolsaapp.ErrConsultaDocumentosBolsaInvalida
		}
		switch clave {
		case "tipo":
			q.Tipo = valores[0]
		case "texto":
			q.Texto = valores[0]
		case "cursor":
			q.Cursor = valores[0]
		case "limite":
			n, e := strconv.Atoi(valores[0])
			if e != nil {
				return q, errors.Join(bolsaapp.ErrConsultaDocumentosBolsaInvalida, e)
			}
			if strconv.Itoa(n) != valores[0] {
				return q, bolsaapp.ErrConsultaDocumentosBolsaInvalida
			}
			q.Limite = n
		default:
			return q, bolsaapp.ErrConsultaDocumentosBolsaInvalida
		}
	}
	if (q.Tipo != "" && q.Tipo != "constitucion") || len(q.Texto) > 120 || q.Limite < 1 || q.Limite > 100 ||
		len(q.Cursor) > 512 || strings.ContainsAny(q.Cursor, "/\x00\r\n") {
		return q, bolsaapp.ErrConsultaDocumentosBolsaInvalida
	}
	return q, nil
}

func (h *bolsasRRHHDesarrollo) responderDocumentosBolsa(w http.ResponseWriter, r *http.Request) {
	bolsaRef, documentoRef, descarga, ok := referenciasDocumentosBolsaRRHH(r.URL.Path)
	if !ok {
		responderBolsaRRHHDesarrollo(w, http.StatusNotFound, map[string]string{"codigo": "recurso_no_encontrado"})
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		responderBolsaRRHHDesarrollo(w, http.StatusMethodNotAllowed, map[string]string{"codigo": "metodo_no_permitido"})
		return
	}
	if h.documentos == nil || h.controlDocumentos == nil || h.controlDocumentos.Autorizar == nil || h.controlDocumentos.Auditar == nil {
		responderBolsaRRHHDesarrollo(w, http.StatusServiceUnavailable, map[string]string{"codigo": "servicio_no_disponible"})
		return
	}
	r = r.WithContext(context.WithValue(r.Context(), claveCapturaNominalDocumentosBolsa{}, &capturaNominalDocumentosBolsa{}))
	accion := accionConsultarDocumentosBolsa
	if descarga {
		accion = accionDescargarReciboBolsa
	}
	if err := h.controlDocumentos.Autorizar(r.Context(), bolsaRef, accion); err != nil {
		if h.controlDocumentos.Auditar(r.Context(), bolsaRef, accion, err) != nil {
			responderBolsaRRHHDesarrollo(w, http.StatusServiceUnavailable, map[string]string{"codigo": "servicio_no_disponible"})
			return
		}
		if errors.Is(err, bolsaapp.ErrConsultaDocumentosBolsaNoDisponible) {
			responderBolsaRRHHDesarrollo(w, http.StatusServiceUnavailable, map[string]string{"codigo": "servicio_no_disponible"})
		} else {
			responderBolsaRRHHDesarrollo(w, http.StatusForbidden, map[string]string{"codigo": "acceso_denegado"})
		}
		return
	}
	if r.URL.RawPath != "" || len(r.TransferEncoding) != 0 ||
		cabeceraCatalogosAltaContratacionTemporalDesarrolloProhibida(r.Header) ||
		r.ContentLength != 0 || (descarga && r.URL.RawQuery != "") {
		if h.controlDocumentos.Auditar(r.Context(), bolsaRef, accion, bolsaapp.ErrConsultaDocumentosBolsaInvalida) != nil {
			responderBolsaRRHHDesarrollo(w, http.StatusServiceUnavailable, map[string]string{"codigo": "servicio_no_disponible"})
			return
		}
		responderBolsaRRHHDesarrollo(w, http.StatusBadRequest, map[string]string{"codigo": "solicitud_invalida"})
		return
	}
	q := bolsaapp.ConsultaDocumentosBolsa{BolsaRef: bolsaRef, Limite: 1}
	if !descarga {
		var errConsulta error
		q, errConsulta = consultaDocumentosBolsaRRHH(r.URL.RawQuery, bolsaRef)
		if errConsulta != nil {
			if h.controlDocumentos.Auditar(r.Context(), bolsaRef, accion, errConsulta) != nil {
				responderBolsaRRHHDesarrollo(w, http.StatusServiceUnavailable, map[string]string{"codigo": "servicio_no_disponible"})
				return
			}
			responderBolsaRRHHDesarrollo(w, http.StatusBadRequest, map[string]string{"codigo": "solicitud_invalida"})
			return
		}
	}
	pagina, err := h.documentos.Consultar(r.Context(), q)
	if err == nil && descarga && (len(pagina.Documentos) != 1 || pagina.Documentos[0].DocumentoRef != documentoRef) {
		err = bolsaapp.ErrDocumentoBolsaNoEncontrado
	}
	if h.controlDocumentos.Auditar(r.Context(), bolsaRef, accion, err) != nil {
		responderBolsaRRHHDesarrollo(w, http.StatusServiceUnavailable, map[string]string{"codigo": "servicio_no_disponible"})
		return
	}
	if err != nil {
		if errors.Is(err, bolsaapp.ErrConsultaDocumentosBolsaInvalida) {
			responderBolsaRRHHDesarrollo(w, http.StatusBadRequest, map[string]string{"codigo": "solicitud_invalida"})
		} else if errors.Is(err, bolsaapp.ErrDocumentoBolsaNoEncontrado) {
			responderBolsaRRHHDesarrollo(w, http.StatusNotFound, map[string]string{"codigo": "recurso_no_encontrado"})
		} else {
			responderBolsaRRHHDesarrollo(w, http.StatusServiceUnavailable, map[string]string{"codigo": "servicio_no_disponible"})
		}
		return
	}
	if descarga {
		d := pagina.Documentos[0]
		w.Header().Set("Content-Disposition", `attachment; filename="recibo-constitucion-bolsa.json"`)
		responderBolsaRRHHDesarrollo(w, http.StatusOK, map[string]any{"data": map[string]any{
			"esquema": "vec.bolsa.rrhh.recibo_constitucion.v1", "documento_ref": d.DocumentoRef,
			"bolsa_ref": d.BolsaRef, "acta_ref": d.ActaRef, "version_bolsa": d.VersionBolsa,
			"instantanea_ref": d.InstantaneaRef, "version_instantanea": d.VersionInstantanea,
			"listado_definitivo_ref": d.ListadoDefinitivoRef, "version_listado": d.VersionListado,
			"huella_listado_sha256": d.HuellaListadoSHA256,
			"confirmada_en":         d.ConfirmadaEn, "total_participaciones": d.TotalParticipaciones,
			"estado_firma": "sin_firma_acreditada",
		}}, r.Method == http.MethodHead)
		return
	}
	documentos := make([]map[string]any, 0, len(pagina.Documentos))
	for _, d := range pagina.Documentos {
		rutaBase := rutaBolsasRRHHDesarrollo + "/" + url.PathEscape(d.BolsaRef)
		documentos = append(documentos, map[string]any{
			"documento_ref": d.DocumentoRef, "tipo": d.Tipo,
			"titulo_clave": "bolsa_documento_constitucion", "fecha": d.ConfirmadaEn,
			"acta_ref": d.ActaRef, "version_bolsa": d.VersionBolsa,
			"instantanea_ref": d.InstantaneaRef, "version_instantanea": d.VersionInstantanea,
			"listado_definitivo_ref": d.ListadoDefinitivoRef, "version_listado": d.VersionListado,
			"huella_listado_sha256": d.HuellaListadoSHA256,
			"total_participaciones": d.TotalParticipaciones,
			"estado_firma":          "sin_firma_acreditada", "formato": "application/json",
			"accion":                map[string]string{"tipo": "descargar_recibo", "href": rutaBase + "/documentos/" + url.PathEscape(d.DocumentoRef) + "/recibo"},
			"ficha_candidatos_href": rutaBase + "/candidatos",
		})
	}
	responderBolsaRRHHDesarrollo(w, http.StatusOK, map[string]any{"data": map[string]any{
		"esquema": "vec.bolsa.rrhh.documentos.v1", "bolsa_ref": bolsaRef,
		"documentos": documentos, "paginacion": map[string]any{
			"limite": q.Limite, "total": pagina.Total, "cursor_siguiente": pagina.CursorSiguiente,
		},
	}}, r.Method == http.MethodHead)
}
