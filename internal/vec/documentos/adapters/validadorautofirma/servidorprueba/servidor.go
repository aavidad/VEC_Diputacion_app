// Package servidorprueba imita, solo para pruebas, el contrato REST
// `POST /v2/verify` del validador de GrxFirma en modo
// `-rest-solo-verificacion`. No verifica firmas: devuelve dictamenes
// sinteticos elegidos por la prueba. No debe conectarse en ninguna
// composicion real de VEC.
//
// La forma de la respuesta reproduce el contrato
// `autofirmav2.dictamen-verificacion.v2`: metadatos de respuesta y `dictamen`
// con firmas, revisiones, huellas de eco calculadas sobre los bytes recibidos
// y extensiones remotas desactivadas. Los valores son sintéticos.
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

// ContratoDictamen es el unico contrato que interpreta el adaptador.
const ContratoDictamen = "autofirmav2.dictamen-verificacion.v2"

// Escenario elige la respuesta sintetica del servidor.
type Escenario string

const (
	// ValidaSinSello: firma integra, cadena hasta anclas locales, CRL local
	// vigente, sin sello de tiempo y vinculo acreditado.
	ValidaSinSello Escenario = "valida_sin_sello"
	// ValidaConSello: como ValidaSinSello con sello RFC 3161 valido.
	ValidaConSello Escenario = "valida_con_sello"
	// SelloNoComprobado: sello presente en un formato no evaluado.
	SelloNoComprobado Escenario = "sello_no_comprobado"
	// SelloNoValido: sello presente que no corresponde a la firma.
	SelloNoValido Escenario = "sello_no_valido"
	// Revocado: la CRL local declara revocado el certificado firmante.
	Revocado Escenario = "revocado"
	// RevocacionNoComprobada: no hay CRL vigente para la ruta.
	RevocacionNoComprobada Escenario = "revocacion_no_comprobada"
	// VinculoNoAcreditado: la firma no cubre el original aportado.
	VinculoNoAcreditado Escenario = "vinculo_no_acreditado"
	// VinculoNoAportado: GrxFirma no recibio original y dictamina valida;
	// VEC siempre lo envia, asi que no le basta.
	VinculoNoAportado Escenario = "vinculo_no_aportado"
	// ContratoDesconocido: dictamen con otro contrato.
	ContratoDesconocido Escenario = "contrato_desconocido"
	// SinDictamen: solo campos heredados, sin `dictamen`.
	SinDictamen Escenario = "sin_dictamen"
	// HuellaEcoDistinta: el eco del firmado no corresponde a lo enviado.
	HuellaEcoDistinta Escenario = "huella_eco_distinta"
	// HuellaOriginalDistinta: el eco del original no corresponde.
	HuellaOriginalDistinta Escenario = "huella_original_distinta"
	// IntegridadRota: el contenido no corresponde a la firma.
	IntegridadRota Escenario = "integridad_rota"
	// IntegridadParcial: la firma no cubre todo el contenido.
	IntegridadParcial Escenario = "integridad_parcial"
	// SinAnclas: no hay ruta hasta las anclas locales.
	SinAnclas Escenario = "sin_anclas"
	// VariosFirmantes: dos firmantes validos; el dictamen global es valida
	// pero sin huella unica de certificado.
	VariosFirmantes Escenario = "varios_firmantes"
	// ValidaIncoherente: dictamen global valida con revocacion no comprobada.
	ValidaIncoherente Escenario = "valida_incoherente"
	// NegativaIncoherente: dictamen no_valida con todos los aspectos correctos.
	NegativaIncoherente Escenario = "negativa_incoherente"
	// EstadoDesconocido: aspecto con un estado fuera del catalogo.
	EstadoDesconocido Escenario = "estado_desconocido"
	// FormatoNoDetectado: GrxFirma responde 400 con texto libre.
	FormatoNoDetectado Escenario = "formato_no_detectado"
	// ErrorInterno: 500 con texto que no debe propagarse.
	ErrorInterno Escenario = "error_interno"
	// RespuestaEnorme: cuerpo mayor que el limite del adaptador.
	RespuestaEnorme Escenario = "respuesta_enorme"
	// TipoIncorrecto: 200 con Content-Type distinto de JSON.
	TipoIncorrecto Escenario = "tipo_incorrecto"
	// Lento: responde ValidaSinSello tras Retardo.
	Lento Escenario = "lento"
	// Redireccion: 307 hacia otra ruta.
	Redireccion Escenario = "redireccion"
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
	s := &Servidor{token: token, esc: ValidaSinSello, retardo: 2 * time.Second}
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
	if r.URL.Path != "/v2/verify" {
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
	copia := p
	s.ultima = &copia
	s.mu.Unlock()
	if len(p.Firmado) == 0 {
		escribir(w, http.StatusBadRequest, map[string]string{"error": "content_base64 no es valido"})
		return
	}
	s.responder(w, r, esc, retardo, p)
}

func (s *Servidor) responder(w http.ResponseWriter, r *http.Request, esc Escenario, retardo time.Duration, p Peticion) {
	switch esc {
	case FormatoNoDetectado:
		escribir(w, http.StatusBadRequest, map[string]string{"error": TextoProveedor})
		return
	case ErrorInterno:
		escribir(w, http.StatusInternalServerError, map[string]string{"error": TextoProveedor})
		return
	case RespuestaEnorme:
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"valid":true,"details":["` + strings.Repeat("x", 1<<20) + `"]}`))
		return
	case TipoIncorrecto:
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<p>" + TextoProveedor + "</p>"))
		return
	case Redireccion:
		http.Redirect(w, r, "/otra", http.StatusTemporaryRedirect)
		return
	case Lento:
		select {
		case <-time.After(retardo):
		case <-r.Context().Done():
			return
		}
	}
	d := dictamenValido(p)
	aplicar(esc, d, p)
	d.recomponer()
	firmantes := make([]string, 0, len(d.Firmas))
	for _, firma := range d.Firmas {
		firmantes = append(firmantes, firma.CertificadoHuellaSHA256)
	}
	cuerpo := map[string]any{
		"ok": true, "valid": d.Estado == "valida", "reason": d.Motivo,
		"details": []string{TextoProveedor}, "signers": firmantes,
	}
	if esc != SinDictamen {
		cuerpo["dictamen"] = d
	}
	escribir(w, http.StatusOK, cuerpo)
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
