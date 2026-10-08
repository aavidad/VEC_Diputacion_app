package ordenescopias

import (
	"context"
	"errors"
	"reflect"
	domain "vec-diputacion-granada/internal/modules/administracion/domain/ordenescopias"
	puerto "vec-diputacion-granada/internal/modules/administracion/ports/ordenescopias"
	registro "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

// RegistroExterno binds an already verified order to CS07's external journal.
// The declaration is resolved by trusted composition, never copied from an order
// submitted by an HTTP client. This adapter does not grant authorization.
type RegistroExterno struct {
	registro    registro.AceptadorOrden
	declaracion registro.Declaracion
}

func NuevoRegistroExterno(r registro.AceptadorOrden, d registro.Declaracion) (*RegistroExterno, error) {
	if r == nil || d.Actor == "" || d.Correlacion == "" {
		return nil, errors.New("orden_registro_invalido")
	}
	v := reflect.ValueOf(r)
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return nil, errors.New("orden_registro_invalido")
	}
	return &RegistroExterno{r, d}, nil
}
func (r *RegistroExterno) AceptarOrden(ctx context.Context, o domain.Orden) (puerto.Aceptacion, error) {
	if r == nil || ctx == nil || ctx.Err() != nil {
		return puerto.Aceptacion{}, errors.New("orden_registro_invalido")
	}
	d, err := o.Datos()
	if err != nil {
		return puerto.Aceptacion{}, err
	}
	h, err := o.SHA256()
	if err != nil {
		return puerto.Aceptacion{}, err
	}
	result, err := r.registro.AceptarOrden(ctx, r.declaracion, registro.RecepcionOrden{Orden: d.Orden, Operacion: d.Operacion, SHA256: h, SolicitudSHA256: d.SolicitudSHA256, Destino: d.Destino, Epoca: d.Epoca, Fence: d.Fence})
	if err != nil {
		return puerto.Aceptacion{}, err
	}
	return puerto.Aceptacion{Orden: result.Orden.Orden, SHA256: result.Orden.SHA256, Recibo: result.Recibo.Referencia, Replay: result.Replay}, nil
}

var _ puerto.AceptadorExterno = (*RegistroExterno)(nil)
