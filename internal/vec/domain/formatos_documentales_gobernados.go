package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	documentalcanonico "vec-diputacion-granada/internal/vec/canonico/documental"
)

var (
	ErrIdentidadSintacticaDocumentalInvalida  = errors.New("vec: identidad sintactica documental invalida")
	ErrReferenciaPerfilDocumentalInvalida     = errors.New("vec: referencia de perfil documental invalida")
	ErrConformidadDocumentalInvalida          = errors.New("vec: conformidad documental invalida")
	ErrPerfilFormatoDocumentalInvalido        = errors.New("vec: perfil de formato documental invalido")
	ErrPublicacionPerfilDocumentalInvalida    = errors.New("vec: publicacion de perfil documental invalida")
	ErrRevisionCatalogoFormatosInvalida       = errors.New("vec: revision de catalogo de formatos invalida")
	ErrReferenciaComponenteDocumentalInvalida = errors.New("vec: referencia de componente documental invalida")
	ErrReferenciaConectorDocumentalInvalida   = errors.New("vec: referencia de conector documental invalida")
	ErrMarcaInstitucionalDocumentoInvalida    = errors.New("vec: metadato institucional documental invalido")
)

const maximoBytesPerfilDocumental uint64 = 4 * 1024 * 1024 * 1024

var (
	identificadorFormatoGobernadoValido = regexp.MustCompile(`^[a-z][a-z0-9._+-]{0,63}$`)
	mimeFormatoGobernadoValido          = regexp.MustCompile(`^[a-z][a-z0-9!#$&^_.+-]{0,63}/[a-z0-9][a-z0-9!#$&^_.+-]{0,126}$`)
	extensionFormatoGobernadoValida     = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{0,30}[a-z0-9]$`)
	charsetFormatoGobernadoValido       = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,31}$`)
	uuidDocumentoV4Valido               = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

// IdentidadSintacticaDocumental solo identifica una familia sintactica. MIME,
// extension, charset, conformidad y capacidades pertenecen al perfil.
type IdentidadSintacticaDocumental struct{ identificador string }

func NuevaIdentidadSintacticaDocumental(identificador string) (IdentidadSintacticaDocumental, error) {
	identidad := IdentidadSintacticaDocumental{identificador: identificador}
	if identidad.Validar() != nil {
		return IdentidadSintacticaDocumental{}, ErrIdentidadSintacticaDocumentalInvalida
	}
	return identidad, nil
}

func (i IdentidadSintacticaDocumental) Validar() error {
	if !identificadorFormatoGobernadoValido.MatchString(i.identificador) ||
		strings.ContainsRune(i.identificador, '*') {
		return ErrIdentidadSintacticaDocumentalInvalida
	}
	return nil
}

func (i IdentidadSintacticaDocumental) Identificador() string { return i.identificador }

type ReferenciaPerfilDocumental struct {
	identificador string
	version       uint64
}

func NuevaReferenciaPerfilDocumental(
	identificador string,
	version uint64,
) (ReferenciaPerfilDocumental, error) {
	referencia := ReferenciaPerfilDocumental{identificador: identificador, version: version}
	if referencia.Validar() != nil {
		return ReferenciaPerfilDocumental{}, ErrReferenciaPerfilDocumentalInvalida
	}
	return referencia, nil
}

func (r ReferenciaPerfilDocumental) Validar() error {
	if !referenciaGobernadaValida(r.identificador) || r.version == 0 ||
		strings.ContainsRune(r.identificador, '*') {
		return ErrReferenciaPerfilDocumentalInvalida
	}
	return nil
}

func (r ReferenciaPerfilDocumental) Identificador() string { return r.identificador }
func (r ReferenciaPerfilDocumental) Version() uint64       { return r.version }

// ReferenciaConformidadDocumental compromete esquema, dialecto,
// canonicalizacion y reglas concretas. Son referencias declarativas; nunca
// contienen codigo, comandos ni URL ejecutables.
type ReferenciaConformidadDocumental struct {
	identificador        string
	version              uint64
	esquemaRef           string
	dialectoRef          string
	canonicalizacionRef  string
	reglasRef            string
	huellaReglasSHA256   string
	politicaRef          string
	huellaPoliticaSHA256 string
	digest               string
}

