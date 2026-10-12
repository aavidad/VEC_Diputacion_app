package domain

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func escenarioRolMeritosCartera(t *testing.T) (CatalogoAccionesAdministracionV1,
	SolicitudPlanGobiernoPerfil, time.Time) {
	t.Helper()
	c, s, instante := escenarioRolNuevoOrdinario(t)
	get := c.Entradas[0]
	get.Referencia = "entrada:meritos:cartera:consultar"
	get.DimensionesAmbito = []string{"persona_ref"}
	get.Concesion = ConcesionRol{Accion: accionRolMeritosConsultar, ModuloID: "meritos",
		TipoRecurso: "cartera_personal", Finalidades: []string{"gestion_cartera_propia"},
		GarantiaMinima:   AuthAssuranceHigh,
		CamposPermitidos: []string{"denominacion", "disponibilidad", "hecho_ref", "retirada", "tipo", "total", "version_hecho"},
		Obligaciones:     []string{"auditar"}}
	post := get
	post.Referencia = "entrada:meritos:cartera:retirar"
	post.Concesion = ConcesionRol{Accion: accionRolMeritosRetirar, ModuloID: "meritos",
		TipoRecurso: "hecho", Finalidades: []string{"gestion_cartera_propia"},
		GarantiaMinima:   AuthAssuranceHigh,
		CamposPermitidos: []string{"disponibilidad", "recibo"}, Obligaciones: []string{"auditar"}}
	c.Entradas = []EntradaAccionAdministracionV1{get, post}
	s.Publicacion.RolPropuesto.RolID = RolIDPersonaMeritosCartera
	s.Publicacion.RolPropuesto.Concesiones = []ConcesionRol{get.Concesion, post.Concesion}
	s.Publicacion.Selecciones = make([]SeleccionAccionAdministracionV1, 2)
	for i, e := range c.Entradas {
		h, err := e.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		s.Publicacion.Selecciones[i] = SeleccionAccionAdministracionV1{EntradaRef: e.Referencia,
			EntradaVersion: e.Version, EntradaHuellaSHA256: h}
	}
	h, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	s.Publicacion.CatalogoHuellaSHA256 = h
	return c, s, instante
}

func TestGobiernoRolMeritosCarteraPublicaSoloDosPermisosOrdenados(t *testing.T) {
	c, s, ahora := escenarioRolMeritosCartera(t)
	plan, _, err := PrepararPlanGobiernoRolNuevoDesdeCatalogo(c, s, ahora,
		"per_"+strings.Repeat("a", 24))
	if err != nil || plan.Base != nil || plan.DefinicionNueva == nil ||
		plan.DefinicionNueva.RolID != RolIDPersonaMeritosCartera ||
		!reflect.DeepEqual(plan.DefinicionNueva.Concesiones, s.Publicacion.RolPropuesto.Concesiones) ||
		len(plan.Selecciones) != 2 {
		t.Fatalf("rol M4 inexacto: plan=%+v err=%v", plan, err)
	}
	casos := []struct {
		nombre string
		mutar  func(*CatalogoAccionesAdministracionV1, *SolicitudPlanGobiernoPerfil)
	}{
		{"rol ajeno con dos", func(_ *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
			s.Publicacion.RolPropuesto.RolID = "otro_rol"
		}},
		{"orden invertido", func(_ *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
			s.Publicacion.Selecciones[0], s.Publicacion.Selecciones[1] = s.Publicacion.Selecciones[1], s.Publicacion.Selecciones[0]
		}},
		{"campo privado extra", func(c *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
			c.Entradas[0].Concesion.CamposPermitidos = append(c.Entradas[0].Concesion.CamposPermitidos, "evidencias")
			s.Publicacion.RolPropuesto.Concesiones[0] = c.Entradas[0].Concesion
			h, _ := c.Entradas[0].HuellaSHA256()
			s.Publicacion.Selecciones[0].EntradaHuellaSHA256 = h
			hc, _ := c.HuellaSHA256()
			s.Publicacion.CatalogoHuellaSHA256 = hc
		}},
		{"ámbito cambiado", func(c *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
			c.Entradas[1].DimensionesAmbito = []string{"candidato_ref"}
			h, _ := c.Entradas[1].HuellaSHA256()
			s.Publicacion.Selecciones[1].EntradaHuellaSHA256 = h
			hc, _ := c.HuellaSHA256()
			s.Publicacion.CatalogoHuellaSHA256 = hc
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			actual, intencion, instante := escenarioRolMeritosCartera(t)
			caso.mutar(&actual, &intencion)
			plan, _, err := PrepararPlanGobiernoRolNuevoDesdeCatalogo(actual, intencion,
				instante, "per_"+strings.Repeat("a", 24))
			if err == nil || plan.DefinicionNueva != nil {
				t.Fatalf("plan ampliado indebidamente: %+v err=%v", plan, err)
			}
		})
	}
}
