// Package memory ofrece adaptadores efimeros y exclusivos para pruebas.
package memory

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const maximoPropuestasLlamamientoMemoria = 65_536

var _ puertosbolsa.TransaccionPropuestasLlamamiento = (*RegistroPropuestasLlamamiento)(nil)

type propuestaLlamamientoGuardada struct {
	propuesta dominiobolsa.PropuestaLlamamiento
	evidencia puertosvec.DatosEvidenciaUsoDecisionAutorizacion
}

// claveNegocioPropuestaLlamamiento identifica la version inmutable de la
// necesidad. No incluye referencias generadas, actor ni correlacion: cambiar
// esos datos no autoriza un segundo efecto para la misma version gobernada.
type claveNegocioPropuestaLlamamiento struct {
	necesidadRef          string
	versionNecesidad      uint64
	huellaNecesidadSHA256 string
}

// RegistroPropuestasLlamamiento simula una unica transaccion bajo un mutex. No
// es un repositorio productivo: no aporta durabilidad, outbox, hora de base de
// datos, bloqueo de la necesidad ni verificacion criptografica de atestacion.
type RegistroPropuestasLlamamiento struct {
	mu sync.RWMutex

	reloj puertosbolsa.RelojLlamamientos

	propuestas            map[string]propuestaLlamamientoGuardada
	propuestaPorUso       map[string]string
	propuestaPorNecesidad map[claveNegocioPropuestaLlamamiento]string
	duenoReferencia       map[string]string
}

// PerfilUsoRegistroPropuestasMemoria es una capacidad deliberadamente opaca.
// Evita cablear por accidente este adaptador efimero en una composicion real.
type PerfilUsoRegistroPropuestasMemoria struct{ soloPruebas bool }

func NuevoRegistroPropuestasLlamamiento(
	reloj puertosbolsa.RelojLlamamientos,
	perfil PerfilUsoRegistroPropuestasMemoria,
) (*RegistroPropuestasLlamamiento, error) {
	if interfazLlamamientoNula(reloj) || !perfil.soloPruebas {
		return nil, puertosbolsa.ErrSolicitudPropuestaLlamamientoInvalida
	}
	ahora, err := ahoraLlamamientoCanonico(reloj)
	if err != nil || ahora.IsZero() {
		return nil, puertosbolsa.ErrSolicitudPropuestaLlamamientoInvalida
	}
	return &RegistroPropuestasLlamamiento{
		reloj:                 reloj,
		propuestas:            make(map[string]propuestaLlamamientoGuardada),
		propuestaPorUso:       make(map[string]string),
		propuestaPorNecesidad: make(map[claveNegocioPropuestaLlamamiento]string),
		duenoReferencia:       make(map[string]string),
	}, nil
}

