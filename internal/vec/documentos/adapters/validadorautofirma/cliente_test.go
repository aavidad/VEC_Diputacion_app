package validadorautofirma_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/adapters/validadorautofirma"
	"vec-diputacion-granada/internal/vec/documentos/adapters/validadorautofirma/servidorprueba"
	"vec-diputacion-granada/internal/vec/documentos/ports"
)

var tokenPrueba = strings.Repeat("t", 40)

func caServidor(s *servidorprueba.Servidor) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: s.Certificate().Raw})
}

func configuracion(s *servidorprueba.Servidor) validadorautofirma.Configuracion {
	return validadorautofirma.Configuracion{URL: s.URL, CAPEM: caServidor(s), Token: []byte(tokenPrueba)}
}

func arrancar(t *testing.T, exigirCertificado bool) *servidorprueba.Servidor {
	t.Helper()
	s := servidorprueba.Nuevo(tokenPrueba, exigirCertificado)
	t.Cleanup(s.Close)
	return s
}

func cliente(t *testing.T, c validadorautofirma.Configuracion) *validadorautofirma.Cliente {
	t.Helper()
	v, err := validadorautofirma.Nuevo(c)
	if err != nil {
		t.Fatalf("configuracion: %v", err)
	}
	return v
}

func solicitud(firmado []byte) ports.SolicitudVerificacionFirma {
	original := []byte("original sintetico custodiado")
	huella := sha256.Sum256(original)
	return ports.SolicitudVerificacionFirma{
		DocumentoID: "ref:" + strings.Repeat("1", 64), Version: 3,
		HuellaOriginalSHA256: hex.EncodeToString(huella[:]),
		ContenidoOriginal:    original,
		ContenidoFirmado:     firmado,
	}
}

func certificadoCliente(t *testing.T) ([]byte, []byte) {
	t.Helper()
	clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	plantilla := &x509.Certificate{
		SerialNumber: big.NewInt(7), Subject: pkix.Name{CommonName: "vec-cliente-sintetico"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, plantilla, plantilla, &clave.PublicKey, clave)
	if err != nil {
		t.Fatal(err)
	}
	claveDER, err := x509.MarshalPKCS8PrivateKey(clave)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: claveDER})
}

func verificar(t *testing.T, v *validadorautofirma.Cliente, s ports.SolicitudVerificacionFirma) ports.VerificacionFirmaMotivada {
	t.Helper()
	r, err := v.VerificarMotivado(context.Background(), s)
	if err != nil {
		t.Fatalf("verificacion: %v", err)
	}
	if err := r.ValidarContra(s); err != nil {
		t.Fatalf("resultado incoherente: %+v", r)
	}
	// Solo `verificada` puede superar la regla estricta del puerto.
	if (r.Motivo == ports.MotivoFirmaVerificada) != (r.Resultado.ValidarContra(s) == nil) {
		t.Fatalf("estado y regla estricta discrepan: %+v", r)
	}
	if texto := fmt.Sprintf("%+v", r); strings.Contains(texto, servidorprueba.TextoProveedor) ||
		strings.Contains(texto, "PERSONA") || strings.Contains(texto, "CA SINTETICA") ||
		strings.Contains(texto, "crl_local") || strings.Contains(texto, "anclas_locales") {
		t.Fatalf("resultado filtra texto o datos del proveedor: %s", texto)
	}
	return r
}

func TestValidaSinSelloAcreditaFirmaConPoliticaV1(t *testing.T) {
	s := arrancar(t, false)
	v := cliente(t, configuracion(s))
	sol := solicitud([]byte{0x30, 0x82, 0x01, 0x00, 0x02})
	r := verificar(t, v, sol)
	if r.Motivo != ports.MotivoFirmaVerificada || r.Resultado.Estado != ports.EstadoVerificacionValida ||
		!r.Resultado.VinculoOriginal ||
		r.Resultado.CertificadoHuellaSHA256 != servidorprueba.HuellaCertificadoSintetica ||
		r.Resultado.FirmanteRef != "ref:"+servidorprueba.HuellaCertificadoSintetica ||
		r.Resultado.RevocacionEstado != ports.RevocacionVigente ||
		r.Resultado.SelloTiempoEstado != ports.SelloTiempoNoPresente ||
		r.Resultado.HuellaOriginalSHA256 != sol.HuellaOriginalSHA256 {
		t.Fatalf("resultado inesperado: %+v", r)
	}
	suma := sha256.Sum256(sol.ContenidoFirmado)
	if r.Resultado.HuellaFirmadoSHA256 != hex.EncodeToString(suma[:]) {
		t.Fatal("huella del firmado no calculada sobre los bytes enviados")
	}
	p, ok := s.Ultima()
	if !ok {
		t.Fatal("sin peticion")
	}
	campos := make([]string, 0, len(p.Campos))
	for k := range p.Campos {
		campos = append(campos, k)
	}
	sort.Strings(campos)
	if strings.Join(campos, ",") != "content_base64,name,original_content_base64" ||
		string(p.Campos["name"]) != `"documento"` || p.Autorizacion != "Bearer "+tokenPrueba ||
		string(p.Original) != string(sol.ContenidoOriginal) || string(p.Firmado) != string(sol.ContenidoFirmado) {
		t.Fatalf("peticion no minimizada o incorrecta: %v %+v", campos, p)
	}
	// El puerto sin motivo devuelve el mismo resultado.
	rp, err := v.Verificar(context.Background(), sol)
	if err != nil || rp.Estado != ports.EstadoVerificacionValida || rp.ValidarContra(sol) != nil {
		t.Fatalf("puerto: %+v %v", rp, err)
	}
}

