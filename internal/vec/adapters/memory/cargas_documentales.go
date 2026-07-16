package memory

import (
	"context"
	"strings"
	"sync"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// RepositorioCargasDocumentalesMemoria verifica la semantica transaccional
// del puerto, pero no es durable ni apto para produccion. En particular,
// ConfirmarPreparacion conserva bajo el mismo bloqueo la transicion, el
// manifiesto, la auditoria y el evento; nunca simula esa atomicidad mediante
// escrituras sucesivas.
type RepositorioCargasDocumentalesMemoria struct {
	mu                    sync.RWMutex
	reloj                 ports.Reloj
	reservasPorIndice     map[string]reservaCargaDocumentalMemoria
	indicePorHuellaToken  map[string]string
	cargasPorID           map[string]domain.CargaDocumental
	manifiestosPorID      map[string]domain.ManifiestoPreparacionCargaDirectaV1
	decisionesPreparacion map[string]reclamacionDecisionPreparacionCargaMemoria
	auditoria             []domain.AuditEntry
	eventos               []domain.Event
}

type estadoReservaCargaDocumentalMemoria uint8

const (
	estadoReservaCargaMemoriaActiva estadoReservaCargaDocumentalMemoria = iota + 1
	estadoReservaCargaMemoriaConfirmada
	estadoReservaCargaMemoriaAbandonada
	estadoReservaCargaMemoriaExpirada
)

type estadoDecisionPreparacionCargaMemoria uint8

const (
	estadoDecisionPreparacionCargaActiva estadoDecisionPreparacionCargaMemoria = iota + 1
	estadoDecisionPreparacionCargaConsumida
	estadoDecisionPreparacionCargaAbandonada
	estadoDecisionPreparacionCargaExpirada
)

type reclamacionDecisionPreparacionCargaMemoria struct {
	estado            estadoDecisionPreparacionCargaMemoria
	consumo           ports.ConsumoDecisionPreparacionCargaDocumentalV1
	indice            string
	huellaTokenSHA256 string
	expiraEn          time.Time
	cargaID           string
}

type reservaCargaDocumentalMemoria struct {
	estado              estadoReservaCargaDocumentalMemoria
	indiceHMAC          string
	huellaSolicitudHMAC string
	huellaTokenSHA256   string
	carga               domain.CargaDocumental
	decisionPreparacion ports.ConsumoDecisionPreparacionCargaDocumentalV1
	expiraEn            time.Time
}

func NuevoRepositorioCargasDocumentalesMemoria(
	reloj ports.Reloj,
) (*RepositorioCargasDocumentalesMemoria, error) {
	if reloj == nil || reloj.Ahora().UTC().IsZero() {
		return nil, ports.ErrRepositorioCargasNoDisponible
	}
	return &RepositorioCargasDocumentalesMemoria{
		reloj: reloj, reservasPorIndice: make(map[string]reservaCargaDocumentalMemoria),
		indicePorHuellaToken: make(map[string]string), cargasPorID: make(map[string]domain.CargaDocumental),
		manifiestosPorID:      make(map[string]domain.ManifiestoPreparacionCargaDirectaV1),
		decisionesPreparacion: make(map[string]reclamacionDecisionPreparacionCargaMemoria),
	}, nil
}

func (r *RepositorioCargasDocumentalesMemoria) Reservar(
	ctx context.Context,
	solicitud ports.SolicitudReservarCargaDocumental,
) (ports.ReservaCargaDocumental, error) {
	if r == nil || ctx == nil || solicitud.Validar() != nil {
		return ports.ReservaCargaDocumental{}, ports.ErrReservaCargaDocumentalInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ReservaCargaDocumental{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return ports.ReservaCargaDocumental{}, err
	}
	ahora := r.reloj.Ahora().UTC()
	if ahora.IsZero() || ahora.Before(solicitud.SolicitadaEn) || !ahora.Before(solicitud.ReservaExpiraEn) {
		return ports.ReservaCargaDocumental{}, ports.ErrReservaCargaDocumentalInvalida
	}
	if existente, existe := r.reservasPorIndice[solicitud.IndiceIdempotenciaHMAC]; existe {
		if existente.huellaSolicitudHMAC != solicitud.HuellaSolicitudHMAC {
			return ports.ReservaCargaDocumental{}, ports.ErrReservaCargaDocumentalOcupada
		}
		switch existente.estado {
		case estadoReservaCargaMemoriaConfirmada:
			carga := clonarCargaDocumentalMemoria(existente.carga)
			resultado := ports.ReservaCargaDocumental{Repetida: true, Carga: carga}
			if resultado.Validar() != nil {
				return ports.ReservaCargaDocumental{}, ports.ErrReservaCargaDocumentalInvalida
			}
			return resultado, nil
		case estadoReservaCargaMemoriaActiva:
			if ahora.Before(existente.expiraEn) {
				return ports.ReservaCargaDocumental{}, ports.ErrReservaCargaDocumentalOcupada
			}
			if !r.marcarDecisionPreparacionFinalizada(
				existente, estadoDecisionPreparacionCargaExpirada,
			) {
				return ports.ReservaCargaDocumental{}, ports.ErrReservaCargaDocumentalInvalida
			}
			delete(r.indicePorHuellaToken, existente.huellaTokenSHA256)
			delete(r.reservasPorIndice, solicitud.IndiceIdempotenciaHMAC)
		case estadoReservaCargaMemoriaAbandonada:
			delete(r.reservasPorIndice, solicitud.IndiceIdempotenciaHMAC)
		case estadoReservaCargaMemoriaExpirada:
			delete(r.reservasPorIndice, solicitud.IndiceIdempotenciaHMAC)
		default:
			return ports.ReservaCargaDocumental{}, ports.ErrReservaCargaDocumentalInvalida
		}
	}
	if _, existe := r.cargasPorID[solicitud.Carga.ID]; existe {
		return ports.ReservaCargaDocumental{}, ports.ErrReservaCargaDocumentalOcupada
	}
	if reclamacion, existe := r.decisionesPreparacion[solicitud.DecisionPreparacion.DecisionRef]; existe {
		if reclamacion.estado == estadoDecisionPreparacionCargaActiva && !ahora.Before(reclamacion.expiraEn) {
			reservaExpirada, existeReserva := r.reservasPorIndice[reclamacion.indice]
			if !existeReserva || reservaExpirada.estado != estadoReservaCargaMemoriaActiva ||
				reservaExpirada.huellaTokenSHA256 != reclamacion.huellaTokenSHA256 ||
				reservaExpirada.decisionPreparacion != reclamacion.consumo {
				return ports.ReservaCargaDocumental{}, ports.ErrReservaCargaDocumentalInvalida
			}
			reservaExpirada.estado = estadoReservaCargaMemoriaExpirada
			reservaExpirada.huellaTokenSHA256 = ""
			r.reservasPorIndice[reclamacion.indice] = reservaExpirada
			delete(r.indicePorHuellaToken, reclamacion.huellaTokenSHA256)
			reclamacion.estado = estadoDecisionPreparacionCargaExpirada
			reclamacion.huellaTokenSHA256 = ""
			r.decisionesPreparacion[solicitud.DecisionPreparacion.DecisionRef] = reclamacion
		}
		return ports.ReservaCargaDocumental{}, ports.ErrDecisionPreparacionCargaNoDisponible
	}
	token, err := ports.NuevoTokenReservaCargaDocumental()
	if err != nil {
		return ports.ReservaCargaDocumental{}, ports.ErrReservaCargaDocumentalInvalida
	}
	huellaToken, err := token.HuellaSHA256()
	if err != nil {
		return ports.ReservaCargaDocumental{}, ports.ErrReservaCargaDocumentalInvalida
	}
	if _, existe := r.indicePorHuellaToken[huellaToken]; existe {
		return ports.ReservaCargaDocumental{}, ports.ErrReservaCargaDocumentalInvalida
	}
	reservaPersistible := reservaCargaDocumentalMemoria{
		estado: estadoReservaCargaMemoriaActiva, indiceHMAC: solicitud.IndiceIdempotenciaHMAC,
		huellaSolicitudHMAC: solicitud.HuellaSolicitudHMAC, huellaTokenSHA256: huellaToken,
		carga:               clonarCargaDocumentalMemoria(solicitud.Carga),
		decisionPreparacion: solicitud.DecisionPreparacion, expiraEn: solicitud.ReservaExpiraEn.UTC(),
	}
	reclamacionPersistible := reclamacionDecisionPreparacionCargaMemoria{
		estado: estadoDecisionPreparacionCargaActiva, consumo: solicitud.DecisionPreparacion,
		indice: solicitud.IndiceIdempotenciaHMAC, huellaTokenSHA256: huellaToken,
		expiraEn: solicitud.ReservaExpiraEn.UTC(), cargaID: solicitud.Carga.ID,
	}
	resultado := ports.ReservaCargaDocumental{Token: token, Carga: clonarCargaDocumentalMemoria(solicitud.Carga)}
	if resultado.Validar() != nil {
		return ports.ReservaCargaDocumental{}, ports.ErrReservaCargaDocumentalInvalida
	}
	// Punto de confirmacion logica: la respuesta y todos los asientos se han
	// construido y validado antes de publicar cualquiera de ellos.
	r.reservasPorIndice[solicitud.IndiceIdempotenciaHMAC] = reservaPersistible
	r.indicePorHuellaToken[huellaToken] = solicitud.IndiceIdempotenciaHMAC
	r.decisionesPreparacion[solicitud.DecisionPreparacion.DecisionRef] = reclamacionPersistible
	return resultado, nil
}

func (r *RepositorioCargasDocumentalesMemoria) ConfirmarPreparacion(
	ctx context.Context,
	solicitud ports.SolicitudConfirmarPreparacionCargaDocumental,
) error {
	if r == nil || ctx == nil {
		return ports.ErrConfirmacionCargaDocumentalInvalida
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	instantanea, err := ports.InstantaneaSolicitudConfirmarPreparacionCargaDocumental(solicitud)
	if err != nil {
		return ports.ErrConfirmacionCargaDocumentalInvalida
	}
	huellaToken, err := instantanea.Token.HuellaSHA256()
	if err != nil {
		return ports.ErrConfirmacionCargaDocumentalInvalida
	}
	consumoDecision, err := ports.ConsumoDecisionDesdeManifiestoPreparacionCargaDocumental(instantanea.Manifiesto)
	if err != nil {
		return ports.ErrConfirmacionCargaDocumentalInvalida
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	ahora := r.reloj.Ahora().UTC()
	indice, existe := r.indicePorHuellaToken[huellaToken]
	reserva, existeReserva := r.reservasPorIndice[indice]
	if ahora.IsZero() || !existe || !existeReserva || reserva.estado != estadoReservaCargaMemoriaActiva ||
		!instantanea.Token.CoincideConHuellaSHA256(reserva.huellaTokenSHA256) {
		return ports.ErrConfirmacionCargaDocumentalInvalida
	}
	if !ahora.Before(reserva.expiraEn) {
		if !r.marcarDecisionPreparacionFinalizada(reserva, estadoDecisionPreparacionCargaExpirada) {
			return ports.ErrConfirmacionCargaDocumentalInvalida
		}
		reserva.estado = estadoReservaCargaMemoriaExpirada
		reserva.huellaTokenSHA256 = ""
		r.reservasPorIndice[indice] = reserva
		delete(r.indicePorHuellaToken, huellaToken)
		return ports.ErrConfirmacionCargaDocumentalInvalida
	}
	if instantanea.Confirmacion.ValidarContra(reserva.carga) != nil ||
		instantanea.Manifiesto.ValidarContraCarga(instantanea.Confirmacion.Carga) != nil ||
		ahora.Before(instantanea.Confirmacion.Carga.PreparadaEn) {
		return ports.ErrConfirmacionCargaDocumentalInvalida
	}
	reclamacion, existeReclamacion := r.decisionesPreparacion[consumoDecision.DecisionRef]
	if !existeReclamacion || reclamacion.estado != estadoDecisionPreparacionCargaActiva ||
		reclamacion.consumo != consumoDecision || reserva.decisionPreparacion != consumoDecision ||
		reclamacion.indice != indice || reclamacion.huellaTokenSHA256 != reserva.huellaTokenSHA256 ||
		reclamacion.cargaID != reserva.carga.ID {
		if existeReclamacion && reclamacion.estado == estadoDecisionPreparacionCargaConsumida {
			return ports.ErrDecisionPreparacionCargaYaConsumida
		}
		return ports.ErrDecisionPreparacionCargaNoDisponible
	}
	id := instantanea.Confirmacion.Carga.ID
	if _, existe := r.cargasPorID[id]; existe {
		return ports.ErrConfirmacionCargaDocumentalInvalida
	}
	if _, existe := r.manifiestosPorID[id]; existe {
		return ports.ErrConfirmacionCargaDocumentalInvalida
	}
	// Un unico tramo critico representa el COMMIT atomico exigido por el
	// puerto. Si se anade persistencia con fallo inyectable, debe prepararse
	// antes de este punto o revertirse toda la transaccion.
	reserva.estado = estadoReservaCargaMemoriaConfirmada
	reserva.huellaTokenSHA256 = ""
	reserva.carga = instantanea.Confirmacion.Carga
	r.reservasPorIndice[indice] = reserva
	delete(r.indicePorHuellaToken, huellaToken)
	r.cargasPorID[id] = instantanea.Confirmacion.Carga
	r.manifiestosPorID[id] = instantanea.Manifiesto
	reclamacion.estado = estadoDecisionPreparacionCargaConsumida
	reclamacion.huellaTokenSHA256 = ""
	reclamacion.cargaID = id
	r.decisionesPreparacion[consumoDecision.DecisionRef] = reclamacion
	r.auditoria = append(r.auditoria, instantanea.Confirmacion.Auditoria)
	r.eventos = append(r.eventos, instantanea.Confirmacion.Evento)
	return nil
}

func (r *RepositorioCargasDocumentalesMemoria) ConfirmarTransicion(
	ctx context.Context,
	confirmacion ports.ConfirmacionTransicionCargaDocumental,
) error {
	if r == nil || ctx == nil {
		return ports.ErrConfirmacionCargaDocumentalInvalida
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	instantanea := ports.InstantaneaConfirmacionTransicionCargaDocumental(confirmacion)
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	anterior, existe := r.cargasPorID[instantanea.Carga.ID]
	if !existe || instantanea.ValidarContra(anterior) != nil {
		return ports.ErrConflictoVersionCargaDocumental
	}
	if _, existe := r.manifiestosPorID[instantanea.Carga.ID]; !existe {
		return ports.ErrManifiestoPreparacionNoEncontrado
	}
	r.cargasPorID[instantanea.Carga.ID] = instantanea.Carga
	r.auditoria = append(r.auditoria, instantanea.Auditoria)
	r.eventos = append(r.eventos, instantanea.Evento)
	if reserva, existe := r.reservasPorIndice[instantanea.Carga.IndiceIdempotenciaHMAC]; existe {
		reserva.carga = instantanea.Carga
		r.reservasPorIndice[instantanea.Carga.IndiceIdempotenciaHMAC] = reserva
	}
	return nil
}

func (r *RepositorioCargasDocumentalesMemoria) AbandonarReserva(
	ctx context.Context,
	token ports.TokenReservaCargaDocumental,
) error {
	if r == nil || ctx == nil {
		return ports.ErrReservaCargaDocumentalInvalida
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	huellaToken, err := token.HuellaSHA256()
	if err != nil {
		return ports.ErrReservaCargaDocumentalInvalida
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	ahora := r.reloj.Ahora().UTC()
	indice, existe := r.indicePorHuellaToken[huellaToken]
	reserva, existeReserva := r.reservasPorIndice[indice]
	if ahora.IsZero() || !existe || !existeReserva || reserva.estado != estadoReservaCargaMemoriaActiva ||
		!token.CoincideConHuellaSHA256(reserva.huellaTokenSHA256) {
		return ports.ErrReservaCargaDocumentalInvalida
	}
	if !ahora.Before(reserva.expiraEn) {
		if !r.marcarDecisionPreparacionFinalizada(reserva, estadoDecisionPreparacionCargaExpirada) {
			return ports.ErrReservaCargaDocumentalInvalida
		}
		reserva.estado = estadoReservaCargaMemoriaExpirada
		reserva.huellaTokenSHA256 = ""
		r.reservasPorIndice[indice] = reserva
		delete(r.indicePorHuellaToken, huellaToken)
		return ports.ErrReservaCargaDocumentalInvalida
	}
	if !r.marcarDecisionPreparacionFinalizada(reserva, estadoDecisionPreparacionCargaAbandonada) {
		return ports.ErrReservaCargaDocumentalInvalida
	}
	reserva.estado = estadoReservaCargaMemoriaAbandonada
	reserva.huellaTokenSHA256 = ""
	r.reservasPorIndice[indice] = reserva
	delete(r.indicePorHuellaToken, huellaToken)
	return nil
}

func (r *RepositorioCargasDocumentalesMemoria) Obtener(
	ctx context.Context,
	id string,
) (domain.CargaDocumental, error) {
	if r == nil || ctx == nil || ctx.Err() != nil || !referenciaCargaMemoriaValida(id) {
		return domain.CargaDocumental{}, ports.ErrCargaDocumentalNoEncontrada
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	carga, existe := r.cargasPorID[id]
	if !existe || carga.Validar() != nil {
		return domain.CargaDocumental{}, ports.ErrCargaDocumentalNoEncontrada
	}
	return clonarCargaDocumentalMemoria(carga), nil
}

func (r *RepositorioCargasDocumentalesMemoria) ObtenerPreparacion(
	ctx context.Context,
	id string,
) (ports.PreparacionCargaDocumentalPersistida, error) {
	if r == nil || ctx == nil || ctx.Err() != nil || !referenciaCargaMemoriaValida(id) {
		return ports.PreparacionCargaDocumentalPersistida{}, ports.ErrManifiestoPreparacionNoEncontrado
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	carga, existeCarga := r.cargasPorID[id]
	manifiesto, existeManifiesto := r.manifiestosPorID[id]
	resultado := ports.PreparacionCargaDocumentalPersistida{
		Carga: clonarCargaDocumentalMemoria(carga), Manifiesto: manifiesto,
	}
	if !existeCarga || !existeManifiesto || resultado.Validar() != nil {
		return ports.PreparacionCargaDocumentalPersistida{}, ports.ErrManifiestoPreparacionNoEncontrado
	}
	return resultado, nil
}

func referenciaCargaMemoriaValida(valor string) bool {
	return valor != "" && valor == strings.TrimSpace(valor) && len(valor) <= 512 &&
		!strings.ContainsAny(valor, "* \t\r\n")
}

// marcarDecisionPreparacionFinalizada se invoca con r.mu adquirido. Abandono
// y expiracion retiran definitivamente la DecisionRef: liberar una decision
// tras un efecto remoto ambiguo permitiria crear otra sesion con la misma
// autoridad. El reintento debe obtener una decision nueva.
func (r *RepositorioCargasDocumentalesMemoria) marcarDecisionPreparacionFinalizada(
	reserva reservaCargaDocumentalMemoria,
	estado estadoDecisionPreparacionCargaMemoria,
) bool {
	if estado != estadoDecisionPreparacionCargaAbandonada && estado != estadoDecisionPreparacionCargaExpirada {
		return false
	}
	referencia := reserva.decisionPreparacion.DecisionRef
	reclamacion, existe := r.decisionesPreparacion[referencia]
	if !existe || reclamacion.estado != estadoDecisionPreparacionCargaActiva ||
		reclamacion.consumo != reserva.decisionPreparacion || reclamacion.indice != reserva.indiceHMAC ||
		reclamacion.huellaTokenSHA256 != reserva.huellaTokenSHA256 || reclamacion.cargaID != reserva.carga.ID {
		return false
	}
	reclamacion.estado = estado
	reclamacion.huellaTokenSHA256 = ""
	r.decisionesPreparacion[referencia] = reclamacion
	return true
}

func clonarCargaDocumentalMemoria(carga domain.CargaDocumental) domain.CargaDocumental {
	clon := carga
	if carga.ContenidoCuarentena != nil {
		contenido := *carga.ContenidoCuarentena
		clon.ContenidoCuarentena = &contenido
	}
	if carga.Analisis != nil {
		analisis := *carga.Analisis
		clon.Analisis = &analisis
	}
	if carga.ContenidoAdmitido != nil {
		contenido := *carga.ContenidoAdmitido
		clon.ContenidoAdmitido = &contenido
	}
	return clon
}

var _ ports.RepositorioCargasDocumentales = (*RepositorioCargasDocumentalesMemoria)(nil)
