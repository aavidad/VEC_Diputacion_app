package httpinterno

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	RutaConsultaCuadroRRHH  = "/api/vec/contratacion-temporal/cuadro/consultas"
	RutaConsultaDetalleRRHH = "/api/vec/contratacion-temporal/expedientes/consultas"
)

var ErrManejadorConsultaRRHHInvalido = errors.New(
	"contratacion temporal http: manejador de consulta RRHH invalido",
)

// ConsultorCuadroRRHH mantiene al adaptador desacoplado de la implementación
// del caso de uso. La identidad y el ámbito se resuelven dentro de aplicación.
type ConsultorCuadroRRHH interface {
	Consultar(context.Context, ports.SolicitudCuadroRRHH) (ports.PaginaCuadroRRHH, error)
}

// ConsultorDetalleRRHH expone únicamente la intención mínima de lectura.
type ConsultorDetalleRRHH interface {
	Consultar(context.Context, ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error)
}
type ResolutorPresentacionFlujoRRHH interface {
	Resolver(context.Context, domain.ReferenciaFlujo, domain.ClaveFase) (ports.PresentacionFlujoRRHH, error)
}

// RenderizadorBorradorRRHHDOCX recibe sólo el detalle que ya superó la misma
// consulta autorizada que PDF. La representación no amplía la autoridad.
type RenderizadorBorradorRRHHDOCX interface {
	RenderizarBorradorDOCX(context.Context, ports.TipoBorradorRRHH, ports.DetalleExpedienteRRHH) ([]byte, error)
}

type manejadorConsultaCuadroRRHH struct {
	consultor ConsultorCuadroRRHH
}

type manejadorConsultaDetalleRRHH struct {
	consultor                  ConsultorDetalleRRHH
	consultorOriginalPropuesta ConsultorDetalleRRHH
	renderizador               ports.RenderizadorBorradorRRHH
	renderizadorDOCX           RenderizadorBorradorRRHHDOCX
	presentacion               ResolutorPresentacionFlujoRRHH
}

func NuevoManejadorConsultaDetalleRRHHConPresentacion(consultor ConsultorDetalleRRHH, presentacion ResolutorPresentacionFlujoRRHH, renderizadores ...ports.RenderizadorBorradorRRHH) (http.Handler, error) {
	h, err := NuevoManejadorConsultaDetalleRRHH(consultor, renderizadores...)
	if err != nil || dependenciaConsultaRRHHNula(presentacion) {
		return nil, ErrManejadorConsultaRRHHInvalido
	}
	h.(*manejadorConsultaDetalleRRHH).presentacion = presentacion
	return h, nil
}

// ConfigurarPresentacionConsultaDetalleRRHH añade el rail a un manejador de detalle ya
// construido, conservando exactamente sus variantes PDF, DOCX y propuesta.
func ConfigurarPresentacionConsultaDetalleRRHH(handler http.Handler, presentacion ResolutorPresentacionFlujoRRHH) (http.Handler, error) {
	h, ok := handler.(*manejadorConsultaDetalleRRHH)
	if !ok || h == nil || dependenciaConsultaRRHHNula(presentacion) {
		return nil, ErrManejadorConsultaRRHHInvalido
	}
	copia := *h
	copia.presentacion = presentacion
	return &copia, nil
}

var (
	_ http.Handler         = (*manejadorConsultaCuadroRRHH)(nil)
	_ http.Handler         = (*manejadorConsultaDetalleRRHH)(nil)
	_ ConsultorCuadroRRHH  = (*application.ServicioConsultaCuadroRRHH)(nil)
	_ ConsultorDetalleRRHH = (*application.ServicioConsultaDetalleRRHH)(nil)
)

func NuevoManejadorConsultaCuadroRRHH(
	consultor ConsultorCuadroRRHH,
) (http.Handler, error) {
	if dependenciaConsultaRRHHNula(consultor) {
		return nil, ErrManejadorConsultaRRHHInvalido
	}
	return &manejadorConsultaCuadroRRHH{consultor: consultor}, nil
}

