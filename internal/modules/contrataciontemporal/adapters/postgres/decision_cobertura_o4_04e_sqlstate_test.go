package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

type transaccionSQLStateDecisionCoberturaO404EPrueba struct {
	*transaccionPreparacionPrueba
	consultas int
}

func (t *transaccionSQLStateDecisionCoberturaO404EPrueba) QueryRow(
	ctx context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	t.consultas++
	return t.transaccionPreparacionPrueba.QueryRow(
		ctx,
		consulta,
		argumentos...,
	)
}

func TestConfirmacionDecisionCoberturaO404ENoReintentaSQLStateTransaccional(
	t *testing.T,
) {
	t.Parallel()
	// Solo 40001/40P01 en la propia sentencia acreditan carrera; la
	// cancelación (57014) o la caída de la conexión (08006) siguen siendo
	// fallos sin esa señal.
	for _, caso := range []struct {
		codigo  string
		carrera bool
	}{{"40001", true}, {"40P01", true}, {"57014", false}, {"08006", false}} {
		codigo, carrera := caso.codigo, caso.carrera
		t.Run(codigo, func(t *testing.T) {
			t.Parallel()
			errorSQL := &pgconn.PgError{
				Code: codigo, Message: "detalle interno reservado",
			}
			base := &transaccionPreparacionPrueba{
				fila: filaBytesDecisionCoberturaO404EPrueba{err: errorSQL},
			}
			tx := &transaccionSQLStateDecisionCoberturaO404EPrueba{
				transaccionPreparacionPrueba: base,
			}
			iniciador := &iniciadorPreparacionPrueba{
				transacciones: []pgx.Tx{
					tx,
					&transaccionPreparacionPrueba{},
				},
			}
			ejecutor, err :=
				nuevoEjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL(
					iniciador,
				)
			if err != nil {
				t.Fatalf("crear ejecutor: %v", err)
			}
			err = ejecutor.EjecutarSesionTCB(
				context.Background(),
				func(
					puerto cobertura.SesionTCBOperacionDecisionCobertura,
				) error {
					sesion, ok := puerto.(*sesionDecisionCoberturaO404E)
					if !ok {
						return errors.New("tipo de sesión inesperado")
					}
					prepararSesionDenegadaSQLStateDecisionCoberturaO404EPrueba(
						t,
						sesion,
					)
					_, errConfirmacion := sesion.Confirmar(
						context.Background(),
					)
					return errConfirmacion
				},
			)
			var recibido *pgconn.PgError
			if !errors.As(err, &recibido) || recibido.Code != codigo {
				t.Fatalf("SQLSTATE %s no se conservó: %v", codigo, err)
			}
			// El ejecutor no repite, pero acredita al núcleo que la base
			// revirtió la transacción para que repita la orden entera.
			if errors.Is(err, cobertura.ErrCarreraSerializableSesionTCBOperacionDecisionCobertura) != carrera {
				t.Fatalf("SQLSTATE %s: acreditación de carrera %v inesperada: %v", codigo, !carrera, err)
			}
			if iniciador.inicios != 1 || tx.consultas != 1 ||
				tx.confirmaciones != 0 || tx.reversiones != 1 {
				t.Fatalf(
					"SQLSTATE %s provocó reintento: begin=%d consulta=%d "+
						"commit=%d rollback=%d",
					codigo,
					iniciador.inicios,
					tx.consultas,
					tx.confirmaciones,
					tx.reversiones,
				)
			}
		})
	}
}

