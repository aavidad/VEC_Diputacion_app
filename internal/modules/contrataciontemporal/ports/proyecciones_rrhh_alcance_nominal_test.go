package ports_test

import (
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestNuevoContextoConsultaRRHHConAmbitoConservaCanonOrganizacion(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	autoridad := autoridadContextoPuertosRRHH(t, ahora, "a", "a")
	const organizacion = "organizacion:diputacion-granada"

	contexto, err := ports.NuevoContextoConsultaRRHH(
		autoridad, organizacion, ahora,
	)
	if err != nil {
		t.Fatalf("crear contexto compatible: %v", err)
	}
	cuadro, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	capacidad := capacidadCuadroPuertosRRHH(t, contexto, cuadro, ahora)
	if capacidad.ClaseAmbito() != ports.AmbitoOrganizacionRRHH ||
		capacidad.AmbitoRef() != organizacion ||
		capacidad.OrganizacionRef() != organizacion {
		t.Fatalf("el constructor histórico alteró su alcance: %#v", capacidad)
	}
}

func TestContextoConsultaRRHHConAmbitoPropagaAlcanceExacto(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	autoridad := autoridadContextoPuertosRRHH(t, ahora, "a", "a")
	const organizacion = "organizacion:diputacion-granada"
	casos := []struct {
		nombre string
		clase  ports.ClaseAmbitoConsultaRRHH
		ambito string
	}{
		{"centro", ports.AmbitoCentroRRHH, "centro:rrhh:001"},
		{"unidad", ports.AmbitoUnidadGestionRRHH, "unidad:gestion:001"},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			contexto, err := ports.NuevoContextoConsultaRRHHConAmbito(
				autoridad, organizacion, caso.clase, caso.ambito, ahora,
			)
			if err != nil {
				t.Fatalf("crear contexto: %v", err)
			}
			cuadro, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 10, "")
			if err != nil {
				t.Fatal(err)
			}
			capacidadCuadro := capacidadCuadroPuertosRRHH(
				t, contexto, cuadro, ahora,
			)
			detalle, err := ports.NuevaSolicitudDetalleRRHH(
				"expediente:rrhh:001", 1,
			)
			if err != nil {
				t.Fatal(err)
			}
			capacidadDetalle := capacidadDetallePuertosRRHH(
				t, contexto, detalle, ahora,
			)
			for _, capacidad := range []ports.CapacidadConsultaRRHH{
				capacidadCuadro, capacidadDetalle,
			} {
				if capacidad.ClaseAmbito() != caso.clase ||
					capacidad.AmbitoRef() != caso.ambito ||
					capacidad.OrganizacionRef() != organizacion {
					t.Fatalf("capacidad fuera de alcance: %#v", capacidad)
				}
			}
		})
	}
}

func TestNuevoContextoConsultaRRHHConAmbitoRechazaCruces(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	autoridad := autoridadContextoPuertosRRHH(t, ahora, "a", "a")
	const organizacion = "organizacion:diputacion-granada"
	casos := []struct {
		nombre string
		clase  ports.ClaseAmbitoConsultaRRHH
		ambito string
	}{
		{"clase_desconocida", "otro", "centro:rrhh:001"},
		{"referencia_vacia", ports.AmbitoCentroRRHH, ""},
		{"organizacion_cruzada", ports.AmbitoOrganizacionRRHH, "organizacion:otra"},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			contexto, err := ports.NuevoContextoConsultaRRHHConAmbito(
				autoridad, organizacion, caso.clase, caso.ambito, ahora,
			)
			if contexto != (ports.ContextoConsultaRRHH{}) ||
				!errors.Is(err, ports.ErrContextoConsultaRRHHInvalido) {
				t.Fatalf("alcance inválido aceptado: %#v, %v", contexto, err)
			}
		})
	}
}

