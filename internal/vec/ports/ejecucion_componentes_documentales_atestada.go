package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	documentalcanonico "vec-diputacion-granada/internal/vec/canonico/documental"
	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrCompromisoEjecucionDocumentalInvalido  = errors.New("vec: compromiso de ejecucion documental invalido")
	ErrSobreReciboEjecucionDocumentalInvalido = errors.New("vec: sobre de recibo de ejecucion documental invalido")
	ErrIdentidadEjecucionDocumentalInvalida   = errors.New("vec: identidad de ejecucion documental invalida")
	ErrReciboEjecucionDocumentalInvalido      = errors.New("vec: recibo de ejecucion documental invalido")
)

const (
	maximoBytesEjecucionComponenteDocumental uint64 = 256 * 1024 * 1024
	maximoBytesSobreCOSEDocumental                  = 64 * 1024
	minimoBytesSobreCOSEDocumental                  = 16
	maximaVigenciaCompromisoDocumental              = 10 * time.Minute
)

// OperacionComponenteDocumental es vocabulario cerrado del protocolo entre el
// nucleo y el despachador. Cada operacion tiene un unico rol admisible.
type OperacionComponenteDocumental string

const (
	OperacionRenderizadoDocumental           OperacionComponenteDocumental = "renderizado"
	OperacionValidacionEstructuralDocumental OperacionComponenteDocumental = "validacion_estructural"
	OperacionVerificacionSemanticaDocumental OperacionComponenteDocumental = "verificacion_semantica"
)

func (o OperacionComponenteDocumental) Valida() bool {
	_, valida := o.rolEsperado()
	return valida
}

func (o OperacionComponenteDocumental) rolEsperado() (domain.RolComponenteDocumental, bool) {
	switch o {
	case OperacionRenderizadoDocumental:
		return domain.RolComponenteRenderizador, true
	case OperacionValidacionEstructuralDocumental:
		return domain.RolComponenteValidadorEstructural, true
	case OperacionVerificacionSemanticaDocumental:
		return domain.RolComponenteVerificadorSemantico, true
	default:
		return "", false
	}
}

// CompromisoEjecucionComponenteDocumental liga una invocacion irrepetible con
// el perfil, su publicacion/revision y el componente exacto. En renderizado la
// huella y el tamano de salida todavia son desconocidos y deben ser cero; en
// las verificaciones son obligatorios.
type CompromisoEjecucionComponenteDocumental struct {
	operacionRef               string
	reto                       [32]byte
	operacion                  OperacionComponenteDocumental
	descriptorPerfil           DescriptorPerfilDocumental
	situacionOperativa         domain.SituacionOperativaPerfilDocumental
	descriptorComponente       DescriptorComponenteDocumentalAtestado
	ordenDespachoConsumida     OrdenDespachoDocumentalV3ConsumidaNominal
	vinculoActivacion          VinculoEstableActivacionDocumentalV3
	reservaRef                 string
	manifiesto                 ManifiestoEjecucionDocumentalV3
	consumoDecision            ConsumoDecisionEjecucionDocumentalV3
	efectoRef                  string
	huellaPlanSHA256           string
	secuenciaCercado           uint64
	huellaVinculoSHA256        string
	inicioEfectoRef            string
	outboxInicioRef            string
	reclamacionDespachoRef     string
	consumoDespachoRef         string
	outboxConsumoRef           string
	versionInicioCAS           uint64
	versionReclamacionCAS      uint64
	versionConsumoCAS          uint64
	huellaOrdenDespachoSHA256  string
	comprobacionKMSRef         string
	borradorRef                string
	huellaContenidoNeutralHMAC string
	huellaDocumentoSHA256      string
	tamanoDocumento            uint64
	limiteBytes                uint64
	emitidoEn                  time.Time
	expiraEn                   time.Time
	huellaCompromisoSHA256     string
}

// NuevoCompromisoEjecucionComponenteDocumental acepta el comando nominal que
// el servicio de aplicacion obtuvo tras verificar por KMS y consumir por CAS.
// El valor no es autoritativo ni sustituye la composicion: handlers no deben
// poseer este constructor junto con el despachador.
func NuevoCompromisoEjecucionComponenteDocumental(
	operacionRef string,
	reto [32]byte,
	operacion OperacionComponenteDocumental,
	descriptorPerfil DescriptorPerfilDocumental,
	situacionOperativa domain.SituacionOperativaPerfilDocumental,
	descriptorComponente DescriptorComponenteDocumentalAtestado,
	ordenConsumida OrdenDespachoDocumentalV3ConsumidaNominal,
	borradorRef, huellaContenidoNeutralHMAC, huellaDocumentoSHA256 string,
	tamanoDocumento, limiteBytes uint64,
	vigencia time.Duration,
) (CompromisoEjecucionComponenteDocumental, error) {
	vinculoActivacion, errVinculo := ordenConsumida.VinculoActivacion()
	datosOrden, errOrden := ordenConsumida.DatosOrden()
	huellaOrden, errHuellaOrden := ordenConsumida.solicitud.orden.HuellaSHA256()
	if errVinculo != nil || errOrden != nil || errHuellaOrden != nil ||
		ordenConsumida.ValidarEn(ordenConsumida.estado.consumidaEn) != nil {
		return CompromisoEjecucionComponenteDocumental{}, ErrCompromisoEjecucionDocumentalInvalido
	}
	reservaRef := vinculoActivacion.ReservaRef
	manifiesto := vinculoActivacion.Manifiesto
	consumoDecision := vinculoActivacion.ConsumoDecision
	datosManifiesto, err := manifiesto.Datos()
	if err != nil {
		return CompromisoEjecucionComponenteDocumental{}, ErrCompromisoEjecucionDocumentalInvalido
	}
	emitidoEn := ordenConsumida.estado.consumidaEn
	expiraEn := emitidoEn.Add(vigencia)
	if expiraEn.After(datosOrden.ExpiraEn) {
		return CompromisoEjecucionComponenteDocumental{}, ErrCompromisoEjecucionDocumentalInvalido
	}
	compromiso := CompromisoEjecucionComponenteDocumental{
		operacionRef: operacionRef, reto: reto, operacion: operacion,
		descriptorPerfil: descriptorPerfil, situacionOperativa: situacionOperativa,
		descriptorComponente:   descriptorComponente,
		ordenDespachoConsumida: ordenConsumida,
		vinculoActivacion:      vinculoActivacion,
		reservaRef:             reservaRef, manifiesto: manifiesto, consumoDecision: consumoDecision,
		efectoRef:                 datosManifiesto.EfectoRef,
		huellaPlanSHA256:          datosManifiesto.HuellaPlanSHA256,
		secuenciaCercado:          datosOrden.ReciboInicio.SecuenciaCercado,
		huellaVinculoSHA256:       datosOrden.ReciboInicio.HuellaVinculoCercadoSHA256,
		inicioEfectoRef:           datosOrden.ReciboInicio.InicioRef,
		outboxInicioRef:           datosOrden.ReciboInicio.OutboxInicioRef,
		reclamacionDespachoRef:    datosOrden.ReclamacionRef,
		consumoDespachoRef:        ordenConsumida.estado.consumoRef,
		outboxConsumoRef:          ordenConsumida.estado.outboxConsumoRef,
		versionInicioCAS:          datosOrden.ReciboInicio.VersionInicioCAS,
		versionReclamacionCAS:     datosOrden.VersionReclamacionCAS,
		versionConsumoCAS:         ordenConsumida.estado.versionConsumoCAS,
		huellaOrdenDespachoSHA256: huellaOrden,
		comprobacionKMSRef:        ordenConsumida.resultado.comprobacionRef,
		borradorRef:               borradorRef, huellaContenidoNeutralHMAC: huellaContenidoNeutralHMAC,
		huellaDocumentoSHA256: huellaDocumentoSHA256, tamanoDocumento: tamanoDocumento,
		limiteBytes: limiteBytes, emitidoEn: emitidoEn, expiraEn: expiraEn,
	}
	compromiso.huellaCompromisoSHA256 = compromiso.calcularHuella()
	if compromiso.Validar() != nil {
		return CompromisoEjecucionComponenteDocumental{}, ErrCompromisoEjecucionDocumentalInvalido
	}
	return compromiso, nil
}

