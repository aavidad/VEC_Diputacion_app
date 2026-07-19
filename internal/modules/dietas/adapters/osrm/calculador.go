// Package osrm implementa el puerto de calculo de rutas contra un motor OSRM
// expresamente autorizado. El conector no usa proxy ambiental, no sigue
// redirecciones y vuelve a comprobar cada direccion resuelta antes de marcarla.
package osrm

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

const (
	maximoBytesNombreAmbito = 160
	maximoBytesNombrePunto  = 160
	maximoBytesRespuesta    = 20 << 20
	maximoPuntosGeometria   = 50_000
	tiempoMaximoSolicitud   = 15 * time.Second
	maximoDistanciaMetros   = 10_000_000
	maximoDuracionSegundos  = 1_200_000
	motorRespuesta          = "osrm_on_premise"
)

// Configuracion forma una concesion atomica: o todos sus campos estan vacios
// y el puerto queda desconectado, o todos son obligatorios y se validan.
type Configuracion struct {
	URLBase        string
	NombreAmbito   string
	LimitesAmbito  string
	CIDRPermitidas []string
	VersionGrafo   string
}

type limites struct {
	latitudMinima  float64
	latitudMaxima  float64
	longitudMinima float64
	longitudMaxima float64
}

type ambito struct {
	nombre  string
	limites limites
}

type destino struct {
	host           string
	puerto         string
	cidrPermitidas []netip.Prefix
}

// Calculador es el adaptador OSRM del puerto de Dietas.
type Calculador struct {
	urlBase      string
	ambito       ambito
	versionGrafo string
	cliente      *http.Client
}

type respuestaOSRM struct {
	Codigo      string          `json:"code"`
	Rutas       []rutaOSRM      `json:"routes"`
	VersionDato json.RawMessage `json:"data_version"`
}

type rutaOSRM struct {
	Distancia float64       `json:"distance"`
	Duracion  float64       `json:"duration"`
	Geometria geometriaOSRM `json:"geometry"`
	Tramos    []tramoOSRM   `json:"legs"`
}

type geometriaOSRM struct {
	Tipo        string      `json:"type"`
	Coordenadas [][]float64 `json:"coordinates"`
}

type tramoOSRM struct {
	Distancia float64 `json:"distance"`
	Duracion  float64 `json:"duration"`
}

// Nuevo construye el conector. Una configuracion totalmente ausente devuelve
// nil sin error para que la composicion productiva pueda responder 503 de
// forma explicita. Cualquier configuracion parcial falla durante el arranque.
func Nuevo(configuracion Configuracion) (*Calculador, error) {
	configurada := strings.TrimSpace(configuracion.URLBase) != "" ||
		strings.TrimSpace(configuracion.NombreAmbito) != "" ||
		strings.TrimSpace(configuracion.LimitesAmbito) != "" ||
		len(configuracion.CIDRPermitidas) != 0 ||
		strings.TrimSpace(configuracion.VersionGrafo) != ""
	if !configurada {
		return nil, nil
	}
	if strings.TrimSpace(configuracion.URLBase) == "" ||
		strings.TrimSpace(configuracion.NombreAmbito) == "" ||
		strings.TrimSpace(configuracion.LimitesAmbito) == "" ||
		len(configuracion.CIDRPermitidas) == 0 ||
		strings.TrimSpace(configuracion.VersionGrafo) == "" {
		return nil, errors.New("dietas osrm: URL, ambito, limites, redes y version de grafo son obligatorios")
	}

	ambitoConfigurado, err := parsearAmbito(configuracion.NombreAmbito, configuracion.LimitesAmbito)
	if err != nil {
		return nil, fmt.Errorf("dietas osrm: ambito no valido: %w", err)
	}
	destinoConfigurado, urlCanonica, err := parsearDestino(configuracion.URLBase)
	if err != nil {
		return nil, fmt.Errorf("dietas osrm: destino no valido: %w", err)
	}
	destinoConfigurado.cidrPermitidas, err = parsearCIDRPermitidas(configuracion.CIDRPermitidas)
	if err != nil {
		return nil, fmt.Errorf("dietas osrm: redes no validas: %w", err)
	}
	version, err := validarVersionGrafo(configuracion.VersionGrafo)
	if err != nil {
		return nil, fmt.Errorf("dietas osrm: version de grafo no valida: %w", err)
	}

	return &Calculador{
		urlBase:      urlCanonica,
		ambito:       ambitoConfigurado,
		versionGrafo: version,
		cliente:      nuevoClienteHTTP(destinoConfigurado),
	}, nil
}

