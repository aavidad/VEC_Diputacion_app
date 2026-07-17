package application

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var (
	// ErrPoliticaUsoDecisionAutorizacionInvalida indica que el caso de uso no
	// ha declarado de forma cerrada como consumira la decision del PDP.
	ErrPoliticaUsoDecisionAutorizacionInvalida = errors.New("vec: politica de uso de decision de autorizacion invalida")
	// ErrSerializacionPoliticaUsoAutorizacionProhibida impide convertir una
	// configuracion interna en un DTO de entrada o salida.
	ErrSerializacionPoliticaUsoAutorizacionProhibida = errors.New("vec: serializacion de politica de uso de autorizacion prohibida")
)

const maximoCamposEsperadosUsoAutorizacion = 512

// PerfilProteccionUsoAutorizacion selecciona las comprobaciones locales que
// el caso de uso aplica ademas de la decision exacta del PDP. El valor cero no
// tiene significado para que una ampliacion no herede un perfil por omision.
type PerfilProteccionUsoAutorizacion uint8

const (
	PerfilProteccionUsoAutorizacionNoDeclarado PerfilProteccionUsoAutorizacion = iota
	// PerfilProteccionUsoAutorizacionOrdinario admite una superficie personal
	// externa; la garantia efectiva sigue teniendo que cumplir la exigida por
	// la decision del PDP.
	PerfilProteccionUsoAutorizacionOrdinario
	// PerfilProteccionUsoAutorizacionInternoAlto exige una sesion de garantia
	// alta en una superficie corporativa o de administracion privilegiada. La
	// propia decision debe conservar tambien garantia minima alta.
	PerfilProteccionUsoAutorizacionInternoAlto
)

type datosPoliticaUsoDecisionAutorizacion struct {
	accion          string
	moduloID        string
	tipoRecurso     string
	finalidad       string
	camposEsperados []string
	perfil          PerfilProteccionUsoAutorizacion
}

// PoliticaUsoDecisionAutorizacion es una configuracion nominal cerrada e
// inmutable para consumidores externos al paquete. No es una capacidad
// criptografica: se construye deliberadamente con su fabrica, pero no
// representa datos aportados por HTTP ni admite codecs de transporte.
type PoliticaUsoDecisionAutorizacion struct {
	datos *datosPoliticaUsoDecisionAutorizacion
}

// NuevaPoliticaUsoDecisionAutorizacion fija la operacion nominal exacta que el
// caso de uso puede solicitar y el conjunto cerrado de campos que interpreta.
// Una lista vacia declara una operacion atomica sin restricciones por campo;
// no significa "cualquier campo".
func NuevaPoliticaUsoDecisionAutorizacion(
	accion string,
	moduloID string,
	tipoRecurso string,
	finalidad string,
	camposEsperados []string,
	perfil PerfilProteccionUsoAutorizacion,
) (PoliticaUsoDecisionAutorizacion, error) {
	if !identificadorPoliticaUsoAutorizacionValido(accion, 256) ||
		!identificadorPoliticaUsoAutorizacionValido(moduloID, 128) ||
		!identificadorPoliticaUsoAutorizacionValido(tipoRecurso, 128) ||
		!identificadorPoliticaUsoAutorizacionValido(finalidad, 512) ||
		!perfilProteccionUsoAutorizacionValido(perfil) ||
		!camposEsperadosUsoAutorizacionValidos(camposEsperados) {
		return PoliticaUsoDecisionAutorizacion{}, ErrPoliticaUsoDecisionAutorizacionInvalida
	}
	camposCanonicos := append([]string(nil), camposEsperados...)
	sort.Strings(camposCanonicos)
	return PoliticaUsoDecisionAutorizacion{datos: &datosPoliticaUsoDecisionAutorizacion{
		accion:          accion,
		moduloID:        moduloID,
		tipoRecurso:     tipoRecurso,
		finalidad:       finalidad,
		camposEsperados: camposCanonicos,
		perfil:          perfil,
	}}, nil
}

func (p PoliticaUsoDecisionAutorizacion) validar() error {
	if p.datos == nil ||
		!identificadorPoliticaUsoAutorizacionValido(p.datos.accion, 256) ||
		!identificadorPoliticaUsoAutorizacionValido(p.datos.moduloID, 128) ||
		!identificadorPoliticaUsoAutorizacionValido(p.datos.tipoRecurso, 128) ||
		!identificadorPoliticaUsoAutorizacionValido(p.datos.finalidad, 512) ||
		!perfilProteccionUsoAutorizacionValido(p.datos.perfil) ||
		!camposEsperadosUsoAutorizacionValidos(p.datos.camposEsperados) ||
		!sort.StringsAreSorted(p.datos.camposEsperados) {
		return ErrPoliticaUsoDecisionAutorizacionInvalida
	}
	return nil
}

func (PoliticaUsoDecisionAutorizacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionPoliticaUsoAutorizacionProhibida
}

func (*PoliticaUsoDecisionAutorizacion) UnmarshalJSON([]byte) error {
	return ErrSerializacionPoliticaUsoAutorizacionProhibida
}

func (PoliticaUsoDecisionAutorizacion) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionPoliticaUsoAutorizacionProhibida
}

func (*PoliticaUsoDecisionAutorizacion) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionPoliticaUsoAutorizacionProhibida
}

func (PoliticaUsoDecisionAutorizacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionPoliticaUsoAutorizacionProhibida
}

func (*PoliticaUsoDecisionAutorizacion) UnmarshalText([]byte) error {
	return ErrSerializacionPoliticaUsoAutorizacionProhibida
}

func (PoliticaUsoDecisionAutorizacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionPoliticaUsoAutorizacionProhibida
}

func (*PoliticaUsoDecisionAutorizacion) UnmarshalBinary([]byte) error {
	return ErrSerializacionPoliticaUsoAutorizacionProhibida
}

func (PoliticaUsoDecisionAutorizacion) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionPoliticaUsoAutorizacionProhibida
}

func (*PoliticaUsoDecisionAutorizacion) GobDecode([]byte) error {
	return ErrSerializacionPoliticaUsoAutorizacionProhibida
}

func (PoliticaUsoDecisionAutorizacion) String() string {
	return "[POLITICA-USO-AUTORIZACION-INTERNA]"
}

func (p PoliticaUsoDecisionAutorizacion) GoString() string { return p.String() }

func (p PoliticaUsoDecisionAutorizacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}

func (p PoliticaUsoDecisionAutorizacion) LogValue() slog.Value {
	return slog.StringValue(p.String())
}

// FachadaUsoDecisionAutorizacion ofrece a los modulos la unica operacion
// general necesaria: exigir una decision vinculada y convertirla en evidencia
// opaca. No expone domain.DecisionAutorizacion al llamador.
type FachadaUsoDecisionAutorizacion struct {
	autorizador ports.Autorizador
	reloj       ports.Reloj
}

// ExigidorEvidenciaUsoDecisionAutorizacion es el contrato minimo que deben
// inyectar los modulos. Depender de esta interfaz evita acoplar sus casos de
// uso al servicio concreto y permite dobles contractuales sin otro PEP.
type ExigidorEvidenciaUsoDecisionAutorizacion interface {
	ExigirEvidencia(
		ctx context.Context,
		actor domain.ContextoActor,
		vinculo domain.VinculoAutenticacionActorV1,
		recurso domain.RecursoAutorizable,
		correlacionRef string,
		motivo string,
		politica PoliticaUsoDecisionAutorizacion,
	) (ports.EvidenciaUsoDecisionAutorizacion, error)
}

// ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2 es un contrato
// distinto y no sustituible por V1. Los flujos nuevos que necesitan demostrar
// solicitud y motivo exactos deben depender expresamente de esta interfaz.
type ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2 interface {
	ExigirEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
		ctx context.Context,
		actor domain.ContextoActor,
		vinculo domain.VinculoAutenticacionActorV1,
		recurso domain.RecursoAutorizable,
		correlacionRef string,
		motivo domain.ReferenciaEntradaCatalogo,
		politica PoliticaUsoDecisionAutorizacion,
	) (ports.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2, error)
}

var _ ExigidorEvidenciaUsoDecisionAutorizacion = (*FachadaUsoDecisionAutorizacion)(nil)

// NuevaFachadaUsoDecisionAutorizacion fija el PEP y el reloj confiable en la
// composicion. Las dependencias no se seleccionan desde una peticion.
func NuevaFachadaUsoDecisionAutorizacion(
	autorizador ports.Autorizador,
	reloj ports.Reloj,
) (*FachadaUsoDecisionAutorizacion, error) {
	if dependenciaAutorizacionNula(autorizador) || dependenciaAutorizacionNula(reloj) {
		return nil, domain.ErrConfiguracionAccesoInvalida
	}
	return &FachadaUsoDecisionAutorizacion{autorizador: autorizador, reloj: reloj}, nil
}

