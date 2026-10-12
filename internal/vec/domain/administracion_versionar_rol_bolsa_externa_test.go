package domain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestVersionarRolInscripcionCoincideConFuenteOchoAcciones(t *testing.T) {
	ruta := filepath.Join("..", "..", "..", "deploy", "principal", "inscripcion_recorrido_20261013", "inscripcion_bolsa_actos_v1.json")
	material, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	var fuente struct {
		Entradas []struct {
			Concesion   ConcesionRol `json:"concesion"`
			Dimensiones []string     `json:"dimensiones_ambito"`
		} `json:"entradas"`
	}
	if err := json.Unmarshal(material, &fuente); err != nil || len(fuente.Entradas) != 8 {
		t.Fatalf("fuente de inscripción ilegible: %v", err)
	}
	vistos := map[string]bool{}
	for _, entrada := range fuente.Entradas {
		accion := entrada.Concesion.Accion
		if vistos[accion] {
			t.Fatalf("acción repetida: %s", accion)
		}
		vistos[accion] = true
		switch {
		case slices.Contains(accionesInscripcionBolsaExterna[:], accion):
			if !dimensionesCandidatoBolsaExterna(entrada.Dimensiones) || !concesionDeltaBolsaExterna(entrada.Concesion, accion) {
				t.Fatalf("acción externa divergente: %s", accion)
			}
		case slices.Contains(accionesInscripcionRRHHB103[:], accion):
			if !dimensionesRRHHBolsaB103(entrada.Dimensiones) || !concesionDeltaRRHHB103(entrada.Concesion, accion) {
				t.Fatalf("acción RRHH divergente: %s", accion)
			}
		default:
			t.Fatalf("acción no asignada a rol: %s", accion)
		}
	}
}

func escenarioVersionarRolBolsaExterna(t *testing.T) (CatalogoAccionesAdministracionV1, PerfilPublicadoAdministracionV1, IntencionVersionarRolBolsaExterna, time.Time) {
	t.Helper()
	c, previo, ahora := escenarioVersionarRolBolsaB1(t)
	c.Referencia = "catalogo:externo"
	c.Perfiles = nil // El censo del catálogo no fabrica el rol externo.
	base := PerfilPublicadoAdministracionV1{TipoPerfil: TipoPerfilAdministracionAdministrableV1}
	base.Rol = VersionRol{RolID: RolIDVersionarBolsaExterna, Version: 1,
		Nombre: "areaPersonal.miBolsa.rolPortal", Estado: EstadoVersionRolPublicada,
		PublicadaPor: "seguridad:desarrollo:no-autoritativa", PublicadaEn: ahora.Add(-time.Hour)}
	for _, accion := range accionesBaseVersionarRolBolsaExterna {
		recurso, finalidad := "participaciones_candidato", "gestion_participaciones_propias"
		var campos []string
		switch accion {
		case "bolsa.historial_propio.consultar":
			finalidad = "consulta_historial_propio"
			campos = []string{"contratos_propios", "llamamientos_propios", "renuncias_propias"}
		case "bolsa.participaciones_propias.consultar":
			finalidad = "consulta_participaciones_propias"
			campos = []string{"participaciones_candidato_minimizadas"}
		case "bolsa.participaciones_propias.manifestar_disposicion":
			recurso = "oferta_bolsa"
		}
		base.Rol.Concesiones = append(base.Rol.Concesiones, ConcesionRol{Accion: accion,
			ModuloID: "bolsa", TipoRecurso: recurso, Finalidades: []string{finalidad},
			GarantiaMinima: AuthAssuranceHigh, CamposPermitidos: campos})
	}
	base.ControlVigencia = ControlVigenciaVersionRol{VersionRolRef: base.Rol.Referencia(), Revision: 1,
		Estado: EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: "per_fuente", ActualizadoEn: ahora.Add(-time.Hour)}
	c.Entradas = nil
	var selecciones []SeleccionAccionAdministracionV1
	for j, accion := range accionesVersionarRolBolsaExterna {
		e := EntradaAccionAdministracionV1{Referencia: "accion:externa:" + string(rune('a'+j)), Version: 1,
			FuenteRef: "fuente:bolsa", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("b", 64),
			Concesion: ConcesionRol{Accion: accion, ModuloID: "bolsa", TipoRecurso: "participaciones_candidato",
				Finalidades: []string{"gestion_preferencias_bolsas_propias"}, GarantiaMinima: AuthAssuranceHigh,
				CamposPermitidos: camposPreferenciasBolsaExterna(accion)},
			DimensionesAmbito: []string{"candidato_ref"}, ClaseControl: string(ClaseControlPerfilOrdinario),
			VigenteDesde: c.VigenteDesde}
		h, err := e.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		c.Entradas = append(c.Entradas, e)
		selecciones = append(selecciones, SeleccionAccionAdministracionV1{EntradaRef: e.Referencia,
			EntradaVersion: 1, EntradaHuellaSHA256: h})
	}
	asignacion := previo.Asignaciones[0].Documento
	asignacion.Version = 1
	asignacion.VersionRolRef = base.Rol.Referencia()
	asignacion.Ambitos = []AmbitoPerfil{{Clave: "candidato_ref", Valores: []string{"can_abcdefghijklmnopqrstuv"}}}
	ha, err := asignacion.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	hc, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	hr, err := base.Rol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	hcontrol, err := base.ControlVigencia.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	i := IntencionVersionarRolBolsaExterna{CatalogoRef: c.Referencia, CatalogoVersion: c.Version,
		CatalogoHuellaSHA256: hc, BaseRef: base.Rol.Referencia(), BaseHuellaSHA256: hr,
		ControlRevision: base.ControlVigencia.Revision, ControlHuellaSHA256: hcontrol,
		Selecciones: selecciones, Asignaciones: []SeleccionAsignacionVersionarRolBolsa{{
			AsignacionRef: asignacion.Referencia(), HuellaSHA256: ha, Documento: asignacion}}, Motivo: previo.Motivo}
	return c, base, i, ahora
}

