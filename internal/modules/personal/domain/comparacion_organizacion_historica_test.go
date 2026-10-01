package domain

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func instantaneaComparacionEjemplo() InstantaneaComparacionOrganizacion {
	instante := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	traza := func(id string) TrazaOrganizacionHistorica {
		return TrazaOrganizacionHistorica{ID: id, Version: 1, FuenteRef: "fuente:ejemplo", ActoRef: "acto:ejemplo", HuellaSHA256: strings.Repeat("a", 64), EfectosDesde: "2026-01-01", ConocidoDesde: instante}
	}
	return InstantaneaComparacionOrganizacion{
		Selector:            SelectorOrganizacionHistorica{OrganismoRef: "organismo:ejemplo", VigenteEn: "2026-02-01", ConocidoEn: instante.Add(24 * time.Hour), VersionRPTRef: "rpt:uno", VersionPlantillaRef: "plantilla:uno", Limite: 100},
		Cobertura:           CoberturaComparacionOrganizacion{"completa", "completa", "completa", "completa", "completa", "completa"},
		Unidades:            []UnidadOrganizacionHistorica{{Traza: traza("unidad:uno"), CatalogoID: "estructura-organizativa-dipgra", CatalogoVersion: 1, CatalogoRevision: 1, ClaveCatalogo: "centro:uno", Tipo: "centro", Etiqueta: "Centro"}},
		PuestosTipo:         []PuestoTipoOrganizacionHistorica{{Traza: traza("tipo:uno"), VersionRPTRef: "rpt:uno", CodigoFuente: "0007", UnidadID: "unidad:uno", Denominacion: "Puesto tipo", ClasificacionRef: "categoria:uno"}},
		Dotaciones:          []DotacionOrganizacionHistorica{{Traza: traza("dotacion:uno"), VersionRPTRef: "rpt:uno", PuestoTipoID: "tipo:uno", Cantidad: 2}},
		Plazas:              []PlazaOrganizacionHistorica{{Traza: traza("plaza:uno"), VersionPlantillaRef: "plantilla:uno", CodigoFuente: "00001", ClasificacionRef: "categoria:uno", UnidadID: "unidad:uno", EstadoEstructural: "vigente"}},
		PuestosIndividuales: []PuestoIndividualOrganizacionHistorica{{Traza: traza("puesto:uno"), VersionRPTRef: "rpt:uno", CodigoFuente: "0001", PuestoTipoID: "tipo:uno", UnidadID: "unidad:uno", EstadoEstructural: "vigente"}},
		Vinculos:            []VinculoPlazaPuestoHistorico{{Traza: traza("vinculo:uno"), PlazaID: "plaza:uno", PuestoID: "puesto:uno"}},
	}
}

func TestComparacionOrganizacionHistoricaCamposYProcedencia(t *testing.T) {
	a, d := instantaneaComparacionEjemplo(), instantaneaComparacionEjemplo()
	d.Unidades[0].Etiqueta = "Otro centro"
	d.PuestosTipo[0].CodigoFuente = "0008"
	d.Dotaciones[0].Cantidad = 3
	d.Plazas[0].EstadoEstructural = "amortizada"
	d.PuestosIndividuales[0].UnidadID = "unidad:otra"
	d.Vinculos[0].PuestoID = "puesto:otro"
	r, err := CompararOrganizacionHistorica(a, d)
	if err != nil {
		t.Fatal(err)
	}
	colecciones := []ColeccionComparacionOrganizacion{r.Unidades, r.PuestosTipo, r.Dotaciones, r.Plazas, r.PuestosIndividuales, r.Vinculos}
	campos := []string{"etiqueta", "codigo_fuente", "cantidad", "estado_estructural", "unidad_id", "puesto_id"}
	for n, c := range colecciones {
		if c.Estado != "completa" || c.Antes != 1 || c.Despues != 1 || len(c.Cambios) != 1 || c.Cambios[0].Tipo != "cambio" || !reflect.DeepEqual(c.Cambios[0].Campos, []string{campos[n]}) {
			t.Fatalf("coleccion %d: %+v", n, c)
		}
	}
	d = instantaneaComparacionEjemplo()
	d.Selector.VersionRPTRef = "rpt:dos"
	d.PuestosTipo[0].VersionRPTRef = "rpt:dos"
	d.Dotaciones[0].VersionRPTRef = "rpt:dos"
	d.PuestosIndividuales[0].VersionRPTRef = "rpt:dos"
	d.PuestosTipo[0].Traza.Version = 2
	r, err = CompararOrganizacionHistorica(a, d)
	if err != nil {
		t.Fatal(err)
	}
	if c := r.PuestosTipo.Cambios; len(c) != 1 || c[0].Tipo != "procedencia" || !reflect.DeepEqual(c[0].Campos, []string{"traza", "version_rpt_ref"}) {
		t.Fatalf("procedencia: %+v", c)
	}
	if len(r.Unidades.Cambios) != 0 || len(r.Plazas.Cambios) != 0 {
		t.Fatal("el cambio de version RPT afecta colecciones ajenas")
	}
}