func (c CompromisoEjecucionComponenteDocumental) Validar() error {
	rolEsperado, operacionValida := c.operacion.rolEsperado()
	consulta := c.descriptorComponente.Consulta()
	perfil := c.descriptorPerfil.Perfil()
	datosManifiesto, errManifiesto := c.manifiesto.Datos()
	datosOrden, errOrden := c.ordenDespachoConsumida.DatosOrden()
	huellaOrden, errHuellaOrden := c.ordenDespachoConsumida.solicitud.orden.HuellaSHA256()
	if !operacionValida || !referenciaOpacaEjecucionDocumentalSegura(c.operacionRef) ||
		!referenciaOpacaEjecucionDocumentalSegura(c.borradorRef) || c.operacionRef == c.borradorRef ||
		retoEjecucionDocumentalNulo(c.reto) || c.descriptorPerfil.Validar() != nil ||
		c.situacionOperativa.Validar() != nil ||
		c.situacionOperativa.PublicacionRef() != c.descriptorPerfil.PublicacionRef() ||
		!c.situacionOperativa.AutorizaEjecucion(perfil, c.descriptorPerfil.Revision()) ||
		c.descriptorComponente.Validar() != nil ||
		c.descriptorComponente.Componente().Rol() != rolEsperado ||
		!descriptorEjecucionDocumentalSinPIIEvidente(c.descriptorPerfil, c.descriptorComponente) ||
		consulta.DescriptorPerfilRef != c.descriptorPerfil.Referencia() ||
		consulta.PublicacionRef != c.descriptorPerfil.PublicacionRef() ||
		consulta.PerfilRef != perfil.Referencia() || consulta.DigestPerfil != perfil.DigestSHA256() ||
		consulta.RevisionCatalogo != c.descriptorPerfil.Revision() ||
		errManifiesto != nil || c.vinculoActivacion.Validar() != nil ||
		c.vinculoActivacion.ReservaRef != c.reservaRef ||
		!manifiestosEjecucionDocumentalV3Coinciden(c.vinculoActivacion.Manifiesto, c.manifiesto) ||
		c.vinculoActivacion.ConsumoDecision != c.consumoDecision || errOrden != nil ||
		errHuellaOrden != nil || c.ordenDespachoConsumida.ValidarEn(c.emitidoEn) != nil ||
		datosOrden.ReciboInicio.ReservaRef != c.reservaRef ||
		datosOrden.ReciboInicio.SecuenciaCercado != c.secuenciaCercado ||
		datosOrden.ReciboInicio.HuellaVinculoCercadoSHA256 != c.huellaVinculoSHA256 ||
		datosOrden.ReciboInicio.InicioRef != c.inicioEfectoRef ||
		datosOrden.ReciboInicio.OutboxInicioRef != c.outboxInicioRef ||
		datosOrden.ReclamacionRef != c.reclamacionDespachoRef ||
		c.ordenDespachoConsumida.estado.consumoRef != c.consumoDespachoRef ||
		c.ordenDespachoConsumida.estado.outboxConsumoRef != c.outboxConsumoRef ||
		datosOrden.ReciboInicio.VersionInicioCAS != c.versionInicioCAS ||
		datosOrden.VersionReclamacionCAS != c.versionReclamacionCAS ||
		c.ordenDespachoConsumida.estado.versionConsumoCAS != c.versionConsumoCAS ||
		c.versionConsumoCAS <= c.versionReclamacionCAS ||
		huellaOrden != c.huellaOrdenDespachoSHA256 ||
		c.ordenDespachoConsumida.resultado.comprobacionRef != c.comprobacionKMSRef ||
		!c.ordenDespachoConsumida.estado.consumidaEn.Equal(c.emitidoEn) ||
		!referenciaOpacaEjecucionDocumentalSegura(c.reservaRef) ||
		!referenciaOpacaEjecucionDocumentalSegura(c.efectoRef) ||
		c.reservaRef == c.efectoRef || c.reservaRef == c.borradorRef ||
		c.reservaRef == c.operacionRef || c.efectoRef == c.borradorRef ||
		c.efectoRef == c.operacionRef || datosManifiesto.DescriptorPerfil != c.descriptorPerfil ||
		datosManifiesto.SituacionOperativa != c.situacionOperativa ||
		datosManifiesto.BorradorRef != c.borradorRef ||
		datosManifiesto.EfectoRef != c.efectoRef ||
		datosManifiesto.HuellaEntradaHMAC != c.huellaContenidoNeutralHMAC ||
		datosManifiesto.LimiteEfectivoBytes != c.limiteBytes ||
		datosManifiesto.HuellaPlanSHA256 != c.huellaPlanSHA256 ||
		c.consumoDecision.EfectoRef != c.efectoRef ||
		c.consumoDecision.HuellaPlanSHA256 != c.huellaPlanSHA256 ||
		!huellaSHA256FormatoDocumentalValida(c.huellaPlanSHA256) ||
		c.secuenciaCercado == 0 ||
		!huellaSHA256FormatoDocumentalValida(c.huellaVinculoSHA256) ||
		!componenteEjecucionDocumentalPerteneceAlPlan(
			c.operacion, c.descriptorComponente, datosManifiesto,
		) ||
		!huellaHMACEjecucionDocumentalValida(c.huellaContenidoNeutralHMAC) ||
		c.limiteBytes == 0 || c.limiteBytes > maximoBytesEjecucionComponenteDocumental ||
		c.limiteBytes > perfil.MaximoBytes() ||
		c.limiteBytes > c.descriptorComponente.MaximoBytes() ||
		!ventanaCompromisoEjecucionDocumentalValida(c.emitidoEn, c.expiraEn) ||
		c.expiraEn.After(datosOrden.ExpiraEn) ||
		!huellaSHA256FormatoDocumentalValida(c.huellaCompromisoSHA256) ||
		c.huellaCompromisoSHA256 != c.calcularHuella() ||
		c.operacionRef == c.descriptorPerfil.Referencia() ||
		c.operacionRef == c.descriptorPerfil.PublicacionRef() ||
		c.borradorRef == c.descriptorPerfil.Referencia() ||
		c.borradorRef == c.descriptorPerfil.PublicacionRef() {
		return ErrCompromisoEjecucionDocumentalInvalido
	}
	if c.operacion == OperacionRenderizadoDocumental {
		if c.huellaDocumentoSHA256 != "" || c.tamanoDocumento != 0 {
			return ErrCompromisoEjecucionDocumentalInvalido
		}
		return nil
	}
	if !huellaSHA256FormatoDocumentalValida(c.huellaDocumentoSHA256) ||
		c.tamanoDocumento == 0 || c.tamanoDocumento > c.limiteBytes {
		return ErrCompromisoEjecucionDocumentalInvalido
	}
	return nil
}

