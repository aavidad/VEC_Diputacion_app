package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var instanteAltaCobroPrueba = time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)

type relojAltaCobroPrueba struct{ ahora time.Time }

func (r relojAltaCobroPrueba) Ahora() time.Time { return r.ahora }

type relojSecuencialAltaCobroPrueba struct {
	mu        sync.Mutex
	instantes []time.Time
	indice    int
}

func (r *relojSecuencialAltaCobroPrueba) Ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.instantes) == 0 {
		return time.Time{}
	}
	if r.indice >= len(r.instantes) {
		return r.instantes[len(r.instantes)-1]
	}
	instante := r.instantes[r.indice]
	r.indice++
	return instante
}

type revalidadorAltaCobroPrueba struct {
	resultado domain.AutenticacionRevalidadaV1
}

func (r revalidadorAltaCobroPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

type verificadorAltaCobroPrueba struct {
	mu        sync.Mutex
	resultado domain.ResultadoVerificacionAutenticacionCobro
	err       error
	llamadas  int
}

func (v *verificadorAltaCobroPrueba) VerificarAutenticacionCobro(
	ctx context.Context,
	solicitud domain.SolicitudVerificacionAutenticacionCobro,
) (domain.ResultadoVerificacionAutenticacionCobro, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.llamadas++
	if err := ctx.Err(); err != nil {
		return domain.ResultadoVerificacionAutenticacionCobro{}, err
	}
	if v.err != nil {
		return domain.ResultadoVerificacionAutenticacionCobro{}, v.err
	}
	return v.resultado, nil
}

type fuenteLiquidacionesAltaCobroPrueba struct {
	mu              sync.Mutex
	resultados      []LiquidacionCobroAutoritativa
	err             error
	consultas       []ConsultaLiquidacionCobro
	cancelar        context.CancelFunc
	despuesDeBuscar func()
}

func (f *fuenteLiquidacionesAltaCobroPrueba) BuscarLiquidacionesCobro(
	ctx context.Context,
	consulta ConsultaLiquidacionCobro,
) ([]LiquidacionCobroAutoritativa, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.consultas = append(f.consultas, consulta)
	if f.cancelar != nil {
		f.cancelar()
	}
	if f.err != nil {
		return nil, f.err
	}
	resultado := append([]LiquidacionCobroAutoritativa(nil), f.resultados...)
	if f.despuesDeBuscar != nil {
		f.despuesDeBuscar()
	}
	return resultado, ctx.Err()
}

type autorizadorAltaCobroPrueba struct {
	mu               sync.Mutex
	ahora            time.Time
	err              error
	llamadas         int
	secuencia        int
	vigenciaDecision time.Duration
	ultima           domain.SolicitudAutorizacion
	mutarSolicitud   func(*domain.SolicitudAutorizacion)
	mutarDecision    func(*domain.DecisionAutorizacion)
	despuesDeDecidir func(domain.DecisionAutorizacion)
}

func (a *autorizadorAltaCobroPrueba) Exigir(
	ctx context.Context,
	solicitud domain.SolicitudAutorizacion,
) (domain.DecisionAutorizacion, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.llamadas++
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	if a.err != nil {
		return domain.DecisionAutorizacion{}, a.err
	}
	if a.mutarSolicitud != nil {
		a.mutarSolicitud(&solicitud)
	}
	a.ultima = solicitud
	campos, conocida := domain.CamposRequeridosAccionCobro(domain.AccionCobro(solicitud.Accion))
	if !conocida {
		return domain.DecisionAutorizacion{}, domain.ErrAutorizacionDenegada
	}
	a.secuencia++
	vigencia := a.vigenciaDecision
	if vigencia == 0 {
		vigencia = time.Minute
	}
	decision := completarDecisionAutorizacionPrueba(solicitud, domain.DecisionAutorizacion{
		DecisionRef: fmt.Sprintf("decision:cobro:%d", a.secuencia), Concedida: true, Codigo: "concedida",
		PrincipalID: solicitud.Principal.ID, PerfilActivoRef: solicitud.PerfilActivoRef,
		Accion: solicitud.Accion, RecursoRef: solicitud.Recurso.Referencia,
		Finalidad: solicitud.Finalidad, CorrelacionRef: solicitud.CorrelacionRef,
		VinculoAutenticacionActor: solicitud.VinculoAutenticacionActor,
		GarantiaMinima:            domain.AuthAssuranceSubstantial, CamposPermitidos: campos,
		EmitidaEn: a.ahora, ValidaHasta: a.ahora.Add(vigencia),
	})
	if a.mutarDecision != nil {
		a.mutarDecision(&decision)
	}
	if a.despuesDeDecidir != nil {
		a.despuesDeDecidir(clonarDecisionAutorizacionAltaCobro(decision))
	}
	return decision, nil
}

type selladorAltaCobroPrueba struct{}

func (selladorAltaCobroPrueba) SellarIndiceAltaCobro(ctx context.Context, contenido []byte) (string, error) {
	return sellarAltaCobroPrueba(ctx, "pagos-v1", contenido)
}
func (selladorAltaCobroPrueba) SellarHuellaPeticionCobro(ctx context.Context, contenido []byte) (string, error) {
	return sellarAltaCobroPrueba(ctx, "peticion-v1", contenido)
}
func (selladorAltaCobroPrueba) SellarIndiceDevolucionCobro(ctx context.Context, contenido []byte) (string, error) {
	return sellarAltaCobroPrueba(ctx, "devoluciones-v1", contenido)
}

func sellarAltaCobroPrueba(ctx context.Context, dominio string, contenido []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	suma := sha256.Sum256(append([]byte("secreto-prueba\x00"+dominio+"\x00"), contenido...))
	return "hmac-sha256:" + dominio + ":" + hex.EncodeToString(suma[:]), nil
}

type generadorAltaCobroPrueba struct {
	mu        sync.Mutex
	siguiente int
	err       error
}

func (g *generadorAltaCobroPrueba) NuevoIDOrdenCobro() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.err != nil {
		return "", g.err
	}
	g.siguiente++
	return fmt.Sprintf("cob_%022d", g.siguiente), nil
}
func (g *generadorAltaCobroPrueba) NuevoIDDevolucionCobro() (string, error) {
	return "", errors.New("capacidad no habilitada en doble de alta")
}

// controlAutorizacionCommitAltaCobroPrueba representa las filas autoritativas
// que un adaptador duradero debe bloquear y comparar en su transaccion. No es
// otra politica: sus valores se materializan exclusivamente desde la evidencia
// emitida por el PDP y solo sirven para simular CAS, retirada y revocacion.
type controlAutorizacionCommitAltaCobroPrueba struct {
	inicializado  bool
	inconsistente bool
	decisiones    map[string]string

	principalRef              string
	perfilActivoRef           string
	asignacionRef             string
	asignacionHuellaSHA256    string
	asignacionActiva          bool
	versionRolRef             string
	versionRolHuellaSHA256    string
	versionRolActiva          bool
	controlRolRef             string
	controlRolRevision        uint64
	controlRolHuellaSHA256    string
	controlRolVigente         bool
	revisionCatalogo          uint64
	catalogoHuellaSHA256      string
	sesionRef                 string
	controlSesionRef          string
	controlSesionRevision     uint64
	controlSesionHuellaSHA256 string
	sesionValidaHasta         time.Time
	sesionActiva              bool
	contextoActorRef          string
	contextoActorVersion      uint64
	contextoActorHuellaSHA256 string
	contextoActorActivo       bool
}

