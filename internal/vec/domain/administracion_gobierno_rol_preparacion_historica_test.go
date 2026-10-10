package domain

import (
	"errors"
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
		{"accion_administracion_con_modulo_ajeno", func(c *CatalogoAccionesAdministracionV1, _ *SolicitudPlanGobiernoPerfil) {
			c.Entradas[0].Concesion.Accion = "administracion.perfiles.definicion.aprobar"
		}},
		{"modulo_intervencion", func(c *CatalogoAccionesAdministracionV1, _ *SolicitudPlanGobiernoPerfil) {
			c.Entradas[0].Concesion.ModuloID = "intervencion"
		}},
		{"modulo_aspirantes", func(c *CatalogoAccionesAdministracionV1, _ *SolicitudPlanGobiernoPerfil) {
			c.Entradas[0].Concesion.ModuloID = "aspirantes"
		}},
		{"accion_fiscalizacion", func(c *CatalogoAccionesAdministracionV1, _ *SolicitudPlanGobiernoPerfil) {
			c.Entradas[0].Concesion.Accion = "sintetico.fiscalizacion.consultar"
		}},
		{"finalidad_fiscaliz", func(c *CatalogoAccionesAdministracionV1, _ *SolicitudPlanGobiernoPerfil) {
			c.Entradas[0].Concesion.Finalidades = []string{"revision_fiscalizadora"}
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
		{"rol_administracion_perfiles", func(_ *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
			s.Publicacion.RolPropuesto.RolID = "administracion_perfiles"
		}},
		{"rol_operador_plataforma", func(_ *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
			s.Publicacion.RolPropuesto.RolID = "operador_plataforma"
		}},
		{"rol_candidato", func(_ *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
			s.Publicacion.RolPropuesto.RolID = "candidato_sintetico"
		}},
		{"rol_externo", func(_ *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
			s.Publicacion.RolPropuesto.RolID = "rol_externo_sintetico"
		}},
		{"rol_intervencion", func(_ *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
			s.Publicacion.RolPropuesto.RolID = "intervencion_sintetica"
		}},
		{"nombre_fiscalizacion", func(_ *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
			s.Publicacion.RolPropuesto.Nombre = "Fiscalización sintética"
		}},
		{"nombre_intervencion", func(_ *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
			s.Publicacion.RolPropuesto.Nombre = "Intervención sintética"
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
			if caso.nombre != "rol_publicado" && !errors.Is(err, ErrPermisoPerfilAdministracionNoCoincide) {
				t.Fatalf("%s: fallo ajeno a la exclusión ordinaria: %v", caso.nombre, err)
			}
		})
	}
}
