package ports

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	dominioCredencialAutoridadFuenteAnalisis  = "VEC-CT-CREDENCIAL-AUTORIDAD-FUENTE-ANALISIS-V1"
	dominioDesafioAutoridadFuenteAnalisis     = "VEC-CT-DESAFIO-AUTORIDAD-FUENTE-ANALISIS-V1"
	maximoRaicesAutoridadFuenteAnalisis       = 16
	maximoRevocacionesAutoridadFuenteAnalisis = 4096
)

var errAutoridadFuenteAnalisisNoConfiable = errors.New(
	"contratacion temporal: autoridad de fuente de analisis no confiable",
)

// RolAutoridadFuenteAnalisis separa las competencias técnicas que intervienen
// en una respuesta. Son invariantes del protocolo, no un catálogo funcional.
type RolAutoridadFuenteAnalisis string

const (
	RolFuentePresupuestaria RolAutoridadFuenteAnalisis = "fuente_presupuestaria"
	RolCalculadorCoste      RolAutoridadFuenteAnalisis = "calculador_coste"
	RolVerificadorRespuesta RolAutoridadFuenteAnalisis = "verificador_respuesta"
	RolPublicadorCatalogo   RolAutoridadFuenteAnalisis = "publicador_catalogo"
)

func (r RolAutoridadFuenteAnalisis) valida() bool {
	switch r {
	case RolFuentePresupuestaria, RolCalculadorCoste,
		RolVerificadorRespuesta, RolPublicadorCatalogo:
		return true
	default:
		return false
	}
}

// DatosCredencialAutoridadFuenteAnalisis es el material que firma la autoridad
// institucional. BackendRef es la identidad canónica del límite de confianza:
// dos aliases o wrappers del mismo backend deben compartirlo.
type DatosCredencialAutoridadFuenteAnalisis struct {
	RaizClaveID        string
	AutoridadRef       string
	BackendRef         string
	OrganizacionRef    string
	Audiencia          string
	Rol                RolAutoridadFuenteAnalisis
	Serie              uint64
	Generacion         uint32
	ClavePruebaEd25519 []byte
	EmitidaEn          time.Time
	ValidaHasta        time.Time
}

func (d DatosCredencialAutoridadFuenteAnalisis) validar() error {
	if !domain.ReferenciaOpacaValida(d.RaizClaveID) ||
		!domain.ReferenciaOpacaValida(d.AutoridadRef) ||
		!domain.ReferenciaOpacaValida(d.BackendRef) ||
		!domain.ReferenciaOpacaValida(d.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(d.Audiencia) ||
		!d.Rol.valida() || d.Serie == 0 ||
		d.Serie > maximoEnteroSeguroFuenteAnalisis ||
		d.Generacion == 0 ||
		len(d.ClavePruebaEd25519) != ed25519.PublicKeySize ||
		!instanteFuenteAnalisisCanonico(d.EmitidaEn) ||
		!instanteFuenteAnalisisCanonico(d.ValidaHasta) ||
		!d.ValidaHasta.After(d.EmitidaEn) {
		return errAutoridadFuenteAnalisisNoConfiable
	}
	return nil
}

// CredencialAutoridadFuenteAnalisis conserva copias defensivas del documento y
// su firma. Su creación solo comprueba estructura: la raíz institucional se
// valida posteriormente contra la confianza fijada por la composición.
type CredencialAutoridadFuenteAnalisis struct {
	datos     *DatosCredencialAutoridadFuenteAnalisis
	documento []byte
	firma     []byte
}

func NuevaCredencialAutoridadFuenteAnalisis(
	datos DatosCredencialAutoridadFuenteAnalisis,
	firmaInstitucional []byte,
) (CredencialAutoridadFuenteAnalisis, error) {
	documento, err := canonCredencialAutoridadFuenteAnalisis(datos)
	if err != nil || len(firmaInstitucional) != ed25519.SignatureSize {
		return CredencialAutoridadFuenteAnalisis{},
			errAutoridadFuenteAnalisisNoConfiable
	}
	copia := datos
	copia.ClavePruebaEd25519 = append([]byte(nil), datos.ClavePruebaEd25519...)
	return CredencialAutoridadFuenteAnalisis{
		datos: &copia, documento: documento,
		firma: append([]byte(nil), firmaInstitucional...),
	}, nil
}

func (c CredencialAutoridadFuenteAnalisis) validarEstructura() error {
	if c.datos == nil || c.datos.validar() != nil ||
		len(c.firma) != ed25519.SignatureSize {
		return errAutoridadFuenteAnalisisNoConfiable
	}
	canon, err := canonCredencialAutoridadFuenteAnalisis(*c.datos)
	if err != nil || !bytes.Equal(canon, c.documento) {
		return errAutoridadFuenteAnalisisNoConfiable
	}
	return nil
}

func (CredencialAutoridadFuenteAnalisis) String() string {
	return "[CREDENCIAL-AUTORIDAD-FUENTE-ANALISIS-REDACTADA]"
}

func (c CredencialAutoridadFuenteAnalisis) GoString() string { return c.String() }
func (c CredencialAutoridadFuenteAnalisis) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, c.String())
}
func (c CredencialAutoridadFuenteAnalisis) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

