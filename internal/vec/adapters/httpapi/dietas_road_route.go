package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	dietasmodule "vec-diputacion-granada/internal/modules/dietas"
	"vec-diputacion-granada/internal/vec/domain"
)

const (
	maxRoadRouteCoordinates    = 25
	maxRoadRouteScopeNameBytes = 160
	maxRoadRouteResponseBytes  = 20 << 20
	roadRouteRequestTimeout    = 15 * time.Second
)

type roadRouteRequest struct {
	Coordinates  []roadRouteCoordinate `json:"coordinates"`
	Alternatives int                   `json:"alternatives"`
}

type roadRouteCoordinate struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	Name      string  `json:"name"`
}

type roadRouteBounds struct {
	minLatitude  float64
	maxLatitude  float64
	minLongitude float64
	maxLongitude float64
}

type roadRouteScope struct {
	name   string
	bounds roadRouteBounds
}

type roadRouteTarget struct {
	host         string
	port         string
	allowedCIDRs []netip.Prefix
}

type roadRouteConnector struct {
	baseURL string
	scope   roadRouteScope
	client  *http.Client
}

func (h *Handler) handleDietasRoadRoute(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	if !h.requireMethod(w, r, http.MethodPost) {
		return
	}
	if !principal.HasPermission(dietasmodule.PermissionRouteRead) {
		h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
		return
	}
	if h.roadRoute == nil {
		h.writeError(w, http.StatusServiceUnavailable, "Motor OSRM interno no configurado de forma completa y explicita.")
		return
	}
	var payload roadRouteRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		h.writeError(w, http.StatusBadRequest, "peticion de ruta invalida")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		h.writeError(w, http.StatusBadRequest, "peticion de ruta invalida")
		return
	}
	if err := validateRoadRouteCoordinates(payload.Coordinates, h.roadRoute.scope); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateRoadRouteAlternatives(payload.Alternatives); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	endpoint, err := osrmRoadRouteEndpoint(h.roadRoute.baseURL, payload.Coordinates, payload.Alternatives)
	if err != nil {
		h.writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), roadRouteRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		h.writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	req.Header.Set("Accept", "application/json")
	response, err := h.roadRoute.client.Do(req)
	if err != nil {
		h.writeError(w, http.StatusBadGateway, "no se pudo consultar el motor OSRM interno autorizado")
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		h.writeError(w, http.StatusBadGateway, fmt.Sprintf("OSRM interno respondio con estado %d", response.StatusCode))
		return
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxRoadRouteResponseBytes+1))
	if err != nil {
		h.writeError(w, http.StatusBadGateway, "no se pudo leer la respuesta del OSRM interno")
		return
	}
	if len(body) > maxRoadRouteResponseBytes {
		h.writeError(w, http.StatusBadGateway, "OSRM interno excedio el limite de respuesta")
		return
	}
	var osrm map[string]any
	if err := json.Unmarshal(body, &osrm); err != nil {
		h.writeError(w, http.StatusBadGateway, "OSRM interno devolvio JSON invalido")
		return
	}
	if codigo, correcto := osrm["code"].(string); !correcto || codigo != "Ok" {
		h.writeError(w, http.StatusBadGateway, "OSRM interno no confirmo una ruta valida")
		return
	}
	if _, correcto := osrm["routes"].([]any); !correcto {
		h.writeError(w, http.StatusBadGateway, "OSRM interno no devolvio una lista de rutas")
		return
	}
	osrm["engine"] = "osrm_on_premise"
	osrm["route_scope"] = h.roadRoute.scope.name
	h.writeJSON(w, http.StatusOK, osrm)
}

func validateRoadRouteAlternatives(alternatives int) error {
	// Cero significa que el cliente no ha solicitado esta opcion y conserva el
	// minimo funcional: una unica ruta. Un valor explicito fuera del contrato se
	// deniega; nunca se recorta ni se amplia silenciosamente.
	if alternatives < 0 || alternatives > 3 {
		return errors.New("el numero de alternativas debe estar entre uno y tres")
	}
	return nil
}