func (c *controlAutorizacionCommitAltaCobroPrueba) incorporar(
	evidencia ports.EvidenciaUsoDecisionAutorizacion,
) error {
	datos, err := evidencia.Datos()
	if err != nil {
		return err
	}
	decision := datos.Decision
	vinculo, err := decision.VinculoAutenticacionActor.Datos()
	if err != nil {
		return err
	}
	if !c.inicializado {
		c.inicializado = true
		c.decisiones = make(map[string]string)
		c.principalRef = decision.PrincipalID
		c.perfilActivoRef = decision.PerfilActivoRef
		c.asignacionRef = decision.AsignacionRef
		c.asignacionHuellaSHA256 = decision.AsignacionHuellaSHA256
		c.asignacionActiva = true
		c.versionRolRef = decision.VersionRolRef
		c.versionRolHuellaSHA256 = decision.VersionRolHuellaSHA256
		c.versionRolActiva = true
		c.controlRolRef = decision.ControlVigenciaVersionRolRef
		c.controlRolRevision = decision.ControlVigenciaVersionRolRevision
		c.controlRolHuellaSHA256 = decision.ControlVigenciaVersionRolHuellaSHA256
		c.controlRolVigente = true
		c.revisionCatalogo = decision.RevisionCatalogoPoliticas
		c.catalogoHuellaSHA256 = decision.CatalogoPoliticasHuellaSHA256
		c.sesionRef = vinculo.SesionRef
		c.controlSesionRef = vinculo.ControlSesionRef
		c.controlSesionRevision = vinculo.ControlSesionRevision
		c.controlSesionHuellaSHA256 = vinculo.ControlSesionHuellaSHA256
		c.sesionValidaHasta = vinculo.SesionValidaHasta
		c.sesionActiva = true
		c.contextoActorRef = vinculo.ContextoActorRef
		c.contextoActorVersion = vinculo.ContextoActorVersion
		c.contextoActorHuellaSHA256 = vinculo.ContextoActorHuellaSHA256
		c.contextoActorActivo = true
	} else if c.principalRef != decision.PrincipalID ||
		c.perfilActivoRef != decision.PerfilActivoRef ||
		c.asignacionRef != decision.AsignacionRef ||
		c.asignacionHuellaSHA256 != decision.AsignacionHuellaSHA256 ||
		c.versionRolRef != decision.VersionRolRef ||
		c.versionRolHuellaSHA256 != decision.VersionRolHuellaSHA256 ||
		c.controlRolRef != decision.ControlVigenciaVersionRolRef ||
		c.controlRolRevision != decision.ControlVigenciaVersionRolRevision ||
		c.controlRolHuellaSHA256 != decision.ControlVigenciaVersionRolHuellaSHA256 ||
		c.revisionCatalogo != decision.RevisionCatalogoPoliticas ||
		c.catalogoHuellaSHA256 != decision.CatalogoPoliticasHuellaSHA256 ||
		c.sesionRef != vinculo.SesionRef || c.controlSesionRef != vinculo.ControlSesionRef ||
		c.controlSesionRevision != vinculo.ControlSesionRevision ||
		c.controlSesionHuellaSHA256 != vinculo.ControlSesionHuellaSHA256 ||
		!c.sesionValidaHasta.Equal(vinculo.SesionValidaHasta) ||
		c.contextoActorRef != vinculo.ContextoActorRef ||
		c.contextoActorVersion != vinculo.ContextoActorVersion ||
		c.contextoActorHuellaSHA256 != vinculo.ContextoActorHuellaSHA256 {
		c.inconsistente = true
		return ports.ErrControlAutorizacionCobroConflicto
	}
	c.decisiones[decision.DecisionRef] = datos.HuellaDecisionSHA256
	return nil
}

func (c controlAutorizacionCommitAltaCobroPrueba) coincide(
	datos ports.DatosEvidenciaUsoDecisionAutorizacion,
	ahora time.Time,
) bool {
	decision := datos.Decision
	vinculo, err := decision.VinculoAutenticacionActor.Datos()
	huellaDecision, decisionActiva := c.decisiones[decision.DecisionRef]
	return err == nil && c.inicializado && !c.inconsistente && decisionActiva &&
		huellaDecision == datos.HuellaDecisionSHA256 &&
		c.principalRef == decision.PrincipalID && c.perfilActivoRef == decision.PerfilActivoRef &&
		c.asignacionActiva && c.asignacionRef == decision.AsignacionRef &&
		c.asignacionHuellaSHA256 == decision.AsignacionHuellaSHA256 &&
		c.versionRolActiva && c.versionRolRef == decision.VersionRolRef &&
		c.versionRolHuellaSHA256 == decision.VersionRolHuellaSHA256 &&
		c.controlRolVigente && c.controlRolRef == decision.ControlVigenciaVersionRolRef &&
		c.controlRolRevision == decision.ControlVigenciaVersionRolRevision &&
		c.controlRolHuellaSHA256 == decision.ControlVigenciaVersionRolHuellaSHA256 &&
		c.revisionCatalogo == decision.RevisionCatalogoPoliticas &&
		c.catalogoHuellaSHA256 == decision.CatalogoPoliticasHuellaSHA256 &&
		c.sesionActiva && c.sesionRef == vinculo.SesionRef &&
		c.controlSesionRef == vinculo.ControlSesionRef &&
		c.controlSesionRevision == vinculo.ControlSesionRevision &&
		c.controlSesionHuellaSHA256 == vinculo.ControlSesionHuellaSHA256 &&
		c.sesionValidaHasta.Equal(vinculo.SesionValidaHasta) && ahora.Before(c.sesionValidaHasta) &&
		c.contextoActorActivo && c.contextoActorRef == vinculo.ContextoActorRef &&
		c.contextoActorVersion == vinculo.ContextoActorVersion &&
		c.contextoActorHuellaSHA256 == vinculo.ContextoActorHuellaSHA256
}

type consumoDecisionAltaCobroPrueba struct {
	ordenRef           string
	huellaEfectoSHA256 string
}

type repositorioAltaCobroPrueba struct {
	mu                     sync.Mutex
	reserva                *ports.SolicitudReservaOrdenCobro
	huellaTokenReserva     string
	ahoraCommit            time.Time
	liquidacionCommit      DatosLiquidacionCobroAutoritativa
	controlAutorizacion    controlAutorizacionCommitAltaCobroPrueba
	errControlAutorizacion error
	consumosDecision       map[string]consumoDecisionAltaCobroPrueba
	orden                  *domain.OrdenCobro
	auditoria              ports.RegistroAuditoriaCobro
	evento                 ports.EventoSalidaCobro
	ultimaConfirmacion     ports.SolicitudConfirmarCreacionOrdenCobro
	errReserva             error
	errConfirmacion        error
	errAbandono            error
	cancelarTrasReserva    context.CancelFunc
	reservas               int
	confirmaciones         int
	abandonos              int
}

func (r *repositorioAltaCobroPrueba) registrarDecisionAutorizacion(
	decision domain.DecisionAutorizacion,
	verificadaEn time.Time,
) {
	evidencia, err := ports.NuevaEvidenciaUsoDecisionAutorizacion(decision, verificadaEn)
	r.mu.Lock()
	defer r.mu.Unlock()
	if err != nil {
		r.errControlAutorizacion = err
		return
	}
	if err := r.controlAutorizacion.incorporar(evidencia); err != nil {
		r.errControlAutorizacion = err
	}
}

func (r *repositorioAltaCobroPrueba) mutarControlAutorizacion(
	mutar func(*controlAutorizacionCommitAltaCobroPrueba),
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	mutar(&r.controlAutorizacion)
}

func (r *repositorioAltaCobroPrueba) ReservarCreacion(
	ctx context.Context,
	solicitud ports.SolicitudReservaOrdenCobro,
) (ports.ReservaOrdenCobro, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reservas++
	if err := ctx.Err(); err != nil {
		return ports.ReservaOrdenCobro{}, err
	}
	if solicitud.Validar() != nil {
		return ports.ReservaOrdenCobro{}, ports.ErrReservaOrdenCobroInvalida
	}
	if r.errReserva != nil {
		return ports.ReservaOrdenCobro{}, r.errReserva
	}
	if r.orden != nil && r.reserva != nil {
		anterior := r.reserva
		if anterior.IndiceIdempotenciaHMAC != solicitud.IndiceIdempotenciaHMAC ||
			anterior.HuellaSolicitudHMAC != solicitud.HuellaSolicitudHMAC ||
			anterior.PrincipalRef != solicitud.PrincipalRef {
			return ports.ReservaOrdenCobro{}, ports.ErrIdempotenciaCobroReutilizada
		}
		clon := r.orden.Clonar()
		return ports.ReservaOrdenCobro{Repetida: true, Orden: &clon}, nil
	}
	if r.reserva != nil {
		return ports.ReservaOrdenCobro{}, ports.ErrOrdenCobroYaExiste
	}
	copia := solicitud
	r.reserva = &copia
	token, err := ports.NuevoTokenReservaOrdenCobro()
	if err != nil {
		return ports.ReservaOrdenCobro{}, ports.ErrReservaOrdenCobroInvalida
	}
	huellaToken, err := token.HuellaSHA256()
	if err != nil {
		return ports.ReservaOrdenCobro{}, ports.ErrReservaOrdenCobroInvalida
	}
	r.huellaTokenReserva = huellaToken
	if r.cancelarTrasReserva != nil {
		r.cancelarTrasReserva()
	}
	return ports.ReservaOrdenCobro{Token: token}, nil
}

