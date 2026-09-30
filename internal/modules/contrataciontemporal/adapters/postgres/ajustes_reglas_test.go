package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/reglas"
)

type filaAjustesReglasPrueba struct {
	version  int64
	huella   string
	canonico string
	desde    time.Time
	err      error
	cancelar context.CancelFunc
}

func (f filaAjustesReglasPrueba) Scan(dest ...any) error {
	if f.cancelar != nil {
		f.cancelar()
	}
	if f.err != nil {
		return f.err
	}
	*dest[0].(*int64) = f.version
	*dest[1].(*string) = f.huella
	*dest[2].(*string) = f.canonico
	*dest[3].(*time.Time) = f.desde
	return nil
}

func TestConsultaAjustesReglasCTConflictosSQLNominales(t *testing.T) {
	instante := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	for _, codigo := range []string{"55P03", "40001"} {
		t.Run(codigo, func(t *testing.T) {
			causa := fmt.Errorf("dsn-prueba: %w", &pgconn.PgError{Code: codigo, Message: "detalle privado"})
			proveedor := &consultadorAjustesReglasPrueba{t: t, instante: instante, fila: filaAjustesReglasPrueba{err: causa}}
			consulta, err := nuevaConsultaAjustesReglasPostgreSQL(proveedor)
			if err != nil {
				t.Fatal(err)
			}
			version, encontrada, err := consulta.AjustesVigentesEn(t.Context(), catalogoAjustesCT, instante)
			if encontrada || version.CatalogoID != "" || version.Version != 0 || version.HuellaSHA256 != "" ||
				!version.VigenteDesde.IsZero() || version.Ajustes != nil || proveedor.consultas != 1 ||
				!errors.Is(err, reglas.ErrAjustesConflicto) || errors.Is(err, reglas.ErrAjustesNoDisponibles) ||
				strings.Contains(err.Error(), "dsn-prueba") || strings.Contains(err.Error(), "detalle privado") {
				t.Fatalf("conflicto SQL expuesto o mal clasificado: %+v, encontrada=%v, err=%v", version, encontrada, err)
			}
		})
	}
}

func TestConsultaAjustesReglasCTCancelacionPrecedeConflictoSQL(t *testing.T) {
	instante := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	ctx, cancelar := context.WithCancel(t.Context())
	defer cancelar()
	proveedor := &consultadorAjustesReglasPrueba{t: t, instante: instante, fila: filaAjustesReglasPrueba{
		err: &pgconn.PgError{Code: "55P03"}, cancelar: cancelar,
	}}
	consulta, err := nuevaConsultaAjustesReglasPostgreSQL(proveedor)
	if err != nil {
		t.Fatal(err)
	}
	_, encontrada, err := consulta.AjustesVigentesEn(ctx, catalogoAjustesCT, instante)
	if encontrada || !errors.Is(err, context.Canceled) || errors.Is(err, reglas.ErrAjustesConflicto) {
		t.Fatalf("cancelación perdió precedencia: %v", err)
	}
}

type consultadorAjustesReglasPrueba struct {
	t         *testing.T
	instante  time.Time
	fila      filaAjustesReglasPrueba
	consultas int
}

func (c *consultadorAjustesReglasPrueba) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	c.t.Helper()
	c.consultas++
	if sql != consultaAjustesReglasEnV1 || len(args) != 2 || args[0] != catalogoAjustesCT || args[1] != c.instante {
		c.t.Fatalf("la consulta debe usar la función CT-148 y el instante exacto: %q, %v", sql, args)
	}
	limite, ok := ctx.Deadline()
	if !ok || time.Until(limite) <= 0 || time.Until(limite) > limiteConsultaAjustesCT {
		c.t.Fatalf("consulta sin límite temporal propio: %v", limite)
	}
	return c.fila
}

func TestConsultaAjustesReglasCTVersionEnInstante(t *testing.T) {
	instante := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	canonico := `{"c03.plazo_fiscalizacion":{"cantidad":"7","unidad":"dias_habiles"}}`
	huella, err := reglas.HuellaAjustes(map[string]map[string]string{
		"c03.plazo_fiscalizacion": {"cantidad": "7", "unidad": "dias_habiles"},
	})
	if err != nil {
		t.Fatal(err)
	}
	proveedor := &consultadorAjustesReglasPrueba{t: t, instante: instante,
		fila: filaAjustesReglasPrueba{version: 2, huella: huella, canonico: canonico, desde: instante.Add(-time.Minute)}}
	consulta, err := nuevaConsultaAjustesReglasPostgreSQL(proveedor)
	if err != nil {
		t.Fatal(err)
	}
	v, encontrada, err := consulta.AjustesVigentesEn(t.Context(), catalogoAjustesCT, instante)
	if err != nil || !encontrada || proveedor.consultas != 1 || v.CatalogoID != catalogoAjustesCT ||
		v.Version != 2 || v.HuellaSHA256 != huella || !v.VigenteDesde.Equal(instante.Add(-time.Minute)) ||
		v.Ajustes["c03.plazo_fiscalizacion"]["cantidad"] != "7" {
		t.Fatalf("versión resuelta inesperada: %+v, encontrada=%v, err=%v", v, encontrada, err)
	}
}

