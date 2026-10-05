package mibolsa

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type ConsultorLecturaPropia interface {
	Consultar(context.Context, Orden) (bolsa.InstantaneaMiBolsa, error)
}

type ConsultorHistorialPropio interface {
	ConsultarHistorial(context.Context, Orden, int) (bolsa.PaginaHistorialMiBolsa, error)
}

// LecturasAuditadas conserva la auditoría de los accesos confirmados en su
// repositorio. Solo registra el resultado fallido cuando el servicio vuelve,
// después de que ese repositorio haya cerrado su transacción y sus reintentos.
type LecturasAuditadas struct {
	consulta    ConsultorLecturaPropia
	historial   ConsultorHistorialPropio
	registrador ports.RegistradorIntentosAuditoria
	proceso     string
}

var procesoAuditoriaLecturas = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,79}$`)

func NuevasLecturasAuditadas(consulta ConsultorLecturaPropia, historial ConsultorHistorialPropio,
	registrador ports.RegistradorIntentosAuditoria, proceso string,
) (*LecturasAuditadas, error) {
	if nula(consulta) || nula(historial) || nula(registrador) || !procesoAuditoriaLecturas.MatchString(proceso) {
		return nil, ports.ErrIntentoAuditoriaNoDisponible
	}
	return &LecturasAuditadas{consulta: consulta, historial: historial, registrador: registrador, proceso: proceso}, nil
}

func (s *LecturasAuditadas) Consultar(ctx context.Context, orden Orden) (bolsa.InstantaneaMiBolsa, error) {
	if !s.valida() || ctx == nil {
		return bolsa.InstantaneaMiBolsa{}, falloAuditoriaLectura(bolsa.ErrMaterialMiBolsaNoDisponible)
	}
	resultado, err := s.consulta.Consultar(ctx, orden)
	if err != nil {
		return bolsa.InstantaneaMiBolsa{}, s.registrarFallo(ctx, orden, bolsa.AccionConsultarMiBolsa,
			bolsa.FinalidadMiBolsa, bolsa.ErrMaterialMiBolsaNoDisponible, err)
	}
	return resultado, nil
}

func (s *LecturasAuditadas) ConsultarHistorial(ctx context.Context, orden Orden, pagina int) (bolsa.PaginaHistorialMiBolsa, error) {
	if !s.valida() || ctx == nil {
		return bolsa.PaginaHistorialMiBolsa{}, falloAuditoriaLectura(bolsa.ErrHistorialMiBolsaNoDisponible)
	}
	resultado, err := s.historial.ConsultarHistorial(ctx, orden, pagina)
	if err != nil {
		return bolsa.PaginaHistorialMiBolsa{}, s.registrarFallo(ctx, orden, bolsa.AccionConsultarHistorialPropio,
			bolsa.FinalidadHistorialMiBolsa, bolsa.ErrHistorialMiBolsaNoDisponible, err)
	}
	return resultado, nil
}

func (s *LecturasAuditadas) valida() bool {
	return s != nil && !nula(s.consulta) && !nula(s.historial) && !nula(s.registrador) && procesoAuditoriaLecturas.MatchString(s.proceso)
}

func (s *LecturasAuditadas) registrarFallo(ctx context.Context, orden Orden, accion, finalidad string, noDisponible, causa error) error {
	// Se selecciona el candidato del contexto histórico acreditado. Una
	// caducidad posterior no crea otra identidad ni renueva el permiso.
	actor, candidato, err := validarOrden(orden, orden.ResultadoContexto.Contexto.ResueltoEn)
	if err != nil {
		return falloAuditoriaLectura(noDisponible)
	}
	correlacion, err := orden.Correlacion.ValorCanonico()
	if err != nil {
		return falloAuditoriaLectura(noDisponible)
	}
	vinculo, err := orden.Vinculo.Datos()
	if err != nil {
		return falloAuditoriaLectura(noDisponible)
	}
	referencia, err := ports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return falloAuditoriaLectura(noDisponible)
	}
	intento, err := ports.NuevaOrdenIntentoAuditoria(referencia, actor, orden.Vinculo, domain.DatosIntentoAuditoria{
		Accion: accion, ModuloID: bolsa.ModuloMiBolsa, RecursoRef: recursoIntentoMiBolsa(candidato),
		FinalidadRef: finalidad, Resultado: resultadoFalloLectura(causa), Motivo: orden.Motivo,
		Proceso: s.proceso, Canal: string(vinculo.Superficie), CorrelacionRef: correlacion,
	})
	if err != nil {
		return falloAuditoriaLectura(noDisponible)
	}
	acuse, err := s.registrador.AppendIntentoAuditoria(ctx, intento)
	if err != nil || acuse.ValidarPara(intento) != nil {
		// No se propaga el error del adaptador ni una denegación previa: el
		// HTTP existente comunica indisponibilidad y nunca datos o acuse falso.
		return falloAuditoriaLectura(noDisponible)
	}
	return errorLecturaAuditada{causa: causa, acuse: acuse}
}

func falloAuditoriaLectura(noDisponible error) error {
	return errors.Join(noDisponible, ports.ErrIntentoAuditoriaNoDisponible)
}

func resultadoFalloLectura(causa error) domain.ResultadoIntentoAuditoria {
	// Los servicios pueden incluir denegación junto con una dependencia
	// indisponible. En ese caso se conserva el fallo técnico observado.
	for _, tecnico := range []error{context.Canceled, context.DeadlineExceeded,
		bolsa.ErrMaterialMiBolsaNoDisponible, bolsa.ErrHistorialMiBolsaNoDisponible,
		ports.ErrFuenteAutorizacionNoDisponible, ports.ErrRegistroDecisionNoDisponible,
		ports.ErrRegistroDenegacionNoDisponible, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible,
		ports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible} {
		if errors.Is(causa, tecnico) {
			return domain.ResultadoIntentoAuditoriaError
		}
	}
	if errors.Is(causa, domain.ErrAutorizacionDenegada) || errors.Is(causa, domain.ErrPermissionDenied) {
		return domain.ResultadoIntentoAuditoriaDenegado
	}
	return domain.ResultadoIntentoAuditoriaError
}

type errorLecturaAuditada struct {
	causa error
	acuse ports.AcuseIntentoAuditoria
}

func (errorLecturaAuditada) Error() string   { return "bolsa: lectura fallida auditada" }
func (e errorLecturaAuditada) Unwrap() error { return e.causa }

// AcuseLecturaFallida devuelve solo el acuse validado del registro confirmado.
// No representa éxito de la lectura ni entrega de datos a la persona.
func AcuseLecturaFallida(err error) (ports.AcuseIntentoAuditoria, bool) {
	var auditada errorLecturaAuditada
	if !errors.As(err, &auditada) {
		return ports.AcuseIntentoAuditoria{}, false
	}
	return auditada.acuse, true
}

// recursoIntentoMiBolsa da una referencia opaca y estable del candidato para
// la auditoría común, que sólo admite minúsculas, dígitos y «._:-». Las
// referencias de candidato son base64url y mezclan mayúsculas, así que se
// conserva su huella SHA-256: identifica el mismo candidato sin copiarlo.
func recursoIntentoMiBolsa(candidato string) string {
	suma := sha256.Sum256([]byte(candidato))
	return "mi-bolsa:candidato-sha256:" + hex.EncodeToString(suma[:])
}
