package cobertura

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const dominioHuellaPoliticaActuacionCobertura = "vec.dipgra." +
	"contratacion-temporal.politica-actuacion-cobertura"

// CanonHuellaPoliticaActuacionCobertura identifica el formato cerrado de la
// política que gobierna una actuación de cobertura.
type CanonHuellaPoliticaActuacionCobertura struct {
	Dominio        string `json:"dominio"`
	VersionEsquema uint16 `json:"version_esquema"`
	Algoritmo      string `json:"algoritmo"`
}

func CanonHuellaPoliticaActuacionCoberturaV1() CanonHuellaPoliticaActuacionCobertura {
	return CanonHuellaPoliticaActuacionCobertura{
		Dominio:        dominioHuellaPoliticaActuacionCobertura,
		VersionEsquema: 1,
		Algoritmo:      "sha-256",
	}
}

func (c CanonHuellaPoliticaActuacionCobertura) valida() bool {
	return c == CanonHuellaPoliticaActuacionCoberturaV1()
}

// PublicacionPoliticaActuacionCobertura es configuración durable y
// transportable. Su SHA-256 aporta integridad nominal, no procedencia ni
// autorización: el resolutor/DB es parte del TCB y O4-04 debe volver a
// bloquear y revalidar la publicación antes de producir efectos.
type PublicacionPoliticaActuacionCobertura struct {
	Referencia                   string                                    `json:"referencia"`
	Version                      uint64                                    `json:"version"`
	HuellaSHA256                 string                                    `json:"huella_sha256"`
	Canon                        CanonHuellaPoliticaActuacionCobertura     `json:"canon"`
	OrganizacionRef              string                                    `json:"organizacion_ref"`
	Accion                       domain.ClaveCatalogo                      `json:"accion"`
	Catalogo                     domain.IdentidadCatalogoViasCobertura     `json:"catalogo"`
	Politica                     domain.IdentidadPoliticaDecisionCobertura `json:"politica"`
	FinalidadContratacionClave   domain.ClaveCatalogo                      `json:"finalidad_contratacion_clave"`
	FinalidadContratacionRef     string                                    `json:"finalidad_contratacion_ref"`
	FinalidadAutorizacionVEC     domain.ClaveCatalogo                      `json:"finalidad_autorizacion_vec"`
	UnidadEjecutoraRef           string                                    `json:"unidad_ejecutora_ref"`
	FaseDestino                  domain.ClaveFase                          `json:"fase_destino"`
	EstadoDestino                domain.EstadoOperativo                    `json:"estado_destino"`
	MotivoAutorizacionDecidir    dominiovec.ReferenciaEntradaCatalogo      `json:"motivo_autorizacion_decidir"`
	MotivoAutorizacionRectificar dominiovec.ReferenciaEntradaCatalogo      `json:"motivo_autorizacion_rectificar"`
	EquivalenciaMotivosRef       string                                    `json:"equivalencia_motivos_ref,omitempty"`
	PublicadaEn                  time.Time                                 `json:"publicada_en"`
	Vigencia                     domain.VigenciaCatalogoCobertura          `json:"vigencia"`
}

func (p PublicacionPoliticaActuacionCobertura) Validar() error {
	huella, err := CalcularHuellaSHA256PoliticaActuacionCobertura(p)
	if err != nil ||
		!huellaSHA256OperacionDecisionCoberturaValida(p.HuellaSHA256) ||
		!referenciasOperacionDecisionCoberturaIguales(
			huella,
			p.HuellaSHA256,
		) {
		return ErrGobiernoOperacionCoberturaNoConfiable
	}
	return nil
}

// CalcularHuellaSHA256PoliticaActuacionCobertura permite al publicador
// autorizado materializar el resumen. No convierte datos de un canal en
// autoridad; solo ObtenerGobierno... puede crear la capacidad efímera.
func CalcularHuellaSHA256PoliticaActuacionCobertura(
	publicacion PublicacionPoliticaActuacionCobertura,
) (string, error) {
	material, err := representacionCanonicaPoliticaActuacionCobertura(
		publicacion,
	)
	if err != nil {
		return "", ErrGobiernoOperacionCoberturaNoConfiable
	}
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:]), nil
}

