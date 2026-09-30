package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	app "vec-diputacion-granada/internal/modules/contrataciontemporal/application/ajustesreglas"
	"vec-diputacion-granada/internal/vec/reglas"
)

type filaActivacionPrueba struct {
	estado                 string
	secuencia, version     *int64
	id, huella, aprobacion *string
	err                    error
}

func (f filaActivacionPrueba) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	*dest[0].(*string) = f.estado
	*dest[1].(**int64) = f.secuencia
	*dest[2].(**string) = f.id
	*dest[3].(**int64) = f.version
	*dest[4].(**string) = f.huella
	*dest[5].(**string) = f.aprobacion
	return nil
}

type ejecutorActivacionPrueba struct {
	fila      filaActivacionPrueba
	consultas int
}

func (e *ejecutorActivacionPrueba) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	e.consultas++
	if sql != leerActivacionBaseAjustesCTSQL || len(args) != 0 {
		panic("lectura fuera de fachada nominal CT158")
	}
	return e.fila
}
func (*ejecutorActivacionPrueba) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	panic("la lectura de activación no abre transacción de escritura")
}

func ptrActivacion[T any](v T) *T { return &v }

func TestActivacionAjustesCTLeeFachadaNominalYFallaCerrado(t *testing.T) {
	pool := &ejecutorActivacionPrueba{fila: filaActivacionPrueba{estado: "activa",
		secuencia: ptrActivacion(int64(2)), id: ptrActivacion(reglas.CatalogoContratacionTemporal),
		version: ptrActivacion(int64(1)), huella: ptrActivacion("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"),
		aprobacion: ptrActivacion("aprobacion:rrhh:sintetica")}}
	repo := &RepositorioAjustesReglasCT{pool: pool}
	got, err := repo.LeerActivacion(t.Context())
	if err != nil || got.Estado != "activa" || got.Version != 1 || got.Secuencia != 2 || pool.consultas != 1 {
		t.Fatalf("activación nominal no disponible: %+v, %v", got, err)
	}
	pool.fila.huella = ptrActivacion("huella-fuente-no-canonica")
	if _, err := repo.LeerActivacion(t.Context()); !errors.Is(err, app.ErrNoDisponible) {
		t.Fatalf("huella inválida aceptada: %v", err)
	}
	pool.fila = filaActivacionPrueba{estado: "inactiva", secuencia: ptrActivacion(int64(3))}
	if got, err := repo.LeerActivacion(t.Context()); err != nil || got.Estado != "inactiva" || got.Secuencia != 3 {
		t.Fatalf("desactivación nominal perdida: %+v, %v", got, err)
	}
	pool.fila = filaActivacionPrueba{estado: "sin_publicar"}
	if got, err := repo.LeerActivacion(t.Context()); err != nil || got.Estado != "sin_publicar" {
		t.Fatalf("ausencia nominal perdida: %+v, %v", got, err)
	}
	pool.fila.err = errors.New("fallo privado")
	if _, err := repo.LeerActivacion(t.Context()); !errors.Is(err, app.ErrNoDisponible) || err.Error() == "fallo privado" {
		t.Fatalf("fallo de fachada filtrado o afirmado como ausencia: %v", err)
	}
}
