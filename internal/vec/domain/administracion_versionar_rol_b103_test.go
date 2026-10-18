package domain

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

// Las listas históricas construyen los datos y cotejan las fuentes de estas pruebas.
// El plan en producción toma las concesiones del catálogo publicado.
// El orden forma parte del material propuesto a los dos ADMIN. Las tres
// primeras capacidades son RRHH interno; las lecturas propias pertenecen al
// perfil externo y no se incorporan a este rol.
var accionesVersionarRolBolsaB103 = [...]string{
	"bolsa.inscripcion.rrhh.listar",
	"bolsa.inscripcion.rrhh.consultar",
	"bolsa.inscripcion.rrhh.convocatorias.listar",
	"bolsa.rrhh.respuestas_llamamiento.consultar",
}

var accionesInscripcionRRHHB103 = [...]string{
	"bolsa.inscripcion.rrhh.motivos", "bolsa.inscripcion.rrhh.decidir", "bolsa.inscripcion.rrhh.incorporar",
}

func concesionDeltaRRHHB103(c ConcesionRol, accion string) bool {
	if !slices.Contains(accionesInscripcionRRHHB103[:], accion) {
		return concesionVersionarRolBolsaB103(c, accion)
	}
	if c.Validar() != nil || c.ModuloID != "bolsa" || c.Accion != accion ||
		c.GarantiaMinima != AuthAssuranceHigh || len(c.Obligaciones) != 0 {
		return false
	}
	switch accion {
	case accionesInscripcionRRHHB103[0]:
		return c.TipoRecurso == "motivos_inscripcion" &&
			slices.Equal(c.Finalidades, []string{"consulta_motivos_inscripcion_rrhh"}) &&
			slices.Equal(c.CamposPermitidos, []string{"catalogo_version", "motivos[].codigo", "motivos[].etiqueta", "motivos[].obligatorio"})
	case accionesInscripcionRRHHB103[1]:
		return c.TipoRecurso == "solicitud_inscripcion" && slices.Equal(c.Finalidades, []string{"revisar_inscripcion"}) && len(c.CamposPermitidos) == 0
	case accionesInscripcionRRHHB103[2]:
		return c.TipoRecurso == "solicitud_inscripcion" && slices.Equal(c.Finalidades, []string{"incorporar_inscripcion"}) && len(c.CamposPermitidos) == 0
	}
	return false
}

func accionPropiaInscripcionBolsa(accion string) bool {
	return accion == "bolsa.inscripcion.propias.listar" || accion == "bolsa.inscripcion.propia.consultar"
}

func camposRRHHBolsaB103(accion string) []string {
	requisitos := []string{"codigo", "descripcion", "obligatorio", "estado", "motivo_codigo", "motivo_etiqueta",
		"procedencia_ref", "hito_cumplimiento", "hito_etiqueta", "hito_fecha"}
	solicitud := []string{"solicitud_ref", "recibo_ref", "convocatoria_ref", "convocatoria_titulo", "categoria_ref", "bolsa_ref",
		"categoria", "declaracion_ref", "bases_ref", "catalogo_version", "plazo_inicio", "plazo_fin",
		"decision_ref", "estado", "version", "registrada_en", "decidida_en", "motivo_codigo",
		"motivo_etiqueta", "participacion_ref", "persona_resumen"}
	for _, hoja := range requisitos {
		solicitud = append(solicitud, "requisitos[]."+hoja)
	}
	var campos []string
	switch accion {
	case accionesVersionarRolBolsaB103[0]:
		campos = []string{"total", "cursor_siguiente", "convocatoria_titulo"}
		for _, hoja := range solicitud {
			campos = append(campos, "solicitudes[]."+hoja)
		}
	case accionesVersionarRolBolsaB103[1]:
		campos = solicitud
	case accionesVersionarRolBolsaB103[2]:
		campos = []string{"convocatorias[].convocatoria_ref", "convocatorias[].titulo", "convocatorias[].categorias_resumen",
			"convocatorias[].plazo_fin", "convocatorias[].estado_publicacion", "convocatorias[].pendientes", "total", "cursor_siguiente"}
	case AccionLecturaOrigenContactoBolsa:
		campos = CamposLecturaOrigenContactoBolsa()
	case accionesVersionarRolBolsaB103[3]:
		campos = []string{"llamamiento_ref", "respuestas[].participacion_ref", "respuestas[].respuesta"}
	default:
		return nil
	}
	slices.Sort(campos)
	return campos
}