func TestComparacionOrganizacionHistoricaIdentidadCoberturaYOrden(t *testing.T) {
	a, d := instantaneaComparacionEjemplo(), instantaneaComparacionEjemplo()
	d.Plazas[0].Traza.ID = "plaza:otra" // El mismo código no enlaza identidades.
	r, err := CompararOrganizacionHistorica(a, d)
	if err != nil {
		t.Fatal(err)
	}
	if c := r.Plazas.Cambios; len(c) != 2 || c[0].ID != "plaza:otra" || c[0].Tipo != "entrada_en_corte" || c[1].Tipo != "salida_del_corte" {
		t.Fatalf("identidades: %+v", c)
	}
	d.Cobertura.Plazas = "parcial"
	r, err = CompararOrganizacionHistorica(a, d)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Plazas.Cambios) != 0 || r.Plazas.Estado != "parcial" || !reflect.DeepEqual(r.Plazas.NoVerificables, []string{"plaza:otra", "plaza:uno"}) {
		t.Fatalf("parcial: %+v", r.Plazas)
	}
	d.Plazas = append(d.Plazas, a.Plazas[0])
	d.Plazas[1].EstadoEstructural = "amortizada"
	r, err = CompararOrganizacionHistorica(a, d)
	if err != nil || len(r.Plazas.Cambios) != 1 || r.Plazas.Cambios[0].Tipo != "cambio" {
		t.Fatalf("comunes parciales: %+v %v", r.Plazas, err)
	}
	a.Plazas = append(a.Plazas, d.Plazas[0])
	r1, err := CompararOrganizacionHistorica(a, d)
	if err != nil {
		t.Fatal(err)
	}
	a.Plazas[0], a.Plazas[1] = a.Plazas[1], a.Plazas[0]
	d.Plazas[0], d.Plazas[1] = d.Plazas[1], d.Plazas[0]
	r2, err := CompararOrganizacionHistorica(a, d)
	if err != nil || !reflect.DeepEqual(r1, r2) {
		t.Fatal("el orden de entrada altera el informe")
	}
	if c := r2.Plazas.Cambios; len(c) != 1 || c[0].ID != "plaza:uno" {
		t.Fatalf("orden cambios: %+v", c)
	}
}

func TestComparacionOrganizacionHistoricaSinDatosYPropiedad(t *testing.T) {
	a := instantaneaComparacionEjemplo()
	r, err := CompararOrganizacionHistorica(a, a)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []ColeccionComparacionOrganizacion{r.Unidades, r.PuestosTipo, r.Dotaciones, r.Plazas, r.PuestosIndividuales, r.Vinculos} {
		if len(c.Cambios) != 0 || len(c.NoVerificables) != 0 {
			t.Fatal("corte idéntico produce cambios")
		}
	}
	equivalente := instantaneaComparacionEjemplo()
	equivalente.Unidades[0].Traza.ConocidoDesde = equivalente.Unidades[0].Traza.ConocidoDesde.In(time.FixedZone("otra representacion UTC", 0))
	r, err = CompararOrganizacionHistorica(a, equivalente)
	if err != nil || len(r.Unidades.Cambios) != 0 {
		t.Fatal("representaciones equivalentes del mismo instante producen procedencia")
	}
	d := instantaneaComparacionEjemplo()
	d.Unidades[0].Etiqueta = "Modificada"
	entrada, _ := json.Marshal(a)
	r, err = CompararOrganizacionHistorica(a, d)
	if err != nil {
		t.Fatal(err)
	}
	r.Unidades.Cambios[0].TrazaAntes.ID = "alterado"
	r.Unidades.Cambios[0].Campos[0] = "alterado"
	actual, _ := json.Marshal(a)
	if string(entrada) != string(actual) {
		t.Fatal("el resultado comparte datos mutables con la entrada")
	}
	d.Plazas = nil
	d.Cobertura.Plazas = "sin_datos"
	d.Selector.VersionPlantillaRef = ""
	r, err = CompararOrganizacionHistorica(a, d)
	if err != nil || len(r.Plazas.Cambios) != 0 || !reflect.DeepEqual(r.Plazas.NoVerificables, []string{"plaza:uno"}) {
		t.Fatalf("sin datos: %+v %v", r.Plazas, err)
	}
	r, err = CompararOrganizacionHistorica(d, d)
	if err != nil || r.Plazas.Estado != "sin_datos" {
		t.Fatalf("ambos sin datos: %+v %v", r.Plazas, err)
	}
}

