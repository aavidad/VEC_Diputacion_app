// Package memory contiene adaptadores efimeros y defensivos del modulo de
// bolsas. Solo son apropiados para pruebas: no sustituyen una transaccion
// durable ni una outbox persistente.
package memory

import (
	"context"
	"sync"

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

// manifiestoBaremacionPersistido conserva el sobre probatorio completo y su
// enlace inmutable con la version que incorporo la decision. Los indices por
// referencia y version apuntan al mismo asiento logico y se validan juntos.
type manifiestoBaremacionPersistido struct {
	Manifiesto          puertosbolsa.ManifiestoProbatorioBaremacion
	BaremacionMeritoRef string
	NumeroVersion       uint64
	DecisionRef         string
}

func (m manifiestoBaremacionPersistido) clonar() manifiestoBaremacionPersistido {
	clon := m
	clon.Manifiesto = m.Manifiesto.Clonar()
	return clon
}

// RepositorioBaremaciones conserva versiones, tombstones de idempotencia,
// auditoria y outbox bajo el mismo mutex. Una clave o token abandonados,
// expirados o invalidados nunca vuelven a habilitarse.
type RepositorioBaremaciones struct {
	mu sync.RWMutex

	reloj       puertosbolsa.Reloj
	verificador puertosbolsa.VerificadorSellosBaremacion

	versionesPorBaremacion    map[string][]puertosbolsa.VersionBaremacion
	manifiestosPorReferencia  map[string]manifiestoBaremacionPersistido
	manifiestoRefPorVersion   map[string]string
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
		manifiestosPorReferencia:  make(map[string]manifiestoBaremacionPersistido),
		manifiestoRefPorVersion:   make(map[string]string),
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
	bloqueoActivo := true
	defer func() {
		if bloqueoActivo {
			r.mu.Unlock()
		}
	}()
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
			instantaneas, err := r.instantaneasManifiestosVersionBloqueada(existente.Resultado.Version)
			if err != nil {
				return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
			}
			version, err := existente.Resultado.Version.Clonar()
			if err != nil {
				return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
			}
			respuesta := respuestaReservaBaremacion(existente.SolicitudReserva, puertosbolsa.TokenReservaBaremacion{}, true, &version)
			if err := respuesta.ValidarPara(solicitud); err != nil {
				return puertosbolsa.ReservaCambioBaremacion{}, err
			}
			r.mu.Unlock()
			bloqueoActivo = false
			if err := r.verificarInstantaneasManifiestos(ctx, instantaneas); err != nil {
				return puertosbolsa.ReservaCambioBaremacion{}, errorVerificacionConContexto(
					ctx, puertosbolsa.ErrEvidenciaBaremacionNoConfiable,
				)
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
	if solicitud.Manifiesto != nil {
		if err := r.verificarSelloManifiesto(ctx, *solicitud.Manifiesto); err != nil {
			return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
		}
	}
	ahora, err := r.ahora()
	if err != nil || solicitud.ConfirmadaEn.After(ahora) {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
	}
	accion, _ := accionConfirmacion(solicitud.Clase)
	if !contextosConfirmacionVigentes(solicitud, accion, ahora) {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if err := r.verificarSelloConfirmacion(ctx, solicitud); err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
	}
	huellaNueva, err := solicitud.Agregado.HuellaEstadoSHA256()
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrHistorialBaremacionNoAnexable
	}
	// La criptografia historica se ejecuta antes del bloqueo exclusivo. La fase
	// de confirmacion que sigue vuelve a comprobar bajo el mutex la reserva, OCC,
	// cadenas, cardinalidad e indices antes del unico punto de escritura.
	instantaneasHistoricas, err := r.instantaneasHistoricasParaConfirmacion(ctx, solicitud)
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
	}
	if err := r.verificarInstantaneasManifiestos(ctx, instantaneasHistoricas); err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, errorVerificacionConContexto(
			ctx, puertosbolsa.ErrEvidenciaBaremacionNoConfiable,
		)
	}

	r.mu.Lock()
	bloqueoActivo := true
	defer func() {
		if bloqueoActivo {
			r.mu.Unlock()
		}
	}()
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
	}
	ahora, err = r.ahora()
	if err != nil || solicitud.ConfirmadaEn.After(ahora) {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
	}
	if !contextosConfirmacionVigentes(solicitud, accion, ahora) {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if !r.cadenasIntegrasBloqueadas() {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	huellaEfecto, err := huellaEfectoConfirmacion(solicitud)
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	usos, err := nuevosUsosAutorizacionConfirmacion(solicitud, ahora, huellaEfecto)
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
		consumida, err := r.comprobarUsosConfirmacionBloqueados(usos)
		if err != nil {
			return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
		}
		if !consumida {
			return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		instantaneas, err := r.instantaneasManifiestosVersionBloqueada(reserva.Resultado.Version)
		if err != nil {
			return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		respuesta, err := reserva.Resultado.Clonar()
		if err != nil {
			return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		r.mu.Unlock()
		bloqueoActivo = false
		if err := r.verificarInstantaneasManifiestos(ctx, instantaneas); err != nil {
			return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, errorVerificacionConContexto(
				ctx, puertosbolsa.ErrEvidenciaBaremacionNoConfiable,
			)
		}
		return respuesta, nil
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
	consumida, err := r.comprobarUsosConfirmacionBloqueados(usos)
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
	manifiestoAlmacenado, err := prepararManifiestoPersistido(solicitud, numeroNuevo)
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if manifiestoAlmacenado != nil {
		claveVersion := claveVersionManifiesto(
			manifiestoAlmacenado.BaremacionMeritoRef, manifiestoAlmacenado.NumeroVersion,
		)
		if _, existe := r.manifiestosPorReferencia[manifiestoAlmacenado.Manifiesto.Referencia]; existe {
			return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
		}
		if _, existe := r.manifiestoRefPorVersion[claveVersion]; existe {
			return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
		}
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
	if manifiestoAlmacenado != nil {
		claveVersion := claveVersionManifiesto(
			manifiestoAlmacenado.BaremacionMeritoRef, manifiestoAlmacenado.NumeroVersion,
		)
		r.manifiestosPorReferencia[manifiestoAlmacenado.Manifiesto.Referencia] = manifiestoAlmacenado.clonar()
		r.manifiestoRefPorVersion[claveVersion] = manifiestoAlmacenado.Manifiesto.Referencia
	}
	r.auditorias = append(r.auditorias, auditoria)
	r.eventosOutbox = append(r.eventosOutbox, evento)
	r.referenciasTransaccion[auditoria.Referencia] = struct{}{}
	r.referenciasTransaccion[evento.Referencia] = struct{}{}
	reserva.Estado = estadoReservaConfirmada
	reserva.HuellaConfirmacionSHA256 = huellaConfirmacionAlmacenada
	reserva.Resultado = &resultadoAlmacenado
	r.reservasPorAmbito[claveAmbito] = reserva
	delete(r.ambitoActivoPorBaremacion, solicitud.Agregado.ID)
	r.usosAutorizacion[usos.confirmacion.DecisionRef] = usos.confirmacion
	if usos.incluyePrevalidacion {
		r.usosAutorizacion[usos.prevalidacion.DecisionRef] = usos.prevalidacion
	}
	return respuestaFinal, nil
}
