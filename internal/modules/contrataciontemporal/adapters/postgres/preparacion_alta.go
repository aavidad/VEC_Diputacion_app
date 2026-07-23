// Package postgres implementa los contratos durables del módulo mediante
// funciones SECURITY DEFINER de lista positiva. La cuenta de ejecución no
// recibe acceso directo a las tablas.
package postgres

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	funcionPrepararAltaV2 = "vec_contratacion_temporal.preparar_alta_v2"
	esquemaPrepararAltaV2 = "vec.contratacion-temporal.preparar-alta.v2"
)

var _ ports.PreparadorAltaIdempotente = (*PreparadorAltaPostgreSQL)(nil)

type iniciadorTransacciones interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type PreparadorAltaPostgreSQL struct {
	pool      iniciadorTransacciones
	sellador  ports.SelladorAmbitoIdempotencia
	generador ports.GeneradorReferenciasAlta
}

func NuevoPreparadorAltaPostgreSQL(
	pool *pgxpool.Pool,
	sellador ports.SelladorAmbitoIdempotencia,
	generador ports.GeneradorReferenciasAlta,
) (*PreparadorAltaPostgreSQL, error) {
	return nuevoPreparadorAltaPostgreSQL(pool, sellador, generador)
}

func nuevoPreparadorAltaPostgreSQL(
	pool iniciadorTransacciones,
	sellador ports.SelladorAmbitoIdempotencia,
	generador ports.GeneradorReferenciasAlta,
) (*PreparadorAltaPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(sellador) || dependenciaNula(generador) {
		return nil, ports.ErrPersistenciaNoDisponible
	}
	return &PreparadorAltaPostgreSQL{
		pool: pool, sellador: sellador, generador: generador,
	}, nil
}

type operacionPrepararAltaV2 struct {
	Esquema               string                    `json:"esquema"`
	SellosHMAC            sellosPrepararAltaV2      `json:"sellos_hmac"`
	OrganizacionRef       string                    `json:"organizacion_ref"`
	ActorRef              string                    `json:"actor_ref"`
	PerfilRef             string                    `json:"perfil_ref"`
	ReservaRefCandidata   string                    `json:"reserva_ref_candidata"`
	ReferenciasCandidatas referenciasPrepararAltaV1 `json:"referencias_candidatas"`
}

type sellosPrepararAltaV2 struct {
	Activo    parSellosPrepararAltaV2   `json:"activo"`
	Retenidos []parSellosPrepararAltaV2 `json:"retenidos"`
}

type parSellosPrepararAltaV2 struct {
	Generacion         uint32 `json:"generacion"`
	AmbitoHMAC         string `json:"ambito_hmac"`
	HuellaPeticionHMAC string `json:"huella_peticion_hmac"`
}

type referenciasPrepararAltaV1 struct {
	ExpedienteRef string `json:"expediente_ref"`
	NumeroVisible string `json:"numero_visible"`
	ReciboRef     string `json:"recibo_ref"`
}

func nuevasReferenciasPrepararAltaV1(
	referencias ports.ReferenciasAlta,
) referenciasPrepararAltaV1 {
	return referenciasPrepararAltaV1{
		ExpedienteRef: referencias.ExpedienteRef,
		NumeroVisible: referencias.NumeroVisible,
		ReciboRef:     referencias.ReciboRef,
	}
}

func (r referenciasPrepararAltaV1) puertos() ports.ReferenciasAlta {
	return ports.ReferenciasAlta{
		ExpedienteRef: r.ExpedienteRef,
		NumeroVisible: r.NumeroVisible,
		ReciboRef:     r.ReciboRef,
	}
}

