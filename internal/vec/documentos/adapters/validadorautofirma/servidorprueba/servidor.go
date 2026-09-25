// Package servidorprueba imita, solo para pruebas, el contrato REST
// `POST /verify` del validador de AutofirmaV2. No verifica firmas: devuelve
// respuestas sinteticas elegidas por la prueba. No debe conectarse en ninguna
// composicion real de VEC.
//
// La forma de las respuestas reproduce la API publicada por AutofirmaV2
// (`ok`, `valid`, `reason`, `details`, `signers` y `result` con `integrity`,
// `certificate`, `trust`, `signerSummaries`, `warnings`, `errors` y
// `evidence`), con valores inventados y sin datos reales.
package servidorprueba

import (
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

// Escenario elige la respuesta sintetica del servidor.
type Escenario string

const (
	// SeparadaCorrecta: firma separada integra, certificado y confianza validos.
	SeparadaCorrecta Escenario = "separada_correcta"
	// PAdESCorrecta: firma PDF integra; el original aportado se ignora.
	PAdESCorrecta Escenario = "pades_correcta"
	// IntegridadRota: el contenido no corresponde a la firma.
	IntegridadRota Escenario = "integridad_rota"
	// CertificadoRevocado: revocacion concluyente.
	CertificadoRevocado Escenario = "certificado_revocado"
	// ConfianzaDesconocida: sin anclas de confianza para el emisor.
	ConfianzaDesconocida Escenario = "confianza_desconocida"
	// FormatoNoDetectado: AutofirmaV2 responde 400 con texto libre.
	FormatoNoDetectado Escenario = "formato_no_detectado"
	// ErrorInterno: 500 con texto que no debe propagarse.
	ErrorInterno Escenario = "error_interno"
	// RespuestaEnorme: cuerpo mayor que el limite del adaptador.
	RespuestaEnorme Escenario = "respuesta_enorme"
	// TipoIncorrecto: 200 con Content-Type distinto de JSON.
	TipoIncorrecto Escenario = "tipo_incorrecto"
	// Lento: responde tras Retardo.
	Lento Escenario = "lento"
	// Redireccion: 307 hacia otra ruta.
	Redireccion Escenario = "redireccion"
	// EstadoDesconocido: aspecto con un estado fuera del catalogo.
	EstadoDesconocido Escenario = "estado_desconocido"
)

// HuellaCertificadoSintetica es la huella que devuelven los escenarios
// positivos. No corresponde a ningun certificado real.
var HuellaCertificadoSintetica = strings.Repeat("ab", 32)

// TextoProveedor es texto libre que el adaptador nunca debe propagar.
const TextoProveedor = "detalle-interno-del-proveedor"

// Peticion es lo que el servidor recibio, para comprobar minimizacion.
type Peticion struct {
	Autorizacion  string
	Campos        map[string]json.RawMessage
	Firmado       []byte
	Original      []byte
	OriginalFalta bool
}

// Servidor es un validador simulado sobre TLS de httptest.
type Servidor struct {
	*httptest.Server
	token    string
	mu       sync.Mutex
	esc      Escenario
	retardo  time.Duration
	ultima   *Peticion
	llamadas int
}

// Nuevo arranca un servidor TLS que exige `Authorization: Bearer <token>` si
// token no esta vacio. Si exigirCertificado es verdadero, pide certificado
// cliente (como el proxy mTLS recomendado en el despliegue).
func Nuevo(token string, exigirCertificado bool) *Servidor {
	s := &Servidor{token: token, esc: SeparadaCorrecta, retardo: 2 * time.Second}
	s.Server = httptest.NewUnstartedServer(http.HandlerFunc(s.atender))
	s.Server.TLS = &tls.Config{MinVersion: tls.VersionTLS12}
	if exigirCertificado {
		s.Server.TLS.ClientAuth = tls.RequireAnyClientCert
	}
	s.Server.Config.ErrorLog = log.New(io.Discard, "", 0)
	s.Server.StartTLS()
	return s
}

// Escenario fija la siguiente respuesta.
func (s *Servidor) Escenario(e Escenario) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.esc = e
}

// Retardo fija la espera del escenario Lento.
func (s *Servidor) Retardo(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retardo = d
}

// Ultima devuelve una copia de la ultima peticion recibida.
func (s *Servidor) Ultima() (Peticion, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ultima == nil {
		return Peticion{}, false
	}
	return *s.ultima, true
}

// Llamadas devuelve cuantas peticiones llegaron al manejador.
func (s *Servidor) Llamadas() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.llamadas
}

