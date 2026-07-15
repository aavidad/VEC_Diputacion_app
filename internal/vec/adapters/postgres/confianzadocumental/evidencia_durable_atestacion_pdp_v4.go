package confianzadocumental

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrEvidenciaDurableAtestacionPDPV4Invalida                      = errors.New("vec: evidencia durable de atestacion PDP v4 invalida")
	ErrSerializacionGeneralEvidenciaDurableAtestacionPDPV4Prohibida = errors.New("vec: serializacion general de evidencia durable de atestacion PDP v4 prohibida")
)

const (
	esquemaEvidenciaDurableAtestacionPDPV4            = "vec.documentos.evidencia-durable-atestacion-pdp.v4"
	versionEvidenciaDurableAtestacionPDPV4     uint16 = 1
	marcaEvidenciaDurableAtestacionPDPV4              = "vec.evidencia-durable-atestacion-pdp.v4"
	maximoBytesEvidenciaDurableAtestacionPDPV4        = 2 * 1024 * 1024
)

// MetadatosEvidenciaDurableAtestacionPDPV4 es la proyeccion deliberada que un
// adaptador duradero puede mapear a columnas. Es evidencia descriptiva, no una
// autorizacion ni un sustituto del sobre y payload que deben persistirse.
type MetadatosEvidenciaDurableAtestacionPDPV4 struct {
	Esquema                        string
	Version                        uint16
	DecisionRef                    string
	HuellaPlanSHA256               string
	EfectoRef                      string
	HuellaSolicitudVinculadaSHA256 string
	FormatoVECADVersion            uint16
	Suite                          string
	ClaveID                        string
	AudienciaDespliegue            string
	AlgoritmoCOSE                  AlgoritmoCOSEDocumental
	AudienciaCOSE                  AudienciaCOSEDocumental
	EstadoConfianza                EstadoConfianzaClaveDocumental
	HuellaClaveSHA256              string
	HuellaPayloadSHA256            string
	HuellaSobreSHA256              string
	VerificadaEn                   time.Time
	RaizValidaDesde                time.Time
	RaizValidaHasta                time.Time
	RevisionConfianza              string
	HuellaConfiguracionSHA256      string
	ConfiguracionPublicadaEn       time.Time
	ConfiguracionExpiraEn          time.Time
	HuellaPreimagenRecursoSHA256   string
	HuellaContextoRecursoSHA256    string
	HuellaAmbitosRecursoSHA256     string
	HuellaEvidenciaDurableSHA256   string
}

// EvidenciaDurableAtestacionAutorizacionPDPV4 conserva payload y sobre para
// que un COMMIT futuro pueda volver a verificar COSE de manera independiente.
// No concede autoridad. Solo una AutoridadInterna ya atestada puede entregarla.
//
// La clave publica no se autocertifica dentro de esta evidencia: recuperarla
// del mismo registro no demostraria confianza. Operacion debe conservar el
// catalogo historico autoritativo de raices y configuraciones, con material
// publico, revisiones y actos. Sin ese registro, la reverificacion futura y el
// gate de produccion deben fallar cerrados.
//
// VEC-AD-1 firma la huella del contexto de recurso, no plan y efecto como
// campos separados. Por eso esta evidencia conserva tambien la preimagen
// canonica completa del recurso. Su validacion exige que los dos atributos
// exactos de plan y efecto recompongan la huella incluida en la decision
// VEC-AD-1 parseada. Aun asi, ese enlace solo se vuelve autentico despues de
// reverificar COSE contra el registro historico confiable.
//
// HuellaSobreSHA256 solo identifica los bytes persistidos y detecta cambios.
// Nunca es clave de replay o idempotencia: incluso una firma valida puede
// repetirse con otros bytes. Esa exclusion se cierra en COMMIT mediante la
// terna semantica DecisionRef, HuellaPlanSHA256 y EfectoRef.
type EvidenciaDurableAtestacionAutorizacionPDPV4 struct {
	marca                 string
	metadatos             MetadatosEvidenciaDurableAtestacionPDPV4
	preimagenRecurso      []byte
	payloadVECAD1         []byte
	sobreCOSESign1        []byte
	serializacionCanonica []byte
}

