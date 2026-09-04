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
	claveAmbitoAltaPrueba     = "vec.contratacion-temporal.ambito-idempotencia/v1"
	clavePeticionAltaPrueba   = "vec.contratacion-temporal.huella-peticion/v1"
	claveAmbitoAltaPruebaV2   = "vec.contratacion-temporal.ambito-idempotencia/v2"
	clavePeticionAltaPruebaV2 = "vec.contratacion-temporal.huella-peticion/v2"
)

type iniciadorPreparacionPrueba struct {
	tx            pgx.Tx
	err           error
	transacciones []pgx.Tx
	errores       []error
	opciones      pgx.TxOptions
	inicios       int
}

func (i *iniciadorPreparacionPrueba) BeginTx(
	_ context.Context,
	opciones pgx.TxOptions,
) (pgx.Tx, error) {
	i.opciones = opciones
	i.inicios++
	indice := i.inicios - 1
	if indice < len(i.errores) && i.errores[indice] != nil {
		return nil, i.errores[indice]
	}
	if indice < len(i.transacciones) {
		return i.transacciones[indice], nil
	}
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
	alConfirmar    func()
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
	if t.alConfirmar != nil {
		t.alConfirmar()
	}
	return t.errConfirmar
}

func (t *transaccionPreparacionPrueba) Rollback(context.Context) error {
	t.reversiones++
	return nil
}

type filaPreparacionPrueba struct {
	valores    []any
	err        error
	alEscanear func()
}