func concesionVersionarRolBolsaB103(c ConcesionRol, accion string) bool {
	if c.Validar() != nil || c.ModuloID != "bolsa" || c.Accion != accion || c.GarantiaMinima != AuthAssuranceHigh ||
		!slices.Equal(c.CamposPermitidos, camposRRHHBolsaB103(accion)) {
		return false
	}
	switch accion {
	case AccionLecturaOrigenContactoBolsa:
		return (len(c.Obligaciones) == 0 || slices.Equal(c.Obligaciones, []string{"auditar"})) && c.TipoRecurso == "participacion_bolsa" && slices.Equal(c.Finalidades, []string{"consulta_origen_datos_contacto_participacion"})
	case accionesVersionarRolBolsaB103[0], accionesVersionarRolBolsaB103[1]:
		return len(c.Obligaciones) == 0 && c.TipoRecurso == "solicitud_inscripcion" &&
			slices.Equal(c.Finalidades, []string{"consulta_inscripcion_rrhh"})
	case accionesVersionarRolBolsaB103[2]:
		return len(c.Obligaciones) == 0 && c.TipoRecurso == "conjunto_gestion_inscripcion" &&
			slices.Equal(c.Finalidades, []string{"consulta_convocatorias_gestion_rrhh"})
	case accionesVersionarRolBolsaB103[3]:
		return (len(c.Obligaciones) == 0 || slices.Equal(c.Obligaciones, []string{"auditar"})) &&
			c.TipoRecurso == "llamamiento_bolsa" && slices.Equal(c.Finalidades, []string{"seguimiento_contacto_rrhh"})
	default:
		return false
	}
}

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
		{"seleccion_duplicada", func(_ *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaB103) {
			i.Selecciones[1] = i.Selecciones[0]
		}},
		{"fuente_distinta", func(_ *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaB103) {
			i.Selecciones[0].EntradaHuellaSHA256 = strings.Repeat("a", 64)
		}},
		{"clase_admin", func(c *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaB103) {
			c.Entradas[0].ClaseControl = "administrativo"
			refrescarCatalogoB103(t, c, i)
		}},
		{"dimension_extra", func(c *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaB103) {
			c.Entradas[0].DimensionesAmbito = append(c.Entradas[0].DimensionesAmbito, "bolsa_ref")
			refrescarCatalogoB103(t, c, i)
		}},
		{"entrada_caducada", func(c *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaB103) {
			c.Entradas[0].VigenteHasta = c.Entradas[0].VigenteDesde.Add(time.Second)
			refrescarCatalogoB103(t, c, i)
		}},
		{"accion_externa", func(c *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaB103) {
			c.Entradas[0].Concesion.Accion = "bolsa.inscripcion.propias.listar"
			refrescarCatalogoB103(t, c, i)
		}},
		{"modulo_ajeno", func(c *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaB103) {
			c.Entradas[0].Concesion.ModuloID = "administracion"
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

func TestVersionarRolBolsaB103CopiaSeleccionVariableDelCatalogo(t *testing.T) {
	c, i, ahora := escenarioVersionarRolBolsaB103(t)
	c.Entradas = c.Entradas[:2]
	i.Selecciones = i.Selecciones[:2]
	c.Entradas[0].Concesion = ConcesionRol{
		Accion: "bolsa.rrhh.informe.consultar", ModuloID: "bolsa", TipoRecurso: "informe_bolsa",
		Finalidades: []string{"consulta_informe_rrhh"}, GarantiaMinima: AuthAssuranceSubstantial,
		CamposPermitidos: []string{"referencia", "estado"}, Obligaciones: []string{"auditar"},
	}
	refrescarCatalogoB103(t, &c, &i)
	i.Selecciones[0], i.Selecciones[1] = i.Selecciones[1], i.Selecciones[0]
	plan, err := PrepararPlanVersionarRolBolsaB103(c, i, ahora)
	if err != nil || len(plan.Selecciones) != 2 || len(plan.DefinicionNueva.Concesiones) != len(c.Perfiles[0].Rol.Concesiones)+2 {
		t.Fatalf("selección variable: %v", err)
	}
	base := len(c.Perfiles[0].Rol.Concesiones)
	if !reflect.DeepEqual(plan.DefinicionNueva.Concesiones[base], c.Entradas[1].Concesion) ||
		!reflect.DeepEqual(plan.DefinicionNueva.Concesiones[base+1], c.Entradas[0].Concesion) {
		t.Fatal("la concesión no procede íntegra de la entrada seleccionada")
	}
	if _, err := plan.HuellaSHA256(); err != nil {
		t.Fatal(err)
	}
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
	hOriginal, err := p.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	p.DefinicionNueva.Concesiones[len(p.Base.Rol.Concesiones)].CamposPermitidos = append(CamposLecturaOrigenContactoBolsa(), "correo")
	hDistinta, err := p.HuellaSHA256()
	if err != nil || hDistinta == hOriginal {
		t.Fatal("la huella no refleja la alteración del plan")
	}
	c.Perfiles[0].Rol.Concesiones = c.Perfiles[0].Rol.Concesiones[:1]
	refrescarCatalogoB103(t, &c, &i)
	if _, err := PrepararPlanVersionarRolBolsaB103(c, i, ahora); err != nil {
		t.Fatal("la entrada vigente no depende de una lista histórica de concesiones")
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
	c.Entradas[0].Concesion.CamposPermitidos = append(c.Entradas[0].Concesion.CamposPermitidos, "dni")
	actualizar()
	actualizado, err := PrepararPlanVersionarRolBolsaB103(c, i, ahora)
	if err != nil || !reflect.DeepEqual(actualizado.DefinicionNueva.Concesiones[1], c.Entradas[0].Concesion) {
		t.Fatal("la nueva concesión debe copiar los campos publicados")
	}
}