func (c CompromisoEjecucionComponenteDocumental) OperacionRef() string {
	return c.operacionRef
}
func (c CompromisoEjecucionComponenteDocumental) Reto() [32]byte { return c.reto }
func (c CompromisoEjecucionComponenteDocumental) Operacion() OperacionComponenteDocumental {
	return c.operacion
}
func (c CompromisoEjecucionComponenteDocumental) DescriptorPerfil() DescriptorPerfilDocumental {
	return c.descriptorPerfil
}
func (c CompromisoEjecucionComponenteDocumental) SituacionOperativa() domain.SituacionOperativaPerfilDocumental {
	return c.situacionOperativa
}
func (c CompromisoEjecucionComponenteDocumental) DescriptorComponente() DescriptorComponenteDocumentalAtestado {
	return c.descriptorComponente
}
func (c CompromisoEjecucionComponenteDocumental) ReservaRef() string { return c.reservaRef }
func (c CompromisoEjecucionComponenteDocumental) VinculoActivacion() VinculoEstableActivacionDocumentalV3 {
	return c.vinculoActivacion
}
func (c CompromisoEjecucionComponenteDocumental) EfectoRef() string { return c.efectoRef }
func (c CompromisoEjecucionComponenteDocumental) HuellaPlanSHA256() string {
	return c.huellaPlanSHA256
}
func (c CompromisoEjecucionComponenteDocumental) SecuenciaCercado() uint64 {
	return c.secuenciaCercado
}
func (c CompromisoEjecucionComponenteDocumental) HuellaVinculoCercadoSHA256() string {
	return c.huellaVinculoSHA256
}
func (c CompromisoEjecucionComponenteDocumental) ManifiestoCercado() ManifiestoEjecucionDocumentalV3 {
	return c.manifiesto
}
func (c CompromisoEjecucionComponenteDocumental) ConsumoDecisionCercado() ConsumoDecisionEjecucionDocumentalV3 {
	return c.consumoDecision
}
func (c CompromisoEjecucionComponenteDocumental) BorradorRef() string { return c.borradorRef }
func (c CompromisoEjecucionComponenteDocumental) HuellaContenidoNeutralHMAC() string {
	return c.huellaContenidoNeutralHMAC
}
func (c CompromisoEjecucionComponenteDocumental) HuellaDocumentoSHA256() string {
	return c.huellaDocumentoSHA256
}
func (c CompromisoEjecucionComponenteDocumental) TamanoDocumento() uint64 {
	return c.tamanoDocumento
}
func (c CompromisoEjecucionComponenteDocumental) LimiteBytes() uint64  { return c.limiteBytes }
func (c CompromisoEjecucionComponenteDocumental) EmitidoEn() time.Time { return c.emitidoEn }
func (c CompromisoEjecucionComponenteDocumental) ExpiraEn() time.Time  { return c.expiraEn }
func (c CompromisoEjecucionComponenteDocumental) VigenteEn(instante time.Time) bool {
	return c.Validar() == nil && !instante.IsZero() && instante.Location() == time.UTC &&
		!instante.Before(c.emitidoEn) && instante.Before(c.expiraEn)
}
func (c CompromisoEjecucionComponenteDocumental) HuellaSHA256() (string, error) {
	if c.Validar() != nil {
		return "", ErrCompromisoEjecucionDocumentalInvalido
	}
	return c.huellaCompromisoSHA256, nil
}

func (CompromisoEjecucionComponenteDocumental) String() string {
	return "[COMPROMISO-EJECUCION-DOCUMENTAL-NOMINAL-NO-AUTORITATIVO-REDACTADO]"
}

func (c CompromisoEjecucionComponenteDocumental) GoString() string { return c.String() }

func (c CompromisoEjecucionComponenteDocumental) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}

func (c CompromisoEjecucionComponenteDocumental) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

func (CompromisoEjecucionComponenteDocumental) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}

func (*CompromisoEjecucionComponenteDocumental) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

func (CompromisoEjecucionComponenteDocumental) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}

func (*CompromisoEjecucionComponenteDocumental) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

func (CompromisoEjecucionComponenteDocumental) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}

func (*CompromisoEjecucionComponenteDocumental) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

func (c CompromisoEjecucionComponenteDocumental) calcularHuella() string {
	perfil := c.descriptorPerfil.Perfil()
	revision := c.descriptorPerfil.Revision()
	situacion := c.situacionOperativa
	componente := c.descriptorComponente.Componente()
	huellaVinculoEstable, _ := c.vinculoActivacion.HuellaSHA256()
	return huellaCanonicaFormatoDocumental([]string{
		"vec.compromiso-ejecucion-componente-documental.v1", c.operacionRef,
		hex.EncodeToString(c.reto[:]), string(c.operacion), c.descriptorPerfil.Referencia(),
		c.descriptorPerfil.PublicacionRef(), perfil.Identidad().Identificador(),
		perfil.Referencia().Identificador(), strconv.FormatUint(perfil.Referencia().Version(), 10),
		perfil.DigestSHA256(), strconv.FormatUint(revision.Numero(), 10), revision.HuellaSHA256(),
		situacion.PublicacionRef(), strconv.FormatUint(situacion.RevisionOperativa(), 10),
		string(situacion.Estado()), situacion.HuellaSHA256(),
		c.descriptorComponente.Referencia(), c.descriptorComponente.DigestDeclaracionSHA256(),
		string(componente.Rol()), componente.Identificador(),
		strconv.FormatUint(componente.Version(), 10), componente.HuellaArtefactoSHA256(),
		c.reservaRef, c.efectoRef, c.huellaPlanSHA256, huellaVinculoEstable,
		strconv.FormatUint(c.secuenciaCercado, 10), c.huellaVinculoSHA256,
		c.consumoDecision.DecisionRef, c.consumoDecision.EsquemaHuellaDecision,
		c.consumoDecision.HuellaDecisionSHA256, c.inicioEfectoRef, c.outboxInicioRef,
		c.reclamacionDespachoRef, c.consumoDespachoRef, c.outboxConsumoRef,
		strconv.FormatUint(c.versionInicioCAS, 10),
		strconv.FormatUint(c.versionReclamacionCAS, 10),
		strconv.FormatUint(c.versionConsumoCAS, 10), c.huellaOrdenDespachoSHA256,
		c.comprobacionKMSRef,
		c.borradorRef, c.huellaContenidoNeutralHMAC, c.huellaDocumentoSHA256,
		strconv.FormatUint(c.tamanoDocumento, 10), strconv.FormatUint(c.limiteBytes, 10),
		c.emitidoEn.Format(time.RFC3339Nano), c.expiraEn.Format(time.RFC3339Nano),
	})
}

// SobreReciboEjecucionDocumentalCrudo conserva un COSE_Sign1 opaco. La comprobacion
// criptografica se coordina dentro del servicio de aplicacion privado; este
// valor solo impide alias, tamanos ilimitados y sobres vacios evidentes.
type SobreReciboEjecucionDocumentalCrudo struct {
	coseSign1         []byte
	huellaSobreSHA256 string
}

func NuevoSobreReciboEjecucionDocumentalCrudo(coseSign1 []byte) (SobreReciboEjecucionDocumentalCrudo, error) {
	sobre := SobreReciboEjecucionDocumentalCrudo{coseSign1: append([]byte(nil), coseSign1...)}
	sobre.huellaSobreSHA256 = huellaBytesFormatoDocumental(sobre.coseSign1)
	if sobre.Validar() != nil {
		return SobreReciboEjecucionDocumentalCrudo{}, ErrSobreReciboEjecucionDocumentalInvalido
	}
	return sobre, nil
}

