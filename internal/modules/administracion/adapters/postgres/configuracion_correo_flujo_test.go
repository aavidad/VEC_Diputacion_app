package postgres

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	admindomain "vec-diputacion-granada/internal/modules/administracion/domain"
	adminports "vec-diputacion-granada/internal/modules/administracion/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestRegistroConfiguracionCorreoFlujoCifraAutorizaYGuardaSinTextoClaro(t *testing.T) {
	traza := make([]string, 0, 3)
	tx := &transaccionConfiguracionCorreoFlujoPrueba{fila: []byte(jsonVistaConfiguracionCorreoFlujoPrueba())}
	pool := &iniciadorConfiguracionCorreoFlujoPrueba{tx: tx}
	protector := &protectorConfiguracionCorreoFlujoPrueba{traza: &traza}
	autorizador := &autorizadorConfiguracionCorreoFlujoPrueba{traza: &traza}
	registro := &RegistroConfiguracionCorreoPostgreSQL{pool: pool, protector: protector}

	preparacion, err := registro.PrepararConfiguracionCorreo(context.Background(), actualizacionConfiguracionCorreoFlujoPrueba(4, "secreto-smtp-sintetico"), auditoriaConfiguracionCorreoFlujoPrueba())
	if err != nil {
		t.Fatalf("preparacion rechazada: %v", err)
	}
	material, err := autorizador.AutorizarConfiguracionCorreo(context.Background(), preparacion, auditoriaConfiguracionCorreoFlujoPrueba())
	if err != nil {
		t.Fatalf("autorizacion rechazada: %v", err)
	}
	vista, err := registro.GuardarConfiguracionCorreo(context.Background(), adminports.OrdenConfiguracionCorreoAutorizada{Preparacion: preparacion, Material: material})
	if err != nil || vista.Version != 5 || strings.Join(traza, ",") != "cifrar,autorizar" || tx.consultas != 1 || tx.commits != 1 || len(pool.opciones) != 1 || pool.opciones[0] != (pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite}) {
		t.Fatalf("flujo durable inesperado: vista=%+v err=%v traza=%v consultas=%d commits=%d", vista, err, traza, tx.consultas, tx.commits)
	}
	if autorizador.preparacion.Sustituir == false || autorizador.preparacion.SobreNuevo.Version != 5 || autorizador.preparacion.HuellaAADSHA256 == "" {
		t.Fatalf("la autorizacion no quedo ligada al sobre nuevo: %+v", autorizador.preparacion)
	}
	if len(tx.argumentos) != 9 || bytes.Contains(tx.argumentos[0], []byte("secreto-smtp-sintetico")) || bytes.Contains(autorizador.preparacion.SobreNuevo.Cifrado, []byte("secreto-smtp-sintetico")) {
		t.Fatal("texto claro alcanzó la frontera durable")
	}
}

func TestRegistroConfiguracionCorreoConservaSecretoNilYCAS(t *testing.T) {
	tx := &transaccionConfiguracionCorreoFlujoPrueba{fila: []byte(jsonVistaConfiguracionCorreoFlujoPrueba())}
	pool := &iniciadorConfiguracionCorreoFlujoPrueba{tx: tx}
	protector := &protectorConfiguracionCorreoFlujoPrueba{}
	autorizador := &autorizadorConfiguracionCorreoFlujoPrueba{}
	registro := &RegistroConfiguracionCorreoPostgreSQL{pool: pool, protector: protector}

	preparacion, err := registro.PrepararConfiguracionCorreo(context.Background(), actualizacionConfiguracionCorreoFlujoPrueba(4, ""), auditoriaConfiguracionCorreoFlujoPrueba())
	if err != nil {
		t.Fatalf("preparacion nil rechazada: %v", err)
	}
	material, err := autorizador.AutorizarConfiguracionCorreo(context.Background(), preparacion, auditoriaConfiguracionCorreoFlujoPrueba())
	if err != nil {
		t.Fatalf("autorizacion nil rechazada: %v", err)
	}
	if _, err := registro.GuardarConfiguracionCorreo(context.Background(), adminports.OrdenConfiguracionCorreoAutorizada{Preparacion: preparacion, Material: material}); err != nil {
		t.Fatalf("actualizacion sin secreto existente rechazada: %v", err)
	}
	if protector.cifrados != 0 || autorizador.preparacion.Sustituir || autorizador.preparacion.SobreNuevo.Version != 0 || autorizador.preparacion.Entrada.VersionEsperada != 4 {
		t.Fatalf("nil no conservó el secreto/CAS: cifrados=%d preparacion=%+v", protector.cifrados, autorizador.preparacion)
	}
}

