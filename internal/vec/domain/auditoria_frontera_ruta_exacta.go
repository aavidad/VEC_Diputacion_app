package domain

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const SuperficieAuditoriaFronteraRutaExactaContratacionTemporal = "api.contratacion_temporal.ruta_exacta"
const SuperficieAuditoriaFronteraRutaExactaOrganizacionHistoricaPersonal = "organizacion_historica_personal"
const SuperficieAuditoriaFronteraRutaExactaPersonal = "api.personal.registro_empleado.ruta_exacta"
const SuperficieAuditoriaFronteraRutaExactaAuditoria = "api.auditoria.ruta_exacta"
const SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias = "api.usuarios.preferencias.ruta_exacta"
const SuperficieAuditoriaFronteraRutaExactaBolsaCandidato = "api.bolsa.candidato.ruta_exacta"

// Aspirantes no guarda `per_`: sus denegaciones de frontera nunca llevan actor.
const SuperficieAuditoriaFronteraRutaExactaAspirantes = "api.aspirantes.ficha.ruta_exacta"
const SuperficieAuditoriaFronteraRutaExactaSeleccionPreparacionBases = "api.seleccion.preparacion_bases.ruta_exacta"
const SuperficieAuditoriaFronteraRutaExactaBolsaReglasBaremo = "api.bolsa.reglas_baremo.ruta_exacta"

var ErrOrdenAuditoriaFronteraRutaExactaInvalida = errors.New(
	"vec ports: orden de auditoria de frontera de ruta exacta invalida",
)

// MotivoAuditoriaFronteraRutaExacta es cerrado para no trasladar errores ni
// detalles de autenticación a la bitácora de frontera.
type MotivoAuditoriaFronteraRutaExacta string

const (
	MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida MotivoAuditoriaFronteraRutaExacta = "autenticacion_requerida"
	MotivoAuditoriaFronteraRutaExactaAccesoDenegado         MotivoAuditoriaFronteraRutaExacta = "acceso_denegado"
)

// OrdenAuditoriaFronteraRutaExacta conserva solo los datos minimizados de una
// denegación temprana. Ruta no contiene consulta y ActorRef solo puede llegar
// de una autoridad que ya haya verificado la identidad.
type OrdenAuditoriaFronteraRutaExacta struct {
	CorrelacionRef string
	Motivo         MotivoAuditoriaFronteraRutaExacta
	Superficie     string
	Ruta           string
	ActorRef       string
}

func (o OrdenAuditoriaFronteraRutaExacta) Validar() error {
	if !correlacionAuditoriaFronteraRutaExactaValida(o.CorrelacionRef) ||
		(o.Motivo != MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida &&
			o.Motivo != MotivoAuditoriaFronteraRutaExactaAccesoDenegado) ||
		!rutaAuditoriaFronteraRutaExactaValidaParaSuperficie(o.Superficie, o.Ruta) ||
		(o.ActorRef != "" && !referenciaActorAuditoriaFronteraRutaExactaValida(o.ActorRef)) ||
		((o.Superficie == SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias ||
			o.Superficie == SuperficieAuditoriaFronteraRutaExactaBolsaCandidato) &&
			((o.Motivo == MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida && o.ActorRef != "") ||
				(o.Motivo == MotivoAuditoriaFronteraRutaExactaAccesoDenegado &&
					!referenciaOpacaContextoActorValida(o.ActorRef, "per_")))) ||
		(o.Superficie == SuperficieAuditoriaFronteraRutaExactaAspirantes && o.ActorRef != "") ||
		(o.Superficie == SuperficieAuditoriaFronteraRutaExactaSeleccionPreparacionBases &&
			(o.ActorRef != "" || o.Motivo != MotivoAuditoriaFronteraRutaExactaAccesoDenegado)) ||
		(o.Superficie == SuperficieAuditoriaFronteraRutaExactaBolsaReglasBaremo && o.ActorRef != "") {
		return ErrOrdenAuditoriaFronteraRutaExactaInvalida
	}
	return nil
}

