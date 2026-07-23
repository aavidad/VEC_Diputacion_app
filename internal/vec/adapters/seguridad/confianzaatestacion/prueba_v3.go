package confianzaatestacion

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const marcaPruebaConfianzaAtestacionAutorizacionV3 = "vec.prueba-confianza-atestacion-autorizacion.v3"

type datosPruebaConfianzaAtestacionAutorizacionV3 struct {
	ReferenciaDecision        string
	HuellaDecisionSHA256      string
	HuellaMotivoSHA256        string
	ReferenciaContexto        string
	HuellaContextoSHA256      string
	HuellaMensajeSHA256       string
	HuellaSobreSHA256         string
	ClaveID                   string
	RaizVersion               uint64
	HuellaClaveSPKISHA256     string
	AlgoritmoCOSE             string
	Suite                     string
	AudienciaDespliegue       string
	EstadoClave               EstadoClaveAtestacionAutorizacionV3
	VerificadaEn              time.Time
	RaizValidaDesde           time.Time
	RaizValidaHasta           time.Time
	RevisionConfiguracion     string
	SecuenciaConfiguracion    uint64
	HuellaConfiguracionSHA256 string
	ConfiguracionPublicadaEn  time.Time
	ConfiguracionExpiraEn     time.Time
	huellaPruebaSHA256        string
}

// DatosPruebaConfianzaAtestacionAutorizacionV3 es una copia minimizada y no
// serializable. No contiene el payload, el sobre ni material de claves.
type DatosPruebaConfianzaAtestacionAutorizacionV3 struct {
	bloqueoSerializacionV3
	ReferenciaDecision        string
	HuellaDecisionSHA256      string
	HuellaMotivoSHA256        string
	ReferenciaContexto        string
	HuellaContextoSHA256      string
	HuellaMensajeSHA256       string
	HuellaSobreSHA256         string
	ClaveID                   string
	RaizVersion               uint64
	HuellaClaveSPKISHA256     string
	AlgoritmoCOSE             string
	Suite                     string
	AudienciaDespliegue       string
	EstadoClave               EstadoClaveAtestacionAutorizacionV3
	VerificadaEn              time.Time
	RaizValidaDesde           time.Time
	RaizValidaHasta           time.Time
	RevisionConfiguracion     string
	SecuenciaConfiguracion    uint64
	HuellaConfiguracionSHA256 string
	ConfiguracionPublicadaEn  time.Time
	ConfiguracionExpiraEn     time.Time
	HuellaPruebaSHA256        string
}