// Calcular consulta OSRM y devuelve solo la proyeccion necesaria para Dietas.
func (c *Calculador) Calcular(ctx context.Context, solicitud dietasports.SolicitudCalculoRuta) (dietasports.ResultadoCalculoRuta, error) {
	if c == nil || c.cliente == nil {
		return dietasports.ResultadoCalculoRuta{}, dietasports.ErrMotorRutasNoDisponible
	}
	if ctx == nil {
		return dietasports.ResultadoCalculoRuta{}, fmt.Errorf("%w: contexto ausente", dietasports.ErrSolicitudRutaInvalida)
	}
	if err := validarSolicitud(solicitud, c.ambito); err != nil {
		return dietasports.ResultadoCalculoRuta{}, err
	}
	endpoint, err := construirEndpoint(c.urlBase, solicitud)
	if err != nil {
		return dietasports.ResultadoCalculoRuta{}, err
	}
	ctx, cancelar := context.WithTimeout(ctx, tiempoMaximoSolicitud)
	defer cancelar()
	peticion, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return dietasports.ResultadoCalculoRuta{}, fmt.Errorf("%w: construir consulta", dietasports.ErrMotorRutasNoDisponible)
	}
	peticion.Header.Set("Accept", "application/json")
	respuesta, err := c.cliente.Do(peticion)
	if err != nil {
		return dietasports.ResultadoCalculoRuta{}, fmt.Errorf("%w: no se pudo consultar el motor autorizado", dietasports.ErrMotorRutasNoDisponible)
	}
	defer respuesta.Body.Close()
	if respuesta.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(respuesta.Body, 64<<10))
		return dietasports.ResultadoCalculoRuta{}, fmt.Errorf("%w: estado HTTP %d", dietasports.ErrMotorRutasNoDisponible, respuesta.StatusCode)
	}
	if err := validarTipoJSON(respuesta.Header.Get("Content-Type")); err != nil {
		return dietasports.ResultadoCalculoRuta{}, fmt.Errorf("%w: %v", dietasports.ErrRespuestaMotorRutasInvalida, err)
	}
	if respuesta.ContentLength > maximoBytesRespuesta {
		return dietasports.ResultadoCalculoRuta{}, fmt.Errorf("%w: respuesta demasiado grande", dietasports.ErrRespuestaMotorRutasInvalida)
	}
	contenido, err := io.ReadAll(io.LimitReader(respuesta.Body, maximoBytesRespuesta+1))
	if err != nil {
		return dietasports.ResultadoCalculoRuta{}, fmt.Errorf("%w: lectura incompleta", dietasports.ErrRespuestaMotorRutasInvalida)
	}
	if len(contenido) == 0 || len(contenido) > maximoBytesRespuesta || !utf8.Valid(contenido) {
		return dietasports.ResultadoCalculoRuta{}, fmt.Errorf("%w: JSON vacio, demasiado grande o no UTF-8", dietasports.ErrRespuestaMotorRutasInvalida)
	}
	var osrm respuestaOSRM
	if err := json.Unmarshal(contenido, &osrm); err != nil {
		return dietasports.ResultadoCalculoRuta{}, fmt.Errorf("%w: JSON no valido", dietasports.ErrRespuestaMotorRutasInvalida)
	}
	return c.proyectarRespuesta(osrm, solicitud)
}

