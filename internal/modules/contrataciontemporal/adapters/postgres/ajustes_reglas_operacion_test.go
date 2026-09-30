package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	app "vec-diputacion-granada/internal/modules/contrataciontemporal/application/ajustesreglas"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	vecpruebas "vec-diputacion-granada/internal/vec/pruebas"
)

type filaHuellaAjustesCTPrueba struct{ huella string }

func (f filaHuellaAjustesCTPrueba) Scan(dest ...any) error {
	*dest[0].(*string) = f.huella
	return nil
}

type ejecutorAjustesCTPrueba struct {
	consultas     int
	transacciones int
}

func (e *ejecutorAjustesCTPrueba) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	e.consultas++
	if sql != huellaMaterialAjustesCTSQL || len(args) != 1 {
		panic("consulta fuera de la huella de material")
	}
	return filaHuellaAjustesCTPrueba{huella: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}
}

func (e *ejecutorAjustesCTPrueba) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	e.transacciones++
	return nil, errors.New("no debe empezar transacción")
}

type proveedorAjustesCTPrueba struct {
	consultas int
	accion    string
}

func (p *proveedorAjustesCTPrueba) AutorizarAjustesReglasCT(_ context.Context, _ vecdomain.ContextoActor, accion string, recurso vecdomain.RecursoAutorizable) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.consultas++
	p.accion = accion
	if recurso.Referencia != recursoAjustesReglasCT || recurso.ModuloID != "contratacion_temporal" || recurso.Tipo != "catalogo_reglas" ||
		recurso.Ambitos["organizacion_ref"] != organizacionAjustesReglasCT || recurso.Atributos["material_sha256"] != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		panic("recurso de autorización divergente")
	}
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, vecdomain.ErrAutorizacionDenegada
}

func (*proveedorAjustesCTPrueba) ComprobarCapacidadAjustesReglasCT(context.Context, vecdomain.ContextoActor, string, vecdomain.RecursoAutorizable) (bool, error) {
	return false, nil
}

func actorAjustesCTPrueba(t *testing.T) vecdomain.ContextoActor {
	t.Helper()
	actor, _, err := vecpruebas.NuevoContextoYVinculo(time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC),
		"per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", vecdomain.AuthMethodCertificate, vecdomain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	return actor
}

func TestOperacionAjustesCTDeniegaAntesDeTransaccionYNoUsaAmbitoDelCliente(t *testing.T) {
	pool := &ejecutorAjustesCTPrueba{}
	proveedor := &proveedorAjustesCTPrueba{}
	r, err := nuevoRepositorioAjustesReglasCT(pool, proveedor, organizacionAjustesReglasCT)
	if err != nil {
		t.Fatal(err)
	}
	material := app.Material{Operacion: "ajustar", CatalogoID: catalogoAjustesCT, ClaveIdempotencia: "00000000-0000-4000-8000-000000000001"}
	_, err = r.Operar(t.Context(), actorAjustesCTPrueba(t), material)
	if !errors.Is(err, vecdomain.ErrAutorizacionDenegada) || proveedor.consultas != 1 || proveedor.accion != "contratacion_temporal.reglas.ajustar" || pool.consultas != 1 || pool.transacciones != 0 {
		t.Fatalf("denegación no cierra antes del efecto: %v, %+v, %+v", err, proveedor, pool)
	}
	material.OrganizacionRef = "organizacion:otra"
	_, err = r.Operar(t.Context(), actorAjustesCTPrueba(t), material)
	if !errors.Is(err, app.ErrEntradaInvalida) || proveedor.consultas != 1 || pool.consultas != 1 {
		t.Fatalf("ámbito de cliente alcanzó autorización: %v", err)
	}
}

