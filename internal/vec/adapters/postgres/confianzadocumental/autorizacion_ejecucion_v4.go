package confianzadocumental

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var (
	// ErrAutoridadInternaEjecucionDocumentalV4Invalida oculta la causa concreta
	// y conserva la politica de denegacion por defecto ante cualquier ausencia,
	// discrepancia, caducidad, ambiguedad o manipulacion.
	ErrAutoridadInternaEjecucionDocumentalV4Invalida = errors.New("vec: autoridad interna de ejecucion documental v4 invalida")
	// ErrSerializacionAutoridadEjecucionDocumentalV4Prohibida impide convertir
	// una autoridad local en una credencial transportable o persistida.
	ErrSerializacionAutoridadEjecucionDocumentalV4Prohibida = errors.New("vec: serializacion de autoridad interna de ejecucion documental v4 prohibida")
)

const marcaAutoridadInternaEjecucionDocumentalV4 = "vec.documentos.autoridad-interna-ejecucion.v4"

// AutoridadInternaEjecucionDocumentalV4 es la autoridad opaca emitida dentro
// del perimetro compilable de application. Liga exactamente DecisionRef, plan,
// efecto y la solicitud estructural que comprobo actor, accion, recurso,
// finalidad, ambitos, tiempo, campos y obligaciones.
//
// Este tipo reduce la superficie capaz de emitir autoridad y conserva la
// prueba criptografica PDP verificada por Servicio sobre el mensaje exacto de
// domain.SerializarMensajeAtestacionAutorizacionV1. Esta autoridad local no es
// la credencial de la ruta productiva: esa ruta exige la capacidad efimera del
// emisor aislado y su consumo durable UNIQUE(DecisionRef)+efecto en el mismo
// COMMIT PostgreSQL. Una evidencia estructural o
// ports.AtestacionAutorizacionV1 nunca sustituyen esa composicion.
type AutoridadInternaEjecucionDocumentalV4 struct {
	marca                 string
	vinculo               ports.SolicitudVinculadaAutorizacionEjecucionDocumentalV4
	pruebaPDP             PruebaCOSESign1DocumentalVerificada
	cabeceraPDP           domain.CabeceraAtestacionAutorizacionV1
	evidenciaDurablePDP   EvidenciaDurableAtestacionAutorizacionPDPV4
	clave                 ports.ClaveAplicacionAutorizacionEjecucionDocumentalV4
	huellaVinculoSHA256   string
	huellaAutoridadSHA256 string
	emitidaEn             time.Time
}

