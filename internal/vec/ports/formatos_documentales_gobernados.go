package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	documentalcanonico "vec-diputacion-granada/internal/vec/canonico/documental"
	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrConsultaFormatoDocumentalInvalida        = errors.New("vec: consulta de formato documental invalida")
	ErrDescriptorFormatoDocumentalInvalido      = errors.New("vec: descriptor de formato documental invalido")
	ErrCatalogoFormatosDocumentalesNoDisponible = errors.New("vec: catalogo de formatos documentales no disponible")
	ErrFormatoDocumentalNoResuelto              = errors.New("vec: formato documental no resuelto")
	ErrRenderizadorDocumentalNoDisponible       = errors.New("vec: renderizador documental no disponible")
	ErrMetadatoInstitucionalDocumentalInvalido  = errors.New("vec: metadato institucional documental invalido")
	ErrSituacionOperativaDocumentalInvalida     = errors.New("vec: situacion operativa documental invalida")
	ErrComponenteDocumentalAtestadoInvalido     = errors.New("vec: componente documental atestado invalido")
	ErrPoliticaInstitucionalDocumentalInvalida  = errors.New("vec: politica institucional documental invalida")
)

type escanerReferenciaDescriptorDocumental struct{}

func (escanerReferenciaDescriptorDocumental) MatchString(valor string) bool {
	return documentalcanonico.ReferenciaASCIIBasicaValida(valor)
}

var referenciaDescriptorDocumentalValida escanerReferenciaDescriptorDocumental

// ConsultaFormatoDocumental fija todos los ejes gobernados. El digest del
// perfil impide reutilizar el mismo ID/version con otra especificacion y la
// revision incluye numero y huella de la instantanea completa del catalogo.
type ConsultaFormatoDocumental struct {
	Identidad          domain.IdentidadSintacticaDocumental
	PerfilRef          domain.ReferenciaPerfilDocumental
	DigestPerfilSHA256 string
	RevisionCatalogo   domain.RevisionCatalogoFormatosDocumentales
}

func (c ConsultaFormatoDocumental) Validar() error {
	if c.Identidad.Validar() != nil || c.PerfilRef.Validar() != nil ||
		!huellaSHA256FormatoDocumentalValida(c.DigestPerfilSHA256) ||
		c.RevisionCatalogo.Validar() != nil {
		return ErrConsultaFormatoDocumentalInvalida
	}
	return nil
}

// DescriptorFormatoDocumental no contiene codigo, comandos, rutas, URL ni
// configuracion libre. Vincula un perfil inmutable a una revision concreta y
// a una unica version de conector homologado.
type DescriptorFormatoDocumental struct {
	referencia string
	perfil     domain.PerfilFormatoDocumental
	revision   domain.RevisionCatalogoFormatosDocumentales
	conector   domain.ReferenciaConectorDocumental
}

func NuevoDescriptorFormatoDocumental(
	referencia string,
	perfil domain.PerfilFormatoDocumental,
	revision domain.RevisionCatalogoFormatosDocumentales,
	conector domain.ReferenciaConectorDocumental,
) (DescriptorFormatoDocumental, error) {
	descriptor := DescriptorFormatoDocumental{
		referencia: referencia, perfil: perfil, revision: revision, conector: conector,
	}
	if descriptor.Validar() != nil {
		return DescriptorFormatoDocumental{}, ErrDescriptorFormatoDocumentalInvalido
	}
	return descriptor, nil
}

func (d DescriptorFormatoDocumental) Validar() error {
	if !referenciaDescriptorDocumentalValida.MatchString(d.referencia) ||
		strings.ContainsRune(d.referencia, '*') || d.perfil.Validar() != nil ||
		d.revision.Validar() != nil || d.conector.Validar() != nil {
		return ErrDescriptorFormatoDocumentalInvalido
	}
	return nil
}

func (d DescriptorFormatoDocumental) Referencia() string                     { return d.referencia }
func (d DescriptorFormatoDocumental) Perfil() domain.PerfilFormatoDocumental { return d.perfil }
func (d DescriptorFormatoDocumental) Revision() domain.RevisionCatalogoFormatosDocumentales {
	return d.revision
}
func (d DescriptorFormatoDocumental) Conector() domain.ReferenciaConectorDocumental {
	return d.conector
}

