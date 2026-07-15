// Package memory contiene adaptadores efimeros y defensivos del modulo de
// bolsas. Solo son apropiados para pruebas: no sustituyen una transaccion
// durable ni una outbox persistente.
package memory

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	transaccionbolsa "vec-diputacion-granada/internal/modules/bolsa/internal/transaccion"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

var _ puertosbolsa.RepositorioBaremaciones = (*RepositorioBaremaciones)(nil)

type estadoReserva string

const (
	estadoReservaActiva                 estadoReserva = "activa"
	estadoReservaConfirmada             estadoReserva = "confirmada"
	estadoReservaAbandonada             estadoReserva = "abandonada"
	estadoReservaExpirada               estadoReserva = "expirada"
	estadoReservaInvalidada             estadoReserva = "invalidada"
	moduloBaremacion                                  = "bolsa"
	maximoReservasMemoria                             = 4_096
	maximoBaremacionesMemoria                         = 4_096
	maximoVersionesPorBaremacionMemoria               = 4_097
	maximoTransaccionesMemoria                        = 65_536
	maximoUsosAutorizacionMemoria                     = 131_072
)

type accionAuditoriaBaremacion = puertosbolsa.AccionAuditoriaBaremacion
type tipoEventoOutboxBaremacion = puertosbolsa.TipoEventoOutboxBaremacion
type estadoEventoOutboxBaremacion = puertosbolsa.EstadoEventoOutboxBaremacion
type registroAuditoriaBaremacion = puertosbolsa.RegistroAuditoriaBaremacion
type eventoOutboxBaremacion = puertosbolsa.EventoOutboxBaremacion

const (
	accionCrearBaremacion       accionAuditoriaBaremacion    = puertosbolsa.AccionAuditoriaCrearBaremacion
	accionIncorporarDecision    accionAuditoriaBaremacion    = puertosbolsa.AccionAuditoriaIncorporarDecision
	eventoBaremacionCreada      tipoEventoOutboxBaremacion   = puertosbolsa.TipoEventoBaremacionCreada
	eventoDecisionIncorporada   tipoEventoOutboxBaremacion   = puertosbolsa.TipoEventoDecisionIncorporada
	estadoEventoOutboxPendiente estadoEventoOutboxBaremacion = puertosbolsa.EstadoEventoOutboxBaremacionPendiente
)

type reservaBaremacion struct {
	SolicitudReserva         puertosbolsa.SolicitudReservarCambioBaremacion
	HuellaTokenSHA256        string
	Estado                   estadoReserva
	HuellaConfirmacionSHA256 string
	Resultado                *puertosbolsa.ResultadoConfirmarCambioBaremacion
}

// usoAutorizacionBaremacion simula en el adaptador de pruebas el indice unico
// que un repositorio duradero debe confirmar junto con el efecto. No sustituye
// la relectura autoritativa de PDP, sesion, asignacion, rol y catalogo exigida
// por el puerto productivo.
type usoAutorizacionBaremacion struct {
	DecisionRef          string
	HuellaDecisionSHA256 string
	HuellaEfectoSHA256   string
}

// RepositorioBaremaciones conserva versiones, tombstones de idempotencia,
// auditoria y outbox bajo el mismo mutex. Una clave o token abandonados,
// expirados o invalidados nunca vuelven a habilitarse.
type RepositorioBaremaciones struct {
	mu sync.RWMutex

	reloj       puertosbolsa.Reloj
	verificador puertosbolsa.VerificadorSellosBaremacion

	versionesPorBaremacion    map[string][]puertosbolsa.VersionBaremacion
	reservasPorAmbito         map[string]reservaBaremacion
	ambitoPorHuellaToken      map[string]string
	ambitoActivoPorBaremacion map[string]string
	auditorias                []registroAuditoriaBaremacion
	eventosOutbox             []eventoOutboxBaremacion
	referenciasTransaccion    map[string]struct{}
	usosAutorizacion          map[string]usoAutorizacionBaremacion
}

// PerfilUsoRepositorioBaremacionesMemoria es una capacidad deliberadamente
// opaca. Solo el constructor de pruebas puede emitirla: evita seleccionar este
// adaptador efimero por accidente en una composicion productiva.
type PerfilUsoRepositorioBaremacionesMemoria struct{ soloPruebas bool }

