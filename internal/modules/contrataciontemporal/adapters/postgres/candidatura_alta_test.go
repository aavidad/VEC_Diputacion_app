package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type iniciadorAltaCandidataPrueba struct {
	transacciones []pgx.Tx
	errores       []error
	inicios       int
	opciones      []pgx.TxOptions
}

func (i *iniciadorAltaCandidataPrueba) BeginTx(
	_ context.Context,
	opciones pgx.TxOptions,
) (pgx.Tx, error) {
	i.opciones = append(i.opciones, opciones)
	indice := i.inicios
	i.inicios++
	if indice < len(i.errores) && i.errores[indice] != nil {
		return nil, i.errores[indice]
	}
	if indice >= len(i.transacciones) {
		return nil, errors.New("transaccion de prueba ausente")
	}
	return i.transacciones[indice], nil
}

type transaccionAltaCandidataPrueba struct {
	pgx.Tx
	fila            pgx.Row
	consulta        string
	argumentos      []any
	errConfigurar   error
	errCommit       error
	configuraciones int
	commits         int
	rollbacks       int
	alCommit        func()
}

func (t *transaccionAltaCandidataPrueba) Exec(
	context.Context,
	string,
	...any,
) (pgconn.CommandTag, error) {
	t.configuraciones++
	return pgconn.NewCommandTag("SELECT 1"), t.errConfigurar
}

func (t *transaccionAltaCandidataPrueba) QueryRow(
	_ context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	t.consulta = consulta
	t.argumentos = append([]any(nil), argumentos...)
	return t.fila
}

func (t *transaccionAltaCandidataPrueba) Commit(context.Context) error {
	t.commits++
	if t.alCommit != nil {
		t.alCommit()
	}
	return t.errCommit
}

func (t *transaccionAltaCandidataPrueba) Rollback(context.Context) error {
	t.rollbacks++
	return nil
}

type filaAltaCandidataPrueba struct {
	valores []any
	err     error
}

func (f filaAltaCandidataPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != len(f.valores) {
		return errors.New("numero de columnas inesperado")
	}
	for indice, destino := range destinos {
		switch puntero := destino.(type) {
		case *string:
			valor, ok := f.valores[indice].(string)
			if !ok {
				return errors.New("texto de prueba invalido")
			}
			*puntero = valor
		case *int64:
			valor, ok := f.valores[indice].(int64)
			if !ok {
				return errors.New("entero de prueba invalido")
			}
			*puntero = valor
		case *time.Time:
			valor, ok := f.valores[indice].(time.Time)
			if !ok {
				return errors.New("instante de prueba invalido")
			}
			*puntero = valor
		default:
			return errors.New("destino de prueba no soportado")
		}
	}
	return nil
}

func selloCandidaturaPrueba(dominio string, generacion int, digito string) string {
	return "hmac-sha256:" + dominio + "/v" +
		string(rune('0'+generacion)) + ":" + strings.Repeat(digito, 64)
}

