package cobertura

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	// MaximoConsumosC1SesionTCBOperacionDecisionCobertura reproduce el límite
	// cerrado del catálogo de cobertura: 64 vías por 32 comprobaciones, con un
	// máximo global de 512 comprobaciones.
	MaximoConsumosC1SesionTCBOperacionDecisionCobertura uint64 = 512
	tiempoMaximoCierreCallbackInfractor                        = 100 * time.Millisecond

	esquemaSesionTCBOperacionDecisionCobertura = "" +
		"VEC-CT-SESION-TCB-OPERACION-DECISION-COBERTURA-V1"
	redaccionSesionTCBOperacionDecisionCobertura = "" +
		"[SESION-TCB-OPERACION-DECISION-COBERTURA-REDACTADA]"
)

var (
	ErrSesionTCBOperacionDecisionCoberturaInvalida = errors.New(
		"contratacion temporal: sesion TCB de decision de cobertura invalida",
	)
	ErrEjecucionSesionTCBOperacionDecisionCoberturaNoDisponible = errors.New(
		"contratacion temporal: ejecucion TCB de decision de cobertura no disponible",
	)
)

// RamaSesionTCBOperacionDecisionCobertura es la unión cerrada que determina
// qué secuencia acepta la sesión. No es una decisión funcional ni una
// autorización.
type RamaSesionTCBOperacionDecisionCobertura string

const (
	RamaSesionTCBOperacionDecisionCoberturaConcedida RamaSesionTCBOperacionDecisionCobertura = "concedida"
	RamaSesionTCBOperacionDecisionCoberturaDenegada  RamaSesionTCBOperacionDecisionCobertura = "denegada"
)

func (r RamaSesionTCBOperacionDecisionCobertura) valida() bool {
	return r == RamaSesionTCBOperacionDecisionCoberturaConcedida ||
		r == RamaSesionTCBOperacionDecisionCoberturaDenegada
}

// EjecutorSesionTCBOperacionDecisionCobertura es la única capacidad técnica
// que una raíz de composición homologada entrega al núcleo. Una
// implementación productiva debe abrir una transacción SERIALIZABLE de
// lectura-escritura, invocar el callback exactamente una vez y de forma
// síncrona, y retornar nil exclusivamente después de COMMIT confirmado.
//
// El callback no puede conservarse ni ejecutarse fuera de este método. Un
// error después de haber intentado COMMIT es ambiguo y nunca autoriza retry.
type EjecutorSesionTCBOperacionDecisionCobertura interface {
	EjecutarSesionTCB(
		context.Context,
		func(SesionTCBOperacionDecisionCobertura) error,
	) error
}

// SesionTCBOperacionDecisionCobertura recibe fragmentos opacos en una
// secuencia cerrada. El adaptador puede desplegar cada fragmento, pero nunca
// recibe OrdenOperacionDecisionCobertura ni decide la rama.
//
// Abrir, Gobierno, DecisionVEC, ConsumoC1, Concesion y Denegacion solo
// acumulan estado transaccional acotado: no escriben, no registran VEC y no
// producen I/O externo. Confirmar es el único punto que puede ejecutar la
// futura función SQL compuesta, todavía ausente en O4-04A.
type SesionTCBOperacionDecisionCobertura interface {
	Abrir(CabeceraSesionTCBOperacionDecisionCobertura) error
	Gobierno(GobiernoSesionTCBOperacionDecisionCobertura) error
	DecisionVEC(DecisionVECSesionTCBOperacionDecisionCobertura) error
	ConsumoC1(ConsumoC1SesionTCBOperacionDecisionCobertura) error
	Concesion(EfectoConcedidoSesionTCBOperacionDecisionCobertura) error
	Denegacion(TerminalDenegadoSesionTCBOperacionDecisionCobertura) error
	Confirmar(context.Context) (
		DatosReciboSesionTCBOperacionDecisionCobertura,
		error,
	)
}

