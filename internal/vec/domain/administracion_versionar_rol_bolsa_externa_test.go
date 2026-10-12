package domain

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

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
