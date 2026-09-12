package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const RutaSubsanacionReparos = "/api/vec/contratacion-temporal/subsanacion-reparos"

type ContextoCanalSubsanacionReparos struct{ AutenticacionRef, SesionRef, PerfilRef, OrganizacionRef string }

func (c ContextoCanalSubsanacionReparos) valido() bool {
	return (ports.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: c.AutenticacionRef, SesionRef: c.SesionRef, PerfilRef: c.PerfilRef}).Validar() == nil && domain.ReferenciaOpacaValida(c.OrganizacionRef)
}

type AutoridadContextoCanalSubsanacionReparos interface {
	ResolverContextoCanalSubsanacionReparos(context.Context) (ContextoCanalSubsanacionReparos, error)
}
type EjecutorSubsanacionReparos interface {
	RegistrarSubsanacionReparo(context.Context, application.SolicitudRegistrarSubsanacionReparo) (ports.ReciboSubsanacionReparo, error)
}
type manejadorSubsanacionReparos struct {
	autoridad AutoridadContextoCanalSubsanacionReparos
	ejecutor  EjecutorSubsanacionReparos
}

func NuevoManejadorSubsanacionReparos(a AutoridadContextoCanalSubsanacionReparos, e EjecutorSubsanacionReparos) (http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(e) {
		return nil, errors.New("contratacion temporal http: subsanacion de reparos no disponible")
	}
	return &manejadorSubsanacionReparos{autoridad: a, ejecutor: e}, nil
}

