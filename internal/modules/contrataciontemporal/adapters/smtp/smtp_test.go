package smtp

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"
)

func TestEnviarTLSImplicitoCapturaMIME(t *testing.T) {
	cert, roots := certificado(t, "smtp.prueba.local")
	captura := make(chan string, 1)
	direccion, cerrar := servidor(t, cert, true, false, func(r *bufio.Reader, w *bufio.Writer) {
		conversacionSMTP(t, r, w, captura, false, false)
	})
	defer cerrar()
	a := nuevoParaPrueba(t, direccion, roots, TLSImplicito)
	r := a.Enviar(context.Background(), mensajePrueba())
	if r.Estado != AceptadoPorRelay {
		t.Fatalf("estado=%v", r.Estado)
	}
	got := <-captura
	for _, fragmento := range []string{"Message-ID: <estable@prueba.local>\r\n", "Subject: ", "Content-Transfer-Encoding: base64", "Y3VlcnBvIMOh"} {
		if !strings.Contains(got, fragmento) {
			t.Fatalf("falta %q en captura", fragmento)
		}
	}
	correo, err := mail.ReadMessage(strings.NewReader(got))
	if err != nil {
		t.Fatal(err)
	}
	fecha, err := correo.Header.Date()
	if err != nil {
		t.Fatal(err)
	}
	if !fecha.Equal(mensajePrueba().FechaOrigen) {
		t.Fatalf("Date=%s", fecha)
	}
}

func TestEnviarSTARTTLSObligatorioYAuth(t *testing.T) {
	cert, roots := certificado(t, "smtp.prueba.local")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		r, w := bufio.NewReader(c), bufio.NewWriter(c)
		_, _ = w.WriteString("220 prueba\r\n")
		_ = w.Flush()
		_, _ = r.ReadString('\n')
		_, _ = w.WriteString("250-prueba\r\n250-STARTTLS\r\n250 AUTH XOAUTH2\r\n")
		_ = w.Flush()
		_, _ = r.ReadString('\n')
		_, _ = w.WriteString("220 siga\r\n")
		_ = w.Flush()
		tlsConn := tls.Server(c, &tls.Config{Certificates: []tls.Certificate{cert}})
		if err := tlsConn.Handshake(); err != nil {
			t.Errorf("handshake: %v", err)
			return
		}
		r, w = bufio.NewReader(tlsConn), bufio.NewWriter(tlsConn)
		_, _ = r.ReadString('\n')
		_, _ = w.WriteString("250-prueba\r\n250 AUTH XOAUTH2\r\n")
		_ = w.Flush()
		if got, _ := r.ReadString('\n'); !strings.HasPrefix(got, "AUTH XOAUTH2 ") {
			t.Errorf("auth %q", got)
		}
		_, _ = w.WriteString("235 ok\r\n")
		_ = w.Flush()
		transaccionSMTP(t, r, w, nil, false)
	}()
	a := nuevoParaPrueba(t, ln.Addr().String(), roots, STARTTLSObligatorio)
	a.configuracion.ModoAutenticacion, a.configuracion.ModoOAuth, a.configuracion.Usuario, a.configuracion.Secreto = ModoAutenticacionXOAUTH2, true, "sintetico", []byte("secreto-sintetico")
	if r := a.Enviar(context.Background(), mensajePrueba()); r.Estado != AceptadoPorRelay {
		t.Fatalf("estado=%v", r.Estado)
	}
}