func (d DescriptorFormatoDocumental) Coincide(c ConsultaFormatoDocumental) bool {
	return d.Validar() == nil && c.Validar() == nil &&
		d.perfil.Identidad() == c.Identidad && d.perfil.Referencia() == c.PerfilRef &&
		d.perfil.DigestSHA256() == c.DigestPerfilSHA256 && d.revision == c.RevisionCatalogo
}

// CatalogoFormatosDocumentales devuelve todas las coincidencias, nunca "la
// primera". El servicio de aplicacion exige cardinalidad exactamente uno para
// detectar duplicados o fuentes contradictorias.
type CatalogoFormatosDocumentales interface {
	BuscarDescriptoresFormatoDocumental(
		context.Context,
		ConsultaFormatoDocumental,
	) ([]DescriptorFormatoDocumental, error)
}

// DescriptorPerfilDocumental es la declaracion de catalogo V2. No contiene
// ejecutores ni componentes: solo enlaza el perfil inmutable con la
// publicacion operativa que debe releerse antes de cada efecto.
type DescriptorPerfilDocumental struct {
	referencia     string
	publicacionRef string
	perfil         domain.PerfilFormatoDocumental
	revision       domain.RevisionCatalogoFormatosDocumentales
}

func NuevoDescriptorPerfilDocumental(
	referencia, publicacionRef string,
	perfil domain.PerfilFormatoDocumental,
	revision domain.RevisionCatalogoFormatosDocumentales,
) (DescriptorPerfilDocumental, error) {
	descriptor := DescriptorPerfilDocumental{
		referencia: referencia, publicacionRef: publicacionRef,
		perfil: perfil, revision: revision,
	}
	if descriptor.Validar() != nil {
		return DescriptorPerfilDocumental{}, ErrDescriptorFormatoDocumentalInvalido
	}
	return descriptor, nil
}

func (d DescriptorPerfilDocumental) Validar() error {
	if !referenciaDescriptorDocumentalValida.MatchString(d.referencia) ||
		!referenciaDescriptorDocumentalValida.MatchString(d.publicacionRef) ||
		strings.ContainsRune(d.referencia, '*') || strings.ContainsRune(d.publicacionRef, '*') ||
		d.referencia == d.publicacionRef || d.perfil.Validar() != nil || d.revision.Validar() != nil {
		return ErrDescriptorFormatoDocumentalInvalido
	}
	return nil
}

func (d DescriptorPerfilDocumental) Referencia() string                     { return d.referencia }
func (d DescriptorPerfilDocumental) PublicacionRef() string                 { return d.publicacionRef }
func (d DescriptorPerfilDocumental) Perfil() domain.PerfilFormatoDocumental { return d.perfil }
func (d DescriptorPerfilDocumental) Revision() domain.RevisionCatalogoFormatosDocumentales {
	return d.revision
}

func (d DescriptorPerfilDocumental) Coincide(c ConsultaFormatoDocumental) bool {
	return d.Validar() == nil && c.Validar() == nil &&
		d.perfil.Identidad() == c.Identidad && d.perfil.Referencia() == c.PerfilRef &&
		d.perfil.DigestSHA256() == c.DigestPerfilSHA256 && d.revision == c.RevisionCatalogo
}

type CatalogoPerfilesDocumentales interface {
	BuscarDescriptoresPerfilDocumental(
		context.Context,
		ConsultaFormatoDocumental,
	) ([]DescriptorPerfilDocumental, error)
}

// ConsultaSituacionOperativaActual obliga al registro a devolver la
// proyeccion actual exacta. Una entrada historica no satisface el contrato.
type ConsultaSituacionOperativaActual struct {
	PublicacionRef   string
	PerfilRef        domain.ReferenciaPerfilDocumental
	DigestPerfil     string
	RevisionCatalogo domain.RevisionCatalogoFormatosDocumentales
}

func (c ConsultaSituacionOperativaActual) Validar() error {
	if !referenciaDescriptorDocumentalValida.MatchString(c.PublicacionRef) ||
		strings.ContainsRune(c.PublicacionRef, '*') || c.PerfilRef.Validar() != nil ||
		!huellaSHA256FormatoDocumentalValida(c.DigestPerfil) || c.RevisionCatalogo.Validar() != nil {
		return ErrSituacionOperativaDocumentalInvalida
	}
	return nil
}