// DatosCabeceraSesionTCBOperacionDecisionCobertura es una vista defensiva
// para el adaptador. Nunca contiene el token propietario en claro.
type DatosCabeceraSesionTCBOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	Esquema                      string
	Rama                         RamaSesionTCBOperacionDecisionCobertura
	HuellaOrdenSHA256            string
	OrganizacionRef              string
	ExpedienteRef                string
	VersionExpediente            uint64
	ReservaRef                   string
	ReciboRef                    string
	ActuacionRef                 string
	AuditoriaRef                 string
	EventoRef                    string
	CorrelacionVECRef            string
	DecisionVECRef               string
	AnalisisRef                  string
	AnalisisHuellaSHA256         string
	TokenPropietarioSHA256       string
	AmbitoIdempotenciaHMAC       string
	HuellaSemanticaHMAC          string
	RevisionCercadoAnterior      uint64
	RevisionCercado              uint64
	ObservadaEnDB                time.Time
	PropiedadHasta               time.Time
	ValidaHastaOrden             time.Time
	PreparacionC1Ref             string
	PreparacionC1HuellaSHA256    string
	PreparacionC1PreparadaEn     time.Time
	PreparacionC1ValidaHasta     time.Time
	NumeroConsumosC1             uint64
	HuellaOrdenesConsumoC1SHA256 string
}

type datosCabeceraSesionTCBOperacionDecisionCobertura struct {
	DatosCabeceraSesionTCBOperacionDecisionCobertura
}

// CabeceraSesionTCBOperacionDecisionCobertura no es construible mediante
// literal desde otro paquete porque conserva su estado en campos privados.
type CabeceraSesionTCBOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	datos *datosCabeceraSesionTCBOperacionDecisionCobertura
}

func (c CabeceraSesionTCBOperacionDecisionCobertura) Datos() (
	DatosCabeceraSesionTCBOperacionDecisionCobertura,
	error,
) {
	if c.validar() != nil {
		return DatosCabeceraSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return c.datos.DatosCabeceraSesionTCBOperacionDecisionCobertura, nil
}

// DatosGobiernoSesionTCBOperacionDecisionCobertura contiene las
// publicaciones exactas que PostgreSQL deberá bloquear y revalidar.
type DatosGobiernoSesionTCBOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	Catalogo           domain.PublicacionCatalogoViasCobertura
	Politica           domain.PublicacionPoliticaDecisionCobertura
	PoliticaActuacion  PublicacionPoliticaActuacionCobertura
	Accion             domain.ClaveCatalogo
	FinalidadCTClave   domain.ClaveCatalogo
	FinalidadCTRef     string
	FinalidadVEC       domain.ClaveCatalogo
	UnidadEjecutoraRef string
	FaseDestino        domain.ClaveFase
	EstadoDestino      domain.EstadoOperativo
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
	EvaluadaEn         time.Time
	ValidaHasta        time.Time
}

type datosGobiernoSesionTCBOperacionDecisionCobertura struct {
	catalogo           domain.CatalogoViasCobertura
	politica           domain.PoliticaDecisionCobertura
	politicaActuacion  PublicacionPoliticaActuacionCobertura
	accion             domain.ClaveCatalogo
	finalidadCTClave   domain.ClaveCatalogo
	finalidadCTRef     string
	finalidadVEC       domain.ClaveCatalogo
	unidadEjecutoraRef string
	faseDestino        domain.ClaveFase
	estadoDestino      domain.EstadoOperativo
	motivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
	evaluadaEn         time.Time
	validaHasta        time.Time
}

type GobiernoSesionTCBOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	datos *datosGobiernoSesionTCBOperacionDecisionCobertura
}

func (g GobiernoSesionTCBOperacionDecisionCobertura) Datos() (
	DatosGobiernoSesionTCBOperacionDecisionCobertura,
	error,
) {
	if g.validar() != nil {
		return DatosGobiernoSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return DatosGobiernoSesionTCBOperacionDecisionCobertura{
		Catalogo:           g.datos.catalogo.Publicacion(),
		Politica:           g.datos.politica.Publicacion(),
		PoliticaActuacion:  g.datos.politicaActuacion,
		Accion:             g.datos.accion,
		FinalidadCTClave:   g.datos.finalidadCTClave,
		FinalidadCTRef:     g.datos.finalidadCTRef,
		FinalidadVEC:       g.datos.finalidadVEC,
		UnidadEjecutoraRef: g.datos.unidadEjecutoraRef,
		FaseDestino:        g.datos.faseDestino,
		EstadoDestino:      g.datos.estadoDestino,
		MotivoAutorizacion: g.datos.motivoAutorizacion,
		EvaluadaEn:         g.datos.evaluadaEn,
		ValidaHasta:        g.datos.validaHasta,
	}, nil
}

type DatosDecisionVECSesionTCBOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	Concedida bool
	Orden     puertosvec.DatosOrdenRegistroAutorizacionLigadaV3
	Resumen   puertosvec.DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3
}