func (d DatosPruebaConfianzaAtestacionAutorizacionV3) Validar() error {
	cabecera := domain.CabeceraAtestacionAutorizacionV3{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3,
		Suite:          d.Suite, ClaveID: d.ClaveID, Audiencia: d.AudienciaDespliegue,
	}
	if !referenciaPruebaConfianzaValida(d.ReferenciaDecision) ||
		!huellaSHA256ConfianzaValida(d.HuellaDecisionSHA256) ||
		!huellaSHA256ConfianzaValida(d.HuellaMotivoSHA256) ||
		!referenciaPruebaConfianzaValida(d.ReferenciaContexto) ||
		!huellaSHA256ConfianzaValida(d.HuellaContextoSHA256) ||
		!huellaSHA256ConfianzaValida(d.HuellaMensajeSHA256) ||
		!huellaSHA256ConfianzaValida(d.HuellaSobreSHA256) ||
		d.RaizVersion == 0 ||
		!huellaSHA256ConfianzaValida(d.HuellaClaveSPKISHA256) ||
		d.AlgoritmoCOSE != AlgoritmoCOSEAtestacionAutorizacionV3EdDSA ||
		d.Suite != SuiteAtestacionAutorizacionV3COSEEdDSA ||
		cabecera.Validar() != nil ||
		d.EstadoClave != EstadoClaveAtestacionAutorizacionV3Activa ||
		!instanteCanonicoConfianza(d.VerificadaEn) ||
		!instanteCanonicoConfianza(d.RaizValidaDesde) ||
		!instanteCanonicoConfianza(d.RaizValidaHasta) ||
		!d.RaizValidaHasta.After(d.RaizValidaDesde) ||
		d.VerificadaEn.Before(d.RaizValidaDesde) ||
		!d.VerificadaEn.Before(d.RaizValidaHasta) ||
		!referenciaConfiguracionConfianzaValida(d.RevisionConfiguracion) ||
		d.SecuenciaConfiguracion == 0 ||
		!huellaSHA256ConfianzaValida(d.HuellaConfiguracionSHA256) ||
		!instanteCanonicoConfianza(d.ConfiguracionPublicadaEn) ||
		!instanteCanonicoConfianza(d.ConfiguracionExpiraEn) ||
		!d.ConfiguracionExpiraEn.After(d.ConfiguracionPublicadaEn) ||
		d.ConfiguracionExpiraEn.Sub(d.ConfiguracionPublicadaEn) >
			maximaVigenciaConfiguracionAtestacionV3 ||
		d.VerificadaEn.Before(d.ConfiguracionPublicadaEn) ||
		!d.VerificadaEn.Before(d.ConfiguracionExpiraEn) ||
		!huellaSHA256ConfianzaValida(d.HuellaPruebaSHA256) {
		return ErrPruebaConfianzaAtestacionV3Invalida
	}
	esperada := calcularHuellaPruebaConfianzaAtestacionV3(d)
	if !huellasConfianzaIguales(esperada, d.HuellaPruebaSHA256) {
		return ErrPruebaConfianzaAtestacionV3Invalida
	}
	return nil
}

// PruebaConfianzaAtestacionAutorizacionV3 acredita COSE y la instantanea de
// confianza. No es una capacidad de efecto; el emisor HMAC separado debe
// volver a validarla y acuñar una credencial de cinco segundos.
type PruebaConfianzaAtestacionAutorizacionV3 struct {
	bloqueoSerializacionV3
	marca string
	datos *datosPruebaConfianzaAtestacionAutorizacionV3
}

func nuevaPruebaConfianzaAtestacionAutorizacionV3(
	datos datosPruebaConfianzaAtestacionAutorizacionV3,
) (PruebaConfianzaAtestacionAutorizacionV3, error) {
	publicos := datos.publicos()
	publicos.HuellaPruebaSHA256 = calcularHuellaPruebaConfianzaAtestacionV3(publicos)
	datos.huellaPruebaSHA256 = publicos.HuellaPruebaSHA256
	if publicos.Validar() != nil {
		return PruebaConfianzaAtestacionAutorizacionV3{},
			ErrPruebaConfianzaAtestacionV3Invalida
	}
	copia := datos
	return PruebaConfianzaAtestacionAutorizacionV3{
		marca: marcaPruebaConfianzaAtestacionAutorizacionV3,
		datos: &copia,
	}, nil
}

func (p PruebaConfianzaAtestacionAutorizacionV3) Validar() error {
	if p.marca != marcaPruebaConfianzaAtestacionAutorizacionV3 ||
		p.datos == nil || p.datos.publicos().Validar() != nil {
		return ErrPruebaConfianzaAtestacionV3Invalida
	}
	return nil
}

func (p PruebaConfianzaAtestacionAutorizacionV3) Datos() (
	DatosPruebaConfianzaAtestacionAutorizacionV3,
	error,
) {
	if p.Validar() != nil {
		return DatosPruebaConfianzaAtestacionAutorizacionV3{},
			ErrPruebaConfianzaAtestacionV3Invalida
	}
	return p.datos.publicos(), nil
}