// ExigirEvidencia ejecuta el PEP comun, comprueba la politica cerrada del caso
// de uso y devuelve solo la capacidad que el adaptador duradero puede consumir
// junto al efecto. Actor, vinculo y recurso deben proceder de fronteras
// confiables ya resueltas; esta operacion nunca completa ni normaliza entradas.
func (f *FachadaUsoDecisionAutorizacion) ExigirEvidencia(
	ctx context.Context,
	actor domain.ContextoActor,
	vinculo domain.VinculoAutenticacionActorV1,
	recurso domain.RecursoAutorizable,
	correlacionRef string,
	motivo string,
	politica PoliticaUsoDecisionAutorizacion,
) (ports.EvidenciaUsoDecisionAutorizacion, error) {
	if f == nil || dependenciaAutorizacionNula(f.autorizador) || dependenciaAutorizacionNula(f.reloj) {
		return ports.EvidenciaUsoDecisionAutorizacion{}, errorUsoDecisionAutorizacion(
			domain.ErrConfiguracionAccesoInvalida,
		)
	}
	if politica.validar() != nil {
		return ports.EvidenciaUsoDecisionAutorizacion{}, errorUsoDecisionAutorizacion(
			ErrPoliticaUsoDecisionAutorizacionInvalida,
		)
	}
	if ctx == nil {
		return ports.EvidenciaUsoDecisionAutorizacion{}, errorUsoDecisionAutorizacion(
			domain.ErrConfiguracionAccesoInvalida,
		)
	}
	if err := ctx.Err(); err != nil {
		return ports.EvidenciaUsoDecisionAutorizacion{}, errorUsoDecisionAutorizacion(err)
	}
	if err := validarPerfilProteccionUsoAutorizacion(actor, vinculo, politica); err != nil {
		return ports.EvidenciaUsoDecisionAutorizacion{}, errorUsoDecisionAutorizacion(err)
	}
	if recurso.ModuloID != politica.datos.moduloID || recurso.Tipo != politica.datos.tipoRecurso {
		return ports.EvidenciaUsoDecisionAutorizacion{}, errorUsoDecisionAutorizacion(
			domain.ErrSolicitudAutorizacionInvalida,
		)
	}

	decision, err := exigirDecisionAutorizacionVinculada(
		ctx,
		f.autorizador,
		f.reloj,
		actor,
		vinculo,
		politica.datos.accion,
		clonarRecursoUsoAutorizacion(recurso),
		politica.datos.finalidad,
		correlacionRef,
		motivo,
		usoCamposDecisionConsumidos,
	)
	if err != nil {
		return ports.EvidenciaUsoDecisionAutorizacion{}, err
	}
	if !camposDecisionCoincidenConPolitica(decision.CamposPermitidos, politica) ||
		(politica.datos.perfil == PerfilProteccionUsoAutorizacionInternoAlto &&
			decision.GarantiaMinima != domain.AuthAssuranceHigh) {
		return ports.EvidenciaUsoDecisionAutorizacion{}, errorUsoDecisionAutorizacion(
			domain.ErrDecisionAutorizacionInvalida,
		)
	}
	if err := ctx.Err(); err != nil {
		return ports.EvidenciaUsoDecisionAutorizacion{}, errorUsoDecisionAutorizacion(err)
	}

	verificadaEn := f.reloj.Ahora().UTC().Truncate(time.Microsecond)
	actorCanonico, errActor := actor.Clonar()
	if errActor != nil || verificadaEn.IsZero() ||
		!vinculo.VigenteEn(verificadaEn, actorCanonico) || !decision.VigenteEn(verificadaEn) {
		return ports.EvidenciaUsoDecisionAutorizacion{}, errorUsoDecisionAutorizacion(
			domain.ErrVinculoAutenticacionActorInvalido,
			errActor,
		)
	}
	evidencia, err := ports.NuevaEvidenciaUsoDecisionAutorizacion(decision, verificadaEn)
	if err != nil {
		return ports.EvidenciaUsoDecisionAutorizacion{}, errorUsoDecisionAutorizacion(err)
	}
	if err := ctx.Err(); err != nil {
		return ports.EvidenciaUsoDecisionAutorizacion{}, errorUsoDecisionAutorizacion(err)
	}
	return evidencia, nil
}

