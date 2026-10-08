package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const preflightFronteraIdentidadPrueba = `{
"operador_login":"vec_identidad_pre_f1_prueba",
"proceso":"vec-server","canal":"identidad_http_interno_preacreditacion",
"superficie":"interna_corporativa",
"rutas":[{"metodo_esperado":"GET","ruta":"/api/vec/session"},{"metodo_esperado":"POST","ruta":"/api/vec/session/start"}],
"codigos":[{"motivo_ref":"certificado_requerido","resultado":"denegado"},{"motivo_ref":"autenticacion_requerida","resultado":"denegado"},{"motivo_ref":"acceso_denegado","resultado":"denegado"},{"motivo_ref":"metodo_no_permitido","resultado":"denegado"},{"motivo_ref":"recurso_no_encontrado","resultado":"denegado"},{"motivo_ref":"solicitud_invalida","resultado":"denegado"},{"motivo_ref":"servicio_no_disponible","resultado":"error"},{"motivo_ref":"respuesta_incompatible","resultado":"error"}]}`

type poolFronteraIdentidadPrueba struct {
	txs     []*txFronteraIdentidadPrueba
	inicios int
}

func (p *poolFronteraIdentidadPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	if o.IsoLevel != pgx.Serializable || o.AccessMode != pgx.ReadWrite || len(p.txs) == 0 {
		return nil, errors.New("transaccion no acreditada")
	}
	tx := p.txs[0]
	p.txs = p.txs[1:]
	p.inicios++
	return tx, nil
}

type txFronteraIdentidadPrueba struct {
	pgx.Tx
	preflight                           []byte
	escrito                             eventoFronteraIdentidadTecnica
	consulta                            string
	consultaErr                         error
	commitErr                           error
	commitHook                          func()
	materialAjeno                       bool
	fechaFutura                         bool
	commits, rollbacks, configuraciones int
}

func (t *txFronteraIdentidadPrueba) Exec(_ context.Context, q string, _ ...any) (pgconn.CommandTag, error) {
	if !strings.Contains(q, "set_config('search_path'") || !strings.Contains(q, "set_config('timezone', 'UTC'") {
		return pgconn.CommandTag{}, errors.New("configuracion transaccional ausente")
	}
	t.configuraciones++
	return pgconn.CommandTag{}, nil
}

func (t *txFronteraIdentidadPrueba) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	t.consulta = q
	if t.consultaErr != nil {
		return filaFronteraIdentidadPrueba{err: t.consultaErr}
	}
	if strings.Contains(q, "acreditar_frontera_identidad_tecnica_v1") {
		if len(args) != 0 {
			return filaFronteraIdentidadPrueba{err: errors.New("preflight con argumentos")}
		}
		return filaFronteraIdentidadPrueba{preflight: t.preflight}
	}
	if !strings.Contains(q, "registrar_frontera_identidad_tecnica_v1") || len(args) != 1 {
		return filaFronteraIdentidadPrueba{err: errors.New("funcion o argumentos ajenos")}
	}
	material, ok := args[0].([]byte)
	if !ok || json.Unmarshal(material, &t.escrito) != nil {
		return filaFronteraIdentidadPrueba{err: errors.New("JSON invalido")}
	}
	sha := digestFronteraIdentidadIndependiente(t.escrito)
	if t.materialAjeno {
		sha = strings.Repeat("a", 64)
	}
	fecha := time.Now().UTC().Truncate(time.Microsecond)
	if t.fechaFutura {
		fecha = fecha.Add(time.Hour)
	}
	return filaFronteraIdentidadPrueba{acuse: ports.AcuseFronteraIdentidadTecnica{
		AuditoriaRef: "aud_v3_pit_" + strings.TrimPrefix(t.escrito.EventoRef, "evento_"),
		Secuencia:    41, MaterialSHA256: sha, CorrelacionRef: t.escrito.CorrelacionRef,
		RegistradaEn: fecha,
	}}
}

func (t *txFronteraIdentidadPrueba) Commit(context.Context) error {
	t.commits++
	if t.commitHook != nil {
		t.commitHook()
	}
	return t.commitErr
}
func (t *txFronteraIdentidadPrueba) Rollback(context.Context) error { t.rollbacks++; return nil }

type filaFronteraIdentidadPrueba struct {
	preflight []byte
	acuse     ports.AcuseFronteraIdentidadTecnica
	err       error
}