func TestRegistroConfiguracionCorreoNoEscribeSiFallaAutorizacionYPropagaFalloCommit(t *testing.T) {
	t.Run("cifrado", func(t *testing.T) {
		pool := &iniciadorConfiguracionCorreoFlujoPrueba{tx: &transaccionConfiguracionCorreoFlujoPrueba{}}
		registro := &RegistroConfiguracionCorreoPostgreSQL{pool: pool, protector: &protectorConfiguracionCorreoFlujoPrueba{err: errors.New("kms caido")}}
		if _, err := registro.PrepararConfiguracionCorreo(context.Background(), actualizacionConfiguracionCorreoFlujoPrueba(4, "secreto-sintetico"), auditoriaConfiguracionCorreoFlujoPrueba()); !errors.Is(err, ErrConfiguracionCorreoNoDisponible) || pool.inicios != 0 {
			t.Fatalf("un fallo de cifrado no debe abrir escritura: err=%v inicios=%d", err, pool.inicios)
		}
	})
	t.Run("autorizacion", func(t *testing.T) {
		pool := &iniciadorConfiguracionCorreoFlujoPrueba{tx: &transaccionConfiguracionCorreoFlujoPrueba{}}
		registro := &RegistroConfiguracionCorreoPostgreSQL{pool: pool, protector: &protectorConfiguracionCorreoFlujoPrueba{}}
		preparacion, err := registro.PrepararConfiguracionCorreo(context.Background(), actualizacionConfiguracionCorreoFlujoPrueba(4, "secreto-sintetico"), auditoriaConfiguracionCorreoFlujoPrueba())
		autorizador := &autorizadorConfiguracionCorreoFlujoPrueba{err: errors.New("denegada")}
		_, err = autorizador.AutorizarConfiguracionCorreo(context.Background(), preparacion, auditoriaConfiguracionCorreoFlujoPrueba())
		if err == nil || pool.inicios != 0 {
			t.Fatalf("un fallo de autorizacion no debe escribir: err=%v inicios=%d", err, pool.inicios)
		}
	})
	t.Run("commit", func(t *testing.T) {
		tx := &transaccionConfiguracionCorreoFlujoPrueba{fila: []byte(jsonVistaConfiguracionCorreoFlujoPrueba()), errCommit: errors.New("commit caido")}
		pool := &iniciadorConfiguracionCorreoFlujoPrueba{tx: tx}
		registro := &RegistroConfiguracionCorreoPostgreSQL{pool: pool, protector: &protectorConfiguracionCorreoFlujoPrueba{}}
		preparacion, err := registro.PrepararConfiguracionCorreo(context.Background(), actualizacionConfiguracionCorreoFlujoPrueba(4, ""), auditoriaConfiguracionCorreoFlujoPrueba())
		if err != nil {
			t.Fatal(err)
		}
		material, err := (&autorizadorConfiguracionCorreoFlujoPrueba{}).AutorizarConfiguracionCorreo(context.Background(), preparacion, auditoriaConfiguracionCorreoFlujoPrueba())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := registro.GuardarConfiguracionCorreo(context.Background(), adminports.OrdenConfiguracionCorreoAutorizada{Preparacion: preparacion, Material: material}); !errors.Is(err, ErrConfiguracionCorreoNoDisponible) || tx.commits != 1 {
			t.Fatalf("fallo de commit no se propago cerrado: err=%v commits=%d", err, tx.commits)
		}
	})
	t.Run("conflicto SQL", func(t *testing.T) {
		tx := &transaccionConfiguracionCorreoFlujoPrueba{errScan: &pgconn.PgError{Code: "P0409"}}
		pool := &iniciadorConfiguracionCorreoFlujoPrueba{tx: tx}
		registro := &RegistroConfiguracionCorreoPostgreSQL{pool: pool, protector: &protectorConfiguracionCorreoFlujoPrueba{}}
		preparacion, err := registro.PrepararConfiguracionCorreo(context.Background(), actualizacionConfiguracionCorreoFlujoPrueba(4, ""), auditoriaConfiguracionCorreoFlujoPrueba())
		if err != nil {
			t.Fatal(err)
		}
		material, err := (&autorizadorConfiguracionCorreoFlujoPrueba{}).AutorizarConfiguracionCorreo(context.Background(), preparacion, auditoriaConfiguracionCorreoFlujoPrueba())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := registro.GuardarConfiguracionCorreo(context.Background(), adminports.OrdenConfiguracionCorreoAutorizada{Preparacion: preparacion, Material: material}); !errors.Is(err, adminports.ErrConfiguracionCorreoConflicto) || tx.commits != 0 {
			t.Fatalf("P0409 debe conservar conflicto y no confirmar: err=%v commits=%d", err, tx.commits)
		}
	})
}

