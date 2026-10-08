package httpinterno

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	MaximoCuerpoAltaBytes      = 256 * 1024
	MaximoDocumentosAdjuntos   = 64
	MaximoCaracteresDetalle    = 4000
	MaximoCaracteresReferencia = 160
	MaximoAniosPeriodoAlta     = 100
	MaximoCentimosJSON         = int64(922_337_203_685_477)
	MaximoVersionJSON          = uint64(9_007_199_254_740_991)
	profundidadMaximaJSONAlta  = 8
	tokensMaximosJSONAlta      = 1024
)

var (
	errEntradaAltaInvalida       = errors.New("contratacion temporal http: entrada invalida")
	errContenidoAltaNoValido     = errors.New("contratacion temporal http: contenido no valido")
	errCuerpoAltaDemasiadoGrande = errors.New("contratacion temporal http: cuerpo demasiado grande")
	patronEnteroJSONAlta         = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)
)

type solicitudCentroJSON struct {
	CentroRef          string               `json:"centro_ref"`
	ContactoRef        string               `json:"contacto_ref"`
	CategoriaRef       string               `json:"categoria_ref"`
	GrupoSubgrupo      string               `json:"grupo_subgrupo"`
	MotivoClave        string               `json:"motivo_clave"`
	Detalle            string               `json:"detalle"`
	Periodo            *periodoPrevistoJSON `json:"periodo"`
	RC                 *declaracionRCJSON   `json:"rc"`
	DocumentosAdjuntos *[]string            `json:"documentos_adjuntos"`
	Observaciones      cadenaOpcionalJSON   `json:"observaciones"`
}

// solicitudAltaJSON es un sobre de datos no autoritativos. La clave solo
// identifica una intención de reintento; identidad, perfil y organización
// continúan procediendo exclusivamente del contexto confiable del servidor.
type solicitudAltaJSON struct {
	NumeroExpedienteMOAD string               `json:"numero_expediente_moad"`
	ClaveIdempotencia    string               `json:"clave_idempotencia"`
	Solicitud            *solicitudCentroJSON `json:"solicitud"`
}

// solicitudAltaVersionadaJSON añade la necesidad sólo a la entrada v3; el
// DTO v1 anterior conserva sus claves y su OpenAPI exactos.
type solicitudAltaVersionadaJSON struct {
	Esquema              string               `json:"esquema"`
	NumeroExpedienteMOAD string               `json:"numero_expediente_moad"`
	ClaveIdempotencia    string               `json:"clave_idempotencia"`
	Solicitud            *solicitudCentroJSON `json:"solicitud"`
	Necesidad            *necesidadAltaJSON   `json:"necesidad,omitempty"`
}

// necesidadAltaJSON sólo acepta hechos estructurados del centro. La
// instantánea del catálogo se añade en aplicación desde una fuente confiable.
type necesidadAltaJSON struct {
	Esquema              string            `json:"esquema"`
	CatalogoRef          string            `json:"catalogo_ref"`
	CatalogoVersion      uint64            `json:"catalogo_version"`
	CatalogoHuellaSHA256 string            `json:"catalogo_huella_sha256"`
	CausaClave           string            `json:"causa_clave"`
	JornadaMinutos       uint16            `json:"jornada_minutos"`
	Campos               map[string]string `json:"campos"`
}

type entradaAltaDecodificada struct {
	EsquemaAlta       string
	ClaveIdempotencia string
	NumeroMOAD        string
	Solicitud         domain.SolicitudCentro
	Necesidad         *domain.DatosNecesidadAlta
}

type periodoPrevistoJSON struct {
	Inicio   string `json:"inicio"`
	Fin      string `json:"fin,omitempty"`
	CausaFin string `json:"causa_fin,omitempty"`
}

type declaracionRCJSON struct {
	Existe       *bool               `json:"existe"`
	Numero       cadenaOpcionalJSON  `json:"numero"`
	Fecha        cadenaOpcionalJSON  `json:"fecha"`
	Importe      importeOpcionalJSON `json:"importe"`
	DocumentoRef cadenaOpcionalJSON  `json:"documento_ref"`
}

type importeJSON struct {
	Centimos *int64 `json:"centimos"`
	Moneda   string `json:"moneda"`
}

type cadenaOpcionalJSON struct {
	presente bool
	valor    string
}

