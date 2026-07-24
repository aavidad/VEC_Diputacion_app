// La cuenta de ejecución de la saga de firma solo invoca fachadas SECURITY
// DEFINER. No recibe acceso directo al expediente, versiones, arrendamientos,
// auditoría ni outbox.
package postgres

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	funcionCrearFlujoFirmaPostgreSQLV1    = "vec_bolsa_firma.crear_o_recuperar_flujo_v1"
	funcionObtenerFlujoFirmaPostgreSQLV1  = "vec_bolsa_firma.obtener_flujo_v1"
	funcionAdquirirFlujoFirmaPostgreSQLV1 = "vec_bolsa_firma.adquirir_arrendamiento_v1"
	funcionGuardarFlujoFirmaPostgreSQLV1  = "vec_bolsa_firma.guardar_flujo_v1"
	funcionLiberarFlujoFirmaPostgreSQLV1  = "vec_bolsa_firma.liberar_arrendamiento_v1"

	esquemaOperacionArrendamientoFlujoFirmaPostgreSQLV1 = "vec.bolsa.firma.arrendamiento-postgresql.v1"
)

var (
	_ puertosbolsa.RepositorioFlujosFirmaBaremacion = (*RepositorioFlujosFirmaBaremacionPostgreSQL)(nil)

	ErrRepositorioFlujoFirmaPostgreSQLNoDisponible = errors.New(
		"bolsa: repositorio PostgreSQL de flujo de firma no disponible",
	)
)

type operacionHMACTokenFlujoFirmaPostgreSQL func(
	token puertosbolsa.TokenArrendamientoFlujoFirmaBaremacion,
	huellaEsperada []byte,
) (huella []byte, coincide bool, err error)

// RepositorioFlujosFirmaBaremacionPostgreSQL conserva únicamente una operación
// HMAC privada. La clave llega desde el gestor de secretos de composición y no
// queda en un campo reflectible, serializable ni registrable.
type RepositorioFlujosFirmaBaremacionPostgreSQL struct {
	pool            iniciadorTransacciones
	verificador     puertosbolsa.VerificadorEstadoFlujoFirmaBaremacion
	operarHMACToken operacionHMACTokenFlujoFirmaPostgreSQL
}

func NuevoRepositorioFlujosFirmaBaremacionPostgreSQL(
	pool *pgxpool.Pool,
	verificador puertosbolsa.VerificadorEstadoFlujoFirmaBaremacion,
	claveHMACToken []byte,
) (*RepositorioFlujosFirmaBaremacionPostgreSQL, error) {
	return nuevoRepositorioFlujosFirmaBaremacionPostgreSQL(
		pool,
		verificador,
		claveHMACToken,
	)
}

func nuevoRepositorioFlujosFirmaBaremacionPostgreSQL(
	pool iniciadorTransacciones,
	verificador puertosbolsa.VerificadorEstadoFlujoFirmaBaremacion,
	claveHMACToken []byte,
) (*RepositorioFlujosFirmaBaremacionPostgreSQL, error) {
	if valorNulo(pool) || valorNulo(verificador) || len(claveHMACToken) != 32 {
		return nil, ErrRepositorioFlujoFirmaPostgreSQLNoDisponible
	}
	var clave [32]byte
	copy(clave[:], claveHMACToken)
	claveCapturada := clave
	for indice := range clave {
		clave[indice] = 0
	}
	operar := func(
		token puertosbolsa.TokenArrendamientoFlujoFirmaBaremacion,
		esperada []byte,
	) ([]byte, bool, error) {
		huella, err := token.HuellaHMACSHA256(claveCapturada[:])
		if err != nil {
			return nil, false, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
		}
		coincide := len(esperada) == len(huella) && hmac.Equal(huella, esperada)
		return huella, coincide, nil
	}
	return &RepositorioFlujosFirmaBaremacionPostgreSQL{
		pool: pool, verificador: verificador, operarHMACToken: operar,
	}, nil
}

