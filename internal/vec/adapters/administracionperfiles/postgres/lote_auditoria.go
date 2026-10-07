package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ConfiguracionAuditoriaLote procede exclusivamente de configuración privada.
// Los motivos son referencias al catálogo común, nunca texto de una petición.
type ConfiguracionAuditoriaLote struct {
	Proceso        string
	Canal          string
	MotivoDenegado domain.ReferenciaEntradaCatalogo
	MotivoError    domain.ReferenciaEntradaCatalogo
	Plazo          time.Duration
}

func (c ConfiguracionAuditoriaLote) validar() error {
	if !procesoFronteraNominal.MatchString(c.Proceso) || c.Canal != "administracion_privilegiada" ||
		c.MotivoDenegado.Validar() != nil || c.MotivoError.Validar() != nil ||
		c.Plazo <= 0 || c.Plazo > 2*time.Second {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return nil
}

// AplicarLoteOrdinario registra todo fallo con sesión V2 original válida. El
// intento común se crea cuando la transacción del efecto ya terminó o su
// COMMIT quedó indeterminado. Un éxito tiene auditoría nominal propia AD190.
func (a *AutoridadLoteOrdinario) AplicarLoteOrdinario(ctx context.Context, s domain.SolicitudLoteAdministracionPerfiles) (recibo domain.ReciboLoteAdministracionPerfiles, err error) {
	if a == nil || ctx == nil || ausente(a.registrador) || a.auditoria.validar() != nil {
		return recibo, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	recibo, err = a.aplicarLoteOrdinario(ctx, s)
	if err == nil {
		return recibo, nil
	}
	return domain.ReciboLoteAdministracionPerfiles{}, a.finalizarFalloLote(ctx, s, err)
}

func (a *AutoridadLoteOrdinario) finalizarFalloLote(ctx context.Context, s domain.SolicitudLoteAdministracionPerfiles, err error) error {
	// Ni la falta de V2 ni una correlación inválida autorizan inventar actor,
	// familia o referencia. Esa frontera técnica pertenece al consumidor HTTP.
	if a.registrarFalloLote(ctx, s, err) != nil || errors.Is(err, errCommitLoteIndeterminado) {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return err
}

func resultadoFalloLote(err error) domain.ResultadoIntentoAuditoria {
	if errors.Is(err, domain.ErrActoAdministracionPerfilesInvalido) ||
		errors.Is(err, domain.ErrControlAdministracionPerfilesInvalido) ||
		errors.Is(err, domain.ErrAutorizacionDenegada) {
		return domain.ResultadoIntentoAuditoriaDenegado
	}
	return domain.ResultadoIntentoAuditoriaError
}

func (a *AutoridadLoteOrdinario) registrarFalloLote(ctx context.Context, s domain.SolicitudLoteAdministracionPerfiles, fallo error) error {
	recursoRef := ""
	if len(s.Cambios) > 0 {
		recursoRef = s.Cambios[0].Objetivo.PersonaRef
		for _, cambio := range s.Cambios[1:] {
			if cambio.Objetivo.PersonaRef != recursoRef {
				recursoRef = ""
				break
			}
		}
	}
	return a.registrarFalloConsumo(ctx, accionLoteOrdinario, s.Actor, s.Evidencia, s.CorrelacionRef, recursoRef, fallo)
}

// registrarFalloConsumo deja el intento común (denegado o error) de una orden
// del lote o de su preparación, cada una con su acción. Sin persona
// destinataria válida usa una referencia derivada de la correlación, nunca un
// dato de la petición.
func (a *AutoridadLoteOrdinario) registrarFalloConsumo(ctx context.Context, accion string, actorOrigen domain.ContextoActor,
	evidenciaOrigen domain.EvidenciaSesionAdministracionPerfiles, correlacion, recursoRef string, fallo error) error {
	if a == nil || ctx == nil || ausente(a.registrador) || a.auditoria.validar() != nil ||
		!domain.ReferenciaCorrelacionAutorizacionV2Valida(correlacion) {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	actor, err := actorOrigen.Clonar()
	if err != nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	resultado, err := evidenciaOrigen.ResultadoContexto.Clonar()
	if err != nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	evidencia := domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: evidenciaOrigen.Vinculo}
	if evidencia.ValidarPara(actor) != nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	vinculo, err := evidencia.Vinculo.Datos()
	if err != nil || string(vinculo.Superficie) != a.auditoria.Canal || !vinculo.CuentaPrivilegiada {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if len(recursoRef) > 128 || !personaFronteraNominal.MatchString(recursoRef) {
		huella := sha256.Sum256([]byte("vec.admin.lote.solicitud.v1\n" + correlacion))
		recursoRef = "solicitud_admin:" + hex.EncodeToString(huella[:16])
	}
	clase := resultadoFalloLote(fallo)
	motivo := a.auditoria.MotivoError
	if clase == domain.ResultadoIntentoAuditoriaDenegado {
		motivo = a.auditoria.MotivoDenegado
	}
	datos := domain.DatosIntentoAuditoria{Accion: accion, ModuloID: "administracion",
		RecursoRef: recursoRef, FinalidadRef: "gestion_perfiles", Resultado: clase, Motivo: motivo,
		Proceso: a.auditoria.Proceso, Canal: a.auditoria.Canal, CorrelacionRef: correlacion}
	if datos.Validar() != nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	intento, err := ports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	orden, err := ports.NuevaOrdenIntentoAuditoria(intento, resultado, evidencia.Vinculo, datos)
	if err != nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	registroCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(a.auditoria.Plazo))
	defer cancel()
	acuse, err := a.registrador.AppendIntentoAuditoria(registroCtx, orden)
	if err != nil || acuse.ValidarPara(orden) != nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return nil
}
