package domain

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func escenarioRolDietas(t *testing.T, rolID string) (CatalogoAccionesAdministracionV1, SolicitudPlanGobiernoPerfil, time.Time) {
	t.Helper()
	c, s, ahora := escenarioRolNuevoOrdinario(t)
	var concesiones []ConcesionRol
	var dimensiones []string
	if rolID == RolIDPersonaDietasComisiones {
		dimensiones = []string{"empleado_ref", "persona_ref"}
		concesiones = []ConcesionRol{
			{Accion: accionesRolDietasComisiones[0], ModuloID: "personal", TipoRecurso: "relacion_empleado_dietas",
				Finalidades: []string{"preparar_borrador_dietas"}, GarantiaMinima: AuthAssuranceHigh,
				CamposPermitidos: camposDietasRelacion},
			{Accion: accionesRolDietasComisiones[1], ModuloID: "dietas", TipoRecurso: "comision_borrador",
				Finalidades: []string{"crear_borrador_propio"}, GarantiaMinima: AuthAssuranceHigh,
				CamposPermitidos: camposDietasBorrador},
			{Accion: accionesRolDietasComisiones[2], ModuloID: "dietas", TipoRecurso: "comision_borrador",
				Finalidades: []string{"consultar_borrador_propio"}, GarantiaMinima: AuthAssuranceHigh,
				CamposPermitidos: camposDietasConsulta},
		}
	} else {
		dimensiones = []string{"ambito_ref"}
		concesiones = []ConcesionRol{
			{Accion: accionesRolDietasRutas[0], ModuloID: "dietas", TipoRecurso: "catalogo_rutas_dietas",
				Finalidades: []string{"consultar_itinerario_dietas"}, GarantiaMinima: AuthAssuranceHigh},
			{Accion: accionesRolDietasRutas[1], ModuloID: "dietas", TipoRecurso: "calculo_rutas_dietas",
				Finalidades: []string{"consultar_itinerario_dietas"}, GarantiaMinima: AuthAssuranceHigh},
		}
	}
	entradaBase := c.Entradas[0]
	c.Entradas = make([]EntradaAccionAdministracionV1, len(concesiones))
	s.Publicacion.RolPropuesto.RolID = rolID
	s.Publicacion.RolPropuesto.Concesiones = append([]ConcesionRol(nil), concesiones...)
	s.Publicacion.Selecciones = make([]SeleccionAccionAdministracionV1, len(concesiones))
	for i, concesion := range concesiones {
		e := entradaBase
		e.Referencia = "entrada:dietas:" + concesion.Accion
		e.Concesion = concesion
		e.DimensionesAmbito = append([]string(nil), dimensiones...)
		c.Entradas[i] = e
	}
	actualizarRolDietas(t, &c, &s)
	return c, s, ahora
}

func actualizarRolDietas(t *testing.T, c *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
	t.Helper()
	for i := range s.Publicacion.Selecciones {
		e := c.Entradas[i]
		h, err := e.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		s.Publicacion.Selecciones[i] = SeleccionAccionAdministracionV1{
			EntradaRef: e.Referencia, EntradaVersion: e.Version, EntradaHuellaSHA256: h}
	}
	h, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	s.Publicacion.CatalogoHuellaSHA256 = h
}

func TestGobiernoRolDietasContratosNominalesYCierre(t *testing.T) {
	for _, rolID := range []string{RolIDPersonaDietasComisiones, RolIDPersonaDietasRutas} {
		t.Run(rolID, func(t *testing.T) {
			c, s, ahora := escenarioRolDietas(t, rolID)
			plan, instante, err := PrepararPlanGobiernoRolNuevoDesdeCatalogo(c, s, ahora,
				"per_"+strings.Repeat("a", 24))
			if err != nil || !instante.Equal(ahora) || !planRolDietasExacto(plan) ||
				!cierreGobiernoRolNuevoAdmitido(plan) ||
				!reflect.DeepEqual(plan.DefinicionNueva.Concesiones, s.Publicacion.RolPropuesto.Concesiones) {
				t.Fatalf("contrato %s: plan=%+v instante=%v error=%v", rolID, plan, instante, err)
			}
			otro := plan.Copia()
			otro.DefinicionNueva.RolID = "otro_rol"
			if cierreGobiernoRolNuevoAdmitido(otro) {
				t.Fatal("cierre aceptó otro rol compuesto")
			}
			otro = plan.Copia()
			otro.DefinicionNueva.Concesiones[0].CamposPermitidos = []string{"campo_ajeno"}
			if cierreGobiernoRolNuevoAdmitido(otro) {
				t.Fatal("cierre aceptó concesión ampliada")
			}
			otro = plan.Copia()
			otro.Selecciones[1] = otro.Selecciones[0]
			if cierreGobiernoRolNuevoAdmitido(otro) {
				t.Fatal("cierre aceptó selección repetida")
			}
		})
	}
}

func TestGobiernoRolDietasRutasAdmiteListasVaciasOmitidas(t *testing.T) {
	c, s, ahora := escenarioRolDietas(t, RolIDPersonaDietasRutas)
	s.Publicacion.RolPropuesto.Concesiones[0].CamposPermitidos = []string{}
	s.Publicacion.RolPropuesto.Concesiones[0].Obligaciones = []string{}
	plan, _, err := PrepararPlanGobiernoRolNuevoDesdeCatalogo(c, s, ahora,
		"per_"+strings.Repeat("a", 24))
	if err != nil || !planRolDietasExacto(plan) {
		t.Fatalf("lista vacía equivalente rechazada: %+v, error %v", plan, err)
	}
}

