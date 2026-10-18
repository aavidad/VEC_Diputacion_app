package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"vec-diputacion-granada/config"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	smtpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/smtp"
)

type tokensAvisoOferta struct {
	material *derivadorIdentidadOperacionDesarrollo
}

func (d tokensAvisoOferta) DerivarTokenAvisoOferta(ctx context.Context, ref, version string) ([]byte, string, error) {
	if ctx == nil || ctx.Err() != nil || d.material == nil || !d.material.valido() || !strings.HasPrefix(ref, "oferta:") || len(ref) > 256 {
		return nil, "", ports.ErrOfertaNoDisponible
	}
	preimagen, err := json.Marshal([]string{"bolsa.aviso-oferta.v1", ref})
	if err != nil {
		return nil, "", ports.ErrOfertaNoDisponible
	}
	defer clear(preimagen)
	resultados, err := d.material.calcularHMAC(preimagen, preimagen)
	if err != nil {
		return nil, "", ports.ErrOfertaNoDisponible
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultados)
	for _, r := range resultados {
		v := fmt.Sprintf("bolsa.aviso-oferta.v1:g%d", r.generacion)
		if version == "" || version == v {
			return append([]byte(nil), r.localizador[:]...), v, nil
		}
	}
	return nil, "", ports.ErrOfertaNoDisponible
}

type emisorAvisoOferta struct {
	smtp enviadorCorreoLlamamientoDesarrollo
}

func (e emisorAvisoOferta) EnviarAvisoOferta(ctx context.Context, m ports.MensajeAvisoOferta) string {
	if e.smtp == nil {
		return ports.AvisoOfertaTransitorio
	}
	r := e.smtp.Enviar(ctx, smtpct.Mensaje{Destino: m.Destino, Asunto: m.Asunto, Cuerpo: m.Cuerpo, MessageID: m.MessageID, FechaOrigen: m.Fecha})
	switch r.Estado {
	case smtpct.AceptadoPorRelay:
		return ports.AvisoOfertaAceptado
	case smtpct.NoAceptadoTransitorio:
		return ports.AvisoOfertaTransitorio
	case smtpct.NoAceptadoPermanente:
		return ports.AvisoOfertaPermanente
	default:
		return ports.AvisoOfertaIndeterminado
	}
}

func componerDespachadorAvisoOferta(r ports.RepositorioAvisosOferta, c ports.FuenteCorreoParticipacion, e enviadorCorreoLlamamientoDesarrollo) *bolsa.DespachadorAvisosOferta {
	enlace := strings.TrimSpace(os.Getenv(config.EnvBolsaAvisoOfertaEnlace))
	u, err := url.Parse(enlace)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || !strings.HasPrefix(u.Path, "/area-personal/") || len(enlace) > 2048 {
		slog.Warn("aviso_oferta_configuracion_pendiente", "variable", config.EnvBolsaAvisoOfertaEnlace)
		return nil
	}
	var catalogo struct {
		Version int    `json:"version"`
		Idioma  string `json:"idioma_defecto"`
		Textos  map[string]struct {
			Asunto string `json:"asunto"`
			Cuerpo string `json:"cuerpo"`
		} `json:"textos"`
	}
	if json.Unmarshal(config.CatalogoAvisoOfertaBolsa(), &catalogo) != nil || catalogo.Version < 1 {
		slog.Warn("aviso_oferta_catalogo_invalido")
		return nil
	}
	t := catalogo.Textos[catalogo.Idioma]
	d, err := bolsa.NuevoDespachadorAvisosOferta(r, c, emisorAvisoOferta{e}, t.Asunto, t.Cuerpo, enlace)
	if err != nil {
		slog.Warn("aviso_oferta_catalogo_invalido")
		return nil
	}
	return d
}
