package postgres

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type verificadorFlujoFirmaPostgreSQLPrueba struct{ error error }

func (v verificadorFlujoFirmaPostgreSQLPrueba) VerificarEstadoFlujoFirmaBaremacion(
	context.Context,
	puertosbolsa.SolicitudVerificarEstadoFlujoFirmaBaremacion,
) error {
	return v.error
}

type iniciadorFlujoFirmaPostgreSQLPrueba struct {
	tx       pgx.Tx
	opciones []pgx.TxOptions
}

func (i *iniciadorFlujoFirmaPostgreSQLPrueba) BeginTx(
	_ context.Context,
	opciones pgx.TxOptions,
) (pgx.Tx, error) {
	i.opciones = append(i.opciones, opciones)
	return i.tx, nil
}

type transaccionFlujoFirmaPostgreSQLPrueba struct {
	pgx.Tx
	filas           []pgx.Row
	consultas       []string
	argumentos      [][]any
	configuraciones int
	confirmaciones  int
	reversiones     int
}

func (t *transaccionFlujoFirmaPostgreSQLPrueba) Exec(
	context.Context,
	string,
	...any,
) (pgconn.CommandTag, error) {
	t.configuraciones++
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func (t *transaccionFlujoFirmaPostgreSQLPrueba) QueryRow(
	_ context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	t.consultas = append(t.consultas, consulta)
	copia := make([]any, len(argumentos))
	for indice, argumento := range argumentos {
		if contenido, valido := argumento.([]byte); valido {
			copia[indice] = append([]byte(nil), contenido...)
			continue
		}
		copia[indice] = argumento
	}
	t.argumentos = append(t.argumentos, copia)
	if len(t.filas) == 0 {
		return filaFlujoFirmaPostgreSQLPrueba{
			error: errors.New("consulta no prevista"),
		}
	}
	fila := t.filas[0]
	t.filas = t.filas[1:]
	return fila
}

func (t *transaccionFlujoFirmaPostgreSQLPrueba) Commit(context.Context) error {
	t.confirmaciones++
	return nil
}

func (t *transaccionFlujoFirmaPostgreSQLPrueba) Rollback(context.Context) error {
	t.reversiones++
	return nil
}

type filaFlujoFirmaPostgreSQLPrueba struct {
	valores []any
	error   error
}

func (f filaFlujoFirmaPostgreSQLPrueba) Scan(destinos ...any) error {
	if f.error != nil {
		return f.error
	}
	if len(destinos) != len(f.valores) {
		return errors.New("cantidad de columnas inesperada")
	}
	for indice, valor := range f.valores {
		switch destino := destinos[indice].(type) {
		case *string:
			texto, valido := valor.(string)
			if !valido {
				return errors.New("texto inesperado")
			}
			*destino = texto
		case *[]byte:
			contenido, valido := valor.([]byte)
			if !valido {
				return errors.New("binario inesperado")
			}
			*destino = append([]byte(nil), contenido...)
		case *pgtype.Timestamptz:
			if valor == nil {
				*destino = pgtype.Timestamptz{}
				continue
			}
			instante, valido := valor.(time.Time)
			if !valido {
				return errors.New("instante inesperado")
			}
			*destino = pgtype.Timestamptz{Time: instante, Valid: true}
		case *pgtype.Text:
			if valor == nil {
				*destino = pgtype.Text{}
				continue
			}
			texto, valido := valor.(string)
			if !valido {
				return errors.New("texto nulo inesperado")
			}
			*destino = pgtype.Text{String: texto, Valid: true}
		default:
			return errors.New("destino inesperado")
		}
	}
	return nil
}

func TestRepositorioFlujoFirmaPostgreSQLOcultaClaveYRechazaDependencias(
	t *testing.T,
) {
	if _, err := nuevoRepositorioFlujosFirmaBaremacionPostgreSQL(
		nil,
		verificadorFlujoFirmaPostgreSQLPrueba{},
		make([]byte, 32),
	); !errors.Is(err, ErrRepositorioFlujoFirmaPostgreSQLNoDisponible) {
		t.Fatalf("dependencia nula aceptada: %v", err)
	}
	repositorio := repositorioFlujoFirmaPostgreSQLPrueba(
		t,
		&transaccionFlujoFirmaPostgreSQLPrueba{},
	)
	formateado := fmt.Sprintf("%v|%#v|%+v", repositorio, repositorio, repositorio)
	if strings.Count(
		formateado,
		"[REPOSITORIO-FLUJOS-FIRMA-BAREMACION-POSTGRESQL-REDACTADO]",
	) != 3 {
		t.Fatalf("formato no redactado: %q", formateado)
	}
	tipo := reflect.TypeOf(repositorio).Elem()
	if _, existe := tipo.FieldByName("claveHMACToken"); existe {
		t.Fatal("la clave HMAC quedó en un campo reflectible")
	}
	campo, existe := tipo.FieldByName("operarHMACToken")
	valor := reflect.ValueOf(repositorio).Elem().FieldByName("operarHMACToken")
	if !existe || campo.Type.Kind() != reflect.Func ||
		valor.CanInterface() || valor.CanSet() {
		t.Fatal("la operación HMAC privada quedó accesible")
	}
}

func TestRepositorioFlujoFirmaPostgreSQLCreaYVerificaAntesDeConfirmar(
	t *testing.T,
) {
	expediente := expedienteFlujoFirmaPostgreSQLPrueba(t)
	documento, cifrado, err := serializarExpedienteFlujoFirmaPostgreSQL(expediente)
	if err != nil {
		t.Fatal(err)
	}
	tx := &transaccionFlujoFirmaPostgreSQLPrueba{
		filas: []pgx.Row{filaFlujoFirmaPostgreSQLPrueba{
			valores: []any{"creado", documento, cifrado},
		}},
	}
	repositorio, iniciador := repositorioEIniciadorFlujoFirmaPostgreSQLPrueba(t, tx)
	resultado, err := repositorio.CrearORecuperarFlujoFirmaBaremacion(
		context.Background(),
		puertosbolsa.SolicitudCrearORecuperarFlujoFirmaBaremacion{
			Expediente: expediente,
		},
	)
	if err != nil || !resultado.Creado ||
		!expedientesFlujoFirmaPostgreSQLExactos(resultado.Expediente, expediente) {
		t.Fatalf("creación fiable rechazada: resultado=%+v error=%v", resultado, err)
	}
	comprobarTransaccionFlujoFirmaPostgreSQL(
		t,
		iniciador,
		tx,
		pgx.ReadWrite,
		funcionCrearFlujoFirmaPostgreSQLV1,
		1,
	)

	documentoAlterado := append([]byte(nil), documento...)
	documentoAlterado[len(documentoAlterado)-2] ^= 1
	txAlterada := &transaccionFlujoFirmaPostgreSQLPrueba{
		filas: []pgx.Row{filaFlujoFirmaPostgreSQLPrueba{
			valores: []any{"creado", documentoAlterado, cifrado},
		}},
	}
	repositorioAlterado := repositorioFlujoFirmaPostgreSQLPrueba(t, txAlterada)
	if _, err := repositorioAlterado.CrearORecuperarFlujoFirmaBaremacion(
		context.Background(),
		puertosbolsa.SolicitudCrearORecuperarFlujoFirmaBaremacion{
			Expediente: expediente,
		},
	); err == nil || txAlterada.confirmaciones != 0 {
		t.Fatalf("respuesta alterada confirmada: %v", err)
	}
}

func TestRepositorioFlujoFirmaPostgreSQLArrendamientoOpacoYCercado(
	t *testing.T,
) {
	expediente := expedienteFlujoFirmaPostgreSQLPrueba(t)
	documento, cifrado, err := serializarExpedienteFlujoFirmaPostgreSQL(expediente)
	if err != nil {
		t.Fatal(err)
	}
	expira := expediente.ActualizadoEn.Add(time.Minute)
	tx := &transaccionFlujoFirmaPostgreSQLPrueba{
		filas: []pgx.Row{filaFlujoFirmaPostgreSQLPrueba{
			valores: []any{"adquirido", documento, cifrado, "7", expira},
		}},
	}
	repositorio, iniciador := repositorioEIniciadorFlujoFirmaPostgreSQLPrueba(t, tx)
	resultado, err := repositorio.AdquirirArrendamientoFlujoFirmaBaremacion(
		context.Background(),
		puertosbolsa.SolicitudAdquirirArrendamientoFlujoFirmaBaremacion{
			Consulta: puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion{
				FlujoRef:               expediente.FlujoRef,
				IndiceIdempotenciaHMAC: expediente.IndiceIdempotenciaHMAC,
				VinculoActorHMAC:       expediente.VinculoActorHMAC,
			},
			VersionEsperada: 1,
			PropietarioRef:  "worker-firma-001",
			Duracion:        time.Minute,
		},
	)
	if err != nil || resultado.Arrendamiento.Validar() != nil ||
		resultado.Arrendamiento.SecuenciaCercado != 7 ||
		!resultado.Arrendamiento.ExpiraEn.Equal(expira) {
		t.Fatalf("arrendamiento fiable rechazado: resultado=%+v error=%v", resultado, err)
	}
	if len(tx.argumentos) != 1 || len(tx.argumentos[0]) != 2 {
		t.Fatal("contrato SQL de arrendamiento inesperado")
	}
	huella, valida := tx.argumentos[0][1].([]byte)
	if !valida || len(huella) != 32 {
		t.Fatalf("PostgreSQL no recibió una huella HMAC: %T %d", huella, len(huella))
	}
	comprobarTransaccionFlujoFirmaPostgreSQL(
		t,
		iniciador,
		tx,
		pgx.ReadWrite,
		funcionAdquirirFlujoFirmaPostgreSQLV1,
		1,
	)
}

func TestRepositorioFlujoFirmaPostgreSQLGuardaSoloTransicionValida(
	t *testing.T,
) {
	anterior := expedienteFlujoFirmaPostgreSQLPrueba(t)
	siguiente := siguienteFlujoFirmaPostgreSQLPrueba(t, anterior)
	documentoAnterior, cifradoAnterior, err :=
		serializarExpedienteFlujoFirmaPostgreSQL(anterior)
	if err != nil {
		t.Fatal(err)
	}
	documentoSiguiente, cifradoSiguiente, err :=
		serializarExpedienteFlujoFirmaPostgreSQL(siguiente)
	if err != nil {
		t.Fatal(err)
	}
	token, err := puertosbolsa.NuevoTokenArrendamientoFlujoFirmaBaremacion()
	if err != nil {
		t.Fatal(err)
	}
	arrendamiento := puertosbolsa.ArrendamientoFlujoFirmaBaremacion{
		FlujoRef: anterior.FlujoRef, PropietarioRef: "worker-firma-001",
		SecuenciaCercado: 3, ExpiraEn: time.Now().UTC().Add(time.Minute), Token: token,
	}
	tx := &transaccionFlujoFirmaPostgreSQLPrueba{
		filas: []pgx.Row{
			filaFlujoFirmaPostgreSQLPrueba{
				valores: []any{documentoAnterior, cifradoAnterior},
			},
			filaFlujoFirmaPostgreSQLPrueba{
				valores: []any{"guardado", documentoSiguiente, cifradoSiguiente},
			},
		},
	}
	repositorio, iniciador := repositorioEIniciadorFlujoFirmaPostgreSQLPrueba(t, tx)
	guardado, err := repositorio.GuardarFlujoFirmaBaremacion(
		context.Background(),
		puertosbolsa.SolicitudGuardarFlujoFirmaBaremacion{
			VersionEsperada: 1, Arrendamiento: arrendamiento, Siguiente: siguiente,
		},
	)
	if err != nil || guardado.Version != 2 {
		t.Fatalf("transición fiable rechazada: resultado=%+v error=%v", guardado, err)
	}
	comprobarTransaccionFlujoFirmaPostgreSQL(
		t,
		iniciador,
		tx,
		pgx.ReadWrite,
		funcionGuardarFlujoFirmaPostgreSQLV1,
		2,
	)
}

func repositorioFlujoFirmaPostgreSQLPrueba(
	t *testing.T,
	tx pgx.Tx,
) *RepositorioFlujosFirmaBaremacionPostgreSQL {
	t.Helper()
	repositorio, _ := repositorioEIniciadorFlujoFirmaPostgreSQLPrueba(t, tx)
	return repositorio
}

func repositorioEIniciadorFlujoFirmaPostgreSQLPrueba(
	t *testing.T,
	tx pgx.Tx,
) (
	*RepositorioFlujosFirmaBaremacionPostgreSQL,
	*iniciadorFlujoFirmaPostgreSQLPrueba,
) {
	t.Helper()
	iniciador := &iniciadorFlujoFirmaPostgreSQLPrueba{tx: tx}
	repositorio, err := nuevoRepositorioFlujosFirmaBaremacionPostgreSQL(
		iniciador,
		verificadorFlujoFirmaPostgreSQLPrueba{},
		[]byte("0123456789abcdef0123456789abcdef"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return repositorio, iniciador
}

func siguienteFlujoFirmaPostgreSQLPrueba(
	t *testing.T,
	anterior puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) puertosbolsa.ExpedienteFlujoFirmaBaremacion {
	t.Helper()
	siguiente, err := anterior.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	siguiente.Version++
	siguiente.ActualizadoEn = siguiente.ActualizadoEn.Add(time.Second)
	siguiente.PuntosControl = append(
		siguiente.PuntosControl,
		puertosbolsa.PuntoControlFirmaBaremacion{
			Paso:                  puertosbolsa.PasoPrepararFirmaBaremacion,
			Estado:                puertosbolsa.EstadoPuntoControlFirmaDeclarado,
			EfectoRef:             "efecto-preparar-firma-001",
			ClaveIdempotenciaHMAC: hmacFlujoFirmaPostgreSQLPrueba("5"),
			DeclaradoEn:           siguiente.ActualizadoEn,
		},
	)
	siguiente.SelloEstadoHMAC = hmacFlujoFirmaPostgreSQLPrueba("6")
	if siguiente.Validar() != nil ||
		puertosbolsa.ValidarTransicionFlujoFirmaBaremacion(anterior, siguiente) != nil {
		t.Fatal("fixture de transición inválido")
	}
	return siguiente
}

func comprobarTransaccionFlujoFirmaPostgreSQL(
	t *testing.T,
	iniciador *iniciadorFlujoFirmaPostgreSQLPrueba,
	tx *transaccionFlujoFirmaPostgreSQLPrueba,
	modo pgx.TxAccessMode,
	funcion string,
	consultas int,
) {
	t.Helper()
	if len(iniciador.opciones) != 1 ||
		iniciador.opciones[0].IsoLevel != pgx.Serializable ||
		iniciador.opciones[0].AccessMode != modo ||
		tx.configuraciones != 1 || tx.confirmaciones != 1 ||
		len(tx.consultas) != consultas ||
		!strings.Contains(tx.consultas[len(tx.consultas)-1], funcion) {
		t.Fatalf(
			"transacción incorrecta: opciones=%+v config=%d commit=%d consultas=%q",
			iniciador.opciones,
			tx.configuraciones,
			tx.confirmaciones,
			tx.consultas,
		)
	}
}