func TestEnviarSTARTTLSObligatorioAuthPlainSoloTrasTLS(t *testing.T) {
	cert, roots := certificado(t, "smtp.prueba.local")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		r, w := bufio.NewReader(c), bufio.NewWriter(c)
		escribir := func(s string) { _, _ = w.WriteString(s); _ = w.Flush() }
		escribir("220 prueba\r\n")
		_, _ = r.ReadString('\n')
		// Un relay no fiable puede anunciar PLAIN antes de TLS. El cliente debe
		// elegir STARTTLS y no enviar credenciales en esa fase.
		escribir("250-prueba\r\n250-STARTTLS\r\n250 AUTH PLAIN\r\n")
		if orden, _ := r.ReadString('\n'); !strings.HasPrefix(orden, "STARTTLS") {
			t.Errorf("orden previa a TLS=%q", orden)
			return
		}
		escribir("220 siga\r\n")
		tlsConn := tls.Server(c, &tls.Config{Certificates: []tls.Certificate{cert}})
		if err := tlsConn.Handshake(); err != nil {
			t.Errorf("handshake: %v", err)
			return
		}
		r, w = bufio.NewReader(tlsConn), bufio.NewWriter(tlsConn)
		_, _ = r.ReadString('\n')
		escribir("250-prueba\r\n250 AUTH PLAIN\r\n")
		orden, _ := r.ReadString('\n')
		const prefijo = "AUTH PLAIN "
		if !strings.HasPrefix(orden, prefijo) {
			t.Errorf("auth=%q", orden)
			return
		}
		material, err := base64.StdEncoding.DecodeString(strings.TrimSpace(strings.TrimPrefix(orden, prefijo)))
		if err != nil || string(material) != "\x00usuario-prueba\x00secreto-prueba" {
			t.Error("AUTH PLAIN no contiene la credencial esperada")
			return
		}
		escribir("235 ok\r\n")
		transaccionSMTP(t, r, w, nil, false)
	}()
	a := nuevoParaPrueba(t, ln.Addr().String(), roots, STARTTLSObligatorio)
	a.configuracion.ModoAutenticacion, a.configuracion.Usuario, a.configuracion.Secreto = ModoAutenticacionPlain, "usuario-prueba", []byte("secreto-prueba")
	if r := a.Enviar(context.Background(), mensajePrueba()); r.Estado != AceptadoPorRelay {
		t.Fatalf("estado=%v", r.Estado)
	}
}

func TestNuevoNormalizaModoAutenticacionYRechazaAmbiguedad(t *testing.T) {
	_, roots := certificado(t, "smtp.prueba.local")
	base := Configuracion{Host: "smtp.prueba.local", Puerto: 465, ServerName: "smtp.prueba.local", CertificadosCA: roots, RemitenteFijo: "rrhh@prueba.local", ModoTLS: TLSImplicito, TiempoMaximo: time.Second}
	pruebas := []struct {
		nombre   string
		modo     ModoAutenticacion
		oauth    bool
		usuario  string
		secreto  []byte
		valido   bool
		esperado ModoAutenticacion
	}{
		{"legacy-anonimo", "", false, "", nil, true, ModoAutenticacionNinguna},
		{"legacy-xoauth2", "", true, "usuario", []byte("secreto"), true, ModoAutenticacionXOAUTH2},
		{"plain-explicito", ModoAutenticacionPlain, false, "usuario", []byte("secreto"), true, ModoAutenticacionPlain},
		{"plain-oauth-ambiguo", ModoAutenticacionPlain, true, "usuario", []byte("secreto"), false, ""},
		{"ninguna-oauth-ambiguo", ModoAutenticacionNinguna, true, "", nil, false, ""},
		{"desconocido", ModoAutenticacion("cram-md5"), false, "usuario", []byte("secreto"), false, ""},
		{"plain-sin-credencial", ModoAutenticacionPlain, false, "", nil, false, ""},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			cfg := base
			cfg.ModoAutenticacion, cfg.ModoOAuth, cfg.Usuario, cfg.Secreto = prueba.modo, prueba.oauth, prueba.usuario, prueba.secreto
			a, err := Nuevo(cfg)
			if prueba.valido != (err == nil) {
				t.Fatalf("err=%v", err)
			}
			if prueba.valido && a.configuracion.ModoAutenticacion != prueba.esperado {
				t.Fatalf("modo=%q", a.configuracion.ModoAutenticacion)
			}
		})
	}
}

