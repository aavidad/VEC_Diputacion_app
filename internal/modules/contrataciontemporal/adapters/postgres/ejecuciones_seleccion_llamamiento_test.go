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

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	claveEjecucionSeleccionO6Prueba = "018f47a2-6b31-4c80-8a95-4d2e707c5a11"
	clavePeticionSeleccionO6Prueba  = "vec.contratacion-temporal.integracion-bolsa-peticion/v1"
	claveRespuestaSeleccionO6Prueba = "vec.contratacion-temporal.integracion-bolsa-respuesta/v1"
)

type iniciadorEjecucionSeleccionO6Prueba struct {
	tx       pgx.Tx
	err      error
	inicios  int
	opciones pgx.TxOptions
}

func (i *iniciadorEjecucionSeleccionO6Prueba) BeginTx(
	_ context.Context,
	opciones pgx.TxOptions,
) (pgx.Tx, error) {
	i.inicios++
	i.opciones = opciones
	return i.tx, i.err
}

type transaccionEjecucionSeleccionO6Prueba struct {
	pgx.Tx
	fila            pgx.Row
	errConfigurar   error
	errCommit       error
	configuraciones int
	confirmaciones  int
	reversiones     int
	consultas       []string
	argumentos      [][]any
}

func (t *transaccionEjecucionSeleccionO6Prueba) Exec(
	_ context.Context,
	_ string,
	_ ...any,
) (pgconn.CommandTag, error) {
	t.configuraciones++
	return pgconn.NewCommandTag("SELECT 1"), t.errConfigurar
}

func (t *transaccionEjecucionSeleccionO6Prueba) QueryRow(
	_ context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	t.consultas = append(t.consultas, consulta)
	t.argumentos = append(t.argumentos, append([]any(nil), argumentos...))
	return t.fila
}

func (t *transaccionEjecucionSeleccionO6Prueba) Commit(context.Context) error {
	t.confirmaciones++
	return t.errCommit
}

func (t *transaccionEjecucionSeleccionO6Prueba) Rollback(context.Context) error {
	t.reversiones++
	return nil
}

type filaEjecucionSeleccionO6Prueba struct {
	valores []any
	err     error
}

func (f filaEjecucionSeleccionO6Prueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != len(f.valores) {
		return errors.New("columnas inesperadas")
	}
	for indice, destino := range destinos {
		switch puntero := destino.(type) {
		case *string:
			valor, ok := f.valores[indice].(string)
			if !ok {
				return errors.New("texto inesperado")
			}
			*puntero = valor
		case *bool:
			valor, ok := f.valores[indice].(bool)
			if !ok {
				return errors.New("booleano inesperado")
			}
			*puntero = valor
		default:
			return errors.New("destino inesperado")
		}
	}
	return nil
}

func TestEjecucionesSeleccionO6ConstructorFallaCerradoSinPool(t *testing.T) {
	t.Parallel()
	adaptador, err := nuevasEjecucionesSeleccionLlamamientoPostgreSQL(nil)
	if adaptador != nil || !errors.Is(err, errEjecucionesSeleccionLlamamientoPostgreSQL) {
		t.Fatalf("constructor abierto: adaptador=%v err=%v", adaptador, err)
	}
}

