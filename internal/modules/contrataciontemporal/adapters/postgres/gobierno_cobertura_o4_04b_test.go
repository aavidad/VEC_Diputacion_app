package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

type filaGobiernoCoberturaO404BPrueba struct {
	catalogo  string
	politica  string
	actuacion string
}

type filasGobiernoCoberturaO404BPrueba struct {
	pgx.Rows
	filas  []filaGobiernoCoberturaO404BPrueba
	indice int
	actual int
}

func (f *filasGobiernoCoberturaO404BPrueba) Next() bool {
	if f.indice >= len(f.filas) {
		return false
	}
	f.actual = f.indice
	f.indice++
	return true
}

func (f *filasGobiernoCoberturaO404BPrueba) Scan(destinos ...any) error {
	if len(destinos) != 3 {
		return errors.New("destinos de gobierno O4-04B inesperados")
	}
	catalogo, okCatalogo := destinos[0].(*string)
	politica, okPolitica := destinos[1].(*string)
	actuacion, okActuacion := destinos[2].(*string)
	if !okCatalogo || !okPolitica || !okActuacion {
		return errors.New("tipos de gobierno O4-04B inesperados")
	}
	fila := f.filas[f.actual]
	*catalogo, *politica, *actuacion =
		fila.catalogo, fila.politica, fila.actuacion
	return nil
}

func (f *filasGobiernoCoberturaO404BPrueba) Close()     {}
func (f *filasGobiernoCoberturaO404BPrueba) Err() error { return nil }

type transaccionGobiernoCoberturaO404BPrueba struct {
	pgx.Tx
	filas          pgx.Rows
	consulta       string
	argumentos     []any
	configurada    bool
	confirmaciones int
}

func (t *transaccionGobiernoCoberturaO404BPrueba) Exec(
	context.Context,
	string,
	...any,
) (pgconn.CommandTag, error) {
	t.configurada = true
	return pgconn.CommandTag{}, nil
}

func (t *transaccionGobiernoCoberturaO404BPrueba) Query(
	_ context.Context,
	consulta string,
	argumentos ...any,
) (pgx.Rows, error) {
	t.consulta = consulta
	t.argumentos = append([]any(nil), argumentos...)
	return t.filas, nil
}

func (t *transaccionGobiernoCoberturaO404BPrueba) Commit(
	context.Context,
) error {
	t.confirmaciones++
	return nil
}

func (t *transaccionGobiernoCoberturaO404BPrueba) Rollback(
	context.Context,
) error {
	return nil
}

type iniciadorGobiernoCoberturaO404BPrueba struct {
	tx       pgx.Tx
	opciones pgx.TxOptions
}

func (i *iniciadorGobiernoCoberturaO404BPrueba) BeginTx(
	_ context.Context,
	opciones pgx.TxOptions,
) (pgx.Tx, error) {
	i.opciones = opciones
	return i.tx, nil
}

func TestResolutorGobiernoCoberturaO404BLeePublicacionExacta(t *testing.T) {
	instante := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	fixture := nuevoFixtureGobiernoCoberturaO404B(t, instante)
	catalogo, politica, actuacion := fixture.JSON(t)
	filas := &filasGobiernoCoberturaO404BPrueba{
		filas: []filaGobiernoCoberturaO404BPrueba{{
			catalogo: catalogo, politica: politica, actuacion: actuacion,
		}},
	}
	tx := &transaccionGobiernoCoberturaO404BPrueba{filas: filas}
	iniciador := &iniciadorGobiernoCoberturaO404BPrueba{tx: tx}
	resolutor, err := nuevoResolutorGobiernoCoberturaO404BPostgreSQL(iniciador)
	if err != nil {
		t.Fatal(err)
	}
	publicacion, err := resolutor.resolverEnTransaccion(
		context.Background(),
		"organizacion:dipgra",
		"expediente:ct:o404b:01",
		2,
		"contratacion_temporal.cobertura.decidir",
		instante,
	)
	if err != nil ||
		publicacion.Catalogo.Identidad() != fixture.catalogo.Identidad() ||
		publicacion.Politica.Identidad() != fixture.politica.Identidad() ||
		iniciador.opciones.IsoLevel != pgx.Serializable ||
		iniciador.opciones.AccessMode != pgx.ReadOnly ||
		!tx.configurada || tx.confirmaciones != 1 ||
		!strings.Contains(
			tx.consulta,
			funcionResolverGobiernoCoberturaO404B,
		) ||
		len(tx.argumentos) != 5 {
		t.Fatalf(
			"resolución inesperada: %#v err=%v tx=%#v",
			publicacion,
			err,
			tx,
		)
	}
}

func TestResolutorGobiernoCoberturaO404BFallaCerradoAnteSalidaNoConfiable(
	t *testing.T,
) {
	instante := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	fixture := nuevoFixtureGobiernoCoberturaO404B(t, instante)
	catalogo, politica, actuacion := fixture.JSON(t)
	casos := []struct {
		nombre string
		filas  []filaGobiernoCoberturaO404BPrueba
	}{
		{
			nombre: "json malformado",
			filas: []filaGobiernoCoberturaO404BPrueba{{
				catalogo: "{", politica: politica, actuacion: actuacion,
			}},
		},
		{
			nombre: "campo desconocido",
			filas: []filaGobiernoCoberturaO404BPrueba{{
				catalogo: strings.Replace(
					catalogo, "{", `{"dato_ajeno":true,`, 1,
				),
				politica:  politica,
				actuacion: actuacion,
			}},
		},
		{
			nombre: "dos filas",
			filas: []filaGobiernoCoberturaO404BPrueba{
				{catalogo: catalogo, politica: politica, actuacion: actuacion},
				{catalogo: catalogo, politica: politica, actuacion: actuacion},
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &transaccionGobiernoCoberturaO404BPrueba{
				filas: &filasGobiernoCoberturaO404BPrueba{
					filas: caso.filas,
				},
			}
			resolutor, err := nuevoResolutorGobiernoCoberturaO404BPostgreSQL(
				&iniciadorGobiernoCoberturaO404BPrueba{tx: tx},
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = resolutor.resolverEnTransaccion(
				context.Background(),
				"organizacion:dipgra",
				"expediente:ct:o404b:01",
				2,
				"contratacion_temporal.cobertura.decidir",
				instante,
			)
			if !errors.Is(
				err,
				cobertura.ErrGobiernoOperacionCoberturaNoConfiable,
			) || tx.confirmaciones != 0 {
				t.Fatalf(
					"salida no confiable no preservada: %v",
					err,
				)
			}
		})
	}
}
