package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"vec-diputacion-granada/internal/candidate/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	versionCredencialesFake      = 1
	tamanoMaximoCredencialesFake = 1 << 20
	maximoCredencialesFake       = 1_000
	longitudMinimaTokenFake      = 43
	longitudMaximaTokenFake      = 128
)

var errCredencialFakeNoValida = errors.New("bootstrap: credencial fake no valida")

type archivoCredencialesFake struct {
	Version     int                      `json:"version"`
	Credentials []registroCredencialFake `json:"credentials"`
}

type registroCredencialFake struct {
	TokenSHA256 string                  `json:"token_sha256"`
	Subject     string                  `json:"subject"`
	DisplayName string                  `json:"display_name"`
	Email       string                  `json:"email,omitempty"`
	Roles       []string                `json:"roles"`
	Mechanism   ports.AuthMechanism     `json:"mechanism"`
	Assurance   vecdomain.AuthAssurance `json:"assurance"`
	LegacyRole  ports.AuthRole          `json:"legacy_role"`
}

type identidadCredencialFake struct {
	principalLegacy ports.AuthPrincipal
	principalVEC    vecdomain.Principal
}

// almacenCredencialesFake conserva solamente huellas SHA-256. El token opaco
// se presenta en cada peticion, pero nunca se carga ni se persiste en claro.
type almacenCredencialesFake struct {
	porHuella map[[sha256.Size]byte]identidadCredencialFake
}

func cargarCredencialesFake(ruta string) (*almacenCredencialesFake, error) {
	ruta = strings.TrimSpace(ruta)
	if ruta == "" {
		return nil, fmt.Errorf("%w: VEC_FAKE_CREDENTIALS_FILE es obligatorio", errCredencialFakeNoValida)
	}

	informacionInicial, err := os.Lstat(ruta)
	if err != nil {
		return nil, fmt.Errorf("%w: no se puede abrir el fichero configurado", errCredencialFakeNoValida)
	}
	if err := validarFicheroCredencialesFake(informacionInicial); err != nil {
		return nil, err
	}

	fichero, err := os.Open(ruta)
	if err != nil {
		return nil, fmt.Errorf("%w: no se puede abrir el fichero configurado", errCredencialFakeNoValida)
	}
	defer fichero.Close()

	informacionAbierta, err := fichero.Stat()
	if err != nil || !os.SameFile(informacionInicial, informacionAbierta) {
		return nil, fmt.Errorf("%w: el fichero cambio durante su apertura", errCredencialFakeNoValida)
	}
	if err := validarFicheroCredencialesFake(informacionAbierta); err != nil {
		return nil, err
	}

	contenidoJSON, err := io.ReadAll(io.LimitReader(fichero, tamanoMaximoCredencialesFake+1))
	if err != nil || len(contenidoJSON) == 0 || len(contenidoJSON) > tamanoMaximoCredencialesFake {
		return nil, fmt.Errorf("%w: no se pudo leer el fichero completo", errCredencialFakeNoValida)
	}
	if err := validarClavesJSONUnicas(contenidoJSON); err != nil {
		return nil, err
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenidoJSON))
	decodificador.DisallowUnknownFields()
	var contenido archivoCredencialesFake
	if err := decodificador.Decode(&contenido); err != nil {
		return nil, fmt.Errorf("%w: JSON incorrecto", errCredencialFakeNoValida)
	}
	var sobrante any
	if err := decodificador.Decode(&sobrante); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: el fichero debe contener un unico documento JSON", errCredencialFakeNoValida)
	}
	if contenido.Version != versionCredencialesFake || len(contenido.Credentials) == 0 ||
		len(contenido.Credentials) > maximoCredencialesFake {
		return nil, fmt.Errorf("%w: version o numero de credenciales incorrecto", errCredencialFakeNoValida)
	}

	almacen := &almacenCredencialesFake{
		porHuella: make(map[[sha256.Size]byte]identidadCredencialFake, len(contenido.Credentials)),
	}
	for indice, registro := range contenido.Credentials {
		huella, identidad, err := validarRegistroCredencialFake(registro)
		if err != nil {
			return nil, fmt.Errorf("%w: registro %d", err, indice+1)
		}
		if _, repetida := almacen.porHuella[huella]; repetida {
			return nil, fmt.Errorf("%w: huella de token duplicada", errCredencialFakeNoValida)
		}
		almacen.porHuella[huella] = identidad
	}
	return almacen, nil
}

