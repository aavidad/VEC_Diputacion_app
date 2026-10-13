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

func TestVersionarRolBolsaB103AmpliaSoloOrigenDeContactoConservandoBase(t *testing.T) {
	c, i, ahora := escenarioVersionarRolBolsaB103(t)
	for _, e := range c.Entradas {
		c.Perfiles[0].Rol.Concesiones = append(c.Perfiles[0].Rol.Concesiones, e.Concesion)
	}
	e := c.Entradas[0]
	e.Referencia = "accion:contacto:origen"
	e.Concesion = ConcesionRol{Accion: AccionLecturaOrigenContactoBolsa, ModuloID: "bolsa", TipoRecurso: "participacion_bolsa", Finalidades: []string{"consulta_origen_datos_contacto_participacion"}, GarantiaMinima: AuthAssuranceHigh, CamposPermitidos: CamposLecturaOrigenContactoBolsa(), Obligaciones: []string{"auditar"}}
	c.Entradas = []EntradaAccionAdministracionV1{e}
	i.Selecciones = i.Selecciones[:1]
	i.Selecciones[0].EntradaRef = e.Referencia
	refrescarCatalogoB103(t, &c, &i)
	p, err := PrepararPlanVersionarRolBolsaB103(c, i, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Selecciones) != 1 || len(p.DefinicionNueva.Concesiones) != len(p.Base.Rol.Concesiones)+1 || !reflect.DeepEqual(p.DefinicionNueva.Concesiones[:len(p.Base.Rol.Concesiones)], p.Base.Rol.Concesiones) {
		t.Fatal("alteró concesiones previas")
	}
	if _, err := p.HuellaSHA256(); err != nil {
		t.Fatal(err)
	}
	p.DefinicionNueva.Concesiones[len(p.Base.Rol.Concesiones)].CamposPermitidos = append(CamposLecturaOrigenContactoBolsa(), "correo")
	if _, err := p.HuellaSHA256(); err == nil {
		t.Fatal("amplió al contacto completo")
	}
	c.Perfiles[0].Rol.Concesiones = c.Perfiles[0].Rol.Concesiones[:1]
	refrescarCatalogoB103(t, &c, &i)
	if _, err := PrepararPlanVersionarRolBolsaB103(c, i, ahora); err == nil {
		t.Fatal("permitió extensión sin base B103")
	}
}