type datosDecisionVECSesionTCBOperacionDecisionCobertura struct {
	candidata puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3
	resumen   puertosvec.ResumenCandidataRegistroDecisionAutorizacionLigadaV3
}

type DecisionVECSesionTCBOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	datos *datosDecisionVECSesionTCBOperacionDecisionCobertura
}

func (d DecisionVECSesionTCBOperacionDecisionCobertura) Datos() (
	DatosDecisionVECSesionTCBOperacionDecisionCobertura,
	error,
) {
	concedida, orden, resumen, err := datosDecisionVECSesionTCB(d)
	if err != nil {
		return DatosDecisionVECSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return DatosDecisionVECSesionTCBOperacionDecisionCobertura{
		Concedida: concedida,
		Orden:     orden,
		Resumen:   resumen,
	}, nil
}

type DatosConsumoC1SesionTCBOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	Posicion         uint64
	Total            uint64
	Orden            puertosct.DatosOrdenConsumoCobertura
	Resumen          puertosct.ResumenOrdenConsumoCobertura
	PruebasCanonicas puertosct.PruebasCanonicasOrdenConsumoCobertura
}

type datosConsumoC1SesionTCBOperacionDecisionCobertura struct {
	posicion uint64
	total    uint64
	orden    puertosct.OrdenConsumoCobertura
	instante time.Time
}

type ConsumoC1SesionTCBOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	datos *datosConsumoC1SesionTCBOperacionDecisionCobertura
}

func (c ConsumoC1SesionTCBOperacionDecisionCobertura) Datos() (
	DatosConsumoC1SesionTCBOperacionDecisionCobertura,
	error,
) {
	if c.validar() != nil {
		return DatosConsumoC1SesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	orden, errOrden := c.datos.orden.Datos()
	resumen, errResumen := c.datos.orden.ResumenPendienteEn(c.datos.instante)
	pruebas, errPruebas := c.datos.orden.PruebasCanonicas()
	if errOrden != nil || errResumen != nil || errPruebas != nil {
		return DatosConsumoC1SesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return DatosConsumoC1SesionTCBOperacionDecisionCobertura{
		Posicion: c.datos.posicion, Total: c.datos.total,
		Orden: orden, Resumen: resumen, PruebasCanonicas: pruebas,
	}, nil
}

type DatosEfectoConcedidoSesionTCBOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	AgregadoAnterior  domain.Expediente
	AgregadoSiguiente domain.Expediente
	Propuesta         domain.PublicacionPropuestaDecisionCobertura
	MotivoFuncional   domain.MotivoGobernadoDecisionCobertura
	EfectoEn          time.Time
	ValidaHasta       time.Time
}

type datosEfectoConcedidoSesionTCBOperacionDecisionCobertura struct {
	agregadoAnterior  domain.Expediente
	agregadoSiguiente domain.Expediente
	propuesta         domain.PropuestaDecisionCobertura
	motivoFuncional   domain.MotivoGobernadoDecisionCobertura
	efectoEn          time.Time
	validaHasta       time.Time
}

type EfectoConcedidoSesionTCBOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	datos *datosEfectoConcedidoSesionTCBOperacionDecisionCobertura
}

func (e EfectoConcedidoSesionTCBOperacionDecisionCobertura) Datos() (
	DatosEfectoConcedidoSesionTCBOperacionDecisionCobertura,
	error,
) {
	if e.validar() != nil {
		return DatosEfectoConcedidoSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return DatosEfectoConcedidoSesionTCBOperacionDecisionCobertura{
		AgregadoAnterior:  e.datos.agregadoAnterior.Clonar(),
		AgregadoSiguiente: e.datos.agregadoSiguiente.Clonar(),
		Propuesta:         e.datos.propuesta.Publicacion(),
		MotivoFuncional:   e.datos.motivoFuncional,
		EfectoEn:          e.datos.efectoEn,
		ValidaHasta:       e.datos.validaHasta,
	}, nil
}

type DatosTerminalDenegadoSesionTCBOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	OrganizacionRef    string
	ExpedienteRef      string
	VersionExpediente  uint64
	ReservaRef         string
	ReciboRef          string
	AuditoriaRef       string
	CorrelacionVECRef  string
	DecisionVECRef     string
	RevisionCercado    uint64
	RecursoVEC         dominiovec.RecursoAutorizable
	ActorRef           string
	PerfilRef          string
	AccionVEC          domain.ClaveCatalogo
	FinalidadVEC       domain.ClaveCatalogo
	MotivoVEC          dominiovec.ReferenciaEntradaCatalogo
	LimitePreparacion  time.Time
	ValidaHasta        time.Time
	PruebaHuellaSHA256 string
}

