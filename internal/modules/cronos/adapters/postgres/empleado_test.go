package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const empleadoPrueba = "emp_0123456789abcdefghijkl"

type filaPrueba struct {
	valor []byte
	err   error
}

func (f filaPrueba) Scan(destino ...any) error {
	if f.err != nil {
		return f.err
	}
	*(destino[0].(*[]byte)) = append([]byte(nil), f.valor...)
	return nil
}

// txLecturaPrueba devuelve una respuesta por llamada y guarda consulta y material.
type txLecturaPrueba struct {
	pgx.Tx
	respuestas [][]byte
	errores    []error
	consultas  []string
	materiales []string
	commitErr  error
	commits    int
}

func (t *txLecturaPrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (t *txLecturaPrueba) QueryRow(_ context.Context, consulta string, args ...any) pgx.Row {
	t.consultas = append(t.consultas, consulta)
	t.materiales = append(t.materiales, args[0].(string))
	i := len(t.consultas) - 1
	var err error
	if i < len(t.errores) {
		err = t.errores[i]
	}
	if i >= len(t.respuestas) {
		return filaPrueba{err: errors.New("sin respuesta")}
	}
	return filaPrueba{valor: t.respuestas[i], err: err}
}
func (t *txLecturaPrueba) Commit(context.Context) error { t.commits++; return t.commitErr }
func (t *txLecturaPrueba) Rollback(context.Context) error {
	return nil
}

type dbLecturaPrueba struct {
	t  *testing.T
	tx *txLecturaPrueba
}

func (d dbLecturaPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	if d.tx == nil {
		d.t.Fatal("abre transacción antes de acreditar V3")
	}
	if o.IsoLevel != pgx.Serializable || o.AccessMode != pgx.ReadWrite {
		d.t.Fatal("aislamiento alterado")
	}
	return d.tx, nil
}

func exportacionPrueba(t *testing.T, accion, audiencia string, recurso vecdomain.RecursoAutorizable) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), accion, recurso.Referencia, huella, audiencia, ahora, ahora.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	e, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func actorPrueba(t *testing.T, empleados ...string) vecdomain.ContextoActor {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_0123456789abcdefghijkl", Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	var vinculos []vecdomain.VinculoReferenciaContextoActor
	for i, e := range empleados {
		vinculos = append(vinculos, vecdomain.VinculoReferenciaContextoActor{VinculoRef: "vin_0123456789abcdefghijk" + string(rune('a'+i)), Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: e, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Minute), VigenteHasta: ahora.Add(time.Minute)})
	}
	instantanea := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 1, PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Minute), VigenteHasta: ahora.Add(time.Minute), Vinculos: vinculos}
	actor, err := vecdomain.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	return actor
}

type proveedorSaldoPrueba struct {
	t         *testing.T
	err       error
	audiencia string
	llamadas  int
}

func (p *proveedorSaldoPrueba) ProveerMaterialConsultaSaldoPropio(_ context.Context, m domain.MaterialConsultaSaldoPropio) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.llamadas++
	if p.err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, p.err
	}
	r, err := application.RecursoConsultaSaldoPropio(m)
	if err != nil {
		p.t.Fatal(err)
	}
	return exportacionPrueba(p.t, application.AccionConsultarSaldoPropio, p.audiencia, r), nil
}

const fuenteSaldoPrueba = `{"empleado_ref":"emp_0123456789abcdefghijkl","desde":"2026-09-24","hasta":"2026-09-24","zona_horaria":"Europe/Madrid","completo":true,"jornadas":[],"marcajes":[{"marcaje_ref":"marcaje:cronos:x","movimiento":"entrada","instante_utc":"2026-09-24T06:00:00+00:00","canal":{"politica_version_ref":"v1","canal_ref":"c1","origen_ref":"remoto","calidad_ref":"q1"},"tipo_origen":"remoto"}],"movimientos_saldo":[]}`

