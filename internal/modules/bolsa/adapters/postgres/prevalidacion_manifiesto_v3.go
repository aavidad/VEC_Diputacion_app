package postgres

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	transaccionbolsa "vec-diputacion-granada/internal/modules/bolsa/internal/transaccion"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const funcionPrevalidacionArchivoProbatorioV3 = "vec_bolsa_baremacion.obtener_archivo_probatorio_previo_cambio_v3"

type operacionPrevalidacionArchivoPostgreSQLV3 struct {
	Esquema                         string `json:"esquema"`
	Clase                           string `json:"clase"`
	BaremacionMeritoRef             string `json:"baremacion_merito_ref"`
	VersionEsperada                 string `json:"version_esperada"`
	HuellaVersionEsperadaSHA256     string `json:"huella_version_esperada_sha256"`
	HuellaTokenSHA256               string `json:"huella_token_sha256"`
	HuellaConfirmacionSHA256        string `json:"huella_confirmacion_sha256"`
	HuellaEfectoPrevalidacionSHA256 string `json:"huella_efecto_prevalidacion_sha256"`
}

type resultadoPrevalidacionArchivoPostgreSQLV3 struct {
	Estado                          string
	Version                         puertosbolsa.VersionBaremacion
	HuellaConfirmacionSHA256        string
	HuellaEfectoPrevalidacionSHA256 string
	HuellaPrevalidacionSHA256       string
	AutorizacionPrevalidacionRef    string
}