type datosTerminalDenegadoSesionTCBOperacionDecisionCobertura struct {
	prueba      pruebaDenegacionOperacionDecisionCobertura
	validaHasta time.Time
}

type TerminalDenegadoSesionTCBOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	datos *datosTerminalDenegadoSesionTCBOperacionDecisionCobertura
}

func (d TerminalDenegadoSesionTCBOperacionDecisionCobertura) Datos() (
	DatosTerminalDenegadoSesionTCBOperacionDecisionCobertura,
	error,
) {
	if d.validar() != nil {
		return DatosTerminalDenegadoSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	p := d.datos.prueba
	r := p.reserva
	return DatosTerminalDenegadoSesionTCBOperacionDecisionCobertura{
		OrganizacionRef:    r.organizacionRef,
		ExpedienteRef:      r.expedienteRef,
		VersionExpediente:  r.versionExpediente,
		ReservaRef:         r.reservaRef,
		ReciboRef:          r.reciboRef,
		AuditoriaRef:       r.auditoriaRef,
		CorrelacionVECRef:  r.correlacionVECRef,
		DecisionVECRef:     r.decisionVECRef,
		RevisionCercado:    r.revisionCercado,
		RecursoVEC:         clonarRecursoOperacionDecisionCobertura(p.recursoVEC),
		ActorRef:           p.actorRef,
		PerfilRef:          p.perfilRef,
		AccionVEC:          p.accionVEC,
		FinalidadVEC:       p.finalidadVEC,
		MotivoVEC:          p.motivoVEC,
		LimitePreparacion:  p.limitePreparacion,
		ValidaHasta:        d.datos.validaHasta,
		PruebaHuellaSHA256: p.huellaSHA256,
	}, nil
}

// DatosReciboSesionTCBOperacionDecisionCobertura es una proyección cruda y no
// autoritativa construible por el adaptador. Fabricarla no concede autoridad:
// el núcleo la liga a la orden dentro del callback y solo publica un resultado
// nominal después de que el ejecutor confirme COMMIT.
type DatosReciboSesionTCBOperacionDecisionCobertura struct {
	ReciboRef               string
	ReservaRef              string
	AuditoriaRef            string
	CorrelacionVECRef       string
	DecisionVECRef          string
	DecisionVECHuellaSHA256 string
	CodigoProbatorioVEC     string
	ConcedidaVEC            bool
	RevisionCercado         uint64
	AmbitoIdempotenciaHMAC  string
	HuellaSemanticaHMAC     string
	ConfirmadaEn            time.Time
	Aplicada                bool
	DenegadaVEC             bool
	DecisionCoberturaRef    string
	DecisionCoberturaHuella string
	VersionResultante       uint64
	EventoRef               string
	ActuacionRef            string
}

func (DatosReciboSesionTCBOperacionDecisionCobertura) String() string {
	return redaccionSesionTCBOperacionDecisionCobertura
}

func (d DatosReciboSesionTCBOperacionDecisionCobertura) GoString() string {
	return d.String()
}

func (d DatosReciboSesionTCBOperacionDecisionCobertura) Format(
	estado fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(estado, d.String())
}

func (d DatosReciboSesionTCBOperacionDecisionCobertura) LogValue() slog.Value {
	return slog.StringValue(d.String())
}

func (DatosReciboSesionTCBOperacionDecisionCobertura) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionOperacionDecisionCoberturaProhibida
}

func (*DatosReciboSesionTCBOperacionDecisionCobertura) UnmarshalJSON([]byte) error {
	return ErrSerializacionOperacionDecisionCoberturaProhibida
}

func (DatosReciboSesionTCBOperacionDecisionCobertura) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionOperacionDecisionCoberturaProhibida
}