type iniciadorConfiguracionCorreoFlujoPrueba struct {
	tx       pgx.Tx
	inicios  int
	opciones []pgx.TxOptions
}

func (i *iniciadorConfiguracionCorreoFlujoPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	i.inicios++
	i.opciones = append(i.opciones, opciones)
	return i.tx, nil
}

type transaccionConfiguracionCorreoFlujoPrueba struct {
	pgx.Tx
	fila               []byte
	argumentos         [][]byte
	consultas, commits int
	errCommit          error
	errScan            error
}

func (t *transaccionConfiguracionCorreoFlujoPrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("SET"), nil
}
func (t *transaccionConfiguracionCorreoFlujoPrueba) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	t.consultas++
	for _, arg := range args {
		if datos, ok := arg.([]byte); ok {
			t.argumentos = append(t.argumentos, append([]byte(nil), datos...))
		}
	}
	return filaConfiguracionCorreoFlujoPrueba{datos: t.fila, err: t.errScan}
}
func (t *transaccionConfiguracionCorreoFlujoPrueba) Commit(context.Context) error {
	t.commits++
	return t.errCommit
}
func (*transaccionConfiguracionCorreoFlujoPrueba) Rollback(context.Context) error { return nil }

type filaConfiguracionCorreoFlujoPrueba struct {
	datos []byte
	err   error
}

func (f filaConfiguracionCorreoFlujoPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*(destinos[0].(*[]byte)) = append([]byte(nil), f.datos...)
	return nil
}

type protectorConfiguracionCorreoFlujoPrueba struct {
	cifrados int
	traza    *[]string
	err      error
	aad      []byte
}

func (p *protectorConfiguracionCorreoFlujoPrueba) CifrarSecretoCorreo(_ context.Context, claro, aad []byte) (SobreSecretoCorreo, error) {
	p.cifrados++
	p.aad = append([]byte(nil), aad...)
	if p.traza != nil {
		*p.traza = append(*p.traza, "cifrar")
	}
	if p.err != nil {
		return SobreSecretoCorreo{}, p.err
	}
	if len(claro) == 0 {
		return SobreSecretoCorreo{}, errors.New("claro vacio")
	}
	return SobreSecretoCorreo{Version: 5, ClaveRef: "kms:correo:v1", Nonce: bytes.Repeat([]byte{1}, 12), Cifrado: bytes.Repeat([]byte{2}, 16)}, nil
}
func (*protectorConfiguracionCorreoFlujoPrueba) ConSecretoCorreoDescifrado(context.Context, SobreSecretoCorreo, []byte, func([]byte) error) error {
	return errors.New("no usado")
}