func TestComparacionOrganizacionHistoricaRechazaEntradasInvalidas(t *testing.T) {
	casos := map[string]func(*InstantaneaComparacionOrganizacion){
		"organismo": func(d *InstantaneaComparacionOrganizacion) { d.Selector.OrganismoRef = "organismo:otro" },
		"unidad":    func(d *InstantaneaComparacionOrganizacion) { d.Selector.UnidadClave = "centro:otro" },
		"fecha":     func(d *InstantaneaComparacionOrganizacion) { d.Selector.VigenteEn = "2026-02-30" },
		"cursor":    func(d *InstantaneaComparacionOrganizacion) { d.Selector.Cursor = "otra_pagina" },
		"traza futura": func(d *InstantaneaComparacionOrganizacion) {
			d.Unidades[0].Traza.ConocidoDesde = d.Selector.ConocidoEn.Add(time.Hour)
		},
		"traza fuera de efectos": func(d *InstantaneaComparacionOrganizacion) { d.Plazas[0].Traza.EfectosHasta = d.Selector.VigenteEn },
		"duplicado":              func(d *InstantaneaComparacionOrganizacion) { d.Plazas = append(d.Plazas, d.Plazas[0]) },
		"cobertura":              func(d *InstantaneaComparacionOrganizacion) { d.Cobertura.Unidades = "desconocida" },
		"sin datos poblado":      func(d *InstantaneaComparacionOrganizacion) { d.Cobertura.Unidades = "sin_datos" },
		"version ausente":        func(d *InstantaneaComparacionOrganizacion) { d.Selector.VersionRPTRef = "" },
		"version incompatible":   func(d *InstantaneaComparacionOrganizacion) { d.Plazas[0].VersionPlantillaRef = "plantilla:otra" },
		"dotacion cero":          func(d *InstantaneaComparacionOrganizacion) { d.Dotaciones[0].Cantidad = 0 },
		"estado plaza":           func(d *InstantaneaComparacionOrganizacion) { d.Plazas[0].EstadoEstructural = "libre" },
		"estado puesto":          func(d *InstantaneaComparacionOrganizacion) { d.PuestosIndividuales[0].EstadoEstructural = "vacante" },
		"vinculo":                func(d *InstantaneaComparacionOrganizacion) { d.Vinculos[0].PlazaID = "" },
		"codigo control":         func(d *InstantaneaComparacionOrganizacion) { d.PuestosTipo[0].CodigoFuente = "0001\n" },
		"referencia invalida":    func(d *InstantaneaComparacionOrganizacion) { d.PuestosIndividuales[0].PuestoTipoID = "001" },
		"texto demasiado grande": func(d *InstantaneaComparacionOrganizacion) { d.Unidades[0].Etiqueta = strings.Repeat("a", 513) },
		"limite hechos": func(d *InstantaneaComparacionOrganizacion) {
			d.Unidades = make([]UnidadOrganizacionHistorica, LimiteHechosComparacionOrganizacion+1)
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			a, d := instantaneaComparacionEjemplo(), instantaneaComparacionEjemplo()
			cambiar(&d)
			r, err := CompararOrganizacionHistorica(a, d)
			if !errors.Is(err, ErrComparacionOrganizacionHistoricaInvalida) || !reflect.DeepEqual(r, ComparacionOrganizacionHistorica{}) {
				t.Fatalf("no rechazada: %+v %v", r, err)
			}
		})
	}
}
