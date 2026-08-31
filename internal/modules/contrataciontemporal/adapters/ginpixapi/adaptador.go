package ginpixapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/ginpixfichero"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	maximosReintentosAPI       = 4
	maximoBytesRespuestaAPI    = int64(256 * 1024)
	maximoBytesAutorizacionAPI = 4 * 1024
)

var (
	ErrConfiguracionAPIGINPIXInvalida = errors.New(
		"contratacion temporal: configuracion api ginpix invalida",
	)
	ErrAutenticacionAPIGINPIXFallida = errors.New(
		"contratacion temporal: autenticacion api ginpix fallida",
	)
	ErrOperacionAPIGINPIXRechazada = errors.New(
		"contratacion temporal: operacion api ginpix rechazada",
	)
	ErrOperacionAPIGINPIXIndeterminada = errors.New(
		"contratacion temporal: operacion api ginpix indeterminada",
	)
	ErrConsultaAPIGINPIXNoDisponible = errors.New(
		"contratacion temporal: consulta api ginpix no disponible",
	)
	ErrOperacionAPIGINPIXNoConfirmada = errors.New(
		"contratacion temporal: operacion api ginpix no confirmada",
	)
	ErrRespuestaAPIGINPIXIncompatible = errors.New(
		"contratacion temporal: respuesta api ginpix incompatible",
	)
)

// ProveedorAutenticacionOpaca aplica autenticación sin entregar la credencial
// al constructor ni conservarla en el adaptador. Solo puede añadir una
// cabecera Authorization; el resto de la petición se vuelve a comprobar.
type ProveedorAutenticacionOpaca interface {
	Autorizar(context.Context, *http.Request) error
}

// Politica gobierna el plazo total, las pausas entre intentos y el límite de
// respuesta. Las pausas se copian al construir el adaptador.
type Politica struct {
	Referencia           string
	Version              uint64
	HuellaSHA256         string
	TiempoMaximo         time.Duration
	EsperasReintento     []time.Duration
	MaximoBytesRespuesta int64
}

// Configuracion recibe URL exactas para no compilar rutas del proveedor.
type Configuracion struct {
	URLEnvio    string
	URLConsulta string
	Politica    Politica
}

type Adaptador struct {
	transporte   http.RoundTripper
	autenticador ProveedorAutenticacionOpaca
	urlEnvio     url.URL
	urlConsulta  url.URL
	politica     Politica
}

func Nuevo(
	configuracion Configuracion,
	transporte http.RoundTripper,
	autenticador ProveedorAutenticacionOpaca,
) (*Adaptador, error) {
	urlEnvio, errEnvio := urlProveedor(configuracion.URLEnvio)
	urlConsulta, errConsulta := urlProveedor(configuracion.URLConsulta)
	if dependenciaNula(transporte) || dependenciaNula(autenticador) ||
		errEnvio != nil || errConsulta != nil ||
		urlEnvio.Scheme != urlConsulta.Scheme || urlEnvio.Host != urlConsulta.Host ||
		!politicaValida(configuracion.Politica) {
		return nil, ErrConfiguracionAPIGINPIXInvalida
	}
	politica := configuracion.Politica
	politica.EsperasReintento = append([]time.Duration(nil), politica.EsperasReintento...)
	return &Adaptador{
		transporte: transporte, autenticador: autenticador,
		urlEnvio: *urlEnvio, urlConsulta: *urlConsulta, politica: politica,
	}, nil
}

// Enviar devuelve éxito únicamente con un recibo completo. Tras emitir un
// intento, cualquier resultado no acreditado se clasifica como indeterminado.
func (a *Adaptador) Enviar(ctx context.Context, preparacion Preparacion) (ReciboExterno, error) {
	return a.ejecutar(ctx, preparacion, claseEnvio)
}