func NuevoRepositorioBaremaciones(
	reloj puertosbolsa.Reloj,
	verificador puertosbolsa.VerificadorSellosBaremacion,
	perfil PerfilUsoRepositorioBaremacionesMemoria,
) (*RepositorioBaremaciones, error) {
	if interfazNula(reloj) || interfazNula(verificador) || !perfil.soloPruebas || reloj.Ahora().IsZero() {
		return nil, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	return &RepositorioBaremaciones{
		reloj:                     reloj,
		verificador:               verificador,
		versionesPorBaremacion:    make(map[string][]puertosbolsa.VersionBaremacion),
		reservasPorAmbito:         make(map[string]reservaBaremacion),
		ambitoPorHuellaToken:      make(map[string]string),
		ambitoActivoPorBaremacion: make(map[string]string),
		auditorias:                make([]registroAuditoriaBaremacion, 0),
		eventosOutbox:             make([]eventoOutboxBaremacion, 0),
		referenciasTransaccion:    make(map[string]struct{}),
		usosAutorizacion:          make(map[string]usoAutorizacionBaremacion),
	}, nil
}

func (r *RepositorioBaremaciones) ReservarCambio(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudReservarCambioBaremacion,
) (puertosbolsa.ReservaCambioBaremacion, error) {
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, err
	}
	// Copiar antes de validar y verificar cierra el TOCTOU del puntero OCC: la
	// representacion autenticada es exactamente la que se conservara.
	solicitud = solicitud.Clonar()
	if err := solicitud.Validar(); err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	ahoraInicial, err := r.ahora()
	if err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	accion, _ := accionReserva(solicitud.Clase)
	if solicitud.Contexto.ValidarVigentePara(accion, puertosbolsa.ClaseRecursoBaremacion, solicitud.BaremacionMeritoRef, ahoraInicial) != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if err := r.verificarSelloReserva(ctx, solicitud); err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, err
	}
	claveAmbito := claveAmbitoReserva(solicitud.Contexto.Proyeccion().PrincipalRef, solicitud.ClaveIdempotencia)

	r.mu.Lock()
	defer r.mu.Unlock()
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, err
	}
	ahora, err := r.ahora()
	if err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if solicitud.Contexto.ValidarVigentePara(accion, puertosbolsa.ClaseRecursoBaremacion, solicitud.BaremacionMeritoRef, ahora) != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if !r.cadenasIntegrasBloqueadas() {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	huellaEfecto, err := huellaEfectoReserva(solicitud)
	if err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	uso, err := nuevoUsoAutorizacionBaremacion(solicitud.Contexto, ahora, huellaEfecto)
	if err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}

	if existente, existe := r.reservasPorAmbito[claveAmbito]; existe {
		if !solicitudesReservaIguales(existente.SolicitudReserva, solicitud) {
			return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrClaveIdempotenciaBaremacionReutilizada
		}
		consumida, err := r.comprobarUsoAutorizacionBloqueado(uso)
		if err != nil {
			return puertosbolsa.ReservaCambioBaremacion{}, err
		}
		if !consumida {
			return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		switch existente.Estado {
		case estadoReservaConfirmada:
			if existente.Resultado == nil || ahora.Before(existente.Resultado.Version.ConfirmadaEn) {
				return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
			}
			version, err := existente.Resultado.Version.Clonar()
			if err != nil {
				return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
			}
			respuesta := respuestaReservaBaremacion(existente.SolicitudReserva, puertosbolsa.TokenReservaBaremacion{}, true, &version)
			if err := respuesta.ValidarPara(solicitud); err != nil {
				return puertosbolsa.ReservaCambioBaremacion{}, err
			}
			return respuesta, nil
		case estadoReservaActiva:
			if ahora.Before(existente.SolicitudReserva.SolicitadaEn.UTC()) {
				return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
			}
			if ahora.Before(existente.SolicitudReserva.ExpiraEn.UTC()) {
				return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrCambioBaremacionEnCurso
			}
			r.cambiarEstadoReservaBloqueado(claveAmbito, existente, estadoReservaExpirada)
			return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrClaveIdempotenciaBaremacionReutilizada
		case estadoReservaAbandonada, estadoReservaExpirada, estadoReservaInvalidada:
			return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrClaveIdempotenciaBaremacionReutilizada
		default:
			return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
		}
	}
	if ahora.Before(solicitud.SolicitadaEn.UTC()) || !ahora.Before(solicitud.ExpiraEn.UTC()) {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}

	if err := r.comprobarVersionEsperadaBloqueado(solicitud, ahora); err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, err
	}
	if otroAmbito, existe := r.ambitoActivoPorBaremacion[solicitud.BaremacionMeritoRef]; existe {
		otra := r.reservasPorAmbito[otroAmbito]
		mismoSujeto := otra.SolicitudReserva.Contexto.Proyeccion().SujetoRef == solicitud.Contexto.Proyeccion().SujetoRef
		if otra.Estado == estadoReservaActiva && ahora.Before(otra.SolicitudReserva.ExpiraEn.UTC()) {
			if !mismoSujeto {
				return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrBaremacionNoEncontrada
			}
			return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrCambioBaremacionEnCurso
		}
		if otra.Estado == estadoReservaActiva {
			r.cambiarEstadoReservaBloqueado(otroAmbito, otra, estadoReservaExpirada)
		} else {
			delete(r.ambitoActivoPorBaremacion, solicitud.BaremacionMeritoRef)
		}
		if !mismoSujeto {
			return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrBaremacionNoEncontrada
		}
	}
	consumida, err := r.comprobarUsoAutorizacionBloqueado(uso)
	if err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, err
	}
	if consumida {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	if len(r.reservasPorAmbito) >= maximoReservasMemoria {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
	}
	token, err := generarTokenReserva()
	if err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
	}
	huellaToken := huellaTokenReserva(token)
	if _, colision := r.ambitoPorHuellaToken[huellaToken]; colision {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
	}
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, err
	}
	reserva := reservaBaremacion{
		SolicitudReserva: solicitud, HuellaTokenSHA256: huellaToken, Estado: estadoReservaActiva,
	}
	respuesta := respuestaReservaBaremacion(solicitud, token, false, nil)
	if err := respuesta.ValidarPara(solicitud); err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
	}
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, err
	}
	r.reservasPorAmbito[claveAmbito] = reserva
	r.ambitoPorHuellaToken[huellaToken] = claveAmbito
	r.ambitoActivoPorBaremacion[solicitud.BaremacionMeritoRef] = claveAmbito
	r.usosAutorizacion[uso.DecisionRef] = uso
	return respuesta, nil
}

func respuestaReservaBaremacion(
	solicitud puertosbolsa.SolicitudReservarCambioBaremacion,
	token puertosbolsa.TokenReservaBaremacion,
	repetida bool,
	version *puertosbolsa.VersionBaremacion,
) puertosbolsa.ReservaCambioBaremacion {
	respuesta := puertosbolsa.ReservaCambioBaremacion{
		Token: token, Repetida: repetida, VersionConfirmada: version,
		BaremacionMeritoRef: solicitud.BaremacionMeritoRef, Clase: solicitud.Clase,
		HuellaSolicitudHMAC: solicitud.HuellaSolicitudHMAC, ExpiraEn: solicitud.ExpiraEn.UTC(),
	}
	if solicitud.VersionEsperada != nil {
		esperada := *solicitud.VersionEsperada
		respuesta.VersionEsperada = &esperada
	}
	return respuesta
}

