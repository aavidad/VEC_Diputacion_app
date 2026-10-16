package httpinterno

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	maximoCuerpoBorradorBytes = 4 * 1024 * 1024
	maximoEnteroSeguroJSON    = 9_007_199_254_740_991
)

var (
	errEntradaBorradorInvalida        = errors.New("bolsa http interno: entrada de borrador invalida")
	errEntradaBorradorDemasiadoGrande = errors.New("bolsa http interno: entrada de borrador demasiado grande")
	patronClaveBorrador               = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,79}$`)
	patronReferenciaBorrador          = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/#@-]{0,511}$`)
	patronIdentificadorBorrador       = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,79}$`)
	patronHuellaBorrador              = regexp.MustCompile(`^[a-f0-9]{64}$`)
	patronCursorBorrador              = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:@-]{0,511}$`)
	patronETagBorrador                = regexp.MustCompile(`^"vec-borrador-v1\.r([1-9][0-9]{0,15})\.sha256-([a-f0-9]{64})"$`)
	patronInstanteBorrador            = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.(\d{1,6}))?Z$`)
)

type envelopeAltaBorradorJSON struct {
	Data *altaBorradorJSON `json:"data"`
}

type altaBorradorJSON struct {
	Esquema               string                 `json:"esquema"`
	PlantillaRef          string                 `json:"plantilla_ref"`
	PlantillaVersion      int                    `json:"plantilla_version"`
	PlantillaHuellaSHA256 string                 `json:"plantilla_huella_sha256"`
	CodigoVersionPublica  string                 `json:"codigo_version_publica"`
	IdentificadorPublico  string                 `json:"identificador_publico"`
	ExpedienteRef         string                 `json:"expediente_ref"`
	ContenidoEditable     *contenidoBorradorJSON `json:"contenido_editable"`
	MotivoRef             string                 `json:"motivo_ref"`
	MotivoVersion         int                    `json:"motivo_version"`
	MotivoHuellaSHA256    string                 `json:"motivo_huella_sha256"`
}

type envelopeActualizacionBorradorJSON struct {
	Data *actualizacionBorradorJSON `json:"data"`
}

type actualizacionBorradorJSON struct {
	Esquema            string                 `json:"esquema"`
	ContenidoEditable  *contenidoBorradorJSON `json:"contenido_editable"`
	MotivoRef          string                 `json:"motivo_ref"`
	MotivoVersion      int                    `json:"motivo_version"`
	MotivoHuellaSHA256 string                 `json:"motivo_huella_sha256"`
}

type contenidoBorradorJSON struct {
	Tipo        string                   `json:"tipo"`
	Categorias  *[]string                `json:"categorias"`
	Titulo      string                   `json:"titulo"`
	Resumen     string                   `json:"resumen"`
	Descripcion string                   `json:"descripcion"`
	Plazos      *[]plazoBorradorJSON     `json:"plazos"`
	Requisitos  *[]requisitoBorradorJSON `json:"requisitos"`
	Ayuda       *[]ayudaBorradorJSON     `json:"ayuda"`
}

type plazoBorradorJSON struct {
	Referencia  string `json:"referencia"`
	Tipo        string `json:"tipo"`
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
	AbreEn      string `json:"abre_en"`
	CierraEn    string `json:"cierra_en"`
}

type requisitoBorradorJSON struct {
	Referencia  string `json:"referencia"`
	Orden       int    `json:"orden"`
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
	Obligatorio *bool  `json:"obligatorio"`
}

type ayudaBorradorJSON struct {
	Referencia string `json:"referencia"`
	Categoria  string `json:"categoria"`
	Orden      int    `json:"orden"`
	Pregunta   string `json:"pregunta"`
	Respuesta  string `json:"respuesta"`
}

func selectorListaDesdePeticion(
	r *http.Request,
) (gobiernoconvocatorias.SelectorListaBorradores, error) {
	if r == nil || r.URL == nil || r.ContentLength != 0 || len(r.TransferEncoding) != 0 ||
		(r.Body != nil && r.Body != http.NoBody) || cabeceraPresente(r.Header, "Content-Type") ||
		cabeceraPresente(r.Header, "Idempotency-Key") || cabeceraPresente(r.Header, "If-Match") || r.URL.ForceQuery {
		return gobiernoconvocatorias.SelectorListaBorradores{}, errEntradaBorradorInvalida
	}
	valores, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return gobiernoconvocatorias.SelectorListaBorradores{}, errEntradaBorradorInvalida
	}
	for nombre, lista := range valores {
		if (nombre != "limite" && nombre != "cursor" && nombre != "texto" && nombre != "categoria") || len(lista) != 1 {
			return gobiernoconvocatorias.SelectorListaBorradores{}, errEntradaBorradorInvalida
		}
	}
	limites, existe := valores["limite"]
	if !existe || len(limites) != 1 {
		return gobiernoconvocatorias.SelectorListaBorradores{}, errEntradaBorradorInvalida
	}
	limite, err := strconv.Atoi(limites[0])
	if err != nil || strconv.Itoa(limite) != limites[0] || limite < 1 || limite > 50 {
		return gobiernoconvocatorias.SelectorListaBorradores{}, errEntradaBorradorInvalida
	}
	selector := gobiernoconvocatorias.SelectorListaBorradores{Limite: limite}
	if lista, presente := valores["cursor"]; presente {
		if !patronCursorBorrador.MatchString(lista[0]) {
			return selector, errEntradaBorradorInvalida
		}
		selector.Cursor = lista[0]
	}
	if lista, presente := valores["texto"]; presente {
		if !cadenaBorradorValida(lista[0], 180, false, false) {
			return selector, errEntradaBorradorInvalida
		}
		selector.Texto = lista[0]
	}
	if lista, presente := valores["categoria"]; presente {
		if !patronClaveBorrador.MatchString(lista[0]) {
			return selector, errEntradaBorradorInvalida
		}
		selector.Categoria = lista[0]
	}
	return selector, nil
}

func solicitudAltaDesdePeticion(
	w http.ResponseWriter, r *http.Request,
) (gobiernoconvocatorias.SolicitudAltaBorrador, error) {
	clave, err := cabecerasMutacionBorrador(r, false)
	if err != nil {
		return gobiernoconvocatorias.SolicitudAltaBorrador{}, err
	}
	var envelope envelopeAltaBorradorJSON
	if err := decodificarCuerpoBorrador(w, r, &envelope); err != nil || envelope.Data == nil {
		return gobiernoconvocatorias.SolicitudAltaBorrador{}, errors.Join(errEntradaBorradorInvalida, err)
	}
	datos := envelope.Data
	contenido, err := convertirContenidoBorrador(datos.ContenidoEditable)
	if err != nil || datos.Esquema != "vec.bolsa.borrador.crear.v1" ||
		!referenciaBorradorValida(datos.PlantillaRef, 512) || !enteroSeguroPositivo(datos.PlantillaVersion) ||
		!patronHuellaBorrador.MatchString(datos.PlantillaHuellaSHA256) ||
		!patronClaveBorrador.MatchString(datos.CodigoVersionPublica) ||
		!patronIdentificadorBorrador.MatchString(datos.IdentificadorPublico) ||
		!referenciaBorradorValida(datos.ExpedienteRef, 512) || !selectorMotivoHTTPValido(
		datos.MotivoRef, datos.MotivoVersion, datos.MotivoHuellaSHA256,
	) {
		return gobiernoconvocatorias.SolicitudAltaBorrador{}, errors.Join(errEntradaBorradorInvalida, err)
	}
	return gobiernoconvocatorias.SolicitudAltaBorrador{
		ClaveIdempotencia: clave,
		Plantilla: gobiernoconvocatorias.SelectorPlantillaBorrador{
			ID: datos.PlantillaRef, Version: datos.PlantillaVersion,
			HuellaContenidoSHA256: datos.PlantillaHuellaSHA256,
		},
		CodigoVersionPublica: datos.CodigoVersionPublica,
		IdentificadorPublico: datos.IdentificadorPublico,
		ExpedienteRef:        datos.ExpedienteRef,
		Contenido:            contenido,
		Motivo: gobiernoconvocatorias.SelectorMotivoBorrador{
			Referencia: datos.MotivoRef, Version: datos.MotivoVersion,
			HuellaSHA256: datos.MotivoHuellaSHA256,
		},
	}, nil
}

func solicitudActualizacionDesdePeticion(
	w http.ResponseWriter,
	r *http.Request,
	selector puertosbolsa.SelectorVersionConvocatoriaExacta,
) (gobiernoconvocatorias.SolicitudActualizacionBorrador, error) {
	clave, err := cabecerasMutacionBorrador(r, true)
	if err != nil {
		return gobiernoconvocatorias.SolicitudActualizacionBorrador{}, err
	}
	etag, err := cabeceraUnicaBorrador(r.Header, "If-Match")
	coincidencia := patronETagBorrador.FindStringSubmatch(etag)
	if err != nil || len(coincidencia) != 3 {
		return gobiernoconvocatorias.SolicitudActualizacionBorrador{}, errEntradaBorradorInvalida
	}
	revision64, err := strconv.ParseUint(coincidencia[1], 10, 53)
	if err != nil || revision64 < 1 || revision64 > uint64(^uint(0)>>1) {
		return gobiernoconvocatorias.SolicitudActualizacionBorrador{}, errEntradaBorradorInvalida
	}
	var envelope envelopeActualizacionBorradorJSON
	if err := decodificarCuerpoBorrador(w, r, &envelope); err != nil || envelope.Data == nil {
		return gobiernoconvocatorias.SolicitudActualizacionBorrador{}, errors.Join(errEntradaBorradorInvalida, err)
	}
	datos := envelope.Data
	contenido, err := convertirContenidoBorrador(datos.ContenidoEditable)
	if err != nil || datos.Esquema != "vec.bolsa.borrador.actualizar.v1" ||
		!selectorMotivoHTTPValido(datos.MotivoRef, datos.MotivoVersion, datos.MotivoHuellaSHA256) {
		return gobiernoconvocatorias.SolicitudActualizacionBorrador{}, errors.Join(errEntradaBorradorInvalida, err)
	}
	esperada := puertosbolsa.ReferenciaEstadoVersionConvocatoria{
		Referencia: selector.Referencia(), Revision: int(revision64), HuellaEstadoSHA256: coincidencia[2],
	}
	if esperada.Validar() != nil {
		return gobiernoconvocatorias.SolicitudActualizacionBorrador{}, errEntradaBorradorInvalida
	}
	return gobiernoconvocatorias.SolicitudActualizacionBorrador{
		ClaveIdempotencia: clave, Esperada: esperada, Contenido: contenido,
		Motivo: gobiernoconvocatorias.SelectorMotivoBorrador{
			Referencia: datos.MotivoRef, Version: datos.MotivoVersion,
			HuellaSHA256: datos.MotivoHuellaSHA256,
		},
	}, nil
}

func cabecerasMutacionBorrador(r *http.Request, actualizacion bool) (string, error) {
	if r == nil || r.URL == nil || r.URL.RawQuery != "" || r.URL.ForceQuery || r.Body == nil || r.Body == http.NoBody ||
		(!actualizacion && cabeceraPresente(r.Header, "If-Match")) {
		return "", errEntradaBorradorInvalida
	}
	tipo, err := cabeceraUnicaBorrador(r.Header, "Content-Type")
	if err != nil {
		return "", errEntradaBorradorInvalida
	}
	medio, parametros, err := mime.ParseMediaType(tipo)
	if err != nil || !strings.EqualFold(medio, "application/json") || len(parametros) > 1 {
		return "", errEntradaBorradorInvalida
	}
	if charset, existe := parametros["charset"]; existe && !strings.EqualFold(charset, "utf-8") {
		return "", errEntradaBorradorInvalida
	}
	clave, err := cabeceraUnicaBorrador(r.Header, "Idempotency-Key")
	if err != nil {
		return "", errEntradaBorradorInvalida
	}
	if _, err := gobiernoconvocatorias.NuevaClaveClienteIdempotenciaConvocatoria(clave); err != nil {
		return "", errEntradaBorradorInvalida
	}
	if r.ContentLength > maximoCuerpoBorradorBytes {
		return "", errEntradaBorradorDemasiadoGrande
	}
	return clave, nil
}

func cabeceraUnicaBorrador(cabeceras http.Header, nombre string) (string, error) {
	valores := make([]string, 0, 1)
	for recibido, lista := range cabeceras {
		if strings.EqualFold(recibido, nombre) {
			valores = append(valores, lista...)
		}
	}
	if len(valores) != 1 || valores[0] == "" || valores[0] != strings.TrimSpace(valores[0]) {
		return "", errEntradaBorradorInvalida
	}
	return valores[0], nil
}

func decodificarCuerpoBorrador(w http.ResponseWriter, r *http.Request, destino any) error {
	lector := http.MaxBytesReader(w, r.Body, maximoCuerpoBorradorBytes+1)
	contenido, err := io.ReadAll(lector)
	if err != nil {
		var demasiadoGrande *http.MaxBytesError
		if errors.As(err, &demasiadoGrande) {
			return errEntradaBorradorDemasiadoGrande
		}
		return errEntradaBorradorInvalida
	}
	if len(contenido) == 0 {
		return errEntradaBorradorInvalida
	}
	if len(contenido) > maximoCuerpoBorradorBytes {
		return errEntradaBorradorDemasiadoGrande
	}
	if !utf8.Valid(contenido) {
		return errEntradaBorradorInvalida
	}
	if err := validarJSONSinDuplicados(contenido); err != nil {
		return err
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return errEntradaBorradorInvalida
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return errEntradaBorradorInvalida
	}
	return nil
}

func validarJSONSinDuplicados(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	const (
		profundidadMaximaJSONBorrador = 32
		tokensMaximosJSONBorrador     = 100_000
	)
	tokens := 0
	var recorrer func(int) error
	recorrer = func(profundidad int) error {
		tokens++
		if profundidad > profundidadMaximaJSONBorrador || tokens > tokensMaximosJSONBorrador {
			return errEntradaBorradorDemasiadoGrande
		}
		token, err := decodificador.Token()
		if err != nil {
			return err
		}
		delimitador, compuesto := token.(json.Delim)
		if !compuesto {
			return nil
		}
		switch delimitador {
		case '{':
			vistas := map[string]struct{}{}
			for decodificador.More() {
				tokens++
				if tokens > tokensMaximosJSONBorrador {
					return errEntradaBorradorDemasiadoGrande
				}
				claveToken, err := decodificador.Token()
				clave, correcta := claveToken.(string)
				// encoding/json enlaza nombres de campos sin distinguir
				// mayusculas. El contrato HTTP, en cambio, solo admite las
				// claves snake_case exactas: aceptar "Data" como "data"
				// permitiria representaciones ambiguas y un ultimo valor ganador.
				claveCanonica := strings.ToLower(clave)
				if err != nil || !correcta || clave != claveCanonica {
					return errEntradaBorradorInvalida
				}
				if _, repetida := vistas[claveCanonica]; repetida {
					return errEntradaBorradorInvalida
				}
				vistas[claveCanonica] = struct{}{}
				if err := recorrer(profundidad + 1); err != nil {
					return err
				}
			}
			cierre, err := decodificador.Token()
			if err != nil || cierre != json.Delim('}') {
				return errEntradaBorradorInvalida
			}
		case '[':
			for decodificador.More() {
				if err := recorrer(profundidad + 1); err != nil {
					return err
				}
			}
			cierre, err := decodificador.Token()
			if err != nil || cierre != json.Delim(']') {
				return errEntradaBorradorInvalida
			}
		default:
			return errEntradaBorradorInvalida
		}
		return nil
	}
	if err := recorrer(0); err != nil {
		return err
	}
	if _, err := decodificador.Token(); err != io.EOF {
		return errEntradaBorradorInvalida
	}
	return nil
}

func convertirContenidoBorrador(
	origen *contenidoBorradorJSON,
) (gobiernoconvocatorias.ContenidoEditableBorrador, error) {
	if origen == nil || origen.Categorias == nil || origen.Plazos == nil || origen.Requisitos == nil || origen.Ayuda == nil ||
		!patronClaveBorrador.MatchString(origen.Tipo) || !cadenaBorradorValida(origen.Titulo, 180, false, false) ||
		!cadenaBorradorValida(origen.Resumen, 500, false, false) || !cadenaBorradorValida(origen.Descripcion, 12000, false, true) ||
		len(*origen.Categorias) < 1 || len(*origen.Categorias) > 1024 || len(*origen.Plazos) < 1 || len(*origen.Plazos) > 64 ||
		len(*origen.Requisitos) > 256 || len(*origen.Ayuda) > 128 {
		return gobiernoconvocatorias.ContenidoEditableBorrador{}, errEntradaBorradorInvalida
	}
	resultado := gobiernoconvocatorias.ContenidoEditableBorrador{
		Tipo: origen.Tipo, Titulo: origen.Titulo, Resumen: origen.Resumen, Descripcion: origen.Descripcion,
		Categorias: append([]string(nil), (*origen.Categorias)...),
		Plazos:     make([]dominiobolsa.PlazoConvocatoria, len(*origen.Plazos)),
		Requisitos: make([]dominiobolsa.RequisitoConvocatoria, len(*origen.Requisitos)),
		Ayuda:      make([]dominiobolsa.AyudaConvocatoria, len(*origen.Ayuda)),
	}
	clavesCategorias := map[string]struct{}{}
	for _, categoria := range resultado.Categorias {
		if !patronClaveBorrador.MatchString(categoria) || insertarRepetida(clavesCategorias, categoria) {
			return resultado, errEntradaBorradorInvalida
		}
	}
	refsPlazos := map[string]struct{}{}
	for indice, plazo := range *origen.Plazos {
		abre, errAbre := instanteBorrador(plazo.AbreEn)
		cierra, errCierra := instanteBorrador(plazo.CierraEn)
		if !referenciaBorradorValida(plazo.Referencia, 160) || insertarRepetida(refsPlazos, plazo.Referencia) ||
			!patronClaveBorrador.MatchString(plazo.Tipo) || !cadenaBorradorValida(plazo.Titulo, 180, false, false) ||
			!cadenaBorradorValida(plazo.Descripcion, 1000, false, true) || errAbre != nil || errCierra != nil || !abre.Before(cierra) {
			return resultado, errEntradaBorradorInvalida
		}
		resultado.Plazos[indice] = dominiobolsa.PlazoConvocatoria{
			Referencia: plazo.Referencia, Tipo: plazo.Tipo, Titulo: plazo.Titulo,
			Descripcion: plazo.Descripcion, AbreEn: abre, CierraEn: cierra,
		}
	}
	refsRequisitos, ordenesRequisitos := map[string]struct{}{}, map[int]struct{}{}
	for indice, requisito := range *origen.Requisitos {
		if requisito.Obligatorio == nil || !enteroSeguroPositivo(requisito.Orden) ||
			!referenciaBorradorValida(requisito.Referencia, 160) || insertarRepetida(refsRequisitos, requisito.Referencia) ||
			insertarEnteroRepetido(ordenesRequisitos, requisito.Orden) ||
			!cadenaBorradorValida(requisito.Titulo, 180, false, false) ||
			!cadenaBorradorValida(requisito.Descripcion, 3000, false, true) {
			return resultado, errEntradaBorradorInvalida
		}
		resultado.Requisitos[indice] = dominiobolsa.RequisitoConvocatoria{
			Referencia: requisito.Referencia, Orden: requisito.Orden, Titulo: requisito.Titulo,
			Descripcion: requisito.Descripcion, Obligatorio: *requisito.Obligatorio,
		}
	}
	refsAyuda, ordenesAyuda := map[string]struct{}{}, map[int]struct{}{}
	for indice, ayuda := range *origen.Ayuda {
		if !enteroSeguroPositivo(ayuda.Orden) || !referenciaBorradorValida(ayuda.Referencia, 160) || insertarRepetida(refsAyuda, ayuda.Referencia) ||
			insertarEnteroRepetido(ordenesAyuda, ayuda.Orden) || !patronClaveBorrador.MatchString(ayuda.Categoria) ||
			!cadenaBorradorValida(ayuda.Pregunta, 300, false, false) || !cadenaBorradorValida(ayuda.Respuesta, 5000, false, true) {
			return resultado, errEntradaBorradorInvalida
		}
		resultado.Ayuda[indice] = dominiobolsa.AyudaConvocatoria{
			Referencia: ayuda.Referencia, Categoria: ayuda.Categoria, Orden: ayuda.Orden,
			Pregunta: ayuda.Pregunta, Respuesta: ayuda.Respuesta,
		}
	}
	return resultado, nil
}

func cadenaBorradorValida(valor string, maximo int, admiteVacia, multilinea bool) bool {
	if !utf8.ValidString(valor) || valor != strings.TrimSpace(valor) || !norm.NFC.IsNormalString(valor) ||
		(!admiteVacia && valor == "") || utf8.RuneCountInString(valor) > maximo {
		return false
	}
	for _, caracter := range valor {
		permitido := multilinea && (caracter == '\t' || caracter == '\n')
		if (!permitido && (caracter < 32 || (caracter >= 127 && caracter <= 159))) ||
			unicode.Is(unicode.Cf, caracter) || caracter == unicode.ReplacementChar {
			return false
		}
	}
	return true
}

func referenciaBorradorValida(valor string, maximo int) bool {
	return len(valor) <= maximo && cadenaBorradorValida(valor, maximo, false, false) && patronReferenciaBorrador.MatchString(valor)
}

func selectorMotivoHTTPValido(referencia string, version int, huella string) bool {
	return referenciaBorradorValida(referencia, 512) && enteroSeguroPositivo(version) && patronHuellaBorrador.MatchString(huella)
}

func enteroSeguroPositivo(valor int) bool {
	return valor >= 1 && uint64(valor) <= uint64(maximoEnteroSeguroJSON)
}

func instanteBorrador(valor string) (time.Time, error) {
	if !patronInstanteBorrador.MatchString(valor) {
		return time.Time{}, errEntradaBorradorInvalida
	}
	instante, err := time.Parse("2006-01-02T15:04:05.999999Z", valor)
	if err != nil || instante.Year() < 1 || instante.Location() != time.UTC || instante.Nanosecond()%1000 != 0 {
		return time.Time{}, errEntradaBorradorInvalida
	}
	return instante, nil
}

func insertarRepetida(vistos map[string]struct{}, clave string) bool {
	_, repetida := vistos[clave]
	vistos[clave] = struct{}{}
	return repetida
}

func insertarEnteroRepetido(vistos map[int]struct{}, clave int) bool {
	_, repetida := vistos[clave]
	vistos[clave] = struct{}{}
	return repetida
}

func responderErrorEntradaBorrador(w http.ResponseWriter, err error, correlacion string) {
	if errors.Is(err, errEntradaBorradorDemasiadoGrande) {
		responderErrorBorrador(w, http.StatusRequestEntityTooLarge, "peticion_demasiado_grande", correlacion)
		return
	}
	responderErrorBorrador(w, http.StatusBadRequest, "peticion_no_valida", correlacion)
}