func TestPAdESEnviaOriginalParaAcreditarVinculo(t *testing.T) {
	s := arrancar(t, false)
	v := cliente(t, configuracion(s))
	sol := solicitud([]byte("%PDF-1.7 sintetico"))
	if r := verificar(t, v, sol); r.Motivo != ports.MotivoFirmaVerificada || !r.Resultado.VinculoOriginal {
		t.Fatalf("PAdES: %+v", r)
	}
	if p, _ := s.Ultima(); p.OriginalFalta || string(p.Original) != string(sol.ContenidoOriginal) {
		t.Fatal("no envio el original en PAdES")
	}
}

func TestTraduccionDeEscenarios(t *testing.T) {
	s := arrancar(t, false)
	v := cliente(t, configuracion(s))
	casos := []struct {
		esc        servidorprueba.Escenario
		motivo     ports.MotivoVerificacionFirma
		revocacion string
		sello      string
	}{
		{servidorprueba.ValidaSinSello, ports.MotivoFirmaVerificada, "vigente", "no_presente"},
		{servidorprueba.ValidaConSello, ports.MotivoFirmaVerificada, "vigente", "valido"},
		{servidorprueba.SelloNoComprobado, ports.MotivoFirmaVerificada, "vigente", "no_comprobado"},
		{servidorprueba.SelloNoValido, ports.MotivoSelloTiempoNoAcreditado, "vigente", "no_valido"},
		{servidorprueba.Revocado, ports.MotivoCertificadoNoValido, "revocado", "no_presente"},
		{servidorprueba.RevocacionNoComprobada, ports.MotivoRevocacionNoAcreditada, "no_comprobada", "no_presente"},
		{servidorprueba.VinculoNoAcreditado, ports.MotivoVinculoOriginalNoAcreditado, "vigente", "no_presente"},
		{servidorprueba.VinculoNoAportado, ports.MotivoVinculoOriginalNoAcreditado, "vigente", "no_presente"},
		{servidorprueba.IntegridadRota, ports.MotivoIntegridadNoValida, "vigente", "no_presente"},
		{servidorprueba.IntegridadParcial, ports.MotivoIntegridadParcial, "vigente", "no_presente"},
		{servidorprueba.SinAnclas, ports.MotivoConfianzaNoAcreditada, "vigente", "no_presente"},
		{servidorprueba.VariosFirmantes, ports.MotivoFirmanteNoIdentificado, "vigente", "no_presente"},
		// Un `valida` del validador con revocacion no comprobada: VEC es mas
		// estricta y conserva su propio motivo.
		{servidorprueba.ValidaIncoherente, ports.MotivoRevocacionNoAcreditada, "no_comprobada", "no_presente"},
		// Respuestas no interpretables: no se adopta ningun aspecto.
		{servidorprueba.ContratoDesconocido, ports.MotivoRespuestaNoInterpretable, "no_informado", "no_informado"},
		{servidorprueba.SinDictamen, ports.MotivoRespuestaNoInterpretable, "no_informado", "no_informado"},
		{servidorprueba.HuellaEcoDistinta, ports.MotivoRespuestaNoInterpretable, "no_informado", "no_informado"},
		{servidorprueba.HuellaOriginalDistinta, ports.MotivoRespuestaNoInterpretable, "no_informado", "no_informado"},
		{servidorprueba.NegativaIncoherente, ports.MotivoRespuestaNoInterpretable, "no_informado", "no_informado"},
		{servidorprueba.EstadoDesconocido, ports.MotivoRespuestaNoInterpretable, "no_informado", "no_informado"},
		{servidorprueba.FormatoNoDetectado, ports.MotivoRechazadaPorValidador, "no_informado", "no_informado"},
		{servidorprueba.ErrorInterno, ports.MotivoValidadorNoDisponible, "no_informado", "no_informado"},
		{servidorprueba.RespuestaEnorme, ports.MotivoRespuestaNoInterpretable, "no_informado", "no_informado"},
		{servidorprueba.TipoIncorrecto, ports.MotivoRespuestaNoInterpretable, "no_informado", "no_informado"},
		{servidorprueba.Redireccion, ports.MotivoValidadorNoDisponible, "no_informado", "no_informado"},
	}
	for _, caso := range casos {
		s.Escenario(caso.esc)
		r := verificar(t, v, solicitud([]byte{0x30, 0x01}))
		if r.Motivo != caso.motivo || r.Resultado.Estado != caso.motivo.EstadoAsociado() ||
			r.Resultado.RevocacionEstado != caso.revocacion || r.Resultado.SelloTiempoEstado != caso.sello {
			t.Fatalf("%s: %+v", caso.esc, r)
		}
		if caso.motivo != ports.MotivoFirmaVerificada && r.Resultado.Estado == ports.EstadoVerificacionValida {
			t.Fatalf("%s: acredito firma", caso.esc)
		}
	}
}