type DesafioAutoridadFuenteAnalisis struct {
	contenido []byte
}

func (d DesafioAutoridadFuenteAnalisis) Bytes() ([]byte, error) {
	if len(d.contenido) == 0 || len(d.contenido) > 4096 {
		return nil, errAutoridadFuenteAnalisisNoConfiable
	}
	return append([]byte(nil), d.contenido...), nil
}

func (DesafioAutoridadFuenteAnalisis) String() string {
	return "[DESAFIO-AUTORIDAD-FUENTE-ANALISIS-REDACTADO]"
}

func (d DesafioAutoridadFuenteAnalisis) GoString() string { return d.String() }
func (d DesafioAutoridadFuenteAnalisis) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, d.String())
}
func (d DesafioAutoridadFuenteAnalisis) LogValue() slog.Value {
	return slog.StringValue(d.String())
}

type PresentacionAutoridadFuenteAnalisis struct {
	credencial CredencialAutoridadFuenteAnalisis
	prueba     []byte
}

func NuevaPresentacionAutoridadFuenteAnalisis(
	credencial CredencialAutoridadFuenteAnalisis,
	pruebaPosesionEd25519 []byte,
) (PresentacionAutoridadFuenteAnalisis, error) {
	if credencial.validarEstructura() != nil ||
		len(pruebaPosesionEd25519) != ed25519.SignatureSize {
		return PresentacionAutoridadFuenteAnalisis{},
			errAutoridadFuenteAnalisisNoConfiable
	}
	return PresentacionAutoridadFuenteAnalisis{
		credencial: credencial,
		prueba:     append([]byte(nil), pruebaPosesionEd25519...),
	}, nil
}

func (PresentacionAutoridadFuenteAnalisis) String() string {
	return "[PRESENTACION-AUTORIDAD-FUENTE-ANALISIS-REDACTADA]"
}

func (p PresentacionAutoridadFuenteAnalisis) GoString() string { return p.String() }
func (p PresentacionAutoridadFuenteAnalisis) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, p.String())
}
func (p PresentacionAutoridadFuenteAnalisis) LogValue() slog.Value {
	return slog.StringValue(p.String())
}

// PresentadorAutoridadFuenteAnalisis obliga a cada adaptador a demostrar la
// posesión de la clave certificada para el desafío único de la operación.
type PresentadorAutoridadFuenteAnalisis interface {
	PresentarAutoridadFuenteAnalisis(
		context.Context,
		DesafioAutoridadFuenteAnalisis,
	) (PresentacionAutoridadFuenteAnalisis, error)
}

type EstadoRaizAutoridadFuenteAnalisis string

const (
	RaizAutoridadActiva   EstadoRaizAutoridadFuenteAnalisis = "activa"
	RaizAutoridadRetenida EstadoRaizAutoridadFuenteAnalisis = "retenida"
	RaizAutoridadRevocada EstadoRaizAutoridadFuenteAnalisis = "revocada"
)

type RaizConfianzaAutoridadFuenteAnalisis struct {
	ClaveID                string
	ClavePublicaEd25519    []byte
	Estado                 EstadoRaizAutoridadFuenteAnalisis
	ValidaDesde            time.Time
	ValidaHasta            time.Time
	UltimaEmisionPermitida time.Time
}