// prevalidarArchivoCambioV3 consume la autorizacion dedicada en una transaccion
// corta y la cierra antes de invocar el verificador HMAC. ConfirmarCambio vuelve
// a validar OCC y la autorizacion de confirmacion al aplicar el efecto.
func (r *RepositorioBaremaciones) prevalidarArchivoCambioV3(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
	instante time.Time,
	huellaConfirmacion string,
) (resultadoPrevalidacionArchivoPostgreSQLV3, error) {
	if solicitud.Clase != puertosbolsa.ClaseCambioIncorporarDecision ||
		solicitud.VersionEsperada == nil || solicitud.ContextoPrevalidacionArchivo.Validar() != nil {
		return resultadoPrevalidacionArchivoPostgreSQLV3{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	huellaConfirmacionCalculada, err := transaccionbolsa.HuellaEfectoConfirmacionV2(solicitud)
	if err != nil || huellaConfirmacionCalculada != huellaConfirmacion {
		return resultadoPrevalidacionArchivoPostgreSQLV3{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	huellaEfectoPrevalidacion, err := transaccionbolsa.HuellaEfectoPrevalidacionArchivoProbatorio(solicitud)
	if err != nil {
		return resultadoPrevalidacionArchivoPostgreSQLV3{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	prueba, decisionCanonica, recursoCanonico, err := serializarPruebaYRecurso(
		solicitud.ContextoPrevalidacionArchivo, instante, huellaEfectoPrevalidacion,
	)
	if err != nil {
		return resultadoPrevalidacionArchivoPostgreSQLV3{}, err
	}
	versionEsperada := solicitud.VersionEsperada
	operacion, err := json.Marshal(operacionPrevalidacionArchivoPostgreSQLV3{
		Esquema: "vec.bolsa.baremacion.prevalidacion-archivo-postgresql.v3",
		Clase:   string(solicitud.Clase), BaremacionMeritoRef: solicitud.Agregado.ID,
		VersionEsperada:                 strconv.FormatUint(versionEsperada.Numero, 10),
		HuellaVersionEsperadaSHA256:     versionEsperada.HuellaEstadoSHA256,
		HuellaTokenSHA256:               transaccionbolsa.HuellaTokenReserva(solicitud.Token),
		HuellaConfirmacionSHA256:        huellaConfirmacion,
		HuellaEfectoPrevalidacionSHA256: huellaEfectoPrevalidacion,
	})
	if err != nil {
		return resultadoPrevalidacionArchivoPostgreSQLV3{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	tx, err := r.iniciar(ctx)
	if err != nil {
		return resultadoPrevalidacionArchivoPostgreSQLV3{}, err
	}
	defer revertir(tx)
	var estado, numero, huella, huellaPrevalidacion string
	var agregado, archivo []byte
	var confirmada pgtype.Timestamptz
	consulta := `SELECT resultado, numero_version, huella_estado_sha256,
		agregado_canonico, confirmada_en, archivo_probatorio_documento,
		huella_prevalidacion_sha256 FROM ` + funcionPrevalidacionArchivoProbatorioV3 +
		`($1::jsonb,$2::jsonb,$3::bytea,$4::bytea)`
	err = tx.QueryRow(ctx, consulta, operacion, prueba, decisionCanonica, recursoCanonico).Scan(
		&estado, &numero, &huella, &agregado, &confirmada, &archivo, &huellaPrevalidacion,
	)
	defer borrarBytesPostgreSQL(agregado, archivo)
	if err != nil {
		return resultadoPrevalidacionArchivoPostgreSQLV3{}, errorPostgreSQL(ctx, err)
	}
	if estado != "activa" && estado != "confirmada" {
		errorResultado, reconocida := errorEstadoPrevalidacion(estado)
		if !reconocida {
			return resultadoPrevalidacionArchivoPostgreSQLV3{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		if err = tx.Commit(ctx); err != nil {
			return resultadoPrevalidacionArchivoPostgreSQLV3{}, errorPostgreSQL(ctx, err)
		}
		return resultadoPrevalidacionArchivoPostgreSQLV3{}, errorResultado
	}
	if !confirmada.Valid {
		return resultadoPrevalidacionArchivoPostgreSQLV3{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	if err = tx.Commit(ctx); err != nil {
		return resultadoPrevalidacionArchivoPostgreSQLV3{}, errorPostgreSQL(ctx, err)
	}
	if err = validarContexto(ctx); err != nil {
		return resultadoPrevalidacionArchivoPostgreSQLV3{}, err
	}
	version, err := construirVersion(solicitud.Agregado.ID, numero, huella, agregado, confirmada.Time)
	if err != nil {
		return resultadoPrevalidacionArchivoPostgreSQLV3{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	if estado == "activa" {
		if version.Referencia != *versionEsperada {
			return resultadoPrevalidacionArchivoPostgreSQLV3{}, puertosbolsa.ErrVersionBaremacionConflicto
		}
	} else {
		huellaSolicitada, errorHuella := solicitud.Agregado.HuellaEstadoSHA256()
		if errorHuella != nil || version.Referencia.Numero != versionEsperada.Numero+1 ||
			version.Referencia.HuellaEstadoSHA256 != huellaSolicitada {
			return resultadoPrevalidacionArchivoPostgreSQLV3{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
	}
	_, partesArchivo, err := r.verificarArchivoProbatorioV3(ctx, version, archivo)
	if err != nil {
		return resultadoPrevalidacionArchivoPostgreSQLV3{}, err
	}
	proyeccion := solicitud.ContextoPrevalidacionArchivo.Proyeccion()
	huellaCalculada := huellaPrevalidacionArchivoPostgreSQLV3(
		solicitud, version, huellaConfirmacion, partesArchivo,
	)
	if huellaCalculada != huellaPrevalidacion {
		return resultadoPrevalidacionArchivoPostgreSQLV3{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	return resultadoPrevalidacionArchivoPostgreSQLV3{
		Estado: estado, Version: version, HuellaConfirmacionSHA256: huellaConfirmacion,
		HuellaEfectoPrevalidacionSHA256: huellaEfectoPrevalidacion,
		HuellaPrevalidacionSHA256:       huellaCalculada,
		AutorizacionPrevalidacionRef:    proyeccion.AutorizacionRef,
	}, nil
}

// huellaPrevalidacionArchivoPostgreSQLV3 reproduce el canonico portable que
// calcula PostgreSQL. La primera prevalidacion liga el archivo de la version
// base; tras aplicar el cambio, PostgreSQL devuelve una huella distinta que
// liga el archivo final. Ambos estados deben usar exactamente la misma norma.
func huellaPrevalidacionArchivoPostgreSQLV3(
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
	version puertosbolsa.VersionBaremacion,
	huellaConfirmacion string,
	partesArchivo []string,
) string {
	proyeccion := solicitud.ContextoPrevalidacionArchivo.Proyeccion()
	partes := []string{
		"prevalidacion-archivo-probatorio-baremacion-v3",
		solicitud.Agregado.ID, strconv.FormatUint(version.Referencia.Numero, 10),
		version.Referencia.HuellaEstadoSHA256, transaccionbolsa.HuellaTokenReserva(solicitud.Token),
		proyeccion.PrincipalRef, proyeccion.SujetoRef, proyeccion.FinalidadClave,
		proyeccion.AutorizacionRef, huellaConfirmacion,
	}
	partes = append(partes, partesArchivo...)
	return transaccionbolsa.HuellaCanonica(partes...)
}

func validarHuellaPrevalidacionConfirmacionPostgreSQLV3(
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
	prevalidacion resultadoPrevalidacionArchivoPostgreSQLV3,
	versionFinal puertosbolsa.VersionBaremacion,
	partesArchivoFinal []string,
	huellaResultado string,
) error {
	switch prevalidacion.Estado {
	case "":
		if solicitud.Clase != puertosbolsa.ClaseCambioIncorporarDecision && huellaResultado == "" {
			return nil
		}
	case "confirmada":
		// En un replay no basta con que dos valores durables coincidan: se
		// vuelve a ligar la huella a la version y al archivo final recuperado.
		huellaFinal := huellaPrevalidacionArchivoPostgreSQLV3(
			solicitud, versionFinal, prevalidacion.HuellaConfirmacionSHA256,
			partesArchivoFinal,
		)
		if huellaResultado == prevalidacion.HuellaPrevalidacionSHA256 &&
			huellaResultado == huellaFinal {
			return nil
		}
	case "activa":
		// En la primera aplicacion, la huella prevalidada cubre la version base
		// y la salida SQL cubre la version final: exigir igualdad entre ambas
		// rechazaria toda mutacion valida.
		huellaFinal := huellaPrevalidacionArchivoPostgreSQLV3(
			solicitud, versionFinal, prevalidacion.HuellaConfirmacionSHA256,
			partesArchivoFinal,
		)
		if huellaResultado == huellaFinal {
			return nil
		}
	}
	return puertosbolsa.ErrEvidenciaBaremacionNoConfiable
}

func errorEstadoPrevalidacion(estado string) (error, bool) {
	switch estado {
	case "autorizacion_reutilizada":
		return puertosbolsa.ErrAutorizacionBaremacionReutilizada, true
	case "autorizacion_obsoleta":
		return puertosbolsa.ErrAutorizacionBaremacionInvalida, true
	case "reserva_invalida":
		return puertosbolsa.ErrReservaBaremacionNoValida, true
	case "evidencia_no_confiable":
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable, true
	case "rechazada":
		return puertosbolsa.ErrSolicitudBaremacionInvalida, true
	default:
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable, false
	}
}

func (r *RepositorioBaremaciones) verificarArchivoRespuestaReservaV3(
	ctx context.Context,
	estado string,
	respuesta puertosbolsa.ReservaCambioBaremacion,
	archivo []byte,
) error {
	if estado == "confirmada" {
		if respuesta.VersionConfirmada == nil {
			return puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		_, _, err := r.verificarArchivoProbatorioV3(ctx, *respuesta.VersionConfirmada, archivo)
		return err
	}
	if estado != "reservada" || len(archivo) != 0 {
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	return validarContexto(ctx)
}
