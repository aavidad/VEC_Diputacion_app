package bootstrap

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"regexp"
	"sync"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/adapters/fuentesintetica"
	appbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// El puente CT recibe la emisión nativa de Bolsa ya compuesta con contexto,
// autorización V3, correo y persistencia propios. La misma clave recupera la
// emisión original; CT no crea otra autoridad de llamamientos.
type emisorNativoLlamamientoCT interface {
	EmitirLlamamiento(context.Context, puertosbolsa.SolicitudEmitirLlamamiento) (puertosbolsa.EmisionLlamamiento, error)
	RecuperarLlamamiento(context.Context, puertosbolsa.SolicitudRecuperarLlamamiento) (puertosbolsa.EmisionLlamamiento, error)
}

type proveedorConfiguracionEmisionCT func(context.Context, preparacionLlamamientoDesarrollo) (
	puertosbolsa.ConfiguracionLlamamiento, string, vecdomain.ReferenciaEntradaCatalogo, error,
)

type registradorLecturaOrdenCT func(context.Context, preparacionLlamamientoDesarrollo,
	string) (puertosbolsa.CapturaOrigenCT, error)

var _ emisorNativoLlamamientoCT = (*appbolsa.ServicioEmisionLlamamiento)(nil)

// El servicio B7 privado usa la misma persistencia y correo que el portal,
// pero revalida el perfil Bolsa administrado bajo la ruta CT original.
type emisorNativoPrivadoCT struct {
	puente *puenteBolsaLlamamientoDesarrollo
}

