// Package postgres implementa la persistencia durable del agregado de
// baremacion. La cuenta de ejecucion no recibe permisos sobre tablas: este
// adaptador solo invoca funciones SECURITY DEFINER de contrato cerrado.
package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	transaccionbolsa "vec-diputacion-granada/internal/modules/bolsa/internal/transaccion"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

var _ puertosbolsa.RepositorioBaremaciones = (*RepositorioBaremaciones)(nil)

type iniciadorTransacciones interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// RepositorioBaremaciones mantiene las dependencias criptograficas fuera de
// PostgreSQL. El ensamblado productivo debe aportar un verificador HMAC real,
// con claves fuera del proceso y comparacion constante; una implementacion que
// acepte siempre no constituye una frontera de seguridad. PostgreSQL no confia
// en esa dependencia: vuelve a validar decision, sesion, rol, catalogo y una
// atestacion COSE durable justo antes de aplicar cada efecto, y falla cerrado
// si falta cualquiera de esas evidencias.
type RepositorioBaremaciones struct {
	pool        iniciadorTransacciones
	reloj       puertosbolsa.Reloj
	verificador puertosbolsa.VerificadorSellosBaremacion
}

func NuevoRepositorioBaremaciones(
	pool *pgxpool.Pool,
	reloj puertosbolsa.Reloj,
	verificador puertosbolsa.VerificadorSellosBaremacion,
) (*RepositorioBaremaciones, error) {
	return nuevoRepositorioBaremaciones(pool, reloj, verificador)
}