// emitirAutoridadInternaEjecucionDocumentalV4 permanece privada y exige una
// comprobacion que contenga la prueba COSE PDP opaca. Ya no existe una via de
// emision basada exclusivamente en EvidenciaUsoDecisionAutorizacion o en DTO.
func emitirAutoridadInternaEjecucionDocumentalV4(
	vinculo ports.SolicitudVinculadaAutorizacionEjecucionDocumentalV4,
	comprobacion comprobacionAtestacionAutorizacionPDPV4,
	emitidaEn time.Time,
) (AutoridadInternaEjecucionDocumentalV4, error) {
	if !instanteCanonicoDocumental(emitidaEn) ||
		comprobacion.validarPara(vinculo, emitidaEn) != nil ||
		vinculo.ValidarEn(emitidaEn) != nil {
		return AutoridadInternaEjecucionDocumentalV4{}, errorAutoridadInternaEjecucionDocumentalV4()
	}
	huellaVinculo, err := vinculo.HuellaSHA256()
	if err != nil || !huellaSHA256DocumentalValida(huellaVinculo) {
		return AutoridadInternaEjecucionDocumentalV4{}, errorAutoridadInternaEjecucionDocumentalV4()
	}

	// La proyeccion se usa solo como segunda comprobacion defensiva de la terna
	// exacta; no se convierte en autoridad ni se conserva como credencial.
	solicitud, err := vinculo.PrepararSolicitudAplicacionEn(emitidaEn)
	if err != nil {
		return AutoridadInternaEjecucionDocumentalV4{}, errorAutoridadInternaEjecucionDocumentalV4()
	}
	evidencia, err := solicitud.EvidenciaEstructural()
	if err != nil {
		return AutoridadInternaEjecucionDocumentalV4{}, errorAutoridadInternaEjecucionDocumentalV4()
	}
	datosEvidencia, err := evidencia.Datos()
	if err != nil || evidencia.ValidarEn(emitidaEn) != nil {
		return AutoridadInternaEjecucionDocumentalV4{}, errorAutoridadInternaEjecucionDocumentalV4()
	}
	proyeccion, err := solicitud.ProyeccionParaTransaccion()
	clave := proyeccion.Clave
	if err != nil ||
		proyeccion.HuellaSolicitudVinculadaSHA256 != huellaVinculo ||
		proyeccion.Clave.DecisionRef != datosEvidencia.Decision.DecisionRef ||
		proyeccion.PerfilActivoRef != datosEvidencia.Decision.PerfilActivoRef ||
		proyeccion.Accion != ports.AccionEjecutarPlanDocumentalV4 ||
		proyeccion.RecursoRef != datosEvidencia.Decision.RecursoRef ||
		proyeccion.ModuloID != datosEvidencia.Decision.ModuloID ||
		proyeccion.TipoRecurso != datosEvidencia.Decision.TipoRecurso ||
		proyeccion.HuellaRecursoSHA256 != datosEvidencia.Decision.ContextoRecursoHuellaSHA256 ||
		proyeccion.Finalidad != datosEvidencia.Decision.Finalidad ||
		proyeccion.CorrelacionRef != datosEvidencia.Decision.CorrelacionRef ||
		!proyeccion.VerificadaEn.Equal(datosEvidencia.VerificadaEn) ||
		!proyeccion.VinculadaEn.Equal(emitidaEn) {
		return AutoridadInternaEjecucionDocumentalV4{}, errorAutoridadInternaEjecucionDocumentalV4()
	}
	preimagenRecurso, err := solicitud.PreimagenRecursoParaEvidenciaDurable()
	if err != nil {
		return AutoridadInternaEjecucionDocumentalV4{}, errorAutoridadInternaEjecucionDocumentalV4()
	}
	evidenciaDurable, err := nuevaEvidenciaDurableAtestacionAutorizacionPDPV4(
		clave, huellaVinculo, preimagenRecurso, comprobacion, emitidaEn,
	)
	if err != nil {
		return AutoridadInternaEjecucionDocumentalV4{}, errorAutoridadInternaEjecucionDocumentalV4()
	}

	autoridad := AutoridadInternaEjecucionDocumentalV4{
		marca: marcaAutoridadInternaEjecucionDocumentalV4, vinculo: vinculo,
		pruebaPDP:           comprobacion.prueba,
		cabeceraPDP:         comprobacion.cabecera,
		evidenciaDurablePDP: evidenciaDurable,
		clave:               clave, huellaVinculoSHA256: huellaVinculo, emitidaEn: emitidaEn,
	}
	autoridad.huellaAutoridadSHA256 = autoridad.calcularHuella()
	if autoridad.validarEstructura() != nil {
		return AutoridadInternaEjecucionDocumentalV4{}, errorAutoridadInternaEjecucionDocumentalV4()
	}
	return autoridad, nil
}

// ValidarEn revalida la autoridad, la evidencia estructural y su ventana. El
// limite superior es exclusivo y un reloj no canonico siempre deniega.
func (a AutoridadInternaEjecucionDocumentalV4) ValidarEn(instante time.Time) error {
	if a.validarEstructura() != nil || !instanteCanonicoDocumental(instante) ||
		instante.Before(a.emitidaEn) ||
		!instante.Before(a.pruebaPDP.raizValidaHasta) ||
		!instante.Before(a.pruebaPDP.configuracionExpiraEn) ||
		a.vinculo.ValidarEn(instante) != nil {
		return errorAutoridadInternaEjecucionDocumentalV4()
	}
	return nil
}