func TestEjecucionesSeleccionO6ReservaClasificaTodosLosEstados(t *testing.T) {
	solicitud, recibo, artefacto := materialesEjecucionSeleccionO6Prueba(t)
	solicitudJSON := string(debeCodificarSolicitudSeleccionO6Prueba(t, solicitud))
	reciboJSON := string(debeJSONSeleccionO6Prueba(t, recibo))
	artefactoJSON := string(debeJSONSeleccionO6Prueba(t, artefacto))
	reserva := prefijoReservaSeleccionO6 + "6e76b7ca-7fa8-41de-b097-e0401d4769aa"

	casos := []struct {
		nombre    string
		fila      []any
		situacion ports.SituacionEjecucionSeleccionLlamamiento
		efecto    ports.EfectoSeleccionLlamamiento
		reserva   string
	}{
		{"propietaria", []any{"propietaria", solicitudJSON, reserva, "", "", ""},
			ports.EjecucionSeleccionLlamamientoPropietaria, "", reserva},
		{"ocupada", []any{"ocupada", solicitudJSON, "", "preparar_orden", "", ""},
			ports.EjecucionSeleccionLlamamientoOcupada,
			ports.EfectoPrepararOrdenSeleccionLlamamiento, ""},
		{"colision", []any{"colision", solicitudJSON, "", "", "", ""},
			ports.EjecucionSeleccionLlamamientoColision, "", ""},
		{"indeterminada", []any{"indeterminada", solicitudJSON, "", "solicitar_llamamiento", "", ""},
			ports.EjecucionSeleccionLlamamientoIndeterminada,
			ports.EfectoSolicitarSeleccionLlamamiento, ""},
		{"confirmada", []any{"confirmada", solicitudJSON, "", "", reciboJSON, artefactoJSON},
			ports.EjecucionSeleccionLlamamientoConfirmada, "", ""},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			adaptador, iniciador, tx := nuevoAdaptadorEjecucionSeleccionO6Prueba(
				t, filaEjecucionSeleccionO6Prueba{valores: caso.fila},
			)
			estado, err := adaptador.Reservar(context.Background(), solicitud)
			if err != nil || estado.Solicitud != solicitud || estado.Situacion != caso.situacion ||
				estado.EfectoPosible != caso.efecto || estado.ReservaRef != caso.reserva {
				t.Fatalf("estado=%#v err=%v", estado, err)
			}
			if caso.situacion == ports.EjecucionSeleccionLlamamientoConfirmada &&
				(estado.ReciboConfirmado != recibo || estado.ArtefactoConfirmado != artefacto) {
				t.Fatal("el terminal no conservo el par exacto")
			}
			exigirTransaccionSeleccionO6Prueba(t, iniciador, tx, pgx.ReadWrite,
				funcionReservarSeleccionO6)
		})
	}
}

func TestEjecucionesSeleccionO6ResuelveSoloTerminalDurable(t *testing.T) {
	solicitud, recibo, artefacto := materialesEjecucionSeleccionO6Prueba(t)
	solicitudJSON := string(debeCodificarSolicitudSeleccionO6Prueba(t, solicitud))
	reciboJSON := string(debeJSONSeleccionO6Prueba(t, recibo))
	artefactoJSON := string(debeJSONSeleccionO6Prueba(t, artefacto))

	for _, caso := range []struct {
		nombre     string
		fila       []any
		confirmada bool
		situacion  ports.SituacionEjecucionSeleccionLlamamiento
	}{
		{"ausente", []any{"", "", "", "", "", ""}, false, ""},
		{"indeterminada", []any{"indeterminada", solicitudJSON, "", "preparar_orden", "", ""},
			false, ports.EjecucionSeleccionLlamamientoIndeterminada},
		{"confirmada", []any{"confirmada", solicitudJSON, "", "", reciboJSON, artefactoJSON},
			true, ports.EjecucionSeleccionLlamamientoConfirmada},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			adaptador, iniciador, tx := nuevoAdaptadorEjecucionSeleccionO6Prueba(
				t, filaEjecucionSeleccionO6Prueba{valores: caso.fila},
			)
			estado, confirmada, err := adaptador.ResolverTerminal(
				context.Background(), claveEjecucionSeleccionO6Prueba,
			)
			if err != nil || confirmada != caso.confirmada || estado.Situacion != caso.situacion {
				t.Fatalf("estado=%#v confirmada=%v err=%v", estado, confirmada, err)
			}
			exigirTransaccionSeleccionO6Prueba(t, iniciador, tx, pgx.ReadOnly,
				funcionResolverTerminalSeleccionO6)
		})
	}
}

func TestEjecucionesSeleccionO6ConsultaEstadoPorParExacto(t *testing.T) {
	t.Parallel()
	solicitud, _, _ := materialesEjecucionSeleccionO6Prueba(t)
	adaptador, iniciador, tx := nuevoAdaptadorEjecucionSeleccionO6Prueba(
		t, filaEjecucionSeleccionO6Prueba{valores: []any{
			"ocupada", string(debeCodificarSolicitudSeleccionO6Prueba(t, solicitud)),
			"", "preparar_orden", "", "",
		}},
	)
	estado, err := adaptador.ConsultarEstado(context.Background(), solicitud)
	if err != nil || estado.Solicitud != solicitud ||
		estado.Situacion != ports.EjecucionSeleccionLlamamientoOcupada ||
		estado.EfectoPosible != ports.EfectoPrepararOrdenSeleccionLlamamiento {
		t.Fatalf("consulta no ligada al par exacto: estado=%#v err=%v", estado, err)
	}
	exigirTransaccionSeleccionO6Prueba(t, iniciador, tx, pgx.ReadOnly,
		funcionConsultarSeleccionO6)
}

