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
}

type ManejadoresCronos struct {
	SaldoPropio        *httpinterno.ManejadorSaldoPropio
	MarcajeRemoto      *httpinterno.ManejadorMarcajeRemoto
	RecuperacionRemota *httpinterno.ManejadorRecuperacionMarcajeRemoto
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
	return ManejadoresCronos{SaldoPropio: saldo, MarcajeRemoto: remoto, RecuperacionRemota: recuperacion}, nil
}
