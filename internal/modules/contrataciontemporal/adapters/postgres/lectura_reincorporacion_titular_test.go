package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

func TestDecodificarLecturaReincorporacionTitularMinimizaYVincula(t *testing.T) {
	m := ports.MaterialReincorporacionTitular{OrganizacionRef: "org:test", ExpedienteRef: "exp:test", RelacionRef: "rel:test",
		ActorRef: "actor:test", PerfilRef: "perfil:test", VersionEsperada: 4,
		FechaEfectiva: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC), DocumentoRef: "doc:test",
		DocumentoSHA256: strings.Repeat("a", 64), ClaveIdempotencia: "11111111-1111-4111-8111-111111111111"}
	if !m.Valido() {
		t.Fatal("fixture de material inválida")
	}
	base := map[string]any{"esquema": esquemaLecturaReincorporacionTitular, "resultado": ports.ResultadoAntecedenteCoincide,
		"expediente_ref": m.ExpedienteRef, "cese_evento_ref": "evento:cese", "cese_recibo_ref": "recibo:cese",
		"lectura_ref": "lectura:" + strings.Repeat("c", 64), "auditoria_ref": "aud_v3_" + strings.Repeat("d", 32), "consumo_huella_sha256": strings.Repeat("b", 64),
		"registrada_en": "2026-09-28T10:00:00Z"}
	decodificar := func(r map[string]any) error {
		t.Helper()
		b, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		_, err = decodificarLecturaReincorporacionTitular(b, m)
		return err
	}
	if err := decodificar(base); err != nil {
		t.Fatalf("coincidencia válida: %v", err)
	}
	noCoincide := make(map[string]any, len(base))
	for k, v := range base {
		noCoincide[k] = v
	}
	noCoincide["resultado"] = ports.ResultadoAntecedenteNoCoincide
	noCoincide["cese_evento_ref"] = ""
	noCoincide["cese_recibo_ref"] = ""
	if err := decodificar(noCoincide); err != nil {
		t.Fatalf("ausencia minimizada válida: %v", err)
	}
	casos := []struct {
		nombre, clave string
		valor         any
	}{
		{"expediente ajeno", "expediente_ref", "exp:otro"},
		{"estado desconocido", "resultado", "sin_cese"},
		{"huella inválida", "consumo_huella_sha256", strings.Repeat("0", 64)},
		{"lectura vacía", "lectura_ref", ""},
		{"lectura falsificada", "lectura_ref", "lectura:sin-huella"},
		{"auditoría falsificada", "auditoria_ref", "auditoria:test"},
		{"fecha ausente", "registrada_en", ""},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			r := make(map[string]any, len(base))
			for k, v := range base {
				r[k] = v
			}
			r[tc.clave] = tc.valor
			if err := decodificar(r); !errors.Is(err, ports.ErrResultadoSeguimientoNoConfiable) {
				t.Fatalf("debe rechazar respuesta: %v", err)
			}
		})
	}
	if err := decodificar(map[string]any{"esquema": esquemaLecturaReincorporacionTitular}); !errors.Is(err, ports.ErrResultadoSeguimientoNoConfiable) {
		t.Fatalf("respuesta parcial aceptada: %v", err)
	}
	noCoincide["cese_evento_ref"] = "evento:filtrado"
	if err := decodificar(noCoincide); !errors.Is(err, ports.ErrResultadoSeguimientoNoConfiable) {
		t.Fatalf("ausencia reveló antecedente: %v", err)
	}
	base["otro_dato"] = "secreto"
	if err := decodificar(base); !errors.Is(err, ports.ErrResultadoSeguimientoNoConfiable) {
		t.Fatalf("campo inesperado aceptado: %v", err)
	}
}

