package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestConsultaCuadroRRHHRechazaFilaDeOtraOrganizacion(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	entorno.sesion.pagina.Expedientes[0].OrganizacionRef = "organizacion:ajena"
	servicio := servicioCuadroRRHHPrueba(t, entorno)
	if _, err := servicio.Consultar(
		context.Background(), entorno.cuadro,
	); !errors.Is(err, ErrResultadoConsultaRRHHNoConfiable) {
		t.Fatalf("fila cross-tenant aceptada: %v", err)
	}
}

func TestConsultaCuadroRRHHRespetaAmbitoCentroYUnidad(t *testing.T) {
	t.Parallel()
	for _, caso := range []struct {
		nombre     string
		clase      ports.ClaseAmbitoConsultaRRHH
		ambitoRef  string
		preparar   func(*entornoConsultaRRHH)
		desalinear func(*entornoConsultaRRHH)
	}{
		{
			nombre: "centro", clase: ports.AmbitoCentroRRHH,
			ambitoRef: "centro:rrhh:001",
			preparar:  func(*entornoConsultaRRHH) {},
			desalinear: func(e *entornoConsultaRRHH) {
				e.sesion.pagina.Expedientes[0].CentroRef = "centro:rrhh:ajeno"
			},
		},
		{
			nombre: "unidad", clase: ports.AmbitoUnidadGestionRRHH,
			ambitoRef: "unidad:rrhh:001",
			preparar: func(e *entornoConsultaRRHH) {
				e.sesion.pagina.Expedientes[0].UnidadRef = "unidad:rrhh:001"
			},
			desalinear: func(e *entornoConsultaRRHH) {
				e.sesion.pagina.Expedientes[0].UnidadRef = "unidad:rrhh:ajena"
			},
		},
	} {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			caso.preparar(entorno)
			configurarAmbitoCuadroRRHH(
				t, entorno, caso.clase, caso.ambitoRef,
				uint16(len(entorno.sesion.pagina.Expedientes)),
			)
			if _, err := servicioCuadroRRHHPrueba(t, entorno).Consultar(
				context.Background(), entorno.cuadro,
			); err != nil {
				t.Fatalf("ámbito exacto rechazado: %v", err)
			}
			entorno = nuevoEntornoConsultaRRHH(t)
			caso.preparar(entorno)
			caso.desalinear(entorno)
			configurarAmbitoCuadroRRHH(
				t, entorno, caso.clase, caso.ambitoRef,
				uint16(len(entorno.sesion.pagina.Expedientes)),
			)
			if _, err := servicioCuadroRRHHPrueba(t, entorno).Consultar(
				context.Background(), entorno.cuadro,
			); !errors.Is(err, ErrResultadoConsultaRRHHNoConfiable) {
				t.Fatalf("fila fuera de ámbito aceptada: %v", err)
			}
		})
	}
}

func TestConsultaDetalleRRHHComparaVersionObservada(t *testing.T) {
	t.Parallel()
	for _, caso := range []struct {
		nombre  string
		version uint64
		acepta  bool
	}{
		{"primera_carga", 0, true},
		{"coincidente", 1, true},
		{"obsoleta", 2, false},
	} {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			solicitud, err := ports.NuevaSolicitudDetalleRRHH(
				entorno.detalle.ExpedienteRef(), caso.version,
			)
			if err != nil {
				t.Fatal(err)
			}
			entorno.detalle = solicitud
			configurarAmbitoDetalleRRHH(
				t,
				entorno,
				ports.AmbitoOrganizacionRRHH,
				entorno.contexto.OrganizacionRef(),
			)
			_, err = servicioDetalleRRHHPrueba(t, entorno).Consultar(
				context.Background(), solicitud,
			)
			if caso.acepta && err != nil {
				t.Fatalf("versión válida rechazada: %v", err)
			}
			if !caso.acepta &&
				!errors.Is(err, ErrResultadoConsultaRRHHNoConfiable) {
				t.Fatalf("versión obsoleta aceptada: %v", err)
			}
		})
	}
}