func validateRoadRouteCoordinates(coords []roadRouteCoordinate, scope roadRouteScope) error {
	if len(coords) < 2 {
		return errors.New("la ruta necesita al menos dos coordenadas")
	}
	if len(coords) > maxRoadRouteCoordinates {
		return fmt.Errorf("la ruta admite como maximo %d coordenadas", maxRoadRouteCoordinates)
	}
	for index, coord := range coords {
		if !validLatitude(coord.Latitude) || !validLongitude(coord.Longitude) {
			return fmt.Errorf("coordenada %d fuera de rango", index+1)
		}
		if !scope.bounds.contains(coord) {
			return fmt.Errorf("coordenada %d fuera del ambito %s", index+1, scope.name)
		}
	}
	return nil
}

func validLatitude(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= -90 && value <= 90
}

func validLongitude(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= -180 && value <= 180
}

func (bounds roadRouteBounds) contains(coord roadRouteCoordinate) bool {
	return coord.Latitude >= bounds.minLatitude &&
		coord.Latitude <= bounds.maxLatitude &&
		coord.Longitude >= bounds.minLongitude &&
		coord.Longitude <= bounds.maxLongitude
}

func newRoadRouteConnector(
	baseURL string,
	scopeName string,
	scopeBounds string,
	allowedCIDRs []string,
) (*roadRouteConnector, error) {
	configured := strings.TrimSpace(baseURL) != "" || strings.TrimSpace(scopeName) != "" ||
		strings.TrimSpace(scopeBounds) != "" || len(allowedCIDRs) != 0
	if !configured {
		return nil, nil
	}
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(scopeName) == "" ||
		strings.TrimSpace(scopeBounds) == "" || len(allowedCIDRs) == 0 {
		return nil, errors.New("vec http dietas osrm: URL, nombre, limites y redes permitidas son obligatorios")
	}

	scope, err := parseRoadRouteScope(scopeName, scopeBounds)
	if err != nil {
		return nil, fmt.Errorf("vec http dietas osrm: ambito no valido: %w", err)
	}
	target, canonicalBaseURL, err := parseRoadRouteTarget(baseURL)
	if err != nil {
		return nil, fmt.Errorf("vec http dietas osrm: destino no valido: %w", err)
	}
	target.allowedCIDRs, err = parseRoadRouteAllowedCIDRs(allowedCIDRs)
	if err != nil {
		return nil, fmt.Errorf("vec http dietas osrm: redes no validas: %w", err)
	}

	return &roadRouteConnector{
		baseURL: canonicalBaseURL,
		scope:   scope,
		client:  newRoadRouteHTTPClient(target),
	}, nil
}

func parseRoadRouteScope(name, rawBounds string) (roadRouteScope, error) {
	if name == "" || name != strings.TrimSpace(name) || len(name) > maxRoadRouteScopeNameBytes || !utf8.ValidString(name) {
		return roadRouteScope{}, errors.New("el nombre debe ser explicito, canonico y no vacio")
	}
	for _, character := range name {
		if unicode.IsControl(character) {
			return roadRouteScope{}, errors.New("el nombre contiene caracteres de control")
		}
	}
	bounds, err := parseRoadRouteBounds(rawBounds)
	if err != nil {
		return roadRouteScope{}, err
	}
	return roadRouteScope{name: name, bounds: bounds}, nil
}

