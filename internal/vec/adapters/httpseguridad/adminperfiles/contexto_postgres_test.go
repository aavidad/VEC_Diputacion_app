package adminperfiles

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type filaContextoFalsa struct{}

func (filaContextoFalsa) Scan(...any) error { return errors.New("fila sintética ausente") }

type txContextoFalsa struct{ pgx.Tx }

func (txContextoFalsa) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

type filaContextoMaterial struct{ dato []byte }

func (f filaContextoMaterial) Scan(destinos ...any) error {
	*destinos[0].(*[]byte) = append([]byte(nil), f.dato...)
	return nil
}

type poolContextoAmbiguo struct {
	actor       domain.ResultadoContextoActorRegistradoV2
	vinculo     VinculoSesionADMIN
	ahora       time.Time
	opciones    []pgx.TxOptions
	consultas   []string
	argumentos  [][]any
	eventoBruto []byte
	commits     int
	rollbacks   int
}

func (p *poolContextoAmbiguo) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.opciones = append(p.opciones, o)
	return &txContextoAmbiguo{pool: p}, nil
}

func (*poolContextoAmbiguo) QueryRow(context.Context, string, ...any) pgx.Row {
	return filaContextoFalsa{}
}

type txContextoAmbiguo struct {
	pgx.Tx
	pool *poolContextoAmbiguo
}

func (*txContextoAmbiguo) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (t *txContextoAmbiguo) QueryRow(_ context.Context, consulta string, args ...any) pgx.Row {
	p := t.pool
	p.consultas = append(p.consultas, consulta)
	p.argumentos = append(p.argumentos, append([]any(nil), args...))
	op, recibo, evento, correlacion := args[0].(string), args[1].(string), args[12].(string), args[13].(string)
	v := p.vinculo
	e := map[string]any{"tipo_registro": "contexto_admin_pre_v2", "evento_ref": evento,
		"operador_login": "login_contexto", "actor_ref": v.PersonaRef,
		"perfil_activo_ref": v.PerfilActivoRef, "accion": "registrar_contexto_admin", "recurso_ref": op,
		"resultado": "permitido", "motivo_ref": "contexto_admin_pre_v2_permitido", "proceso": "vec_admin",
		"canal": "administracion_privilegiada", "finalidad_ref": "establecer_contexto_admin",
		"correlacion_ref": correlacion, "fuente_ref": v.FuenteRef, "fuente_sha256": v.FuenteSHA256}
	a := map[string]any{"auditoria_ref": "aud_v3_ap2_" + strings.TrimPrefix(evento, "evento_"), "secuencia": 1,
		"huella_sha256": strings.Repeat("a", 64), "correlacion_ref": correlacion, "registrada_en": p.ahora}
	r := p.actor
	c := map[string]any{"operacion_ref": op, "registro_contexto_ref": recibo,
		"representacion_canonica_base64":         base64.StdEncoding.EncodeToString(r.RepresentacionCanonica),
		"huella_sha256":                          r.HuellaSHA256,
		"manifiesto_procedencia_canonico_base64": base64.StdEncoding.EncodeToString(r.ManifiestoProcedenciaCanonico),
		"manifiesto_procedencia_huella_sha256":   r.ManifiestoProcedenciaHuellaSHA256,
		"autoridad_efectiva":                     r.AutoridadEfectiva, "resuelto_en": r.ResueltoEnAutoritativo}
	bruto, _ := json.Marshal(map[string]any{"estado": "permitido", "motivo_ref": "contexto_admin_pre_v2_permitido",
		"evento": e, "acuse": a, "contexto": c})
	p.eventoBruto, _ = json.Marshal(e)
	return filaContextoMaterial{dato: bruto}
}

func (t *txContextoAmbiguo) Commit(context.Context) error {
	t.pool.commits++
	if t.pool.commits == 1 {
		return errors.New("commit sin confirmación")
	}
	return nil
}

func (t *txContextoAmbiguo) Rollback(context.Context) error {
	t.pool.rollbacks++
	return nil
}