func (r *repositorioAltaCobroPrueba) ConfirmarCreacion(
	ctx context.Context,
	solicitud ports.SolicitudConfirmarCreacionOrdenCobro,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.confirmaciones++
	if err := ctx.Err(); err != nil {
		return err
	}
	if solicitud.Validar() != nil || r.reserva == nil ||
		!solicitud.Token.CoincideConHuellaSHA256(r.huellaTokenReserva) ||
		solicitud.OrdenRef != r.reserva.OrdenRef ||
		solicitud.PrincipalRef != r.reserva.PrincipalRef ||
		solicitud.IndiceIdempotenciaHMAC != r.reserva.IndiceIdempotenciaHMAC ||
		solicitud.HuellaSolicitudHMAC != r.reserva.HuellaSolicitudHMAC ||
		!solicitud.ReservaSolicitadaEn.Equal(r.reserva.SolicitadaEn) ||
		!solicitud.ReservaExpiraEn.Equal(r.reserva.ExpiraEn) {
		return ports.ErrMutacionOrdenCobroInvalida
	}
	// Simula el reloj de la propia transaccion y el bloqueo/CAS del registro
	// oficial. No consulta de nuevo el doble FuenteLiquidaciones: una lectura
	// externa anterior no podria ofrecer esta garantia de commit.
	ahora := r.ahoraCommit
	if !instanteAplicacionCobroCanonico(ahora) || ahora.Before(solicitud.ReservaSolicitadaEn) ||
		!ahora.Before(solicitud.ReservaExpiraEn) {
		return ports.ErrReservaOrdenCobroCaducada
	}
	datosEvidencia, errEvidencia := solicitud.EvidenciaAutorizacion.Datos()
	if r.errControlAutorizacion != nil || errEvidencia != nil ||
		solicitud.EvidenciaAutorizacion.ValidarEn(ahora) != nil ||
		!ahora.Before(solicitud.DecisionValidaHasta) ||
		!ahora.Before(solicitud.SesionValidaHasta) ||
		!r.controlAutorizacion.coincide(datosEvidencia, ahora) {
		return ports.ErrControlAutorizacionCobroConflicto
	}
	liquidacion := r.liquidacionCommit
	huellaLiquidacion, errHuella := CalcularHuellaLiquidacionCobroAutoritativa(liquidacion)
	if errHuella != nil || huellaLiquidacion != liquidacion.HuellaSHA256 ||
		liquidacion.LiquidacionRef != solicitud.LiquidacionRef ||
		liquidacion.Revision != solicitud.LiquidacionRevision ||
		liquidacion.HuellaSHA256 != solicitud.LiquidacionHuellaSHA256 ||
		ports.EstadoControlLiquidacionCobro(liquidacion.Estado) != solicitud.LiquidacionEstado ||
		!liquidacion.ExigibleDesde.Equal(solicitud.LiquidacionExigibleDesde) ||
		!liquidacion.ExigibleHasta.Equal(solicitud.LiquidacionExigibleHasta) ||
		liquidacion.Estado != EstadoLiquidacionCobroExigible ||
		ahora.Before(liquidacion.ExigibleDesde) || !ahora.Before(liquidacion.ExigibleHasta) {
		return ports.ErrControlLiquidacionCobroConflicto
	}
	if r.errConfirmacion != nil {
		return r.errConfirmacion
	}
	datos, err := solicitud.Mutacion.Datos()
	if err != nil || datos.Orden.ID != r.reserva.OrdenRef || datos.Auditoria.Validar() != nil ||
		datos.Evento.Validar() != nil {
		return ports.ErrMutacionOrdenCobroInvalida
	}
	consumoEsperado := consumoDecisionAltaCobroPrueba{
		ordenRef: solicitud.OrdenRef, huellaEfectoSHA256: solicitud.HuellaEfectoSHA256,
	}
	if consumo, consumida := r.consumosDecision[solicitud.DecisionAutorizacionRef]; consumida {
		if consumo != consumoEsperado || r.orden == nil {
			return ports.ErrControlAutorizacionCobroConflicto
		}
		_, huellaExistente, errControl := r.orden.ControlConcurrencia()
		if errControl != nil || r.orden.ID != solicitud.OrdenRef ||
			huellaExistente != solicitud.HuellaEfectoSHA256 ||
			r.auditoria.ID != datos.Auditoria.ID || r.evento.ID != datos.Evento.ID {
			return ports.ErrControlAutorizacionCobroConflicto
		}
		// El efecto exacto ya existe: se consume la nueva reserva sin volver a
		// escribir agregado, auditoria ni outbox.
		r.huellaTokenReserva = ""
		return nil
	}
	if r.consumosDecision == nil {
		r.consumosDecision = make(map[string]consumoDecisionAltaCobroPrueba)
	}
	orden := datos.Orden.Clonar()
	r.consumosDecision[solicitud.DecisionAutorizacionRef] = consumoEsperado
	r.orden = &orden
	r.auditoria = datos.Auditoria
	r.evento = datos.Evento
	confirmacionAuditable := solicitud
	confirmacionAuditable.Token = ports.TokenReservaOrdenCobro{}
	r.ultimaConfirmacion = confirmacionAuditable
	// Se conserva solo la traza no secreta de la confirmacion; ni el material
	// de la capacidad ni su huella sobreviven al consumo.
	r.huellaTokenReserva = ""
	return nil
}

func (r *repositorioAltaCobroPrueba) AbandonarReservaCreacion(
	ctx context.Context,
	solicitud ports.SolicitudAbandonarReservaOrdenCobro,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.abandonos++
	if err := ctx.Err(); err != nil {
		return err
	}
	if solicitud.Validar() != nil || r.reserva == nil ||
		!solicitud.Token.CoincideConHuellaSHA256(r.huellaTokenReserva) ||
		solicitud.OrdenRef != r.reserva.OrdenRef || solicitud.PrincipalRef != r.reserva.PrincipalRef ||
		solicitud.HuellaSolicitudHMAC != r.reserva.HuellaSolicitudHMAC {
		return ports.ErrReservaOrdenCobroInvalida
	}
	if r.errAbandono != nil {
		return r.errAbandono
	}
	r.reserva = nil
	r.huellaTokenReserva = ""
	return nil
}

func (*repositorioAltaCobroPrueba) ReservarDevolucion(
	context.Context, ports.SolicitudReservaDevolucionCobro,
) (ports.ReservaDevolucionCobro, error) {
	return ports.ReservaDevolucionCobro{}, ports.ErrCapacidadPasarelaCobroNoDisponible
}
func (*repositorioAltaCobroPrueba) ConfirmarDevolucion(
	context.Context, ports.SolicitudConfirmarReservaDevolucionCobro,
) error {
	return ports.ErrCapacidadPasarelaCobroNoDisponible
}
func (*repositorioAltaCobroPrueba) AbandonarReservaDevolucion(
	context.Context, ports.SolicitudAbandonarReservaDevolucionCobro,
) error {
	return ports.ErrCapacidadPasarelaCobroNoDisponible
}
func (r *repositorioAltaCobroPrueba) ObtenerOrden(context.Context, string) (domain.OrdenCobro, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.orden == nil {
		return domain.OrdenCobro{}, ports.ErrOrdenCobroNoEncontrada
	}
	return r.orden.Clonar(), nil
}
func (*repositorioAltaCobroPrueba) ObtenerOrdenPorOperacion(
	context.Context, ports.ReferenciaOperacionCobro,
) (domain.OrdenCobro, error) {
	return domain.OrdenCobro{}, ports.ErrOrdenCobroNoEncontrada
}
func (*repositorioAltaCobroPrueba) ConfirmarTransicion(
	context.Context, ports.SolicitudConfirmarTransicionOrdenCobro,
) error {
	return ports.ErrCapacidadPasarelaCobroNoDisponible
}

func contextoAltaCobroPrueba(t *testing.T) (
	domain.ContextoActor,
	domain.VinculoAutenticacionActorV1,
	domain.AutenticacionRevalidadaV1,
) {
	return contextoAltaCobroPruebaConVigenciaSesion(t, 10*time.Minute)
}

