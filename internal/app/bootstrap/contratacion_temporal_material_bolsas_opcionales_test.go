package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/config"
)

func escribirMaterialBolsasOpcionalPrueba(t *testing.T, ruta string, dato any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(dato)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSeleccionMaterialBolsasInternasOpcional(t *testing.T) {
	raiz := t.TempDir()
	interno, externo := filepath.Join(raiz, "interno"), filepath.Join(raiz, "externo")
	cfg := config.Config{ExecutionProfile: config.ExecutionProfileDevelopment,
		AuthMode: config.AuthModeDevelopment, DevelopmentGuard: config.DevelopmentGuardAcknowledgement,
		PortalProceso: config.ValorPortalProcesoInterno, DevelopmentMaterialDir: interno, ExternoMaterialDir: externo}
	t.Setenv(envBolsaPreferenciasInternas, "true")
	if pref, rrhh := seleccionarMaterialBolsasInternasOpcional(cfg); pref || rrhh {
		t.Fatal("la ausencia de material publicó una audiencia interna")
	}
	escribirMaterialBolsasOpcionalPrueba(t, filepath.Join(interno, "identidad", "bolsa-preferencias-interna.json"), map[string]any{
		"version": 1, "superficie": "interna_corporativa", "dsn_bolsa": "postgresql://nominal@localhost/postgres",
		"certificado_sha256": strings.Repeat("a", 64), "sujeto": "persona-ficticia",
		"cuenta_ref": "cta_0123456789abcdefghijkl", "perfil_ref": "prf_0123456789abcdefghijkl",
		"persona_ref": "per_0123456789abcdefghijkl", "candidato_ref": "can_0123456789abcdefghijkl",
	})
	if pref, rrhh := seleccionarMaterialBolsasInternasOpcional(cfg); !pref || rrhh {
		t.Fatal("el material propio válido no seleccionó solo preferencias")
	}
	escribirMaterialBolsasOpcionalPrueba(t, filepath.Join(interno, "identidad", "bolsa-justificantes-rrhh.json"), map[string]any{
		"version": 1, "superficie": "interna_corporativa", "dsn_documentos": "postgresql://nominal@localhost/postgres",
		"certificado_sha256": strings.Repeat("b", 64), "sujeto": "tecnica-ficticia",
		"cuenta_ref": "cta_0123456789abcdefghijkl", "perfil_ref": "prf_0123456789abcdefghijkl",
		"persona_ref": "per_0123456789abcdefghijkl",
	})
	almacen := filepath.Join(raiz, "almacen")
	if err := os.MkdirAll(almacen, 0o700); err != nil {
		t.Fatal(err)
	}
	rutaJustificantes := filepath.Join(externo, "externo", "documentos", "justificantes-bolsa.json")
	escribirMaterialBolsasOpcionalPrueba(t, rutaJustificantes, map[string]any{
		"version": 1,
		"datos": map[string]any{"version": 1, "dsn_datos": "postgresql://nominal@localhost/postgres",
			"dsn_frontera": "", "directorio": almacen, "retencion_minima_dias": 0,
			"clave_correlacion_archivo": "correlacion.bin", "clave_seudonimo_archivo": "seudonimo.bin",
			"clave_solicitud_archivo": "solicitud.bin",
			"analizador": map[string]any{"endpoint": "https://127.0.0.1", "nombre_servidor": "analizador",
				"ruta_ca": "ca.crt", "ruta_certificado": "cliente.crt", "ruta_clave": "cliente.key",
				"conector_id": "conector-ficticio"}},
		"motivo": map[string]any{"catalogo_id": "motivos_autorizacion_justificante_renuncia_bolsa",
			"catalogo_version": 1, "catalogo_huella_sha256": strings.Repeat("c", 64),
			"entrada_clave": "motivo_" + strings.Repeat("d", 32)},
	})
	if pref, rrhh := seleccionarMaterialBolsasInternasOpcional(cfg); !pref || !rrhh {
		t.Fatal("material RRHH válido no seleccionó sus dos audiencias")
	}
	if err := os.WriteFile(rutaJustificantes, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if pref, rrhh := seleccionarMaterialBolsasInternasOpcional(cfg); !pref || rrhh {
		t.Fatal("material RRHH inválido afectó preferencias o abrió RRHH")
	}
	t.Setenv(envBolsaPreferenciasInternas, "incorrecto")
	if pref, rrhh := seleccionarMaterialBolsasInternasOpcional(cfg); pref || rrhh {
		t.Fatal("el selector inválido publicó material")
	}
	t.Setenv(envBolsaPreferenciasInternas, "false")
	if pref, rrhh := seleccionarMaterialBolsasInternasOpcional(cfg); pref || rrhh {
		t.Fatal("el selector apagado publicó material")
	}
	t.Setenv(envBolsaPreferenciasInternas, "true")
	cfg.DevelopmentGuard = ""
	if pref, rrhh := seleccionarMaterialBolsasInternasOpcional(cfg); pref || rrhh {
		t.Fatal("sin doble llave se publicó material interno")
	}
}

func TestAudienciasBolsasInternasSoloConSeleccionYPreflight(t *testing.T) {
	base := descriptoresMaterialSeleccionadosCTDesarrollo(seleccionMaterialCTDesarrollo{})
	activas := descriptoresMaterialSeleccionadosCTDesarrollo(seleccionMaterialCTDesarrollo{
		preferenciasBolsasInternas: true, justificanteRenunciaRRHH: true})
	esperadas := []string{descriptorMaterialOrdenarPreferenciasBolsasInternaV3().Audiencia,
		audienciaJustificanteBolsaRRHH, audienciaJustificanteBolsaPrepararRRHH}
	if len(activas) != len(base)+len(esperadas) {
		t.Fatalf("se añadieron %d audiencias; esperadas %d", len(activas)-len(base), len(esperadas))
	}
	for _, d := range activas {
		if slices.Contains(esperadas, d.Audiencia) && !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(d.Audiencia) {
			t.Fatalf("audiencia interna seleccionada fuera del gobierno: %s", d.Audiencia)
		}
	}
	for _, d := range base {
		if slices.Contains(esperadas, d.Audiencia) {
			t.Fatalf("audiencia interna publicada con selector apagado: %s", d.Audiencia)
		}
	}
}

type consultaMaterialBolsasOpcionalPrueba struct {
	preferencias, rrhh bool
	err                error
}

func (c consultaMaterialBolsasOpcionalPrueba) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	return filaGobiernoContinuidadPrueba(func(destinos ...any) error {
		if !strings.Contains(sql, "registrar_y_consumir_preferencias_bolsas_interna_v3_atestada") ||
			!strings.Contains(sql, "consumir_subida_local_bolsa_v1") ||
			!strings.Contains(sql, "consultar_justificante_renuncia_rrhh_v1") {
			return errors.New("preflight nominal incompleto")
		}
		if c.err != nil {
			return c.err
		}
		*destinos[0].(*bool), *destinos[1].(*bool) = c.preferencias, c.rrhh
		return nil
	})
}

func TestPreflightMaterialBolsasInternasFallaSoloCerrandoClaves(t *testing.T) {
	for _, c := range []consultaMaterialBolsasOpcionalPrueba{
		{false, false, nil}, {true, false, nil}, {false, true, nil}, {true, true, nil},
		{true, true, errors.New("PostgreSQL no disponible")},
	} {
		pref, rrhh := comprobarPreflightMaterialBolsasInternas(t.Context(), c)
		if c.err != nil {
			if pref || rrhh {
				t.Fatal("una sonda fallida abrió material interno")
			}
			continue
		}
		if pref != c.preferencias || rrhh != c.rrhh {
			t.Fatalf("preflight cambió capacidades: pref=%t rrhh=%t", pref, rrhh)
		}
	}
}