func nuevoRepositorioBaremaciones(
	pool iniciadorTransacciones,
	reloj puertosbolsa.Reloj,
	verificador puertosbolsa.VerificadorSellosBaremacion,
) (*RepositorioBaremaciones, error) {
	if valorNulo(pool) || valorNulo(reloj) || valorNulo(verificador) || reloj.Ahora().IsZero() {
		return nil, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	return &RepositorioBaremaciones{pool: pool, reloj: reloj, verificador: verificador}, nil
}

type operacionReservaPostgreSQL struct {
	Esquema                     string `json:"esquema"`
	ReservaRef                  string `json:"reserva_ref"`
	HuellaTokenSHA256           string `json:"huella_token_sha256"`
	AmbitoIdempotenciaSHA256    string `json:"ambito_idempotencia_sha256"`
	Clase                       string `json:"clase"`
	BaremacionMeritoRef         string `json:"baremacion_merito_ref"`
	VersionEsperada             string `json:"version_esperada"`
	HuellaVersionEsperadaSHA256 string `json:"huella_version_esperada_sha256"`
	HuellaSolicitudHMAC         string `json:"huella_solicitud_hmac"`
	HuellaEfectoSHA256          string `json:"huella_efecto_sha256"`
	SolicitadaEn                string `json:"solicitada_en"`
	ExpiraEn                    string `json:"expira_en"`
}

func (r *RepositorioBaremaciones) ReservarCambio(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudReservarCambioBaremacion,
) (puertosbolsa.ReservaCambioBaremacion, error) {
	if err := validarContexto(ctx); err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, err
	}
	solicitud = solicitud.Clonar()
	if solicitud.Validar() != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	ahora, err := r.ahora()
	if err != nil || solicitud.Contexto.ValidarVigentePara(
		accionReserva(solicitud.Clase), puertosbolsa.ClaseRecursoBaremacion,
		solicitud.BaremacionMeritoRef, ahora,
	) != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	if err = r.verificarSelloReserva(ctx, solicitud); err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, err
	}
	huellaEfecto, err := transaccionbolsa.HuellaEfectoReserva(solicitud)
	if err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	prueba, decisionCanonica, recursoCanonico, err := serializarPruebaYRecurso(
		solicitud.Contexto, ahora, huellaEfecto,
	)
	if err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, err
	}
	token, err := transaccionbolsa.GenerarTokenReserva()
	if err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrGeneracionReferenciaNoDisponible
	}
	reservaRef, err := transaccionbolsa.NuevaReferenciaOpaca()
	if err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrGeneracionReferenciaNoDisponible
	}
	version, huellaVersion := "0", ""
	if solicitud.VersionEsperada != nil {
		version = strconv.FormatUint(solicitud.VersionEsperada.Numero, 10)
		huellaVersion = solicitud.VersionEsperada.HuellaEstadoSHA256
	}
	proyeccion := solicitud.Contexto.Proyeccion()
	operacion, err := json.Marshal(operacionReservaPostgreSQL{
		Esquema:    "vec.bolsa.baremacion.reserva-postgresql.v1",
		ReservaRef: reservaRef, HuellaTokenSHA256: transaccionbolsa.HuellaTokenReserva(token),
		AmbitoIdempotenciaSHA256: transaccionbolsa.HuellaCanonica(
			"ambito-idempotencia-baremacion-v1", proyeccion.PrincipalRef, solicitud.ClaveIdempotencia,
		),
		Clase: string(solicitud.Clase), BaremacionMeritoRef: solicitud.BaremacionMeritoRef,
		VersionEsperada: version, HuellaVersionEsperadaSHA256: huellaVersion,
		HuellaSolicitudHMAC: solicitud.HuellaSolicitudHMAC, HuellaEfectoSHA256: huellaEfecto,
		SolicitadaEn: solicitud.SolicitadaEn.UTC().Format(time.RFC3339Nano),
		ExpiraEn:     solicitud.ExpiraEn.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}

	tx, err := r.iniciar(ctx)
	if err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, err
	}
	defer revertir(tx)
	var estado, reservaResultado, numero, huella, auditoriaRef, huellaAuditoria, eventoRef, huellaEvento string
	var expira, confirmada pgtype.Timestamptz
	var agregado []byte
	err = tx.QueryRow(ctx, `
		SELECT resultado, reserva_ref, expira_en, numero_version,
		       huella_estado_sha256, agregado_canonico, confirmada_en,
		       auditoria_ref, huella_auditoria_sha256,
		       evento_outbox_ref, huella_evento_outbox_sha256
		FROM vec_bolsa_baremacion.reservar_cambio(
			$1::jsonb, $2::jsonb, $3::bytea, $4::bytea
		)`, operacion, prueba, decisionCanonica, recursoCanonico,
	).Scan(
		&estado, &reservaResultado, &expira, &numero, &huella, &agregado,
		&confirmada, &auditoriaRef, &huellaAuditoria, &eventoRef, &huellaEvento,
	)
	if err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, errorPostgreSQL(ctx, err)
	}
	var respuesta puertosbolsa.ReservaCambioBaremacion
	var errorResultado error
	switch estado {
	case "reservada":
		respuesta = puertosbolsa.ReservaCambioBaremacion{
			Token: token, BaremacionMeritoRef: solicitud.BaremacionMeritoRef,
			Clase: solicitud.Clase, HuellaSolicitudHMAC: solicitud.HuellaSolicitudHMAC,
			ExpiraEn: solicitud.ExpiraEn.UTC(),
		}
		if solicitud.VersionEsperada != nil {
			esperada := *solicitud.VersionEsperada
			respuesta.VersionEsperada = &esperada
		}
		if reservaResultado != reservaRef || !expira.Valid ||
			!expira.Time.UTC().Equal(solicitud.ExpiraEn.UTC().Round(time.Microsecond)) ||
			numero != "" || huella != "" || agregado != nil || confirmada.Valid ||
			auditoriaRef != "" || huellaAuditoria != "" || eventoRef != "" || huellaEvento != "" ||
			respuesta.ValidarPara(solicitud) != nil {
			return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
	case "confirmada":
		if reservaResultado == "" || !expira.Valid ||
			!expira.Time.UTC().Equal(solicitud.ExpiraEn.UTC().Round(time.Microsecond)) ||
			!confirmada.Valid {
			return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		versionConfirmada, err := construirVersion(
			solicitud.BaremacionMeritoRef, numero, huella, agregado, confirmada.Time,
		)
		if err != nil {
			return puertosbolsa.ReservaCambioBaremacion{}, err
		}
		evidencia := puertosbolsa.EvidenciaTransaccionBaremacion{
			AuditoriaRef: auditoriaRef, HuellaAuditoriaSHA256: huellaAuditoria,
			EventoOutboxRef: eventoRef, HuellaEventoOutboxSHA256: huellaEvento,
			ConfirmadaEn: confirmada.Time.UTC(),
		}
		respuesta = puertosbolsa.ReservaCambioBaremacion{
			Repetida: true, VersionConfirmada: &versionConfirmada,
			BaremacionMeritoRef: solicitud.BaremacionMeritoRef, Clase: solicitud.Clase,
			HuellaSolicitudHMAC: solicitud.HuellaSolicitudHMAC, ExpiraEn: solicitud.ExpiraEn.UTC(),
		}
		if solicitud.VersionEsperada != nil {
			esperada := *solicitud.VersionEsperada
			respuesta.VersionEsperada = &esperada
		}
		if evidencia.Validar() != nil || respuesta.ValidarPara(solicitud) != nil {
			return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
	default:
		var reconocida bool
		errorResultado, reconocida = errorEstadoReserva(estado)
		if !reconocida {
			return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
	}
	if estado == "reservada" || estado == "confirmada" {
		respuesta, err = respuesta.Clonar()
		if err != nil {
			return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, errorPostgreSQL(ctx, err)
	}
	if errorResultado != nil {
		return puertosbolsa.ReservaCambioBaremacion{}, errorResultado
	}
	return respuesta, nil
}

type operacionConfirmacionPostgreSQL struct {
	Esquema                     string `json:"esquema"`
	HuellaTokenSHA256           string `json:"huella_token_sha256"`
	Clase                       string `json:"clase"`
	VersionEsperada             string `json:"version_esperada"`
	HuellaVersionEsperadaSHA256 string `json:"huella_version_esperada_sha256"`
	HuellaSolicitudHMAC         string `json:"huella_solicitud_hmac"`
	HuellaEfectoSHA256          string `json:"huella_efecto_sha256"`
	HuellaAgregadoSHA256        string `json:"huella_agregado_sha256"`
	MotivoClave                 string `json:"motivo_clave"`
	Motivo                      string `json:"motivo"`
	ConfirmadaEn                string `json:"confirmada_en"`
	AuditoriaRef                string `json:"auditoria_ref"`
	EventoOutboxRef             string `json:"evento_outbox_ref"`
}

func (r *RepositorioBaremaciones) ConfirmarCambio(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
) (puertosbolsa.ResultadoConfirmarCambioBaremacion, error) {
	if err := validarContexto(ctx); err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
	}
	clon, err := solicitud.Clonar()
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	solicitud = clon
	ahora, err := r.ahora()
	if err != nil || solicitud.ConfirmadaEn.After(ahora) || solicitud.Contexto.ValidarVigentePara(
		accionConfirmacion(solicitud.Clase), puertosbolsa.ClaseRecursoBaremacion,
		solicitud.Agregado.ID, ahora,
	) != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	if err = r.verificarSelloConfirmacion(ctx, solicitud); err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
	}
	huellaAgregado, err := solicitud.Agregado.HuellaEstadoSHA256()
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrHistorialBaremacionNoAnexable
	}
	agregadoCanonico, err := json.Marshal(solicitud.Agregado)
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrHistorialBaremacionNoAnexable
	}
	sumaAgregado := sha256.Sum256(agregadoCanonico)
	if hex.EncodeToString(sumaAgregado[:]) != huellaAgregado {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrHistorialBaremacionNoAnexable
	}
	huellaEfecto, err := transaccionbolsa.HuellaEfectoConfirmacion(solicitud)
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	prueba, decisionCanonica, recursoCanonico, err := serializarPruebaYRecurso(
		solicitud.Contexto, ahora, huellaEfecto,
	)
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
	}
	auditoriaRef, err := transaccionbolsa.NuevaReferenciaOpaca()
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrGeneracionReferenciaNoDisponible
	}
	eventoRef, err := transaccionbolsa.NuevaReferenciaOpaca()
	if err != nil || eventoRef == auditoriaRef {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrGeneracionReferenciaNoDisponible
	}
	version, huellaVersion := "0", ""
	if solicitud.VersionEsperada != nil {
		version = strconv.FormatUint(solicitud.VersionEsperada.Numero, 10)
		huellaVersion = solicitud.VersionEsperada.HuellaEstadoSHA256
	}
	operacion, err := json.Marshal(operacionConfirmacionPostgreSQL{
		Esquema:           "vec.bolsa.baremacion.confirmacion-postgresql.v1",
		HuellaTokenSHA256: transaccionbolsa.HuellaTokenReserva(solicitud.Token),
		Clase:             string(solicitud.Clase), VersionEsperada: version,
		HuellaVersionEsperadaSHA256: huellaVersion,
		HuellaSolicitudHMAC:         solicitud.HuellaSolicitudHMAC,
		HuellaEfectoSHA256:          huellaEfecto, HuellaAgregadoSHA256: huellaAgregado,
		MotivoClave: solicitud.Trazabilidad.MotivoClave, Motivo: solicitud.Trazabilidad.Motivo,
		ConfirmadaEn: solicitud.ConfirmadaEn.UTC().Format(time.RFC3339Nano),
		AuditoriaRef: auditoriaRef, EventoOutboxRef: eventoRef,
	})
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	tx, err := r.iniciar(ctx)
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
	}
	defer revertir(tx)
	var estado, numero, huella, auditoriaResultado, huellaAuditoria, eventoResultado, huellaEvento string
	var agregado []byte
	var confirmada pgtype.Timestamptz
	err = tx.QueryRow(ctx, `
		SELECT resultado, numero_version, huella_estado_sha256,
		       agregado_canonico, confirmada_en, auditoria_ref,
		       huella_auditoria_sha256, evento_outbox_ref,
		       huella_evento_outbox_sha256
		FROM vec_bolsa_baremacion.confirmar_cambio(
			$1::jsonb, $2::jsonb, $3::bytea, $4::bytea, $5::bytea
		)`, operacion, prueba, decisionCanonica, recursoCanonico, agregadoCanonico,
	).Scan(
		&estado, &numero, &huella, &agregado, &confirmada,
		&auditoriaResultado, &huellaAuditoria, &eventoResultado, &huellaEvento,
	)
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, errorPostgreSQL(ctx, err)
	}
	if estado != "confirmada" {
		errorResultado, reconocida := errorEstadoConfirmacion(estado)
		if !reconocida {
			return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		if err = tx.Commit(ctx); err != nil {
			return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, errorPostgreSQL(ctx, err)
		}
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, errorResultado
	}
	if !confirmada.Valid {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	versionConfirmada, err := construirVersion(
		solicitud.Agregado.ID, numero, huella, agregado, confirmada.Time,
	)
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
	}
	resultado := puertosbolsa.ResultadoConfirmarCambioBaremacion{
		Version: versionConfirmada,
		Evidencia: puertosbolsa.EvidenciaTransaccionBaremacion{
			AuditoriaRef: auditoriaResultado, HuellaAuditoriaSHA256: huellaAuditoria,
			EventoOutboxRef: eventoResultado, HuellaEventoOutboxSHA256: huellaEvento,
			ConfirmadaEn: confirmada.Time.UTC(),
		},
	}
	if resultado.ValidarPara(solicitud) != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	resultadoClonado, err := resultado.Clonar()
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	if err = tx.Commit(ctx); err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, errorPostgreSQL(ctx, err)
	}
	return resultadoClonado, nil
}