func contextoAltaCobroPruebaConVigenciaSesion(
	t *testing.T,
	vigenciaSesion time.Duration,
) (
	domain.ContextoActor,
	domain.VinculoAutenticacionActorV1,
	domain.AutenticacionRevalidadaV1,
) {
	t.Helper()
	ahora := instanteAltaCobroPrueba
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl", Metodo: domain.AuthMethodCertificate,
		Garantia: domain.AuthAssuranceHigh,
	}
	instantanea := domain.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 3,
		CuentaRef: cuenta.CuentaRef, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 5,
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 7,
		Estado:       domain.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
		Vinculos: []domain.VinculoReferenciaContextoActor{{
			VinculoRef: "vin_0123456789abcdefghijkl", Version: 2,
			Tipo: domain.TipoReferenciaContextoActorCandidato, Referencia: "can_0123456789abcdefghijkl",
			Estado:       domain.EstadoVinculoContextoActorActivo,
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
		}},
	}
	actor, err := domain.NuevoContextoActor(cuenta, instantanea, ahora.Add(-time.Minute))
	if err != nil {
		t.Fatalf("NuevoContextoActor() error = %v", err)
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef: "aut_0123456789abcdefghijkl", AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef: "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 4,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64),
		CuentaRef:                 cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef,
		Superficie:      domain.SuperficieAutenticacionExternaPersonalV1,
		MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-11 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-10 * time.Minute),
		SesionRevalidadaEn:           ahora.Add(-2 * time.Minute), SesionValidaHasta: ahora.Add(vigenciaSesion),
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV1(
		context.Background(), revalidadorAltaCobroPrueba{resultado: autenticacion},
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef,
		}, actor, ahora,
	)
	if err != nil {
		t.Fatalf("CrearVinculoAutenticacionActorV1() error = %v", err)
	}
	return actor, vinculo, autenticacion
}

func vinculoOtroPerfilMismaSesionAltaCobroPrueba(
	t *testing.T,
	solicitud SolicitudAltaOrdenCobro,
) domain.VinculoAutenticacionActorV1 {
	t.Helper()
	datosVinculo, err := solicitud.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatalf("Vinculo.Datos() error = %v", err)
	}
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: datosVinculo.CuentaRef, Metodo: datosVinculo.MetodoObservado,
		Garantia: datosVinculo.GarantiaObservada,
	}
	instantanea := solicitud.ContextoActor.Instantanea
	instantanea.VinculoRef = "vca_abcdefghijkl0123456789"
	instantanea.VinculoVersion++
	instantanea.PerfilActivoRef = "prf_abcdefghijkl0123456789"
	instantanea.PerfilVersion++
	otroActor, err := domain.NuevoContextoActor(cuenta, instantanea, solicitud.ContextoActor.ResueltoEn)
	if err != nil {
		t.Fatalf("NuevoContextoActor(otro perfil) error = %v", err)
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV1(
		context.Background(), revalidadorAltaCobroPrueba{resultado: datosVinculo.Autenticacion()},
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: datosVinculo.AutenticacionRef, SesionRef: datosVinculo.SesionRef,
		}, otroActor, instanteAltaCobroPrueba,
	)
	if err != nil {
		t.Fatalf("CrearVinculoAutenticacionActorV1(otro perfil) error = %v", err)
	}
	return vinculo
}

func liquidacionAltaCobroPrueba(
	t *testing.T,
	actor domain.ContextoActor,
	estado EstadoLiquidacionCobro,
) LiquidacionCobroAutoritativa {
	t.Helper()
	datos := DatosLiquidacionCobroAutoritativa{
		LiquidacionRef: "liquidacion:bolsa:2026:1", Revision: 3,
		ExpedienteRef: "expediente:bolsa:2026:1", SolicitudRef: "solicitud:bolsa:2026:1",
		Tarifa: domain.ReferenciaTarifaCobro{
			TarifaID: "tasa_bolsa", Version: 4, HuellaSHA256: strings.Repeat("a", 64),
			ReglaCalculoRef: "regla:tasa_bolsa:v4",
		},
		SujetoRef: actor.PersonaRef, Importe: domain.DineroCobro{UnidadesMenores: 2500, Moneda: "EUR"},
		Concepto: "Tasa de inscripcion en bolsa", Finalidad: "inscripcion_bolsa",
		Estado: estado, ExigibleDesde: instanteAltaCobroPrueba.Add(-time.Hour),
		ExigibleHasta: instanteAltaCobroPrueba.Add(24 * time.Hour),
	}
	huella, err := CalcularHuellaLiquidacionCobroAutoritativa(datos)
	if err != nil {
		t.Fatalf("CalcularHuellaLiquidacionCobroAutoritativa() error = %v", err)
	}
	datos.HuellaSHA256 = huella
	liquidacion, err := NuevaLiquidacionCobroAutoritativa(datos)
	if err != nil {
		t.Fatalf("NuevaLiquidacionCobroAutoritativa() error = %v", err)
	}
	return liquidacion
}

func rehacerLiquidacionAltaCobroPrueba(
	t *testing.T,
	datos DatosLiquidacionCobroAutoritativa,
) (DatosLiquidacionCobroAutoritativa, LiquidacionCobroAutoritativa) {
	t.Helper()
	huella, err := CalcularHuellaLiquidacionCobroAutoritativa(datos)
	if err != nil {
		t.Fatalf("CalcularHuellaLiquidacionCobroAutoritativa() error = %v", err)
	}
	datos.HuellaSHA256 = huella
	liquidacion, err := NuevaLiquidacionCobroAutoritativa(datos)
	if err != nil {
		t.Fatalf("NuevaLiquidacionCobroAutoritativa() error = %v", err)
	}
	return datos, liquidacion
}

type escenarioAltaCobroPrueba struct {
	servicio      *ServicioAltaOrdenCobro
	repositorio   *repositorioAltaCobroPrueba
	liquidaciones *fuenteLiquidacionesAltaCobroPrueba
	autorizador   *autorizadorAltaCobroPrueba
	verificador   *verificadorAltaCobroPrueba
	generador     *generadorAltaCobroPrueba
	solicitud     SolicitudAltaOrdenCobro
}

func nuevoEscenarioAltaCobroPrueba(t *testing.T) escenarioAltaCobroPrueba {
	return nuevoEscenarioAltaCobroPruebaConVigencias(t, 10*time.Minute, time.Minute)
}

func nuevoEscenarioAltaCobroPruebaConVigencias(
	t *testing.T,
	vigenciaSesion, vigenciaDecision time.Duration,
) escenarioAltaCobroPrueba {
	t.Helper()
	actor, vinculo, autenticacion := contextoAltaCobroPruebaConVigenciaSesion(t, vigenciaSesion)
	datosVinculo, err := vinculo.Datos()
	if err != nil {
		t.Fatalf("Vinculo.Datos() error = %v", err)
	}
	liquidacion := liquidacionAltaCobroPrueba(t, actor, EstadoLiquidacionCobroExigible)
	datosLiquidacion, err := liquidacion.Datos()
	if err != nil {
		t.Fatalf("Liquidacion.Datos() error = %v", err)
	}
	repositorio := &repositorioAltaCobroPrueba{
		ahoraCommit: instanteAltaCobroPrueba, liquidacionCommit: datosLiquidacion,
	}
	liquidaciones := &fuenteLiquidacionesAltaCobroPrueba{
		resultados: []LiquidacionCobroAutoritativa{liquidacion},
	}
	autorizador := &autorizadorAltaCobroPrueba{
		ahora: instanteAltaCobroPrueba, vigenciaDecision: vigenciaDecision,
	}
	autorizador.despuesDeDecidir = func(decision domain.DecisionAutorizacion) {
		repositorio.registrarDecisionAutorizacion(decision, instanteAltaCobroPrueba)
	}
	verificador := &verificadorAltaCobroPrueba{resultado: domain.ResultadoVerificacionAutenticacionCobro{
		PrincipalRef: actor.PersonaRef, Metodo: autenticacion.MetodoObservado,
		Garantia: autenticacion.GarantiaObservada, AutenticacionRef: datosVinculo.AutenticacionRef,
		SesionRef:        datosVinculo.SesionRef,
		HuellaSesionHMAC: "hmac-sha256:sesion-v1:" + strings.Repeat("4", 64),
		EmitidaEn:        instanteAltaCobroPrueba.Add(-10 * time.Minute),
		ValidaHasta:      instanteAltaCobroPrueba.Add(10 * time.Minute),
	}}
	generador := &generadorAltaCobroPrueba{}
	servicio, err := NuevoServicioAltaOrdenCobro(
		repositorio, liquidaciones, autorizador, verificador, selladorAltaCobroPrueba{},
		generador, relojAltaCobroPrueba{ahora: instanteAltaCobroPrueba},
	)
	if err != nil {
		t.Fatalf("NuevoServicioAltaOrdenCobro() error = %v", err)
	}
	return escenarioAltaCobroPrueba{
		servicio: servicio, repositorio: repositorio, liquidaciones: liquidaciones,
		autorizador: autorizador, verificador: verificador, generador: generador,
		solicitud: SolicitudAltaOrdenCobro{
			ContextoActor: actor, VinculoAutenticacionActor: vinculo,
			SesionRef: datosVinculo.SesionRef, HuellaSesionHMAC: verificador.resultado.HuellaSesionHMAC,
			LiquidacionRef: "liquidacion:bolsa:2026:1", CorrelacionRef: "correlacion:cobro:2026:1",
		},
	}
}

