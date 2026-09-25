package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var ErrDocumentoInvalido = errors.New("documentos: documento invalido")

const EstadoFirmaPendienteProveedor = "pendiente_proveedor"

// Estado de la politica de conservacion con que se incorporo el documento.
// Con una politica provisional el plazo solo consta como metadato: el almacen
// no fija retencion hasta aplicar el catalogo definitivo.
const (
	EstadoPoliticaAprobada    = "aprobada"
	EstadoPoliticaProvisional = "provisional"
)

// Custodia indica quien guarda los bytes. Con custodia VEC el original esta en
// AlmacenObjetos y se puede descargar; con custodia externa VEC solo conserva
// la referencia opaca del custodio y la huella, nunca el contenido.
const (
	CustodiaVEC     = "vec"
	CustodiaExterna = "externa"
)

// Documento identifica una incorporacion administrativa; la identidad no se
// deduce de la huella porque dos productores pueden conservar los mismos bytes.
// NumeroVEC es numeracion interna y nunca acredita asiento registral general.
type Documento struct {
	ID                   string
	NumeroVEC            string
	ModuloID             string
	ExpedienteRef        string
	TipoRef              string
	Version              uint64
	MIME                 string
	HuellaSHA256         string
	Tamano               int64
	ObjetoRef            string
	ObjetoVersion        string
	PoliticaRef          string
	VersionPolitica      uint64
	HuellaPoliticaSHA256 string
	ConservacionHasta    time.Time
	Proteccion           string
	EstadoPolitica       string
	EstadoFirma          string
	CreadoEn             time.Time
	// Custodia es CustodiaVEC o CustodiaExterna. Con custodia externa,
	// ObjetoRef y ObjetoVersion van vacios y CustodiaExternaRef identifica el
	// original en su custodio; MIME y Tamano pueden no estar declarados.
	Custodia           string
	CustodiaExternaRef ReferenciaCustodiaExterna
}

func (d Documento) Validar() error {
	if !ReferenciaOpacaValida(d.ID) || !NumeroVECValido(d.NumeroVEC) ||
		!IdentificadorTecnicoValido(d.ModuloID) || !ReferenciaOpacaValida(d.ExpedienteRef) ||
		!ReferenciaOpacaValida(d.TipoRef) || d.Version == 0 ||
		!ReferenciaOpacaValida(d.PoliticaRef) || d.VersionPolitica == 0 ||
		!HuellaValida(d.HuellaSHA256) || !HuellaValida(d.HuellaPoliticaSHA256) ||
		d.EstadoFirma != EstadoFirmaPendienteProveedor ||
		d.ConservacionHasta.IsZero() || d.CreadoEn.IsZero() ||
		(d.Proteccion != "conservacion" && d.Proteccion != "bloqueo") ||
		(d.EstadoPolitica != EstadoPoliticaAprobada && d.EstadoPolitica != EstadoPoliticaProvisional) ||
		(d.EstadoPolitica == EstadoPoliticaProvisional && d.Proteccion != "conservacion") {
		return ErrDocumentoInvalido
	}
	switch d.Custodia {
	case CustodiaVEC:
		if !ReferenciaValida(d.ObjetoRef) || !ReferenciaValida(d.ObjetoVersion) ||
			d.Tamano < 1 || !MIMEValido(d.MIME) || d.CustodiaExternaRef != (ReferenciaCustodiaExterna{}) {
			return ErrDocumentoInvalido
		}
	case CustodiaExterna:
		if d.ObjetoRef != "" || d.ObjetoVersion != "" || d.Tamano < 0 ||
			(d.MIME != "" && !MIMEValido(d.MIME)) ||
			d.CustodiaExternaRef.Validar() != nil || d.CustodiaExternaRef.HuellaSHA256 != d.HuellaSHA256 {
			return ErrDocumentoInvalido
		}
	default:
		return ErrDocumentoInvalido
	}
	return nil
}

// Descargable indica si VEC custodia los bytes y puede entregarlos.
func (d Documento) Descargable() bool { return d.Custodia == CustodiaVEC }

// ReferenciaCustodiaExterna es el contrato comun para un original que guarda
// otro sistema: quien lo custodia, la referencia opaca que ese custodio da y
// la huella SHA-256 de los bytes. Es lo que hoy anotan los justificantes de
// Dietas, Bolsa o los correos de Contratacion. No acredita firma, registro ni
// entrega, y VEC no recibe el contenido.
type ReferenciaCustodiaExterna struct {
	CustodioID   string
	Referencia   string
	HuellaSHA256 string
}

func (r ReferenciaCustodiaExterna) Validar() error {
	if !IdentificadorTecnicoValido(r.CustodioID) || !ReferenciaCustodioValida(r.Referencia) || !HuellaValida(r.HuellaSHA256) {
		return ErrDocumentoInvalido
	}
	return nil
}

var mimeValido = regexp.MustCompile(`^[a-z0-9.+-]+/[a-z0-9.+-]+$`)

// MIMEValido admite un tipo/subtipo en minusculas, sin parametros.
func MIMEValido(s string) bool { return len(s) >= 3 && len(s) <= 255 && mimeValido.MatchString(s) }

// ReferenciaCustodioValida es mas estricta que ReferenciaValida: ademas de
// imprimible y sin espacios, excluye rutas y comodines. Coincide con
// vec_documentos.referencia_custodio_v1.
func ReferenciaCustodioValida(s string) bool {
	return ReferenciaValida(s) && !strings.ContainsAny(s, "/\\*?%")
}

func ReferenciaValida(s string) bool {
	if len(s) < 3 || len(s) > 512 || strings.ContainsAny(s, " \t\r\n/\\\x00") ||
		strings.Contains(s, "..") || strings.ContainsAny(s, "*?%") {
		return false
	}
	for _, c := range s {
		if c < 33 || c > 126 {
			return false
		}
	}
	return true
}

var referenciaHash = regexp.MustCompile(`^ref:[0-9a-f]{64}$`)
var referenciaUUID = regexp.MustCompile(`^[a-z][a-z0-9_]{1,31}:[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var identificadorTecnico = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,127}$`)
var numeroVEC = regexp.MustCompile(`^VEC-[0-9]{4}-[0-9]{1,12}$`)

func ReferenciaOpacaValida(s string) bool {
	return referenciaHash.MatchString(s) && s != "ref:"+strings.Repeat("0", 64) || referenciaUUID.MatchString(s)
}

func IdentificadorTecnicoValido(s string) bool { return identificadorTecnico.MatchString(s) }
func NumeroVECValido(s string) bool            { return numeroVEC.MatchString(s) }

func HuellaValida(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			if c < 'a' || c > 'f' {
				return false
			}
		}
	}
	return s != strings.Repeat("0", 64)
}

// NotificacionPreparada solo declara una intencion durable; no representa
// envio, puesta a disposicion, recepcion ni entrega.
type NotificacionPreparada struct {
	ID              string
	DocumentoID     string
	Version         uint64
	DestinatarioRef string
	Canal           string
	PreparadaEn     time.Time
}

func (n NotificacionPreparada) Validar() error {
	if !ReferenciaOpacaValida(n.ID) || !ReferenciaOpacaValida(n.DocumentoID) ||
		n.Version == 0 || !ReferenciaOpacaValida(n.DestinatarioRef) ||
		!IdentificadorTecnicoValido(n.Canal) || n.PreparadaEn.IsZero() {
		return ErrDocumentoInvalido
	}
	return nil
}