func (f filaFronteraIdentidadPrueba) Scan(d ...any) error {
	if f.err != nil {
		return f.err
	}
	switch len(d) {
	case 1:
		*d[0].(*[]byte) = append([]byte(nil), f.preflight...)
	case 5:
		*d[0].(*string) = f.acuse.AuditoriaRef
		*d[1].(*string) = fmt.Sprint(f.acuse.Secuencia)
		*d[2].(*string) = f.acuse.MaterialSHA256
		*d[3].(*string) = f.acuse.CorrelacionRef
		*d[4].(*time.Time) = f.acuse.RegistradaEn
	default:
		return errors.New("numero destinos invalido")
	}
	return nil
}

// Vector independiente: encuadra cada valor sin llamar al helper productivo.
func digestFronteraIdentidadIndependiente(e eventoFronteraIdentidadTecnica) string {
	h := sha256.New()
	for _, s := range []string{
		"vec.auditoria.pre-identidad-tecnica.v1", e.TipoRegistro, e.EventoRef,
		e.OperadorLogin, e.Fase, e.MetodoEsperado, e.Ruta, e.Accion,
		e.RecursoRef, e.Resultado, e.MotivoRef, e.Proceso, e.Canal,
		e.Superficie, e.FinalidadRef, e.CorrelacionRef,
	} {
		_, _ = fmt.Fprintf(h, "%d:%s\n", len([]byte(s)), s)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func entornoFronteraIdentidadPrueba(t *testing.T) (*RegistradorFronteraIdentidadTecnicaPostgreSQL, *poolFronteraIdentidadPrueba, *txFronteraIdentidadPrueba) {
	t.Helper()
	preflight := &txFronteraIdentidadPrueba{preflight: []byte(preflightFronteraIdentidadPrueba)}
	registro := &txFronteraIdentidadPrueba{}
	pool := &poolFronteraIdentidadPrueba{txs: []*txFronteraIdentidadPrueba{preflight, registro}}
	r, err := nuevoRegistradorFronteraIdentidadTecnicaPostgreSQL(context.Background(), pool)
	if err != nil || preflight.commits != 1 || preflight.configuraciones != 1 || pool.inicios != 1 {
		t.Fatalf("preflight no fijo autoridad: %v", err)
	}
	return r, pool, registro
}

func TestFronteraIdentidadTecnicaConfirmaMaterialExactoSinActor(t *testing.T) {
	r, pool, tx := entornoFronteraIdentidadPrueba(t)
	orden, err := ports.NuevaOrdenFronteraIdentidadTecnica(domain.MetodoInicioSesionGET,
		domain.RutaSesionActual, ports.MotivoFronteraIdentidadCertificadoRequerido)
	if err != nil {
		t.Fatal(err)
	}
	acuse, err := r.RegistrarRechazoInicioSesion(context.Background(), orden)
	if err != nil || acuse.MaterialSHA256 == "" || acuse.CorrelacionRef != orden.CorrelacionRef() ||
		tx.commits != 1 || tx.configuraciones != 1 || pool.inicios != 2 {
		t.Fatalf("registro tecnico no confirmado: %v", err)
	}
	if tx.escrito.EventoRef == "" || tx.escrito.OperadorLogin != "vec_identidad_pre_f1_prueba" ||
		tx.escrito.MetodoEsperado != "GET" || tx.escrito.Ruta != domain.RutaSesionActual ||
		tx.escrito.Resultado != "denegado" || tx.escrito.RecursoRef == "" ||
		tx.escrito.CorrelacionRef != orden.CorrelacionRef() ||
		acuse.MaterialSHA256 != digestFronteraIdentidadIndependiente(tx.escrito) {
		t.Fatal("evento no coincide con contrato técnico AD222")
	}
	b, _ := json.Marshal(tx.escrito)
	var campos map[string]json.RawMessage
	if json.Unmarshal(b, &campos) != nil || len(campos) != 15 {
		t.Fatal("JSON AD222 no tiene exactamente 15 campos")
	}
	for _, privado := range []string{"actor_ref", "perfil_ref", "decision_ref", "cuenta_ref", "certificado", "sujeto", "ip", "error"} {
		if _, ok := campos[privado]; ok {
			t.Fatalf("campo privado %s en auditoria", privado)
		}
	}
}

func TestFronteraIdentidadTecnicaDeniegaAcuseFalsoYCommitIncierto(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		mutar  func(*txFronteraIdentidadPrueba)
		causa  error
	}{
		{"material", func(tx *txFronteraIdentidadPrueba) { tx.materialAjeno = true }, ports.ErrAcuseFronteraIdentidadTecnicaInvalido},
		{"fecha_futura", func(tx *txFronteraIdentidadPrueba) { tx.fechaFutura = true }, ports.ErrAcuseFronteraIdentidadTecnicaInvalido},
		{"commit_incierto", func(tx *txFronteraIdentidadPrueba) {
			tx.commitErr = errors.New("respuesta SQL perdida con dato privado")
		}, ports.ErrFronteraIdentidadTecnicaCommitIncierto},
		{"error_sql", func(tx *txFronteraIdentidadPrueba) { tx.consultaErr = errors.New("dato privado del servidor") }, ports.ErrFronteraIdentidadTecnicaNoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			r, _, tx := entornoFronteraIdentidadPrueba(t)
			caso.mutar(tx)
			orden, err := ports.NuevaOrdenFronteraIdentidadTecnica(domain.MetodoInicioSesionPOST,
				domain.RutaInicioSesion, ports.MotivoFronteraIdentidadRespuestaIncompatible)
			if err != nil {
				t.Fatal(err)
			}
			acuse, err := r.RegistrarRechazoInicioSesion(context.Background(), orden)
			if !errors.Is(err, caso.causa) || acuse != (ports.AcuseFronteraIdentidadTecnica{}) ||
				strings.Contains(fmt.Sprint(err), "privado") {
				t.Fatalf("fallo no cerro START: %v", err)
			}
			if caso.nombre != "commit_incierto" && tx.commits != 0 {
				t.Fatal("acuse falso confirmó COMMIT")
			}
		})
	}
}

