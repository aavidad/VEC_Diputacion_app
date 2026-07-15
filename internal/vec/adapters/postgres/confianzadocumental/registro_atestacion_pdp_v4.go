package confianzadocumental

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	esquemaOrdenGeneracionDocumentalV4 = "vec.documentos.orden-generacion.v4"
	estadoOrdenGeneracionPendienteV4   = "pendiente_generacion"
)

// datosRegistroAtestacionPDPDocumentalV4 no cruza la frontera del paquete.
// Es la proyeccion exacta que solo puede construir Servicio despues de volver
// a verificar COSE, VEC-AD-1, la decision completa y la preimagen. Al no haber
// constructor ni interfaz publica, un paquete del servidor no puede crear una
// pareja propia y convertir sus datos en autoridad.
type datosRegistroAtestacionPDPDocumentalV4 struct {
	ProyeccionAplicacion     ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4
	PreimagenRecurso         ports.PreimagenRecursoAutorizacionEjecucionDocumentalV4
	FormatoVECADVersion      uint16
	Suite                    string
	ClaveID                  string
	AudienciaDespliegue      string
	AlgoritmoCOSE            string
	AudienciaCOSE            string
	EstadoConfianza          string
	HuellaClaveSHA256        string
	HuellaPayloadSHA256      string
	HuellaSobreSHA256        string
	VerificadaEn             time.Time
	RaizValidaDesde          time.Time
	RaizValidaHasta          time.Time
	RevisionConfianza        string
	HuellaConfiguracion      string
	ConfiguracionPublicadaEn time.Time
	ConfiguracionExpiraEn    time.Time
	HuellaEvidenciaSHA256    string
	PayloadVECAD1            []byte
	SobreCOSESign1           []byte
	EvidenciaCanonica        []byte
	DecisionCanonica         []byte
}

// pruebaRegistroAtestacionPDPDocumentalV4 es inforjable fuera de este paquete:
// todos sus campos y su constructor son privados. El repositorio configurado
// en Servicio recibe este valor directamente, nunca un DTO del llamador.
type pruebaRegistroAtestacionPDPDocumentalV4 struct {
	datos *datosRegistroAtestacionPDPDocumentalV4
}

type ordenGeneracionDocumentalV4 struct {
	Esquema           string
	OrdenRef          string
	Estado            string
	DecisionRef       string
	EfectoRef         string
	HuellaPlanSHA256  string
	HuellaDecision    string
	HuellaAplicacion  string
	HuellaOrdenSHA256 string
	AuditoriaRef      string
	EventoOutboxRef   string
	CorrelacionRef    string
	SolicitadaEn      time.Time
}

type solicitudEjecucionDocumentalAtestadaV4 struct {
	prueba pruebaRegistroAtestacionPDPDocumentalV4
	orden  ordenGeneracionDocumentalV4
}

// repositorioEjecucionDocumentalV4 es deliberadamente privado. Ni un handler,
// ni otro modulo, ni un adaptador proporcionado por el llamador pueden
// implementarlo o sustituirlo al invocar el caso de uso.
type repositorioEjecucionDocumentalV4 interface {
	ejecutarPlanAtestado(
		context.Context,
		solicitudEjecucionDocumentalAtestadaV4,
	) (ResultadoEjecucionPlanDocumentalV4, error)
}

// ResultadoEjecucionPlanDocumentalV4 solo expone referencias operativas. No
// contiene payload, COSE, identidad personal ni material capaz de reejecutar.
type ResultadoEjecucionPlanDocumentalV4 struct {
	OrdenRef        string
	Estado          string
	AuditoriaRef    string
	EventoOutboxRef string
	RegistradaEn    time.Time
}

func (r ResultadoEjecucionPlanDocumentalV4) validarContra(
	s solicitudEjecucionDocumentalAtestadaV4,
) error {
	if s.validar() != nil || r.OrdenRef != s.orden.OrdenRef ||
		r.Estado != estadoOrdenGeneracionPendienteV4 ||
		r.AuditoriaRef != s.orden.AuditoriaRef ||
		r.EventoOutboxRef != s.orden.EventoOutboxRef ||
		!instanteCanonicoDocumental(r.RegistradaEn) ||
		r.RegistradaEn.Before(s.orden.SolicitadaEn) {
		return errorAutoridadInternaEjecucionDocumentalV4()
	}
	return nil
}

