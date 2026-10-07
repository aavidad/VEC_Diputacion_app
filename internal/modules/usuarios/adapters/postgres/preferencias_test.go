package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type filaPrueba struct {
	datos any
	err   error
}

func (f filaPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	switch destino := destinos[0].(type) {
	case *bool:
		*destino = f.datos.(bool)
	case *[]byte:
		*destino = f.datos.([]byte)
	}
	return nil
}

type llamadaPrueba struct {
	sql  string
	args []any
}

type txPrueba struct {
	valido    bool
	respuesta []byte
	errSQL    error
	errAjuste error
	errCommit error
	consultas []llamadaPrueba
	ajustes   []string
	commits   int
	rollbacks int
}

func (t *txPrueba) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	t.ajustes = append(t.ajustes, sql)
	return pgconn.CommandTag{}, t.errAjuste
}
func (t *txPrueba) QueryRow(_ context.Context, sql string, args ...any) filaPreferencias {
	t.consultas = append(t.consultas, llamadaPrueba{sql, args})
	if sql == acreditarEjecutorSQL {
		return filaPrueba{datos: t.valido}
	}
	return filaPrueba{datos: t.respuesta, err: t.errSQL}
}
func (t *txPrueba) Commit(context.Context) error   { t.commits++; return t.errCommit }
func (t *txPrueba) Rollback(context.Context) error { t.rollbacks++; return nil }

func repositorioPrueba(tx *txPrueba, superficie vecdomain.SuperficieAutenticacionActorV1) *RegistroPreferenciasPostgreSQL {
	return &RegistroPreferenciasPostgreSQL{superficie: superficie, rol: rolEjecutorPreferencias(superficie), iniciar: func(context.Context) (transaccionPreferencias, error) { return tx, nil }}
}

type proveedorPrueba struct{}

func (proveedorPrueba) ProveerMaterialPreferencias(context.Context, vecdomain.VinculoAutenticacionActorV2, ports.MaterialPreferencias) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
}

type revalidadorPrefPGPrueba struct {
	a vecdomain.AutenticacionRevalidadaV1
}

func (r revalidadorPrefPGPrueba) RevalidarAutenticacionActorV1(context.Context, vecdomain.SolicitudRevalidacionAutenticacionActorV1) (vecdomain.AutenticacionRevalidadaV1, error) {
	return r.a, nil
}

type resolutorPrefPGPrueba struct {
	r vecdomain.ResultadoContextoActorRegistradoV2
}

func (r resolutorPrefPGPrueba) ResolverContextoActorRegistradoV2(context.Context, vecdomain.SolicitudContextoActor) (vecdomain.ResultadoContextoActorRegistradoV2, error) {
	return r.r, nil
}

type relojPrefPGPrueba struct{ ahora time.Time }

func (r relojPrefPGPrueba) Ahora() time.Time { return r.ahora }

