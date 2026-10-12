package domain

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func escenarioVersionarRolBolsaB103(t *testing.T) (CatalogoAccionesAdministracionV1, IntencionVersionarRolBolsaB103, time.Time) {
	t.Helper()
	base, anterior, ahora := escenarioVersionarRolBolsaB1(t)
	base.Referencia = "catalogo:b103"
	base.Perfiles[0].Rol.Concesiones[0].CamposPermitidos = []string{"estado"}
	concesiones := []ConcesionRol{
		{Accion: accionesVersionarRolBolsaB103[0], ModuloID: "bolsa", TipoRecurso: "solicitud_inscripcion",
			Finalidades: []string{"consulta_inscripcion_rrhh"}, GarantiaMinima: AuthAssuranceHigh,
			CamposPermitidos: camposRRHHBolsaB103(accionesVersionarRolBolsaB103[0]), Obligaciones: []string{}},
		{Accion: accionesVersionarRolBolsaB103[1], ModuloID: "bolsa", TipoRecurso: "solicitud_inscripcion",
			Finalidades: []string{"consulta_inscripcion_rrhh"}, GarantiaMinima: AuthAssuranceHigh,
			CamposPermitidos: camposRRHHBolsaB103(accionesVersionarRolBolsaB103[1]), Obligaciones: []string{}},
		{Accion: accionesVersionarRolBolsaB103[2], ModuloID: "bolsa", TipoRecurso: "conjunto_gestion_inscripcion",
			Finalidades: []string{"consulta_convocatorias_gestion_rrhh"}, GarantiaMinima: AuthAssuranceHigh,
			CamposPermitidos: camposRRHHBolsaB103(accionesVersionarRolBolsaB103[2]), Obligaciones: []string{}},
		{Accion: accionesVersionarRolBolsaB103[3], ModuloID: "bolsa", TipoRecurso: "llamamiento_bolsa",
			Finalidades: []string{"seguimiento_contacto_rrhh"}, GarantiaMinima: AuthAssuranceHigh,
			CamposPermitidos: camposRRHHBolsaB103(accionesVersionarRolBolsaB103[3]), Obligaciones: []string{"auditar"}},
	}
	base.Entradas = nil
	selecciones := make([]SeleccionAccionAdministracionV1, 0, len(concesiones))
	for j, concesion := range concesiones {
		e := EntradaAccionAdministracionV1{Referencia: "accion:b103:" + string(rune('a'+j)), Version: 1,
			FuenteRef: "fuente:bolsa", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("b", 64),
			Concesion: concesion, DimensionesAmbito: []string{"unidad_ref", "ambito_ref"},
			ClaseControl: "ordinario", VigenteDesde: base.VigenteDesde}
		h, err := e.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		base.Entradas = append(base.Entradas, e)
		selecciones = append(selecciones, SeleccionAccionAdministracionV1{EntradaRef: e.Referencia,
			EntradaVersion: e.Version, EntradaHuellaSHA256: h})
	}
	base.Perfiles[0].Rol.Concesiones[0].Finalidades = []string{"gestion_bolsa"}
	asignacion := anterior.Asignaciones[0].Documento
	asignacion.Ambitos = []AmbitoPerfil{{Clave: "unidad_ref", Valores: []string{"unidad:uno"}},
		{Clave: "ambito_ref", Valores: []string{"ambito:uno"}}}
	ha, err := asignacion.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	hc, err := base.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	hr, err := base.Perfiles[0].Rol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	intencion := IntencionVersionarRolBolsaB103{CatalogoRef: base.Referencia, CatalogoVersion: base.Version,
		CatalogoHuellaSHA256: hc, BaseRef: base.Perfiles[0].Rol.Referencia(), BaseHuellaSHA256: hr,
		ControlRevision:     base.Perfiles[0].ControlVigencia.Revision,
		ControlHuellaSHA256: anterior.ControlHuellaSHA256, Selecciones: selecciones,
		Asignaciones: []SeleccionAsignacionVersionarRolBolsa{{AsignacionRef: asignacion.Referencia(),
			HuellaSHA256: ha, Documento: asignacion}}, Motivo: anterior.Motivo}
	return base, intencion, ahora
}