func TestConsultaSaldoConsumeV3YLeeEnUnaTransaccion(t *testing.T) {
	if r, err := NuevoRepositorioConsultaSaldo(nil); r != nil || !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("constructor abierto sin pool")
	}
	actor := actorPrueba(t, empleadoPrueba)
	sinProveedor, _ := ports.NuevaOrdenConsultaSaldo(actor)
	r := &RepositorioConsultaSaldo{db: dbLecturaPrueba{t: t}}
	if _, err := r.ConsultarFuenteSaldo(context.Background(), sinProveedor, empleadoPrueba, "2026-09-24", "2026-09-24", "Europe/Madrid"); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("lee sin autoridad V3", err)
	}
	denegado := &proveedorSaldoPrueba{t: t, err: errors.Join(vecdomain.ErrPermissionDenied, errors.New("detalle"))}
	orden, _ := ports.NuevaOrdenConsultaSaldoAutorizada(actor, denegado)
	if _, err := r.ConsultarFuenteSaldo(context.Background(), orden, empleadoPrueba, "2026-09-24", "2026-09-24", "Europe/Madrid"); !errors.Is(err, vecdomain.ErrPermissionDenied) || denegado.llamadas != 1 {
		t.Fatal("denegación del PDP no conservada", err)
	}
	ajena := &proveedorSaldoPrueba{t: t, audiencia: application.AudienciaRecuperacionMarcajeRemoto}
	orden, _ = ports.NuevaOrdenConsultaSaldoAutorizada(actor, ajena)
	if _, err := r.ConsultarFuenteSaldo(context.Background(), orden, empleadoPrueba, "2026-09-24", "2026-09-24", "Europe/Madrid"); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta V3 de otra audiencia", err)
	}
	if _, err := r.ConsultarFuenteSaldo(context.Background(), orden, "emp_otroempleado0123456789", "2026-09-24", "2026-09-24", "Europe/Madrid"); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("lee un empleado ajeno al contexto", err)
	}
	tx := &txLecturaPrueba{respuestas: [][]byte{[]byte(fuenteSaldoPrueba)}}
	r = &RepositorioConsultaSaldo{db: dbLecturaPrueba{t: t, tx: tx}}
	orden, _ = ports.NuevaOrdenConsultaSaldoAutorizada(actor, &proveedorSaldoPrueba{t: t, audiencia: application.AudienciaConsultaSaldoPropio})
	f, err := r.ConsultarFuenteSaldo(context.Background(), orden, empleadoPrueba, "2026-09-24", "2026-09-24", "Europe/Madrid")
	if err != nil || tx.commits != 1 || tx.consultas[0] != consultaSaldoPropio || len(f.Marcajes) != 1 || f.Marcajes[0].InstanteUTC.Location() != time.UTC {
		t.Fatal(err, tx.commits, f)
	}
	esperado, _ := domain.MaterialConsultaSaldoPropio{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleadoPrueba, Desde: "2026-09-24", Hasta: "2026-09-24", ZonaHoraria: "Europe/Madrid"}.Canonico()
	if tx.materiales[0] != string(esperado) {
		t.Fatal("material SQL distinto del autorizado")
	}
	tx = &txLecturaPrueba{respuestas: [][]byte{[]byte(fuenteSaldoPrueba)}, commitErr: errors.New("commit perdido")}
	r = &RepositorioConsultaSaldo{db: dbLecturaPrueba{t: t, tx: tx}}
	if _, err := r.ConsultarFuenteSaldo(context.Background(), orden, empleadoPrueba, "2026-09-24", "2026-09-24", "Europe/Madrid"); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("entrega lectura sin COMMIT", err)
	}
}

type proveedorDisponibilidadPrueba struct {
	t          *testing.T
	materiales []domain.MaterialDisponibilidadMarcajeRemoto
}

func (p *proveedorDisponibilidadPrueba) ProveerMaterialDisponibilidadMarcajeRemoto(_ context.Context, m domain.MaterialDisponibilidadMarcajeRemoto) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.materiales = append(p.materiales, m)
	r, err := application.RecursoDisponibilidadMarcajeRemoto(m)
	if err != nil {
		p.t.Fatal(err)
	}
	return exportacionPrueba(p.t, application.AccionConsultarDisponibilidadRemota, application.AudienciaDisponibilidadMarcajeRemoto, r), nil
}

