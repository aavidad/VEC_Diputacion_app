package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type autorizadorListaB2Prueba struct {
	operacion string
	llamadas  int
}

func (a *autorizadorListaB2Prueba) AutorizarConsultaRegistroEmpleadoB2(_ context.Context, m domain.MaterialConsultaRegistroEmpleadoB2) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	a.operacion = m.Operacion()
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, domain.ErrRegistroEmpleadoB2Denegado
}

type repositorioListaB2Prueba struct{ llamadas int }

func (r *repositorioListaB2Prueba) ListarEmpleadosRRHH(context.Context, ports.OrdenEmpleadosB2) (ports.ResultadoEmpleadosB2, error) {
	r.llamadas++
	return ports.ResultadoEmpleadosB2{}, nil
}

func TestListaEmpleadosB2MaterialLigadoAlOrganismo(t *testing.T) {
	actor := solicitudP(t).Actor
	fecha, _ := domain.NuevaFechaCivil("2026-09-20")
	s := domain.SolicitudEmpleadosB2{OrganismoRef: "organismo:dipgra", Corte: domain.CorteEmpleadoB2{VigenteEn: fecha, ConocidoEn: time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)}, Limite: 25, Actor: actor}
	m, err := domain.NuevoMaterialEmpleadosB2(s)
	if err != nil || m.Operacion() != "empleados" || m.Recurso().Tipo != "empleados_rrhh" || m.Recurso().Referencia != s.OrganismoRef || m.EmpleadoRef() != "" {
		t.Fatal("material de lista incorrecto", err)
	}
	for _, mal := range []func(*domain.SolicitudEmpleadosB2){
		func(x *domain.SolicitudEmpleadosB2) { x.OrganismoRef = "" },
		func(x *domain.SolicitudEmpleadosB2) { x.Limite = 0 },
		func(x *domain.SolicitudEmpleadosB2) { x.Limite = domain.LimiteEmpleadosB2 + 1 },
		func(x *domain.SolicitudEmpleadosB2) { x.Cursor = "p;1" },
	} {
		copia := s
		mal(&copia)
		if _, err := domain.NuevoMaterialEmpleadosB2(copia); !errors.Is(err, domain.ErrRegistroEmpleadoB2Invalido) {
			t.Fatal("solicitud inválida aceptada")
		}
	}
	vacia := domain.PaginaEmpleadosB2{OrganismoRef: s.OrganismoRef, Corte: s.Corte, Limite: 25, Empleados: []domain.EmpleadoOrganismoB2{}}
	if err := vacia.ValidarPara(m); err != nil {
		t.Fatal("página vacía rechazada", err)
	}
	repetida := vacia
	repetida.Empleados = []domain.EmpleadoOrganismoB2{{EmpleadoRef: "emp_" + strings.Repeat("a", 24), Relaciones: []domain.RelacionVigenteEmpleadoB2{}}, {EmpleadoRef: "emp_" + strings.Repeat("a", 24), Relaciones: []domain.RelacionVigenteEmpleadoB2{}}}
	if repetida.ValidarPara(m) == nil {
		t.Fatal("empleado repetido aceptado")
	}
	futura := vacia
	manana, _ := domain.NuevaFechaCivil("2026-09-21")
	futura.Empleados = []domain.EmpleadoOrganismoB2{{EmpleadoRef: "emp_" + strings.Repeat("a", 24), Relaciones: []domain.RelacionVigenteEmpleadoB2{{
		RelacionRef: "rel_" + strings.Repeat("a", 24), Estado: "vigente", UnidadRef: "unidad:uno",
		Traza: domain.TrazaEmpleadoB2{Desde: manana, RegistradaEn: s.Corte.ConocidoEn, Version: 1, ActoRef: "acto:uno", FuenteRef: "fuente:rrhh", FuenteVersion: 1},
	}}}}
	if futura.ValidarPara(m) == nil {
		t.Fatal("relación aún no vigente aceptada como vigente")
	}
}

func TestListaEmpleadosB2DenegacionNoLlegaAlRepositorio(t *testing.T) {
	a, r := &autorizadorListaB2Prueba{}, &repositorioListaB2Prueba{}
	s, err := NuevoServicioEmpleadosRegistroB2(a, r)
	if err != nil {
		t.Fatal(err)
	}
	fecha, _ := domain.NuevaFechaCivil("2026-09-20")
	_, err = s.ConsultarEmpleados(context.Background(), domain.SolicitudEmpleadosB2{OrganismoRef: "organismo:dipgra", Corte: domain.CorteEmpleadoB2{VigenteEn: fecha, ConocidoEn: time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)}, Limite: 25, Actor: solicitudP(t).Actor})
	if !errors.Is(err, domain.ErrRegistroEmpleadoB2Denegado) || a.operacion != "empleados" || r.llamadas != 0 {
		t.Fatal("denegación V3 no detuvo la lista", err)
	}
	if _, err := NuevoServicioEmpleadosRegistroB2(nil, r); !errors.Is(err, domain.ErrRegistroEmpleadoB2NoDisponible) {
		t.Fatal("servicio sin autorizador")
	}
}
