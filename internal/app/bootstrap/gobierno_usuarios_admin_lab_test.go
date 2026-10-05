package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type configSetupRootDEV struct {
	configuracionEnsayoGobiernoUsuarios
	AutorizaRootDEV    bool
	AutorizacionRef    string
	PlanAprobadoSHA256 string
}
type planSetupRootDEV struct {
	Version             uint64    `json:"version"`
	AutorizacionRef     string    `json:"autorizacion_ref"`
	ConfiguracionSHA256 string    `json:"configuracion_sha256"`
	PreimagenSHA256     string    `json:"preimagen_sha256"`
	SPIAnteriorSHA256   string    `json:"spki_anterior_sha256"`
	SPKIDEVSHA256       string    `json:"spki_dev_sha256"`
	PreparadoEn         time.Time `json:"preparado_en"`
	CaducaEn            time.Time `json:"caduca_en"`
	AuditoriaEsperada   uint64    `json:"auditoria_esperada"`
}

// Setup de LAB autorizado por dirección11:34, no operación de producto ni
// permiso humano. Usa el publicador existente y conserva AUD/historia.
func TestGobiernoUsuariosRootDEVSetupPrivado(t *testing.T) {
	ruta := os.Getenv("VEC_GOBIERNO_USUARIOS_LAB_CONFIG")
	if ruta == "" {
		t.Skip("setup_lab_no_configurado")
	}
	raw, err := leerFicheroMaterialSeguro(ruta, 16384)
	if err != nil {
		t.Fatal("setup_config_invalida")
	}
	defer borrarBytes(raw)
	var cfg configSetupRootDEV
	if decodificarGobiernoUsuarios(raw, &cfg) != nil || validarConfiguracionEnsayoGobiernoUsuarios(cfg.configuracionEnsayoGobiernoUsuarios) != nil || !cfg.AutorizaRootDEV || cfg.AutorizacionRef == "" || (cfg.Fase != "preparar" && cfg.Fase != "aplicar") {
		t.Fatal("setup_clon_no_autorizado")
	}
	rootSalida, err := AbrirRaizPrivadaDenominacionPersona(filepath.Join(cfg.Salida, "plan-setup.json"))
	if err != nil {
		t.Fatal("setup_salida_no_privada")
	}
	defer rootSalida.Close()
	escribir := func(n string, b []byte) {
		t.Helper()
		if escribirPrivadoGobiernoUsuarios(rootSalida, n, b) != nil {
			t.Fatal("setup_salida_no_privada")
		}
	}
	var semillaSalida, actaSalida *os.File
	if cfg.Fase == "aplicar" {
		semillaSalida, actaSalida, err = reservarSalidasSetupRootDEV(rootSalida)
		if err != nil {
			t.Fatal("setup_salida_existente_o_no_privada")
		}
		defer semillaSalida.Close()
		defer actaSalida.Close()
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, cfg.DSNPropietario)
	if err != nil {
		t.Fatal("setup_pool_ausente")
	}
	defer pool.Close()
	mat, err := cargarMaterialIdempotenciaDesarrollo(cfg.DirectorioMaterial, cfg.RutaConfiguracionHMAC)
	if err != nil {
		t.Fatal("setup_material_existente_ausente")
	}
	defer mat.borrar()
	derivador, err := nuevoDerivadorIdentidadOperacionDesarrollo(&mat)
	if err != nil {
		t.Fatal("setup_derivador_ausente")
	}
	defer derivador.borrar()
	ahora := relojGobiernoUsuariosEnsayo{}.Ahora()
	dev, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, ahora)
	if err != nil {
		t.Fatal("setup_proveedor_ausente")
	}
	defer borrarBytes(dev.privada)
	defer borrarBytes(dev.claveHMAC)
	baseCfg := cfg
	baseCfg.Fase = ""
	baseCfg.PlanAprobadoSHA256 = ""
	canonCfg, err := json.Marshal(baseCfg)
	if err != nil {
		t.Fatal("setup_config_invalida")
	}
	cfgSHA := sha256.Sum256(canonCfg)
	snapshot := func(q interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	}) (string, string, uint64) {
		t.Helper()
		var pre, sig string
		var count uint64
		if q.QueryRow(ctx, `SELECT encode(sha256(convert_to(vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1()::text,'UTF8')),'hex'),(SELECT r.huella_spki_sha256 FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual a JOIN vec_autorizacion_atestada_v3.configuracion_raiz c ON c.configuracion_revision=a.configuracion_revision JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r ON r.clave_id=c.raiz_clave_id AND r.version=c.raiz_version ORDER BY a.orden DESC LIMIT 1),(SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)`).Scan(&pre, &sig, &count) != nil {
			t.Fatal("setup_preimagen_ausente")
		}
		return pre, sig, count
	}
	pre, spki, count := snapshot(pool)
	if cfg.Fase == "preparar" {
		p := planSetupRootDEV{Version: 1, AutorizacionRef: cfg.AutorizacionRef, ConfiguracionSHA256: hex.EncodeToString(cfgSHA[:]), PreimagenSHA256: pre, SPIAnteriorSHA256: spki, SPKIDEVSHA256: dev.spkiHuella, PreparadoEn: ahora, CaducaEn: ahora.Add(30 * time.Minute), AuditoriaEsperada: count}
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatal("setup_plan_invalido")
		}
		escribir("plan-setup.json", b)
		h := sha256.Sum256(b)
		escribir("plan-setup.sha256", []byte(hex.EncodeToString(h[:])))
		return
	}
	planRaw, err := LeerEnRaizPrivadaDenominacionPersona(rootSalida, "plan-setup.json", 16384)
	if err != nil {
		t.Fatal("setup_plan_ausente")
	}
	h := sha256.Sum256(planRaw)
	var p planSetupRootDEV
	if decodificarGobiernoUsuarios(planRaw, &p) != nil || hex.EncodeToString(h[:]) != cfg.PlanAprobadoSHA256 || p.ConfiguracionSHA256 != hex.EncodeToString(cfgSHA[:]) || p.AutorizacionRef != cfg.AutorizacionRef || !ahora.Before(p.CaducaEn) || p.PreimagenSHA256 != pre || p.SPIAnteriorSHA256 != spki || p.SPKIDEVSHA256 != dev.spkiHuella || p.AuditoriaEsperada != count {
		t.Fatal("setup_plan_CAS_no_aprobado")
	}
	con, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal("setup_conexion_ausente")
	}
	defer con.Release()
	if _, err = con.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended('vec:ct:desarrollo:gobierno-atestacion',0))`); err != nil {
		t.Fatal("setup_cerrojo_ausente")
	}
	defer func() {
		_, _ = con.Exec(ctx, `SELECT pg_advisory_unlock(hashtextextended('vec:ct:desarrollo:gobierno-atestacion',0))`)
	}()
	tx, err := con.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		t.Fatal("setup_transaccion_ausente")
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;SET LOCAL timezone='UTC'`); err != nil {
		t.Fatal("setup_owner_ausente")
	}
	pre2, sig2, count2 := snapshot(tx)
	if pre2 != p.PreimagenSHA256 || sig2 != p.SPIAnteriorSHA256 || count2 != p.AuditoriaEsperada || !(relojGobiernoUsuariosEnsayo{}).Ahora().Before(p.CaducaEn) {
		t.Fatal("setup_CAS_distinto")
	}
	if _, err = semillaSalida.Write(dev.privada.Seed()); err != nil {
		t.Fatal("setup_semilla_no_escrita")
	}
	if err = semillaSalida.Sync(); err != nil {
		t.Fatal("setup_semilla_no_durable")
	}
	if publicarGobiernoAtestacionCTEnTxDesarrollo(ctx, tx, &dev) != nil {
		t.Fatal("setup_publicador_existente_rechaza")
	}
	_, sigFinal, countFinal := snapshot(tx)
	if sigFinal != p.SPKIDEVSHA256 || countFinal != count {
		t.Fatal("setup_resultado_incompatible")
	}
	if tx.Commit(ctx) != nil {
		t.Fatal("setup_COMMIT_indeterminado")
	}
	acta := map[string]any{"setup_lab_explicito": true, "no_root_principal": true, "autorizacion_ref": cfg.AutorizacionRef, "plan_sha256": cfg.PlanAprobadoSHA256, "preimagen_sha256": p.PreimagenSHA256, "spki_anterior_sha256": p.SPIAnteriorSHA256, "spki_dev_sha256": dev.spkiHuella, "root_clave_id": dev.claveID, "root_version": dev.claveVersion, "gobierno_ref": dev.configuracionRef, "gobierno_secuencia": dev.configuracionOrden, "gobierno_sha256": dev.configuracionHuella, "auditoria_antes": count, "auditoria_despues": countFinal, "publicador": "publicarGobiernoAtestacionCTEnTxDesarrollo"}
	b, err := json.Marshal(acta)
	if err != nil {
		t.Fatal("setup_acta_invalida")
	}
	if _, err = actaSalida.Write(b); err != nil {
		t.Fatal("setup_acta_no_escrita")
	}
	if err = actaSalida.Sync(); err != nil {
		t.Fatal("setup_acta_no_durable")
	}
}

