// Package smtp proporciona un transporte SMTP saliente para comunicaciones de
// contratación temporal. El resultado acredita únicamente la aceptación por
// el relay; no acredita entrega, lectura, plazo ni eficacia administrativa.
package smtp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/mail"
	stdsmtp "net/smtp"
	"net/textproto"
	"strings"
	"sync"
	"time"
)

var ErrConfiguracionInvalida = errors.New("contratacion temporal smtp: configuracion invalida")
var ErrMensajeInvalido = errors.New("contratacion temporal smtp: mensaje invalido")

// Estado describe lo que el cliente puede afirmar del mensaje. AceptadoPorRelay
// no equivale a entrega al destinatario ni proporciona exactamente una vez.
type Estado uint8

const (
	NoAceptadoTransitorio Estado = iota + 1
	NoAceptadoPermanente
	Indeterminado
	AceptadoPorRelay
)

type Resultado struct{ Estado Estado }

// Mensaje es efímero: el adaptador no lo conserva ni lo registra. MessageID
// debe ser estable y lo suministra el llamador para la conciliación posterior.
type Mensaje struct {
	Destino   string
	Asunto    string
	Cuerpo    string
	MessageID string
	// FechaOrigen la aporta el llamador y se conserva estable en reintentos;
	// el transporte nunca usa time.Now para construir el encabezado Date.
	FechaOrigen time.Time
}

func (Mensaje) String() string   { return "smtp.Mensaje{redactado}" }
func (Mensaje) GoString() string { return "smtp.Mensaje{redactado}" }
func (Mensaje) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Redactado bool `json:"redactado"`
	}{true})
}

type ModoTLS uint8

const (
	TLSImplicito ModoTLS = iota + 1
	STARTTLSObligatorio
)

// ModoAutenticacion selecciona de forma explícita el mecanismo SMTP. El
// transporte no deduce nunca el mecanismo de la presencia de credenciales.
type ModoAutenticacion string

const (
	ModoAutenticacionNinguna ModoAutenticacion = "ninguna"
	ModoAutenticacionPlain   ModoAutenticacion = "plain"
	ModoAutenticacionXOAUTH2 ModoAutenticacion = "xoauth2"
)

// Configuracion contiene solo parámetros de transporte. Secreto se recibe en
// memoria; nunca se serializa, registra ni se incluye en errores.
type Configuracion struct {
	Host              string
	Puerto            uint16
	ServerName        string
	CertificadosCA    *x509.CertPool
	RemitenteFijo     string
	ModoTLS           ModoTLS
	Usuario           string
	Secreto           []byte
	ModoAutenticacion ModoAutenticacion
	// ModoOAuth se conserva para configuraciones anteriores al selector
	// explícito. Solo tiene efecto si ModoAutenticacion es cero.
	ModoOAuth    bool
	TiempoMaximo time.Duration
}

func (Configuracion) String() string   { return "smtp.Configuracion{redactada}" }
func (Configuracion) GoString() string { return "smtp.Configuracion{redactada}" }
func (Configuracion) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Redactado bool `json:"redactado"`
	}{true})
}

type Adaptador struct{ configuracion Configuracion }

func (Adaptador) String() string   { return "smtp.Adaptador{redactado}" }
func (Adaptador) GoString() string { return "smtp.Adaptador{redactado}" }
func (Adaptador) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Redactado bool `json:"redactado"`
	}{true})
}

func Nuevo(c Configuracion) (*Adaptador, error) {
	modo, ok := normalizarModoAutenticacion(c)
	if c.Host == "" || c.Puerto == 0 || c.ServerName == "" || c.CertificadosCA == nil ||
		c.RemitenteFijo == "" || (c.ModoTLS != TLSImplicito && c.ModoTLS != STARTTLSObligatorio) ||
		c.TiempoMaximo <= 0 || contieneCRLF(c.Host) || contieneCRLF(c.ServerName) || contieneCRLF(c.RemitenteFijo) ||
		!ok ||
		(modo != ModoAutenticacionNinguna && (c.Usuario == "" || len(c.Secreto) == 0)) ||
		(modo == ModoAutenticacionNinguna && (c.Usuario != "" || len(c.Secreto) != 0)) {
		return nil, ErrConfiguracionInvalida
	}
	remitente, ok := direccionSobre(c.RemitenteFijo)
	if !ok {
		return nil, ErrConfiguracionInvalida
	}
	c.RemitenteFijo = remitente
	c.ModoAutenticacion = modo
	c.Secreto = append([]byte(nil), c.Secreto...)
	c.CertificadosCA = c.CertificadosCA.Clone()
	return &Adaptador{configuracion: c}, nil
}