func (c *Calculador) proyectarRespuesta(respuesta respuestaOSRM, solicitud dietasports.SolicitudCalculoRuta) (dietasports.ResultadoCalculoRuta, error) {
	if respuesta.Codigo != "Ok" {
		return dietasports.ResultadoCalculoRuta{}, fmt.Errorf("%w: el motor no confirmo una ruta", dietasports.ErrRespuestaMotorRutasInvalida)
	}
	alternativasSolicitadas := solicitud.Alternativas
	if alternativasSolicitadas == 0 {
		alternativasSolicitadas = 1
	}
	if len(respuesta.Rutas) < 1 || len(respuesta.Rutas) > alternativasSolicitadas || len(respuesta.Rutas) > dietasports.MaximoAlternativasRuta {
		return dietasports.ResultadoCalculoRuta{}, fmt.Errorf("%w: numero de rutas inesperado", dietasports.ErrRespuestaMotorRutasInvalida)
	}
	if err := cotejarVersionOSRM(respuesta.VersionDato, c.versionGrafo); err != nil {
		return dietasports.ResultadoCalculoRuta{}, err
	}

	resultado := dietasports.ResultadoCalculoRuta{
		Alternativas: make([]dietasports.AlternativaRuta, 0, len(respuesta.Rutas)),
		VersionGrafo: c.versionGrafo,
		Motor:        motorRespuesta,
		Ambito:       c.ambito.nombre,
	}
	for _, ruta := range respuesta.Rutas {
		alternativa, err := proyectarRuta(ruta, len(solicitud.Coordenadas)-1)
		if err != nil {
			return dietasports.ResultadoCalculoRuta{}, err
		}
		resultado.Alternativas = append(resultado.Alternativas, alternativa)
	}
	return resultado, nil
}

func proyectarRuta(ruta rutaOSRM, tramosEsperados int) (dietasports.AlternativaRuta, error) {
	if !numeroPositivoAcotado(ruta.Distancia, maximoDistanciaMetros) ||
		!numeroPositivoAcotado(ruta.Duracion, maximoDuracionSegundos) ||
		ruta.Geometria.Tipo != "LineString" || len(ruta.Geometria.Coordenadas) < 2 ||
		len(ruta.Geometria.Coordenadas) > maximoPuntosGeometria || len(ruta.Tramos) != tramosEsperados {
		return dietasports.AlternativaRuta{}, fmt.Errorf("%w: ruta incompleta", dietasports.ErrRespuestaMotorRutasInvalida)
	}
	alternativa := dietasports.AlternativaRuta{
		DistanciaMetros:  ruta.Distancia,
		DuracionSegundos: ruta.Duracion,
		Geometria: dietasports.GeometriaRuta{
			Tipo:        "LineString",
			Coordenadas: make([]dietasports.PuntoGeometriaRuta, 0, len(ruta.Geometria.Coordenadas)),
		},
		Tramos: make([]dietasports.TramoRuta, 0, len(ruta.Tramos)),
	}
	for _, punto := range ruta.Geometria.Coordenadas {
		if len(punto) != 2 || !longitudValida(punto[0]) || !latitudValida(punto[1]) {
			return dietasports.AlternativaRuta{}, fmt.Errorf("%w: geometria no valida", dietasports.ErrRespuestaMotorRutasInvalida)
		}
		alternativa.Geometria.Coordenadas = append(alternativa.Geometria.Coordenadas, dietasports.PuntoGeometriaRuta{
			Longitud: punto[0],
			Latitud:  punto[1],
		})
	}
	for _, tramo := range ruta.Tramos {
		if !numeroPositivoAcotado(tramo.Distancia, maximoDistanciaMetros) ||
			!numeroPositivoAcotado(tramo.Duracion, maximoDuracionSegundos) {
			return dietasports.AlternativaRuta{}, fmt.Errorf("%w: tramo no valido", dietasports.ErrRespuestaMotorRutasInvalida)
		}
		alternativa.Tramos = append(alternativa.Tramos, dietasports.TramoRuta{
			DistanciaMetros:  tramo.Distancia,
			DuracionSegundos: tramo.Duracion,
		})
	}
	return alternativa, nil
}