func TestServicioAltaOrdenCobroCreaMutacionAtomicaDesdeLiquidacionAutoritativa(t *testing.T) {
	escenario := nuevoEscenarioAltaCobroPrueba(t)
	resultado, err := escenario.servicio.Crear(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatalf("Crear() error = %v", err)
	}
	datos, err := resultado.Datos()
	if err != nil {
		t.Fatalf("resultado.Datos() error = %v", err)
	}
	if datos.Repetida || datos.Vista.Importe != (domain.DineroCobro{UnidadesMenores: 2500, Moneda: "EUR"}) ||
		datos.Vista.Estado != domain.EstadoCobroCreada {
		t.Fatalf("resultado inesperado: %+v", datos)
	}
	if escenario.repositorio.confirmaciones != 1 || escenario.repositorio.abandonos != 0 ||
		escenario.repositorio.orden == nil || escenario.repositorio.auditoria.Validar() != nil ||
		escenario.repositorio.evento.Validar() != nil {
		t.Fatalf("la mutacion atomica no quedo completa: repo=%+v", escenario.repositorio)
	}
	orden := escenario.repositorio.orden
	if orden.Importe != datos.Vista.Importe || orden.Tarifa.TarifaID != "tasa_bolsa" ||
		orden.Historial[0].HuellaEvidenciaSHA256 == "" ||
		escenario.repositorio.auditoria.Accion != domain.AccionCobroCrearOrden ||
		escenario.repositorio.evento.Tipo != ports.EventoCobroOrdenCreada {
		t.Fatalf("orden/auditoria/outbox no derivados de la misma mutacion")
	}
	if escenario.autorizador.ultima.Accion != string(domain.AccionCobroCrearOrden) ||
		escenario.autorizador.ultima.Recurso.Referencia != escenario.solicitud.LiquidacionRef ||
		escenario.autorizador.ultima.Recurso.Atributos["estado_liquidacion"] != "exigible" ||
		escenario.autorizador.ultima.Recurso.Atributos["liquidacion_huella_sha256"] == "" {
		t.Fatalf("solicitud de autorizacion no ligada a la instantanea: %+v", escenario.autorizador.ultima)
	}
}

func TestServicioAltaOrdenCobroEsIdempotenteSinDuplicarAuditoriaNiOutbox(t *testing.T) {
	escenario := nuevoEscenarioAltaCobroPrueba(t)
	primero, err := escenario.servicio.Crear(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatalf("primer Crear() error = %v", err)
	}
	segundo, err := escenario.servicio.Crear(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatalf("segundo Crear() error = %v", err)
	}
	datosPrimero, _ := primero.Datos()
	datosSegundo, _ := segundo.Datos()
	if datosPrimero.Vista != datosSegundo.Vista || datosPrimero.Repetida || !datosSegundo.Repetida {
		t.Fatalf("reintento no idempotente: primero=%+v segundo=%+v", datosPrimero, datosSegundo)
	}
	if escenario.repositorio.reservas != 2 || escenario.repositorio.confirmaciones != 1 ||
		escenario.repositorio.abandonos != 0 || escenario.generador.siguiente != 2 {
		t.Fatalf("el reintento produjo efectos adicionales: repo=%+v ids=%d", escenario.repositorio, escenario.generador.siguiente)
	}
}

func TestRepositorioAltaCobroNoReutilizaDecisionParaOtroEfecto(t *testing.T) {
	escenario := nuevoEscenarioAltaCobroPrueba(t)
	if _, err := escenario.servicio.Crear(context.Background(), escenario.solicitud); err != nil {
		t.Fatalf("Crear() error = %v", err)
	}
	escenario.repositorio.mu.Lock()
	primeraConfirmacion := escenario.repositorio.ultimaConfirmacion
	primeraOrden := escenario.repositorio.orden.Clonar()
	primeraAuditoriaID := escenario.repositorio.auditoria.ID
	primerEventoID := escenario.repositorio.evento.ID
	escenario.repositorio.mu.Unlock()

	hechoCreacion := primeraOrden.Historial[0]
	altaDistinta := domain.AltaOrdenCobro{
		ID:                     "cob_abcdefghijkl0123456789",
		IndiceIdempotenciaHMAC: "hmac-sha256:pagos-v1:" + strings.Repeat("8", 64),
		ExpedienteRef:          primeraOrden.ExpedienteRef,
		SolicitudRef:           primeraOrden.SolicitudRef,
		LiquidacionRef:         primeraOrden.LiquidacionRef,
		Tarifa:                 primeraOrden.Tarifa,
		SujetoRef:              primeraOrden.SujetoRef,
		RepresentacionRef:      primeraOrden.RepresentacionRef,
		Importe:                primeraOrden.Importe,
		Concepto:               primeraOrden.Concepto,
		Finalidad:              primeraOrden.Finalidad,
		CorrelacionRef:         primeraOrden.CorrelacionRef,
		CreadaEn:               primeraOrden.CreadaEn,
		CaducaEn:               primeraOrden.CaducaEn,
		EvidenciaCreacionRef:   hechoCreacion.EvidenciaRef,
		HuellaEvidenciaSHA256:  hechoCreacion.HuellaEvidenciaSHA256,
		Motivo:                 hechoCreacion.Motivo,
	}
	ordenDistinta, err := domain.NuevaOrdenCobro(
		altaDistinta,
		primeraConfirmacion.ContextoAutorizacion,
	)
	if err != nil {
		t.Fatalf("crear segundo efecto valido: %v", err)
	}
	mutacionDistinta, err := ports.NuevaMutacionOrdenCobro(ordenDistinta)
	if err != nil {
		t.Fatalf("crear segunda mutacion valida: %v", err)
	}
	_, huellaEfectoDistinto, err := ordenDistinta.ControlConcurrencia()
	if err != nil {
		t.Fatalf("huella segundo efecto: %v", err)
	}
	segundaConfirmacion := primeraConfirmacion
	segundoToken, err := ports.NuevoTokenReservaOrdenCobro()
	if err != nil {
		t.Fatalf("generar capacidad para segundo efecto: %v", err)
	}
	segundaConfirmacion.Token = segundoToken
	segundaConfirmacion.OrdenRef = ordenDistinta.ID
	segundaConfirmacion.IndiceIdempotenciaHMAC = ordenDistinta.IndiceIdempotenciaHMAC
	segundaConfirmacion.HuellaSolicitudHMAC = "hmac-sha256:peticion-v1:" + strings.Repeat("7", 64)
	segundaConfirmacion.HuellaEfectoSHA256 = huellaEfectoDistinto
	segundaConfirmacion.Mutacion = mutacionDistinta
	if err := segundaConfirmacion.Validar(); err != nil {
		t.Fatalf("precondicion de segundo efecto valido: %v", err)
	}
	reservaDistinta := ports.SolicitudReservaOrdenCobro{
		OrdenRef: ordenDistinta.ID, IndiceIdempotenciaHMAC: ordenDistinta.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC: segundaConfirmacion.HuellaSolicitudHMAC,
		PrincipalRef:        primeraConfirmacion.PrincipalRef,
		SolicitadaEn:        primeraConfirmacion.ReservaSolicitadaEn,
		ExpiraEn:            primeraConfirmacion.ReservaExpiraEn,
	}
	if err := reservaDistinta.Validar(); err != nil {
		t.Fatalf("reserva del segundo efecto valida: %v", err)
	}
	escenario.repositorio.mu.Lock()
	escenario.repositorio.reserva = &reservaDistinta
	escenario.repositorio.huellaTokenReserva, err = segundaConfirmacion.Token.HuellaSHA256()
	escenario.repositorio.mu.Unlock()
	if err != nil {
		t.Fatalf("obtener huella de capacidad para segundo efecto: %v", err)
	}

	err = escenario.repositorio.ConfirmarCreacion(context.Background(), segundaConfirmacion)
	if !errors.Is(err, ports.ErrControlAutorizacionCobroConflicto) {
		t.Fatalf("reutilizacion de DecisionRef error = %v", err)
	}
	escenario.repositorio.mu.Lock()
	defer escenario.repositorio.mu.Unlock()
	if escenario.repositorio.orden == nil || escenario.repositorio.orden.ID != primeraOrden.ID ||
		escenario.repositorio.auditoria.ID != primeraAuditoriaID ||
		escenario.repositorio.evento.ID != primerEventoID ||
		len(escenario.repositorio.consumosDecision) != 1 {
		t.Fatalf("reutilizar la decision altero el primer efecto: %+v", escenario.repositorio)
	}
}