func canalRemotoPrueba(t *testing.T) domain.AcreditacionCanalMarcaje {
	t.Helper()
	c, err := domain.NuevaAcreditacionCanalMarcaje(domain.DatosAcreditacionCanalMarcaje{PoliticaVersionRef: "politica:canal:cronos:v1", CanalRef: "portal-empleado-web", OrigenRef: domain.OrigenMarcajeRemoto, CalidadRef: "mtls-certificado"})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestEstadoRemotoUsaCacheDeLaPeticionYReleeConClave(t *testing.T) {
	actor := actorPrueba(t, empleadoPrueba)
	instante := time.Now().UTC().Truncate(time.Microsecond)
	periodo := `"desde":"2026-09-24T06:00:00+00:00","hasta":"2026-09-24T15:00:00+00:00"`
	tx := &txLecturaPrueba{respuestas: [][]byte{
		[]byte(`{"autorizado":true,` + periodo + `,"continuidad_confirmada":true,"movimientos_permitidos":["salida","inicio_pausa"]}`),
		[]byte(`{"autorizado":true,` + periodo + `,"continuidad_confirmada":true,"movimientos_permitidos":["entrada"]}`),
	}}
	proveedor := &proveedorDisponibilidadPrueba{t: t}
	r := &RepositorioMarcajesRemotos{db: dbLecturaPrueba{t: t, tx: tx}, proveedor: proveedor, canal: canalRemotoPrueba(t)}
	if _, _, err := r.ConsultarTeletrabajoPropio(context.Background(), actor, empleadoPrueba, instante); !errors.Is(err, ports.ErrDependenciaNoDisponible) || len(tx.consultas) != 0 {
		t.Fatal("consulta sin ámbito de petición", err)
	}
	ctx := ContextoConEstadoRemoto(context.Background())
	p, ok, err := r.ConsultarTeletrabajoPropio(ctx, actor, empleadoPrueba, instante)
	if err != nil || !ok || p.DesdeUTC.Location() != time.UTC || tx.consultas[0] != consultaEstadoRemotoPropio {
		t.Fatal(p, ok, err)
	}
	e, err := r.ConfirmarContinuidadMarcajeRemoto(ctx, actor, empleadoPrueba, p, "")
	if err != nil || len(tx.consultas) != 1 || len(e.MovimientosPermitidos) != 2 {
		t.Fatal("continuidad sin clave no reutiliza la lectura", err, len(tx.consultas))
	}
	e, err = r.ConfirmarContinuidadMarcajeRemoto(ctx, actor, empleadoPrueba, p, "clave-remota-0001")
	if err != nil || len(tx.consultas) != 2 || len(e.MovimientosPermitidos) != 1 || proveedor.materiales[1].ClaveOperacion != "clave-remota-0001" || !proveedor.materiales[1].InstanteUTC.Equal(instante) {
		t.Fatal("la clave no se concilia bajo nueva lectura", err)
	}
	otro := p
	otro.HastaUTC = otro.HastaUTC.Add(time.Hour)
	if _, err := r.ConfirmarContinuidadMarcajeRemoto(ctx, actor, empleadoPrueba, otro, ""); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta un periodo distinto del leído", err)
	}
	if _, err := r.ConfirmarContinuidadMarcajeRemoto(ctx, actor, "emp_otroempleado0123456789", p, ""); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta otro empleado", err)
	}
}

func TestDecodificarEstadoRemotoFallaCerrado(t *testing.T) {
	for _, bruto := range []string{
		`{"autorizado":false,"continuidad_confirmada":true,"movimientos_permitidos":[]}`,
		`{"autorizado":false,"continuidad_confirmada":false,"movimientos_permitidos":["entrada"]}`,
		`{"autorizado":true,"continuidad_confirmada":true,"movimientos_permitidos":["entrada"]}`,
		`{"autorizado":true,"desde":"2026-09-24T15:00:00Z","hasta":"2026-09-24T06:00:00Z","continuidad_confirmada":true,"movimientos_permitidos":["entrada"]}`,
		`{"autorizado":true,"desde":"2026-09-24T06:00:00Z","hasta":"2026-09-24T15:00:00Z","continuidad_confirmada":false,"movimientos_permitidos":["entrada"]}`,
		`{"autorizado":true,"desde":"2026-09-24T06:00:00Z","hasta":"2026-09-24T15:00:00Z","continuidad_confirmada":true,"movimientos_permitidos":["volar"]}`,
		`{"autorizado":false,"continuidad_confirmada":false,"movimientos_permitidos":[],"otro":1}`,
	} {
		if _, err := decodificarEstadoRemoto([]byte(bruto)); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
			t.Fatalf("acepta %s", bruto)
		}
	}
	if e, err := decodificarEstadoRemoto([]byte(`{"autorizado":false,"continuidad_confirmada":false,"movimientos_permitidos":[]}`)); err != nil || *e.Autorizado {
		t.Fatal(err)
	}
}

