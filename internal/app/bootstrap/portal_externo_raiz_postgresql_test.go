package bootstrap

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
)

// Solo una copia desechable de la principal sintética con AD3-123 instalado.
// No aplica migraciones ni modifica roles; publica y rota material propio en
// esa copia para probar el mismo protocolo que usa la preparación del operador.
func TestRaizExternaPostgreSQLConservaConsumoYRenovacionInternos(t *testing.T) {
	if os.Getenv("VEC_RAIZ_EXTERNA_V3_PG_DESECHABLE") != "1" {
		t.Skip("requiere clon PostgreSQL 18 desechable")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	c, err := pgxpool.ParseConfig(os.Getenv("VEC_RAIZ_EXTERNA_V3_PRUEBA_DATABASE_URL"))
	if err != nil {
		t.Fatal("configuración de prueba no disponible")
	}
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = map[string]string{}
	}
	c.ConnConfig.RuntimeParams["timezone"] = "UTC"
	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		t.Fatal("pool de prueba no disponible")
	}
	defer pool.Close()
	var valida bool
	if err := pool.QueryRow(ctx, `SELECT current_setting('server_version_num')::int/10000=18 AND to_regprocedure('vec_autorizacion_atestada_v3.publicar_confianza_externa_v1(jsonb,text,text)') IS NOT NULL AND NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.puntero_configuracion_externa)`).Scan(&valida); err != nil || !valida {
		t.Fatal("requiere clon PG18 con publicación externa vacía")
	}
	interna := materialPublicoInternoClonPrueba(t, ctx, pool)
	ahora := time.Now().UTC()
	interna, err = renovarConfiguracionConfianzaCTDesarrollo(ctx, pool, interna, ahora)
	if err != nil {
		t.Fatalf("renovación interna previa: %v", err)
	}
	reloj := &relojRenovableCTPrueba{}
	reloj.fijar(ahora)
	fuente, err := nuevaFuenteConfianzaRenovableCTDesarrollo(pool, interna, reloj)
	if err != nil {
		t.Fatal(err)
	}
	consumirInterna := func() {
		t.Helper()
		if _, err := fuente.instantanea(ctx); err != nil {
			t.Fatalf("consumo del lector interno: %v", err)
		}
		adoptada, err := renovarConfiguracionConfianzaCTDesarrollo(ctx, pool, interna, ahora)
		if err != nil || adoptada.configuracionRef != interna.configuracionRef || adoptada.configuracionHuella != interna.configuracionHuella {
			t.Fatalf("renovación interna cambió tras publicación externa: %v", err)
		}
	}
	consumirInterna()
	huellaInterna := instantaneaInternaClonPrueba(t, ctx, pool)
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	base, err := prepararBasePropiaPortalExterno(cfg.DevelopmentMaterialDir, "", ahora)
	if err != nil {
		t.Fatal(err)
	}
	defer base.borrarCopiasEfimeras()
	materiales := consumidoresCompletosExternoPrueba(t, base)
	defer func() {
		for _, ms := range materiales {
			borrarMaterialesV3PortalExterno(ms)
		}
	}()
	propuesta, err := prepararPublicacionV3PortalExterno(ctx, pool, materiales, OpcionesPreparacionPortalExterno{})
	if err != nil || !propuesta.PendientePublicacion {
		t.Fatalf("propuesta propia: %v", err)
	}
	o := OpcionesPreparacionPortalExterno{HuellaAprobacionSHA256: propuesta.HuellaAprobacionSHA256, PreimagenSHA256: propuesta.PreimagenSHA256}
	if _, err := prepararPublicacionV3PortalExterno(ctx, pool, materiales, o); err != nil {
		t.Fatalf("publicación propia: %v", err)
	}
	inv, err := inventarioDesdeMateriales(materiales)
	if err != nil {
		t.Fatal(err)
	}
	comprobarPreflightExternoClonPrueba(t, ctx, pool, inv, true)
	consumirInterna()
	if instantaneaInternaClonPrueba(t, ctx, pool) != huellaInterna {
		t.Fatal("publicación externa alteró punteros, asociaciones, checkpoint o recibos internos")
	}
	// Reintento con la aprobación y preimagen originales, después del COMMIT.
	if _, err := prepararPublicacionV3PortalExterno(ctx, pool, materiales, o); err != nil {
		t.Fatalf("recuperación con preimagen original: %v", err)
	}
	if err := escribirMaterialV3PortalExterno(cfg.DevelopmentMaterialDir, materiales); err != nil {
		t.Fatal(err)
	}
	rotada, err := prepararBasePropiaPortalExterno(cfg.DevelopmentMaterialDir, base.spkiHuella, ahora)
	if err != nil {
		t.Fatal(err)
	}
	defer rotada.borrarCopiasEfimeras()
	nuevos := consumidoresCompletosExternoPrueba(t, rotada)
	defer func() {
		for _, ms := range nuevos {
			borrarMaterialesV3PortalExterno(ms)
		}
	}()
	opcionesRotacion := OpcionesPreparacionPortalExterno{RotarRaizPreimagenSHA256: base.spkiHuella}
	r, err := prepararPublicacionV3PortalExterno(ctx, pool, nuevos, opcionesRotacion)
	if err != nil || !r.PendientePublicacion {
		t.Fatalf("propuesta de rotación: %v", err)
	}
	opcionesRotacion.HuellaAprobacionSHA256, opcionesRotacion.PreimagenSHA256 = r.HuellaAprobacionSHA256, r.PreimagenSHA256
	if _, err := prepararPublicacionV3PortalExterno(ctx, pool, nuevos, opcionesRotacion); err != nil {
		t.Fatalf("rotación aprobada: %v", err)
	}
	nuevaInv, err := inventarioDesdeMateriales(nuevos)
	if err != nil {
		t.Fatal(err)
	}
	comprobarPreflightExternoClonPrueba(t, ctx, pool, nuevaInv, true)
	comprobarPreflightExternoClonPrueba(t, ctx, pool, inv, false)
	consumirInterna()
	if instantaneaInternaClonPrueba(t, ctx, pool) != huellaInterna {
		t.Fatal("rotación externa alteró historia o gobierno internos")
	}
	cruzada := nuevaInv
	cruzada.Raiz.ClaveID, cruzada.Raiz.Version, cruzada.Raiz.SPKISHA256 = interna.claveID, interna.claveVersion, interna.spkiHuella
	comprobarPreflightExternoClonPrueba(t, ctx, pool, cruzada, false)
}

