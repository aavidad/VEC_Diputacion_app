package almacen

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrIdentificadorConectorInvalido  = errors.New("vec: identificador de conector de almacen invalido")
	ErrConectorAlmacenYaRegistrado    = errors.New("vec: conector de almacen ya registrado")
	ErrConectorAlmacenNoRegistrado    = errors.New("vec: conector de almacen no registrado")
	ErrFabricaConectorAlmacenInvalida = errors.New("vec: fabrica de conector de almacen invalida")
	ErrRegistroConectoresInvalido     = errors.New("vec: registro de conectores de almacen invalido")
)

// ConfiguracionConectorAlmacen solo circula por la raiz de composicion. El
// nucleo y los modulos no reciben direcciones, buckets ni credenciales.
type ConfiguracionConectorAlmacen map[string]string

type FabricaConectorAlmacen func(context.Context, ConfiguracionConectorAlmacen) (ports.AlmacenObjetos, error)

// RegistroConectoresAlmacen permite seleccionar por configuracion cualquiera
// de los conectores instalados. Incorporar otro producto anade una fabrica,
// no condicionales en el nucleo ni cambios en los casos de uso.
type RegistroConectoresAlmacen struct {
	mu       sync.RWMutex
	fabricas map[string]FabricaConectorAlmacen
}

func NuevoRegistroConectoresAlmacen() *RegistroConectoresAlmacen {
	return &RegistroConectoresAlmacen{fabricas: make(map[string]FabricaConectorAlmacen)}
}

func (r *RegistroConectoresAlmacen) Registrar(identificador string, fabrica FabricaConectorAlmacen) error {
	if r == nil {
		return ErrRegistroConectoresInvalido
	}
	identificador = strings.TrimSpace(identificador)
	if !identificadorConectorValido(identificador) {
		return ErrIdentificadorConectorInvalido
	}
	if fabrica == nil {
		return ErrFabricaConectorAlmacenInvalida
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, existe := r.fabricas[identificador]; existe {
		return ErrConectorAlmacenYaRegistrado
	}
	r.fabricas[identificador] = fabrica
	return nil
}

func (r *RegistroConectoresAlmacen) Crear(
	ctx context.Context,
	identificador string,
	configuracion ConfiguracionConectorAlmacen,
	requisitos ports.RequisitosAlmacenObjetos,
) (ports.AlmacenObjetos, error) {
	if r == nil {
		return nil, ErrRegistroConectoresInvalido
	}
	if ctx == nil {
		return nil, ports.ErrSolicitudAlmacenInvalida
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	identificador = strings.TrimSpace(identificador)
	if !identificadorConectorValido(identificador) {
		return nil, ErrIdentificadorConectorInvalido
	}
	r.mu.RLock()
	fabrica, existe := r.fabricas[identificador]
	r.mu.RUnlock()
	if !existe {
		return nil, ErrConectorAlmacenNoRegistrado
	}
	conector, err := fabrica(ctx, clonarConfiguracion(configuracion))
	if err != nil {
		return nil, fmt.Errorf("crear conector de almacen %s: %w", identificador, err)
	}
	if dependenciaAlmacenNula(conector) {
		return nil, ErrFabricaConectorAlmacenInvalida
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	capacidades, err := conector.Capacidades(ctx)
	if err != nil {
		return nil, fmt.Errorf("consultar capacidades del conector %s: %w", identificador, err)
	}
	if capacidades.ConectorID != identificador {
		return nil, ErrFabricaConectorAlmacenInvalida
	}
	if err := ports.VerificarCapacidadesAlmacen(capacidades, requisitos); err != nil {
		return nil, fmt.Errorf("verificar capacidades del conector %s: %w", identificador, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return conector, nil
}

func (r *RegistroConectoresAlmacen) Listar() []string {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	identificadores := make([]string, 0, len(r.fabricas))
	for identificador := range r.fabricas {
		identificadores = append(identificadores, identificador)
	}
	sort.Strings(identificadores)
	return identificadores
}

func dependenciaAlmacenNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

func clonarConfiguracion(configuracion ConfiguracionConectorAlmacen) ConfiguracionConectorAlmacen {
	clon := make(ConfiguracionConectorAlmacen, len(configuracion))
	for clave, valor := range configuracion {
		clon[clave] = valor
	}
	return clon
}

func identificadorConectorValido(valor string) bool {
	if len(valor) < 2 || len(valor) > 64 || valor != strings.ToLower(valor) || valor[0] < 'a' || valor[0] > 'z' {
		return false
	}
	for _, caracter := range valor[1:] {
		if (caracter < 'a' || caracter > 'z') && (caracter < '0' || caracter > '9') && caracter != '-' && caracter != '_' {
			return false
		}
	}
	return true
}
