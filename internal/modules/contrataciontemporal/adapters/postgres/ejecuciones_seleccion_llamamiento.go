package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	funcionResolverTerminalSeleccionO6 = "vec_contratacion_temporal.resolver_terminal_seleccion_llamamiento_o6_v1"
	funcionReservarSeleccionO6         = "vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1"
	funcionAbrirVentanaSeleccionO6     = "vec_contratacion_temporal.abrir_ventana_seleccion_llamamiento_o6_v1"
	funcionMarcarIndeterminadaO6       = "vec_contratacion_temporal.marcar_indeterminada_seleccion_llamamiento_o6_v1"
	funcionLiberarSeleccionO6          = "vec_contratacion_temporal.liberar_seleccion_llamamiento_o6_v1"
	funcionConfirmarSeleccionO6        = "vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1"
	funcionConsultarSeleccionO6        = "vec_contratacion_temporal.consultar_seleccion_llamamiento_o6_v1"

	prefijoReservaSeleccionO6 = "reserva:seleccion-llamamiento:"
	maximoCargaSeleccionO6    = 1024 * 1024
)

var errEjecucionesSeleccionLlamamientoPostgreSQL = errors.New(
	"contratacion temporal: ejecuciones de seleccion y llamamiento no disponibles",
)

var _ ports.EjecucionesSeleccionLlamamiento = (*EjecucionesSeleccionLlamamientoPostgreSQL)(nil)

// EjecucionesSeleccionLlamamientoPostgreSQL solo invoca fachadas nominales.
// Cada método abre una única transacción y nunca reintenta una mutación: un
// error de COMMIT queda cerrado hasta que una lectura posterior lo resuelva.
type EjecucionesSeleccionLlamamientoPostgreSQL struct {
	pool iniciadorTransacciones
}

func NuevasEjecucionesSeleccionLlamamientoPostgreSQL(
	pool *pgxpool.Pool,
) (*EjecucionesSeleccionLlamamientoPostgreSQL, error) {
	return nuevasEjecucionesSeleccionLlamamientoPostgreSQL(pool)
}

func nuevasEjecucionesSeleccionLlamamientoPostgreSQL(
	pool iniciadorTransacciones,
) (*EjecucionesSeleccionLlamamientoPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	return &EjecucionesSeleccionLlamamientoPostgreSQL{pool: pool}, nil
}

type solicitudEjecucionSeleccionO6 struct {
	ClaveIdempotencia  string                                     `json:"clave_idempotencia"`
	HuellaSemantica    string                                     `json:"huella_semantica"`
	OrganizacionRef    string                                     `json:"organizacion_ref"`
	ExpedienteRef      string                                     `json:"expediente_ref"`
	VersionExpediente  uint64                                     `json:"version_expediente"`
	CorrelacionRef     string                                     `json:"correlacion_ref"`
	AccionOrden        ports.ReferenciaVersionadaIntegracionBolsa `json:"accion_orden"`
	Finalidad          ports.ReferenciaVersionadaIntegracionBolsa `json:"finalidad"`
	Necesidad          ports.ReferenciaVersionadaIntegracionBolsa `json:"necesidad"`
	Bolsa              ports.ReferenciaVersionadaIntegracionBolsa `json:"bolsa"`
	Politica           ports.ReferenciaVersionadaIntegracionBolsa `json:"politica"`
	MaximoPosiciones   uint32                                     `json:"maximo_posiciones"`
	CantidadDisponible uint32                                     `json:"cantidad_disponible"`
}

type filaEjecucionSeleccionO6 struct {
	Situacion, SolicitudJSON, ReservaRef, Efecto, ReciboJSON, ArtefactoJSON string
}

func (e *EjecucionesSeleccionLlamamientoPostgreSQL) ResolverTerminal(
	ctx context.Context,
	clave string,
) (ports.EstadoEjecucionSeleccionLlamamiento, bool, error) {
	if !e.valido(ctx) || !ports.ClaveIdempotenciaValida(clave) {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, false,
			errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	fila, err := e.consultarFila(ctx, pgx.ReadOnly, `
		SELECT situacion, solicitud_json::text, reserva_ref, efecto,
		       recibo_json::text, artefacto_json::text
		  FROM `+funcionResolverTerminalSeleccionO6+`($1::uuid)`, clave)
	if err != nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, false, err
	}
	estado, err := estadoDesdeFilaSeleccionO6(fila)
	if err != nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, false, err
	}
	if estado.Situacion != "" && estado.Solicitud.ClaveIdempotencia != clave {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, false,
			errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	return estado, estado.Situacion == ports.EjecucionSeleccionLlamamientoConfirmada, nil
}