// Consultar usa un cuerpo minimizado de ligaduras; nunca reenvía los campos
// funcionales de la carga.
func (a *Adaptador) Consultar(ctx context.Context, preparacion Preparacion) (ReciboExterno, error) {
	return a.ejecutar(ctx, preparacion, claseConsulta)
}

type clasePeticion uint8

const (
	claseEnvio clasePeticion = iota + 1
	claseConsulta
)

func (a *Adaptador) ejecutar(
	ctx context.Context,
	preparacion Preparacion,
	clase clasePeticion,
) (ReciboExterno, error) {
	if a == nil || ctx == nil || dependenciaNula(a.transporte) ||
		dependenciaNula(a.autenticador) || !politicaValida(a.politica) ||
		preparacion.Validar() != nil || (clase != claseEnvio && clase != claseConsulta) {
		return ReciboExterno{}, ErrConfiguracionAPIGINPIXInvalida
	}
	if err := ctx.Err(); err != nil {
		return ReciboExterno{}, err
	}
	operacion, cancelar := context.WithTimeout(ctx, a.politica.TiempoMaximo)
	defer cancelar()

	emitida := false
	for intento := 0; intento <= len(a.politica.EsperasReintento); intento++ {
		if err := operacion.Err(); err != nil {
			return ReciboExterno{}, errorContextoOperacion(err, clase, emitida)
		}
		peticion, err := a.nuevaPeticion(operacion, preparacion, clase)
		if err != nil {
			if errContexto := operacion.Err(); errContexto != nil {
				return ReciboExterno{}, errorContextoOperacion(errContexto, clase, emitida)
			}
			return ReciboExterno{}, ErrAutenticacionAPIGINPIXFallida
		}
		emitida = true
		respuesta, errTransporte := a.transporte.RoundTrip(peticion)
		if errTransporte != nil {
			if respuesta != nil {
				cerrarRespuesta(respuesta)
				return ReciboExterno{}, errorSinResultado(clase)
			}
			cerrarRespuesta(respuesta)
			if errContexto := operacion.Err(); errContexto != nil {
				return ReciboExterno{}, errorContextoOperacion(errContexto, clase, emitida)
			}
			if intento < len(a.politica.EsperasReintento) {
				if err := esperarReintento(operacion, a.politica.EsperasReintento[intento]); err != nil {
					return ReciboExterno{}, errorContextoOperacion(err, clase, emitida)
				}
				continue
			}
			return ReciboExterno{}, errorSinResultado(clase)
		}
		if respuesta == nil {
			return ReciboExterno{}, errorSinResultado(clase)
		}
		if codigoReintentable(respuesta.StatusCode) {
			cerrarRespuesta(respuesta)
			if intento < len(a.politica.EsperasReintento) {
				if err := esperarReintento(operacion, a.politica.EsperasReintento[intento]); err != nil {
					return ReciboExterno{}, errorContextoOperacion(err, clase, emitida)
				}
				continue
			}
			return ReciboExterno{}, errorSinResultado(clase)
		}
		if !codigoExito(respuesta.StatusCode, clase) {
			codigo := respuesta.StatusCode
			cerrarRespuesta(respuesta)
			return ReciboExterno{}, errorCodigoNoExitoso(codigo, clase)
		}
		recibo, err := leerRecibo(respuesta, preparacion, a.politica.MaximoBytesRespuesta)
		if err != nil {
			return ReciboExterno{}, errorRespuestaInvalida(clase)
		}
		// El recibo completo prueba el resultado aunque la cancelación se
		// observe después de recibir y validar todos sus bytes.
		return recibo, nil
	}
	return ReciboExterno{}, errorSinResultado(clase)
}