func (c *cadenaOpcionalJSON) UnmarshalJSON(contenido []byte) error {
	if c == nil || bytes.Equal(bytes.TrimSpace(contenido), []byte("null")) {
		return errEntradaAltaInvalida
	}
	var valor string
	if err := json.Unmarshal(contenido, &valor); err != nil {
		return errEntradaAltaInvalida
	}
	c.presente, c.valor = true, valor
	return nil
}

type importeOpcionalJSON struct {
	presente bool
	valor    importeJSON
}

func (i *importeOpcionalJSON) UnmarshalJSON(contenido []byte) error {
	if i == nil || bytes.Equal(bytes.TrimSpace(contenido), []byte("null")) {
		return errEntradaAltaInvalida
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&i.valor); err != nil {
		return errEntradaAltaInvalida
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return errEntradaAltaInvalida
	}
	i.presente = true
	return nil
}

func validarMetadatosAlta(r *http.Request) *errorPublicoAlta {
	if r == nil || r.URL == nil || r.URL.RawQuery != "" || r.URL.ForceQuery {
		problema := errorPeticionNoPermitida
		return &problema
	}
	if r.ContentLength > MaximoCuerpoAltaBytes {
		problema := errorPeticionDemasiadoGrande
		return &problema
	}
	if r.ContentLength == 0 || r.Body == nil || r.Body == http.NoBody ||
		len(r.Trailer) != 0 || !transferenciaAltaPermitida(r.TransferEncoding) {
		problema := errorPeticionNoValida
		return &problema
	}
	if !tipoContenidoJSON(r.Header) {
		problema := errorTipoContenidoNoAdmitido
		return &problema
	}
	if !acceptCompatibleJSON(r.Header) {
		problema := errorRepresentacionNoAceptable
		return &problema
	}
	if cabeceraAltaProhibida(r.Header) {
		problema := errorPeticionNoPermitida
		return &problema
	}
	return nil
}

func transferenciaAltaPermitida(codificaciones []string) bool {
	return len(codificaciones) == 0 ||
		(len(codificaciones) == 1 && strings.EqualFold(codificaciones[0], "chunked"))
}

func tipoContenidoJSON(cabeceras http.Header) bool {
	valor, correcto := cabeceraUnicaAlta(cabeceras, "Content-Type")
	if !correcto {
		return false
	}
	tipo, parametros, err := mime.ParseMediaType(valor)
	if err != nil || !strings.EqualFold(tipo, "application/json") || len(parametros) > 1 {
		return false
	}
	if len(parametros) == 1 {
		charset, presente := parametros["charset"]
		if !presente || !strings.EqualFold(charset, "utf-8") {
			return false
		}
	}
	return true
}

func acceptCompatibleJSON(cabeceras http.Header) bool {
	valores := valoresCabeceraAlta(cabeceras, "Accept")
	if len(valores) == 0 {
		return true
	}
	especificidadElegida := -1
	calidadElegida := 0.0
	for _, linea := range valores {
		for _, elemento := range strings.Split(linea, ",") {
			tipo, parametros, err := mime.ParseMediaType(strings.TrimSpace(elemento))
			if err != nil {
				return false
			}
			calidad := 1.0
			if texto, presente := parametros["q"]; presente {
				calidad, err = strconv.ParseFloat(texto, 64)
				delete(parametros, "q")
				if err != nil || math.IsNaN(calidad) || math.IsInf(calidad, 0) ||
					calidad < 0 || calidad > 1 {
					return false
				}
			}
			// Un parámetro de representación (por ejemplo profile) solo
			// coincidiría con una respuesta que lo emitiese. Esta API devuelve
			// application/json sin esos parámetros.
			if len(parametros) != 0 {
				continue
			}
			especificidad := -1
			switch strings.ToLower(tipo) {
			case "application/json":
				especificidad = 2
			case "application/*":
				especificidad = 1
			case "*/*":
				especificidad = 0
			}
			if especificidad > especificidadElegida ||
				(especificidad == especificidadElegida && calidad > calidadElegida) {
				especificidadElegida = especificidad
				calidadElegida = calidad
			}
		}
	}
	return especificidadElegida >= 0 && calidadElegida > 0
}