func TestRepresentacionesRedactadasYNormalizaSobre(t *testing.T) {
	cert, roots := certificado(t, "smtp.prueba.local")
	_ = cert
	c := Configuracion{Host: "smtp.privado", Puerto: 465, ServerName: "smtp.privado", CertificadosCA: roots, RemitenteFijo: "Nombre <rrhh@prueba.local>", ModoTLS: TLSImplicito, Usuario: "persona@prueba.local", Secreto: []byte("secreto-sintetico"), ModoOAuth: true, TiempoMaximo: time.Second}
	a, err := Nuevo(c)
	if err != nil {
		t.Fatal(err)
	}
	if a.configuracion.CertificadosCA == roots {
		t.Fatal("el pool CA conserva alias del llamador")
	}
	m := Mensaje{Destino: "Nombre <destino@prueba.local>", Asunto: "contenido secreto", Cuerpo: "cuerpo secreto", MessageID: "<estable@prueba.local>"}
	for _, valor := range []any{c, m, a, *a} {
		texto := fmt.Sprintf("%v|%+v|%#v", valor, valor, valor)
		if strings.Contains(texto, "secreto") || strings.Contains(texto, "@prueba.local") {
			t.Fatalf("representacion filtrada: %q", texto)
		}
		jsonValor, err := json.Marshal(valor)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(jsonValor), "secreto") || strings.Contains(string(jsonValor), "@prueba.local") {
			t.Fatalf("json filtrado: %q", jsonValor)
		}
	}
	if a.configuracion.RemitenteFijo != "rrhh@prueba.local" {
		t.Fatalf("remitente=%q", a.configuracion.RemitenteFijo)
	}
	if _, ok := direccionSobre(m.Destino); !ok {
		t.Fatal("destino no normalizable")
	}
	for _, id := range []string{"estable@prueba.local", "<a@>", "<a@b\x01>", "<a@b extra>"} {
		if messageIDValido(id) {
			t.Fatalf("id admitido %q", id)
		}
	}
	if sobre, ok := direccionSobre("Nombre <destino@prueba.local>"); !ok || sobre != "destino@prueba.local" {
		t.Fatalf("display name=%q,%t", sobre, ok)
	}
	for _, adversaria := range []string{`"nombre con espacio"@prueba.local`, `"x>y"@prueba.local`} {
		if _, ok := direccionSobre(adversaria); ok {
			t.Fatalf("sobre quoted admitido: %q", adversaria)
		}
	}
}

func TestEnviarRechazaLocalPartEntrecomilladoAntesDeConectar(t *testing.T) {
	_, roots := certificado(t, "smtp.prueba.local")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	aceptada := make(chan struct{}, 1)
	go func() {
		if c, err := ln.Accept(); err == nil {
			aceptada <- struct{}{}
			_ = c.Close()
		}
	}()
	a := nuevoParaPrueba(t, ln.Addr().String(), roots, TLSImplicito)
	m := mensajePrueba()
	m.Destino = `"nombre con espacio"@prueba.local`
	if got := a.Enviar(context.Background(), m).Estado; got != NoAceptadoPermanente {
		t.Fatalf("estado=%v", got)
	}
	select {
	case <-aceptada:
		t.Fatal("conexion para local-part entrecomillado")
	case <-time.After(40 * time.Millisecond):
	}
}

func TestEnviarAuthRechazadoNoRepiteSecreto(t *testing.T) {
	cert, roots := certificado(t, "smtp.prueba.local")
	direccion, cerrar := servidor(t, cert, true, false, func(r *bufio.Reader, w *bufio.Writer) {
		escribir := func(s string) { _, _ = w.WriteString(s); _ = w.Flush() }
		escribir("220 prueba\r\n")
		_, _ = r.ReadString('\n')
		escribir("250-AUTH XOAUTH2\r\n250 prueba\r\n")
		primero, _ := r.ReadString('\n')
		if !strings.HasPrefix(primero, "AUTH XOAUTH2 ") {
			t.Errorf("auth inicial %q", primero)
		}
		escribir("334 e30=\r\n")
		segundo, _ := r.ReadString('\n')
		if segundo != "\r\n" {
			t.Errorf("dummy=%q", segundo)
		}
		escribir("535 rechazado\r\n")
	})
	defer cerrar()
	a := nuevoParaPrueba(t, direccion, roots, TLSImplicito)
	a.configuracion.ModoAutenticacion, a.configuracion.ModoOAuth, a.configuracion.Usuario, a.configuracion.Secreto = ModoAutenticacionXOAUTH2, true, "usuario", []byte("secreto-sintetico")
	if got := a.Enviar(context.Background(), mensajePrueba()).Estado; got != NoAceptadoPermanente {
		t.Fatalf("estado=%v", got)
	}
}