func (*RepositorioFlujosFirmaBaremacionPostgreSQL) String() string {
	return "[REPOSITORIO-FLUJOS-FIRMA-BAREMACION-POSTGRESQL-REDACTADO]"
}
func (r *RepositorioFlujosFirmaBaremacionPostgreSQL) GoString() string {
	return r.String()
}
func (r *RepositorioFlujosFirmaBaremacionPostgreSQL) Format(
	estado fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(estado, r.String())
}
func (r *RepositorioFlujosFirmaBaremacionPostgreSQL) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

func (r *RepositorioFlujosFirmaBaremacionPostgreSQL) CrearORecuperarFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudCrearORecuperarFlujoFirmaBaremacion,
) (puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion, error) {
	if err := r.validarContextoYEstado(ctx, solicitud.Expediente); err != nil ||
		solicitud.Expediente.Version != 1 ||
		len(solicitud.Expediente.PuntosControl) != 0 ||
		solicitud.Expediente.Estado != puertosbolsa.EstadoExpedienteFirmaPreparando {
		return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{},
			combinarErrorFlujoFirmaPostgreSQL(
				puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida,
				err,
			)
	}
	documento, cifrado, err := serializarExpedienteFlujoFirmaPostgreSQL(
		solicitud.Expediente,
	)
	if err != nil {
		return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{}, err
	}
	defer borrarBytesPostgreSQL(documento, cifrado)
	tx, err := r.iniciar(ctx, pgx.ReadWrite)
	if err != nil {
		return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{}, err
	}
	defer revertir(tx)
	var resultado string
	var documentoPersistido, cifradoPersistido []byte
	err = tx.QueryRow(ctx, `
		SELECT resultado, expediente_documento, estado_cifrado
		  FROM `+funcionCrearFlujoFirmaPostgreSQLV1+`(
		       $1::jsonb, $2::bytea
		  )`,
		documento,
		cifrado,
	).Scan(&resultado, &documentoPersistido, &cifradoPersistido)
	defer borrarBytesPostgreSQL(documentoPersistido, cifradoPersistido)
	if err != nil {
		return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{},
			errorFlujoFirmaPostgreSQL(ctx, err)
	}
	persistido, err := r.decodificarYVerificar(
		ctx,
		documentoPersistido,
		cifradoPersistido,
	)
	if err != nil {
		return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{}, err
	}
	creado := false
	switch resultado {
	case "creado":
		creado = true
		if !expedientesFlujoFirmaPostgreSQLExactos(
			solicitud.Expediente,
			persistido,
		) {
			return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{},
				puertosbolsa.ErrEstadoFlujoFirmaAlterado
		}
	case "recuperado":
		if !puertosbolsa.MismaSolicitudInicialFlujoFirmaBaremacion(
			persistido,
			solicitud.Expediente,
		) {
			return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{},
				puertosbolsa.ErrClaveFlujoFirmaBaremacionReutilizada
		}
	case "reutilizada":
		return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{},
			puertosbolsa.ErrClaveFlujoFirmaBaremacionReutilizada
	case "conflicto":
		return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{},
			puertosbolsa.ErrConflictoFlujoFirmaBaremacion
	default:
		return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{},
			ErrRepositorioFlujoFirmaPostgreSQLNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{},
			errorFlujoFirmaPostgreSQL(ctx, err)
	}
	return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{
		Expediente: persistido,
		Creado:     creado,
	}, nil
}

func (r *RepositorioFlujosFirmaBaremacionPostgreSQL) ObtenerFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion,
) (puertosbolsa.ExpedienteFlujoFirmaBaremacion, error) {
	if err := validarContextoFlujoFirmaPostgreSQL(ctx); err != nil ||
		r == nil || valorNulo(r.pool) || valorNulo(r.verificador) ||
		solicitud.Validar() != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			combinarErrorFlujoFirmaPostgreSQL(
				puertosbolsa.ErrFlujoFirmaBaremacionNoEncontrado,
				err,
			)
	}
	tx, err := r.iniciar(ctx, pgx.ReadOnly)
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	defer revertir(tx)
	expediente, err := r.obtenerEnTransaccion(ctx, tx, solicitud)
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			errorFlujoFirmaPostgreSQL(ctx, err)
	}
	return expediente, nil
}

