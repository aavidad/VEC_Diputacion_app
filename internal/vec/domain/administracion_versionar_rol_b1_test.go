package domain

import (
	"strings"
	"testing"
	"time"
)

func escenarioVersionarRolBolsaB1(t *testing.T) (CatalogoAccionesAdministracionV1, IntencionVersionarRolBolsa, time.Time) {
	t.Helper()
	desde := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	ahora := desde.Add(24 * time.Hour)
	anterior := ConcesionRol{Accion: "bolsa.llamamiento.consultar", ModuloID: "bolsa", TipoRecurso: "llamamiento",
		Finalidades: []string{"gestion_bolsa"}, GarantiaMinima: AuthAssuranceHigh, CamposPermitidos: []string{}}
	b1 := ConcesionRol{Accion: "bolsa.carga_convoca.confirmar", ModuloID: "bolsa", TipoRecurso: "carga_convoca",
		Finalidades: []string{"carga_bolsa_convoca"}, GarantiaMinima: AuthAssuranceHigh, CamposPermitidos: []string{}}
	rol := VersionRol{RolID: RolIDVersionarBolsaB1, Version: 6, Nombre: "Rol RRHH sintético", Estado: EstadoVersionRolPublicada,
		Concesiones: []ConcesionRol{anterior}, PublicadaPor: "per_fuente", PublicadaEn: desde}
	c := CatalogoAccionesAdministracionV1{Referencia: "catalogo:b1", Version: 1, FuenteRef: "fuente:b1", FuenteVersion: 1,
		FuenteHuellaSHA256: strings.Repeat("a", 64), VigenteDesde: desde,
		Entradas: []EntradaAccionAdministracionV1{{Referencia: "accion:b1", Version: 1, FuenteRef: "fuente:bolsa", FuenteVersion: 1,
			FuenteHuellaSHA256: strings.Repeat("b", 64), Concesion: b1, DimensionesAmbito: []string{"unidad_ref"},
			ClaseControl: "ordinario", VigenteDesde: desde}},
		Perfiles: []PerfilPublicadoAdministracionV1{{Rol: rol, TipoPerfil: TipoPerfilAdministracionAdministrableV1,
			ControlVigencia: ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 3,
				Estado: EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: "per_fuente", ActualizadoEn: desde}}}}
	hc, ec := c.HuellaSHA256()
	hr, er := rol.HuellaSHA256()
	hcontrol, ecc := c.Perfiles[0].ControlVigencia.HuellaSHA256()
	he, ee := c.Entradas[0].HuellaSHA256()
	if ec != nil || er != nil || ecc != nil || ee != nil {
		t.Fatalf("fuente: %v %v %v %v", ec, er, ecc, ee)
	}
	asignacion := AsignacionPerfil{AsignacionID: "asg_rrhh_uno", Version: 4, PerfilActivoRef: "prf_rrhh_uno",
		PrincipalID: "per_rrhh_uno", VersionRolRef: "rol:" + RolIDVersionarBolsaB1 + ":v5",
		Estado: EstadoAsignacionPerfilActiva, Ambitos: []AmbitoPerfil{{Clave: "unidad_ref", Valores: []string{"unidad:uno"}}},
		VigenteDesde: desde, VigenteHasta: desde.Add(72 * time.Hour), EmitidaPor: "per_fuente", EmitidaEn: desde}
	ha, err := asignacion.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	intencion := IntencionVersionarRolBolsa{CatalogoRef: c.Referencia, CatalogoVersion: c.Version,
		CatalogoHuellaSHA256: hc, BaseRef: rol.Referencia(), BaseHuellaSHA256: hr,
		ControlRevision: c.Perfiles[0].ControlVigencia.Revision, ControlHuellaSHA256: hcontrol,
		Seleccion: SeleccionAccionAdministracionV1{EntradaRef: c.Entradas[0].Referencia,
			EntradaVersion: 1, EntradaHuellaSHA256: he},
		Asignaciones: []SeleccionAsignacionVersionarRolBolsa{{AsignacionRef: asignacion.Referencia(),
			HuellaSHA256: ha, Documento: asignacion}},
		Motivo: ReferenciaEntradaCatalogo{CatalogoID: "motivos_administracion", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("e", 64), EntradaClave: "motivo_" + strings.Repeat("f", 32)}}
	return c, intencion, ahora
}