func TestVersionarRolBolsaB103InscripcionRRHHExacta(t *testing.T) {
	c, i, ahora := escenarioVersionarRolBolsaB103(t)
	for _, entrada := range c.Entradas {
		c.Perfiles[0].Rol.Concesiones = append(c.Perfiles[0].Rol.Concesiones, entrada.Concesion)
	}
	contacto := ConcesionRol{Accion: AccionLecturaOrigenContactoBolsa, ModuloID: "bolsa",
		TipoRecurso: "participacion_bolsa", Finalidades: []string{"consulta_origen_datos_contacto_participacion"},
		GarantiaMinima: AuthAssuranceHigh, CamposPermitidos: CamposLecturaOrigenContactoBolsa()}
	c.Perfiles[0].Rol.Concesiones = append(c.Perfiles[0].Rol.Concesiones, contacto)
	c.Perfiles[0].Rol.Version = 8
	c.Perfiles[0].ControlVigencia.VersionRolRef = c.Perfiles[0].Rol.Referencia()
	i.Asignaciones[0].Documento.VersionRolRef = c.Perfiles[0].Rol.Referencia()
	c.Entradas = nil
	i.Selecciones = nil
	for j, accion := range accionesInscripcionRRHHB103 {
		concesion := ConcesionRol{Accion: accion, ModuloID: "bolsa", GarantiaMinima: AuthAssuranceHigh}
		switch j {
		case 0:
			concesion.TipoRecurso = "motivos_inscripcion"
			concesion.Finalidades = []string{"consulta_motivos_inscripcion_rrhh"}
			concesion.CamposPermitidos = []string{"catalogo_version", "motivos[].codigo", "motivos[].etiqueta", "motivos[].obligatorio"}
		case 1:
			concesion.TipoRecurso = "solicitud_inscripcion"
			concesion.Finalidades = []string{"revisar_inscripcion"}
		case 2:
			concesion.TipoRecurso = "solicitud_inscripcion"
			concesion.Finalidades = []string{"incorporar_inscripcion"}
		}
		e := EntradaAccionAdministracionV1{Referencia: "accion:inscripcion:rrhh:" + string(rune('a'+j)), Version: 1,
			FuenteRef: "fuente:bolsa", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("b", 64),
			Concesion: concesion, DimensionesAmbito: []string{"unidad_ref", "ambito_ref"},
			ClaseControl: "ordinario", VigenteDesde: c.VigenteDesde}
		c.Entradas = append(c.Entradas, e)
		i.Selecciones = append(i.Selecciones, SeleccionAccionAdministracionV1{EntradaRef: e.Referencia, EntradaVersion: 1})
	}
	actualizar := func() {
		t.Helper()
		for j := range c.Entradas {
			h, err := c.Entradas[j].HuellaSHA256()
			if err != nil {
				t.Fatal(err)
			}
			i.Selecciones[j].EntradaHuellaSHA256 = h
		}
		h, err := c.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		i.CatalogoHuellaSHA256 = h
		h, err = c.Perfiles[0].Rol.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		i.BaseHuellaSHA256 = h
		i.BaseRef = c.Perfiles[0].Rol.Referencia()
		h, err = c.Perfiles[0].ControlVigencia.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		i.ControlHuellaSHA256 = h
		i.Asignaciones[0].AsignacionRef = i.Asignaciones[0].Documento.Referencia()
		h, err = i.Asignaciones[0].Documento.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		i.Asignaciones[0].HuellaSHA256 = h
	}
	actualizar()
	plan, err := PrepararPlanVersionarRolBolsaB103(c, i, ahora)
	if err != nil || len(plan.DefinicionNueva.Concesiones) != len(c.Perfiles[0].Rol.Concesiones)+3 {
		t.Fatalf("delta RRHH inválido: %v", err)
	}
	c.Perfiles[0].Rol.Concesiones = c.Perfiles[0].Rol.Concesiones[:1]
	c.Perfiles[0].Rol.Version = 7
	c.Perfiles[0].ControlVigencia.VersionRolRef = c.Perfiles[0].Rol.Referencia()
	i.Asignaciones[0].Documento.VersionRolRef = c.Perfiles[0].Rol.Referencia()
	actualizar()
	plan, err = PrepararPlanVersionarRolBolsaB103(c, i, ahora)
	if err != nil || len(plan.DefinicionNueva.Concesiones) != 4 {
		t.Fatalf("delta RRHH desde v7 sin B103 previo inválido: %v", err)
	}
	posterior := c.Perfiles[0].Rol
	posterior.Version = plan.DefinicionNueva.Version
	posterior.Concesiones = copiarConcesionesVersionarBolsa(plan.DefinicionNueva.Concesiones)
	if len(accionesDeltaVersionarBolsa(posterior, 4)) != 4 {
		t.Fatal("inscripción previa bloqueó B103")
	}
	baseB103, _, _ := escenarioVersionarRolBolsaB103(t)
	for _, entrada := range baseB103.Entradas {
		posterior.Concesiones = append(posterior.Concesiones, entrada.Concesion)
	}
	if len(accionesDeltaVersionarBolsa(posterior, 1)) != 1 {
		t.Fatal("inscripción previa bloqueó C3")
	}
	c.Entradas[0].Concesion.CamposPermitidos = append(c.Entradas[0].Concesion.CamposPermitidos, "dni")
	actualizar()
	if _, err := PrepararPlanVersionarRolBolsaB103(c, i, ahora); err == nil {
		t.Fatal("aceptó campo extra")
	}
}