func (s SobreReciboEjecucionDocumentalCrudo) Validar() error {
	if len(s.coseSign1) < minimoBytesSobreCOSEDocumental ||
		len(s.coseSign1) > maximoBytesSobreCOSEDocumental || bytesEjecucionDocumentalNulos(s.coseSign1) ||
		!huellaSHA256FormatoDocumentalValida(s.huellaSobreSHA256) ||
		s.huellaSobreSHA256 != huellaBytesFormatoDocumental(s.coseSign1) {
		return ErrSobreReciboEjecucionDocumentalInvalido
	}
	return nil
}

func (s SobreReciboEjecucionDocumentalCrudo) COSESign1() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrSobreReciboEjecucionDocumentalInvalido
	}
	return append([]byte(nil), s.coseSign1...), nil
}

func (s SobreReciboEjecucionDocumentalCrudo) HuellaSHA256() (string, error) {
	if s.Validar() != nil {
		return "", ErrSobreReciboEjecucionDocumentalInvalido
	}
	return s.huellaSobreSHA256, nil
}

func (SobreReciboEjecucionDocumentalCrudo) String() string {
	return "[SOBRE-COSE-RECIBO-EJECUCION-DOCUMENTAL-CRUDO-NO-AUTORITATIVO-REDACTADO]"
}

func (s SobreReciboEjecucionDocumentalCrudo) GoString() string { return s.String() }

func (s SobreReciboEjecucionDocumentalCrudo) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}

func (s SobreReciboEjecucionDocumentalCrudo) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

func (SobreReciboEjecucionDocumentalCrudo) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}

func (*SobreReciboEjecucionDocumentalCrudo) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

func (SobreReciboEjecucionDocumentalCrudo) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}

func (*SobreReciboEjecucionDocumentalCrudo) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

func (SobreReciboEjecucionDocumentalCrudo) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}

func (*SobreReciboEjecucionDocumentalCrudo) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// IdentidadEjecucionComponenteDocumental identifica la carga de trabajo y la
// instancia concreta que firmo el recibo. La medicion debe coincidir despues
// con el artefacto homologado del descriptor.
type IdentidadEjecucionComponenteDocumental struct {
	cargaTrabajoRef       string
	instanciaProcesoRef   string
	dominioAislamientoRef string
	claveFirmaRef         string
	huellaClaveFirma      string
	huellaMedicion        string
}

func NuevaIdentidadEjecucionComponenteDocumental(
	cargaTrabajoRef, instanciaProcesoRef, dominioAislamientoRef, claveFirmaRef string,
	huellaClaveFirmaSHA256, huellaMedicionSHA256 string,
) (IdentidadEjecucionComponenteDocumental, error) {
	identidad := IdentidadEjecucionComponenteDocumental{
		cargaTrabajoRef: cargaTrabajoRef, instanciaProcesoRef: instanciaProcesoRef,
		dominioAislamientoRef: dominioAislamientoRef, claveFirmaRef: claveFirmaRef,
		huellaClaveFirma: huellaClaveFirmaSHA256, huellaMedicion: huellaMedicionSHA256,
	}
	if identidad.Validar() != nil {
		return IdentidadEjecucionComponenteDocumental{}, ErrIdentidadEjecucionDocumentalInvalida
	}
	return identidad, nil
}

func (i IdentidadEjecucionComponenteDocumental) Validar() error {
	referencias := []string{
		i.cargaTrabajoRef, i.instanciaProcesoRef, i.dominioAislamientoRef, i.claveFirmaRef,
	}
	for indice, referencia := range referencias {
		if !referenciaOpacaEjecucionDocumentalSegura(referencia) {
			return ErrIdentidadEjecucionDocumentalInvalida
		}
		for otroIndice := indice + 1; otroIndice < len(referencias); otroIndice++ {
			if referencia == referencias[otroIndice] {
				return ErrIdentidadEjecucionDocumentalInvalida
			}
		}
	}
	if !huellaSHA256FormatoDocumentalValida(i.huellaClaveFirma) ||
		!huellaSHA256FormatoDocumentalValida(i.huellaMedicion) ||
		i.huellaClaveFirma == i.huellaMedicion {
		return ErrIdentidadEjecucionDocumentalInvalida
	}
	return nil
}

func (i IdentidadEjecucionComponenteDocumental) CargaTrabajoRef() string {
	return i.cargaTrabajoRef
}
func (i IdentidadEjecucionComponenteDocumental) InstanciaProcesoRef() string {
	return i.instanciaProcesoRef
}
func (i IdentidadEjecucionComponenteDocumental) DominioAislamientoRef() string {
	return i.dominioAislamientoRef
}
func (i IdentidadEjecucionComponenteDocumental) ClaveFirmaRef() string { return i.claveFirmaRef }
func (i IdentidadEjecucionComponenteDocumental) HuellaClaveFirmaSHA256() string {
	return i.huellaClaveFirma
}
func (i IdentidadEjecucionComponenteDocumental) HuellaMedicionSHA256() string {
	return i.huellaMedicion
}

func (IdentidadEjecucionComponenteDocumental) String() string {
	return "[IDENTIDAD-EJECUCION-COMPONENTE-DOCUMENTAL-REFERENCIAS-REDACTADAS]"
}
func (i IdentidadEjecucionComponenteDocumental) GoString() string { return i.String() }
func (i IdentidadEjecucionComponenteDocumental) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, i.String())
}
func (i IdentidadEjecucionComponenteDocumental) LogValue() slog.Value {
	return slog.StringValue(i.String())
}
func (IdentidadEjecucionComponenteDocumental) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*IdentidadEjecucionComponenteDocumental) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (IdentidadEjecucionComponenteDocumental) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*IdentidadEjecucionComponenteDocumental) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (IdentidadEjecucionComponenteDocumental) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*IdentidadEjecucionComponenteDocumental) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type ResultadoEjecucionComponenteDocumental string

const (
	ResultadoRenderizadoDocumentalCorrecto  ResultadoEjecucionComponenteDocumental = "renderizado_correcto"
	ResultadoEstructuraDocumentalConforme   ResultadoEjecucionComponenteDocumental = "estructura_conforme"
	ResultadoSemanticaDocumentalEquivalente ResultadoEjecucionComponenteDocumental = "semantica_equivalente"
)

func (r ResultadoEjecucionComponenteDocumental) corresponde(
	operacion OperacionComponenteDocumental,
) bool {
	switch operacion {
	case OperacionRenderizadoDocumental:
		return r == ResultadoRenderizadoDocumentalCorrecto
	case OperacionValidacionEstructuralDocumental:
		return r == ResultadoEstructuraDocumentalConforme
	case OperacionVerificacionSemanticaDocumental:
		return r == ResultadoSemanticaDocumentalEquivalente
	default:
		return false
	}
}

