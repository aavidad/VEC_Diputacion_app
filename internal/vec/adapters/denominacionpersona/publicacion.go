package denominacionpersona

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ConfiguracionIntentosPublicacion procede de la composición privada. El plazo
// permite registrar el resultado observado aunque la petición se cancele.
type ConfiguracionIntentosPublicacion struct {
	Proceso, Canal              string
	MotivoDenegado, MotivoError domain.ReferenciaEntradaCatalogo
	PlazoAuditoria              time.Duration
}

// Publicador conserva la autoridad durable detrás de su puerto. La confirmación
// permitida pertenece a la transacción de ese registro; sólo añade intentos que
// terminaron sin un recibo confirmado, sin afirmar que un COMMIT incierto falló.
type Publicador struct {
	registro      ports.RegistroDenominacionPersona
	intentos      ports.RegistradorIntentosAuditoria
	configuracion ConfiguracionIntentosPublicacion
}

var _ ports.RegistroDenominacionPersona = (*Publicador)(nil)

func NuevoPublicador(registro ports.RegistroDenominacionPersona, intentos ports.RegistradorIntentosAuditoria, configuracion ConfiguracionIntentosPublicacion) (*Publicador, error) {
	if nulo(registro) || nulo(intentos) || configuracion.Proceso == "" || configuracion.Canal == "" || configuracion.PlazoAuditoria <= 0 || configuracion.MotivoDenegado.Validar() != nil || configuracion.MotivoError.Validar() != nil {
		return nil, ErrNoDisponible
	}
	return &Publicador{registro: registro, intentos: intentos, configuracion: configuracion}, nil
}

func (p *Publicador) datosIntento(a ports.AccesoDenominacionPersona, persona string, resultado domain.ResultadoIntentoAuditoria) domain.DatosIntentoAuditoria {
	motivo := p.configuracion.MotivoError
	if resultado == domain.ResultadoIntentoAuditoriaDenegado {
		motivo = p.configuracion.MotivoDenegado
	}
	return domain.DatosIntentoAuditoria{Accion: ports.AccionPublicarDenominacionPersona, ModuloID: "vec", RecursoRef: persona, FinalidadRef: "presentacion_persona", Resultado: resultado, Motivo: motivo, Proceso: p.configuracion.Proceso, Canal: p.configuracion.Canal, CorrelacionRef: a.Auditoria.CorrelationRef}
}

func (p *Publicador) evidenciaValida(a ports.AccesoDenominacionPersona, persona string) bool {
	h, err := a.Contexto.HuellaSHA256VinculadaV2()
	v, errV := a.Vinculo.Datos()
	return domain.ReferenciaPersonaDenominacionValida(persona) && len(persona) <= 128 && err == nil && errV == nil && a.ResultadoContexto.Validar() == nil && a.ResultadoContexto.HuellaSHA256 == h && a.Vinculo.ValidarPara(a.ResultadoContexto) == nil && string(v.Superficie) == p.configuracion.Canal && p.datosIntento(a, persona, domain.ResultadoIntentoAuditoriaError).Validar() == nil && p.datosIntento(a, persona, domain.ResultadoIntentoAuditoriaDenegado).Validar() == nil
}

func (p *Publicador) PublicarDenominacionPersona(ctx context.Context, orden ports.OrdenDenominacionPersona) (ports.ReciboDenominacionPersona, error) {
	if p == nil || ctx == nil || !p.evidenciaValida(orden.Acceso, orden.Preparacion.PersonaRef) {
		return ports.ReciboDenominacionPersona{}, ErrNoDisponible
	}
	// La autoridad recibe copias; el intento conserva evidencia y recurso originales.
	original := clonarAcceso(orden.Acceso)
	orden.Acceso = clonarAcceso(original)
	orden.Preparacion.Sobre = clonarSobre(orden.Preparacion.Sobre)
	persona, procedencia, version, huella := orden.Preparacion.PersonaRef, orden.Preparacion.ProcedenciaRef, orden.Preparacion.Sobre.Version, orden.Preparacion.SobreSHA256
	var recibo ports.ReciboDenominacionPersona
	err := ErrNoDisponible
	// El recurso nominal es Persona, igual que en CA32. Una entrada divergente
	// no llega al registro, pero su error observado conserva ese recurso propio.
	if orden.Acceso.PersonaRef == persona && orden.Acceso.Recurso.Referencia == persona && orden.Acceso.Recurso.ModuloID == "vec" && orden.Acceso.Recurso.Tipo == "persona_denominacion" && orden.Acceso.FinalidadRef == "presentacion_persona" {
		recibo, err = p.registro.PublicarDenominacionPersona(ctx, orden)
	}
	if err == nil && recibo.PersonaRef == persona && recibo.ProcedenciaRef == procedencia && recibo.Version == version && recibo.Version > 0 && recibo.SobreSHA256 == huella && len(huella) == 64 && recibo.AuditoriaRef != "" {
		return recibo, nil
	}
	resultado := domain.ResultadoIntentoAuditoriaError
	if errors.Is(err, domain.ErrAutorizacionDenegada) {
		resultado = domain.ResultadoIntentoAuditoriaDenegado
	}
	ref, fallo := ports.NuevaReferenciaIntentoAuditoria()
	if fallo != nil {
		return ports.ReciboDenominacionPersona{}, ErrNoDisponible
	}
	intento, fallo := ports.NuevaOrdenIntentoAuditoria(ref, original.ResultadoContexto, original.Vinculo, p.datosIntento(original, persona, resultado))
	if fallo != nil {
		return ports.ReciboDenominacionPersona{}, ErrNoDisponible
	}
	// La operación durable ya retornó y cerró su transacción. El intento usa un
	// plazo propio, conserva valores del contexto y nunca reinicia la publicación.
	auditCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), p.configuracion.PlazoAuditoria)
	defer cancelar()
	acuse, fallo := p.intentos.AppendIntentoAuditoria(auditCtx, intento)
	if errors.Is(fallo, ports.ErrIntentoAuditoriaNoDisponible) {
		acuse, fallo = p.intentos.AppendIntentoAuditoria(auditCtx, intento)
	}
	if fallo != nil || acuse.ValidarPara(intento) != nil {
		return ports.ReciboDenominacionPersona{}, ErrNoDisponible
	}
	if resultado == domain.ResultadoIntentoAuditoriaDenegado {
		return ports.ReciboDenominacionPersona{}, domain.ErrAutorizacionDenegada
	}
	return ports.ReciboDenominacionPersona{}, ErrNoDisponible
}
