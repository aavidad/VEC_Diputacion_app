package application

import (
	"encoding/hex"
	"reflect"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func prepararContenidoDecision(
	agregado dominiobolsa.BaremacionMerito,
	clase dominiobolsa.ClaseDecisionTecnica,
	propuesta dominiobolsa.PropuestaDecisionTecnica,
) (dominiobolsa.ContenidoDecisionTecnica, error) {
	switch clase {
	case dominiobolsa.ClaseDecisionInicial:
		return agregado.PrepararDecisionInicial(propuesta)
	case dominiobolsa.ClaseDecisionRectificacion:
		return agregado.PrepararRectificacion(propuesta)
	case dominiobolsa.ClaseDecisionRevocacion:
		return agregado.PrepararRevocacion(propuesta)
	case dominiobolsa.ClaseDecisionRehabilitacion:
		return agregado.PrepararRehabilitacion(propuesta)
	default:
		return dominiobolsa.ContenidoDecisionTecnica{}, ErrOrdenBaremacionInvalida
	}
}

func validarActorBaremacion(actor ActorBaremacion) error {
	if !textoAplicacionBaremacionValido(actor.Motivo, 1024, true) {
		return ErrOrdenBaremacionInvalida
	}
	return nil
}

func validarActorRevision(actor ActorBaremacion, revision RevisionBaremacionIniciada) error {
	if validarActorBaremacion(actor) != nil || validarRevisionIniciada(revision) != nil {
		return ErrOrdenBaremacionInvalida
	}
	return nil
}

func validarRevisionIniciada(r RevisionBaremacionIniciada) error {
	if r.version.Validar() != nil || !referenciaAplicacionBaremacionValida(r.principalReservaRef) ||
		!referenciaAplicacionBaremacionValida(r.perfilActorClave) || r.version.Agregado.SujetoRef != r.sujetoRef ||
		!claveAplicacionBaremacionValida(r.finalidadClave) ||
		!referenciaAplicacionBaremacionValida(r.correlacionRef) || len(r.autorizacionesRefs) != 1 ||
		validarReferenciasAutorizacionBaremacion(r.autorizacionesRefs) != nil {
		return ErrResultadoBaremacionNoConfiable
	}
	return nil
}

func validarRevisionAdoptada(r RevisionBaremacionAdoptada) error {
	accionAdopcion, conocida := puertosbolsa.AccionAdopcionParaClase(r.contenido.Clase)
	if !conocida || validarRevisionIniciada(r.revision) != nil || r.contenido.Validar() != nil ||
		r.contextoAdopcion.ValidarPara(accionAdopcion,
			puertosbolsa.ClaseRecursoBaremacion, r.revision.version.Agregado.ID) != nil ||
		r.calculo.Validar() != nil || !r.calculo.Calculo.CoincideCon(r.contenido.CalculoOficial) ||
		r.contextoAdopcion.Proyeccion().AutorizacionRef != r.contenido.AutorizacionRef ||
		r.contextoAdopcion.Proyeccion().PrincipalRef != r.contenido.DecisorRef ||
		r.contextoAdopcion.Proyeccion().PerfilActorClave != r.contenido.PerfilDecisorClave ||
		validarExtensionAutorizacionesBaremacion(
			r.revision.autorizacionesRefs, r.autorizacionesRefs, 3+2*len(r.calculo.Calculo.Evidencias),
		) != nil {
		return ErrResultadoBaremacionNoConfiable
	}
	return nil
}

func validarDecisionCodificada(d DecisionBaremacionCodificada) error {
	if validarRevisionAdoptada(d.revision) != nil || d.politica.Validar() != nil || d.codificacion.Validar() != nil ||
		d.codificacion.DecisionRef != d.revision.contenido.ID ||
		d.codificacion.HuellaContenidoSHA256 == "" ||
		validarExtensionAutorizacionesBaremacion(d.revision.autorizacionesRefs, d.autorizacionesRefs, 2) != nil {
		return ErrResultadoBaremacionNoConfiable
	}
	return nil
}

func validarDecisionCustodiada(d DecisionBaremacionCustodiada) error {
	if validarDecisionCodificada(d.decision) != nil || d.solicitud.Validar() != nil ||
		d.documento.ValidarPara(d.solicitud) != nil ||
		validarExtensionAutorizacionesBaremacion(d.decision.autorizacionesRefs, d.autorizacionesRefs, 1) != nil {
		return ErrResultadoBaremacionNoConfiable
	}
	return nil
}

func validarFirmaPreparada(f FirmaBaremacionPreparada) error {
	if validarDecisionCustodiada(f.decision) != nil || f.solicitud.Validar() != nil ||
		f.sesion.ValidarPara(f.solicitud) != nil ||
		!huellaHMACAplicacionBaremacionValida(f.seudonimoFirmado) ||
		!referenciaAplicacionBaremacionValida(f.efectoCustodiaRef) ||
		validarExtensionAutorizacionesBaremacion(f.decision.autorizacionesRefs, f.autorizacionesRefs, 1) != nil {
		return ErrResultadoBaremacionNoConfiable
	}
	return nil
}

func validarCalculoParaAgregado(c dominiobolsa.CalculoOficialBaremacion, b dominiobolsa.BaremacionMerito) error {
	if c.Validar() != nil || c.ProcesoRef != b.ProcesoRef || c.SolicitudRef != b.SolicitudRef ||
		c.SujetoRef != b.SujetoRef || c.BaremacionMeritoRef != b.ID || c.Criterio != b.Criterio {
		return ErrResultadoBaremacionNoConfiable
	}
	return nil
}

func autorizacionesRepetidas(contextos ...puertosbolsa.ContextoOperacionBaremacion) bool {
	vistas := make(map[string]struct{}, len(contextos))
	for _, contexto := range contextos {
		ref := contexto.Proyeccion().AutorizacionRef
		if ref == "" {
			return true
		}
		if _, existe := vistas[ref]; existe {
			return true
		}
		vistas[ref] = struct{}{}
	}
	return false
}

func autorizacionYaUsadaEnCustodia(c puertosbolsa.ContextoOperacionBaremacion, d DecisionBaremacionCustodiada) bool {
	ref := c.Proyeccion().AutorizacionRef
	return ref == d.decision.revision.contextoAdopcion.Proyeccion().AutorizacionRef ||
		ref == d.decision.codificacion.AutorizacionCodificacionRef ||
		ref == d.documento.AutorizacionCustodiaRef
}

func referenciasAutorizacionPrevias(f FirmaBaremacionPreparada) map[string]struct{} {
	resultado := make(map[string]struct{}, len(f.autorizacionesRefs))
	for _, referencia := range f.autorizacionesRefs {
		resultado[referencia] = struct{}{}
	}
	return resultado
}

func incorporarAutorizacionesBaremacion(
	actuales []string,
	contextos ...puertosbolsa.ContextoOperacionBaremacion,
) ([]string, error) {
	referencias := make([]string, len(contextos))
	for indice, contexto := range contextos {
		if contexto.Validar() != nil {
			return nil, ErrResultadoBaremacionNoConfiable
		}
		referencias[indice] = contexto.Proyeccion().AutorizacionRef
	}
	return incorporarReferenciasAutorizacionBaremacion(actuales, referencias...)
}

func incorporarReferenciasAutorizacionBaremacion(actuales []string, nuevas ...string) ([]string, error) {
	resultado := append([]string(nil), actuales...)
	if len(resultado) != 0 && validarReferenciasAutorizacionBaremacion(resultado) != nil {
		return nil, ErrResultadoBaremacionNoConfiable
	}
	vistas := make(map[string]struct{}, len(resultado)+len(nuevas))
	for _, referencia := range resultado {
		vistas[referencia] = struct{}{}
	}
	for _, referencia := range nuevas {
		if !referenciaAplicacionBaremacionValida(referencia) {
			return nil, ErrResultadoBaremacionNoConfiable
		}
		if _, repetida := vistas[referencia]; repetida {
			return nil, ErrResultadoBaremacionNoConfiable
		}
		vistas[referencia] = struct{}{}
		resultado = append(resultado, referencia)
	}
	return resultado, nil
}

func validarReferenciasAutorizacionBaremacion(referencias []string) error {
	if len(referencias) == 0 || len(referencias) > 4096 {
		return ErrResultadoBaremacionNoConfiable
	}
	vistas := make(map[string]struct{}, len(referencias))
	for _, referencia := range referencias {
		if !referenciaAplicacionBaremacionValida(referencia) {
			return ErrResultadoBaremacionNoConfiable
		}
		if _, repetida := vistas[referencia]; repetida {
			return ErrResultadoBaremacionNoConfiable
		}
		vistas[referencia] = struct{}{}
	}
	return nil
}

func validarExtensionAutorizacionesBaremacion(anteriores, actuales []string, incremento int) error {
	if incremento < 1 || len(actuales) != len(anteriores)+incremento ||
		validarReferenciasAutorizacionBaremacion(actuales) != nil {
		return ErrResultadoBaremacionNoConfiable
	}
	for indice := range anteriores {
		if actuales[indice] != anteriores[indice] {
			return ErrResultadoBaremacionNoConfiable
		}
	}
	return nil
}

func agregadoFinalidad(r RevisionBaremacionIniciada) string { return r.finalidadClave }

func (s *ServicioBaremacion) validarRevisionVigente(r RevisionBaremacionIniciada) error {
	if validarRevisionIniciada(r) != nil {
		return ErrResultadoBaremacionNoConfiable
	}
	return nil
}

func (s *ServicioBaremacion) ahora() (time.Time, error) {
	ahora := s.reloj.Ahora().UTC()
	if ahora.IsZero() {
		return time.Time{}, ErrResultadoBaremacionNoConfiable
	}
	return ahora, nil
}

func dependenciaBaremacionNula(dependencia any) bool {
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

func referenciaAplicacionBaremacionValida(valor string) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > 512 {
		return false
	}
	for _, caracter := range valor {
		if caracter <= 0x20 || caracter == 0x7f {
			return false
		}
	}
	return true
}

func huellaHMACAplicacionBaremacionValida(valor string) bool {
	partes := strings.Split(valor, ":")
	if len(partes) != 3 || partes[0] != "hmac-sha256" ||
		!referenciaAplicacionBaremacionValida(partes[1]) || len(partes[2]) != 64 ||
		partes[2] != strings.ToLower(partes[2]) {
		return false
	}
	decodificada, err := hex.DecodeString(partes[2])
	return err == nil && len(decodificada) == 32
}

func selloGeneradoBaremacionValido(valor string) bool {
	return valor != hmacBaremacionPendiente && huellaHMACAplicacionBaremacionValida(valor)
}

func claveAplicacionBaremacionValida(valor string) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > 128 {
		return false
	}
	for indice, caracter := range valor {
		valido := caracter >= 'a' && caracter <= 'z' || caracter >= '0' && caracter <= '9' ||
			(indice > 0 && (caracter == '.' || caracter == '_' || caracter == '-'))
		if !valido {
			return false
		}
	}
	return true
}

func huellaAplicacionBaremacionValida(valor string) bool {
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

func textoAplicacionBaremacionValido(valor string, maximo int, permiteEspacios bool) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > maximo {
		return false
	}
	for _, caracter := range valor {
		if caracter < 0x20 || caracter == 0x7f || (!permiteEspacios && caracter == ' ') {
			return false
		}
	}
	return true
}
