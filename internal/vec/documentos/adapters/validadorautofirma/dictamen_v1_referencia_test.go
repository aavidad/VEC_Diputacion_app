// Estas referencias v1 conservan la regresion historica del contrato publico.
// No forman parte del cliente compilado ni permiten una caida silenciosa a v1.
package validadorautofirma

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"io"
	"strings"
	"vec-diputacion-granada/internal/vec/documentos/ports"
)

const contratoDictamenV1 = "autofirmav2.dictamen-verificacion.v1"
const maximoFirmantes = 16

// respuestaAutofirma toma el dictamen en bruto; los campos heredados de
// primer nivel se ignoran. El dictamen se decodifica despues con
// DisallowUnknownFields contra el contrato v1 completo.
type respuestaAutofirma struct {
	Dictamen json.RawMessage `json:"dictamen"`
}

// dictamenAutofirma enumera todas las claves del contrato v1. Las que VEC no
// usa (formato, fecha de comprobacion, asunto, emisor, serie, motivos
// tecnicos, fuentes y fechas de aspecto) se aceptan como json.RawMessage para
// que DisallowUnknownFields no las rechace, y se descartan sin interpretarlas.
type dictamenAutofirma struct {
	Contrato                string              `json:"contrato"`
	Estado                  string              `json:"estado"`
	Motivo                  string              `json:"motivo"`
	Formato                 string              `json:"formato"`
	ComprobadoEn            json.RawMessage     `json:"comprobadoEn"`
	Integridad              aspectoAutofirma    `json:"integridad"`
	Cadena                  aspectoAutofirma    `json:"cadena"`
	Certificado             aspectoAutofirma    `json:"certificado"`
	Revocacion              aspectoAutofirma    `json:"revocacion"`
	SelloTiempo             aspectoAutofirma    `json:"selloTiempo"`
	VinculoOriginal         aspectoAutofirma    `json:"vinculoOriginal"`
	HuellaFirmadoSHA256     string              `json:"huellaFirmadoSHA256"`
	HuellaOriginalSHA256    string              `json:"huellaOriginalSHA256"`
	CertificadoHuellaSHA256 string              `json:"certificadoHuellaSHA256"`
	Firmantes               []firmanteAutofirma `json:"firmantes"`
	Extensiones             extensionesDictamen `json:"extensiones"`
}

type aspectoAutofirma struct {
	Estado string          `json:"estado"`
	Motivo json.RawMessage `json:"motivo"`
	Fuente json.RawMessage `json:"fuente"`
	Fecha  json.RawMessage `json:"fecha"`
}

type firmanteAutofirma struct {
	CertificadoHuellaSHA256 string           `json:"certificadoHuellaSHA256"`
	Serie                   json.RawMessage  `json:"serie"`
	Asunto                  json.RawMessage  `json:"asunto"`
	Emisor                  json.RawMessage  `json:"emisor"`
	Cadena                  aspectoAutofirma `json:"cadena"`
	Certificado             aspectoAutofirma `json:"certificado"`
	Revocacion              aspectoAutofirma `json:"revocacion"`
	SelloTiempo             aspectoAutofirma `json:"selloTiempo"`
}

type extensionesDictamen struct {
	RevocacionRemota  string `json:"revocacionRemota"`
	SelloTiempoRemoto string `json:"selloTiempoRemoto"`
}

// Catalogos cerrados del contrato v1. Son constantes de lectura: no se
// modifican en ejecucion.
var (
	estadosIntegridad  = []string{"valida", "parcial", "no_valida"}
	estadosCadena      = []string{"valida", "no_valida", "no_comprobada"}
	estadosCertificado = []string{"vigente", "no_vigente", "uso_no_permitido", "no_comprobado"}
	estadosRevocacion  = []string{ports.RevocacionVigente, ports.RevocacionRevocado, ports.RevocacionNoComprobada}
	estadosSello       = []string{ports.SelloTiempoNoPresente, ports.SelloTiempoValido,
		ports.SelloTiempoNoValido, ports.SelloTiempoNoComprobado}
	estadosVinculo = []string{"acreditado", "no_acreditado", "no_aportado"}
	// Ordenes de agregacion del contrato v1, de peor a mejor. El agregado de
	// cada aspecto es el peor estado de sus firmantes; sin firmantes, el
	// estado por defecto del contrato.
	ordenCadena      = []string{"no_valida", "no_comprobada", "valida"}
	ordenCertificado = []string{"uso_no_permitido", "no_vigente", "no_comprobado", "vigente"}
	ordenRevocacion  = []string{ports.RevocacionRevocado, ports.RevocacionNoComprobada, ports.RevocacionVigente}
	ordenSello       = []string{ports.SelloTiempoNoValido, ports.SelloTiempoNoComprobado,
		ports.SelloTiempoNoPresente, ports.SelloTiempoValido}
	estadosExtension = []string{"desactivada", "activa"}
)

