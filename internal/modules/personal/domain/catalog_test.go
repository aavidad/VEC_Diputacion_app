package domain

import (
	"errors"
	"testing"
)

func TestValidarPuestoRPT(t *testing.T) {
	t.Parallel()
	pruebas := []struct {
		nombre        string
		puesto        RPTPosition
		errorEsperado error
	}{
		{
			nombre: "valido",
			puesto: RPTPosition{Code: "001", Name: "Puesto", State: "vigente", Dot: 1, DestinationLevel: 0, AnnualAmountCents: 100},
		},
		{
			nombre:        "codigo vacio",
			puesto:        RPTPosition{Code: "   ", Name: "Puesto", State: "vigente"},
			errorEsperado: ErrRPTPositionInvalid,
		},
		{
			nombre:        "nombre vacio",
			puesto:        RPTPosition{Code: "001", Name: " \t ", State: "vigente"},
			errorEsperado: ErrRPTPositionInvalid,
		},
		{
			nombre:        "dotacion negativa",
			puesto:        RPTPosition{Code: "001", Name: "Puesto", State: "vigente", Dot: -1},
			errorEsperado: ErrRPTPositionInvalid,
		},
		{
			nombre:        "nivel de destino negativo",
			puesto:        RPTPosition{Code: "001", Name: "Puesto", State: "vigente", DestinationLevel: -1},
			errorEsperado: ErrRPTPositionInvalid,
		},
		{
			nombre:        "importe anual negativo",
			puesto:        RPTPosition{Code: "001", Name: "Puesto", State: "vigente", AnnualAmountCents: -1},
			errorEsperado: ErrRPTPositionInvalid,
		},
		{
			nombre:        "estado ausente",
			puesto:        RPTPosition{Code: "001", Name: "Puesto"},
			errorEsperado: ErrRPTPositionInvalid,
		},
		{
			nombre:        "estado no canonico",
			puesto:        RPTPosition{Code: "001", Name: "Puesto", State: " vigente "},
			errorEsperado: ErrRPTPositionInvalid,
		},
	}

	for _, prueba := range pruebas {
		prueba := prueba
		t.Run(prueba.nombre, func(t *testing.T) {
			t.Parallel()
			if err := prueba.puesto.Validate(); !errors.Is(err, prueba.errorEsperado) {
				t.Fatalf("Validate() error = %v; esperado %v", err, prueba.errorEsperado)
			}
		})
	}
}

func TestValidarCategoriaProfesional(t *testing.T) {
	t.Parallel()
	pruebas := []struct {
		nombre        string
		categoria     ProfessionalCategory
		errorEsperado error
	}{
		{nombre: "valida", categoria: ProfessionalCategory{Slug: "categoria", Name: "Categoria", State: "vigente"}},
		{nombre: "slug vacio", categoria: ProfessionalCategory{Slug: "   ", Name: "Categoria", State: "vigente"}, errorEsperado: ErrProfessionalCategoryInvalid},
		{nombre: "nombre vacio", categoria: ProfessionalCategory{Slug: "categoria", Name: " \n\t ", State: "vigente"}, errorEsperado: ErrProfessionalCategoryInvalid},
		{nombre: "estado ausente", categoria: ProfessionalCategory{Slug: "categoria", Name: "Categoria"}, errorEsperado: ErrProfessionalCategoryInvalid},
		{nombre: "estado no canonico", categoria: ProfessionalCategory{Slug: "categoria", Name: "Categoria", State: " vigente "}, errorEsperado: ErrProfessionalCategoryInvalid},
	}
	for _, prueba := range pruebas {
		prueba := prueba
		t.Run(prueba.nombre, func(t *testing.T) {
			t.Parallel()
			if err := prueba.categoria.Validate(); !errors.Is(err, prueba.errorEsperado) {
				t.Fatalf("Validate() error = %v; esperado %v", err, prueba.errorEsperado)
			}
		})
	}
}

func TestValidarEntradaCatalogo(t *testing.T) {
	t.Parallel()
	pruebas := []struct {
		nombre        string
		entrada       CatalogEntry
		errorEsperado error
	}{
		{nombre: "valida", entrada: CatalogEntry{Catalog: "catalogo", Code: "001", Label: "Etiqueta", State: "vigente"}},
		{nombre: "catalogo vacio", entrada: CatalogEntry{Catalog: " ", Code: "001", Label: "Etiqueta", State: "vigente"}, errorEsperado: ErrCatalogEntryInvalid},
		{nombre: "codigo vacio", entrada: CatalogEntry{Catalog: "catalogo", Code: "\t", Label: "Etiqueta", State: "vigente"}, errorEsperado: ErrCatalogEntryInvalid},
		{nombre: "etiqueta vacia", entrada: CatalogEntry{Catalog: "catalogo", Code: "001", Label: "\n", State: "vigente"}, errorEsperado: ErrCatalogEntryInvalid},
		{nombre: "estado ausente", entrada: CatalogEntry{Catalog: "catalogo", Code: "001", Label: "Etiqueta"}, errorEsperado: ErrCatalogEntryInvalid},
		{nombre: "estado no canonico", entrada: CatalogEntry{Catalog: "catalogo", Code: "001", Label: "Etiqueta", State: " vigente "}, errorEsperado: ErrCatalogEntryInvalid},
	}
	for _, prueba := range pruebas {
		prueba := prueba
		t.Run(prueba.nombre, func(t *testing.T) {
			t.Parallel()
			if err := prueba.entrada.Validate(); !errors.Is(err, prueba.errorEsperado) {
				t.Fatalf("Validate() error = %v; esperado %v", err, prueba.errorEsperado)
			}
		})
	}
}
