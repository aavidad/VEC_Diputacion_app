// Package seguridad implementa los puertos criptográficos de Aspirantes:
// AES-256-GCM por campo, índice ciego del documento y huella semántica de
// cada petición. Las claves llegan de una fuente inyectada, que la
// composición conecta con el material del portal externo.
package seguridad

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strconv"

	"vec-diputacion-granada/internal/modules/aspirantes/domain"
	"vec-diputacion-granada/internal/modules/aspirantes/ports"
)

var ErrCriptoNoDisponible = errors.New("aspirantes: criptografia no disponible")

// Límites que también impone SQL sobre cada sobre.
const (
	MaximoClaro   = 1024
	largoNonce    = 12
	largoEtiqueta = 16
)

// Clave es material ya obtenido del gestor de claves. Una clave revocada no
// descifra aunque siga en la lista de retenidas.
type Clave struct {
	Ref      string
	Material [32]byte
	Revocada bool
}

// ClavesAspirantes separa funciones: cifrar, indexar y sellar peticiones
// nunca comparten clave. Rotar el índice exige reindexar antes de abrir altas.
type ClavesAspirantes struct {
	CifradoActivo      Clave
	CifradoRetenidas   []Clave
	Indice             Clave
	SemanticaActiva    Clave
	SemanticaRetenidas []Clave
}

type FuenteClaves interface {
	CargarClavesAspirantes(context.Context) (ClavesAspirantes, error)
}

type Adaptador struct {
	fuente FuenteClaves
	azar   io.Reader
}

func NuevoAdaptador(fuente FuenteClaves) (*Adaptador, error) {
	if fuente == nil {
		return nil, ErrCriptoNoDisponible
	}
	return &Adaptador{fuente: fuente, azar: rand.Reader}, nil
}