func TestEnviarCanceladoAntesDelSaludoYCierreQUITTras250(t *testing.T) {
	cert, roots := certificado(t, "smtp.prueba.local")
	cerrada := make(chan struct{}, 1)
	direccion, cerrar := servidor(t, cert, true, false, func(r *bufio.Reader, _ *bufio.Writer) { _, _ = r.ReadByte(); cerrada <- struct{}{} })
	defer cerrar()
	a := nuevoParaPrueba(t, direccion, roots, TLSImplicito)
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancelar()
	if got := a.Enviar(ctx, mensajePrueba()).Estado; got != NoAceptadoTransitorio {
		t.Fatalf("cancelado=%v", got)
	}
	select {
	case <-cerrada:
	case <-time.After(time.Second):
		t.Fatal("la conexion no se cerro")
	}

	direccion2, cerrar2 := servidor(t, cert, true, false, func(r *bufio.Reader, w *bufio.Writer) {
		escribir := func(s string) { _, _ = w.WriteString(s); _ = w.Flush() }
		leer := func() string { s, _ := r.ReadString('\n'); return s }
		escribir("220 prueba\r\n")
		_ = leer()
		escribir("250 prueba\r\n")
		_ = leer()
		escribir("250 ok\r\n")
		_ = leer()
		escribir("250 ok\r\n")
		_ = leer()
		escribir("354 siga\r\n")
		for {
			if leer() == ".\r\n" {
				break
			}
		}
		escribir("250 accepted\r\n")
	})
	defer cerrar2()
	if got := nuevoParaPrueba(t, direccion2, roots, TLSImplicito).Enviar(context.Background(), mensajePrueba()).Estado; got != AceptadoPorRelay {
		t.Fatalf("quit=%v", got)
	}
}

func TestEnviarRechazaCertificadoNombreYAusenciaSTARTTLS(t *testing.T) {
	cert, roots := certificado(t, "otro.nombre")
	comandoNombre := make(chan string, 1)
	direccion, cerrar := servidor(t, cert, true, false, func(r *bufio.Reader, w *bufio.Writer) {
		_, _ = w.WriteString("220 prueba\r\n")
		_ = w.Flush()
		linea, _ := r.ReadString('\n')
		comandoNombre <- linea
	})
	defer cerrar()
	if r := nuevoParaPrueba(t, direccion, roots, TLSImplicito).Enviar(context.Background(), mensajePrueba()); r.Estado != NoAceptadoTransitorio {
		t.Fatalf("nombre: %v", r.Estado)
	}
	if linea := <-comandoNombre; linea != "" {
		t.Fatalf("comando tras nombre TLS invalido: %q", linea)
	}

	_, roots2 := certificado(t, "smtp.prueba.local")
	direccion2, cerrar2 := servidor(t, cert, false, false, func(r *bufio.Reader, w *bufio.Writer) {
		_, _ = w.WriteString("220 prueba\r\n")
		_ = w.Flush()
		_, _ = r.ReadString('\n')
		_, _ = w.WriteString("250 prueba\r\n")
		_ = w.Flush()
	})
	defer cerrar2()
	if r := nuevoParaPrueba(t, direccion2, roots2, STARTTLSObligatorio).Enviar(context.Background(), mensajePrueba()); r.Estado != NoAceptadoPermanente {
		t.Fatalf("sin tls: %v", r.Estado)
	}
}

