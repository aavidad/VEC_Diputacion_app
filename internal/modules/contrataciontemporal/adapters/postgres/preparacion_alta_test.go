package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	claveAmbitoAltaPrueba   = "vec.contratacion-temporal.ambito-idempotencia/v1"
	clavePeticionAltaPrueba = "vec.contratacion-temporal.huella-peticion/v1"
)

type iniciadorPreparacionPrueba struct {
	tx       pgx.Tx
	err      error
	opciones pgx.TxOptions
	inicios  int
}

func (i *iniciadorPreparacionPrueba) BeginTx(
	_ context.Context,
	opciones pgx.TxOptions,
) (pgx.Tx, error) {
	i.opciones = opciones
	i.inicios++
	return i.tx, i.err
}

type transaccionPreparacionPrueba struct {
	pgx.Tx
	fila           pgx.Row
	consulta       string
	operacion      []byte
	configurada    bool
	confirmaciones int
	reversiones    int
	errConfigurar  error
	errConfirmar   error
}

func (t *transaccionPreparacionPrueba) Exec(
	_ context.Context,
	_ string,
	_ ...any,
) (pgconn.CommandTag, error) {
	t.configurada = true
	return pgconn.NewCommandTag("SELECT 1"), t.errConfigurar
}

func (t *transaccionPreparacionPrueba) QueryRow(
	_ context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	t.consulta = consulta
	if len(argumentos) == 1 {
		if operacion, ok := argumentos[0].([]byte); ok {
			t.operacion = append([]byte(nil), operacion...)
		}
	}
	return t.fila
}

func (t *transaccionPreparacionPrueba) Commit(context.Context) error {
	t.confirmaciones++
	return t.errConfirmar
}

func (t *transaccionPreparacionPrueba) Rollback(context.Context) error {
	t.reversiones++
	return nil
}

type filaPreparacionPrueba struct {
	valores []any
	err     error
}

func (f filaPreparacionPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != len(f.valores) {
		return errors.New("columnas inesperadas")
	}
	for indice, destino := range destinos {
		valor := f.valores[indice]
		switch puntero := destino.(type) {
		case *string:
			texto, valido := valor.(string)
			if !valido {
				return errors.New("texto inválido")
			}
			*puntero = texto
		case *pgtype.Int8:
			numero, valido := valor.(pgtype.Int8)
			if !valido {
				return errors.New("entero inválido")
			}
			*puntero = numero
		case *pgtype.Text:
			texto, valido := valor.(pgtype.Text)
			if !valido {
				return errors.New("texto opcional inválido")
			}
			*puntero = texto
		case *pgtype.Timestamptz:
			instante, valido := valor.(pgtype.Timestamptz)
			if !valido {
				return errors.New("instante inválido")
			}
			*puntero = instante
		default:
			return errors.New("destino no soportado")
		}
	}
	return nil
}

type selladorAmbitoPrueba struct {
	huella    string
	err       error
	alLlamar  func()
	solicitud ports.SolicitudSellarAmbitoIdempotencia
}

func (s *selladorAmbitoPrueba) SellarAmbitoIdempotencia(
	_ context.Context,
	solicitud ports.SolicitudSellarAmbitoIdempotencia,
) (string, error) {
	if s.alLlamar != nil {
		s.alLlamar()
	}
	s.solicitud = solicitud
	return s.huella, s.err
}

type generadorReferenciasPrueba struct {
	referencias ports.ReferenciasAlta
	reservaRef  string
	errRefs     error
	errReserva  error
	alGenerar   func()
	alReservar  func()
}

func (g *generadorReferenciasPrueba) GenerarReferenciasAlta(
	context.Context,
) (ports.ReferenciasAlta, error) {
	if g.alGenerar != nil {
		g.alGenerar()
	}
	return g.referencias, g.errRefs
}

func (g *generadorReferenciasPrueba) NuevaReferenciaReservaAlta(
	context.Context,
) (string, error) {
	if g.alReservar != nil {
		g.alReservar()
	}
	return g.reservaRef, g.errReserva
}

