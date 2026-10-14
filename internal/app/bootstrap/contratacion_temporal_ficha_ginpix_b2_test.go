package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	pp "vec-diputacion-granada/internal/modules/personal/ports"
)

func TestFichaGINPIXB2ExigeAltaConfirmadaYRevalidaAntesDeEntregar(t *testing.T) {
	fachada, origen, _, _ := escenarioConfirmadaB2Prueba(t)
	origen.contrato.SolicitudPersonalRef = "solicitud:personal:original"
	origen.contrato.IdempotenciaPersonalUUID = "44444444-4444-4444-8444-444444444444"
	origen.contrato.PlanReciboRef = "recibo:plan:b2:original"
	fachada.fuentes.organismoRef = "organismo:dipgra"
	h := origen.origen.Confirmacion.Hechos
	datos := datosPlanB2AmbitosPrueba(fachada.fuentes.organismoRef)
	datos.ExpedienteRef, datos.OrigenCTRef = origen.contrato.Material.ExpedienteRef, origen.contrato.PlanRef
	datos.OrigenCTReciboRef, datos.OrigenCTHuellaSHA256 = origen.contrato.PlanReciboRef, origen.contrato.PlanSHA256
	plan := personal.PlanIncorporacionCT{ClasesOcupacionCatalogoRef: "personal:clases_ocupacion_ct",
		ClasesOcupacionCatalogoVersion: 1, ClasesOcupacionCatalogoHuellaSHA256: strings.Repeat("f", 64),
		PlanRef: h.PersonalPlanRef, ReciboRef: h.PersonalPlanReciboRef, Version: 1, Datos: datos,
		Modo: "alta_empleado", ClaveAltaRelacion: "10000000-0000-4000-8000-000000000005",
		ClaveOcupacion: "10000000-0000-4000-8000-000000000006", UsoRPTRef: "uso:uno",
		ReservaRPTRef: "reserva:uno", ConfirmacionRPTRef: "confirmacion:uno"}
	plan.HuellaSHA256 = plan.CalcularHuellaSHA256()
	origen.origen.Confirmacion.Hechos.PersonalPlanSHA256 = plan.HuellaSHA256
	h = origen.origen.Confirmacion.Hechos
	estado := pp.EstadoPlanIncorporacionCT{Plan: plan, Estado: "ejecutado", EjecucionReciboRef: h.SeguimientoRef,
		ReciboAltaRelacion: &pp.ReciboActoRegistroEmpleadoB2{Tipo: "alta", ReciboRef: h.AltaReciboRef,
			EmpleadoRef: h.EmpleadoRef, RelacionRef: h.RelacionRef, Version: 1},
		ReciboOcupacion: &pp.ReciboActoRegistroEmpleadoB2{Tipo: "ocupacion", ReciboRef: h.OcupacionReciboRef,
			EmpleadoRef: h.EmpleadoRef, RelacionRef: h.RelacionRef, HechoRef: h.OcupacionRef, Version: 1}}
	mapa, err := nuevoMapeoFichaGINPIXB2()
	if err != nil {
		t.Fatal(err)
	}
	consultas := 0
	x := &fichaGINPIXB2{f: fachada, m: mapa,
		consultarPlan: func(_ context.Context, ref, org string) (pp.EstadoPlanIncorporacionCT, error) {
			consultas++
			if ref != h.PersonalPlanRef || org != fachada.fuentes.organismoRef {
				t.Fatal("plan propietario ajeno")
			}
			return estado, nil
		}}
	exp := origen.contrato.Material.ExpedienteRef
	origen.encontrado = false
	if _, err := x.Preparar(context.Background(), exp); !errors.Is(err, ct.ErrFichaGINPIXV2Conflicto) || consultas != 0 {
		t.Fatalf("sin recibo B2 confirmado: %v; consultas=%d", err, consultas)
	}
	origen.encontrado = true
	x.consultarPlan = func(context.Context, string, string) (pp.EstadoPlanIncorporacionCT, error) {
		return pp.EstadoPlanIncorporacionCT{}, ct.ErrDenegadaIncorporacionAplicacion
	}
	if _, err := x.Preparar(context.Background(), exp); !errors.Is(err, ct.ErrFichaGINPIXV2Denegada) {
		t.Fatalf("denegación propietaria Personal: %v", err)
	}
	x.consultarPlan = func(_ context.Context, _, _ string) (pp.EstadoPlanIncorporacionCT, error) {
		consultas++
		return estado, nil
	}
	estado.ReciboAltaRelacion.ReciboRef = "recibo:otro"
	if _, err := x.Preparar(context.Background(), exp); !errors.Is(err, ct.ErrFichaGINPIXV2Conflicto) {
		t.Fatalf("recibo Personal ajeno: %v", err)
	}
	estado.ReciboAltaRelacion.ReciboRef = h.AltaReciboRef
	x.consultarPlan = func(_ context.Context, _, _ string) (pp.EstadoPlanIncorporacionCT, error) {
		origen.origen.ReciboRef = "recibo:incorporacion-personal-b2:sustituido"
		return estado, nil
	}
	if _, err := x.Preparar(context.Background(), exp); !errors.Is(err, ct.ErrFichaGINPIXV2Conflicto) {
		t.Fatalf("cambio entre verificación y descarga: %v", err)
	}
	origen.origen.ReciboRef = "recibo:incorporacion-personal-b2:prueba"
	x.consultarPlan = func(_ context.Context, _, _ string) (pp.EstadoPlanIncorporacionCT, error) {
		consultas++
		return estado, nil
	}
	out, err := x.Preparar(context.Background(), exp)
	if err != nil || consultas != 2 {
		t.Fatalf("alta B2 histórica confirmada: %v; consultas=%d", err, consultas)
	}
	meta, err := out.Metadatos()
	if err != nil || meta.IncorporacionRef != origen.origen.ReciboRef || meta.CorrelacionRef != origen.contrato.SolicitudPersonalRef || meta.IdempotenciaRef != origen.contrato.IdempotenciaPersonalUUID {
		t.Fatalf("coordenadas B2 originales: %+v; %v", meta, err)
	}
	contenido, err := out.Contenido()
	if err != nil || !strings.Contains(string(contenido), origen.origen.Confirmacion.Hechos.AltaReciboRef) {
		t.Fatalf("ficha sin recibo de alta Personal: %v", err)
	}
	if meta.VersionExpediente != 9 || strings.Contains(meta.MapeoRef, "ejercicio") {
		t.Fatalf("versión posterior o mapeo B2: %+v", meta)
	}
}

func TestFichaGINPIXB2SoloPermiteLecturasNominalesEnGET(t *testing.T) {
	ctx := context.WithValue(context.Background(), claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: http.MethodGet, ruta: httpct.RutaFichaGINPIXV2})
	for _, accion := range []string{ct.AccionLeerPlanNominalB2, ct.AccionConsultarDetalleRRHH, "personal.plan_incorporacion_ct.consultar"} {
		if !operacionPermitidaEnRutaIncorporacionB2(ctx, accion) {
			t.Fatalf("lectura requerida denegada: %s", accion)
		}
	}
	if operacionPermitidaEnRutaIncorporacionB2(ctx, ct.AccionConfirmarOrigenB2) ||
		operacionPermitidaEnRutaIncorporacionB2(ctx, "personal.registro_empleado.hecho.registrar") ||
		operacionPermitidaEnRutaIncorporacionB2(context.WithValue(context.Background(), claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: http.MethodPost, ruta: httpct.RutaFichaGINPIXV2}), ct.AccionLeerPlanNominalB2) {
		t.Fatal("la ficha permitió una escritura o un método ajeno")
	}
}
