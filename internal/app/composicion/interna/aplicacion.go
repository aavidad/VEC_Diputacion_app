package interna

import (
	"context"
	"errors"
	"reflect"
	"sync"
)

var ErrAplicacionInternaNoDisponible = errors.New(
	"composicion interna: ciclo de vida de aplicacion no disponible",
)

type estadoAplicacionInterna uint8

const (
	estadoAplicacionInternaNueva estadoAplicacionInterna = iota
	estadoAplicacionInternaEscuchando
	estadoAplicacionInternaTerminada
	estadoAplicacionInternaCerrada
)

// recursoCerrableAplicacionInterna no puede implementarse desde cmd ni desde
// adaptadores. La raiz futura debera envolver cada recurso concreto dentro de
// este paquete y transferir su propiedad al constructor privado.
type recursoCerrableAplicacionInterna interface {
	reclamarPropiedad() bool
	cerrar() error
}

// AplicacionInterna posee el servidor y los recursos que lo sustentan. No
// publica handlers, listeners, pools ni proveedores, y rechaza copias del
// valor para conservar un unico propietario del ciclo de vida.
type AplicacionInterna struct {
	propietario *AplicacionInterna
	servidor    *ServidorInterno
	recursos    []recursoCerrableAplicacionInterna

	mu        sync.Mutex
	estado    estadoAplicacionInterna
	terminado chan struct{}

	cerrarUnaVez sync.Once
	errorCierre  error
}

type identidadRecursoAplicacionInterna struct {
	tipo    reflect.Type
	puntero uintptr
}

// nuevaAplicacionInterna es deliberadamente privada. Su uso futuro pertenece
// a la raiz del mismo paquete una vez que existan dependencias productivas
// acreditadas; este corte no modifica NuevoServidor ni abre produccion.
//
// La llamada transfiere el servidor y todos los recursos declarados. Ante
// cualquier fallo, cancela el servidor valido y cierra una sola vez cada
// recurso valido, en orden inverso.
func nuevaAplicacionInterna(
	servidor *ServidorInterno,
	recursos ...recursoCerrableAplicacionInterna,
) (*AplicacionInterna, error) {
	inventario, valido := inventariarRecursosAplicacionInterna(recursos)
	servidorValido := validarServidorInterno(servidor) == nil
	servidorReclamado, recursosReclamados, propiedadExclusiva :=
		reclamarPropiedadAplicacionInterna(servidor, inventario)
	if !servidorValido || !valido || !propiedadExclusiva {
		if servidorReclamado {
			_ = servidor.Apagar(context.Background())
		}
		_ = cerrarRecursosAplicacionInterna(recursosReclamados)
		return nil, ErrAplicacionInternaNoDisponible
	}
	aplicacion := &AplicacionInterna{
		servidor:  servidor,
		recursos:  append([]recursoCerrableAplicacionInterna(nil), inventario...),
		terminado: make(chan struct{}),
	}
	aplicacion.propietario = aplicacion
	return aplicacion, nil
}

// reclamarPropiedadAplicacionInterna reserva cada propietario en el propio
// objeto. La marca no se libera tras el cierre: un recurso transferido no puede
// reutilizarse, pero la aplicacion tampoco crea un registro global que retenga
// pools, proveedores o material sensible.
func reclamarPropiedadAplicacionInterna(
	servidor *ServidorInterno,
	recursos []recursoCerrableAplicacionInterna,
) (bool, []recursoCerrableAplicacionInterna, bool) {
	servidorReclamado := servidor != nil && servidor.propiedadAplicacion != nil &&
		servidor.propiedadAplicacion.CompareAndSwap(false, true)
	exclusiva := servidorReclamado

	reclamados := make([]recursoCerrableAplicacionInterna, 0, len(recursos))
	for _, recurso := range recursos {
		if !recurso.reclamarPropiedad() {
			exclusiva = false
			continue
		}
		reclamados = append(reclamados, recurso)
	}
	return servidorReclamado, reclamados, exclusiva
}

func inventariarRecursosAplicacionInterna(
	recursos []recursoCerrableAplicacionInterna,
) ([]recursoCerrableAplicacionInterna, bool) {
	if len(recursos) == 0 {
		return nil, false
	}
	inventario := make([]recursoCerrableAplicacionInterna, 0, len(recursos))
	vistos := make(map[identidadRecursoAplicacionInterna]struct{}, len(recursos))
	valido := true
	for _, recurso := range recursos {
		valor := reflect.ValueOf(recurso)
		if !valor.IsValid() || valor.Kind() != reflect.Pointer || valor.IsNil() {
			valido = false
			continue
		}
		identidad := identidadRecursoAplicacionInterna{
			tipo:    valor.Type(),
			puntero: valor.Pointer(),
		}
		if _, repetido := vistos[identidad]; repetido {
			valido = false
			continue
		}
		vistos[identidad] = struct{}{}
		inventario = append(inventario, recurso)
	}
	return inventario, valido && len(inventario) == len(recursos)
}