func NuevaReferenciaConformidadDocumental(
	identificador string,
	version uint64,
	esquemaRef, dialectoRef, canonicalizacionRef, reglasRef, huellaReglasSHA256 string,
	politicaRef, huellaPoliticaSHA256 string,
) (ReferenciaConformidadDocumental, error) {
	referencia := ReferenciaConformidadDocumental{
		identificador: identificador, version: version, esquemaRef: esquemaRef,
		dialectoRef: dialectoRef, canonicalizacionRef: canonicalizacionRef,
		reglasRef: reglasRef, huellaReglasSHA256: huellaReglasSHA256,
		politicaRef: politicaRef, huellaPoliticaSHA256: huellaPoliticaSHA256,
	}
	referencia.digest = referencia.calcularDigest()
	if referencia.Validar() != nil {
		return ReferenciaConformidadDocumental{}, ErrConformidadDocumentalInvalida
	}
	return referencia, nil
}

func (r ReferenciaConformidadDocumental) Validar() error {
	if !referenciaGobernadaValida(r.identificador) || r.version == 0 ||
		!referenciaGobernadaValida(r.esquemaRef) ||
		!referenciaGobernadaValida(r.dialectoRef) ||
		!referenciaGobernadaValida(r.canonicalizacionRef) ||
		!referenciaGobernadaValida(r.reglasRef) ||
		!esHuellaSHA256DocumentalGobernada(r.huellaReglasSHA256) ||
		!referenciaGobernadaValida(r.politicaRef) ||
		!esHuellaSHA256DocumentalGobernada(r.huellaPoliticaSHA256) ||
		!esHuellaSHA256DocumentalGobernada(r.digest) || r.calcularDigest() != r.digest ||
		contieneComodinFormatoGobernado(r.identificador, r.esquemaRef, r.dialectoRef,
			r.canonicalizacionRef, r.reglasRef, r.politicaRef) {
		return ErrConformidadDocumentalInvalida
	}
	return nil
}

func (r ReferenciaConformidadDocumental) Identificador() string        { return r.identificador }
func (r ReferenciaConformidadDocumental) Version() uint64              { return r.version }
func (r ReferenciaConformidadDocumental) EsquemaRef() string           { return r.esquemaRef }
func (r ReferenciaConformidadDocumental) DialectoRef() string          { return r.dialectoRef }
func (r ReferenciaConformidadDocumental) CanonicalizacionRef() string  { return r.canonicalizacionRef }
func (r ReferenciaConformidadDocumental) ReglasRef() string            { return r.reglasRef }
func (r ReferenciaConformidadDocumental) HuellaReglasSHA256() string   { return r.huellaReglasSHA256 }
func (r ReferenciaConformidadDocumental) PoliticaRef() string          { return r.politicaRef }
func (r ReferenciaConformidadDocumental) HuellaPoliticaSHA256() string { return r.huellaPoliticaSHA256 }
func (r ReferenciaConformidadDocumental) DigestSHA256() string         { return r.digest }

func (r ReferenciaConformidadDocumental) calcularDigest() string {
	return huellaCanonicaDocumentalGobernada([]string{
		"vec.conformidad-documental.v1", r.identificador, strconv.FormatUint(r.version, 10),
		r.esquemaRef, r.dialectoRef, r.canonicalizacionRef, r.reglasRef, r.huellaReglasSHA256,
		r.politicaRef, r.huellaPoliticaSHA256,
	})
}

// CapacidadPerfilFormatoDocumental es vocabulario compilado del protocolo, no
// un catalogo abierto de formatos. Una capacidad semantica nueva exige
// desplegar codigo que la valide y cumpla; un registro no puede inventarla.
type CapacidadPerfilFormatoDocumental string

const (
	CapacidadPerfilRenderizar             CapacidadPerfilFormatoDocumental = "renderizar"
	CapacidadPerfilMetadatoInstitucional  CapacidadPerfilFormatoDocumental = "metadato_institucional"
	CapacidadPerfilFirmaElectronica       CapacidadPerfilFormatoDocumental = "firma_electronica"
	CapacidadPerfilEdicion                CapacidadPerfilFormatoDocumental = "edicion"
	CapacidadPerfilPreservacionLargoPlazo CapacidadPerfilFormatoDocumental = "preservacion_largo_plazo"
)