func TestServicioAltaOrdenCobroConcurrenteNoDuplicaEfectos(t *testing.T) {
	escenario := nuevoEscenarioAltaCobroPrueba(t)
	type resultadoConcurrente struct {
		alta AltaOrdenCobroCompletada
		err  error
	}
	resultados := make(chan resultadoConcurrente, 2)
	inicio := make(chan struct{})
	var grupo sync.WaitGroup
	for i := 0; i < 2; i++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			<-inicio
			alta, err := escenario.servicio.Crear(context.Background(), escenario.solicitud)
			resultados <- resultadoConcurrente{alta: alta, err: err}
		}()
	}
	close(inicio)
	grupo.Wait()
	close(resultados)

	exitos := 0
	for resultado := range resultados {
		if resultado.err == nil {
			exitos++
			if _, err := resultado.alta.Datos(); err != nil {
				t.Fatalf("resultado concurrente invalido: %v", err)
			}
			continue
		}
		if !errors.Is(resultado.err, ErrPersistenciaAltaCobroIncompleta) ||
			!errors.Is(resultado.err, ports.ErrOrdenCobroYaExiste) {
			t.Fatalf("error concurrente inesperado: %v", resultado.err)
		}
	}
	if exitos < 1 {
		t.Fatal("ninguna alta concurrente completo")
	}
	escenario.repositorio.mu.Lock()
	defer escenario.repositorio.mu.Unlock()
	if escenario.repositorio.orden == nil || escenario.repositorio.confirmaciones != 1 ||
		escenario.repositorio.auditoria.Validar() != nil || escenario.repositorio.evento.Validar() != nil {
		t.Fatalf("las altas concurrentes duplicaron o perdieron efectos: %+v", escenario.repositorio)
	}
}

func TestRepositorioAltaCobroRechazaColisionDeClaveConPeticionDistinta(t *testing.T) {
	escenario := nuevoEscenarioAltaCobroPrueba(t)
	if _, err := escenario.servicio.Crear(context.Background(), escenario.solicitud); err != nil {
		t.Fatalf("Crear() error = %v", err)
	}
	escenario.repositorio.mu.Lock()
	colision := *escenario.repositorio.reserva
	confirmaciones := escenario.repositorio.confirmaciones
	auditoriaID := escenario.repositorio.auditoria.ID
	eventoID := escenario.repositorio.evento.ID
	escenario.repositorio.mu.Unlock()
	colision.OrdenRef = "cob_abcdefghijkl0123456789"
	colision.HuellaSolicitudHMAC = "hmac-sha256:peticion-v1:" + strings.Repeat("8", 64)

	_, err := escenario.repositorio.ReservarCreacion(context.Background(), colision)
	if !errors.Is(err, ports.ErrIdempotenciaCobroReutilizada) {
		t.Fatalf("ReservarCreacion(colision) error = %v", err)
	}
	escenario.repositorio.mu.Lock()
	defer escenario.repositorio.mu.Unlock()
	if escenario.repositorio.confirmaciones != confirmaciones ||
		escenario.repositorio.auditoria.ID != auditoriaID || escenario.repositorio.evento.ID != eventoID {
		t.Fatalf("la colision altero efectos confirmados: %+v", escenario.repositorio)
	}
}

func TestServicioAltaOrdenCobroDeniegaAusenciaAmbiguedadYEstadoNoExigible(t *testing.T) {
	prueba := func(t *testing.T, preparar func(*escenarioAltaCobroPrueba)) {
		t.Helper()
		escenario := nuevoEscenarioAltaCobroPrueba(t)
		preparar(&escenario)
		_, err := escenario.servicio.Crear(context.Background(), escenario.solicitud)
		if !errors.Is(err, domain.ErrAutorizacionDenegada) {
			t.Fatalf("Crear() error = %v; se esperaba denegacion", err)
		}
		if escenario.autorizador.llamadas != 0 || escenario.repositorio.reservas != 0 ||
			escenario.repositorio.confirmaciones != 0 {
			t.Fatalf("una liquidacion no resoluble produjo efectos")
		}
	}
	t.Run("ausente", func(t *testing.T) {
		prueba(t, func(e *escenarioAltaCobroPrueba) { e.liquidaciones.resultados = nil })
	})
	t.Run("ambigua", func(t *testing.T) {
		prueba(t, func(e *escenarioAltaCobroPrueba) {
			e.liquidaciones.resultados = append(e.liquidaciones.resultados, e.liquidaciones.resultados[0])
		})
	})
	t.Run("sin estado", func(t *testing.T) {
		prueba(t, func(e *escenarioAltaCobroPrueba) {
			e.liquidaciones.resultados = []LiquidacionCobroAutoritativa{{}}
		})
	})
	t.Run("suspendida", func(t *testing.T) {
		prueba(t, func(e *escenarioAltaCobroPrueba) {
			e.liquidaciones.resultados = []LiquidacionCobroAutoritativa{
				liquidacionAltaCobroPrueba(t, e.solicitud.ContextoActor, EstadoLiquidacionCobroSuspendida),
			}
		})
	})
}

func TestServicioAltaOrdenCobroPrimerCorteDeniegaTerceroYRepresentacion(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		muta   func(*DatosLiquidacionCobroAutoritativa)
	}{
		{"otro sujeto", func(d *DatosLiquidacionCobroAutoritativa) { d.SujetoRef = "per_abcdefghijkl0123456789" }},
		{"representacion", func(d *DatosLiquidacionCobroAutoritativa) { d.RepresentacionRef = "rep_0123456789abcdefghijkl" }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioAltaCobroPrueba(t)
			datos, _ := escenario.liquidaciones.resultados[0].Datos()
			caso.muta(&datos)
			huella, err := CalcularHuellaLiquidacionCobroAutoritativa(datos)
			if err != nil {
				t.Fatalf("calcular huella: %v", err)
			}
			datos.HuellaSHA256 = huella
			liquidacion, err := NuevaLiquidacionCobroAutoritativa(datos)
			if err != nil {
				t.Fatalf("crear liquidacion: %v", err)
			}
			escenario.liquidaciones.resultados = []LiquidacionCobroAutoritativa{liquidacion}
			_, err = escenario.servicio.Crear(context.Background(), escenario.solicitud)
			if !errors.Is(err, domain.ErrAutorizacionDenegada) || escenario.autorizador.llamadas != 0 ||
				escenario.repositorio.reservas != 0 {
				t.Fatalf("Crear() = %v; el pago no acreditado no se cerro", err)
			}
		})
	}
}