func (*DatosReciboSesionTCBOperacionDecisionCobertura) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionOperacionDecisionCoberturaProhibida
}

func (DatosReciboSesionTCBOperacionDecisionCobertura) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionOperacionDecisionCoberturaProhibida
}

func (*DatosReciboSesionTCBOperacionDecisionCobertura) UnmarshalText([]byte) error {
	return ErrSerializacionOperacionDecisionCoberturaProhibida
}

func (DatosReciboSesionTCBOperacionDecisionCobertura) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionOperacionDecisionCoberturaProhibida
}

func (*DatosReciboSesionTCBOperacionDecisionCobertura) UnmarshalBinary([]byte) error {
	return ErrSerializacionOperacionDecisionCoberturaProhibida
}

func (DatosReciboSesionTCBOperacionDecisionCobertura) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionOperacionDecisionCoberturaProhibida
}

func (*DatosReciboSesionTCBOperacionDecisionCobertura) GobDecode([]byte) error {
	return ErrSerializacionOperacionDecisionCoberturaProhibida
}

func (DatosReciboSesionTCBOperacionDecisionCobertura) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionOperacionDecisionCoberturaProhibida
}

func (*DatosReciboSesionTCBOperacionDecisionCobertura) UnmarshalCBOR([]byte) error {
	return ErrSerializacionOperacionDecisionCoberturaProhibida
}

func (DatosReciboSesionTCBOperacionDecisionCobertura) MarshalYAML() (any, error) {
	return nil, ErrSerializacionOperacionDecisionCoberturaProhibida
}

func (*DatosReciboSesionTCBOperacionDecisionCobertura) UnmarshalYAML(
	func(any) error,
) error {
	return ErrSerializacionOperacionDecisionCoberturaProhibida
}

type transaccionOperacionDecisionCoberturaTCB struct {
	ejecutor EjecutorSesionTCBOperacionDecisionCobertura
}

// NuevaTransaccionOperacionDecisionCoberturaTCB es el constructor reservado a
// la raíz de composición. Fija un ejecutor para toda la vida del servicio; ni
// la solicitud ni el canal pueden seleccionar o sustituirlo.
func NuevaTransaccionOperacionDecisionCoberturaTCB(
	ejecutor EjecutorSesionTCBOperacionDecisionCobertura,
) (TransaccionOperacionDecisionCobertura, error) {
	if dependenciaGobiernoOperacionCoberturaNula(ejecutor) {
		return nil, ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return &transaccionOperacionDecisionCoberturaTCB{ejecutor: ejecutor}, nil
}

type controlInvocacionEjecutorSesionTCB struct {
	mu                    sync.Mutex
	terminadaCh           chan struct{}
	iniciada              bool
	terminada             bool
	terminadaAntesRetorno bool
	retornada             bool
	violacion             bool
	valida                bool
	recibo                ReciboOperacionDecisionCobertura
}

func (c *controlInvocacionEjecutorSesionTCB) iniciar() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.iniciada || c.retornada {
		c.violacion = true
		return false
	}
	c.iniciada = true
	return true
}

func (c *controlInvocacionEjecutorSesionTCB) terminar(
	recibo ReciboOperacionDecisionCobertura,
	valida bool,
) {
	c.mu.Lock()
	if c.terminada {
		c.violacion = true
		c.mu.Unlock()
		return
	}
	c.terminadaAntesRetorno = !c.retornada
	if c.retornada {
		c.violacion = true
	}
	c.terminada = true
	c.valida = valida
	if valida {
		c.recibo = clonarReciboOperacionDecisionCobertura(recibo)
	}
	close(c.terminadaCh)
	c.mu.Unlock()
}

func (c *controlInvocacionEjecutorSesionTCB) marcarRetornoEjecutor() {
	c.mu.Lock()
	c.retornada = true
	c.mu.Unlock()
}