func (c CapacidadPerfilFormatoDocumental) Valida() bool {
	_, valida := bitCapacidadPerfilFormatoDocumental(c)
	return valida
}

type CapacidadesPerfilFormatoDocumental struct{ bits uint16 }

func NuevasCapacidadesPerfilFormatoDocumental(
	capacidades ...CapacidadPerfilFormatoDocumental,
) (CapacidadesPerfilFormatoDocumental, error) {
	if len(capacidades) == 0 || len(capacidades) > 5 {
		return CapacidadesPerfilFormatoDocumental{}, ErrPerfilFormatoDocumentalInvalido
	}
	var resultado CapacidadesPerfilFormatoDocumental
	for _, capacidad := range capacidades {
		bit, valida := bitCapacidadPerfilFormatoDocumental(capacidad)
		if !valida || resultado.bits&bit != 0 {
			return CapacidadesPerfilFormatoDocumental{}, ErrPerfilFormatoDocumentalInvalido
		}
		resultado.bits |= bit
	}
	return resultado, nil
}

func (c CapacidadesPerfilFormatoDocumental) Validar() error {
	const todos uint16 = 1 | 2 | 4 | 8 | 16
	if c.bits == 0 || c.bits&^todos != 0 {
		return ErrPerfilFormatoDocumentalInvalido
	}
	return nil
}

func (c CapacidadesPerfilFormatoDocumental) Tiene(capacidad CapacidadPerfilFormatoDocumental) bool {
	bit, valida := bitCapacidadPerfilFormatoDocumental(capacidad)
	return valida && c.Validar() == nil && c.bits&bit != 0
}

func bitCapacidadPerfilFormatoDocumental(capacidad CapacidadPerfilFormatoDocumental) (uint16, bool) {
	switch capacidad {
	case CapacidadPerfilRenderizar:
		return 1, true
	case CapacidadPerfilMetadatoInstitucional:
		return 2, true
	case CapacidadPerfilFirmaElectronica:
		return 4, true
	case CapacidadPerfilEdicion:
		return 8, true
	case CapacidadPerfilPreservacionLargoPlazo:
		return 16, true
	default:
		return 0, false
	}
}

// PerfilFormatoDocumental es una especificacion inmutable. Su estado operativo
// no forma parte del perfil: se consulta en PublicacionPerfilFormatoDocumental
// en cada ejecucion, permitiendo revocar sin reescribir historia.
type PerfilFormatoDocumental struct {
	referencia  ReferenciaPerfilDocumental
	identidad   IdentidadSintacticaDocumental
	mime        string
	extension   string
	charset     string
	capacidades CapacidadesPerfilFormatoDocumental
	conformidad ReferenciaConformidadDocumental
	maximoBytes uint64
	digest      string
}

func NuevoPerfilFormatoDocumentalConforme(
	referencia ReferenciaPerfilDocumental,
	identidad IdentidadSintacticaDocumental,
	mime, extension, charset string,
	capacidades CapacidadesPerfilFormatoDocumental,
	conformidad ReferenciaConformidadDocumental,
	maximoBytes uint64,
) (PerfilFormatoDocumental, error) {
	perfil := PerfilFormatoDocumental{
		referencia: referencia, identidad: identidad, mime: mime, extension: extension,
		charset: charset, capacidades: capacidades, conformidad: conformidad, maximoBytes: maximoBytes,
	}
	perfil.digest = perfil.calcularDigest()
	if perfil.Validar() != nil {
		return PerfilFormatoDocumental{}, ErrPerfilFormatoDocumentalInvalido
	}
	return perfil, nil
}

// NuevoPerfilFormatoDocumental conserva solo compatibilidad de compilacion
// durante la migracion. Carece de conformidad y limite, por lo que siempre
// deniega; ningun perfil legacy obtiene autoridad positiva por defecto.
func NuevoPerfilFormatoDocumental(
	ReferenciaPerfilDocumental,
	IdentidadSintacticaDocumental,
	string, string, string,
	EstadoPerfilDocumental,
	CapacidadesPerfilFormatoDocumental,
) (PerfilFormatoDocumental, error) {
	return PerfilFormatoDocumental{}, ErrPerfilFormatoDocumentalInvalido
}