func TestServicioAltaOrdenCobroCruzaSesionVinculoYAtestacion(t *testing.T) {
	t.Run("sesion distinta del vinculo", func(t *testing.T) {
		escenario := nuevoEscenarioAltaCobroPrueba(t)
		escenario.solicitud.SesionRef = "ses_abcdefghijkl0123456789"
		_, err := escenario.servicio.Crear(context.Background(), escenario.solicitud)
		if !errors.Is(err, domain.ErrAutorizacionDenegada) || escenario.verificador.llamadas != 0 ||
			len(escenario.liquidaciones.consultas) != 0 {
			t.Fatalf("Crear() = %v; sesion divergente alcanzo una autoridad", err)
		}
	})
	t.Run("atestacion de otra autenticacion", func(t *testing.T) {
		escenario := nuevoEscenarioAltaCobroPrueba(t)
		escenario.verificador.resultado.AutenticacionRef = "aut_abcdefghijkl0123456789"
		_, err := escenario.servicio.Crear(context.Background(), escenario.solicitud)
		if !errors.Is(err, domain.ErrAutorizacionDenegada) || escenario.repositorio.reservas != 0 {
			t.Fatalf("Crear() = %v; se mezclaron autenticaciones", err)
		}
	})
	t.Run("decision de otro contexto y perfil en la misma sesion", func(t *testing.T) {
		escenario := nuevoEscenarioAltaCobroPrueba(t)
		otroVinculo := vinculoOtroPerfilMismaSesionAltaCobroPrueba(t, escenario.solicitud)
		datosOriginales, _ := escenario.solicitud.VinculoAutenticacionActor.Datos()
		datosOtros, _ := otroVinculo.Datos()
		if datosOriginales.SesionRef != datosOtros.SesionRef ||
			datosOriginales.PerfilActivoRef == datosOtros.PerfilActivoRef ||
			escenario.solicitud.VinculoAutenticacionActor.CoincideExactamenteCon(otroVinculo) {
			t.Fatal("la precondicion no representa dos contextos distintos de la misma sesion")
		}
		escenario.autorizador.mutarDecision = func(d *domain.DecisionAutorizacion) {
			d.VinculoAutenticacionActor = otroVinculo
		}
		_, err := escenario.servicio.Crear(context.Background(), escenario.solicitud)
		if !errors.Is(err, domain.ErrAutorizacionDenegada) || escenario.repositorio.reservas != 0 ||
			escenario.repositorio.confirmaciones != 0 {
			t.Fatalf("Crear() = %v; se acepto una decision ligada a otro contexto V1", err)
		}
	})
}

func TestServicioAltaOrdenCobroDeniegaCamposObligacionesYContextoNoExactos(t *testing.T) {
	casos := []struct {
		nombre         string
		mutarDecision  func(*domain.DecisionAutorizacion)
		mutarSolicitud func(*domain.SolicitudAutorizacion)
	}{
		{
			nombre: "campo ausente",
			mutarDecision: func(d *domain.DecisionAutorizacion) {
				d.CamposPermitidos = d.CamposPermitidos[1:]
			},
		},
		{
			nombre: "campo desconocido",
			mutarDecision: func(d *domain.DecisionAutorizacion) {
				d.CamposPermitidos = append(d.CamposPermitidos, "orden.capacidad_desconocida")
			},
		},
		{
			nombre: "obligacion no implementada",
			mutarDecision: func(d *domain.DecisionAutorizacion) {
				d.Obligaciones = []string{"doble_validacion_pendiente"}
			},
		},
		{
			nombre: "recurso mutado por conector",
			mutarSolicitud: func(s *domain.SolicitudAutorizacion) {
				s.Recurso.Atributos["estado_liquidacion"] = "pagada"
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioAltaCobroPrueba(t)
			escenario.autorizador.mutarDecision = caso.mutarDecision
			escenario.autorizador.mutarSolicitud = caso.mutarSolicitud
			_, err := escenario.servicio.Crear(context.Background(), escenario.solicitud)
			if !errors.Is(err, domain.ErrAutorizacionDenegada) || escenario.repositorio.reservas != 0 ||
				escenario.repositorio.confirmaciones != 0 {
				t.Fatalf("Crear() = %v; concesion no exacta produjo efectos", err)
			}
		})
	}
}

func TestServicioAltaOrdenCobroCancelaYAbandonaReservaSinEfectos(t *testing.T) {
	t.Run("cancelacion tras reserva", func(t *testing.T) {
		escenario := nuevoEscenarioAltaCobroPrueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		escenario.repositorio.cancelarTrasReserva = cancelar
		_, err := escenario.servicio.Crear(ctx, escenario.solicitud)
		if !errors.Is(err, context.Canceled) || !errors.Is(err, ErrPersistenciaAltaCobroIncompleta) {
			t.Fatalf("Crear() error = %v", err)
		}
		if escenario.repositorio.abandonos != 1 || escenario.repositorio.confirmaciones != 0 ||
			escenario.repositorio.orden != nil {
			t.Fatalf("la cancelacion dejo efectos: %+v", escenario.repositorio)
		}
	})
	t.Run("fallo de confirmacion", func(t *testing.T) {
		escenario := nuevoEscenarioAltaCobroPrueba(t)
		escenario.repositorio.errConfirmacion = errors.New("fallo atomico")
		_, err := escenario.servicio.Crear(context.Background(), escenario.solicitud)
		if !errors.Is(err, ErrPersistenciaAltaCobroIncompleta) || escenario.repositorio.abandonos != 1 ||
			escenario.repositorio.orden != nil {
			t.Fatalf("Crear() = %v; reserva/efectos inesperados: %+v", err, escenario.repositorio)
		}
	})
}

func TestServicioAltaOrdenCobroCommitHaceCASAutoritativoYFallaCerrado(t *testing.T) {
	comprobarSinEfectos := func(t *testing.T, escenario escenarioAltaCobroPrueba, causa error) {
		t.Helper()
		_, err := escenario.servicio.Crear(context.Background(), escenario.solicitud)
		if !errors.Is(err, ErrPersistenciaAltaCobroIncompleta) || !errors.Is(err, causa) ||
			escenario.repositorio.reservas != 1 || escenario.repositorio.confirmaciones != 1 ||
			escenario.repositorio.abandonos != 1 || escenario.repositorio.orden != nil {
			t.Fatalf("Crear() = %v; el commit conflictivo dejo efectos: %+v", err, escenario.repositorio)
		}
	}

	t.Run("liquidacion anulada entre lectura y commit", func(t *testing.T) {
		escenario := nuevoEscenarioAltaCobroPrueba(t)
		escenario.liquidaciones.despuesDeBuscar = func() {
			actual := escenario.repositorio.liquidacionCommit
			actual.Revision++
			actual.Estado = EstadoLiquidacionCobroAnulada
			actual, _ = rehacerLiquidacionAltaCobroPrueba(t, actual)
			escenario.repositorio.liquidacionCommit = actual
		}
		comprobarSinEfectos(t, escenario, ports.ErrControlLiquidacionCobroConflicto)
	})

	t.Run("revision de liquidacion cambia entre lectura y commit", func(t *testing.T) {
		escenario := nuevoEscenarioAltaCobroPrueba(t)
		escenario.liquidaciones.despuesDeBuscar = func() {
			actual := escenario.repositorio.liquidacionCommit
			actual.Revision++
			actual, _ = rehacerLiquidacionAltaCobroPrueba(t, actual)
			escenario.repositorio.liquidacionCommit = actual
		}
		comprobarSinEfectos(t, escenario, ports.ErrControlLiquidacionCobroConflicto)
	})

	t.Run("reserva expira en el reloj del commit", func(t *testing.T) {
		escenario := nuevoEscenarioAltaCobroPrueba(t)
		escenario.repositorio.ahoraCommit = instanteAltaCobroPrueba.Add(duracionReservaAltaCobro)
		comprobarSinEfectos(t, escenario, ports.ErrReservaOrdenCobroCaducada)
	})

	t.Run("liquidacion expira en el reloj del commit", func(t *testing.T) {
		escenario := nuevoEscenarioAltaCobroPrueba(t)
		actual := escenario.repositorio.liquidacionCommit
		actual.ExigibleHasta = instanteAltaCobroPrueba.Add(20 * time.Second)
		actual, leida := rehacerLiquidacionAltaCobroPrueba(t, actual)
		escenario.repositorio.liquidacionCommit = actual
		escenario.liquidaciones.resultados = []LiquidacionCobroAutoritativa{leida}
		escenario.repositorio.ahoraCommit = actual.ExigibleHasta
		comprobarSinEfectos(t, escenario, ports.ErrControlLiquidacionCobroConflicto)
	})

	t.Run("decision retirada entre PDP y commit", func(t *testing.T) {
		escenario := nuevoEscenarioAltaCobroPrueba(t)
		escenario.autorizador.despuesDeDecidir = func(decision domain.DecisionAutorizacion) {
			escenario.repositorio.registrarDecisionAutorizacion(decision, instanteAltaCobroPrueba)
			escenario.repositorio.mutarControlAutorizacion(func(c *controlAutorizacionCommitAltaCobroPrueba) {
				delete(c.decisiones, decision.DecisionRef)
			})
		}
		comprobarSinEfectos(t, escenario, ports.ErrControlAutorizacionCobroConflicto)
	})

	for _, caso := range []struct {
		nombre string
		mutar  func(*controlAutorizacionCommitAltaCobroPrueba)
	}{
		{"asignacion revocada entre PDP y commit", func(c *controlAutorizacionCommitAltaCobroPrueba) {
			c.asignacionActiva = false
		}},
		{"version de rol retirada entre PDP y commit", func(c *controlAutorizacionCommitAltaCobroPrueba) {
			c.versionRolActiva = false
			c.controlRolVigente = false
		}},
		{"catalogo revisado entre PDP y commit", func(c *controlAutorizacionCommitAltaCobroPrueba) {
			c.revisionCatalogo++
			c.catalogoHuellaSHA256 = strings.Repeat("8", 64)
		}},
		{"sesion revocada con fin futuro entre PDP y commit", func(c *controlAutorizacionCommitAltaCobroPrueba) {
			c.controlSesionRevision++
			c.controlSesionHuellaSHA256 = strings.Repeat("9", 64)
			c.sesionValidaHasta = instanteAltaCobroPrueba.Add(20 * time.Second)
		}},
		{"contexto de actor revocado entre PDP y commit", func(c *controlAutorizacionCommitAltaCobroPrueba) {
			c.contextoActorActivo = false
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioAltaCobroPrueba(t)
			escenario.autorizador.despuesDeDecidir = func(decision domain.DecisionAutorizacion) {
				escenario.repositorio.registrarDecisionAutorizacion(decision, instanteAltaCobroPrueba)
				escenario.repositorio.mutarControlAutorizacion(caso.mutar)
			}
			comprobarSinEfectos(t, escenario, ports.ErrControlAutorizacionCobroConflicto)
		})
	}

	t.Run("decision expira exactamente en commit", func(t *testing.T) {
		escenario := nuevoEscenarioAltaCobroPruebaConVigencias(
			t, 10*time.Minute, 20*time.Second,
		)
		escenario.repositorio.ahoraCommit = instanteAltaCobroPrueba.Add(20 * time.Second)
		comprobarSinEfectos(t, escenario, ports.ErrControlAutorizacionCobroConflicto)
	})

	t.Run("sesion y decision expiran exactamente en limite comun", func(t *testing.T) {
		escenario := nuevoEscenarioAltaCobroPruebaConVigencias(
			t, 20*time.Second, 20*time.Second,
		)
		escenario.repositorio.ahoraCommit = instanteAltaCobroPrueba.Add(20 * time.Second)
		comprobarSinEfectos(t, escenario, ports.ErrControlAutorizacionCobroConflicto)
	})
}