func TestConsultaDetalleRRHHRespetaAmbitoCentro(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	configurarAmbitoDetalleRRHH(
		t, entorno, ports.AmbitoCentroRRHH, "centro:rrhh:001",
	)
	if _, err := servicioDetalleRRHHPrueba(t, entorno).Consultar(
		context.Background(), entorno.detalle,
	); err != nil {
		t.Fatalf("detalle en centro autorizado rechazado: %v", err)
	}
	entorno = nuevoEntornoConsultaRRHH(t)
	entorno.sesion.detalle.Resumen.CentroRef = "centro:rrhh:ajeno"
	configurarAmbitoDetalleRRHH(
		t, entorno, ports.AmbitoCentroRRHH, "centro:rrhh:001",
	)
	if _, err := servicioDetalleRRHHPrueba(t, entorno).Consultar(
		context.Background(), entorno.detalle,
	); !errors.Is(err, ErrResultadoConsultaRRHHNoConfiable) {
		t.Fatalf("detalle fuera del centro autorizado aceptado: %v", err)
	}
}

func TestConsultaCuadroRRHHValidaFiltrosYOrdenEstable(t *testing.T) {
	t.Parallel()
	preparar := func(t *testing.T) *entornoConsultaRRHH {
		t.Helper()
		entorno := nuevoEntornoConsultaRRHH(t)
		solicitud, err := ports.NuevaSolicitudCuadroRRHH(
			"2026/CT", domain.EstadoEnCurso, "solicitud", 50, "",
		)
		if err != nil {
			t.Fatal(err)
		}
		entorno.cuadro = solicitud
		primero := entorno.sesion.pagina.Expedientes[0]
		primero.ExpedienteRef = "expediente:rrhh:002"
		primero.NumeroVisible = "2026/CT-002"
		segundo := entorno.sesion.pagina.Expedientes[0]
		entorno.sesion.pagina.Expedientes =
			[]ports.ResumenExpedienteRRHH{primero, segundo}
		configurarAmbitoCuadroRRHH(
			t, entorno, ports.AmbitoOrganizacionRRHH,
			entorno.contexto.OrganizacionRef(), 2,
		)
		return entorno
	}
	entornoValido := preparar(t)
	if _, err := servicioCuadroRRHHPrueba(t, entornoValido).Consultar(
		context.Background(), entornoValido.cuadro,
	); err != nil {
		t.Fatalf("página ordenada rechazada: %v", err)
	}
	for _, caso := range []struct {
		nombre  string
		alterar func(*entornoConsultaRRHH)
	}{
		{"prefijo", func(e *entornoConsultaRRHH) {
			e.sesion.pagina.Expedientes[1].NumeroVisible = "2025/CT-001"
		}},
		{"estado", func(e *entornoConsultaRRHH) {
			e.sesion.pagina.Expedientes[1].EstadoClave = domain.EstadoCompletado
		}},
		{"fase", func(e *entornoConsultaRRHH) {
			e.sesion.pagina.Expedientes[1].FaseClave = "analisis"
		}},
		{"orden", func(e *entornoConsultaRRHH) {
			e.sesion.pagina.Expedientes[0],
				e.sesion.pagina.Expedientes[1] =
				e.sesion.pagina.Expedientes[1],
				e.sesion.pagina.Expedientes[0]
		}},
	} {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			entorno := preparar(t)
			caso.alterar(entorno)
			if _, err := servicioCuadroRRHHPrueba(t, entorno).Consultar(
				context.Background(), entorno.cuadro,
			); !errors.Is(err, ErrResultadoConsultaRRHHNoConfiable) {
				t.Fatalf("resultado que incumple %s aceptado: %v", caso.nombre, err)
			}
		})
	}
}

