package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var ErrDocumentoInvalido = errors.New("documentos: documento invalido")

const EstadoFirmaPendienteProveedor = "pendiente_proveedor"

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
	EstadoFirma          string
	CreadoEn             time.Time
}

func (d Documento) Validar() error {
	if !ReferenciaOpacaValida(d.ID) || !NumeroVECValido(d.NumeroVEC) ||
		!IdentificadorTecnicoValido(d.ModuloID) || !ReferenciaOpacaValida(d.ExpedienteRef) ||
		!ReferenciaOpacaValida(d.TipoRef) || d.Version == 0 ||
		!ReferenciaValida(d.ObjetoRef) || !ReferenciaValida(d.ObjetoVersion) ||
		!ReferenciaOpacaValida(d.PoliticaRef) || d.VersionPolitica == 0 ||
		!HuellaValida(d.HuellaSHA256) || !HuellaValida(d.HuellaPoliticaSHA256) ||
		d.Tamano < 1 || d.MIME == "" || len(d.MIME) > 255 ||
		strings.ContainsAny(d.MIME, "\r\n\x00") ||
		d.EstadoFirma != EstadoFirmaPendienteProveedor ||
		d.ConservacionHasta.IsZero() || d.CreadoEn.IsZero() ||
		(d.Proteccion != "conservacion" && d.Proteccion != "bloqueo") {
		return ErrDocumentoInvalido
	}
	return nil
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