// ExportacionCanonicaParaConsumidor entrega la evidencia cerrada cuyos bytes
// compromete HuellaPruebaSHA256. Es la única serialización autorizada para
// cruzar hacia el consumidor durable; Datos sigue bloqueado para codecs.
func (p PruebaConfianzaAtestacionAutorizacionV3) ExportacionCanonicaParaConsumidor() (
	[]byte,
	error,
) {
	if p.Validar() != nil {
		return nil, ErrPruebaConfianzaAtestacionV3Invalida
	}
	contenido := representacionCanonicaPruebaConfianzaAtestacionV3(
		p.datos.publicos(),
	)
	suma := sha256.Sum256(contenido)
	if !huellasConfianzaIguales(
		hex.EncodeToString(suma[:]),
		p.datos.huellaPruebaSHA256,
	) {
		return nil, ErrPruebaConfianzaAtestacionV3Invalida
	}
	return contenido, nil
}

func (p PruebaConfianzaAtestacionAutorizacionV3) ValidarPara(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	decision domain.DecisionAutorizacionLigadaV3,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
	resultadoContexto domain.ResultadoContextoActorRegistradoV2,
	atestacion ports.AtestacionAutorizacionV3,
) error {
	if p.Validar() != nil || decision.ValidarPara(solicitud) != nil ||
		resultadoContexto.Validar() != nil {
		return ErrPruebaConfianzaAtestacionV3Invalida
	}
	datosSolicitud, err := solicitud.Datos()
	emitidaEn, validaHasta, errVentana := decision.VentanaValidez()
	if err != nil || errVentana != nil ||
		datosSolicitud.ReferenciaMotivo != referenciaMotivo ||
		datosSolicitud.VinculoAutenticacionActor.ValidarPara(resultadoContexto) != nil ||
		p.datos.ReferenciaContexto != resultadoContexto.RegistroContextoRef ||
		!huellasConfianzaIguales(
			p.datos.HuellaContextoSHA256,
			resultadoContexto.HuellaSHA256,
		) ||
		p.datos.VerificadaEn.Before(emitidaEn) ||
		!p.datos.VerificadaEn.Before(validaHasta) {
		return ErrPruebaConfianzaAtestacionV3Invalida
	}
	huellaDecision, errDecision := domain.HuellaSHA256DecisionAutorizacionV3(decision)
	huellaMotivo, errMotivo := domain.HuellaSHA256MotivoAutorizacionV2(referenciaMotivo)
	if errDecision != nil || errMotivo != nil ||
		!huellasConfianzaIguales(p.datos.HuellaDecisionSHA256, huellaDecision) ||
		!huellasConfianzaIguales(p.datos.HuellaMotivoSHA256, huellaMotivo) {
		return ErrPruebaConfianzaAtestacionV3Invalida
	}
	cabecera := domain.CabeceraAtestacionAutorizacionV3{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3,
		Suite:          p.datos.Suite, ClaveID: p.datos.ClaveID,
		Audiencia: p.datos.AudienciaDespliegue,
	}
	esperada, err := ports.NuevaSolicitudFirmaAtestacionAutorizacionV3(
		cabecera, decision, referenciaMotivo, resultadoContexto,
	)
	if err != nil || atestacion.ValidarPara(esperada) != nil {
		return ErrPruebaConfianzaAtestacionV3Invalida
	}
	mensaje, errMensaje := esperada.Mensaje()
	resultado, errResultado := atestacion.Resultado()
	if errMensaje != nil || errResultado != nil {
		return ErrPruebaConfianzaAtestacionV3Invalida
	}
	defer borrarBytesConfianzaAtestacion(mensaje)
	sobre, err := resultado.Firma()
	if err != nil {
		return ErrPruebaConfianzaAtestacionV3Invalida
	}
	defer borrarBytesConfianzaAtestacion(sobre)
	if !huellasConfianzaIguales(
		p.datos.HuellaMensajeSHA256,
		huellaBytesConfianzaAtestacion(mensaje),
	) || !huellasConfianzaIguales(
		p.datos.HuellaSobreSHA256,
		huellaBytesConfianzaAtestacion(sobre),
	) {
		return ErrPruebaConfianzaAtestacionV3Invalida
	}
	return nil
}