func (c ConsultaSituacionOperativaActual) Coincide(
	situacion domain.SituacionOperativaPerfilDocumental,
) bool {
	return c.Validar() == nil && situacion.Validar() == nil &&
		situacion.PublicacionRef() == c.PublicacionRef && situacion.PerfilRef() == c.PerfilRef &&
		situacion.DigestPerfilSHA256() == c.DigestPerfil &&
		situacion.RevisionCatalogo() == c.RevisionCatalogo
}

type RegistroSituacionesOperativasPerfilDocumental interface {
	// Debe consultar la proyeccion vigente en el origen autoritativo, no una
	// revision historica ni una cache que ignore revocaciones.
	BuscarSituacionesOperativasActuales(
		context.Context,
		ConsultaSituacionOperativaActual,
	) ([]domain.SituacionOperativaPerfilDocumental, error)
}

type ConsultaComponenteDocumentalAtestado struct {
	Rol                 domain.RolComponenteDocumental
	DescriptorPerfilRef string
	PublicacionRef      string
	PerfilRef           domain.ReferenciaPerfilDocumental
	DigestPerfil        string
	RevisionCatalogo    domain.RevisionCatalogoFormatosDocumentales
}

func (c ConsultaComponenteDocumentalAtestado) Validar() error {
	if !c.Rol.Valido() || !referenciaDescriptorDocumentalValida.MatchString(c.DescriptorPerfilRef) ||
		!referenciaDescriptorDocumentalValida.MatchString(c.PublicacionRef) ||
		strings.ContainsRune(c.DescriptorPerfilRef, '*') || strings.ContainsRune(c.PublicacionRef, '*') ||
		c.DescriptorPerfilRef == c.PublicacionRef || c.PerfilRef.Validar() != nil ||
		!huellaSHA256FormatoDocumentalValida(c.DigestPerfil) || c.RevisionCatalogo.Validar() != nil {
		return ErrComponenteDocumentalAtestadoInvalido
	}
	return nil
}

// DescriptorComponenteDocumentalAtestado es una declaracion de valor emitida
// por el registro/broker. El ejecutable no declara su ID ni su digest. La
// verificacion criptografica de la atestacion corresponde al adaptador del
// broker antes de construir este valor.
type DescriptorComponenteDocumentalAtestado struct {
	referencia                   string
	consulta                     ConsultaComponenteDocumentalAtestado
	componente                   domain.ReferenciaComponenteDocumental
	dominioConfianzaRef          string
	brokerRef                    string
	atestacionBrokerRef          string
	huellaAtestacionBrokerSHA256 string
	maximoBytes                  uint64
	digestDeclaracion            string
}

const maximoBytesComponenteDocumental uint64 = 4 * 1024 * 1024 * 1024

func NuevoDescriptorComponenteDocumentalAtestado(
	referencia string,
	consulta ConsultaComponenteDocumentalAtestado,
	componente domain.ReferenciaComponenteDocumental,
	dominioConfianzaRef, brokerRef, atestacionBrokerRef, huellaAtestacionBrokerSHA256 string,
	maximoBytes uint64,
) (DescriptorComponenteDocumentalAtestado, error) {
	descriptor := DescriptorComponenteDocumentalAtestado{
		referencia: referencia, consulta: consulta, componente: componente,
		dominioConfianzaRef: dominioConfianzaRef, brokerRef: brokerRef,
		atestacionBrokerRef:          atestacionBrokerRef,
		huellaAtestacionBrokerSHA256: huellaAtestacionBrokerSHA256,
		maximoBytes:                  maximoBytes,
	}
	descriptor.digestDeclaracion = descriptor.calcularDigestDeclaracion()
	if descriptor.Validar() != nil {
		return DescriptorComponenteDocumentalAtestado{}, ErrComponenteDocumentalAtestadoInvalido
	}
	return descriptor, nil
}