func (p PerfilFormatoDocumental) Validar() error {
	if p.referencia.Validar() != nil || p.identidad.Validar() != nil ||
		!mimeFormatoGobernadoValido.MatchString(p.mime) || !extensionFormatoDocumentalValida(p.extension) ||
		!charsetFormatoGobernadoValido.MatchString(p.charset) || p.capacidades.Validar() != nil ||
		!p.capacidades.Tiene(CapacidadPerfilRenderizar) || p.conformidad.Validar() != nil ||
		p.maximoBytes == 0 || p.maximoBytes > maximoBytesPerfilDocumental ||
		!esHuellaSHA256DocumentalGobernada(p.digest) || p.calcularDigest() != p.digest {
		return ErrPerfilFormatoDocumentalInvalido
	}
	return nil
}

func (p PerfilFormatoDocumental) Referencia() ReferenciaPerfilDocumental   { return p.referencia }
func (p PerfilFormatoDocumental) Identidad() IdentidadSintacticaDocumental { return p.identidad }
func (p PerfilFormatoDocumental) MIME() string                             { return p.mime }
func (p PerfilFormatoDocumental) Extension() string                        { return p.extension }
func (p PerfilFormatoDocumental) Charset() string                          { return p.charset }
func (p PerfilFormatoDocumental) Capacidades() CapacidadesPerfilFormatoDocumental {
	return p.capacidades
}
func (p PerfilFormatoDocumental) Conformidad() ReferenciaConformidadDocumental { return p.conformidad }
func (p PerfilFormatoDocumental) MaximoBytes() uint64                          { return p.maximoBytes }
func (p PerfilFormatoDocumental) DigestSHA256() string                         { return p.digest }

func (p PerfilFormatoDocumental) calcularDigest() string {
	return huellaCanonicaDocumentalGobernada([]string{
		"vec.perfil-formato-documental.v2", p.referencia.identificador,
		strconv.FormatUint(p.referencia.version, 10), p.identidad.identificador,
		p.mime, p.extension, p.charset, strconv.FormatUint(uint64(p.capacidades.bits), 10),
		p.conformidad.digest, strconv.FormatUint(p.maximoBytes, 10),
	})
}

// EstadoPerfilDocumental solo conserva compatibilidad fail-closed del primer
// corte. El perfil ya no almacena estado y Estado() devuelve valor invalido.
type EstadoPerfilDocumental string

const (
	EstadoPerfilDocumentalVigente  EstadoPerfilDocumental = "vigente"
	EstadoPerfilDocumentalRetirado EstadoPerfilDocumental = "retirado"
)

func (p PerfilFormatoDocumental) Estado() EstadoPerfilDocumental { return "" }

type EstadoPublicacionPerfilDocumental string

const (
	EstadoPublicacionPerfilVigente  EstadoPublicacionPerfilDocumental = "vigente"
	EstadoPublicacionPerfilRevocada EstadoPublicacionPerfilDocumental = "revocada"
	EstadoPublicacionPerfilRetirada EstadoPublicacionPerfilDocumental = "retirada"
)

func (e EstadoPublicacionPerfilDocumental) Valido() bool {
	return e == EstadoPublicacionPerfilVigente || e == EstadoPublicacionPerfilRevocada ||
		e == EstadoPublicacionPerfilRetirada
}

type PublicacionPerfilFormatoDocumental struct {
	publicacionRef    string
	perfilRef         ReferenciaPerfilDocumental
	digestPerfil      string
	revisionCatalogo  RevisionCatalogoFormatosDocumentales
	revisionOperativa uint64
	estado            EstadoPublicacionPerfilDocumental
	huella            string
}

func NuevaPublicacionPerfilFormatoDocumental(
	publicacionRef string,
	perfil PerfilFormatoDocumental,
	revision RevisionCatalogoFormatosDocumentales,
	revisionOperativa uint64,
	estado EstadoPublicacionPerfilDocumental,
) (PublicacionPerfilFormatoDocumental, error) {
	publicacion := PublicacionPerfilFormatoDocumental{
		publicacionRef: publicacionRef, perfilRef: perfil.Referencia(), digestPerfil: perfil.DigestSHA256(),
		revisionCatalogo: revision, revisionOperativa: revisionOperativa, estado: estado,
	}
	publicacion.huella = publicacion.calcularHuella()
	if perfil.Validar() != nil || publicacion.Validar() != nil {
		return PublicacionPerfilFormatoDocumental{}, ErrPublicacionPerfilDocumentalInvalida
	}
	return publicacion, nil
}