// El publicador ADMIN no puede hacer que CT seleccione una clave ajena.
func TestGobiernoUsuariosCoexistenciaCTPrivado(t *testing.T) {
	ruta := os.Getenv("VEC_GOBIERNO_USUARIOS_ENSAYO_CONFIG")
	if ruta == "" {
		t.Skip("ensayo_privado_no_configurado")
	}
	raw, err := leerFicheroMaterialSeguro(ruta, 16384)
	if err != nil {
		t.Fatal("coexistencia_config_invalida")
	}
	defer borrarBytes(raw)
	var cfg configuracionEnsayoGobiernoUsuarios
	if decodificarGobiernoUsuarios(raw, &cfg) != nil || validarConfiguracionEnsayoGobiernoUsuarios(cfg) != nil {
		t.Fatal("coexistencia_config_invalida")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, cfg.DSNPropietario)
	if err != nil {
		t.Fatal("coexistencia_pool_ausente")
	}
	defer pool.Close()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Fatal("coexistencia_tx_ausente")
	}
	defer tx.Rollback(ctx)
	propio, err := gobiernoActualPostgreSQLContratacionTemporalDesarrolloEsPropio(ctx, tx)
	if err != nil || !propio {
		t.Fatal("coexistencia_CT_ajeno")
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal("coexistencia_commit_ausente")
	}
}

