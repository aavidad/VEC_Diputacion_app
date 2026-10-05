package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const RutaConsultaCircuitoRRHH = "/api/vec/contratacion-temporal/circuito/consulta"

type ContextoCanalCircuitoRRHH struct {
	AutenticacionRef string
	SesionRef        string
	PerfilRef        string
	OrganizacionRef  string
}

func (c ContextoCanalCircuitoRRHH) valido() bool {
	return (ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: c.AutenticacionRef,
		SesionRef:        c.SesionRef,
		PerfilRef:        c.PerfilRef,
	}).Validar() == nil && domain.ReferenciaOpacaValida(c.OrganizacionRef)
}

type AutoridadContextoCanalCircuitoRRHH interface {
	ResolverContextoCanalCircuitoRRHH(context.Context) (ContextoCanalCircuitoRRHH, error)
}

type ConsultorCircuitoRRHH interface {
	Consultar(context.Context, ports.SolicitudConsultaCircuitoRRHH) (ports.ResultadoConsultaCircuitoRRHH, error)
}

type manejadorConsultaCircuitoRRHH struct {
	autoridad AutoridadContextoCanalCircuitoRRHH
	consultor ConsultorCircuitoRRHH
}

func NuevoManejadorConsultaCircuitoRRHH(autoridad AutoridadContextoCanalCircuitoRRHH, consultor ConsultorCircuitoRRHH) (http.Handler, error) {
	if dependenciaNula(autoridad) || dependenciaNula(consultor) {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	return &manejadorConsultaCircuitoRRHH{autoridad: autoridad, consultor: consultor}, nil
}

type entradaConsultaCircuitoRRHH struct {
	ExpedienteRef    string  `json:"expediente_ref"`
	VersionObservada *uint64 `json:"version_observada"`
}

type hitoCircuitoRRHHJSON struct {
	Secuencia    uint64 `json:"secuencia"`
	Clave        string `json:"clave"`
	Tipo         string `json:"tipo"`
	Origen       string `json:"origen"`
	Destino      string `json:"destino"`
	RegistradoEn string `json:"registrado_en"`
	ReciboRef    string `json:"recibo_ref"`
}

type transicionCircuitoRRHHJSON struct {
	Clave             string `json:"clave"`
	Tipo              string `json:"tipo"`
	PerfilClave       string `json:"perfil_clave"`
	RequiereDocumento bool   `json:"requiere_documento"`
	RequiereFirma     bool   `json:"requiere_firma"`
}

type circuitoRRHHJSON struct {
	Definicion   domain.ReferenciaFlujo `json:"definicion"`
	EstadoActual string                 `json:"estado_actual"`
	Hitos        []hitoCircuitoRRHHJSON `json:"hitos"`
}

type resultadoConsultaCircuitoRRHHJSON struct {
	Flujo                  domain.ReferenciaFlujo       `json:"flujo"`
	Circuito               circuitoRRHHJSON             `json:"circuito"`
	VersionExpediente      uint64                       `json:"version_expediente"`
	TransicionesPermitidas []transicionCircuitoRRHHJSON `json:"transiciones_permitidas"`
}

func (h *manejadorConsultaCircuitoRRHH) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || dependenciaNula(h.autoridad) || dependenciaNula(h.consultor) {
		responderErrorConsultaRRHH(w, r, nil, errorServicioConsultaRRHHNoDisponible)
		return
	}
	if !rutaConsultaRRHHExacta(r, RutaConsultaCircuitoRRHH) {
		responderErrorConsultaRRHH(w, r, nil, errorRecursoConsultaRRHHNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorConsultaRRHH(w, r, nil, errorMetodoConsultaRRHHNoPermitido)
		return
	}
	if problema := validarMetadatosConsultaRRHH(r, MaximoCuerpoConsultaDetalleRRHHBytes); problema != nil {
		responderErrorConsultaRRHH(w, r, nil, *problema)
		return
	}
	var entrada entradaConsultaCircuitoRRHH
	if err := decodificarConsultaRRHH(w, r, MaximoCuerpoConsultaDetalleRRHHBytes, &entrada); err != nil {
		responderErrorConsultaRRHH(w, r, nil, errorEntradaConsultaRRHH(err))
		return
	}
	if entrada.VersionObservada == nil || !domain.ReferenciaOpacaValida(entrada.ExpedienteRef) || *entrada.VersionObservada == 0 {
		responderErrorConsultaRRHH(w, r, nil, errorContenidoConsultaRRHHNoValido)
		return
	}
	canal, err := h.autoridad.ResolverContextoCanalCircuitoRRHH(r.Context())
	if err != nil || !canal.valido() {
		responderErrorConsultaRRHH(w, r, err, errorRecursoConsultaRRHHNoEncontrado)
		return
	}
	solicitud := ports.SolicitudConsultaCircuitoRRHH{
		AutenticacionRef: canal.AutenticacionRef, SesionRef: canal.SesionRef,
		PerfilRef: canal.PerfilRef, OrganizacionRef: canal.OrganizacionRef,
		ExpedienteRef: entrada.ExpedienteRef, VersionObservada: *entrada.VersionObservada,
	}
	if solicitud.Validar() != nil {
		responderErrorConsultaRRHH(w, r, nil, errorContenidoConsultaRRHHNoValido)
		return
	}
	resultado, err := h.consultor.Consultar(r.Context(), solicitud)
	if err != nil {
		responderErrorConsultaRRHH(w, r, err, errorConsultaCircuitoRRHH(err))
		return
	}
	if resultado.ValidarPara(solicitud) != nil {
		responderErrorConsultaRRHH(w, r, nil, errorResultadoConsultaRRHHNoConfiable)
		return
	}
	responderJSONConsultaRRHH(w, r, http.StatusOK, struct {
		Data resultadoConsultaCircuitoRRHHJSON `json:"data"`
	}{Data: proyectarConsultaCircuitoRRHH(resultado)})
}

func errorConsultaCircuitoRRHH(err error) errorPublicoConsultaRRHH {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return clasificarErrorConsultaRRHH(err)
	case errors.Is(err, domain.ErrVersionEnConflicto):
		return errorPublicoConsultaRRHH{estado: http.StatusConflict, codigo: "version_en_conflicto", claveI18n: "api.contratacion_temporal.consulta_rrhh.error.version_en_conflicto"}
	case errors.Is(err, ports.ErrAutorizacionDenegada), errors.Is(err, ports.ErrConsultaCircuitoRRHHDenegada):
		return errorRecursoConsultaRRHHNoEncontrado
	case errors.Is(err, ports.ErrConsultaCircuitoRRHHInvalida):
		return errorContenidoConsultaRRHHNoValido
	case errors.Is(err, ports.ErrResultadoCircuitoRRHHNoConfiable):
		return errorResultadoConsultaRRHHNoConfiable
	default:
		return errorServicioConsultaRRHHNoDisponible
	}
}