func validarSolicitud(solicitud dietasports.SolicitudCalculoRuta, ambito ambito) error {
	if len(solicitud.Coordenadas) < 2 || len(solicitud.Coordenadas) > dietasports.MaximoCoordenadasRuta {
		return fmt.Errorf("%w: se requieren entre 2 y %d coordenadas", dietasports.ErrSolicitudRutaInvalida, dietasports.MaximoCoordenadasRuta)
	}
	if solicitud.Alternativas < 0 || solicitud.Alternativas > dietasports.MaximoAlternativasRuta {
		return fmt.Errorf("%w: alternativas fuera de rango", dietasports.ErrSolicitudRutaInvalida)
	}
	for indice, coordenada := range solicitud.Coordenadas {
		if !latitudValida(coordenada.Latitud) || !longitudValida(coordenada.Longitud) {
			return fmt.Errorf("%w: coordenada %d fuera de rango", dietasports.ErrSolicitudRutaInvalida, indice+1)
		}
		if !ambito.limites.contiene(coordenada) {
			return fmt.Errorf("%w: coordenada %d fuera del ambito %s", dietasports.ErrSolicitudRutaInvalida, indice+1, ambito.nombre)
		}
		if err := validarNombrePunto(coordenada.Nombre); err != nil {
			return fmt.Errorf("%w: nombre de coordenada %d no valido", dietasports.ErrSolicitudRutaInvalida, indice+1)
		}
	}
	return nil
}

func validarNombrePunto(nombre string) error {
	if nombre == "" {
		return nil
	}
	if nombre != strings.TrimSpace(nombre) || len(nombre) > maximoBytesNombrePunto || !utf8.ValidString(nombre) {
		return errors.New("nombre no canonico")
	}
	for _, caracter := range nombre {
		if unicode.IsControl(caracter) {
			return errors.New("nombre con caracteres de control")
		}
	}
	return nil
}

func numeroPositivoAcotado(valor, maximo float64) bool {
	return !math.IsNaN(valor) && !math.IsInf(valor, 0) && valor > 0 && valor <= maximo
}

func latitudValida(valor float64) bool {
	return !math.IsNaN(valor) && !math.IsInf(valor, 0) && valor >= -90 && valor <= 90
}

func longitudValida(valor float64) bool {
	return !math.IsNaN(valor) && !math.IsInf(valor, 0) && valor >= -180 && valor <= 180
}

func (l limites) contiene(coordenada dietasports.CoordenadaRuta) bool {
	return coordenada.Latitud >= l.latitudMinima && coordenada.Latitud <= l.latitudMaxima &&
		coordenada.Longitud >= l.longitudMinima && coordenada.Longitud <= l.longitudMaxima
}

func parsearAmbito(nombre, limitesSinParsear string) (ambito, error) {
	if nombre == "" || nombre != strings.TrimSpace(nombre) || len(nombre) > maximoBytesNombreAmbito || !utf8.ValidString(nombre) {
		return ambito{}, errors.New("el nombre debe ser explicito, canonico y no vacio")
	}
	for _, caracter := range nombre {
		if unicode.IsControl(caracter) {
			return ambito{}, errors.New("el nombre contiene caracteres de control")
		}
	}
	limitesParseados, err := parsearLimites(limitesSinParsear)
	if err != nil {
		return ambito{}, err
	}
	return ambito{nombre: nombre, limites: limitesParseados}, nil
}