// decodificarRespuesta acepta exactamente un objeto JSON de hasta
// maximaRespuesta bytes, sin claves duplicadas en ningun nivel (tampoco por
// plegado de mayusculas, que encoding/json haria coincidir) y con un dictamen
// sin claves ajenas al contrato v1. Un dictamen ausente o nulo devuelve nil,
// que traducir tambien trata como no interpretable.
func decodificarRespuesta(contenido []byte) (*dictamenAutofirma, bool) {
	if len(contenido) > maximaRespuesta || !sinClavesDuplicadas(contenido) {
		return nil, false
	}
	lector := json.NewDecoder(bytes.NewReader(contenido))
	var recibida respuestaAutofirma
	if lector.Decode(&recibida) != nil || lector.Decode(new(any)) != io.EOF {
		return nil, false
	}
	if len(recibida.Dictamen) == 0 || string(recibida.Dictamen) == "null" {
		return nil, true
	}
	estricto := json.NewDecoder(bytes.NewReader(recibida.Dictamen))
	estricto.DisallowUnknownFields()
	dictamen := new(dictamenAutofirma)
	if estricto.Decode(dictamen) != nil || estricto.Decode(new(any)) != io.EOF {
		return nil, false
	}
	return dictamen, true
}

// traducir interpreta el dictamen con fallo cerrado. Primero valida contrato,
// catalogos y huellas de eco; despues recalcula el veredicto con la politica
// de VEC y lo contrasta con el declarado por el validador.
func traducir(r ports.ResultadoVerificacionFirma, d *dictamenAutofirma) ports.VerificacionFirmaMotivada {
	// VEC envia siempre el original: salvo que el validador declare no haberlo
	// recibido (`no_aportado`, sin eco), su eco es obligatorio y exacto.
	if !dictamenInterpretable(d) || d.HuellaFirmadoSHA256 != r.HuellaFirmadoSHA256 ||
		(d.VinculoOriginal.Estado == "no_aportado") != (d.HuellaOriginalSHA256 == "") ||
		(d.HuellaOriginalSHA256 != "" && d.HuellaOriginalSHA256 != r.HuellaOriginalSHA256) {
		return motivar(r, ports.MotivoRespuestaNoInterpretable)
	}
	declarado := motivosDictamen[d.Motivo]
	if string(declarado.EstadoAsociado()) != d.Estado {
		return motivar(r, ports.MotivoRespuestaNoInterpretable)
	}
	sinAdoptar := r
	r.VinculoOriginal = d.VinculoOriginal.Estado == "acreditado"
	r.Formato = d.Formato
	r.RevocacionEstado = d.Revocacion.Estado
	r.SelloTiempoEstado = d.SelloTiempo.Estado
	if len(d.Firmantes) == 1 && d.CertificadoHuellaSHA256 != "" {
		// La referencia del firmante es la huella de su certificado; su
		// correspondencia con la persona la resuelve la aplicacion.
		r.CertificadoHuellaSHA256 = d.CertificadoHuellaSHA256
		r.FirmanteRef = "ref:" + d.CertificadoHuellaSHA256
	}
	propio := veredictoVEC(d)
	switch {
	case propio == declarado:
		return motivar(r, propio)
	case declarado == ports.MotivoFirmaVerificada:
		// VEC es mas estricta que el validador (original no aportado o varios
		// firmantes): prevalece su motivo, nunca el positivo.
		return motivar(r, propio)
	default:
		// Un veredicto negativo que no casa con sus aspectos indica deriva
		// del contrato: no se adopta ningun aspecto de esa respuesta.
		return motivar(sinAdoptar, ports.MotivoRespuestaNoInterpretable)
	}
}

// veredictoVEC aplica ports.PoliticaVerificacionFirmaV1 sobre los aspectos
// agregados, con la misma precedencia que el contrato: defectos
// concluyentes antes que comprobaciones no concluidas. Solo es seguro porque
// dictamenInterpretable ya exigio que cada agregado sea exactamente el peor
// estado de los firmantes: ningun defecto de un firmante queda oculto.
func veredictoVEC(d *dictamenAutofirma) ports.MotivoVerificacionFirma {
	switch {
	case d.Integridad.Estado == "no_valida":
		return ports.MotivoIntegridadNoValida
	case d.Certificado.Estado == "no_vigente" || d.Certificado.Estado == "uso_no_permitido" ||
		d.Revocacion.Estado == ports.RevocacionRevocado:
		return ports.MotivoCertificadoNoValido
	case d.Cadena.Estado == "no_valida":
		return ports.MotivoConfianzaNoValida
	case d.Integridad.Estado != "valida":
		return ports.MotivoIntegridadParcial
	case len(d.Firmantes) == 0:
		return ports.MotivoFirmanteNoIdentificado
	case d.Certificado.Estado != "vigente":
		return ports.MotivoCertificadoNoAcreditado
	case d.Cadena.Estado != "valida":
		return ports.MotivoConfianzaNoAcreditada
	case d.Revocacion.Estado != ports.RevocacionVigente:
		return ports.MotivoRevocacionNoAcreditada
	case !ports.SelloTiempoAdmisible(d.SelloTiempo.Estado):
		return ports.MotivoSelloTiempoNoAcreditado
	case d.VinculoOriginal.Estado != "acreditado":
		return ports.MotivoVinculoOriginalNoAcreditado
	case len(d.Firmantes) != 1 || d.CertificadoHuellaSHA256 == "":
		return ports.MotivoFirmanteNoIdentificado
	default:
		return ports.MotivoFirmaVerificada
	}
}