func TestVersionarRolBolsaB1ConservaConcesionesYAsignacionAnterior(t *testing.T) {
	c, s, ahora := escenarioVersionarRolBolsaB1(t)
	plan, err := PrepararPlanVersionarRolBolsa(c, s, ahora)
	if err != nil || plan.DefinicionNueva.Version != 7 || plan.VersionRolObjetivoRef != "rol:"+RolIDVersionarBolsaB1+":v7" ||
		len(plan.DefinicionNueva.Concesiones) != 2 || plan.Asignaciones[0].Documento.VersionRolRef != "rol:"+RolIDVersionarBolsaB1+":v5" ||
		plan.Base.Rol.Concesiones[0].CamposPermitidos == nil || plan.DefinicionNueva.Concesiones[1].CamposPermitidos == nil {
		t.Fatalf("plan B1: %+v %v", plan, err)
	}
	if _, err := plan.HuellaSHA256(); err != nil {
		t.Fatal(err)
	}
	plan.DefinicionNueva.Concesiones[0].Finalidades[0] = "alterada"
	plan.DefinicionNueva.Concesiones[1].Finalidades[0] = "alterada"
	if c.Perfiles[0].Rol.Concesiones[0].Finalidades[0] == "alterada" ||
		c.Entradas[0].Concesion.Finalidades[0] == "alterada" ||
		s.Asignaciones[0].Documento.Ambitos[0].Valores[0] != "unidad:uno" {
		t.Fatal("el plan comparte slices mutables con la fuente")
	}
}

func TestVersionarRolBolsaB1RechazaPreimagenOCapacidadDistinta(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		mutar  func(*CatalogoAccionesAdministracionV1, *IntencionVersionarRolBolsa)
	}{
		{"control", func(_ *CatalogoAccionesAdministracionV1, s *IntencionVersionarRolBolsa) { s.ControlRevision++ }},
		{"rol", func(_ *CatalogoAccionesAdministracionV1, s *IntencionVersionarRolBolsa) { s.BaseRef = "rol:otro:v6" }},
		{"asignacion", func(_ *CatalogoAccionesAdministracionV1, s *IntencionVersionarRolBolsa) {
			s.Asignaciones[0].Documento.PrincipalID = "per_ajena"
		}},
		{"rol_asignado", func(_ *CatalogoAccionesAdministracionV1, s *IntencionVersionarRolBolsa) {
			s.Asignaciones[0].Documento.VersionRolRef = "rol:otro:v5"
		}},
		{"sin_documento", func(_ *CatalogoAccionesAdministracionV1, s *IntencionVersionarRolBolsa) {
			s.Asignaciones[0].Documento = AsignacionPerfil{}
		}},
		{"concesion_alterada", func(c *CatalogoAccionesAdministracionV1, s *IntencionVersionarRolBolsa) {
			c.Entradas[0].Concesion.CamposPermitidos = []string{"dato_personal"}
			hc, _ := c.HuellaSHA256()
			he, _ := c.Entradas[0].HuellaSHA256()
			s.CatalogoHuellaSHA256, s.Seleccion.EntradaHuellaSHA256 = hc, he
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			c, s, ahora := escenarioVersionarRolBolsaB1(t)
			caso.mutar(&c, &s)
			if _, err := PrepararPlanVersionarRolBolsa(c, s, ahora); err == nil {
				t.Fatal("preimagen o capacidad ajena admitida")
			}
		})
	}
}
