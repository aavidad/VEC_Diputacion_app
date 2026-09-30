package bootstrap

import (
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