func (d DescriptorComponenteDocumentalAtestado) Validar() error {
	if !referenciaDescriptorDocumentalValida.MatchString(d.referencia) ||
		strings.ContainsRune(d.referencia, '*') || d.consulta.Validar() != nil ||
		d.componente.Validar() != nil || d.componente.Rol() != d.consulta.Rol ||
		!referenciaDescriptorDocumentalValida.MatchString(d.dominioConfianzaRef) ||
		!referenciaDescriptorDocumentalValida.MatchString(d.brokerRef) ||
		!referenciaDescriptorDocumentalValida.MatchString(d.atestacionBrokerRef) ||
		contieneComodinDescriptorDocumental(
			d.dominioConfianzaRef, d.brokerRef, d.atestacionBrokerRef,
		) || d.dominioConfianzaRef == d.brokerRef || d.brokerRef == d.atestacionBrokerRef ||
		!huellaSHA256FormatoDocumentalValida(d.huellaAtestacionBrokerSHA256) ||
		d.maximoBytes == 0 || d.maximoBytes > maximoBytesComponenteDocumental ||
		!huellaSHA256FormatoDocumentalValida(d.digestDeclaracion) ||
		d.calcularDigestDeclaracion() != d.digestDeclaracion {
		return ErrComponenteDocumentalAtestadoInvalido
	}
	return nil
}

func (d DescriptorComponenteDocumentalAtestado) Referencia() string { return d.referencia }
func (d DescriptorComponenteDocumentalAtestado) Consulta() ConsultaComponenteDocumentalAtestado {
	return d.consulta
}
func (d DescriptorComponenteDocumentalAtestado) Componente() domain.ReferenciaComponenteDocumental {
	return d.componente
}
func (d DescriptorComponenteDocumentalAtestado) DominioConfianzaRef() string {
	return d.dominioConfianzaRef
}
func (d DescriptorComponenteDocumentalAtestado) BrokerRef() string { return d.brokerRef }
func (d DescriptorComponenteDocumentalAtestado) AtestacionBrokerRef() string {
	return d.atestacionBrokerRef
}
func (d DescriptorComponenteDocumentalAtestado) HuellaAtestacionBrokerSHA256() string {
	return d.huellaAtestacionBrokerSHA256
}
func (d DescriptorComponenteDocumentalAtestado) MaximoBytes() uint64 { return d.maximoBytes }
func (d DescriptorComponenteDocumentalAtestado) DigestDeclaracionSHA256() string {
	return d.digestDeclaracion
}

func (d DescriptorComponenteDocumentalAtestado) Coincide(
	consulta ConsultaComponenteDocumentalAtestado,
) bool {
	return d.Validar() == nil && consulta.Validar() == nil && d.consulta == consulta
}

// IndependienteDe exige segregacion real: rol, ID/version, artefacto,
// homologacion y dominio de confianza diferentes. Renombrar el mismo binario
// bajo otra funcion no crea una barrera independiente.
func (d DescriptorComponenteDocumentalAtestado) IndependienteDe(
	otro DescriptorComponenteDocumentalAtestado,
) bool {
	if d.Validar() != nil || otro.Validar() != nil || d.consulta.Rol == otro.consulta.Rol ||
		d.dominioConfianzaRef == otro.dominioConfianzaRef ||
		(d.componente.Identificador() == otro.componente.Identificador() &&
			d.componente.Version() == otro.componente.Version()) ||
		d.componente.HomologacionRef() == otro.componente.HomologacionRef() ||
		d.componente.HuellaHomologacionSHA256() == otro.componente.HuellaHomologacionSHA256() ||
		d.componente.HuellaArtefactoSHA256() == otro.componente.HuellaArtefactoSHA256() {
		return false
	}
	return true
}

func (d DescriptorComponenteDocumentalAtestado) calcularDigestDeclaracion() string {
	valores := []string{
		"vec.descriptor-componente-documental-atestado.v1", d.referencia,
		string(d.consulta.Rol), d.consulta.DescriptorPerfilRef, d.consulta.PublicacionRef,
		d.consulta.PerfilRef.Identificador(), strconv.FormatUint(d.consulta.PerfilRef.Version(), 10),
		d.consulta.DigestPerfil, strconv.FormatUint(d.consulta.RevisionCatalogo.Numero(), 10),
		d.consulta.RevisionCatalogo.HuellaSHA256(), d.componente.Identificador(),
		strconv.FormatUint(d.componente.Version(), 10), d.componente.HomologacionRef(),
		d.componente.HuellaHomologacionSHA256(), d.componente.HuellaArtefactoSHA256(),
		d.dominioConfianzaRef, d.brokerRef, d.atestacionBrokerRef,
		d.huellaAtestacionBrokerSHA256, strconv.FormatUint(d.maximoBytes, 10),
	}
	return huellaCanonicaFormatoDocumental(valores)
}

type RegistroComponentesDocumentalesAtestados interface {
	BuscarComponentesDocumentalesAtestados(
		context.Context,
		ConsultaComponenteDocumentalAtestado,
	) ([]DescriptorComponenteDocumentalAtestado, error)
}