func materialPrueba(t *testing.T, accion string, superficie vecdomain.SuperficieAutenticacionActorV1) (ports.OrdenPreferencias, ports.MaterialPreferencias, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	z := strings.Repeat("a", 24)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	snap := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + strings.Repeat("r", 24), PersonaVersion: 1, PerfilActivoRef: "prf_" + strings.Repeat("p", 24), PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := vecdomain.NuevoContextoActor(cuenta, snap, ahora)
	if err != nil {
		t.Fatal(err)
	}
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	huella, _ := actor.HuellaSHA256VinculadaV2()
	ac := vecdomain.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_" + z, ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("a", 64), ProcedenciaAutoridad: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	man := vecdomain.ManifiestoProcedenciaContextoActorV1{Esquema: vecdomain.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, Cuenta: vecdomain.ProcedenciaCuentaContextoActorV1{CuentaRef: cuenta.CuentaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Persona: vecdomain.ProcedenciaPersonaContextoActorV1{PersonaRef: actor.PersonaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Perfil: vecdomain.ProcedenciaPerfilContextoActorV1{PerfilRef: actor.PerfilActivoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Contexto: vecdomain.ProcedenciaVinculoContextoActorV1{VinculoRef: snap.VinculoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Vinculos: []vecdomain.ProcedenciaVinculoReferenciaContextoActorV1{}}
	bm, _ := man.RepresentacionCanonicaV1()
	hm, _ := vecdomain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(bm)
	res := vecdomain.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "rca_" + z, Contexto: actor, RepresentacionCanonica: canon, HuellaSHA256: huella, ManifiestoProcedenciaCanonico: bm, ManifiestoProcedenciaHuellaSHA256: hm, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, ResueltoEnAutoritativo: ahora}
	if err := res.Validar(); err != nil {
		t.Fatal(err)
	}
	auth := vecdomain.AutenticacionRevalidadaV1{AutenticacionRef: "aut_" + z, AutenticacionHuellaSHA256: strings.Repeat("a", 64), AsercionRef: "ase_" + z, SesionRef: "ses_" + z, ControlSesionRef: "cse_" + z, ControlSesionRevision: 1, ControlSesionHuellaSHA256: strings.Repeat("b", 64), CuentaRef: cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef, Superficie: superficie, MetodoObservado: vecdomain.AuthMethodCertificate, GarantiaObservada: vecdomain.AuthAssuranceHigh, PoliticaGarantiaRef: "pga_" + z, PoliticaGarantiaHuellaSHA256: strings.Repeat("c", 64), AutenticacionVerificadaEn: ahora.Add(-time.Minute), SesionEmitidaEn: ahora.Add(-time.Minute), SesionRevalidadaEn: ahora.Add(-time.Second), SesionValidaHasta: ahora.Add(time.Minute)}
	if err := auth.Validar(); err != nil {
		t.Fatal(err)
	}
	vinculo, err := vecdomain.CrearVinculoAutenticacionActorV2(context.Background(), revalidadorPrefPGPrueba{auth}, vecdomain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: auth.AutenticacionRef, SesionRef: auth.SesionRef}, resolutorPrefPGPrueba{res}, vecdomain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef}, relojPrefPGPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenPreferencias(actor, vinculo, superficie, proveedorPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	m := ports.MaterialPreferencias{Superficie: superficie, PersonaRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, Accion: accion, FinalidadRef: ports.FinalidadPreferenciasPropias, CatalogoVersionRef: domain.CatalogoBasePreferencias().VersionRef, Valores: domain.CatalogoBasePreferencias().Predeterminados}
	if accion == ports.AccionActualizarPreferencias {
		m.ClaveOperacion = "operacion-1234567890"
		m.HuellaPeticion = strings.Repeat("a", 64)
	}
	audiencia, _ := ports.AudienciaPreferencias(accion, superficie)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), accion, actor.PersonaRef, strings.Repeat("d", 64), audiencia, ahora, ahora.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if err != nil {
		t.Fatal(err)
	}
	v3, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return orden, m, v3
}