func nuevosSellosPrepararAltaV2(
	ambitos ports.ColeccionSellosHMAC,
	huellas ports.ColeccionSellosHMAC,
) (sellosPrepararAltaV2, error) {
	datosAmbitos, err := ambitos.Datos()
	if err != nil {
		return sellosPrepararAltaV2{}, ports.ErrPreparacionAltaInvalida
	}
	datosHuellas, err := huellas.Datos()
	if err != nil ||
		datosAmbitos.Activo.Generacion != datosHuellas.Activo.Generacion ||
		len(datosAmbitos.Retenidos) != len(datosHuellas.Retenidos) {
		return sellosPrepararAltaV2{}, ports.ErrPreparacionAltaInvalida
	}
	sellos := sellosPrepararAltaV2{
		Activo: parSellosPrepararAltaV2{
			Generacion:         datosAmbitos.Activo.Generacion,
			AmbitoHMAC:         datosAmbitos.Activo.Valor,
			HuellaPeticionHMAC: datosHuellas.Activo.Valor,
		},
		Retenidos: make([]parSellosPrepararAltaV2, len(datosAmbitos.Retenidos)),
	}
	for indice := range datosAmbitos.Retenidos {
		if datosAmbitos.Retenidos[indice].Generacion !=
			datosHuellas.Retenidos[indice].Generacion {
			return sellosPrepararAltaV2{}, ports.ErrPreparacionAltaInvalida
		}
		sellos.Retenidos[indice] = parSellosPrepararAltaV2{
			Generacion:         datosAmbitos.Retenidos[indice].Generacion,
			AmbitoHMAC:         datosAmbitos.Retenidos[indice].Valor,
			HuellaPeticionHMAC: datosHuellas.Retenidos[indice].Valor,
		}
	}
	return sellos, nil
}

func (s sellosPrepararAltaV2) contieneAmbito(valor string) bool {
	if hmac.Equal([]byte(s.Activo.AmbitoHMAC), []byte(valor)) {
		return true
	}
	for _, retenido := range s.Retenidos {
		if hmac.Equal([]byte(retenido.AmbitoHMAC), []byte(valor)) {
			return true
		}
	}
	return false
}

func (s sellosPrepararAltaV2) contieneHuella(valor string) bool {
	if hmac.Equal([]byte(s.Activo.HuellaPeticionHMAC), []byte(valor)) {
		return true
	}
	for _, retenido := range s.Retenidos {
		if hmac.Equal([]byte(retenido.HuellaPeticionHMAC), []byte(valor)) {
			return true
		}
	}
	return false
}