// Los ejecutores son contratos por operacion. No exponen identidad propia y
// nunca se devuelven desde un resultado de aplicacion.
type EjecutorRenderizadoDocumental interface {
	Renderizar(
		context.Context,
		DescriptorComponenteDocumentalAtestado,
		domain.PerfilFormatoDocumental,
		domain.ContenidoDocumento,
		uint64,
		io.Writer,
	) error
}

type EjecutorValidacionConformidadDocumental interface {
	ValidarConformidad(
		context.Context,
		DescriptorComponenteDocumentalAtestado,
		domain.PerfilFormatoDocumental,
		[]byte,
		uint64,
	) error
}

type EjecutorMarcadoInstitucionalDocumental interface {
	IncorporarMetadatoAntesFirma(
		context.Context,
		DescriptorComponenteDocumentalAtestado,
		domain.PerfilFormatoDocumental,
		[]byte,
		string,
		domain.MarcaInstitucionalDocumento,
		uint64,
		io.Writer,
	) error
}

type ResultadoExtraccionMetadatoInstitucional struct {
	Metadato                domain.MarcaInstitucionalDocumento
	HuellaContenidoSHA256   string
	DigestConformidadSHA256 string
}

func (r ResultadoExtraccionMetadatoInstitucional) ValidarContra(
	perfil domain.PerfilFormatoDocumental,
	contenido []byte,
) error {
	if perfil.Validar() != nil || len(contenido) == 0 || r.Metadato.Validar() != nil ||
		r.Metadato.Perfil() != perfil.Referencia() ||
		r.HuellaContenidoSHA256 != huellaBytesFormatoDocumental(contenido) ||
		r.DigestConformidadSHA256 != perfil.Conformidad().DigestSHA256() {
		return ErrMetadatoInstitucionalDocumentalInvalido
	}
	return nil
}

type EjecutorExtraccionMetadatoInstitucional interface {
	// Debe analizar el documento completo y fallar ante metadato ausente,
	// duplicado o escondido en comentarios/zero-width/canales no autorizados.
	ExtraerYValidarMetadatoInstitucional(
		context.Context,
		DescriptorComponenteDocumentalAtestado,
		domain.PerfilFormatoDocumental,
		[]byte,
		uint64,
	) (ResultadoExtraccionMetadatoInstitucional, error)
}

type EjecutorVerificacionSemanticaDocumental interface {
	VerificarEquivalenciaSemantica(
		context.Context,
		DescriptorComponenteDocumentalAtestado,
		domain.PerfilFormatoDocumental,
		[]byte,
		[]byte,
		uint64,
	) error
}

// GeneradorReferenciaBorradorDocumental crea una referencia opaca antes del
// renderizado. La implementacion productiva debe garantizar unicidad en el
// repositorio durable; el valor no puede contener datos personales.
type GeneradorReferenciaBorradorDocumental interface {
	NuevaReferenciaBorradorDocumental(context.Context) (string, error)
}

type ConsultaPoliticaInstitucionalDocumental struct {
	Institucion        domain.ReferenciaInstitucionalDocumento
	PerfilRef          domain.ReferenciaPerfilDocumental
	ManifiestoRef      string
	RequiereURIPublica bool
}

func (c ConsultaPoliticaInstitucionalDocumental) Validar() error {
	if c.Institucion.Validar() != nil || c.PerfilRef.Validar() != nil ||
		!referenciaDescriptorDocumentalValida.MatchString(c.ManifiestoRef) ||
		strings.ContainsRune(c.ManifiestoRef, '*') {
		return ErrPoliticaInstitucionalDocumentalInvalida
	}
	return nil
}

// PoliticaInstitucionalDocumentalAtestada es una entrada positiva de catalogo.
// El usuario no proporciona URI: ConstruirMarca la deriva de la base HTTPS
// exacta que el catalogo institucional ha permitido.
type PoliticaInstitucionalDocumentalAtestada struct {
	referencia              string
	revision                uint64
	consulta                ConsultaPoliticaInstitucionalDocumental
	endpointPublicoRef      string
	baseURIPublicaPermitida string
	huellaPoliticaSHA256    string
	digestDeclaracion       string
}

