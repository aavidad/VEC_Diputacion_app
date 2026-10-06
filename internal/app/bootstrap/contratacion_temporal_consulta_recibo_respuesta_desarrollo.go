package bootstrap

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type claveConsultaReciboRespuestaDesarrollo struct{}

type proveedorConsultaReciboRespuestaDesarrollo struct {
	soporte     *soporteAltaContratacionTemporalDesarrollo
	autorizador autorizacionComunicacionLlamamientoDesarrollo
	reloj       ports.Reloj
}

func nuevoManejadorConsultaReciboRespuestaDesarrollo(
	alta *dependenciasAltaContratacionTemporalDesarrollo, reloj ports.Reloj,
) (http.Handler, error) {
	if alta == nil || alta.soporte == nil || alta.postgresql.ejecucion == nil ||
		alta.postgresql.proveedorMaterial == nil || dependenciaEsNulaContratacionTemporalDesarrollo(reloj) {
		return nil, ports.ErrConsultaReciboRespuestaFallo
	}
	proveedor := &proveedorConsultaReciboRespuestaDesarrollo{
		soporte: alta.soporte, reloj: reloj,
		autorizador: &autorizadorLlamamientoDesarrollo{alta: alta, material: alta.postgresql.proveedorMaterial, consultaReciboRespuesta: true},
	}
	lector, err := postgresct.NuevoLectorReciboRespuestaPostgreSQL(alta.postgresql.ejecucion, proveedor)
	if err != nil {
		return nil, err
	}
	lectorAuditado, err := nuevoLectorReciboRespuestaAuditadoCT(lector, alta.soporte, alta.auditoriaLecturasCT,
		configuracionAuditoriaLecturasCT{Proceso: alta.procesoAuditoriaLecturasCT, Canal: string(dominiovec.SuperficieAutenticacionInternaCorporativaV1)})
	if err != nil {
		return nil, err
	}
	servicio, err := application.NuevoServicioConsultaReciboRespuesta(lectorAuditado)
	if err != nil {
		return nil, err
	}
	return httpinterno.NuevoManejadorConsultaReciboRespuesta(servicio)
}

func (p *proveedorConsultaReciboRespuestaDesarrollo) AutorizarConsultaReciboRespuesta(
	ctx context.Context, s ports.SolicitudConsultaReciboRespuesta,
) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || contextoInterfazNulo(ctx) || p.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.autorizador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.reloj) || s.Validar() != nil {
		return vacio, ports.ErrConsultaReciboRespuestaDenegada
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	capacidad, valida := p.soporte.capacidadValida(ctx)
	if !valida || capacidad.ruta != httpinterno.RutaConsultaReciboRespuesta ||
		s.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		!certificadoConsultaReciboRespuestaVigente(capacidad, p.reloj.Ahora()) {
		return vacio, ports.ErrConsultaReciboRespuestaDenegada
	}
	if _, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(p.reloj.Ahora()); !vigente {
		return vacio, ports.ErrConsultaReciboRespuestaDenegada
	}
	recurso, err := postgresct.RecursoConsultaReciboRespuesta(s)
	if err != nil {
		return vacio, ports.ErrConsultaReciboRespuestaDenegada
	}
	ctx = context.WithValue(ctx, claveConsultaReciboRespuestaDesarrollo{}, s)
	a, err := p.autorizador.AutorizarOperacion(ctx, postgresct.AccionConsultaReciboRespuesta, recurso)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil {
		return vacio, errorAutorizacionConsultaReciboRespuesta(err)
	}
	if a.ValidarEstructura() != nil {
		return vacio, ports.ErrConsultaReciboRespuestaFallo
	}
	r, ahora := a.ResumenCapacidad(), p.reloj.Ahora()
	if ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) || r.ExpiraEn().Sub(r.EmitidaEn()) > 5*time.Minute {
		return vacio, ports.ErrConsultaReciboRespuestaDenegada
	}
	return a, nil
}

func errorAutorizacionConsultaReciboRespuesta(err error) error {
	for _, dependencia := range []error{
		ports.ErrConsultaRRHHNoDisponible, ports.ErrPersistenciaNoDisponible, puertosvec.ErrFuenteAutorizacionNoDisponible,
		puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible,
		puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible,
		dominiovec.ErrConfiguracionAccesoInvalida,
	} {
		if errors.Is(err, dependencia) {
			return ports.ErrConsultaReciboRespuestaFallo
		}
	}
	if errors.Is(err, ports.ErrAutorizacionDenegada) || errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		return ports.ErrConsultaReciboRespuestaDenegada
	}
	return ports.ErrConsultaReciboRespuestaFallo
}

func certificadoConsultaReciboRespuestaVigente(c capacidadConsultaContratacionTemporalDesarrollo, ahora time.Time) bool {
	return !c.certificadoVerificadoEn.IsZero() && !c.certificadoValidoHasta.IsZero() &&
		!ahora.Before(c.certificadoVerificadoEn) && ahora.Before(c.certificadoValidoHasta)
}