func prepararSesionDenegadaSQLStateDecisionCoberturaO404EPrueba(
	t *testing.T,
	sesion *sesionDecisionCoberturaO404E,
) {
	t.Helper()
	recurso := recursoDenegacionDecisionCoberturaO404EPrueba()
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("calcular huella del recurso: %v", err)
	}
	sesion.estado = estadoSesionDecisionCoberturaLista
	sesion.rama = cobertura.RamaSesionTCBOperacionDecisionCoberturaDenegada
	sesion.carga = cargaConfirmarDecisionCoberturaO404E{
		Esquema: esquemaCargaDecisionCoberturaO404E,
		Rama:    cobertura.RamaSesionTCBOperacionDecisionCoberturaDenegada,
		DecisionVEC: decisionVECDecisionCoberturaO404E{
			DecisionCanonica:            []byte{1},
			MotivoCanonico:              []byte{2},
			RecursoRef:                  recurso.Referencia,
			RecursoModulo:               recurso.ModuloID,
			RecursoTipo:                 recurso.Tipo,
			Ambitos:                     clonarMapaDecisionCoberturaO404E(recurso.Ambitos),
			Atributos:                   clonarMapaDecisionCoberturaO404E(recurso.Atributos),
			ContextoRecursoHuellaSHA256: huellaRecurso,
		},
		Denegacion: &denegacionDecisionCoberturaO404E{
			RecursoRef:          recurso.Referencia,
			RecursoModulo:       recurso.ModuloID,
			RecursoTipo:         recurso.Tipo,
			Ambitos:             clonarMapaDecisionCoberturaO404E(recurso.Ambitos),
			Atributos:           clonarMapaDecisionCoberturaO404E(recurso.Atributos),
			RecursoHuellaSHA256: huellaRecurso,
		},
		ConsumosC1: []consumoC1DecisionCoberturaO404E{},
	}
}

// Un 40001/40P01 al COMMIT también es una reversión segura y se acredita; un
// fallo de COMMIT de otro tipo sigue siendo ambiguo y se devuelve tal cual.
func TestEjecutorDecisionCoberturaO404EAcreditaCarreraEnCommit(t *testing.T) {
	t.Parallel()
	for _, caso := range []struct {
		err     error
		carrera bool
	}{
		{&pgconn.PgError{Code: "40001", Message: "serializacion"}, true},
		{&pgconn.PgError{Code: "40P01", Message: "interbloqueo"}, true},
		{&pgconn.PgError{Code: "08006", Message: "conexion"}, false},
		{errors.New("respuesta COMMIT perdida"), false},
	} {
		tx := &transaccionPreparacionPrueba{errConfirmar: caso.err}
		iniciador := &iniciadorPreparacionPrueba{transacciones: []pgx.Tx{tx}}
		ejecutor, err := nuevoEjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL(iniciador)
		if err != nil {
			t.Fatal(err)
		}
		err = ejecutor.EjecutarSesionTCB(context.Background(),
			func(puerto cobertura.SesionTCBOperacionDecisionCobertura) error {
				sesion := puerto.(*sesionDecisionCoberturaO404E)
				sesion.mu.Lock()
				sesion.estado = estadoSesionDecisionCoberturaConsumida
				sesion.confirmada = true
				sesion.mu.Unlock()
				return nil
			})
		if !errors.Is(err, caso.err) || errors.Is(err, cobertura.ErrCarreraSerializableSesionTCBOperacionDecisionCobertura) != caso.carrera ||
			iniciador.inicios != 1 || tx.confirmaciones != 1 {
			t.Fatalf("COMMIT %v: err=%v begin=%d commit=%d", caso.err, err, iniciador.inicios, tx.confirmaciones)
		}
	}
}

// Un error del callback sin carrera en Confirmar no se acredita como carrera.
func TestEjecutorDecisionCoberturaO404ENoAcreditaCarreraSinSQLState(t *testing.T) {
	t.Parallel()
	errCallback := errors.New("núcleo rechaza recibo")
	tx := &transaccionPreparacionPrueba{}
	ejecutor, err := nuevoEjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL(&iniciadorPreparacionPrueba{tx: tx})
	if err != nil {
		t.Fatal(err)
	}
	err = ejecutor.EjecutarSesionTCB(context.Background(),
		func(cobertura.SesionTCBOperacionDecisionCobertura) error { return errCallback })
	if !errors.Is(err, errCallback) || errors.Is(err, cobertura.ErrCarreraSerializableSesionTCBOperacionDecisionCobertura) {
		t.Fatalf("error inesperado: %v", err)
	}
}