// EscucharYServir delega exclusivamente en la capsula C4 poseida. La
// terminacion se publica despues de que esa llamada haya retornado.
func (aplicacion *AplicacionInterna) EscucharYServir() error {
	if !aplicacion.valida() {
		return ErrAplicacionInternaNoDisponible
	}
	aplicacion.mu.Lock()
	if aplicacion.estado != estadoAplicacionInternaNueva {
		aplicacion.mu.Unlock()
		return ErrAplicacionInternaNoDisponible
	}
	aplicacion.estado = estadoAplicacionInternaEscuchando
	aplicacion.mu.Unlock()

	err := aplicacion.servidor.EscucharYServir()

	aplicacion.mu.Lock()
	aplicacion.estado = estadoAplicacionInternaTerminada
	close(aplicacion.terminado)
	aplicacion.mu.Unlock()
	return normalizarErrorAplicacionInterna(err)
}

// Apagar detiene el servidor y no retorna exito hasta que EscucharYServir haya
// publicado su terminacion. Antes de escuchar cancela atomicamente el unico
// intento posible.
func (aplicacion *AplicacionInterna) Apagar(ctx context.Context) error {
	if !aplicacion.valida() || ctx == nil {
		return ErrAplicacionInternaNoDisponible
	}
	aplicacion.mu.Lock()
	switch aplicacion.estado {
	case estadoAplicacionInternaNueva:
		err := aplicacion.servidor.Apagar(ctx)
		if err == nil {
			aplicacion.estado = estadoAplicacionInternaTerminada
			close(aplicacion.terminado)
		}
		aplicacion.mu.Unlock()
		return normalizarErrorAplicacionInterna(err)
	case estadoAplicacionInternaEscuchando:
		terminado := aplicacion.terminado
		aplicacion.mu.Unlock()
		if err := aplicacion.servidor.Apagar(ctx); err != nil {
			return ErrAplicacionInternaNoDisponible
		}
		return esperarTerminacionAplicacionInterna(ctx, terminado)
	case estadoAplicacionInternaTerminada, estadoAplicacionInternaCerrada:
		aplicacion.mu.Unlock()
		return nil
	default:
		aplicacion.mu.Unlock()
		return ErrAplicacionInternaNoDisponible
	}
}

func esperarTerminacionAplicacionInterna(
	ctx context.Context,
	terminado <-chan struct{},
) error {
	select {
	case <-terminado:
		return nil
	default:
	}
	select {
	case <-terminado:
		return nil
	case <-ctx.Done():
		return ErrAplicacionInternaNoDisponible
	}
}

// Cerrar libera en orden inverso y una sola vez. Durante una escucha activa
// falla cerrado: el llamador debe ejecutar primero Apagar con un contexto
// acotado.
func (aplicacion *AplicacionInterna) Cerrar() error {
	if !aplicacion.valida() {
		return ErrAplicacionInternaNoDisponible
	}
	aplicacion.mu.Lock()
	switch aplicacion.estado {
	case estadoAplicacionInternaNueva:
		// No existe escucha que esperar y mantener el mutex impide que pueda
		// comenzar una mientras se cancela la capsula C4.
		if aplicacion.servidor.Apagar(context.Background()) != nil {
			aplicacion.mu.Unlock()
			return ErrAplicacionInternaNoDisponible
		}
		aplicacion.estado = estadoAplicacionInternaTerminada
		close(aplicacion.terminado)
		aplicacion.estado = estadoAplicacionInternaCerrada
	case estadoAplicacionInternaTerminada:
		aplicacion.estado = estadoAplicacionInternaCerrada
	case estadoAplicacionInternaCerrada:
	case estadoAplicacionInternaEscuchando:
		aplicacion.mu.Unlock()
		return ErrAplicacionInternaNoDisponible
	default:
		aplicacion.mu.Unlock()
		return ErrAplicacionInternaNoDisponible
	}
	aplicacion.mu.Unlock()

	aplicacion.cerrarUnaVez.Do(func() {
		aplicacion.errorCierre = cerrarRecursosAplicacionInterna(
			aplicacion.recursos,
		)
	})
	return normalizarErrorAplicacionInterna(aplicacion.errorCierre)
}

func (aplicacion *AplicacionInterna) valida() bool {
	return aplicacion != nil && aplicacion.propietario == aplicacion &&
		aplicacion.servidor != nil && aplicacion.terminado != nil &&
		len(aplicacion.recursos) != 0
}

func cerrarRecursosAplicacionInterna(
	recursos []recursoCerrableAplicacionInterna,
) error {
	conError := false
	for indice := len(recursos) - 1; indice >= 0; indice-- {
		if cerrarRecursoAplicacionInterna(recursos[indice]) != nil {
			conError = true
		}
	}
	if conError {
		return ErrAplicacionInternaNoDisponible
	}
	return nil
}

func cerrarRecursoAplicacionInterna(
	recurso recursoCerrableAplicacionInterna,
) (err error) {
	defer func() {
		if recover() != nil {
			err = ErrAplicacionInternaNoDisponible
		}
	}()
	if recurso == nil {
		return ErrAplicacionInternaNoDisponible
	}
	if err := recurso.cerrar(); err != nil {
		return ErrAplicacionInternaNoDisponible
	}
	return nil
}

func normalizarErrorAplicacionInterna(err error) error {
	if err == nil {
		return nil
	}
	return ErrAplicacionInternaNoDisponible
}