func validarClavesJSONUnicas(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	if err := consumirValorJSONSinDuplicados(decodificador); err != nil {
		return fmt.Errorf("%w: JSON con claves duplicadas o estructura incorrecta", errCredencialFakeNoValida)
	}
	if _, err := decodificador.Token(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: contenido JSON adicional", errCredencialFakeNoValida)
	}
	return nil
}

func consumirValorJSONSinDuplicados(decodificador *json.Decoder) error {
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
		claves := map[string]struct{}{}
		for decodificador.More() {
			tokenClave, err := decodificador.Token()
			if err != nil {
				return err
			}
			clave, ok := tokenClave.(string)
			if !ok {
				return errors.New("clave JSON no textual")
			}
			if _, repetida := claves[clave]; repetida {
				return errors.New("clave JSON duplicada")
			}
			claves[clave] = struct{}{}
			if err := consumirValorJSONSinDuplicados(decodificador); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim('}') {
			return errors.New("objeto JSON sin cierre")
		}
	case '[':
		for decodificador.More() {
			if err := consumirValorJSONSinDuplicados(decodificador); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim(']') {
			return errors.New("lista JSON sin cierre")
		}
	default:
		return errors.New("delimitador JSON inesperado")
	}
	return nil
}

func validarFicheroCredencialesFake(informacion os.FileInfo) error {
	if informacion == nil || !informacion.Mode().IsRegular() {
		return fmt.Errorf("%w: se exige un fichero regular, no un enlace", errCredencialFakeNoValida)
	}
	if informacion.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("%w: el fichero no puede conceder permisos a grupo ni a otros", errCredencialFakeNoValida)
	}
	if informacion.Size() <= 0 || informacion.Size() > tamanoMaximoCredencialesFake {
		return fmt.Errorf("%w: tamano de fichero incorrecto", errCredencialFakeNoValida)
	}
	return nil
}

func validarRegistroCredencialFake(registro registroCredencialFake) ([sha256.Size]byte, identidadCredencialFake, error) {
	var huella [sha256.Size]byte
	if len(registro.TokenSHA256) != hex.EncodedLen(sha256.Size) ||
		registro.TokenSHA256 != strings.ToLower(registro.TokenSHA256) {
		return huella, identidadCredencialFake{}, fmt.Errorf("%w: token_sha256 debe ser hexadecimal minusculo SHA-256", errCredencialFakeNoValida)
	}
	bytesHuella, err := hex.DecodeString(registro.TokenSHA256)
	if err != nil {
		return huella, identidadCredencialFake{}, fmt.Errorf("%w: token_sha256 incorrecto", errCredencialFakeNoValida)
	}
	copy(huella[:], bytesHuella)

	if registro.Subject != strings.TrimSpace(registro.Subject) || registro.Subject == "" ||
		registro.DisplayName != strings.TrimSpace(registro.DisplayName) || registro.DisplayName == "" ||
		registro.Email != strings.TrimSpace(registro.Email) {
		return huella, identidadCredencialFake{}, fmt.Errorf("%w: identidad ausente o no canonica", errCredencialFakeNoValida)
	}
	if len(registro.Roles) != 1 {
		return huella, identidadCredencialFake{}, fmt.Errorf("%w: se exige un unico rol VEC exacto", errCredencialFakeNoValida)
	}
	roles := append([]string(nil), registro.Roles...)
	for _, rol := range roles {
		if rol == "" || rol != strings.TrimSpace(rol) {
			return huella, identidadCredencialFake{}, fmt.Errorf("%w: rol vacio o no canonico", errCredencialFakeNoValida)
		}
	}

	metodoVEC, ok := metodoVECFake(registro.Mechanism)
	if !ok || !registro.Assurance.Valida() || !registro.LegacyRole.IsValid() {
		return huella, identidadCredencialFake{}, fmt.Errorf("%w: mecanismo, garantia o rol legacy incorrecto", errCredencialFakeNoValida)
	}
	rolLegacyEsperado, ok := rolLegacyCompatibleFake(roles[0])
	if !ok || registro.LegacyRole != rolLegacyEsperado {
		return huella, identidadCredencialFake{}, fmt.Errorf("%w: combinacion de roles VEC y legacy no permitida", errCredencialFakeNoValida)
	}
	principalLegacy := ports.AuthPrincipal{
		Subject:     registro.Subject,
		DisplayName: registro.DisplayName,
		Email:       registro.Email,
		Role:        registro.LegacyRole,
		Roles:       []ports.AuthRole{registro.LegacyRole},
		Mechanism:   registro.Mechanism,
		Method:      registro.Mechanism,
	}
	if err := principalLegacy.Validate(); err != nil {
		return huella, identidadCredencialFake{}, fmt.Errorf("%w: principal legacy incorrecto", errCredencialFakeNoValida)
	}
	principalVEC := vecdomain.Principal{
		ID:            registro.Subject,
		DisplayName:   registro.DisplayName,
		Email:         registro.Email,
		Roles:         roles,
		Permissions:   []string{},
		AuthMethod:    metodoVEC,
		AuthAssurance: registro.Assurance,
	}
	if err := principalVEC.Validate(); err != nil {
		return huella, identidadCredencialFake{}, fmt.Errorf("%w: principal VEC incorrecto", errCredencialFakeNoValida)
	}
	return huella, identidadCredencialFake{
		principalLegacy: principalLegacy,
		principalVEC:    principalVEC,
	}, nil
}