type autorizadorConfiguracionCorreoFlujoPrueba struct {
	preparacion adminports.PreparacionConfiguracionCorreo
	err         error
	traza       *[]string
}

func (a *autorizadorConfiguracionCorreoFlujoPrueba) AutorizarConfiguracionCorreo(_ context.Context, preparacion adminports.PreparacionConfiguracionCorreo, _ vecdomain.AuditEntry) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.preparacion = preparacion
	if a.traza != nil {
		*a.traza = append(*a.traza, "autorizar")
	}
	if a.err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, a.err
	}
	return materialConfiguracionCorreoFlujoPrueba(), nil
}

func actualizacionConfiguracionCorreoFlujoPrueba(version uint64, secreto string) admindomain.ActualizacionConfiguracionCorreo {
	vista := admindomain.VistaConfiguracionCorreo{Configurada: true, Host: "smtp.intranet.local", Puerto: 465, NombreServidor: "smtp.intranet.local", ReferenciaCA: "ca:correo-interno:v1", RemitenteFijo: "rrhh@diputacion.example", Usuario: "rrhh-smtp", ModoTLS: admindomain.ModoTLSCorreoImplicito, ModoAutenticacion: admindomain.ModoAutenticacionCorreoXOAUTH2, TiempoMaximoMillis: 5000, SecretoConfigurado: true}
	actualizacion := admindomain.ActualizacionConfiguracionCorreo{VistaConfiguracionCorreo: vista, VersionEsperada: version}
	if secreto != "" {
		nuevo, err := admindomain.NuevoSecretoCorreo([]byte(secreto))
		if err != nil {
			panic(err)
		}
		actualizacion.SecretoNuevo = &nuevo
	}
	return actualizacion
}
func auditoriaConfiguracionCorreoFlujoPrueba() vecdomain.AuditEntry {
	return vecdomain.AuditEntry{ActorID: "admin-correo", Action: "administracion.configuracion_correo.actualizar", ModuleID: "vec.module.administracion", SubjectRef: referenciaConfiguracionCorreo, Result: "accepted", OccurredAt: time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)}
}
func jsonVistaConfiguracionCorreoFlujoPrueba() string {
	datos, _ := json.Marshal(admindomain.VistaConfiguracionCorreo{Configurada: true, Host: "smtp.intranet.local", Puerto: 465, NombreServidor: "smtp.intranet.local", ReferenciaCA: "ca:correo-interno:v1", RemitenteFijo: "rrhh@diputacion.example", Usuario: "rrhh-smtp", ModoTLS: admindomain.ModoTLSCorreoImplicito, ModoAutenticacion: admindomain.ModoAutenticacionCorreoXOAUTH2, TiempoMaximoMillis: 5000, SecretoConfigurado: true, Version: 5})
	return string(datos)
}
func materialConfiguracionCorreoFlujoPrueba() vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	emite := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:correo", strings.Repeat("a", 64), strings.Repeat("b", 64), "contexto:correo", strings.Repeat("c", 64), "administracion.configuracion_correo.actualizar", "configuracion:smtp:diputacion", strings.Repeat("d", 64), "vec.administracion.configuracion-correo.v1", emite, emite.Add(time.Second))
	if err != nil {
		panic(err)
	}
	privada := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{9}, ed25519.SeedSize))
	spki, err := x509.MarshalPKIXPublicKey(privada.Public())
	if err != nil {
		panic(err)
	}
	material, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("c"), 512), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload-ad3"), []byte("cose"), []byte("evidencia"), spki)
	if err != nil {
		panic(err)
	}
	return material
}