func TestCatalogoSQLDecodificaClavesYNoAmpliaVocabulario(t *testing.T) {
	superficie := vecdomain.SuperficieAutenticacionInternaCorporativaV1
	orden, _, _ := materialPrueba(t, ports.AccionConsultarPreferencias, superficie)
	base := domain.CatalogoBasePreferencias()
	// El catálogo publicado en SQL usa el JSON canónico del núcleo.
	datos, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(datos, []byte(`"tamanos_texto"`)) {
		t.Fatal("nombre de catálogo ajeno al contrato Go")
	}
	tx := &txPrueba{valido: true, respuesta: datos}
	c, err := repositorioPrueba(tx, superficie).CatalogoVigente(context.Background(), orden)
	if err != nil || !reflect.DeepEqual(c, base) || tx.commits != 1 || len(tx.ajustes) != 1 || tx.ajustes[0] != ajustesTransaccionSQL {
		t.Fatalf("catalogo/tx: %v %#v", err, c)
	}
	if len(tx.consultas) != 2 || tx.consultas[0].args[0] != rolEjecutorPreferencias(superficie) || tx.consultas[1].sql != consultarCatalogoSQL || tx.consultas[1].args[0] != string(superficie) {
		t.Fatal("rol o funcion nominal no consultados")
	}
	tx = &txPrueba{valido: false, respuesta: datos}
	if _, err := repositorioPrueba(tx, superficie).CatalogoVigente(context.Background(), orden); !errors.Is(err, ports.ErrNoDisponible) || len(tx.consultas) != 1 || tx.commits != 0 {
		t.Fatal("login no exclusivo paso la sonda")
	}
	tx = &txPrueba{valido: true, errAjuste: errors.New("ajuste fallido")}
	if _, err := repositorioPrueba(tx, superficie).CatalogoVigente(context.Background(), orden); !errors.Is(err, ports.ErrNoDisponible) || len(tx.consultas) != 0 || tx.commits != 0 || tx.rollbacks == 0 {
		t.Fatal("ajuste fallido llegó a acreditar el login o consultar datos")
	}
	conClaveAjena := bytes.Replace(datos, []byte(`"tamanos_texto"`), []byte(`"tamano_textos"`), 1)
	if _, err := decodificarCatalogo(conClaveAjena); !errors.Is(err, ports.ErrNoDisponible) {
		t.Fatal("catálogo con clave vieja aceptado")
	}
}

// Se ejecuta con un PostgreSQL 18 efímero. Comprueba el alcance real de los
// ajustes en una conexión reutilizada, también después de un error SQL.
func TestAjustesPreferenciasPG18LocalesATransaccion(t *testing.T) {
	dsn := os.Getenv("VEC_USUARIOS_AJUSTES_PG18_DSN")
	if dsn == "" {
		t.Skip("PostgreSQL 18 efímero no configurado")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal("no se pudo conectar al PostgreSQL 18 efímero")
	}
	defer conn.Close(context.Background())
	const leer = `SELECT current_setting('search_path'),current_setting('row_security'),
 current_setting('timezone'),current_setting('lock_timeout'),current_setting('statement_timeout'),
 current_setting('idle_in_transaction_session_timeout')`
	valores := func(consultor interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	}) [6]string {
		t.Helper()
		var got [6]string
		if err := consultor.QueryRow(ctx, leer).Scan(&got[0], &got[1], &got[2], &got[3], &got[4], &got[5]); err != nil {
			t.Fatal("no se pudieron leer los ajustes de la conexión")
		}
		return got
	}
	for _, ajuste := range []struct{ nombre, valor string }{
		{"search_path", "public"}, {"row_security", "off"}, {"timezone", "Europe/Madrid"},
		{"lock_timeout", "1s"}, {"statement_timeout", "5s"}, {"idle_in_transaction_session_timeout", "5s"},
	} {
		if _, err := conn.Exec(ctx, `SELECT pg_catalog.set_config($1,$2,false)`, ajuste.nombre, ajuste.valor); err != nil {
			t.Fatal("no se pudo preparar la sesión de prueba")
		}
	}
	base := valores(conn)
	esperados := [6]string{"pg_catalog, pg_temp", "on", "UTC", "3s", "15s", "20s"}
	for _, confirmar := range []bool{true, false} {
		tx, err := conn.Begin(ctx)
		if err != nil {
			t.Fatal("no se pudo abrir la transacción de prueba")
		}
		if _, err := tx.Exec(ctx, ajustesTransaccionSQL); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatal("no se pudieron aplicar los ajustes transaccionales")
		}
		if got := valores(tx); got != esperados {
			_ = tx.Rollback(ctx)
			t.Fatalf("ajustes transaccionales = %q; esperados %q", got, esperados)
		}
		if confirmar {
			err = tx.Commit(ctx)
		} else {
			err = tx.Rollback(ctx)
		}
		if err != nil || valores(conn) != base {
			t.Fatal("los ajustes transaccionales persistieron tras COMMIT o ROLLBACK")
		}
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal("no se pudo abrir la transacción de fallo")
	}
	if _, err := tx.Exec(ctx, `SELECT pg_catalog.set_config('row_security','valor_invalido',true)`); err == nil {
		_ = tx.Rollback(ctx)
		t.Fatal("PostgreSQL aceptó un ajuste inválido")
	}
	if err := tx.Rollback(ctx); err != nil || valores(conn) != base {
		t.Fatal("la conexión retuvo un ajuste después del error y ROLLBACK")
	}
}