// rolLegacyCompatibleFake es una lista positiva deliberadamente pequena. La
// API heredada de Bolsa aun concede capacidades mediante cuatro perfiles
// gruesos; solo se permite emparejar cada uno con el perfil VEC canonico que
// expresa la misma responsabilidad. Un perfil nuevo o un alias no hereda una
// equivalencia por semejanza de nombre.
func rolLegacyCompatibleFake(rolVEC string) (ports.AuthRole, bool) {
	switch rolVEC {
	case "ciudadano":
		return ports.AuthRoleCandidate, true
	case "administrativo":
		return ports.AuthRoleValidatorL1, true
	case "tecnico_rrhh", "jefatura_rrhh":
		return ports.AuthRoleValidatorL2, true
	case "administrador":
		return ports.AuthRoleSystemAdmin, true
	default:
		return "", false
	}
}

func metodoVECFake(metodo ports.AuthMechanism) (vecdomain.AuthMethod, bool) {
	switch metodo {
	case ports.AuthMechanismClave:
		return vecdomain.AuthMethodClave, true
	case ports.AuthMechanismDNIe:
		return vecdomain.AuthMethodDNIe, true
	case ports.AuthMechanismKerberosAD:
		return vecdomain.AuthMethodKerberos, true
	default:
		return "", false
	}
}

func (a *almacenCredencialesFake) AuthenticateRequest(ctx context.Context, peticion *http.Request) (ports.AuthPrincipal, error) {
	identidad, err := a.resolverPeticion(ctx, peticion)
	if err != nil {
		return ports.AuthPrincipal{}, err
	}
	return clonarPrincipalLegacyFake(identidad.principalLegacy), nil
}

func (a *almacenCredencialesFake) Authenticate(ctx context.Context, credenciales ports.AuthCredentials) (ports.AuthPrincipal, error) {
	if a == nil || a.porHuella == nil {
		return ports.AuthPrincipal{}, ports.ErrAuthenticationFailed
	}
	if err := ctx.Err(); err != nil {
		return ports.AuthPrincipal{}, err
	}
	if err := credenciales.Validate(); err != nil {
		return ports.AuthPrincipal{}, err
	}
	identidad, err := a.resolverToken(credenciales.Token)
	if err != nil || identidad.principalLegacy.Subject != credenciales.Subject ||
		identidad.principalLegacy.AuthMethod() != credenciales.Mechanism {
		return ports.AuthPrincipal{}, ports.ErrAuthenticationFailed
	}
	return clonarPrincipalLegacyFake(identidad.principalLegacy), nil
}

func (a *almacenCredencialesFake) ResolveDemoIdentity(ctx context.Context, peticion *http.Request) (vecdomain.Principal, error) {
	identidad, err := a.resolverPeticion(ctx, peticion)
	if err != nil {
		return vecdomain.Principal{}, err
	}
	return clonarPrincipalVECFake(identidad.principalVEC), nil
}