// proyeccionCompromisoEjecucionDocumental conserva exclusivamente datos no
// secretos necesarios para auditar y cotejar un recibo. El token, su MAC, el
// manifiesto-capacidad y la solicitud criptografica no atraviesan este corte.
type proyeccionCompromisoEjecucionDocumental struct {
	huellaCompromisoSHA256     string
	operacionRef               string
	huellaRetoSHA256           string
	operacion                  OperacionComponenteDocumental
	descriptorPerfil           DescriptorPerfilDocumental
	situacionOperativa         domain.SituacionOperativaPerfilDocumental
	descriptorComponente       DescriptorComponenteDocumentalAtestado
	reservaRef                 string
	efectoRef                  string
	huellaPlanSHA256           string
	secuenciaCercado           uint64
	huellaVinculoSHA256        string
	decisionRef                string
	esquemaHuellaDecision      string
	huellaDecisionSHA256       string
	inicioEfectoRef            string
	outboxInicioRef            string
	reclamacionDespachoRef     string
	consumoDespachoRef         string
	outboxConsumoRef           string
	versionInicioCAS           uint64
	versionReclamacionCAS      uint64
	versionConsumoCAS          uint64
	huellaOrdenDespachoSHA256  string
	comprobacionKMSRef         string
	consumidaEn                time.Time
	borradorRef                string
	huellaContenidoNeutralHMAC string
	huellaDocumentoSHA256      string
	tamanoDocumento            uint64
	limiteBytes                uint64
	emitidoEn                  time.Time
	expiraEn                   time.Time
	huellaProyeccionSHA256     string
}

func nuevaProyeccionCompromisoEjecucionDocumental(
	compromiso CompromisoEjecucionComponenteDocumental,
) (proyeccionCompromisoEjecucionDocumental, error) {
	huellaCompromiso, err := compromiso.HuellaSHA256()
	if err != nil {
		return proyeccionCompromisoEjecucionDocumental{}, ErrReciboEjecucionDocumentalInvalido
	}
	reto := compromiso.Reto()
	proyeccion := proyeccionCompromisoEjecucionDocumental{
		huellaCompromisoSHA256: huellaCompromiso,
		operacionRef:           compromiso.OperacionRef(),
		huellaRetoSHA256:       huellaBytesFormatoDocumental(reto[:]),
		operacion:              compromiso.Operacion(), descriptorPerfil: compromiso.DescriptorPerfil(),
		situacionOperativa:   compromiso.SituacionOperativa(),
		descriptorComponente: compromiso.DescriptorComponente(),
		reservaRef:           compromiso.ReservaRef(), efectoRef: compromiso.EfectoRef(),
		huellaPlanSHA256:           compromiso.HuellaPlanSHA256(),
		secuenciaCercado:           compromiso.SecuenciaCercado(),
		huellaVinculoSHA256:        compromiso.HuellaVinculoCercadoSHA256(),
		decisionRef:                compromiso.consumoDecision.DecisionRef,
		esquemaHuellaDecision:      compromiso.consumoDecision.EsquemaHuellaDecision,
		huellaDecisionSHA256:       compromiso.consumoDecision.HuellaDecisionSHA256,
		inicioEfectoRef:            compromiso.inicioEfectoRef,
		outboxInicioRef:            compromiso.outboxInicioRef,
		reclamacionDespachoRef:     compromiso.reclamacionDespachoRef,
		consumoDespachoRef:         compromiso.consumoDespachoRef,
		outboxConsumoRef:           compromiso.outboxConsumoRef,
		versionInicioCAS:           compromiso.versionInicioCAS,
		versionReclamacionCAS:      compromiso.versionReclamacionCAS,
		versionConsumoCAS:          compromiso.versionConsumoCAS,
		huellaOrdenDespachoSHA256:  compromiso.huellaOrdenDespachoSHA256,
		comprobacionKMSRef:         compromiso.comprobacionKMSRef,
		consumidaEn:                compromiso.emitidoEn,
		borradorRef:                compromiso.BorradorRef(),
		huellaContenidoNeutralHMAC: compromiso.HuellaContenidoNeutralHMAC(),
		huellaDocumentoSHA256:      compromiso.HuellaDocumentoSHA256(),
		tamanoDocumento:            compromiso.TamanoDocumento(), limiteBytes: compromiso.LimiteBytes(),
		emitidoEn: compromiso.EmitidoEn(), expiraEn: compromiso.ExpiraEn(),
	}
	proyeccion.huellaProyeccionSHA256 = proyeccion.calcularHuella()
	if proyeccion.Validar() != nil {
		return proyeccionCompromisoEjecucionDocumental{}, ErrReciboEjecucionDocumentalInvalido
	}
	return proyeccion, nil
}

func (p proyeccionCompromisoEjecucionDocumental) Validar() error {
	rol, operacionValida := p.operacion.rolEsperado()
	perfil := p.descriptorPerfil.Perfil()
	consulta := p.descriptorComponente.Consulta()
	if !operacionValida || !huellaSHA256FormatoDocumentalValida(p.huellaCompromisoSHA256) ||
		!referenciaOpacaEjecucionDocumentalSegura(p.operacionRef) ||
		!huellaSHA256FormatoDocumentalValida(p.huellaRetoSHA256) ||
		p.descriptorPerfil.Validar() != nil || p.situacionOperativa.Validar() != nil ||
		p.situacionOperativa.PublicacionRef() != p.descriptorPerfil.PublicacionRef() ||
		!p.situacionOperativa.AutorizaEjecucion(perfil, p.descriptorPerfil.Revision()) ||
		p.descriptorComponente.Validar() != nil || p.descriptorComponente.Componente().Rol() != rol ||
		consulta.DescriptorPerfilRef != p.descriptorPerfil.Referencia() ||
		consulta.PublicacionRef != p.descriptorPerfil.PublicacionRef() ||
		consulta.PerfilRef != perfil.Referencia() || consulta.DigestPerfil != perfil.DigestSHA256() ||
		consulta.RevisionCatalogo != p.descriptorPerfil.Revision() ||
		!referenciaOpacaEjecucionDocumentalSegura(p.reservaRef) ||
		!referenciaOpacaEjecucionDocumentalSegura(p.efectoRef) ||
		!huellaSHA256FormatoDocumentalValida(p.huellaPlanSHA256) || p.secuenciaCercado == 0 ||
		!huellaSHA256FormatoDocumentalValida(p.huellaVinculoSHA256) ||
		!referenciaOpacaEjecucionDocumentalSegura(p.decisionRef) ||
		p.esquemaHuellaDecision != EsquemaHuellaDecisionAutorizacionReforzadaV1 ||
		!huellaSHA256FormatoDocumentalValida(p.huellaDecisionSHA256) ||
		!referenciaOpacaEjecucionDocumentalSegura(p.inicioEfectoRef) ||
		!referenciaOpacaEjecucionDocumentalSegura(p.outboxInicioRef) ||
		!referenciaOpacaEjecucionDocumentalSegura(p.reclamacionDespachoRef) ||
		!referenciaOpacaEjecucionDocumentalSegura(p.consumoDespachoRef) ||
		!referenciaOpacaEjecucionDocumentalSegura(p.outboxConsumoRef) ||
		p.versionInicioCAS == 0 || p.versionReclamacionCAS == 0 ||
		p.versionConsumoCAS <= p.versionReclamacionCAS ||
		!huellaSHA256FormatoDocumentalValida(p.huellaOrdenDespachoSHA256) ||
		!referenciaOpacaEjecucionDocumentalSegura(p.comprobacionKMSRef) ||
		!p.consumidaEn.Equal(p.emitidoEn) ||
		!referenciaOpacaEjecucionDocumentalSegura(p.borradorRef) ||
		!huellaHMACEjecucionDocumentalValida(p.huellaContenidoNeutralHMAC) ||
		p.limiteBytes == 0 || p.limiteBytes > maximoBytesEjecucionComponenteDocumental ||
		!ventanaCompromisoEjecucionDocumentalValida(p.emitidoEn, p.expiraEn) ||
		!huellaSHA256FormatoDocumentalValida(p.huellaProyeccionSHA256) ||
		p.huellaProyeccionSHA256 != p.calcularHuella() {
		return ErrReciboEjecucionDocumentalInvalido
	}
	if p.operacion == OperacionRenderizadoDocumental {
		if p.huellaDocumentoSHA256 != "" || p.tamanoDocumento != 0 {
			return ErrReciboEjecucionDocumentalInvalido
		}
		return nil
	}
	if !huellaSHA256FormatoDocumentalValida(p.huellaDocumentoSHA256) ||
		p.tamanoDocumento == 0 || p.tamanoDocumento > p.limiteBytes {
		return ErrReciboEjecucionDocumentalInvalido
	}
	return nil
}

