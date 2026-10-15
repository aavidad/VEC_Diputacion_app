package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/config"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
)

type filaArranqueConfianzaPrueba struct {
	valores []any
	err     error
}

func (f filaArranqueConfianzaPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != len(f.valores) {
		return errors.New("cantidad de columnas inesperada")
	}
	for i := range destinos {
		reflect.ValueOf(destinos[i]).Elem().Set(reflect.ValueOf(f.valores[i]))
	}
	return nil
}

type transaccionArranqueConfianzaPrueba struct {
	pgx.Tx
	actual      materialAtestacionContratacionTemporalDesarrollo
	admin       bool
	rechazo     string
	consultas   []string
	inserciones int
}

func (tx *transaccionArranqueConfianzaPrueba) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	tx.consultas = append(tx.consultas, sql)
	m := tx.actual
	fila := func(v ...any) pgx.Row { return filaArranqueConfianzaPrueba{valores: v} }
	switch {
	case strings.Contains(sql, "SELECT (SELECT count(*) FROM vec_autorizacion_atestada_v3.puntero_clave_emision"):
		return fila(int64(1), int64(1))
	case strings.Contains(sql, "SELECT EXISTS (") && strings.Contains(sql, "puntero_clave_emision p"):
		return fila(true)
	case strings.Contains(sql, "SELECT clave_id,version,revision_gobierno"):
		return fila(m.claveHMACID, int64(m.claveHMACVersion), int64(m.claveHMACRevision))
	case strings.Contains(sql, "SELECT clave_id,version"):
		return fila(m.claveID, int64(m.claveVersion))
	case strings.Contains(sql, "SELECT orden FROM vec_autorizacion_atestada_v3.puntero_clave_emision"):
		return fila(int64(m.claveHMACOrden))
	case strings.Contains(sql, "SELECT configuracion_secuencia_minima,raiz_version_minima"):
		if tx.rechazo == "checkpoint" {
			return fila(int64(m.configuracionOrden+1), int64(m.claveVersion))
		}
		return fila(int64(1), int64(1))
	case strings.Contains(sql, "revocacion_raiz rr"):
		if tx.rechazo == "revocada" || tx.rechazo == "raices" {
			return filaArranqueConfianzaPrueba{err: pgx.ErrNoRows}
		}
		return fila(m.configuracionRef, int64(m.configuracionOrden), m.configuracionHuella, m.publicadaEn, m.expiraEn)
	case strings.Contains(sql, "SELECT c.revision,c.secuencia,c.huella_configuracion_sha256,c.publicada_en,c.expira_en"):
		return fila(m.configuracionRef, int64(m.configuracionOrden), m.configuracionHuella, m.publicadaEn, m.expiraEn)
	case strings.Contains(sql, "gobierno_usuarios_detalle->>'configuracion_destino_ref'"):
		return fila(tx.admin)
	case strings.Contains(sql, "SELECT EXISTS (") && strings.Contains(sql, "clave_capacidad_version"):
		return fila(true)
	default:
		return filaArranqueConfianzaPrueba{err: errors.New("consulta inesperada")}
	}
}

func (tx *transaccionArranqueConfianzaPrueba) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	if !strings.Contains(sql, "INSERT INTO vec_autorizacion_atestada_v3.") {
		return pgconn.CommandTag{}, errors.New("escritura inesperada")
	}
	tx.inserciones++
	return pgconn.CommandTag{}, nil
}