func TestOperacionAjustesCTNormalizaConflictosYNoFiltraDetalle(t *testing.T) {
	for _, codigo := range []string{"40001", "55P03", "23505"} {
		t.Run(codigo, func(t *testing.T) {
			err := normalizarErrorOperacionAjustesCT(t.Context(), &pgconn.PgError{Code: codigo, Message: "dato privado"})
			if !errors.Is(err, ErrOperacionAjustesReglasConflicto) || !errors.Is(errorAjustesCTAplicacion(err), app.ErrConflicto) || err.Error() == "dato privado" {
				t.Fatalf("conflicto mal clasificado: %v", err)
			}
		})
	}
	ctx, cancelar := context.WithCancel(t.Context())
	cancelar()
	if err := normalizarErrorOperacionAjustesCT(ctx, &pgconn.PgError{Code: "40001"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación perdió precedencia: %v", err)
	}
}

func TestOperacionAjustesCTDistingueCaidaV3DeDenegacion(t *testing.T) {
	caida := fmt.Errorf("detalle privado del registro: %w", errors.Join(
		vecdomain.ErrAutorizacionDenegada,
		vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible,
	))
	normalizada := normalizarErrorOperacionAjustesCT(t.Context(), caida)
	if !errors.Is(normalizada, ErrOperacionAjustesReglasNoDisponible) ||
		errors.Is(normalizada, ErrOperacionAjustesReglasDenegada) ||
		!errors.Is(errorAjustesCTAplicacion(normalizada), app.ErrNoDisponible) ||
		strings.Contains(normalizada.Error(), "detalle privado") {
		t.Fatalf("caída V3 expuesta o clasificada como denegación: %v", normalizada)
	}
	denegada := normalizarErrorOperacionAjustesCT(t.Context(), vecdomain.ErrAutorizacionDenegada)
	if !errors.Is(denegada, ErrOperacionAjustesReglasDenegada) ||
		errors.Is(denegada, ErrOperacionAjustesReglasNoDisponible) ||
		!errors.Is(errorAjustesCTAplicacion(denegada), vecdomain.ErrAutorizacionDenegada) {
		t.Fatalf("denegación real no conservada: %v", denegada)
	}
}

func TestOperacionAjustesCTHuellaYConsultaSinRepositorio(t *testing.T) {
	canonico := []byte(`{"c03.plazo_fiscalizacion":{"cantidad":"7","unidad":"dias_habiles"}}`)
	suma := sha256.Sum256(canonico)
	ajustes := map[string]map[string]string{"c03.plazo_fiscalizacion": {"cantidad": "7", "unidad": "dias_habiles"}}
	if err := validarHuellaAjustesCT(ajustes, hex.EncodeToString(suma[:])); err != nil {
		t.Fatalf("huella válida: %v", err)
	}
	if err := validarHuellaAjustesCT(ajustes, "otra"); !errors.Is(err, app.ErrNoDisponible) {
		t.Fatalf("huella divergente no cerrada: %v", err)
	}
	invalido := map[string]map[string]string{"c03.plazo_fiscalizacion": {"campo_desconocido": "dato_privado"}}
	if err := validarHuellaAjustesCT(invalido, hex.EncodeToString(suma[:])); !errors.Is(err, app.ErrNoDisponible) || strings.Contains(err.Error(), "dato_privado") {
		t.Fatalf("fallo de canonización no propagado o detalle expuesto: %v", err)
	}
	if !huellaSHA256CTValida(hex.EncodeToString(suma[:])) || huellaSHA256CTValida("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF") {
		t.Fatal("huella de base inválida aceptada")
	}
	var r *RepositorioAjustesReglasCT
	if _, err := r.Consultar(t.Context(), actorAjustesCTPrueba(t), 50, nil); !errors.Is(err, app.ErrNoDisponible) {
		t.Fatalf("consulta nula: %v", err)
	}
	pool := &ejecutorAjustesCTPrueba{}
	proveedor := &proveedorAjustesCTPrueba{}
	r, _ = nuevoRepositorioAjustesReglasCT(pool, proveedor, organizacionAjustesReglasCT)
	if _, err := r.Consultar(t.Context(), actorAjustesCTPrueba(t), 51, nil); !errors.Is(err, app.ErrEntradaInvalida) || pool.consultas != 0 {
		t.Fatalf("límite inválido llegó a SQL: %v", err)
	}
}