func NuevaPoliticaInstitucionalDocumentalAtestada(
	referencia string,
	revision uint64,
	consulta ConsultaPoliticaInstitucionalDocumental,
	endpointPublicoRef, baseURIPublicaPermitida, huellaPoliticaSHA256 string,
) (PoliticaInstitucionalDocumentalAtestada, error) {
	politica := PoliticaInstitucionalDocumentalAtestada{
		referencia: referencia, revision: revision, consulta: consulta,
		endpointPublicoRef:      endpointPublicoRef,
		baseURIPublicaPermitida: baseURIPublicaPermitida,
		huellaPoliticaSHA256:    huellaPoliticaSHA256,
	}
	politica.digestDeclaracion = politica.calcularDigestDeclaracion()
	if politica.Validar() != nil {
		return PoliticaInstitucionalDocumentalAtestada{}, ErrPoliticaInstitucionalDocumentalInvalida
	}
	return politica, nil
}

func (p PoliticaInstitucionalDocumentalAtestada) Validar() error {
	if !referenciaDescriptorDocumentalValida.MatchString(p.referencia) || p.revision == 0 ||
		p.consulta.Validar() != nil || !huellaSHA256FormatoDocumentalValida(p.huellaPoliticaSHA256) ||
		!huellaSHA256FormatoDocumentalValida(p.digestDeclaracion) ||
		p.calcularDigestDeclaracion() != p.digestDeclaracion {
		return ErrPoliticaInstitucionalDocumentalInvalida
	}
	if p.consulta.RequiereURIPublica {
		if !referenciaDescriptorDocumentalValida.MatchString(p.endpointPublicoRef) ||
			!baseURIPublicaInstitucionalValida(p.baseURIPublicaPermitida) {
			return ErrPoliticaInstitucionalDocumentalInvalida
		}
	} else if p.endpointPublicoRef != "" || p.baseURIPublicaPermitida != "" {
		return ErrPoliticaInstitucionalDocumentalInvalida
	}
	return nil
}

func (p PoliticaInstitucionalDocumentalAtestada) Referencia() string { return p.referencia }
func (p PoliticaInstitucionalDocumentalAtestada) Revision() uint64   { return p.revision }
func (p PoliticaInstitucionalDocumentalAtestada) Consulta() ConsultaPoliticaInstitucionalDocumental {
	return p.consulta
}
func (p PoliticaInstitucionalDocumentalAtestada) EndpointPublicoRef() string {
	return p.endpointPublicoRef
}
func (p PoliticaInstitucionalDocumentalAtestada) HuellaPoliticaSHA256() string {
	return p.huellaPoliticaSHA256
}
func (p PoliticaInstitucionalDocumentalAtestada) DigestDeclaracionSHA256() string {
	return p.digestDeclaracion
}

func (p PoliticaInstitucionalDocumentalAtestada) Coincide(
	consulta ConsultaPoliticaInstitucionalDocumental,
) bool {
	return p.Validar() == nil && consulta.Validar() == nil && p.consulta == consulta
}

func (p PoliticaInstitucionalDocumentalAtestada) ConstruirMarca(
	documentoUUID string,
	fecha time.Time,
) (domain.MarcaInstitucionalDocumento, error) {
	if p.Validar() != nil {
		return domain.MarcaInstitucionalDocumento{}, ErrPoliticaInstitucionalDocumentalInvalida
	}
	uriPublica := ""
	if p.consulta.RequiereURIPublica {
		uriPublica = strings.TrimSuffix(p.baseURIPublicaPermitida, "/") + "/" + documentoUUID
	}
	marca, err := domain.NuevaMarcaInstitucionalDocumento(
		p.consulta.Institucion, documentoUUID, p.consulta.PerfilRef, fecha,
		p.consulta.ManifiestoRef, uriPublica,
	)
	if err != nil {
		return domain.MarcaInstitucionalDocumento{}, ErrPoliticaInstitucionalDocumentalInvalida
	}
	return marca, nil
}

func (p PoliticaInstitucionalDocumentalAtestada) calcularDigestDeclaracion() string {
	return huellaCanonicaFormatoDocumental([]string{
		"vec.politica-institucional-documental-atestada.v1", p.referencia,
		strconv.FormatUint(p.revision, 10), p.consulta.Institucion.Entidad(),
		p.consulta.Institucion.Organo(), p.consulta.PerfilRef.Identificador(),
		strconv.FormatUint(p.consulta.PerfilRef.Version(), 10), p.consulta.ManifiestoRef,
		strconv.FormatBool(p.consulta.RequiereURIPublica), p.endpointPublicoRef,
		p.baseURIPublicaPermitida, p.huellaPoliticaSHA256,
	})
}