// dictamenInterpretable exige contrato exacto, estados de catalogo en todos
// los aspectos y firmantes, y huellas bien formadas y coherentes.
func dictamenInterpretable(d *dictamenAutofirma) bool {
	if d == nil || d.Contrato != contratoDictamenV1 ||
		!en(d.Integridad.Estado, estadosIntegridad) || !en(d.Cadena.Estado, estadosCadena) ||
		!en(d.Certificado.Estado, estadosCertificado) || !en(d.Revocacion.Estado, estadosRevocacion) ||
		!en(d.SelloTiempo.Estado, estadosSello) || !en(d.VinculoOriginal.Estado, estadosVinculo) ||
		!en(d.Extensiones.RevocacionRemota, estadosExtension) ||
		!en(d.Extensiones.SelloTiempoRemoto, estadosExtension) ||
		len(d.Firmantes) > maximoFirmantes || !huellaValida(d.HuellaFirmadoSHA256) ||
		(d.HuellaOriginalSHA256 != "" && !huellaValida(d.HuellaOriginalSHA256)) {
		return false
	}
	if _, ok := motivosDictamen[d.Motivo]; !ok {
		return false
	}
	for _, f := range d.Firmantes {
		if !huellaValida(f.CertificadoHuellaSHA256) || !en(f.Cadena.Estado, estadosCadena) ||
			!en(f.Certificado.Estado, estadosCertificado) || !en(f.Revocacion.Estado, estadosRevocacion) ||
			!en(f.SelloTiempo.Estado, estadosSello) {
			return false
		}
	}
	// Coherencia exacta firmantes-agregados: con un firmante, sus cuatro
	// aspectos son los agregados; con varios, cada agregado es el peor.
	if d.Cadena.Estado != agregado(d.Firmantes, func(f firmanteAutofirma) string { return f.Cadena.Estado }, ordenCadena, "no_comprobada") ||
		d.Certificado.Estado != agregado(d.Firmantes, func(f firmanteAutofirma) string { return f.Certificado.Estado }, ordenCertificado, "no_comprobado") ||
		d.Revocacion.Estado != agregado(d.Firmantes, func(f firmanteAutofirma) string { return f.Revocacion.Estado }, ordenRevocacion, ports.RevocacionNoComprobada) ||
		d.SelloTiempo.Estado != agregado(d.Firmantes, func(f firmanteAutofirma) string { return f.SelloTiempo.Estado }, ordenSello, ports.SelloTiempoNoPresente) {
		return false
	}
	switch {
	case d.CertificadoHuellaSHA256 == "":
		return len(d.Firmantes) != 1
	case len(d.Firmantes) != 1:
		return false
	default:
		return huellaValida(d.CertificadoHuellaSHA256) &&
			d.CertificadoHuellaSHA256 == d.Firmantes[0].CertificadoHuellaSHA256 &&
			d.CertificadoHuellaSHA256 != strings.Repeat("0", sha256.Size*2)
	}
}

// agregado devuelve el peor estado de los firmantes segun orden (de peor a
// mejor), o porDefecto sin firmantes, como compone el contrato v1. Los
// estados ya se validaron contra catalogo, que coincide con el orden.
func agregado(firmantes []firmanteAutofirma, estado func(firmanteAutofirma) string, orden []string, porDefecto string) string {
	if len(firmantes) == 0 {
		return porDefecto
	}
	peor := len(orden)
	for _, f := range firmantes {
		for i, candidato := range orden {
			if candidato == estado(f) && i < peor {
				peor = i
			}
		}
	}
	if peor == len(orden) {
		return ""
	}
	return orden[peor]
}

func en(valor string, catalogo []string) bool {
	for _, c := range catalogo {
		if valor == c {
			return true
		}
	}
	return false
}

func huellaValida(s string) bool {
	if len(s) != sha256.Size*2 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if (s[i] < '0' || s[i] > '9') && (s[i] < 'a' || s[i] > 'f') {
			return false
		}
	}
	return true
}