func TestEnviarRechazaCAAjenaYClasificaEHLOYDATATemporizados(t *testing.T) {
	cert, raizValida := certificado(t, "smtp.prueba.local")
	_, raizAjena := certificado(t, "smtp.prueba.local")
	observadoCA := make(chan string, 1)
	direccion, cerrar := servidor(t, cert, true, false, func(r *bufio.Reader, w *bufio.Writer) {
		// Escribir el saludo fuerza Handshake en el servidor TLS. Si la CA es
		// ajena, el cliente no llega a iniciar una orden MAIL.
		_, _ = w.WriteString("220 prueba\r\n")
		_ = w.Flush()
		linea, _ := r.ReadString('\n')
		observadoCA <- linea
	})
	defer cerrar()
	if got := nuevoParaPrueba(t, direccion, raizAjena, TLSImplicito).Enviar(context.Background(), mensajePrueba()).Estado; got != NoAceptadoTransitorio {
		t.Fatalf("ca ajena=%v", got)
	}
	if linea := <-observadoCA; linea != "" {
		t.Fatalf("comando SMTP tras CA ajena: %q", linea)
	}

	// Tras el saludo, EHLO que no recibe respuesta debe conservar semántica
	// transitoria, no confundirse con la extensión STARTTLS ausente.
	ehloRecibido := make(chan string, 1)
	direccionEHLO, cerrarEHLO := servidor(t, cert, false, false, func(r *bufio.Reader, w *bufio.Writer) {
		_, _ = w.WriteString("220 prueba\r\n")
		_ = w.Flush()
		linea, _ := r.ReadString('\n')
		ehloRecibido <- linea
		time.Sleep(time.Second)
	})
	defer cerrarEHLO()
	aEHLO := nuevoParaPrueba(t, direccionEHLO, raizValida, STARTTLSObligatorio)
	aEHLO.configuracion.TiempoMaximo = 30 * time.Millisecond
	if got := aEHLO.Enviar(context.Background(), mensajePrueba()).Estado; got != NoAceptadoTransitorio {
		t.Fatalf("ehlo=%v", got)
	}
	if linea := <-ehloRecibido; !strings.HasPrefix(linea, "EHLO ") {
		t.Fatalf("no llego EHLO: %q", linea)
	}

	cert2, roots2 := certificado(t, "smtp.prueba.local")
	direccionDATA, cerrarDATA := servidor(t, cert2, true, false, func(r *bufio.Reader, w *bufio.Writer) {
		escribir := func(s string) { _, _ = w.WriteString(s); _ = w.Flush() }
		leer := func() string { s, _ := r.ReadString('\n'); return s }
		escribir("220 prueba\r\n")
		_ = leer()
		escribir("250 prueba\r\n")
		_ = leer()
		escribir("250 ok\r\n")
		_ = leer()
		escribir("250 ok\r\n")
		_ = leer()
		escribir("354 siga\r\n")
		for {
			if leer() == ".\r\n" {
				break
			}
		}
		time.Sleep(time.Second)
	})
	defer cerrarDATA()
	aDATA := nuevoParaPrueba(t, direccionDATA, roots2, TLSImplicito)
	aDATA.configuracion.TiempoMaximo = 30 * time.Millisecond
	if got := aDATA.Enviar(context.Background(), mensajePrueba()).Estado; got != Indeterminado {
		t.Fatalf("data=%v", got)
	}
}

func TestEnviarClasificaNegativosYDesconexionTrasDATA(t *testing.T) {
	cert, roots := certificado(t, "smtp.prueba.local")
	// Una conexión que cae tras recibir el terminador no permite saber si el relay
	// ya aceptó el mensaje.
	direccion, cerrar := servidor(t, cert, true, false, func(r *bufio.Reader, w *bufio.Writer) { conversacionSMTP(t, r, w, nil, false, true) })
	defer cerrar()
	if r := nuevoParaPrueba(t, direccion, roots, TLSImplicito).Enviar(context.Background(), mensajePrueba()); r.Estado != Indeterminado {
		t.Fatalf("desconexion=%v", r.Estado)
	}
}

func TestEnviarClasificaRCPTYDATAFinalNegativos(t *testing.T) {
	cert, roots := certificado(t, "smtp.prueba.local")
	for _, caso := range []struct {
		nombre, fase string
		estado       Estado
		codigo       string
	}{{"rcpt-5xx", "rcpt", NoAceptadoPermanente, "550"}, {"data-4xx", "final", NoAceptadoTransitorio, "450"}, {"data-600", "final", Indeterminado, "600"}} {
		t.Run(caso.nombre, func(t *testing.T) {
			direccion, cerrar := servidor(t, cert, true, false, func(r *bufio.Reader, w *bufio.Writer) {
				escribir := func(s string) { _, _ = w.WriteString(s); _ = w.Flush() }
				leer := func() string { s, _ := r.ReadString('\n'); return s }
				escribir("220 prueba\r\n")
				_ = leer()
				escribir("250 prueba\r\n")
				_ = leer()
				escribir("250 ok\r\n") // MAIL
				_ = leer()
				if caso.fase == "rcpt" {
					escribir(caso.codigo + " no\r\n")
					return
				}
				escribir("250 ok\r\n")
				_ = leer()
				escribir("354 siga\r\n")
				for {
					if leer() == ".\r\n" {
						break
					}
				}
				escribir(caso.codigo + " temporal\r\n")
			})
			defer cerrar()
			if got := nuevoParaPrueba(t, direccion, roots, TLSImplicito).Enviar(context.Background(), mensajePrueba()).Estado; got != caso.estado {
				t.Fatalf("estado=%v", got)
			}
		})
	}
}