type RevocacionAutoridadFuenteAnalisis struct {
	AutoridadRef string
	Serie        uint64
	RevocadaEn   time.Time
}

// ConfianzaAutoridadesFuenteAnalisis es configuración exclusiva de la
// composición del servidor. No pertenece a ninguna solicitud de cliente.
type ConfianzaAutoridadesFuenteAnalisis struct {
	organizacionRef string
	audiencia       string
	raices          map[string]raizConfianzaAutoridadFuenteAnalisis
	revocaciones    map[claveRevocacionAutoridadFuenteAnalisis]time.Time
}

type raizConfianzaAutoridadFuenteAnalisis struct {
	clave                  ed25519.PublicKey
	estado                 EstadoRaizAutoridadFuenteAnalisis
	validaDesde            time.Time
	validaHasta            time.Time
	ultimaEmisionPermitida time.Time
}

type claveRevocacionAutoridadFuenteAnalisis struct {
	autoridadRef string
	serie        uint64
}

func NuevaConfianzaAutoridadesFuenteAnalisis(
	organizacionRef string,
	audiencia string,
	raices []RaizConfianzaAutoridadFuenteAnalisis,
	revocaciones []RevocacionAutoridadFuenteAnalisis,
) (ConfianzaAutoridadesFuenteAnalisis, error) {
	if !domain.ReferenciaOpacaValida(organizacionRef) ||
		!domain.ReferenciaOpacaValida(audiencia) ||
		len(raices) == 0 || len(raices) > maximoRaicesAutoridadFuenteAnalisis ||
		len(revocaciones) > maximoRevocacionesAutoridadFuenteAnalisis {
		return ConfianzaAutoridadesFuenteAnalisis{},
			errAutoridadFuenteAnalisisNoConfiable
	}
	confianza := ConfianzaAutoridadesFuenteAnalisis{
		organizacionRef: organizacionRef,
		audiencia:       audiencia,
		raices:          make(map[string]raizConfianzaAutoridadFuenteAnalisis, len(raices)),
		revocaciones: make(
			map[claveRevocacionAutoridadFuenteAnalisis]time.Time,
			len(revocaciones),
		),
	}
	for _, raiz := range raices {
		if !domain.ReferenciaOpacaValida(raiz.ClaveID) ||
			len(raiz.ClavePublicaEd25519) != ed25519.PublicKeySize ||
			(raiz.Estado != RaizAutoridadActiva &&
				raiz.Estado != RaizAutoridadRetenida &&
				raiz.Estado != RaizAutoridadRevocada) ||
			!instanteFuenteAnalisisCanonico(raiz.ValidaDesde) ||
			!instanteFuenteAnalisisCanonico(raiz.ValidaHasta) ||
			!instanteFuenteAnalisisCanonico(raiz.UltimaEmisionPermitida) ||
			!raiz.ValidaHasta.After(raiz.ValidaDesde) ||
			raiz.UltimaEmisionPermitida.Before(raiz.ValidaDesde) ||
			raiz.UltimaEmisionPermitida.After(raiz.ValidaHasta) {
			return ConfianzaAutoridadesFuenteAnalisis{},
				errAutoridadFuenteAnalisisNoConfiable
		}
		if _, repetida := confianza.raices[raiz.ClaveID]; repetida {
			return ConfianzaAutoridadesFuenteAnalisis{},
				errAutoridadFuenteAnalisisNoConfiable
		}
		for _, existente := range confianza.raices {
			if bytes.Equal(existente.clave, raiz.ClavePublicaEd25519) {
				return ConfianzaAutoridadesFuenteAnalisis{},
					errAutoridadFuenteAnalisisNoConfiable
			}
		}
		confianza.raices[raiz.ClaveID] = raizConfianzaAutoridadFuenteAnalisis{
			clave:  append(ed25519.PublicKey(nil), raiz.ClavePublicaEd25519...),
			estado: raiz.Estado, validaDesde: raiz.ValidaDesde,
			validaHasta:            raiz.ValidaHasta,
			ultimaEmisionPermitida: raiz.UltimaEmisionPermitida,
		}
	}
	for _, revocacion := range revocaciones {
		clave := claveRevocacionAutoridadFuenteAnalisis{
			autoridadRef: revocacion.AutoridadRef,
			serie:        revocacion.Serie,
		}
		if !domain.ReferenciaOpacaValida(revocacion.AutoridadRef) ||
			revocacion.Serie == 0 ||
			revocacion.Serie > maximoEnteroSeguroFuenteAnalisis ||
			!instanteFuenteAnalisisCanonico(revocacion.RevocadaEn) {
			return ConfianzaAutoridadesFuenteAnalisis{},
				errAutoridadFuenteAnalisisNoConfiable
		}
		if _, repetida := confianza.revocaciones[clave]; repetida {
			return ConfianzaAutoridadesFuenteAnalisis{},
				errAutoridadFuenteAnalisisNoConfiable
		}
		confianza.revocaciones[clave] = revocacion.RevocadaEn
	}
	return confianza, nil
}