func (ResultadoEjecucionPlanDocumentalV4) String() string {
	return "[RESULTADO-EJECUCION-PLAN-DOCUMENTAL-V4-REDACTADO]"
}
func (r ResultadoEjecucionPlanDocumentalV4) GoString() string { return r.String() }
func (r ResultadoEjecucionPlanDocumentalV4) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
func (r ResultadoEjecucionPlanDocumentalV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}

// EjecutarPlanDocumentalV4 es la unica salida de alto nivel hacia
// PostgreSQL. Revalida el COSE completo en el instante del reloj interno y
// entrega una prueba privada al repositorio fijado durante el ensamblado. El
// repositorio debe registrar/confirmar atestacion, consumir DecisionRef, crear
// la orden documental, auditoria y outbox en un unico COMMIT.
func (s *Servicio) EjecutarPlanDocumentalV4(
	ctx context.Context,
	autoridad AutoridadInternaEjecucionDocumentalV4,
) (ResultadoEjecucionPlanDocumentalV4, error) {
	instante, err := s.capturarInstanteAtestacionPDP(ctx)
	if err != nil || s == nil || s.repositorioEjecucionV4 == nil ||
		autoridad.ValidarEn(instante) != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, denegarAutoridadAtestacionPDP(err)
	}
	solicitudAplicacion, evidencia, err := autoridad.PrepararAplicacionExactaConEvidenciaEn(
		autoridad.clave.DecisionRef,
		autoridad.clave.HuellaPlanSHA256,
		autoridad.clave.EfectoRef,
		instante,
	)
	if err != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, denegarAutoridadAtestacionPDP(nil)
	}
	proyeccion, err := solicitudAplicacion.ProyeccionParaTransaccion()
	if err != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, denegarAutoridadAtestacionPDP(nil)
	}
	payload, errPayload := evidencia.PayloadVECAD1()
	sobreBytes, errSobre := evidencia.SobreCOSESign1()
	sobre, errSobreCrudo := ports.NuevoSobreCriptograficoDocumentalCrudoV4(sobreBytes)
	solicitudCOSE, errSolicitudCOSE := NuevaSolicitudVerificacionCOSESign1(
		payload, AudienciaCOSEAtestacionAutorizacionPDP,
	)
	proyeccionHistorica, errParseo := domain.ParsearMensajeAtestacionAutorizacionV1NoAutoritativo(payload)
	cabeceraFirmada, errCabecera := proyeccionHistorica.Cabecera()
	if errPayload != nil || errSobre != nil || errSobreCrudo != nil ||
		errSolicitudCOSE != nil || errParseo != nil || errCabecera != nil ||
		cabeceraFirmada != autoridad.cabeceraPDP ||
		s.validarCabeceraAtestacionPDP(cabeceraFirmada) != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, denegarAutoridadAtestacionPDP(nil)
	}
	pruebaReverificada, err := s.verificarCOSESign1En(ctx, solicitudCOSE, sobre, instante)
	if err != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, denegarAutoridadAtestacionPDP(err)
	}
	prueba, err := prepararPruebaRegistroAtestacionPDPDocumentalV4(
		autoridad, solicitudAplicacion, proyeccion, evidencia, pruebaReverificada,
	)
	if err != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, denegarAutoridadAtestacionPDP(nil)
	}
	solicitud, err := nuevaSolicitudEjecucionDocumentalAtestadaV4(prueba)
	if err != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, denegarAutoridadAtestacionPDP(nil)
	}
	if err := ctx.Err(); err != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, denegarAutoridadAtestacionPDP(err)
	}
	resultado, err := s.repositorioEjecucionV4.ejecutarPlanAtestado(ctx, solicitud)
	if err != nil || resultado.validarContra(solicitud) != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, denegarAutoridadAtestacionPDP(err)
	}
	return resultado, nil
}

