package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type iniciadorRegistroUsuariosExternoPrueba struct {
	transacciones []pgx.Tx
	invocaciones  int
}

func (i *iniciadorRegistroUsuariosExternoPrueba) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	indice := i.invocaciones
	i.invocaciones++
	if indice >= len(i.transacciones) {
		return nil, errors.New("transaccion de prueba ausente")
	}
	return i.transacciones[indice], nil
}

func TestRegistroUsuariosExternoReintentaErroresDeScanYCursor(t *testing.T) {
	escenario := nuevoEscenarioRegistroContextoActorV3PostgreSQLPrueba(t, true)
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(escenario.solicitud, escenario.decision, escenario.motivo, escenario.resultado)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := domain.HuellaSHA256DecisionAutorizacionV3(escenario.decision)
	if err != nil {
		t.Fatal(err)
	}
	registrada := escenario.ahora.Add(time.Microsecond)
	for _, origen := range []string{"scan", "cursor"} {
		for _, codigo := range []string{"40001", "40P01"} {
			t.Run(origen+codigo, func(t *testing.T) {
				primera := nuevaTransaccionRegistroContextoActorV3PostgreSQLPrueba(true, "concedida", huella, registrada)
				segunda := nuevaTransaccionRegistroContextoActorV3PostgreSQLPrueba(true, "concedida", huella, registrada)
				fallo := &pgconn.PgError{Code: codigo, Message: "detalle_privado"}
				if origen == "scan" {
					primera.filas.resultados[0].errorEscaneo = fallo
				} else {
					primera.filas.err = fallo
				}
				iniciador := &iniciadorRegistroUsuariosExternoPrueba{transacciones: []pgx.Tx{primera, segunda}}
				almacen, err := nuevoAlmacenAutorizacionUsuariosExterno(iniciador)
				if err != nil {
					t.Fatal(err)
				}
				fecha, err := almacen.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(t.Context(), orden)
				if err != nil || !fecha.Equal(registrada) || iniciador.invocaciones != 2 || primera.commitInvocado || !primera.rollbackInvocado || !primera.filas.cerradas || !segunda.commitInvocado {
					t.Fatalf("recuperacion incorrecta: error=%v intentos=%d", err, iniciador.invocaciones)
				}
			})
		}
	}
}

func TestRegistroUsuariosExternoSaneaErroresDeScanYCursor(t *testing.T) {
	escenario := nuevoEscenarioRegistroContextoActorV3PostgreSQLPrueba(t, true)
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(escenario.solicitud, escenario.decision, escenario.motivo, escenario.resultado)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := domain.HuellaSHA256DecisionAutorizacionV3(escenario.decision)
	if err != nil {
		t.Fatal(err)
	}
	for _, origen := range []string{"scan", "cursor"} {
		t.Run(origen, func(t *testing.T) {
			tx := nuevaTransaccionRegistroContextoActorV3PostgreSQLPrueba(true, "concedida", huella, escenario.ahora.Add(time.Microsecond))
			fallo := errors.New("detalle_privado")
			if origen == "scan" {
				tx.filas.resultados[0].errorEscaneo = fallo
			} else {
				tx.filas.err = fallo
			}
			iniciador := &iniciadorRegistroUsuariosExternoPrueba{transacciones: []pgx.Tx{tx}}
			almacen, err := nuevoAlmacenAutorizacionUsuariosExterno(iniciador)
			if err != nil {
				t.Fatal(err)
			}
			_, err = almacen.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(t.Context(), orden)
			if !errors.Is(err, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) || strings.Contains(err.Error(), "detalle_privado") || iniciador.invocaciones != 1 || tx.commitInvocado || !tx.rollbackInvocado || !tx.filas.cerradas {
				t.Fatalf("error no cerrado: %v", err)
			}
		})
	}
}

func TestRegistroUsuariosExternoSerializacionInvalidaNoConsulta(t *testing.T) {
	iniciador := &iniciadorRegistroUsuariosExternoPrueba{}
	almacen, err := nuevoAlmacenAutorizacionUsuariosExterno(iniciador)
	if err != nil {
		t.Fatal(err)
	}
	_, err = almacen.registrar(t.Context(), ports.DatosOrdenRegistroAutorizacionLigadaV3{}, true, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible)
	if !errors.Is(err, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) || iniciador.invocaciones != 0 {
		t.Fatalf("serializacion invalida alcanzo SQL: %v", err)
	}
}
