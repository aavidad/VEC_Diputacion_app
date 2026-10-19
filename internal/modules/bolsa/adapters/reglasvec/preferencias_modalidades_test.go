package reglasvec

import (
	"errors"
	"slices"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	"vec-diputacion-granada/internal/vec/reglas"
)

func TestModalidadesPreferenciasDesdeCatalogoComun(t *testing.T) {
	consulta, err := fichero.NuevaConsultaCatalogoProvisionable("../../../../../data/catalogos/contratacion-temporal/reglas.v3.json")
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: reglas.CatalogoContratacionTemporal,
		ModuloID: reglas.ModuloContratacionTemporal, Reloj: relojFijo(time.Date(2026, 10, 18, 8, 0, 0, 0, time.UTC)),
	})
	if err != nil {
		t.Fatal(err)
	}
	catalogo := NuevoCatalogoPreferenciasModalidades(resolutor)
	opciones, err := catalogo.ConsultarModalidadesDisponibles(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var claves []string
	for _, opcion := range opciones {
		claves = append(claves, opcion.Clave)
		if opcion.Fuente.Validar() != nil || opcion.Fuente.CatalogoVersion != 3 ||
			opcion.Fuente.EntradaClave != reglas.CTPrefijoModalidad+opcion.Clave {
			t.Fatalf("referencia de modalidad incompleta: %+v", opcion)
		}
	}
	if !slices.Equal(claves, []string{"acumulacion_tareas", "programa", "relevo", "sustitucion", "vacante"}) {
		t.Fatalf("modalidades=%v", claves)
	}
	// La llamada siguiente vuelve al catálogo y no comparte el resultado mutable.
	opciones[0].Clave = "alterada"
	opciones[0].Fuente.EntradaClave = "alterada"
	nuevas, err := catalogo.ConsultarModalidadesDisponibles(t.Context())
	if err != nil || nuevas[0].Clave != claves[0] || nuevas[0].Fuente.EntradaClave == "alterada" {
		t.Fatalf("resultado compartido: opciones=%v err=%v", nuevas, err)
	}
}

func TestModalidadesPreferenciasSinFuenteNoInventaOpciones(t *testing.T) {
	var sinCatalogo *CatalogoPreferenciasModalidades
	for _, catalogo := range []*CatalogoPreferenciasModalidades{sinCatalogo, NuevoCatalogoPreferenciasModalidades(nil)} {
		if opciones, err := catalogo.ConsultarModalidadesDisponibles(t.Context()); !errors.Is(err, ports.ErrPreferenciasNoDisponibles) || opciones != nil {
			t.Fatalf("opciones=%v err=%v", opciones, err)
		}
	}
	// Un catálogo válido de otro módulo tampoco produce modalidades por defecto.
	catalogo := NuevoCatalogoPreferenciasModalidades(catalogoPrueba(t, nil).resolutor)
	if opciones, err := catalogo.ConsultarModalidadesDisponibles(t.Context()); !errors.Is(err, ports.ErrPreferenciasNoDisponibles) || opciones != nil {
		t.Fatalf("opciones sin catálogo CT=%v err=%v", opciones, err)
	}
}
