package domain

import (
	"strings"
	"testing"
	"time"
)

func escenarioRolNuevoOrdinario(t *testing.T) (CatalogoAccionesAdministracionV1, SolicitudPlanGobiernoPerfil, time.Time) {
	t.Helper()
	c, s, instante := gobiernoPerfilPrueba(t)
	s.Operacion = OperacionCrearPerfilGobernado
	s.Publicacion.RolPropuesto.RolID = "nuevo_ordinario_sintetico"
	s.Publicacion.RolPropuesto.Version = 1
	s.Publicacion.VersionRolBaseRef = ""
	s.Publicacion.VersionRolBaseHuellaSHA256 = ""
	c.Entradas[0].ClaseControl = string(ClaseControlPerfilOrdinario)
	return c, s, instante
}

func actualizarSeleccionRolNuevo(t *testing.T, c CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
	t.Helper()
	s.Publicacion.RolPropuesto.Concesiones[0] = c.Entradas[0].Concesion
	he, err := c.Entradas[0].HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	hc, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	s.Publicacion.Selecciones[0].EntradaHuellaSHA256 = he
	s.Publicacion.CatalogoHuellaSHA256 = hc
}

func TestGobiernoRolNuevoSoloAceptaEntradaOrdinariaDeModuloNoAdministrativo(t *testing.T) {
	c, s, instante := escenarioRolNuevoOrdinario(t)
	actualizarSeleccionRolNuevo(t, c, &s)
	plan, _, err := PrepararPlanGobiernoRolNuevoDesdeCatalogo(c, s, instante, "per_"+strings.Repeat("a", 24))
	if err != nil || plan.Operacion != OperacionCrearPerfilGobernado || plan.Base != nil ||
		plan.DefinicionNueva == nil || plan.DefinicionNueva.Version != 1 || len(plan.DefinicionNueva.Concesiones) != 1 {
		t.Fatalf("plan ordinario: %+v, error: %v", plan, err)
	}
}

func TestGobiernoRolNuevoRechazaConcesionAdministrativaYClaseNoOrdinaria(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*CatalogoAccionesAdministracionV1, *SolicitudPlanGobiernoPerfil)
	}{
		{"accion_administracion_aprobar", func(c *CatalogoAccionesAdministracionV1, _ *SolicitudPlanGobiernoPerfil) {
			c.Entradas[0].Concesion.ModuloID = "administracion"
			c.Entradas[0].Concesion.Accion = "administracion.perfiles.definicion.aprobar"
		}},
		{"administrador_aplicacion", func(c *CatalogoAccionesAdministracionV1, _ *SolicitudPlanGobiernoPerfil) {
			c.Entradas[0].ClaseControl = "administrador_aplicacion"
		}},
		{"administrador_sistemas", func(c *CatalogoAccionesAdministracionV1, _ *SolicitudPlanGobiernoPerfil) {
			c.Entradas[0].ClaseControl = "administrador_sistemas"
		}},
		{"rol_sensible", func(c *CatalogoAccionesAdministracionV1, _ *SolicitudPlanGobiernoPerfil) {
			c.Entradas[0].ClaseControl = "rol_sensible"
		}},
		{"clase_auditada_no_ordinaria", func(c *CatalogoAccionesAdministracionV1, _ *SolicitudPlanGobiernoPerfil) {
			c.Entradas[0].ClaseControl = "consulta_auditada"
		}},
		{"rol_publicado", func(c *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
			s.Publicacion.RolPropuesto.RolID = c.Perfiles[0].Rol.RolID
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			c, s, instante := escenarioRolNuevoOrdinario(t)
			caso.mutar(&c, &s)
			actualizarSeleccionRolNuevo(t, c, &s)
			plan, _, err := PrepararPlanGobiernoRolNuevoDesdeCatalogo(c, s, instante, "per_"+strings.Repeat("a", 24))
			if err == nil || plan.VersionRolObjetivoRef != "" {
				t.Fatalf("%s admitido: %+v", caso.nombre, plan)
			}
		})
	}
}