func normalizarModoAutenticacion(c Configuracion) (ModoAutenticacion, bool) {
	switch c.ModoAutenticacion {
	case "":
		if c.ModoOAuth {
			return ModoAutenticacionXOAUTH2, true
		}
		return ModoAutenticacionNinguna, true
	case ModoAutenticacionNinguna:
		return ModoAutenticacionNinguna, !c.ModoOAuth
	case ModoAutenticacionPlain:
		return ModoAutenticacionPlain, !c.ModoOAuth
	case ModoAutenticacionXOAUTH2:
		// false también representa el valor cero de configuraciones nuevas;
		// true es la representación heredada equivalente, no una selección
		// alternativa.
		return ModoAutenticacionXOAUTH2, true
	default:
		return "", false
	}
}

func (a *Adaptador) Enviar(ctx context.Context, m Mensaje) Resultado {
	if a == nil || ctx == nil {
		return Resultado{Estado: NoAceptadoPermanente}
	}
	destino, ok := direccionSobre(m.Destino)
	if !ok || !mensajeValido(m) {
		return Resultado{Estado: NoAceptadoPermanente}
	}
	m.Destino = destino
	if ctx.Err() != nil {
		return Resultado{Estado: NoAceptadoTransitorio}
	}
	cfg := a.configuracion
	op, cancelar := context.WithTimeout(ctx, cfg.TiempoMaximo)
	defer cancelar()

	conn, err := (&net.Dialer{}).DialContext(op, "tcp", net.JoinHostPort(cfg.Host, fmt.Sprint(cfg.Puerto)))
	if err != nil {
		return Resultado{Estado: NoAceptadoTransitorio}
	}
	defer conn.Close()
	terminar := cerrarAlCancelar(op, conn)
	defer terminar()
	if limite, ok := op.Deadline(); ok {
		_ = conn.SetDeadline(limite)
	}

	if cfg.ModoTLS == TLSImplicito {
		tlsConn := tls.Client(conn, configuracionTLS(cfg))
		if err := tlsConn.HandshakeContext(op); err != nil {
			return Resultado{Estado: NoAceptadoTransitorio}
		}
		conn = tlsConn
	}
	cliente, err := stdsmtp.NewClient(conn, cfg.ServerName)
	if err != nil {
		return Resultado{Estado: NoAceptadoTransitorio}
	}
	if cfg.ModoTLS == STARTTLSObligatorio {
		if err := cliente.Hello("localhost"); err != nil {
			return resultadoAntesDeData(err)
		}
		ok, _ := cliente.Extension("STARTTLS")
		if !ok {
			return Resultado{Estado: NoAceptadoPermanente}
		}
		if err := cliente.StartTLS(configuracionTLS(cfg)); err != nil {
			return Resultado{Estado: NoAceptadoTransitorio}
		}
	}
	if cfg.ModoAutenticacion == ModoAutenticacionXOAUTH2 {
		if err := cliente.Auth(&xoauth2Auth{usuario: cfg.Usuario, secreto: cfg.Secreto}); err != nil {
			return resultadoAntesDeData(err)
		}
	}
	if cfg.ModoAutenticacion == ModoAutenticacionPlain {
		// Este punto sólo se alcanza después del handshake implícito o de
		// STARTTLS; nunca se transmite una credencial sobre la sesión inicial.
		if err := cliente.Auth(stdsmtp.PlainAuth("", cfg.Usuario, string(cfg.Secreto), cfg.ServerName)); err != nil {
			return resultadoAntesDeData(err)
		}
	}
	if err := cliente.Mail(cfg.RemitenteFijo); err != nil {
		return resultadoAntesDeData(err)
	}
	if err := cliente.Rcpt(m.Destino); err != nil {
		return resultadoAntesDeData(err)
	}
	w, err := cliente.Data()
	if err != nil {
		return resultadoAntesDeData(err)
	}
	material := construirMIME(cfg.RemitenteFijo, m)
	if _, err := io.WriteString(w, material); err != nil {
		// No cerrar w: Close emitiría el terminador DATA tras un fallo local.
		return Resultado{Estado: Indeterminado}
	}
	if err := w.Close(); err != nil {
		return resultadoTrasData(err)
	}
	// Una aceptación final 250 ya hace responsable al relay. QUIT solo libera
	// la sesión; su fallo no revoca esa aceptación.
	_ = cliente.Quit()
	return Resultado{Estado: AceptadoPorRelay}
}

func configuracionTLS(c Configuracion) *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12, ServerName: c.ServerName, RootCAs: c.CertificadosCA}
}

func cerrarAlCancelar(ctx context.Context, conn net.Conn) func() {
	done := make(chan struct{})
	var once sync.Once
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()
	return func() { once.Do(func() { close(done) }) }
}

func resultadoAntesDeData(err error) Resultado {
	if codigo, ok := codigoSMTP(err); ok && codigo >= 500 && codigo <= 599 {
		return Resultado{Estado: NoAceptadoPermanente}
	}
	return Resultado{Estado: NoAceptadoTransitorio}
}

