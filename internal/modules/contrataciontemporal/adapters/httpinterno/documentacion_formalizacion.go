package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Documentación y plazos de la formalización tras aceptar una oferta:
// GET /api/vec/contratacion-temporal/formalizacion/documentacion?expediente_ref=<ref>&aceptada_en=<RFC 3339>[&modalidad=<clave>]
// Cálculo de solo lectura sobre el catálogo de reglas y Calendarios. No lleva
// datos personales ni registra efectos; el estado de cada documento lo
// conserva Documentos.
const (
	RutaDocumentacionFormalizacion    = "/api/vec/contratacion-temporal/formalizacion/documentacion"
	EsquemaDocumentacionFormalizacion = "vec.ct.formalizacion.documentacion.v1"
)

var (
	ErrManejadorDocumentacionFormalizacionInvalido = errors.New("contratacion temporal http: manejador de documentacion de formalizacion invalido")
	errorReglasFormalizacionNoConfiguradas         = nuevoErrorConsultaRRHH(http.StatusServiceUnavailable, "reglas_no_configuradas")
	errConsultaDocumentacionFormalizacion          = errors.New("contratacion temporal http: consulta de documentacion de formalizacion no valida")
)

// ConsultorDocumentacionFormalizacion es el caso de uso de aplicación.
type ConsultorDocumentacionFormalizacion interface {
	Consultar(context.Context, application.SolicitudDocumentacionFormalizacion) (application.DocumentacionFormalizacion, error)
}

type manejadorDocumentacionFormalizacion struct {
	consultor ConsultorDocumentacionFormalizacion
}

func NuevoManejadorDocumentacionFormalizacion(consultor ConsultorDocumentacionFormalizacion) (http.Handler, error) {
	if dependenciaConsultaRRHHNula(consultor) {
		return nil, ErrManejadorDocumentacionFormalizacionInvalido
	}
	return &manejadorDocumentacionFormalizacion{consultor: consultor}, nil
}

type reglaFormalizacionJSON struct {
	Clave          string `json:"clave"`
	Unidad         string `json:"unidad"`
	Cantidad       int    `json:"cantidad,omitempty"`
	Computo        string `json:"computo,omitempty"`
	Origen         string `json:"origen"`
	Ejemplo        bool   `json:"ejemplo"`
	Articulo       string `json:"articulo,omitempty"`
	Norma          string `json:"norma"`
	Duda           string `json:"duda"`
	ParteEjemplo   string `json:"parte_ejemplo,omitempty"`
	Referencia     string `json:"referencia"`
	HuellaCatalogo string `json:"huella_catalogo"`
}

type plazoFormalizacionJSON struct {
	UltimoDia    string                 `json:"ultimo_dia"`
	VenceAntesDe string                 `json:"vence_antes_de"`
	Prorrogado   bool                   `json:"prorrogado"`
	Estado       string                 `json:"estado"`
	Regla        reglaFormalizacionJSON `json:"regla"`
}

type documentoFormalizacionJSON struct {
	Clave          string `json:"clave"`
	TipoDocumental string `json:"tipo_documental"`
	Registrable    bool   `json:"registrable"`
}

type documentacionFormalizacionJSON struct {
	Esquema                 string                       `json:"esquema"`
	ExpedienteDocumentalRef string                       `json:"expediente_documental_ref"`
	AceptadaEn              string                       `json:"aceptada_en"`
	Modalidad               string                       `json:"modalidad,omitempty"`
	PorModalidad            bool                         `json:"por_modalidad"`
	Documentos              []documentoFormalizacionJSON `json:"documentos"`
	ReglaDocumentos         reglaFormalizacionJSON       `json:"regla_documentos"`
	PlazoDocumentacion      plazoFormalizacionJSON       `json:"plazo_documentacion"`
	PlazoIncorporacion      *plazoFormalizacionJSON      `json:"plazo_incorporacion,omitempty"`
	ConsultadaEn            string                       `json:"consultada_en"`
}

func (h *manejadorDocumentacionFormalizacion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || dependenciaConsultaRRHHNula(h.consultor) {
		responderErrorConsultaRRHH(w, r, nil, errorServicioConsultaRRHHNoDisponible)
		return
	}
	if !rutaDocumentacionFormalizacionExacta(r) {
		responderErrorConsultaRRHH(w, r, nil, errorRecursoConsultaRRHHNoEncontrado)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		responderErrorConsultaRRHH(w, r, nil, errorMetodoConsultaRRHHNoPermitido)
		return
	}
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
		responderErrorConsultaRRHH(w, r, nil, errorPeticionConsultaRRHHNoValida)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorConsultaRRHH(w, r, err, clasificarErrorConsultaRRHH(err))
		return
	}
	solicitud, err := solicitudDocumentacionFormalizacionDesdeQuery(r.URL.RawQuery)
	if err != nil {
		responderErrorConsultaRRHH(w, r, err, errorPeticionConsultaRRHHNoValida)
		return
	}
	resultado, err := h.consultor.Consultar(r.Context(), solicitud)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorConsultaRRHH(w, r, errContexto, clasificarErrorConsultaRRHH(errContexto))
		return
	}
	switch {
	case err == nil:
	case errors.Is(err, ports.ErrSolicitudDocumentacionFormalizacion):
		responderErrorConsultaRRHH(w, r, err, errorContenidoConsultaRRHHNoValido)
		return
	case errors.Is(err, ports.ErrReglasFormalizacionNoConfiguradas):
		responderErrorConsultaRRHH(w, r, err, errorReglasFormalizacionNoConfiguradas)
		return
	default:
		responderErrorConsultaRRHH(w, r, err, errorServicioConsultaRRHHNoDisponible)
		return
	}
	responderJSONConsultaRRHH(w, r, http.StatusOK, struct {
		Data documentacionFormalizacionJSON `json:"data"`
	}{proyectarDocumentacionFormalizacion(resultado)})
}

