package protectorstagingdesarrollo

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	postgres "vec-diputacion-granada/internal/modules/bolsa/adapters/postgresimportacionconvoca"
	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

const cifradoInfo = "vec/bolsa/importacion-convoca/cifrado/v1"
const derivacionInfo = "vec/bolsa/importacion-convoca/derivacion/v1"
const atestacionInfo = "vec/bolsa/importacion-convoca/atestacion/v1"

type Protector struct {
	cifrado, derivacion, atestacion          [32]byte
	cifradoRef, derivacionRef, atestacionRef string
}

func Nuevo(maestra [32]byte) (*Protector, error) {
	h := sha256.Sum256(maestra[:])
	suf := hex.EncodeToString(h[:4])
	p := &Protector{cifradoRef: "kms-desarrollo:" + cifradoInfo + ":" + suf, derivacionRef: "kms-desarrollo:" + derivacionInfo + ":" + suf, atestacionRef: "kms-desarrollo:" + atestacionInfo + ":" + suf}
	var err error
	if p.cifrado, err = clave(maestra, cifradoInfo); err != nil {
		return nil, err
	}
	if p.derivacion, err = clave(maestra, derivacionInfo); err != nil {
		return nil, err
	}
	p.atestacion, err = clave(maestra, atestacionInfo)
	if err != nil {
		return nil, err
	}
	return p, nil
}
func clave(m [32]byte, info string) ([32]byte, error) {
	var r [32]byte
	v, e := hkdf.Key(sha256.New, m[:], nil, info, 32)
	if e != nil {
		return r, e
	}
	copy(r[:], v)
	return r, nil
}
func (p *Protector) ProtegerStaging(ctx context.Context, s postgres.SolicitudProteccionStaging) (postgres.ResultadoProteccionStaging, error) {
	if p == nil || ctx == nil {
		return postgres.ResultadoProteccionStaging{}, postgres.ErrProtectorRequerido
	}
	if e := ctx.Err(); e != nil {
		return postgres.ResultadoProteccionStaging{}, e
	}
	a, e := p.aead()
	if e != nil {
		return postgres.ResultadoProteccionStaging{}, e
	}
	r := postgres.ResultadoProteccionStaging{Filas: make([]postgres.FilaStagingProtegida, len(s.Filas))}
	for i, f := range s.Filas {
		v, e := json.Marshal(f)
		if e != nil {
			return postgres.ResultadoProteccionStaging{}, e
		}
		n := make([]byte, a.NonceSize())
		if _, e = io.ReadFull(rand.Reader, n); e != nil {
			return postgres.ResultadoProteccionStaging{}, e
		}
		r.Filas[i] = postgres.FilaStagingProtegida{Numero: f.Numero, EsquemaProteccion: postgres.EsquemaProteccionStagingV1, ClaveRef: p.cifradoRef, ClaveDerivacionRef: p.derivacionRef, ClaveAtestacionRef: p.atestacionRef, Nonce: n, ContenidoCifrado: a.Seal(nil, n, v, aad(s.ImportacionRef, s.HuellaFicheroSHA256, s.Esquema, f.Numero)), DerivacionDocumentoHMACSHA256: p.mac(p.derivacion, []byte(f.Identidad.Documento)), AtestacionFilaHMACSHA256: p.mac(p.atestacion, append(aad(s.ImportacionRef, s.HuellaFicheroSHA256, s.Esquema, f.Numero), v...))}
		for j := range v {
			v[j] = 0
		}
	}
	return r, nil
}
func (p *Protector) RecuperarStaging(ctx context.Context, s postgres.SolicitudRecuperacionStaging) ([]dominio.FilaAceptada, error) {
	if p == nil || ctx == nil {
		return nil, postgres.ErrProtectorRequerido
	}
	a, e := p.aead()
	if e != nil {
		return nil, e
	}
	r := make([]dominio.FilaAceptada, len(s.Filas))
	for i, f := range s.Filas {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		if f.ClaveRef != p.cifradoRef || f.ClaveDerivacionRef != p.derivacionRef || f.ClaveAtestacionRef != p.atestacionRef {
			return nil, postgres.ErrMaterialNoConfiable
		}
		v, e := a.Open(nil, f.Nonce, f.ContenidoCifrado, aad(s.ImportacionRef, s.HuellaFicheroSHA256, s.Esquema, f.Numero))
		if e != nil {
			return nil, postgres.ErrMaterialNoConfiable
		}
		if subtle.ConstantTimeCompare(f.AtestacionFilaHMACSHA256, p.mac(p.atestacion, append(aad(s.ImportacionRef, s.HuellaFicheroSHA256, s.Esquema, f.Numero), v...))) != 1 {
			return nil, postgres.ErrMaterialNoConfiable
		}
		if e = json.Unmarshal(v, &r[i]); e != nil {
			return nil, postgres.ErrMaterialNoConfiable
		}
		if subtle.ConstantTimeCompare(f.DerivacionDocumentoHMACSHA256, p.mac(p.derivacion, []byte(r[i].Identidad.Documento))) != 1 {
			return nil, postgres.ErrMaterialNoConfiable
		}
	}
	return r, nil
}
func (p *Protector) aead() (cipher.AEAD, error) {
	b, e := aes.NewCipher(p.cifrado[:])
	if e != nil {
		return nil, e
	}
	return cipher.NewGCM(b)
}
func (p *Protector) mac(k [32]byte, v []byte) []byte {
	m := hmac.New(sha256.New, k[:])
	_, _ = m.Write(v)
	return m.Sum(nil)
}
func aad(i, h string, e dominio.EsquemaExportacion, n int) []byte {
	return []byte(fmt.Sprintf("%s\\x1f%s\\x1f%s\\x1f%d", i, h, e, n))
}

var _ postgres.ProtectorStagingConvoca = (*Protector)(nil)