// Reservar ambas salidas antes de conectar evita cambiar el gobierno cuando
// alguna evidencia ya existe. El descriptor del acta se completa tras COMMIT.
func reservarSalidasSetupRootDEV(root *os.Root) (*os.File, *os.File, error) {
	if root == nil {
		return nil, nil, ErrGobiernoUsuariosAdmin
	}
	acta, err := root.OpenFile("acta-setup-root-dev.json", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, nil, ErrGobiernoUsuariosAdmin
	}
	semilla, err := root.OpenFile("semilla-root-dev.bin", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		cerrar := acta.Close()
		retirar := root.Remove("acta-setup-root-dev.json")
		if cerrar != nil || retirar != nil {
			return nil, nil, ErrGobiernoUsuariosAdmin
		}
		return nil, nil, ErrGobiernoUsuariosAdmin
	}
	return semilla, acta, nil
}

func TestGobiernoUsuariosRootDEVReservaAntesEfectoSinPG(t *testing.T) {
	for _, nombre := range []string{"semilla-root-dev.bin", "acta-setup-root-dev.json"} {
		t.Run(nombre, func(t *testing.T) {
			d := t.TempDir()
			if os.Chmod(d, 0700) != nil {
				t.Fatal("setup_directorio_invalido")
			}
			if os.WriteFile(filepath.Join(d, nombre), []byte("original"), 0600) != nil {
				t.Fatal("setup_fixture_invalido")
			}
			root, err := AbrirRaizPrivadaDenominacionPersona(filepath.Join(d, nombre))
			if err != nil {
				t.Fatal("setup_root_invalido")
			}
			defer root.Close()
			semilla, acta, err := reservarSalidasSetupRootDEV(root)
			if err == nil || semilla != nil || acta != nil {
				t.Fatal("setup_salida_existente_aceptada")
			}
			original, err := os.ReadFile(filepath.Join(d, nombre))
			if err != nil || string(original) != "original" {
				t.Fatal("setup_evidencia_original_alterada")
			}
			entradas, err := os.ReadDir(d)
			if err != nil || len(entradas) != 1 {
				t.Fatal("setup_reserva_parcial_conservada")
			}
		})
	}
}
