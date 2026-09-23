package bootstrap

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Una generación anterior permite cotejar el mismo correo después de rotar
// la clave HMAC, mientras siga conservada en el gestor de desarrollo.
type huellasContactoDesarrollo struct {
	derivador *derivadorIdentidadOperacionDesarrollo
}

func (h huellasContactoDesarrollo) DerivarHuellasContactoUsuario(ctx context.Context, sujeto string, version uint64, correo string) ([]ports.HuellaSolicitudContactoUsuario, error) {
	if ctx == nil || ctx.Err() != nil || h.derivador == nil || !h.derivador.valido() || !domain.ReferenciaSujetoContactoUsuarioValida(sujeto) || version > 1<<53-1 || correo == "" || len(correo) > 254 {
		return nil, errContactoPropioDesarrolloNoDisponible
	}
	localizador, e := json.Marshal(struct {
		Esquema, SujetoRef string
		VersionEsperada    uint64
	}{"vec.contacto_usuario.replay-localizador.v1", sujeto, version})
	if e != nil {
		return nil, errContactoPropioDesarrolloNoDisponible
	}
	defer borrarBytes(localizador)
	huella, e := json.Marshal(struct {
		Esquema, SujetoRef, Correo string
		VersionEsperada            uint64
	}{"vec.contacto_usuario.replay-huella.v1", sujeto, correo, version})
	if e != nil {
		return nil, errContactoPropioDesarrolloNoDisponible
	}
	defer borrarBytes(huella)
	resultados, e := h.derivador.calcularHMAC(localizador, huella)
	if e != nil || len(resultados) == 0 || len(resultados) > 4 {
		return nil, errContactoPropioDesarrolloNoDisponible
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultados)
	salida := make([]ports.HuellaSolicitudContactoUsuario, 0, len(resultados))
	for _, r := range resultados {
		salida = append(salida, ports.HuellaSolicitudContactoUsuario{ClaveRef: fmt.Sprintf("clave:contacto-usuario:hmac:g%d", r.generacion), ValorHMACSHA256: hex.EncodeToString(r.huellaSolicitud[:])})
	}
	return salida, nil
}

var _ ports.DerivadorHuellasContactoUsuario = huellasContactoDesarrollo{}
