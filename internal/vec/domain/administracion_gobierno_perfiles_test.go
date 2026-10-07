package domain

import (
	"strings"
	"testing"
	"time"
)

func gobiernoPerfilPrueba(t *testing.T) (CatalogoAccionesAdministracionV1, SolicitudPlanGobiernoPerfil, time.Time) {
	t.Helper()
	c, p, instante := escenarioCatalogoAccionesAdministracion(t)
	return c, SolicitudPlanGobiernoPerfil{Operacion: OperacionVersionarPerfilGobernado, Publicacion: &p,
		Motivo: ReferenciaEntradaCatalogo{CatalogoID: "motivos_administracion", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("e", 64), EntradaClave: "motivo_" + strings.Repeat("f", 32)}}, instante
}

func retiradaGobiernoPerfilPrueba(t *testing.T) (CatalogoAccionesAdministracionV1, SolicitudPlanGobiernoPerfil, time.Time) {
	t.Helper()
	c, s, instante := gobiernoPerfilPrueba(t)
	hr, err := c.Perfiles[0].Rol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	hcontrol, err := c.Perfiles[0].ControlVigencia.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	s.Operacion, s.Publicacion = OperacionDeshabilitarVersionPerfil, nil
	hc, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	s.Deshabilitacion = &SeleccionRetiradaVersionPerfil{CatalogoRef: c.Referencia, CatalogoVersion: c.Version, CatalogoHuellaSHA256: hc,
		VersionRolRef: c.Perfiles[0].Rol.Referencia(), VersionRolHuellaSHA256: hr,
		ControlRevision: c.Perfiles[0].ControlVigencia.Revision, ControlHuellaSHA256: hcontrol}
	return c, s, instante
}

func TestGobiernoPerfilPlanCreaYVersionaSinAutorDeArchivo(t *testing.T) {
	for _, operacion := range []OperacionGobiernoPerfil{OperacionCrearPerfilGobernado, OperacionVersionarPerfilGobernado} {
		t.Run(string(operacion), func(t *testing.T) {
			c, s, instante := gobiernoPerfilPrueba(t)
			s.Operacion = operacion
			if operacion == OperacionCrearPerfilGobernado {
				s.Publicacion.RolPropuesto.RolID, s.Publicacion.RolPropuesto.Version = "nuevo_sintetico", 1
				s.Publicacion.VersionRolBaseRef, s.Publicacion.VersionRolBaseHuellaSHA256 = "", ""
			}
			plan, err := PrepararPlanGobiernoPerfil(c, s, instante)
			if err != nil || plan.DefinicionNueva == nil || plan.VersionRolObjetivoRef != s.Publicacion.RolPropuesto.Referencia() {
				t.Fatalf("plan=%+v error=%v", plan, err)
			}
			h, err := plan.HuellaSHA256()
			if err != nil {
				t.Fatal(err)
			}
			s.Publicacion.RolPropuesto.PublicadaPor = "actor:archivo:otro"
			s.Publicacion.RolPropuesto.PublicadaEn = s.Publicacion.RolPropuesto.PublicadaEn.Add(-time.Microsecond)
			otro, err := PrepararPlanGobiernoPerfil(c, s, instante)
			if err != nil {
				t.Fatal(err)
			}
			ho, err := otro.HuellaSHA256()
			if err != nil || ho != h {
				t.Fatal("autor/fecha libres del archivo adquirieron autoridad en el plan")
			}
			plan.DefinicionNueva.Concesiones[0].Finalidades[0] = "alterada"
			if c.Entradas[0].Concesion.Finalidades[0] == "alterada" || s.Publicacion.RolPropuesto.Concesiones[0].Finalidades[0] == "alterada" {
				t.Fatal("plan comparte concesiones mutables con entrada")
			}
		})
	}
}