func TestVersionarRolBolsaExternaOrdenSinCamposEnCanonGo(t *testing.T) {
	c, base, i, ahora := escenarioVersionarRolBolsaExterna(t)
	orden, err := json.Marshal(c.Entradas[1].Concesion)
	if err != nil {
		t.Fatal(err)
	}
	var canon map[string]json.RawMessage
	if err := json.Unmarshal(orden, &canon); err != nil {
		t.Fatal(err)
	}
	if _, presente := canon["campos_permitidos"]; presente {
		t.Fatal("el canon Go de ordenar incluye campos vacíos")
	}
	if _, err := PrepararPlanVersionarRolBolsaExterna(c, base, i, ahora); err != nil {
		t.Fatal("el plan válido perdió la concesión sin campos:", err)
	}
}

func TestVersionarRolBolsaExternaV2ConservaBaseYExigeDosConcesiones(t *testing.T) {
	c, base, i, ahora := escenarioVersionarRolBolsaExterna(t)
	plan, err := PrepararPlanVersionarRolBolsaExterna(c, base, i, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if plan.VersionRolObjetivoRef != "rol:"+RolIDVersionarBolsaExterna+":v2" ||
		len(plan.DefinicionNueva.Concesiones) != 9 ||
		!reflect.DeepEqual(plan.Base.Rol.Concesiones, base.Rol.Concesiones) ||
		len(plan.Asignaciones) != 1 || !ambitosCandidatoBolsaExterna(plan.Asignaciones[0].Documento.Ambitos) {
		t.Fatal("plan externo perdió base o asignación")
	}
	base.Rol.Concesiones[0].Finalidades[0] = "alterada"
	i.Asignaciones[0].Documento.Ambitos[0].Valores[0] = "can_otro"
	if plan.Base.Rol.Concesiones[0].Finalidades[0] == "alterada" ||
		plan.Asignaciones[0].Documento.Ambitos[0].Valores[0] == "can_otro" {
		t.Fatal("plan comparte memoria")
	}
}

func TestVersionarRolBolsaExternaInscripcionCincoExactas(t *testing.T) {
	c, base, previa, ahora := escenarioVersionarRolBolsaExterna(t)
	base.Rol.Nombre = "Consulta y acciones propias de bolsa en desarrollo"
	for _, e := range c.Entradas {
		base.Rol.Concesiones = append(base.Rol.Concesiones, e.Concesion)
	}
	base.Rol.Version = 2
	base.ControlVigencia.VersionRolRef = base.Rol.Referencia()
	i := IntencionVersionarRolBolsaExterna{CatalogoRef: c.Referencia, CatalogoVersion: c.Version,
		ControlRevision: base.ControlVigencia.Revision, Motivo: previa.Motivo}
	i.Asignaciones = previa.Asignaciones
	i.Asignaciones[0].Documento.VersionRolRef = base.Rol.Referencia()
	c.Entradas = nil
	for j, accion := range accionesInscripcionBolsaExterna {
		concesion := ConcesionRol{Accion: accion, ModuloID: "bolsa", GarantiaMinima: AuthAssuranceHigh,
			CamposPermitidos: camposInscripcionBolsaExterna(accion)}
		switch j {
		case 0, 1:
			concesion.TipoRecurso = "convocatoria_inscripcion"
			concesion.Finalidades = []string{"consulta_convocatoria_abierta"}
		case 2, 3:
			concesion.TipoRecurso = "solicitud_inscripcion"
			concesion.Finalidades = []string{"consulta_inscripcion_propia"}
		case 4:
			concesion.TipoRecurso = "inscripcion_convocatoria"
			concesion.Finalidades = []string{"presentar_inscripcion"}
		}
		e := EntradaAccionAdministracionV1{Referencia: "accion:inscripcion:externa:" + string(rune('a'+j)),
			Version: 1, FuenteRef: "fuente:bolsa", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("b", 64),
			Concesion: concesion, DimensionesAmbito: []string{"candidato_ref"},
			ClaseControl: "ordinario", VigenteDesde: c.VigenteDesde}
		c.Entradas = append(c.Entradas, e)
		h, err := e.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		i.Selecciones = append(i.Selecciones, SeleccionAccionAdministracionV1{EntradaRef: e.Referencia, EntradaVersion: 1, EntradaHuellaSHA256: h})
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
		i.BaseRef = base.Rol.Referencia()
		h, err = base.Rol.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		i.BaseHuellaSHA256 = h
		h, err = base.ControlVigencia.HuellaSHA256()
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
	plan, err := PrepararPlanVersionarRolBolsaExterna(c, base, i, ahora)
	if err != nil || len(plan.DefinicionNueva.Concesiones) != 14 {
		t.Fatalf("delta aspirante inválido: %v", err)
	}
	baseV1 := base
	baseV1.Rol.Version = 1
	baseV1.Rol.Concesiones = copiarConcesionesVersionarBolsa(base.Rol.Concesiones[:7])
	baseV1.ControlVigencia.VersionRolRef = baseV1.Rol.Referencia()
	iV1 := i
	iV1.Asignaciones = append([]SeleccionAsignacionVersionarRolBolsa(nil), i.Asignaciones...)
	iV1.Asignaciones[0].Documento.VersionRolRef = baseV1.Rol.Referencia()
	iV1.BaseRef = baseV1.Rol.Referencia()
	h, err := baseV1.Rol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	iV1.BaseHuellaSHA256 = h
	h, err = baseV1.ControlVigencia.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	iV1.ControlHuellaSHA256 = h
	h, err = iV1.Asignaciones[0].Documento.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	iV1.Asignaciones[0].AsignacionRef = iV1.Asignaciones[0].Documento.Referencia()
	iV1.Asignaciones[0].HuellaSHA256 = h
	planV1, err := PrepararPlanVersionarRolBolsaExterna(c, baseV1, iV1, ahora)
	if err != nil || len(planV1.DefinicionNueva.Concesiones) != 12 {
		t.Fatalf("delta desde v1 inválido: %v", err)
	}
	cPreferencias, basePreferencias, iPreferencias, _ := escenarioVersionarRolBolsaExterna(t)
	basePreferencias.Rol.Version = planV1.DefinicionNueva.Version
	basePreferencias.Rol.Concesiones = copiarConcesionesVersionarBolsa(planV1.DefinicionNueva.Concesiones)
	basePreferencias.ControlVigencia.VersionRolRef = basePreferencias.Rol.Referencia()
	iPreferencias.BaseRef = basePreferencias.Rol.Referencia()
	h, err = basePreferencias.Rol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	iPreferencias.BaseHuellaSHA256 = h
	h, err = basePreferencias.ControlVigencia.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	iPreferencias.ControlHuellaSHA256 = h
	iPreferencias.Asignaciones[0].Documento.VersionRolRef = basePreferencias.Rol.Referencia()
	iPreferencias.Asignaciones[0].AsignacionRef = iPreferencias.Asignaciones[0].Documento.Referencia()
	h, err = iPreferencias.Asignaciones[0].Documento.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	iPreferencias.Asignaciones[0].HuellaSHA256 = h
	preferenciasPosteriores, err := PrepararPlanVersionarRolBolsaExterna(cPreferencias, basePreferencias, iPreferencias, ahora)
	if err != nil || preferenciasPosteriores.DefinicionNueva.Version != 3 ||
		len(preferenciasPosteriores.DefinicionNueva.Concesiones) != 14 {
		t.Fatalf("preferencias tras inscripción inválidas: %v", err)
	}
	baseNoAcreditada := basePreferencias.Rol
	baseNoAcreditada.Version = 3
	if accionesDeltaBolsaExterna(baseNoAcreditada, 2) != nil ||
		accionesDeltaBolsaExterna(baseNoAcreditada, 5) != nil {
		t.Fatal("aceptó una cabeza externa sin soporte AUT84")
	}
	c.Entradas[0].Concesion.CamposPermitidos = append(c.Entradas[0].Concesion.CamposPermitidos, "convocatorias[].dni")
	actualizar()
	if _, err := PrepararPlanVersionarRolBolsaExterna(c, base, i, ahora); err == nil {
		t.Fatal("aceptó campo extra")
	}
}

func TestVersionarRolBolsaExternaConservaNombrePublicado(t *testing.T) {
	for _, nombre := range []string{
		"areaPersonal.miBolsa.rolPortal",
		"Consulta y acciones propias de bolsa en desarrollo",
	} {
		t.Run(nombre, func(t *testing.T) {
			c, base, i, ahora := escenarioVersionarRolBolsaExterna(t)
			base.Rol.Nombre = nombre
			refrescarBaseExterna(t, &base, &i)
			plan, err := PrepararPlanVersionarRolBolsaExterna(c, base, i, ahora)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Base.Rol.Nombre != nombre || plan.DefinicionNueva.Nombre != nombre {
				t.Fatal("el plan cambió el nombre publicado del rol")
			}
		})
	}
}

func TestVersionarRolBolsaExternaRecuperaPlanTrasVencerDescriptor(t *testing.T) {
	c, base, i, ahora := escenarioVersionarRolBolsaExterna(t)
	vencimiento := ahora.Add(time.Minute)
	c.VigenteHasta = vencimiento
	for j := range c.Entradas {
		c.Entradas[j].VigenteHasta = vencimiento
		huella, err := c.Entradas[j].HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		i.Selecciones[j].EntradaHuellaSHA256 = huella
	}
	huellaCatalogo, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	i.CatalogoHuellaSHA256 = huellaCatalogo
	primero, err := PrepararPlanVersionarRolBolsaExterna(c, base, i, ahora)
	if err != nil {
		t.Fatal(err)
	}
	despues := vencimiento.Add(time.Second)
	if _, err := PrepararPlanVersionarRolBolsaExterna(c, base, i, despues); err == nil {
		t.Fatal("aceptó crear una propuesta con descriptor caducado")
	}
	recuperado, err := ReconstruirPlanVersionarRolBolsaExterna(c, base, i, despues)
	if err != nil {
		t.Fatal(err)
	}
	antesSHA, err := primero.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	despuesSHA, err := recuperado.HuellaSHA256()
	if err != nil || despuesSHA != antesSHA {
		t.Fatal("la recuperación cambió la intención confirmada")
	}
}

func TestVersionarRolBolsaExternaDeniegaAmpliacionesYPreimagenes(t *testing.T) {
	for _, caso := range []struct {
		name  string
		mutar func(*CatalogoAccionesAdministracionV1, *PerfilPublicadoAdministracionV1, *IntencionVersionarRolBolsaExterna)
	}{
		{"rol_rrhh", func(_ *CatalogoAccionesAdministracionV1, b *PerfilPublicadoAdministracionV1, i *IntencionVersionarRolBolsaExterna) {
			b.Rol.RolID = RolIDVersionarBolsaB1
			refrescarBaseExterna(t, b, i)
		}},
		{"version_tres", func(_ *CatalogoAccionesAdministracionV1, b *PerfilPublicadoAdministracionV1, i *IntencionVersionarRolBolsaExterna) {
			b.Rol.Version = 2
			refrescarBaseExterna(t, b, i)
		}},
		{"orden", func(_ *CatalogoAccionesAdministracionV1, _ *PerfilPublicadoAdministracionV1, i *IntencionVersionarRolBolsaExterna) {
			i.Selecciones[0], i.Selecciones[1] = i.Selecciones[1], i.Selecciones[0]
		}},
		{"ambito_rrhh", func(_ *CatalogoAccionesAdministracionV1, _ *PerfilPublicadoAdministracionV1, i *IntencionVersionarRolBolsaExterna) {
			i.Asignaciones[0].Documento.Ambitos[0].Clave = "unidad_ref"
		}},
		{"preimagen", func(_ *CatalogoAccionesAdministracionV1, _ *PerfilPublicadoAdministracionV1, i *IntencionVersionarRolBolsaExterna) {
			i.Asignaciones[0].Documento.PrincipalID = "per_otra"
		}},
		{"campo_dni", func(c *CatalogoAccionesAdministracionV1, _ *PerfilPublicadoAdministracionV1, i *IntencionVersionarRolBolsaExterna) {
			c.Entradas[0].Concesion.CamposPermitidos = append(c.Entradas[0].Concesion.CamposPermitidos, "dni")
			refrescarCatalogoExterno(t, c, i)
		}},
		{"put_con_campos", func(c *CatalogoAccionesAdministracionV1, _ *PerfilPublicadoAdministracionV1, i *IntencionVersionarRolBolsaExterna) {
			c.Entradas[1].Concesion.CamposPermitidos = []string{"orden_guardado"}
			refrescarCatalogoExterno(t, c, i)
		}},
		{"otra_finalidad", func(c *CatalogoAccionesAdministracionV1, _ *PerfilPublicadoAdministracionV1, i *IntencionVersionarRolBolsaExterna) {
			c.Entradas[0].Concesion.Finalidades = []string{"otra"}
			refrescarCatalogoExterno(t, c, i)
		}},
	} {
		t.Run(caso.name, func(t *testing.T) {
			c, b, i, ahora := escenarioVersionarRolBolsaExterna(t)
			caso.mutar(&c, &b, &i)
			if _, err := PrepararPlanVersionarRolBolsaExterna(c, b, i, ahora); err == nil {
				t.Fatal("plan ajeno admitido")
			}
		})
	}
}

func refrescarCatalogoExterno(t *testing.T, c *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaExterna) {
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
}

func refrescarBaseExterna(t *testing.T, b *PerfilPublicadoAdministracionV1, i *IntencionVersionarRolBolsaExterna) {
	t.Helper()
	b.ControlVigencia.VersionRolRef = b.Rol.Referencia()
	i.Asignaciones[0].Documento.VersionRolRef = b.Rol.Referencia()
	ha, err := i.Asignaciones[0].Documento.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	i.Asignaciones[0].AsignacionRef = i.Asignaciones[0].Documento.Referencia()
	i.Asignaciones[0].HuellaSHA256 = ha
	hr, err := b.Rol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	hc, err := b.ControlVigencia.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	i.BaseRef, i.BaseHuellaSHA256 = b.Rol.Referencia(), hr
	i.ControlHuellaSHA256 = hc
}