func nuevaEvidenciaDurableAtestacionAutorizacionPDPV4(
	clave ports.ClaveAplicacionAutorizacionEjecucionDocumentalV4,
	huellaVinculoSHA256 string,
	preimagen ports.PreimagenRecursoAutorizacionEjecucionDocumentalV4,
	comprobacion comprobacionAtestacionAutorizacionPDPV4,
	verificadaEn time.Time,
) (EvidenciaDurableAtestacionAutorizacionPDPV4, error) {
	if comprobacion.prueba.Validar() != nil ||
		validarCabeceraAtestacionPDPContraPrueba(
			comprobacion.cabecera, comprobacion.prueba,
		) != nil || !instanteCanonicoDocumental(verificadaEn) ||
		!comprobacion.prueba.verificadaEn.Equal(verificadaEn) {
		return EvidenciaDurableAtestacionAutorizacionPDPV4{}, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	payload, err := comprobacion.solicitud.PayloadEsperado()
	if err != nil {
		return EvidenciaDurableAtestacionAutorizacionPDPV4{}, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	sobre, err := comprobacion.sobre.COSESign1()
	if err != nil {
		return EvidenciaDurableAtestacionAutorizacionPDPV4{}, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	preimagenBytes, err := preimagen.SerializacionCanonicaParaPersistencia()
	huellaPreimagen, errHuellaPreimagen := preimagen.HuellaSHA256()
	huellaContexto, errHuellaContexto := preimagen.HuellaContextoRecursoSHA256()
	huellaAmbitos, errHuellaAmbitos := preimagen.HuellaAmbitosSHA256()
	if err != nil || errHuellaPreimagen != nil || errHuellaContexto != nil ||
		errHuellaAmbitos != nil {
		return EvidenciaDurableAtestacionAutorizacionPDPV4{}, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	prueba := comprobacion.prueba
	evidencia := EvidenciaDurableAtestacionAutorizacionPDPV4{
		marca: marcaEvidenciaDurableAtestacionPDPV4,
		metadatos: MetadatosEvidenciaDurableAtestacionPDPV4{
			Esquema:     esquemaEvidenciaDurableAtestacionPDPV4,
			Version:     versionEvidenciaDurableAtestacionPDPV4,
			DecisionRef: clave.DecisionRef, HuellaPlanSHA256: clave.HuellaPlanSHA256,
			EfectoRef:                      clave.EfectoRef,
			HuellaSolicitudVinculadaSHA256: huellaVinculoSHA256,
			FormatoVECADVersion:            comprobacion.cabecera.FormatoVersion,
			Suite:                          comprobacion.cabecera.Suite, ClaveID: comprobacion.cabecera.ClaveID,
			AudienciaDespliegue: comprobacion.cabecera.Audiencia,
			AlgoritmoCOSE:       prueba.algoritmo, AudienciaCOSE: prueba.audiencia,
			EstadoConfianza:     prueba.estadoConfianza,
			HuellaClaveSHA256:   prueba.huellaClaveSHA256,
			HuellaPayloadSHA256: prueba.huellaPayloadSHA256,
			HuellaSobreSHA256:   prueba.huellaSobreSHA256, VerificadaEn: verificadaEn,
			RaizValidaDesde: prueba.raizValidaDesde, RaizValidaHasta: prueba.raizValidaHasta,
			RevisionConfianza:            prueba.revisionConfianza,
			HuellaConfiguracionSHA256:    prueba.huellaConfiguracionSHA256,
			ConfiguracionPublicadaEn:     prueba.configuracionPublicadaEn,
			ConfiguracionExpiraEn:        prueba.configuracionExpiraEn,
			HuellaPreimagenRecursoSHA256: huellaPreimagen,
			HuellaContextoRecursoSHA256:  huellaContexto,
			HuellaAmbitosRecursoSHA256:   huellaAmbitos,
		},
		preimagenRecurso: append([]byte(nil), preimagenBytes...),
		payloadVECAD1:    append([]byte(nil), payload...),
		sobreCOSESign1:   append([]byte(nil), sobre...),
	}
	evidencia.serializacionCanonica = evidencia.calcularSerializacionCanonica()
	evidencia.metadatos.HuellaEvidenciaDurableSHA256 = huellaBytesDocumentales(
		evidencia.serializacionCanonica,
	)
	if evidencia.Validar() != nil {
		return EvidenciaDurableAtestacionAutorizacionPDPV4{}, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	return evidencia, nil
}

// Validar comprueba integridad y completitud historicas. No comprueba vigencia
// actual ni revocaciones posteriores y, por tanto, nunca concede autoridad.
func (e EvidenciaDurableAtestacionAutorizacionPDPV4) Validar() error {
	m := e.metadatos
	cabecera := domain.CabeceraAtestacionAutorizacionV1{
		FormatoVersion: m.FormatoVECADVersion, Suite: m.Suite,
		ClaveID: m.ClaveID, Audiencia: m.AudienciaDespliegue,
	}
	suiteEsperada, suiteSoportada := suiteAtestacionAutorizacionPDP(m.AlgoritmoCOSE)
	sobre, errSobre := ports.NuevoSobreCriptograficoDocumentalCrudoV4(e.sobreCOSESign1)
	huellaSobre, errHuellaSobre := sobre.HuellaSHA256()
	preimagen, errPreimagen := ports.InterpretarPreimagenRecursoAutorizacionEjecucionDocumentalV4(
		e.preimagenRecurso,
		m.HuellaPreimagenRecursoSHA256,
	)
	recurso, errRecurso := preimagen.RecursoCanonico()
	huellaContexto, errHuellaContexto := preimagen.HuellaContextoRecursoSHA256()
	huellaAmbitos, errHuellaAmbitos := preimagen.HuellaAmbitosSHA256()
	if e.marca != marcaEvidenciaDurableAtestacionPDPV4 ||
		m.Esquema != esquemaEvidenciaDurableAtestacionPDPV4 ||
		m.Version != versionEvidenciaDurableAtestacionPDPV4 ||
		!referenciaDurableAtestacionPDPValida(m.DecisionRef) ||
		!huellaSHA256DocumentalValida(m.HuellaPlanSHA256) ||
		!referenciaDurableAtestacionPDPValida(m.EfectoRef) ||
		!huellaSHA256DocumentalValida(m.HuellaSolicitudVinculadaSHA256) ||
		cabecera.Validar() != nil || !audienciaDespliegueAtestacionPDPValida(m.AudienciaDespliegue) ||
		!suiteSoportada || m.Suite != suiteEsperada ||
		m.AudienciaCOSE != AudienciaCOSEAtestacionAutorizacionPDP ||
		m.EstadoConfianza != EstadoConfianzaClaveDocumentalActiva ||
		!huellaSHA256DocumentalValida(m.HuellaClaveSHA256) ||
		!huellaSHA256DocumentalValida(m.HuellaPayloadSHA256) ||
		!huellaSHA256DocumentalValida(m.HuellaSobreSHA256) ||
		m.HuellaPayloadSHA256 != huellaBytesDocumentales(e.payloadVECAD1) ||
		errSobre != nil || errHuellaSobre != nil || huellaSobre != m.HuellaSobreSHA256 ||
		len(e.payloadVECAD1) == 0 || len(e.payloadVECAD1) > domain.TamanoMaximoMensajeAtestacionAutorizacionV1 ||
		!instanteCanonicoDocumental(m.VerificadaEn) ||
		!instanteCanonicoDocumental(m.RaizValidaDesde) ||
		!instanteCanonicoDocumental(m.RaizValidaHasta) ||
		!m.RaizValidaHasta.After(m.RaizValidaDesde) ||
		m.VerificadaEn.Before(m.RaizValidaDesde) || !m.VerificadaEn.Before(m.RaizValidaHasta) ||
		!referenciaConfiguracionDocumentalValida(m.RevisionConfianza) ||
		!huellaSHA256DocumentalValida(m.HuellaConfiguracionSHA256) ||
		!instanteCanonicoDocumental(m.ConfiguracionPublicadaEn) ||
		!instanteCanonicoDocumental(m.ConfiguracionExpiraEn) ||
		!m.ConfiguracionExpiraEn.After(m.ConfiguracionPublicadaEn) ||
		m.ConfiguracionExpiraEn.Sub(m.ConfiguracionPublicadaEn) >
			maximaVigenciaConfiguracionConfianzaV4 ||
		m.VerificadaEn.Before(m.ConfiguracionPublicadaEn) ||
		!m.VerificadaEn.Before(m.ConfiguracionExpiraEn) ||
		!huellaSHA256DocumentalValida(m.HuellaPreimagenRecursoSHA256) ||
		!huellaSHA256DocumentalValida(m.HuellaContextoRecursoSHA256) ||
		!huellaSHA256DocumentalValida(m.HuellaAmbitosRecursoSHA256) ||
		errPreimagen != nil || errRecurso != nil || errHuellaContexto != nil ||
		errHuellaAmbitos != nil || huellaContexto != m.HuellaContextoRecursoSHA256 ||
		huellaAmbitos != m.HuellaAmbitosRecursoSHA256 ||
		recurso.Atributos[ports.AtributoAutorizacionDocumentalHuellaPlanSHA256] !=
			m.HuellaPlanSHA256 ||
		recurso.Atributos[ports.AtributoAutorizacionDocumentalEfectoRef] != m.EfectoRef ||
		validarVinculoFirmadoRecursoEvidenciaAtestacionPDPV4(
			e.payloadVECAD1, cabecera, m, recurso,
		) != nil ||
		!huellaSHA256DocumentalValida(m.HuellaEvidenciaDurableSHA256) ||
		len(e.serializacionCanonica) == 0 ||
		len(e.serializacionCanonica) > maximoBytesEvidenciaDurableAtestacionPDPV4 ||
		!bytes.Equal(e.serializacionCanonica, e.calcularSerializacionCanonica()) ||
		m.HuellaEvidenciaDurableSHA256 != huellaBytesDocumentales(e.serializacionCanonica) {
		return ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	return nil
}

func validarVinculoFirmadoRecursoEvidenciaAtestacionPDPV4(
	payload []byte,
	cabeceraEsperada domain.CabeceraAtestacionAutorizacionV1,
	metadatos MetadatosEvidenciaDurableAtestacionPDPV4,
	recurso domain.RecursoAutorizable,
) error {
	proyeccion, err := domain.ParsearMensajeAtestacionAutorizacionV1NoAutoritativo(payload)
	if err != nil {
		return ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	cabeceraFirmada, errCabecera := proyeccion.Cabecera()
	datosFirmados, errDatos := proyeccion.Datos()
	if errCabecera != nil || errDatos != nil || cabeceraFirmada != cabeceraEsperada ||
		datosFirmados.DecisionRef != metadatos.DecisionRef ||
		datosFirmados.Accion != ports.AccionEjecutarPlanDocumentalV4 ||
		datosFirmados.RecursoRef != recurso.Referencia ||
		datosFirmados.ModuloID != recurso.ModuloID ||
		datosFirmados.TipoRecurso != recurso.Tipo ||
		datosFirmados.ContextoRecursoHuellaSHA256 !=
			metadatos.HuellaContextoRecursoSHA256 ||
		metadatos.VerificadaEn.Before(datosFirmados.EmitidaEn) ||
		!metadatos.VerificadaEn.Before(datosFirmados.ValidaHasta) {
		return ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	return nil
}

func (e EvidenciaDurableAtestacionAutorizacionPDPV4) coincideConAutoridad(
	a AutoridadInternaEjecucionDocumentalV4,
) bool {
	m := e.metadatos
	return e.Validar() == nil &&
		m.DecisionRef == a.clave.DecisionRef &&
		m.HuellaPlanSHA256 == a.clave.HuellaPlanSHA256 &&
		m.EfectoRef == a.clave.EfectoRef &&
		m.HuellaSolicitudVinculadaSHA256 == a.huellaVinculoSHA256 &&
		m.FormatoVECADVersion == a.cabeceraPDP.FormatoVersion &&
		m.Suite == a.cabeceraPDP.Suite && m.ClaveID == a.cabeceraPDP.ClaveID &&
		m.AudienciaDespliegue == a.cabeceraPDP.Audiencia &&
		m.AlgoritmoCOSE == a.pruebaPDP.algoritmo &&
		m.AudienciaCOSE == a.pruebaPDP.audiencia &&
		m.EstadoConfianza == a.pruebaPDP.estadoConfianza &&
		m.HuellaClaveSHA256 == a.pruebaPDP.huellaClaveSHA256 &&
		m.HuellaPayloadSHA256 == a.pruebaPDP.huellaPayloadSHA256 &&
		m.HuellaSobreSHA256 == a.pruebaPDP.huellaSobreSHA256 &&
		m.VerificadaEn.Equal(a.emitidaEn) &&
		m.RaizValidaDesde.Equal(a.pruebaPDP.raizValidaDesde) &&
		m.RaizValidaHasta.Equal(a.pruebaPDP.raizValidaHasta) &&
		m.RevisionConfianza == a.pruebaPDP.revisionConfianza &&
		m.HuellaConfiguracionSHA256 == a.pruebaPDP.huellaConfiguracionSHA256 &&
		m.ConfiguracionPublicadaEn.Equal(a.pruebaPDP.configuracionPublicadaEn) &&
		m.ConfiguracionExpiraEn.Equal(a.pruebaPDP.configuracionExpiraEn) &&
		m.HuellaContextoRecursoSHA256 == a.evidenciaDurablePDP.metadatos.HuellaContextoRecursoSHA256 &&
		m.HuellaAmbitosRecursoSHA256 == a.evidenciaDurablePDP.metadatos.HuellaAmbitosRecursoSHA256 &&
		m.HuellaPreimagenRecursoSHA256 == a.evidenciaDurablePDP.metadatos.HuellaPreimagenRecursoSHA256
}

func (e EvidenciaDurableAtestacionAutorizacionPDPV4) Metadatos() (
	MetadatosEvidenciaDurableAtestacionPDPV4,
	error,
) {
	if e.Validar() != nil {
		return MetadatosEvidenciaDurableAtestacionPDPV4{}, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	return e.metadatos, nil
}

func (e EvidenciaDurableAtestacionAutorizacionPDPV4) PayloadVECAD1() ([]byte, error) {
	if e.Validar() != nil {
		return nil, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	return append([]byte(nil), e.payloadVECAD1...), nil
}

func (e EvidenciaDurableAtestacionAutorizacionPDPV4) SobreCOSESign1() ([]byte, error) {
	if e.Validar() != nil {
		return nil, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	return append([]byte(nil), e.sobreCOSESign1...), nil
}

func (e EvidenciaDurableAtestacionAutorizacionPDPV4) PreimagenRecursoCanonica() (
	[]byte,
	error,
) {
	if e.Validar() != nil {
		return nil, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	return append([]byte(nil), e.preimagenRecurso...), nil
}

// SerializacionCanonicaParaPersistencia es la unica salida binaria general
// permitida. Es versionada, cerrada y devuelve una copia defensiva. No debe
// enviarse a clientes ni registrarse en logs porque contiene la decision.
func (e EvidenciaDurableAtestacionAutorizacionPDPV4) SerializacionCanonicaParaPersistencia() (
	[]byte,
	error,
) {
	if e.Validar() != nil {
		return nil, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	return append([]byte(nil), e.serializacionCanonica...), nil
}

func (e EvidenciaDurableAtestacionAutorizacionPDPV4) HuellaSHA256() (string, error) {
	if e.Validar() != nil {
		return "", ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	return e.metadatos.HuellaEvidenciaDurableSHA256, nil
}

func (e EvidenciaDurableAtestacionAutorizacionPDPV4) clonar() (
	EvidenciaDurableAtestacionAutorizacionPDPV4,
	error,
) {
	if e.Validar() != nil {
		return EvidenciaDurableAtestacionAutorizacionPDPV4{}, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	e.payloadVECAD1 = append([]byte(nil), e.payloadVECAD1...)
	e.sobreCOSESign1 = append([]byte(nil), e.sobreCOSESign1...)
	e.preimagenRecurso = append([]byte(nil), e.preimagenRecurso...)
	e.serializacionCanonica = append([]byte(nil), e.serializacionCanonica...)
	return e, nil
}

// interpretarEvidenciaHistoricaAtestacionPDPV4 decodifica exclusivamente el
// formato durable cerrado. El resultado sigue siendo historico y NO
// autoritativo: la huella esperada detecta corrupcion accidental, pero quien
// pueda reescribir registro y huella puede recalcular ambos. Antes de cualquier
// efecto se debe verificar COSE contra el catalogo historico autenticado e
// inmutable y cotejar el payload con la decision durable esperada.
//
// Permanece privado para que no aparezca una fabrica general confundible con
// autoridad. Un caso de uso de recuperacion futuro debera envolverlo en un
// Servicio provisto desde el registro historico confiable.
func interpretarEvidenciaHistoricaAtestacionPDPV4(
	serializacion []byte,
	huellaEsperadaSHA256 string,
) (EvidenciaDurableAtestacionAutorizacionPDPV4, error) {
	if len(serializacion) == 0 || len(serializacion) > maximoBytesEvidenciaDurableAtestacionPDPV4 ||
		!huellaSHA256DocumentalValida(huellaEsperadaSHA256) ||
		huellaBytesDocumentales(serializacion) != huellaEsperadaSHA256 {
		return EvidenciaDurableAtestacionAutorizacionPDPV4{}, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	prefijo := append([]byte(esquemaEvidenciaDurableAtestacionPDPV4), 0)
	minimo := len(prefijo) + 2 + 8
	if len(serializacion) < minimo || !bytes.Equal(serializacion[:len(prefijo)], prefijo) ||
		binary.BigEndian.Uint16(serializacion[len(prefijo):len(prefijo)+2]) !=
			versionEvidenciaDurableAtestacionPDPV4 ||
		binary.BigEndian.Uint64(serializacion[len(serializacion)-8:]) != uint64(len(serializacion)) {
		return EvidenciaDurableAtestacionAutorizacionPDPV4{}, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}

	limites := [...]uint64{
		128, 5, 512, 64, 512, 64, 5, 128, 128, 512,
		32, 64, 32, 64, 64, 64, 64, 64, 64, 128,
		64, 64, 64, 64, 64, 64,
		maximoBytesEvidenciaDurableAtestacionPDPV4,
		domain.TamanoMaximoMensajeAtestacionAutorizacionV1,
		domain.TamanoMaximoMensajeAtestacionAutorizacionV1 + margenMaximoSobreCOSEDocumentalV4,
	}
	posicion := len(prefijo) + 2
	finCampos := len(serializacion) - 8
	campos := make([][]byte, 0, len(limites))
	for _, limite := range limites {
		if posicion > finCampos-8 {
			return EvidenciaDurableAtestacionAutorizacionPDPV4{}, ErrEvidenciaDurableAtestacionPDPV4Invalida
		}
		longitud := binary.BigEndian.Uint64(serializacion[posicion : posicion+8])
		posicion += 8
		if longitud > limite || longitud > uint64(finCampos-posicion) {
			return EvidenciaDurableAtestacionAutorizacionPDPV4{}, ErrEvidenciaDurableAtestacionPDPV4Invalida
		}
		final := posicion + int(longitud)
		campos = append(campos, append([]byte(nil), serializacion[posicion:final]...))
		posicion = final
	}
	if posicion != finCampos || len(campos) != len(limites) {
		return EvidenciaDurableAtestacionAutorizacionPDPV4{}, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}

	version, err := interpretarUint16CanonicoEvidenciaAtestacionPDP(campos[1])
	if err != nil {
		return EvidenciaDurableAtestacionAutorizacionPDPV4{}, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	formatoVECAD, err := interpretarUint16CanonicoEvidenciaAtestacionPDP(campos[6])
	if err != nil {
		return EvidenciaDurableAtestacionAutorizacionPDPV4{}, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	instantes := make([]time.Time, 0, 5)
	for _, indice := range []int{16, 17, 18, 21, 22} {
		instante, err := interpretarInstanteCanonicoEvidenciaAtestacionPDP(campos[indice])
		if err != nil {
			return EvidenciaDurableAtestacionAutorizacionPDPV4{}, ErrEvidenciaDurableAtestacionPDPV4Invalida
		}
		instantes = append(instantes, instante)
	}
	evidencia := EvidenciaDurableAtestacionAutorizacionPDPV4{
		marca: marcaEvidenciaDurableAtestacionPDPV4,
		metadatos: MetadatosEvidenciaDurableAtestacionPDPV4{
			Esquema: string(campos[0]), Version: version,
			DecisionRef: string(campos[2]), HuellaPlanSHA256: string(campos[3]),
			EfectoRef: string(campos[4]), HuellaSolicitudVinculadaSHA256: string(campos[5]),
			FormatoVECADVersion: formatoVECAD, Suite: string(campos[7]),
			ClaveID: string(campos[8]), AudienciaDespliegue: string(campos[9]),
			AlgoritmoCOSE:     AlgoritmoCOSEDocumental(campos[10]),
			AudienciaCOSE:     AudienciaCOSEDocumental(campos[11]),
			EstadoConfianza:   EstadoConfianzaClaveDocumental(campos[12]),
			HuellaClaveSHA256: string(campos[13]), HuellaPayloadSHA256: string(campos[14]),
			HuellaSobreSHA256: string(campos[15]), VerificadaEn: instantes[0],
			RaizValidaDesde: instantes[1], RaizValidaHasta: instantes[2],
			RevisionConfianza: string(campos[19]), HuellaConfiguracionSHA256: string(campos[20]),
			ConfiguracionPublicadaEn: instantes[3], ConfiguracionExpiraEn: instantes[4],
			HuellaPreimagenRecursoSHA256: string(campos[23]),
			HuellaContextoRecursoSHA256:  string(campos[24]),
			HuellaAmbitosRecursoSHA256:   string(campos[25]),
			HuellaEvidenciaDurableSHA256: huellaEsperadaSHA256,
		},
		preimagenRecurso:      append([]byte(nil), campos[26]...),
		payloadVECAD1:         append([]byte(nil), campos[27]...),
		sobreCOSESign1:        append([]byte(nil), campos[28]...),
		serializacionCanonica: append([]byte(nil), serializacion...),
	}
	if evidencia.Validar() != nil ||
		!bytes.Equal(evidencia.calcularSerializacionCanonica(), serializacion) {
		return EvidenciaDurableAtestacionAutorizacionPDPV4{}, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	return evidencia, nil
}

func interpretarUint16CanonicoEvidenciaAtestacionPDP(contenido []byte) (uint16, error) {
	valor, err := strconv.ParseUint(string(contenido), 10, 16)
	if err != nil || strconv.FormatUint(valor, 10) != string(contenido) {
		return 0, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	return uint16(valor), nil
}

func interpretarInstanteCanonicoEvidenciaAtestacionPDP(contenido []byte) (time.Time, error) {
	instante, err := time.Parse(time.RFC3339Nano, string(contenido))
	if err != nil || !instanteCanonicoDocumental(instante) ||
		instante.Format(time.RFC3339Nano) != string(contenido) {
		return time.Time{}, ErrEvidenciaDurableAtestacionPDPV4Invalida
	}
	return instante, nil
}

func (e EvidenciaDurableAtestacionAutorizacionPDPV4) calcularSerializacionCanonica() []byte {
	m := e.metadatos
	campos := [][]byte{
		[]byte(m.Esquema), []byte(strconv.FormatUint(uint64(m.Version), 10)),
		[]byte(m.DecisionRef), []byte(m.HuellaPlanSHA256), []byte(m.EfectoRef),
		[]byte(m.HuellaSolicitudVinculadaSHA256),
		[]byte(strconv.FormatUint(uint64(m.FormatoVECADVersion), 10)),
		[]byte(m.Suite), []byte(m.ClaveID), []byte(m.AudienciaDespliegue),
		[]byte(m.AlgoritmoCOSE), []byte(m.AudienciaCOSE), []byte(m.EstadoConfianza),
		[]byte(m.HuellaClaveSHA256),
		[]byte(m.HuellaPayloadSHA256), []byte(m.HuellaSobreSHA256),
		[]byte(m.VerificadaEn.Format(time.RFC3339Nano)),
		[]byte(m.RaizValidaDesde.Format(time.RFC3339Nano)),
		[]byte(m.RaizValidaHasta.Format(time.RFC3339Nano)),
		[]byte(m.RevisionConfianza), []byte(m.HuellaConfiguracionSHA256),
		[]byte(m.ConfiguracionPublicadaEn.Format(time.RFC3339Nano)),
		[]byte(m.ConfiguracionExpiraEn.Format(time.RFC3339Nano)),
		[]byte(m.HuellaPreimagenRecursoSHA256),
		[]byte(m.HuellaContextoRecursoSHA256),
		[]byte(m.HuellaAmbitosRecursoSHA256),
		e.preimagenRecurso, e.payloadVECAD1, e.sobreCOSESign1,
	}
	var destino bytes.Buffer
	destino.WriteString(esquemaEvidenciaDurableAtestacionPDPV4)
	destino.WriteByte(0)
	var version [2]byte
	binary.BigEndian.PutUint16(version[:], versionEvidenciaDurableAtestacionPDPV4)
	destino.Write(version[:])
	for _, campo := range campos {
		escribirCampoBinarioEvidenciaDurableAtestacionPDP(&destino, campo)
	}
	var longitud [8]byte
	binary.BigEndian.PutUint64(longitud[:], uint64(destino.Len()+len(longitud)))
	destino.Write(longitud[:])
	return destino.Bytes()
}

func escribirCampoBinarioEvidenciaDurableAtestacionPDP(destino *bytes.Buffer, campo []byte) {
	var longitud [8]byte
	binary.BigEndian.PutUint64(longitud[:], uint64(len(campo)))
	destino.Write(longitud[:])
	destino.Write(campo)
}

func referenciaDurableAtestacionPDPValida(valor string) bool {
	if valor == "" || len(valor) > 512 || strings.TrimSpace(valor) != valor ||
		!utf8.ValidString(valor) || strings.ContainsRune(valor, '*') {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || unicode.IsSpace(caracter) {
			return false
		}
	}
	return true
}

func (EvidenciaDurableAtestacionAutorizacionPDPV4) String() string {
	return "[EVIDENCIA-DURABLE-ATESTACION-PDP-V4-REDACTADA]"
}
func (e EvidenciaDurableAtestacionAutorizacionPDPV4) GoString() string { return e.String() }
func (e EvidenciaDurableAtestacionAutorizacionPDPV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, e.String())
}
func (e EvidenciaDurableAtestacionAutorizacionPDPV4) LogValue() slog.Value {
	return slog.StringValue(e.String())
}
func (EvidenciaDurableAtestacionAutorizacionPDPV4) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionGeneralEvidenciaDurableAtestacionPDPV4Prohibida
}
func (*EvidenciaDurableAtestacionAutorizacionPDPV4) UnmarshalJSON([]byte) error {
	return ErrSerializacionGeneralEvidenciaDurableAtestacionPDPV4Prohibida
}
func (EvidenciaDurableAtestacionAutorizacionPDPV4) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionGeneralEvidenciaDurableAtestacionPDPV4Prohibida
}
func (*EvidenciaDurableAtestacionAutorizacionPDPV4) UnmarshalText([]byte) error {
	return ErrSerializacionGeneralEvidenciaDurableAtestacionPDPV4Prohibida
}
func (EvidenciaDurableAtestacionAutorizacionPDPV4) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionGeneralEvidenciaDurableAtestacionPDPV4Prohibida
}
func (*EvidenciaDurableAtestacionAutorizacionPDPV4) UnmarshalBinary([]byte) error {
	return ErrSerializacionGeneralEvidenciaDurableAtestacionPDPV4Prohibida
}

func (MetadatosEvidenciaDurableAtestacionPDPV4) String() string {
	return "[METADATOS-EVIDENCIA-DURABLE-ATESTACION-PDP-V4-REDACTADOS]"
}
func (m MetadatosEvidenciaDurableAtestacionPDPV4) GoString() string { return m.String() }
func (m MetadatosEvidenciaDurableAtestacionPDPV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, m.String())
}
func (m MetadatosEvidenciaDurableAtestacionPDPV4) LogValue() slog.Value {
	return slog.StringValue(m.String())
}
func (MetadatosEvidenciaDurableAtestacionPDPV4) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionGeneralEvidenciaDurableAtestacionPDPV4Prohibida
}
func (*MetadatosEvidenciaDurableAtestacionPDPV4) UnmarshalJSON([]byte) error {
	return ErrSerializacionGeneralEvidenciaDurableAtestacionPDPV4Prohibida
}
func (MetadatosEvidenciaDurableAtestacionPDPV4) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionGeneralEvidenciaDurableAtestacionPDPV4Prohibida
}
func (*MetadatosEvidenciaDurableAtestacionPDPV4) UnmarshalText([]byte) error {
	return ErrSerializacionGeneralEvidenciaDurableAtestacionPDPV4Prohibida
}
func (MetadatosEvidenciaDurableAtestacionPDPV4) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionGeneralEvidenciaDurableAtestacionPDPV4Prohibida
}
func (*MetadatosEvidenciaDurableAtestacionPDPV4) UnmarshalBinary([]byte) error {
	return ErrSerializacionGeneralEvidenciaDurableAtestacionPDPV4Prohibida
}
