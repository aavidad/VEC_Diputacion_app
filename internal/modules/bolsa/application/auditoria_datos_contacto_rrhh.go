package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// OperadorDatosContactoRRHH es el servicio de datos de contacto existente.
type OperadorDatosContactoRRHH interface {
	Registrar(context.Context, ports.SolicitudRegistrarDatosContactoParticipacion) (ports.RegistroDatosContactoParticipacion, error)
	Consultar(context.Context, ports.SolicitudConsultarDatosContactoParticipacion) (ports.DatosContactoParticipacionLeidos, error)
}

// DatosContactoRRHHAuditados deja en la auditoría común (AD169) los intentos
// fallidos de la consulta completa: denegaciones y errores. La consulta que
// sale bien ya dejó su asiento al consumir la decisión V3 en su transacción.
// El registro y la consulta enmascarada se delegan sin cambios.
type DatosContactoRRHHAuditados struct {
	operador    OperadorDatosContactoRRHH
	registrador vecports.RegistradorIntentosAuditoria
	proceso     string
}

func NuevosDatosContactoRRHHAuditados(operador OperadorDatosContactoRRHH,
	registrador vecports.RegistradorIntentosAuditoria, proceso string,
) (*DatosContactoRRHHAuditados, error) {
	if dependenciaLlamamientoNula(operador) || dependenciaLlamamientoNula(registrador) || !procesoAuditoriaDocumentales.MatchString(proceso) {
		return nil, ErrRegistroDatosContactoParticipacionNoDisponible
	}
	return &DatosContactoRRHHAuditados{operador: operador, registrador: registrador, proceso: proceso}, nil
}

// Registrar es una escritura con su propia autorización e idempotencia.
func (s *DatosContactoRRHHAuditados) Registrar(ctx context.Context,
	q ports.SolicitudRegistrarDatosContactoParticipacion,
) (ports.RegistroDatosContactoParticipacion, error) {
	if s == nil || dependenciaLlamamientoNula(s.operador) {
		return ports.RegistroDatosContactoParticipacion{}, ErrRegistroDatosContactoParticipacionNoDisponible
	}
	return s.operador.Registrar(ctx, q)
}

func (s *DatosContactoRRHHAuditados) Consultar(ctx context.Context,
	q ports.SolicitudConsultarDatosContactoParticipacion,
) (ports.DatosContactoParticipacionLeidos, error) {
	vacia := ports.DatosContactoParticipacionLeidos{}
	if ctx == nil || s == nil || dependenciaLlamamientoNula(s.operador) {
		return vacia, falloAuditoriaConsultaDatosContacto()
	}
	if !q.Completo {
		return s.operador.Consultar(ctx, q)
	}
	if dependenciaLlamamientoNula(s.registrador) || !procesoAuditoriaDocumentales.MatchString(s.proceso) || q.ValidarCompleta() != nil {
		return vacia, falloAuditoriaConsultaDatosContacto()
	}
	actor, err := q.ResultadoContexto.Clonar()
	vinculo, errVinculo := q.Vinculo.Datos()
	correlacion, errCorrelacion := q.Correlacion.ValorCanonico()
	if err != nil || errVinculo != nil || errCorrelacion != nil || vinculo.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 {
		return vacia, falloAuditoriaConsultaDatosContacto()
	}
	// El actor se captura antes de leer: una revocación o desconexión
	// posterior no sustituye la identidad de este intento.
	leidos, causa := s.operador.Consultar(ctx, q)
	if ctx.Err() != nil {
		causa = ctx.Err()
	} else if causa == nil && (leidos.ParticipacionRef != q.ParticipacionRef || leidos.AuditoriaRef == "" || leidos.DecisionRef == "") {
		// Una lectura completa sin su acuse de consumo no se entrega.
		causa = ErrRegistroDatosContactoParticipacionNoDisponible
	}
	if causa == nil {
		return leidos, nil
	}
	referencia, err := vecports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return vacia, falloAuditoriaConsultaDatosContacto()
	}
	orden, err := vecports.NuevaOrdenIntentoAuditoria(referencia, actor, q.Vinculo, core.DatosIntentoAuditoria{
		Accion: ports.AccionConsultarDatosContactoParticipacion, ModuloID: ports.ModuloSituacionParticipacion,
		RecursoRef: q.ParticipacionRef, FinalidadRef: ports.FinalidadConsultarDatosContactoParticipacion,
		Resultado: resultadoFalloConsultaDatosContacto(causa), Motivo: q.MotivoAutorizacion,
		Proceso: s.proceso, Canal: string(vinculo.Superficie), CorrelacionRef: correlacion,
	})
	if err != nil {
		return vacia, falloAuditoriaConsultaDatosContacto()
	}
	// El registro AD169 usa su propia transacción y no depende del plazo HTTP.
	acuse, err := s.registrador.AppendIntentoAuditoria(context.WithoutCancel(ctx), orden)
	if err != nil || acuse.ValidarPara(orden) != nil {
		return vacia, falloAuditoriaConsultaDatosContacto()
	}
	return vacia, falloConsultaDatosContactoAuditado{causa: causa, acuse: acuse}
}

func falloAuditoriaConsultaDatosContacto() error {
	return errors.Join(ErrRegistroDatosContactoParticipacionNoDisponible, vecports.ErrIntentoAuditoriaNoDisponible)
}

// Solo la denegación es «denegado»; la falta de datos (la lectura se revierte
// sin consumo) y cualquier fallo técnico quedan como «error».
func resultadoFalloConsultaDatosContacto(causa error) core.ResultadoIntentoAuditoria {
	if errors.Is(causa, ErrRegistroDatosContactoParticipacionNoDisponible) ||
		errors.Is(causa, ports.ErrDatosContactoParticipacionNoDisponibles) ||
		errors.Is(causa, ports.ErrDatosContactoParticipacionNoEncontrados) {
		return core.ResultadoIntentoAuditoriaError
	}
	return resultadoFalloConsultaDocumentales(causa)
}

type falloConsultaDatosContactoAuditado struct {
	causa error
	acuse vecports.AcuseIntentoAuditoria
}

func (falloConsultaDatosContactoAuditado) Error() string {
	return "bolsa: consulta de datos de contacto fallida auditada"
}
func (e falloConsultaDatosContactoAuditado) Unwrap() error { return e.causa }

// AcuseConsultaDatosContactoFallida devuelve el acuse ya validado de un
// intento fallido para que el transporte publique sus referencias.
func AcuseConsultaDatosContactoFallida(err error) (vecports.AcuseIntentoAuditoria, bool) {
	var auditado falloConsultaDatosContactoAuditado
	if !errors.As(err, &auditado) {
		return vecports.AcuseIntentoAuditoria{}, false
	}
	return auditado.acuse, true
}