func (r *RepositorioBaremaciones) ConfirmarCambio(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
) (puertosbolsa.ResultadoConfirmarCambioBaremacion, error) {
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
	}
	clon, err := solicitud.Clonar()
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	solicitud = clon
	ahora, err := r.ahora()
	if err != nil || solicitud.ConfirmadaEn.After(ahora) {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
	}
	accion, _ := accionConfirmacion(solicitud.Clase)
	if solicitud.Contexto.ValidarVigentePara(accion, puertosbolsa.ClaseRecursoBaremacion, solicitud.Agregado.ID, ahora) != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if err := r.verificarSelloConfirmacion(ctx, solicitud); err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
	}
	huellaNueva, err := solicitud.Agregado.HuellaEstadoSHA256()
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrHistorialBaremacionNoAnexable
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
	}
	ahora, err = r.ahora()
	if err != nil || solicitud.ConfirmadaEn.After(ahora) {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
	}
	if solicitud.Contexto.ValidarVigentePara(accion, puertosbolsa.ClaseRecursoBaremacion, solicitud.Agregado.ID, ahora) != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if !r.cadenasIntegrasBloqueadas() {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	huellaEfecto, err := huellaEfectoConfirmacion(solicitud)
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	uso, err := nuevoUsoAutorizacionBaremacion(solicitud.Contexto, ahora, huellaEfecto)
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	huellaToken := huellaTokenReserva(solicitud.Token)
	claveAmbito, existe := r.ambitoPorHuellaToken[huellaToken]
	if !existe {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
	}
	reserva, existe := r.reservasPorAmbito[claveAmbito]
	if !existe || !cadenasConstantesIguales(reserva.HuellaTokenSHA256, huellaToken) {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
	}
	if reserva.Estado == estadoReservaConfirmada {
		huellaConfirmacion, err := huellaConfirmacion(solicitud)
		if err != nil || reserva.Resultado == nil || ahora.Before(reserva.Resultado.Version.ConfirmadaEn) ||
			!cadenasConstantesIguales(reserva.HuellaConfirmacionSHA256, huellaConfirmacion) {
			return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrClaveIdempotenciaBaremacionReutilizada
		}
		consumida, err := r.comprobarUsoAutorizacionBloqueado(uso)
		if err != nil {
			return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
		}
		if !consumida {
			return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		return reserva.Resultado.Clonar()
	}
	if reserva.Estado != estadoReservaActiva {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
	}
	if !ahora.Before(reserva.SolicitudReserva.ExpiraEn.UTC()) {
		r.cambiarEstadoReservaBloqueado(claveAmbito, reserva, estadoReservaExpirada)
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
	}
	if !confirmacionCorrespondeAReserva(solicitud, reserva.SolicitudReserva) ||
		solicitud.ConfirmadaEn.Before(reserva.SolicitudReserva.SolicitadaEn.UTC()) ||
		!solicitud.ConfirmadaEn.Before(reserva.SolicitudReserva.ExpiraEn.UTC()) {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
	}
	consumida, err := r.comprobarUsoAutorizacionBloqueado(uso)
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
	}
	if consumida {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}

	versionAnterior, numeroNuevo, err := r.validarCambioBloqueado(
		reserva.SolicitudReserva, solicitud.Agregado, huellaNueva, ahora,
	)
	if err != nil {
		r.cambiarEstadoReservaBloqueado(claveAmbito, reserva, estadoReservaInvalidada)
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
	}
	huellaAnterior := ""
	versionAnteriorNumero := uint64(0)
	if versionAnterior != nil {
		huellaAnterior = versionAnterior.Referencia.HuellaEstadoSHA256
		versionAnteriorNumero = versionAnterior.Referencia.Numero
	}
	version := puertosbolsa.VersionBaremacion{
		Referencia: puertosbolsa.ReferenciaVersionBaremacion{
			BaremacionMeritoRef: solicitud.Agregado.ID,
			Numero:              numeroNuevo,
			HuellaEstadoSHA256:  huellaNueva,
		},
		Agregado:     solicitud.Agregado,
		ConfirmadaEn: ahora,
	}
	versionAlmacenada, err := version.Clonar()
	if err != nil {
		r.cambiarEstadoReservaBloqueado(claveAmbito, reserva, estadoReservaInvalidada)
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrHistorialBaremacionNoAnexable
	}
	if !r.capacidadConfirmacionDisponibleBloqueada(solicitud.Agregado.ID) {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	huellaAuditoriaAnterior, huellaEventoAnterior := "", ""
	if len(r.auditorias) != 0 {
		if ahora.Before(r.auditorias[len(r.auditorias)-1].RegistradaEn) {
			return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
		}
		huellaAuditoriaAnterior = r.auditorias[len(r.auditorias)-1].HuellaRegistroSHA256
	}
	if len(r.eventosOutbox) != 0 {
		huellaEventoAnterior = r.eventosOutbox[len(r.eventosOutbox)-1].HuellaRegistroSHA256
	}
	auditoria, evento, evidencia, err := derivarEvidenciaTransaccion(
		solicitud, versionAnteriorNumero, numeroNuevo, huellaAnterior, huellaNueva,
		huellaAuditoriaAnterior, huellaEventoAnterior, uint64(len(r.auditorias)+1), uint64(len(r.eventosOutbox)+1),
		ahora,
	)
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	resultado := puertosbolsa.ResultadoConfirmarCambioBaremacion{Version: versionAlmacenada, Evidencia: evidencia}
	if err := resultado.ValidarPara(solicitud); err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	resultadoAlmacenado, err := resultado.Clonar()
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	huellaConfirmacionAlmacenada, err := huellaConfirmacion(solicitud)
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if auditoria.Referencia == evento.Referencia {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if _, existe := r.referenciasTransaccion[auditoria.Referencia]; existe {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if _, existe := r.referenciasTransaccion[evento.Referencia]; existe {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
	}
	respuestaFinal, err := resultadoAlmacenado.Clonar()
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
	}

	// Punto de confirmacion logica: todos los valores se han validado, clonado
	// y sellado antes de modificar cualquiera de las estructuras enlazadas.
	r.versionesPorBaremacion[solicitud.Agregado.ID] = append(
		r.versionesPorBaremacion[solicitud.Agregado.ID], versionAlmacenada,
	)
	r.auditorias = append(r.auditorias, auditoria)
	r.eventosOutbox = append(r.eventosOutbox, evento)
	r.referenciasTransaccion[auditoria.Referencia] = struct{}{}
	r.referenciasTransaccion[evento.Referencia] = struct{}{}
	reserva.Estado = estadoReservaConfirmada
	reserva.HuellaConfirmacionSHA256 = huellaConfirmacionAlmacenada
	reserva.Resultado = &resultadoAlmacenado
	r.reservasPorAmbito[claveAmbito] = reserva
	delete(r.ambitoActivoPorBaremacion, solicitud.Agregado.ID)
	r.usosAutorizacion[uso.DecisionRef] = uso
	return respuestaFinal, nil
}

func (r *RepositorioBaremaciones) AbandonarReserva(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudAbandonarReservaBaremacion,
) error {
	if err := validarContextoEjecucion(ctx); err != nil {
		return err
	}
	if err := solicitud.Validar(); err != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	ahora, err := r.ahora()
	if err != nil {
		return puertosbolsa.ErrReservaBaremacionNoValida
	}
	accion, _ := accionAbandono(solicitud.Clase)
	if solicitud.Contexto.ValidarVigentePara(
		accion, puertosbolsa.ClaseRecursoBaremacion, solicitud.BaremacionMeritoRef, ahora,
	) != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := validarContextoEjecucion(ctx); err != nil {
		return err
	}
	ahora, err = r.ahora()
	if err != nil {
		return puertosbolsa.ErrReservaBaremacionNoValida
	}
	if solicitud.Contexto.ValidarVigentePara(
		accion, puertosbolsa.ClaseRecursoBaremacion, solicitud.BaremacionMeritoRef, ahora,
	) != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	huellaToken := huellaTokenReserva(solicitud.Token)
	huellaEfecto := huellaCanonica(
		"abandono-reserva-baremacion-v1", huellaToken, string(solicitud.Clase), solicitud.BaremacionMeritoRef,
	)
	uso, err := nuevoUsoAutorizacionBaremacion(solicitud.Contexto, ahora, huellaEfecto)
	if err != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	claveAmbito, existe := r.ambitoPorHuellaToken[huellaToken]
	if !existe {
		return puertosbolsa.ErrReservaBaremacionNoValida
	}
	reserva, existe := r.reservasPorAmbito[claveAmbito]
	if !existe || !cadenasConstantesIguales(reserva.HuellaTokenSHA256, huellaToken) ||
		!mismoVinculoOperacion(reserva.SolicitudReserva.Contexto, solicitud.Contexto) ||
		reserva.SolicitudReserva.Clase != solicitud.Clase ||
		reserva.SolicitudReserva.BaremacionMeritoRef != solicitud.BaremacionMeritoRef {
		return puertosbolsa.ErrReservaBaremacionNoValida
	}
	consumida, err := r.comprobarUsoAutorizacionBloqueado(uso)
	if err != nil {
		return err
	}
	switch reserva.Estado {
	case estadoReservaAbandonada:
		if !consumida {
			return puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		return nil
	case estadoReservaActiva:
		if consumida {
			return puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		if !ahora.Before(reserva.SolicitudReserva.ExpiraEn.UTC()) {
			r.cambiarEstadoReservaBloqueado(claveAmbito, reserva, estadoReservaExpirada)
			return puertosbolsa.ErrReservaBaremacionNoValida
		}
		r.cambiarEstadoReservaBloqueado(claveAmbito, reserva, estadoReservaAbandonada)
		r.usosAutorizacion[uso.DecisionRef] = uso
		return nil
	default:
		return puertosbolsa.ErrReservaBaremacionNoValida
	}
}

func (r *RepositorioBaremaciones) ObtenerVersionVigente(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerBaremacionVigente,
) (puertosbolsa.VersionBaremacion, error) {
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.VersionBaremacion{}, err
	}
	if err := solicitud.Validar(); err != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if r == nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrBaremacionNoEncontrada
	}
	ahora, err := r.ahora()
	if err != nil || solicitud.Contexto.ValidarVigentePara(
		puertosbolsa.AccionConsultarBaremacionVigente, puertosbolsa.ClaseRecursoBaremacion,
		solicitud.BaremacionMeritoRef, ahora,
	) != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.VersionBaremacion{}, err
	}
	ahora, err = r.ahora()
	if err != nil || solicitud.Contexto.ValidarVigentePara(
		puertosbolsa.AccionConsultarBaremacionVigente, puertosbolsa.ClaseRecursoBaremacion,
		solicitud.BaremacionMeritoRef, ahora,
	) != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if !r.cadenasIntegrasBloqueadas() {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	versiones := r.versionesPorBaremacion[solicitud.BaremacionMeritoRef]
	if len(versiones) == 0 || versiones[len(versiones)-1].Agregado.SujetoRef != solicitud.Contexto.Proyeccion().SujetoRef {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrBaremacionNoEncontrada
	}
	version, err := versiones[len(versiones)-1].Clonar()
	if err != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrVersionBaremacionNoEncontrada
	}
	return version, nil
}

func (r *RepositorioBaremaciones) ObtenerVersion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerVersionBaremacion,
) (puertosbolsa.VersionBaremacion, error) {
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.VersionBaremacion{}, err
	}
	if err := solicitud.Validar(); err != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if r == nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrVersionBaremacionNoEncontrada
	}
	ahora, err := r.ahora()
	if err != nil || solicitud.Contexto.ValidarVigentePara(
		puertosbolsa.AccionConsultarVersionBaremacion, puertosbolsa.ClaseRecursoBaremacion,
		solicitud.BaremacionMeritoRef, ahora,
	) != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.VersionBaremacion{}, err
	}
	ahora, err = r.ahora()
	if err != nil || solicitud.Contexto.ValidarVigentePara(
		puertosbolsa.AccionConsultarVersionBaremacion, puertosbolsa.ClaseRecursoBaremacion,
		solicitud.BaremacionMeritoRef, ahora,
	) != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if !r.cadenasIntegrasBloqueadas() {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	versiones := r.versionesPorBaremacion[solicitud.BaremacionMeritoRef]
	if solicitud.Numero > uint64(len(versiones)) ||
		versiones[solicitud.Numero-1].Agregado.SujetoRef != solicitud.Contexto.Proyeccion().SujetoRef {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrVersionBaremacionNoEncontrada
	}
	version, err := versiones[solicitud.Numero-1].Clonar()
	if err != nil || version.Referencia.Numero != solicitud.Numero {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrVersionBaremacionNoEncontrada
	}
	return version, nil
}