func (p proyeccionCompromisoEjecucionDocumental) calcularHuella() string {
	return huellaCanonicaFormatoDocumental([]string{
		"vec.proyeccion-segura-compromiso-ejecucion-documental.v1", p.huellaCompromisoSHA256,
		p.operacionRef, p.huellaRetoSHA256, string(p.operacion), p.descriptorPerfil.Referencia(),
		p.descriptorPerfil.PublicacionRef(), p.descriptorPerfil.Perfil().DigestSHA256(),
		p.situacionOperativa.HuellaSHA256(), p.descriptorComponente.Referencia(),
		p.descriptorComponente.DigestDeclaracionSHA256(), p.reservaRef, p.efectoRef,
		p.huellaPlanSHA256, strconv.FormatUint(p.secuenciaCercado, 10), p.huellaVinculoSHA256,
		p.decisionRef, p.esquemaHuellaDecision, p.huellaDecisionSHA256,
		p.inicioEfectoRef, p.outboxInicioRef, p.reclamacionDespachoRef,
		p.consumoDespachoRef, p.outboxConsumoRef,
		strconv.FormatUint(p.versionInicioCAS, 10), strconv.FormatUint(p.versionReclamacionCAS, 10),
		strconv.FormatUint(p.versionConsumoCAS, 10),
		p.huellaOrdenDespachoSHA256, p.comprobacionKMSRef,
		p.consumidaEn.Format(time.RFC3339Nano), p.borradorRef,
		p.huellaContenidoNeutralHMAC, p.huellaDocumentoSHA256,
		strconv.FormatUint(p.tamanoDocumento, 10), strconv.FormatUint(p.limiteBytes, 10),
		p.emitidoEn.Format(time.RFC3339Nano), p.expiraEn.Format(time.RFC3339Nano),
	})
}

type ReciboEjecucionComponenteDocumentalNominal struct {
	compromiso            proyeccionCompromisoEjecucionDocumental
	reciboRef             string
	resultado             ResultadoEjecucionComponenteDocumental
	huellaSalidaSHA256    string
	tamanoSalida          uint64
	identidad             IdentidadEjecucionComponenteDocumental
	emitidoEn             time.Time
	huellaSobreCOSESHA256 string
	huellaReciboSHA256    string
}

// DatosReciboEjecucionComponenteDocumentalNominal es una proyeccion segura
// para evidencia. Nunca expone CompromisoEjecucionComponenteDocumental,
// TokenCercadoEjecucionDocumentalV3Nominal, su MAC ni el valor secreto de cercado.
type DatosReciboEjecucionComponenteDocumentalNominal struct {
	HuellaCompromisoSHA256     string
	OperacionRef               string
	Operacion                  OperacionComponenteDocumental
	DescriptorPerfil           DescriptorPerfilDocumental
	SituacionOperativa         domain.SituacionOperativaPerfilDocumental
	DescriptorComponente       DescriptorComponenteDocumentalAtestado
	ReservaRef                 string
	EfectoRef                  string
	HuellaPlanSHA256           string
	SecuenciaCercado           uint64
	HuellaVinculoCercadoSHA256 string
	DecisionRef                string
	EsquemaHuellaDecision      string
	HuellaDecisionSHA256       string
	InicioEfectoRef            string
	OutboxInicioRef            string
	ReclamacionDespachoRef     string
	ConsumoDespachoRef         string
	OutboxConsumoRef           string
	VersionInicioCAS           uint64
	VersionReclamacionCAS      uint64
	VersionConsumoCAS          uint64
	HuellaOrdenDespachoSHA256  string
	ComprobacionKMSRef         string
	ConsumidaEn                time.Time
	BorradorRef                string
	HuellaContenidoNeutralHMAC string
	HuellaDocumentoSHA256      string
	TamanoDocumento            uint64
	LimiteBytes                uint64
	CompromisoEmitidoEn        time.Time
	CompromisoExpiraEn         time.Time
	ReciboRef                  string
	Resultado                  ResultadoEjecucionComponenteDocumental
	HuellaSalidaSHA256         string
	TamanoSalida               uint64
	Identidad                  IdentidadEjecucionComponenteDocumental
	EmitidoEn                  time.Time
	HuellaSobreCOSESHA256      string
	HuellaReciboSHA256         string
}

func (DatosReciboEjecucionComponenteDocumentalNominal) String() string {
	return "[DATOS-RECIBO-EJECUCION-COMPONENTE-NOMINALES-HMAC-REDACTADOS]"
}
func (d DatosReciboEjecucionComponenteDocumentalNominal) GoString() string { return d.String() }
func (d DatosReciboEjecucionComponenteDocumentalNominal) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}
func (d DatosReciboEjecucionComponenteDocumentalNominal) LogValue() slog.Value {
	return slog.StringValue(d.String())
}
func (DatosReciboEjecucionComponenteDocumentalNominal) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosReciboEjecucionComponenteDocumentalNominal) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosReciboEjecucionComponenteDocumentalNominal) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosReciboEjecucionComponenteDocumentalNominal) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosReciboEjecucionComponenteDocumentalNominal) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosReciboEjecucionComponenteDocumentalNominal) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// NuevoReciboEjecucionComponenteDocumentalNominal solo comprueba el contrato
// de valor: NO acredita la firma ni la atestacion criptografica y nunca concede
// autoridad por si mismo.
func NuevoReciboEjecucionComponenteDocumentalNominal(
	compromiso CompromisoEjecucionComponenteDocumental,
	sobre SobreReciboEjecucionDocumentalCrudo,
	reciboRef string,
	resultado ResultadoEjecucionComponenteDocumental,
	huellaSalidaSHA256 string,
	tamanoSalida uint64,
	identidad IdentidadEjecucionComponenteDocumental,
	emitidoEn time.Time,
) (ReciboEjecucionComponenteDocumentalNominal, error) {
	huellaSobre, err := sobre.HuellaSHA256()
	if err != nil {
		return ReciboEjecucionComponenteDocumentalNominal{}, ErrReciboEjecucionDocumentalInvalido
	}
	proyeccion, err := nuevaProyeccionCompromisoEjecucionDocumental(compromiso)
	if err != nil {
		return ReciboEjecucionComponenteDocumentalNominal{}, ErrReciboEjecucionDocumentalInvalido
	}
	recibo := ReciboEjecucionComponenteDocumentalNominal{
		compromiso: proyeccion, reciboRef: reciboRef, resultado: resultado,
		huellaSalidaSHA256: huellaSalidaSHA256, tamanoSalida: tamanoSalida,
		identidad: identidad, emitidoEn: emitidoEn, huellaSobreCOSESHA256: huellaSobre,
	}
	recibo.huellaReciboSHA256 = recibo.calcularHuella()
	if recibo.ValidarContra(compromiso, sobre) != nil {
		return ReciboEjecucionComponenteDocumentalNominal{}, ErrReciboEjecucionDocumentalInvalido
	}
	return recibo, nil
}

