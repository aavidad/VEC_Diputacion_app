package application

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestConsultaResultadoClasificaDenegacionesDirectasYEnvueltas(t *testing.T) {
	sentinelas := []error{
		ports.ErrAutorizacionDenegada,
		dominiovec.ErrAutorizacionDenegada,
	}
	for indice, sentinel := range sentinelas {
		for _, envuelta := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d_%t", indice, envuelta), func(t *testing.T) {
				escenario := nuevoEscenarioConfirmacionCobertura(t, true)
				causa := sentinel
				if envuelta {
					causa = fmt.Errorf("detalle privado: %w", sentinel)
				}
				accesos := &autorizadorLecturaResultadoCoberturaPrueba{
					err: causa,
				}
				lector := &lectorResultadoHistoricoCoberturaPrueba{
					noObservable: true,
				}
				servicio := nuevoServicioConsultaResultadoPrueba(
					t,
					contextoRecuperacionDesdeDecisionPrueba(escenario),
					accesos,
					&selladorAmbitoConsultaCoberturaPrueba{},
					escenario.base.reloj,
					lector,
				)
				_, err := servicio.Consultar(
					context.Background(),
					solicitudConsultaDesdeDecisionPrueba(escenario),
				)
				if !errors.Is(err, ErrConsultaResultadoCoberturaDenegada) {
					t.Fatalf("denegación mal clasificada: %v", err)
				}
				if consultas, efectos := lector.totales(); consultas != 0 ||
					efectos != 0 {
					t.Fatalf("denegación alcanzó persistencia: %d/%d", consultas, efectos)
				}
			})
		}
	}
}

func TestDatosSolicitudLecturaResultadoCoberturaSiempreRedactados(t *testing.T) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, true)
	accesos := &autorizadorLecturaResultadoCoberturaPrueba{}
	servicio := nuevoServicioConsultaResultadoPrueba(
		t,
		contextoRecuperacionDesdeDecisionPrueba(escenario),
		accesos,
		&selladorAmbitoConsultaCoberturaPrueba{},
		escenario.base.reloj,
		&lectorResultadoHistoricoCoberturaPrueba{noObservable: true},
	)
	if _, err := servicio.Consultar(
		context.Background(),
		solicitudConsultaDesdeDecisionPrueba(escenario),
	); err != nil {
		t.Fatal(err)
	}
	datos, existe := accesos.ultima()
	if !existe {
		t.Fatal("PDP no recibió solicitud")
	}
	representaciones := map[string]string{
		"v":       fmt.Sprintf("%v", datos),
		"+v":      fmt.Sprintf("%+v", datos),
		"#v":      fmt.Sprintf("%#v", datos),
		"puntero": fmt.Sprintf("%+v", &datos),
	}
	var registro bytes.Buffer
	slog.New(slog.NewJSONHandler(&registro, nil)).Info(
		"prueba_redaccion",
		"datos",
		datos,
	)
	representaciones["slog"] = registro.String()
	sensibles := []string{
		datos.OrganizacionRef,
		datos.ExpedienteRef,
		string(datos.Accion),
		string(datos.Finalidad),
	}
	for nombre, representacion := range representaciones {
		if !strings.Contains(representacion, "REDACTAD") {
			t.Errorf("%s no aplicó redacción: %q", nombre, representacion)
		}
		for _, sensible := range sensibles {
			if sensible != "" && strings.Contains(representacion, sensible) {
				t.Errorf("%s filtró %q", nombre, sensible)
			}
		}
	}
}

func TestConsultaResultadoClasificaFallosTCBSinFiltrarCausa(t *testing.T) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, true)
	causaPrivada := errors.New("dsn y detalle privado del primario")
	casos := map[string]struct {
		lector   *lectorResultadoHistoricoCoberturaPrueba
		esperado error
	}{
		"no_disponible": {
			lector: &lectorResultadoHistoricoCoberturaPrueba{
				err: causaPrivada,
			},
			esperado: ErrConsultaResultadoCoberturaNoDisponible,
		},
		"no_confiable": {
			lector:   &lectorResultadoHistoricoCoberturaPrueba{},
			esperado: ErrConsultaResultadoCoberturaNoConfiable,
		},
		"conflicto": {
			lector: &lectorResultadoHistoricoCoberturaPrueba{
				err: cobertura.ErrHistoriaResultadoOperacionDecisionCoberturaDivergente,
			},
			esperado: ErrConsultaResultadoCoberturaConflicto,
		},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			servicio := nuevoServicioConsultaResultadoPrueba(
				t,
				contextoRecuperacionDesdeDecisionPrueba(escenario),
				&autorizadorLecturaResultadoCoberturaPrueba{},
				&selladorAmbitoConsultaCoberturaPrueba{},
				escenario.base.reloj,
				caso.lector,
			)
			_, err := servicio.Consultar(
				context.Background(),
				solicitudConsultaDesdeDecisionPrueba(escenario),
			)
			if !errors.Is(err, caso.esperado) {
				t.Fatalf("fallo TCB mal clasificado: %v", err)
			}
			if strings.Contains(err.Error(), causaPrivada.Error()) {
				t.Fatalf("causa privada filtrada: %v", err)
			}
		})
	}
}
