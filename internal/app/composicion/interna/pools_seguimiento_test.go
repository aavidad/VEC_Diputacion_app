package interna

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMaterialPoolsSeguimientoNoRevelaDSN(t *testing.T) {
	material := MaterialPoolSeguimiento{DSN: "postgres://secreto@db.invalid/vec", Login: "secreto"}
	conjunto := MaterialPoolsSeguimiento{AltaPersonal: material}
	for _, valor := range []any{material, conjunto} {
		for _, formato := range []string{"%v", "%+v", "%#v", "%s"} {
			texto := fmt.Sprintf(formato, valor)
			if strings.Contains(texto, "secreto") || strings.Contains(texto, "postgres://") {
				t.Fatalf("formato %s reveló material privado", formato)
			}
		}
		codificado, err := json.Marshal(valor)
		if err != nil || strings.Contains(string(codificado), "secreto") || !json.Valid(codificado) {
			t.Fatalf("JSON reveló material privado o falló: %v", err)
		}
		texto, err := valor.(interface{ MarshalText() ([]byte, error) }).MarshalText()
		if err != nil || strings.Contains(string(texto), "secreto") {
			t.Fatalf("texto reveló material privado: %v", err)
		}
		binario, err := valor.(interface{ MarshalBinary() ([]byte, error) }).MarshalBinary()
		if err != nil || strings.Contains(string(binario), "secreto") {
			t.Fatalf("binario reveló material privado: %v", err)
		}
		var salida bytes.Buffer
		if err := gob.NewEncoder(&salida).Encode(valor); err != nil || strings.Contains(salida.String(), "secreto") {
			t.Fatalf("gob reveló material privado: %v", err)
		}
		if strings.Contains(valor.(interface{ LogValue() slog.Value }).LogValue().String(), "secreto") {
			t.Fatal("slog reveló material privado")
		}
	}
	if err := json.Unmarshal([]byte(`"valor externo"`), &material); !errors.Is(err, ErrMaterialPoolSeguimientoNoSerializable) {
		t.Fatalf("material reconstruido desde JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(`"valor externo"`), &conjunto); !errors.Is(err, ErrMaterialPoolSeguimientoNoSerializable) {
		t.Fatalf("conjunto reconstruido desde JSON: %v", err)
	}
	if err := material.GobDecode([]byte("valor externo")); !errors.Is(err, ErrMaterialPoolSeguimientoNoSerializable) {
		t.Fatalf("material reconstruido desde gob: %v", err)
	}
	if err := conjunto.GobDecode([]byte("valor externo")); !errors.Is(err, ErrMaterialPoolSeguimientoNoSerializable) {
		t.Fatalf("conjunto reconstruido desde gob: %v", err)
	}
	for _, destino := range []interface {
		UnmarshalText([]byte) error
		UnmarshalBinary([]byte) error
	}{&material, &conjunto} {
		if err := destino.UnmarshalText([]byte("valor externo")); !errors.Is(err, ErrMaterialPoolSeguimientoNoSerializable) {
			t.Fatalf("material reconstruido desde texto: %v", err)
		}
		if err := destino.UnmarshalBinary([]byte("valor externo")); !errors.Is(err, ErrMaterialPoolSeguimientoNoSerializable) {
			t.Fatalf("material reconstruido desde binario: %v", err)
		}
	}
	anidado, err := json.Marshal(struct{ Pools MaterialPoolsSeguimiento }{Pools: conjunto})
	if err != nil || strings.Contains(string(anidado), "secreto") {
		t.Fatalf("JSON anidado reveló material privado: %v", err)
	}
}

func TestConfiguracionPoolsSeguimientoRechazaTLSNoVerificadoYLoginAjeno(t *testing.T) {
	base := perfilPoolSeguimiento{
		rol: "vec_personal_ejecutor", funcion: "vec_personal.registrar_alta_ejercicio_v1(jsonb)",
		aplicacion: "vec-interno-prueba", material: MaterialPoolSeguimiento{DSN: "postgres://login@db.invalid/vec?sslmode=verify-full", Login: "login"},
	}
	if _, err := configurarPoolSeguimiento(base); err != nil {
		t.Fatalf("verify-full: %v", err)
	}
	for _, dsn := range []string{
		"postgres://login@db.invalid/vec?sslmode=disable",
		"postgres://login@db.invalid/vec?sslmode=require",
		"postgres://login@db.invalid/vec?sslmode=prefer",
		"postgres://login@db.invalid/vec?sslmode=verify-ca",
	} {
		perfil := base
		perfil.material.DSN = dsn
		if _, err := configurarPoolSeguimiento(perfil); !errors.Is(err, ErrPoolsSeguimientoNoDisponibles) {
			t.Fatalf("se admitió TLS no verificado: %q", dsn)
		}
	}
	perfil := base
	perfil.material.Login = "otro"
	if _, err := configurarPoolSeguimiento(perfil); !errors.Is(err, ErrPoolsSeguimientoNoDisponibles) {
		t.Fatal("se admitió LOGIN distinto de la DSN")
	}
}

func TestTLSSeguimientoRechazaFallbackInseguro(t *testing.T) {
	seguro := &tls.Config{ServerName: "db.invalid", MinVersion: tls.VersionTLS13}
	configuracion := &pgconn.Config{Host: "db.invalid", TLSConfig: seguro}
	if !tlsPoolSeguimientoVerificado(configuracion) {
		t.Fatal("verify-full sin fallback fue rechazado")
	}
	configuracion.Fallbacks = []*pgconn.FallbackConfig{{Host: "db.invalid", TLSConfig: nil}}
	if tlsPoolSeguimientoVerificado(configuracion) {
		t.Fatal("fallback sin TLS fue admitido")
	}
	configuracion.Fallbacks[0].TLSConfig = &tls.Config{ServerName: "otro.invalid"}
	if tlsPoolSeguimientoVerificado(configuracion) {
		t.Fatal("fallback con nombre ajeno fue admitido")
	}
	configuracion.Fallbacks[0].TLSConfig = &tls.Config{ServerName: "db.invalid", InsecureSkipVerify: true}
	if tlsPoolSeguimientoVerificado(configuracion) {
		t.Fatal("fallback sin verificación fue admitido")
	}
}

type filaACLSeguimientoPrueba struct {
	sesion, efectivo string
	login, acl       bool
	err              error
}

func (f filaACLSeguimientoPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*destinos[0].(*string) = f.sesion
	*destinos[1].(*string) = f.efectivo
	*destinos[2].(*bool) = f.login
	*destinos[3].(*bool) = f.acl
	return nil
}

type consultadorACLSeguimientoPrueba struct{ fila filaACLSeguimientoPrueba }

func (c consultadorACLSeguimientoPrueba) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return c.fila
}

