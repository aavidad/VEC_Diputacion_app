package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type registradorDenegacionPreferenciasUsuarios = vecports.RegistradorAuditoriaFronteraRutaExacta

// El despachador exacto sólo llega aquí si su propia guarda falla. El
// registrador de Usuarios no recibe las denegaciones que ya resolvió el
// wrapper o el handler; las otras superficies siguen en su autoridad previa.
type registradorFronterasConUsuariosPreferencias struct {
	delegado   vecports.RegistradorAuditoriaFronteraRutaExacta
	interna    registradorDenegacionPreferenciasUsuarios
	externa    registradorDenegacionPreferenciasUsuarios
	aspirantes registradorDenegacionPreferenciasUsuarios
}

func (r registradorFronterasConUsuariosPreferencias) RegistrarAuditoriaFronteraRutaExacta(ctx context.Context, orden vecports.OrdenAuditoriaFronteraRutaExacta) error {
	if orden.Validar() != nil {
		return errComposicionUsuariosPreferencias
	}
	if orden.Superficie == vecports.SuperficieAuditoriaFronteraRutaExactaAspirantes {
		if r.aspirantes == nil {
			return errComposicionUsuariosPreferencias
		}
		return r.aspirantes.RegistrarAuditoriaFronteraRutaExacta(ctx, orden)
	}
	if orden.Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias {
		if r.delegado == nil {
			return errComposicionUsuariosPreferencias
		}
		return r.delegado.RegistrarAuditoriaFronteraRutaExacta(ctx, orden)
	}
	// La guarda de vigencia del dispatcher puede rechazar después de capturar
	// el GET interno. Ese intento conserva identidad y correlación originales;
	// la orden de frontera no sustituye actor, motivo ni material capturados.
	if ctx != nil {
		if intento, presente := ctx.Value(claveIntentoConsultaCorreos{}).(*intentoConsultaCorreos); presente {
			if intento == nil || intento.autoridad == nil || orden.Ruta != usuarioshttp.RutaMisCorreos {
				return errComposicionUsuariosCorreos
			}
			return intento.autoridad.AuditarIntentoConsultaCorreos(ctx, http.StatusForbidden)
		}
	}
	var seleccionado registradorDenegacionPreferenciasUsuarios
	switch orden.Ruta {
	case usuarioshttp.RutaMisPreferencias, usuarioshttp.RutaMisCorreos, usuarioshttp.RutaMiImagen:
		seleccionado = r.interna
	case usuarioshttp.RutaMisPreferenciasAreaPersonal, usuarioshttp.RutaMisCorreosAreaPersonal, usuarioshttp.RutaMiImagenAreaPersonal:
		seleccionado = r.externa
	default:
		return errComposicionUsuariosPreferencias
	}
	if seleccionado == nil {
		return errComposicionUsuariosPreferencias
	}
	return seleccionado.RegistrarAuditoriaFronteraRutaExacta(ctx, orden)
}

func correlacionDenegacionPreferenciasDesde(entropia io.Reader) (string, error) {
	var b [16]byte
	if entropia == nil {
		return "", errComposicionUsuariosPreferencias
	}
	if _, err := io.ReadFull(entropia, b[:]); err != nil {
		return "", errComposicionUsuariosPreferencias
	}
	return "corr_" + hex.EncodeToString(b[:]), nil
}

func nuevaCorrelacionDenegacionPreferenciasUsuarios() (string, error) {
	return correlacionDenegacionPreferenciasDesde(rand.Reader)
}

// Sólo el wrapper registra fallos anteriores al despacho. El handler llama
// este mismo registrador para 401/403 del caso de uso con actor V2 acreditado.
func (a *autoridadPreferenciasUsuariosDesarrollo) registrarDenegacion(ctx context.Context, estado int, actorRef string) error {
	superficie := vecports.SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias
	if a != nil && a.superficieAuditoria != "" {
		superficie = a.superficieAuditoria
	}
	// Aspirantes nunca anota persona; Usuarios la exige en un 403.
	sinPersona := superficie == vecports.SuperficieAuditoriaFronteraRutaExactaAspirantes
	if a == nil || a.registrador == nil || ctx == nil || (estado != http.StatusUnauthorized && estado != http.StatusForbidden) ||
		(estado == http.StatusUnauthorized && actorRef != "") || (estado == http.StatusForbidden && (actorRef == "") != sinPersona) {
		return errComposicionUsuariosPreferencias
	}
	motivo := vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
	if estado == http.StatusForbidden {
		motivo = vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado
	}
	ctxAuditoria, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	correlacion, err := nuevaCorrelacionDenegacionPreferenciasUsuarios()
	if err != nil {
		return errComposicionUsuariosPreferencias
	}
	orden := vecports.OrdenAuditoriaFronteraRutaExacta{CorrelacionRef: correlacion, Motivo: motivo,
		Superficie: superficie, Ruta: a.ruta, ActorRef: actorRef}
	if orden.Validar() != nil {
		return errComposicionUsuariosPreferencias
	}
	return a.registrador.RegistrarAuditoriaFronteraRutaExacta(ctxAuditoria, orden)
}

func (a *autoridadPreferenciasUsuariosDesarrollo) AuditarDenegacionPreferencias(ctx context.Context, estado int) error {
	if a == nil || ctx == nil {
		return errComposicionUsuariosPreferencias
	}
	c, ok := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	if !ok || c.autoridad != a || c.resultado.Validar() != nil || c.vinculo.ValidarPara(c.resultado) != nil {
		return errComposicionUsuariosPreferencias
	}
	actorRef := ""
	if estado == http.StatusForbidden && a.superficieAuditoria != vecports.SuperficieAuditoriaFronteraRutaExactaAspirantes {
		actorRef = c.resultado.Contexto.PersonaRef
	}
	return a.registrarDenegacion(ctx, estado, actorRef)
}