func parseRoadRouteBounds(raw string) (roadRouteBounds, error) {
	if raw == "" || raw != strings.TrimSpace(raw) {
		return roadRouteBounds{}, errors.New("los limites deben declararse en formato canonico minLat,minLon,maxLat,maxLon")
	}
	parts := strings.Split(raw, ",")
	if len(parts) != 4 {
		return roadRouteBounds{}, errors.New("los limites deben contener cuatro numeros")
	}
	numbers := make([]float64, 4)
	for index, part := range parts {
		if part == "" || part != strings.TrimSpace(part) {
			return roadRouteBounds{}, errors.New("los limites no admiten valores vacios ni espacios")
		}
		parsed, err := strconv.ParseFloat(part, 64)
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return roadRouteBounds{}, errors.New("los limites contienen un numero no valido")
		}
		if canonical := strconv.FormatFloat(parsed, 'f', -1, 64); canonical != part {
			return roadRouteBounds{}, fmt.Errorf("el limite %q no es canonico; usa %q", part, canonical)
		}
		numbers[index] = parsed
	}
	if !validLatitude(numbers[0]) || !validLongitude(numbers[1]) ||
		!validLatitude(numbers[2]) || !validLongitude(numbers[3]) {
		return roadRouteBounds{}, errors.New("los limites exceden los rangos de latitud o longitud")
	}
	bounds := roadRouteBounds{
		minLatitude:  numbers[0],
		minLongitude: numbers[1],
		maxLatitude:  numbers[2],
		maxLongitude: numbers[3],
	}
	if bounds.minLatitude >= bounds.maxLatitude || bounds.minLongitude >= bounds.maxLongitude {
		return roadRouteBounds{}, errors.New("los limites minimos deben ser inferiores a los maximos")
	}
	return bounds, nil
}

func parseRoadRouteTarget(baseURL string) (roadRouteTarget, string, error) {
	if baseURL == "" || baseURL != strings.TrimSpace(baseURL) {
		return roadRouteTarget{}, "", errors.New("VEC_OSRM_BASE_URL debe ser explicita y canonica")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" {
		return roadRouteTarget{}, "", errors.New("VEC_OSRM_BASE_URL no es una URL absoluta valida")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return roadRouteTarget{}, "", errors.New("VEC_OSRM_BASE_URL solo admite los esquemas http o https declarados expresamente")
	}
	if parsed.User != nil || parsed.Opaque != "" || parsed.Path != "" || parsed.RawPath != "" ||
		parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return roadRouteTarget{}, "", errors.New("VEC_OSRM_BASE_URL no admite credenciales, ruta, consulta ni fragmento")
	}

	host, err := canonicalRoadRouteHost(parsed.Hostname())
	if err != nil {
		return roadRouteTarget{}, "", err
	}
	if host == "router.project-osrm.org" {
		return roadRouteTarget{}, "", errors.New("el OSRM publico de demostracion no es un destino autorizado")
	}
	port := parsed.Port()
	if port != "" {
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 || strconv.Itoa(portNumber) != port {
			return roadRouteTarget{}, "", errors.New("el puerto del destino no es canonico")
		}
	}
	effectivePort := port
	if effectivePort == "" {
		if parsed.Scheme == "https" {
			effectivePort = "443"
		} else {
			effectivePort = "80"
		}
	}
	authority := host
	if strings.Contains(host, ":") {
		authority = "[" + host + "]"
	}
	if port != "" {
		authority = net.JoinHostPort(host, port)
	}
	canonicalBaseURL := parsed.Scheme + "://" + authority
	if baseURL != canonicalBaseURL {
		return roadRouteTarget{}, "", fmt.Errorf("VEC_OSRM_BASE_URL no es canonica; usa %q", canonicalBaseURL)
	}
	return roadRouteTarget{host: host, port: effectivePort}, canonicalBaseURL, nil
}

func canonicalRoadRouteHost(host string) (string, error) {
	if host == "" || host != strings.TrimSpace(host) || len(host) > 253 {
		return "", errors.New("el host del destino no es valido")
	}
	if address, err := netip.ParseAddr(host); err == nil {
		if address.Zone() != "" || address.Is4In6() || address.String() != host {
			return "", errors.New("la direccion IP del destino no es canonica")
		}
		return host, nil
	}
	if host != strings.ToLower(host) || strings.HasSuffix(host, ".") {
		return "", errors.New("el nombre DNS del destino debe estar en minusculas y sin punto final")
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", errors.New("el nombre DNS del destino no es canonico")
		}
		for _, character := range label {
			if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
				return "", errors.New("el nombre DNS del destino contiene caracteres no admitidos")
			}
		}
	}
	return host, nil
}

