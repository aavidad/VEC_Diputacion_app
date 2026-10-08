package ordenescopias

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	domain "vec-diputacion-granada/internal/modules/administracion/domain/ordenescopias"
	destino "vec-diputacion-granada/internal/modules/administracion/ports/destinocopias"
	puerto "vec-diputacion-granada/internal/modules/administracion/ports/ordenescopias"
)

var ErrProtector = errors.New("orden_protector_no_disponible")

// ProtectorOrden reuses CS03 authenticated encryption. Trusted composition must
// provide the existing KMS with a key dedicated to orders, not the component key.
// The envelope is authenticated transport, never legal document signing.
type ProtectorOrden struct{ protector destino.Protector }

func NuevoProtectorOrden(p destino.Protector) (*ProtectorOrden, error) {
	if p == nil {
		return nil, ErrProtector
	}
	v := reflect.ValueOf(p)
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return nil, ErrProtector
	}
	return &ProtectorOrden{p}, nil
}
func (p *ProtectorOrden) Sellar(ctx context.Context, o domain.Orden) ([]byte, error) {
	v, b, err := material(o)
	if err != nil || p == nil || p.protector == nil {
		return nil, ErrProtector
	}
	return p.protector.Proteger(ctx, v, b)
}
func (p *ProtectorOrden) Verificar(ctx context.Context, o domain.Orden, sobre []byte) error {
	if p == nil || p.protector == nil || ctx == nil || ctx.Err() != nil || len(sobre) == 0 || len(sobre) > 8192 {
		return ErrProtector
	}
	v, expected, err := material(o)
	if err != nil {
		return ErrProtector
	}
	cleartext, err := p.protector.Recuperar(ctx, v, append([]byte(nil), sobre...))
	defer clear(cleartext)
	if err != nil || !bytes.Equal(cleartext, expected) || ctx.Err() != nil {
		return ErrProtector
	}
	return nil
}
func material(o domain.Orden) (destino.Vinculo, []byte, error) {
	d, err := o.Datos()
	if err != nil {
		return destino.Vinculo{}, nil, ErrProtector
	}
	h, err := o.SHA256()
	if err != nil {
		return destino.Vinculo{}, nil, ErrProtector
	}
	b, err := o.Bytes()
	if err != nil {
		return destino.Vinculo{}, nil, ErrProtector
	}
	cleartext, err := json.Marshal(struct {
		Esquema string          `json:"esquema"`
		Orden   json.RawMessage `json:"orden"`
	}{"vec.administracion.sobre-orden-copias.v1", b})
	return destino.Vinculo{ConjuntoRef: d.Operacion, ComponenteRef: d.Orden, Posicion: d.Fence, ManifiestoSHA256: h}, cleartext, err
}

var _ puerto.Autenticador = (*ProtectorOrden)(nil)
