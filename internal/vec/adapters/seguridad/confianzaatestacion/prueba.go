package confianzaatestacion

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const marcaPruebaConfianzaAtestacionAutorizacionV2 = "vec.prueba-confianza-atestacion-autorizacion.v2"

type datosPruebaConfianzaAtestacionAutorizacionV2 struct {
	ReferenciaDecision          string
	HuellaSolicitudLigadaSHA256 string
	HuellaMotivoCatalogoSHA256  string
	HuellaMensajeSHA256         string
	HuellaSobreSHA256           string
	ClaveID                     string
	HuellaClaveSPKISHA256       string
	AlgoritmoCOSE               string
	Suite                       string
	AudienciaDespliegue         string
	EstadoClave                 EstadoClaveAtestacionAutorizacionV2
	VerificadaEn                time.Time
	RaizValidaDesde             time.Time
	RaizValidaHasta             time.Time
	RevisionConfiguracion       string
	HuellaConfiguracionSHA256   string
	ConfiguracionPublicadaEn    time.Time
	ConfiguracionExpiraEn       time.Time
}

// DatosPruebaConfianzaAtestacionAutorizacionV2 es una copia no serializable de
// compromisos aptos para cotejo durable. No contiene payload, sobre ni clave.
type DatosPruebaConfianzaAtestacionAutorizacionV2 struct {
	bloqueoSerializacion
	ReferenciaDecision          string
	HuellaSolicitudLigadaSHA256 string
	HuellaMotivoCatalogoSHA256  string
	HuellaMensajeSHA256         string
	HuellaSobreSHA256           string
	ClaveID                     string
	HuellaClaveSPKISHA256       string
	AlgoritmoCOSE               string
	Suite                       string
	AudienciaDespliegue         string
	EstadoClave                 EstadoClaveAtestacionAutorizacionV2
	VerificadaEn                time.Time
	RaizValidaDesde             time.Time
	RaizValidaHasta             time.Time
	RevisionConfiguracion       string
	HuellaConfiguracionSHA256   string
	ConfiguracionPublicadaEn    time.Time
	ConfiguracionExpiraEn       time.Time
}

func (d DatosPruebaConfianzaAtestacionAutorizacionV2) Validar() error {
	cabecera := domain.CabeceraAtestacionAutorizacionV2{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV2,
		Suite:          d.Suite,
		ClaveID:        d.ClaveID,
		Audiencia:      d.AudienciaDespliegue,
	}
	if !referenciaPruebaConfianzaValida(d.ReferenciaDecision) ||
		!huellaSHA256ConfianzaValida(d.HuellaSolicitudLigadaSHA256) ||
		!huellaSHA256ConfianzaValida(d.HuellaMotivoCatalogoSHA256) ||
		!huellaSHA256ConfianzaValida(d.HuellaMensajeSHA256) ||
		!huellaSHA256ConfianzaValida(d.HuellaSobreSHA256) ||
		!huellaSHA256ConfianzaValida(d.HuellaClaveSPKISHA256) ||
		d.AlgoritmoCOSE != AlgoritmoCOSEAtestacionAutorizacionV2EdDSA ||
		d.Suite != SuiteAtestacionAutorizacionV2COSEEdDSA ||
		cabecera.Validar() != nil ||
		!audienciaDespliegueAtestacionV2Valida(d.AudienciaDespliegue) ||
		d.EstadoClave != EstadoClaveAtestacionAutorizacionV2Activa ||
		!instanteCanonicoConfianza(d.VerificadaEn) ||
		!instanteCanonicoConfianza(d.RaizValidaDesde) ||
		!instanteCanonicoConfianza(d.RaizValidaHasta) ||
		!d.RaizValidaHasta.After(d.RaizValidaDesde) ||
		d.VerificadaEn.Before(d.RaizValidaDesde) ||
		!d.VerificadaEn.Before(d.RaizValidaHasta) ||
		!referenciaConfiguracionConfianzaValida(d.RevisionConfiguracion) ||
		!huellaSHA256ConfianzaValida(d.HuellaConfiguracionSHA256) ||
		!instanteCanonicoConfianza(d.ConfiguracionPublicadaEn) ||
		!instanteCanonicoConfianza(d.ConfiguracionExpiraEn) ||
		!d.ConfiguracionExpiraEn.After(d.ConfiguracionPublicadaEn) ||
		d.ConfiguracionExpiraEn.Sub(d.ConfiguracionPublicadaEn) >
			maximaVigenciaConfiguracionAtestacionV2 ||
		d.VerificadaEn.Before(d.ConfiguracionPublicadaEn) ||
		!d.VerificadaEn.Before(d.ConfiguracionExpiraEn) {
		return ErrPruebaConfianzaAtestacionV2Invalida
	}
	return nil
}

// PruebaConfianzaAtestacionAutorizacionV2 acredita una comprobacion local
// contra una instantanea fijada. No es autoridad de negocio ni sustituye el
// consumo unico y la revalidacion dentro de PostgreSQL u otro conector.
type PruebaConfianzaAtestacionAutorizacionV2 struct {
	bloqueoSerializacion
	marca string
	datos *datosPruebaConfianzaAtestacionAutorizacionV2
}

