package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

func fuenteProvisionParticipacionBolsaPrueba() FuenteProvisionCandidatoExterno {
	f := fuenteProvisionCandidatoPrueba()
	f.Bolsa = &ContextoParticipacionBolsaExterna{ParticipacionRef: "participacion:clara-navarro", BolsaRef: "bolsa:administrativo",
		CandidatoRef: f.Snapshot.VinculoCandidato.CandidatoRef, PersonaRef: f.Snapshot.Persona.Referencia,
		PerfilRef: f.Snapshot.Perfil.Referencia, ContextoActorRef: f.Snapshot.Contexto.Referencia,
		Estado: "activo", VigenteDesde: "2026-01-01T00:00:00.000000Z", VigenteHasta: "2027-01-01T00:00:00.000000Z",
		FuenteRef: "prc_clasificacion_aprobada_1234567890123456", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("a", 64)}
	return f
}

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

func TestProvisionParticipacionBolsaRequiereFuentePropiaYPreimagen(t *testing.T) {
	f := fuenteProvisionParticipacionBolsaPrueba()
	p, err := PrepararProvisionCandidatoExterno(cfgProvisionCandidatoPrueba(), f, "bolsa", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if p.bolsa == nil || p.huellaBolsa != "" {
		t.Fatal("preparación pura sustituyó la huella SQL")
	}
	if _, err := EjecutarProvisionCandidatoExterno(context.Background(), cfgProvisionCandidatoPrueba(), "dsn rechazada", p,
		p.resumen.HuellaSHA256, p.resumen.PreimagenSHA256); err != ErrProvisionCandidatoExterno {
		t.Fatal("ejecutó un plan sin huella de la preimagen SQL")
	}
	for _, mutar := range []func(*FuenteProvisionCandidatoExterno){
		func(f *FuenteProvisionCandidatoExterno) { f.Bolsa = nil },
		func(f *FuenteProvisionCandidatoExterno) { f.Bolsa.PerfilRef = "prf_otro_perfil_1234567890123456" },
		func(f *FuenteProvisionCandidatoExterno) {
			f.Bolsa.ContextoActorRef = "vca_otro_contexto_1234567890123456"
		},
		func(f *FuenteProvisionCandidatoExterno) { f.Preimagen.VersionBolsa = 1 },
	} {
		fuente := fuenteProvisionParticipacionBolsaPrueba()
		mutar(&fuente)
		if _, err := PrepararProvisionCandidatoExterno(cfgProvisionCandidatoPrueba(), fuente, "bolsa", time.Now()); err == nil {
			t.Fatal("preparó una participación sin fuente exacta o sin CAS")
		}
	}
}

func TestProvisionParticipacionBolsaConsumeHuellaSQLYConfirmacion(t *testing.T) {
	p, err := PrepararProvisionCandidatoExterno(cfgProvisionCandidatoPrueba(), fuenteProvisionParticipacionBolsaPrueba(), "bolsa", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	huella := strings.Repeat("b", 64)
	p.huellaBolsa = huella
	p.actualizarHuella()
	q := &consultaProvisionPrueba{respuesta: func(n int, sql string, args []any) pgx.Row {
		switch n {
		case 1:
			if !strings.Contains(sql, "preimagen_contexto_participacion_externa") {
				t.Fatal(sql)
			}
			return filaProvisionPrueba{err: pgx.ErrNoRows}
		case 2:
			if !strings.Contains(sql, "huella_contexto_participacion_externa") || args[1] != int64(1) {
				t.Fatal("perdió versión aprobada")
			}
			return filaProvisionPrueba{valores: []any{huella}}
		case 3:
			if !strings.Contains(sql, "publicar_contexto_participacion_externa") || args[1] != int64(0) || args[2] != nil || args[3] != huella {
				t.Fatal("publicó con otro CAS o aprobación")
			}
			return filaProvisionPrueba{valores: []any{p.bolsa.ParticipacionRef, int64(1), huella}}
		default:
			t.Fatal("consulta inesperada")
			return nil
		}
	}}
	if err := publicarParticipacionBolsaExterna(context.Background(), q, p); err != nil || q.llamadas != 3 {
		t.Fatal(err)
	}
	q = &consultaProvisionPrueba{respuesta: func(_ int, _ string, _ []any) pgx.Row {
		return filaProvisionPrueba{valores: []any{int64(2), strings.Repeat("c", 64)}}
	}}
	if err := publicarParticipacionBolsaExterna(context.Background(), q, p); err != ErrProvisionCandidatoExterno || q.llamadas != 1 {
		t.Fatal("preimagen divergente llegó a publicar")
	}
}