func TestEjecucionesSeleccionO6MutaSoloMedianteFachadasNominales(t *testing.T) {
	solicitud, recibo, artefacto := materialesEjecucionSeleccionO6Prueba(t)
	reserva := ports.ReservaEjecucionSeleccionLlamamiento{
		Solicitud:  solicitud,
		ReservaRef: prefijoReservaSeleccionO6 + "6e76b7ca-7fa8-41de-b097-e0401d4769aa",
	}
	casos := []struct {
		nombre  string
		funcion string
		llamar  func(*EjecucionesSeleccionLlamamientoPostgreSQL) error
	}{
		{"abrir orden", funcionAbrirVentanaSeleccionO6, func(a *EjecucionesSeleccionLlamamientoPostgreSQL) error {
			return a.AbrirVentanaEfecto(context.Background(), reserva,
				ports.EfectoPrepararOrdenSeleccionLlamamiento)
		}},
		{"abrir llamamiento", funcionAbrirVentanaSeleccionO6, func(a *EjecucionesSeleccionLlamamientoPostgreSQL) error {
			return a.AbrirVentanaEfecto(context.Background(), reserva,
				ports.EfectoSolicitarSeleccionLlamamiento)
		}},
		{"indeterminada", funcionMarcarIndeterminadaO6, func(a *EjecucionesSeleccionLlamamientoPostgreSQL) error {
			return a.MarcarIndeterminada(context.Background(), reserva,
				ports.EfectoPrepararOrdenSeleccionLlamamiento)
		}},
		{"liberar", funcionLiberarSeleccionO6, func(a *EjecucionesSeleccionLlamamientoPostgreSQL) error {
			return a.LiberarAntesDeEfectos(context.Background(), reserva)
		}},
		{"confirmar", funcionConfirmarSeleccionO6, func(a *EjecucionesSeleccionLlamamientoPostgreSQL) error {
			return a.Confirmar(context.Background(), reserva, recibo, artefacto)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			adaptador, iniciador, tx := nuevoAdaptadorEjecucionSeleccionO6Prueba(
				t, filaEjecucionSeleccionO6Prueba{valores: []any{true}},
			)
			if err := caso.llamar(adaptador); err != nil {
				t.Fatal(err)
			}
			exigirTransaccionSeleccionO6Prueba(t, iniciador, tx, pgx.ReadWrite, caso.funcion)
		})
	}
}

func TestEjecucionesSeleccionO6CommitAmbiguoNuncaEsExitoNiSeReintenta(t *testing.T) {
	solicitud, _, _ := materialesEjecucionSeleccionO6Prueba(t)
	reserva := ports.ReservaEjecucionSeleccionLlamamiento{
		Solicitud:  solicitud,
		ReservaRef: prefijoReservaSeleccionO6 + "6e76b7ca-7fa8-41de-b097-e0401d4769aa",
	}
	causaPrivada := errors.New("detalle-interno-no-publicable")
	adaptador, iniciador, tx := nuevoAdaptadorEjecucionSeleccionO6Prueba(
		t, filaEjecucionSeleccionO6Prueba{valores: []any{true}},
	)
	tx.errCommit = causaPrivada
	err := adaptador.AbrirVentanaEfecto(
		context.Background(), reserva, ports.EfectoPrepararOrdenSeleccionLlamamiento,
	)
	if !errors.Is(err, errEjecucionesSeleccionLlamamientoPostgreSQL) ||
		errors.Is(err, causaPrivada) || strings.Contains(err.Error(), "no-publicable") ||
		iniciador.inicios != 1 || tx.confirmaciones != 1 {
		t.Fatalf("commit ambiguo mal clasificado: inicios=%d commits=%d err=%v",
			iniciador.inicios, tx.confirmaciones, err)
	}
}

func TestEjecucionesSeleccionO6NoReintentaSQLStateConcurrente(t *testing.T) {
	solicitud, _, _ := materialesEjecucionSeleccionO6Prueba(t)
	adaptador, iniciador, _ := nuevoAdaptadorEjecucionSeleccionO6Prueba(
		t, filaEjecucionSeleccionO6Prueba{err: &pgconn.PgError{Code: "40001",
			Message: "detalle privado"}},
	)
	_, err := adaptador.Reservar(context.Background(), solicitud)
	if !errors.Is(err, errEjecucionesSeleccionLlamamientoPostgreSQL) ||
		strings.Contains(err.Error(), "detalle privado") || iniciador.inicios != 1 {
		t.Fatalf("fallo concurrente reintentado o filtrado: inicios=%d err=%v",
			iniciador.inicios, err)
	}
}