type CatalogoPoliticasInstitucionalesDocumentales interface {
	BuscarPoliticasInstitucionalesExactas(
		context.Context,
		ConsultaPoliticaInstitucionalDocumental,
	) ([]PoliticaInstitucionalDocumentalAtestada, error)
}

// RenderizadorDocumentalPorPerfil declara su identidad gobernada; no recibe
// comandos ni configuracion procedente del catalogo.
type RenderizadorDocumentalPorPerfil interface {
	PerfilDocumental() domain.ReferenciaPerfilDocumental
	DigestPerfilSHA256() string
	ConectorDocumental() domain.ReferenciaConectorDocumental
	Renderizar(context.Context, domain.ContenidoDocumento) ([]byte, error)
	ValidarSalida(context.Context, []byte) error
}

// RegistroRenderizadoresDocumentales devuelve candidatos exactos. Cero o mas
// de uno se cierran en la capa de aplicacion.
type RegistroRenderizadoresDocumentales interface {
	BuscarRenderizadoresDocumentales(
		context.Context,
		domain.ReferenciaPerfilDocumental,
		domain.ReferenciaConectorDocumental,
	) ([]RenderizadorDocumentalPorPerfil, error)
}

type EtapaMetadatoInstitucionalDocumental string

const EtapaMetadatoInstitucionalAntesFirma EtapaMetadatoInstitucionalDocumental = "antes_firma"

// SolicitudIncorporarMetadatoInstitucional solo acepta bytes aun no firmados.
// El metadato es normalizado y no renderizado por defecto; el conector debe
// incorporarlo mediante el mecanismo estandar del perfil, nunca mediante
// zero-width, comentarios secretos o alteracion del significado visible.
type SolicitudIncorporarMetadatoInstitucional struct {
	Perfil              domain.PerfilFormatoDocumental
	Conector            domain.ReferenciaConectorDocumental
	Etapa               EtapaMetadatoInstitucionalDocumental
	ContenidoSinFirma   []byte
	HuellaEntradaSHA256 string
	Metadato            domain.MarcaInstitucionalDocumento
}

func (s SolicitudIncorporarMetadatoInstitucional) Validar() error {
	if s.Perfil.Validar() != nil || s.Perfil.Estado() != domain.EstadoPerfilDocumentalVigente ||
		!s.Perfil.Capacidades().Tiene(domain.CapacidadPerfilMetadatoInstitucional) ||
		s.Conector.Validar() != nil || s.Etapa != EtapaMetadatoInstitucionalAntesFirma ||
		len(s.ContenidoSinFirma) == 0 || !huellaSHA256FormatoDocumentalValida(s.HuellaEntradaSHA256) ||
		huellaBytesFormatoDocumental(s.ContenidoSinFirma) != s.HuellaEntradaSHA256 ||
		s.Metadato.Validar() != nil || s.Metadato.Perfil() != s.Perfil.Referencia() {
		return ErrMetadatoInstitucionalDocumentalInvalido
	}
	return nil
}

type ResultadoMetadatoInstitucional struct {
	Contenido            []byte
	HuellaFinalSHA256    string
	HuellaMetadatoSHA256 string
	PerfilRef            domain.ReferenciaPerfilDocumental
	Conector             domain.ReferenciaConectorDocumental
}

// ValidarContra modela exclusivamente incorporacion embebida mediante el
// mecanismo estandar del perfil, por eso los bytes finales deben cambiar. Un
// formato sin metadato estandar queda cerrado hasta disponer de un puerto
// distinto de manifiesto lateral; nunca se simula con contenido invisible.
func (r ResultadoMetadatoInstitucional) ValidarContra(
	solicitud SolicitudIncorporarMetadatoInstitucional,
) error {
	huellaMetadato, err := solicitud.Metadato.HuellaSHA256()
	if solicitud.Validar() != nil || err != nil || len(r.Contenido) == 0 ||
		!huellaSHA256FormatoDocumentalValida(r.HuellaFinalSHA256) ||
		huellaBytesFormatoDocumental(r.Contenido) != r.HuellaFinalSHA256 ||
		r.HuellaFinalSHA256 == solicitud.HuellaEntradaSHA256 ||
		r.HuellaMetadatoSHA256 != huellaMetadato || r.PerfilRef != solicitud.Perfil.Referencia() ||
		r.Conector != solicitud.Conector {
		return ErrMetadatoInstitucionalDocumentalInvalido
	}
	return nil
}

