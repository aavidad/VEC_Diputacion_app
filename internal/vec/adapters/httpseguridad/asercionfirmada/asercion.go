// Package asercionfirmada protege aserciones de inicio y peticion con Ed25519.
package asercionfirmada

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

var ErrAsercionFirmada = errors.New("asercion firmada no valida")

const algoritmoEdDSA = "EdDSA"

type ConfiguracionEmisor struct {
	ClaveID string
	Clave   ed25519.PrivateKey
}
type Emisor struct {
	claveID string
	clave   ed25519.PrivateKey
}

func NuevoEmisor(c ConfiguracionEmisor) (*Emisor, error) {
	if c.ClaveID == "" || strings.TrimSpace(c.ClaveID) != c.ClaveID || len(c.Clave) != ed25519.PrivateKeySize {
		return nil, ErrAsercionFirmada
	}
	return &Emisor{claveID: c.ClaveID, clave: append(ed25519.PrivateKey(nil), c.Clave...)}, nil
}

type ConfiguracionVerificador struct{ Claves map[string]ed25519.PublicKey }
type Verificador struct{ claves map[string]ed25519.PublicKey }

func NuevoVerificador(c ConfiguracionVerificador) (*Verificador, error) {
	if len(c.Claves) == 0 {
		return nil, ErrAsercionFirmada
	}
	r := &Verificador{claves: map[string]ed25519.PublicKey{}}
	for id, k := range c.Claves {
		if id == "" || len(k) != ed25519.PublicKeySize {
			return nil, ErrAsercionFirmada
		}
		r.claves[id] = append(ed25519.PublicKey(nil), k...)
	}
	return r, nil
}

type inicio struct {
	Tipo     string                               `json:"tipo"`
	Asercion httpseguridad.AsercionProxyIdentidad `json:"asercion"`
}
type Peticion struct {
	Emisor            string                   `json:"emisor"`
	Audiencia         string                   `json:"audiencia"`
	Superficie        httpseguridad.Superficie `json:"superficie"`
	SesionID          string                   `json:"sesion_id"`
	Metodo            string                   `json:"metodo"`
	Destino           string                   `json:"destino"`
	CuerpoSHA256      string                   `json:"cuerpo_sha256"`
	NonceSHA256       string                   `json:"nonce_sha256"`
	EmitidaEn         time.Time                `json:"emitida_en"`
	ExpiraEn          time.Time                `json:"expira_en"`
	CanalVinculadoRef string                   `json:"canal_vinculado_ref"`
}
type peticion struct {
	Tipo     string   `json:"tipo"`
	Peticion Peticion `json:"peticion"`
}

func (e *Emisor) EmitirInicio(ctx context.Context, a httpseguridad.AsercionProxyIdentidad) ([]byte, error) {
	if ctx == nil || ctx.Err() != nil {
		return nil, ErrAsercionFirmada
	}
	return e.emitir(inicio{"INICIO", a})
}
func (e *Emisor) EmitirPeticion(ctx context.Context, p Peticion) ([]byte, error) {
	if ctx == nil || ctx.Err() != nil {
		return nil, ErrAsercionFirmada
	}
	return e.emitir(peticion{"PETICION", p})
}
func (e *Emisor) emitir(valor any) ([]byte, error) {
	if e == nil || len(e.clave) != ed25519.PrivateKeySize {
		return nil, ErrAsercionFirmada
	}
	carga, err := json.Marshal(valor)
	if err != nil {
		return nil, ErrAsercionFirmada
	}
	opciones := (&jose.SignerOptions{}).WithType(jose.ContentType("VEC-IDENTIDAD-V1")).WithHeader(jose.HeaderKey("kid"), e.claveID)
	firmante, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.EdDSA, Key: e.clave}, opciones)
	if err != nil {
		return nil, ErrAsercionFirmada
	}
	firmado, err := firmante.Sign(carga)
	if err != nil {
		return nil, ErrAsercionFirmada
	}
	compacta, err := firmado.CompactSerialize()
	if err != nil {
		return nil, ErrAsercionFirmada
	}
	return []byte(compacta), nil
}
func (v *Verificador) Verificar(ctx context.Context, b []byte) (httpseguridad.AsercionProxyIdentidad, error) {
	var i inicio
	if !v.abrir(ctx, b, "INICIO", &i) {
		return httpseguridad.AsercionProxyIdentidad{}, ErrAsercionFirmada
	}
	return i.Asercion, nil
}
func (v *Verificador) VerificarPeticion(ctx context.Context, b []byte) (httpseguridad.AsercionPeticionVerificada, error) {
	var p peticion
	if !v.abrir(ctx, b, "PETICION", &p) {
		return httpseguridad.AsercionPeticionVerificada{}, ErrAsercionFirmada
	}
	x := p.Peticion
	return httpseguridad.AsercionPeticionVerificada{Emisor: x.Emisor, Audiencia: x.Audiencia, Superficie: x.Superficie, SesionID: x.SesionID, Metodo: x.Metodo, Destino: x.Destino, CuerpoSHA256: x.CuerpoSHA256, NonceSHA256: x.NonceSHA256, EmitidaEn: x.EmitidaEn, ExpiraEn: x.ExpiraEn, CanalVinculadoRef: x.CanalVinculadoRef}, nil
}
func (v *Verificador) abrir(ctx context.Context, b []byte, tipo string, dest any) bool {
	if v == nil || ctx == nil || ctx.Err() != nil || len(b) == 0 || len(b) > 64*1024 {
		return false
	}
	firmado, err := jose.ParseSigned(string(b), []jose.SignatureAlgorithm{jose.EdDSA})
	if err != nil || len(firmado.Signatures) != 1 {
		return false
	}
	tip, tieneTip := firmado.Signatures[0].Protected.ExtraHeaders[jose.HeaderType]
	if firmado.Signatures[0].Protected.Algorithm != algoritmoEdDSA || !tieneTip || tip != "VEC-IDENTIDAD-V1" || firmado.Signatures[0].Protected.KeyID == "" {
		return false
	}
	k := v.claves[firmado.Signatures[0].Protected.KeyID]
	if len(k) != ed25519.PublicKeySize {
		return false
	}
	c, err := firmado.Verify(k)
	if err != nil || !decodificarEstricto(c, dest) {
		return false
	}
	switch x := dest.(type) {
	case *inicio:
		return x.Tipo == tipo
	case *peticion:
		return x.Tipo == tipo
	}
	return false
}

func decodificarEstricto(contenido []byte, destino any) bool {
	d := json.NewDecoder(strings.NewReader(string(contenido)))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil {
		return false
	}
	return d.Decode(&struct{}{}) == io.EOF
}
func HuellaCuerpo(cuerpo []byte) string {
	s := sha256.Sum256(cuerpo)
	return hex.EncodeToString(s[:])
}
