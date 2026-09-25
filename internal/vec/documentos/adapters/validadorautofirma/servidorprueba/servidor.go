// Package servidorprueba imita, solo para pruebas, el contrato REST
// `POST /verify` del validador de AutofirmaV2 en modo
// `-rest-solo-verificacion`. No verifica firmas: devuelve dictamenes
// sinteticos elegidos por la prueba. No debe conectarse en ninguna
// composicion real de VEC.
//
// La forma de la respuesta reproduce el contrato
// `autofirmav2.dictamen-verificacion.v1`: campos heredados (`ok`, `valid`,
// `reason`, `details`, `signers`, `result`) y `dictamen` con sus aspectos,
// huellas de eco calculadas sobre los bytes recibidos, firmantes y
// extensiones remotas desactivadas. Los valores son inventados y sin datos
// reales.
package servidorprueba

import (
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
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
const ContratoDictamen = "autofirmav2.dictamen-verificacion.v1"

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
	// VinculoNoAportado: AutofirmaV2 no recibio original y dictamina valida;
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
	// FormatoNoDetectado: AutofirmaV2 responde 400 con texto libre.
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
	aplicar(esc, d)
	d.recomponer()
	cuerpo := map[string]any{
		"ok": true, "valid": d.Estado == "valida", "reason": TextoProveedor,
		"details": []string{"formato_detectado=CAdES", TextoProveedor}, "signers": []string{"firmante-sintetico"},
		"result": map[string]any{"valid": d.Estado == "valida", "reason": TextoProveedor,
			"integrity": map[string]string{"status": "valid", "reason": TextoProveedor}},
	}
	if esc != SinDictamen {
		cuerpo["dictamen"] = d
	}
	escribir(w, http.StatusOK, cuerpo)
}

// Aspecto reproduce `{estado, motivo?, fuente?, fecha?}` del contrato.
type Aspecto struct {
	Estado string `json:"estado"`
	Motivo string `json:"motivo,omitempty"`
	Fuente string `json:"fuente,omitempty"`
	Fecha  string `json:"fecha,omitempty"`
}

// Firmante reproduce un firmante del dictamen, con datos inventados.
type Firmante struct {
	CertificadoHuellaSHA256 string  `json:"certificadoHuellaSHA256"`
	Serie                   string  `json:"serie,omitempty"`
	Asunto                  string  `json:"asunto,omitempty"`
	Emisor                  string  `json:"emisor,omitempty"`
	Cadena                  Aspecto `json:"cadena"`
	Certificado             Aspecto `json:"certificado"`
	Revocacion              Aspecto `json:"revocacion"`
	SelloTiempo             Aspecto `json:"selloTiempo"`
}

// Dictamen reproduce `dictamen` de `autofirmav2.dictamen-verificacion.v1`.
type Dictamen struct {
	Contrato                string            `json:"contrato"`
	Estado                  string            `json:"estado"`
	Motivo                  string            `json:"motivo"`
	Formato                 string            `json:"formato,omitempty"`
	ComprobadoEn            string            `json:"comprobadoEn"`
	Integridad              Aspecto           `json:"integridad"`
	Cadena                  Aspecto           `json:"cadena"`
	Certificado             Aspecto           `json:"certificado"`
	Revocacion              Aspecto           `json:"revocacion"`
	SelloTiempo             Aspecto           `json:"selloTiempo"`
	VinculoOriginal         Aspecto           `json:"vinculoOriginal"`
	HuellaFirmadoSHA256     string            `json:"huellaFirmadoSHA256"`
	HuellaOriginalSHA256    string            `json:"huellaOriginalSHA256,omitempty"`
	CertificadoHuellaSHA256 string            `json:"certificadoHuellaSHA256,omitempty"`
	Firmantes               []Firmante        `json:"firmantes"`
	Extensiones             map[string]string `json:"extensiones"`

	// fijado impide que recomponer sustituya un veredicto deliberadamente
	// incoherente de la prueba.
	fijado bool
}

func dictamenValido(p Peticion) *Dictamen {
	fecha := "2026-09-25T10:00:00Z"
	firmante := Firmante{
		CertificadoHuellaSHA256: HuellaCertificadoSintetica, Serie: "07",
		Asunto: "CN=PERSONA SINTETICA", Emisor: "CN=CA SINTETICA",
		Cadena:      Aspecto{Estado: "valida", Fuente: "anclas_locales"},
		Certificado: Aspecto{Estado: "vigente", Fecha: "2028-01-31T23:59:59Z"},
		Revocacion:  Aspecto{Estado: "vigente", Fuente: "crl_local", Fecha: fecha},
		SelloTiempo: Aspecto{Estado: "no_presente"},
	}
	d := &Dictamen{
		Contrato: ContratoDictamen, Formato: "CAdES", ComprobadoEn: fecha,
		Integridad:          Aspecto{Estado: "valida"},
		VinculoOriginal:     Aspecto{Estado: "acreditado", Fuente: "cms_message_digest"},
		HuellaFirmadoSHA256: huella(p.Firmado),
		Firmantes:           []Firmante{firmante},
		Extensiones:         map[string]string{"revocacionRemota": "desactivada", "selloTiempoRemoto": "desactivada"},
	}
	if !p.OriginalFalta {
		d.HuellaOriginalSHA256 = huella(p.Original)
	} else {
		d.VinculoOriginal = Aspecto{Estado: "no_aportado"}
	}
	return d
}

