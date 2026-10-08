package ordenescopias

import (
	"context"
	"errors"
	"reflect"
	"time"
	dominio "vec-diputacion-granada/internal/modules/administracion/domain/ordenescopias"
	puerto "vec-diputacion-granada/internal/modules/administracion/ports/ordenescopias"
)

var ErrDenegada = errors.New("orden_denegada")
var ErrCommit = errors.New("orden_commit_no_acreditado")

// Servicio publishes a committed order and verifies it before the external
// acceptance. It neither grants permissions nor stops or restores PostgreSQL.
type Servicio struct {
	lector    puerto.LectorCommit
	cripto    puerto.Autenticador
	anclaje   puerto.Anclaje
	aceptador puerto.AceptadorExterno
}

func Nuevo(l puerto.LectorCommit, c puerto.Autenticador, a puerto.Anclaje, e puerto.AceptadorExterno) (*Servicio, error) {
	if nulo(l) || nulo(c) || nulo(a) || nulo(e) {
		return nil, ErrDenegada
	}
	return &Servicio{l, c, a, e}, nil
}
func (s *Servicio) Publicar(ctx context.Context, esperada dominio.Orden, t time.Time) (puerto.Sobre, error) {
	if !s.valido(ctx) || esperada.ValidarEn(t) != nil {
		return puerto.Sobre{}, ErrDenegada
	}
	d, _ := esperada.Datos()
	obtenida, err := s.lector.LeerOrdenComprometida(ctx, d.Orden)
	h, _ := esperada.SHA256()
	actual, e := obtenida.SHA256()
	if err != nil || e != nil || h != actual {
		return puerto.Sobre{}, ErrCommit
	}
	if s.anclaje.ValidarActual(ctx, obtenida, t) != nil || ctx.Err() != nil {
		return puerto.Sobre{}, ErrDenegada
	}
	firma, err := s.cripto.Sellar(ctx, obtenida)
	if err != nil || len(firma) == 0 || len(firma) > 8192 || s.cripto.Verificar(ctx, obtenida, firma) != nil || ctx.Err() != nil {
		return puerto.Sobre{}, ErrDenegada
	}
	return puerto.Sobre{Orden: obtenida, Firma: append([]byte(nil), firma...)}, nil
}
func (s *Servicio) Verificar(ctx context.Context, sobre puerto.Sobre, t time.Time) error {
	if !s.valido(ctx) || sobre.Orden.ValidarEn(t) != nil || len(sobre.Firma) == 0 || len(sobre.Firma) > 8192 {
		return ErrDenegada
	}
	if s.cripto.Verificar(ctx, sobre.Orden, append([]byte(nil), sobre.Firma...)) != nil || s.anclaje.ValidarActual(ctx, sobre.Orden, t) != nil || ctx.Err() != nil {
		return ErrDenegada
	}
	return nil
}
func (s *Servicio) Aceptar(ctx context.Context, sobre puerto.Sobre, t time.Time) (puerto.Aceptacion, error) {
	if s.Verificar(ctx, sobre, t) != nil {
		return puerto.Aceptacion{}, ErrDenegada
	}
	// CS07 must still enforce UNIQUE and fencing under its own lock/fsync.
	r, err := s.aceptador.AceptarOrden(ctx, sobre.Orden)
	d, _ := sobre.Orden.Datos()
	h, _ := sobre.Orden.SHA256()
	if err != nil || r.Orden != d.Orden || r.SHA256 != h || r.Recibo == "" || ctx.Err() != nil {
		return puerto.Aceptacion{}, ErrDenegada
	}
	return r, nil
}
func (s *Servicio) valido(ctx context.Context) bool {
	return s != nil && ctx != nil && ctx.Err() == nil && !nulo(s.lector) && !nulo(s.cripto) && !nulo(s.anclaje) && !nulo(s.aceptador)
}
func nulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Func, reflect.Slice, reflect.Chan:
		return r.IsNil()
	}
	return false
}