// MarcadorMetadatoInstitucionalDocumental debe usar metadatos estandar del
// perfil. No se le permite autocertificar la equivalencia semantica de su
// propia salida: esa barrera pertenece a un puerto independiente.
type MarcadorMetadatoInstitucionalDocumental interface {
	PerfilDocumental() domain.ReferenciaPerfilDocumental
	DigestPerfilSHA256() string
	ConectorDocumental() domain.ReferenciaConectorDocumental
	IncorporarMetadatoInstitucional(
		context.Context,
		SolicitudIncorporarMetadatoInstitucional,
	) (ResultadoMetadatoInstitucional, error)
}

type RegistroMarcadoresMetadatoInstitucional interface {
	BuscarMarcadoresMetadatoInstitucional(
		context.Context,
		domain.ReferenciaPerfilDocumental,
		domain.ReferenciaConectorDocumental,
	) ([]MarcadorMetadatoInstitucionalDocumental, error)
}

// VerificadorEquivalenciaSemanticaDocumental es una dependencia distinta del
// marcador. Sin una implementacion homologada independiente, la integracion
// queda cerrada; una autocomprobacion del mismo conector no es garantia
// productiva suficiente. Esta interfaz aun no modela identidad, digest ni
// atestacion del verificador: el bootstrap productivo no debe habilitarla hasta
// incorporar y cotejar esas pruebas contra un componente distinto del marcador.
type VerificadorEquivalenciaSemanticaDocumental interface {
	VerificarEquivalenciaSemantica(
		context.Context,
		domain.PerfilFormatoDocumental,
		[]byte,
		[]byte,
	) error
}

func RenderizadorDocumentalNulo(renderizador RenderizadorDocumentalPorPerfil) bool {
	return dependenciaFormatoDocumentalNula(renderizador)
}

func MarcadorMetadatoInstitucionalNulo(marcador MarcadorMetadatoInstitucionalDocumental) bool {
	return dependenciaFormatoDocumentalNula(marcador)
}

func dependenciaFormatoDocumentalNula(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}

func contieneComodinDescriptorDocumental(valores ...string) bool {
	for _, valor := range valores {
		if strings.ContainsRune(valor, '*') {
			return true
		}
	}
	return false
}

func huellaCanonicaFormatoDocumental(valores []string) string {
	calculador := sha256.New()
	for _, valor := range valores {
		_, _ = calculador.Write([]byte(strconv.Itoa(len(valor))))
		_, _ = calculador.Write([]byte{':'})
		_, _ = calculador.Write([]byte(valor))
		_, _ = calculador.Write([]byte{'\n'})
	}
	return hex.EncodeToString(calculador.Sum(nil))
}

func baseURIPublicaInstitucionalValida(valor string) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > 2_048 ||
		strings.ContainsAny(valor, "\r\n\t") || strings.HasSuffix(valor, "/") {
		return false
	}
	analizada, err := url.Parse(valor)
	if err != nil || analizada.Scheme != "https" || analizada.Opaque != "" ||
		analizada.User != nil || analizada.Host == "" || analizada.RawQuery != "" ||
		analizada.ForceQuery || analizada.Fragment != "" || analizada.Port() != "" {
		return false
	}
	host := analizada.Hostname()
	if host == "" || host != strings.ToLower(host) || !strings.ContainsRune(host, '.') ||
		host == "localhost" || strings.HasSuffix(host, ".local") || net.ParseIP(host) != nil {
		return false
	}
	ruta := analizada.EscapedPath()
	return ruta != "" && ruta == analizada.Path && !strings.Contains(ruta, "//") &&
		!strings.Contains(ruta, "/../") && !strings.Contains(ruta, "/./")
}

func huellaBytesFormatoDocumental(contenido []byte) string {
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:])
}

func huellaSHA256FormatoDocumentalValida(valor string) bool {
	if len(valor) != 64 || valor != strings.TrimSpace(valor) || valor != strings.ToLower(valor) {
		return false
	}
	decodificada, err := hex.DecodeString(valor)
	return err == nil && len(decodificada) == sha256.Size
}