func proyectarDocumentacionFormalizacion(d application.DocumentacionFormalizacion) documentacionFormalizacionJSON {
	documentos := make([]documentoFormalizacionJSON, 0, len(d.Documentos))
	for _, documento := range d.Documentos {
		documentos = append(documentos, documentoFormalizacionJSON{Clave: documento.Clave, TipoDocumental: documento.TipoDocumental, Registrable: documento.Registrable})
	}
	salida := documentacionFormalizacionJSON{
		Esquema: EsquemaDocumentacionFormalizacion, ExpedienteDocumentalRef: d.ExpedienteDocumentalRef, AceptadaEn: d.AceptadaEn.UTC().Format(time.RFC3339Nano),
		Modalidad: d.Modalidad, PorModalidad: d.PorModalidad, Documentos: documentos,
		ReglaDocumentos: proyectarReglaFormalizacion(d.ReglaDocumentos), PlazoDocumentacion: proyectarPlazoFormalizacion(d.PlazoDocumentacion),
		ConsultadaEn: d.ConsultadaEn.UTC().Format(time.RFC3339Nano),
	}
	if d.PlazoIncorporacion != nil {
		plazo := proyectarPlazoFormalizacion(*d.PlazoIncorporacion)
		salida.PlazoIncorporacion = &plazo
	}
	return salida
}

func proyectarPlazoFormalizacion(p application.PlazoFormalizacion) plazoFormalizacionJSON {
	return plazoFormalizacionJSON{
		UltimoDia: p.Vencimiento.UltimoDia, VenceAntesDe: p.Vencimiento.VenceAntesDe.UTC().Format(time.RFC3339Nano),
		Prorrogado: p.Vencimiento.Prorrogado, Estado: string(p.Estado), Regla: proyectarReglaFormalizacion(p.Regla),
	}
}

func proyectarReglaFormalizacion(r ports.ReglaFormalizacion) reglaFormalizacionJSON {
	return reglaFormalizacionJSON{
		Clave: r.Clave, Unidad: r.Unidad, Cantidad: r.Cantidad, Computo: r.Computo, Origen: r.Origen, Ejemplo: r.Ejemplo,
		Articulo: r.Articulo, Norma: r.Norma, Duda: r.Duda, ParteEjemplo: r.ParteEjemplo,
		Referencia: r.Referencia, HuellaCatalogo: r.HuellaCatalogo,
	}
}

// solicitudDocumentacionFormalizacionDesdeQuery exige expediente_ref y
// aceptada_en una vez y admite modalidad como mucho una vez; cualquier otro
// parámetro se rechaza.
func solicitudDocumentacionFormalizacionDesdeQuery(rawQuery string) (application.SolicitudDocumentacionFormalizacion, error) {
	var vacia application.SolicitudDocumentacionFormalizacion
	if len(rawQuery) > 512 || strings.Contains(rawQuery, ";") {
		return vacia, errConsultaDocumentacionFormalizacion
	}
	valores, err := url.ParseQuery(rawQuery)
	if err != nil {
		return vacia, errors.Join(errConsultaDocumentacionFormalizacion, err)
	}
	for clave, lista := range valores {
		if (clave != "expediente_ref" && clave != "aceptada_en" && clave != "modalidad") || len(lista) != 1 || lista[0] == "" {
			return vacia, errConsultaDocumentacionFormalizacion
		}
	}
	aceptada, err := time.Parse(time.RFC3339Nano, valores.Get("aceptada_en"))
	if err != nil {
		return vacia, errors.Join(errConsultaDocumentacionFormalizacion, err)
	}
	if valores.Get("expediente_ref") == "" {
		return vacia, errConsultaDocumentacionFormalizacion
	}
	return application.SolicitudDocumentacionFormalizacion{
		ExpedienteRef: valores.Get("expediente_ref"), AceptadaEn: aceptada, Modalidad: valores.Get("modalidad"),
	}, nil
}

func rutaDocumentacionFormalizacionExacta(r *http.Request) bool {
	if r == nil || r.URL == nil || r.URL.ForceQuery || r.URL.RawPath != "" || r.URL.Scheme != "" ||
		r.URL.Host != "" || r.URL.User != nil || r.URL.Opaque != "" || r.URL.Fragment != "" ||
		r.URL.RawFragment != "" || r.URL.Path != RutaDocumentacionFormalizacion {
		return false
	}
	return r.URL.EscapedPath() == RutaDocumentacionFormalizacion
}
