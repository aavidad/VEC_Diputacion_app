package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"maps"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/shared/plazoarranque"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// DestinoFronteraNominal procede de configuración privada. Ningún valor de
// ruta, cabecera o cuerpo puede sustituir sus tres referencias.
type DestinoFronteraNominal struct {
	Accion       string
	RecursoRef   string
	FinalidadRef string
	TipoRecurso  string
}

type ConfiguracionAuditoriaFronteraNominal struct {
	Proceso        string
	Canal          string
	MotivoDenegado domain.ReferenciaEntradaCatalogo
	MotivoError    domain.ReferenciaEntradaCatalogo
	Plazo          time.Duration
	Destinos       map[string]DestinoFronteraNominal
}

type AuditorFronteraNominal struct {
	registrador ports.RegistradorIntentosAuditoria
	config      ConfiguracionAuditoriaFronteraNominal
}

var _ api.AuditorFrontera = (*AuditorFronteraNominal)(nil)

var accionesFronteraNominal = map[string]struct{}{
	"consultar": {}, "buscar_personas": {}, "consultar_persona": {}, "consultar_recibo": {},
	"escribir": {}, "aplicar_ordinario": {}, "proponer": {}, "cerrar_propuesta": {}, "aplicar_lote_ordinario": {},
}

// accionesOpcionalesFronteraNominal pueden faltar en configuraciones previas
// al lote; si están, se validan igual. El proceso con lote las exige.
var accionesOpcionalesFronteraNominal = map[string]struct{}{"preparar_lote_ordinario": {}}

var claveFronteraNominal = regexp.MustCompile(`^[a-z][a-z0-9._:-]{1,127}$`)
var procesoFronteraNominal = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,79}$`)
var recursoFronteraNominal = regexp.MustCompile(`^administracion:[a-z0-9_:-]{1,185}$`)
var conjuntoFronteraNominal = regexp.MustCompile(`^conjunto_admin:[0-9a-f]{32}$`)
var personaFronteraNominal = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,124}$`)

// El error interno puede contener datos del proveedor. La frontera expone
// sólo el ID estable de indisponibilidad, también antes de tener actor V2.
func falloFronteraNominal(_ error) error { return ports.ErrAutoridadAdministracionPerfilesNoDisponible }

