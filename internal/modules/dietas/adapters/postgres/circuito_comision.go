package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/dietas/application"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

const decidirComisionSQL = `SELECT vec_dietas.decidir_comision_v2($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
const consultarDocumentoCircuitoSQL = `SELECT vec_dietas.consultar_documento_circuito_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
const listarBandejaComisionesSQL = `SELECT vec_dietas.listar_bandeja_comisiones_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`

var cursorBandeja = regexp.MustCompile(`^dco_[A-Za-z0-9_-]{22,128}$`)
var _ dietasports.RepositorioCircuitoComision = (*RepositorioBorradorComisionPostgreSQL)(nil)

func (r *RepositorioBorradorComisionPostgreSQL) Decidir(ctx context.Context, identidad dietasports.IdentidadEfectivaCircuito, solicitud dietasports.SolicitudDecisionCircuito) (dietasports.ResultadoCircuitoComision, error) {
	var cero dietasports.ResultadoCircuitoComision
	op := dietasports.SolicitudOperacionCircuito{Operacion: dietasports.OperacionDecidirCircuito, Decision: solicitud}
	bruto, err := r.ejecutarCircuito(ctx, identidad, op, decidirComisionSQL, true)
	if err != nil {
		return cero, err
	}
	var x struct {
		Comision dietasports.VistaComisionCircuito `json:"comision"`
		Recibo   reciboJSON                        `json:"recibo"`
	}
	if json.Unmarshal(bruto, &x) != nil || !cursorBandeja.MatchString(x.Comision.Referencia) || x.Comision.Referencia != solicitud.Referencia ||
		x.Comision.Version != x.Recibo.Version || x.Comision.Version != solicitud.VersionEsperada+1 ||
		!referenciaReciboBorrador.MatchString(x.Recibo.Referencia) {
		return cero, dietasports.ErrResultadoCircuitoIncierto
	}
	esperado := solicitud.Etapa.EstadoTrasAprobar()
	if solicitud.Decision == domain.DecisionDevolver {
		esperado = domain.EstadoDevuelta
	}
	if x.Comision.Estado != esperado {
		return cero, dietasports.ErrResultadoCircuitoIncierto
	}
	en, err := instanteReciboCircuito(x.Recibo.RegistradoEn)
	if err != nil {
		return cero, dietasports.ErrResultadoCircuitoIncierto
	}
	return dietasports.ResultadoCircuitoComision{Comision: x.Comision, Recibo: dietasports.ReciboBorradorComision{Referencia: x.Recibo.Referencia, Version: x.Recibo.Version, RegistradoEn: en, Repeticion: x.Recibo.Repeticion}}, nil
}

func (r *RepositorioBorradorComisionPostgreSQL) ListarPendientes(ctx context.Context, identidad dietasports.IdentidadEfectivaCircuito, consulta dietasports.ConsultaBandejaCircuito) (dietasports.PaginaBandejaCircuito, error) {
	var cero dietasports.PaginaBandejaCircuito
	if consulta.Limite == 0 {
		consulta.Limite = 20
	}
	op := dietasports.SolicitudOperacionCircuito{Operacion: dietasports.OperacionListarBandeja, Consulta: consulta}
	bruto, err := r.ejecutarCircuito(ctx, identidad, op, listarBandejaComisionesSQL, false)
	if err != nil {
		return cero, err
	}
	var forma map[string]json.RawMessage
	if json.Unmarshal(bruto, &forma) != nil || len(forma["items"]) == 0 || forma["items"][0] != '[' || len(forma["siguiente_cursor"]) == 0 || forma["siguiente_cursor"][0] != '"' {
		return cero, dietasports.ErrCircuitoNoDisponible
	}
	var pagina dietasports.PaginaBandejaCircuito
	if json.Unmarshal(bruto, &pagina) != nil || len(pagina.Items) > consulta.Limite || (pagina.SiguienteCursor != "" && !cursorBandeja.MatchString(pagina.SiguienteCursor)) {
		return cero, dietasports.ErrCircuitoNoDisponible
	}
	for _, item := range pagina.Items {
		if !cursorBandeja.MatchString(item.Referencia) || item.Estado != consulta.Etapa.EstadoPendiente() || item.Version == 0 ||
			!domain.FechaCircuitoValida(item.FechaInicio) || !domain.FechaCircuitoValida(item.FechaFin) || item.FechaInicio > item.FechaFin {
			return cero, dietasports.ErrCircuitoNoDisponible
		}
	}
	if pagina.Items == nil {
		pagina.Items = []dietasports.VistaComisionCircuito{}
	}
	return pagina, nil
}