func cabeceraAltaProhibida(cabeceras http.Header) bool {
	for nombre := range cabeceras {
		minusculas := strings.ToLower(nombre)
		switch {
		case minusculas == "authorization",
			minusculas == "cookie",
			minusculas == "set-cookie",
			minusculas == "proxy-authorization",
			minusculas == "proxy-connection",
			minusculas == "forwarded",
			minusculas == "remote-user",
			minusculas == "x-remote-user",
			minusculas == "x-forwarded-user",
			minusculas == "idempotency-key",
			minusculas == "content-encoding",
			minusculas == "trailer",
			minusculas == "te",
			minusculas == "expect",
			minusculas == "x-http-method-override",
			cabeceraAutoridadLibreAlta(minusculas),
			strings.Contains(minusculas, "role"),
			strings.HasPrefix(minusculas, "x-auth-"),
			strings.HasPrefix(minusculas, "x-vec-"),
			strings.HasPrefix(minusculas, "x-forwarded-"),
			strings.HasPrefix(minusculas, "x-envoy-"):
			return true
		}
	}
	return false
}

func cabeceraAutoridadLibreAlta(nombre string) bool {
	switch nombre {
	case "actor", "user", "usuario", "identity", "identidad", "profile", "perfil",
		"organization", "organizacion", "session", "sesion", "account", "cuenta",
		"permissions", "permission", "permisos", "permiso",
		"x-actor", "x-user", "x-usuario", "x-identity", "x-identidad",
		"x-profile", "x-perfil", "x-organization", "x-organizacion",
		"x-session", "x-sesion", "x-account", "x-cuenta",
		"x-permissions", "x-permission", "x-permisos", "x-permiso":
		return true
	default:
		return false
	}
}

func cabeceraUnicaAlta(cabeceras http.Header, nombre string) (string, bool) {
	valores := valoresCabeceraAlta(cabeceras, nombre)
	return valorUnicoNoVacio(valores)
}

func valoresCabeceraAlta(cabeceras http.Header, nombre string) []string {
	var valores []string
	for recibido, lista := range cabeceras {
		if strings.EqualFold(recibido, nombre) {
			valores = append(valores, lista...)
		}
	}
	return valores
}

func valorUnicoNoVacio(valores []string) (string, bool) {
	if len(valores) != 1 || valores[0] == "" || valores[0] != strings.TrimSpace(valores[0]) {
		return "", false
	}
	return valores[0], true
}

func solicitudAltaDesdePeticion(
	w http.ResponseWriter,
	r *http.Request,
) (entradaAltaDecodificada, error) {
	lector := http.MaxBytesReader(w, r.Body, MaximoCuerpoAltaBytes+1)
	contenido, err := io.ReadAll(lector)
	if err != nil {
		var demasiadoGrande *http.MaxBytesError
		if errors.As(err, &demasiadoGrande) {
			return entradaAltaDecodificada{}, errCuerpoAltaDemasiadoGrande
		}
		return entradaAltaDecodificada{}, errEntradaAltaInvalida
	}
	if len(contenido) == 0 {
		return entradaAltaDecodificada{}, errEntradaAltaInvalida
	}
	if len(contenido) > MaximoCuerpoAltaBytes {
		return entradaAltaDecodificada{}, errCuerpoAltaDemasiadoGrande
	}
	if !utf8.Valid(contenido) {
		return entradaAltaDecodificada{}, errEntradaAltaInvalida
	}
	if err := validarJSONAltaSinDuplicados(contenido); err != nil {
		return entradaAltaDecodificada{}, err
	}
	var entrada solicitudAltaVersionadaJSON
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&entrada); err != nil {
		return entradaAltaDecodificada{}, errEntradaAltaInvalida
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return entradaAltaDecodificada{}, errEntradaAltaInvalida
	}
	var camposSobre map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &camposSobre); err != nil {
		return entradaAltaDecodificada{}, errEntradaAltaInvalida
	}
	if !ports.ClaveIdempotenciaValida(entrada.ClaveIdempotencia) ||
		entrada.Solicitud == nil ||
		(entrada.NumeroExpedienteMOAD != "" && !domain.NumeroExpedienteValido(entrada.NumeroExpedienteMOAD)) {
		return entradaAltaDecodificada{}, errContenidoAltaNoValido
	}
	if (entrada.Esquema == "" && (camposSobre["esquema"] != nil || camposSobre["necesidad"] != nil)) ||
		(entrada.Esquema == application.EsquemaAltaNecesidadV1 && entrada.Necesidad == nil) ||
		(entrada.Esquema == application.EsquemaAltaNecesidadV1 && entrada.NumeroExpedienteMOAD == "") ||
		(entrada.Esquema != "" && entrada.Esquema != application.EsquemaAltaNecesidadV1) {
		return entradaAltaDecodificada{}, errContenidoAltaNoValido
	}
	solicitud, err := entrada.Solicitud.dominio()
	if err != nil {
		return entradaAltaDecodificada{}, err
	}
	salida := entradaAltaDecodificada{
		EsquemaAlta: entrada.Esquema, ClaveIdempotencia: entrada.ClaveIdempotencia,
		NumeroMOAD: entrada.NumeroExpedienteMOAD, Solicitud: solicitud,
	}
	if entrada.Necesidad != nil {
		campos := make(map[string]string, len(entrada.Necesidad.Campos))
		for clave, valor := range entrada.Necesidad.Campos {
			campos[clave] = valor
		}
		salida.Necesidad = &domain.DatosNecesidadAlta{
			Esquema:              entrada.Necesidad.Esquema,
			CatalogoRef:          entrada.Necesidad.CatalogoRef,
			CatalogoVersion:      entrada.Necesidad.CatalogoVersion,
			CatalogoHuellaSHA256: entrada.Necesidad.CatalogoHuellaSHA256,
			CausaClave:           domain.ClaveCatalogo(entrada.Necesidad.CausaClave),
			Periodo:              solicitud.Periodo, JornadaMinutos: entrada.Necesidad.JornadaMinutos,
			Campos: campos,
		}
	}
	return salida, nil
}