func TestEjecucionesSeleccionO6RechazaEntradasAntesDePGX(t *testing.T) {
	adaptador, iniciador, _ := nuevoAdaptadorEjecucionSeleccionO6Prueba(
		t, filaEjecucionSeleccionO6Prueba{valores: []any{true}},
	)
	if err := adaptador.AbrirVentanaEfecto(context.Background(),
		ports.ReservaEjecucionSeleccionLlamamiento{}, "efecto_libre"); !errors.Is(err, errEjecucionesSeleccionLlamamientoPostgreSQL) {
		t.Fatalf("efecto libre aceptado: %v", err)
	}
	if _, err := adaptador.Reservar(context.Background(),
		ports.SolicitudReservaEjecucionSeleccionLlamamiento{}); !errors.Is(err, errEjecucionesSeleccionLlamamientoPostgreSQL) {
		t.Fatalf("solicitud vacia aceptada: %v", err)
	}
	if iniciador.inicios != 0 {
		t.Fatalf("entrada invalida cruzo PGX: %d", iniciador.inicios)
	}
}

func TestEjecucionesSeleccionO6TerminalAdulteradoFallaOpaco(t *testing.T) {
	solicitud, recibo, _ := materialesEjecucionSeleccionO6Prueba(t)
	adaptador, _, _ := nuevoAdaptadorEjecucionSeleccionO6Prueba(t,
		filaEjecucionSeleccionO6Prueba{valores: []any{
			"confirmada", string(debeCodificarSolicitudSeleccionO6Prueba(t, solicitud)),
			"", "", string(debeJSONSeleccionO6Prueba(t, recibo)), `{}`,
		}},
	)
	_, _, err := adaptador.ResolverTerminal(context.Background(), claveEjecucionSeleccionO6Prueba)
	if !errors.Is(err, errEjecucionesSeleccionLlamamientoPostgreSQL) {
		t.Fatalf("terminal adulterado aceptado: %v", err)
	}
}

func nuevoAdaptadorEjecucionSeleccionO6Prueba(
	t *testing.T,
	fila pgx.Row,
) (*EjecucionesSeleccionLlamamientoPostgreSQL,
	*iniciadorEjecucionSeleccionO6Prueba, *transaccionEjecucionSeleccionO6Prueba) {
	t.Helper()
	tx := &transaccionEjecucionSeleccionO6Prueba{fila: fila}
	iniciador := &iniciadorEjecucionSeleccionO6Prueba{tx: tx}
	adaptador, err := nuevasEjecucionesSeleccionLlamamientoPostgreSQL(iniciador)
	if err != nil {
		t.Fatal(err)
	}
	return adaptador, iniciador, tx
}

func exigirTransaccionSeleccionO6Prueba(
	t *testing.T,
	iniciador *iniciadorEjecucionSeleccionO6Prueba,
	tx *transaccionEjecucionSeleccionO6Prueba,
	modo pgx.TxAccessMode,
	funcion string,
) {
	t.Helper()
	if iniciador.inicios != 1 || iniciador.opciones.IsoLevel != pgx.Serializable ||
		iniciador.opciones.AccessMode != modo || tx.configuraciones != 1 ||
		tx.confirmaciones != 1 || len(tx.consultas) != 1 ||
		!strings.Contains(tx.consultas[0], funcion) {
		t.Fatalf("frontera fisica incorrecta: iniciador=%#v tx=%#v", iniciador, tx)
	}
}

