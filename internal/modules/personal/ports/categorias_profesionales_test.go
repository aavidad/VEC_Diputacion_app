package ports

import (
	"errors"
	"testing"
	"time"
)

func referenciaCategoriasPrueba() ReferenciaCatalogoCategoriasProfesionales {
	return ReferenciaCatalogoCategoriasProfesionales{
		CatalogoID:           "categorias-profesionales",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
}

func TestCatalogoCategoriasProfesionalesValidaOrdenYClona(t *testing.T) {
	catalogo := CatalogoCategoriasProfesionalesConsultable{
		Referencia: referenciaCategoriasPrueba(),
		Fuente: FuenteCategoriasProfesionalesConsultable{
			Revision: "fuente-prueba-v1", ActualizadaEn: time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC),
			Demostracion: true, Aviso: "DEMOSTRACIÓN: fuente sintética de prueba.",
		},
		Categorias: []CategoriaProfesionalConsultable{
			{Clave: "administrativo", Etiqueta: "Administrativo", Orden: 1, Area: "administracion_general", AreaEtiqueta: "Administración general"},
			{Clave: "medico", Etiqueta: "Médico", Orden: 2, Area: "administracion_especial", AreaEtiqueta: "Administración especial"},
		},
	}
	if err := catalogo.Validar(); err != nil {
		t.Fatal(err)
	}
	clon := catalogo.Clonar()
	clon.Categorias[0].Etiqueta = "alterada"
	if catalogo.Categorias[0].Etiqueta != "Administrativo" {
		t.Fatal("el clon comparte el slice de categorias")
	}

	desordenado := catalogo.Clonar()
	desordenado.Categorias[0], desordenado.Categorias[1] = desordenado.Categorias[1], desordenado.Categorias[0]
	if err := desordenado.Validar(); !errors.Is(err, ErrCatalogoCategoriasProfesionalesNoDisponible) {
		t.Fatalf("orden invalido aceptado: %v", err)
	}

	duplicado := catalogo.Clonar()
	duplicado.Categorias[1].Clave = duplicado.Categorias[0].Clave
	if err := duplicado.Validar(); !errors.Is(err, ErrCatalogoCategoriasProfesionalesNoDisponible) {
		t.Fatalf("clave duplicada aceptada: %v", err)
	}

	fuenteInvalida := catalogo.Clonar()
	fuenteInvalida.Fuente.Aviso = "contenido de prueba sin marca explicita"
	if err := fuenteInvalida.Validar(); !errors.Is(err, ErrCatalogoCategoriasProfesionalesNoDisponible) {
		t.Fatalf("fuente DEMO sin aviso explicito aceptada: %v", err)
	}
}

func TestReferenciaCatalogoCategoriasProfesionalesExigeTuplaExacta(t *testing.T) {
	valida := referenciaCategoriasPrueba()
	if err := valida.Validar(); err != nil {
		t.Fatal(err)
	}
	casos := []ReferenciaCatalogoCategoriasProfesionales{
		{CatalogoID: "categorias-profesionales", CatalogoVersion: 1},
		{CatalogoID: "Categorias Profesionales", CatalogoVersion: 1, CatalogoHuellaSHA256: valida.CatalogoHuellaSHA256},
		{CatalogoID: "categorias-profesionales", CatalogoVersion: 0, CatalogoHuellaSHA256: valida.CatalogoHuellaSHA256},
	}
	for _, referencia := range casos {
		if err := referencia.Validar(); !errors.Is(err, ErrConsultaCategoriasProfesionalesInvalida) {
			t.Fatalf("referencia invalida aceptada: %#v, %v", referencia, err)
		}
	}
}