func TestCapacidadConsultaRRHHNoCruzaContextoNiAmbito(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	const organizacion = "organizacion:diputacion-granada"
	autoridad := autoridadContextoPuertosRRHH(t, ahora, "a", "a")
	contextoA, err := ports.NuevoContextoConsultaRRHHConAmbito(
		autoridad, organizacion, ports.AmbitoCentroRRHH, "centro:rrhh:001", ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	contextoB, err := ports.NuevoContextoConsultaRRHHConAmbito(
		autoridad, organizacion, ports.AmbitoCentroRRHH, "centro:rrhh:002", ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	material, err := materialCuadroPuertosRRHH(t, contextoA, solicitud, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ports.NuevaCapacidadConsultaCuadroRRHH(
		contextoB, material, solicitud, ahora,
	); !errors.Is(err, ports.ErrCapacidadConsultaRRHHInvalida) {
		t.Fatalf("material de otro centro aceptado: %v", err)
	}
	autoridadAjena := autoridadContextoPuertosRRHH(t, ahora, "b", "b")
	actorAjeno, err := ports.NuevoContextoConsultaRRHHConAmbito(
		autoridadAjena, organizacion, ports.AmbitoCentroRRHH,
		"centro:rrhh:001", ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ports.NuevaCapacidadConsultaCuadroRRHH(
		actorAjeno, material, solicitud, ahora,
	); !errors.Is(err, ports.ErrCapacidadConsultaRRHHInvalida) {
		t.Fatalf("material de otro actor aceptado: %v", err)
	}
}

func TestPaginaCuadroRRHHRechazaResultadoDeUnidadAjena(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	contexto := contextoConsultaRRHHConAlcancePrueba(
		t, ahora, ports.AmbitoUnidadGestionRRHH, "unidad:rrhh:001",
	)
	solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	capacidad := capacidadCuadroPuertosRRHH(t, contexto, solicitud, ahora)
	orden, err := ports.NuevaOrdenConsultaCuadroRRHH(
		contexto, capacidad, solicitud, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	lectura, err := ports.NuevoReciboLecturaRRHH(
		"lectura:rrhh:alcance-cuadro", "auditoria:rrhh:alcance-cuadro",
		contexto, capacidad, "", 0, 1, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	resumen := resumenPuertosRRHH(ahora)
	resumen.UnidadRef = "unidad:rrhh:001"
	pagina := ports.PaginaCuadroRRHH{
		GeneradaEn: ahora, Expedientes: []ports.ResumenExpedienteRRHH{resumen},
		Lectura: lectura,
	}
	if err := pagina.ValidarPara(orden); err != nil {
		t.Fatalf("resultado propio rechazado: %v", err)
	}
	pagina.Expedientes[0].UnidadRef = "unidad:rrhh:otra"
	if err := pagina.ValidarPara(orden); !errors.Is(
		err, ports.ErrResultadoConsultaRRHHNoConfiable,
	) {
		t.Fatalf("resultado de unidad ajena aceptado: %v", err)
	}
}

func TestDetalleExpedienteRRHHRechazaResultadoDeUnidadAjena(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	const unidadPropia = "unidad:rrhh:minimizada"
	contexto := contextoConsultaRRHHConAlcancePrueba(
		t, ahora, ports.AmbitoUnidadGestionRRHH, unidadPropia,
	)
	datos := datosDetalleMinimizadoPrueba(3)
	entrada := construirEntradaDetalleMinimizadaPrueba(t, datos)
	solicitud, err := ports.NuevaSolicitudDetalleRRHH(
		datos.resumen.ExpedienteRef, datos.resumen.Version,
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad := capacidadDetallePuertosRRHH(t, contexto, solicitud, ahora)
	orden, err := ports.NuevaOrdenConsultaDetalleRRHH(
		contexto, capacidad, solicitud, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	lectura, err := ports.NuevoReciboLecturaRRHH(
		"lectura:rrhh:alcance-detalle", "auditoria:rrhh:alcance-detalle",
		contexto, capacidad, datos.resumen.ExpedienteRef, datos.resumen.Version,
		1, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	detalle, err := ports.NuevoDetalleExpedienteRRHHMinimizado(entrada, lectura)
	if err != nil {
		t.Fatal(err)
	}
	if err := detalle.ValidarPara(orden); err != nil {
		t.Fatalf("detalle propio rechazado: %v", err)
	}
	datosAjeno := datosDetalleMinimizadoPrueba(3)
	datosAjeno.resumen.UnidadRef = "unidad:rrhh:otra"
	datosAjeno.asignacion.UnidadRef = "unidad:rrhh:otra"
	entradaAjena := construirEntradaDetalleMinimizadaPrueba(t, datosAjeno)
	detalleAjeno, err := ports.NuevoDetalleExpedienteRRHHMinimizado(
		entradaAjena, lectura,
	)
	if err != nil {
		t.Fatalf("construir detalle ajeno coherente: %v", err)
	}
	if err := detalleAjeno.ValidarPara(orden); !errors.Is(
		err, ports.ErrResultadoConsultaRRHHNoConfiable,
	) {
		t.Fatalf("detalle de unidad ajena aceptado: %v", err)
	}
}

func TestOrdenConsultaRRHHRechazaCapacidadEmitidaParaOtroAmbito(t *testing.T) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	const organizacion = "organizacion:diputacion-granada"
	autoridad := autoridadContextoPuertosRRHH(t, ahora, "a", "a")
	contextoUnidad, err := ports.NuevoContextoConsultaRRHHConAmbito(
		autoridad, organizacion, ports.AmbitoUnidadGestionRRHH,
		"unidad:rrhh:001", ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	capacidad := capacidadCuadroPuertosRRHH(t, contextoUnidad, solicitud, ahora)
	casos := []struct {
		nombre string
		clase  ports.ClaseAmbitoConsultaRRHH
		ambito string
	}{
		{"otra_unidad", ports.AmbitoUnidadGestionRRHH, "unidad:rrhh:002"},
		{"otra_clase_misma_referencia", ports.AmbitoCentroRRHH, "unidad:rrhh:001"},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			contexto, err := ports.NuevoContextoConsultaRRHHConAmbito(
				autoridad, organizacion, caso.clase, caso.ambito, ahora,
			)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ports.NuevaOrdenConsultaCuadroRRHH(
				contexto, capacidad, solicitud, ahora,
			); !errors.Is(err, ports.ErrOrdenConsultaRRHHInvalida) {
				t.Fatalf("capacidad de otro ámbito aceptada: %v", err)
			}
		})
	}
}

func contextoConsultaRRHHConAlcancePrueba(
	t *testing.T,
	ahora time.Time,
	clase ports.ClaseAmbitoConsultaRRHH,
	ambito string,
) ports.ContextoConsultaRRHH {
	t.Helper()
	contexto, err := ports.NuevoContextoConsultaRRHHConAmbito(
		autoridadContextoPuertosRRHH(t, ahora, "a", "a"),
		"organizacion:diputacion-granada", clase, ambito, ahora,
	)
	if err != nil {
		t.Fatalf("crear contexto con alcance: %v", err)
	}
	return contexto
}