func refValida(s string) bool {
	if len(s) < 1 || len(s) > 128 {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != ':' && r != '.' && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

func claveUsable(c Clave) bool { return refValida(c.Ref) && c.Material != [32]byte{} && !c.Revocada }

// cargar valida la separación: referencias únicas y materiales distintos.
func (a *Adaptador) cargar(ctx context.Context) (ClavesAspirantes, error) {
	if a == nil || a.fuente == nil || ctx == nil || ctx.Err() != nil {
		return ClavesAspirantes{}, ErrCriptoNoDisponible
	}
	c, err := a.fuente.CargarClavesAspirantes(ctx)
	if err != nil || !claveUsable(c.CifradoActivo) || !claveUsable(c.Indice) || !claveUsable(c.SemanticaActiva) ||
		len(c.CifradoRetenidas) > 8 || len(c.SemanticaRetenidas) > 8 {
		return ClavesAspirantes{}, ErrCriptoNoDisponible
	}
	refs := map[string]bool{}
	materiales := map[[32]byte]bool{}
	todas := append(append([]Clave{c.CifradoActivo, c.Indice, c.SemanticaActiva}, c.CifradoRetenidas...), c.SemanticaRetenidas...)
	for _, k := range todas {
		if !refValida(k.Ref) || k.Material == [32]byte{} || refs[k.Ref] || materiales[k.Material] {
			return ClavesAspirantes{}, ErrCriptoNoDisponible
		}
		refs[k.Ref], materiales[k.Material] = true, true
	}
	return c, nil
}

func aadValor(claveRef, asp string, campo domain.CampoFicha, version uint64) []byte {
	return []byte("vec.aspirantes.valor.v1\x00" + claveRef + "\x00" + asp + "\x00" + string(campo) + "\x00" + strconv.FormatUint(version, 10))
}

func aadDocumento(claveRef, asp, doc string) []byte {
	return []byte("vec.aspirantes.documento.v1\x00" + claveRef + "\x00" + asp + "\x00" + doc)
}

func (a *Adaptador) cifrar(k Clave, aad, claro []byte) (ports.SobreCifrado, error) {
	if len(claro) == 0 || len(claro) > MaximoClaro {
		return ports.SobreCifrado{}, ErrCriptoNoDisponible
	}
	bloque, err := aes.NewCipher(k.Material[:])
	if err != nil {
		return ports.SobreCifrado{}, ErrCriptoNoDisponible
	}
	gcm, err := cipher.NewGCM(bloque)
	if err != nil {
		return ports.SobreCifrado{}, ErrCriptoNoDisponible
	}
	nonce := make([]byte, largoNonce)
	if _, err := io.ReadFull(a.azar, nonce); err != nil {
		return ports.SobreCifrado{}, ErrCriptoNoDisponible
	}
	return ports.SobreCifrado{ClaveRef: k.Ref, Nonce: nonce, Cifrado: gcm.Seal(nil, nonce, claro, aad)}, nil
}

func descifrar(claves ClavesAspirantes, s ports.SobreCifrado, aad func(string) []byte) ([]byte, error) {
	if len(s.Nonce) != largoNonce || len(s.Cifrado) <= largoEtiqueta || len(s.Cifrado) > MaximoClaro+largoEtiqueta {
		return nil, ErrCriptoNoDisponible
	}
	var k Clave
	for _, c := range append([]Clave{claves.CifradoActivo}, claves.CifradoRetenidas...) {
		if c.Ref == s.ClaveRef {
			k = c
			break
		}
	}
	if !claveUsable(k) {
		return nil, ErrCriptoNoDisponible
	}
	bloque, err := aes.NewCipher(k.Material[:])
	if err != nil {
		return nil, ErrCriptoNoDisponible
	}
	gcm, err := cipher.NewGCM(bloque)
	if err != nil {
		return nil, ErrCriptoNoDisponible
	}
	claro, err := gcm.Open(nil, s.Nonce, s.Cifrado, aad(k.Ref))
	if err != nil {
		return nil, ErrCriptoNoDisponible
	}
	return claro, nil
}

func (a *Adaptador) CifrarValor(ctx context.Context, asp string, campo domain.CampoFicha, version uint64, claro []byte) (ports.SobreCifrado, error) {
	if !domain.ReferenciaAspiranteValida(asp) || !campo.Valido() || version == 0 {
		return ports.SobreCifrado{}, ErrCriptoNoDisponible
	}
	c, err := a.cargar(ctx)
	if err != nil {
		return ports.SobreCifrado{}, err
	}
	return a.cifrar(c.CifradoActivo, aadValor(c.CifradoActivo.Ref, asp, campo, version), claro)
}

func (a *Adaptador) DescifrarValor(ctx context.Context, asp string, campo domain.CampoFicha, version uint64, s ports.SobreCifrado) ([]byte, error) {
	if !domain.ReferenciaAspiranteValida(asp) || !campo.Valido() || version == 0 {
		return nil, ErrCriptoNoDisponible
	}
	c, err := a.cargar(ctx)
	if err != nil {
		return nil, err
	}
	return descifrar(c, s, func(ref string) []byte { return aadValor(ref, asp, campo, version) })
}

func (a *Adaptador) CifrarDocumento(ctx context.Context, asp, doc string, claro []byte) (ports.SobreCifrado, error) {
	if !domain.ReferenciaAspiranteValida(asp) || !domain.ReferenciaDocumentoValida(doc) {
		return ports.SobreCifrado{}, ErrCriptoNoDisponible
	}
	c, err := a.cargar(ctx)
	if err != nil {
		return ports.SobreCifrado{}, err
	}
	return a.cifrar(c.CifradoActivo, aadDocumento(c.CifradoActivo.Ref, asp, doc), claro)
}

func (a *Adaptador) DescifrarDocumento(ctx context.Context, asp, doc string, s ports.SobreCifrado) ([]byte, error) {
	if !domain.ReferenciaAspiranteValida(asp) || !domain.ReferenciaDocumentoValida(doc) {
		return nil, ErrCriptoNoDisponible
	}
	c, err := a.cargar(ctx)
	if err != nil {
		return nil, err
	}
	return descifrar(c, s, func(ref string) []byte { return aadDocumento(ref, asp, doc) })
}

// IndiceDocumento es HMAC-SHA256 con clave propia sobre tipo, país y número
// normalizado. Sin la clave no se puede comprobar un DNI candidato.
func (a *Adaptador) IndiceDocumento(ctx context.Context, d domain.DocumentoIdentidad) (ports.IndiceDocumento, error) {
	if d.Validar() != nil {
		return ports.IndiceDocumento{}, ErrCriptoNoDisponible
	}
	c, err := a.cargar(ctx)
	if err != nil {
		return ports.IndiceDocumento{}, err
	}
	mac := hmac.New(sha256.New, c.Indice.Material[:])
	_, _ = io.WriteString(mac, "vec.aspirantes.documento.indice.v1\x00"+string(d.Tipo)+"\x00"+d.Pais+"\x00"+d.Numero)
	return ports.IndiceDocumento{ClaveRef: c.Indice.Ref, Valor: hex.EncodeToString(mac.Sum(nil))}, nil
}

// SellarHuella devuelve el HMAC de la preimagen con la clave activa y con
// cada retenida, para reconocer una repetición tras rotar.
func (a *Adaptador) SellarHuella(ctx context.Context, preimagen []byte) (ports.HuellasSemanticas, error) {
	if len(preimagen) == 0 || len(preimagen) > 16<<10 {
		return ports.HuellasSemanticas{}, ErrCriptoNoDisponible
	}
	c, err := a.cargar(ctx)
	if err != nil {
		return ports.HuellasSemanticas{}, err
	}
	sellar := func(k Clave) ports.HuellaSemantica {
		mac := hmac.New(sha256.New, k.Material[:])
		_, _ = io.WriteString(mac, "vec.aspirantes.huella.v1\x00")
		_, _ = mac.Write(preimagen)
		return ports.HuellaSemantica{ClaveRef: k.Ref, Valor: hex.EncodeToString(mac.Sum(nil))}
	}
	h := ports.HuellasSemanticas{Activa: sellar(c.SemanticaActiva), Retenidas: []ports.HuellaSemantica{}}
	for _, k := range c.SemanticaRetenidas {
		if !k.Revocada {
			h.Retenidas = append(h.Retenidas, sellar(k))
		}
	}
	return h, nil
}