func (a *almacenCredencialesFake) resolverPeticion(ctx context.Context, peticion *http.Request) (identidadCredencialFake, error) {
	if a == nil || a.porHuella == nil || peticion == nil {
		return identidadCredencialFake{}, ports.ErrAuthenticationFailed
	}
	if err := ctx.Err(); err != nil {
		return identidadCredencialFake{}, err
	}
	if !peticionFakeProcedeDeLoopback(peticion) || contieneCabeceraProxyFake(peticion.Header) {
		return identidadCredencialFake{}, ports.ErrAuthenticationFailed
	}
	token, err := tokenBearerFake(peticion.Header.Values("Authorization"))
	if err != nil {
		return identidadCredencialFake{}, ports.ErrAuthenticationFailed
	}
	return a.resolverToken(token)
}

func peticionFakeProcedeDeLoopback(peticion *http.Request) bool {
	if peticion == nil || peticion.RemoteAddr != strings.TrimSpace(peticion.RemoteAddr) {
		return false
	}
	host, puertoTexto, err := net.SplitHostPort(peticion.RemoteAddr)
	if err != nil || host == "" || puertoTexto == "" {
		return false
	}
	ip := net.ParseIP(host)
	puerto, err := strconv.Atoi(puertoTexto)
	return err == nil && puerto > 0 && puerto <= 65535 && ip != nil && ip.IsLoopback()
}

func contieneCabeceraProxyFake(cabeceras http.Header) bool {
	for nombre := range cabeceras {
		nombre = strings.TrimSpace(nombre)
		if strings.EqualFold(nombre, "Forwarded") || strings.EqualFold(nombre, "Via") ||
			strings.HasPrefix(strings.ToLower(nombre), "x-forwarded-") {
			return true
		}
	}
	return false
}

func (a *almacenCredencialesFake) resolverToken(token string) (identidadCredencialFake, error) {
	if !tokenOpacoFakeValido(token) {
		return identidadCredencialFake{}, ports.ErrAuthenticationFailed
	}
	huella := sha256.Sum256([]byte(token))
	identidad, existe := a.porHuella[huella]
	if !existe {
		return identidadCredencialFake{}, ports.ErrAuthenticationFailed
	}
	return identidad, nil
}

func tokenBearerFake(valores []string) (string, error) {
	if len(valores) != 1 {
		return "", ports.ErrAuthenticationFailed
	}
	valor := valores[0]
	separador := strings.IndexByte(valor, ' ')
	if separador <= 0 || separador == len(valor)-1 || strings.Contains(valor[separador+1:], " ") ||
		!strings.EqualFold(valor[:separador], "Bearer") {
		return "", ports.ErrAuthenticationFailed
	}
	token := valor[separador+1:]
	if !tokenOpacoFakeValido(token) {
		return "", ports.ErrAuthenticationFailed
	}
	return token, nil
}

func tokenOpacoFakeValido(token string) bool {
	if len(token) < longitudMinimaTokenFake || len(token) > longitudMaximaTokenFake {
		return false
	}
	for _, caracter := range token {
		if (caracter >= 'a' && caracter <= 'z') || (caracter >= 'A' && caracter <= 'Z') ||
			(caracter >= '0' && caracter <= '9') || caracter == '-' || caracter == '_' {
			continue
		}
		return false
	}
	return true
}

func clonarPrincipalLegacyFake(principal ports.AuthPrincipal) ports.AuthPrincipal {
	principal.Roles = append([]ports.AuthRole(nil), principal.Roles...)
	if principal.Attributes != nil {
		atributos := make(map[string]string, len(principal.Attributes))
		for clave, valor := range principal.Attributes {
			atributos[clave] = valor
		}
		principal.Attributes = atributos
	}
	return principal
}

func clonarPrincipalVECFake(principal vecdomain.Principal) vecdomain.Principal {
	principal.Roles = append([]string(nil), principal.Roles...)
	principal.Permissions = append([]string(nil), principal.Permissions...)
	if principal.Attributes != nil {
		atributos := make(map[string]string, len(principal.Attributes))
		for clave, valor := range principal.Attributes {
			atributos[clave] = valor
		}
		principal.Attributes = atributos
	}
	return principal
}
