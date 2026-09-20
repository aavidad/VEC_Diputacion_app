package httpseguridad

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
)

// claveCapsulaPeticionSesion es deliberadamente privada: una confirmacion
// durable solo circula por el contexto que emitio este adaptador.
type claveCapsulaPeticionSesion struct{}

type capsulaPeticionSesionVinculada struct {
	servicio     *ServicioPeticionSesion
	canal        string
	confirmacion ConfirmacionPeticionSesion
	consumida    atomic.Bool
}

func (*capsulaPeticionSesionVinculada) String() string {
	return "[CÁPSULA DE PETICIÓN CONFIDENCIAL]"
}

func (*capsulaPeticionSesionVinculada) GoString() string {
	return "[CÁPSULA DE PETICIÓN CONFIDENCIAL]"
}

func (*capsulaPeticionSesionVinculada) LogValue() slog.Value {
	return slog.StringValue("[CÁPSULA DE PETICIÓN CONFIDENCIAL]")
}

func (*capsulaPeticionSesionVinculada) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte("[CÁPSULA DE PETICIÓN CONFIDENCIAL]"))
}

// ResolverYVincular consume y revalida una asercion una sola vez y enlaza su
// confirmacion durable al contexto de esta misma peticion. No acepta una
// confirmacion del llamador ni crea una sesion.
func (s *ServicioPeticionSesion) ResolverYVincular(
	ctx context.Context,
	sobre []byte,
	canal CanalProxyAutenticado,
	metodo, destino string,
	cuerpo []byte,
) (context.Context, error) {
	if ctx == nil {
		return nil, ErrAsercionPeticionNoValida
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if ctx.Value(claveCapsulaPeticionSesion{}) != nil {
		return nil, ErrAsercionPeticionNoValida
	}
	confirmacion, err := s.Resolver(ctx, sobre, canal, metodo, destino, cuerpo)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.reloj == nil || canal.validar(s.autoridadIdentidad) != nil ||
		!confirmacionPeticionValida(confirmacion, s.reloj.Ahora()) {
		return nil, ErrAsercionPeticionNoValida
	}
	return context.WithValue(ctx, claveCapsulaPeticionSesion{}, &capsulaPeticionSesionVinculada{
		servicio: s, canal: canal.ReferenciaVinculacion(), confirmacion: confirmacion,
	}), nil
}

// ExtraerConfirmacionVinculada entrega una copia de la confirmacion asociada a
// esta instancia y la consume de forma atomica. La vigencia se vuelve a cerrar
// contra el reloj configurado; no hay una segunda revalidacion durable.
func (s *ServicioPeticionSesion) ExtraerConfirmacionVinculada(
	ctx context.Context,
) (ConfirmacionPeticionSesion, error) {
	if ctx == nil {
		return ConfirmacionPeticionSesion{}, ErrAsercionPeticionNoValida
	}
	if err := ctx.Err(); err != nil {
		return ConfirmacionPeticionSesion{}, err
	}
	capsula, ok := ctx.Value(claveCapsulaPeticionSesion{}).(*capsulaPeticionSesionVinculada)
	if !ok || capsula == nil || s == nil || capsula.servicio != s ||
		capsula.canal == "" || s.reloj == nil ||
		!confirmacionPeticionValida(capsula.confirmacion, s.reloj.Ahora()) {
		return ConfirmacionPeticionSesion{}, ErrAsercionPeticionNoValida
	}
	if !capsula.consumida.CompareAndSwap(false, true) {
		return ConfirmacionPeticionSesion{}, ErrAsercionPeticionNoValida
	}
	return capsula.confirmacion, nil
}
