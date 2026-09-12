package config

import _ "embed"

// El manifiesto de presentación acompaña al binario para no depender del
// directorio de arranque. No concede permisos ni modifica el flujo guardado.
//
//go:embed contratacion_temporal_flujo_visual_rrhh_v1.json
var presentacionFlujoRRHHDesarrollo string

func PresentacionFlujoRRHHDesarrollo() string {
	return presentacionFlujoRRHHDesarrollo
}
