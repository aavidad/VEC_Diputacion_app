package almacen

import (
	"context"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/almacen/ficheros"
	"vec-diputacion-granada/internal/vec/ports"
)

// RegistrarFicheros instala la fábrica del almacén de ficheros local. Claves
// admitidas: directorio (absoluto y privado), tamano_maximo (bytes) y
// retencion_minima_dias. Ninguna procede del cliente: las fija la raíz de
// composición desde su material privado.
func RegistrarFicheros(registro *RegistroConectoresAlmacen, identificador string, reloj ports.Reloj) error {
	if registro == nil || reloj == nil {
		return ErrFabricaConectorAlmacenInvalida
	}
	identificador = strings.TrimSpace(identificador)
	return registro.Registrar(identificador, func(_ context.Context, valores ConfiguracionConectorAlmacen) (ports.AlmacenObjetos, error) {
		for clave := range valores {
			switch clave {
			case "directorio", "tamano_maximo", "retencion_minima_dias":
			default:
				return nil, ficheros.ErrConfiguracionInvalida
			}
		}
		// retencion_minima_dias es obligatoria y explícita: su ausencia no
		// equivale a cero. Cero días: el conector no fija retención al escribir
		// (política de conservación provisional); se aplica después con
		// AplicarRetencion.
		textoDias, declarada := valores["retencion_minima_dias"]
		tamano, errT := strconv.ParseInt(valores["tamano_maximo"], 10, 64)
		dias, errD := strconv.ParseInt(textoDias, 10, 64)
		if !declarada || errT != nil || errD != nil || dias < 0 || dias > 36500 {
			return nil, ficheros.ErrConfiguracionInvalida
		}
		return ficheros.Nuevo(ficheros.Configuracion{
			ConectorID: identificador, Directorio: valores["directorio"], TamanoMaximo: tamano,
			RetencionMinimaAdmitida: time.Duration(dias) * 24 * time.Hour,
		}, reloj)
	})
}