func TestContextoADMINCommitInciertoConsultaMismoEventoSinRepetirRegistro(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	actor, _, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora,
		"per_"+strings.Repeat("a", 22), "prf_"+strings.Repeat("b", 22),
		domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	v := vinculoContextoADMINPrueba(ahora)
	v.PersonaRef, v.CuentaRef, v.PerfilActivoRef = actor.Contexto.PersonaRef,
		actor.Contexto.Instantanea.CuentaRef, actor.Contexto.PerfilActivoRef
	s := ports.SolicitudResolucionRegistroContextoActorV2{OperacionRef: "oca_" + strings.Repeat("c", 32), SolicitadoEn: ahora,
		Contexto: domain.SolicitudContextoActor{Cuenta: domain.CuentaAutenticadaContextoActor{
			CuentaRef: v.CuentaRef, Metodo: domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh},
			PerfilActivoRef: v.PerfilActivoRef}}
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, err = ContextoConVinculoSesionADMIN(ctx, v)
	if err != nil {
		t.Fatal(err)
	}
	pool := &poolContextoAmbiguo{actor: actor, vinculo: v, ahora: ahora}
	r := &resolutorContexto{base: &PostgreSQL{pool: pool, reloj: relojPrueba{ahora: ahora}}, proceso: "vec_admin", login: "login_contexto"}
	c, err := r.ResolverYRegistrarContextoActorV2(ctx, s)
	if err != nil || c.OperacionRef != s.OperacionRef || len(pool.opciones) != 2 ||
		pool.opciones[0].IsoLevel != pgx.Serializable || pool.opciones[1].IsoLevel != pgx.ReadCommitted ||
		pool.commits != 2 || len(pool.consultas) != 2 || pool.consultas[0] != registrarContexto ||
		pool.consultas[1] != recuperarContexto || len(pool.argumentos[0]) != 15 || len(pool.argumentos[1]) != 16 {
		t.Fatal("COMMIT incierto no recuperó una sola operación")
	}
	for i := 0; i < 15; i++ {
		if pool.argumentos[0][i] != pool.argumentos[1][i] {
			t.Fatal("recuperación sustituyó una coordenada original")
		}
	}
	if pool.argumentos[1][15] != string(pool.eventoBruto) {
		t.Fatal("recuperación no usó el evento15 original")
	}
}
func (txContextoFalsa) QueryRow(context.Context, string, ...any) pgx.Row { return filaContextoFalsa{} }
func (txContextoFalsa) Rollback(context.Context) error                   { return nil }

type poolContextoFalso struct{ opciones []pgx.TxOptions }

func (p *poolContextoFalso) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	p.opciones = append(p.opciones, opciones)
	return txContextoFalsa{}, nil
}
func (*poolContextoFalso) QueryRow(context.Context, string, ...any) pgx.Row {
	return filaContextoFalsa{}
}

func TestContextoADMINSeparaAislamientoDeRegistroYRecuperacion(t *testing.T) {
	pool := &poolContextoFalso{}
	resolver := &resolutorContexto{base: &PostgreSQL{pool: pool}}
	for _, consulta := range []string{registrarContexto, recuperarContexto} {
		_, _, _, err := resolver.ejecutar(context.Background(), consulta, nil,
			ports.SolicitudResolucionRegistroContextoActorV2{}, VinculoSesionADMIN{}, "", "", "")
		if err == nil {
			t.Fatal("la fila sintética ausente se aceptó")
		}
	}
	if len(pool.opciones) != 2 || pool.opciones[0].IsoLevel != pgx.Serializable ||
		pool.opciones[1].IsoLevel != pgx.ReadCommitted ||
		pool.opciones[0].AccessMode != pgx.ReadWrite || pool.opciones[1].AccessMode != pgx.ReadWrite {
		t.Fatalf("aislamiento de registrar/reconciliar incorrecto: %+v", pool.opciones)
	}
}