// PrepararAplicacionExactaEn produce una solicitud no autoritativa ligada a
// la terna que el caso de uso pretende aplicar. La ruta V4 productiva no
// consume esta proyeccion local: revalida el COSE en el emisor aislado y exige
// UNIQUE(DecisionRef)+efecto en el mismo COMMIT PostgreSQL. Prepararla no
// consume nada.
func (a AutoridadInternaEjecucionDocumentalV4) PrepararAplicacionExactaEn(
	decisionRef, huellaPlanSHA256, efectoRef string,
	instante time.Time,
) (ports.SolicitudAplicacionAutorizacionEjecucionDocumentalV4, error) {
	solicitud, _, err := a.PrepararAplicacionExactaConEvidenciaEn(
		decisionRef, huellaPlanSHA256, efectoRef, instante,
	)
	return solicitud, err
}

// PrepararAplicacionExactaConEvidenciaEn entrega conjuntamente la solicitud
// no autoritativa y la prueba durable que el repositorio debera confirmar en
// el mismo COMMIT. Entregarla no consume la decision ni acredita persistencia.
func (a AutoridadInternaEjecucionDocumentalV4) PrepararAplicacionExactaConEvidenciaEn(
	decisionRef, huellaPlanSHA256, efectoRef string,
	instante time.Time,
) (
	ports.SolicitudAplicacionAutorizacionEjecucionDocumentalV4,
	EvidenciaDurableAtestacionAutorizacionPDPV4,
	error,
) {
	if a.ValidarEn(instante) != nil ||
		a.clave.DecisionRef != decisionRef ||
		a.clave.HuellaPlanSHA256 != huellaPlanSHA256 ||
		a.clave.EfectoRef != efectoRef {
		return ports.SolicitudAplicacionAutorizacionEjecucionDocumentalV4{},
			EvidenciaDurableAtestacionAutorizacionPDPV4{},
			errorAutoridadInternaEjecucionDocumentalV4()
	}
	solicitud, err := a.vinculo.PrepararSolicitudAplicacionEn(instante)
	if err != nil || solicitud.ValidarContraEn(
		decisionRef,
		huellaPlanSHA256,
		efectoRef,
		instante,
	) != nil {
		return ports.SolicitudAplicacionAutorizacionEjecucionDocumentalV4{},
			EvidenciaDurableAtestacionAutorizacionPDPV4{},
			errorAutoridadInternaEjecucionDocumentalV4()
	}
	evidencia, err := a.evidenciaDurablePDP.clonar()
	if err != nil || !evidencia.coincideConAutoridad(a) {
		return ports.SolicitudAplicacionAutorizacionEjecucionDocumentalV4{},
			EvidenciaDurableAtestacionAutorizacionPDPV4{},
			errorAutoridadInternaEjecucionDocumentalV4()
	}
	return solicitud, evidencia, nil
}