func TestPreflightPoolsSeguimientoDeniegaLoginYACLAjenas(t *testing.T) {
	perfil := perfilPoolSeguimiento{rol: "rol", funcion: "esquema.funcion()", material: MaterialPoolSeguimiento{Login: "login"}}
	valida := filaACLSeguimientoPrueba{sesion: "login", efectivo: "login", login: true, acl: true}
	if err := acreditarPoolSeguimiento(context.Background(), consultadorACLSeguimientoPrueba{valida}, perfil); err != nil {
		t.Fatalf("identidad y ACL nominales: %v", err)
	}
	casos := []filaACLSeguimientoPrueba{
		{sesion: "otro", efectivo: "otro", login: true, acl: true},
		{sesion: "login", efectivo: "propietario", login: true, acl: true},
		{sesion: "login", efectivo: "login", login: false, acl: true},
		{sesion: "login", efectivo: "login", login: true, acl: false},
		{err: errors.New("dependencia no disponible")},
	}
	for i, fila := range casos {
		if err := acreditarPoolSeguimiento(context.Background(), consultadorACLSeguimientoPrueba{fila}, perfil); !errors.Is(err, ErrPoolsSeguimientoNoDisponibles) {
			t.Fatalf("caso %d abrió con preflight fallido: %v", i, err)
		}
	}
}

func TestPoolsSeguimientoIncompletosNoAbrenConexion(t *testing.T) {
	salida, err := AbrirPoolsSeguimiento(context.Background(), MaterialPoolsSeguimiento{})
	if !errors.Is(err, ErrPoolsSeguimientoNoDisponibles) || len(poolsConsultaSeguimiento(salida)) != 11 {
		t.Fatalf("material incompleto: %v", err)
	}
	for _, pool := range poolsConsultaSeguimiento(salida) {
		if pool != nil {
			t.Fatal("se devolvió un pool parcial")
		}
	}
}