func parsearLimites(valor string) (limites, error) {
	if valor == "" || valor != strings.TrimSpace(valor) {
		return limites{}, errors.New("los limites deben ser canonicos")
	}
	partes := strings.Split(valor, ",")
	if len(partes) != 4 {
		return limites{}, errors.New("los limites deben contener cuatro numeros")
	}
	numeros := make([]float64, 4)
	for indice, parte := range partes {
		if parte == "" || parte != strings.TrimSpace(parte) {
			return limites{}, errors.New("los limites no admiten espacios ni valores vacios")
		}
		numero, err := strconv.ParseFloat(parte, 64)
		if err != nil || math.IsNaN(numero) || math.IsInf(numero, 0) {
			return limites{}, errors.New("los limites contienen un numero no valido")
		}
		if canonico := strconv.FormatFloat(numero, 'f', -1, 64); canonico != parte {
			return limites{}, fmt.Errorf("el limite %q no es canonico; usa %q", parte, canonico)
		}
		numeros[indice] = numero
	}
	if !latitudValida(numeros[0]) || !longitudValida(numeros[1]) ||
		!latitudValida(numeros[2]) || !longitudValida(numeros[3]) {
		return limites{}, errors.New("los limites exceden latitud o longitud")
	}
	resultado := limites{
		latitudMinima: numeros[0], longitudMinima: numeros[1],
		latitudMaxima: numeros[2], longitudMaxima: numeros[3],
	}
	if resultado.latitudMinima >= resultado.latitudMaxima || resultado.longitudMinima >= resultado.longitudMaxima {
		return limites{}, errors.New("los limites minimos deben ser inferiores a los maximos")
	}
	return resultado, nil
}

func validarVersionGrafo(version string) (string, error) {
	if version == "" || version != strings.TrimSpace(version) || len(version) > 100 || !utf8.ValidString(version) {
		return "", errors.New("la version debe ser explicita y canonica")
	}
	for indice, caracter := range version {
		alfanumerico := caracter >= 'a' && caracter <= 'z' || caracter >= 'A' && caracter <= 'Z' || caracter >= '0' && caracter <= '9'
		if alfanumerico || indice > 0 && (caracter == '.' || caracter == '_' || caracter == ':' || caracter == '+' || caracter == '-') {
			continue
		}
		return "", errors.New("la version contiene caracteres no admitidos")
	}
	return version, nil
}

func cotejarVersionOSRM(valor json.RawMessage, versionConfigurada string) error {
	if len(valor) == 0 || string(valor) == "null" {
		return nil
	}
	var declarada string
	if err := json.Unmarshal(valor, &declarada); err != nil {
		return fmt.Errorf("%w: data_version no es texto ni null", dietasports.ErrRespuestaMotorRutasInvalida)
	}
	if _, err := validarVersionGrafo(declarada); err != nil || declarada != versionConfigurada {
		return fmt.Errorf("%w: data_version contradice la version gobernada", dietasports.ErrRespuestaMotorRutasInvalida)
	}
	return nil
}

func validarTipoJSON(valor string) error {
	tipo, parametros, err := mime.ParseMediaType(valor)
	if err != nil || !strings.EqualFold(tipo, "application/json") {
		return errors.New("el motor no devolvio application/json")
	}
	for nombre, valorParametro := range parametros {
		if !strings.EqualFold(nombre, "charset") || !strings.EqualFold(valorParametro, "utf-8") {
			return errors.New("el motor declaro parametros JSON no admitidos")
		}
	}
	return nil
}

