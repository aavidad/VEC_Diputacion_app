package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCanonConsumoC1CoberturaO404DCubreLimitesYMutaciones(
	t *testing.T,
) {
	base := time.Now().UTC().Truncate(time.Microsecond)
	for _, cantidad := range []int{1, 512} {
		lote, err := nuevoLoteFixtureConsumoC1O404D(cantidad, base)
		if err != nil {
			t.Fatalf("lote %d: %v", cantidad, err)
		}
		contenido, err := json.Marshal(lote)
		if err != nil {
			t.Fatal(err)
		}
		if len(contenido) >= 64*1024*1024 {
			t.Fatalf(
				"lote %d excede presupuesto SQL: %d bytes",
				cantidad,
				len(contenido),
			)
		}
		canon := canonLoteFixtureConsumoC1O404D(lote)
		huella := sha256.Sum256(canon)
		if obtenido := textoFixtureConsumoC1O404D(
			lote,
			"lote_huella_sha256",
		); obtenido != hex.EncodeToString(huella[:]) {
			t.Fatalf("canon de lote %d divergente", cantidad)
		}
	}
	for _, cantidad := range []int{0, 513} {
		if _, err := nuevoLoteFixtureConsumoC1O404D(
			cantidad,
			base,
		); err == nil {
			t.Fatalf("lote %d fuera de límite aceptado", cantidad)
		}
	}

	original, err := nuevoLoteFixtureConsumoC1O404D(2, base)
	if err != nil {
		t.Fatal(err)
	}
	canonOriginal := canonLoteFixtureConsumoC1O404D(original)
	mutaciones := map[string]func(map[string]any){
		"orden": func(lote map[string]any) {
			evidencias := lote["evidencias"].([]any)
			evidencias[0], evidencias[1] = evidencias[1], evidencias[0]
		},
		"peticion": func(lote map[string]any) {
			evidencias := lote["evidencias"].([]any)
			evidencia := evidencias[0].(map[string]any)
			evidencia["peticion_ref"] = "peticion:o404d:alterada"
			canon := canonEvidenciaFixtureConsumoC1O404D(evidencia)
			huella := sha256.Sum256(canon)
			evidencia["evidencia_huella_sha256"] =
				hex.EncodeToString(huella[:])
		},
		"prueba": func(lote map[string]any) {
			evidencias := lote["evidencias"].([]any)
			evidencia := evidencias[0].(map[string]any)
			evidencia["atestacion_canon_hex"] =
				hex.EncodeToString([]byte("atestacion:alterada"))
			canon := canonEvidenciaFixtureConsumoC1O404D(evidencia)
			huella := sha256.Sum256(canon)
			evidencia["evidencia_huella_sha256"] =
				hex.EncodeToString(huella[:])
		},
		"decision": func(lote map[string]any) {
			lote["decision_vec_ref"] = "decision:o404d:alterada"
		},
		"reserva": func(lote map[string]any) {
			lote["reserva_ref"] = "reserva:o404d:alterada"
		},
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			alterado := clonarLoteFixtureConsumoC1O404D(t, original)
			mutar(alterado)
			if string(canonOriginal) == string(
				canonLoteFixtureConsumoC1O404D(alterado),
			) {
				t.Fatal("la mutación no cambió el canon")
			}
		})
	}
}

func TestMigracionesConsumoC1CoberturaO404DTienenFronteraCerrada(
	t *testing.T,
) {
	raiz := raizRepositorioPreparacionDecisionCobertura(t)
	rutas := []string{
		"deploy/postgresql/contratacion_temporal/migraciones/" +
			"000023_consumo_c1_esquema_canones_o4_04d.up.sql",
		"deploy/postgresql/contratacion_temporal/migraciones/" +
			"000023_consumo_c1_esquema_canones_o4_04d.down.sql",
		"deploy/postgresql/contratacion_temporal/migraciones/" +
			"000024_consumo_c1_operaciones_o4_04d.up.sql",
		"deploy/postgresql/contratacion_temporal/migraciones/" +
			"000024_consumo_c1_operaciones_o4_04d.down.sql",
		"deploy/postgresql/contratacion_temporal/migraciones_autorizacion/" +
			"000004_wrapper_vec_cobertura_o4_04d.up.sql",
		"deploy/postgresql/contratacion_temporal/migraciones_autorizacion/" +
			"000004_wrapper_vec_cobertura_o4_04d.down.sql",
	}
	for _, relativa := range rutas {
		contenido, err := os.ReadFile(filepath.Join(raiz, relativa))
		if err != nil {
			t.Fatal(err)
		}
		if lineas := strings.Count(string(contenido), "\n"); lineas > 800 {
			t.Errorf("%s excede DEC-051: %d líneas", relativa, lineas)
		}
	}
	esquema := leerMigracionO404DPrueba(t, raiz, rutas[0])
	operaciones := leerMigracionO404DPrueba(t, raiz, rutas[2])
	principal := esquema + operaciones
	wrapper := leerMigracionO404DPrueba(t, raiz, rutas[4])
	exigidosPrincipal := []string{
		"version_esquema = 3", "SET version_esquema = 4",
		"consumo_cobertura_lote", "consumo_cobertura_evidencia",
		"prevalidar_bloquear_lote_consumo_c1_cobertura_o404d_v1",
		"persistir_lote_consumo_c1_cobertura_o404d_v1",
		"FORCE ROW LEVEL SECURITY", "SECURITY DEFINER",
		"pg_advisory_xact_lock", "prueba_vinculo_sha256",
		"sin EXECUTE runtime", "no aceptará resultados VEC externos",
	}
	for _, fragmento := range exigidosPrincipal {
		if !strings.Contains(principal, fragmento) {
			t.Errorf("000023/000024 no contiene %q", fragmento)
		}
	}
	exigidosWrapper := []string{
		"registrar_decision_cobertura_contratacion_temporal_v1",
		"BEGIN ATOMIC", "huella_orden_sha256", "lote_huella_sha256",
		"o404d_material_recurso_cobertura_v1", "IS DISTINCT FROM",
		"TO vec_contratacion_temporal_propietario",
		"decision_contexto_actor_v3_valida",
		"decision_contexto_actor_v3_canonica",
	}
	for _, fragmento := range exigidosWrapper {
		if !strings.Contains(wrapper, fragmento) {
			t.Errorf("000004 no contiene %q", fragmento)
		}
	}
	if strings.Contains(wrapper, ") TO vec_contratacion_temporal_ejecutor") ||
		strings.Contains(principal, ") TO vec_contratacion_temporal_ejecutor") {
		t.Fatal("una función interna O4-04D fue concedida al ejecutor")
	}
}

func leerMigracionO404DPrueba(
	t *testing.T,
	raiz string,
	relativa string,
) string {
	t.Helper()
	contenido, err := os.ReadFile(filepath.Join(raiz, relativa))
	if err != nil {
		t.Fatal(err)
	}
	return string(contenido)
}