func (p PublicacionPerfilFormatoDocumental) Validar() error {
	if !referenciaGobernadaValida(p.publicacionRef) ||
		strings.ContainsRune(p.publicacionRef, '*') || p.perfilRef.Validar() != nil ||
		!esHuellaSHA256DocumentalGobernada(p.digestPerfil) || p.revisionCatalogo.Validar() != nil ||
		p.revisionOperativa == 0 || !p.estado.Valido() ||
		!esHuellaSHA256DocumentalGobernada(p.huella) || p.calcularHuella() != p.huella {
		return ErrPublicacionPerfilDocumentalInvalida
	}
	return nil
}

func (p PublicacionPerfilFormatoDocumental) PublicacionRef() string { return p.publicacionRef }
func (p PublicacionPerfilFormatoDocumental) PerfilRef() ReferenciaPerfilDocumental {
	return p.perfilRef
}
func (p PublicacionPerfilFormatoDocumental) DigestPerfilSHA256() string { return p.digestPerfil }
func (p PublicacionPerfilFormatoDocumental) RevisionCatalogo() RevisionCatalogoFormatosDocumentales {
	return p.revisionCatalogo
}
func (p PublicacionPerfilFormatoDocumental) RevisionOperativa() uint64 { return p.revisionOperativa }
func (p PublicacionPerfilFormatoDocumental) Secuencia() uint64         { return p.revisionOperativa }
func (p PublicacionPerfilFormatoDocumental) Estado() EstadoPublicacionPerfilDocumental {
	return p.estado
}
func (p PublicacionPerfilFormatoDocumental) HuellaSHA256() string { return p.huella }

func (p PublicacionPerfilFormatoDocumental) Coincide(
	perfil PerfilFormatoDocumental,
	revision RevisionCatalogoFormatosDocumentales,
) bool {
	return p.Validar() == nil && perfil.Validar() == nil && revision.Validar() == nil &&
		p.perfilRef == perfil.Referencia() && p.digestPerfil == perfil.DigestSHA256() &&
		p.revisionCatalogo == revision
}

// AutorizaEjecucion solo concede autoridad positiva a la proyeccion operativa
// actual y vigente que coincide exactamente con perfil y revision de catalogo.
// Una revision historica, aunque fuera vigente en su momento, no debe usarse
// sin releer el registro operativo actual.
func (p PublicacionPerfilFormatoDocumental) AutorizaEjecucion(
	perfil PerfilFormatoDocumental,
	revision RevisionCatalogoFormatosDocumentales,
) bool {
	return p.estado == EstadoPublicacionPerfilVigente && p.Coincide(perfil, revision)
}

// EsSucesoraDe valida la cadena append-only de la situacion operativa. Revocar
// o retirar es terminal; una nueva publicacion requiere otra referencia.
func (p PublicacionPerfilFormatoDocumental) EsSucesoraDe(
	anterior PublicacionPerfilFormatoDocumental,
) bool {
	if p.Validar() != nil || anterior.Validar() != nil ||
		p.publicacionRef != anterior.publicacionRef || p.perfilRef != anterior.perfilRef ||
		p.digestPerfil != anterior.digestPerfil || p.revisionCatalogo != anterior.revisionCatalogo ||
		p.revisionOperativa != anterior.revisionOperativa+1 ||
		anterior.estado != EstadoPublicacionPerfilVigente {
		return false
	}
	return p.estado == EstadoPublicacionPerfilRevocada ||
		p.estado == EstadoPublicacionPerfilRetirada
}

// SituacionOperativaPerfilDocumental nombra expresamente el registro
// append-only cuya ultima secuencia forma la proyeccion operativa actual.
type SituacionOperativaPerfilDocumental = PublicacionPerfilFormatoDocumental

func NuevaSituacionOperativaPerfilDocumental(
	publicacionRef string,
	perfil PerfilFormatoDocumental,
	revision RevisionCatalogoFormatosDocumentales,
	secuencia uint64,
	estado EstadoPublicacionPerfilDocumental,
) (SituacionOperativaPerfilDocumental, error) {
	return NuevaPublicacionPerfilFormatoDocumental(
		publicacionRef, perfil, revision, secuencia, estado,
	)
}