func parsearDestino(urlBase string) (destino, string, error) {
	if urlBase == "" || urlBase != strings.TrimSpace(urlBase) {
		return destino{}, "", errors.New("VEC_OSRM_BASE_URL debe ser explicita y canonica")
	}
	parseada, err := url.Parse(urlBase)
	if err != nil || !parseada.IsAbs() || parseada.Host == "" {
		return destino{}, "", errors.New("VEC_OSRM_BASE_URL no es una URL absoluta valida")
	}
	if parseada.Scheme != "http" && parseada.Scheme != "https" {
		return destino{}, "", errors.New("solo se admiten los esquemas http y https")
	}
	if parseada.User != nil || parseada.Opaque != "" || parseada.Path != "" || parseada.RawPath != "" ||
		parseada.RawQuery != "" || parseada.ForceQuery || parseada.Fragment != "" {
		return destino{}, "", errors.New("la URL no admite credenciales, ruta, consulta ni fragmento")
	}
	host, err := hostCanonico(parseada.Hostname())
	if err != nil {
		return destino{}, "", err
	}
	if host == "router.project-osrm.org" {
		return destino{}, "", errors.New("el OSRM publico no es un destino autorizado")
	}
	puerto := parseada.Port()
	if puerto != "" {
		numero, err := strconv.Atoi(puerto)
		if err != nil || numero < 1 || numero > 65535 || strconv.Itoa(numero) != puerto {
			return destino{}, "", errors.New("el puerto no es canonico")
		}
	}
	puertoEfectivo := puerto
	if puertoEfectivo == "" {
		if parseada.Scheme == "https" {
			puertoEfectivo = "443"
		} else {
			puertoEfectivo = "80"
		}
	}
	autoridad := host
	if strings.Contains(host, ":") {
		autoridad = "[" + host + "]"
	}
	if puerto != "" {
		autoridad = net.JoinHostPort(host, puerto)
	}
	canonica := parseada.Scheme + "://" + autoridad
	if urlBase != canonica {
		return destino{}, "", fmt.Errorf("VEC_OSRM_BASE_URL no es canonica; usa %q", canonica)
	}
	return destino{host: host, puerto: puertoEfectivo}, canonica, nil
}

func hostCanonico(host string) (string, error) {
	if host == "" || host != strings.TrimSpace(host) || len(host) > 253 {
		return "", errors.New("el host no es valido")
	}
	if direccion, err := netip.ParseAddr(host); err == nil {
		if direccion.Zone() != "" || direccion.Is4In6() || direccion.String() != host {
			return "", errors.New("la direccion IP no es canonica")
		}
		return host, nil
	}
	if host != strings.ToLower(host) || strings.HasSuffix(host, ".") {
		return "", errors.New("el nombre DNS debe estar en minusculas y sin punto final")
	}
	for _, etiqueta := range strings.Split(host, ".") {
		if etiqueta == "" || len(etiqueta) > 63 || etiqueta[0] == '-' || etiqueta[len(etiqueta)-1] == '-' {
			return "", errors.New("el nombre DNS no es canonico")
		}
		for _, caracter := range etiqueta {
			if (caracter < 'a' || caracter > 'z') && (caracter < '0' || caracter > '9') && caracter != '-' {
				return "", errors.New("el nombre DNS contiene caracteres no admitidos")
			}
		}
	}
	return host, nil
}

func parsearCIDRPermitidas(valores []string) ([]netip.Prefix, error) {
	if len(valores) == 0 || len(valores) > 32 {
		return nil, errors.New("se deben enumerar entre una y 32 redes")
	}
	resultado := make([]netip.Prefix, 0, len(valores))
	vistas := make(map[netip.Prefix]struct{}, len(valores))
	for _, valor := range valores {
		if valor == "" || valor != strings.TrimSpace(valor) {
			return nil, errors.New("red vacia o no canonica")
		}
		prefijo, err := netip.ParsePrefix(valor)
		if err != nil || prefijo.Addr().Zone() != "" || prefijo.Addr().Is4In6() {
			return nil, fmt.Errorf("red %q no valida", valor)
		}
		prefijo = prefijo.Masked()
		if prefijo.String() != valor || prefijo.Bits() == 0 || prefijo.Addr().IsMulticast() {
			return nil, fmt.Errorf("red %q no es una concesion concreta y canonica", valor)
		}
		if _, repetida := vistas[prefijo]; repetida {
			return nil, fmt.Errorf("red %q repetida", valor)
		}
		vistas[prefijo] = struct{}{}
		resultado = append(resultado, prefijo)
	}
	return resultado, nil
}