func (d datosPruebaConfianzaAtestacionAutorizacionV3) publicos() (
	resultado DatosPruebaConfianzaAtestacionAutorizacionV3,
) {
	resultado.ReferenciaDecision = d.ReferenciaDecision
	resultado.HuellaDecisionSHA256 = d.HuellaDecisionSHA256
	resultado.HuellaMotivoSHA256 = d.HuellaMotivoSHA256
	resultado.ReferenciaContexto = d.ReferenciaContexto
	resultado.HuellaContextoSHA256 = d.HuellaContextoSHA256
	resultado.HuellaMensajeSHA256 = d.HuellaMensajeSHA256
	resultado.HuellaSobreSHA256 = d.HuellaSobreSHA256
	resultado.ClaveID = d.ClaveID
	resultado.RaizVersion = d.RaizVersion
	resultado.HuellaClaveSPKISHA256 = d.HuellaClaveSPKISHA256
	resultado.AlgoritmoCOSE = d.AlgoritmoCOSE
	resultado.Suite = d.Suite
	resultado.AudienciaDespliegue = d.AudienciaDespliegue
	resultado.EstadoClave = d.EstadoClave
	resultado.VerificadaEn = d.VerificadaEn
	resultado.RaizValidaDesde = d.RaizValidaDesde
	resultado.RaizValidaHasta = d.RaizValidaHasta
	resultado.RevisionConfiguracion = d.RevisionConfiguracion
	resultado.SecuenciaConfiguracion = d.SecuenciaConfiguracion
	resultado.HuellaConfiguracionSHA256 = d.HuellaConfiguracionSHA256
	resultado.ConfiguracionPublicadaEn = d.ConfiguracionPublicadaEn
	resultado.ConfiguracionExpiraEn = d.ConfiguracionExpiraEn
	resultado.HuellaPruebaSHA256 = d.huellaPruebaSHA256
	return resultado
}

func calcularHuellaPruebaConfianzaAtestacionV3(
	d DatosPruebaConfianzaAtestacionAutorizacionV3,
) string {
	suma := sha256.Sum256(representacionCanonicaPruebaConfianzaAtestacionV3(d))
	return hex.EncodeToString(suma[:])
}

func representacionCanonicaPruebaConfianzaAtestacionV3(
	d DatosPruebaConfianzaAtestacionAutorizacionV3,
) []byte {
	var salida bytes.Buffer
	for _, campo := range []string{
		"vec.prueba-confianza-atestacion-autorizacion.v3",
		d.ReferenciaDecision, d.HuellaDecisionSHA256, d.HuellaMotivoSHA256,
		d.ReferenciaContexto, d.HuellaContextoSHA256, d.HuellaMensajeSHA256,
		d.HuellaSobreSHA256, d.ClaveID, d.HuellaClaveSPKISHA256,
		strconv.FormatUint(d.RaizVersion, 10),
		d.AlgoritmoCOSE, d.Suite, d.AudienciaDespliegue, string(d.EstadoClave),
		d.VerificadaEn.Format(time.RFC3339Nano),
		d.RaizValidaDesde.Format(time.RFC3339Nano),
		d.RaizValidaHasta.Format(time.RFC3339Nano),
		d.RevisionConfiguracion,
		strconv.FormatUint(d.SecuenciaConfiguracion, 10),
		d.HuellaConfiguracionSHA256,
		d.ConfiguracionPublicadaEn.Format(time.RFC3339Nano),
		d.ConfiguracionExpiraEn.Format(time.RFC3339Nano),
	} {
		var longitud [8]byte
		binary.BigEndian.PutUint64(longitud[:], uint64(len([]byte(campo))))
		_, _ = salida.Write(longitud[:])
		_, _ = salida.WriteString(campo)
	}
	return salida.Bytes()
}

func huellasConfianzaIguales(primera, segunda string) bool {
	if !huellaSHA256ConfianzaValida(primera) ||
		!huellaSHA256ConfianzaValida(segunda) {
		return false
	}
	a, errA := hex.DecodeString(primera)
	b, errB := hex.DecodeString(segunda)
	return errA == nil && errB == nil &&
		subtle.ConstantTimeCompare(a, b) == 1
}