func validarJSONAltaSinDuplicados(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	tokens := 0
	var recorrer func(int) error
	recorrer = func(profundidad int) error {
		tokens++
		if profundidad > profundidadMaximaJSONAlta || tokens > tokensMaximosJSONAlta {
			return errCuerpoAltaDemasiadoGrande
		}
		token, err := decodificador.Token()
		if err != nil {
			return errEntradaAltaInvalida
		}
		delimitador, compuesto := token.(json.Delim)
		if !compuesto {
			if token == nil {
				return errEntradaAltaInvalida
			}
			if numero, esNumero := token.(json.Number); esNumero &&
				!patronEnteroJSONAlta.MatchString(numero.String()) {
				return errEntradaAltaInvalida
			}
			return nil
		}
		switch delimitador {
		case '{':
			vistas := map[string]struct{}{}
			for decodificador.More() {
				claveToken, err := decodificador.Token()
				clave, correcta := claveToken.(string)
				if err != nil || !correcta || clave != strings.ToLower(clave) {
					return errEntradaAltaInvalida
				}
				if _, repetida := vistas[clave]; repetida {
					return errEntradaAltaInvalida
				}
				vistas[clave] = struct{}{}
				if err := recorrer(profundidad + 1); err != nil {
					return err
				}
			}
			cierre, err := decodificador.Token()
			if err != nil || cierre != json.Delim('}') {
				return errEntradaAltaInvalida
			}
		case '[':
			for decodificador.More() {
				if err := recorrer(profundidad + 1); err != nil {
					return err
				}
			}
			cierre, err := decodificador.Token()
			if err != nil || cierre != json.Delim(']') {
				return errEntradaAltaInvalida
			}
		default:
			return errEntradaAltaInvalida
		}
		return nil
	}
	if err := recorrer(0); err != nil {
		return err
	}
	if _, err := decodificador.Token(); err != io.EOF {
		return errEntradaAltaInvalida
	}
	return nil
}