func TestConsultaAjustesReglasCTSinFilasYCierreAnteDatosInvalidos(t *testing.T) {
	instante := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	huellaVacia, err := reglas.HuellaAjustes(nil)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre   string
		fila     filaAjustesReglasPrueba
		sinFilas bool
	}{
		{nombre: "sin versiones", fila: filaAjustesReglasPrueba{err: pgx.ErrNoRows}, sinFilas: true},
		{nombre: "error SQL", fila: filaAjustesReglasPrueba{err: errors.New("detalle de base")}},
		{nombre: "version cero", fila: filaAjustesReglasPrueba{canonico: `{}`, huella: huellaVacia, desde: instante}},
		{nombre: "version fuera de CT-148", fila: filaAjustesReglasPrueba{version: maximoVersionAjustesCT + 1, canonico: `{}`, huella: huellaVacia, desde: instante}},
		{nombre: "vigencia futura", fila: filaAjustesReglasPrueba{version: 1, canonico: `{}`, huella: huellaVacia, desde: instante.Add(time.Second)}},
		{nombre: "huella ajena", fila: filaAjustesReglasPrueba{version: 1, canonico: `{}`, huella: strings.Repeat("a", 64), desde: instante}},
		{nombre: "JSON no canónico", fila: filaAjustesReglasPrueba{version: 1, canonico: `{ }`, huella: huellaVacia, desde: instante}},
		{nombre: "JSON nulo", fila: filaAjustesReglasPrueba{version: 1, canonico: `null`, huella: huellaVacia, desde: instante}},
		{nombre: "exceso de tamaño", fila: filaAjustesReglasPrueba{version: 1, canonico: strings.Repeat("x", maximoCanonicoAjustesCT+1), huella: huellaVacia, desde: instante}},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			proveedor := &consultadorAjustesReglasPrueba{t: t, instante: instante, fila: tc.fila}
			consulta, err := nuevaConsultaAjustesReglasPostgreSQL(proveedor)
			if err != nil {
				t.Fatal(err)
			}
			v, encontrada, err := consulta.AjustesVigentesEn(t.Context(), catalogoAjustesCT, instante)
			if v.CatalogoID != "" || v.Version != 0 || v.HuellaSHA256 != "" || !v.VigenteDesde.IsZero() || v.Ajustes != nil ||
				encontrada || proveedor.consultas != 1 {
				t.Fatalf("lectura inválida expuesta: %+v, encontrada=%v", v, encontrada)
			}
			if tc.sinFilas {
				if err != nil {
					t.Fatalf("sin filas: %v", err)
				}
			} else if !errors.Is(err, reglas.ErrAjustesNoDisponibles) || (err != nil && strings.Contains(err.Error(), "detalle de base")) {
				t.Fatalf("error sin cierre: %v", err)
			}
		})
	}
}

func TestConsultaAjustesReglasCTRechazaEntradaYRespetaCancelacion(t *testing.T) {
	instante := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	proveedor := &consultadorAjustesReglasPrueba{t: t, instante: instante, fila: filaAjustesReglasPrueba{err: pgx.ErrNoRows}}
	consulta, err := nuevaConsultaAjustesReglasPostgreSQL(proveedor)
	if err != nil {
		t.Fatal(err)
	}
	for _, entrada := range []struct {
		ctx      context.Context
		catalogo string
		instante time.Time
	}{
		{nil, catalogoAjustesCT, instante},
		{t.Context(), "vec.bolsa.reglas.ajustes", instante},
		{t.Context(), catalogoAjustesCT, time.Time{}},
	} {
		_, encontrada, err := consulta.AjustesVigentesEn(entrada.ctx, entrada.catalogo, entrada.instante)
		if encontrada || !errors.Is(err, reglas.ErrAjustesNoDisponibles) || proveedor.consultas != 0 {
			t.Fatalf("entrada inválida pasó a SQL: %v", err)
		}
	}
	ctx, cancelar := context.WithCancel(t.Context())
	cancelar()
	_, encontrada, err := consulta.AjustesVigentesEn(ctx, catalogoAjustesCT, instante)
	if encontrada || !errors.Is(err, context.Canceled) || proveedor.consultas != 0 {
		t.Fatalf("cancelación no respetada: %v", err)
	}
	if _, err := nuevaConsultaAjustesReglasPostgreSQL(nil); !errors.Is(err, reglas.ErrAjustesNoDisponibles) {
		t.Fatalf("dependencia nula: %v", err)
	}
}