func (f filaPreparacionPrueba) Scan(destinos ...any) error {
	if f.alEscanear != nil {
		f.alEscanear()
	}
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
		case *int64:
			numero, valido := valor.(int64)
			if !valido {
				return errors.New("entero nativo inválido")
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
	retenidas []string
	err       error
	alLlamar  func()
	solicitud ports.SolicitudSellarAmbitoIdempotencia
}

func (s *selladorAmbitoPrueba) SellarAmbitoIdempotencia(
	_ context.Context,
	solicitud ports.SolicitudSellarAmbitoIdempotencia,
) (ports.ColeccionSellosHMAC, error) {
	if s.alLlamar != nil {
		s.alLlamar()
	}
	s.solicitud = solicitud
	if s.err != nil {
		return ports.ColeccionSellosHMAC{}, s.err
	}
	return ports.NuevaColeccionSellosHMAC(s.huella, s.retenidas)
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
		ClaveIdempotencia: "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
		HuellasPeticionHMAC: coleccionPostgreSQLPrueba(
			selloHMACPrueba(clavePeticionAltaPrueba, "b"),
		),
		OrganizacionRef: "organizacion:diputacion-granada",
		ActorRef:        "actor:tecnica-rrhh-001",
		PerfilRef:       "perfil:tecnica-rrhh",
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

func filaConfirmadaPreparacionPrueba() filaPreparacionPrueba {
	referencias := referenciasPreparacionPrueba()
	instante := time.Date(
		2026, 7, 23, 10, 0, 0, 123_456_000,
		time.FixedZone("postgresql-utc", 0),
	)
	return filaPreparacionPrueba{valores: []any{
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
		!strings.Contains(tx.consulta, funcionPrepararAltaV2) {
		t.Fatalf("resultado inesperado: preparacion=%#v tx=%#v", preparacion, tx)
	}
	if strings.Contains(string(tx.operacion), solicitud.ClaveIdempotencia) {
		t.Fatal("la operación PostgreSQL contiene la clave idempotente en claro")
	}
	var operacion operacionPrepararAltaV2
	if err := json.Unmarshal(tx.operacion, &operacion); err != nil {
		t.Fatal(err)
	}
	if operacion.SellosHMAC.Activo.AmbitoHMAC !=
		selloHMACPrueba(claveAmbitoAltaPrueba, "d") ||
		operacion.Esquema != esquemaPrepararAltaV2 {
		t.Fatalf("operación no ligada: %#v", operacion)
	}
	var campos map[string]json.RawMessage
	if err := json.Unmarshal(tx.operacion, &campos); err != nil {
		t.Fatal(err)
	}
	for _, nombre := range []string{
		"esquema", "sellos_hmac", "organizacion_ref", "actor_ref", "perfil_ref",
		"reserva_ref_candidata", "referencias_candidatas",
	} {
		if _, existe := campos[nombre]; !existe {
			t.Fatalf("falta el campo contractual %q", nombre)
		}
	}
	if len(campos) != 7 {
		t.Fatalf("campos no versionados en la operación: %v", campos)
	}
}

func TestPreparadorPostgreSQLSerializaMatrizV2Cerrada(t *testing.T) {
	fila := filaReservadaPreparacionPrueba("reservada").(filaPreparacionPrueba)
	fila.valores[5] = selloHMACPrueba(claveAmbitoAltaPruebaV2, "e")
	fila.valores[6] = selloHMACPrueba(clavePeticionAltaPruebaV2, "c")
	tx := &transaccionPreparacionPrueba{fila: fila}
	preparador, err := nuevoPreparadorAltaPostgreSQL(
		&iniciadorPreparacionPrueba{tx: tx},
		&selladorAmbitoPrueba{
			huella: selloHMACPrueba(claveAmbitoAltaPruebaV2, "e"),
			retenidas: []string{
				selloHMACPrueba(claveAmbitoAltaPrueba, "d"),
			},
		},
		&generadorReferenciasPrueba{
			referencias: referenciasPreparacionPrueba(),
			reservaRef:  "reserva:alta-candidata-001",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := solicitudPreparacionPrueba()
	solicitud.HuellasPeticionHMAC = coleccionPostgreSQLPrueba(
		selloHMACPrueba(clavePeticionAltaPruebaV2, "c"),
		selloHMACPrueba(clavePeticionAltaPrueba, "b"),
	)
	if _, err := preparador.PrepararAlta(
		context.Background(),
		solicitud,
	); err != nil {
		t.Fatal(err)
	}
	var operacion operacionPrepararAltaV2
	if err := json.Unmarshal(tx.operacion, &operacion); err != nil {
		t.Fatal(err)
	}
	if operacion.SellosHMAC.Activo.Generacion != 2 ||
		len(operacion.SellosHMAC.Retenidos) != 1 ||
		operacion.SellosHMAC.Retenidos[0].Generacion != 1 {
		t.Fatalf("matriz v2 incorrecta: %#v", operacion.SellosHMAC)
	}
	contenido := map[string]json.RawMessage{}
	if err := json.Unmarshal(tx.operacion, &contenido); err != nil {
		t.Fatal(err)
	}
	sellos := map[string]json.RawMessage{}
	if err := json.Unmarshal(contenido["sellos_hmac"], &sellos); err != nil {
		t.Fatal(err)
	}
	if len(sellos) != 2 || sellos["activo"] == nil || sellos["retenidos"] == nil {
		t.Fatalf("contrato anidado abierto o incompleto: %v", sellos)
	}
}

func TestPreparadorPostgreSQLAceptaCanonicoV1MedianteAliasRetenido(t *testing.T) {
	fila := filaConfirmadaPreparacionPrueba()
	tx := &transaccionPreparacionPrueba{fila: fila}
	preparador, err := nuevoPreparadorAltaPostgreSQL(
		&iniciadorPreparacionPrueba{tx: tx},
		&selladorAmbitoPrueba{
			huella: selloHMACPrueba(claveAmbitoAltaPruebaV2, "e"),
			retenidas: []string{
				selloHMACPrueba(claveAmbitoAltaPrueba, "d"),
			},
		},
		&generadorReferenciasPrueba{
			referencias: referenciasPreparacionPrueba(),
			reservaRef:  "reserva:alta-candidata-001",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := solicitudPreparacionPrueba()
	solicitud.HuellasPeticionHMAC = coleccionPostgreSQLPrueba(
		selloHMACPrueba(clavePeticionAltaPruebaV2, "c"),
		selloHMACPrueba(clavePeticionAltaPrueba, "b"),
	)
	preparacion, err := preparador.PrepararAlta(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	if preparacion.AmbitoIdempotenciaHMAC !=
		selloHMACPrueba(claveAmbitoAltaPrueba, "d") ||
		preparacion.HuellaPeticionHMAC !=
			selloHMACPrueba(clavePeticionAltaPrueba, "b") ||
		preparacion.Estado != ports.PreparacionConfirmada ||
		preparacion.ReciboConfirmado == nil {
		t.Fatalf("el canónico histórico no se conservó: %#v", preparacion)
	}
}

func TestPreparadorPostgreSQLRestauraConfirmacionExacta(t *testing.T) {
	instante := time.Date(2026, 7, 23, 10, 0, 0, 123_456_000, time.UTC)
	fila := filaConfirmadaPreparacionPrueba()
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

func TestPreparadorPostgreSQLCotejaAmbitoEnTodoReintento(t *testing.T) {
	casos := map[string]filaPreparacionPrueba{
		"reserva reutilizada": filaReservadaPreparacionPrueba(
			"reutilizada",
		).(filaPreparacionPrueba),
		"confirmacion": filaConfirmadaPreparacionPrueba(),
	}
	for nombre, fila := range casos {
		t.Run(nombre, func(t *testing.T) {
			fila.valores[5] = selloHMACPrueba(claveAmbitoAltaPrueba, "e")
			preparador, tx := nuevoPreparadorPrueba(t, fila)

			_, err := preparador.PrepararAlta(
				context.Background(),
				solicitudPreparacionPrueba(),
			)
			if !errors.Is(err, ports.ErrPersistenciaNoDisponible) {
				t.Fatalf("ámbito sustituido no rechazado: %v", err)
			}
			if tx.confirmaciones != 0 {
				t.Fatal("se confirmó una respuesta ligada a otro ámbito")
			}
		})
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
		solicitud.HuellasPeticionHMAC = ports.ColeccionSellosHMAC{}

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

func coleccionPostgreSQLPrueba(
	activo string,
	retenidos ...string,
) ports.ColeccionSellosHMAC {
	coleccion, err := ports.NuevaColeccionSellosHMAC(activo, retenidos)
	if err != nil {
		panic(err)
	}
	return coleccion
}