func TestIndisponibilidadEsIndeterminada(t *testing.T) {
	s := arrancar(t, false)
	s.Escenario(servidorprueba.Lento)
	c := configuracion(s)
	c.Timeout = 150 * time.Millisecond
	v := cliente(t, c)
	if r := verificar(t, v, solicitud([]byte{0x30})); r.Motivo != ports.MotivoValidadorNoDisponible {
		t.Fatalf("lento: %+v", r)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	r, err := v.VerificarMotivado(ctx, solicitud([]byte{0x30}))
	if err != nil || r.Motivo != ports.MotivoValidadorNoDisponible {
		t.Fatalf("cancelado: %+v %v", r, err)
	}
	s.Close()
	if r := verificar(t, cliente(t, configuracion(s)), solicitud([]byte{0x30})); r.Motivo != ports.MotivoValidadorNoDisponible {
		t.Fatalf("cerrado: %+v", r)
	}
}

func TestCredencialYConfianzaTLS(t *testing.T) {
	s := arrancar(t, false)
	c := configuracion(s)
	c.Token = []byte(strings.Repeat("x", 40))
	if r := verificar(t, cliente(t, c), solicitud([]byte{0x30})); r.Motivo != ports.MotivoCredencialRechazada {
		t.Fatalf("token ajeno: %+v", r)
	}
	// httptest comparte certificado entre servidores: la CA ajena se genera.
	c = configuracion(s)
	c.CAPEM, _ = certificadoCliente(t)
	if r := verificar(t, cliente(t, c), solicitud([]byte{0x30})); r.Motivo != ports.MotivoValidadorNoDisponible {
		t.Fatalf("CA ajena: %+v", r)
	}
	// El certificado de httptest cubre example.com; un nombre ajeno se rechaza.
	c = configuracion(s)
	c.NombreServidorTLS = "example.com"
	if r := verificar(t, cliente(t, c), solicitud([]byte{0x30})); r.Motivo != ports.MotivoFirmaVerificada {
		t.Fatalf("nombre fijado: %+v", r)
	}
	c.NombreServidorTLS = "validador.invalid"
	if r := verificar(t, cliente(t, c), solicitud([]byte{0x30})); r.Motivo != ports.MotivoValidadorNoDisponible {
		t.Fatalf("nombre ajeno: %+v", r)
	}
}

func TestMTLSExigeCertificadoCliente(t *testing.T) {
	s := arrancar(t, true)
	if r := verificar(t, cliente(t, configuracion(s)), solicitud([]byte{0x30})); r.Motivo != ports.MotivoValidadorNoDisponible {
		t.Fatalf("sin certificado cliente: %+v", r)
	}
	c := configuracion(s)
	c.CertificadoClientePEM, c.ClaveClientePEM = certificadoCliente(t)
	if r := verificar(t, cliente(t, c), solicitud([]byte{0x30})); r.Motivo != ports.MotivoFirmaVerificada {
		t.Fatalf("con certificado cliente: %+v", r)
	}
}

func TestSolicitudInvalidaNoLlamaAlValidador(t *testing.T) {
	s := arrancar(t, false)
	v := cliente(t, configuracion(s))
	sol := solicitud([]byte{0x30})
	sol.HuellaOriginalSHA256 = strings.Repeat("0", 64)
	if _, err := v.VerificarMotivado(context.Background(), sol); !errors.Is(err, ports.ErrVerificacionFirmaInvalida) {
		t.Fatalf("huella falsa aceptada: %v", err)
	}
	var nulo *validadorautofirma.Cliente
	if _, err := nulo.Verificar(context.Background(), solicitud([]byte{0x30})); !errors.Is(err, ports.ErrVerificacionFirmaInvalida) {
		t.Fatalf("cliente nulo: %v", err)
	}
	if s.Llamadas() != 0 {
		t.Fatal("llamo al validador con una solicitud invalida")
	}
}

func TestConcurrenciaSinEstadoCompartido(t *testing.T) {
	s := arrancar(t, false)
	v := cliente(t, configuracion(s))
	var grupo sync.WaitGroup
	for i := 0; i < 8; i++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			r, err := v.VerificarMotivado(context.Background(), solicitud([]byte{0x30, byte(i)}))
			if err != nil || r.Motivo != ports.MotivoFirmaVerificada {
				t.Errorf("concurrente: %+v %v", r, err)
			}
		}()
	}
	grupo.Wait()
}
