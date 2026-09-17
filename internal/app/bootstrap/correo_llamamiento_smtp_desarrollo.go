package bootstrap

import (
	"crypto/x509"
	"errors"
	"os"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/smtp"
)

func nuevoEnviadorCorreoLlamamientoDesarrollo(cfg config.Config) (enviadorCorreoLlamamientoDesarrollo, error) {
	if strings.TrimSpace(cfg.SMTPHost) == "" {
		return nil, nil
	}
	modo := smtp.STARTTLSObligatorio
	switch strings.ToLower(strings.TrimSpace(cfg.SMTPModoTLS)) {
	case "starttls":
	case "implicito":
		modo = smtp.TLSImplicito
	default:
		return nil, errors.New("modo TLS SMTP no valido")
	}
	if cfg.SMTPPort < 1 || cfg.SMTPPort > 65535 || strings.TrimSpace(cfg.SMTPFrom) == "" || strings.TrimSpace(cfg.SMTPCAFile) == "" {
		return nil, errors.New("configuracion SMTP incompleta")
	}
	ca, err := os.ReadFile(cfg.SMTPCAFile)
	if err != nil {
		return nil, err
	}
	raices := x509.NewCertPool()
	if !raices.AppendCertsFromPEM(ca) {
		return nil, errors.New("CA SMTP invalida")
	}
	adaptador, err := smtp.Nuevo(smtp.Configuracion{Host: strings.TrimSpace(cfg.SMTPHost), Puerto: uint16(cfg.SMTPPort), ServerName: strings.TrimSpace(cfg.SMTPHost), CertificadosCA: raices, RemitenteFijo: strings.TrimSpace(cfg.SMTPFrom), ModoTLS: modo, ModoAutenticacion: smtp.ModoAutenticacionNinguna, TiempoMaximo: 10 * time.Second})
	if err != nil {
		return nil, err
	}
	return adaptador, nil
}