func (s *Servidor) atender(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.llamadas++
	esc, retardo := s.esc, s.retardo
	s.mu.Unlock()
	if r.URL.Path != "/verify" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		escribir(w, http.StatusMethodNotAllowed, map[string]string{"error": "metodo no permitido"})
		return
	}
	if s.token != "" {
		recibido := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(recibido), []byte(s.token)) != 1 {
			escribir(w, http.StatusUnauthorized, map[string]string{"error": "autorizacion requerida"})
			return
		}
	}
	var campos map[string]json.RawMessage
	r.Body = http.MaxBytesReader(w, r.Body, 64<<20)
	if err := json.NewDecoder(r.Body).Decode(&campos); err != nil {
		escribir(w, http.StatusBadRequest, map[string]string{"error": "json de verificacion invalido"})
		return
	}
	p := Peticion{Autorizacion: r.Header.Get("Authorization"), Campos: campos}
	p.Firmado = decodificar(campos["content_base64"])
	if raw, ok := campos["original_content_base64"]; ok {
		p.Original = decodificar(raw)
	} else {
		p.OriginalFalta = true
	}
	s.mu.Lock()
	s.ultima = &p
	s.mu.Unlock()
	if len(p.Firmado) == 0 {
		escribir(w, http.StatusBadRequest, map[string]string{"error": "content_base64 no es valido"})
		return
	}
	s.responder(w, r, esc, retardo)
}

func (s *Servidor) responder(w http.ResponseWriter, r *http.Request, esc Escenario, retardo time.Duration) {
	switch esc {
	case FormatoNoDetectado:
		escribir(w, http.StatusBadRequest, map[string]string{"error": TextoProveedor})
	case ErrorInterno:
		escribir(w, http.StatusInternalServerError, map[string]string{"error": TextoProveedor})
	case RespuestaEnorme:
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"valid":true,"details":["` + strings.Repeat("x", 1<<20) + `"]}`))
	case TipoIncorrecto:
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<p>" + TextoProveedor + "</p>"))
	case Lento:
		select {
		case <-time.After(retardo):
		case <-r.Context().Done():
			return
		}
		escribir(w, http.StatusOK, respuesta("valid", "valid", "valid", true, "modo=detached"))
	case Redireccion:
		http.Redirect(w, r, "/otra", http.StatusTemporaryRedirect)
	case PAdESCorrecta:
		escribir(w, http.StatusOK, respuesta("valid", "valid", "valid", true, "modo=embedded", "original_aportado_ignorado_en_pades"))
	case IntegridadRota:
		escribir(w, http.StatusOK, respuesta("invalid", "unknown", "unknown", false, "modo=detached"))
	case CertificadoRevocado:
		escribir(w, http.StatusOK, respuesta("valid", "invalid", "warning", false, "modo=detached"))
	case ConfianzaDesconocida:
		escribir(w, http.StatusOK, respuesta("valid", "valid", "unknown", true, "modo=detached"))
	case EstadoDesconocido:
		escribir(w, http.StatusOK, respuesta("valid", "quiza", "valid", true, "modo=detached"))
	default:
		escribir(w, http.StatusOK, respuesta("valid", "valid", "valid", true, "modo=detached"))
	}
}

func respuesta(integridad, certificado, confianza string, valida bool, detalles ...string) map[string]any {
	detalles = append([]string{"formato_detectado=CAdES"}, detalles...)
	detalles = append(detalles, TextoProveedor)
	firmante := map[string]string{
		"id": "firmante-sintetico", "subject": "CN=PERSONA SINTETICA",
		"issuer": "CN=CA SINTETICA", "fingerprint": HuellaCertificadoSintetica,
	}
	resultado := map[string]any{
		"valid": valida, "reason": TextoProveedor, "details": detalles,
		"signers": []string{"firmante-sintetico"}, "format": "CAdES", "coverage": "full",
		"integrity":       map[string]any{"status": integridad, "reason": TextoProveedor},
		"certificate":     map[string]any{"status": certificado, "reason": TextoProveedor},
		"trust":           map[string]any{"status": confianza, "reason": TextoProveedor},
		"signerSummaries": []map[string]string{firmante},
		"warnings":        []string{TextoProveedor},
		"evidence":        []map[string]string{{"type": "revocation", "summary": TextoProveedor}},
	}
	return map[string]any{
		"ok": true, "valid": valida, "reason": TextoProveedor, "details": detalles,
		"signers": []string{"firmante-sintetico"}, "result": resultado,
	}
}

func escribir(w http.ResponseWriter, estado int, cuerpo any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(cuerpo)
}

func decodificar(raw json.RawMessage) []byte {
	var texto string
	if json.Unmarshal(raw, &texto) != nil {
		return nil
	}
	datos, err := base64.StdEncoding.DecodeString(texto)
	if err != nil {
		return nil
	}
	return datos
}