func nuevaPruebaConfianzaAtestacionAutorizacionV2(
	datos datosPruebaConfianzaAtestacionAutorizacionV2,
) (PruebaConfianzaAtestacionAutorizacionV2, error) {
	publicos := datos.publicos()
	if publicos.Validar() != nil {
		return PruebaConfianzaAtestacionAutorizacionV2{}, ErrPruebaConfianzaAtestacionV2Invalida
	}
	copia := datos
	return PruebaConfianzaAtestacionAutorizacionV2{
		marca: marcaPruebaConfianzaAtestacionAutorizacionV2,
		datos: &copia,
	}, nil
}

func (p PruebaConfianzaAtestacionAutorizacionV2) Validar() error {
	if p.marca != marcaPruebaConfianzaAtestacionAutorizacionV2 || p.datos == nil ||
		p.datos.publicos().Validar() != nil {
		return ErrPruebaConfianzaAtestacionV2Invalida
	}
	return nil
}

func (p PruebaConfianzaAtestacionAutorizacionV2) Datos() (
	DatosPruebaConfianzaAtestacionAutorizacionV2,
	error,
) {
	if p.Validar() != nil {
		return DatosPruebaConfianzaAtestacionAutorizacionV2{}, ErrPruebaConfianzaAtestacionV2Invalida
	}
	return p.datos.publicos(), nil
}

func (p PruebaConfianzaAtestacionAutorizacionV2) ValidarPara(
	decision domain.DecisionAutorizacion,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
	atestacion ports.AtestacionAutorizacionV2,
) error {
	if p.Validar() != nil || decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2() != nil {
		return ErrPruebaConfianzaAtestacionV2Invalida
	}
	datos := p.datos.publicos()
	if datos.ReferenciaDecision != decision.DecisionRef ||
		datos.HuellaSolicitudLigadaSHA256 != decision.SolicitudHuellaSHA256 ||
		datos.HuellaMotivoCatalogoSHA256 != decision.MotivoHuellaSHA256 ||
		datos.VerificadaEn.Before(decision.EmitidaEn) ||
		!datos.VerificadaEn.Before(decision.ValidaHasta) {
		return ErrPruebaConfianzaAtestacionV2Invalida
	}
	cabecera := domain.CabeceraAtestacionAutorizacionV2{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV2,
		Suite:          datos.Suite,
		ClaveID:        datos.ClaveID,
		Audiencia:      datos.AudienciaDespliegue,
	}
	solicitud, err := ports.NuevaSolicitudFirmaAtestacionAutorizacionV2(
		cabecera,
		decision,
		referenciaMotivo,
	)
	if err != nil || atestacion.ValidarPara(solicitud) != nil {
		return ErrPruebaConfianzaAtestacionV2Invalida
	}
	mensaje, err := solicitud.Mensaje()
	if err != nil {
		return ErrPruebaConfianzaAtestacionV2Invalida
	}
	defer borrarBytesConfianzaAtestacion(mensaje)
	resultado, err := atestacion.Resultado()
	if err != nil {
		return ErrPruebaConfianzaAtestacionV2Invalida
	}
	sobre, err := resultado.Firma()
	if err != nil {
		return ErrPruebaConfianzaAtestacionV2Invalida
	}
	defer borrarBytesConfianzaAtestacion(sobre)
	if datos.HuellaMensajeSHA256 != huellaBytesConfianzaAtestacion(mensaje) ||
		datos.HuellaSobreSHA256 != huellaBytesConfianzaAtestacion(sobre) {
		return ErrPruebaConfianzaAtestacionV2Invalida
	}
	return nil
}

func (d datosPruebaConfianzaAtestacionAutorizacionV2) publicos() (
	resultado DatosPruebaConfianzaAtestacionAutorizacionV2,
) {
	resultado.ReferenciaDecision = d.ReferenciaDecision
	resultado.HuellaSolicitudLigadaSHA256 = d.HuellaSolicitudLigadaSHA256
	resultado.HuellaMotivoCatalogoSHA256 = d.HuellaMotivoCatalogoSHA256
	resultado.HuellaMensajeSHA256 = d.HuellaMensajeSHA256
	resultado.HuellaSobreSHA256 = d.HuellaSobreSHA256
	resultado.ClaveID = d.ClaveID
	resultado.HuellaClaveSPKISHA256 = d.HuellaClaveSPKISHA256
	resultado.AlgoritmoCOSE = d.AlgoritmoCOSE
	resultado.Suite = d.Suite
	resultado.AudienciaDespliegue = d.AudienciaDespliegue
	resultado.EstadoClave = d.EstadoClave
	resultado.VerificadaEn = d.VerificadaEn
	resultado.RaizValidaDesde = d.RaizValidaDesde
	resultado.RaizValidaHasta = d.RaizValidaHasta
	resultado.RevisionConfiguracion = d.RevisionConfiguracion
	resultado.HuellaConfiguracionSHA256 = d.HuellaConfiguracionSHA256
	resultado.ConfiguracionPublicadaEn = d.ConfiguracionPublicadaEn
	resultado.ConfiguracionExpiraEn = d.ConfiguracionExpiraEn
	return resultado
}

func referenciaPruebaConfianzaValida(valor string) bool {
	return textoReferenciaConfianzaValido(valor, 512)
}

func huellaBytesConfianzaAtestacion(contenido []byte) string {
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:])
}