func (h *manejadorSubsanacionReparos) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil || r.URL.Path != RutaSubsanacionReparos || r.URL.RawQuery != "" || r.URL.ForceQuery {
		responderErrorSubsanacionReparos(w, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodPost {
		responderErrorSubsanacionReparos(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if !tipoContenidoJSON(r.Header) || cabeceraCoberturaProhibida(r.Header) {
		responderErrorSubsanacionReparos(w, http.StatusBadRequest, "peticion_no_permitida")
		return
	}
	contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 16*1024+1))
	if err != nil || len(contenido) == 0 || len(contenido) > 16*1024 {
		responderErrorSubsanacionReparos(w, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	var in struct {
		ExpedienteRef     string `json:"expediente_ref"`
		VersionEsperada   uint64 `json:"version_esperada"`
		ClaveIdempotencia string `json:"clave_idempotencia"`
		Observaciones     string `json:"observaciones"`
	}
	if !jsonSubsanacionReparosCerrado(contenido) {
		responderErrorSubsanacionReparos(w, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if dec.Decode(&in) != nil || dec.Decode(&struct{}{}) != io.EOF || !domain.ReferenciaOpacaValida(in.ExpedienteRef) || !ports.VersionOperacionAnalisisConIncrementoValida(in.VersionEsperada) || !ports.ClaveIdempotenciaValida(in.ClaveIdempotencia) || (domain.DatosSubsanacionReparo{RetornoRef: "retorno:pendiente", Observaciones: in.Observaciones}).Validar() != nil {
		responderErrorSubsanacionReparos(w, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	c, err := h.autoridad.ResolverContextoCanalSubsanacionReparos(r.Context())
	if err != nil || !c.valido() {
		responderErrorSubsanacionReparos(w, http.StatusForbidden, "acceso_denegado")
		return
	}
	recibo, err := h.ejecutor.RegistrarSubsanacionReparo(r.Context(), application.SolicitudRegistrarSubsanacionReparo{AutenticacionRef: c.AutenticacionRef, SesionRef: c.SesionRef, PerfilRef: c.PerfilRef, OrganizacionRef: c.OrganizacionRef, ExpedienteRef: in.ExpedienteRef, VersionEsperada: in.VersionEsperada, ClaveIdempotencia: in.ClaveIdempotencia, Observaciones: in.Observaciones})
	if err != nil {
		if errors.Is(err, application.ErrSubsanacionReparoDenegada) || errors.Is(err, ports.ErrAutorizacionDenegada) {
			responderErrorSubsanacionReparos(w, http.StatusForbidden, "acceso_denegado")
		} else if errors.Is(err, domain.ErrVersionEnConflicto) || errors.Is(err, ports.ErrClaveIdempotenciaUsada) {
			responderErrorSubsanacionReparos(w, http.StatusConflict, "conflicto")
		} else {
			responderErrorSubsanacionReparos(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		}
		return
	}
	if recibo.Operacion != ports.OperacionRegistrarSubsanacionReparo || recibo.OrganizacionRef != c.OrganizacionRef || recibo.ExpedienteRef != in.ExpedienteRef || recibo.VersionAnterior != in.VersionEsperada || recibo.VersionResultante != in.VersionEsperada+1 || recibo.FaseResultante != domain.FaseSubsanacionUnidad || recibo.EstadoResultante != domain.EstadoIncidencia || !domain.ReferenciaOpacaValida(recibo.ReciboRef) || !domain.ReferenciaOpacaValida(recibo.AuditoriaRef) || !domain.ReferenciaOpacaValida(recibo.EventoRef) || !domain.ReferenciaOpacaValida(recibo.ActorRef) || !domain.InstanteUTCCanonico(recibo.RegistradaEn) {
		responderErrorSubsanacionReparos(w, http.StatusBadGateway, "resultado_no_confiable")
		return
	}
	responderJSONCobertura(w, http.StatusCreated, envoltorioReciboSubsanacionReparos{Data: reciboSubsanacionReparosJSON{
		Esquema: "vec.contratacion-temporal.recibo-subsanacion-reparos.v1", Operacion: recibo.Operacion,
		ExpedienteRef: recibo.ExpedienteRef, VersionResultante: recibo.VersionResultante,
		FaseResultante: string(recibo.FaseResultante), EstadoResultante: string(recibo.EstadoResultante),
		ReciboRef: recibo.ReciboRef, AuditoriaRef: recibo.AuditoriaRef, EventoRef: recibo.EventoRef,
		ActorRef: recibo.ActorRef, RegistradaEn: recibo.RegistradaEn.UTC().Format(time.RFC3339Nano),
	}})
}

type envoltorioReciboSubsanacionReparos struct {
	Data reciboSubsanacionReparosJSON `json:"data"`
}
type reciboSubsanacionReparosJSON struct {
	Esquema           string `json:"esquema"`
	Operacion         string `json:"operacion"`
	ExpedienteRef     string `json:"expediente_ref"`
	VersionResultante uint64 `json:"version_resultante"`
	FaseResultante    string `json:"fase_resultante"`
	EstadoResultante  string `json:"estado_resultante"`
	ReciboRef         string `json:"recibo_ref"`
	AuditoriaRef      string `json:"auditoria_ref"`
	EventoRef         string `json:"evento_ref"`
	ActorRef          string `json:"actor_ref"`
	RegistradaEn      string `json:"registrada_en"`
}

func jsonSubsanacionReparosCerrado(contenido []byte) bool {
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.UseNumber()
	inicio, err := dec.Token()
	if err != nil || inicio != json.Delim('{') {
		return false
	}
	vistas := make(map[string]struct{}, 4)
	for dec.More() {
		clave, err := dec.Token()
		texto, ok := clave.(string)
		if err != nil || !ok {
			return false
		}
		switch texto {
		case "expediente_ref", "version_esperada", "clave_idempotencia", "observaciones":
		default:
			return false
		}
		if _, repetida := vistas[texto]; repetida {
			return false
		}
		vistas[texto] = struct{}{}
		valor, err := dec.Token()
		if err != nil {
			return false
		}
		if _, compuesto := valor.(json.Delim); compuesto {
			return false
		}
	}
	fin, err := dec.Token()
	if err != nil || fin != json.Delim('}') || len(vistas) != 4 {
		return false
	}
	_, err = dec.Token()
	return err == io.EOF
}

func responderErrorSubsanacionReparos(w http.ResponseWriter, estado int, codigo string) {
	responderJSONCobertura(w, estado, envoltorioErrorCobertura{Error: detalleErrorCobertura{Codigo: codigo, ClaveI18n: "api.contratacion_temporal.subsanacion_reparos.error." + codigo, CorrelacionRef: nuevaCorrelacionCobertura()}})
}
