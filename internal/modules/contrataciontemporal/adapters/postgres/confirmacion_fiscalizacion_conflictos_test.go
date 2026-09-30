package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Los dobles simulan los errores del transporte PostgreSQL. Estas pruebas
// comprueban el bucle de confirmación, no la ejecución de la función SQL.
func TestConfirmacionesFiscalizacionDistinguenConflictoDeclaradoDeSerializacion(t *testing.T) {
	confirmaciones := []struct {
		nombre    string
		confirmar func(*testing.T, context.Context, *iniciadorPreparacionPrueba) error
	}{
		{
			nombre: "fiscalizacion_inicial",
			confirmar: func(t *testing.T, ctx context.Context, pool *iniciadorPreparacionPrueba) error {
				orden := ports.OrdenConfirmarFiscalizacion{}
				orden.Preparacion.Material.VersionExpediente = 5
				transaccion := TransaccionFiscalizacionesPostgreSQL{pool: pool}
				recibo, err := transaccion.confirmarFiscalizacionConReintentos(ctx, orden, entradasConfirmarFiscalizacion{})
				if recibo != (ports.ReciboFiscalizacion{}) {
					t.Fatal("el fallo devolvió un recibo de fiscalización")
				}
				return err
			},
		},
		{
			nombre: "refiscalizacion",
			confirmar: func(t *testing.T, ctx context.Context, pool *iniciadorPreparacionPrueba) error {
				orden := ports.OrdenConfirmarFiscalizacion{}
				orden.Preparacion.Material.VersionExpediente = 7
				transaccion := TransaccionFiscalizacionesPostgreSQL{pool: pool}
				_, err := transaccion.confirmarFiscalizacionConReintentos(ctx, orden, entradasConfirmarFiscalizacion{})
				return err
			},
		},
		{
			nombre: "subsanacion",
			confirmar: func(t *testing.T, ctx context.Context, pool *iniciadorPreparacionPrueba) error {
				transaccion := TransaccionSubsanacionReparosPostgreSQL{pool: pool}
				recibo, err := transaccion.confirmarSubsanacionConReintentos(ctx, ports.OrdenConfirmarSubsanacionReparo{}, entradasConfirmarFiscalizacion{})
				if recibo != (ports.ReciboSubsanacionReparo{}) {
					t.Fatal("el fallo devolvió un recibo de subsanación")
				}
				return err
			},
		},
	}
	for _, confirmacion := range confirmaciones {
		t.Run(confirmacion.nombre, func(t *testing.T) {
			for _, caso := range []struct {
				nombre        string
				causa         error
				inicios       int
				errorEsperado error
			}{
				{"conflicto_de_funcion", &pgconn.PgError{Code: "40001", Routine: rutinaRaisePLpgSQL}, 1, domain.ErrVersionEnConflicto},
				{"conflicto_envuelto", fmt.Errorf("transporte: %w", &pgconn.PgError{Code: "40001", Routine: rutinaRaisePLpgSQL}), 1, domain.ErrVersionEnConflicto},
				{"serializacion_del_servidor", &pgconn.PgError{Code: "40001", Routine: "CheckForSerializationFailure"}, 2, ports.ErrPersistenciaFiscalizacionNoDisponible},
				{"serializacion_sin_rutina", &pgconn.PgError{Code: "40001"}, 2, ports.ErrPersistenciaFiscalizacionNoDisponible},
				{"interbloqueo", &pgconn.PgError{Code: "40P01", Routine: rutinaRaisePLpgSQL}, 2, ports.ErrPersistenciaFiscalizacionNoDisponible},
				{"denegacion", &pgconn.PgError{Code: "42501", Routine: rutinaRaisePLpgSQL}, 1, ports.ErrPersistenciaFiscalizacionNoDisponible},
			} {
				t.Run(caso.nombre, func(t *testing.T) {
					primera := &transaccionPreparacionPrueba{errConfigurar: caso.causa}
					segunda := &transaccionPreparacionPrueba{errConfigurar: &pgconn.PgError{Code: "42501"}}
					pool := &iniciadorPreparacionPrueba{transacciones: []pgx.Tx{primera, segunda}}
					ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancelar()
					err := confirmacion.confirmar(t, ctx, pool)
					if !errors.Is(err, caso.errorEsperado) || pool.inicios != caso.inicios {
						t.Fatalf("error=%v, inicios=%d; esperados %v y %d", err, pool.inicios, caso.errorEsperado, caso.inicios)
					}
					if primera.confirmaciones != 0 || segunda.confirmaciones != 0 || primera.reversiones != 1 || segunda.reversiones != caso.inicios-1 {
						t.Fatalf("confirmaciones=%d/%d, reversiones=%d/%d", primera.confirmaciones, segunda.confirmaciones, primera.reversiones, segunda.reversiones)
					}
					if pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite {
						t.Fatalf("opciones de transacción inesperadas: %+v", pool.opciones)
					}
				})
			}
		})
	}
}