func validarPerfilProteccionUsoAutorizacion(
	actor domain.ContextoActor,
	vinculo domain.VinculoAutenticacionActorV1,
	politica PoliticaUsoDecisionAutorizacion,
) error {
	if politica.validar() != nil {
		return ErrPoliticaUsoDecisionAutorizacionInvalida
	}
	actorCanonico, err := actor.Clonar()
	datosVinculo, errVinculo := vinculo.Datos()
	if err != nil || errVinculo != nil || vinculo.ValidarPara(actorCanonico) != nil ||
		actorCanonico.Principal.AuthAssurance != datosVinculo.GarantiaObservada {
		return errors.Join(domain.ErrVinculoAutenticacionActorInvalido, err, errVinculo)
	}
	if politica.datos.perfil == PerfilProteccionUsoAutorizacionOrdinario {
		if datosVinculo.Superficie == domain.SuperficieAutenticacionExternaPersonalV1 {
			return nil
		}
		return domain.ErrVinculoAutenticacionActorInvalido
	}
	if actorCanonico.Principal.AuthAssurance != domain.AuthAssuranceHigh ||
		datosVinculo.GarantiaObservada != domain.AuthAssuranceHigh {
		return domain.ErrVinculoAutenticacionActorInvalido
	}
	switch datosVinculo.Superficie {
	case domain.SuperficieAutenticacionInternaCorporativaV1,
		domain.SuperficieAutenticacionAdministracionPrivilegiadaV1:
		return nil
	default:
		return domain.ErrVinculoAutenticacionActorInvalido
	}
}

func camposDecisionCoincidenConPolitica(
	camposDecision []string,
	politica PoliticaUsoDecisionAutorizacion,
) bool {
	if politica.validar() != nil || len(camposDecision) != len(politica.datos.camposEsperados) {
		return false
	}
	camposCanonicos := append([]string(nil), camposDecision...)
	sort.Strings(camposCanonicos)
	for indice, campo := range camposCanonicos {
		if campo != politica.datos.camposEsperados[indice] {
			return false
		}
	}
	return true
}

func camposEsperadosUsoAutorizacionValidos(campos []string) bool {
	if len(campos) > maximoCamposEsperadosUsoAutorizacion {
		return false
	}
	vistos := make(map[string]struct{}, len(campos))
	for _, campo := range campos {
		if len(campo) == 0 || len(campo) > 512 || campo != strings.TrimSpace(campo) ||
			strings.ContainsRune(campo, '*') {
			return false
		}
		for _, caracter := range []byte(campo) {
			if caracter < 0x21 || caracter > 0x7e {
				return false
			}
		}
		if _, repetido := vistos[campo]; repetido {
			return false
		}
		vistos[campo] = struct{}{}
	}
	return true
}

func identificadorPoliticaUsoAutorizacionValido(valor string, maximo int) bool {
	if maximo < 1 || len(valor) == 0 || len(valor) > maximo || valor != strings.TrimSpace(valor) ||
		strings.ContainsRune(valor, '*') {
		return false
	}
	for indice := 0; indice < len(valor); indice++ {
		if valor[indice] < 0x21 || valor[indice] > 0x7e {
			return false
		}
	}
	return true
}

func perfilProteccionUsoAutorizacionValido(perfil PerfilProteccionUsoAutorizacion) bool {
	return perfil == PerfilProteccionUsoAutorizacionOrdinario ||
		perfil == PerfilProteccionUsoAutorizacionInternoAlto
}

func clonarRecursoUsoAutorizacion(recurso domain.RecursoAutorizable) domain.RecursoAutorizable {
	clon := recurso
	clon.Ambitos = clonarMapaUsoAutorizacion(recurso.Ambitos)
	clon.Atributos = clonarMapaUsoAutorizacion(recurso.Atributos)
	return clon
}

func clonarMapaUsoAutorizacion(origen map[string]string) map[string]string {
	if origen == nil {
		return nil
	}
	clon := make(map[string]string, len(origen))
	for clave, valor := range origen {
		clon[clave] = valor
	}
	return clon
}

func errorUsoDecisionAutorizacion(causas ...error) error {
	argumentos := make([]error, 0, len(causas)+1)
	argumentos = append(argumentos, domain.ErrAutorizacionDenegada)
	argumentos = append(argumentos, causas...)
	return errors.Join(argumentos...)
}