func NuevoManejadorConsultaDetalleRRHH(
	consultor ConsultorDetalleRRHH,
	renderizadores ...ports.RenderizadorBorradorRRHH,
) (http.Handler, error) {
	if dependenciaConsultaRRHHNula(consultor) || len(renderizadores) > 1 {
		return nil, ErrManejadorConsultaRRHHInvalido
	}
	h := &manejadorConsultaDetalleRRHH{consultor: consultor}
	if len(renderizadores) == 1 {
		if dependenciaConsultaRRHHNula(renderizadores[0]) {
			return nil, ErrManejadorConsultaRRHHInvalido
		}
		h.renderizador = renderizadores[0]
	}
	return h, nil
}

// NuevoManejadorConsultaDetalleRRHHConOriginalPropuesta añade la segunda
// lectura autorizada para representar la propuesta v7 cuando la anotación ya
// llevó el expediente a v9. Conserva la consulta ordinaria para la petición
// inicial y para cualquier detalle no histórico.
func NuevoManejadorConsultaDetalleRRHHConOriginalPropuesta(
	consultor ConsultorDetalleRRHH,
	originalPropuesta ConsultorDetalleRRHH,
	renderizadores ...ports.RenderizadorBorradorRRHH,
) (http.Handler, error) {
	h, err := NuevoManejadorConsultaDetalleRRHH(consultor, renderizadores...)
	if err != nil || dependenciaConsultaRRHHNula(originalPropuesta) {
		return nil, ErrManejadorConsultaRRHHInvalido
	}
	manejador, ok := h.(*manejadorConsultaDetalleRRHH)
	if !ok {
		return nil, ErrManejadorConsultaRRHHInvalido
	}
	manejador.consultorOriginalPropuesta = originalPropuesta
	return manejador, nil
}

// NuevoManejadorConsultaDetalleRRHHConDOCX conserva el constructor PDF para
// consumidores existentes y añade DOCX como otra representación de la misma
// lectura ya autorizada.
func NuevoManejadorConsultaDetalleRRHHConDOCX(
	consultor ConsultorDetalleRRHH,
	pdf ports.RenderizadorBorradorRRHH,
	docx RenderizadorBorradorRRHHDOCX,
) (http.Handler, error) {
	if dependenciaConsultaRRHHNula(docx) {
		return nil, ErrManejadorConsultaRRHHInvalido
	}
	h, err := NuevoManejadorConsultaDetalleRRHH(consultor, pdf)
	if err != nil {
		return nil, err
	}
	manejador, ok := h.(*manejadorConsultaDetalleRRHH)
	if !ok {
		return nil, ErrManejadorConsultaRRHHInvalido
	}
	manejador.renderizadorDOCX = docx
	return manejador, nil
}

// NuevoManejadorConsultaDetalleRRHHConOriginalPropuestaYDOCX conserva las
// representaciones ya publicadas y entrega la recuperación v7 sólo al
// consultor de fachada histórica.
func NuevoManejadorConsultaDetalleRRHHConOriginalPropuestaYDOCX(
	consultor ConsultorDetalleRRHH,
	originalPropuesta ConsultorDetalleRRHH,
	pdf ports.RenderizadorBorradorRRHH,
	docx RenderizadorBorradorRRHHDOCX,
) (http.Handler, error) {
	if dependenciaConsultaRRHHNula(docx) {
		return nil, ErrManejadorConsultaRRHHInvalido
	}
	h, err := NuevoManejadorConsultaDetalleRRHHConOriginalPropuesta(
		consultor, originalPropuesta, pdf,
	)
	if err != nil {
		return nil, err
	}
	manejador, ok := h.(*manejadorConsultaDetalleRRHH)
	if !ok {
		return nil, ErrManejadorConsultaRRHHInvalido
	}
	manejador.renderizadorDOCX = docx
	return manejador, nil
}