func (a AutoridadInternaEjecucionDocumentalV4) validarEstructura() error {
	if a.marca != marcaAutoridadInternaEjecucionDocumentalV4 ||
		a.clave.DecisionRef == "" || a.clave.HuellaPlanSHA256 == "" || a.clave.EfectoRef == "" ||
		!huellaSHA256DocumentalValida(a.huellaVinculoSHA256) ||
		!huellaSHA256DocumentalValida(a.huellaAutoridadSHA256) ||
		!instanteCanonicoDocumental(a.emitidaEn) || a.vinculo.ValidarEn(a.emitidaEn) != nil ||
		verificarPruebaAtestacionPDPContraVinculo(
			a.pruebaPDP, a.cabeceraPDP, a.vinculo, a.emitidaEn,
		) != nil || a.evidenciaDurablePDP.Validar() != nil ||
		!a.evidenciaDurablePDP.coincideConAutoridad(a) {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	huellaVinculo, err := a.vinculo.HuellaSHA256()
	if err != nil || huellaVinculo != a.huellaVinculoSHA256 ||
		a.calcularHuella() != a.huellaAutoridadSHA256 {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	solicitud, err := a.vinculo.PrepararSolicitudAplicacionEn(a.emitidaEn)
	if err != nil || solicitud.ValidarContraEn(
		a.clave.DecisionRef,
		a.clave.HuellaPlanSHA256,
		a.clave.EfectoRef,
		a.emitidaEn,
	) != nil {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	proyeccion, err := solicitud.ProyeccionParaTransaccion()
	if err != nil || proyeccion.Clave != a.clave ||
		proyeccion.HuellaSolicitudVinculadaSHA256 != a.huellaVinculoSHA256 {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	return nil
}

func (a AutoridadInternaEjecucionDocumentalV4) calcularHuella() string {
	campos := []string{
		marcaAutoridadInternaEjecucionDocumentalV4,
		a.huellaVinculoSHA256,
		a.clave.DecisionRef,
		a.clave.HuellaPlanSHA256,
		a.clave.EfectoRef,
		a.emitidaEn.Format(time.RFC3339Nano),
		strconv.FormatUint(uint64(a.cabeceraPDP.FormatoVersion), 10),
		a.cabeceraPDP.Suite,
		a.cabeceraPDP.ClaveID,
		a.cabeceraPDP.Audiencia,
		a.evidenciaDurablePDP.metadatos.HuellaEvidenciaDurableSHA256,
		string(a.pruebaPDP.algoritmo),
		fmt.Sprintf("%x", a.pruebaPDP.claveID),
		a.pruebaPDP.huellaClaveSHA256,
		string(a.pruebaPDP.estadoConfianza),
		string(a.pruebaPDP.audiencia),
		a.pruebaPDP.huellaPayloadSHA256,
		a.pruebaPDP.huellaSobreSHA256,
		a.pruebaPDP.verificadaEn.Format(time.RFC3339Nano),
		a.pruebaPDP.raizValidaDesde.Format(time.RFC3339Nano),
		a.pruebaPDP.raizValidaHasta.Format(time.RFC3339Nano),
		a.pruebaPDP.revisionConfianza,
		a.pruebaPDP.huellaConfiguracionSHA256,
		a.pruebaPDP.configuracionPublicadaEn.Format(time.RFC3339Nano),
		a.pruebaPDP.configuracionExpiraEn.Format(time.RFC3339Nano),
	}
	var canonico strings.Builder
	for _, campo := range campos {
		canonico.WriteString(strconv.Itoa(len(campo)))
		canonico.WriteByte(':')
		canonico.WriteString(campo)
	}
	return huellaBytesDocumentales([]byte(canonico.String()))
}

func errorAutoridadInternaEjecucionDocumentalV4() error {
	return errors.Join(
		domain.ErrAutorizacionDenegada,
		ErrAutoridadInternaEjecucionDocumentalV4Invalida,
	)
}

func (AutoridadInternaEjecucionDocumentalV4) String() string {
	return "[AUTORIDAD-INTERNA-EJECUCION-DOCUMENTAL-V4-REDACTADA]"
}

func (a AutoridadInternaEjecucionDocumentalV4) GoString() string { return a.String() }
func (a AutoridadInternaEjecucionDocumentalV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, a.String())
}
func (a AutoridadInternaEjecucionDocumentalV4) LogValue() slog.Value {
	return slog.StringValue(a.String())
}
func (AutoridadInternaEjecucionDocumentalV4) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAutoridadEjecucionDocumentalV4Prohibida
}
func (*AutoridadInternaEjecucionDocumentalV4) UnmarshalJSON([]byte) error {
	return ErrSerializacionAutoridadEjecucionDocumentalV4Prohibida
}
func (AutoridadInternaEjecucionDocumentalV4) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAutoridadEjecucionDocumentalV4Prohibida
}
func (*AutoridadInternaEjecucionDocumentalV4) UnmarshalText([]byte) error {
	return ErrSerializacionAutoridadEjecucionDocumentalV4Prohibida
}
func (AutoridadInternaEjecucionDocumentalV4) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionAutoridadEjecucionDocumentalV4Prohibida
}
func (*AutoridadInternaEjecucionDocumentalV4) UnmarshalBinary([]byte) error {
	return ErrSerializacionAutoridadEjecucionDocumentalV4Prohibida
}