func NuevaAuditorFronteraNominal(registrador ports.RegistradorIntentosAuditoria, c ConfiguracionAuditoriaFronteraNominal) (*AuditorFronteraNominal, error) {
	if ausente(registrador) || !procesoFronteraNominal.MatchString(c.Proceso) || c.Canal != "administracion_privilegiada" || c.MotivoDenegado.Validar() != nil || c.MotivoError.Validar() != nil ||
		c.Plazo <= 0 || c.Plazo > 2*time.Second {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	claves := maps.Clone(accionesFronteraNominal)
	for clave := range accionesOpcionalesFronteraNominal {
		if _, ok := c.Destinos[clave]; ok {
			claves[clave] = struct{}{}
		}
	}
	if len(c.Destinos) != len(claves) {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	for clave := range claves {
		d, ok := c.Destinos[clave]
		if !ok || !claveFronteraNominal.MatchString(d.Accion) || !strings.HasPrefix(d.Accion, "administracion.") ||
			!claveFronteraNominal.MatchString(d.FinalidadRef) || !strings.HasPrefix(d.FinalidadRef, "gestion_") ||
			strings.Contains(d.Accion, "..") || strings.Contains(d.RecursoRef, "..") || strings.Contains(d.FinalidadRef, "..") {
			return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		switch clave {
		case "buscar_personas":
			if d.TipoRecurso != "conjunto" || !conjuntoFronteraNominal.MatchString(d.RecursoRef) {
				return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
		case "consultar_persona":
			if d.TipoRecurso != "persona" || d.RecursoRef != "" {
				return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
		default:
			if d.TipoRecurso != "fijo" || !recursoFronteraNominal.MatchString(d.RecursoRef) {
				return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
		}
	}
	// Ningún goroutine ni cambio de configuración posterior altera el mapa.
	c.Destinos = maps.Clone(c.Destinos)
	return &AuditorFronteraNominal{registrador: registrador, config: c}, nil
}

func resultadoFronteraNominal(codigo string) (domain.ResultadoIntentoAuditoria, bool) {
	switch codigo {
	case "solicitud_invalida", "metodo_no_permitido", "acceso_denegado", "autenticacion_requerida", "conflicto_estado", "recurso_no_encontrado":
		return domain.ResultadoIntentoAuditoriaDenegado, true
	case "servicio_no_disponible", "respuesta_incompatible":
		return domain.ResultadoIntentoAuditoriaError, true
	default:
		return "", false
	}
}

func (a *AuditorFronteraNominal) RegistrarDenegacionADMIN(ctx context.Context, d api.DenegacionADMIN) error {
	f := ports.ErrAutoridadAdministracionPerfilesNoDisponible
	if a == nil || ausente(a.registrador) || ctx == nil {
		return f
	}
	// Antes de la sesión V2 no existe actor acreditado para la corriente común.
	actor, err := d.Actor.Clonar()
	if err != nil {
		return falloFronteraNominal(err)
	}
	resultado, err := d.Evidencia.ResultadoContexto.Clonar()
	if err != nil {
		return falloFronteraNominal(err)
	}
	evidencia := domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: d.Evidencia.Vinculo}
	if evidencia.ValidarPara(actor) != nil {
		return f
	}
	vinculo, err := evidencia.Vinculo.Datos()
	if err != nil {
		return falloFronteraNominal(err)
	}
	if string(vinculo.Superficie) != a.config.Canal || !vinculo.CuentaPrivilegiada {
		return f
	}
	destino, ok := a.config.Destinos[d.Accion]
	if !ok {
		return f
	}
	resultadoIntento, ok := resultadoFronteraNominal(d.Codigo)
	if !ok {
		return f
	}
	// La sesión ADMIN emite esta referencia opaca por su autoridad propia; es
	// distinta de la correlación técnica de incidencias del middleware.
	ref := d.CorrelacionRef
	if !domain.ReferenciaCorrelacionAutorizacionV2Valida(ref) {
		return f
	}
	recursoRef := destino.RecursoRef
	if destino.TipoRecurso == "persona" {
		if len(d.RecursoRef) <= 128 && personaFronteraNominal.MatchString(d.RecursoRef) {
			recursoRef = d.RecursoRef
		} else {
			huella := sha256.Sum256([]byte("vec.admin.frontera.solicitud.v1\n" + ref))
			recursoRef = "solicitud_admin:" + hex.EncodeToString(huella[:16])
		}
	}
	motivo := a.config.MotivoError
	if resultadoIntento == domain.ResultadoIntentoAuditoriaDenegado {
		motivo = a.config.MotivoDenegado
	}
	datos := domain.DatosIntentoAuditoria{Accion: destino.Accion, ModuloID: "administracion", RecursoRef: recursoRef,
		FinalidadRef: destino.FinalidadRef, Resultado: resultadoIntento, Motivo: motivo, Proceso: a.config.Proceso, Canal: a.config.Canal, CorrelacionRef: ref}
	if datos.Validar() != nil {
		return f
	}
	intento, err := ports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return falloFronteraNominal(err)
	}
	orden, err := ports.NuevaOrdenIntentoAuditoria(intento, resultado, evidencia.Vinculo, datos)
	if err != nil {
		return falloFronteraNominal(err)
	}
	registroCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(a.config.Plazo))
	defer cancel()
	acuse, err := a.registrador.AppendIntentoAuditoria(registroCtx, orden)
	if errors.Is(err, ports.ErrIntentoAuditoriaNoDisponible) {
		// Reintento de la MISMA orden si el COMMIT fue ambiguo.
		acuse, err = a.registrador.AppendIntentoAuditoria(registroCtx, orden)
	}
	if err != nil {
		return falloFronteraNominal(err)
	}
	if acuse.ValidarPara(orden) != nil {
		return f
	}
	return nil
}
