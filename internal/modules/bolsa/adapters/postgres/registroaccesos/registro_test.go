package registroaccesos

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	registroaplicacion "vec-diputacion-granada/internal/modules/bolsa/application/registroaccesos"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type poolNoUsable struct{}

func (poolNoUsable) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	panic("AppendAudit no debe abrir una transacción sin capacidad VEC")
}

func TestAppendAuditComunSiempreFallaCerrado(t *testing.T) {
	t.Parallel()
	registro := &RegistroPostgreSQL{
		pool:  poolNoUsable{},
		ahora: func() time.Time { return time.Now().UTC() },
	}
	resultado, err := registro.AppendAudit(
		context.Background(), entradaAdaptadorValida(),
	)
	if !reflect.DeepEqual(resultado, vecdomain.AuditEntry{}) ||
		!errors.Is(err, vecdomain.ErrPermissionDenied) {
		t.Fatalf("AppendAudit no fallo cerrado: resultado=%v error=%v", resultado, err)
	}
}

func TestListAuditSiempreFallaCerrado(t *testing.T) {
	t.Parallel()
	var registro *RegistroPostgreSQL
	resultado, err := registro.ListAudit(context.Background(), "cualquier-recurso")
	if resultado != nil || !errors.Is(err, vecdomain.ErrPermissionDenied) {
		t.Fatalf("ListAudit no fallo cerrado: resultado=%v error=%v", resultado, err)
	}
}

func TestSerializacionEntradaEsCerradaYNoAceptaCamposDeServidor(t *testing.T) {
	t.Parallel()
	entrada := entradaAdaptadorValida()
	contenido, err := serializarEntrada(entrada)
	if err != nil || !strings.Contains(string(contenido), `"actor_id"`) {
		t.Fatalf("serializacion valida: %s, %v", contenido, err)
	}
	entrada.Signature = strings.Repeat("a", 64)
	if _, err := serializarEntrada(entrada); !errors.Is(
		err, registroaplicacion.ErrRegistroAccesosInvalido,
	) {
		t.Fatalf("firma cliente aceptada: %v", err)
	}
}

func TestDecodificacionRechazaCamposDesconocidosYCola(t *testing.T) {
	t.Parallel()
	for _, contenido := range [][]byte{
		[]byte(`{"desconocido":1}`),
		[]byte(`{} {}`),
	} {
		var destino paginaJSON
		if err := decodificarCerrado(contenido, &destino); err == nil {
			t.Fatalf("respuesta abierta aceptada: %s", contenido)
		}
	}
}

func entradaAdaptadorValida() vecdomain.AuditEntry {
	return vecdomain.AuditEntry{
		ActorID:    "hmac-sha256:bolsa_accesos_v1:" + strings.Repeat("a", 64),
		Purpose:    "tramitacion",
		Action:     "expediente.leer",
		ModuleID:   "vec.module.bolsa",
		SubjectRef: "expediente:sha256:" + strings.Repeat("b", 64),
		Result:     "permitido",
		CorrelationRef: "correlacion:sha256:" +
			strings.Repeat("c", 64),
		OccurredAt: time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC),
	}
}