// prepararPruebaRegistroAtestacionPDPDocumentalV4 permanece privada. Cada
// cotejo se realiza sobre la solicitud opaca viva; la huella reforzada se
// recalcula desde los bytes canonicos exactos y esos bytes se conservan para
// que PostgreSQL vuelva a calcularla dentro del COMMIT.
func prepararPruebaRegistroAtestacionPDPDocumentalV4(
	autoridad AutoridadInternaEjecucionDocumentalV4,
	solicitudAplicacion ports.SolicitudAplicacionAutorizacionEjecucionDocumentalV4,
	proyeccion ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4,
	evidencia EvidenciaDurableAtestacionAutorizacionPDPV4,
	pruebaReverificada PruebaCOSESign1DocumentalVerificada,
) (pruebaRegistroAtestacionPDPDocumentalV4, error) {
	proyeccionDerivada, err := solicitudAplicacion.ProyeccionParaTransaccion()
	if err != nil || proyeccion != proyeccionDerivada ||
		autoridad.ValidarEn(proyeccionDerivada.SolicitadaEn) != nil ||
		!evidencia.coincideConAutoridad(autoridad) ||
		verificarPruebaAtestacionPDPContraVinculo(
			pruebaReverificada, autoridad.cabeceraPDP, autoridad.vinculo,
			proyeccionDerivada.SolicitadaEn,
		) != nil {
		return pruebaRegistroAtestacionPDPDocumentalV4{},
			errorAutoridadInternaEjecucionDocumentalV4()
	}

	metadatos, errMetadatos := evidencia.Metadatos()
	payload, errPayload := evidencia.PayloadVECAD1()
	sobre, errSobre := evidencia.SobreCOSESign1()
	preimagenCanonica, errPreimagen := evidencia.PreimagenRecursoCanonica()
	evidenciaCanonica, errEvidencia := evidencia.SerializacionCanonicaParaPersistencia()
	preimagen, errInterpretarPreimagen := ports.InterpretarPreimagenRecursoAutorizacionEjecucionDocumentalV4(
		preimagenCanonica,
		metadatos.HuellaPreimagenRecursoSHA256,
	)
	proyeccionHistorica, errParseo := domain.ParsearMensajeAtestacionAutorizacionV1NoAutoritativo(payload)
	cabeceraFirmada, errCabecera := proyeccionHistorica.Cabecera()
	datosFirmados, errDatos := proyeccionHistorica.Datos()
	evidenciaEstructural, errEvidenciaEstructural := solicitudAplicacion.EvidenciaEstructural()
	datosDecision, errDatosDecision := evidenciaEstructural.Datos()
	decisionCanonica, errDecisionCanonica := datosDecision.RepresentacionCanonica()
	if errMetadatos != nil || errPayload != nil || errSobre != nil || errPreimagen != nil ||
		errEvidencia != nil || errInterpretarPreimagen != nil || errParseo != nil ||
		errCabecera != nil || errDatos != nil || errEvidenciaEstructural != nil ||
		errDatosDecision != nil || errDecisionCanonica != nil ||
		cabeceraFirmada != autoridad.cabeceraPDP ||
		cabeceraFirmada.FormatoVersion != metadatos.FormatoVECADVersion ||
		cabeceraFirmada.Suite != metadatos.Suite || cabeceraFirmada.ClaveID != metadatos.ClaveID ||
		cabeceraFirmada.Audiencia != metadatos.AudienciaDespliegue ||
		solicitudAplicacion.CotejarConDecisionHistoricaAtestacionPDPV1(datosFirmados, preimagen) != nil ||
		metadatos.DecisionRef != proyeccion.Clave.DecisionRef ||
		metadatos.HuellaPlanSHA256 != proyeccion.Clave.HuellaPlanSHA256 ||
		metadatos.EfectoRef != proyeccion.Clave.EfectoRef ||
		metadatos.HuellaSolicitudVinculadaSHA256 != proyeccion.HuellaSolicitudVinculadaSHA256 ||
		metadatos.HuellaContextoRecursoSHA256 != proyeccion.HuellaRecursoSHA256 ||
		metadatos.HuellaAmbitosRecursoSHA256 != proyeccion.HuellaAmbitosSHA256 ||
		pruebaReverificada.huellaPayloadSHA256 != metadatos.HuellaPayloadSHA256 ||
		pruebaReverificada.huellaSobreSHA256 != metadatos.HuellaSobreSHA256 ||
		!pruebaReverificada.verificadaEn.Equal(proyeccion.SolicitadaEn) ||
		huellaBytesDocumentales(decisionCanonica) != proyeccion.HuellaDecisionSHA256 {
		return pruebaRegistroAtestacionPDPDocumentalV4{},
			errorAutoridadInternaEjecucionDocumentalV4()
	}

	datos := &datosRegistroAtestacionPDPDocumentalV4{
		ProyeccionAplicacion:     proyeccion,
		PreimagenRecurso:         preimagen,
		FormatoVECADVersion:      metadatos.FormatoVECADVersion,
		Suite:                    metadatos.Suite,
		ClaveID:                  metadatos.ClaveID,
		AudienciaDespliegue:      metadatos.AudienciaDespliegue,
		AlgoritmoCOSE:            string(pruebaReverificada.algoritmo),
		AudienciaCOSE:            string(pruebaReverificada.audiencia),
		EstadoConfianza:          string(pruebaReverificada.estadoConfianza),
		HuellaClaveSHA256:        pruebaReverificada.huellaClaveSHA256,
		HuellaPayloadSHA256:      pruebaReverificada.huellaPayloadSHA256,
		HuellaSobreSHA256:        pruebaReverificada.huellaSobreSHA256,
		VerificadaEn:             pruebaReverificada.verificadaEn,
		RaizValidaDesde:          pruebaReverificada.raizValidaDesde,
		RaizValidaHasta:          pruebaReverificada.raizValidaHasta,
		RevisionConfianza:        pruebaReverificada.revisionConfianza,
		HuellaConfiguracion:      pruebaReverificada.huellaConfiguracionSHA256,
		ConfiguracionPublicadaEn: pruebaReverificada.configuracionPublicadaEn,
		ConfiguracionExpiraEn:    pruebaReverificada.configuracionExpiraEn,
		HuellaEvidenciaSHA256:    metadatos.HuellaEvidenciaDurableSHA256,
		PayloadVECAD1:            append([]byte(nil), payload...),
		SobreCOSESign1:           append([]byte(nil), sobre...),
		EvidenciaCanonica:        append([]byte(nil), evidenciaCanonica...),
		DecisionCanonica:         append([]byte(nil), decisionCanonica...),
	}
	prueba := pruebaRegistroAtestacionPDPDocumentalV4{datos: datos}
	if prueba.validar() != nil {
		return pruebaRegistroAtestacionPDPDocumentalV4{},
			errorAutoridadInternaEjecucionDocumentalV4()
	}
	return prueba, nil
}