type consultaOperacion struct {
	VersionExpediente      uint64 `json:"version_expediente"`
	ExpedienteRef          string `json:"expediente_ref"`
	IncorporacionRef       string `json:"incorporacion_ref"`
	CorrelacionRef         string `json:"correlacion_ref"`
	IdempotenciaRef        string `json:"idempotencia_ref"`
	ModeloHuellaSHA256     string `json:"modelo_huella_sha256"`
	MapeoRef               string `json:"mapeo_ref"`
	MapeoVersion           uint64 `json:"mapeo_version"`
	MapeoHuellaSHA256      string `json:"mapeo_huella_sha256"`
	CargaHuellaSHA256      string `json:"carga_huella_sha256"`
	CuerpoHuellaSHA256     string `json:"cuerpo_huella_sha256"`
	ReciboIncorporacionRef string `json:"recibo_incorporacion_ref"`
	ResultadoPersonalRef   string `json:"resultado_personal_ref"`
	ReciboPersonalRef      string `json:"recibo_personal_ref"`
}

func (a *Adaptador) nuevaPeticion(
	ctx context.Context,
	preparacion Preparacion,
	clase clasePeticion,
) (*http.Request, error) {
	metadatos, err := preparacion.Metadatos()
	if err != nil {
		return nil, ErrPreparacionAPIGINPIXInvalida
	}
	var destino url.URL
	var cuerpo []byte
	if clase == claseEnvio {
		destino = a.urlEnvio
		cuerpo, err = preparacion.Cuerpo()
	} else {
		destino = a.urlConsulta
		cuerpo, err = json.Marshal(consultaOperacion{
			VersionExpediente: metadatos.VersionExpediente,
			ExpedienteRef:     metadatos.ExpedienteRef, IncorporacionRef: metadatos.IncorporacionRef,
			CorrelacionRef: metadatos.CorrelacionRef, IdempotenciaRef: metadatos.IdempotenciaRef,
			ModeloHuellaSHA256: metadatos.ModeloHuellaSHA256,
			MapeoRef:           metadatos.MapeoRef, MapeoVersion: metadatos.MapeoVersion,
			MapeoHuellaSHA256:      metadatos.MapeoHuellaSHA256,
			CargaHuellaSHA256:      metadatos.CargaHuellaSHA256,
			CuerpoHuellaSHA256:     metadatos.CuerpoHuellaSHA256,
			ReciboIncorporacionRef: metadatos.ReciboIncorporacionRef,
			ResultadoPersonalRef:   metadatos.ResultadoPersonalRef,
			ReciboPersonalRef:      metadatos.ReciboPersonalRef,
		})
	}
	if err != nil || len(cuerpo) == 0 || len(cuerpo) > ginpixficheroMaximoSolicitud() {
		return nil, ErrPreparacionAPIGINPIXInvalida
	}
	peticion := (&http.Request{
		Method: http.MethodPost, URL: &destino, Header: make(http.Header), Body: http.NoBody,
	}).WithContext(ctx)
	peticion.Header.Set("Accept", "application/json")
	peticion.Header.Set("Accept-Encoding", "identity")
	peticion.Header.Set("Cache-Control", "no-store")
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Idempotency-Key", metadatos.IdempotenciaRef)
	peticion.Header.Set("X-Correlation-ID", metadatos.CorrelacionRef)
	metodo, urlExacta := peticion.Method, peticion.URL.String()
	if err := a.autenticador.Autorizar(ctx, peticion); err != nil ||
		!peticionAutenticadaValida(peticion, metodo, urlExacta, metadatos) {
		return nil, ErrAutenticacionAPIGINPIXFallida
	}
	peticion.Body = io.NopCloser(bytes.NewReader(cuerpo))
	peticion.ContentLength = int64(len(cuerpo))
	return peticion, nil
}