func (r *RepositorioBaremaciones) ObtenerEvidenciaTransaccion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerEvidenciaTransaccionBaremacion,
) (puertosbolsa.EvidenciaTransaccionBaremacionRecuperada, error) {
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, err
	}
	if solicitud.Validar() != nil || r == nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	ahora, err := r.ahora()
	if err != nil || solicitud.Contexto.ValidarVigentePara(
		puertosbolsa.AccionConsultarEvidenciaTransaccionBaremacion, puertosbolsa.ClaseRecursoTransaccion,
		solicitud.AuditoriaRef, ahora,
	) != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, err
	}
	ahora, err = r.ahora()
	if err != nil || solicitud.Contexto.ValidarVigentePara(
		puertosbolsa.AccionConsultarEvidenciaTransaccionBaremacion, puertosbolsa.ClaseRecursoTransaccion,
		solicitud.AuditoriaRef, ahora,
	) != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if !r.cadenasIntegrasBloqueadas() {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	indice := -1
	for actual := range r.auditorias {
		if r.auditorias[actual].Referencia == solicitud.AuditoriaRef {
			indice = actual
			break
		}
	}
	if indice < 0 || indice >= len(r.eventosOutbox) ||
		r.eventosOutbox[indice].Referencia != solicitud.EventoOutboxRef {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrEvidenciaBaremacionNoEncontrada
	}
	auditoria, evento := r.auditorias[indice], r.eventosOutbox[indice]
	versiones := r.versionesPorBaremacion[solicitud.BaremacionMeritoRef]
	if solicitud.NumeroVersion > uint64(len(versiones)) || auditoria.SujetoRef != solicitud.Contexto.Proyeccion().SujetoRef {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrEvidenciaBaremacionNoEncontrada
	}
	version, err := versiones[solicitud.NumeroVersion-1].Clonar()
	if err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	auditoria.CamposPermitidos = append([]string(nil), auditoria.CamposPermitidos...)
	resultado := puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{
		Version: version, Auditoria: auditoria, Evento: evento,
		Evidencia: puertosbolsa.EvidenciaTransaccionBaremacion{
			AuditoriaRef: auditoria.Referencia, HuellaAuditoriaSHA256: auditoria.HuellaRegistroSHA256,
			EventoOutboxRef: evento.Referencia, HuellaEventoOutboxSHA256: evento.HuellaRegistroSHA256,
			ConfirmadaEn: auditoria.RegistradaEn,
		},
	}
	if resultado.ValidarPara(solicitud) != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	return resultado, nil
}