func TestArranquePublicadorAdoptaConfianzaADMINPersistida(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileDevelopment)
	t.Setenv(config.EnvAuthMode, config.AuthModeDevelopment)
	t.Setenv(config.EnvDevelopmentGuard, config.DevelopmentGuardAcknowledgement)
	for _, tc := range []struct {
		nombre string
		inicio time.Time
		vida   time.Duration
	}{
		{"24h_fuera_de_medianoche", ahora.Add(-time.Minute), 24 * time.Hour},
		{"60_dias", ahora.Add(-time.Minute), 60 * 24 * time.Hour},
		{"90_dias", ahora.Add(-time.Minute), 90 * 24 * time.Hour},
		{"24h_medianoche", ahora.Truncate(24 * time.Hour), 24 * time.Hour},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			m := materialRenovableCTPrueba(t, ahora)
			admin := m
			admin.publicadaEn, admin.expiraEn = tc.inicio, tc.inicio.Add(tc.vida)
			admin.configuracionOrden = m.configuracionOrden + 1
			admin.configuracionRef += ":r" + numeroDecimal64(admin.configuracionOrden)
			var err error
			admin.configuracion, err = confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(admin.configuracionRef, admin.configuracionOrden, admin.publicadaEn, admin.expiraEn, admin.raiz)
			if err != nil {
				t.Fatal(err)
			}
			admin.configuracionHuella, err = admin.configuracion.HuellaSHA256ParaGobierno()
			if err != nil {
				t.Fatal(err)
			}
			tx := &transaccionArranqueConfianzaPrueba{actual: admin, admin: true}
			for reinicio := range 2 {
				arranque := m
				if err := publicarGobiernoAtestacionCTEnTxDesarrollo(t.Context(), tx, &arranque); err != nil {
					t.Fatalf("arranque %d: %v", reinicio, err)
				}
				if arranque.configuracionRef != admin.configuracionRef || arranque.configuracionOrden != admin.configuracionOrden || arranque.configuracionHuella != admin.configuracionHuella || !arranque.publicadaEn.Equal(admin.publicadaEn) || !arranque.expiraEn.Equal(admin.expiraEn) {
					t.Fatal("publicador sustituyó la revisión ADMIN")
				}
			}
			if tx.inserciones != 12 {
				t.Fatalf("publicador no atravesó la publicación inicial dos veces: %d", tx.inserciones)
			}
			for _, consulta := range tx.consultas {
				if strings.Contains(consulta, "SELECT GREATEST(") {
					t.Fatal("arranque reservó nueva secuencia")
				}
			}
		})
	}
}

func TestArranquePublicadorRechazaConfianzaIncompatible(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileDevelopment)
	t.Setenv(config.EnvAuthMode, config.AuthModeDevelopment)
	t.Setenv(config.EnvDevelopmentGuard, config.DevelopmentGuardAcknowledgement)
	for _, caso := range []struct {
		nombre, rechazo string
		admin           bool
		publicada       time.Time
		duracion        time.Duration
	}{
		{"configuracion_revocada", "revocada", true, ahora.Add(-time.Minute), 90 * 24 * time.Hour},
		{"raices_incompatibles", "raices", true, ahora.Add(-time.Minute), 90 * 24 * time.Hour},
		{"checkpoint_adelantado", "checkpoint", true, ahora.Add(-time.Minute), 90 * 24 * time.Hour},
		{"sin_asiento_admin", "", false, ahora.Add(-time.Minute), 90 * 24 * time.Hour},
		{"admin_caducada", "", true, ahora.Add(-48 * time.Hour), 24 * time.Hour},
		{"admin_24h_utc_caducada", "", true, ahora.Truncate(24 * time.Hour).Add(-48 * time.Hour), 24 * time.Hour},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			m := materialRenovableCTPrueba(t, ahora)
			actual := m
			actual.publicadaEn, actual.expiraEn = caso.publicada, caso.publicada.Add(caso.duracion)
			actual.configuracionOrden++
			actual.configuracionRef += ":r" + numeroDecimal64(actual.configuracionOrden)
			var err error
			actual.configuracion, err = confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(actual.configuracionRef, actual.configuracionOrden, actual.publicadaEn, actual.expiraEn, actual.raiz)
			if err != nil {
				t.Fatal(err)
			}
			actual.configuracionHuella, err = actual.configuracion.HuellaSHA256ParaGobierno()
			if err != nil {
				t.Fatal(err)
			}
			tx := &transaccionArranqueConfianzaPrueba{actual: actual, admin: caso.admin, rechazo: caso.rechazo}
			if err := publicarGobiernoAtestacionCTEnTxDesarrollo(t.Context(), tx, &m); err == nil || tx.inserciones != 0 {
				t.Fatalf("publicó gobierno incompatible: error=%v inserciones=%d", err, tx.inserciones)
			}
		})
	}
}
