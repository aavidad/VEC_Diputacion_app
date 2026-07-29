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
	renovarEmisorConsultaRRHHPrueba(t, entorno)
	servicio, err := NuevoServicioConsultaCuadroRRHH(
		entorno.autoridad, entorno.emisor, entorno.sesion, entorno.reloj,
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
	renovarEmisorConsultaRRHHPrueba(t, entorno)
	servicio, err := NuevoServicioConsultaDetalleRRHH(
		entorno.autoridad, entorno.emisor, entorno.sesion, entorno.reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	return servicio
}

func renovarEmisorConsultaRRHHPrueba(
	t *testing.T,
	entorno *entornoConsultaRRHH,
) {
	t.Helper()
	if entorno.emision != nil &&
		entorno.emision.cuadro.instante.Equal(entorno.reloj.instante) &&
		entorno.emision.detalle.instante.Equal(entorno.reloj.instante) {
		return
	}
	emision := nuevoEmisorConsultaRRHHV3Prueba(t, entorno.reloj.instante)
	entorno.emision = emision
	entorno.emisor = emision.emisor
}
