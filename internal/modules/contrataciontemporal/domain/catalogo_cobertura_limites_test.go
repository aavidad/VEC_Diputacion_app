package domain

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestCatalogoCoberturaAdmiteLimitesTemporalesYRoundTripJSON(t *testing.T) {
	minimo := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	maximo := time.Date(9999, 12, 31, 23, 59, 59, 999999000, time.UTC)
	borrador := borradorCatalogoCoberturaValido()
	borrador.PublicadoEn = minimo
	borrador.Vigencia = VigenciaCatalogoCobertura{
		Desde: minimo,
		Hasta: maximo,
	}
	catalogo, err := PublicarCatalogoViasCobertura(borrador)
	if err != nil {
		t.Fatalf("publicar límites temporales: %v", err)
	}
	if !catalogo.VigenteEn(minimo) || catalogo.VigenteEn(maximo) {
		t.Fatal("los límites no respetan la vigencia semiabierta")
	}

	codificado, err := json.Marshal(catalogo.Publicacion())
	if err != nil {
		t.Fatalf("codificar límites como JSON: %v", err)
	}
	var publicacion PublicacionCatalogoViasCobertura
	if err := json.Unmarshal(codificado, &publicacion); err != nil {
		t.Fatalf("decodificar límites desde JSON: %v", err)
	}
	restaurado, err := RestaurarCatalogoViasCobertura(publicacion)
	if err != nil {
		t.Fatalf("restaurar límites desde JSON: %v", err)
	}
	if !catalogo.Identidad().CoincideExactamente(restaurado.Identidad()) ||
		!restaurado.PublicadoEn().Equal(minimo) ||
		!restaurado.Vigencia().Hasta.Equal(maximo) {
		t.Fatal("el round-trip JSON alteró los límites temporales")
	}

	borrador.PublicadoEn = maximo
	borrador.Vigencia = VigenciaCatalogoCobertura{Desde: maximo}
	catalogoMaximo, err := PublicarCatalogoViasCobertura(borrador)
	if err != nil || !catalogoMaximo.VigenteEn(maximo) {
		t.Fatalf("el máximo exacto no es operativo: %v", err)
	}
}

func TestCatalogoCoberturaRechazaInstantesFueraDelIntervaloTransportable(t *testing.T) {
	anterior := time.Date(0, 12, 31, 23, 59, 59, 999999000, time.UTC)
	posterior := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	casos := []struct {
		nombre    string
		modificar func(*BorradorCatalogoViasCobertura)
	}{
		{"publicación anterior", func(b *BorradorCatalogoViasCobertura) {
			b.PublicadoEn = anterior
		}},
		{"publicación posterior", func(b *BorradorCatalogoViasCobertura) {
			b.PublicadoEn = posterior
		}},
		{"inicio anterior", func(b *BorradorCatalogoViasCobertura) {
			b.Vigencia.Desde = anterior
		}},
		{"inicio posterior", func(b *BorradorCatalogoViasCobertura) {
			b.Vigencia.Desde = posterior
		}},
		{"fin anterior", func(b *BorradorCatalogoViasCobertura) {
			b.Vigencia.Hasta = anterior
		}},
		{"fin posterior", func(b *BorradorCatalogoViasCobertura) {
			b.Vigencia.Hasta = posterior
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			borrador := borradorCatalogoCoberturaValido()
			caso.modificar(&borrador)
			if _, err := PublicarCatalogoViasCobertura(borrador); !errors.Is(err, ErrDatoInvalido) {
				t.Fatalf("se aceptó un instante no transportable: %v", err)
			}
		})
	}
}

func TestCatalogoCoberturaRechazaMismaProcedenciaConDefinicionDistinta(t *testing.T) {
	borrador := borradorCatalogoCoberturaValido()
	borrador.Vias[0].Comprobaciones[0] = ComprobacionExigibleCobertura{
		Clave: "comprobacion_compartida", Orden: 1, Obligatoria: true,
		Procedencia: procedencia("fuente_compartida", "definicion_fuente_a"),
	}
	borrador.Vias[1].Comprobaciones[0] = ComprobacionExigibleCobertura{
		Clave: "comprobacion_compartida", Orden: 20, Obligatoria: false,
		Procedencia: procedencia("fuente_compartida", "definicion_fuente_b"),
	}
	if borrador.Vias[0].Validar() != nil || borrador.Vias[1].Validar() != nil {
		t.Fatal("el caso no alcanza la coherencia global de procedencia")
	}
	if _, err := PublicarCatalogoViasCobertura(borrador); !errors.Is(err, ErrDatoInvalido) {
		t.Fatalf("se aceptaron definiciones distintas para la misma clave: %v", err)
	}
}

func TestCatalogoCoberturaAdmiteLimitesPositivosDeCardinalidad(t *testing.T) {
	casos := []struct {
		nombre string
		vias   []DefinicionViaCobertura
	}{
		{"sesenta y cuatro vías", viasNumeradas(maximoViasCobertura)},
		{
			"treinta y dos comprobaciones por vía",
			[]DefinicionViaCobertura{{
				Clave: "via_limite", Orden: 1,
				Comprobaciones: comprobacionesNumeradas(
					maximoComprobacionesPorViaCobertura,
				),
			}},
		},
		{
			"quinientas doce comprobaciones totales",
			viasConComprobacionesNumeradas(16, 32),
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			borrador := borradorCatalogoCoberturaValido()
			borrador.Vias = caso.vias
			if _, err := PublicarCatalogoViasCobertura(borrador); err != nil {
				t.Fatalf("se rechazó el límite positivo: %v", err)
			}
		})
	}
}