func (r *RepositorioBaremaciones) comprobarVersionEsperadaBloqueado(
	solicitud puertosbolsa.SolicitudReservarCambioBaremacion,
	ahora time.Time,
) error {
	versiones := r.versionesPorBaremacion[solicitud.BaremacionMeritoRef]
	switch solicitud.Clase {
	case puertosbolsa.ClaseCambioAltaBaremacion:
		if len(versiones) != 0 {
			if versiones[len(versiones)-1].Agregado.SujetoRef != solicitud.Contexto.Proyeccion().SujetoRef {
				return puertosbolsa.ErrBaremacionNoEncontrada
			}
			if ahora.Before(versiones[len(versiones)-1].ConfirmadaEn) {
				return puertosbolsa.ErrSolicitudBaremacionInvalida
			}
			return puertosbolsa.ErrBaremacionYaExiste
		}
		return nil
	case puertosbolsa.ClaseCambioIncorporarDecision:
		if len(versiones) == 0 {
			return puertosbolsa.ErrBaremacionNoEncontrada
		}
		if versiones[len(versiones)-1].Agregado.SujetoRef != solicitud.Contexto.Proyeccion().SujetoRef {
			return puertosbolsa.ErrBaremacionNoEncontrada
		}
		if ahora.Before(versiones[len(versiones)-1].ConfirmadaEn) {
			return puertosbolsa.ErrSolicitudBaremacionInvalida
		}
		actual := versiones[len(versiones)-1].Referencia
		if !referenciasVersionIguales(&actual, solicitud.VersionEsperada) {
			return puertosbolsa.ErrVersionBaremacionConflicto
		}
		return nil
	default:
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
}

func (r *RepositorioBaremaciones) validarCambioBloqueado(
	reserva puertosbolsa.SolicitudReservarCambioBaremacion,
	agregado dominiobolsa.BaremacionMerito,
	huellaNueva string,
	confirmadaEn time.Time,
) (*puertosbolsa.VersionBaremacion, uint64, error) {
	versiones := r.versionesPorBaremacion[agregado.ID]
	switch reserva.Clase {
	case puertosbolsa.ClaseCambioAltaBaremacion:
		if reserva.VersionEsperada != nil || len(versiones) != 0 {
			return nil, 0, puertosbolsa.ErrBaremacionYaExiste
		}
		if len(agregado.Decisiones) != 0 {
			return nil, 0, puertosbolsa.ErrHistorialBaremacionNoAnexable
		}
		return nil, 1, nil
	case puertosbolsa.ClaseCambioIncorporarDecision:
		if reserva.VersionEsperada == nil || len(versiones) == 0 {
			return nil, 0, puertosbolsa.ErrBaremacionNoEncontrada
		}
		actual, err := versiones[len(versiones)-1].Clonar()
		if err != nil || !referenciasVersionIguales(&actual.Referencia, reserva.VersionEsperada) {
			return nil, 0, puertosbolsa.ErrVersionBaremacionConflicto
		}
		if confirmadaEn.Before(actual.ConfirmadaEn) {
			return nil, 0, puertosbolsa.ErrHistorialBaremacionNoAnexable
		}
		if len(agregado.Decisiones) != len(actual.Agregado.Decisiones)+1 {
			return nil, 0, puertosbolsa.ErrHistorialBaremacionNoAnexable
		}
		esperado, err := actual.Agregado.IncorporarDecision(agregado.Decisiones[len(agregado.Decisiones)-1])
		if err != nil {
			return nil, 0, puertosbolsa.ErrHistorialBaremacionNoAnexable
		}
		huellaEsperada, err := esperado.HuellaEstadoSHA256()
		if err != nil || !cadenasConstantesIguales(huellaEsperada, huellaNueva) {
			return nil, 0, puertosbolsa.ErrHistorialBaremacionNoAnexable
		}
		return &actual, actual.Referencia.Numero + 1, nil
	default:
		return nil, 0, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
}

func (r *RepositorioBaremaciones) cambiarEstadoReservaBloqueado(
	claveAmbito string,
	reserva reservaBaremacion,
	estado estadoReserva,
) {
	reserva.Estado = estado
	r.reservasPorAmbito[claveAmbito] = reserva
	if r.ambitoActivoPorBaremacion[reserva.SolicitudReserva.BaremacionMeritoRef] == claveAmbito {
		delete(r.ambitoActivoPorBaremacion, reserva.SolicitudReserva.BaremacionMeritoRef)
	}
}

func (r *RepositorioBaremaciones) capacidadConfirmacionDisponibleBloqueada(baremacionRef string) bool {
	if r == nil || len(r.auditorias) != len(r.eventosOutbox) ||
		len(r.auditorias) >= maximoTransaccionesMemoria ||
		len(r.referenciasTransaccion) != len(r.auditorias)*2 || !r.cadenasIntegrasBloqueadas() {
		return false
	}
	versiones := r.versionesPorBaremacion[baremacionRef]
	if len(versiones) >= maximoVersionesPorBaremacionMemoria {
		return false
	}
	if len(versiones) == 0 && len(r.versionesPorBaremacion) >= maximoBaremacionesMemoria {
		return false
	}
	return true
}

func (r *RepositorioBaremaciones) cadenasIntegrasBloqueadas() bool {
	if r == nil || len(r.auditorias) != len(r.eventosOutbox) ||
		len(r.referenciasTransaccion) != len(r.auditorias)*2 {
		return false
	}
	huellaAuditoriaAnterior, huellaEventoAnterior := "", ""
	referencias := make(map[string]struct{}, len(r.auditorias)*2)
	for indice := range r.auditorias {
		auditoria, evento := r.auditorias[indice], r.eventosOutbox[indice]
		secuencia := uint64(indice + 1)
		if auditoria.Validar() != nil || evento.Validar() != nil || auditoria.Secuencia != secuencia ||
			evento.Secuencia != secuencia || auditoria.HuellaAnteriorAuditoriaSHA256 != huellaAuditoriaAnterior ||
			evento.HuellaEventoAnteriorSHA256 != huellaEventoAnterior ||
			auditoria.HuellaRegistroSHA256 != huellaAuditoria(auditoria) ||
			evento.HuellaRegistroSHA256 != huellaEvento(evento) || evento.AuditoriaRef != auditoria.Referencia ||
			evento.HuellaAuditoriaSHA256 != auditoria.HuellaRegistroSHA256 ||
			evento.BaremacionMeritoRef != auditoria.BaremacionMeritoRef || evento.VersionNueva != auditoria.VersionNueva ||
			evento.HuellaNuevaSHA256 != auditoria.HuellaNuevaSHA256 || evento.SujetoRef != auditoria.SujetoRef ||
			evento.PrincipalRef != auditoria.PrincipalRef || evento.CorrelacionRef != auditoria.CorrelacionRef ||
			!evento.RegistradoEn.Equal(auditoria.RegistradaEn) {
			return false
		}
		versiones := r.versionesPorBaremacion[auditoria.BaremacionMeritoRef]
		if auditoria.VersionNueva > uint64(len(versiones)) {
			return false
		}
		version := versiones[auditoria.VersionNueva-1]
		if version.Validar() != nil || version.Referencia.Numero != auditoria.VersionNueva ||
			version.Referencia.HuellaEstadoSHA256 != auditoria.HuellaNuevaSHA256 ||
			version.Agregado.SujetoRef != auditoria.SujetoRef || !version.ConfirmadaEn.Equal(auditoria.RegistradaEn) {
			return false
		}
		for _, referencia := range []string{auditoria.Referencia, evento.Referencia} {
			if _, repetida := referencias[referencia]; repetida {
				return false
			}
			referencias[referencia] = struct{}{}
			if _, reservada := r.referenciasTransaccion[referencia]; !reservada {
				return false
			}
		}
		huellaAuditoriaAnterior = auditoria.HuellaRegistroSHA256
		huellaEventoAnterior = evento.HuellaRegistroSHA256
	}
	return true
}

func (r *RepositorioBaremaciones) ahora() (time.Time, error) {
	if r == nil || interfazNula(r.reloj) {
		return time.Time{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	ahora := r.reloj.Ahora()
	if ahora.IsZero() {
		return time.Time{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	return ahora.UTC(), nil
}

func (r *RepositorioBaremaciones) verificarSelloReserva(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudReservarCambioBaremacion,
) error {
	if r == nil || interfazNula(r.verificador) {
		return puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible
	}
	representacion, err := puertosbolsa.RepresentacionCanonicaReservaBaremacion(solicitud)
	if err != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	peticion := puertosbolsa.SolicitudVerificarSelloBaremacion{
		Finalidad: puertosbolsa.FinalidadSelloReservaBaremacion, RepresentacionCanonica: representacion,
		SelloHMAC: solicitud.HuellaSolicitudHMAC,
	}
	if peticion.Validar() != nil {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	if err := r.verificador.VerificarSelloBaremacion(ctx, peticion); err != nil {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	return nil
}

func (r *RepositorioBaremaciones) verificarSelloConfirmacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
) error {
	if r == nil || interfazNula(r.verificador) {
		return puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible
	}
	representacion, err := puertosbolsa.RepresentacionCanonicaConfirmacionBaremacion(solicitud)
	if err != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	peticion := puertosbolsa.SolicitudVerificarSelloBaremacion{
		Finalidad: puertosbolsa.FinalidadSelloConfirmacionBaremacion, RepresentacionCanonica: representacion,
		SelloHMAC: solicitud.HuellaSolicitudHMAC,
	}
	if peticion.Validar() != nil {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	if err := r.verificador.VerificarSelloBaremacion(ctx, peticion); err != nil {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	return nil
}

func accionReserva(clase puertosbolsa.ClaseCambioBaremacion) (puertosbolsa.AccionOperacionBaremacion, bool) {
	switch clase {
	case puertosbolsa.ClaseCambioAltaBaremacion:
		return puertosbolsa.AccionReservarAltaBaremacion, true
	case puertosbolsa.ClaseCambioIncorporarDecision:
		return puertosbolsa.AccionReservarDecisionBaremacion, true
	default:
		return "", false
	}
}

func accionConfirmacion(clase puertosbolsa.ClaseCambioBaremacion) (puertosbolsa.AccionOperacionBaremacion, bool) {
	switch clase {
	case puertosbolsa.ClaseCambioAltaBaremacion:
		return puertosbolsa.AccionConfirmarAltaBaremacion, true
	case puertosbolsa.ClaseCambioIncorporarDecision:
		return puertosbolsa.AccionConfirmarDecisionBaremacion, true
	default:
		return "", false
	}
}

func accionAbandono(clase puertosbolsa.ClaseCambioBaremacion) (puertosbolsa.AccionOperacionBaremacion, bool) {
	switch clase {
	case puertosbolsa.ClaseCambioAltaBaremacion:
		return puertosbolsa.AccionAbandonarAltaBaremacion, true
	case puertosbolsa.ClaseCambioIncorporarDecision:
		return puertosbolsa.AccionAbandonarDecisionBaremacion, true
	default:
		return "", false
	}
}

func confirmacionCorrespondeAReserva(
	confirmacion puertosbolsa.SolicitudConfirmarCambioBaremacion,
	reserva puertosbolsa.SolicitudReservarCambioBaremacion,
) bool {
	return mismoVinculoOperacion(confirmacion.Contexto, reserva.Contexto) && confirmacion.Clase == reserva.Clase &&
		confirmacion.Agregado.ID == reserva.BaremacionMeritoRef &&
		referenciasVersionOpcionalesIguales(confirmacion.VersionEsperada, reserva.VersionEsperada)
}

func solicitudesReservaIguales(
	a, b puertosbolsa.SolicitudReservarCambioBaremacion,
) bool {
	return proyeccionesAutorizacionIguales(a.Contexto, b.Contexto) && a.Clase == b.Clase && a.ClaveIdempotencia == b.ClaveIdempotencia &&
		a.BaremacionMeritoRef == b.BaremacionMeritoRef &&
		referenciasVersionOpcionalesIguales(a.VersionEsperada, b.VersionEsperada) &&
		cadenasConstantesIguales(a.HuellaSolicitudHMAC, b.HuellaSolicitudHMAC) &&
		a.SolicitadaEn.Equal(b.SolicitadaEn) && a.ExpiraEn.Equal(b.ExpiraEn)
}

func proyeccionesAutorizacionIguales(a, b puertosbolsa.ContextoOperacionBaremacion) bool {
	return a.CoincideExactamenteCon(b)
}

func mismoVinculoOperacion(a, b puertosbolsa.ContextoOperacionBaremacion) bool {
	pa, pb := a.Proyeccion(), b.Proyeccion()
	return a.MismoVinculoAutenticacionQue(b) && pa.RecursoRef == pb.RecursoRef &&
		pa.FinalidadClave == pb.FinalidadClave &&
		pa.CorrelacionRef == pb.CorrelacionRef
}

func huellaEfectoReserva(solicitud puertosbolsa.SolicitudReservarCambioBaremacion) (string, error) {
	return transaccionbolsa.HuellaEfectoReserva(solicitud)
}

func huellaEfectoConfirmacion(solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion) (string, error) {
	return transaccionbolsa.HuellaEfectoConfirmacion(solicitud)
}

func nuevoUsoAutorizacionBaremacion(
	contexto puertosbolsa.ContextoOperacionBaremacion,
	instante time.Time,
	huellaEfecto string,
) (usoAutorizacionBaremacion, error) {
	evidencia, err := contexto.EvidenciaUsoAutorizacion()
	if err != nil || evidencia.ValidarEn(instante) != nil || !huellaSHA256MemoriaValida(huellaEfecto) {
		return usoAutorizacionBaremacion{}, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	datos, err := evidencia.Datos()
	proyeccion := contexto.Proyeccion()
	if err != nil || datos.Decision.DecisionRef != proyeccion.AutorizacionRef ||
		!huellaSHA256MemoriaValida(datos.HuellaDecisionSHA256) {
		return usoAutorizacionBaremacion{}, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	return usoAutorizacionBaremacion{
		DecisionRef:          datos.Decision.DecisionRef,
		HuellaDecisionSHA256: datos.HuellaDecisionSHA256,
		HuellaEfectoSHA256:   huellaEfecto,
	}, nil
}

func (r *RepositorioBaremaciones) comprobarUsoAutorizacionBloqueado(
	uso usoAutorizacionBaremacion,
) (bool, error) {
	if r == nil || r.usosAutorizacion == nil || uso.DecisionRef == "" ||
		!huellaSHA256MemoriaValida(uso.HuellaDecisionSHA256) || !huellaSHA256MemoriaValida(uso.HuellaEfectoSHA256) {
		return false, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	existente, consumida := r.usosAutorizacion[uso.DecisionRef]
	if !consumida {
		if len(r.usosAutorizacion) >= maximoUsosAutorizacionMemoria {
			return false, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		return false, nil
	}
	if !cadenasConstantesIguales(existente.HuellaDecisionSHA256, uso.HuellaDecisionSHA256) ||
		!cadenasConstantesIguales(existente.HuellaEfectoSHA256, uso.HuellaEfectoSHA256) {
		return true, puertosbolsa.ErrAutorizacionBaremacionReutilizada
	}
	return true, nil
}

func huellaSHA256MemoriaValida(valor string) bool {
	if len(valor) != sha256.Size*2 || strings.ToLower(valor) != valor {
		return false
	}
	bytes, err := hex.DecodeString(valor)
	return err == nil && len(bytes) == sha256.Size
}

func huellaConfirmacion(s puertosbolsa.SolicitudConfirmarCambioBaremacion) (string, error) {
	return huellaEfectoConfirmacion(s)
}

func derivarEvidenciaTransaccion(
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
	versionAnterior, versionNueva uint64,
	huellaAnterior, huellaNueva, huellaAuditoriaAnterior, huellaEventoAnterior string,
	secuenciaAuditoria, secuenciaEvento uint64,
	registradaEn time.Time,
) (registroAuditoriaBaremacion, eventoOutboxBaremacion, puertosbolsa.EvidenciaTransaccionBaremacion, error) {
	return transaccionbolsa.DerivarEvidencia(
		solicitud, versionAnterior, versionNueva, huellaAnterior, huellaNueva,
		huellaAuditoriaAnterior, huellaEventoAnterior, secuenciaAuditoria, secuenciaEvento, registradaEn,
	)
}

func huellaAuditoria(a registroAuditoriaBaremacion) string {
	return transaccionbolsa.HuellaAuditoria(a)
}

func huellaEvento(e eventoOutboxBaremacion) string {
	return transaccionbolsa.HuellaEvento(e)
}

func huellaCanonica(partes ...string) string {
	return transaccionbolsa.HuellaCanonica(partes...)
}

func generarTokenReserva() (puertosbolsa.TokenReservaBaremacion, error) {
	return transaccionbolsa.GenerarTokenReserva()
}

func claveAmbitoReserva(principalRef, claveIdempotencia string) string {
	return strconv.Itoa(len(principalRef)) + ":" + principalRef + claveIdempotencia
}

func referenciasVersionOpcionalesIguales(a, b *puertosbolsa.ReferenciaVersionBaremacion) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return referenciasVersionIguales(a, b)
}

func referenciasVersionIguales(a, b *puertosbolsa.ReferenciaVersionBaremacion) bool {
	return a != nil && b != nil && a.BaremacionMeritoRef == b.BaremacionMeritoRef && a.Numero == b.Numero &&
		cadenasConstantesIguales(a.HuellaEstadoSHA256, b.HuellaEstadoSHA256)
}

func huellaTokenReserva(token puertosbolsa.TokenReservaBaremacion) string {
	return transaccionbolsa.HuellaTokenReserva(token)
}

func cadenasConstantesIguales(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func validarContextoEjecucion(ctx context.Context) error {
	if ctx == nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	return ctx.Err()
}

func interfazNula(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}