func (c *controlInvocacionEjecutorSesionTCB) cerrar() (
	ReciboOperacionDecisionCobertura,
	bool,
	bool,
	bool,
) {
	c.mu.Lock()
	esperar := c.iniciada && !c.terminada
	canalTerminado := c.terminadaCh
	c.mu.Unlock()

	if esperar {
		temporizador := time.NewTimer(tiempoMaximoCierreCallbackInfractor)
		defer temporizador.Stop()
		select {
		case <-canalTerminado:
		case <-temporizador.C:
			c.mu.Lock()
			c.violacion = true
			c.mu.Unlock()
			return ReciboOperacionDecisionCobertura{}, false, false, true
		}
		c.mu.Lock()
		c.violacion = true
		c.mu.Unlock()
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	huboReciboValido := c.iniciada && c.terminada && c.valida
	ejecucionPotencialmenteEfectiva := huboReciboValido ||
		(c.iniciada && c.terminada && !c.terminadaAntesRetorno)
	publicable := huboReciboValido &&
		c.terminadaAntesRetorno &&
		!c.violacion
	if !huboReciboValido {
		return ReciboOperacionDecisionCobertura{},
			false,
			ejecucionPotencialmenteEfectiva,
			false
	}
	return clonarReciboOperacionDecisionCobertura(c.recibo),
		publicable,
		ejecucionPotencialmenteEfectiva,
		false
}

func (t *transaccionOperacionDecisionCoberturaTCB) confirmarOperacionDecisionCobertura(
	ctx context.Context,
	orden OrdenOperacionDecisionCobertura,
) (ResultadoConfirmacionOperacionDecisionCobertura, error) {
	if dependenciaGobiernoOperacionCoberturaNula(ctx) ||
		dependenciaGobiernoOperacionCoberturaNula(t) ||
		dependenciaGobiernoOperacionCoberturaNula(t.ejecutor) ||
		orden.validar() != nil {
		return ResultadoConfirmacionOperacionDecisionCobertura{},
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	if err := ctx.Err(); err != nil {
		return ResultadoConfirmacionOperacionDecisionCobertura{}, err
	}

	control := &controlInvocacionEjecutorSesionTCB{
		terminadaCh: make(chan struct{}),
	}
	ctxSesion, cancelarSesion := context.WithCancel(ctx)
	defer cancelarSesion()
	guardia := &guardiaCicloSesionTCBOperacionDecisionCobertura{}
	errEjecucion := t.ejecutor.EjecutarSesionTCB(
		ctxSesion,
		func(sesion SesionTCBOperacionDecisionCobertura) error {
			if !control.iniciar() {
				return ErrSesionTCBOperacionDecisionCoberturaInvalida
			}
			recibo, err := desplegarOrdenOperacionDecisionCoberturaEnSesionTCB(
				ctxSesion,
				orden,
				sesion,
				guardia,
			)
			valida := err == nil &&
				validarReciboParaOrdenOperacionDecisionCobertura(
					orden,
					recibo,
				) == nil
			control.terminar(recibo, valida)
			if !valida {
				return ErrSesionTCBOperacionDecisionCoberturaInvalida
			}
			return nil
		},
	)
	// Desde el retorno del ejecutor se rechazan nuevas operaciones, se cancela
	// la sesión y ningún recibo que termine después puede publicarse.
	control.marcarRetornoEjecutor()
	guardia.marcarRetornoEjecutor()
	cancelarSesion()
	recibo, invocacionPublicable, ejecucionPotencialmenteEfectiva,
		callbackPendiente :=
		control.cerrar()
	ejecucionPotencialmenteEfectiva =
		ejecucionPotencialmenteEfectiva || guardia.intentoConfirmar()
	if callbackPendiente {
		return ResultadoConfirmacionOperacionDecisionCobertura{},
			ErrEjecucionSesionTCBOperacionDecisionCoberturaNoDisponible
	}
	if ejecucionPotencialmenteEfectiva &&
		(errEjecucion != nil || !invocacionPublicable) {
		return ResultadoConfirmacionOperacionDecisionCobertura{},
			ErrEjecucionSesionTCBOperacionDecisionCoberturaNoDisponible
	}
	if errEjecucion != nil || !invocacionPublicable {
		return ResultadoConfirmacionOperacionDecisionCobertura{},
			errFalloAntesCommitOperacionDecisionCobertura
	}
	// El ejecutor ya retornó nil y, por contrato, el COMMIT está confirmado.
	// Solo ahora se publica el resultado nominal.
	resultado, err := NuevaResultadoConfirmacionOperacionDecisionCobertura(
		orden,
		recibo,
	)
	if err != nil {
		return ResultadoConfirmacionOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return resultado, nil
}
