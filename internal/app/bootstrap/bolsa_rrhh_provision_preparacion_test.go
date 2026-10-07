package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/config"
)

func TestPreparacionProvisionRRHHBolsaLeeCertificadoEIdentidadActuales(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	principal, err := cargarPrincipalPreparacionRRHHBolsa(cfg, time.Now())
	if err != nil || !principalContratacionTemporalDesarrolloValido(principal) ||
		principal.Attributes["certificate_sha256"] == "" {
		t.Fatalf("identidad RRHH de material público no disponible: %v", err)
	}
}

func TestPreparacionProvisionRRHHBolsaSoloLeeGobiernoYPreservaVersion(t *testing.T) {
	politica, autoridad, vinculo := politicaProvisionBolsaPrueba(t, 6)
	lecturas := 0
	lector := &lectorSoloLecturaProvisionRRHHBolsa{
		leer: func(_ context.Context, perfilRef string) (instantaneaPublicadaDesarrollo, bool, error) {
			lecturas++
			if perfilRef != vinculo.PerfilActivoRef {
				t.Fatalf("perfil ajeno consultado: %q", perfilRef)
			}
			return autoridad.leida, true, nil
		},
	}
	preparada, err := prepararProvisionLecturasRRHHBolsaDesdeLectura(
		context.Background(), politica.soporte, lector, relojContratacionTemporalDesarrollo{},
	)
	if err != nil {
		t.Fatal(err)
	}
	esperadaPreimagen, esperadaObjetivo, err := HuellasProvisionLecturasNominalesRRHHBolsa(context.Background(), politica)
	if err != nil {
		t.Fatal(err)
	}
	if preparada.Estado != estadoProvisionRRHHBolsaPendienteAprobacion ||
		preparada.VersionPreimagen != 6 || preparada.VersionObjetivo != 22 ||
		preparada.PreimagenSHA256 != esperadaPreimagen || preparada.ObjetivoSHA256 != esperadaObjetivo ||
		lecturas != 1 || lector.mutaciones != 0 || autoridad.publicadas != 0 {
		t.Fatalf("preparación alteró autoridad o huellas: %+v lecturas=%d mutaciones=%d publicadas=%d",
			preparada, lecturas, lector.mutaciones, autoridad.publicadas)
	}
}

func TestPreparacionProvisionRRHHBolsaFallaCerradaSinAsignacionOConfiguracion(t *testing.T) {
	politica, _, _ := politicaProvisionBolsaPrueba(t, 0)
	lector := &lectorSoloLecturaProvisionRRHHBolsa{
		leer: func(context.Context, string) (instantaneaPublicadaDesarrollo, bool, error) {
			return instantaneaPublicadaDesarrollo{}, false, nil
		},
	}
	if _, err := prepararProvisionLecturasRRHHBolsaDesdeLectura(context.Background(), politica.soporte, lector,
		relojContratacionTemporalDesarrollo{}); err == nil || lector.mutaciones != 0 {
		t.Fatalf("asignación ausente admitida o escrita: %v, mutaciones=%d", err, lector.mutaciones)
	}
	if _, err := PrepararProvisionLecturasRRHHBolsaSinServidor(context.Background(), config.Config{}); !errors.Is(err, ErrActivacionDesarrolloInvalida) {
		t.Fatalf("doble llave ausente admitida: %v", err)
	}
}