func (r ReciboEjecucionComponenteDocumentalNominal) Validar() error {
	if r.compromiso.Validar() != nil || !referenciaOpacaEjecucionDocumentalSegura(r.reciboRef) ||
		r.reciboRef == r.compromiso.operacionRef || r.reciboRef == r.compromiso.borradorRef ||
		!r.resultado.corresponde(r.compromiso.operacion) ||
		!huellaSHA256FormatoDocumentalValida(r.huellaSalidaSHA256) || r.tamanoSalida == 0 ||
		r.tamanoSalida > r.compromiso.limiteBytes || r.identidad.Validar() != nil ||
		r.identidad.DominioAislamientoRef() != r.compromiso.descriptorComponente.DominioConfianzaRef() ||
		r.identidad.HuellaMedicionSHA256() !=
			r.compromiso.descriptorComponente.Componente().HuellaArtefactoSHA256() ||
		r.emitidoEn.IsZero() || r.emitidoEn.Location() != time.UTC ||
		r.emitidoEn.Before(r.compromiso.emitidoEn) || !r.emitidoEn.Before(r.compromiso.expiraEn) ||
		r.emitidoEn.Before(r.compromiso.consumidaEn) ||
		!huellaSHA256FormatoDocumentalValida(r.huellaSobreCOSESHA256) ||
		!huellaSHA256FormatoDocumentalValida(r.huellaReciboSHA256) ||
		r.huellaReciboSHA256 != r.calcularHuella() {
		return ErrReciboEjecucionDocumentalInvalido
	}
	if r.compromiso.operacion != OperacionRenderizadoDocumental &&
		(r.huellaSalidaSHA256 != r.compromiso.huellaDocumentoSHA256 ||
			r.tamanoSalida != r.compromiso.tamanoDocumento) {
		return ErrReciboEjecucionDocumentalInvalido
	}
	return nil
}

func (r ReciboEjecucionComponenteDocumentalNominal) ValidarContra(
	compromiso CompromisoEjecucionComponenteDocumental,
	sobre SobreReciboEjecucionDocumentalCrudo,
) error {
	huellaSobre, err := sobre.HuellaSHA256()
	proyeccionEsperada, errProyeccion := nuevaProyeccionCompromisoEjecucionDocumental(compromiso)
	if err != nil || r.Validar() != nil || compromiso.Validar() != nil ||
		errProyeccion != nil || r.compromiso != proyeccionEsperada ||
		r.huellaSobreCOSESHA256 != huellaSobre {
		return ErrReciboEjecucionDocumentalInvalido
	}
	return nil
}

func (r ReciboEjecucionComponenteDocumentalNominal) Datos() (
	DatosReciboEjecucionComponenteDocumentalNominal,
	error,
) {
	if r.Validar() != nil {
		return DatosReciboEjecucionComponenteDocumentalNominal{}, ErrReciboEjecucionDocumentalInvalido
	}
	return DatosReciboEjecucionComponenteDocumentalNominal{
		HuellaCompromisoSHA256: r.compromiso.huellaCompromisoSHA256,
		OperacionRef:           r.compromiso.operacionRef, Operacion: r.compromiso.operacion,
		DescriptorPerfil:     r.compromiso.descriptorPerfil,
		SituacionOperativa:   r.compromiso.situacionOperativa,
		DescriptorComponente: r.compromiso.descriptorComponente,
		ReservaRef:           r.compromiso.reservaRef, EfectoRef: r.compromiso.efectoRef,
		HuellaPlanSHA256:           r.compromiso.huellaPlanSHA256,
		SecuenciaCercado:           r.compromiso.secuenciaCercado,
		HuellaVinculoCercadoSHA256: r.compromiso.huellaVinculoSHA256,
		DecisionRef:                r.compromiso.decisionRef,
		EsquemaHuellaDecision:      r.compromiso.esquemaHuellaDecision,
		HuellaDecisionSHA256:       r.compromiso.huellaDecisionSHA256,
		InicioEfectoRef:            r.compromiso.inicioEfectoRef,
		OutboxInicioRef:            r.compromiso.outboxInicioRef,
		ReclamacionDespachoRef:     r.compromiso.reclamacionDespachoRef,
		ConsumoDespachoRef:         r.compromiso.consumoDespachoRef,
		OutboxConsumoRef:           r.compromiso.outboxConsumoRef,
		VersionInicioCAS:           r.compromiso.versionInicioCAS,
		VersionReclamacionCAS:      r.compromiso.versionReclamacionCAS,
		VersionConsumoCAS:          r.compromiso.versionConsumoCAS,
		HuellaOrdenDespachoSHA256:  r.compromiso.huellaOrdenDespachoSHA256,
		ComprobacionKMSRef:         r.compromiso.comprobacionKMSRef,
		ConsumidaEn:                r.compromiso.consumidaEn,
		BorradorRef:                r.compromiso.borradorRef,
		HuellaContenidoNeutralHMAC: r.compromiso.huellaContenidoNeutralHMAC,
		HuellaDocumentoSHA256:      r.compromiso.huellaDocumentoSHA256,
		TamanoDocumento:            r.compromiso.tamanoDocumento, LimiteBytes: r.compromiso.limiteBytes,
		CompromisoEmitidoEn: r.compromiso.emitidoEn, CompromisoExpiraEn: r.compromiso.expiraEn,
		ReciboRef: r.reciboRef, Resultado: r.resultado,
		HuellaSalidaSHA256: r.huellaSalidaSHA256, TamanoSalida: r.tamanoSalida,
		Identidad: r.identidad, EmitidoEn: r.emitidoEn,
		HuellaSobreCOSESHA256: r.huellaSobreCOSESHA256,
		HuellaReciboSHA256:    r.huellaReciboSHA256,
	}, nil
}

func (r ReciboEjecucionComponenteDocumentalNominal) HuellaSHA256() (string, error) {
	if r.Validar() != nil {
		return "", ErrReciboEjecucionDocumentalInvalido
	}
	return r.huellaReciboSHA256, nil
}

func (ReciboEjecucionComponenteDocumentalNominal) String() string {
	return "[RECIBO-EJECUCION-COMPONENTE-DOCUMENTAL-NOMINAL-REDACTADO]"
}
func (r ReciboEjecucionComponenteDocumentalNominal) GoString() string { return r.String() }
func (r ReciboEjecucionComponenteDocumentalNominal) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r ReciboEjecucionComponenteDocumentalNominal) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
func (ReciboEjecucionComponenteDocumentalNominal) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ReciboEjecucionComponenteDocumentalNominal) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (ReciboEjecucionComponenteDocumentalNominal) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ReciboEjecucionComponenteDocumentalNominal) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (ReciboEjecucionComponenteDocumentalNominal) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ReciboEjecucionComponenteDocumentalNominal) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// IndependienteDe exige segregacion tanto declarativa como observada durante
// la ejecucion. El broker y el nodo fisico pueden ser compartidos; la carga de
// trabajo, proceso, dominio, clave y medicion no.
func (r ReciboEjecucionComponenteDocumentalNominal) IndependienteDe(
	otro ReciboEjecucionComponenteDocumentalNominal,
) bool {
	componente := r.compromiso.descriptorComponente
	otroComponente := otro.compromiso.descriptorComponente
	if r.Validar() != nil || otro.Validar() != nil ||
		!componente.IndependienteDe(otroComponente) ||
		r.compromiso.operacion == otro.compromiso.operacion ||
		r.compromiso.operacionRef == otro.compromiso.operacionRef ||
		r.compromiso.huellaRetoSHA256 == otro.compromiso.huellaRetoSHA256 ||
		r.reciboRef == otro.reciboRef ||
		r.identidad.CargaTrabajoRef() == otro.identidad.CargaTrabajoRef() ||
		r.identidad.InstanciaProcesoRef() == otro.identidad.InstanciaProcesoRef() ||
		r.identidad.DominioAislamientoRef() == otro.identidad.DominioAislamientoRef() ||
		r.identidad.ClaveFirmaRef() == otro.identidad.ClaveFirmaRef() ||
		r.identidad.HuellaClaveFirmaSHA256() == otro.identidad.HuellaClaveFirmaSHA256() ||
		r.identidad.HuellaMedicionSHA256() == otro.identidad.HuellaMedicionSHA256() {
		return false
	}
	return true
}