func consumidoresCompletosExternoPrueba(t *testing.T, base materialAtestacionContratacionTemporalDesarrollo) map[string][]materialAtestacionContratacionTemporalDesarrollo {
	t.Helper()
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptoresMaterialPortalExternoV3())
	if err != nil {
		t.Fatal(err)
	}
	ms := map[string][]materialAtestacionContratacionTemporalDesarrollo{}
	for _, consumidor := range consumidoresPortalExternoV3 {
		for _, audiencia := range audienciasConsumidorPortalExternoV3(consumidor) {
			d, _ := catalogo.descriptorPara(audiencia)
			m, err := derivarMaterialConsumidorV3Desarrollo(base, d)
			if err != nil {
				t.Fatal(err)
			}
			m.privada = append(m.privada[:0:0], base.privada...)
			m.spki = append([]byte(nil), base.spki...)
			ms[consumidor] = append(ms[consumidor], m)
		}
	}
	return ms
}

func materialPublicoInternoClonPrueba(t *testing.T, ctx context.Context, pool *pgxpool.Pool) materialAtestacionContratacionTemporalDesarrollo {
	t.Helper()
	var m materialAtestacionContratacionTemporalDesarrollo
	err := pool.QueryRow(ctx, `SELECT c.revision,c.secuencia,c.huella_configuracion_sha256,c.publicada_en,c.expira_en,r.clave_id,r.version,r.clave_publica_spki,r.huella_spki_sha256,r.valida_desde,r.valida_hasta
 FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version c ON c.revision=p.configuracion_revision JOIN vec_autorizacion_atestada_v3.configuracion_raiz cr ON cr.configuracion_revision=c.revision JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r ON (r.clave_id,r.version)=(cr.raiz_clave_id,cr.raiz_version) WHERE p.orden=(SELECT max(orden) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual)`).Scan(&m.configuracionRef, &m.configuracionOrden, &m.configuracionHuella, &m.publicadaEn, &m.expiraEn, &m.claveID, &m.claveVersion, &m.spki, &m.spkiHuella, &m.validaDesde, &m.validaHasta)
	if err != nil {
		t.Fatal("publicación interna no disponible")
	}
	publica, err := x509.ParsePKIXPublicKey(m.spki)
	if err != nil {
		t.Fatal("raíz interna no disponible")
	}
	clave, ok := publica.(ed25519.PublicKey)
	if !ok {
		t.Fatal("suite interna no disponible")
	}
	m.publicadaEn, m.expiraEn, m.validaDesde, m.validaHasta = m.publicadaEn.UTC(), m.expiraEn.UTC(), m.validaDesde.UTC(), m.validaHasta.UTC()
	m.raiz, err = confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(m.claveID, m.claveVersion, clave, audienciaAtestacionContratacionTemporalDesarrollo, confianza.EstadoClaveAtestacionAutorizacionV3Activa, m.validaDesde, m.validaHasta, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	m.configuracion, err = confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(m.configuracionRef, m.configuracionOrden, m.publicadaEn, m.expiraEn, m.raiz)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func instantaneaInternaClonPrueba(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()
	var s string
	err := pool.QueryRow(ctx, `SELECT jsonb_build_object(
 'punteros',(SELECT jsonb_agg(to_jsonb(p) ORDER BY p.orden) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p),
 'asociaciones',(SELECT jsonb_agg(to_jsonb(cr) ORDER BY cr.configuracion_revision,cr.raiz_clave_id) FROM vec_autorizacion_atestada_v3.configuracion_raiz cr WHERE EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p WHERE p.configuracion_revision=cr.configuracion_revision)),
 'checkpoint',(SELECT jsonb_agg(to_jsonb(cp)) FROM vec_autorizacion_atestada_v3.checkpoint_gobierno cp),
 'recibos',(SELECT jsonb_agg(to_jsonb(r) ORDER BY r.decision_ref) FROM vec_autorizacion_atestada_v3.consumo_decision_v3 r))::text`).Scan(&s)
	if err != nil {
		t.Fatal("instantánea interna no disponible")
	}
	return s
}

func comprobarPreflightExternoClonPrueba(t *testing.T, ctx context.Context, pool *pgxpool.Pool, inv inventarioV3PortalExterno, esperado bool) {
	t.Helper()
	material, err := materialJSONV3PortalExterno(inv, "usuarios_preferencias", inv.Configuracion)
	if err != nil {
		t.Fatal(err)
	}
	defer clear(material)
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Fatal("preflight no disponible")
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL SESSION AUTHORIZATION vec_externo_preflight_v3_desarrollo`); err != nil {
		t.Fatal("identidad nominal de preflight no disponible")
	}
	var aceptado bool
	err = tx.QueryRow(ctx, `SELECT vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1('usuarios_preferencias',$1::jsonb)`, string(material)).Scan(&aceptado)
	if esperado && (err != nil || !aceptado) {
		t.Fatalf("material propio rechazado: %v", err)
	}
	if !esperado && err == nil && aceptado {
		t.Fatal("material cruzado o sustituido admitido")
	}
	if esperado {
		var respuesta string
		if err := tx.QueryRow(ctx, `SELECT vec_autorizacion_atestada_v3.leer_configuracion_externa_v1('usuarios_preferencias',$1::jsonb)::text`, string(material)).Scan(&respuesta); err != nil {
			t.Fatalf("configuración externa no disponible: %v", err)
		}
		var leida struct {
			Revision string `json:"revision"`
		}
		if json.Unmarshal([]byte(respuesta), &leida) != nil || leida.Revision != inv.Configuracion.Revision {
			t.Fatal("el preflight no consumió su publicación propia")
		}
	}
}
