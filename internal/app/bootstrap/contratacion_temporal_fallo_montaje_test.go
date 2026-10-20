package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/config"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func capturarAvisoMontajeCTPrueba(t *testing.T, ejecutar func() error) (error, map[string]any) {
	t.Helper()
	var salida bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&salida, nil)))
	defer slog.SetDefault(anterior)
	err := ejecutar()
	var aviso map[string]any
	if fallo := json.Unmarshal(bytes.TrimSpace(salida.Bytes()), &aviso); fallo != nil {
		t.Fatalf("aviso no es un único JSON: %v (%s)", fallo, salida.String())
	}
	if aviso["msg"] != "contratacion_temporal_no_disponible" || aviso["level"] != "WARN" {
		t.Fatalf("aviso incorrecto: %v", aviso)
	}
	return err, aviso
}

func TestFalloMontajeCTConservaCausaSeguraYConstructorInterior(t *testing.T) {
	causa := &pgconn.PgError{Code: "42501", Message: "postgres://usuario:clave@equipo/base", Detail: "/privado/material", ConstraintName: "persona:privada"}
	err, aviso := capturarAvisoMontajeCTPrueba(t, func() error {
		fallo := marcarFalloComponenteArranque("consultas_tecnico_nuevo_perfil", causa)
		return registrarFalloMontajeCTDesarrollo("contratacion_temporal", fallo)
	})
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg != causa || !errors.Is(err, causa) || aviso["componente"] != "consultas_tecnico_nuevo_perfil" || aviso["causa"] != "bd_42501" || aviso["mensaje"] != "PostgreSQL bd_42501 restriccion=presente" {
		t.Fatalf("perdió diagnóstico o causa: %v %v", err, aviso)
	}
	_, aviso = capturarAvisoMontajeCTPrueba(t, func() error {
		return registrarFalloMontajeCTDesarrollo("constructor_ct", errors.New(causa.Message+causa.Detail+causa.ConstraintName))
	})
	for _, privado := range []string{"postgres://", "clave", "/privado", "persona:privada"} {
		if strings.Contains(aviso["mensaje"].(string), privado) {
			t.Fatalf("mensaje divulga %q", privado)
		}
	}
	if registrarFalloMontajeCTDesarrollo("constructor_ct", nil) != nil {
		t.Fatal("transformó un éxito")
	}
}

func TestGuardasConsultasTecnicoRegistranEtapaSinIdentidad(t *testing.T) {
	for _, caso := range []struct {
		nombre, etapa string
		ejecutar      func() error
	}{
		{"soporte", "consultas_tecnico_soporte", func() error { return componerPerfilFijoConsultasTecnicoCTDesarrollo(nil, time.Time{}) }},
		{"principal", "consultas_tecnico_principal", func() error {
			return componerPerfilFijoConsultasTecnicoCTDesarrollo(&soporteAltaContratacionTemporalDesarrollo{principalID: " ", certificadoSHA256: "huella:privada"}, time.Time{})
		}},
		{"ligar perfil", "consultas_tecnico_ligar_perfil", func() error { return ligarAutoridadConsultasTecnicoPerfilFijoDesarrollo(nil, nil) }},
		{"ligar sesión", "consultas_tecnico_ligar_sesion", func() error {
			_, autoridad, _, perfil, _ := escenarioConsultasTecnicoFijasPrueba(t)
			perfil.sesionOperativa = nil
			return ligarAutoridadConsultasTecnicoPerfilFijoDesarrollo(autoridad, perfil)
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			err, aviso := capturarAvisoMontajeCTPrueba(t, caso.ejecutar)
			if !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) || ComponenteFalloArranque(err) != caso.etapa || aviso["componente"] != caso.etapa || aviso["mensaje"] != ports.ErrConsultaRRHHNoDisponible.Error() {
				t.Fatalf("guarda silenciosa o sin etapa: %v %v", err, aviso)
			}
		})
	}
}

func TestGuardaLectorTecnicoRegistraCondicionYConservaCierre(t *testing.T) {
	for _, caso := range []struct {
		etapa   string
		alterar func(*dependenciasAltaContratacionTemporalDesarrollo, *identidadConsultaRRHHDesarrollo, *dependenciasConsultasRRHHDesarrollo)
	}{
		{"lector_tecnico_titular_alta", func(_ *dependenciasAltaContratacionTemporalDesarrollo, l *identidadConsultaRRHHDesarrollo, _ *dependenciasConsultasRRHHDesarrollo) {
			l.identidad.principal.ID = "persona:privada"
		}},
		{"lector_tecnico_perfil_consultas", func(a *dependenciasAltaContratacionTemporalDesarrollo, _ *identidadConsultaRRHHDesarrollo, _ *dependenciasConsultasRRHHDesarrollo) {
			a.soporte.perfilesFijos = nil
		}},
		{"lector_tecnico_registrar_mux", func(_ *dependenciasAltaContratacionTemporalDesarrollo, _ *identidadConsultaRRHHDesarrollo, b *dependenciasConsultasRRHHDesarrollo) {
			b.cuadroHTTP = nil
		}},
	} {
		t.Run(caso.etapa, func(t *testing.T) {
			alta, _, lector := escenarioLectorTecnicoFijoPrueba(t)
			if err := componerPerfilFijoConsultasTecnicoCTDesarrollo(alta.soporte, alta.soporte.reloj.Ahora()); err != nil {
				t.Fatal(err)
			}
			cierres := 0
			base := dependenciasConsultasRRHHDesarrollo{cuadroHTTP: cuadroRRHHPrueba{}, detalleHTTP: detalleRRHHPrueba{}, originalPropuestaHTTP: detalleRRHHPrueba{}, cerrar: func() { cierres++ }}
			caso.alterar(alta, &lector, &base)
			err, aviso := capturarAvisoMontajeCTPrueba(t, func() error {
				_, err := nuevasDependenciasLectoresRRHHDesarrollo(context.Background(), config.Config{}, alta, nil, relojContratacionTemporalDesarrollo{}, []identidadConsultaRRHHDesarrollo{lector}, &postgresct.PoolConsultasRRHHPostgreSQL{}, motivosConsultaRRHHPrueba{}, dominiovec.ReferenciaEntradaCatalogo{}, dominiovec.ReferenciaEntradaCatalogo{}, base, catalogoFronterasComunDesarrollo{}, false)
				return err
			})
			if !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) || aviso["componente"] != caso.etapa || aviso["mensaje"] != ports.ErrConsultaRRHHNoDisponible.Error() || cierres != 1 {
				t.Fatalf("guarda sin diagnóstico o cierre: %v %v cierres=%d", err, aviso, cierres)
			}
		})
	}
}
