package destinocopias

import (
	"bytes"
	"context"
	jose "github.com/go-jose/go-jose/v4"
	app "vec-diputacion-granada/internal/modules/administracion/application/destinocopias"
	ports "vec-diputacion-granada/internal/modules/administracion/ports/destinocopias"
)

// ProtectorJWE delega todo cifrado autenticado en go-jose. La clave dedicada
// ya viene del KMS/composición; no se deriva ni se genera aquí.
type ProtectorJWE struct {
	clave        [32]byte
	ref, version string
	max          int64
}

func NuevoProtectorJWE(clave [32]byte, ref, version string, max int64) (*ProtectorJWE, error) {
	if clave == [32]byte{} || !app.ReferenciaOpaca(ref) || !app.ReferenciaOpaca(version) || max < 1 || max > (1<<30) {
		return nil, app.ErrConfiguracion
	}
	return &ProtectorJWE{clave, ref, version, max}, nil
}
func (p *ProtectorJWE) Proteger(ctx context.Context, v ports.Vinculo, claro []byte) ([]byte, error) {
	if p == nil || ctx == nil || ctx.Err() != nil || len(claro) == 0 || int64(len(claro)) > p.max {
		return nil, app.ErrMaterial
	}
	aad, err := p.aad(v)
	if err != nil {
		return nil, err
	}
	e, err := jose.NewEncrypter(jose.A256GCM, jose.Recipient{Algorithm: jose.A256KW, Key: p.clave[:], KeyID: p.ref}, nil)
	if err != nil {
		return nil, app.ErrNoDisponible
	}
	sobre, err := e.EncryptWithAuthData(claro, aad)
	if err != nil || ctx.Err() != nil {
		return nil, app.ErrNoDisponible
	}
	return []byte(sobre.FullSerialize()), nil
}
func (p *ProtectorJWE) Recuperar(ctx context.Context, v ports.Vinculo, cifrado []byte) ([]byte, error) {
	if p == nil || ctx == nil || ctx.Err() != nil || len(cifrado) == 0 || int64(len(cifrado)) > p.max*2+8192 {
		return nil, app.ErrMaterial
	}
	aad, err := p.aad(v)
	if err != nil {
		return nil, err
	}
	sobre, err := jose.ParseEncrypted(string(cifrado), []jose.KeyAlgorithm{jose.A256KW}, []jose.ContentEncryption{jose.A256GCM})
	if err != nil || sobre.Header.KeyID != p.ref || !bytes.Equal(sobre.GetAuthData(), aad) {
		return nil, app.ErrMaterial
	}
	claro, err := sobre.Decrypt(p.clave[:])
	if err != nil || ctx.Err() != nil || int64(len(claro)) > p.max {
		clear(claro)
		return nil, app.ErrMaterial
	}
	return claro, nil
}
func (p *ProtectorJWE) aad(v ports.Vinculo) ([]byte, error) {
	a, err := app.AAD(v)
	if err != nil {
		return nil, err
	}
	// Both key identifier and version are authenticated, beyond the JOSE kid.
	return append(append(append(a, '\n'), []byte(p.ref+"\n"+p.version)...), '\n'), nil
}

var _ ports.Protector = (*ProtectorJWE)(nil)
