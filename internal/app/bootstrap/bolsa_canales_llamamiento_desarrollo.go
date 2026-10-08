package bootstrap

import (
	"context"
	"log/slog"

	"vec-diputacion-granada/config"
	canalesbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/canales"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// comprobadorRegistroTelefono lo implementa el repositorio PostgreSQL de
// contactos: dice si B87 está instalada para el ejecutor.
type comprobadorRegistroTelefono interface {
	RegistroTelefonoInstalado(context.Context) (bool, error)
}

// componerCanalesLlamamientoBolsaDesarrollo une el catálogo de canales con
// los adaptadores reales: correo (si la emisión está compuesta) y teléfono
// (si B87 está instalada). SMS y Telegram no tienen proveedor corporativo:
// no se componen y, aunque el catálogo los active, no se publican.
func componerCanalesLlamamientoBolsaDesarrollo(emisionCompuesta bool, telefono comprobadorRegistroTelefono) (*aplicacionbolsa.RegistroCanalesLlamamiento, error) {
	datos, err := config.CatalogoCanalesLlamamientoBolsa()
	if err != nil {
		return nil, err
	}
	catalogo, err := dominiobolsa.ParsearCatalogoCanalesLlamamiento(datos)
	if err != nil {
		return nil, err
	}
	adaptadores := []puertosbolsa.CanalAvisoLlamamiento{canalesbolsa.NuevoCorreo(emisionCompuesta)}
	if telefono != nil {
		canal, errTelefono := canalesbolsa.NuevoTelefono(telefono.RegistroTelefonoInstalado)
		if errTelefono != nil {
			return nil, errTelefono
		}
		adaptadores = append(adaptadores, canal)
	}
	registro, err := aplicacionbolsa.NuevoRegistroCanalesLlamamiento(catalogo, adaptadores...)
	if err != nil {
		return nil, err
	}
	for _, canal := range registro.CanalesSinProveedor() {
		slog.Warn("canal de llamamiento activo en el catálogo sin proveedor; no se ofrece", "canal", canal)
	}
	return registro, nil
}

// CanalesLlamamientoActivos publica en la lectura de candidatos los canales
// que RRHH puede usar ahora en un llamamiento.
func (m *manejadorParticipacionBolsaDesarrollo) CanalesLlamamientoActivos(ctx context.Context) []dominiobolsa.CanalLlamamiento {
	if m == nil || m.canales == nil {
		return nil
	}
	return m.canales.Activos(ctx)
}