func (p PublicacionPerfilFormatoDocumental) calcularHuella() string {
	return huellaCanonicaDocumentalGobernada([]string{
		"vec.publicacion-perfil-formato-documental.v1", p.publicacionRef,
		p.perfilRef.identificador, strconv.FormatUint(p.perfilRef.version, 10), p.digestPerfil,
		strconv.FormatUint(p.revisionCatalogo.numero, 10), p.revisionCatalogo.huellaSHA256,
		strconv.FormatUint(p.revisionOperativa, 10), string(p.estado),
	})
}

type RevisionCatalogoFormatosDocumentales struct {
	numero       uint64
	huellaSHA256 string
}

func NuevaRevisionCatalogoFormatosDocumentales(
	numero uint64,
	huellaSHA256 string,
) (RevisionCatalogoFormatosDocumentales, error) {
	revision := RevisionCatalogoFormatosDocumentales{numero: numero, huellaSHA256: huellaSHA256}
	if revision.Validar() != nil {
		return RevisionCatalogoFormatosDocumentales{}, ErrRevisionCatalogoFormatosInvalida
	}
	return revision, nil
}

func (r RevisionCatalogoFormatosDocumentales) Validar() error {
	if r.numero == 0 || !esHuellaSHA256DocumentalGobernada(r.huellaSHA256) {
		return ErrRevisionCatalogoFormatosInvalida
	}
	return nil
}

func (r RevisionCatalogoFormatosDocumentales) Numero() uint64       { return r.numero }
func (r RevisionCatalogoFormatosDocumentales) HuellaSHA256() string { return r.huellaSHA256 }

type RolComponenteDocumental string

const (
	RolComponenteRenderizador       RolComponenteDocumental = "renderizador"
	RolComponenteMarcador           RolComponenteDocumental = "marcador"
	RolComponenteExtractorMetadatos RolComponenteDocumental = "extractor_metadatos"
	RolComponenteVerificador        RolComponenteDocumental = "verificador"
	// RolComponenteValidadorEstructural explicita el unico significado del
	// valor historico "verificador". Se conserva el alias para no reinterpretar
	// silenciosamente referencias ya publicadas.
	RolComponenteValidadorEstructural = RolComponenteVerificador
	// RolComponenteVerificadorSemantico corresponde a una carga de trabajo
	// distinta, que compara el contenido neutral con el documento producido.
	RolComponenteVerificadorSemantico RolComponenteDocumental = "verificador_semantico"
)

func (r RolComponenteDocumental) Valido() bool {
	return r == RolComponenteRenderizador || r == RolComponenteMarcador ||
		r == RolComponenteExtractorMetadatos || r == RolComponenteVerificador ||
		r == RolComponenteVerificadorSemantico
}

// ReferenciaComponenteDocumental es el valor atestado por el registro/broker
// para un rol concreto. El componente ejecutable nunca se autoacredita.
type ReferenciaComponenteDocumental struct {
	rol                      RolComponenteDocumental
	identificador            string
	version                  uint64
	homologacionRef          string
	huellaHomologacionSHA256 string
	huellaArtefactoSHA256    string
}

func NuevaReferenciaComponenteDocumental(
	rol RolComponenteDocumental,
	identificador string,
	version uint64,
	homologacionRef, huellaHomologacionSHA256, huellaArtefactoSHA256 string,
) (ReferenciaComponenteDocumental, error) {
	referencia := ReferenciaComponenteDocumental{
		rol: rol, identificador: identificador, version: version, homologacionRef: homologacionRef,
		huellaHomologacionSHA256: huellaHomologacionSHA256, huellaArtefactoSHA256: huellaArtefactoSHA256,
	}
	if referencia.Validar() != nil {
		return ReferenciaComponenteDocumental{}, ErrReferenciaComponenteDocumentalInvalida
	}
	return referencia, nil
}

func (r ReferenciaComponenteDocumental) Validar() error {
	if !r.rol.Valido() || !referenciaGobernadaValida(r.identificador) || r.version == 0 ||
		!referenciaGobernadaValida(r.homologacionRef) ||
		!esHuellaSHA256DocumentalGobernada(r.huellaHomologacionSHA256) ||
		!esHuellaSHA256DocumentalGobernada(r.huellaArtefactoSHA256) ||
		contieneComodinFormatoGobernado(r.identificador, r.homologacionRef) {
		return ErrReferenciaComponenteDocumentalInvalida
	}
	return nil
}