func TestGobiernoPerfilRetiradaConservaDocumentoYControlAnterior(t *testing.T) {
	c, s, instante := retiradaGobiernoPerfilPrueba(t)
	plan, err := PrepararPlanGobiernoPerfil(c, s, instante)
	if err != nil || plan.Base == nil || plan.DefinicionNueva != nil || plan.Base.ControlVigencia.Revision != s.Deshabilitacion.ControlRevision {
		t.Fatalf("retirada: %v", err)
	}
	h, err := plan.Base.Rol.HuellaSHA256()
	if err != nil || h != s.Deshabilitacion.VersionRolHuellaSHA256 || c.Perfiles[0].ControlVigencia.Estado != EstadoControlVigenciaVersionRolHabilitada {
		t.Fatal("preparación reescribe la historia o retira sin autoridad")
	}
}

func TestGobiernoPerfilRetiradaRechazaFijosYPreimagenAjena(t *testing.T) {
	for _, caso := range []string{"fijo", "control_revision", "control_huella", "rol_huella", "rol_ajeno", "catalogo", "operaciones_cruzadas", "motivo"} {
		t.Run(caso, func(t *testing.T) {
			c, s, instante := retiradaGobiernoPerfilPrueba(t)
			switch caso {
			case "fijo":
				c.Perfiles[0].TipoPerfil = TipoPerfilAdministracionFijoSistemaV1
				h, err := c.HuellaSHA256()
				if err != nil {
					t.Fatal(err)
				}
				s.Deshabilitacion.CatalogoHuellaSHA256 = h
			case "control_revision":
				s.Deshabilitacion.ControlRevision++
			case "control_huella":
				s.Deshabilitacion.ControlHuellaSHA256 = strings.Repeat("e", 64)
			case "rol_huella":
				s.Deshabilitacion.VersionRolHuellaSHA256 = strings.Repeat("e", 64)
			case "rol_ajeno":
				s.Deshabilitacion.VersionRolRef = "rol:otro:v1"
			case "catalogo":
				s.Deshabilitacion.CatalogoVersion++
			case "operaciones_cruzadas":
				s.Publicacion = &PropuestaPerfilAdministracionV1{}
			case "motivo":
				s.Motivo.EntradaClave = "texto_libre"
			}
			if p, err := PrepararPlanGobiernoPerfil(c, s, instante); err == nil || p.VersionRolObjetivoRef != "" {
				t.Fatal("retirada no gobernada produce plan")
			}
		})
	}
}

func TestGobiernoPerfilVersionarConservaCASYConcesionesExactas(t *testing.T) {
	for _, caso := range []string{"crear_existente", "versionar_nuevo", "salto", "concesion", "control_retirado", "fijo"} {
		t.Run(caso, func(t *testing.T) {
			c, s, instante := gobiernoPerfilPrueba(t)
			switch caso {
			case "crear_existente":
				s.Operacion = OperacionCrearPerfilGobernado
			case "versionar_nuevo":
				s.Publicacion.RolPropuesto.RolID, s.Publicacion.RolPropuesto.Version = "nuevo", 1
				s.Publicacion.VersionRolBaseRef, s.Publicacion.VersionRolBaseHuellaSHA256 = "", ""
			case "salto":
				s.Publicacion.RolPropuesto.Version++
			case "concesion":
				s.Publicacion.RolPropuesto.Concesiones[0].Accion = "sintetico.modificar"
			case "control_retirado":
				c.Perfiles[0].ControlVigencia.Estado = EstadoControlVigenciaVersionRolRetirada
				c.Perfiles[0].ControlVigencia.ActoRef = "acto:retirada"
				c.Perfiles[0].ControlVigencia.MotivoCodigo = "retirada"
				h, err := c.HuellaSHA256()
				if err != nil {
					t.Fatal(err)
				}
				s.Publicacion.CatalogoHuellaSHA256 = h
			case "fijo":
				c.Perfiles[0].TipoPerfil = TipoPerfilAdministracionFijoSistemaV1
				h, err := c.HuellaSHA256()
				if err != nil {
					t.Fatal(err)
				}
				s.Publicacion.CatalogoHuellaSHA256 = h
			}
			if _, err := PrepararPlanGobiernoPerfil(c, s, instante); err == nil {
				t.Fatal("plan amplía permisos, reescribe base o cambia un fijo")
			}
		})
	}
}
