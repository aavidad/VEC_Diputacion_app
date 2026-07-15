package almacen

import (
	"context"
	"strings"

	s3compatible "vec-diputacion-granada/internal/vec/adapters/almacen/s3"
	"vec-diputacion-granada/internal/vec/ports"
)

// RegistrarS3Compatible instala la fabrica, pero no crea conexiones ni lee
// credenciales. La sonda de capacidades se ejecuta al seleccionar el conector
// mediante RegistroConectoresAlmacen.Crear.
func RegistrarS3Compatible(registro *RegistroConectoresAlmacen, identificador string) error {
	if registro == nil {
		return ErrFabricaConectorAlmacenInvalida
	}
	identificador = strings.TrimSpace(identificador)
	return registro.Registrar(identificador, func(ctx context.Context, valores ConfiguracionConectorAlmacen) (ports.AlmacenObjetos, error) {
		if declarado := strings.TrimSpace(valores["conector_id"]); declarado != "" && declarado != identificador {
			return nil, s3compatible.ErrConfiguracionInvalida
		}
		configuracion := make(map[string]string, len(valores)+1)
		for clave, valor := range valores {
			configuracion[clave] = valor
		}
		configuracion["conector_id"] = identificador
		parametros, err := s3compatible.ConfiguracionDesdeMapa(configuracion)
		if err != nil {
			return nil, err
		}
		return s3compatible.Nuevo(ctx, parametros)
	})
}

func RegistrarS3CompatiblePredeterminado(registro *RegistroConectoresAlmacen) error {
	return RegistrarS3Compatible(registro, s3compatible.IdentificadorPredeterminado)
}