func representacionCanonicaPoliticaActuacionCobertura(
	p PublicacionPoliticaActuacionCobertura,
) ([]byte, error) {
	if !contenidoPoliticaActuacionCoberturaValido(p) {
		return nil, ErrGobiernoOperacionCoberturaNoConfiable
	}
	canon := nuevoCanonOperacionDecisionCobertura()
	canon.texto(p.Canon.Dominio)
	canon.entero(uint64(p.Canon.VersionEsquema))
	canon.texto(p.Canon.Algoritmo)
	canon.texto(p.Referencia)
	canon.entero(p.Version)
	canon.texto(p.OrganizacionRef)
	canon.texto(string(p.Accion))
	escribirIdentidadCatalogoPoliticaActuacion(canon, p.Catalogo)
	escribirIdentidadDecisionPoliticaActuacion(canon, p.Politica)
	canon.texto(string(p.FinalidadContratacionClave))
	canon.texto(p.FinalidadContratacionRef)
	canon.texto(string(p.FinalidadAutorizacionVEC))
	canon.texto(p.UnidadEjecutoraRef)
	canon.texto(string(p.FaseDestino))
	canon.texto(string(p.EstadoDestino))
	escribirMotivoPoliticaActuacion(
		canon,
		p.MotivoAutorizacionDecidir,
	)
	escribirMotivoPoliticaActuacion(
		canon,
		p.MotivoAutorizacionRectificar,
	)
	canon.texto(p.EquivalenciaMotivosRef)
	canon.texto(p.PublicadaEn.Format(time.RFC3339Nano))
	canon.texto(p.Vigencia.Desde.Format(time.RFC3339Nano))
	if p.Vigencia.Hasta.IsZero() {
		canon.texto("")
	} else {
		canon.texto(p.Vigencia.Hasta.Format(time.RFC3339Nano))
	}
	return canon.resultado()
}

func contenidoPoliticaActuacionCoberturaValido(
	p PublicacionPoliticaActuacionCobertura,
) bool {
	motivosIguales := p.MotivoAutorizacionDecidir ==
		p.MotivoAutorizacionRectificar
	equivalenciaDeclarada := p.EquivalenciaMotivosRef != ""
	return p.Canon.valida() &&
		domain.ReferenciaOpacaValida(p.Referencia) &&
		p.Version > 0 &&
		p.Version <= MaximoEnteroSeguroOperacionDecisionCobertura &&
		domain.ReferenciaOpacaValida(p.OrganizacionRef) &&
		accionGobiernoOperacionCoberturaValida(p.Accion) &&
		p.Catalogo.Validar() == nil &&
		p.Politica.Validar() == nil &&
		p.FinalidadContratacionClave.Valida() &&
		domain.ReferenciaOpacaValida(p.FinalidadContratacionRef) &&
		p.FinalidadAutorizacionVEC.Valida() &&
		domain.ReferenciaOpacaValida(p.UnidadEjecutoraRef) &&
		p.FaseDestino.Valida() &&
		p.EstadoDestino.Valido() &&
		dominiovec.ReferenciaMotivoAutorizacionV2Valida(
			p.MotivoAutorizacionDecidir,
		) &&
		dominiovec.ReferenciaMotivoAutorizacionV2Valida(
			p.MotivoAutorizacionRectificar,
		) &&
		motivosIguales == equivalenciaDeclarada &&
		(!equivalenciaDeclarada ||
			domain.ReferenciaOpacaValida(p.EquivalenciaMotivosRef)) &&
		instanteOperacionDecisionCoberturaValido(p.PublicadaEn) &&
		p.Vigencia.Validar() == nil &&
		!p.Vigencia.Hasta.IsZero() &&
		!p.PublicadaEn.After(p.Vigencia.Desde)
}

func accionGobiernoOperacionCoberturaValida(
	accion domain.ClaveCatalogo,
) bool {
	return accion == domain.AccionDecidirCoberturaGobernada ||
		accion == domain.AccionRectificarCoberturaGobernada
}

func escribirIdentidadCatalogoPoliticaActuacion(
	canon *canonOperacionDecisionCobertura,
	identidad domain.IdentidadCatalogoViasCobertura,
) {
	canon.texto(identidad.Referencia)
	canon.entero(identidad.Version)
	canon.texto(identidad.HuellaSHA256)
}

func escribirIdentidadDecisionPoliticaActuacion(
	canon *canonOperacionDecisionCobertura,
	identidad domain.IdentidadPoliticaDecisionCobertura,
) {
	canon.texto(identidad.Referencia)
	canon.entero(identidad.Version)
	canon.texto(identidad.HuellaSHA256)
}

func escribirMotivoPoliticaActuacion(
	canon *canonOperacionDecisionCobertura,
	motivo dominiovec.ReferenciaEntradaCatalogo,
) {
	canon.texto(motivo.CatalogoID)
	canon.entero(uint64(motivo.CatalogoVersion))
	canon.texto(motivo.CatalogoHuellaSHA256)
	canon.texto(motivo.EntradaClave)
}