func (r ReferenciaComponenteDocumental) Rol() RolComponenteDocumental { return r.rol }
func (r ReferenciaComponenteDocumental) Identificador() string        { return r.identificador }
func (r ReferenciaComponenteDocumental) Version() uint64              { return r.version }
func (r ReferenciaComponenteDocumental) HomologacionRef() string      { return r.homologacionRef }
func (r ReferenciaComponenteDocumental) HuellaHomologacionSHA256() string {
	return r.huellaHomologacionSHA256
}
func (r ReferenciaComponenteDocumental) HuellaArtefactoSHA256() string {
	return r.huellaArtefactoSHA256
}

// ReferenciaConectorDocumental conserva compatibilidad de compilacion hasta
// retirar el contrato anterior. El nuevo camino usa siempre referencia por rol.
type ReferenciaConectorDocumental = ReferenciaComponenteDocumental

func NuevaReferenciaConectorDocumental(
	identificador string,
	version uint64,
	homologacionRef, huellaHomologacionSHA256, huellaArtefactoSHA256 string,
) (ReferenciaConectorDocumental, error) {
	referencia, err := NuevaReferenciaComponenteDocumental(
		RolComponenteRenderizador, identificador, version, homologacionRef,
		huellaHomologacionSHA256, huellaArtefactoSHA256,
	)
	if err != nil {
		return ReferenciaConectorDocumental{}, ErrReferenciaConectorDocumentalInvalida
	}
	return referencia, nil
}

// Las referencias institucionales son opacas. Su pertenencia institucional,
// ausencia de PII y URI permitida no se acreditan por prefijo ni regex: eso lo
// decide una politica/catalogo institucional positivo en aplicacion.
type ReferenciaInstitucionalDocumento struct {
	entidadRef string
	organoRef  string
}

func NuevaReferenciaInstitucionalDocumento(
	entidadRef, organoRef string,
) (ReferenciaInstitucionalDocumento, error) {
	referencia := ReferenciaInstitucionalDocumento{entidadRef: entidadRef, organoRef: organoRef}
	if referencia.Validar() != nil {
		return ReferenciaInstitucionalDocumento{}, ErrMarcaInstitucionalDocumentoInvalida
	}
	return referencia, nil
}

func (r ReferenciaInstitucionalDocumento) Validar() error {
	if !referenciaGobernadaValida(r.entidadRef) ||
		!referenciaGobernadaValida(r.organoRef) || r.entidadRef == r.organoRef ||
		contieneComodinFormatoGobernado(r.entidadRef, r.organoRef) {
		return ErrMarcaInstitucionalDocumentoInvalida
	}
	return nil
}

func (r ReferenciaInstitucionalDocumento) Entidad() string { return r.entidadRef }
func (r ReferenciaInstitucionalDocumento) Organo() string  { return r.organoRef }

type MarcaInstitucionalDocumento struct {
	institucion   ReferenciaInstitucionalDocumento
	documentoUUID string
	perfil        ReferenciaPerfilDocumental
	fecha         time.Time
	manifiestoRef string
	uriPublica    string
}

func NuevaMarcaInstitucionalDocumento(
	institucion ReferenciaInstitucionalDocumento,
	documentoUUID string,
	perfil ReferenciaPerfilDocumental,
	fecha time.Time,
	manifiestoRef, uriPublica string,
) (MarcaInstitucionalDocumento, error) {
	marca := MarcaInstitucionalDocumento{
		institucion: institucion, documentoUUID: documentoUUID, perfil: perfil,
		fecha: fecha, manifiestoRef: manifiestoRef, uriPublica: uriPublica,
	}
	if marca.Validar() != nil {
		return MarcaInstitucionalDocumento{}, ErrMarcaInstitucionalDocumentoInvalida
	}
	return marca, nil
}