func (c ConfianzaAutoridadesFuenteAnalisis) Validar() error {
	if !domain.ReferenciaOpacaValida(c.organizacionRef) ||
		!domain.ReferenciaOpacaValida(c.audiencia) ||
		len(c.raices) == 0 ||
		len(c.raices) > maximoRaicesAutoridadFuenteAnalisis ||
		len(c.revocaciones) > maximoRevocacionesAutoridadFuenteAnalisis {
		return errAutoridadFuenteAnalisisNoConfiable
	}
	return nil
}

type identidadAutoridadFuenteAnalisis struct {
	raizClaveID             string
	autoridadRef            string
	backendRef              string
	rol                     RolAutoridadFuenteAnalisis
	serie                   uint64
	generacion              uint32
	huellaClavePruebaSHA256 string
	credencialEmitidaEn     time.Time
	credencialValidaHasta   time.Time
	clavePrueba             ed25519.PublicKey
}

func (c ConfianzaAutoridadesFuenteAnalisis) verificarPresentacion(
	presentacion PresentacionAutoridadFuenteAnalisis,
	desafio DesafioAutoridadFuenteAnalisis,
	rolEsperado RolAutoridadFuenteAnalisis,
	comprobadaEn time.Time,
) (identidadAutoridadFuenteAnalisis, error) {
	if presentacion.credencial.validarEstructura() != nil ||
		!rolEsperado.valida() ||
		!instanteFuenteAnalisisCanonico(comprobadaEn) ||
		c.organizacionRef == "" || c.audiencia == "" {
		return identidadAutoridadFuenteAnalisis{},
			errAutoridadFuenteAnalisisNoConfiable
	}
	datos := *presentacion.credencial.datos
	raiz, existe := c.raices[datos.RaizClaveID]
	horizonteOperacion := comprobadaEn.Add(TiempoMaximoFuenteAnalisis)
	revocadaEn, revocada := c.revocaciones[claveRevocacionAutoridadFuenteAnalisis{
		autoridadRef: datos.AutoridadRef,
		serie:        datos.Serie,
	}]
	materialDesafio, err := desafio.Bytes()
	if !existe || raiz.estado == RaizAutoridadRevocada ||
		comprobadaEn.Before(raiz.validaDesde) ||
		!comprobadaEn.Before(raiz.validaHasta) ||
		horizonteOperacion.After(raiz.validaHasta) ||
		datos.OrganizacionRef != c.organizacionRef ||
		datos.Audiencia != c.audiencia ||
		datos.Rol != rolEsperado ||
		datos.EmitidaEn.Before(raiz.validaDesde) ||
		datos.EmitidaEn.After(raiz.ultimaEmisionPermitida) ||
		comprobadaEn.Before(datos.EmitidaEn) ||
		!comprobadaEn.Before(datos.ValidaHasta) ||
		horizonteOperacion.After(datos.ValidaHasta) ||
		(revocada && revocadaEn.Before(horizonteOperacion)) ||
		err != nil ||
		!ed25519.Verify(raiz.clave, presentacion.credencial.documento,
			presentacion.credencial.firma) ||
		!ed25519.Verify(
			ed25519.PublicKey(datos.ClavePruebaEd25519),
			materialDesafio,
			presentacion.prueba,
		) {
		return identidadAutoridadFuenteAnalisis{},
			errAutoridadFuenteAnalisisNoConfiable
	}
	huellaClave := sha256.Sum256(datos.ClavePruebaEd25519)
	return identidadAutoridadFuenteAnalisis{
		raizClaveID:             datos.RaizClaveID,
		autoridadRef:            datos.AutoridadRef,
		backendRef:              datos.BackendRef,
		rol:                     datos.Rol,
		serie:                   datos.Serie,
		generacion:              datos.Generacion,
		huellaClavePruebaSHA256: fmt.Sprintf("%x", huellaClave[:]),
		credencialEmitidaEn:     datos.EmitidaEn,
		credencialValidaHasta:   datos.ValidaHasta,
		clavePrueba: append(
			ed25519.PublicKey(nil),
			datos.ClavePruebaEd25519...,
		),
	}, nil
}