type operacionAbandonoPostgreSQL struct {
	Esquema             string `json:"esquema"`
	HuellaTokenSHA256   string `json:"huella_token_sha256"`
	Clase               string `json:"clase"`
	BaremacionMeritoRef string `json:"baremacion_merito_ref"`
	HuellaEfectoSHA256  string `json:"huella_efecto_sha256"`
}

func (r *RepositorioBaremaciones) AbandonarReserva(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudAbandonarReservaBaremacion,
) error {
	if err := validarContexto(ctx); err != nil {
		return err
	}
	if solicitud.Validar() != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	ahora, err := r.ahora()
	if err != nil || solicitud.Contexto.ValidarVigentePara(
		accionAbandono(solicitud.Clase), puertosbolsa.ClaseRecursoBaremacion,
		solicitud.BaremacionMeritoRef, ahora,
	) != nil {
		return puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	huellaEfecto, err := transaccionbolsa.HuellaEfectoAbandono(solicitud)
	if err != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	prueba, decisionCanonica, recursoCanonico, err := serializarPruebaYRecurso(
		solicitud.Contexto, ahora, huellaEfecto,
	)
	if err != nil {
		return err
	}
	operacion, err := json.Marshal(operacionAbandonoPostgreSQL{
		Esquema:           "vec.bolsa.baremacion.abandono-postgresql.v1",
		HuellaTokenSHA256: transaccionbolsa.HuellaTokenReserva(solicitud.Token),
		Clase:             string(solicitud.Clase), BaremacionMeritoRef: solicitud.BaremacionMeritoRef,
		HuellaEfectoSHA256: huellaEfecto,
	})
	if err != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	tx, err := r.iniciar(ctx)
	if err != nil {
		return err
	}
	defer revertir(tx)
	var estado string
	err = tx.QueryRow(ctx, `SELECT vec_bolsa_baremacion.abandonar_reserva(
		$1::jsonb, $2::jsonb, $3::bytea, $4::bytea
	)`, operacion, prueba, decisionCanonica, recursoCanonico).Scan(&estado)
	if err != nil {
		return errorPostgreSQL(ctx, err)
	}
	if estado == "abandonada" {
		if err = tx.Commit(ctx); err != nil {
			return errorPostgreSQL(ctx, err)
		}
		return nil
	}
	errorResultado, reconocida := errorEstadoAbandono(estado)
	if !reconocida {
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	if err = tx.Commit(ctx); err != nil {
		return errorPostgreSQL(ctx, err)
	}
	return errorResultado
}

type operacionLecturaVigentePostgreSQL struct {
	Esquema             string `json:"esquema"`
	BaremacionMeritoRef string `json:"baremacion_merito_ref"`
	HuellaEfectoSHA256  string `json:"huella_efecto_sha256"`
}

func (r *RepositorioBaremaciones) ObtenerVersionVigente(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerBaremacionVigente,
) (puertosbolsa.VersionBaremacion, error) {
	if err := validarContexto(ctx); err != nil {
		return puertosbolsa.VersionBaremacion{}, err
	}
	if solicitud.Validar() != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	huellaEfecto := transaccionbolsa.HuellaCanonica(
		"lectura-baremacion-vigente-v1", solicitud.BaremacionMeritoRef,
	)
	return r.obtenerVersion(ctx, solicitud.Contexto, operacionLecturaVigentePostgreSQL{
		Esquema:             "vec.bolsa.baremacion.lectura-vigente-postgresql.v1",
		BaremacionMeritoRef: solicitud.BaremacionMeritoRef,
		HuellaEfectoSHA256:  huellaEfecto,
	}, "vec_bolsa_baremacion.obtener_version_vigente", solicitud.BaremacionMeritoRef, 0, huellaEfecto)
}

type operacionLecturaVersionPostgreSQL struct {
	Esquema             string `json:"esquema"`
	BaremacionMeritoRef string `json:"baremacion_merito_ref"`
	NumeroVersion       string `json:"numero_version"`
	HuellaEfectoSHA256  string `json:"huella_efecto_sha256"`
}

func (r *RepositorioBaremaciones) ObtenerVersion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerVersionBaremacion,
) (puertosbolsa.VersionBaremacion, error) {
	if err := validarContexto(ctx); err != nil {
		return puertosbolsa.VersionBaremacion{}, err
	}
	if solicitud.Validar() != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	numero := strconv.FormatUint(solicitud.Numero, 10)
	huellaEfecto := transaccionbolsa.HuellaCanonica(
		"lectura-version-baremacion-v1", solicitud.BaremacionMeritoRef, numero,
	)
	return r.obtenerVersion(ctx, solicitud.Contexto, operacionLecturaVersionPostgreSQL{
		Esquema:             "vec.bolsa.baremacion.lectura-version-postgresql.v1",
		BaremacionMeritoRef: solicitud.BaremacionMeritoRef, NumeroVersion: numero,
		HuellaEfectoSHA256: huellaEfecto,
	}, "vec_bolsa_baremacion.obtener_version", solicitud.BaremacionMeritoRef, solicitud.Numero, huellaEfecto)
}

func (r *RepositorioBaremaciones) obtenerVersion(
	ctx context.Context,
	contexto puertosbolsa.ContextoOperacionBaremacion,
	operacion any,
	funcion, baremacionRef string,
	numeroEsperado uint64,
	huellaEfecto string,
) (puertosbolsa.VersionBaremacion, error) {
	ahora, err := r.ahora()
	if err != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	prueba, decisionCanonica, recursoCanonico, err := serializarPruebaYRecurso(contexto, ahora, huellaEfecto)
	if err != nil {
		return puertosbolsa.VersionBaremacion{}, err
	}
	operacionJSON, err := json.Marshal(operacion)
	if err != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	tx, err := r.iniciar(ctx)
	if err != nil {
		return puertosbolsa.VersionBaremacion{}, err
	}
	defer revertir(tx)
	var estado, numero, huella, auditoriaRef string
	var agregado []byte
	var confirmada pgtype.Timestamptz
	consulta := `SELECT resultado, numero_version, huella_estado_sha256,
		agregado_canonico, confirmada_en, auditoria_ref FROM ` + funcion + `(
		$1::jsonb, $2::jsonb, $3::bytea, $4::bytea)`
	err = tx.QueryRow(ctx, consulta, operacionJSON, prueba, decisionCanonica, recursoCanonico).Scan(
		&estado, &numero, &huella, &agregado, &confirmada, &auditoriaRef,
	)
	if err != nil {
		return puertosbolsa.VersionBaremacion{}, errorPostgreSQL(ctx, err)
	}
	if estado != "obtenida" {
		if estado == "no_encontrada" {
			if err = tx.Commit(ctx); err != nil {
				return puertosbolsa.VersionBaremacion{}, errorPostgreSQL(ctx, err)
			}
			if numeroEsperado == 0 {
				return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrBaremacionNoEncontrada
			}
			return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrVersionBaremacionNoEncontrada
		}
		errorResultado, reconocida := errorEstadoLectura(estado)
		if !reconocida {
			return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		if err = tx.Commit(ctx); err != nil {
			return puertosbolsa.VersionBaremacion{}, errorPostgreSQL(ctx, err)
		}
		return puertosbolsa.VersionBaremacion{}, errorResultado
	}
	if !confirmada.Valid {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	version, err := construirVersion(baremacionRef, numero, huella, agregado, confirmada.Time)
	if err != nil || (numeroEsperado != 0 && version.Referencia.Numero != numeroEsperado) {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	version, err = version.Clonar()
	if err != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	if auditoriaRef == "" {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	if err = tx.Commit(ctx); err != nil {
		return puertosbolsa.VersionBaremacion{}, errorPostgreSQL(ctx, err)
	}
	return version, nil
}

type operacionLecturaEvidenciaPostgreSQL struct {
	Esquema             string `json:"esquema"`
	BaremacionMeritoRef string `json:"baremacion_merito_ref"`
	NumeroVersion       string `json:"numero_version"`
	AuditoriaRef        string `json:"auditoria_ref"`
	EventoOutboxRef     string `json:"evento_outbox_ref"`
	HuellaEfectoSHA256  string `json:"huella_efecto_sha256"`
}

func (r *RepositorioBaremaciones) ObtenerEvidenciaTransaccion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerEvidenciaTransaccionBaremacion,
) (puertosbolsa.EvidenciaTransaccionBaremacionRecuperada, error) {
	if err := validarContexto(ctx); err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, err
	}
	if solicitud.Validar() != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	numero := strconv.FormatUint(solicitud.NumeroVersion, 10)
	huellaEfecto := transaccionbolsa.HuellaCanonica(
		"lectura-evidencia-transaccion-baremacion-v1", solicitud.BaremacionMeritoRef,
		numero, solicitud.AuditoriaRef, solicitud.EventoOutboxRef,
	)
	ahora, err := r.ahora()
	if err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	prueba, decisionCanonica, recursoCanonico, err := serializarPruebaYRecurso(
		solicitud.Contexto, ahora, huellaEfecto,
	)
	if err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, err
	}
	operacion, err := json.Marshal(operacionLecturaEvidenciaPostgreSQL{
		Esquema:             "vec.bolsa.baremacion.lectura-evidencia-postgresql.v1",
		BaremacionMeritoRef: solicitud.BaremacionMeritoRef, NumeroVersion: numero,
		AuditoriaRef: solicitud.AuditoriaRef, EventoOutboxRef: solicitud.EventoOutboxRef,
		HuellaEfectoSHA256: huellaEfecto,
	})
	if err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	tx, err := r.iniciar(ctx)
	if err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, err
	}
	defer revertir(tx)
	var estado, numeroResultado, huella string
	var agregado, auditoriaJSON, eventoJSON []byte
	var confirmada pgtype.Timestamptz
	err = tx.QueryRow(ctx, `
		SELECT resultado, numero_version, huella_estado_sha256,
		       agregado_canonico, confirmada_en,
		       auditoria_documento, evento_documento
		FROM vec_bolsa_baremacion.obtener_evidencia_transaccion(
			$1::jsonb, $2::jsonb, $3::bytea, $4::bytea
		)`, operacion, prueba, decisionCanonica, recursoCanonico,
	).Scan(&estado, &numeroResultado, &huella, &agregado, &confirmada, &auditoriaJSON, &eventoJSON)
	if err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, errorPostgreSQL(ctx, err)
	}
	if estado != "obtenida" {
		if estado == "no_encontrada" {
			if err = tx.Commit(ctx); err != nil {
				return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, errorPostgreSQL(ctx, err)
			}
			return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrEvidenciaBaremacionNoEncontrada
		}
		errorResultado, reconocida := errorEstadoLectura(estado)
		if !reconocida {
			return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		if err = tx.Commit(ctx); err != nil {
			return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, errorPostgreSQL(ctx, err)
		}
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, errorResultado
	}
	if !confirmada.Valid {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	version, err := construirVersion(
		solicitud.BaremacionMeritoRef, numeroResultado, huella, agregado, confirmada.Time,
	)
	if err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, err
	}
	var filaAuditoria registroAuditoriaPostgreSQL
	var filaEvento eventoOutboxPostgreSQL
	if decodificarJSONEstricto(auditoriaJSON, &filaAuditoria) != nil ||
		decodificarJSONEstricto(eventoJSON, &filaEvento) != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	auditoria, err := filaAuditoria.dominio()
	if err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, err
	}
	evento, err := filaEvento.dominio()
	if err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, err
	}
	resultado := puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{
		Version: version, Auditoria: auditoria, Evento: evento,
		Evidencia: puertosbolsa.EvidenciaTransaccionBaremacion{
			AuditoriaRef: auditoria.Referencia, HuellaAuditoriaSHA256: auditoria.HuellaRegistroSHA256,
			EventoOutboxRef: evento.Referencia, HuellaEventoOutboxSHA256: evento.HuellaRegistroSHA256,
			ConfirmadaEn: auditoria.RegistradaEn,
		},
	}
	if resultado.ValidarPara(solicitud) != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	if err = tx.Commit(ctx); err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, errorPostgreSQL(ctx, err)
	}
	return resultado, nil
}