func nuevoClienteHTTP(destino destino) *http.Client {
	transporte := &http.Transport{
		Proxy:                  nil,
		DialContext:            destino.marcar,
		ForceAttemptHTTP2:      true,
		DisableCompression:     true,
		MaxIdleConns:           4,
		MaxIdleConnsPerHost:    2,
		MaxConnsPerHost:        8,
		IdleConnTimeout:        30 * time.Second,
		TLSHandshakeTimeout:    5 * time.Second,
		ResponseHeaderTimeout:  10 * time.Second,
		ExpectContinueTimeout:  time.Second,
		MaxResponseHeaderBytes: 64 << 10,
		TLSClientConfig:        &tls.Config{MinVersion: tls.VersionTLS13},
	}
	return &http.Client{
		Transport: transporte,
		Timeout:   tiempoMaximoSolicitud,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func (d destino) marcar(ctx context.Context, red, direccion string) (net.Conn, error) {
	if red != "tcp" && red != "tcp4" && red != "tcp6" {
		return nil, errors.New("OSRM: protocolo de transporte no autorizado")
	}
	host, puerto, err := net.SplitHostPort(direccion)
	if err != nil || host != d.host || puerto != d.puerto {
		return nil, errors.New("OSRM: destino distinto del configurado")
	}
	direcciones, err := d.resolver(ctx)
	if err != nil {
		return nil, err
	}
	marcador := net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	var ultimoError error
	for _, candidata := range direcciones {
		candidata = candidata.Unmap()
		if !d.permite(candidata) {
			continue
		}
		conexion, errorConexion := marcador.DialContext(ctx, red, net.JoinHostPort(candidata.String(), puerto))
		if errorConexion == nil {
			return conexion, nil
		}
		ultimoError = errorConexion
	}
	if ultimoError != nil {
		return nil, fmt.Errorf("OSRM: no se pudo conectar con una direccion autorizada: %w", ultimoError)
	}
	return nil, errors.New("OSRM: el host no resuelve dentro de las redes autorizadas")
}

func (d destino) resolver(ctx context.Context) ([]netip.Addr, error) {
	if direccion, err := netip.ParseAddr(d.host); err == nil {
		return []netip.Addr{direccion}, nil
	}
	direcciones, err := net.DefaultResolver.LookupNetIP(ctx, "ip", d.host)
	if err != nil || len(direcciones) == 0 {
		return nil, errors.New("OSRM: no se pudo resolver el host configurado")
	}
	return direcciones, nil
}

func (d destino) permite(direccion netip.Addr) bool {
	if !direccion.IsValid() || direccion.IsUnspecified() || direccion.IsMulticast() {
		return false
	}
	for _, prefijo := range d.cidrPermitidas {
		if prefijo.Contains(direccion) {
			return true
		}
	}
	return false
}

func construirEndpoint(urlBase string, solicitud dietasports.SolicitudCalculoRuta) (string, error) {
	_, urlCanonica, err := parsearDestino(urlBase)
	if err != nil {
		return "", fmt.Errorf("%w: destino no valido", dietasports.ErrMotorRutasNoDisponible)
	}
	partes := make([]string, 0, len(solicitud.Coordenadas))
	for _, coordenada := range solicitud.Coordenadas {
		partes = append(partes, strconv.FormatFloat(coordenada.Longitud, 'f', 6, 64)+","+
			strconv.FormatFloat(coordenada.Latitud, 'f', 6, 64))
	}
	alternativas := solicitud.Alternativas
	if alternativas == 0 {
		alternativas = 1
	}
	return urlCanonica + "/route/v1/driving/" + strings.Join(partes, ";") +
		"?overview=full&geometries=geojson&steps=false&alternatives=" + strconv.Itoa(alternativas), nil
}

var _ dietasports.CalculadorRutas = (*Calculador)(nil)
