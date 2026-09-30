package bootstrap

import (
	"errors"
	"strings"
	"testing"
	"time"

	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

func TestIdentidadUsuariosUsaSuSnapshotSinPerfilBolsa(t *testing.T) {
	f := fuenteProvisionCandidatoPrueba()
	f.Snapshot.Poblacion = "usuarios"
	f.Snapshot.ProvisionRef = "pue_usuarios_sinteticos_1234567890123456"
	f.Snapshot.VinculoCandidato = nil
	f.Identidad = &IdentidadProvisionCandidatoExterno{CuentaRef: f.Snapshot.Cuenta.Referencia,
		Esquema: postgresidentidad.EsquemaHMACSHA256V1, DominioRef: dominioIdentidadSesionDesarrollo,
		ClaveID: "vec.identidad.desarrollo.externo.g1", ClaveVersion: 1,
		CuentaHMAC: strings.Repeat("a", 64), SujetoHMAC: strings.Repeat("b", 64)}
	p, err := PrepararProvisionCandidatoExterno(cfgProvisionCandidatoPrueba(), f, "identidad", time.Now())
	if err != nil || p.identidad == nil || len(p.rol.RolDocumento) != 0 || len(p.asignacion.Documento) != 0 || len(p.motivos) != 0 {
		t.Fatalf("identidad Usuarios preparó autoridad de Bolsa: %v", err)
	}
	for _, fase := range []string{"autorizacion", "contexto", "bolsa", "motivos"} {
		if _, err := PrepararProvisionCandidatoExterno(cfgProvisionCandidatoPrueba(), f, fase, time.Now()); err == nil {
			t.Fatalf("la herramienta candidata habilitó %s para Usuarios", fase)
		}
	}
}

func TestProvisionCandidatoNoPublicaCorrespondenciaEnBolsa(t *testing.T) {
	if _, err := PrepararProvisionCandidatoExterno(cfgProvisionCandidatoPrueba(), fuenteProvisionCandidatoPrueba(), "bolsa", time.Now()); err != ErrProvisionCandidatoExterno {
		t.Fatal("la herramienta admitió publicar datos de identidad en Bolsa")
	}
}

func TestHuellaProvisionPropagaFalloJSONYRetiraHuellaPrevia(t *testing.T) {
	if huella, err := huellaJSONProvisionExterna(make(chan int)); huella != "" || !errors.Is(err, ErrProvisionCandidatoExterno) {
		t.Fatal("la huella ocultó el fallo de serialización")
	}
	p := PlanProvisionCandidatoExterno{desde: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)}
	p.resumen.HuellaSHA256 = strings.Repeat("a", 64)
	if err := p.actualizarHuella(); !errors.Is(err, ErrProvisionCandidatoExterno) || p.resumen.HuellaSHA256 != "" {
		t.Fatal("el plan conservó una huella aprobable tras fallar la serialización")
	}
}