func debeCodificarSolicitudSeleccionO6Prueba(
	t *testing.T,
	solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento,
) []byte {
	t.Helper()
	contenido, err := codificarSolicitudSeleccionO6(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	return contenido
}

func debeJSONSeleccionO6Prueba(t *testing.T, valor any) []byte {
	t.Helper()
	contenido, err := json.Marshal(valor)
	if err != nil {
		t.Fatal(err)
	}
	return contenido
}

type selladorContextoSeleccionO6Prueba struct{}

func (selladorContextoSeleccionO6Prueba) SellarDatos(context.Context, []byte) (string, error) {
	return selloSeleccionO6Prueba(clavePeticionSeleccionO6Prueba, 'a'), nil
}

type verificadorSeleccionO6Prueba struct{}

func (verificadorSeleccionO6Prueba) VerificarDatos(
	ctx context.Context,
	_ string,
	_ []byte,
	_ string,
) error {
	return ctx.Err()
}

func materialesEjecucionSeleccionO6Prueba(t *testing.T) (
	ports.SolicitudReservaEjecucionSeleccionLlamamiento,
	ports.ReciboSolicitudLlamamientoBolsa,
	ports.ArtefactoProbatorioLlamamientoBolsa,
) {
	t.Helper()
	base := time.Date(2026, time.August, 31, 9, 0, 0, 0, time.UTC)
	emisor, err := ports.NuevoEmisorContextoPeticionIntegracionBolsa(
		"autoridad:contratacion-temporal", clavePeticionSeleccionO6Prueba,
		selladorContextoSeleccionO6Prueba{},
	)
	if err != nil {
		t.Fatal(err)
	}
	bolsa := referenciaSeleccionO6Prueba("bolsa:vigente:001", 'b')
	necesidad := referenciaSeleccionO6Prueba("necesidad:cobertura:001", 'c')
	politica := referenciaSeleccionO6Prueba("politica:llamamiento:001", 'd')
	finalidad := referenciaSeleccionO6Prueba("finalidad:seleccion:001", 'e')
	contextoOrden := emitirContextoSeleccionO6Prueba(
		t, emisor, base, "operacion:orden:001", bolsa,
		referenciaSeleccionO6Prueba("accion:orden:001", 'f'), finalidad,
	)
	comandoOrden := ports.ComandoPrepararOrdenBolsa{
		Contexto: contextoOrden, Necesidad: necesidad, Bolsa: bolsa,
		Politica: politica, MaximoPosiciones: 3,
	}
	reciboOrden := ports.ReciboOrdenBolsa{
		OperacionRef: "operacion:orden:001", OrganizacionRef: "organizacion:diputacion",
		ExpedienteRef: "expediente:temporal:001", VersionExpediente: 7,
		CorrelacionRef: "correlacion:seleccion:001", Necesidad: necesidad, Bolsa: bolsa,
		Politica: politica, Resultado: referenciaSeleccionO6Prueba("resultado:orden:001", '1'),
		OrdenGenerada: true, OrdenCompleta: true,
		Orden:             referenciaSeleccionO6Prueba("orden:bolsa:001", '2'),
		AccionLlamamiento: referenciaSeleccionO6Prueba("accion:llamamiento:001", '3'),
		TotalPosiciones:   3, ReciboRef: "recibo:orden:001",
		AuditoriaRef: "auditoria:orden:001", EventoRef: "evento:orden:001",
		ConfirmadaEn: base.Add(time.Minute),
		Procedencia:  procedenciaSeleccionO6Prueba(base, "respuesta:orden:001", "evidencia:orden:001"),
	}
	verificador, err := ports.NuevoVerificadorEvidenciaIntegracionBolsa(
		"autoridad:bolsa", claveRespuestaSeleccionO6Prueba, nil,
		verificadorSeleccionO6Prueba{},
	)
	if err != nil {
		t.Fatal(err)
	}
	comprobanteOrden, _, err := verificador.VerificarReciboOrden(
		context.Background(), comandoOrden, reciboOrden, base.Add(3*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	contextoLlamamiento := emitirContextoSeleccionO6Prueba(
		t, emisor, base, "operacion:llamamiento:001", reciboOrden.Orden,
		reciboOrden.AccionLlamamiento, finalidad,
	)
	comandoLlamamiento, err := ports.NuevoComandoSolicitarLlamamientoBolsa(
		ports.PreparacionComandoSolicitarLlamamientoBolsa{
			Contexto: contextoLlamamiento, ComandoOrden: comandoOrden,
			ReciboOrden: reciboOrden, ComprobanteOrden: comprobanteOrden,
			MaximaPosicionEvaluable: 3,
		}, base.Add(3*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	seudonimo, err := ports.NuevoSeudonimoSeleccionBolsa(
		"hmac-sha256:vec.contratacion-temporal.seleccion/v1:" + strings.Repeat("9", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	recibo := ports.ReciboSolicitudLlamamientoBolsa{
		OperacionRef: "operacion:llamamiento:001", OrganizacionRef: "organizacion:diputacion",
		ExpedienteRef: "expediente:temporal:001", VersionExpediente: 7,
		CorrelacionRef: "correlacion:seleccion:001", Necesidad: necesidad, Bolsa: bolsa,
		Orden: reciboOrden.Orden, Politica: politica,
		Resultado:         referenciaSeleccionO6Prueba("resultado:llamamiento:001", '4'),
		PropuestaGenerada: true,
		Propuesta:         referenciaSeleccionO6Prueba("propuesta:llamamiento:001", '5'),
		AccionEvento:      referenciaSeleccionO6Prueba("accion:evento:001", '6'),
		LlamamientoRef:    "llamamiento:seleccion:001", SeleccionRef: seudonimo,
		RetencionSeleccion: referenciaSeleccionO6Prueba("retencion:seleccion:001", '7'),
		OrdenSeleccionado:  2, ReciboRef: "recibo:llamamiento:001",
		AuditoriaRef: "auditoria:llamamiento:001", EventoRef: "evento:llamamiento:001",
		ConfirmadaEn: base.Add(time.Minute),
		Procedencia:  procedenciaSeleccionO6Prueba(base, "respuesta:llamamiento:001", "evidencia:llamamiento:001"),
	}
	comprobante, evidencia, err := verificador.VerificarReciboLlamamiento(
		context.Background(), comandoLlamamiento, recibo, base.Add(3*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	artefacto, err := ports.NuevoArtefactoProbatorioLlamamientoBolsa(
		comandoLlamamiento, recibo, evidencia, comprobante,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := ports.NuevaSolicitudReservaEjecucionSeleccionLlamamiento(
		claveEjecucionSeleccionO6Prueba, comandoOrden, 3, base.Add(3*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud, recibo, artefacto
}

func emitirContextoSeleccionO6Prueba(
	t *testing.T,
	emisor *ports.EmisorContextoPeticionIntegracionBolsa,
	base time.Time,
	operacion string,
	recurso ports.ReferenciaVersionadaIntegracionBolsa,
	accion ports.ReferenciaVersionadaIntegracionBolsa,
	finalidad ports.ReferenciaVersionadaIntegracionBolsa,
) ports.ContextoPeticionIntegracionBolsa {
	t.Helper()
	contexto, err := emisor.Emitir(context.Background(),
		ports.DatosContextoPeticionIntegracionBolsa{
			OperacionRef: operacion, OrganizacionRef: "organizacion:diputacion",
			ExpedienteRef: "expediente:temporal:001", VersionExpediente: 7,
			CorrelacionRef:       "correlacion:seleccion:001",
			ContratoVersion:      ports.VersionContratoIntegracionBolsa,
			AutoridadSolicitante: "autoridad:contratacion-temporal",
			Autorizacion:         referenciaSeleccionO6Prueba("autorizacion:seleccion:001", '8'),
			Accion:               accion, Recurso: recurso, Finalidad: finalidad,
			SolicitadaEn: base, ValidaHasta: base.Add(10 * time.Minute),
		}, base)
	if err != nil {
		t.Fatal(err)
	}
	return contexto
}

func referenciaSeleccionO6Prueba(
	referencia string,
	caracter byte,
) ports.ReferenciaVersionadaIntegracionBolsa {
	return ports.ReferenciaVersionadaIntegracionBolsa{
		Referencia: referencia, Version: 1,
		HuellaSHA256: strings.Repeat(string(caracter), 64),
	}
}

func procedenciaSeleccionO6Prueba(
	base time.Time,
	respuesta string,
	evidencia string,
) ports.ProcedenciaIntegracionBolsa {
	return ports.ProcedenciaIntegracionBolsa{
		AutoridadRef: "autoridad:bolsa", RespuestaRef: respuesta,
		ContratoVersion: ports.VersionContratoIntegracionBolsa,
		Fuente:          referenciaSeleccionO6Prueba("fuente:bolsa:001", 'a'),
		Evidencia: ports.EvidenciaNominalIntegracionBolsa{
			EvidenciaRef: evidencia, ClaveVerificacionRef: claveRespuestaSeleccionO6Prueba,
			SelloHMAC: selloSeleccionO6Prueba(claveRespuestaSeleccionO6Prueba, 'b'),
			EmitidaEn: base.Add(2 * time.Minute), ValidaHasta: base.Add(8 * time.Minute),
			RetenerHasta: base.Add(24 * time.Hour),
		},
	}
}

func selloSeleccionO6Prueba(clave string, caracter byte) string {
	return "hmac-sha256:" + clave + ":" + strings.Repeat(string(caracter), 64)
}
