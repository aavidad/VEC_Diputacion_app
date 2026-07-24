package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type operacionArrendamientoFlujoFirmaPostgreSQL struct {
	Esquema                string `json:"esquema"`
	FlujoRef               string `json:"flujo_ref"`
	IndiceIdempotenciaHMAC string `json:"indice_idempotencia_hmac,omitempty"`
	VinculoActorHMAC       string `json:"vinculo_actor_hmac,omitempty"`
	VersionEsperada        string `json:"version_esperada,omitempty"`
	PropietarioRef         string `json:"propietario_ref"`
	SecuenciaCercado       string `json:"secuencia_cercado,omitempty"`
	ExpiraEn               string `json:"expira_en,omitempty"`
	DuracionMicrosegundos  string `json:"duracion_microsegundos,omitempty"`
}

func (r *RepositorioFlujosFirmaBaremacionPostgreSQL) GuardarFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudGuardarFlujoFirmaBaremacion,
) (puertosbolsa.ExpedienteFlujoFirmaBaremacion, error) {
	if err := validarContextoFlujoFirmaPostgreSQL(ctx); err != nil ||
		r == nil || valorNulo(r.pool) || valorNulo(r.verificador) ||
		r.operarHMACToken == nil || solicitud.Validar() != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			combinarErrorFlujoFirmaPostgreSQL(
				puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida,
				err,
			)
	}
	if err := r.verificarExpediente(ctx, solicitud.Siguiente); err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	documentoSiguiente, cifradoSiguiente, err :=
		serializarExpedienteFlujoFirmaPostgreSQL(solicitud.Siguiente)
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	defer borrarBytesPostgreSQL(documentoSiguiente, cifradoSiguiente)
	huellaToken, _, err := r.operarHMACToken(solicitud.Arrendamiento.Token, nil)
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	}
	defer borrarBytesPostgreSQL(huellaToken)
	operacion, err := serializarOperacionArrendamientoFlujoFirmaPostgreSQL(
		solicitud.Arrendamiento,
		solicitud.VersionEsperada,
	)
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	defer borrarBytesPostgreSQL(operacion)
	tx, err := r.iniciar(ctx, pgx.ReadWrite)
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	defer revertir(tx)
	consulta := puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion{
		FlujoRef:               solicitud.Siguiente.FlujoRef,
		IndiceIdempotenciaHMAC: solicitud.Siguiente.IndiceIdempotenciaHMAC,
		VinculoActorHMAC:       solicitud.Siguiente.VinculoActorHMAC,
	}
	anterior, err := r.obtenerEnTransaccion(ctx, tx, consulta)
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	if puertosbolsa.ValidarTransicionFlujoFirmaBaremacion(
		anterior,
		solicitud.Siguiente,
	) != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	var resultado string
	var documentoPersistido, cifradoPersistido []byte
	err = tx.QueryRow(ctx, `
		SELECT resultado, expediente_documento, estado_cifrado
		  FROM `+funcionGuardarFlujoFirmaPostgreSQLV1+`(
		       $1::jsonb, $2::jsonb, $3::bytea, $4::bytea
		  )`,
		operacion,
		documentoSiguiente,
		cifradoSiguiente,
		huellaToken,
	).Scan(&resultado, &documentoPersistido, &cifradoPersistido)
	defer borrarBytesPostgreSQL(documentoPersistido, cifradoPersistido)
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			errorFlujoFirmaPostgreSQL(ctx, err)
	}
	switch resultado {
	case "conflicto":
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			puertosbolsa.ErrConflictoFlujoFirmaBaremacion
	case "arrendamiento_invalido":
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	case "no_encontrado":
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			puertosbolsa.ErrFlujoFirmaBaremacionNoEncontrado
	case "estado_alterado":
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			puertosbolsa.ErrEstadoFlujoFirmaAlterado
	case "guardado":
	default:
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			ErrRepositorioFlujoFirmaPostgreSQLNoDisponible
	}
	persistido, err := r.decodificarYVerificar(
		ctx,
		documentoPersistido,
		cifradoPersistido,
	)
	if err != nil || !expedientesFlujoFirmaPostgreSQLExactos(
		solicitud.Siguiente,
		persistido,
	) {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			combinarErrorFlujoFirmaPostgreSQL(
				puertosbolsa.ErrEstadoFlujoFirmaAlterado,
				err,
			)
	}
	if err := tx.Commit(ctx); err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			errorFlujoFirmaPostgreSQL(ctx, err)
	}
	return persistido, nil
}