func nuevoDesafioAutoridadFuenteAnalisis(
	materialPeticion []byte,
	organizacionRef string,
	audiencia string,
	rol RolAutoridadFuenteAnalisis,
) (DesafioAutoridadFuenteAnalisis, error) {
	if len(materialPeticion) == 0 || len(materialPeticion) > 64*1024 ||
		!domain.ReferenciaOpacaValida(organizacionRef) ||
		!domain.ReferenciaOpacaValida(audiencia) || !rol.valida() {
		return DesafioAutoridadFuenteAnalisis{},
			errAutoridadFuenteAnalisisNoConfiable
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return DesafioAutoridadFuenteAnalisis{},
			errAutoridadFuenteAnalisisNoConfiable
	}
	huella := sha256.Sum256(materialPeticion)
	escritor := bytes.NewBuffer(nil)
	escribirTextoAutoridad(escritor, dominioDesafioAutoridadFuenteAnalisis)
	escribirTextoAutoridad(escritor, organizacionRef)
	escribirTextoAutoridad(escritor, audiencia)
	escribirTextoAutoridad(escritor, string(rol))
	_, _ = escritor.Write(nonce)
	_, _ = escritor.Write(huella[:])
	return DesafioAutoridadFuenteAnalisis{
		contenido: append([]byte(nil), escritor.Bytes()...),
	}, nil
}

func canonCredencialAutoridadFuenteAnalisis(
	datos DatosCredencialAutoridadFuenteAnalisis,
) ([]byte, error) {
	if datos.validar() != nil {
		return nil, errAutoridadFuenteAnalisisNoConfiable
	}
	buffer := bytes.NewBuffer(nil)
	escribirTextoAutoridad(buffer, dominioCredencialAutoridadFuenteAnalisis)
	escribirTextoAutoridad(buffer, datos.RaizClaveID)
	escribirTextoAutoridad(buffer, datos.AutoridadRef)
	escribirTextoAutoridad(buffer, datos.BackendRef)
	escribirTextoAutoridad(buffer, datos.OrganizacionRef)
	escribirTextoAutoridad(buffer, datos.Audiencia)
	escribirTextoAutoridad(buffer, string(datos.Rol))
	var entero [8]byte
	binary.BigEndian.PutUint64(entero[:], datos.Serie)
	_, _ = buffer.Write(entero[:])
	var generacion [4]byte
	binary.BigEndian.PutUint32(generacion[:], datos.Generacion)
	_, _ = buffer.Write(generacion[:])
	_, _ = buffer.Write(datos.ClavePruebaEd25519)
	binary.BigEndian.PutUint64(entero[:], uint64(datos.EmitidaEn.UnixMicro()))
	_, _ = buffer.Write(entero[:])
	binary.BigEndian.PutUint64(entero[:], uint64(datos.ValidaHasta.UnixMicro()))
	_, _ = buffer.Write(entero[:])
	return append([]byte(nil), buffer.Bytes()...), nil
}

func escribirTextoAutoridad(buffer *bytes.Buffer, valor string) {
	var longitud [4]byte
	binary.BigEndian.PutUint32(longitud[:], uint32(len(valor)))
	_, _ = buffer.Write(longitud[:])
	_, _ = buffer.WriteString(valor)
}