func proyectarConsultaCircuitoRRHH(resultado ports.ResultadoConsultaCircuitoRRHH) resultadoConsultaCircuitoRRHHJSON {
	salida := resultadoConsultaCircuitoRRHHJSON{
		Flujo: resultado.Flujo, VersionExpediente: resultado.VersionExpediente,
		Circuito: circuitoRRHHJSON{
			Definicion:   resultado.Circuito.Definicion,
			EstadoActual: string(resultado.Circuito.EstadoActual),
			Hitos:        make([]hitoCircuitoRRHHJSON, len(resultado.Circuito.Hitos)),
		},
		TransicionesPermitidas: make([]transicionCircuitoRRHHJSON, len(resultado.TransicionesPermitidas)),
	}
	for i, hito := range resultado.Circuito.Hitos {
		salida.Circuito.Hitos[i] = hitoCircuitoRRHHJSON{
			Secuencia: hito.Secuencia, Clave: string(hito.Clave), Tipo: string(hito.Tipo),
			Origen: string(hito.Origen), Destino: string(hito.Destino),
			RegistradoEn: hito.RegistradoEn.UTC().Format(time.RFC3339Nano), ReciboRef: hito.ReciboRef,
		}
	}
	for i, transicion := range resultado.TransicionesPermitidas {
		salida.TransicionesPermitidas[i] = transicionCircuitoRRHHJSON{
			Clave: string(transicion.Clave), Tipo: string(transicion.Tipo),
			PerfilClave:       string(transicion.PerfilClave),
			RequiereDocumento: transicion.RequiereDocumento, RequiereFirma: transicion.RequiereFirma,
		}
	}
	return salida
}
