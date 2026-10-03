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
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type relojFijo time.Time

func (r relojFijo) Ahora() time.Time { return time.Time(r) }

type filaFalsa struct {
	dato any
	err  error
}

func (f filaFalsa) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	switch d := destinos[0].(type) {
	case *bool:
		*d = f.dato.(bool)
	case *[]byte:
		*d = append([]byte(nil), f.dato.([]byte)...)
	default:
		panic("destino inesperado")
	}
	return nil
}

type txFalsa struct {
	pgx.Tx
	fila                          pgx.Row
	consultas, commits, rollbacks int
	falloCommit                   error
}

func (t *txFalsa) QueryRow(context.Context, string, ...any) pgx.Row { t.consultas++; return t.fila }
func (t *txFalsa) Commit(context.Context) error                     { t.commits++; return t.falloCommit }
func (t *txFalsa) Rollback(context.Context) error                   { t.rollbacks++; return nil }

type poolFalso struct {
	tx        *txFalsa
	fila      pgx.Row
	comienzos int
	opciones  pgx.TxOptions
}

func (p *poolFalso) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.comienzos++
	p.opciones = o
	return p.tx, nil
}
func (p *poolFalso) QueryRow(context.Context, string, ...any) pgx.Row { return p.fila }

type emisorFalso struct {
	material ports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	err      error
}

func (e *emisorFalso) EmitirAdministracionPerfiles(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, domain.InstantaneaAutorizacion, Efecto) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return e.material, e.err
}

// Este transporte sólo satisface la forma estructural de los puertos. No es
// una autorización firmada y nunca se monta como proveedor de ejecución real.
func materialSintetico(t *testing.T, e Efecto, ahora time.Time) ports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	h := sha256.Sum256(e.Material)
	r, err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:ejemplo", strings.Repeat("a", 64), strings.Repeat("b", 64), "contexto:ejemplo", strings.Repeat("c", 64), e.Accion, e.Referencia, hex.EncodeToString(h[:]), e.Audiencia, ahora, ahora.Add(4*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := x509.MarshalPKIXPublicKey(ed25519.PublicKey(make([]byte, ed25519.PublicKeySize)))
	if err != nil {
		t.Fatal(err)
	}
	m, err := ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(make([]byte, 512), r, []byte("decision"), []byte("motivo"), []byte("contexto"), 3, 4, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestTransaccionValidaSalidaAntesDeCommit(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	e := Efecto{Accion: "administracion.perfiles.revocar", Audiencia: "vec_autorizacion.administracion_perfiles.ordinario.v1", Referencia: "acto_admin:" + strings.Repeat("a", 32), Material: []byte(`{"esquema":"ejemplo"}`), CorrelacionAccesoRef: "correlacion_" + strings.Repeat("a", 32)}
	actor, evidencia := actorYEvidenciaPrueba(t, ahora)
	for _, caso := range []struct {
		nombre      string
		fila        pgx.Row
		validacion  error
		commitError error
		commits     int
	}{
		{"confirmado", filaFalsa{dato: []byte(`{}`)}, nil, nil, 1},
		{"respuesta_ajena", filaFalsa{dato: []byte(`{}`)}, domain.ErrActoAdministracionPerfilesInvalido, nil, 0},
		{"sql_denegado", filaFalsa{err: &pgconn.PgError{Code: "42501", Message: "dato privado"}}, nil, nil, 0},
		{"commit_incierto", filaFalsa{dato: []byte(`{}`)}, nil, errors.New("conexion perdida privada"), 1},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &txFalsa{fila: caso.fila, falloCommit: caso.commitError}
			pool := &poolFalso{tx: tx}
			a := &Autoridad{pool: pool, emisor: &emisorFalso{material: materialSintetico(t, e, ahora)}, reloj: relojFijo(ahora)}
			err := a.ejecutar(context.Background(), actor, evidencia, domain.InstantaneaAutorizacion{}, e, aplicarSQL, func([]byte) error { return caso.validacion })
			if (err == nil) != (caso.nombre == "confirmado") {
				t.Fatalf("resultado=%v", err)
			}
			if tx.commits != caso.commits || tx.rollbacks != 1 || pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite {
				t.Fatalf("tx=%+v opciones=%+v", tx, pool.opciones)
			}
			if err != nil && strings.Contains(err.Error(), "privad") {
				t.Fatal("error filtra datos del proveedor")
			}
		})
	}
}