func peticionAutenticadaValida(
	peticion *http.Request,
	metodo, urlExacta string,
	metadatos MetadatosOperacion,
) bool {
	if peticion == nil || peticion.Method != metodo || peticion.URL == nil ||
		peticion.URL.String() != urlExacta || peticion.Body != http.NoBody ||
		peticion.ContentLength != 0 || peticion.Host != "" || peticion.Close ||
		len(peticion.TransferEncoding) != 0 || len(peticion.Trailer) != 0 {
		return false
	}
	esperadas := map[string]string{
		"Accept": "application/json", "Accept-Encoding": "identity", "Cache-Control": "no-store",
		"Content-Type": "application/json", "Idempotency-Key": metadatos.IdempotenciaRef,
		"X-Correlation-Id": metadatos.CorrelacionRef,
	}
	for nombre, valores := range peticion.Header {
		if nombre == "Authorization" {
			if len(valores) != 1 || !valorAutorizacionValido(valores[0]) {
				return false
			}
			continue
		}
		esperado, existe := esperadas[nombre]
		if !existe || len(valores) != 1 || valores[0] != esperado {
			return false
		}
	}
	if len(peticion.Header) != len(esperadas)+1 {
		return false
	}
	for nombre, esperado := range esperadas {
		if valores := peticion.Header.Values(nombre); len(valores) != 1 || valores[0] != esperado {
			return false
		}
	}
	return len(peticion.Header.Values("Authorization")) == 1
}

func valorAutorizacionValido(valor string) bool {
	if len(valor) == 0 || len(valor) > maximoBytesAutorizacionAPI ||
		valor != strings.TrimSpace(valor) || !utf8.ValidString(valor) {
		return false
	}
	for _, caracter := range valor {
		if caracter < 0x20 || caracter == 0x7f {
			return false
		}
	}
	return true
}

func leerRecibo(
	respuesta *http.Response,
	preparacion Preparacion,
	limite int64,
) (ReciboExterno, error) {
	if respuesta == nil || respuesta.Body == nil || !contentTypeJSON(respuesta.Header) ||
		len(respuesta.Header.Values("Set-Cookie")) != 0 || respuesta.ContentLength < -1 ||
		respuesta.ContentLength > limite {
		cerrarRespuesta(respuesta)
		return ReciboExterno{}, ErrRespuestaAPIGINPIXIncompatible
	}
	lector := io.LimitReader(respuesta.Body, limite+1)
	contenido, errLectura := io.ReadAll(lector)
	errCierre := respuesta.Body.Close()
	if errLectura != nil || errCierre != nil || len(contenido) == 0 || int64(len(contenido)) > limite {
		return ReciboExterno{}, ErrRespuestaAPIGINPIXIncompatible
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var datos DatosReciboExterno
	if !objetoReciboJSONCerrado(contenido) || decodificador.Decode(&datos) != nil {
		return ReciboExterno{}, ErrRespuestaAPIGINPIXIncompatible
	}
	var adicional any
	if err := decodificador.Decode(&adicional); !errors.Is(err, io.EOF) {
		return ReciboExterno{}, ErrRespuestaAPIGINPIXIncompatible
	}
	return nuevoReciboExterno(datos, preparacion)
}

func objetoReciboJSONCerrado(contenido []byte) bool {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	inicio, err := decodificador.Token()
	if err != nil || inicio != json.Delim('{') {
		return false
	}
	vistos := make(map[string]struct{}, 19)
	for decodificador.More() {
		token, err := decodificador.Token()
		nombre, esNombre := token.(string)
		if err != nil || !esNombre || !campoReciboPermitido(nombre) {
			return false
		}
		if _, repetido := vistos[nombre]; repetido {
			return false
		}
		vistos[nombre] = struct{}{}
		var valor json.RawMessage
		if decodificador.Decode(&valor) != nil {
			return false
		}
	}
	fin, err := decodificador.Token()
	if err != nil || fin != json.Delim('}') || len(vistos) != 19 {
		return false
	}
	return decodificador.Decode(new(any)) == io.EOF
}

func campoReciboPermitido(nombre string) bool {
	switch nombre {
	case "esquema", "version", "recibo_externo_ref", "evidencia_externa_ref",
		"evidencia_externa_huella_sha256", "version_expediente", "expediente_ref",
		"incorporacion_ref", "correlacion_ref", "idempotencia_ref",
		"modelo_huella_sha256", "mapeo_ref", "mapeo_version", "mapeo_huella_sha256",
		"carga_huella_sha256", "cuerpo_huella_sha256", "recibo_incorporacion_ref",
		"resultado_personal_ref", "recibo_personal_ref":
		return true
	default:
		return false
	}
}

func contentTypeJSON(cabeceras http.Header) bool {
	valores := cabeceras.Values("Content-Type")
	if len(valores) != 1 {
		return false
	}
	tipo, parametros, err := mime.ParseMediaType(valores[0])
	if err != nil || tipo != "application/json" || len(parametros) > 1 {
		return false
	}
	charset, existe := parametros["charset"]
	return !existe || strings.EqualFold(charset, "utf-8")
}

func cerrarRespuesta(respuesta *http.Response) {
	if respuesta == nil || respuesta.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(respuesta.Body, 4*1024))
	_ = respuesta.Body.Close()
}

