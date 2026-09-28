package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

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
	errCommit error
	consultas []llamadaPrueba
	ajustes   []string
	commits   int
	rollbacks int
}

func (t *txPrueba) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	t.ajustes = append(t.ajustes, sql)
	return pgconn.CommandTag{}, nil
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

func repositorioPrueba(tx *txPrueba) *RegistroPreferenciasPostgreSQL {
	return &RegistroPreferenciasPostgreSQL{iniciar: func(context.Context) (transaccionPreferencias, error) { return tx, nil }}
}

type proveedorPrueba struct{}

func (proveedorPrueba) ProveerMaterialPreferencias(context.Context, ports.MaterialPreferencias) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
}

func materialPrueba(t *testing.T, accion string) (ports.OrdenPreferencias, ports.MaterialPreferencias, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_0123456789abcdefghijkl", Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	snap := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 1, PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := vecdomain.NuevoContextoActor(cuenta, snap, ahora)
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenPreferencias(actor, proveedorPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	m := ports.MaterialPreferencias{PersonaRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, Accion: accion, FinalidadRef: ports.FinalidadPreferenciasPropias, CatalogoVersionRef: domain.CatalogoBasePreferencias().VersionRef, Valores: domain.CatalogoBasePreferencias().Predeterminados}
	if accion == ports.AccionActualizarPreferencias {
		m.ClaveOperacion = "operacion-1234567890"
		m.HuellaPeticion = strings.Repeat("a", 64)
	}
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), accion, actor.PersonaRef, strings.Repeat("d", 64), "usuarios_preferencias", ahora, ahora.Add(3*time.Second))
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
	base := domain.CatalogoBasePreferencias()
	opciones := func(lista []domain.OpcionPreferencia) []opcionSQL {
		salida := make([]opcionSQL, 0, len(lista))
		for _, o := range lista {
			salida = append(salida, opcionSQL{o.Codigo, o.NombreKey})
		}
		return salida
	}
	datos, err := json.Marshal(catalogoSQL{base.VersionRef, opciones(base.Idiomas), opciones(base.TamanosTexto), opciones(base.Temas), opciones(base.Inicios), base.Filas, base.Predeterminados})
	if err != nil {
		t.Fatal(err)
	}
	tx := &txPrueba{valido: true, respuesta: datos}
	c, err := repositorioPrueba(tx).CatalogoVigente(context.Background())
	if err != nil || !reflect.DeepEqual(c, base) || tx.commits != 1 || len(tx.ajustes) != 6 {
		t.Fatalf("catalogo/tx: %v %#v", err, c)
	}
	if len(tx.consultas) != 2 || tx.consultas[1].sql != consultarCatalogoSQL {
		t.Fatal("rol o funcion nominal no consultados")
	}
	tx = &txPrueba{valido: false, respuesta: datos}
	if _, err := repositorioPrueba(tx).CatalogoVigente(context.Background()); !errors.Is(err, ports.ErrNoDisponible) || len(tx.consultas) != 1 || tx.commits != 0 {
		t.Fatal("login no exclusivo paso la sonda")
	}
}

func TestConsultaEnviaMaterialLiteralYDiezPiezas(t *testing.T) {
	orden, m, v3 := materialPrueba(t, ports.AccionConsultarPreferencias)
	estado := ports.EstadoPreferencias{PersonaRef: m.PersonaRef, Version: 0, CatalogoVersionRef: m.CatalogoVersionRef, Valores: m.Valores}
	datos, _ := json.Marshal(struct {
		Existe bool `json:"existe"`
		ports.EstadoPreferencias
	}{false, estado})
	tx := &txPrueba{valido: true, respuesta: datos}
	obtenido, existe, err := repositorioPrueba(tx).ConsultarPropias(context.Background(), orden, m, v3)
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

func TestReplayConservaReciboYGuardarRespetaCAS(t *testing.T) {
	orden, m, v3 := materialPrueba(t, ports.AccionActualizarPreferencias)
	fecha := time.Date(2026, 9, 29, 7, 45, 0, 123000, time.UTC)
	recibo := ports.ReciboPreferencias{ReciboRef: "recibo:original", PersonaRef: m.PersonaRef, Version: 1, CatalogoVersionRef: m.CatalogoVersionRef, Valores: m.Valores, FechaUTC: fecha}
	datosGuardar, _ := json.Marshal(recibo)
	recibo.Replay = true
	datosReplay, _ := json.Marshal(recibo)
	tx := &txPrueba{valido: true, respuesta: datosReplay}
	recuperado, existe, err := repositorioPrueba(tx).RecuperarOperacion(context.Background(), orden, m, v3)
	if err != nil || !existe || !recuperado.Replay || recuperado.ReciboRef != recibo.ReciboRef || !recuperado.FechaUTC.Equal(fecha) || tx.commits != 1 || tx.consultas[1].sql != recuperarSQL {
		t.Fatalf("replay: %v, %#v", err, recuperado)
	}
	tx = &txPrueba{valido: true, respuesta: []byte("null")}
	_, existe, err = repositorioPrueba(tx).RecuperarOperacion(context.Background(), orden, m, v3)
	if err != nil || existe || tx.commits != 1 {
		t.Fatal("ausencia de replay incorrecta")
	}
	peticion := ports.PeticionGuardarPreferencias{VersionEsperada: m.VersionEsperada, CatalogoVersionRef: m.CatalogoVersionRef, ClaveOperacion: m.ClaveOperacion, Valores: m.Valores}
	tx = &txPrueba{valido: true, respuesta: datosGuardar}
	guardado, err := repositorioPrueba(tx).Guardar(context.Background(), orden, peticion, m, v3)
	if err != nil || guardado.Replay || tx.commits != 1 || tx.consultas[1].sql != guardarSQL || len(tx.consultas[1].args) != 12 {
		t.Fatalf("guardar: %v", err)
	}
	if !bytes.Equal(tx.consultas[1].args[1].([]byte), mustJSON(t, m.Valores)) {
		t.Fatal("valores SQL distintos del material")
	}
	peticion.ClaveOperacion = "otra-clave-123456789"
	if _, err := repositorioPrueba(&txPrueba{valido: true}).Guardar(context.Background(), orden, peticion, m, v3); !errors.Is(err, ports.ErrPeticionInvalida) {
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
