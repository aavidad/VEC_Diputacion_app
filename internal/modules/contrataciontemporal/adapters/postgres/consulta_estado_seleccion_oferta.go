package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var _ ports.LectorContextoExpedienteSeleccionOfertaCT = (*RegistroSeleccionOfertaPostgreSQL)(nil)
var _ ports.RegistroConsultaEstadoSeleccionOfertaCT = (*RegistroSeleccionOfertaPostgreSQL)(nil)

func (r *RegistroSeleccionOfertaPostgreSQL) LeerContextoExpedienteSeleccionOfertaCT(
	ctx context.Context, organizacion, expediente string,
) (ports.ContextoExpedienteSeleccionOfertaCT, error) {
	vacio := ports.ContextoExpedienteSeleccionOfertaCT{}
	if r == nil || r.pool == nil || ctx == nil ||
		!domain.ReferenciaOpacaValida(organizacion) || !domain.ReferenciaOpacaValida(expediente) {
		return vacio, ports.ErrSeleccionOfertaCTInvalida
	}
	operacion, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	tx, err := r.pool.BeginTx(operacion, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return vacio, errorRegistroSeleccionOferta(operacion, err)
	}
	defer tx.Rollback(context.Background())
	var version int64
	var categoria, centro, unidad string
	err = tx.QueryRow(operacion, `SELECT version_actual,categoria_ref,centro_ref,unidad_gestora_ref
		FROM vec_contratacion_temporal.preparar_contexto_estado_oferta_ct227($1,$2)`,
		organizacion, expediente).Scan(&version, &categoria, &centro, &unidad)
	if errors.Is(err, pgx.ErrNoRows) {
		return vacio, ports.ErrEstadoExpedienteNoSeleccionable
	}
	if err != nil {
		return vacio, errorRegistroSeleccionOferta(operacion, err)
	}
	if version < 1 || version > 1<<53-1 || !domain.ReferenciaOpacaValida(categoria) ||
		!domain.ReferenciaOpacaValida(centro) || !domain.ReferenciaOpacaValida(unidad) {
		return vacio, errRegistroSeleccionOfertaPostgreSQL
	}
	if err := tx.Commit(operacion); err != nil {
		return vacio, errorRegistroSeleccionOferta(operacion, err)
	}
	return ports.ContextoExpedienteSeleccionOfertaCT{VersionActual: uint64(version),
		CategoriaRef: categoria, CentroRef: centro, UnidadGestoraRef: unidad}, nil
}