func TestServicioAltaOrdenCobroRevalidaVigenciaAntesDeCadaEfecto(t *testing.T) {
	t.Run("expira antes de reservar", func(t *testing.T) {
		escenario := nuevoEscenarioAltaCobroPrueba(t)
		escenario.servicio.reloj = &relojSecuencialAltaCobroPrueba{instantes: []time.Time{
			instanteAltaCobroPrueba, instanteAltaCobroPrueba.Add(2 * time.Minute),
		}}
		_, err := escenario.servicio.Crear(context.Background(), escenario.solicitud)
		if !errors.Is(err, domain.ErrAutorizacionDenegada) || escenario.repositorio.reservas != 0 {
			t.Fatalf("Crear() = %v; una autorizacion expirada alcanzo la reserva", err)
		}
	})
	t.Run("expira despues de reservar", func(t *testing.T) {
		escenario := nuevoEscenarioAltaCobroPrueba(t)
		escenario.servicio.reloj = &relojSecuencialAltaCobroPrueba{instantes: []time.Time{
			instanteAltaCobroPrueba, instanteAltaCobroPrueba,
			instanteAltaCobroPrueba.Add(2 * time.Minute),
		}}
		_, err := escenario.servicio.Crear(context.Background(), escenario.solicitud)
		if !errors.Is(err, ErrPersistenciaAltaCobroIncompleta) || escenario.repositorio.reservas != 1 ||
			escenario.repositorio.abandonos != 1 || escenario.repositorio.confirmaciones != 0 {
			t.Fatalf("Crear() = %v; reserva expirada no se abandono: %+v", err, escenario.repositorio)
		}
	})
}

func TestNuevoServicioAltaOrdenCobroRechazaNulosTipados(t *testing.T) {
	escenario := nuevoEscenarioAltaCobroPrueba(t)
	var repositorio *repositorioAltaCobroPrueba
	var liquidaciones *fuenteLiquidacionesAltaCobroPrueba
	var autorizador *autorizadorAltaCobroPrueba
	var verificador *verificadorAltaCobroPrueba
	var generador *generadorAltaCobroPrueba
	casos := []struct {
		nombre string
		crear  func() (*ServicioAltaOrdenCobro, error)
	}{
		{"repositorio", func() (*ServicioAltaOrdenCobro, error) {
			return NuevoServicioAltaOrdenCobro(repositorio, escenario.liquidaciones, escenario.autorizador, escenario.verificador, selladorAltaCobroPrueba{}, escenario.generador, relojAltaCobroPrueba{instanteAltaCobroPrueba})
		}},
		{"liquidaciones", func() (*ServicioAltaOrdenCobro, error) {
			return NuevoServicioAltaOrdenCobro(escenario.repositorio, liquidaciones, escenario.autorizador, escenario.verificador, selladorAltaCobroPrueba{}, escenario.generador, relojAltaCobroPrueba{instanteAltaCobroPrueba})
		}},
		{"autorizador", func() (*ServicioAltaOrdenCobro, error) {
			return NuevoServicioAltaOrdenCobro(escenario.repositorio, escenario.liquidaciones, autorizador, escenario.verificador, selladorAltaCobroPrueba{}, escenario.generador, relojAltaCobroPrueba{instanteAltaCobroPrueba})
		}},
		{"verificador", func() (*ServicioAltaOrdenCobro, error) {
			return NuevoServicioAltaOrdenCobro(escenario.repositorio, escenario.liquidaciones, escenario.autorizador, verificador, selladorAltaCobroPrueba{}, escenario.generador, relojAltaCobroPrueba{instanteAltaCobroPrueba})
		}},
		{"generador", func() (*ServicioAltaOrdenCobro, error) {
			return NuevoServicioAltaOrdenCobro(escenario.repositorio, escenario.liquidaciones, escenario.autorizador, escenario.verificador, selladorAltaCobroPrueba{}, generador, relojAltaCobroPrueba{instanteAltaCobroPrueba})
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio, err := caso.crear()
			if servicio != nil || !errors.Is(err, ErrDependenciaAltaCobroRequerida) {
				t.Fatalf("constructor = (%v, %v)", servicio, err)
			}
		})
	}
}

func TestLiquidacionYOrdenAltaCobroNoSonDTO(t *testing.T) {
	escenario := nuevoEscenarioAltaCobroPrueba(t)
	liquidacion := escenario.liquidaciones.resultados[0]
	if _, err := json.Marshal(liquidacion); !errors.Is(err, ErrSerializacionAltaCobroProhibida) {
		t.Fatalf("json.Marshal(liquidacion) error = %v", err)
	}
	if _, err := json.Marshal(escenario.solicitud); !errors.Is(err, ErrSerializacionAltaCobroProhibida) {
		t.Fatalf("json.Marshal(solicitud) error = %v", err)
	}
	resultado, err := escenario.servicio.Crear(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatalf("Crear() error = %v", err)
	}
	if _, err := json.Marshal(resultado); !errors.Is(err, ErrSerializacionAltaCobroProhibida) {
		t.Fatalf("json.Marshal(resultado) error = %v", err)
	}
}

func TestLiquidacionCobroAutoritativaExigeEstadoYHuellaExactos(t *testing.T) {
	escenario := nuevoEscenarioAltaCobroPrueba(t)
	datos, _ := escenario.liquidaciones.resultados[0].Datos()
	for _, muta := range []func(*DatosLiquidacionCobroAutoritativa){
		func(d *DatosLiquidacionCobroAutoritativa) { d.Estado = "" },
		func(d *DatosLiquidacionCobroAutoritativa) { d.Estado = "activa" },
		func(d *DatosLiquidacionCobroAutoritativa) { d.Importe.UnidadesMenores++ },
		func(d *DatosLiquidacionCobroAutoritativa) { d.Concepto = " " + d.Concepto },
	} {
		alterada := datos
		muta(&alterada)
		if _, err := NuevaLiquidacionCobroAutoritativa(alterada); !errors.Is(err, ErrLiquidacionCobroNoConfiable) {
			t.Fatalf("NuevaLiquidacionCobroAutoritativa(%+v) error = %v", alterada, err)
		}
	}
}