func (e *EjecucionesSeleccionLlamamientoPostgreSQL) Reservar(
	ctx context.Context,
	solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento,
) (ports.EstadoEjecucionSeleccionLlamamiento, error) {
	contenido, err := codificarSolicitudSeleccionO6(solicitud)
	if !e.valido(ctx) || err != nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{},
			errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	fila, err := e.consultarFila(ctx, pgx.ReadWrite, `
		SELECT situacion, solicitud_json::text, reserva_ref, efecto,
		       recibo_json::text, artefacto_json::text
		  FROM `+funcionReservarSeleccionO6+`($1::uuid, $2::text, $3::jsonb)`,
		solicitud.ClaveIdempotencia, solicitud.HuellaSemantica, contenido)
	if err != nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, err
	}
	estado, err := estadoDesdeFilaSeleccionO6(fila)
	if err != nil || estado.Situacion == "" || estado.Solicitud != solicitud {
		return ports.EstadoEjecucionSeleccionLlamamiento{},
			errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	return estado, nil
}

func (e *EjecucionesSeleccionLlamamientoPostgreSQL) AbrirVentanaEfecto(
	ctx context.Context,
	reserva ports.ReservaEjecucionSeleccionLlamamiento,
	efecto ports.EfectoSeleccionLlamamiento,
) error {
	if !efectoSeleccionO6Valido(efecto) {
		return errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	return e.ejecutarReserva(ctx, funcionAbrirVentanaSeleccionO6, reserva, string(efecto))
}

func (e *EjecucionesSeleccionLlamamientoPostgreSQL) MarcarIndeterminada(
	ctx context.Context,
	reserva ports.ReservaEjecucionSeleccionLlamamiento,
	efecto ports.EfectoSeleccionLlamamiento,
) error {
	if !efectoSeleccionO6Valido(efecto) {
		return errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	return e.ejecutarReserva(ctx, funcionMarcarIndeterminadaO6, reserva, string(efecto))
}

func (e *EjecucionesSeleccionLlamamientoPostgreSQL) LiberarAntesDeEfectos(
	ctx context.Context,
	reserva ports.ReservaEjecucionSeleccionLlamamiento,
) error {
	return e.ejecutarReserva(ctx, funcionLiberarSeleccionO6, reserva)
}

func (e *EjecucionesSeleccionLlamamientoPostgreSQL) Confirmar(
	ctx context.Context,
	reserva ports.ReservaEjecucionSeleccionLlamamiento,
	recibo ports.ReciboSolicitudLlamamientoBolsa,
	artefacto ports.ArtefactoProbatorioLlamamientoBolsa,
) error {
	if !e.valido(ctx) || !reservaSeleccionO6Valida(reserva) ||
		!confirmacionSeleccionO6DentroDeLimite(recibo, artefacto) {
		return errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	solicitudJSON, errSolicitud := codificarSolicitudSeleccionO6(reserva.Solicitud)
	reciboJSON, errRecibo := json.Marshal(recibo)
	artefactoJSON, errArtefacto := json.Marshal(artefacto)
	if errSolicitud != nil || errRecibo != nil || errArtefacto != nil ||
		!confirmacionSeleccionO6Ligada(reserva, recibo, artefacto) ||
		len(reciboJSON) == 0 || len(artefactoJSON) == 0 ||
		len(reciboJSON)+len(artefactoJSON) > maximoCargaSeleccionO6 {
		return errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	return e.ejecutar(ctx, `
		SELECT `+funcionConfirmarSeleccionO6+`(
			$1::uuid, $2::text, $3::text, $4::jsonb, $5::jsonb, $6::text
		)`, reserva.Solicitud.ClaveIdempotencia, reserva.Solicitud.HuellaSemantica,
		reserva.ReservaRef, solicitudJSON, reciboJSON, string(artefactoJSON))
}

func (e *EjecucionesSeleccionLlamamientoPostgreSQL) ConsultarEstado(
	ctx context.Context,
	solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento,
) (ports.EstadoEjecucionSeleccionLlamamiento, error) {
	contenido, err := codificarSolicitudSeleccionO6(solicitud)
	if !e.valido(ctx) || err != nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{},
			errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	fila, err := e.consultarFila(ctx, pgx.ReadOnly, `
		SELECT situacion, solicitud_json::text, reserva_ref, efecto,
		       recibo_json::text, artefacto_json::text
		  FROM `+funcionConsultarSeleccionO6+`($1::uuid, $2::text, $3::jsonb)`,
		solicitud.ClaveIdempotencia, solicitud.HuellaSemantica, contenido)
	if err != nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, err
	}
	estado, err := estadoDesdeFilaSeleccionO6(fila)
	if err != nil || estado.Situacion == "" || estado.Solicitud != solicitud {
		return ports.EstadoEjecucionSeleccionLlamamiento{},
			errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	return estado, nil
}

func (e *EjecucionesSeleccionLlamamientoPostgreSQL) ejecutarReserva(
	ctx context.Context,
	funcion string,
	reserva ports.ReservaEjecucionSeleccionLlamamiento,
	extra ...any,
) error {
	contenido, err := codificarSolicitudSeleccionO6(reserva.Solicitud)
	if !e.valido(ctx) || err != nil || !reservaSeleccionO6Valida(reserva) {
		return errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	argumentos := []any{reserva.Solicitud.ClaveIdempotencia,
		reserva.Solicitud.HuellaSemantica, reserva.ReservaRef, contenido}
	argumentos = append(argumentos, extra...)
	marcadores := "$1::uuid, $2::text, $3::text, $4::jsonb"
	if len(extra) == 1 {
		marcadores += ", $5::text"
	}
	return e.ejecutar(ctx, "SELECT "+funcion+"("+marcadores+")", argumentos...)
}

func (e *EjecucionesSeleccionLlamamientoPostgreSQL) consultarFila(
	ctx context.Context,
	modo pgx.TxAccessMode,
	consulta string,
	argumentos ...any,
) (filaEjecucionSeleccionO6, error) {
	tx, err := e.iniciar(ctx, modo)
	if err != nil {
		return filaEjecucionSeleccionO6{}, errorEjecucionesSeleccionO6(ctx)
	}
	defer revertirTransaccion(tx)
	var fila filaEjecucionSeleccionO6
	err = tx.QueryRow(ctx, consulta, argumentos...).Scan(
		&fila.Situacion, &fila.SolicitudJSON, &fila.ReservaRef,
		&fila.Efecto, &fila.ReciboJSON, &fila.ArtefactoJSON,
	)
	if err != nil || tx.Commit(ctx) != nil {
		return filaEjecucionSeleccionO6{}, errorEjecucionesSeleccionO6(ctx)
	}
	return fila, nil
}

func (e *EjecucionesSeleccionLlamamientoPostgreSQL) ejecutar(
	ctx context.Context,
	consulta string,
	argumentos ...any,
) error {
	tx, err := e.iniciar(ctx, pgx.ReadWrite)
	if err != nil {
		return errorEjecucionesSeleccionO6(ctx)
	}
	defer revertirTransaccion(tx)
	var aplicada bool
	if err = tx.QueryRow(ctx, consulta, argumentos...).Scan(&aplicada); err != nil || !aplicada ||
		tx.Commit(ctx) != nil {
		return errorEjecucionesSeleccionO6(ctx)
	}
	return nil
}

func (e *EjecucionesSeleccionLlamamientoPostgreSQL) iniciar(
	ctx context.Context,
	modo pgx.TxAccessMode,
) (pgx.Tx, error) {
	tx, err := e.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: modo,
	})
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config('idle_in_transaction_session_timeout', '20s', true)`)
	if err != nil {
		revertirTransaccion(tx)
		return nil, err
	}
	return tx, nil
}

func (e *EjecucionesSeleccionLlamamientoPostgreSQL) valido(ctx context.Context) bool {
	return e != nil && !dependenciaNula(e.pool) && ctx != nil && !dependenciaNula(ctx)
}

func codificarSolicitudSeleccionO6(
	solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento,
) ([]byte, error) {
	dto := solicitudEjecucionSeleccionO6{
		ClaveIdempotencia: solicitud.ClaveIdempotencia, HuellaSemantica: solicitud.HuellaSemantica,
		OrganizacionRef: solicitud.OrganizacionRef, ExpedienteRef: solicitud.ExpedienteRef,
		VersionExpediente: solicitud.VersionExpediente, CorrelacionRef: solicitud.CorrelacionRef,
		AccionOrden: solicitud.AccionOrden, Finalidad: solicitud.Finalidad,
		Necesidad: solicitud.Necesidad, Bolsa: solicitud.Bolsa, Politica: solicitud.Politica,
		MaximoPosiciones: solicitud.MaximoPosiciones, CantidadDisponible: solicitud.CantidadDisponible,
	}
	if !dto.valida() {
		return nil, errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	contenido, err := json.Marshal(dto)
	if err != nil || len(contenido) == 0 || len(contenido) > maximoCargaSeleccionO6 {
		return nil, errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	return contenido, nil
}

func (s solicitudEjecucionSeleccionO6) valida() bool {
	return s.puertos().Validar() == nil
}

func (s solicitudEjecucionSeleccionO6) puertos() ports.SolicitudReservaEjecucionSeleccionLlamamiento {
	return ports.SolicitudReservaEjecucionSeleccionLlamamiento{
		ClaveIdempotencia: s.ClaveIdempotencia, HuellaSemantica: s.HuellaSemantica,
		OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		VersionExpediente: s.VersionExpediente, CorrelacionRef: s.CorrelacionRef,
		AccionOrden: s.AccionOrden, Finalidad: s.Finalidad, Necesidad: s.Necesidad,
		Bolsa: s.Bolsa, Politica: s.Politica, MaximoPosiciones: s.MaximoPosiciones,
		CantidadDisponible: s.CantidadDisponible,
	}
}

func estadoDesdeFilaSeleccionO6(
	fila filaEjecucionSeleccionO6,
) (ports.EstadoEjecucionSeleccionLlamamiento, error) {
	if fila.Situacion == "" {
		if fila.SolicitudJSON == "" && fila.ReservaRef == "" && fila.Efecto == "" &&
			fila.ReciboJSON == "" && fila.ArtefactoJSON == "" {
			return ports.EstadoEjecucionSeleccionLlamamiento{}, nil
		}
		return ports.EstadoEjecucionSeleccionLlamamiento{}, errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	var dto solicitudEjecucionSeleccionO6
	if decodificarJSONEstricto([]byte(fila.SolicitudJSON), &dto) != nil || !dto.valida() {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	estado := ports.EstadoEjecucionSeleccionLlamamiento{
		Solicitud: dto.puertos(), Situacion: ports.SituacionEjecucionSeleccionLlamamiento(fila.Situacion),
		EfectoPosible: ports.EfectoSeleccionLlamamiento(fila.Efecto),
	}
	switch estado.Situacion {
	case ports.EjecucionSeleccionLlamamientoPropietaria:
		if !referenciaReservaSeleccionO6Valida(fila.ReservaRef) || fila.Efecto != "" ||
			fila.ReciboJSON != "" || fila.ArtefactoJSON != "" {
			return ports.EstadoEjecucionSeleccionLlamamiento{}, errEjecucionesSeleccionLlamamientoPostgreSQL
		}
		estado.ReservaRef = fila.ReservaRef
	case ports.EjecucionSeleccionLlamamientoOcupada:
		if fila.ReservaRef != "" || !efectoSeleccionO6ValidoOVacio(estado.EfectoPosible) ||
			fila.ReciboJSON != "" || fila.ArtefactoJSON != "" {
			return ports.EstadoEjecucionSeleccionLlamamiento{}, errEjecucionesSeleccionLlamamientoPostgreSQL
		}
	case ports.EjecucionSeleccionLlamamientoColision:
		if fila.ReservaRef != "" || fila.Efecto != "" || fila.ReciboJSON != "" || fila.ArtefactoJSON != "" {
			return ports.EstadoEjecucionSeleccionLlamamiento{}, errEjecucionesSeleccionLlamamientoPostgreSQL
		}
	case ports.EjecucionSeleccionLlamamientoIndeterminada:
		if fila.ReservaRef != "" || !efectoSeleccionO6Valido(estado.EfectoPosible) ||
			fila.ReciboJSON != "" || fila.ArtefactoJSON != "" {
			return ports.EstadoEjecucionSeleccionLlamamiento{}, errEjecucionesSeleccionLlamamientoPostgreSQL
		}
	case ports.EjecucionSeleccionLlamamientoConfirmada:
		if fila.ReservaRef != "" || fila.Efecto != "" || fila.ReciboJSON == "" || fila.ArtefactoJSON == "" {
			return ports.EstadoEjecucionSeleccionLlamamiento{}, errEjecucionesSeleccionLlamamientoPostgreSQL
		}
		artefacto, err := ports.DecodificarArtefactoProbatorioLlamamientoBolsa([]byte(fila.ArtefactoJSON))
		var recibo ports.ReciboSolicitudLlamamientoBolsa
		if err != nil || decodificarJSONEstricto([]byte(fila.ReciboJSON), &recibo) != nil ||
			recibo != artefacto.Recibo || !confirmacionSeleccionO6Ligada(
			ports.ReservaEjecucionSeleccionLlamamiento{Solicitud: estado.Solicitud,
				ReservaRef: prefijoReservaSeleccionO6 + estado.Solicitud.ClaveIdempotencia},
			recibo, artefacto,
		) {
			return ports.EstadoEjecucionSeleccionLlamamiento{}, errEjecucionesSeleccionLlamamientoPostgreSQL
		}
		estado.ReciboConfirmado, estado.ArtefactoConfirmado = recibo, artefacto
	default:
		return ports.EstadoEjecucionSeleccionLlamamiento{}, errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	return estado, nil
}

func confirmacionSeleccionO6Ligada(
	reserva ports.ReservaEjecucionSeleccionLlamamiento,
	recibo ports.ReciboSolicitudLlamamientoBolsa,
	artefacto ports.ArtefactoProbatorioLlamamientoBolsa,
) bool {
	s := reserva.Solicitud
	d := artefacto.Comando.Contexto.Datos
	return artefacto.Validar() == nil && recibo == artefacto.Recibo && recibo.PropuestaGenerada &&
		d.OrganizacionRef == s.OrganizacionRef && d.ExpedienteRef == s.ExpedienteRef &&
		d.VersionExpediente == s.VersionExpediente && d.CorrelacionRef == s.CorrelacionRef &&
		d.Finalidad == s.Finalidad && artefacto.Comando.Necesidad == s.Necesidad &&
		artefacto.Comando.Bolsa == s.Bolsa && artefacto.Comando.Politica == s.Politica &&
		artefacto.Comando.TotalPosicionesOrden >= s.CantidadDisponible &&
		artefacto.Comando.TotalPosicionesOrden <= s.MaximoPosiciones &&
		artefacto.Comando.MaximaPosicionEvaluable == artefacto.Comando.TotalPosicionesOrden
}

var tipoInstanteSeleccionO6 = reflect.TypeFor[time.Time]()

// confirmacionSeleccionO6DentroDeLimite recorre solo la estructura estatica
// antes de que json.Marshal o Validar recodifiquen el artefacto.
func confirmacionSeleccionO6DentroDeLimite(
	recibo ports.ReciboSolicitudLlamamientoBolsa,
	artefacto ports.ArtefactoProbatorioLlamamientoBolsa,
) bool {
	restante := maximoCargaSeleccionO6
	return descontarTextoSeleccionO6(reflect.ValueOf(recibo), &restante) &&
		descontarTextoSeleccionO6(reflect.ValueOf(artefacto), &restante)
}

func descontarTextoSeleccionO6(valor reflect.Value, restante *int) bool {
	if !valor.IsValid() || restante == nil || *restante < 0 {
		return false
	}
	if valor.Type() == tipoInstanteSeleccionO6 {
		return true
	}
	switch valor.Kind() {
	case reflect.String:
		// Un byte de entrada puede ocupar seis bytes como escape JSON.
		if valor.Len() > *restante/6 {
			return false
		}
		*restante -= valor.Len() * 6
		return true
	case reflect.Struct:
		for indice := 0; indice < valor.NumField(); indice++ {
			if !descontarTextoSeleccionO6(valor.Field(indice), restante) {
				return false
			}
		}
		return true
	case reflect.Bool, reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func reservaSeleccionO6Valida(reserva ports.ReservaEjecucionSeleccionLlamamiento) bool {
	return referenciaReservaSeleccionO6Valida(reserva.ReservaRef)
}

func referenciaReservaSeleccionO6Valida(valor string) bool {
	return strings.HasPrefix(valor, prefijoReservaSeleccionO6) &&
		ports.ClaveIdempotenciaValida(strings.TrimPrefix(valor, prefijoReservaSeleccionO6))
}

func efectoSeleccionO6Valido(efecto ports.EfectoSeleccionLlamamiento) bool {
	return efecto == ports.EfectoPrepararOrdenSeleccionLlamamiento ||
		efecto == ports.EfectoSolicitarSeleccionLlamamiento
}

func efectoSeleccionO6ValidoOVacio(efecto ports.EfectoSeleccionLlamamiento) bool {
	return efecto == "" || efectoSeleccionO6Valido(efecto)
}

func errorEjecucionesSeleccionO6(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return errEjecucionesSeleccionLlamamientoPostgreSQL
}