func rutaAuditoriaFronteraRutaExactaValidaParaSuperficie(superficie, ruta string) bool {
	switch superficie {
	case SuperficieAuditoriaFronteraRutaExactaContratacionTemporal:
		return rutaAuditoriaFronteraRutaExactaValida(ruta)
	case SuperficieAuditoriaFronteraRutaExactaOrganizacionHistoricaPersonal:
		return ruta == "/api/vec/personal/organizacion-historica"
	case SuperficieAuditoriaFronteraRutaExactaPersonal:
		if ruta == "/api/vec/personal/vacantes" || ruta == "/api/vec/personal/empleados" ||
			ruta == "/api/vec/personal/empleados-organismo" ||
			ruta == "/api/vec/personal/hechos" || ruta == "/api/vec/personal/empleados/{emp_ref}" ||
			ruta == "/api/vec/personal/catalogos-registro-empleado" {
			return true
		}
		const prefijo = "/api/vec/personal/empleados/"
		if !strings.HasPrefix(ruta, prefijo) || len(ruta) > 512 || strings.ContainsAny(ruta, "?#%\\") {
			return false
		}
		referencia := strings.TrimPrefix(ruta, prefijo)
		if len(referencia) < len("emp_")+22 || len(referencia) > len("emp_")+128 || !strings.HasPrefix(referencia, "emp_") {
			return false
		}
		for _, caracter := range referencia[4:] {
			if (caracter < 'a' || caracter > 'z') && (caracter < 'A' || caracter > 'Z') &&
				(caracter < '0' || caracter > '9') && caracter != '_' && caracter != '-' {
				return false
			}
		}
		return true
	case SuperficieAuditoriaFronteraRutaExactaAuditoria:
		return ruta == "/api/vec/auditoria/opciones" || ruta == "/api/vec/auditoria/consultas"
	case SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias:
		return ruta == "/api/vec/usuarios/mis-preferencias" ||
			ruta == "/api/vec/usuarios/area-personal/mis-preferencias" ||
			ruta == "/api/vec/usuarios/mis-correos" ||
			ruta == "/api/vec/usuarios/area-personal/mis-correos" ||
			ruta == "/api/vec/usuarios/mi-imagen" ||
			ruta == "/api/vec/usuarios/area-personal/mi-imagen"
	case SuperficieAuditoriaFronteraRutaExactaBolsaCandidato:
		return ruta == "/api/vec/bolsa/mi-bolsa" ||
			ruta == "/api/vec/bolsa/mi-bolsa/historial" ||
			ruta == "/api/vec/bolsa/mi-bolsa/solicitudes" ||
			ruta == "/api/vec/bolsa/mi-bolsa/solicitudes-documentales" ||
			ruta == "/api/vec/bolsa/mi-bolsa/respuestas" ||
			ruta == "/api/vec/bolsa/mi-bolsa/disposiciones" ||
			ruta == "/api/vec/bolsa/mi-bolsa/contacto"
	case SuperficieAuditoriaFronteraRutaExactaAspirantes:
		return ruta == "/api/vec/aspirantes/area-personal/mi-ficha"
	case SuperficieAuditoriaFronteraRutaExactaSeleccionPreparacionBases:
		return ruta == "/api/vec/seleccion/preparacion-bases/guardar" ||
			ruta == "/api/vec/seleccion/preparacion-bases/consultar"
	case SuperficieAuditoriaFronteraRutaExactaBolsaReglasBaremo:
		return ruta == "/api/vec/bolsa/reglas-baremo/borradores/alta" ||
			ruta == "/api/vec/bolsa/reglas-baremo/versiones/consultar" ||
			ruta == "/api/vec/bolsa/reglas-baremo/recibos/recuperar"
	default:
		return false
	}
}

func correlacionAuditoriaFronteraRutaExactaValida(valor string) bool {
	if valor == "corr_no_disponible" {
		return true
	}
	if len(valor) != len("corr_")+32 || !strings.HasPrefix(valor, "corr_") {
		return false
	}
	for _, caracter := range valor[len("corr_"):] {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

func rutaAuditoriaFronteraRutaExactaValida(ruta string) bool {
	if len(ruta) <= len("/api/vec/contratacion-temporal/") || len(ruta) > 512 ||
		!strings.HasPrefix(ruta, "/api/vec/contratacion-temporal/") ||
		strings.HasSuffix(ruta, "/") || strings.ContainsAny(ruta, "?#%\\") {
		return false
	}
	for _, segmento := range strings.Split(strings.TrimPrefix(ruta, "/api/vec/"), "/") {
		if segmento == "" || len(segmento) > 64 {
			return false
		}
		for _, caracter := range segmento {
			if (caracter < 'a' || caracter > 'z') &&
				(caracter < '0' || caracter > '9') && caracter != '-' && caracter != '_' {
				return false
			}
		}
	}
	return true
}

func referenciaActorAuditoriaFronteraRutaExactaValida(actor string) bool {
	if actor == "" || len(actor) > 512 || actor != strings.TrimSpace(actor) || !utf8.ValidString(actor) {
		return false
	}
	for _, caracter := range actor {
		if (caracter < 'a' || caracter > 'z') &&
			(caracter < 'A' || caracter > 'Z') &&
			(caracter < '0' || caracter > '9') &&
			caracter != ':' && caracter != '-' && caracter != '_' {
			return false
		}
	}
	return true
}