func (e emisorNativoPrivadoCT) EmitirLlamamiento(ctx context.Context,
	q puertosbolsa.SolicitudEmitirLlamamiento) (puertosbolsa.EmisionLlamamiento, error) {
	vacia := puertosbolsa.EmisionLlamamiento{}
	if e.puente == nil || e.puente.operacionNativa == nil || e.puente.operacionNativa.emision == nil || ctx == nil || ctx.Err() != nil {
		return vacia, puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	captura, err := e.puente.capturaOperacionNativa(ctx)
	if err != nil || !q.Vinculo.CoincideExactamenteCon(captura.ct.Vinculo) ||
		!reflect.DeepEqual(q.ResultadoContexto, captura.ct.Resultado) {
		return vacia, ports.ErrAutorizacionDenegada
	}
	q.Vinculo, q.ResultadoContexto = captura.bolsa.Vinculo, captura.bolsa.Resultado
	return e.puente.operacionNativa.emision.EmitirLlamamiento(captura.ctx, q)
}

func (e emisorNativoPrivadoCT) RecuperarLlamamiento(ctx context.Context,
	q puertosbolsa.SolicitudRecuperarLlamamiento) (puertosbolsa.EmisionLlamamiento, error) {
	vacia := puertosbolsa.EmisionLlamamiento{}
	if e.puente == nil || e.puente.operacionNativa == nil || e.puente.operacionNativa.emision == nil || ctx == nil || ctx.Err() != nil {
		return vacia, puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	captura, err := e.puente.capturaOperacionNativa(ctx)
	if err != nil || !reflect.DeepEqual(q.ContextoActor, captura.ct.Resultado.Contexto) {
		return vacia, ports.ErrAutorizacionDenegada
	}
	q.ContextoActor = captura.bolsa.Resultado.Contexto
	return e.puente.operacionNativa.emision.RecuperarLlamamiento(captura.ctx, q)
}

var _ emisorNativoLlamamientoCT = emisorNativoPrivadoCT{}

// La raíz inyecta el servicio de Bolsa ya compuesto una sola vez. El puente
// no construye otro emisor ni sustituye una autoridad después del arranque.
func (p *puenteBolsaLlamamientoDesarrollo) configurarEmisorNativoLlamamientoCT(emisor emisorNativoLlamamientoCT) error {
	if p == nil || dependenciaEsNulaContratacionTemporalDesarrollo(emisor) || p.emisorNativo != nil {
		return puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	p.emisorNativo = emisor
	return nil
}

func (p *puenteBolsaLlamamientoDesarrollo) configurarOperacionNativaCT(o *operacionBolsaNativaCT) error {
	if p == nil || o == nil || o.ct == nil || o.emision == nil || p.alta == nil ||
		o.ct != p.alta.soporte || p.operacionNativa != nil {
		return ports.ErrIntegracionBolsaNoDisponible
	}
	p.operacionNativa = o
	return nil
}

func (p *puenteBolsaLlamamientoDesarrollo) emisorNativoLlamamientoConfigurado() emisorNativoLlamamientoCT {
	if p == nil {
		return nil
	}
	return p.emisorNativo
}

// La lectura auditada decide entre la reserva histórica y el corte vigente
// dentro de la misma transacción que registra el acuse nominal positivo.
func (p *puenteBolsaLlamamientoDesarrollo) configurarOrigenConstituidoLlamamientoCT(
	fuente puertosbolsa.FuenteOrigenLlamamientoCTAuditada,
) error {
	if p == nil || dependenciaEsNulaContratacionTemporalDesarrollo(fuente) || p.origenNativo != nil {
		return ports.ErrIntegracionBolsaNoDisponible
	}
	p.origenNativo = fuente
	return nil
}

func (p *puenteBolsaLlamamientoDesarrollo) configurarDatosEmisionConstituidaCT(proveedor proveedorConfiguracionEmisionCT) error {
	if p == nil || proveedor == nil || p.configuracionNativa != nil {
		return ports.ErrIntegracionBolsaNoDisponible
	}
	p.configuracionNativa = proveedor
	return nil
}

func (p *puenteBolsaLlamamientoDesarrollo) configurarAuditoriaLecturaOrdenCT(registrar registradorLecturaOrdenCT) error {
	if p == nil || registrar == nil || p.auditarOrdenNativo != nil {
		return ports.ErrIntegracionBolsaNoDisponible
	}
	p.auditarOrdenNativo = registrar
	return nil
}

func (p *puenteBolsaLlamamientoDesarrollo) configurarLectorResultadoEmisionCT(lector puertosbolsa.LectorResultadoEmisionCT) error {
	if p == nil || dependenciaEsNulaContratacionTemporalDesarrollo(lector) || p.resultadoEmisionNativo != nil {
		return ports.ErrIntegracionBolsaNoDisponible
	}
	p.resultadoEmisionNativo = lector
	return nil
}

type origenConstituidoVerificadoCT struct {
	registro     puertosbolsa.OrigenLlamamientoCT
	ordenPermiso json.RawMessage
	bolsa        dominiobolsa.BolsaConstituida
	instantanea  dominiobolsa.InstantaneaOrdenBolsa
	reserva      *puertosbolsa.ReservaOrigenCT
	auditoriaRef string
}

type preparacionOrigenConstituidoCT struct {
	origen        origenConstituidoVerificadoCT
	necesidad     dominiobolsa.NecesidadCobertura
	configuracion puertosbolsa.ConfiguracionLlamamiento
	motivo        vecdomain.ReferenciaEntradaCatalogo
	seleccion     puertosbolsa.EntradaOrigenLlamamientoCT
	refNecesidad  ports.ReferenciaVersionadaIntegracionBolsa
	refBolsa      ports.ReferenciaVersionadaIntegracionBolsa
	refPolitica   ports.ReferenciaVersionadaIntegracionBolsa
}

type claveEstadoOrigenConstituidoCT struct{}

// Vive únicamente en el contexto de una POST ya autorizada. La consulta
// nominal y su acuse se conservan durante las seis etapas del mismo servicio;
// no hay estado compartido entre peticiones, actores ni reinicios.
type estadoOrigenConstituidoCT struct {
	organizacion, expediente, clave string
	version                         uint64
	unaLectura                      sync.Once
	preparado                       preparacionOrigenConstituidoCT
	fallo                           error
	unaSesion                       sync.Once
	sesion                          capturaOperacionNativaCT
	falloSesion                     error
}

type capturaOperacionNativaCT struct {
	ctx   context.Context
	ct    ports.ContextoAutorizacionAltaV3
	bolsa ports.ContextoAutorizacionAltaV3
}

func (p *puenteBolsaLlamamientoDesarrollo) capturaOperacionNativa(ctx context.Context) (capturaOperacionNativaCT, error) {
	vacia := capturaOperacionNativaCT{}
	if p == nil || p.alta == nil || p.alta.soporte == nil || p.operacionNativa == nil ||
		ctx == nil || ctx.Err() != nil {
		return vacia, ports.ErrIntegracionBolsaNoDisponible
	}
	estado, ok := ctx.Value(claveEstadoOrigenConstituidoCT{}).(*estadoOrigenConstituidoCT)
	if !ok || estado == nil {
		return vacia, ports.ErrAutorizacionDenegada
	}
	estado.unaSesion.Do(func() {
		actorCT, err := p.alta.soporte.contextoOperativoDesarrollo(ctx)
		if err != nil || actorCT.Resultado.Validar() != nil || actorCT.Vinculo.ValidarPara(actorCT.Resultado) != nil {
			estado.falloSesion = errors.Join(ports.ErrAutorizacionDenegada, err)
			return
		}
		hijo, bolsa, err := p.operacionNativa.Contexto(ctx, nil)
		if err != nil {
			estado.falloSesion = err
			return
		}
		estado.sesion = capturaOperacionNativaCT{ctx: hijo, ct: actorCT, bolsa: bolsa}
	})
	actual, err := p.alta.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil || estado.falloSesion != nil || !actual.Vinculo.CoincideExactamenteCon(estado.sesion.ct.Vinculo) ||
		!reflect.DeepEqual(actual.Resultado, estado.sesion.ct.Resultado) ||
		!actual.Vinculo.VigenteEn(p.reloj.Ahora(), actual.Resultado) || estado.sesion.ctx == nil {
		return vacia, errors.Join(ports.ErrAutorizacionDenegada, err, estado.falloSesion)
	}
	return estado.sesion, nil
}

func (p *puenteBolsaLlamamientoDesarrollo) capturaOperacionNativaDesdeHijo(
	ctx context.Context, d preparacionLlamamientoDesarrollo,
) (capturaOperacionNativaCT, error) {
	vacia := capturaOperacionNativaCT{}
	if p == nil || p.reloj == nil || ctx == nil || ctx.Err() != nil {
		return vacia, ports.ErrAutorizacionDenegada
	}
	estado, ok := ctx.Value(claveEstadoOrigenConstituidoCT{}).(*estadoOrigenConstituidoCT)
	if !ok || estado == nil || estado.sesion.ctx != ctx || estado.falloSesion != nil ||
		estado.organizacion != d.expediente.Fiscalizado.OrganizacionRef ||
		estado.expediente != d.expediente.Fiscalizado.Referencia ||
		estado.clave != d.clave || estado.version != d.expediente.Fiscalizado.Version ||
		estado.sesion.ct.Resultado.Validar() != nil ||
		!estado.sesion.ct.Vinculo.VigenteEn(p.reloj.Ahora(), estado.sesion.ct.Resultado) ||
		estado.sesion.bolsa.Resultado.Validar() != nil ||
		!estado.sesion.bolsa.Vinculo.VigenteEn(p.reloj.Ahora(), estado.sesion.bolsa.Resultado) {
		return vacia, ports.ErrAutorizacionDenegada
	}
	return estado.sesion, nil
}

func (e *estadoOrigenConstituidoCT) cargar(ctx context.Context, d preparacionLlamamientoDesarrollo,
	consultar func() (preparacionOrigenConstituidoCT, error),
) (preparacionOrigenConstituidoCT, error) {
	vacio := preparacionOrigenConstituidoCT{}
	if e == nil || ctx == nil || ctx.Err() != nil || consultar == nil ||
		e.organizacion != d.expediente.Fiscalizado.OrganizacionRef ||
		e.expediente != d.expediente.Fiscalizado.Referencia ||
		e.version != d.expediente.Fiscalizado.Version || e.clave != d.clave {
		return vacio, ports.ErrPeticionIntegracionBolsaInvalida
	}
	e.unaLectura.Do(func() { e.preparado, e.fallo = consultar() })
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if e.fallo != nil {
		return vacio, e.fallo
	}
	return e.preparado, nil
}

func posicionesOrigenConstituidoCT(o puertosbolsa.OrigenLlamamientoCT) (uint32, uint32, error) {
	if len(o.Entradas) == 0 || len(o.Entradas) > int(ports.MaximoElementosIntegracionBolsa) {
		return 0, 0, ports.ErrRespuestaBolsaNoConfiable
	}
	var disponibles uint32
	for _, e := range o.Entradas {
		if e.Elegible {
			disponibles++
		}
	}
	return uint32(len(o.Entradas)), disponibles, nil
}

func huellaDecisionOrigenConstituidoCT(o puertosbolsa.OrigenLlamamientoCT) (string, error) {
	material, err := json.Marshal(struct {
		Acta, Bolsa, Instantanea, Politica, HuellaOrden, HuellaPolitica string
		PoliticaProvisional                                             bool
		Entradas                                                        []puertosbolsa.EntradaOrigenLlamamientoCT
		CriteriosOfrecimiento                                           *puertosbolsa.CriteriosOfrecimientoAplicados `json:"criterios_ofrecimiento,omitempty"`
	}{o.ActaRef, o.BolsaRef, o.InstantaneaRef, o.PoliticaRef, o.HuellaInstantaneaSHA256,
		o.HuellaPoliticaSHA256, o.PoliticaProvisional, o.Entradas, o.CriteriosOfrecimiento})
	if err != nil {
		return "", ports.ErrRespuestaBolsaNoConfiable
	}
	return huellaPuenteLlamamientoDesarrollo(material), nil
}

func referenciaResultadoOrigenCT(tipo string, c ports.DatosContextoPeticionIntegracionBolsa,
	o puertosbolsa.OrigenLlamamientoCT) (ports.ReferenciaVersionadaIntegracionBolsa, error) {
	huella, err := huellaDecisionOrigenConstituidoCT(o)
	if err != nil {
		return ports.ReferenciaVersionadaIntegracionBolsa{}, err
	}
	return referenciaVersionadaPuenteLlamamientoDesarrollo(
		referenciaPuenteLlamamientoDesarrollo(tipo, c.OperacionRef), 1, huella), nil
}

func (p *puenteBolsaLlamamientoDesarrollo) procedenciaOrigenConstituidoCT(
	c ports.DatosContextoPeticionIntegracionBolsa, o origenConstituidoVerificadoCT, tipo string, ahora time.Time,
) ports.ProcedenciaIntegracionBolsa {
	return ports.ProcedenciaIntegracionBolsa{
		AutoridadRef:    autoridadRespuestaLlamamientoDesarrollo,
		RespuestaRef:    referenciaPuenteLlamamientoDesarrollo("respuesta-"+tipo, c.OperacionRef, ahora.Format(time.RFC3339Nano)),
		ContratoVersion: 1,
		Fuente: referenciaVersionadaPuenteLlamamientoDesarrollo(o.registro.InstantaneaRef,
			o.registro.VersionInstantanea, o.registro.HuellaInstantaneaSHA256),
		Evidencia: ports.EvidenciaNominalIntegracionBolsa{
			EvidenciaRef: referenciaPuenteLlamamientoDesarrollo("evidencia", c.OperacionRef, tipo),
			EmitidaEn:    ahora, ValidaHasta: c.ValidaHasta,
		},
		SinRetencionDeclarada: true,
		PoliticaProvisional:   o.registro.PoliticaProvisional,
	}
}

func (p *puenteBolsaLlamamientoDesarrollo) prepararConsultaConstituidaCT(
	ctx context.Context, d preparacionLlamamientoDesarrollo,
) (ports.SolicitudDisponibilidadBolsa, error) {
	origen, err := p.prepararOrigenConstituidoCT(ctx, d)
	if err != nil {
		return ports.SolicitudDisponibilidadBolsa{}, err
	}
	posiciones, _, err := posicionesOrigenConstituidoCT(origen.origen.registro)
	if err != nil {
		return ports.SolicitudDisponibilidadBolsa{}, err
	}
	c, err := p.contexto(ctx, d, referenciaPuenteLlamamientoDesarrollo("operacion-consulta", d.necesidad, d.clave),
		"bolsa.disponibilidad.consultar", origen.refNecesidad)
	if err != nil {
		return ports.SolicitudDisponibilidadBolsa{}, err
	}
	return ports.SolicitudDisponibilidadBolsa{Contexto: c, Necesidad: origen.refNecesidad,
		CategoriaRef: d.categoria, MaximoResultados: posiciones}, nil
}

func (p *puenteBolsaLlamamientoDesarrollo) consultarDisponibilidadConstituidaCT(
	ctx context.Context, d preparacionLlamamientoDesarrollo, solicitud ports.SolicitudDisponibilidadBolsa,
) (ports.ResultadoDisponibilidadBolsa, error) {
	vacio := ports.ResultadoDisponibilidadBolsa{}
	origen, err := p.prepararOrigenConstituidoCT(ctx, d)
	if err != nil {
		return vacio, err
	}
	posiciones, disponibles, err := posicionesOrigenConstituidoCT(origen.origen.registro)
	if err != nil {
		return vacio, err
	}
	c, err := p.verificarPeticion(ctx, d, solicitud.Contexto,
		referenciaPuenteLlamamientoDesarrollo("operacion-consulta", d.necesidad, d.clave),
		"bolsa.disponibilidad.consultar", origen.refNecesidad)
	if err != nil || solicitud.Necesidad != origen.refNecesidad || solicitud.CategoriaRef != d.categoria ||
		solicitud.MaximoResultados != posiciones {
		return vacio, ports.ErrPeticionIntegracionBolsaInvalida
	}
	ahora := p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	resultadoRef, err := referenciaResultadoOrigenCT("resultado-disponibilidad", c, origen.origen.registro)
	if err != nil {
		return vacio, err
	}
	r := ports.ResultadoDisponibilidadBolsa{
		OperacionRef: c.OperacionRef, OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
		VersionExpediente: c.VersionExpediente, CorrelacionRef: c.CorrelacionRef,
		Necesidad: origen.refNecesidad, CategoriaRef: d.categoria, Resultado: resultadoRef,
		BolsaEncontrada: true, Bolsa: origen.refBolsa, Disponible: disponibles > 0,
		CantidadDisponible: disponibles, CantidadExacta: true,
		Procedencia: p.procedenciaOrigenConstituidoCT(c, origen.origen, "disponibilidad", ahora),
	}
	return p.emisorRespuesta.FirmarDisponibilidad(ctx, solicitud, r, ahora)
}

func (p *puenteBolsaLlamamientoDesarrollo) prepararOrdenCompletoConstituidaCT(
	ctx context.Context, d preparacionLlamamientoDesarrollo, resultado ports.ResultadoDisponibilidadBolsa,
) (ports.ComandoPrepararOrdenBolsa, error) {
	vacio := ports.ComandoPrepararOrdenBolsa{}
	origen, err := p.prepararOrigenConstituidoCT(ctx, d)
	if err != nil {
		return vacio, err
	}
	posiciones, disponibles, err := posicionesOrigenConstituidoCT(origen.origen.registro)
	if err != nil {
		return vacio, err
	}
	refResultado, err := referenciaResultadoOrigenCT("resultado-disponibilidad",
		ports.DatosContextoPeticionIntegracionBolsa{OperacionRef: resultado.OperacionRef}, origen.origen.registro)
	if err != nil || resultado.Necesidad != origen.refNecesidad || resultado.Bolsa != origen.refBolsa ||
		resultado.CategoriaRef != d.categoria || resultado.Resultado != refResultado ||
		resultado.Procedencia.Fuente != referenciaVersionadaPuenteLlamamientoDesarrollo(
			origen.origen.registro.InstantaneaRef, origen.origen.registro.VersionInstantanea,
			origen.origen.registro.HuellaInstantaneaSHA256) ||
		resultado.CantidadDisponible != disponibles || disponibles == 0 ||
		resultado.CantidadDisponible > posiciones || !resultado.CantidadExacta || !resultado.Disponible ||
		!resultado.Procedencia.SinRetencionDeclarada ||
		resultado.Procedencia.PoliticaProvisional != origen.origen.registro.PoliticaProvisional {
		return vacio, ports.ErrRespuestaBolsaNoConfiable
	}
	c, err := p.contexto(ctx, d, d.operacionOrden, puertosbolsa.AccionConsultarOrigenCT, origen.refBolsa)
	if err != nil {
		return vacio, err
	}
	return ports.ComandoPrepararOrdenBolsa{Contexto: c, Necesidad: origen.refNecesidad,
		Bolsa: origen.refBolsa, Politica: origen.refPolitica, MaximoPosiciones: posiciones}, nil
}

func (p *puenteBolsaLlamamientoDesarrollo) prepararOrdenConstituidaCT(
	ctx context.Context, d preparacionLlamamientoDesarrollo, comando ports.ComandoPrepararOrdenBolsa,
) (ports.ReciboOrdenBolsa, error) {
	vacio := ports.ReciboOrdenBolsa{}
	origen, err := p.prepararOrigenConstituidoCT(ctx, d)
	if err != nil {
		return vacio, err
	}
	posiciones, _, err := posicionesOrigenConstituidoCT(origen.origen.registro)
	if err != nil {
		return vacio, err
	}
	c, err := p.verificarPeticion(ctx, d, comando.Contexto, d.operacionOrden,
		puertosbolsa.AccionConsultarOrigenCT, origen.refBolsa)
	if err != nil || comando.Necesidad != origen.refNecesidad || comando.Bolsa != origen.refBolsa ||
		comando.Politica != origen.refPolitica || comando.MaximoPosiciones != posiciones {
		return vacio, ports.ErrPeticionIntegracionBolsaInvalida
	}
	ahora := p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	refResultado, err := referenciaResultadoOrigenCT("resultado-orden", c, origen.origen.registro)
	if err != nil {
		return vacio, err
	}
	o := origen.origen.registro
	recibo := ports.ReciboOrdenBolsa{
		OperacionRef: c.OperacionRef, OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
		VersionExpediente: c.VersionExpediente, CorrelacionRef: c.CorrelacionRef,
		Necesidad: origen.refNecesidad, Bolsa: origen.refBolsa, Politica: origen.refPolitica,
		Resultado: refResultado, OrdenGenerada: true, OrdenCompleta: true, OrdenAtestada: true,
		Orden: referenciaVersionadaPuenteLlamamientoDesarrollo(o.InstantaneaRef,
			o.VersionInstantanea, o.HuellaInstantaneaSHA256),
		AccionLlamamiento: referenciaCatalogoPuenteLlamamientoDesarrollo(puertosbolsa.AccionEmitirLlamamiento),
		TotalPosiciones:   posiciones, AuditoriaRef: origen.origen.auditoriaRef,
		ConfirmadaEn: ahora, Procedencia: p.procedenciaOrigenConstituidoCT(c, origen.origen, "orden", ahora),
	}
	return p.emisorRespuesta.FirmarOrden(ctx, comando, recibo, ahora)
}

func (p *puenteBolsaLlamamientoDesarrollo) prepararContextoLlamamientoConstituidoCT(
	ctx context.Context, d preparacionLlamamientoDesarrollo, recibo ports.ReciboOrdenBolsa,
) (ports.ContextoPeticionIntegracionBolsa, error) {
	origen, err := p.prepararOrigenConstituidoCT(ctx, d)
	if err != nil {
		return ports.ContextoPeticionIntegracionBolsa{}, err
	}
	posiciones, _, err := posicionesOrigenConstituidoCT(origen.origen.registro)
	if err != nil {
		return ports.ContextoPeticionIntegracionBolsa{}, err
	}
	o := origen.origen.registro
	refResultado, err := referenciaResultadoOrigenCT("resultado-orden",
		ports.DatosContextoPeticionIntegracionBolsa{OperacionRef: recibo.OperacionRef}, o)
	if err != nil || d.expediente.VersionActual != d.expediente.Fiscalizado.Version ||
		recibo.OperacionRef != d.operacionOrden || !recibo.OrdenGenerada || !recibo.OrdenCompleta ||
		!recibo.OrdenAtestada || !recibo.Procedencia.SinRetencionDeclarada ||
		recibo.Procedencia.PoliticaProvisional != o.PoliticaProvisional ||
		recibo.Necesidad != origen.refNecesidad || recibo.Bolsa != origen.refBolsa ||
		recibo.Politica != origen.refPolitica || recibo.Resultado != refResultado ||
		recibo.TotalPosiciones != posiciones || recibo.AccionLlamamiento !=
		referenciaCatalogoPuenteLlamamientoDesarrollo(puertosbolsa.AccionEmitirLlamamiento) ||
		recibo.Orden != referenciaVersionadaPuenteLlamamientoDesarrollo(
			o.InstantaneaRef, o.VersionInstantanea, o.HuellaInstantaneaSHA256) {
		return ports.ContextoPeticionIntegracionBolsa{}, ports.ErrPeticionIntegracionBolsaInvalida
	}
	return p.contexto(ctx, d, d.operacionPropuesta, puertosbolsa.AccionEmitirLlamamiento, recibo.Orden)
}

func (p *puenteBolsaLlamamientoDesarrollo) solicitarLlamamientoConstituidoCT(
	ctx context.Context, d preparacionLlamamientoDesarrollo, comando ports.ComandoSolicitarLlamamientoBolsa,
) (ports.ReciboSolicitudLlamamientoBolsa, error) {
	vacio := ports.ReciboSolicitudLlamamientoBolsa{}
	preparado, err := p.prepararOrigenConstituidoCT(ctx, d)
	if err != nil {
		return vacio, err
	}
	posiciones, _, err := posicionesOrigenConstituidoCT(preparado.origen.registro)
	if err != nil || preparado.seleccion.OrdenVigente == nil {
		return vacio, ports.ErrRespuestaBolsaNoConfiable
	}
	ahora := p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	datos, err := comando.DatosEn(ahora)
	if err != nil {
		return vacio, err
	}
	o := preparado.origen.registro
	orden := referenciaVersionadaPuenteLlamamientoDesarrollo(o.InstantaneaRef,
		o.VersionInstantanea, o.HuellaInstantaneaSHA256)
	c, err := p.verificarPeticion(ctx, d, datos.Contexto, d.operacionPropuesta,
		puertosbolsa.AccionEmitirLlamamiento, orden)
	if err != nil || datos.Necesidad != preparado.refNecesidad || datos.Bolsa != preparado.refBolsa ||
		datos.Politica != preparado.refPolitica || datos.Orden != orden ||
		datos.TotalPosicionesOrden != posiciones || datos.MaximaPosicionEvaluable != posiciones {
		return vacio, ports.ErrPeticionIntegracionBolsaInvalida
	}
	emision, err := p.emitirLlamamientoConstituidoCT(ctx, d, preparado)
	if err != nil {
		return vacio, err
	}
	return p.firmarLlamamientoConstituidoCT(ctx, comando, c, preparado, emision)
}

func (p *puenteBolsaLlamamientoDesarrollo) firmarLlamamientoConstituidoCT(
	ctx context.Context, comando ports.ComandoSolicitarLlamamientoBolsa,
	c ports.DatosContextoPeticionIntegracionBolsa, preparado preparacionOrigenConstituidoCT,
	emision puertosbolsa.EmisionLlamamiento,
) (ports.ReciboSolicitudLlamamientoBolsa, error) {
	vacio := ports.ReciboSolicitudLlamamientoBolsa{}
	r := emision.OrigenCT
	if r == nil || !referenciasConsumoLlamamientoCTValidas(r.DecisionRef, r.AuditoriaRef, r.EventoRef) ||
		r.EmitidoEn.IsZero() || r.EmitidoEn.Location() != time.UTC {
		return vacio, ports.ErrRespuestaBolsaNoConfiable
	}
	material, err := json.Marshal([]string{r.LlamamientoRef, r.ReciboRef, r.DecisionRef,
		r.AuditoriaRef, r.EventoRef, r.ParticipacionRef, r.CandidatoRef,
		r.EmitidoEn.Format(time.RFC3339Nano)})
	if err != nil {
		return vacio, ports.ErrRespuestaBolsaNoConfiable
	}
	huella := huellaPuenteLlamamientoDesarrollo(material)
	seleccionMaterial, err := json.Marshal([]string{c.OrganizacionRef, c.ExpedienteRef,
		r.LlamamientoRef, r.ParticipacionRef, r.CandidatoRef})
	if err != nil {
		return vacio, ports.ErrRespuestaBolsaNoConfiable
	}
	sello, err := p.seleccion.SellarDatos(ctx, seleccionMaterial)
	clear(seleccionMaterial)
	if err != nil {
		return vacio, err
	}
	seudonimo, err := ports.NuevoSeudonimoSeleccionBolsa(sello)
	if err != nil {
		return vacio, err
	}
	ahora := p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	datos, err := comando.DatosEn(ahora)
	if err != nil {
		return vacio, err
	}
	recibo := ports.ReciboSolicitudLlamamientoBolsa{
		OperacionRef: c.OperacionRef, OrganizacionRef: c.OrganizacionRef,
		ExpedienteRef: c.ExpedienteRef, VersionExpediente: c.VersionExpediente,
		CorrelacionRef: c.CorrelacionRef, Necesidad: preparado.refNecesidad,
		Bolsa: preparado.refBolsa, Orden: datos.Orden, Politica: preparado.refPolitica,
		Resultado:         referenciaVersionadaPuenteLlamamientoDesarrollo(r.ReciboRef, 1, huella),
		PropuestaGenerada: true,
		Propuesta:         referenciaVersionadaPuenteLlamamientoDesarrollo(r.DecisionRef, 1, huella),
		AccionEvento:      referenciaCatalogoPuenteLlamamientoDesarrollo(puertosbolsa.AccionEmitirLlamamiento),
		LlamamientoRef:    r.LlamamientoRef, SeleccionRef: seudonimo,
		OrdenSeleccionado: uint32(r.OrdenSeleccionado), ReciboRef: r.ReciboRef,
		AuditoriaRef: r.AuditoriaRef, EventoRef: r.EventoRef, ConfirmadaEn: r.EmitidoEn,
		LlamamientoRecuperado: r.EmitidoEn.Before(c.SolicitadaEn),
		Procedencia:           p.procedenciaOrigenConstituidoCT(c, preparado.origen, "llamamiento", ahora),
	}
	return p.emisorRespuesta.FirmarLlamamiento(ctx, comando, recibo, ahora)
}

var (
	patronDecisionConsumoLlamamientoCT  = regexp.MustCompile(`^decision:[0-9a-f]{32}$`)
	patronAuditoriaConsumoLlamamientoCT = regexp.MustCompile(`^aud_v3_[0-9a-f]{32}$`)
)

// El SQL devuelve referencias V3 ya consumidas. Sus nonces hexadecimales
// pueden parecer un DNI por azar; aquí se exige la familia exacta de cada
// acuse sin relajar las demás referencias de Bolsa.
func referenciasConsumoLlamamientoCTValidas(decisionRef, auditoriaRef, eventoRef string) bool {
	return patronDecisionConsumoLlamamientoCT.MatchString(decisionRef) &&
		patronAuditoriaConsumoLlamamientoCT.MatchString(auditoriaRef) &&
		puertosbolsa.ReferenciaOpacaLlamamientoValida(eventoRef)
}

// La respuesta firme pertenece al portal de Bolsa. La lectura B98 la aporta
// composición dentro del detalle CT autorizado y auditado; un contacto o una
// situación de participación nunca sustituyen la respuesta de la persona.
func (p *puenteBolsaLlamamientoDesarrollo) verificarRespuestaPortalConstituidaCT(
	ctx context.Context, s ports.SolicitudResolverLlamamiento, seleccion ports.ReciboSolicitudLlamamientoBolsa,
) (puertosbolsa.ResultadoParticipacionEmisionCT, error) {
	vacia := puertosbolsa.ResultadoParticipacionEmisionCT{}
	if p == nil || p.alta == nil || p.alta.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.resultadoEmisionNativo) ||
		ctx == nil || ctx.Err() != nil || s.Validar() != nil ||
		!seleccion.Procedencia.SinRetencionDeclarada || !seleccion.PropuestaGenerada ||
		seleccion.OrganizacionRef != s.OrganizacionRef || seleccion.ExpedienteRef != s.ExpedienteRef ||
		seleccion.LlamamientoRef != s.LlamamientoRef || seleccion.Bolsa.Validar() != nil ||
		!puertosbolsa.ReferenciaOpacaLlamamientoValida(seleccion.ReciboRef) {
		return vacia, ports.ErrRespuestaBolsaNoConfiable
	}
	capacidad, valida := p.alta.soporte.capacidadValida(ctx)
	if !valida || capacidad.ruta != httpinterno.RutaResolucionComunicacionLlamamiento {
		return vacia, ports.ErrAutorizacionDenegada
	}
	r, err := p.resultadoEmisionNativo.LeerResultadoEmisionCT(ctx,
		seleccion.Bolsa.Referencia, s.LlamamientoRef, seleccion.ReciboRef)
	if err != nil {
		return vacia, err
	}
	if r.BolsaRef != seleccion.Bolsa.Referencia || r.LlamamientoRef != s.LlamamientoRef ||
		r.ReciboEmisionRef != seleccion.ReciboRef || !r.EmitidoEn.Equal(seleccion.ConfirmadaEn) ||
		len(r.Participaciones) != 1 {
		return vacia, ports.ErrRespuestaBolsaNoConfiable
	}
	part := r.Participaciones[0]
	esperada := puertosbolsa.RespuestaPortalAcepta
	if s.Respuesta == ports.RespuestaLlamamientoRenunciada {
		esperada = puertosbolsa.RespuestaPortalRenuncia
	}
	if part.Modo != puertosbolsa.ModoRespuestaPortalFirme ||
		(part.Respuesta != esperada &&
			!(s.Respuesta == ports.RespuestaLlamamientoRenunciada && part.Respuesta == puertosbolsa.RespuestaPortalRenunciaJustificada)) ||
		!puertosbolsa.ReferenciaOpacaLlamamientoValida(part.ParticipacionRef) ||
		!puertosbolsa.ReferenciaOpacaLlamamientoValida(part.ReciboRespuestaRef) ||
		part.RespondidaEn == nil || part.RespondidaEn.Location() != time.UTC ||
		part.RespondidaEn.Before(r.EmitidoEn) || part.RespondidaEn.After(p.reloj.Ahora()) {
		return vacia, ports.ErrRespuestaBolsaNoConfiable
	}
	return part, nil
}

func decodificarCanonBolsaCT(raw []byte, destino any) error {
	if len(raw) == 0 || len(raw) > 1<<20 {
		return ports.ErrRespuestaBolsaNoConfiable
	}
	decodificador := json.NewDecoder(bytes.NewReader(raw))
	decodificador.DisallowUnknownFields()
	if decodificador.Decode(destino) != nil || decodificador.Decode(new(any)) != io.EOF {
		return ports.ErrRespuestaBolsaNoConfiable
	}
	return nil
}

func clonarLecturaOrigenCT(o puertosbolsa.OrigenLlamamientoCT) puertosbolsa.OrigenLlamamientoCT {
	copia := o
	copia.CriteriosOfrecimiento = o.CriteriosOfrecimiento.Clonar()
	copia.BolsaCanonica = bytes.Clone(o.BolsaCanonica)
	copia.InstantaneaCanonica = bytes.Clone(o.InstantaneaCanonica)
	copia.Entradas = append([]puertosbolsa.EntradaOrigenLlamamientoCT(nil), o.Entradas...)
	for i := range copia.Entradas {
		if o.Entradas[i].EventosRelacion != nil {
			eventos := append([]string{}, (*o.Entradas[i].EventosRelacion)...)
			copia.Entradas[i].EventosRelacion = &eventos
		}
		if anterior := o.Entradas[i].OrdenVigente; anterior != nil {
			orden := *anterior
			copia.Entradas[i].OrdenVigente = &orden
		}
	}
	if o.BolsaVigenteHasta != nil {
		fecha := *o.BolsaVigenteHasta
		copia.BolsaVigenteHasta = &fecha
	}
	if o.PoliticaVigenteHasta != nil {
		fecha := *o.PoliticaVigenteHasta
		copia.PoliticaVigenteHasta = &fecha
	}
	return copia
}

func (p *puenteBolsaLlamamientoDesarrollo) cargarOrigenConstituidoCT(ctx context.Context, d preparacionLlamamientoDesarrollo) (origenConstituidoVerificadoCT, error) {
	vacio := origenConstituidoVerificadoCT{}
	if p == nil || p.alta == nil || p.alta.soporte == nil || p.origenNativo == nil ||
		p.auditarOrdenNativo == nil || ctx == nil || ctx.Err() != nil {
		return vacio, ports.ErrIntegracionBolsaNoDisponible
	}
	sesion, err := p.capturaOperacionNativa(ctx)
	if err != nil || sesion.bolsa.Resultado.Validar() != nil ||
		sesion.bolsa.Vinculo.ValidarPara(sesion.bolsa.Resultado) != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	vinculoCanonico, err := json.Marshal(sesion.bolsa.Vinculo)
	if err != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	var captura puertosbolsa.CapturaOrigenCT
	o, reserva, acuse, err := p.origenNativo.ConsultarOrigenCTAuditado(sesion.ctx,
		d.expediente.Fiscalizado.Referencia, d.clave, d.categoria, d.unidad,
		sesion.bolsa.Resultado.RepresentacionCanonica, vinculoCanonico,
		func(bolsaRef string) (puertosbolsa.CapturaOrigenCT, error) {
			construida, fallo := p.auditarOrdenNativo(sesion.ctx, d, bolsaRef)
			captura = construida
			return construida, fallo
		})
	if err != nil {
		return vacio, err
	}
	if captura.Permiso.BolsaRef != o.BolsaRef {
		return vacio, ports.ErrRespuestaBolsaNoConfiable
	}
	if err := acuseOrigenCTVinculadoAlIntento(acuse, captura.Auditoria); err != nil {
		return vacio, errors.Join(ports.ErrRespuestaBolsaNoConfiable, err)
	}
	if reserva != nil && (reserva.ExpedienteRef != d.expediente.Fiscalizado.Referencia ||
		reserva.UnidadRef != d.unidad || reserva.ClaveCT != d.clave ||
		reserva.ParticipacionRef == "" || reserva.CandidatoRef == "" ||
		reserva.OrdenSeleccionado == 0) {
		return vacio, ports.ErrRespuestaBolsaNoConfiable
	}
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	var bolsa dominiobolsa.BolsaConstituida
	var instantanea dominiobolsa.InstantaneaOrdenBolsa
	if o.CriteriosOfrecimiento.Validar() != nil ||
		decodificarCanonBolsaCT(o.BolsaCanonica, &bolsa) != nil ||
		decodificarCanonBolsaCT(o.InstantaneaCanonica, &instantanea) != nil ||
		bolsa.Validar() != nil || instantanea.Validar() != nil {
		return vacio, ports.ErrRespuestaBolsaNoConfiable
	}
	huella, err := bolsa.HuellaCanonicaSHA256()
	huellaRawInstantanea := sha256.Sum256(o.InstantaneaCanonica)
	if err != nil || o.CategoriaRef != d.categoria || bolsa.CategoriaRef != d.categoria || o.BolsaRef != bolsa.BolsaRef ||
		!puertosbolsa.ReferenciaOpacaLlamamientoValida(o.ActaRef) ||
		!puertosbolsa.ReferenciaOpacaLlamamientoValida(o.PoliticaRef) || o.VersionPolitica == 0 ||
		!huellaSHA256ValidaContratacionTemporalDesarrollo(o.HuellaPoliticaSHA256) ||
		o.Corte.IsZero() || o.Corte.Location() != time.UTC ||
		o.VersionBolsa != bolsa.Version || o.HuellaBolsaSHA256 != huella ||
		o.InstantaneaRef != instantanea.InstantaneaRef || o.VersionInstantanea != instantanea.Version ||
		o.HuellaInstantaneaSHA256 != hex.EncodeToString(huellaRawInstantanea[:]) ||
		instantanea.BolsaRef != bolsa.BolsaRef || instantanea.VersionBolsa != bolsa.Version ||
		instantanea.HuellaBolsaSHA256 != huella || len(o.Entradas) == 0 ||
		len(o.Entradas) > int(ports.MaximoElementosIntegracionBolsa) ||
		len(o.Entradas) != len(instantanea.Entradas) {
		return vacio, ports.ErrRespuestaBolsaNoConfiable
	}
	for n, entrada := range o.Entradas {
		if entrada.OrdenActa != instantanea.Entradas[n].Orden ||
			entrada.ParticipacionRef != instantanea.Entradas[n].Participacion.ParticipacionRef ||
			(entrada.Elegible && (entrada.OrdenVigente == nil ||
				!puertosbolsa.ReferenciaOpacaLlamamientoValida(entrada.CandidatoRef))) {
			return vacio, ports.ErrRespuestaBolsaNoConfiable
		}
	}
	return origenConstituidoVerificadoCT{registro: o, ordenPermiso: bytes.Clone(captura.Permiso.Orden), bolsa: bolsa, instantanea: instantanea,
		reserva: reserva, auditoriaRef: acuse.AuditoriaRef}, nil
}

// AD259 deriva la referencia del asiento del nonce del intento validado.
// Esta familia se coteja con su orden original; el filtro de documentos de
// Bolsa puede confundir por azar cifras hexadecimales con un DNI.
func acuseOrigenCTVinculadoAlIntento(acuse vecdomain.AcuseIntentoAuditoria, orden vecdomain.OrdenIntentoAuditoria) error {
	datos, err := orden.Datos()
	if err != nil {
		return err
	}
	if err := acuse.ValidarPara(orden); err != nil {
		return err
	}
	if acuse.AuditoriaRef != "aud_v3_i_"+datos.IntentoRef[len("intento_"):] {
		return ports.ErrRespuestaBolsaNoConfiable
	}
	return nil
}

// El orden y la elegibilidad ya los decidió Bolsa. CT sólo toma el menor
// orden vigente inequívoco; nunca interpreta la situación de la persona.
func primeraElegibleOrigenConstituidoCT(o puertosbolsa.OrigenLlamamientoCT) (puertosbolsa.EntradaOrigenLlamamientoCT, error) {
	var elegida puertosbolsa.EntradaOrigenLlamamientoCT
	var primeraPendiente *uint64
	ordenes := make(map[uint64]struct{}, len(o.Entradas))
	for _, entrada := range o.Entradas {
		if entrada.Razon == "pendiente_comprobacion_rrhh" {
			if entrada.Elegible || entrada.OrdenVigente != nil {
				return puertosbolsa.EntradaOrigenLlamamientoCT{}, ports.ErrRespuestaBolsaNoConfiable
			}
			if primeraPendiente == nil || entrada.OrdenActa < *primeraPendiente {
				orden := entrada.OrdenActa
				primeraPendiente = &orden
			}
		}
		if entrada.OrdenVigente != nil {
			if *entrada.OrdenVigente == 0 {
				return puertosbolsa.EntradaOrigenLlamamientoCT{}, ports.ErrRespuestaBolsaNoConfiable
			}
			if _, repetido := ordenes[*entrada.OrdenVigente]; repetido {
				return puertosbolsa.EntradaOrigenLlamamientoCT{}, ports.ErrRespuestaBolsaNoConfiable
			}
			ordenes[*entrada.OrdenVigente] = struct{}{}
		}
		if !entrada.Elegible {
			continue
		}
		if entrada.OrdenVigente == nil || entrada.CandidatoRef == "" {
			return puertosbolsa.EntradaOrigenLlamamientoCT{}, ports.ErrRespuestaBolsaNoConfiable
		}
		if elegida.OrdenVigente == nil || *entrada.OrdenVigente < *elegida.OrdenVigente {
			elegida = entrada
		}
	}
	if primeraPendiente != nil && (elegida.OrdenVigente == nil || *primeraPendiente < elegida.OrdenActa) {
		var version int64
		if o.CriteriosOfrecimiento != nil {
			version = o.CriteriosOfrecimiento.VersionPoliticaOfertas
		}
		return puertosbolsa.EntradaOrigenLlamamientoCT{}, &ports.ErrorCriteriosOfrecimientoPendientesRRHH{
			Comprobacion: ports.ComprobacionOfrecimientoPendiente{
				BolsaRef: o.BolsaRef, OrdenActa: *primeraPendiente,
				VersionPoliticaOfertas: version, MotivoClave: "pendiente_comprobacion_rrhh",
			},
		}
	}
	if elegida.OrdenVigente == nil {
		return puertosbolsa.EntradaOrigenLlamamientoCT{}, ports.ErrIntegracionBolsaNoDisponible
	}
	if _, err := ordenActaPrimeraElegibleOrigenCT(elegida); err != nil {
		return puertosbolsa.EntradaOrigenLlamamientoCT{}, err
	}
	return elegida, nil
}

// El rango vigente decide a quién llamar; B110 conserva como orden seleccionado
// su posición en el acta original, aunque haya personas anteriores sin turno.
func ordenActaPrimeraElegibleOrigenCT(e puertosbolsa.EntradaOrigenLlamamientoCT) (uint64, error) {
	if e.OrdenActa == 0 || !e.Elegible || e.OrdenVigente == nil || *e.OrdenVigente != 1 ||
		e.ParticipacionRef == "" || e.CandidatoRef == "" {
		return 0, ports.ErrRespuestaBolsaNoConfiable
	}
	return e.OrdenActa, nil
}

// El primer paso de CT precede a ResolverTerminal. Un recibo legado ya
// confirmado debe poder consultar su fuente original sin fabricar otra
// bolsa ni permitir una apertura sintética nueva.
func (p *puenteBolsaLlamamientoDesarrollo) fuenteLegadaReservadaCT(
	ctx context.Context, d preparacionLlamamientoDesarrollo,
) (fuentesintetica.DocumentoFuenteLlamamientos, bool, error) {
	vacio := fuentesintetica.DocumentoFuenteLlamamientos{}
	if p == nil || dependenciaEsNulaContratacionTemporalDesarrollo(p.repositorio) ||
		len(p.privadaFuente) != ed25519.PrivateKeySize {
		return vacio, false, ports.ErrIntegracionBolsaNoDisponible
	}
	apertura, existe, err := p.repositorio.BuscarOperacion(ctx, d.operacionPropuesta)
	if err != nil || !existe {
		return vacio, false, err
	}
	orden, existe, err := p.repositorio.BuscarOperacion(ctx, d.operacionOrden)
	if err != nil || !existe || apertura.Tipo != "propuesta" || orden.Tipo != "orden" ||
		apertura.OperacionRef != d.operacionPropuesta || apertura.OrdenOperacionRef != d.operacionOrden ||
		apertura.NecesidadRef != d.necesidad || apertura.VersionNecesidad != 6 ||
		apertura.CategoriaRef != d.categoria || apertura.UnidadRef != d.unidad ||
		orden.NecesidadRef != d.necesidad || orden.VersionNecesidad != 6 ||
		!bytes.Equal(apertura.Fuente, orden.Fuente) || !bytes.Equal(apertura.FirmaFuente, orden.FirmaFuente) {
		return vacio, false, ports.ErrRespuestaBolsaNoConfiable
	}
	if _, err := fuentesintetica.NuevaFuenteLlamamientos(apertura.Fuente, apertura.FirmaFuente,
		p.privadaFuente.Public().(ed25519.PublicKey), p.reloj); err != nil {
		return vacio, false, ports.ErrRespuestaBolsaNoConfiable
	}
	var fuente fuentesintetica.DocumentoFuenteLlamamientos
	if json.Unmarshal(apertura.Fuente, &fuente) != nil ||
		fuente.Datos.Necesidad.NecesidadRef != d.necesidad ||
		fuente.Datos.Necesidad.CategoriaRef != d.categoria ||
		fuente.Datos.Necesidad.UnidadRef != d.unidad ||
		fuente.Datos.Bolsa.BolsaRef != apertura.Instantanea.BolsaRef ||
		apertura.Instantanea.InstantaneaRef != orden.Instantanea.InstantaneaRef ||
		apertura.Instantanea.Version != orden.Instantanea.Version ||
		apertura.Instantanea.HuellaContenidoSHA256 != orden.Instantanea.HuellaContenidoSHA256 {
		return vacio, false, ports.ErrRespuestaBolsaNoConfiable
	}
	return fuente, true, nil
}

func (p *puenteBolsaLlamamientoDesarrollo) prepararOrigenConstituidoCT(
	ctx context.Context, d preparacionLlamamientoDesarrollo,
) (preparacionOrigenConstituidoCT, error) {
	if ctx == nil {
		return preparacionOrigenConstituidoCT{}, ports.ErrPeticionIntegracionBolsaInvalida
	}
	estado, ok := ctx.Value(claveEstadoOrigenConstituidoCT{}).(*estadoOrigenConstituidoCT)
	if !ok {
		return preparacionOrigenConstituidoCT{}, ports.ErrPeticionIntegracionBolsaInvalida
	}
	return estado.cargar(ctx, d, func() (preparacionOrigenConstituidoCT, error) {
		return p.prepararOrigenConstituidoSinCacheCT(ctx, d)
	})
}

func (p *puenteBolsaLlamamientoDesarrollo) prepararOrigenConstituidoSinCacheCT(
	ctx context.Context, d preparacionLlamamientoDesarrollo,
) (preparacionOrigenConstituidoCT, error) {
	vacio := preparacionOrigenConstituidoCT{}
	if p == nil || p.emisorNativo == nil || p.origenNativo == nil ||
		p.configuracionNativa == nil || p.auditarOrdenNativo == nil || d.expediente.Fiscalizado.Analisis == nil ||
		d.expediente.Fiscalizado.Fiscalizacion == nil {
		return vacio, ports.ErrIntegracionBolsaNoDisponible
	}
	origen, err := p.cargarOrigenConstituidoCT(ctx, d)
	if err != nil {
		return vacio, err
	}
	configuracion, puesto, motivo, err := p.configuracionNativa(ctx, d)
	if err != nil || !puertosbolsa.ReferenciaOpacaLlamamientoValida(puesto) || motivo.Validar() != nil {
		return vacio, ports.ErrIntegracionBolsaNoDisponible
	}
	reserva := origen.reserva
	if reserva != nil {
		if reserva.UnidadRef != d.unidad ||
			reserva.ExpedienteRef != d.expediente.Fiscalizado.Referencia ||
			reserva.ClaveCT != d.clave {
			return vacio, ports.ErrRespuestaBolsaNoConfiable
		}
		// La concesión actual incluye exactamente los diez campos del comando
		// B7 que se necesitan para reproducir la reserva original.
		configuracion = reserva.Configuracion
	}
	e := d.expediente.Fiscalizado
	fin := time.Time{}
	if !e.Analisis.Periodo.Fin.IsZero() {
		fin = e.Analisis.Periodo.Fin.AddDate(0, 0, 1)
	}
	necesidad, err := dominiobolsa.NuevaNecesidadCobertura(dominiobolsa.AltaNecesidadCobertura{
		NecesidadRef: d.necesidad, Version: 6, BolsaRef: origen.bolsa.BolsaRef,
		VersionBolsa: origen.bolsa.Version, HuellaBolsaSHA256: origen.registro.HuellaBolsaSHA256,
		CategoriaRef: d.categoria, PuestoRef: puesto, UnidadRef: d.unidad,
		TipoCoberturaRef: string(e.Analisis.ModalidadClave), NumeroPuestos: 1,
		InicioPrevisto: e.Analisis.Periodo.Inicio, FinPrevisto: fin,
		CausaFinClave: string(e.Analisis.Periodo.CausaFin), CreadaEn: e.Fiscalizacion.FiscalizadaEn,
	})
	if err != nil {
		return vacio, ports.ErrRespuestaBolsaNoConfiable
	}
	seleccion, err := primeraElegibleOrigenConstituidoCT(origen.registro)
	if err != nil {
		var pendiente *ports.ErrorCriteriosOfrecimientoPendientesRRHH
		if errors.As(err, &pendiente) {
			return vacio, enriquecerErrorCriteriosPendientesCT(pendiente, origen, d)
		}
		// Una bolsa sin candidato disponible se informa por disponibilidad;
		// aquí se conserva el origen para que esa consulta firme cero exacto.
		if err != ports.ErrIntegracionBolsaNoDisponible {
			return vacio, err
		}
	}
	if reserva != nil {
		ordenActa, err := ordenActaPrimeraElegibleOrigenCT(seleccion)
		if err != nil || reserva.ParticipacionRef != seleccion.ParticipacionRef ||
			reserva.CandidatoRef != seleccion.CandidatoRef || reserva.OrdenSeleccionado != ordenActa {
			return vacio, ports.ErrRespuestaBolsaNoConfiable
		}
	}
	huellaNecesidad, err := necesidad.HuellaCanonicaSHA256()
	if err != nil {
		return vacio, ports.ErrRespuestaBolsaNoConfiable
	}
	o := origen.registro
	return preparacionOrigenConstituidoCT{
		origen: origen, necesidad: necesidad, configuracion: configuracion, motivo: motivo, seleccion: seleccion,
		refNecesidad: referenciaVersionadaPuenteLlamamientoDesarrollo(necesidad.NecesidadRef, necesidad.Version, huellaNecesidad),
		refBolsa:     referenciaVersionadaPuenteLlamamientoDesarrollo(o.BolsaRef, o.VersionBolsa, o.HuellaBolsaSHA256),
		refPolitica:  referenciaVersionadaPuenteLlamamientoDesarrollo(o.PoliticaRef, o.VersionPolitica, o.HuellaPoliticaSHA256),
	}, nil
}

func (p *puenteBolsaLlamamientoDesarrollo) emitirLlamamientoConstituidoCT(
	ctx context.Context, d preparacionLlamamientoDesarrollo, preparado preparacionOrigenConstituidoCT,
) (puertosbolsa.EmisionLlamamiento, error) {
	vacio := puertosbolsa.EmisionLlamamiento{}
	if p == nil || p.emisorNativo == nil || preparado.seleccion.OrdenVigente == nil {
		return vacio, ports.ErrIntegracionBolsaNoDisponible
	}
	ordenActa, err := ordenActaPrimeraElegibleOrigenCT(preparado.seleccion)
	if err != nil {
		return vacio, err
	}
	operativo, err := p.alta.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return vacio, err
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, err
	}
	e := d.expediente.Fiscalizado
	o := preparado.origen.registro
	origenCT := puertosbolsa.OrigenEmisionCT{
		CriteriosOfrecimiento: o.CriteriosOfrecimiento.Clonar(),
		OrganizacionRef:       e.OrganizacionRef, UnidadRef: d.unidad, ExpedienteRef: e.Referencia, VersionExpediente: e.Version,
		ClaveCT: d.clave, NecesidadRef: d.necesidad, ActaRef: o.ActaRef, CategoriaRef: o.CategoriaRef,
		BolsaRef: o.BolsaRef, VersionBolsa: o.VersionBolsa, HuellaBolsaSHA256: o.HuellaBolsaSHA256,
		InstantaneaRef: o.InstantaneaRef, VersionInstantanea: o.VersionInstantanea,
		HuellaInstantaneaSHA256: o.HuellaInstantaneaSHA256,
		PoliticaRef:             o.PoliticaRef, VersionPolitica: o.VersionPolitica, HuellaPoliticaSHA256: o.HuellaPoliticaSHA256,
		ParticipacionRef: preparado.seleccion.ParticipacionRef, CandidatoRef: preparado.seleccion.CandidatoRef,
		OrdenSeleccionado: ordenActa, Lectura: clonarLecturaOrigenCT(o),
	}
	emision, err := p.emisorNativo.EmitirLlamamiento(ctx, puertosbolsa.SolicitudEmitirLlamamiento{
		Vinculo: operativo.Vinculo, ResultadoContexto: operativo.Resultado,
		BolsaRef: o.BolsaRef, Participaciones: []string{preparado.seleccion.ParticipacionRef},
		Configuracion: preparado.configuracion, ClaveIdempotencia: d.clave,
		Correlacion: correlacion, MotivoAutorizacion: preparado.motivo, OrigenCT: &origenCT,
	})
	if err != nil {
		return vacio, err
	}
	r := emision.OrigenCT
	if r == nil || emision.BolsaRef != o.BolsaRef || len(emision.Participaciones) != 1 ||
		emision.Participaciones[0] != origenCT.ParticipacionRef ||
		r.OrganizacionRef != origenCT.OrganizacionRef || r.ExpedienteRef != origenCT.ExpedienteRef ||
		r.VersionExpediente != origenCT.VersionExpediente || r.ClaveCT != origenCT.ClaveCT ||
		r.ActaRef != origenCT.ActaRef || r.CategoriaRef != origenCT.CategoriaRef ||
		r.BolsaRef != origenCT.BolsaRef || r.VersionBolsa != origenCT.VersionBolsa ||
		r.HuellaBolsaSHA256 != origenCT.HuellaBolsaSHA256 ||
		r.InstantaneaRef != origenCT.InstantaneaRef || r.VersionInstantanea != origenCT.VersionInstantanea ||
		r.HuellaInstantaneaSHA256 != origenCT.HuellaInstantaneaSHA256 ||
		r.PoliticaRef != origenCT.PoliticaRef || r.VersionPolitica != origenCT.VersionPolitica ||
		r.HuellaPoliticaSHA256 != origenCT.HuellaPoliticaSHA256 ||
		!r.CriteriosOfrecimiento.Iguales(origenCT.CriteriosOfrecimiento) ||
		!r.Lectura.CriteriosOfrecimiento.Iguales(o.CriteriosOfrecimiento) ||
		r.ParticipacionRef != origenCT.ParticipacionRef || r.CandidatoRef != origenCT.CandidatoRef ||
		r.OrdenSeleccionado != origenCT.OrdenSeleccionado ||
		r.LlamamientoRef != emision.LlamamientoRef || r.ReciboRef != emision.ReciboRef ||
		!bytes.Equal(r.Lectura.BolsaCanonica, o.BolsaCanonica) ||
		!bytes.Equal(r.Lectura.InstantaneaCanonica, o.InstantaneaCanonica) {
		return vacio, ports.ErrRespuestaBolsaNoConfiable
	}
	return emision, nil
}