func (r ReciboEjecucionComponenteDocumentalNominal) calcularHuella() string {
	huellaCompromiso := r.compromiso.huellaCompromisoSHA256
	componente := r.compromiso.descriptorComponente
	return huellaCanonicaFormatoDocumental([]string{
		"vec.recibo-ejecucion-componente-documental-nominal.v1", huellaCompromiso,
		r.reciboRef, string(r.resultado), r.huellaSalidaSHA256,
		strconv.FormatUint(r.tamanoSalida, 10), r.identidad.CargaTrabajoRef(),
		r.identidad.InstanciaProcesoRef(), r.identidad.DominioAislamientoRef(),
		r.identidad.ClaveFirmaRef(), r.identidad.HuellaClaveFirmaSHA256(),
		r.identidad.HuellaMedicionSHA256(), componente.AtestacionBrokerRef(),
		componente.HuellaAtestacionBrokerSHA256(), r.emitidoEn.Format(time.RFC3339Nano),
		r.huellaSobreCOSESHA256,
	})
}

// DespachadorComponentesDocumentalesAtestados es transporte hacia procesos
// aislados; no es por si mismo un ejecutor homologado. El recibo firmado por la
// carga de trabajo es obligatorio para aceptar cualquier resultado.
type DespachadorComponentesDocumentalesAtestados interface {
	// Cada metodo acepta un compromiso nominal ligado al consumo CAS. Este puerto
	// solo puede estar en el servicio de aplicacion precompuesto, que verifica KMS,
	// consume y despacha dentro de la misma llamada. Nunca se entrega a handlers.
	Renderizar(
		context.Context,
		CompromisoEjecucionComponenteDocumental,
		domain.PerfilFormatoDocumental,
		domain.ContenidoDocumento,
		io.Writer,
	) (SobreReciboEjecucionDocumentalCrudo, error)
	ValidarEstructura(
		context.Context,
		CompromisoEjecucionComponenteDocumental,
		domain.PerfilFormatoDocumental,
		[]byte,
	) (SobreReciboEjecucionDocumentalCrudo, error)
	VerificarSemantica(
		context.Context,
		CompromisoEjecucionComponenteDocumental,
		domain.PerfilFormatoDocumental,
		domain.ContenidoDocumento,
		[]byte,
	) (SobreReciboEjecucionDocumentalCrudo, error)
}

type GeneradorRetosEjecucionDocumental interface {
	NuevoRetoEjecucionDocumental(context.Context) ([32]byte, error)
}

// VerificadorCrudoRecibosDocumentales es el conector intercambiable que coteja
// COSE, confianza, vigencia/revocacion, ventana/reto y clave. Su salida es
// nominal y nunca concede por si sola permiso para confirmar un efecto.
type VerificadorCrudoRecibosDocumentales interface {
	VerificarReciboCrudo(
		context.Context,
		CompromisoEjecucionComponenteDocumental,
		SobreReciboEjecucionDocumentalCrudo,
	) (ReciboEjecucionComponenteDocumentalNominal, error)
}

func referenciaOpacaEjecucionDocumentalSegura(valor string) bool {
	if !documentalcanonico.ReferenciaASCIIBasicaValida(valor) ||
		strings.ContainsRune(valor, '*') || valor == "nil" || valor == "null" {
		return false
	}
	segmentos := strings.FieldsFunc(valor, func(r rune) bool {
		return r == ':' || r == '.' || r == '_' || r == '-'
	})
	for _, segmento := range segmentos {
		switch segmento {
		case "dni", "nie", "nif", "email", "correo", "telefono", "movil", "nombre", "apellidos":
			return false
		}
		if documentalcanonico.DNINIEASCIIMinusculoEvidente(segmento) {
			return false
		}
	}
	return true
}

func componenteEjecucionDocumentalPerteneceAlPlan(
	operacion OperacionComponenteDocumental,
	componente DescriptorComponenteDocumentalAtestado,
	datos DatosManifiestoEjecucionDocumentalV3,
) bool {
	switch operacion {
	case OperacionRenderizadoDocumental:
		return componente == datos.ComponenteRender
	case OperacionValidacionEstructuralDocumental:
		return componente == datos.ComponenteVerificador
	case OperacionVerificacionSemanticaDocumental:
		return componente == datos.ComponenteSemantico
	default:
		return false
	}
}

func descriptorEjecucionDocumentalSinPIIEvidente(
	perfil DescriptorPerfilDocumental,
	componente DescriptorComponenteDocumentalAtestado,
) bool {
	perfilValor := perfil.Perfil()
	componenteValor := componente.Componente()
	valores := []string{
		perfil.Referencia(), perfil.PublicacionRef(), perfilValor.Referencia().Identificador(),
		componente.Referencia(), componenteValor.Identificador(), componenteValor.HomologacionRef(),
		componente.DominioConfianzaRef(), componente.BrokerRef(), componente.AtestacionBrokerRef(),
	}
	for _, valor := range valores {
		if !referenciaOpacaEjecucionDocumentalSegura(valor) {
			return false
		}
	}
	return true
}

func huellaHMACEjecucionDocumentalValida(valor string) bool {
	if len(valor) == 0 || len(valor) > 512 || valor != strings.TrimSpace(valor) ||
		strings.ContainsRune(valor, '*') {
		return false
	}
	partes := strings.Split(valor, ":")
	if len(partes) != 3 || partes[0] != "hmac-sha256" ||
		!documentalcanonico.IDClaveHMACASCIIBasicoValido(partes[1]) ||
		!referenciaOpacaEjecucionDocumentalSegura(partes[1]) ||
		len(partes[2]) != sha256.Size*2 || partes[2] != strings.ToLower(partes[2]) {
		return false
	}
	decodificada, err := hex.DecodeString(partes[2])
	return err == nil && len(decodificada) == sha256.Size
}

func retoEjecucionDocumentalNulo(reto [32]byte) bool {
	var nulo [32]byte
	return reto == nulo
}

func ventanaCompromisoEjecucionDocumentalValida(emitidoEn, expiraEn time.Time) bool {
	return instanteEjecucionDocumentalV3Valido(emitidoEn) &&
		instanteEjecucionDocumentalV3Valido(expiraEn) &&
		expiraEn.After(emitidoEn) && expiraEn.Sub(emitidoEn) <= maximaVigenciaCompromisoDocumental
}

func bytesEjecucionDocumentalNulos(valor []byte) bool {
	for _, octeto := range valor {
		if octeto != 0 {
			return false
		}
	}
	return true
}