func (s solicitudCentroJSON) dominio() (domain.SolicitudCentro, error) {
	if s.Periodo == nil || s.RC == nil || s.DocumentosAdjuntos == nil {
		return domain.SolicitudCentro{}, errContenidoAltaNoValido
	}
	inicio, errInicio := fechaCivilUTC(s.Periodo.Inicio)
	var fin time.Time
	var errFin error
	if s.Periodo.Fin != "" {
		fin, errFin = fechaCivilUTC(s.Periodo.Fin)
	}
	rc, errRC := s.RC.dominio()
	observaciones := ""
	if s.Observaciones.presente {
		observaciones = s.Observaciones.valor
	}
	solicitud := domain.SolicitudCentro{
		CentroRef:          s.CentroRef,
		ContactoRef:        s.ContactoRef,
		CategoriaRef:       s.CategoriaRef,
		GrupoSubgrupo:      s.GrupoSubgrupo,
		MotivoClave:        domain.ClaveCatalogo(s.MotivoClave),
		Detalle:            s.Detalle,
		Periodo:            domain.PeriodoPrevisto{Inicio: inicio, Fin: fin, CausaFin: domain.ClaveCatalogo(s.Periodo.CausaFin)},
		RC:                 rc,
		DocumentosAdjuntos: append([]string(nil), (*s.DocumentosAdjuntos)...),
		Observaciones:      observaciones,
	}
	if errInicio != nil || errFin != nil || errRC != nil ||
		solicitud.Periodo.Fin.After(
			solicitud.Periodo.Inicio.AddDate(MaximoAniosPeriodoAlta, 0, 0),
		) ||
		len(solicitud.DocumentosAdjuntos) > MaximoDocumentosAdjuntos ||
		utf8.RuneCountInString(solicitud.Detalle) > MaximoCaracteresDetalle ||
		solicitud.Validar() != nil {
		return domain.SolicitudCentro{}, errContenidoAltaNoValido
	}
	return solicitud, nil
}

func (r declaracionRCJSON) dominio() (domain.DeclaracionRC, error) {
	if r.Existe == nil {
		return domain.DeclaracionRC{}, errContenidoAltaNoValido
	}
	if !*r.Existe {
		if r.Numero.presente || r.Fecha.presente || r.Importe.presente || r.DocumentoRef.presente {
			return domain.DeclaracionRC{}, errContenidoAltaNoValido
		}
		return domain.DeclaracionRC{}, nil
	}
	if !r.Numero.presente || !r.Fecha.presente || !r.Importe.presente ||
		!r.DocumentoRef.presente || r.Importe.valor.Centimos == nil ||
		*r.Importe.valor.Centimos > MaximoCentimosJSON {
		return domain.DeclaracionRC{}, errContenidoAltaNoValido
	}
	fecha, err := fechaCivilUTC(r.Fecha.valor)
	resultado := domain.DeclaracionRC{
		Existe:       true,
		Numero:       r.Numero.valor,
		Fecha:        fecha,
		Importe:      domain.Importe{Centimos: *r.Importe.valor.Centimos, Moneda: r.Importe.valor.Moneda},
		DocumentoRef: r.DocumentoRef.valor,
	}
	if err != nil || resultado.Validar() != nil {
		return domain.DeclaracionRC{}, errContenidoAltaNoValido
	}
	return resultado, nil
}

func fechaCivilUTC(valor string) (time.Time, error) {
	const formato = "2006-01-02T15:04:05Z"
	if len(valor) != len(formato) {
		return time.Time{}, errContenidoAltaNoValido
	}
	fecha, err := time.Parse(formato, valor)
	if err != nil || fecha.Year() < 1 || fecha.Format(formato) != valor ||
		!domain.InstanteUTCCanonico(fecha) ||
		fecha.Hour() != 0 || fecha.Minute() != 0 || fecha.Second() != 0 {
		return time.Time{}, errContenidoAltaNoValido
	}
	return fecha, nil
}

type envoltorioExitoAlta struct {
	Datos reciboAltaJSON `json:"data"`
}

type reciboAltaJSON struct {
	ExpedienteRef string `json:"expediente_ref"`
	NumeroVisible string `json:"numero_visible"`
	Version       uint64 `json:"version"`
	ReciboRef     string `json:"recibo_ref"`
	ConfirmadaEn  string `json:"confirmada_en"`
}

func responderExitoAlta(w http.ResponseWriter, r *http.Request, recibo ports.ReciboAlta) {
	responderJSONAlta(w, r, http.StatusCreated, envoltorioExitoAlta{Datos: reciboAltaJSON{
		ExpedienteRef: recibo.ExpedienteRef,
		NumeroVisible: recibo.NumeroVisible,
		Version:       recibo.Version,
		ReciboRef:     recibo.ReciboRef,
		ConfirmadaEn:  recibo.ConfirmadaEn.UTC().Format(time.RFC3339Nano),
	}})
}