func (p *PreparadorAltaPostgreSQL) PrepararAlta(
	ctx context.Context,
	solicitud ports.SolicitudPrepararAlta,
) (ports.PreparacionAlta, error) {
	if ctx == nil || p == nil || dependenciaNula(p.pool) ||
		dependenciaNula(p.sellador) || dependenciaNula(p.generador) ||
		solicitud.Validar() != nil {
		return ports.PreparacionAlta{}, ports.ErrPreparacionAltaInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.PreparacionAlta{}, err
	}
	sellar := ports.SolicitudSellarAmbitoIdempotencia{
		ClaveIdempotencia: solicitud.ClaveIdempotencia,
		OrganizacionRef:   solicitud.OrganizacionRef,
		ActorRef:          solicitud.ActorRef,
		PerfilRef:         solicitud.PerfilRef,
	}
	if sellar.Validar() != nil {
		return ports.PreparacionAlta{}, ports.ErrPreparacionAltaInvalida
	}
	ambitosHMAC, err := p.sellador.SellarAmbitoIdempotencia(ctx, sellar)
	if err != nil {
		return ports.PreparacionAlta{}, errorDependencia(ctx)
	}
	if err := ctx.Err(); err != nil {
		return ports.PreparacionAlta{}, err
	}
	if ambitosHMAC.ValidarDominio(
		"vec.contratacion-temporal.ambito-idempotencia",
	) != nil {
		return ports.PreparacionAlta{}, ports.ErrPersistenciaNoDisponible
	}
	sellos, err := nuevosSellosPrepararAltaV2(
		ambitosHMAC,
		solicitud.HuellasPeticionHMAC,
	)
	if err != nil {
		return ports.PreparacionAlta{}, err
	}
	referencias, err := p.generador.GenerarReferenciasAlta(ctx)
	if err != nil {
		return ports.PreparacionAlta{}, errorDependencia(ctx)
	}
	if referencias.Validar() != nil {
		return ports.PreparacionAlta{}, ports.ErrPersistenciaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.PreparacionAlta{}, err
	}
	reservaRef, err := p.generador.NuevaReferenciaReservaAlta(ctx)
	if err != nil {
		return ports.PreparacionAlta{}, errorDependencia(ctx)
	}
	if !domain.ReferenciaOpacaValida(reservaRef) {
		return ports.PreparacionAlta{}, ports.ErrPersistenciaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.PreparacionAlta{}, err
	}
	operacion, err := json.Marshal(operacionPrepararAltaV2{
		Esquema:               esquemaPrepararAltaV2,
		SellosHMAC:            sellos,
		OrganizacionRef:       solicitud.OrganizacionRef,
		ActorRef:              solicitud.ActorRef,
		PerfilRef:             solicitud.PerfilRef,
		ReservaRefCandidata:   reservaRef,
		ReferenciasCandidatas: nuevasReferenciasPrepararAltaV1(referencias),
	})
	if err != nil {
		return ports.PreparacionAlta{}, ports.ErrPersistenciaNoDisponible
	}
	defer borrarBytes(operacion)

	tx, err := p.iniciar(ctx)
	if err != nil {
		return ports.PreparacionAlta{}, err
	}
	defer revertirTransaccion(tx)
	fila := filaPreparacionAlta{}
	err = tx.QueryRow(ctx, `
		SELECT resultado, reserva_ref, expediente_ref, numero_visible,
		       recibo_ref, ambito_hmac, huella_peticion_hmac,
		       organizacion_ref, actor_ref, perfil_ref,
		       estado, version_expediente, auditoria_ref, evento_ref,
		       confirmada_en
		  FROM `+funcionPrepararAltaV2+`($1::jsonb)`,
		operacion,
	).Scan(
		&fila.resultado, &fila.reservaRef, &fila.expedienteRef,
		&fila.numeroVisible, &fila.reciboRef, &fila.ambitoHMAC,
		&fila.huellaPeticionHMAC, &fila.organizacionRef,
		&fila.actorRef, &fila.perfilRef, &fila.estado,
		&fila.versionExpediente, &fila.auditoriaRef, &fila.eventoRef,
		&fila.confirmadaEn,
	)
	if err != nil {
		return ports.PreparacionAlta{}, errorPostgreSQL(ctx, err)
	}
	if fila.resultado == "idempotencia_reutilizada" {
		return ports.PreparacionAlta{}, ports.ErrClaveIdempotenciaUsada
	}
	preparacion, err := fila.restaurar(solicitud, operacion)
	if err != nil {
		return ports.PreparacionAlta{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.PreparacionAlta{}, errorPostgreSQL(ctx, err)
	}
	// Tras un COMMIT confirmado no convertimos una cancelación tardía en un
	// fallo ambiguo para el llamador.
	return preparacion, nil
}

type filaPreparacionAlta struct {
	resultado          string
	reservaRef         string
	expedienteRef      string
	numeroVisible      string
	reciboRef          string
	ambitoHMAC         string
	huellaPeticionHMAC string
	organizacionRef    string
	actorRef           string
	perfilRef          string
	estado             string
	versionExpediente  pgtype.Int8
	auditoriaRef       pgtype.Text
	eventoRef          pgtype.Text
	confirmadaEn       pgtype.Timestamptz
}

func (f filaPreparacionAlta) restaurar(
	solicitud ports.SolicitudPrepararAlta,
	operacionJSON []byte,
) (ports.PreparacionAlta, error) {
	var operacion operacionPrepararAltaV2
	if json.Unmarshal(operacionJSON, &operacion) != nil ||
		!operacion.SellosHMAC.contieneAmbito(f.ambitoHMAC) ||
		!operacion.SellosHMAC.contieneHuella(f.huellaPeticionHMAC) {
		return ports.PreparacionAlta{}, ports.ErrPersistenciaNoDisponible
	}
	preparacion := ports.PreparacionAlta{
		ReservaRef: f.reservaRef,
		Referencias: ports.ReferenciasAlta{
			ExpedienteRef: f.expedienteRef,
			NumeroVisible: f.numeroVisible,
			ReciboRef:     f.reciboRef,
		},
		AmbitoIdempotenciaHMAC: f.ambitoHMAC,
		HuellaPeticionHMAC:     f.huellaPeticionHMAC,
		OrganizacionRef:        f.organizacionRef,
		ActorRef:               f.actorRef,
		PerfilRef:              f.perfilRef,
	}
	switch f.estado {
	case string(ports.PreparacionReservada):
		if f.resultado != "reservada" && f.resultado != "reutilizada" {
			return ports.PreparacionAlta{}, ports.ErrPersistenciaNoDisponible
		}
		if f.versionExpediente.Valid || f.auditoriaRef.Valid ||
			f.eventoRef.Valid || f.confirmadaEn.Valid {
			return ports.PreparacionAlta{}, ports.ErrPersistenciaNoDisponible
		}
		preparacion.Estado = ports.PreparacionReservada
		if f.resultado == "reservada" &&
			!respuestaReservadaCoincideConCandidatos(preparacion, operacion) {
			return ports.PreparacionAlta{}, ports.ErrPersistenciaNoDisponible
		}
	case string(ports.PreparacionConfirmada):
		if f.resultado != "confirmada" || !f.versionExpediente.Valid ||
			f.versionExpediente.Int64 < 1 || !f.auditoriaRef.Valid ||
			!f.eventoRef.Valid || !f.confirmadaEn.Valid {
			return ports.PreparacionAlta{}, ports.ErrPersistenciaNoDisponible
		}
		preparacion.Estado = ports.PreparacionConfirmada
		recibo := ports.ReciboAlta{
			ExpedienteRef: f.expedienteRef,
			NumeroVisible: f.numeroVisible,
			Version:       uint64(f.versionExpediente.Int64),
			ReciboRef:     f.reciboRef,
			AuditoriaRef:  f.auditoriaRef.String,
			EventoRef:     f.eventoRef.String,
			ConfirmadaEn:  f.confirmadaEn.Time.UTC(),
		}
		preparacion.ReciboConfirmado = &recibo
	default:
		return ports.PreparacionAlta{}, ports.ErrPersistenciaNoDisponible
	}
	if preparacion.ValidarPara(solicitud) != nil {
		return ports.PreparacionAlta{}, ports.ErrPersistenciaNoDisponible
	}
	return preparacion, nil
}

func respuestaReservadaCoincideConCandidatos(
	preparacion ports.PreparacionAlta,
	operacion operacionPrepararAltaV2,
) bool {
	return preparacion.ReservaRef == operacion.ReservaRefCandidata &&
		preparacion.Referencias == operacion.ReferenciasCandidatas.puertos() &&
		hmac.Equal(
			[]byte(preparacion.AmbitoIdempotenciaHMAC),
			[]byte(operacion.SellosHMAC.Activo.AmbitoHMAC),
		) &&
		hmac.Equal(
			[]byte(preparacion.HuellaPeticionHMAC),
			[]byte(operacion.SellosHMAC.Activo.HuellaPeticionHMAC),
		)
}

func (p *PreparadorAltaPostgreSQL) iniciar(ctx context.Context) (pgx.Tx, error) {
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return nil, errorPostgreSQL(ctx, err)
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
		return nil, errorPostgreSQL(ctx, err)
	}
	return tx, nil
}

func revertirTransaccion(tx pgx.Tx) {
	if tx == nil {
		return
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelar()
	_ = tx.Rollback(ctx)
}

func errorPostgreSQL(ctx context.Context, causa error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	_ = causa
	return ports.ErrPersistenciaNoDisponible
}

func errorDependencia(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ports.ErrPersistenciaNoDisponible
}

func dependenciaNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

func borrarBytes(contenido []byte) {
	for indice := range contenido {
		contenido[indice] = 0
	}
}