func (prueba pruebaRegistroAtestacionPDPDocumentalV4) validar() error {
	if prueba.datos == nil {
		return errorAutoridadInternaEjecucionDocumentalV4()
	}
	d := prueba.datos
	preimagen, errPreimagen := d.PreimagenRecurso.SerializacionCanonicaParaPersistencia()
	huellaPreimagen, errHuellaPreimagen := d.PreimagenRecurso.HuellaSHA256()
	recurso, errRecurso := d.PreimagenRecurso.RecursoCanonico()
	huellaRecurso, errHuellaRecurso := d.PreimagenRecurso.HuellaContextoRecursoSHA256()
	huellaAmbitos, errHuellaAmbitos := d.PreimagenRecurso.HuellaAmbitosSHA256()
	sobre, errSobre := ports.NuevoSobreCriptograficoDocumentalCrudoV4(d.SobreCOSESign1)
	huellaSobre, errHuellaSobre := sobre.HuellaSHA256()
	proyeccion := d.ProyeccionAplicacion
	if errPreimagen != nil || errHuellaPreimagen != nil || errRecurso != nil ||
		errHuellaRecurso != nil || errHuellaAmbitos != nil || errSobre != nil ||
		errHuellaSobre != nil || len(preimagen) == 0 ||
		!huellaSHA256DocumentalValida(huellaPreimagen) ||
		recurso.Referencia != proyeccion.RecursoRef || recurso.ModuloID != proyeccion.ModuloID ||
		recurso.Tipo != proyeccion.TipoRecurso || huellaRecurso != proyeccion.HuellaRecursoSHA256 ||
		huellaAmbitos != proyeccion.HuellaAmbitosSHA256 ||
		d.FormatoVECADVersion != domain.VersionFormatoAtestacionAutorizacionV1 ||
		d.Suite != suiteAtestacionAutorizacionPDPCOSEEdDSAV1 ||
		d.AlgoritmoCOSE != string(AlgoritmoCOSEDocumentalEdDSA) ||
		d.AudienciaCOSE != string(AudienciaCOSEAtestacionAutorizacionPDP) ||
		d.EstadoConfianza != string(EstadoConfianzaClaveDocumentalActiva) ||
		!referenciaDurableAtestacionPDPValida(d.ClaveID) ||
		!referenciaDurableAtestacionPDPValida(d.AudienciaDespliegue) ||
		!referenciaConfiguracionDocumentalValida(d.RevisionConfianza) ||
		!huellaSHA256DocumentalValida(d.HuellaClaveSHA256) ||
		!huellaSHA256DocumentalValida(d.HuellaPayloadSHA256) ||
		!huellaSHA256DocumentalValida(d.HuellaSobreSHA256) ||
		!huellaSHA256DocumentalValida(d.HuellaConfiguracion) ||
		!huellaSHA256DocumentalValida(d.HuellaEvidenciaSHA256) ||
		huellaBytesDocumentales(d.PayloadVECAD1) != d.HuellaPayloadSHA256 ||
		huellaSobre != d.HuellaSobreSHA256 ||
		huellaBytesDocumentales(d.EvidenciaCanonica) != d.HuellaEvidenciaSHA256 ||
		huellaBytesDocumentales(d.DecisionCanonica) != proyeccion.HuellaDecisionSHA256 ||
		!instanteCanonicoDocumental(d.VerificadaEn) ||
		!instanteCanonicoDocumental(d.RaizValidaDesde) ||
		!instanteCanonicoDocumental(d.RaizValidaHasta) ||
		!instanteCanonicoDocumental(d.ConfiguracionPublicadaEn) ||
		!instanteCanonicoDocumental(d.ConfiguracionExpiraEn) ||
		d.VerificadaEn.Before(d.RaizValidaDesde) || !d.VerificadaEn.Before(d.RaizValidaHasta) ||
		d.VerificadaEn.Before(d.ConfiguracionPublicadaEn) ||
		!d.VerificadaEn.Before(d.ConfiguracionExpiraEn) {
		return errorAutoridadInternaEjecucionDocumentalV4()
	}
	return nil
}