func (r *RegistroPropuestasLlamamiento) GuardarPropuestaLlamamiento(
	ctx context.Context,
	propuesta dominiobolsa.PropuestaLlamamiento,
	evidencia puertosvec.EvidenciaUsoDecisionAutorizacion,
) error {
	if ctx == nil {
		return errorPersistenciaLlamamiento(puertosbolsa.ErrSolicitudPropuestaLlamamientoInvalida)
	}
	if err := ctx.Err(); err != nil {
		return errorPersistenciaLlamamiento(err)
	}
	if r == nil || interfazLlamamientoNula(r.reloj) {
		return errorPersistenciaLlamamiento(puertosbolsa.ErrPersistenciaPropuestaNoDisponible)
	}
	propuestaCanonica, err := propuesta.ClonarCanonica()
	if err != nil {
		return errorPersistenciaLlamamiento(err)
	}
	datosEvidencia, err := evidencia.Datos()
	if err != nil || !vinculoAutorizacionPropuestaValido(propuestaCanonica, datosEvidencia) {
		return errorPersistenciaLlamamiento(err)
	}
	referencias, err := referenciasUnicasPropuesta(propuestaCanonica, datosEvidencia.Decision.DecisionRef)
	if err != nil {
		return errorPersistenciaLlamamiento(err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return errorPersistenciaLlamamiento(err)
	}
	ahora, err := ahoraLlamamientoCanonico(r.reloj)
	if err != nil || propuestaCanonica.GeneradaEn.After(ahora) || evidencia.ValidarEn(ahora) != nil {
		return errorPersistenciaLlamamiento(puertosvec.ErrEvidenciaUsoDecisionAutorizacionInvalida)
	}
	if existente, existe := r.propuestas[propuestaCanonica.PropuestaRef]; existe {
		if reflect.DeepEqual(existente.propuesta, propuestaCanonica) &&
			reflect.DeepEqual(existente.evidencia, datosEvidencia) {
			return nil
		}
		return errorPersistenciaLlamamiento(puertosbolsa.ErrPropuestaLlamamientoYaExiste)
	}
	if propuestaRef, consumida := r.propuestaPorUso[datosEvidencia.Decision.DecisionRef]; consumida {
		return errorPersistenciaLlamamiento(errors.Join(
			puertosbolsa.ErrDecisionAutorizacionLlamamientoUsada,
			puertosvec.ErrDecisionAutorizacionConsumida,
			fmtReferenciaOmitida(propuestaRef),
		))
	}
	for _, referencia := range referencias {
		if _, usada := r.duenoReferencia[referencia]; usada {
			return errorPersistenciaLlamamiento(puertosbolsa.ErrReferenciaLlamamientoYaUtilizada)
		}
	}
	claveNegocio := claveNegocioDePropuesta(propuestaCanonica)
	if _, yaPropuesta := r.propuestaPorNecesidad[claveNegocio]; yaPropuesta {
		return errorPersistenciaLlamamiento(puertosbolsa.ErrNecesidadLlamamientoYaPropuesta)
	}
	if len(r.propuestas) >= maximoPropuestasLlamamientoMemoria {
		return errorPersistenciaLlamamiento(puertosbolsa.ErrCapacidadMemoriaLlamamientosAgotada)
	}
	if err := ctx.Err(); err != nil {
		return errorPersistenciaLlamamiento(err)
	}
	// Desde este punto no hay llamadas externas ni caminos de error: las cuatro
	// escrituras representan un unico COMMIT logico en memoria.
	r.propuestas[propuestaCanonica.PropuestaRef] = propuestaLlamamientoGuardada{
		propuesta: propuestaCanonica,
		evidencia: clonarDatosEvidencia(datosEvidencia),
	}
	r.propuestaPorUso[datosEvidencia.Decision.DecisionRef] = propuestaCanonica.PropuestaRef
	r.propuestaPorNecesidad[claveNegocio] = propuestaCanonica.PropuestaRef
	for _, referencia := range referencias {
		r.duenoReferencia[referencia] = propuestaCanonica.PropuestaRef
	}
	return nil
}

func claveNegocioDePropuesta(propuesta dominiobolsa.PropuestaLlamamiento) claveNegocioPropuestaLlamamiento {
	return claveNegocioPropuestaLlamamiento{
		necesidadRef:          propuesta.NecesidadRef,
		versionNecesidad:      propuesta.VersionNecesidad,
		huellaNecesidadSHA256: propuesta.HuellaNecesidadSHA256,
	}
}

// ObtenerPropuestaParaPruebas no forma parte del puerto productivo. Devuelve
// una copia profunda para comprobar aislamiento en tests de adaptadores.
func (r *RegistroPropuestasLlamamiento) ObtenerPropuestaParaPruebas(
	ctx context.Context,
	referencia string,
) (dominiobolsa.PropuestaLlamamiento, error) {
	if ctx == nil || r == nil || !puertosbolsa.ReferenciaOpacaLlamamientoValida(referencia) {
		return dominiobolsa.PropuestaLlamamiento{}, puertosbolsa.ErrDatosLlamamientoNoEncontrados
	}
	if err := ctx.Err(); err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	guardada, existe := r.propuestas[referencia]
	if !existe {
		return dominiobolsa.PropuestaLlamamiento{}, puertosbolsa.ErrDatosLlamamientoNoEncontrados
	}
	return guardada.propuesta.ClonarCanonica()
}

func (r *RegistroPropuestasLlamamiento) NumeroPropuestasParaPruebas() int {
	if r == nil {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.propuestas)
}

func vinculoAutorizacionPropuestaValido(
	propuesta dominiobolsa.PropuestaLlamamiento,
	evidencia puertosvec.DatosEvidenciaUsoDecisionAutorizacion,
) bool {
	decision := evidencia.Decision
	return propuesta.Validar() == nil && decision.ValidarEvidenciaInstantanea() == nil && decision.Concedida &&
		decision.Accion == puertosbolsa.AccionProponerLlamamiento &&
		decision.RecursoRef == propuesta.NecesidadRef && decision.ModuloID == puertosbolsa.ModuloLlamamientos &&
		decision.TipoRecurso == puertosbolsa.TipoRecursoNecesidad &&
		decision.Finalidad == puertosbolsa.FinalidadProponerLlamamiento &&
		len(decision.CamposPermitidos) == 0 && len(decision.Obligaciones) == 0 &&
		evidencia.VerificadaEn.Equal(propuesta.GeneradaEn) && decision.VigenteEn(propuesta.GeneradaEn)
}

func referenciasUnicasPropuesta(
	propuesta dominiobolsa.PropuestaLlamamiento,
	decisionRef string,
) ([]string, error) {
	referencias := make([]string, 0, 3+len(propuesta.Evaluaciones)*2)
	vistas := make(map[string]struct{}, cap(referencias))
	agregar := func(referencia string) bool {
		if !puertosbolsa.ReferenciaOpacaLlamamientoValida(referencia) {
			return false
		}
		if _, existe := vistas[referencia]; existe {
			return false
		}
		vistas[referencia] = struct{}{}
		referencias = append(referencias, referencia)
		return true
	}
	if !agregar(propuesta.PropuestaRef) || !agregar(propuesta.InstantaneaRef) || !agregar(decisionRef) {
		return nil, puertosbolsa.ErrReferenciaLlamamientoYaUtilizada
	}
	for _, evaluacion := range propuesta.Evaluaciones {
		if !agregar(evaluacion.EntradaEvaluacionRef) || !agregar(evaluacion.ResultadoEvaluacionRef) {
			return nil, puertosbolsa.ErrReferenciaLlamamientoYaUtilizada
		}
	}
	return referencias, nil
}

func clonarDatosEvidencia(
	datos puertosvec.DatosEvidenciaUsoDecisionAutorizacion,
) puertosvec.DatosEvidenciaUsoDecisionAutorizacion {
	decision := datos.Decision
	decision.PoliticasEvaluadasRefs = clonarCadenasLlamamiento(decision.PoliticasEvaluadasRefs)
	decision.PoliticasEvaluadasHuellasSHA256 = clonarMapaLlamamiento(decision.PoliticasEvaluadasHuellasSHA256)
	decision.PoliticasRefs = clonarCadenasLlamamiento(decision.PoliticasRefs)
	decision.PoliticasHuellasSHA256 = clonarMapaLlamamiento(decision.PoliticasHuellasSHA256)
	decision.CamposPermitidos = clonarCadenasLlamamiento(decision.CamposPermitidos)
	decision.Obligaciones = clonarCadenasLlamamiento(decision.Obligaciones)
	datos.Decision = decision
	return datos
}

func clonarCadenasLlamamiento(origen []string) []string {
	if origen == nil {
		return nil
	}
	clon := make([]string, len(origen))
	copy(clon, origen)
	return clon
}

func clonarMapaLlamamiento(origen map[string]string) map[string]string {
	if origen == nil {
		return nil
	}
	clon := make(map[string]string, len(origen))
	for clave, valor := range origen {
		clon[clave] = valor
	}
	return clon
}

func ahoraLlamamientoCanonico(reloj puertosbolsa.RelojLlamamientos) (time.Time, error) {
	if interfazLlamamientoNula(reloj) {
		return time.Time{}, puertosbolsa.ErrSolicitudPropuestaLlamamientoInvalida
	}
	instante := reloj.Ahora()
	if instante.IsZero() || instante.Year() < 1 || instante.Year() > 9999 {
		return time.Time{}, puertosbolsa.ErrSolicitudPropuestaLlamamientoInvalida
	}
	return instante.UTC().Truncate(time.Microsecond), nil
}

func errorPersistenciaLlamamiento(causa error) error {
	if causa == nil {
		return puertosbolsa.ErrPersistenciaPropuestaNoDisponible
	}
	return errors.Join(dominiovec.ErrAutorizacionDenegada, puertosbolsa.ErrPersistenciaPropuestaNoDisponible, causa)
}

// fmtReferenciaOmitida conserva una rama de error no nula sin filtrar la
// referencia del efecto que ya consumio la decision.
func fmtReferenciaOmitida(string) error {
	return errors.New("bolsa: decision consumida por otro efecto")
}

func interfazLlamamientoNula(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}