func TestGobiernoRolDietasRechazaContratosAjenos(t *testing.T) {
	mutaciones := []struct {
		nombre string
		mutar  func(*CatalogoAccionesAdministracionV1, *SolicitudPlanGobiernoPerfil)
	}{
		{"accion", func(c *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
			c.Entradas[0].Concesion.Accion = "personal.relacion.ajena.consultar_dietas"
			s.Publicacion.RolPropuesto.Concesiones[0] = c.Entradas[0].Concesion
		}},
		{"campo", func(c *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
			c.Entradas[0].Concesion.CamposPermitidos = append(append([]string(nil), camposDietasRelacion...), "dni")
			s.Publicacion.RolPropuesto.Concesiones[0] = c.Entradas[0].Concesion
		}},
		{"obligacion", func(c *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
			c.Entradas[0].Concesion.Obligaciones = []string{"auditar"}
			s.Publicacion.RolPropuesto.Concesiones[0] = c.Entradas[0].Concesion
		}},
		{"ambito", func(c *CatalogoAccionesAdministracionV1, _ *SolicitudPlanGobiernoPerfil) {
			c.Entradas[0].DimensionesAmbito = []string{"persona_ref"}
		}},
		{"orden", func(_ *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
			s.Publicacion.Selecciones[0], s.Publicacion.Selecciones[1] = s.Publicacion.Selecciones[1], s.Publicacion.Selecciones[0]
		}},
		{"duplicado", func(_ *CatalogoAccionesAdministracionV1, s *SolicitudPlanGobiernoPerfil) {
			s.Publicacion.Selecciones[1] = s.Publicacion.Selecciones[0]
		}},
	}
	for _, caso := range mutaciones {
		t.Run(caso.nombre, func(t *testing.T) {
			c, s, ahora := escenarioRolDietas(t, RolIDPersonaDietasComisiones)
			caso.mutar(&c, &s)
			if caso.nombre != "orden" && caso.nombre != "duplicado" {
				actualizarRolDietas(t, &c, &s)
			}
			plan, _, err := PrepararPlanGobiernoRolNuevoDesdeCatalogo(c, s, ahora,
				"per_"+strings.Repeat("a", 24))
			if err == nil || plan.DefinicionNueva != nil {
				t.Fatalf("contrato ajeno admitido: %+v, error %v", plan, err)
			}
		})
	}
	for _, caso := range []string{"rol", "campo", "obligacion", "ambito", "orden"} {
		t.Run("rutas_"+caso, func(t *testing.T) {
			c, s, ahora := escenarioRolDietas(t, RolIDPersonaDietasRutas)
			switch caso {
			case "rol":
				s.Publicacion.RolPropuesto.RolID = "otro_rol"
			case "campo":
				c.Entradas[0].Concesion.CamposPermitidos = []string{"distancia"}
				s.Publicacion.RolPropuesto.Concesiones[0] = c.Entradas[0].Concesion
			case "obligacion":
				c.Entradas[0].Concesion.Obligaciones = []string{"auditar"}
				s.Publicacion.RolPropuesto.Concesiones[0] = c.Entradas[0].Concesion
			case "ambito":
				c.Entradas[0].DimensionesAmbito = []string{"persona_ref"}
			case "orden":
				s.Publicacion.Selecciones[0], s.Publicacion.Selecciones[1] = s.Publicacion.Selecciones[1], s.Publicacion.Selecciones[0]
			}
			if caso != "orden" {
				actualizarRolDietas(t, &c, &s)
			}
			plan, _, err := PrepararPlanGobiernoRolNuevoDesdeCatalogo(c, s, ahora,
				"per_"+strings.Repeat("a", 24))
			if err == nil || plan.DefinicionNueva != nil {
				t.Fatalf("rutas ajenas admitidas: %+v, error %v", plan, err)
			}
		})
	}
}

func TestGobiernoRolDietasReplaySoloConIntervaloComun(t *testing.T) {
	c, s, ahora := escenarioRolDietas(t, RolIDPersonaDietasComisiones)
	c.VigenteHasta = ahora.Add(time.Hour)
	for i := range c.Entradas {
		c.Entradas[i].VigenteHasta = c.VigenteHasta
	}
	actualizarRolDietas(t, &c, &s)
	plan, instante, err := PrepararPlanGobiernoRolNuevoDesdeCatalogo(c, s,
		ahora.Add(2*time.Hour), "per_"+strings.Repeat("a", 24))
	if err != nil || !planRolDietasExacto(plan) || !instante.Equal(c.VigenteDesde) {
		t.Fatalf("replay histórico: plan=%+v instante=%v error=%v", plan, instante, err)
	}
	// Las entradas son válidas individualmente, pero no comparten instante.
	finPrimera := c.Entradas[0].VigenteHasta
	c.Entradas[1].VigenteDesde = finPrimera
	c.Entradas[1].VigenteHasta = c.Entradas[1].VigenteDesde.Add(time.Hour)
	c.VigenteHasta = c.Entradas[1].VigenteHasta
	c.Entradas[2].VigenteHasta = c.VigenteHasta
	actualizarRolDietas(t, &c, &s)
	plan, _, err = PrepararPlanGobiernoRolNuevoDesdeCatalogo(c, s,
		c.VigenteHasta.Add(time.Hour), "per_"+strings.Repeat("a", 24))
	if err == nil || plan.DefinicionNueva != nil {
		t.Fatalf("intervalos sin intersección admitidos: %+v, error %v", plan, err)
	}
}