func TestRecuperacionRemotaAusenteSoloTrasCommit(t *testing.T) {
	canal := canalRemotoPrueba(t)
	material := domain.MaterialRecuperacionMarcajeRemoto{ActorRef: "per_0123456789abcdefghijkl", PerfilRef: "prf_0123456789abcdefghijkl", EmpleadoRef: empleadoPrueba, ClaveOperacion: "clave-remota-0001", Movimiento: domain.PunchEntry, Canal: canal}
	recurso, err := application.RecursoRecuperacionMarcajeRemoto(material)
	if err != nil {
		t.Fatal(err)
	}
	v3 := exportacionPrueba(t, application.AccionRecuperarMarcajeRemoto, application.AudienciaRecuperacionMarcajeRemoto, recurso)
	tx := &txLecturaPrueba{respuestas: [][]byte{[]byte(`{"ausente": true}`)}}
	r := &RepositorioMarcajesRemotos{db: dbLecturaPrueba{t: t, tx: tx}, canal: canal}
	if _, err := r.RecuperarOriginalRemotoAutorizado(context.Background(), material, v3); !errors.Is(err, ports.ErrMarcajeRemotoNoEncontrado) || tx.commits != 1 {
		t.Fatal(err)
	}
	tx = &txLecturaPrueba{respuestas: [][]byte{[]byte(`{"ausente": true}`)}, commitErr: errors.New("perdido")}
	r.db = dbLecturaPrueba{t: t, tx: tx}
	if _, err := r.RecuperarOriginalRemotoAutorizado(context.Background(), material, v3); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("afirma ausencia sin COMMIT", err)
	}
	recibo := `{"referencia":"recibo:cronos:00000000-0000-4000-8000-000000000001","instante_utc":"2026-09-24T06:00:00.000001+00:00","marcaje_original_ref":"marcaje:cronos:clave-remota-0001","replay":%s}`
	tx = &txLecturaPrueba{respuestas: [][]byte{[]byte(strings.Replace(recibo, "%s", "false", 1))}}
	r.db = dbLecturaPrueba{t: t, tx: tx}
	if _, err := r.RecuperarOriginalRemotoAutorizado(context.Background(), material, v3); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("recuperación que no es replay", err)
	}
	tx = &txLecturaPrueba{respuestas: [][]byte{[]byte(strings.Replace(recibo, "%s", "true", 1))}}
	r.db = dbLecturaPrueba{t: t, tx: tx}
	got, err := r.RecuperarOriginalRemotoAutorizado(context.Background(), material, v3)
	if err != nil || !got.Replay || got.InstanteUTC.Location() != time.UTC {
		t.Fatal(got, err)
	}
	ajeno := exportacionPrueba(t, application.AccionRecuperarMarcajeRemoto, application.AudienciaMarcajePropio, recurso)
	r.db = dbLecturaPrueba{t: t}
	if _, err := r.RecuperarOriginalRemotoAutorizado(context.Background(), material, ajeno); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta V3 de otra audiencia", err)
	}
}

func TestErroresRemotosSQLCerrados(t *testing.T) {
	for codigo, esperado := range map[string]error{"PC004": ports.ErrTeletrabajoNoAutorizado, "PC005": ports.ErrMovimientoRemotoNoPermitido, "PC006": ports.ErrContinuidadMarcajeNoConfirmada, "PC001": ports.ErrDependenciaNoDisponible} {
		if err := errorSeguro(context.Background(), &pgconn.PgError{Code: codigo, Message: "detalle privado"}); !errors.Is(err, esperado) {
			t.Fatalf("%s mal traducido: %v", codigo, err)
		}
	}
	if _, err := NuevoRepositorioMarcajesRemotos(nil, nil, nil, domain.AcreditacionCanalMarcaje{}); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("constructor remoto abierto")
	}
}

func TestOrdenDenegacionFronteraCerrada(t *testing.T) {
	valida := ports.OrdenDenegacionFronteraCronos{CorrelacionRef: "corr_" + strings.Repeat("a", 32), Motivo: ports.MotivoFronteraSinEmpleado, Ruta: "/api/interna/cronos/saldos/propio", Metodo: "GET"}
	if valida.Validar() != nil {
		t.Fatal("orden válida rechazada")
	}
	for _, cambiar := range []func(*ports.OrdenDenegacionFronteraCronos){
		func(o *ports.OrdenDenegacionFronteraCronos) { o.Motivo = "otro" },
		func(o *ports.OrdenDenegacionFronteraCronos) { o.Ruta = "/api/interna/cronos/x" },
		func(o *ports.OrdenDenegacionFronteraCronos) { o.Metodo = "DELETE" },
		func(o *ports.OrdenDenegacionFronteraCronos) { o.ActorRef = "12345678Z" },
		func(o *ports.OrdenDenegacionFronteraCronos) { o.CorrelacionRef = "libre" },
	} {
		o := valida
		cambiar(&o)
		if o.Validar() == nil {
			t.Fatalf("acepta %+v", o)
		}
	}
	var r *RegistroDenegacionFronteraPostgreSQL
	if !errors.Is(r.RegistrarDenegacionFronteraCronos(context.Background(), valida), ports.ErrDenegacionFronteraNoRegistrada) {
		t.Fatal("registro nulo abierto")
	}
}
