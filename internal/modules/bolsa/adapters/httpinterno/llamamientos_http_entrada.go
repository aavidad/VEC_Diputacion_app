package httpinterno

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const maximoCuerpoSolicitudLlamamientoBytes = 4 * 1024

var (
	errEntradaLlamamientoInvalida = errors.New(
		"bolsa http interno: entrada de propuesta de llamamiento invalida",
	)
	errEntradaLlamamientoDemasiadoGrande = errors.New(
		"bolsa http interno: entrada de propuesta de llamamiento demasiado grande",
	)
)

type envelopeSolicitudPropuestaLlamamientoJSON struct {
	Data *solicitudPropuestaLlamamientoJSON `json:"data"`
}

type solicitudPropuestaLlamamientoJSON struct {
	Esquema     string `json:"esquema"`
	NecesidadID string `json:"necesidad_id"`
}

func metadatosPropuestaLlamamientoPermitidos(r *http.Request) bool {
	if r == nil || r.URL == nil || r.Body == nil || r.Body == http.NoBody ||
		r.ContentLength <= 0 ||
		len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 {
		return false
	}
	if !cabeceraExactaLlamamiento(r.Header, "Accept", "application/json") ||
		!cabeceraEnConjuntoLlamamiento(
			r.Header,
			"Content-Type",
			"application/json",
			"application/json; charset=utf-8",
		) {
		return false
	}
	for _, nombre := range []string{
		"Cookie", "Proxy-Authorization", "Proxy-Connection", "Forwarded", "Via",
		"Trailer", "TE", "Transfer-Encoding", "Content-Encoding", "Expect",
		"If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since", "Range",
	} {
		if cabeceraPresente(r.Header, nombre) {
			return false
		}
	}
	return !cabeceraEntornoNoConfiableLlamamientoPresente(r.Header)
}

func cabeceraEntornoNoConfiableLlamamientoPresente(cabeceras http.Header) bool {
	if cabeceraIdentidadHeredadaPresente(cabeceras) {
		return true
	}
	for nombre := range cabeceras {
		minusculas := strings.ToLower(nombre)
		if minusculas == "x-real-ip" || minusculas == "x-original-url" ||
			minusculas == "x-original-uri" || minusculas == "x-rewrite-url" ||
			minusculas == "x-http-method-override" || minusculas == "cf-connecting-ip" ||
			minusculas == "true-client-ip" || minusculas == "fastly-client-ip" ||
			strings.HasPrefix(minusculas, "x-envoy-") || strings.HasPrefix(minusculas, "x-original-") {
			return true
		}
	}
	return false
}

func cabeceraExactaLlamamiento(cabeceras http.Header, nombre, esperada string) bool {
	valor, err := cabeceraUnicaLlamamiento(cabeceras, nombre)
	return err == nil && valor == esperada
}

func cabeceraEnConjuntoLlamamiento(cabeceras http.Header, nombre string, permitidas ...string) bool {
	valor, err := cabeceraUnicaLlamamiento(cabeceras, nombre)
	if err != nil {
		return false
	}
	for _, permitida := range permitidas {
		if valor == permitida {
			return true
		}
	}
	return false
}

func cabeceraUnicaLlamamiento(cabeceras http.Header, nombre string) (string, error) {
	valores := make([]string, 0, 1)
	for recibido, lista := range cabeceras {
		if strings.EqualFold(recibido, nombre) {
			valores = append(valores, lista...)
		}
	}
	if len(valores) != 1 || valores[0] == "" || valores[0] != strings.TrimSpace(valores[0]) {
		return "", errEntradaLlamamientoInvalida
	}
	return valores[0], nil
}

func entradaPropuestaLlamamientoDesdePeticion(
	w http.ResponseWriter,
	r *http.Request,
) (EntradaPreparacionPropuestaLlamamientoInterno, error) {
	var envelope envelopeSolicitudPropuestaLlamamientoJSON
	if err := decodificarCuerpoLlamamiento(w, r, &envelope); err != nil || envelope.Data == nil {
		return EntradaPreparacionPropuestaLlamamientoInterno{}, errors.Join(errEntradaLlamamientoInvalida, err)
	}
	datos := envelope.Data
	if datos.Esquema != "vec.bolsa.propuesta-llamamiento.solicitud.v1" ||
		!puertosbolsa.ReferenciaOpacaLlamamientoValida(datos.NecesidadID) ||
		!norm.NFC.IsNormalString(datos.NecesidadID) {
		return EntradaPreparacionPropuestaLlamamientoInterno{}, errEntradaLlamamientoInvalida
	}
	return EntradaPreparacionPropuestaLlamamientoInterno{NecesidadRef: datos.NecesidadID}, nil
}

func decodificarCuerpoLlamamiento(w http.ResponseWriter, r *http.Request, destino any) error {
	lector := http.MaxBytesReader(w, r.Body, maximoCuerpoSolicitudLlamamientoBytes+1)
	contenido, err := io.ReadAll(lector)
	if err != nil {
		var demasiadoGrande *http.MaxBytesError
		if errors.As(err, &demasiadoGrande) {
			return errEntradaLlamamientoDemasiadoGrande
		}
		return errEntradaLlamamientoInvalida
	}
	if len(contenido) == 0 {
		return errEntradaLlamamientoInvalida
	}
	if len(contenido) > maximoCuerpoSolicitudLlamamientoBytes {
		return errEntradaLlamamientoDemasiadoGrande
	}
	if !utf8.Valid(contenido) {
		return errEntradaLlamamientoInvalida
	}
	if err := validarJSONSinDuplicados(contenido); err != nil {
		if errors.Is(err, errEntradaBorradorDemasiadoGrande) {
			return errEntradaLlamamientoDemasiadoGrande
		}
		return errEntradaLlamamientoInvalida
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return errEntradaLlamamientoInvalida
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return errEntradaLlamamientoInvalida
	}
	return nil
}

func responderErrorEntradaLlamamiento(w http.ResponseWriter, err error, correlacion string) {
	if errors.Is(err, errEntradaLlamamientoDemasiadoGrande) {
		responderErrorLlamamiento(w, http.StatusRequestEntityTooLarge, "peticion_demasiado_grande", correlacion)
		return
	}
	responderErrorLlamamiento(w, http.StatusBadRequest, "peticion_no_valida", correlacion)
}