func BenchmarkAjustesPreferenciasPG18(b *testing.B) {
	dsn := os.Getenv("VEC_USUARIOS_AJUSTES_PG18_DSN")
	if dsn == "" {
		b.Skip("PostgreSQL 18 efímero no configurado")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		b.Fatal("no se pudo conectar al PostgreSQL 18 efímero")
	}
	defer conn.Close(ctx)
	for nombre, ajustes := range map[string][]string{
		"seis_SET_LOCAL": {
			"SET LOCAL search_path = pg_catalog, pg_temp", "SET LOCAL row_security = on",
			"SET LOCAL TIME ZONE 'UTC'", "SET LOCAL lock_timeout = '3s'",
			"SET LOCAL statement_timeout = '15s'", "SET LOCAL idle_in_transaction_session_timeout = '20s'",
		},
		"una_SELECT": {ajustesTransaccionSQL},
	} {
		b.Run(nombre, func(b *testing.B) {
			for range b.N {
				tx, err := conn.Begin(ctx)
				if err != nil {
					b.Fatal("no se pudo abrir la transacción de medida")
				}
				for _, sql := range ajustes {
					if _, err := tx.Exec(ctx, sql); err != nil {
						_ = tx.Rollback(ctx)
						b.Fatal("falló un ajuste durante la medida")
					}
				}
				if err := tx.Rollback(ctx); err != nil {
					b.Fatal("no se pudo cerrar la transacción de medida")
				}
			}
		})
	}
}

func TestConsultaEnviaMaterialLiteralYDiezPiezas(t *testing.T) {
	superficie := vecdomain.SuperficieAutenticacionInternaCorporativaV1
	orden, m, v3 := materialPrueba(t, ports.AccionConsultarPreferencias, superficie)
	estado := ports.EstadoPreferencias{PersonaRef: m.PersonaRef, Version: 0, CatalogoVersionRef: m.CatalogoVersionRef, Valores: m.Valores}
	datos, _ := json.Marshal(struct {
		Existe bool `json:"existe"`
		ports.EstadoPreferencias
	}{false, estado})
	tx := &txPrueba{valido: true, respuesta: datos}
	obtenido, existe, err := repositorioPrueba(tx, superficie).ConsultarPropias(context.Background(), orden, m, v3)
	if err != nil || existe || obtenido != estado || tx.commits != 1 {
		t.Fatalf("consulta: %v, existe=%t", err, existe)
	}
	call := tx.consultas[1]
	if call.sql != consultarPropiasSQL || len(call.args) != 11 {
		t.Fatal("firma SQL incorrecta")
	}
	literal, _ := json.Marshal(m)
	if call.args[0] != string(literal) || !bytes.Equal(call.args[1].([]byte), v3.CapacidadCanonica()) || call.args[5] != int64(1) || call.args[6] != int64(1) {
		t.Fatal("material V3 mezclado o transformado")
	}
}