func solicitudPreparacionPrueba() ports.SolicitudPrepararAlta {
	return ports.SolicitudPrepararAlta{
		ClaveIdempotencia:  "01J2F8X4K4R9T2Y7W3M6Q8P1AB",
		HuellaPeticionHMAC: selloHMACPrueba(clavePeticionAltaPrueba, "b"),
		OrganizacionRef:    "organizacion:diputacion-granada",
		ActorRef:           "actor:tecnica-rrhh-001",
		PerfilRef:          "perfil:tecnica-rrhh",
	}
}

func referenciasPreparacionPrueba() ports.ReferenciasAlta {
	return ports.ReferenciasAlta{
		ExpedienteRef: "expediente:ct-2026-0001",
		NumeroVisible: "2026/CT-0001",
		ReciboRef:     "recibo:alta-001",
	}
}

func filaReservadaPreparacionPrueba(resultado string) pgx.Row {
	referencias := referenciasPreparacionPrueba()
	return filaPreparacionPrueba{valores: []any{
		resultado,
		"reserva:alta-candidata-001",
		referencias.ExpedienteRef,
		referencias.NumeroVisible,
		referencias.ReciboRef,
		selloHMACPrueba(claveAmbitoAltaPrueba, "d"),
		selloHMACPrueba(clavePeticionAltaPrueba, "b"),
		"organizacion:diputacion-granada",
		"actor:tecnica-rrhh-001",
		"perfil:tecnica-rrhh",
		string(ports.PreparacionReservada),
		pgtype.Int8{},
		pgtype.Text{},
		pgtype.Text{},
		pgtype.Timestamptz{},
	}}
}