func aplicar(esc Escenario, d *Dictamen) {
	f := &d.Firmantes[0]
	switch esc {
	case ValidaConSello:
		f.SelloTiempo = Aspecto{Estado: "valido", Fuente: "rfc3161", Fecha: "2026-09-25T09:59:00Z"}
	case SelloNoComprobado:
		f.SelloTiempo = Aspecto{Estado: "no_comprobado", Motivo: "formato_sin_evaluacion_de_sello"}
	case SelloNoValido:
		f.SelloTiempo = Aspecto{Estado: "no_valido", Motivo: "imprint_no_coincide"}
	case Revocado:
		f.Revocacion = Aspecto{Estado: "revocado", Fuente: "crl_local", Fecha: "2026-09-01T00:00:00Z"}
	case RevocacionNoComprobada:
		f.Revocacion = Aspecto{Estado: "no_comprobada", Motivo: "sin_crl_vigente"}
	case VinculoNoAcreditado:
		d.VinculoOriginal = Aspecto{Estado: "no_acreditado", Motivo: "original_no_cubierto"}
	case VinculoNoAportado:
		d.HuellaOriginalSHA256 = ""
		d.VinculoOriginal = Aspecto{Estado: "no_aportado"}
	case ContratoDesconocido:
		d.Contrato = "autofirmav2.dictamen-verificacion.v2"
	case HuellaEcoDistinta:
		d.HuellaFirmadoSHA256 = strings.Repeat("cd", 32)
	case HuellaOriginalDistinta:
		d.HuellaOriginalSHA256 = strings.Repeat("ef", 32)
	case IntegridadRota:
		d.Integridad = Aspecto{Estado: "no_valida", Motivo: TextoProveedor}
	case IntegridadParcial:
		d.Integridad = Aspecto{Estado: "parcial", Motivo: "contenido_no_cubierto"}
	case SinAnclas:
		f.Cadena = Aspecto{Estado: "no_comprobada", Motivo: "sin_ruta_hasta_anclas"}
	case VariosFirmantes:
		otro := *f
		otro.CertificadoHuellaSHA256 = strings.Repeat("12", 32)
		d.Firmantes = append(d.Firmantes, otro)
	case ValidaIncoherente:
		f.Revocacion = Aspecto{Estado: "no_comprobada"}
		d.Estado, d.Motivo, d.fijado = "valida", "verificada", true
	case NegativaIncoherente:
		d.Estado, d.Motivo, d.fijado = "no_valida", "integridad_no_valida", true
	case EstadoDesconocido:
		f.Certificado = Aspecto{Estado: "quiza"}
	}
}

// recomponer agrega los aspectos por el peor estado y deriva el veredicto
// con la misma precedencia que AutofirmaV2.
func (d *Dictamen) recomponer() {
	peor := func(orden []string, valor func(Firmante) Aspecto) Aspecto {
		mejor, rango := Aspecto{}, len(orden)+1
		for _, f := range d.Firmantes {
			a := valor(f)
			r := len(orden)
			for i, e := range orden {
				if e == a.Estado {
					r = i
				}
			}
			if r < rango {
				mejor, rango = a, r
			}
		}
		return mejor
	}
	d.Cadena = peor([]string{"no_valida", "no_comprobada", "valida"}, func(f Firmante) Aspecto { return f.Cadena })
	d.Certificado = peor([]string{"uso_no_permitido", "no_vigente", "no_comprobado", "vigente"}, func(f Firmante) Aspecto { return f.Certificado })
	d.Revocacion = peor([]string{"revocado", "no_comprobada", "vigente"}, func(f Firmante) Aspecto { return f.Revocacion })
	d.SelloTiempo = peor([]string{"no_valido", "no_comprobado", "no_presente", "valido"}, func(f Firmante) Aspecto { return f.SelloTiempo })
	if len(d.Firmantes) == 1 {
		d.CertificadoHuellaSHA256 = d.Firmantes[0].CertificadoHuellaSHA256
	}
	if d.fijado {
		return
	}
	d.Estado, d.Motivo = "valida", "verificada"
	switch {
	case d.Integridad.Estado == "no_valida":
		d.Estado, d.Motivo = "no_valida", "integridad_no_valida"
	case d.Certificado.Estado == "no_vigente" || d.Certificado.Estado == "uso_no_permitido" || d.Revocacion.Estado == "revocado":
		d.Estado, d.Motivo = "no_valida", "certificado_no_valido"
	case d.Cadena.Estado == "no_valida":
		d.Estado, d.Motivo = "no_valida", "confianza_no_valida"
	case d.Integridad.Estado != "valida":
		d.Estado, d.Motivo = "indeterminada", "integridad_parcial"
	case d.Certificado.Estado != "vigente":
		d.Estado, d.Motivo = "indeterminada", "certificado_no_acreditado"
	case d.Cadena.Estado != "valida":
		d.Estado, d.Motivo = "indeterminada", "confianza_no_acreditada"
	case d.Revocacion.Estado != "vigente":
		d.Estado, d.Motivo = "indeterminada", "revocacion_no_acreditada"
	case d.SelloTiempo.Estado == "no_valido":
		d.Estado, d.Motivo = "indeterminada", "sello_tiempo_no_acreditado"
	case d.VinculoOriginal.Estado == "no_acreditado":
		d.Estado, d.Motivo = "indeterminada", "vinculo_original_no_acreditado"
	}
}

func huella(b []byte) string {
	suma := sha256.Sum256(b)
	return hex.EncodeToString(suma[:])
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
