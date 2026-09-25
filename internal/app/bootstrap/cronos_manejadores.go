package bootstrap

import (
	"errors"

	"vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

var ErrManejadoresCronosNoDisponibles = errors.New("cronos: dependencias nominales no disponibles")

// DependenciasManejadoresCronos sólo acepta las autoridades de identidad y
// autorización compuestas por el enclave interno. No construye identidades,
// concesiones, repositorios ni servidores.
type DependenciasManejadoresCronos struct {
	ConsultaSaldo              ports.CasoUsoConsultarSaldo
	ResolverSaldo              httpinterno.ResolverConsultaSaldoPropio
	MarcajesRemotos            ports.CasoUsoMarcajesRemotos
	ResolverMarcajeRemoto      httpinterno.ResolverMarcajeRemoto
	ResolverRecuperacionRemota httpinterno.ResolverRecuperacionMarcajeRemoto
	Movimientos                ports.CasoUsoConsultarMovimientos
	ResolverMovimientos        httpinterno.ResolverConsultaMovimientosPropios
	Correcciones               httpinterno.CasoUsoSolicitarOlvido
	ResolverCorreccion         httpinterno.ResolverSolicitudCorreccionPropia
	Permisos                   ports.CasoUsoPermisosPropios
	ResolverPermisos           httpinterno.ResolverPermisosPropios
	// Resolución de permisos y avisos: opcionales, pero todos o ninguno.
	Resolucion         ports.CasoUsoResolucionPermisos
	ResolverResolucion httpinterno.ResolverResolucionPermisos
	Avisos             ports.CasoUsoAvisosPropios
	ResolverAvisos     httpinterno.ResolverAvisosPropios
}

type ManejadoresCronos struct {
	SaldoPropio        *httpinterno.ManejadorSaldoPropio
	MarcajeRemoto      *httpinterno.ManejadorMarcajeRemoto
	RecuperacionRemota *httpinterno.ManejadorRecuperacionMarcajeRemoto
	Movimientos        *httpinterno.ManejadorMovimientosPropios
	CorreccionPropia   *httpinterno.ManejadorCorreccionPropia
	PermisosPropios    *httpinterno.ManejadorPermisosPropios
	Resolucion         *httpinterno.ManejadorResolucionPermisos
	Avisos             *httpinterno.ManejadorAvisosPropios
}

// PrepararManejadoresCronos no registra rutas: las monta la frontera de
// cronos_empleado.go sólo cuando identidad, ContextoActor {empleado}, emisor
// V3 y PostgreSQL están compuestos. Un error deja las capacidades sin publicar.
func PrepararManejadoresCronos(d DependenciasManejadoresCronos) (ManejadoresCronos, error) {
	saldo, err := httpinterno.NuevoManejadorSaldoPropio(d.ConsultaSaldo, d.ResolverSaldo)
	if err != nil {
		return ManejadoresCronos{}, ErrManejadoresCronosNoDisponibles
	}
	remoto, err := httpinterno.NuevoManejadorMarcajeRemoto(d.MarcajesRemotos, d.ResolverMarcajeRemoto)
	if err != nil {
		return ManejadoresCronos{}, ErrManejadoresCronosNoDisponibles
	}
	recuperacion, err := httpinterno.NuevoManejadorRecuperacionMarcajeRemoto(d.MarcajesRemotos, d.ResolverRecuperacionRemota)
	if err != nil {
		return ManejadoresCronos{}, ErrManejadoresCronosNoDisponibles
	}
	movimientos, err := httpinterno.NuevoManejadorMovimientosPropios(d.Movimientos, d.ResolverMovimientos)
	if err != nil {
		return ManejadoresCronos{}, ErrManejadoresCronosNoDisponibles
	}
	correccion, err := httpinterno.NuevoManejadorCorreccionPropia(d.Correcciones, d.ResolverCorreccion)
	if err != nil {
		return ManejadoresCronos{}, ErrManejadoresCronosNoDisponibles
	}
	permisos, err := httpinterno.NuevoManejadorPermisosPropios(d.Permisos, d.ResolverPermisos)
	if err != nil {
		return ManejadoresCronos{}, ErrManejadoresCronosNoDisponibles
	}
	m := ManejadoresCronos{SaldoPropio: saldo, MarcajeRemoto: remoto, RecuperacionRemota: recuperacion,
		Movimientos: movimientos, CorreccionPropia: correccion, PermisosPropios: permisos}
	nulos := 0
	for _, v := range []any{d.Resolucion, d.ResolverResolucion, d.Avisos, d.ResolverAvisos} {
		if dependenciaDietasNula(v) {
			nulos++
		}
	}
	switch nulos {
	case 4:
		return m, nil
	case 0:
	default:
		return ManejadoresCronos{}, ErrManejadoresCronosNoDisponibles
	}
	if m.Resolucion, err = httpinterno.NuevoManejadorResolucionPermisos(d.Resolucion, d.ResolverResolucion); err != nil {
		return ManejadoresCronos{}, ErrManejadoresCronosNoDisponibles
	}
	if m.Avisos, err = httpinterno.NuevoManejadorAvisosPropios(d.Avisos, d.ResolverAvisos); err != nil {
		return ManejadoresCronos{}, ErrManejadoresCronosNoDisponibles
	}
	return m, nil
}