// La auditoría de frontera se escribe fuera de la transacción del lector. Una
// denegación SQL revierte esa transacción; aquí se conserva el fallo HTTP final.
type auditorConsultaCTDenegada struct {
	siguiente   http.Handler
	registrador puertosvec.RegistradorAuditoriaFronteraRutaExacta
	soporte     *soporteAltaContratacionTemporalDesarrollo
	ruta        string
}

type auditorConsultaReciboRespuestaDenegada = auditorConsultaCTDenegada

func (a auditorConsultaCTDenegada) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if a.siguiente == nil || dependenciaEsNulaContratacionTemporalDesarrollo(a.registrador) {
		responderConsultaReciboRespuestaNoDisponible(w)
		return
	}
	ruta := a.ruta
	if ruta == "" {
		ruta = httpinterno.RutaConsultaReciboRespuesta
	}
	if ruta != httpinterno.RutaConsultaReciboRespuesta && ruta != httpinterno.RutaConsultaComunicacionesExpediente {
		responderConsultaReciboRespuestaNoDisponible(w)
		return
	}
	respuesta := &respuestaConsultaReciboDiferida{cabeceras: make(http.Header)}
	a.siguiente.ServeHTTP(respuesta, r)
	estado := respuesta.estado
	if estado == 0 {
		estado = http.StatusOK
	}
	if estado == http.StatusForbidden || estado == http.StatusUnauthorized {
		var cuerpo struct {
			Error struct {
				CorrelacionRef string `json:"correlacion_ref"`
			} `json:"error"`
		}
		if json.Unmarshal(respuesta.cuerpo.Bytes(), &cuerpo) != nil {
			responderConsultaReciboRespuestaNoDisponible(w)
			return
		}
		motivo := puertosvec.MotivoAuditoriaFronteraRutaExactaAccesoDenegado
		if estado == http.StatusUnauthorized {
			motivo = puertosvec.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
		}
		orden := puertosvec.OrdenAuditoriaFronteraRutaExacta{
			CorrelacionRef: cuerpo.Error.CorrelacionRef,
			Motivo:         motivo,
			Superficie:     puertosvec.SuperficieAuditoriaFronteraRutaExactaContratacionTemporal,
			Ruta:           ruta,
		}
		if estado == http.StatusForbidden && a.soporte != nil {
			if capacidad, valida := a.soporte.capacidadValida(r.Context()); valida && capacidad.ruta == ruta {
				orden.ActorRef = capacidad.principal.ID
			}
		}
		if orden.Validar() != nil {
			responderConsultaReciboRespuestaNoDisponible(w)
			return
		}
		ctx, cancelar := context.WithTimeout(context.WithoutCancel(r.Context()), plazoarranque.Ampliar(250*time.Millisecond))
		err := a.registrador.RegistrarAuditoriaFronteraRutaExacta(ctx, orden)
		cancelar()
		if err != nil {
			responderConsultaReciboRespuestaNoDisponible(w)
			return
		}
	}
	for clave, valores := range respuesta.cabeceras {
		for _, valor := range valores {
			w.Header().Add(clave, valor)
		}
	}
	w.WriteHeader(estado)
	_, _ = w.Write(respuesta.cuerpo.Bytes())
}

func responderConsultaReciboRespuestaNoDisponible(w http.ResponseWriter) {
	contenido, _ := json.Marshal(map[string]any{"error": map[string]string{
		"codigo":          "servicio_no_disponible",
		"clave_i18n":      "api.contratacion_temporal.respuesta_recibida.error.servicio_no_disponible",
		"correlacion_ref": nuevaCorrelacionConsultaReciboRespuesta(),
	}})
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write(contenido)
}

type respuestaConsultaReciboDiferida struct {
	cabeceras http.Header
	cuerpo    bytes.Buffer
	estado    int
}

func (w *respuestaConsultaReciboDiferida) Header() http.Header { return w.cabeceras }
func (w *respuestaConsultaReciboDiferida) WriteHeader(estado int) {
	if w.estado == 0 {
		w.estado = estado
	}
}
func (w *respuestaConsultaReciboDiferida) Write(b []byte) (int, error) {
	if w.estado == 0 {
		w.estado = http.StatusOK
	}
	return w.cuerpo.Write(b)
}

var _ postgresct.ProveedorConsultaReciboRespuesta = (*proveedorConsultaReciboRespuestaDesarrollo)(nil)

func nuevaCorrelacionConsultaReciboRespuesta() string {
	aleatorio := make([]byte, 16)
	if _, err := rand.Read(aleatorio); err != nil {
		slog.Error("fallo al generar correlacion de consulta de recibo CT", "causa", err)
		return "corr_no_disponible"
	}
	return "corr_" + hex.EncodeToString(aleatorio)
}
