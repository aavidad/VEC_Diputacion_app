package main

import (
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/app/administracion"
)

// El arranque dice en qué etapa falló sin dejar de ser ErrConfiguracion y
// sin copiar rutas ni valores de la configuración privada.
func TestArranqueIndicaEtapaSinDatosPrivados(t *testing.T) {
	ruta := "/ruta/privada/no-debe-salir.json"
	base := configuracionPerfilesPrivada{ActivosDirectorio: ruta}
	for nombre, componer := range map[string]func() error{
		"perfiles": func() error { _, _, err := componerProcesoADMIN(administracion.Configuracion{}, base); return err },
		"usuarios": func() error {
			_, _, err := componerProcesoUsuariosMetadatosADMIN(administracion.Configuracion{}, base, configuracionUsuariosMetadatosPrivada{PoolLector: ruta})
			return err
		},
	} {
		err := componer()
		if !errors.Is(err, administracion.ErrConfiguracion) || !strings.HasSuffix(err.Error(), ": etapa=configuracion") || strings.Contains(err.Error(), ruta) {
			t.Fatalf("%s: error de arranque sin etapa o con datos privados: %v", nombre, err)
		}
	}
	base.Identidad.EspacioIdentidad = "espacio:admin"
	_, _, err := componerProcesoADMIN(administracion.Configuracion{EmisorIdentidad: "emisor:otro"}, base)
	if !errors.Is(err, administracion.ErrConfiguracion) || !strings.HasSuffix(err.Error(), ": etapa=emisor_identidad") {
		t.Fatalf("emisor distinto del espacio de identidad aceptado: %v", err)
	}
	if err := errorArranque("escucha"); !errors.Is(err, administracion.ErrConfiguracion) || !strings.HasSuffix(err.Error(), ": etapa=escucha") {
		t.Fatalf("etapa perdida: %v", err)
	}
}