func (r *RepositorioFlujosFirmaBaremacionPostgreSQL) LiberarArrendamientoFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudLiberarArrendamientoFlujoFirmaBaremacion,
) error {
	if err := validarContextoFlujoFirmaPostgreSQL(ctx); err != nil ||
		r == nil || valorNulo(r.pool) || r.operarHMACToken == nil ||
		solicitud.Arrendamiento.Validar() != nil {
		return combinarErrorFlujoFirmaPostgreSQL(
			puertosbolsa.ErrArrendamientoFlujoFirmaInvalido,
			err,
		)
	}
	operacion, err := serializarOperacionArrendamientoFlujoFirmaPostgreSQL(
		solicitud.Arrendamiento,
		0,
	)
	if err != nil {
		return err
	}
	defer borrarBytesPostgreSQL(operacion)
	huellaToken, _, err := r.operarHMACToken(solicitud.Arrendamiento.Token, nil)
	if err != nil {
		return puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	}
	defer borrarBytesPostgreSQL(huellaToken)
	tx, err := r.iniciar(ctx, pgx.ReadWrite)
	if err != nil {
		return err
	}
	defer revertir(tx)
	var resultado string
	err = tx.QueryRow(ctx, `
		SELECT resultado
		  FROM `+funcionLiberarFlujoFirmaPostgreSQLV1+`(
		       $1::jsonb, $2::bytea
		  )`,
		operacion,
		huellaToken,
	).Scan(&resultado)
	if err != nil {
		return errorFlujoFirmaPostgreSQL(ctx, err)
	}
	switch resultado {
	case "liberado", "ausente":
	case "arrendamiento_invalido":
		return puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	default:
		return ErrRepositorioFlujoFirmaPostgreSQLNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return errorFlujoFirmaPostgreSQL(ctx, err)
	}
	return nil
}

func serializarOperacionArrendamientoFlujoFirmaPostgreSQL(
	arrendamiento puertosbolsa.ArrendamientoFlujoFirmaBaremacion,
	versionEsperada uint64,
) ([]byte, error) {
	if arrendamiento.Validar() != nil {
		return nil, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	}
	operacion := operacionArrendamientoFlujoFirmaPostgreSQL{
		Esquema:        esquemaOperacionArrendamientoFlujoFirmaPostgreSQLV1,
		FlujoRef:       arrendamiento.FlujoRef,
		PropietarioRef: arrendamiento.PropietarioRef,
		SecuenciaCercado: strconvFlujoFirmaPostgreSQL(
			arrendamiento.SecuenciaCercado,
		),
		ExpiraEn: instanteFlujoFirmaPostgreSQL(arrendamiento.ExpiraEn),
	}
	if versionEsperada > 0 {
		operacion.VersionEsperada = strconvFlujoFirmaPostgreSQL(versionEsperada)
	}
	contenido, err := json.Marshal(operacion)
	if err != nil {
		return nil, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	}
	return contenido, nil
}

func (r *RepositorioFlujosFirmaBaremacionPostgreSQL) obtenerEnTransaccion(
	ctx context.Context,
	tx pgx.Tx,
	solicitud puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion,
) (puertosbolsa.ExpedienteFlujoFirmaBaremacion, error) {
	var documento, cifrado []byte
	err := tx.QueryRow(ctx, `
		SELECT expediente_documento, estado_cifrado
		  FROM `+funcionObtenerFlujoFirmaPostgreSQLV1+`(
		       $1::text, $2::text, $3::text
		  )`,
		solicitud.FlujoRef,
		solicitud.IndiceIdempotenciaHMAC,
		solicitud.VinculoActorHMAC,
	).Scan(&documento, &cifrado)
	defer borrarBytesPostgreSQL(documento, cifrado)
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			errorFlujoFirmaPostgreSQL(ctx, err)
	}
	expediente, err := r.decodificarYVerificar(ctx, documento, cifrado)
	if err != nil || expediente.FlujoRef != solicitud.FlujoRef ||
		expediente.IndiceIdempotenciaHMAC != solicitud.IndiceIdempotenciaHMAC ||
		expediente.VinculoActorHMAC != solicitud.VinculoActorHMAC {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			combinarErrorFlujoFirmaPostgreSQL(
				puertosbolsa.ErrFlujoFirmaBaremacionNoEncontrado,
				err,
			)
	}
	return expediente, nil
}