func nuevoPreparadorPrueba(
	t *testing.T,
	fila pgx.Row,
) (*PreparadorAltaPostgreSQL, *transaccionPreparacionPrueba) {
	t.Helper()
	tx := &transaccionPreparacionPrueba{fila: fila}
	preparador, err := nuevoPreparadorAltaPostgreSQL(
		&iniciadorPreparacionPrueba{tx: tx},
		&selladorAmbitoPrueba{
			huella: selloHMACPrueba(claveAmbitoAltaPrueba, "d"),
		},
		&generadorReferenciasPrueba{
			referencias: referenciasPreparacionPrueba(),
			reservaRef:  "reserva:alta-candidata-001",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return preparador, tx
}

func TestPreparadorPostgreSQLReservaSinPersistirClaveCruda(t *testing.T) {
	preparador, tx := nuevoPreparadorPrueba(t, filaReservadaPreparacionPrueba("reservada"))
	solicitud := solicitudPreparacionPrueba()

	preparacion, err := preparador.PrepararAlta(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if preparacion.Estado != ports.PreparacionReservada ||
		preparacion.ReservaRef != "reserva:alta-candidata-001" ||
		tx.confirmaciones != 1 || !tx.configurada ||
		!strings.Contains(tx.consulta, funcionPrepararAltaV1) {
		t.Fatalf("resultado inesperado: preparacion=%#v tx=%#v", preparacion, tx)
	}
	if strings.Contains(string(tx.operacion), solicitud.ClaveIdempotencia) {
		t.Fatal("la operación PostgreSQL contiene la clave idempotente en claro")
	}
	var operacion operacionPrepararAltaV1
	if err := json.Unmarshal(tx.operacion, &operacion); err != nil {
		t.Fatal(err)
	}
	if operacion.AmbitoHMAC != selloHMACPrueba(claveAmbitoAltaPrueba, "d") ||
		operacion.Esquema != esquemaPrepararAltaV1 {
		t.Fatalf("operación no ligada: %#v", operacion)
	}
	var campos map[string]json.RawMessage
	if err := json.Unmarshal(tx.operacion, &campos); err != nil {
		t.Fatal(err)
	}
	for _, nombre := range []string{
		"esquema", "ambito_hmac", "huella_peticion_hmac",
		"organizacion_ref", "actor_ref", "perfil_ref",
		"reserva_ref_candidata", "referencias_candidatas",
	} {
		if _, existe := campos[nombre]; !existe {
			t.Fatalf("falta el campo contractual %q", nombre)
		}
	}
	if len(campos) != 8 {
		t.Fatalf("campos no versionados en la operación: %v", campos)
	}
}

func TestPreparadorPostgreSQLRestauraConfirmacionExacta(t *testing.T) {
	referencias := referenciasPreparacionPrueba()
	instante := time.Date(2026, 7, 23, 10, 0, 0, 123_456_000, time.UTC)
	fila := filaPreparacionPrueba{valores: []any{
		"confirmada",
		"reserva:alta-candidata-001",
		referencias.ExpedienteRef,
		referencias.NumeroVisible,
		referencias.ReciboRef,
		selloHMACPrueba(claveAmbitoAltaPrueba, "d"),
		selloHMACPrueba(clavePeticionAltaPrueba, "b"),
		"organizacion:diputacion-granada",
		"actor:tecnica-rrhh-001",
		"perfil:tecnica-rrhh",
		string(ports.PreparacionConfirmada),
		pgtype.Int8{Int64: 1, Valid: true},
		pgtype.Text{String: "auditoria:alta-001", Valid: true},
		pgtype.Text{String: "evento:alta-001", Valid: true},
		pgtype.Timestamptz{Time: instante, Valid: true},
	}}
	preparador, tx := nuevoPreparadorPrueba(t, fila)

	preparacion, err := preparador.PrepararAlta(
		context.Background(),
		solicitudPreparacionPrueba(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if preparacion.Estado != ports.PreparacionConfirmada ||
		preparacion.ReciboConfirmado == nil ||
		preparacion.ReciboConfirmado.ConfirmadaEn != instante ||
		tx.confirmaciones != 1 {
		t.Fatalf("confirmación inesperada: %#v", preparacion)
	}
}

func TestPreparadorPostgreSQLRechazaReutilizacionSemantica(t *testing.T) {
	preparador, tx := nuevoPreparadorPrueba(
		t,
		filaReservadaPreparacionPrueba("idempotencia_reutilizada"),
	)

	_, err := preparador.PrepararAlta(
		context.Background(),
		solicitudPreparacionPrueba(),
	)
	if !errors.Is(err, ports.ErrClaveIdempotenciaUsada) {
		t.Fatalf("error inesperado: %v", err)
	}
	if tx.confirmaciones != 0 {
		t.Fatal("se confirmó una reutilización semántica")
	}
}

func TestPreparadorPostgreSQLNoConfirmaRespuestaAdulterada(t *testing.T) {
	casos := map[string]struct {
		indice int
		valor  any
	}{
		"reserva candidata": {indice: 1, valor: "reserva:alta-ajena-001"},
		"expediente":        {indice: 2, valor: "expediente:ct-ajeno-001"},
		"numero visible":    {indice: 3, valor: "2026/CT-9999"},
		"recibo":            {indice: 4, valor: "recibo:alta-ajeno-001"},
		"ambito": {
			indice: 5,
			valor:  selloHMACPrueba(claveAmbitoAltaPrueba, "e"),
		},
		"huella": {
			indice: 6,
			valor:  selloHMACPrueba(clavePeticionAltaPrueba, "c"),
		},
		"organizacion": {indice: 7, valor: "organizacion:ajena"},
		"actor":        {indice: 8, valor: "actor:ajeno-001"},
		"perfil":       {indice: 9, valor: "perfil:ajeno"},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			fila := filaReservadaPreparacionPrueba("reservada").(filaPreparacionPrueba)
			fila.valores[caso.indice] = caso.valor
			preparador, tx := nuevoPreparadorPrueba(t, fila)

			_, err := preparador.PrepararAlta(
				context.Background(),
				solicitudPreparacionPrueba(),
			)
			if !errors.Is(err, ports.ErrPersistenciaNoDisponible) {
				t.Fatalf("error inesperado: %v", err)
			}
			if tx.confirmaciones != 0 {
				t.Fatal("se confirmó una respuesta adulterada")
			}
		})
	}
}

func TestPreparadorPostgreSQLPropagaCancelacionDeDependencias(t *testing.T) {
	t.Run("sellador", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		tx := &transaccionPreparacionPrueba{
			fila: filaReservadaPreparacionPrueba("reservada"),
		}
		preparador, err := nuevoPreparadorAltaPostgreSQL(
			&iniciadorPreparacionPrueba{tx: tx},
			&selladorAmbitoPrueba{
				huella:   selloHMACPrueba(claveAmbitoAltaPrueba, "d"),
				err:      context.Canceled,
				alLlamar: cancelar,
			},
			&generadorReferenciasPrueba{},
		)
		if err != nil {
			t.Fatal(err)
		}
		_, err = preparador.PrepararAlta(ctx, solicitudPreparacionPrueba())
		if !errors.Is(err, context.Canceled) || tx.configurada {
			t.Fatalf("cancelación perdida: %v", err)
		}
	})

	t.Run("referencias", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		tx := &transaccionPreparacionPrueba{
			fila: filaReservadaPreparacionPrueba("reservada"),
		}
		preparador, err := nuevoPreparadorAltaPostgreSQL(
			&iniciadorPreparacionPrueba{tx: tx},
			&selladorAmbitoPrueba{
				huella: selloHMACPrueba(claveAmbitoAltaPrueba, "d"),
			},
			&generadorReferenciasPrueba{
				errRefs:   context.Canceled,
				alGenerar: cancelar,
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		_, err = preparador.PrepararAlta(ctx, solicitudPreparacionPrueba())
		if !errors.Is(err, context.Canceled) || tx.configurada {
			t.Fatalf("cancelación perdida: %v", err)
		}
	})

	t.Run("reserva", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		tx := &transaccionPreparacionPrueba{
			fila: filaReservadaPreparacionPrueba("reservada"),
		}
		preparador, err := nuevoPreparadorAltaPostgreSQL(
			&iniciadorPreparacionPrueba{tx: tx},
			&selladorAmbitoPrueba{
				huella: selloHMACPrueba(claveAmbitoAltaPrueba, "d"),
			},
			&generadorReferenciasPrueba{
				referencias: referenciasPreparacionPrueba(),
				errReserva:  context.Canceled,
				alReservar:  cancelar,
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		_, err = preparador.PrepararAlta(ctx, solicitudPreparacionPrueba())
		if !errors.Is(err, context.Canceled) || tx.configurada {
			t.Fatalf("cancelación perdida: %v", err)
		}
	})
}

func TestPreparadorPostgreSQLFallaCerradoEnFronterasTransaccionales(t *testing.T) {
	t.Run("inicio", func(t *testing.T) {
		iniciador := &iniciadorPreparacionPrueba{
			err: errors.New("fallo de inicio"),
		}
		preparador, err := nuevoPreparadorAltaPostgreSQL(
			iniciador,
			&selladorAmbitoPrueba{
				huella: selloHMACPrueba(claveAmbitoAltaPrueba, "d"),
			},
			&generadorReferenciasPrueba{
				referencias: referenciasPreparacionPrueba(),
				reservaRef:  "reserva:alta-candidata-001",
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		_, err = preparador.PrepararAlta(
			context.Background(),
			solicitudPreparacionPrueba(),
		)
		if !errors.Is(err, ports.ErrPersistenciaNoDisponible) {
			t.Fatalf("error inesperado: %v", err)
		}
	})

	t.Run("configuracion", func(t *testing.T) {
		preparador, tx := nuevoPreparadorPrueba(
			t,
			filaReservadaPreparacionPrueba("reservada"),
		)
		tx.errConfigurar = errors.New("fallo de configuración")
		_, err := preparador.PrepararAlta(
			context.Background(),
			solicitudPreparacionPrueba(),
		)
		if !errors.Is(err, ports.ErrPersistenciaNoDisponible) ||
			tx.reversiones != 1 {
			t.Fatalf("fallo no cerrado: err=%v tx=%#v", err, tx)
		}
	})

	t.Run("lectura", func(t *testing.T) {
		preparador, tx := nuevoPreparadorPrueba(
			t,
			filaPreparacionPrueba{err: errors.New("fallo de lectura")},
		)
		_, err := preparador.PrepararAlta(
			context.Background(),
			solicitudPreparacionPrueba(),
		)
		if !errors.Is(err, ports.ErrPersistenciaNoDisponible) ||
			tx.confirmaciones != 0 {
			t.Fatalf("fallo no cerrado: err=%v tx=%#v", err, tx)
		}
	})

	t.Run("commit", func(t *testing.T) {
		preparador, tx := nuevoPreparadorPrueba(
			t,
			filaReservadaPreparacionPrueba("reservada"),
		)
		tx.errConfirmar = errors.New("resultado indeterminado")
		_, err := preparador.PrepararAlta(
			context.Background(),
			solicitudPreparacionPrueba(),
		)
		if !errors.Is(err, ports.ErrPersistenciaNoDisponible) ||
			tx.confirmaciones != 1 {
			t.Fatalf("fallo no cerrado: err=%v tx=%#v", err, tx)
		}
	})
}

func TestPreparadorPostgreSQLRechazaDependenciaNulaTipada(t *testing.T) {
	var sellador *selladorAmbitoPrueba
	_, err := nuevoPreparadorAltaPostgreSQL(
		&iniciadorPreparacionPrueba{},
		sellador,
		&generadorReferenciasPrueba{},
	)
	if !errors.Is(err, ports.ErrPersistenciaNoDisponible) {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestPreparadorPostgreSQLRechazaHuellasNulas(t *testing.T) {
	t.Run("huella de la peticion", func(t *testing.T) {
		preparador, tx := nuevoPreparadorPrueba(
			t,
			filaReservadaPreparacionPrueba("reservada"),
		)
		solicitud := solicitudPreparacionPrueba()
		solicitud.HuellaPeticionHMAC =
			"hmac-sha256:" + clavePeticionAltaPrueba + ":" + strings.Repeat("0", 64)

		_, err := preparador.PrepararAlta(context.Background(), solicitud)
		if !errors.Is(err, ports.ErrPreparacionAltaInvalida) {
			t.Fatalf("error inesperado: %v", err)
		}
		if tx.configurada {
			t.Fatal("se inició PostgreSQL con una huella nula")
		}
	})

	t.Run("ambito idempotente", func(t *testing.T) {
		tx := &transaccionPreparacionPrueba{
			fila: filaReservadaPreparacionPrueba("reservada"),
		}
		preparador, err := nuevoPreparadorAltaPostgreSQL(
			&iniciadorPreparacionPrueba{tx: tx},
			&selladorAmbitoPrueba{
				huella: "hmac-sha256:" + claveAmbitoAltaPrueba + ":" +
					strings.Repeat("0", 64),
			},
			&generadorReferenciasPrueba{
				referencias: referenciasPreparacionPrueba(),
				reservaRef:  "reserva:alta-candidata-001",
			},
		)
		if err != nil {
			t.Fatal(err)
		}

		_, err = preparador.PrepararAlta(
			context.Background(),
			solicitudPreparacionPrueba(),
		)
		if !errors.Is(err, ports.ErrPersistenciaNoDisponible) {
			t.Fatalf("error inesperado: %v", err)
		}
		if tx.configurada {
			t.Fatal("se inició PostgreSQL con un ámbito nulo")
		}
	})
}

func selloHMACPrueba(dominio, caracter string) string {
	return "hmac-sha256:" + dominio + ":" + strings.Repeat(caracter, 64)
}