func (h *manejadorConsultaCuadroRRHH) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || dependenciaConsultaRRHHNula(h.consultor) {
		responderErrorConsultaRRHH(w, errorServicioConsultaRRHHNoDisponible)
		return
	}
	if !rutaConsultaRRHHExacta(r, RutaConsultaCuadroRRHH) {
		responderErrorConsultaRRHH(w, errorRecursoConsultaRRHHNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorConsultaRRHH(w, errorMetodoConsultaRRHHNoPermitido)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(err))
		return
	}
	if problema := validarMetadatosConsultaRRHH(r, MaximoCuerpoConsultaCuadroRRHHBytes); problema != nil {
		responderErrorConsultaRRHH(w, *problema)
		return
	}
	solicitud, err := solicitudCuadroRRHHDesdePeticion(w, r)
	if err != nil {
		responderErrorConsultaRRHH(w, errorEntradaConsultaRRHH(err))
		return
	}
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(errContexto))
		return
	}
	pagina, err := h.consultor.Consultar(r.Context(), solicitud)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(errContexto))
		return
	}
	if err != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(err))
		return
	}
	if pagina.ValidarContenidoPublicablePara(solicitud) != nil {
		responderErrorConsultaRRHH(w, errorResultadoConsultaRRHHNoConfiable)
		return
	}
	responderJSONConsultaRRHH(
		w,
		http.StatusOK,
		envoltorioCuadroRRHH{Data: proyectarPaginaCuadroRRHH(pagina)},
	)
}

func (h *manejadorConsultaDetalleRRHH) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || dependenciaConsultaRRHHNula(h.consultor) {
		responderErrorConsultaRRHH(w, errorServicioConsultaRRHHNoDisponible)
		return
	}
	if !rutaConsultaRRHHExacta(r, RutaConsultaDetalleRRHH) {
		responderErrorConsultaRRHH(w, errorRecursoConsultaRRHHNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorConsultaRRHH(w, errorMetodoConsultaRRHHNoPermitido)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(err))
		return
	}
	if problema := validarMetadatosConsultaRRHHConPDF(r, MaximoCuerpoConsultaDetalleRRHHBytes, true); problema != nil {
		responderErrorConsultaRRHH(w, *problema)
		return
	}
	solicitud, err := solicitudDetalleRRHHDesdePeticion(w, r)
	if err != nil {
		responderErrorConsultaRRHH(w, errorEntradaConsultaRRHH(err))
		return
	}
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(errContexto))
		return
	}
	detalle, err := h.consultor.Consultar(r.Context(), solicitud)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(errContexto))
		return
	}
	if err != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(err))
		return
	}
	if detalle.ValidarContenidoPublicablePara(solicitud) != nil {
		responderErrorConsultaRRHH(w, errorResultadoConsultaRRHHNoConfiable)
		return
	}
	if borrador, solicitado := borradorRRHHSolicitado(r.Header); solicitado {
		h.responderBorrador(w, r, detalle, borrador)
		return
	}
	proyeccion := proyectarDetalleRRHH(detalle)
	if h.presentacion != nil {
		origen := domain.ReferenciaFlujo{DefinicionRef: detalle.Resumen.FlujoRef, Version: detalle.Resumen.FlujoVersion, HuellaSHA256: detalle.Resumen.FlujoHuella}
		if presentacion, err := h.presentacion.Resolver(r.Context(), origen, detalle.Resumen.FaseClave); err == nil && presentacion.Validar() == nil {
			proyeccion.PresentacionFlujo = proyectarPresentacionFlujoRRHH(presentacion)
		}
	}
	responderJSONConsultaRRHH(
		w,
		http.StatusOK,
		envoltorioDetalleRRHH{Data: proyeccion},
	)
}

