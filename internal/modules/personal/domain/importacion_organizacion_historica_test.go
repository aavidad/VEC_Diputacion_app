package domain

import (
	"errors"
	"strings"
	"testing"
)

func manifiestoImportacionPrueba() ManifiestoImportacionOrganizacion {
	h := strings.Repeat("a", 64)
	return ManifiestoImportacionOrganizacion{
		OrganismoRef: "organismo:dipgra", Tipo: "rpt", VersionRef: "018f47a2-6b31-4c80-8a95-4d2e707c5a11", VersionRevision: 1, FuenteRef: "fuente:preparacion", FuenteVersion: "v1", FuenteHuellaSHA256: h,
		CatalogoUnidades:        ReferenciaCatalogoImportacion{ID: "estructura-organizativa-dipgra", Version: 1, Revision: 2, HuellaSHA256: h},
		CatalogoClasificaciones: ReferenciaCatalogoImportacion{ID: "clasificaciones-dipgra", Version: 3, Revision: 1, HuellaSHA256: h},
	}
}

func TestManifiestoPreparacionNoEsPublicacion(t *testing.T) {
	m := manifiestoImportacionPrueba()
	if err := m.Validar(true); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(m.Validar(false), ErrImportacionOrganizacionInvalida) {
		t.Fatal("publicacion sin fuente y acto admitidos")
	}
	m.DocumentoRef, m.CustodiaRef, m.DiccionarioRef, m.ActoRef = "documento:uno", "custodia:uno", "diccionario:uno", "acto:uno"
	m.AprobadaEn, m.PublicadaEn, m.EfectosDesde = "2026-01-01", "2026-01-02", "2026-01-03"
	if err := m.Validar(false); err != nil {
		t.Fatal(err)
	}
	m.EfectosHasta = "2026-01-03"
	if !errors.Is(m.Validar(false), ErrImportacionOrganizacionInvalida) {
		t.Fatal("fin de efectos no exclusivo")
	}
}

func TestConciliacionNoInventaDestino(t *testing.T) {
	d := DecisionConciliacionOrganizacion{FilaFuenteRef: "fila:15", Clase: "dotacion", Resultado: "pendiente", Motivo: "codigo individual no acreditado", EvidenciaRef: "evidencia:15"}
	if err := d.Validar(); err != nil {
		t.Fatal(err)
	}
	d.DestinoRef = "puesto:inventado"
	if !errors.Is(d.Validar(), ErrImportacionOrganizacionInvalida) {
		t.Fatal("pendiente con puesto asignado")
	}
	d.Resultado = "vinculada"
	d.DestinoRef = ""
	if !errors.Is(d.Validar(), ErrImportacionOrganizacionInvalida) {
		t.Fatal("vinculada sin identidad acreditada")
	}
}

func TestHechoDotacionNoCreaPuestoIndividual(t *testing.T) {
	m := manifiestoImportacionPrueba()
	h := HechoImportacionOrganizacion{Clase: "dotacion", HechoRef: "018f47a2-6b31-4c80-8a95-4d2e707c5a21", Revision: 1,
		FilaFuenteRef: "fila:19", PaginaFuente: 7, OrganismoRef: m.OrganismoRef, UnidadRef: "centro:uno", VigenteDesde: "2026-01-03",
		TipoRef: "018f47a2-6b31-4c80-8a95-4d2e707c5a22", TipoRevision: 1, Cantidad: 19, Reconciliacion: "pendiente"}
	if err := h.Validar(m); err != nil {
		t.Fatal(err)
	}
	h.CodigoFuente = "0123"
	if !errors.Is(h.Validar(m), ErrImportacionOrganizacionInvalida) {
		t.Fatal("dotacion recibio codigo de puesto")
	}
	h.Clase = "puesto_individual"
	if !errors.Is(h.Validar(m), ErrImportacionOrganizacionInvalida) {
		t.Fatal("dotacion tratada como puesto individual")
	}
	p := HechoImportacionOrganizacion{Clase: "puesto_individual", HechoRef: "018f47a2-6b31-4c80-8a95-4d2e707c5a23", Revision: 1, FilaFuenteRef: "fila:individual",
		OrganismoRef: m.OrganismoRef, UnidadRef: "centro:uno", VigenteDesde: "2026-01-03", CodigoFuente: "0123", TipoRef: h.TipoRef, TipoRevision: 1, EstadoEstructural: "vigente"}
	if err := p.Validar(m); err != nil {
		t.Fatalf("perdio cero inicial del codigo literal: %v", err)
	}
}