func resultadoTrasData(err error) Resultado {
	if codigo, ok := codigoSMTP(err); ok {
		if codigo >= 500 && codigo <= 599 {
			return Resultado{Estado: NoAceptadoPermanente}
		}
		if codigo >= 400 && codigo <= 499 {
			return Resultado{Estado: NoAceptadoTransitorio}
		}
	}
	return Resultado{Estado: Indeterminado}
}

func codigoSMTP(err error) (int, bool) {
	var e *textproto.Error
	if errors.As(err, &e) {
		return e.Code, true
	}
	return 0, false
}

func mensajeValido(m Mensaje) bool {
	if m.Destino == "" || m.Asunto == "" || m.FechaOrigen.IsZero() || !messageIDValido(m.MessageID) || contieneCRLF(m.Destino) || contieneCRLF(m.Asunto) {
		return false
	}
	_, ok := direccionSobre(m.Destino)
	return ok
}

func direccionSobre(valor string) (string, bool) {
	if valor == "" || contieneCRLF(valor) {
		return "", false
	}
	direccion, err := mail.ParseAddress(valor)
	if err != nil || !direccionDotAtomValida(direccion.Address) {
		return "", false
	}
	return direccion.Address, true
}

func direccionDotAtomValida(valor string) bool {
	if len(valor) > 254 || strings.Count(valor, "@") != 1 {
		return false
	}
	local, dominio, _ := strings.Cut(valor, "@")
	if !dotAtomValido(local, true) || !dominioValido(dominio) {
		return false
	}
	return true
}

func dotAtomValido(valor string, local bool) bool {
	if valor == "" || valor[0] == '.' || valor[len(valor)-1] == '.' || strings.Contains(valor, "..") {
		return false
	}
	for _, r := range valor {
		permitido := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.'
		if local {
			permitido = permitido || strings.ContainsRune("!#$%&'*+-/=?^_`{|}~", r)
		}
		if !permitido {
			return false
		}
	}
	return true
}

func dominioValido(dominio string) bool {
	if dominio == "" || len(dominio) > 253 || strings.HasPrefix(dominio, ".") || strings.HasSuffix(dominio, ".") || strings.Contains(dominio, "..") {
		return false
	}
	for _, etiqueta := range strings.Split(dominio, ".") {
		if len(etiqueta) == 0 || len(etiqueta) > 63 || etiqueta[0] == '-' || etiqueta[len(etiqueta)-1] == '-' {
			return false
		}
		for _, r := range etiqueta {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}
	return true
}

func messageIDValido(valor string) bool {
	if len(valor) < 5 || len(valor) > 254 || valor[0] != '<' || valor[len(valor)-1] != '>' {
		return false
	}
	medio := valor[1 : len(valor)-1]
	if strings.Count(medio, "@") != 1 || strings.HasPrefix(medio, "@") || strings.HasSuffix(medio, "@") {
		return false
	}
	for _, r := range medio {
		if r < 33 || r > 126 || r == '<' || r == '>' {
			return false
		}
	}
	return true
}

func contieneCRLF(s string) bool { return strings.ContainsAny(s, "\r\n") }

func construirMIME(remitente string, m Mensaje) string {
	cuerpo := base64.StdEncoding.EncodeToString([]byte(m.Cuerpo))
	var lineas []string
	for len(cuerpo) > 76 {
		lineas, cuerpo = append(lineas, cuerpo[:76]), cuerpo[76:]
	}
	lineas = append(lineas, cuerpo)
	return "From: " + remitente + "\r\n" + "To: " + m.Destino + "\r\n" +
		"Subject: " + mime.QEncoding.Encode("UTF-8", m.Asunto) + "\r\n" +
		"Date: " + m.FechaOrigen.UTC().Format(time.RFC1123Z) + "\r\n" +
		"Message-ID: " + m.MessageID + "\r\n" + "MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n" +
		strings.Join(lineas, "\r\n") + "\r\n"
}

type xoauth2Auth struct {
	usuario      string
	secreto      []byte
	dummyEnviado bool
}

func (a *xoauth2Auth) Start(*stdsmtp.ServerInfo) (string, []byte, error) {
	return "XOAUTH2", []byte("user=" + a.usuario + "\x01auth=Bearer " + string(a.secreto) + "\x01\x01"), nil
}
func (a *xoauth2Auth) Next(_ []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	if a.dummyEnviado {
		return nil, errors.New("smtp oauth exchange rejected")
	}
	a.dummyEnviado = true
	// RFC 7628 exige una respuesta ficticia tras el 334 de error; nunca vuelve
	// a transmitir el bearer token en una continuación.
	return []byte{}, nil
}