func nuevaSolicitudEjecucionDocumentalAtestadaV4(
	prueba pruebaRegistroAtestacionPDPDocumentalV4,
) (solicitudEjecucionDocumentalAtestadaV4, error) {
	if prueba.validar() != nil {
		return solicitudEjecucionDocumentalAtestadaV4{}, errorAutoridadInternaEjecucionDocumentalV4()
	}
	p := prueba.datos.ProyeccionAplicacion
	base := []byte(p.Clave.DecisionRef + "\x00" + p.Clave.EfectoRef + "\x00" +
		p.Clave.HuellaPlanSHA256 + "\x00" + p.HuellaDecisionSHA256 + "\x00" +
		p.HuellaSolicitudAplicacionSHA256)
	huellaOrden := huellaBytesDocumentales(append([]byte(esquemaOrdenGeneracionDocumentalV4+"\x00"), base...))
	orden := ordenGeneracionDocumentalV4{
		Esquema:           esquemaOrdenGeneracionDocumentalV4,
		OrdenRef:          p.Clave.EfectoRef,
		Estado:            estadoOrdenGeneracionPendienteV4,
		DecisionRef:       p.Clave.DecisionRef,
		EfectoRef:         p.Clave.EfectoRef,
		HuellaPlanSHA256:  p.Clave.HuellaPlanSHA256,
		HuellaDecision:    p.HuellaDecisionSHA256,
		HuellaAplicacion:  p.HuellaSolicitudAplicacionSHA256,
		HuellaOrdenSHA256: huellaOrden,
		AuditoriaRef:      "auditoria:documental:v4:" + huellaBytesDocumentales(append([]byte("auditoria\x00"), base...)),
		EventoOutboxRef:   "evento:documental:v4:" + huellaBytesDocumentales(append([]byte("outbox\x00"), base...)),
		CorrelacionRef:    p.CorrelacionRef,
		SolicitadaEn:      p.SolicitadaEn,
	}
	solicitud := solicitudEjecucionDocumentalAtestadaV4{prueba: prueba, orden: orden}
	if solicitud.validar() != nil {
		return solicitudEjecucionDocumentalAtestadaV4{}, errorAutoridadInternaEjecucionDocumentalV4()
	}
	return solicitud, nil
}