func TestRegistroRechazaCruceDeSuperficieYRol(t *testing.T) {
	interna := vecdomain.SuperficieAutenticacionInternaCorporativaV1
	externa := vecdomain.SuperficieAutenticacionExternaPersonalV1
	ordenExterna, materialExterno, v3Externo := materialPrueba(t, ports.AccionConsultarPreferencias, externa)
	base := domain.CatalogoBasePreferencias()
	datos, _ := json.Marshal(base)
	tx := &txPrueba{valido: true, respuesta: datos}
	rInterno := repositorioPrueba(tx, interna)
	if _, err := rInterno.CatalogoVigente(context.Background(), ordenExterna); !errors.Is(err, ports.ErrProhibido) || len(tx.consultas) != 0 {
		t.Fatal("catálogo exterior leído con pool interno")
	}
	if _, _, err := rInterno.ConsultarPropias(context.Background(), ordenExterna, materialExterno, v3Externo); !errors.Is(err, ports.ErrProhibido) || len(tx.consultas) != 0 {
		t.Fatal("material exterior consumido con pool interno")
	}
	rExterno := repositorioPrueba(tx, externa)
	if _, err := rExterno.CatalogoVigente(context.Background(), ordenExterna); err != nil || tx.consultas[0].args[0] != "vec_usuarios_ejecutor_externo" || tx.consultas[1].args[0] != string(externa) {
		t.Fatalf("catálogo exterior/rol: %v", err)
	}
	ordenInterna, materialInterno, v3Interno := materialPrueba(t, ports.AccionActualizarPreferencias, interna)
	txCruce := &txPrueba{valido: true}
	rCruce := repositorioPrueba(txCruce, externa)
	if _, _, err := rCruce.RecuperarOperacion(context.Background(), ordenInterna, materialInterno, v3Interno); !errors.Is(err, ports.ErrProhibido) || len(txCruce.consultas) != 0 {
		t.Fatal("recuperación interna cruzó pool exterior")
	}
	peticion := ports.PeticionGuardarPreferencias{VersionEsperada: materialInterno.VersionEsperada, CatalogoVersionRef: materialInterno.CatalogoVersionRef, ClaveOperacion: materialInterno.ClaveOperacion, Valores: materialInterno.Valores}
	if _, err := rCruce.Guardar(context.Background(), ordenInterna, peticion, materialInterno, v3Interno); !errors.Is(err, ports.ErrProhibido) || len(txCruce.consultas) != 0 {
		t.Fatal("guardado interno cruzó pool exterior")
	}
	materialExterno.Superficie = interna
	if _, _, err := rExterno.ConsultarPropias(context.Background(), ordenExterna, materialExterno, v3Externo); !errors.Is(err, ports.ErrProhibido) {
		t.Fatal("material de superficie mutada aceptado")
	}
}

func TestMaterialLiteralIncluyeSuperficieSinTransformacion(t *testing.T) {
	m := ports.MaterialPreferencias{
		Superficie: vecdomain.SuperficieAutenticacionInternaCorporativaV1,
		PersonaRef: "per_0123456789abcdefghijkl", PerfilRef: "prf_0123456789abcdefghijkl",
		Accion: ports.AccionActualizarPreferencias, FinalidadRef: ports.FinalidadPreferenciasPropias,
		CatalogoVersionRef: "usuarios-preferencias-v1", VersionEsperada: 0,
		ClaveOperacion: "operacion-1234567890",
		HuellaPeticion: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Valores:        domain.CatalogoBasePreferencias().Predeterminados,
	}
	args, err := argumentosV3(m, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{})
	if err != nil || len(args) != 11 {
		t.Fatal("material SQL no construido")
	}
	literal, ok := args[0].(string)
	if !ok {
		t.Fatal("material no transmitido como texto")
	}
	canon, _ := json.Marshal(m)
	if literal != string(canon) || !strings.Contains(literal, `"superficie":"interna_corporativa"`) {
		t.Fatal("superficie perdida o material SQL transformado")
	}
	compararVector := func(material ports.MaterialPreferencias, esperado string) {
		t.Helper()
		args, err := argumentosV3(material, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{})
		if err != nil {
			t.Fatal(err)
		}
		huella := sha256.Sum256([]byte(args[0].(string)))
		if got := hex.EncodeToString(huella[:]); got != esperado {
			t.Fatalf("vector Go/SQL divergente: %s", got)
		}
	}
	compararVector(m, "93e51bfaf653b06d10b6033cd348af16ae15796e4c36d99fe7ae19fa6ccda517")
	m.Superficie = vecdomain.SuperficieAutenticacionExternaPersonalV1
	compararVector(m, "e22d42d7a997fdbdd9a71ff10e7b805767bc5b7d28bb61870b4097f78cd4e723")
}