func parseRoadRouteAllowedCIDRs(values []string) ([]netip.Prefix, error) {
	if len(values) == 0 || len(values) > 32 {
		return nil, errors.New("VEC_OSRM_ALLOWED_CIDRS debe enumerar entre una y 32 redes")
	}
	prefixes := make([]netip.Prefix, 0, len(values))
	seen := make(map[netip.Prefix]struct{}, len(values))
	for _, value := range values {
		if value == "" || value != strings.TrimSpace(value) {
			return nil, errors.New("VEC_OSRM_ALLOWED_CIDRS contiene una red vacia o no canonica")
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil || prefix.Addr().Zone() != "" || prefix.Addr().Is4In6() {
			return nil, fmt.Errorf("red %q no valida", value)
		}
		prefix = prefix.Masked()
		if prefix.String() != value || prefix.Bits() == 0 || prefix.Addr().IsMulticast() {
			return nil, fmt.Errorf("red %q no es una concesion concreta y canonica", value)
		}
		if _, duplicated := seen[prefix]; duplicated {
			return nil, fmt.Errorf("red %q repetida", value)
		}
		seen[prefix] = struct{}{}
		prefixes = append(prefixes, prefix)
	}
	return prefixes, nil
}

func newRoadRouteHTTPClient(target roadRouteTarget) *http.Client {
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           target.dialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          4,
		MaxIdleConnsPerHost:   2,
		MaxConnsPerHost:       8,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   roadRouteRequestTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			// OSRM no necesita redirecciones. Bloquearlas todas evita que un destino
			// inicialmente autorizado derive la consulta a otro host, puerto o red.
			return http.ErrUseLastResponse
		},
	}
}

func (target roadRouteTarget) dialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		return nil, errors.New("OSRM: protocolo de transporte no autorizado")
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil || host != target.host || port != target.port {
		return nil, errors.New("OSRM: destino distinto del configurado")
	}
	addresses, err := target.resolve(ctx)
	if err != nil {
		return nil, err
	}
	dialer := net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	var lastErr error
	for _, candidate := range addresses {
		candidate = candidate.Unmap()
		if !target.allows(candidate) {
			continue
		}
		connection, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(candidate.String(), port))
		if dialErr == nil {
			return connection, nil
		}
		lastErr = dialErr
	}
	if lastErr != nil {
		return nil, fmt.Errorf("OSRM: no se pudo conectar con una direccion autorizada: %w", lastErr)
	}
	return nil, errors.New("OSRM: el host no resuelve dentro de las redes autorizadas")
}

func (target roadRouteTarget) resolve(ctx context.Context) ([]netip.Addr, error) {
	if address, err := netip.ParseAddr(target.host); err == nil {
		return []netip.Addr{address}, nil
	}
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", target.host)
	if err != nil || len(addresses) == 0 {
		return nil, errors.New("OSRM: no se pudo resolver el host configurado")
	}
	return addresses, nil
}

func (target roadRouteTarget) allows(address netip.Addr) bool {
	if !address.IsValid() || address.IsUnspecified() || address.IsMulticast() {
		return false
	}
	for _, prefix := range target.allowedCIDRs {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func osrmRoadRouteEndpoint(baseURL string, coords []roadRouteCoordinate, alternatives int) (string, error) {
	_, canonicalBaseURL, err := parseRoadRouteTarget(baseURL)
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(coords))
	for _, coord := range coords {
		parts = append(parts, strconv.FormatFloat(coord.Longitude, 'f', 6, 64)+","+strconv.FormatFloat(coord.Latitude, 'f', 6, 64))
	}
	if alternatives == 0 {
		alternatives = 1
	}
	if err := validateRoadRouteAlternatives(alternatives); err != nil {
		return "", err
	}
	return canonicalBaseURL + "/route/v1/driving/" + strings.Join(parts, ";") + "?overview=full&geometries=geojson&steps=false&alternatives=" + strconv.Itoa(alternatives), nil
}