func (h *manejadorConsultaDetalleRRHH) responderBorrador(
	w http.ResponseWriter,
	r *http.Request,
	detalle ports.DetalleExpedienteRRHH,
	borrador representacionBorradorRRHH,
) {
	tipoContenido := borrador.tipoContenido
	// Las pruebas y llamadas internas anteriores construyen la representación
	// PDF directamente; la ausencia de formato conserva esa semántica.
	if tipoContenido == "" {
		tipoContenido = "application/pdf"
	}
	if tipoContenido == MIMEDOCXBorradorRRHH && dependenciaConsultaRRHHNula(h.renderizadorDOCX) {
		responderErrorConsultaRRHH(w, errorServicioConsultaRRHHNoDisponible)
		return
	}
	if tipoContenido == "application/pdf" && dependenciaConsultaRRHHNula(h.renderizador) {
		responderErrorConsultaRRHH(w, errorServicioConsultaRRHHNoDisponible)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(err))
		return
	}
	// Los seis borradores siguen representando el original de propuesta v7.
	// La resolución v8 conserva la recuperación histórica ya publicada; la
	// anotación v9 exige la segunda fachada, que no es alcanzable por la lectura
	// ordinaria ni por una versión enviada de nuevo por el navegador.
	if requiereOriginalPropuestaRRHH(detalle) {
		solicitud, err := ports.NuevaSolicitudDetalleRRHH(detalle.Resumen.ExpedienteRef, 7)
		if err != nil {
			responderErrorConsultaRRHH(w, errorResultadoConsultaRRHHNoConfiable)
			return
		}
		consultorOriginal := h.consultor
		if detalle.Resumen.Version == 9 {
			if dependenciaConsultaRRHHNula(h.consultorOriginalPropuesta) {
				responderErrorConsultaRRHH(w, errorServicioConsultaRRHHNoDisponible)
				return
			}
			consultorOriginal = h.consultorOriginalPropuesta
		}
		original, err := consultorOriginal.Consultar(r.Context(), solicitud)
		if r.Context().Err() != nil {
			responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(r.Context().Err()))
			return
		}
		if err != nil || original.ValidarContenidoPublicablePara(solicitud) != nil {
			responderErrorConsultaRRHH(w, errorServicioConsultaRRHHNoDisponible)
			return
		}
		detalle = original
	}
	var contenido []byte
	var err error
	switch tipoContenido {
	case "application/pdf":
		contenido, err = h.renderizador.RenderizarBorrador(r.Context(), borrador.tipo, detalle.Clonar())
	case MIMEDOCXBorradorRRHH:
		contenido, err = h.renderizadorDOCX.RenderizarBorradorDOCX(r.Context(), borrador.tipo, detalle.Clonar())
	default:
		responderErrorConsultaRRHH(w, errorResultadoConsultaRRHHNoConfiable)
		return
	}
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(errContexto))
		return
	}
	if errors.Is(err, ports.ErrBorradorRRHHNoDisponible) {
		responderErrorConsultaRRHH(w, nuevoErrorConsultaRRHH(http.StatusConflict, "documento_no_disponible"))
		return
	}
	if err != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(err))
		return
	}
	if len(contenido) > MaximoPDFBorradorRRHHBytes ||
		(tipoContenido == "application/pdf" && !bytes.HasPrefix(contenido, []byte("%PDF-"))) ||
		(tipoContenido == MIMEDOCXBorradorRRHH && !bytes.HasPrefix(contenido, []byte("PK\x03\x04"))) {
		responderErrorConsultaRRHH(w, errorResultadoConsultaRRHHNoConfiable)
		return
	}
	aplicarCabecerasCobertura(w)
	w.Header().Set("Content-Type", tipoContenido)
	w.Header().Set("Content-Disposition", `attachment; filename="`+borrador.nombreArchivo+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(contenido)
}

func requiereOriginalPropuestaRRHH(detalle ports.DetalleExpedienteRRHH) bool {
	if detalle.Resumen.Version == 8 && len(detalle.Hitos) == 8 {
		return detalle.Hitos[7].AccionClave == "registrar_resolucion_formalizacion"
	}
	return detalle.Resumen.Version == 9 && len(detalle.Hitos) == 9 &&
		detalle.Hitos[7].AccionClave == "registrar_resolucion_formalizacion" &&
		detalle.Hitos[8].AccionClave == domain.AccionRegistrarAnotacionAdministrativa
}

func rutaConsultaRRHHExacta(r *http.Request, esperada string) bool {
	if r == nil || r.URL == nil || r.URL.RawQuery != "" || r.URL.ForceQuery ||
		r.URL.RawPath != "" || r.URL.Scheme != "" || r.URL.Host != "" ||
		r.URL.User != nil || r.URL.Opaque != "" || r.URL.Fragment != "" ||
		r.URL.RawFragment != "" || r.URL.Path != esperada {
		return false
	}
	return r.URL.EscapedPath() == esperada && !strings.Contains(r.URL.Path, "%")
}

func dependenciaConsultaRRHHNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