func TestMaterialAjenoNoAbreTransaccion(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	e := Efecto{Accion: "administracion.perfiles.revocar", Audiencia: "vec_autorizacion.administracion_perfiles.ordinario.v1", Referencia: "acto_admin:" + strings.Repeat("a", 32), Material: []byte(`{}`), CorrelacionAccesoRef: "correlacion_" + strings.Repeat("a", 32)}
	actor, evidencia := actorYEvidenciaPrueba(t, ahora)
	for _, campo := range []string{"audiencia", "referencia", "accion", "material"} {
		t.Run(campo, func(t *testing.T) {
			ajeno := e
			switch campo {
			case "audiencia":
				ajeno.Audiencia += ".ajena"
			case "referencia":
				ajeno.Referencia = "acto_admin:" + strings.Repeat("b", 32)
			case "accion":
				ajeno.Accion = "administracion.perfiles.otorgar"
			case "material":
				ajeno.Material = []byte(`{"otro":true}`)
			}
			pool := &poolFalso{}
			a := &Autoridad{pool: pool, emisor: &emisorFalso{material: materialSintetico(t, ajeno, ahora)}, reloj: relojFijo(ahora)}
			if a.ejecutar(context.Background(), actor, evidencia, domain.InstantaneaAutorizacion{}, e, aplicarSQL, func([]byte) error { return nil }) == nil || pool.comienzos != 0 {
				t.Fatal("material ajeno alcanzo SQL")
			}
		})
	}
}

func actorYEvidenciaPrueba(t *testing.T, ahora time.Time) (domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles) {
	t.Helper()
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora,
		"per_"+strings.Repeat("a", 22), "prf_"+strings.Repeat("b", 22), domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	return resultado.Contexto, domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: vinculo}
}

func TestPoolPrivilegiadoYProveedorNuloSeRechazan(t *testing.T) {
	ctx := context.Background()
	reloj := relojFijo(time.Now())
	emisor := &emisorFalso{}
	if _, err := nueva(ctx, &poolFalso{fila: filaFalsa{dato: false}}, emisor, reloj); err == nil {
		t.Fatal("acepta pool no acreditado")
	}
	var nulo *emisorFalso
	if _, err := nueva(ctx, &poolFalso{}, nulo, reloj); err == nil {
		t.Fatal("acepta emisor tipado nulo")
	}
}

func TestDecoderRechazaFormaYContenidoAdicional(t *testing.T) {
	for _, b := range []string{`{"desconocido":true}`, `{} {}`, `null`, strings.Repeat("a", 65537)} {
		var x rolJSON
		if decodificar([]byte(b), &x) == nil {
			t.Fatalf("acepta %q", b[:min(len(b), 50)])
		}
	}
}

func TestCierreReconstruyeParticipantesYClaseEnSQL(t *testing.T) {
	s := domain.SolicitudCierrePropuestaAdministracionPerfiles{
		OperacionRef:          "cierre_admin:" + strings.Repeat("a", 32),
		PropuestaRef:          "propuesta_admin:" + strings.Repeat("b", 32),
		PropuestaHuellaSHA256: strings.Repeat("c", 64),
		ProponentePersonaRef:  "per_no_debe_serializarse", ObjetivoPersonaRef: "per_tampoco_serializado",
		Decision: domain.DecisionRechazarPropuestaPerfil,
	}
	s.Aprobador.PersonaRef = "per_actor_acreditado"
	s.Aprobador.PerfilActivoRef = "prf_actor_acreditado"
	e, err := materialCierre(s)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(e.Material, &m) != nil {
		t.Fatal("material no JSON")
	}
	for _, prohibido := range []string{"clase", "operacion", "proponente_persona_ref", "objetivo_persona_ref", "aprobador", "instantanea_autorizacion", "actor"} {
		if _, existe := m[prohibido]; existe {
			t.Fatalf("filtra campo %s", prohibido)
		}
	}
	if strings.Contains(string(e.Material), "no_debe") || strings.Contains(string(e.Material), "tampoco") || e.Accion != "administracion.perfiles.rechazar" {
		t.Fatal("cierre adopta participantes de la petición")
	}
}

func TestCatalogoExigeDecisionExplicitaDeUnidad(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	base := `{"version_ref":"rol:cronos_rrhh:v1","clase":"ordinario","huella_sha256":"` + strings.Repeat("a", 64) + `","vigente_desde":"2026-10-01T00:00:00Z","vigente_hasta":"2027-01-01T00:00:00Z"`
	for _, caso := range []struct {
		nombre, final string
		valido        bool
		unidad        bool
	}{
		{"ausente", "}", false, false}, {"nulo", `,"unidad_requerida":null}`, false, false},
		{"no_requiere", `,"unidad_requerida":false}`, true, false}, {"requiere", `,"unidad_requerida":true}`, true, true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			a := &Autoridad{pool: &poolFalso{fila: filaFalsa{dato: []byte(base + caso.final)}}, emisor: &emisorFalso{}, reloj: relojFijo(ahora)}
			rol, err := a.ResolverRolAdministrable(context.Background(), "rol:cronos_rrhh:v1")
			if (err == nil) != caso.valido || err == nil && rol.UnidadRequerida != caso.unidad {
				t.Fatalf("rol=%+v err=%v", rol, err)
			}
		})
	}
}

func TestFechasPersistiblesEnPropuestaYCierre(t *testing.T) {
	base := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	if !instantePersistible(base) || instantePersistible(base.Add(time.Nanosecond)) || instantePersistible(time.Time{}) {
		t.Fatal("acepta precisión que PostgreSQL perdería")
	}
}