func nuevoParaPrueba(t *testing.T, direccion string, roots *x509.CertPool, modo ModoTLS) *Adaptador {
	t.Helper()
	host, puerto, err := net.SplitHostPort(direccion)
	if err != nil {
		t.Fatal(err)
	}
	p, err := net.LookupPort("tcp", puerto)
	if err != nil {
		t.Fatal(err)
	}
	a, err := Nuevo(Configuracion{Host: host, Puerto: uint16(p), ServerName: "smtp.prueba.local", CertificadosCA: roots, RemitenteFijo: "rrhh@prueba.local", ModoTLS: modo, TiempoMaximo: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func mensajePrueba() Mensaje {
	return Mensaje{Destino: "destino@prueba.local", Asunto: "oferta ágil", Cuerpo: "cuerpo á", MessageID: "<estable@prueba.local>", FechaOrigen: time.Date(2026, time.September, 12, 10, 30, 0, 0, time.FixedZone("CEST", 2*60*60)).UTC()}
}

func servidor(t *testing.T, cert tls.Certificate, implicito, anunciaTLS bool, atender func(*bufio.Reader, *bufio.Writer)) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		if implicito {
			c = tls.Server(c, &tls.Config{Certificates: []tls.Certificate{cert}})
		}
		atender(bufio.NewReader(c), bufio.NewWriter(c))
		_ = c.Close()
	}()
	return ln.Addr().String(), func() { _ = ln.Close() }
}

func conversacionSMTP(t *testing.T, r *bufio.Reader, w *bufio.Writer, captura chan<- string, startTLS, cortarFinal bool) {
	t.Helper()
	escribir := func(s string) { _, _ = w.WriteString(s); _ = w.Flush() }
	leer := func() string {
		s, err := r.ReadString('\n')
		if err != nil {
			t.Errorf("lectura smtp: %v", err)
		}
		return s
	}
	escribir("220 prueba\r\n")
	_ = leer()
	if startTLS {
		escribir("250-prueba\r\n250-STARTTLS\r\n250 AUTH XOAUTH2\r\n")
		if got := leer(); !strings.HasPrefix(got, "STARTTLS") {
			t.Errorf("esperaba STARTTLS: %q", got)
		}
		escribir("220 siga\r\n")
		return
	}
	escribir("250 prueba\r\n")
	transaccionSMTP(t, r, w, captura, cortarFinal)
}

func transaccionSMTP(t *testing.T, r *bufio.Reader, w *bufio.Writer, captura chan<- string, cortarFinal bool) {
	t.Helper()
	escribir := func(s string) { _, _ = w.WriteString(s); _ = w.Flush() }
	leer := func() string {
		s, err := r.ReadString('\n')
		if err != nil {
			t.Errorf("lectura smtp: %v", err)
		}
		return s
	}
	if got := leer(); !strings.HasPrefix(got, "MAIL FROM:") {
		t.Errorf("mail %q", got)
	}
	escribir("250 ok\r\n")
	if got := leer(); !strings.HasPrefix(got, "RCPT TO:") {
		t.Errorf("rcpt %q", got)
	}
	escribir("250 ok\r\n")
	if got := leer(); !strings.HasPrefix(got, "DATA") {
		t.Errorf("data %q", got)
	}
	escribir("354 continue\r\n")
	var b strings.Builder
	for {
		l := leer()
		if l == ".\r\n" {
			break
		}
		b.WriteString(l)
	}
	if captura != nil {
		captura <- b.String()
	}
	if cortarFinal {
		return
	}
	escribir("250 accepted\r\n")
	_ = leer()
	escribir("221 bye\r\n")
}

func certificado(t *testing.T, nombre string) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	clave, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	plantilla := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: nombre}, DNSNames: []string{nombre}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, plantilla, plantilla, &clave.PublicKey, clave)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(clave)}))
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(raiz)
	return cert, roots
}