func (r *RepositorioFlujosFirmaBaremacionPostgreSQL) AdquirirArrendamientoFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudAdquirirArrendamientoFlujoFirmaBaremacion,
) (puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion, error) {
	if err := validarContextoFlujoFirmaPostgreSQL(ctx); err != nil ||
		r == nil || valorNulo(r.pool) || r.operarHMACToken == nil ||
		solicitud.Validar() != nil || solicitud.Duracion%time.Microsecond != 0 {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{},
			combinarErrorFlujoFirmaPostgreSQL(
				puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida,
				err,
			)
	}
	token, err := puertosbolsa.NuevoTokenArrendamientoFlujoFirmaBaremacion()
	if err != nil {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{},
			puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	}
	huellaToken, _, err := r.operarHMACToken(token, nil)
	if err != nil {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{},
			puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	}
	defer borrarBytesPostgreSQL(huellaToken)
	operacion, err := json.Marshal(operacionArrendamientoFlujoFirmaPostgreSQL{
		Esquema:                esquemaOperacionArrendamientoFlujoFirmaPostgreSQLV1,
		FlujoRef:               solicitud.Consulta.FlujoRef,
		IndiceIdempotenciaHMAC: solicitud.Consulta.IndiceIdempotenciaHMAC,
		VinculoActorHMAC:       solicitud.Consulta.VinculoActorHMAC,
		VersionEsperada:        strconvFlujoFirmaPostgreSQL(solicitud.VersionEsperada),
		PropietarioRef:         solicitud.PropietarioRef,
		DuracionMicrosegundos: strconvFlujoFirmaPostgreSQL(
			uint64(solicitud.Duracion / time.Microsecond),
		),
	})
	if err != nil {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{},
			puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	defer borrarBytesPostgreSQL(operacion)
	tx, err := r.iniciar(ctx, pgx.ReadWrite)
	if err != nil {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{}, err
	}
	defer revertir(tx)
	var resultado, secuencia string
	var documento, cifrado []byte
	var expira pgtype.Timestamptz
	err = tx.QueryRow(ctx, `
		SELECT resultado, expediente_documento, estado_cifrado,
		       secuencia_cercado, expira_en
		  FROM `+funcionAdquirirFlujoFirmaPostgreSQLV1+`(
		       $1::jsonb, $2::bytea
		  )`,
		operacion,
		huellaToken,
	).Scan(&resultado, &documento, &cifrado, &secuencia, &expira)
	defer borrarBytesPostgreSQL(documento, cifrado)
	if err != nil {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{},
			errorFlujoFirmaPostgreSQL(ctx, err)
	}
	switch resultado {
	case "ocupado":
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{},
			puertosbolsa.ErrFlujoFirmaBaremacionOcupado
	case "conflicto":
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{},
			puertosbolsa.ErrConflictoFlujoFirmaBaremacion
	case "no_encontrado":
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{},
			puertosbolsa.ErrFlujoFirmaBaremacionNoEncontrado
	case "adquirido":
	default:
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{},
			ErrRepositorioFlujoFirmaPostgreSQLNoDisponible
	}
	expediente, err := r.decodificarYVerificar(ctx, documento, cifrado)
	if err != nil || expediente.FlujoRef != solicitud.Consulta.FlujoRef ||
		expediente.Version != solicitud.VersionEsperada ||
		!expira.Valid {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{},
			combinarErrorFlujoFirmaPostgreSQL(
				puertosbolsa.ErrArrendamientoFlujoFirmaInvalido,
				err,
			)
	}
	numeroSecuencia, err := enteroCanonicoFlujoFirmaPostgreSQL(secuencia)
	if err != nil {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{},
			puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	}
	arrendamiento := puertosbolsa.ArrendamientoFlujoFirmaBaremacion{
		FlujoRef: expediente.FlujoRef, PropietarioRef: solicitud.PropietarioRef,
		SecuenciaCercado: numeroSecuencia, ExpiraEn: expira.Time.UTC(), Token: token,
	}
	if arrendamiento.Validar() != nil {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{},
			puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	}
	if err := tx.Commit(ctx); err != nil {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{},
			errorFlujoFirmaPostgreSQL(ctx, err)
	}
	return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{
		Expediente: expediente, Arrendamiento: arrendamiento,
	}, nil
}