func TestVersionarRolBolsaB103SoloAgregaCuatroConcesionesRRHHExactas(t *testing.T) {
	c, i, ahora := escenarioVersionarRolBolsaB103(t)
	plan, err := PrepararPlanVersionarRolBolsaB103(c, i, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.DefinicionNueva.Concesiones) != len(c.Perfiles[0].Rol.Concesiones)+4 ||
		plan.VersionRolObjetivoRef != "rol:"+RolIDVersionarBolsaB1+":v7" ||
		!reflect.DeepEqual(plan.DefinicionNueva.Concesiones[0], c.Perfiles[0].Rol.Concesiones[0]) ||
		len(plan.Asignaciones) != 1 || !reflect.DeepEqual(plan.Asignaciones[0].Documento.Ambitos, i.Asignaciones[0].Documento.Ambitos) {
		t.Fatalf("plan alteró preimagen: %+v", plan)
	}
	for j, accion := range accionesVersionarRolBolsaB103 {
		concesion := plan.DefinicionNueva.Concesiones[len(c.Perfiles[0].Rol.Concesiones)+j]
		if concesion.Accion != accion || !slices.Equal(concesion.CamposPermitidos, camposRRHHBolsaB103(accion)) ||
			accionPropiaInscripcionBolsa(concesion.Accion) {
			t.Fatalf("capacidad %d distinta: %+v", j, concesion)
		}
	}
	if len(camposRRHHBolsaB103(accionesVersionarRolBolsaB103[0])) != 34 ||
		len(camposRRHHBolsaB103(accionesVersionarRolBolsaB103[1])) != 31 ||
		len(camposRRHHBolsaB103(accionesVersionarRolBolsaB103[2])) != 8 ||
		len(camposRRHHBolsaB103(accionesVersionarRolBolsaB103[3])) != 3 {
		t.Fatal("hojas B99/B103 divergentes")
	}
	if _, err := plan.HuellaSHA256(); err != nil {
		t.Fatal(err)
	}
	c.Perfiles[0].Rol.Concesiones[0].Finalidades[0] = "alterada"
	i.Asignaciones[0].Documento.Ambitos[0].Valores[0] = "alterado"
	if plan.Base.Rol.Concesiones[0].Finalidades[0] == "alterada" ||
		plan.Asignaciones[0].Documento.Ambitos[0].Valores[0] == "alterado" {
		t.Fatal("el plan comparte memoria mutable con sus entradas")
	}
	copia := plan.Copia()
	copia.DefinicionNueva.Concesiones[1].CamposPermitidos[0] = "alterado"
	if plan.DefinicionNueva.Concesiones[1].CamposPermitidos[0] == "alterado" {
		t.Fatal("la copia comparte concesiones")
	}
}

func TestVersionarRolBolsaB103DeniegaConcesionesYAmbitosDivergentes(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*CatalogoAccionesAdministracionV1, *IntencionVersionarRolBolsaB103)
	}{
		{"catalogo", func(_ *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaB103) {
			i.CatalogoHuellaSHA256 = strings.Repeat("f", 64)
		}},
		{"orden", func(_ *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaB103) {
			i.Selecciones[0], i.Selecciones[1] = i.Selecciones[1], i.Selecciones[0]
		}},
		{"hoja_extra", func(c *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaB103) {
			c.Entradas[0].Concesion.CamposPermitidos = append(c.Entradas[0].Concesion.CamposPermitidos, "solicitudes[].dni")
			refrescarCatalogoB103(t, c, i)
		}},
		{"hoja_ausente", func(c *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaB103) {
			c.Entradas[2].Concesion.CamposPermitidos = slices.Delete(c.Entradas[2].Concesion.CamposPermitidos, 0, 1)
			refrescarCatalogoB103(t, c, i)
		}},
		{"obligacion_b99", func(c *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaB103) {
			c.Entradas[1].Concesion.Obligaciones = []string{"auditar"}
			refrescarCatalogoB103(t, c, i)
		}},
		{"accion_externa", func(c *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaB103) {
			c.Entradas[0].Concesion.Accion = "bolsa.inscripcion.propias.listar"
			refrescarCatalogoB103(t, c, i)
		}},
		{"duplicada_base", func(c *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaB103) {
			c.Perfiles[0].Rol.Concesiones = append(c.Perfiles[0].Rol.Concesiones, c.Entradas[0].Concesion)
			refrescarCatalogoB103(t, c, i)
		}},
		{"ambito_bolsa", func(_ *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaB103) {
			i.Asignaciones[0].Documento.Ambitos = append(i.Asignaciones[0].Documento.Ambitos,
				AmbitoPerfil{Clave: "bolsa_ref", Valores: []string{"bolsa:una"}})
		}},
		{"preimagen_asignacion", func(_ *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaB103) {
			i.Asignaciones[0].Documento.PrincipalID = "per_otra"
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			c, i, ahora := escenarioVersionarRolBolsaB103(t)
			caso.mutar(&c, &i)
			if _, err := PrepararPlanVersionarRolBolsaB103(c, i, ahora); err == nil {
				t.Fatal("plan no autorizado aceptado")
			}
		})
	}
}

func refrescarCatalogoB103(t *testing.T, c *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaB103) {
	t.Helper()
	for j := range c.Entradas {
		h, err := c.Entradas[j].HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		i.Selecciones[j].EntradaHuellaSHA256 = h
	}
	hc, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	i.CatalogoHuellaSHA256 = hc
	hr, err := c.Perfiles[0].Rol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	i.BaseHuellaSHA256 = hr
}