// ConsultarDocumento lee el documento pendiente en la etapa del revisor. La
// función nominal consume AD3-80 y repite competencia y separación; una
// respuesta "no_encontrado" no distingue ausencia de falta de competencia.
func (r *RepositorioBorradorComisionPostgreSQL) ConsultarDocumento(ctx context.Context, identidad dietasports.IdentidadEfectivaCircuito, solicitud dietasports.SolicitudDocumentoCircuito) (dietasports.DocumentoCircuito, error) {
	var cero dietasports.DocumentoCircuito
	op := dietasports.SolicitudOperacionCircuito{Operacion: dietasports.OperacionConsultarDocumentoCircuito, Documento: solicitud}
	bruto, err := r.ejecutarCircuito(ctx, identidad, op, consultarDocumentoCircuitoSQL, false)
	if err != nil {
		return cero, err
	}
	return decodificarDocumentoCircuito(bruto, solicitud)
}

// decodificarDocumentoCircuito traduce la salida de
// consultar_documento_circuito_v1, incluida la devolución anterior de un
// reenvío (000010), y la valida con la forma de la aplicación.
func decodificarDocumentoCircuito(bruto []byte, solicitud dietasports.SolicitudDocumentoCircuito) (dietasports.DocumentoCircuito, error) {
	var cero dietasports.DocumentoCircuito
	var x struct {
		Resultado string                         `json:"resultado"`
		Comision  *dietasports.DocumentoCircuito `json:"comision"`
	}
	if json.Unmarshal(bruto, &x) != nil {
		return cero, dietasports.ErrCircuitoNoDisponible
	}
	if x.Resultado == "no_encontrado" && x.Comision == nil {
		return cero, dietasports.ErrComisionNoEncontrada
	}
	if x.Resultado != "concedido" || x.Comision == nil || application.ValidarDocumentoCircuito(*x.Comision, solicitud) != nil {
		return cero, dietasports.ErrCircuitoNoDisponible
	}
	return *x.Comision, nil
}

func (r *RepositorioBorradorComisionPostgreSQL) ejecutarCircuito(ctx context.Context, identidad dietasports.IdentidadEfectivaCircuito, op dietasports.SolicitudOperacionCircuito, funcion string, escritura bool) ([]byte, error) {
	if err := r.valido(ctx); err != nil {
		if ctx != nil && ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, dietasports.ErrCircuitoNoDisponible
	}
	accion, recurso, finalidad := application.ContratoCircuito(op)
	a := identidad.Autorizacion
	if accion == "" || identidad.ContextoRegistrado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.ContextoRegistrado) != nil ||
		a.Accion != accion || a.RecursoRef != recurso || a.Finalidad != finalidad || a.Material.ValidarEstructura() != nil {
		return nil, dietasports.ErrAccesoCircuitoDenegado
	}
	efecto, err := application.ConstruirEfectoAutorizacionCircuito(identidad.ContextoRegistrado, identidad.UnidadCompetenciaRef, op)
	if err != nil || efecto.Recurso.Referencia != recurso || len(efecto.Material) == 0 {
		return nil, dietasports.ErrAccesoCircuitoDenegado
	}
	argumentos, err := argumentosAD3(a.Material)
	if err != nil {
		return nil, dietasports.ErrAccesoCircuitoDenegado
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		argumentos.limpiar()
		return nil, normalizarErrorCircuito(ctx, err, escritura)
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	var bruto []byte
	err = tx.QueryRow(ctx, funcion, string(efecto.Material), argumentos.capacidad, argumentos.decision, argumentos.motivo, argumentos.contexto, argumentos.personaVersion, argumentos.perfilVersion, argumentos.payload, argumentos.sobre, argumentos.evidencia, argumentos.raiz).Scan(&bruto)
	argumentos.limpiar()
	if err != nil {
		return nil, normalizarErrorCircuito(ctx, err, escritura)
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, normalizarErrorCircuito(ctx, err, escritura)
	}
	return bruto, nil
}

func normalizarErrorCircuito(ctx context.Context, err error, escritura bool) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "PD002":
			return dietasports.ErrConflictoIdempotencia
		case "PD003", "42501":
			return dietasports.ErrAccesoCircuitoDenegado
		case "PD004":
			return dietasports.ErrComisionNoEncontrada
		case "PD005":
			return dietasports.ErrEstadoCircuitoConflicto
		case "P7201":
			return dietasports.ErrRelacionNoDisponible
		}
	}
	if escritura {
		return dietasports.ErrResultadoCircuitoIncierto
	}
	return dietasports.ErrCircuitoNoDisponible
}

func instanteReciboCircuito(valor string) (time.Time, error) {
	en, err := time.Parse(time.RFC3339Nano, valor)
	_, desfase := en.Zone()
	if err != nil || desfase != 0 || en.Nanosecond()%1000 != 0 {
		return time.Time{}, errors.New("instante de circuito inválido")
	}
	return en.UTC(), nil
}