func solicitudCandidaturaPostgreSQLPrueba(t *testing.T) (
	ports.SolicitudResolverCandidaturaAlta,
	ports.DatosCandidaturaAlta,
) {
	t.Helper()
	instante := time.Date(2026, 8, 31, 10, 11, 12, 123456000, time.UTC)
	ambitos, err := ports.NuevaColeccionSellosHMAC(
		selloCandidaturaPrueba("vec.contratacion-temporal.ambito-idempotencia", 2, "a"),
		[]string{selloCandidaturaPrueba("vec.contratacion-temporal.ambito-idempotencia", 1, "b")},
	)
	if err != nil {
		t.Fatal(err)
	}
	huellas, err := ports.NuevaColeccionSellosHMAC(
		selloCandidaturaPrueba("vec.contratacion-temporal.huella-peticion", 2, "c"),
		[]string{selloCandidaturaPrueba("vec.contratacion-temporal.huella-peticion", 1, "d")},
	)
	if err != nil {
		t.Fatal(err)
	}
	propuesta := ports.DatosCandidaturaAlta{
		ReservaRef: "reserva:alta:postgre-r3b",
		Referencias: ports.ReferenciasAlta{ExpedienteRef: "expediente:ct:r3b-postgre",
			NumeroVisible: "2026/R3B-PG", ReciboRef: "recibo:alta:r3b-postgre"},
		AmbitoIdempotenciaHMAC: ambitosActivo(t, ambitos),
		HuellaPeticionHMAC:     huellasActivo(t, huellas),
		OrganizacionRef:        "organizacion:dipgra", ActorRef: "actor:rrhh:r3b",
		PerfilRef: "perfil:rrhh:r3b", InstanteEfecto: instante,
	}
	candidatura, err := ports.NuevaCandidaturaAlta(propuesta)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := ports.NuevaSolicitudResolverCandidaturaAlta(
		ports.DatosSolicitudResolverCandidaturaAlta{
			AmbitosIdempotenciaHMAC: ambitos, HuellasPeticionHMAC: huellas,
			OrganizacionRef: propuesta.OrganizacionRef, ActorRef: propuesta.ActorRef,
			PerfilRef: propuesta.PerfilRef, Propuesta: candidatura,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud, propuesta
}

func ambitosActivo(t *testing.T, coleccion ports.ColeccionSellosHMAC) string {
	t.Helper()
	datos, err := coleccion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return datos.Activo.Valor
}

func huellasActivo(t *testing.T, coleccion ports.ColeccionSellosHMAC) string {
	return ambitosActivo(t, coleccion)
}

func filaCandidaturaPostgreSQLPrueba(
	resultado string,
	datos ports.DatosCandidaturaAlta,
) pgx.Row {
	return filaAltaCandidataPrueba{valores: []any{
		resultado, datos.ReservaRef, datos.Referencias.ExpedienteRef,
		datos.Referencias.NumeroVisible, datos.Referencias.ReciboRef,
		datos.AmbitoIdempotenciaHMAC, datos.HuellaPeticionHMAC,
		datos.OrganizacionRef, datos.ActorRef, datos.PerfilRef,
		datos.InstanteEfecto,
	}}
}

func TestResolutorCandidaturaPostgreSQLCierraContratoYCommit(t *testing.T) {
	solicitud, propuesta := solicitudCandidaturaPostgreSQLPrueba(t)
	tx := &transaccionAltaCandidataPrueba{
		fila: filaCandidaturaPostgreSQLPrueba("estabilizada", propuesta),
	}
	iniciador := &iniciadorAltaCandidataPrueba{transacciones: []pgx.Tx{tx}}
	resolutor, err := nuevoResolutorCandidaturaAltaPostgreSQL(iniciador)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := resolutor.ResolverCandidaturaAlta(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	datos, _ := resultado.Datos()
	if datos != propuesta || tx.commits != 1 || tx.configuraciones != 1 ||
		!strings.Contains(tx.consulta, funcionResolverCandidaturaAlta) ||
		len(tx.argumentos) != 10 || iniciador.opciones[0].IsoLevel != pgx.Serializable {
		t.Fatalf("frontera SQL divergente: %+v, tx=%+v", datos, tx)
	}
}

func TestResolutorCandidaturaPostgreSQLReintentaSoloTransitorios(t *testing.T) {
	solicitud, propuesta := solicitudCandidaturaPostgreSQLPrueba(t)
	fallo := &transaccionAltaCandidataPrueba{fila: filaAltaCandidataPrueba{
		err: &pgconn.PgError{Code: "40001"},
	}}
	exito := &transaccionAltaCandidataPrueba{
		fila: filaCandidaturaPostgreSQLPrueba("recuperada", propuesta),
	}
	iniciador := &iniciadorAltaCandidataPrueba{transacciones: []pgx.Tx{fallo, exito}}
	resolutor, _ := nuevoResolutorCandidaturaAltaPostgreSQL(iniciador)
	if _, err := resolutor.ResolverCandidaturaAlta(context.Background(), solicitud); err != nil {
		t.Fatal(err)
	}
	if iniciador.inicios != 2 || exito.commits != 1 {
		t.Fatalf("reintento inesperado: %d", iniciador.inicios)
	}
}

func TestResolutorCandidaturaPostgreSQLConflictoEsOpaco(t *testing.T) {
	solicitud, _ := solicitudCandidaturaPostgreSQLPrueba(t)
	tx := &transaccionAltaCandidataPrueba{fila: filaAltaCandidataPrueba{
		err: &pgconn.PgError{Code: "23505", Message: "referencia colisionada"},
	}}
	resolutor, _ := nuevoResolutorCandidaturaAltaPostgreSQL(
		&iniciadorAltaCandidataPrueba{transacciones: []pgx.Tx{tx}},
	)
	_, err := resolutor.ResolverCandidaturaAlta(context.Background(), solicitud)
	if !errors.Is(err, ports.ErrClaveIdempotenciaUsada) ||
		strings.Contains(err.Error(), "referencia") {
		t.Fatalf("conflicto no normalizado: %v", err)
	}
}

func TestResolutorCandidaturaPostgreSQLColisionUnicaNoEsIdempotencia(t *testing.T) {
	solicitud, _ := solicitudCandidaturaPostgreSQLPrueba(t)
	tx := &transaccionAltaCandidataPrueba{fila: filaAltaCandidataPrueba{
		err: &pgconn.PgError{Code: "23505", ConstraintName: "candidatura_expediente_ref_key"},
	}}
	resolutor, _ := nuevoResolutorCandidaturaAltaPostgreSQL(
		&iniciadorAltaCandidataPrueba{transacciones: []pgx.Tx{tx}},
	)
	_, err := resolutor.ResolverCandidaturaAlta(context.Background(), solicitud)
	if !errors.Is(err, ports.ErrPersistenciaNoDisponible) ||
		errors.Is(err, ports.ErrClaveIdempotenciaUsada) {
		t.Fatalf("colision unica confundida con idempotencia: %v", err)
	}
}

func TestResolutorCandidaturaPostgreSQLCommitConfirmadoVenceCancelacion(t *testing.T) {
	solicitud, propuesta := solicitudCandidaturaPostgreSQLPrueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	tx := &transaccionAltaCandidataPrueba{
		fila:     filaCandidaturaPostgreSQLPrueba("estabilizada", propuesta),
		alCommit: cancelar,
	}
	resolutor, _ := nuevoResolutorCandidaturaAltaPostgreSQL(
		&iniciadorAltaCandidataPrueba{transacciones: []pgx.Tx{tx}},
	)
	if _, err := resolutor.ResolverCandidaturaAlta(ctx, solicitud); err != nil {
		t.Fatalf("commit confirmado convertido en cancelacion: %v", err)
	}
}