func TestDetalleRRHHJSONEsSuficienteSinEvidenciaInternaNiPII(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	configurarDetalleCompletoRRHHPrueba(t, entorno)
	obtenido, err := servicioDetalleRRHHPrueba(t, entorno).Consultar(
		context.Background(), entorno.detalle,
	)
	if err != nil {
		t.Fatalf("detalle operativo rechazado: %v", err)
	}
	contenido, err := json.Marshal(obtenido)
	if err != nil {
		t.Fatal(err)
	}
	jsonTexto := string(contenido)
	for _, obligatoria := range []string{
		`"grupo_subgrupo"`, `"motivo_clave"`, `"periodo_inicio"`,
		`"causa_clave"`, `"porcentaje_jornada"`, `"resultado_rc"`,
		`"coste_previsto"`, `"fuente_coste_ref"`, `"via_clave"`,
		`"procedimiento_ref"`, `"comprobaciones"`, `"asignacion"`,
		`"unidad_ref"`, `"asignada_en"`,
	} {
		if !strings.Contains(jsonTexto, obligatoria) {
			t.Fatalf("falta dato operativo %s: %s", obligatoria, jsonTexto)
		}
	}
	for _, prohibida := range []string{
		"actor_ref", "contacto", "observaciones", "responsable",
		"notificacion", "motivacion", "documentos", "lectura_ref",
		"auditoria_ref", `"lectura"`, `"registrada_en"`,
		"lectura:rrhh:001", "auditoria:rrhh:001",
	} {
		if strings.Contains(jsonTexto, prohibida) {
			t.Fatalf("se serializa material prohibido %q: %s", prohibida, jsonTexto)
		}
	}
	representacion := fmt.Sprintf("%v %#v", obtenido, obtenido)
	if strings.Contains(representacion, "lectura:rrhh:001") ||
		strings.Contains(representacion, "auditoria:rrhh:001") {
		t.Fatalf("el formateo filtra evidencia interna: %s", representacion)
	}
}

func servicioCuadroRRHHPrueba(
	t *testing.T,
	entorno *entornoConsultaRRHH,
) *ServicioConsultaCuadroRRHH {
	t.Helper()
	servicio, err := NuevoServicioConsultaCuadroRRHH(
		entorno.autoridad, entorno.autorizador, entorno.sesion, entorno.reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	return servicio
}

func servicioDetalleRRHHPrueba(
	t *testing.T,
	entorno *entornoConsultaRRHH,
) *ServicioConsultaDetalleRRHH {
	t.Helper()
	servicio, err := NuevoServicioConsultaDetalleRRHH(
		entorno.autoridad, entorno.autorizador, entorno.sesion, entorno.reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	return servicio
}

func configurarAmbitoCuadroRRHH(
	t *testing.T,
	entorno *entornoConsultaRRHH,
	clase ports.ClaseAmbitoConsultaRRHH,
	ambitoRef string,
	total uint16,
) {
	t.Helper()
	capacidad := capacidadConsultaCuadroRRHHV3Prueba(
		t,
		entorno.contexto,
		entorno.cuadro,
		entorno.ahora,
		clase,
		ambitoRef,
	)
	entorno.autorizador.capacidadCuadro = capacidad
	entorno.sesion.pagina.Lectura = reciboConsultaRRHHPrueba(
		t, entorno.contexto, capacidad, entorno.ahora, "", 0, total,
	)
}

func configurarAmbitoDetalleRRHH(
	t *testing.T,
	entorno *entornoConsultaRRHH,
	clase ports.ClaseAmbitoConsultaRRHH,
	ambitoRef string,
) {
	t.Helper()
	capacidad := capacidadConsultaDetalleRRHHV3Prueba(
		t,
		entorno.contexto,
		entorno.detalle,
		entorno.ahora,
		clase,
		ambitoRef,
	)
	entorno.autorizador.capacidadDetalle = capacidad
	entorno.sesion.detalle.Lectura = reciboConsultaRRHHPrueba(
		t, entorno.contexto, capacidad, entorno.ahora,
		entorno.detalle.ExpedienteRef(),
		entorno.sesion.detalle.Resumen.Version, 1,
	)
}
