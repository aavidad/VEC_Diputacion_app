package application

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestPreparacionConjuntosViasC1NormalizaResultadoCompartido(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	vias := viasCompartidasC1(5, false)
	catalogo := publicarCatalogoGlobalC1(
		t,
		entorno.inicio,
		"catalogo_cobertura_compartida_c1",
		1,
		vias,
	)
	escenario := crearEscenarioConCatalogoC1(
		t,
		entorno,
		catalogo,
		"",
		"",
	)
	preparacion := prepararGlobalC1(t, escenario, escenario.conjuntos)
	datos, err := preparacion.DatosCrearPropuestaEn(escenario.instante)
	if err != nil {
		t.Fatal(err)
	}
	if len(datos.Resultados) != 1 ||
		datos.Resultados[0].Clave != "comprobacion_compartida" ||
		datos.Resultados[0].Resultado != domain.ComprobacionAfirmativa {
		t.Fatalf("proyeccion compartida inesperada: %#v", datos.Resultados)
	}
	ordenes, err := preparacion.OrdenesPendientesEn(escenario.instante)
	if err != nil || len(ordenes) != 5 {
		t.Fatalf("se perdieron ordenes por via: %d, %v", len(ordenes), err)
	}
	if _, err := domain.CrearPropuestaDecisionCobertura(datos); err != nil {
		t.Fatalf("cinco vias compartidas no crean propuesta: %v", err)
	}
	comprobarCeroConsumoGlobalC1(t, entorno)
}

func TestPreparacionConjuntosViasC1RechazaFuenteDistintaParaMismaClave(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	_, err := domain.PublicarCatalogoViasCobertura(
		domain.BorradorCatalogoViasCobertura{
			Referencia:  "catalogo_fuente_cruzada_c1",
			Version:     1,
			PublicadoEn: entorno.inicio.Add(-time.Hour),
			Vigencia: domain.VigenciaCatalogoCobertura{
				Desde: entorno.inicio.Add(-time.Hour),
				Hasta: entorno.inicio.Add(time.Hour),
			},
			ProcedenciaRef: "procedimiento_fuente_cruzada_c1",
			Vias:           viasCompartidasC1(2, true),
		},
	)
	if !errors.Is(err, domain.ErrDatoInvalido) {
		t.Fatalf("misma clave con otra fuente aceptada: %v", err)
	}
}

func viasCompartidasC1(
	cantidad int,
	alterarUltima bool,
) []domain.DefinicionViaCobertura {
	vias := make([]domain.DefinicionViaCobertura, cantidad)
	for indice := range vias {
		procedencia := domain.ProcedenciaComprobacionCobertura{
			Clave:               "fuente_compartida",
			DefinicionFuenteRef: "fuente_definicion_bolsa_v3",
		}
		if alterarUltima && indice == cantidad-1 {
			procedencia.Clave = "fuente_cruzada"
			procedencia.DefinicionFuenteRef = "fuente_definicion_ajena_v1"
		}
		vias[indice] = domain.DefinicionViaCobertura{
			Clave: domain.ClaveCatalogo(
				fmt.Sprintf("via_compartida_%02d", indice+1),
			),
			Orden: uint16(indice + 1),
			Comprobaciones: []domain.ComprobacionExigibleCobertura{{
				Clave:       "comprobacion_compartida",
				Orden:       1,
				Obligatoria: true,
				Procedencia: procedencia,
			}},
		}
	}
	return vias
}