func (r *RepositorioFlujosFirmaBaremacionPostgreSQL) decodificarYVerificar(
	ctx context.Context,
	documento, cifrado []byte,
) (puertosbolsa.ExpedienteFlujoFirmaBaremacion, error) {
	expediente, err := decodificarExpedienteFlujoFirmaPostgreSQL(documento, cifrado)
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	if err := r.verificarExpediente(ctx, expediente); err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	return expediente, nil
}

func (r *RepositorioFlujosFirmaBaremacionPostgreSQL) verificarExpediente(
	ctx context.Context,
	expediente puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) error {
	canonica, err := puertosbolsa.RepresentacionCanonicaExpedienteFlujoFirmaBaremacion(
		expediente,
	)
	if err != nil {
		return puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	solicitud := puertosbolsa.SolicitudVerificarEstadoFlujoFirmaBaremacion{
		RepresentacionCanonica: canonica,
		SelloHMAC:              expediente.SelloEstadoHMAC,
	}
	if solicitud.Validar() != nil ||
		r.verificador.VerificarEstadoFlujoFirmaBaremacion(ctx, solicitud) != nil {
		return puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	return nil
}

func expedientesFlujoFirmaPostgreSQLExactos(
	primero, segundo puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) bool {
	canonicaPrimero, errPrimero :=
		puertosbolsa.RepresentacionCanonicaExpedienteFlujoFirmaBaremacion(primero)
	canonicaSegundo, errSegundo :=
		puertosbolsa.RepresentacionCanonicaExpedienteFlujoFirmaBaremacion(segundo)
	if errPrimero != nil || errSegundo != nil ||
		primero.SelloEstadoHMAC != segundo.SelloEstadoHMAC {
		return false
	}
	bytesPrimero := canonicaPrimero.Revelar()
	bytesSegundo := canonicaSegundo.Revelar()
	defer borrarBytesPostgreSQL(bytesPrimero, bytesSegundo)
	return bytes.Equal(bytesPrimero, bytesSegundo)
}

func strconvFlujoFirmaPostgreSQL(valor uint64) string {
	return strconv.FormatUint(valor, 10)
}

func (r *RepositorioFlujosFirmaBaremacionPostgreSQL) validarContextoYEstado(
	ctx context.Context,
	expediente puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) error {
	if err := validarContextoFlujoFirmaPostgreSQL(ctx); err != nil {
		return err
	}
	if r == nil || valorNulo(r.pool) || valorNulo(r.verificador) ||
		r.operarHMACToken == nil || expediente.Validar() != nil {
		return puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	return r.verificarExpediente(ctx, expediente)
}

func (r *RepositorioFlujosFirmaBaremacionPostgreSQL) iniciar(
	ctx context.Context,
	modo pgx.TxAccessMode,
) (pgx.Tx, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: modo,
	})
	if err != nil {
		return nil, errorFlujoFirmaPostgreSQL(ctx, err)
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
		return nil, errorFlujoFirmaPostgreSQL(ctx, err)
	}
	return tx, nil
}

func validarContextoFlujoFirmaPostgreSQL(ctx context.Context) error {
	if ctx == nil {
		return puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	return ctx.Err()
}

func errorFlujoFirmaPostgreSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return puertosbolsa.ErrFlujoFirmaBaremacionNoEncontrado
	}
	var errorPG *pgconn.PgError
	if errors.As(err, &errorPG) {
		switch errorPG.Code {
		case "40001", "40P01", "55P03":
			return puertosbolsa.ErrConflictoFlujoFirmaBaremacion
		case "23514", "22023", "22001":
			return puertosbolsa.ErrEstadoFlujoFirmaAlterado
		}
	}
	return ErrRepositorioFlujoFirmaPostgreSQLNoDisponible
}

func combinarErrorFlujoFirmaPostgreSQL(base, causa error) error {
	if causa == nil || errors.Is(causa, base) {
		return base
	}
	return errors.Join(base, causa)
}