func TestReplayConservaReciboYGuardarRespetaCAS(t *testing.T) {
	superficie := vecdomain.SuperficieAutenticacionInternaCorporativaV1
	orden, m, v3 := materialPrueba(t, ports.AccionActualizarPreferencias, superficie)
	fecha := time.Date(2026, 9, 29, 7, 45, 0, 123000, time.UTC)
	recibo := ports.ReciboPreferencias{ReciboRef: "recibo:original", PersonaRef: m.PersonaRef, Version: 1, CatalogoVersionRef: m.CatalogoVersionRef, Valores: m.Valores, FechaUTC: fecha}
	datosGuardar, _ := json.Marshal(recibo)
	recibo.Replay = true
	datosReplay, _ := json.Marshal(recibo)
	tx := &txPrueba{valido: true, respuesta: datosReplay}
	recuperado, existe, err := repositorioPrueba(tx, superficie).RecuperarOperacion(context.Background(), orden, m, v3)
	if err != nil || !existe || !recuperado.Replay || recuperado.ReciboRef != recibo.ReciboRef || !recuperado.FechaUTC.Equal(fecha) || tx.commits != 1 || tx.consultas[1].sql != recuperarSQL {
		t.Fatalf("replay: %v, %#v", err, recuperado)
	}
	tx = &txPrueba{valido: true, respuesta: []byte("null")}
	_, existe, err = repositorioPrueba(tx, superficie).RecuperarOperacion(context.Background(), orden, m, v3)
	if err != nil || existe || tx.commits != 1 {
		t.Fatal("ausencia de replay incorrecta")
	}
	peticion := ports.PeticionGuardarPreferencias{VersionEsperada: m.VersionEsperada, CatalogoVersionRef: m.CatalogoVersionRef, ClaveOperacion: m.ClaveOperacion, Valores: m.Valores}
	tx = &txPrueba{valido: true, respuesta: datosGuardar}
	guardado, err := repositorioPrueba(tx, superficie).Guardar(context.Background(), orden, peticion, m, v3)
	if err != nil || guardado.Replay || tx.commits != 1 || tx.consultas[1].sql != guardarSQL || len(tx.consultas[1].args) != 12 {
		t.Fatalf("guardar: %v", err)
	}
	if !bytes.Equal(tx.consultas[1].args[1].([]byte), mustJSON(t, m.Valores)) {
		t.Fatal("valores SQL distintos del material")
	}
	peticion.ClaveOperacion = "otra-clave-123456789"
	if _, err := repositorioPrueba(&txPrueba{valido: true}, superficie).Guardar(context.Background(), orden, peticion, m, v3); !errors.Is(err, ports.ErrPeticionInvalida) {
		t.Fatal("material de PUT no ligado a peticion")
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestErroresSQLRedactados(t *testing.T) {
	casos := []struct {
		codigo string
		quiere error
	}{{"P1409", ports.ErrConflicto}, {"22023", ports.ErrPeticionInvalida}, {"42501", ports.ErrProhibido}, {"55000", ports.ErrNoDisponible}, {"08006", ports.ErrNoDisponible}}
	for _, caso := range casos {
		err := errorSeguro(context.Background(), &pgconn.PgError{Code: caso.codigo, Message: "persona_ref privada"})
		if !errors.Is(err, caso.quiere) || strings.Contains(err.Error(), "privada") {
			t.Fatalf("SQLSTATE %s: %v", caso.codigo, err)
		}
	}
}