func TestFronteraIdentidadTecnicaPreflightRechazaCatalogoAlterado(t *testing.T) {
	for _, caso := range []struct{ anterior, nuevo string }{
		{`"superficie":"interna_corporativa"`, `"superficie":"externa_personal"`},
		{`"metodo_esperado":"GET"`, `"metodo_esperado":"DELETE"`},
		{`"motivo_ref":"certificado_requerido"`, `"motivo_ref":"codigo_de_cliente"`},
		{`"canal":"identidad_http_interno_preacreditacion"`, `"canal":"canal_del_cliente"`},
	} {
		preflight := &txFronteraIdentidadPrueba{preflight: []byte(strings.Replace(preflightFronteraIdentidadPrueba, caso.anterior, caso.nuevo, 1))}
		pool := &poolFronteraIdentidadPrueba{txs: []*txFronteraIdentidadPrueba{preflight}}
		if r, err := nuevoRegistradorFronteraIdentidadTecnicaPostgreSQL(context.Background(), pool); r != nil ||
			!errors.Is(err, ports.ErrFronteraIdentidadTecnicaNoDisponible) || preflight.commits != 0 {
			t.Fatalf("preflight adulterado admitido: %v", err)
		}
	}
}

func TestFronteraIdentidadTecnicaPreflightAceptaLoginTecnicoSQL(t *testing.T) {
	preflight := &txFronteraIdentidadPrueba{preflight: []byte(strings.Replace(
		preflightFronteraIdentidadPrueba,
		`"operador_login":"vec_identidad_pre_f1_prueba"`,
		`"operador_login":"VecIdentity_Role"`, 1,
	))}
	pool := &poolFronteraIdentidadPrueba{txs: []*txFronteraIdentidadPrueba{preflight}}
	r, err := nuevoRegistradorFronteraIdentidadTecnicaPostgreSQL(context.Background(), pool)
	if err != nil || r == nil || r.config.OperadorLogin != "VecIdentity_Role" || preflight.commits != 1 {
		t.Fatalf("login devuelto por session_user rechazado localmente: %v", err)
	}
}

func TestFronteraIdentidadTecnicaConservaCancelacionDuranteCommitIncierto(t *testing.T) {
	r, _, tx := entornoFronteraIdentidadPrueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	tx.commitHook = cancelar
	tx.commitErr = errors.New("confirmacion perdida")
	orden, err := ports.NuevaOrdenFronteraIdentidadTecnica(domain.MetodoInicioSesionGET,
		domain.RutaSesionActual, ports.MotivoFronteraIdentidadAutenticacionRequerida)
	if err != nil {
		t.Fatal(err)
	}
	acuse, err := r.RegistrarRechazoInicioSesion(ctx, orden)
	if acuse != (ports.AcuseFronteraIdentidadTecnica{}) ||
		!errors.Is(err, ports.ErrFronteraIdentidadTecnicaCommitIncierto) ||
		!errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "confirmacion perdida") {
		t.Fatalf("commit incierto/cancelacion no cerraron START: %v", err)
	}
}
