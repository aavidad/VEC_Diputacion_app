package bootstrap

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	smtpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/smtp"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
	"vec-diputacion-granada/internal/shared/i18n"
)

// Claves del catálogo i18n del servidor (locales/<idioma>.json). Los textos
// no viven en el código: aquí sólo se sustituyen {codigo} y {minutos}.
const (
	claveAsuntoCodigoCorreo = "usuarios.correos.verificacion.asunto"
	claveCuerpoCodigoCorreo = "usuarios.correos.verificacion.cuerpo"
	claveAsuntoAvisoCorreo  = "usuarios.correos.aviso_cambio.asunto"
	claveCuerpoAvisoCorreo  = "usuarios.correos.aviso_cambio.cuerpo"
)

var patronEnvioCorreoRef = regexp.MustCompile(`^correo_envio:([0-9a-f]{32})$`)

type traductorCorreosPropios interface {
	Message(locale, key string) (string, bool)
}

// transporteCorreosPropiosDesarrollo redacta el mensaje desde el catálogo y
// lo entrega al relay SMTP configurado. Sólo informa aceptación del relay.
type transporteCorreosPropiosDesarrollo struct {
	smtp      enviadorCorreoLlamamientoDesarrollo
	textos    traductorCorreosPropios
	dominioID string
	ahora     func() time.Time
}

func nuevoTransporteCorreosPropiosDesarrollo(smtp enviadorCorreoLlamamientoDesarrollo, textos traductorCorreosPropios, remitente string) (*transporteCorreosPropiosDesarrollo, error) {
	dominio := dominioRemitente(remitente)
	if smtp == nil || textos == nil || dominio == "" {
		return nil, errComposicionUsuariosCorreos
	}
	for _, clave := range []string{claveAsuntoCodigoCorreo, claveCuerpoCodigoCorreo, claveAsuntoAvisoCorreo, claveCuerpoAvisoCorreo} {
		if texto, ok := textos.Message(i18n.DefaultLocale, clave); !ok || strings.TrimSpace(texto) == "" {
			return nil, errComposicionUsuariosCorreos
		}
	}
	if cuerpo, _ := textos.Message(i18n.DefaultLocale, claveCuerpoCodigoCorreo); !strings.Contains(cuerpo, "{codigo}") {
		return nil, errComposicionUsuariosCorreos
	}
	return &transporteCorreosPropiosDesarrollo{smtp: smtp, textos: textos, dominioID: dominio, ahora: time.Now}, nil
}

// codigoLegible agrupa el código en dos bloques de cuatro para leerlo mejor.
func codigoLegible(codigo string) string {
	if len(codigo) != 8 {
		return codigo
	}
	return codigo[:4] + " " + codigo[4:]
}

func (t *transporteCorreosPropiosDesarrollo) EnviarCorreoPropio(ctx context.Context, m usuariosports.MensajeCorreoPropio) bool {
	if t == nil || t.smtp == nil || t.textos == nil || ctx == nil {
		return false
	}
	partes := patronEnvioCorreoRef.FindStringSubmatch(m.EnvioRef)
	if partes == nil || m.Destino == "" {
		return false
	}
	claveAsunto, claveCuerpo := claveAsuntoAvisoCorreo, claveCuerpoAvisoCorreo
	if m.Tipo == usuariosports.TipoEnvioCodigo {
		claveAsunto, claveCuerpo = claveAsuntoCodigoCorreo, claveCuerpoCodigoCorreo
	} else if m.Tipo != usuariosports.TipoEnvioAviso {
		return false
	}
	asunto, okA := t.textos.Message(i18n.DefaultLocale, claveAsunto)
	cuerpo, okC := t.textos.Message(i18n.DefaultLocale, claveCuerpo)
	if !okA || !okC {
		return false
	}
	ahora := t.ahora().UTC().Truncate(time.Second)
	if m.Tipo == usuariosports.TipoEnvioCodigo {
		minutos := int(m.VenceUTC.Sub(ahora).Minutes())
		if len(m.Codigo) != 8 || minutos < 1 {
			return false
		}
		cuerpo = strings.NewReplacer("{codigo}", codigoLegible(m.Codigo), "{minutos}", strconv.Itoa(minutos)).Replace(cuerpo)
	}
	resultado := t.smtp.Enviar(ctx, smtpct.Mensaje{Destino: m.Destino, Asunto: asunto, Cuerpo: cuerpo,
		MessageID: "<correo-" + partes[1] + "@" + t.dominioID + ">", FechaOrigen: ahora})
	return resultado.Estado == smtpct.AceptadoPorRelay
}