func (m MarcaInstitucionalDocumento) Validar() error {
	if m.institucion.Validar() != nil || !uuidDocumentoV4Valido.MatchString(m.documentoUUID) ||
		m.documentoUUID == "00000000-0000-4000-8000-000000000000" || m.perfil.Validar() != nil ||
		m.fecha.IsZero() || m.fecha.Location() != time.UTC || m.fecha.Nanosecond()%1_000 != 0 ||
		!referenciaGobernadaValida(m.manifiestoRef) ||
		strings.ContainsRune(m.manifiestoRef, '*') ||
		(m.uriPublica != "" && !uriPublicaDocumentoSintacticamenteValida(m.uriPublica, m.documentoUUID)) {
		return ErrMarcaInstitucionalDocumentoInvalida
	}
	return nil
}

func (m MarcaInstitucionalDocumento) Institucion() ReferenciaInstitucionalDocumento {
	return m.institucion
}
func (m MarcaInstitucionalDocumento) DocumentoUUID() string              { return m.documentoUUID }
func (m MarcaInstitucionalDocumento) Perfil() ReferenciaPerfilDocumental { return m.perfil }
func (m MarcaInstitucionalDocumento) Fecha() time.Time                   { return m.fecha }
func (m MarcaInstitucionalDocumento) ManifiestoRef() string              { return m.manifiestoRef }
func (m MarcaInstitucionalDocumento) URIPublica() string                 { return m.uriPublica }

func (m MarcaInstitucionalDocumento) HuellaSHA256() (string, error) {
	if m.Validar() != nil {
		return "", ErrMarcaInstitucionalDocumentoInvalida
	}
	return huellaCanonicaDocumentalGobernada([]string{
		"vec.metadato-institucional-documento.v2", m.institucion.entidadRef,
		m.institucion.organoRef, m.documentoUUID, m.perfil.identificador,
		strconv.FormatUint(m.perfil.version, 10), m.fecha.Format(time.RFC3339Nano),
		m.manifiestoRef, m.uriPublica,
	}), nil
}

func huellaCanonicaDocumentalGobernada(valores []string) string {
	calculador := sha256.New()
	for _, valor := range valores {
		_, _ = calculador.Write([]byte(strconv.Itoa(len(valor))))
		_, _ = calculador.Write([]byte{':'})
		_, _ = calculador.Write([]byte(valor))
		_, _ = calculador.Write([]byte{'\n'})
	}
	return hex.EncodeToString(calculador.Sum(nil))
}

func esHuellaSHA256DocumentalGobernada(valor string) bool {
	return documentalcanonico.SHA256HexadecimalValido(valor)
}

func referenciaGobernadaValida(valor string) bool {
	return documentalcanonico.ReferenciaASCIIBasicaValida(valor)
}

func extensionFormatoDocumentalValida(valor string) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > 32 ||
		strings.HasPrefix(valor, ".") || strings.HasSuffix(valor, ".") || strings.Contains(valor, "..") ||
		strings.ContainsAny(valor, "/\\\r\n\t") {
		return false
	}
	if len(valor) == 1 {
		return valor[0] >= 'a' && valor[0] <= 'z' || valor[0] >= '0' && valor[0] <= '9'
	}
	return extensionFormatoGobernadoValida.MatchString(valor)
}

func uriPublicaDocumentoSintacticamenteValida(valor, documentoUUID string) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > 2_048 ||
		strings.ContainsAny(valor, "\r\n\t") {
		return false
	}
	analizada, err := url.Parse(valor)
	if err != nil || analizada.Scheme != "https" || analizada.Opaque != "" || analizada.User != nil ||
		analizada.Host == "" || analizada.RawQuery != "" || analizada.ForceQuery ||
		analizada.Fragment != "" || analizada.Port() != "" {
		return false
	}
	host := analizada.Hostname()
	if host == "" || host != strings.ToLower(host) || !strings.ContainsRune(host, '.') ||
		host == "localhost" || strings.HasSuffix(host, ".local") || net.ParseIP(host) != nil {
		return false
	}
	ruta := analizada.EscapedPath()
	if ruta == "" || ruta != analizada.Path || strings.Contains(ruta, "//") ||
		strings.Contains(ruta, "/../") || strings.Contains(ruta, "/./") {
		return false
	}
	segmentos := strings.Split(strings.Trim(ruta, "/"), "/")
	return len(segmentos) >= 2 && segmentos[len(segmentos)-1] == documentoUUID
}

func contieneComodinFormatoGobernado(valores ...string) bool {
	for _, valor := range valores {
		if strings.ContainsRune(valor, '*') {
			return true
		}
	}
	return false
}