func (r *RepositorioBaremaciones) iniciar(ctx context.Context) (pgx.Tx, error) {
	if r == nil || valorNulo(r.pool) {
		return nil, puertosbolsa.ErrFuenteBaremacionNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
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
		revertir(tx)
		return nil, errorPostgreSQL(ctx, err)
	}
	return tx, nil
}

func (r *RepositorioBaremaciones) ahora() (time.Time, error) {
	if r == nil || valorNulo(r.reloj) {
		return time.Time{}, puertosbolsa.ErrFuenteBaremacionNoDisponible
	}
	ahora := r.reloj.Ahora().UTC()
	if ahora.IsZero() {
		return time.Time{}, puertosbolsa.ErrFuenteBaremacionNoDisponible
	}
	return ahora, nil
}

func (r *RepositorioBaremaciones) verificarSelloReserva(
	ctx context.Context, solicitud puertosbolsa.SolicitudReservarCambioBaremacion,
) error {
	if r == nil || valorNulo(r.verificador) {
		return puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible
	}
	representacion, err := puertosbolsa.RepresentacionCanonicaReservaBaremacion(solicitud)
	if err != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	peticion := puertosbolsa.SolicitudVerificarSelloBaremacion{
		Finalidad:              puertosbolsa.FinalidadSelloReservaBaremacion,
		RepresentacionCanonica: representacion, SelloHMAC: solicitud.HuellaSolicitudHMAC,
	}
	if peticion.Validar() != nil || r.verificador.VerificarSelloBaremacion(ctx, peticion) != nil {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	return nil
}

func (r *RepositorioBaremaciones) verificarSelloConfirmacion(
	ctx context.Context, solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
) error {
	if r == nil || valorNulo(r.verificador) {
		return puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible
	}
	representacion, err := puertosbolsa.RepresentacionCanonicaConfirmacionBaremacion(solicitud)
	if err != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	peticion := puertosbolsa.SolicitudVerificarSelloBaremacion{
		Finalidad:              puertosbolsa.FinalidadSelloConfirmacionBaremacion,
		RepresentacionCanonica: representacion, SelloHMAC: solicitud.HuellaSolicitudHMAC,
	}
	if peticion.Validar() != nil || r.verificador.VerificarSelloBaremacion(ctx, peticion) != nil {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	return nil
}

func accionReserva(clase puertosbolsa.ClaseCambioBaremacion) puertosbolsa.AccionOperacionBaremacion {
	if clase == puertosbolsa.ClaseCambioAltaBaremacion {
		return puertosbolsa.AccionReservarAltaBaremacion
	}
	return puertosbolsa.AccionReservarDecisionBaremacion
}

func accionConfirmacion(clase puertosbolsa.ClaseCambioBaremacion) puertosbolsa.AccionOperacionBaremacion {
	if clase == puertosbolsa.ClaseCambioAltaBaremacion {
		return puertosbolsa.AccionConfirmarAltaBaremacion
	}
	return puertosbolsa.AccionConfirmarDecisionBaremacion
}

func accionAbandono(clase puertosbolsa.ClaseCambioBaremacion) puertosbolsa.AccionOperacionBaremacion {
	if clase == puertosbolsa.ClaseCambioAltaBaremacion {
		return puertosbolsa.AccionAbandonarAltaBaremacion
	}
	return puertosbolsa.AccionAbandonarDecisionBaremacion
}

func errorEstadoReserva(estado string) (error, bool) {
	switch estado {
	case "en_curso":
		return puertosbolsa.ErrCambioBaremacionEnCurso, true
	case "idempotencia_reutilizada":
		return puertosbolsa.ErrClaveIdempotenciaBaremacionReutilizada, true
	case "autorizacion_reutilizada":
		return puertosbolsa.ErrAutorizacionBaremacionReutilizada, true
	case "autorizacion_obsoleta":
		return puertosbolsa.ErrAutorizacionBaremacionInvalida, true
	case "ya_existe":
		return puertosbolsa.ErrBaremacionYaExiste, true
	case "conflicto_version":
		return puertosbolsa.ErrVersionBaremacionConflicto, true
	case "no_encontrada":
		return puertosbolsa.ErrBaremacionNoEncontrada, true
	case "evidencia_no_confiable":
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable, true
	case "colision":
		return puertosbolsa.ErrCambioBaremacionEnCurso, true
	case "rechazada":
		return puertosbolsa.ErrSolicitudBaremacionInvalida, true
	default:
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable, false
	}
}

func errorEstadoConfirmacion(estado string) (error, bool) {
	switch estado {
	case "reserva_invalida":
		return puertosbolsa.ErrReservaBaremacionNoValida, true
	case "idempotencia_reutilizada":
		return puertosbolsa.ErrClaveIdempotenciaBaremacionReutilizada, true
	case "autorizacion_reutilizada":
		return puertosbolsa.ErrAutorizacionBaremacionReutilizada, true
	case "autorizacion_obsoleta":
		return puertosbolsa.ErrAutorizacionBaremacionInvalida, true
	case "ya_existe":
		return puertosbolsa.ErrBaremacionYaExiste, true
	case "conflicto_version":
		return puertosbolsa.ErrVersionBaremacionConflicto, true
	case "no_encontrada":
		return puertosbolsa.ErrBaremacionNoEncontrada, true
	case "historial_no_anexable":
		return puertosbolsa.ErrHistorialBaremacionNoAnexable, true
	case "evidencia_no_confiable":
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable, true
	case "colision":
		return puertosbolsa.ErrCambioBaremacionEnCurso, true
	case "rechazada":
		return puertosbolsa.ErrSolicitudBaremacionInvalida, true
	default:
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable, false
	}
}

func errorEstadoAbandono(estado string) (error, bool) {
	switch estado {
	case "reserva_invalida":
		return puertosbolsa.ErrReservaBaremacionNoValida, true
	case "idempotencia_reutilizada":
		return puertosbolsa.ErrClaveIdempotenciaBaremacionReutilizada, true
	case "autorizacion_reutilizada":
		return puertosbolsa.ErrAutorizacionBaremacionReutilizada, true
	case "autorizacion_obsoleta":
		return puertosbolsa.ErrAutorizacionBaremacionInvalida, true
	case "colision":
		return puertosbolsa.ErrCambioBaremacionEnCurso, true
	case "rechazada":
		return puertosbolsa.ErrSolicitudBaremacionInvalida, true
	default:
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable, false
	}
}

func errorEstadoLectura(estado string) (error, bool) {
	switch estado {
	case "autorizacion_reutilizada":
		return puertosbolsa.ErrAutorizacionBaremacionReutilizada, true
	case "autorizacion_obsoleta":
		return puertosbolsa.ErrAutorizacionBaremacionInvalida, true
	case "evidencia_no_confiable":
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable, true
	case "colision":
		return puertosbolsa.ErrCambioBaremacionEnCurso, true
	case "rechazada":
		return puertosbolsa.ErrSolicitudBaremacionInvalida, true
	default:
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable, false
	}
}

func errorPostgreSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		switch pgError.Code {
		case "40001", "40P01", "55P03", "57014":
			return puertosbolsa.ErrCambioBaremacionEnCurso
		}
	}
	return puertosbolsa.ErrFuenteBaremacionNoDisponible
}

func validarContexto(ctx context.Context) error {
	if ctx == nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	return ctx.Err()
}

func revertir(tx pgx.Tx) {
	if tx == nil {
		return
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelar()
	_ = tx.Rollback(ctx)
}

func valorNulo(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