func TestPrepararReincorporacionConservaConflictoYRevierteRespuestaMalformada(t *testing.T) {
	m := ports.MaterialReincorporacionTitular{OrganizacionRef: "org:test", ExpedienteRef: "exp:test", RelacionRef: "rel:test",
		ActorRef: "actor:test", PerfilRef: "perfil:test", VersionEsperada: 4,
		FechaEfectiva: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC), DocumentoRef: "doc:test",
		DocumentoSHA256: strings.Repeat("a", 64), ClaveIdempotencia: "11111111-1111-4111-8111-111111111111"}
	lectura := ports.AntecedenteReincorporacionTitular{Resultado: ports.ResultadoAntecedenteCoincide, ExpedienteRef: m.ExpedienteRef,
		CeseEventoRef: "evento:cese", CeseReciboRef: "recibo:cese", LecturaRef: "lectura:" + strings.Repeat("c", 64),
		AuditoriaRef: "aud_v3_" + strings.Repeat("d", 32), ConsumoHuellaSHA256: strings.Repeat("b", 64),
		RegistradaEn: time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)}
	ambito, err := ports.NuevaColeccionSellosHMAC("hmac-sha256:"+ports.DominioAmbitoReincorporacionTitular+"/v1:"+strings.Repeat("a", 64), nil)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := ports.NuevaColeccionSellosHMAC("hmac-sha256:"+ports.DominioHuellaReincorporacionTitular+"/v1:"+strings.Repeat("b", 64), nil)
	if err != nil {
		t.Fatal(err)
	}
	sellos := ports.SellosOperacionSeguimiento{Ambitos: ambito, Huellas: huella}
	refs := ports.ReferenciasEfectoSeguimiento{ReservaRef: "reserva:test", ReciboRef: "recibo:test", EventoRef: "evento:test"}
	for _, tc := range []struct {
		nombre, respuesta string
		esperado          error
		commits           int
	}{
		{"conflicto confiable", `{"esquema":"vec.contratacion-temporal.resultado-reincorporacion-titular.v1","resultado":"version_en_conflicto"}`, domain.ErrVersionEnConflicto, 1},
		{"conflicto contradictorio", `{"esquema":"vec.contratacion-temporal.resultado-reincorporacion-titular.v1","resultado":"version_en_conflicto","material":{}}`, ports.ErrResultadoSeguimientoNoConfiable, 0},
		{"respuesta malformada", `{"esquema":"vec.contratacion-temporal.resultado-reincorporacion-titular.v1","resultado":"preparada"}`, ports.ErrResultadoSeguimientoNoConfiable, 0},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			tx := &txLecturaReincorporacionPrueba{respuesta: []byte(tc.respuesta)}
			r := &RepositorioReincorporacionTitularPostgreSQL{pool: &poolLecturaReincorporacionPrueba{tx: tx}}
			_, err := r.PrepararReincorporacionTitular(context.Background(), m, lectura, sellos, refs)
			if !errors.Is(err, tc.esperado) || tx.commits != tc.commits {
				t.Fatalf("error=%v, commits=%d; esperado error=%v, commits=%d", err, tx.commits, tc.esperado, tc.commits)
			}
		})
	}
}

type txLecturaReincorporacionPrueba struct {
	pgx.Tx
	consulta           string
	args               []any
	respuesta          []byte
	commits, rollbacks int
}

func (t *txLecturaReincorporacionPrueba) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (t *txLecturaReincorporacionPrueba) QueryRow(_ context.Context, consulta string, args ...any) pgx.Row {
	t.consulta, t.args = consulta, append([]any(nil), args...)
	t.args[0] = append([]byte(nil), args[0].([]byte)...)
	return filaLecturaReincorporacionPrueba{t.respuesta}
}

func (t *txLecturaReincorporacionPrueba) Commit(context.Context) error {
	t.commits++
	return nil
}

func (t *txLecturaReincorporacionPrueba) Rollback(context.Context) error {
	t.rollbacks++
	return nil
}

type filaLecturaReincorporacionPrueba struct{ respuesta []byte }

func (f filaLecturaReincorporacionPrueba) Scan(destinos ...any) error {
	*destinos[0].(*[]byte) = append([]byte(nil), f.respuesta...)
	return nil
}

type poolLecturaReincorporacionPrueba struct {
	tx       *txLecturaReincorporacionPrueba
	opciones pgx.TxOptions
	inicios  int
}

func (p *poolLecturaReincorporacionPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	p.opciones = opciones
	p.inicios++
	return p.tx, nil
}

