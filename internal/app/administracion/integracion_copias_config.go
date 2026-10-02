package administracion

import (
	"vec-diputacion-granada/config"
	pol "vec-diputacion-granada/internal/modules/administracion/adapters/politicacopias"
	reg "vec-diputacion-granada/internal/modules/administracion/adapters/registrocopias"
	app "vec-diputacion-granada/internal/modules/administracion/application/httpcopias"
	ej "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

// AbrirLecturasCopias abre los proveedores privados existentes. El destino CS03
// ya debe estar construido con su KMS/catálogo; no se deriva una clave del JSON.
func AbrirLecturasCopias(c config.CopiasADMIN, autoridad p.Autorizador, auditor AuditorLecturasCopias, destino ej.Destino, instalacion ej.Inventario) (*LecturasCopias, func() error, error) {
	if c.Validar() != nil || app.Ausente(autoridad) || app.Ausente(auditor) || app.Ausente(destino) {
		return nil, nil, p.ErrNoDisponible
	}
	diario, err := reg.Abrir(reg.Config{Directorio: c.DiarioDirectorio, RaicesRestauradas: c.RaicesRestauradas, LimiteListado: c.LimiteListado})
	if err != nil {
		return nil, nil, p.ErrNoDisponible
	}
	politica, err := pol.AbrirExterno(pol.ConfigExterna{Directorio: c.PoliticaDirectorio, RaicesRestauradas: c.RaicesRestauradas})
	if err != nil {
		return nil, nil, p.ErrNoDisponible
	}
	return &LecturasCopias{Autoridad: autoridad, Auditor: auditor, Diario: diario, Destino: destino, Instalacion: instalacion, DestinoRef: c.DestinoRef, LimiteListado: c.LimiteListado, Politica: politica}, politica.Cerrar, nil
}