func (s solicitudEjecucionDocumentalAtestadaV4) validar() error {
	if s.prueba.validar() != nil {
		return errorAutoridadInternaEjecucionDocumentalV4()
	}
	p := s.prueba.datos.ProyeccionAplicacion
	if s.orden.Esquema != esquemaOrdenGeneracionDocumentalV4 ||
		s.orden.Estado != estadoOrdenGeneracionPendienteV4 ||
		s.orden.OrdenRef != p.Clave.EfectoRef || s.orden.EfectoRef != p.Clave.EfectoRef ||
		s.orden.DecisionRef != p.Clave.DecisionRef ||
		s.orden.HuellaPlanSHA256 != p.Clave.HuellaPlanSHA256 ||
		s.orden.HuellaDecision != p.HuellaDecisionSHA256 ||
		s.orden.HuellaAplicacion != p.HuellaSolicitudAplicacionSHA256 ||
		s.orden.CorrelacionRef != p.CorrelacionRef ||
		!instanteCanonicoDocumental(s.orden.SolicitadaEn) ||
		!s.orden.SolicitadaEn.Equal(p.SolicitadaEn) ||
		!huellaSHA256DocumentalValida(s.orden.HuellaOrdenSHA256) ||
		!referenciaDurableAtestacionPDPValida(s.orden.AuditoriaRef) ||
		!referenciaDurableAtestacionPDPValida(s.orden.EventoOutboxRef) ||
		s.orden.AuditoriaRef == s.orden.EventoOutboxRef {
		return errorAutoridadInternaEjecucionDocumentalV4()
	}
	return nil
}

func clonarDatosRegistroAtestacionPDPDocumentalV4(
	d datosRegistroAtestacionPDPDocumentalV4,
) datosRegistroAtestacionPDPDocumentalV4 {
	d.PayloadVECAD1 = append([]byte(nil), d.PayloadVECAD1...)
	d.SobreCOSESign1 = append([]byte(nil), d.SobreCOSESign1...)
	d.EvidenciaCanonica = append([]byte(nil), d.EvidenciaCanonica...)
	d.DecisionCanonica = append([]byte(nil), d.DecisionCanonica...)
	return d
}

func (pruebaRegistroAtestacionPDPDocumentalV4) String() string {
	return "[PRUEBA-REGISTRO-ATESTACION-PDP-DOCUMENTAL-V4-PRIVADA]"
}
func (p pruebaRegistroAtestacionPDPDocumentalV4) GoString() string { return p.String() }
func (p pruebaRegistroAtestacionPDPDocumentalV4) LogValue() slog.Value {
	return slog.StringValue(p.String())
}
func (p pruebaRegistroAtestacionPDPDocumentalV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}

func denegarEjecucionDocumentalV4(causa error) error {
	if errors.Is(causa, context.Canceled) || errors.Is(causa, context.DeadlineExceeded) {
		return errors.Join(errorAutoridadInternaEjecucionDocumentalV4(), causa)
	}
	return errorAutoridadInternaEjecucionDocumentalV4()
}
