package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	delegado vecports.RegistradorAuditoriaFronteraRutaExacta
	usuarios registradorDenegacionPreferenciasUsuarios
}

func (r registradorFronterasConUsuariosPreferencias) RegistrarAuditoriaFronteraRutaExacta(ctx context.Context, orden vecports.OrdenAuditoriaFronteraRutaExacta) error {
	if orden.Validar() != nil {
		return errComposicionUsuariosPreferencias
	}
	if orden.Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias {
		if r.delegado == nil {
			return errComposicionUsuariosPreferencias
		}
		return r.delegado.RegistrarAuditoriaFronteraRutaExacta(ctx, orden)
	}
	if r.usuarios == nil {
		return errComposicionUsuariosPreferencias
	}
	return r.usuarios.RegistrarAuditoriaFronteraRutaExacta(ctx, orden)
}

func nuevaCorrelacionDenegacionPreferenciasUsuarios() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "corr_no_disponible"
	}
	return "corr_" + hex.EncodeToString(b[:])
}

// Sólo el wrapper registra fallos anteriores al despacho. El handler llama
// este mismo registrador para 401/403 del caso de uso con actor V2 acreditado.
func (a *autoridadPreferenciasUsuariosDesarrollo) registrarDenegacion(ctx context.Context, estado int, actorRef string) error {
	if a == nil || a.registrador == nil || ctx == nil || (estado != http.StatusUnauthorized && estado != http.StatusForbidden) ||
		(estado == http.StatusUnauthorized && actorRef != "") || (estado == http.StatusForbidden && actorRef == "") {
		return errComposicionUsuariosPreferencias
	}
	motivo := vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
	if estado == http.StatusForbidden {
		motivo = vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado
	}
	ctxAuditoria, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	orden := vecports.OrdenAuditoriaFronteraRutaExacta{CorrelacionRef: nuevaCorrelacionDenegacionPreferenciasUsuarios(), Motivo: motivo,
		Superficie: vecports.SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias, Ruta: usuarioshttp.RutaMisPreferencias, ActorRef: actorRef}
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
	if estado == http.StatusForbidden {
		actorRef = c.resultado.Contexto.PersonaRef
	}
	return a.registrarDenegacion(ctx, estado, actorRef)
}