func (r *RegistroSeleccionOfertaPostgreSQL) ConsultarEstadoPorExpediente(
	ctx context.Context, q ports.SolicitudAutorizacionConsultaEstadoSeleccionOfertaCT,
	a ports.AutorizacionSeleccionOfertaCT,
) (ports.EstadoSeleccionOfertaPorExpedienteCT, error) {
	vacio := ports.EstadoSeleccionOfertaPorExpedienteCT{}
	canon, err := ports.CanonConsultaEstadoSeleccionOfertaCT(q)
	if err != nil || a.ActorRef != q.ActorRef || a.PerfilRef != q.PerfilRef ||
		a.MaterialCanonico != canon || a.Material.ValidarEstructura() != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	args := append([]any{canon}, argumentosV3SeleccionOferta(a)...)
	raw, err := r.ejecutar(ctx, `SELECT vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(
		$1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,
		$8::bytea,$9::bytea,$10::bytea,$11::bytea)`, args...)
	if err != nil {
		return vacio, err
	}
	var salida struct {
		Situacion         string          `json:"situacion"`
		TipoSeleccion     string          `json:"tipo_seleccion"`
		IntencionOriginal json.RawMessage `json:"intencion_original"`
		Estado            json.RawMessage `json:"estado"`
	}
	if json.Unmarshal(raw, &salida) != nil {
		return vacio, errRegistroSeleccionOfertaPostgreSQL
	}
	if salida.Situacion == "ausente" {
		if salida.TipoSeleccion != "" || len(salida.IntencionOriginal) != 0 || len(salida.Estado) != 0 {
			return vacio, errRegistroSeleccionOfertaPostgreSQL
		}
		return vacio, nil
	}
	if salida.TipoSeleccion == "llamamiento" {
		return decodificarIntencionSingularPendiente(salida.Situacion,
			salida.IntencionOriginal, salida.Estado, q)
	}
	if salida.TipoSeleccion != "" {
		return vacio, errRegistroSeleccionOfertaPostgreSQL
	}
	if salida.Situacion != "propietaria" && salida.Situacion != "indeterminada" &&
		salida.Situacion != "confirmada" && salida.Situacion != "descartada" {
		return vacio, errRegistroSeleccionOfertaPostgreSQL
	}
	var original intencionSeleccionOfertaSQL
	if json.Unmarshal(salida.IntencionOriginal, &original) != nil ||
		original.OrganizacionRef != q.OrganizacionRef || original.ExpedienteRef != q.ExpedienteRef ||
		original.CategoriaRef != q.CategoriaRef || original.CentroRef != q.CentroRef ||
		original.ActorRef != q.ActorRef || original.PerfilRef != q.PerfilRef ||
		original.VersionExpediente > q.VersionObservada {
		return vacio, errRegistroSeleccionOfertaPostgreSQL
	}
	i := ports.IntencionSeleccionOfertaCT{OrganizacionRef: original.OrganizacionRef,
		ExpedienteRef: original.ExpedienteRef, CategoriaRef: original.CategoriaRef,
		CentroRef: original.CentroRef, VersionExpediente: original.VersionExpediente,
		UnidadGestoraRef:  q.UnidadGestoraRef,
		ClaveIdempotencia: original.ClaveIdempotencia, ActorRef: original.ActorRef,
		PerfilRef: original.PerfilRef, HuellaIntencion: original.HuellaIntencion,
		Vinculo: q.Vinculo, ResultadoContexto: q.ResultadoContexto,
		Correlacion: q.Correlacion, MotivoAutorizacion: q.MotivoAutorizacion}
	if i.Validar() != nil {
		return vacio, errRegistroSeleccionOfertaPostgreSQL
	}
	estado, err := decodificarEstadoSeleccionOferta(salida.Estado, i)
	if err != nil {
		return vacio, err
	}
	if salida.Situacion == "propietaria" && estado.Situacion != ports.SeleccionOfertaReservada ||
		salida.Situacion == "indeterminada" && estado.Situacion != ports.SeleccionOfertaIndeterminada ||
		salida.Situacion == "confirmada" && estado.Situacion != ports.SeleccionOfertaConfirmada ||
		salida.Situacion == "descartada" && estado.Situacion != ports.SeleccionOfertaDescartada {
		return vacio, errRegistroSeleccionOfertaPostgreSQL
	}
	return ports.EstadoSeleccionOfertaPorExpedienteCT{Encontrada: true, Intencion: i, Estado: estado}, nil
}

func decodificarIntencionSingularPendiente(situacion string, original, estadoRaw []byte,
	q ports.SolicitudAutorizacionConsultaEstadoSeleccionOfertaCT,
) (ports.EstadoSeleccionOfertaPorExpedienteCT, error) {
	vacio := ports.EstadoSeleccionOfertaPorExpedienteCT{}
	if situacion != "propietaria" && situacion != "indeterminada" {
		return vacio, errRegistroSeleccionOfertaPostgreSQL
	}
	var datos struct {
		OrganizacionRef   string `json:"organizacion_ref"`
		ExpedienteRef     string `json:"expediente_ref"`
		VersionExpediente uint64 `json:"version_expediente"`
		Clave             string `json:"clave_idempotencia"`
	}
	var estado struct {
		Efecto         string `json:"efecto"`
		VentanaAbierta bool   `json:"ventana_abierta"`
	}
	if json.Unmarshal(original, &datos) != nil || json.Unmarshal(estadoRaw, &estado) != nil {
		return vacio, errRegistroSeleccionOfertaPostgreSQL
	}
	i := ports.IntencionSingularPendienteCT{OrganizacionRef: datos.OrganizacionRef,
		ExpedienteRef: datos.ExpedienteRef, VersionExpediente: datos.VersionExpediente,
		ClaveIdempotencia: datos.Clave, Efecto: ports.EfectoSeleccionLlamamiento(estado.Efecto),
		VentanaAbierta: estado.VentanaAbierta}
	if i.Validar() != nil || i.OrganizacionRef != q.OrganizacionRef ||
		i.ExpedienteRef != q.ExpedienteRef || i.VersionExpediente != q.VersionObservada ||
		(situacion == "indeterminada" && !i.VentanaAbierta) {
		return vacio, errRegistroSeleccionOfertaPostgreSQL
	}
	return ports.EstadoSeleccionOfertaPorExpedienteCT{Encontrada: true, Singular: &i}, nil
}