func esperarReintento(ctx context.Context, espera time.Duration) error {
	if espera == 0 {
		return ctx.Err()
	}
	temporizador := time.NewTimer(espera)
	defer temporizador.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-temporizador.C:
		return nil
	}
}

func codigoReintentable(codigo int) bool {
	return codigo == http.StatusTooManyRequests || codigo >= 500 && codigo <= 599
}

func codigoExito(codigo int, clase clasePeticion) bool {
	if clase == claseConsulta {
		return codigo == http.StatusOK
	}
	return codigo == http.StatusOK || codigo == http.StatusCreated
}

func errorCodigoNoExitoso(codigo int, clase clasePeticion) error {
	if clase == claseConsulta {
		if codigo == http.StatusNotFound {
			return ErrOperacionAPIGINPIXNoConfirmada
		}
		return ErrConsultaAPIGINPIXNoDisponible
	}
	if codigo >= 400 && codigo <= 499 {
		return ErrOperacionAPIGINPIXRechazada
	}
	return ErrOperacionAPIGINPIXIndeterminada
}

func errorSinResultado(clase clasePeticion) error {
	if clase == claseConsulta {
		return ErrConsultaAPIGINPIXNoDisponible
	}
	return ErrOperacionAPIGINPIXIndeterminada
}

func errorRespuestaInvalida(clase clasePeticion) error {
	if clase == claseConsulta {
		return ErrRespuestaAPIGINPIXIncompatible
	}
	return ErrOperacionAPIGINPIXIndeterminada
}

func errorContextoOperacion(err error, clase clasePeticion, emitida bool) error {
	if clase == claseEnvio && emitida {
		return errors.Join(ErrOperacionAPIGINPIXIndeterminada, err)
	}
	return err
}

func politicaValida(p Politica) bool {
	if !domain.ReferenciaOpacaValida(p.Referencia) || p.Version == 0 ||
		!huellaValida(p.HuellaSHA256) || p.TiempoMaximo <= 0 || p.MaximoBytesRespuesta <= 0 ||
		p.MaximoBytesRespuesta > maximoBytesRespuestaAPI ||
		len(p.EsperasReintento) > maximosReintentosAPI {
		return false
	}
	for _, espera := range p.EsperasReintento {
		if espera < 0 || espera >= p.TiempoMaximo {
			return false
		}
	}
	return true
}

func urlProveedor(valor string) (*url.URL, error) {
	u, err := url.Parse(valor)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil ||
		u.Opaque != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" ||
		u.RawPath != "" || u.Path == "" || !strings.HasPrefix(u.Path, "/") ||
		path.Clean(u.Path) != u.Path {
		return nil, ErrConfiguracionAPIGINPIXInvalida
	}
	return u, nil
}

func dependenciaNula(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func ginpixficheroMaximoSolicitud() int {
	return ginpixfichero.MaximoBytesFicheroGINPIX
}