func TestLecturaReincorporacionTitularConsumeV3EnTransaccionDeEscritura(t *testing.T) {
	m := ports.MaterialReincorporacionTitular{OrganizacionRef: "org:test", ExpedienteRef: "exp:test", RelacionRef: "rel:test",
		ActorRef: "actor:test", PerfilRef: "perfil:test", VersionEsperada: 4,
		FechaEfectiva: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC), DocumentoRef: "doc:test",
		DocumentoSHA256: strings.Repeat("a", 64), ClaveIdempotencia: "11111111-1111-4111-8111-111111111111"}
	recurso := vd.RecursoAutorizable{Referencia: m.ExpedienteRef, ModuloID: ports.ModuloContratacion,
		Tipo:    ports.TipoRecursoLecturaReincorporacionTitular,
		Ambitos: map[string]string{"organizacion_ref": m.OrganizacionRef},
		Atributos: map[string]string{"version_expediente": "4", "relacion_ref": m.RelacionRef,
			"fecha_efectiva": "2026-09-28", "documento_ref": m.DocumentoRef, "documento_sha256": m.DocumentoSHA256}}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	decision, _ := json.Marshal(map[string]string{"decision_ref": "decision:lectura", "accion": string(ports.AccionConsultarAntecedenteReincorporacionTitular),
		"finalidad": ports.FinalidadLecturaReincorporacionTitular, "recurso_ref": m.ExpedienteRef,
		"principal_id": m.ActorRef, "perfil_activo_ref": m.PerfilRef, "contexto_recurso_huella_sha256": huella})
	hs := sha256.Sum256(decision)
	instante := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	resumen, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:lectura", hex.EncodeToString(hs[:]), strings.Repeat("c", 64),
		"contexto:lectura", strings.Repeat("d", 64), string(ports.AccionConsultarAntecedenteReincorporacionTitular),
		m.ExpedienteRef, huella, ports.AudienciaLecturaReincorporacionTitularV1, instante, instante.Add(4*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(make([]byte, 32)).Public())
	if err != nil {
		t.Fatal(err)
	}
	a, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("x", 512)), resumen, decision,
		[]byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), spki)
	if err != nil {
		t.Fatal(err)
	}
	respuesta, _ := json.Marshal(map[string]any{"esquema": esquemaLecturaReincorporacionTitular,
		"resultado": ports.ResultadoAntecedenteNoCoincide, "expediente_ref": m.ExpedienteRef,
		"cese_evento_ref": "", "cese_recibo_ref": "", "lectura_ref": "lectura:" + strings.Repeat("c", 64),
		"auditoria_ref":         "aud_v3_" + strings.Repeat("d", 32),
		"consumo_huella_sha256": strings.Repeat("b", 64), "registrada_en": "2026-09-28T10:00:00Z"})
	tx := &txLecturaReincorporacionPrueba{respuesta: respuesta}
	p := &poolLecturaReincorporacionPrueba{tx: tx}
	l := &LectorAntecedenteReincorporacionTitularPostgreSQL{pool: p}
	r, err := l.LeerAntecedenteReincorporacionTitular(context.Background(), m, a)
	if err != nil || r.Resultado != ports.ResultadoAntecedenteNoCoincide || tx.commits != 1 || p.inicios != 1 ||
		p.opciones.IsoLevel != pgx.Serializable || p.opciones.AccessMode != pgx.ReadWrite {
		t.Fatalf("lectura/auditoría no confirmadas juntas: recibo=%+v err=%v tx=%+v", r, err, p.opciones)
	}
	if tx.consulta != consultaLecturaReincorporacionTitular || len(tx.args) != 11 {
		t.Fatalf("consulta/argumentos inesperados: %q %d", tx.consulta, len(tx.args))
	}
	var material map[string]any
	if json.Unmarshal(tx.args[0].([]byte), &material) != nil || len(material) != 9 || material["clave_idempotencia"] != nil {
		t.Fatalf("material SQL no minimizado: %+v", material)
	}
	m.ActorRef = "actor:otro"
	if _, err := l.LeerAntecedenteReincorporacionTitular(context.Background(), m, a); !errors.Is(err, ports.ErrAutorizacionDenegada) || p.inicios != 1 {
		t.Fatalf("atestación de otro actor alcanzó SQL: %v; inicios=%d", err, p.inicios)
	}
}
