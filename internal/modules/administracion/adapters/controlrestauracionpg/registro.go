// Package controlrestauracionpg adapta el control nominal de propuestas de
// restauración a las fachadas PostgreSQL que consumen V3 de forma atómica.
package controlrestauracionpg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	ordenpg "vec-diputacion-granada/internal/modules/administracion/adapters/ordenescopias/postgres"
	dominio "vec-diputacion-granada/internal/modules/administracion/domain/ordenescopias"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrRegistro = errors.New("control_restauracion_postgres_no_disponible")

const (
	accionProponer    = "copias_restauracion_proponer"
	accionRevisar     = "copias_restauracion_revisar"
	finalidadProponer = "proponer_restauracion_copia"
	finalidadRevisar  = "revisar_restauracion_copia"
	modulo            = "administracion"
	tipoRecurso       = "propuesta_restauracion_copia"
	maxRespuesta      = 16 << 10

	registrarPropuestaSQL = `SELECT vec_administracion_copias.registrar_propuesta_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	registrarRevisionSQL  = `SELECT vec_administracion_copias.registrar_revision_v1($1,$2::numeric,$3,$4,$5,$6,$7::numeric,$8::numeric,$9,$10,$11,$12)`
)

// Resultado conserva el recibo de la escritura realizada por PostgreSQL. No
// constituye una autorización ni acredita una restauración de plataforma.
type Resultado struct {
	Orden        string    `json:"orden"`
	PlanSHA256   string    `json:"plan_sha256"`
	PersonaRef   string    `json:"persona_ref"`
	Version      uint64    `json:"version"`
	Recibo       string    `json:"recibo"`
	RegistradaEn time.Time `json:"registrada_en"`
	Replay       bool      `json:"replay"`
}

// Proponer registra sólo la propuesta nominal del proponente de la orden.
// La función SQL revalida y consume la autoridad V3 actual en la misma
// transacción que conserva la propuesta y el recibo.
func (r *Registro) Proponer(
	ctx context.Context,
	orden dominio.Orden,
	solicitud vecdomain.SolicitudAutorizacionLigadaV3,
	decision vecdomain.DecisionAutorizacionLigadaV3,
	confirmacion vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
) (Resultado, error) {
	return r.registrar(ctx, orden, 0, solicitud, decision, confirmacion, accionProponer, finalidadProponer, 1)
}

// RevisarCAS registra únicamente la segunda versión de la propuesta y exige
// que el aprobador nominal de la orden sea quien presenta la revisión.
func (r *Registro) RevisarCAS(
	ctx context.Context,
	orden dominio.Orden,
	versionEsperada uint64,
	solicitud vecdomain.SolicitudAutorizacionLigadaV3,
	decision vecdomain.DecisionAutorizacionLigadaV3,
	confirmacion vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
) (Resultado, error) {
	if versionEsperada == 0 {
		return Resultado{}, ErrRegistro
	}
	return r.registrar(ctx, orden, versionEsperada, solicitud, decision, confirmacion, accionRevisar, finalidadRevisar, 2)
}

func (r *Registro) registrar(
	ctx context.Context,
	orden dominio.Orden,
	versionEsperada uint64,
	solicitud vecdomain.SolicitudAutorizacionLigadaV3,
	decision vecdomain.DecisionAutorizacionLigadaV3,
	confirmacion vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	accion, finalidad string,
	versionResultado uint64,
) (Resultado, error) {
	if r == nil || esNulo(r.material) || ctx == nil || ctx.Err() != nil {
		return Resultado{}, ErrRegistro
	}
	datosOrden, plan, planSHA256, personaEsperada, err := validarNominal(
		orden, solicitud, decision, confirmacion, accion, finalidad,
	)
	if err != nil {
		return Resultado{}, ErrRegistro
	}
	if esNulo(r.pool) {
		return Resultado{}, ErrRegistro
	}
	canonico, err := vecdomain.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		return Resultado{}, ErrRegistro
	}
	return r.enTransaccion(ctx, func(canal CanalSQL) (Resultado, error) {
		material, err := r.material.MaterializarOrdenV3EnTX(ctx, canal, orden, solicitud, decision, confirmacion)
		if err != nil || !materialValido(material) || !bytes.Equal(canonico, material.Decision) || ctx.Err() != nil {
			return Resultado{}, ErrRegistro
		}
		argumentos := []any{plan, material.Capacidad, material.Decision, material.Motivo, material.Contexto,
			material.PersonaVersion, material.PerfilVersion, material.Payload, material.Sobre, material.Evidencia, material.Raiz}
		consulta := registrarPropuestaSQL
		if versionEsperada != 0 {
			consulta = registrarRevisionSQL
			argumentos = append([]any{plan, versionEsperada}, argumentos[1:]...)
		}
		var bruto []byte
		if err := canal.QueryRow(ctx, consulta, argumentos...).Scan(&bruto); err != nil || ctx.Err() != nil {
			return Resultado{}, ErrRegistro
		}
		resultado, err := resultadoDesdeSQL(bruto)
		if err != nil || resultado.Orden != datosOrden.Orden || resultado.PlanSHA256 != planSHA256 ||
			resultado.PersonaRef != personaEsperada || resultado.Version != versionResultado {
			return Resultado{}, ErrRegistro
		}
		return resultado, nil
	})
}

func validarNominal(
	orden dominio.Orden,
	solicitud vecdomain.SolicitudAutorizacionLigadaV3,
	decision vecdomain.DecisionAutorizacionLigadaV3,
	confirmacion vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	accion, finalidad string,
) (dominio.Datos, []byte, string, string, error) {
	datosOrden, err := orden.Datos()
	if err != nil || decision.ValidarPara(solicitud) != nil {
		return dominio.Datos{}, nil, "", "", ErrRegistro
	}
	concedida, _, err := decision.Resultado()
	datosSolicitud, errSolicitud := solicitud.Datos()
	datosConfirmacion, errConfirmacion := confirmacion.Datos()
	huellaDecision, errHuella := vecdomain.HuellaSHA256DecisionAutorizacionV3(decision)
	canonico, errCanonico := vecdomain.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	var resumen struct {
		DecisionRef string `json:"decision_ref"`
	}
	errResumen := json.Unmarshal(canonico, &resumen)
	plan, errPlan := orden.PlanBytes()
	planSHA256, errPlanSHA256 := orden.PlanSHA256()
	personaEsperada := datosOrden.ProponentePersona
	if accion == accionRevisar {
		personaEsperada = datosOrden.AprobadorPersona
	}
	if err != nil || !concedida || errSolicitud != nil || errConfirmacion != nil || errHuella != nil ||
		errCanonico != nil || errResumen != nil || resumen.DecisionRef == "" || errPlan != nil || errPlanSHA256 != nil || len(plan) == 0 ||
		datosSolicitud.Accion != accion || datosSolicitud.Finalidad != finalidad ||
		datosSolicitud.Recurso.ModuloID != modulo || datosSolicitud.Recurso.Tipo != tipoRecurso ||
		datosSolicitud.Recurso.Referencia != datosOrden.Orden ||
		datosConfirmacion.DecisionRef != resumen.DecisionRef || datosConfirmacion.DecisionHuellaSHA256 != huellaDecision {
		return dominio.Datos{}, nil, "", "", ErrRegistro
	}
	return datosOrden, plan, planSHA256, personaEsperada, nil
}

func resultadoDesdeSQL(bruto []byte) (Resultado, error) {
	if len(bruto) == 0 || len(bruto) > maxRespuesta || !formaResultadoCerrada(bruto) {
		return Resultado{}, ErrRegistro
	}
	var resultado Resultado
	decodificador := json.NewDecoder(bytes.NewReader(bruto))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&resultado); err != nil || decodificador.Decode(new(any)) != io.EOF || !resultadoValido(resultado) {
		return Resultado{}, ErrRegistro
	}
	resultado.RegistradaEn = resultado.RegistradaEn.UTC()
	return resultado, nil
}

func formaResultadoCerrada(bruto []byte) bool {
	decodificador := json.NewDecoder(bytes.NewReader(bruto))
	inicio, err := decodificador.Token()
	if err != nil || inicio != json.Delim('{') {
		return false
	}
	esperadas := map[string]bool{
		"orden": false, "plan_sha256": false, "persona_ref": false, "version": false,
		"recibo": false, "registrada_en": false, "replay": false,
	}
	for decodificador.More() {
		token, err := decodificador.Token()
		clave, ok := token.(string)
		if err != nil || !ok || esperadas[clave] {
			return false
		}
		if _, conocida := esperadas[clave]; !conocida {
			return false
		}
		esperadas[clave] = true
		var valor json.RawMessage
		if decodificador.Decode(&valor) != nil {
			return false
		}
	}
	fin, err := decodificador.Token()
	if err != nil || fin != json.Delim('}') || decodificador.Decode(new(any)) != io.EOF {
		return false
	}
	for _, presente := range esperadas {
		if !presente {
			return false
		}
	}
	return true
}

func resultadoValido(resultado Resultado) bool {
	if !referencia(resultado.Orden) || !referencia(resultado.PersonaRef) || !referencia(resultado.Recibo) ||
		!huella(resultado.PlanSHA256) || resultado.Version == 0 || resultado.RegistradaEn.IsZero() ||
		resultado.RegistradaEn.Location() != time.UTC || resultado.RegistradaEn.Nanosecond()%1000 != 0 {
		return false
	}
	return true
}

func referencia(valor string) bool {
	if len(valor) == 0 || len(valor) > 128 {
		return false
	}
	for i, caracter := range valor {
		if !((caracter >= 'a' && caracter <= 'z') || (caracter >= 'A' && caracter <= 'Z') ||
			(caracter >= '0' && caracter <= '9') || caracter == ':' || caracter == '_' || caracter == '-') ||
			(i == 0 && (caracter == ':' || caracter == '_' || caracter == '-')) {
			return false
		}
	}
	return true
}

func huella(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	for _, caracter := range valor {
		if !((caracter >= '0' && caracter <= '9') || (caracter >= 'a' && caracter <= 'f')) {
			return false
		}
	}
	return true
}

func materialValido(material ordenpg.MaterialV3) bool {
	if material.PersonaVersion == 0 || material.PerfilVersion == 0 {
		return false
	}
	for _, valor := range [][]byte{material.Capacidad, material.Decision, material.Motivo, material.Contexto, material.Payload, material.Sobre, material.Evidencia, material.Raiz} {
		if len(valor) == 0 || len(valor) > 1<<20 {
			return false
		}
	}
	return true
}
