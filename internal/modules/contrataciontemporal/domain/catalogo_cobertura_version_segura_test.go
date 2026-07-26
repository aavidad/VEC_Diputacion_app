package domain

import (
	"testing"
	"time"
)

func TestCatalogoCoberturaRechazaVersionFueraDeEnteroSeguro(t *testing.T) {
	borrador := BorradorCatalogoViasCobertura{
		Referencia:  "catalogo_cobertura_limites_01",
		Version:     maximoEnteroSeguroCatalogoCobertura + 1,
		PublicadoEn: time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC),
		Vigencia: VigenciaCatalogoCobertura{
			Desde: time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC),
			Hasta: time.Date(2027, 7, 23, 10, 0, 0, 0, time.UTC),
		},
		ProcedenciaRef: "acto_publicacion_catalogo_01",
		Vias: []DefinicionViaCobertura{{
			Clave: "via_catalogo_limite",
			Orden: 1,
			Comprobaciones: []ComprobacionExigibleCobertura{{
				Clave:       "comprobacion_catalogo_limite",
				Orden:       1,
				Obligatoria: true,
				Procedencia: ProcedenciaComprobacionCobertura{
					Clave:               "fuente_catalogo_limite",
					DefinicionFuenteRef: "conector_catalogo_limite_01",
				},
			}},
		}},
	}
	if _, err := PublicarCatalogoViasCobertura(borrador); err == nil {
		t.Fatal("el catálogo aceptó una versión no segura para clientes")
	}
	identidad := IdentidadCatalogoViasCobertura{
		Referencia:   "catalogo_cobertura_limites_01",
		Version:      maximoEnteroSeguroCatalogoCobertura + 1,
		HuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	if identidad.Validar() == nil {
		t.Fatal("la identidad aceptó una versión no segura")
	}
}

func TestCatalogoCoberturaAdmiteVersionMaximaSegura(t *testing.T) {
	borrador := borradorCatalogoCoberturaValido()
	borrador.Version = maximoEnteroSeguroCatalogoCobertura
	catalogo, err := PublicarCatalogoViasCobertura(borrador)
	if err != nil {
		t.Fatalf("se rechazó la versión máxima segura: %v", err)
	}
	if catalogo.Version() != maximoEnteroSeguroCatalogoCobertura {
		t.Fatal("la versión máxima segura no se conservó exactamente")
	}
}
