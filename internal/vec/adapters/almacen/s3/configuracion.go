// Package s3 implementa un conector de objetos compatible con la API S3.
// El paquete pertenece a la capa de adaptadores: ningun tipo del SDK cruza
// hacia el nucleo ni hacia los modulos de negocio.
package s3

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const (
	IdentificadorPredeterminado = "s3_compatible"
	tamanoMaximoPutS3           = int64(5 * 1024 * 1024 * 1024)
	duracionMaximaCargaDirecta  = 10 * time.Minute
)

var (
	ErrConfiguracionInvalida = errors.New("vec: configuracion del conector s3 invalida")
	ErrOperacionS3           = errors.New("vec: operacion del conector s3 no disponible")
	ErrSondaS3NoSuperada     = errors.New("vec: sonda fuerte del conector s3 no superada")
)

// Configuracion contiene exclusivamente parametros de composicion. Las
// credenciales no se incluyen nunca en errores, capacidades ni evidencias.
// Si no se proporcionan credenciales estaticas se usa la cadena segura del
// SDK (identidad de carga, fichero protegido o proveedor externo).
type Configuracion struct {
	ConectorID              string
	Endpoint                string
	Region                  string
	BucketCuarentena        string
	BucketAdmitida          string
	AccessKeyID             string
	SecretAccessKey         string
	SessionToken            string
	RutaCA                  string
	RedesPermitidas         []netip.Prefix
	PathStyle               bool
	TamanoMaximo            int64
	DuracionCargaDirecta    time.Duration
	RetencionMinimaAdmitida time.Duration
	ClaveDerivacion         []byte
	Cifrado                 types.ServerSideEncryption
	ClaveKMS                string
	UsarBucketKeyKMS        bool
	PerfilFuerte            bool
	ProbarCapacidades       bool
	PermitirEliminacion     bool
	ModoRetencion           types.ObjectLockRetentionMode
}

// Configuracion contiene credenciales y material de derivacion. Sus
// representaciones genericas se cierran para que un log, traza o respuesta de
// diagnostico no pueda volcarlos accidentalmente.
func (Configuracion) String() string   { return "configuracion_s3_redactada" }
func (Configuracion) GoString() string { return "configuracion_s3_redactada" }

func (Configuracion) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte("configuracion_s3_redactada"))
}

func (Configuracion) MarshalJSON() ([]byte, error) {
	return []byte(`"configuracion_s3_redactada"`), nil
}

func (Configuracion) MarshalText() ([]byte, error) {
	return []byte("configuracion_s3_redactada"), nil
}

func (Configuracion) LogValue() slog.Value {
	return slog.StringValue("configuracion_s3_redactada")
}

func ConfiguracionDesdeMapa(valores map[string]string) (Configuracion, error) {
	configuracion := Configuracion{
		ConectorID: IdentificadorPredeterminado,
		Region:     "us-east-1", PathStyle: true,
		TamanoMaximo:         256 * 1024 * 1024,
		DuracionCargaDirecta: 5 * time.Minute,
		Cifrado:              types.ServerSideEncryptionAes256,
		ModoRetencion:        types.ObjectLockRetentionModeCompliance,
	}
	if valor := strings.TrimSpace(valores["conector_id"]); valor != "" {
		configuracion.ConectorID = valor
	}
	configuracion.Endpoint = strings.TrimSpace(valores["endpoint"])
	if valor := strings.TrimSpace(valores["region"]); valor != "" {
		configuracion.Region = valor
	}
	configuracion.BucketCuarentena = strings.TrimSpace(valores["bucket_cuarentena"])
	configuracion.BucketAdmitida = strings.TrimSpace(valores["bucket_admitida"])
	configuracion.AccessKeyID = valores["access_key_id"]
	configuracion.SecretAccessKey = valores["secret_access_key"]
	configuracion.SessionToken = valores["session_token"]
	configuracion.RutaCA = strings.TrimSpace(valores["ruta_ca"])
	var err error
	configuracion.RedesPermitidas, err = parsearRedesPermitidas(valores["redes_permitidas"])
	if err != nil {
		return Configuracion{}, ErrConfiguracionInvalida
	}

	if valor := strings.TrimSpace(valores["path_style"]); valor != "" {
		configuracion.PathStyle, err = strconv.ParseBool(valor)
		if err != nil {
			return Configuracion{}, ErrConfiguracionInvalida
		}
	}
	if valor := strings.TrimSpace(valores["tamano_maximo_bytes"]); valor != "" {
		configuracion.TamanoMaximo, err = strconv.ParseInt(valor, 10, 64)
		if err != nil {
			return Configuracion{}, ErrConfiguracionInvalida
		}
	}
	if valor := strings.TrimSpace(valores["duracion_carga_directa"]); valor != "" {
		configuracion.DuracionCargaDirecta, err = time.ParseDuration(valor)
		if err != nil {
			return Configuracion{}, ErrConfiguracionInvalida
		}
	}
	if valor := strings.TrimSpace(valores["retencion_minima_admitida"]); valor != "" {
		configuracion.RetencionMinimaAdmitida, err = time.ParseDuration(valor)
		if err != nil {
			return Configuracion{}, ErrConfiguracionInvalida
		}
	}
	for clave, destino := range map[string]*bool{
		"perfil_fuerte":        &configuracion.PerfilFuerte,
		"probar_capacidades":   &configuracion.ProbarCapacidades,
		"permitir_eliminacion": &configuracion.PermitirEliminacion,
		"usar_bucket_key_kms":  &configuracion.UsarBucketKeyKMS,
	} {
		if valor := strings.TrimSpace(valores[clave]); valor != "" {
			*destino, err = strconv.ParseBool(valor)
			if err != nil {
				return Configuracion{}, ErrConfiguracionInvalida
			}
		}
	}
	clave, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(valores["clave_derivacion_base64url"]))
	if err != nil {
		return Configuracion{}, ErrConfiguracionInvalida
	}
	configuracion.ClaveDerivacion = clave
	switch strings.TrimSpace(valores["cifrado"]) {
	case "", "AES256":
		configuracion.Cifrado = types.ServerSideEncryptionAes256
	case "aws:kms":
		configuracion.Cifrado = types.ServerSideEncryptionAwsKms
	default:
		return Configuracion{}, ErrConfiguracionInvalida
	}
	configuracion.ClaveKMS = strings.TrimSpace(valores["clave_kms"])
	switch strings.ToUpper(strings.TrimSpace(valores["modo_retencion"])) {
	case "", "COMPLIANCE":
		configuracion.ModoRetencion = types.ObjectLockRetentionModeCompliance
	case "GOVERNANCE":
		configuracion.ModoRetencion = types.ObjectLockRetentionModeGovernance
	default:
		return Configuracion{}, ErrConfiguracionInvalida
	}
	if err := configuracion.Validar(); err != nil {
		return Configuracion{}, err
	}
	return configuracion, nil
}

func (c Configuracion) Validar() error {
	direccion, err := url.Parse(c.Endpoint)
	credencialesCompletas := (c.AccessKeyID == "") == (c.SecretAccessKey == "")
	if err != nil || direccion.Scheme != "https" || direccion.Host == "" || direccion.User != nil ||
		direccion.RawQuery != "" || direccion.Fragment != "" || direccion.Opaque != "" ||
		(direccion.Path != "" && direccion.Path != "/") ||
		!identificadorValido(c.ConectorID) || !textoTecnicoValido(c.Region, 128) ||
		!bucketValido(c.BucketCuarentena) || !bucketValido(c.BucketAdmitida) ||
		c.BucketCuarentena == c.BucketAdmitida || !credencialesCompletas || (c.SessionToken != "" && c.AccessKeyID == "") ||
		len(c.ClaveDerivacion) != 32 || c.TamanoMaximo < 1 || c.TamanoMaximo > tamanoMaximoPutS3 ||
		c.DuracionCargaDirecta <= 0 || c.DuracionCargaDirecta > duracionMaximaCargaDirecta ||
		(c.Cifrado != types.ServerSideEncryptionAes256 && c.Cifrado != types.ServerSideEncryptionAwsKms) ||
		(c.Cifrado == types.ServerSideEncryptionAwsKms && !textoTecnicoValido(c.ClaveKMS, 2048)) ||
		(c.Cifrado != types.ServerSideEncryptionAwsKms && (c.ClaveKMS != "" || c.UsarBucketKeyKMS)) ||
		(c.ModoRetencion != types.ObjectLockRetentionModeCompliance && c.ModoRetencion != types.ObjectLockRetentionModeGovernance) ||
		c.RetencionMinimaAdmitida < 0 || c.RetencionMinimaAdmitida > 100*365*24*time.Hour ||
		(c.PerfilFuerte && (!c.ProbarCapacidades || c.ModoRetencion != types.ObjectLockRetentionModeCompliance ||
			c.RetencionMinimaAdmitida < time.Hour || len(c.RedesPermitidas) == 0)) ||
		!redesPermitidasValidas(c.RedesPermitidas) {
		return ErrConfiguracionInvalida
	}
	return nil
}

func (c Configuracion) origen() string {
	direccion, err := url.Parse(c.Endpoint)
	if err != nil {
		return ""
	}
	return strings.ToLower(direccion.Scheme + "://" + direccion.Host)
}

func (c Configuracion) origenCargaDirecta() string {
	direccion, err := url.Parse(c.Endpoint)
	if err != nil {
		return ""
	}
	if c.PathStyle {
		return strings.ToLower(direccion.Scheme + "://" + direccion.Host)
	}
	return strings.ToLower(direccion.Scheme + "://" + c.BucketCuarentena + "." + direccion.Host)
}

func clienteHTTPSeguro(rutaCA string, redesPermitidas []netip.Prefix) (*http.Client, error) {
	raices, err := x509.SystemCertPool()
	if err != nil || raices == nil {
		raices = x509.NewCertPool()
	}
	if rutaCA != "" {
		pem, err := os.ReadFile(rutaCA)
		if err != nil || !raices.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("cargar autoridad TLS del almacen: %w", ErrConfiguracionInvalida)
		}
	}
	transporte := http.DefaultTransport.(*http.Transport).Clone()
	// El almacen documental no hereda HTTPS_PROXY del proceso. Una variable de
	// entorno olvidada no puede convertir un proxy ajeno en intermediario de
	// documentos sensibles. Si un despliegue necesitara proxy, debera existir
	// un conector explicito con destino, CA y politica de red gobernados.
	transporte.Proxy = nil
	if len(redesPermitidas) > 0 {
		transporte.DialContext = dialContextRestringido(redesPermitidas, net.DefaultResolver, &net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		})
	}
	transporte.TLSClientConfig = &tls.Config{ //nolint:gosec // TLS 1.2 sigue siendo interoperable con RGW soportado.
		MinVersion: tls.VersionTLS12,
		RootCAs:    raices,
	}
	transporte.MaxIdleConns = 64
	transporte.MaxIdleConnsPerHost = 32
	transporte.IdleConnTimeout = 90 * time.Second
	transporte.ResponseHeaderTimeout = 30 * time.Second
	transporte.TLSHandshakeTimeout = 10 * time.Second
	return &http.Client{
		Transport: transporte,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errRedireccionS3NoPermitida
		},
	}, nil
}

var errRedireccionS3NoPermitida = errors.New("vec: redireccion del almacen no permitida")

type resolvedorRed interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

type marcadorRed interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}

func dialContextRestringido(
	redesPermitidas []netip.Prefix,
	resolvedor resolvedorRed,
	marcador marcadorRed,
) func(context.Context, string, string) (net.Conn, error) {
	redes := append([]netip.Prefix(nil), redesPermitidas...)
	return func(ctx context.Context, red, direccion string) (net.Conn, error) {
		if ctx == nil {
			return nil, ErrOperacionS3
		}
		host, puerto, err := net.SplitHostPort(direccion)
		if err != nil || (red != "tcp" && red != "tcp4" && red != "tcp6") {
			return nil, ErrOperacionS3
		}
		direcciones, err := resolverDireccionesPermitidas(ctx, host, redes, resolvedor)
		if err != nil {
			return nil, err
		}
		var ultimoError error
		for _, ip := range direcciones {
			conexion, err := marcador.DialContext(ctx, red, net.JoinHostPort(ip.String(), puerto))
			if err == nil {
				return conexion, nil
			}
			ultimoError = err
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if ultimoError != nil {
			return nil, ErrOperacionS3
		}
		return nil, ErrOperacionS3
	}
}

func resolverDireccionesPermitidas(
	ctx context.Context,
	host string,
	redesPermitidas []netip.Prefix,
	resolvedor resolvedorRed,
) ([]netip.Addr, error) {
	if ctx == nil || resolvedor == nil || len(redesPermitidas) == 0 {
		return nil, ErrOperacionS3
	}
	direcciones := make([]netip.Addr, 0, 4)
	if ip, err := netip.ParseAddr(strings.TrimSuffix(host, ".")); err == nil {
		direcciones = append(direcciones, ip.Unmap())
	} else {
		resueltas, err := resolvedor.LookupNetIP(ctx, "ip", host)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, ErrOperacionS3
		}
		for _, ip := range resueltas {
			direcciones = append(direcciones, ip.Unmap())
		}
	}
	permitidas := make([]netip.Addr, 0, len(direcciones))
	vistas := make(map[netip.Addr]struct{}, len(direcciones))
	for _, ip := range direcciones {
		if !direccionUnicastValida(ip) || !direccionEnRedes(ip, redesPermitidas) {
			continue
		}
		if _, repetida := vistas[ip]; repetida {
			continue
		}
		vistas[ip] = struct{}{}
		permitidas = append(permitidas, ip)
	}
	if len(permitidas) == 0 {
		return nil, ErrOperacionS3
	}
	return permitidas, nil
}

func parsearRedesPermitidas(valor string) ([]netip.Prefix, error) {
	partes := strings.FieldsFunc(valor, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	if len(partes) > 64 {
		return nil, ErrConfiguracionInvalida
	}
	redes := make([]netip.Prefix, 0, len(partes))
	for _, parte := range partes {
		red, err := netip.ParsePrefix(parte)
		if err != nil {
			return nil, ErrConfiguracionInvalida
		}
		redes = append(redes, red)
	}
	if !redesPermitidasValidas(redes) {
		return nil, ErrConfiguracionInvalida
	}
	return redes, nil
}

func redesPermitidasValidas(redes []netip.Prefix) bool {
	vistas := make(map[netip.Prefix]struct{}, len(redes))
	for _, red := range redes {
		if !red.IsValid() || red != red.Masked() || red.Bits() == 0 || red.Addr().Is4In6() ||
			red.Addr().IsMulticast() || red.Addr().IsUnspecified() {
			return false
		}
		if _, repetida := vistas[red]; repetida {
			return false
		}
		vistas[red] = struct{}{}
	}
	return true
}

func direccionEnRedes(ip netip.Addr, redes []netip.Prefix) bool {
	for _, red := range redes {
		if red.Contains(ip) {
			return true
		}
	}
	return false
}

func direccionUnicastValida(ip netip.Addr) bool {
	return ip.IsValid() && !ip.IsMulticast() && !ip.IsUnspecified() &&
		(ip.IsGlobalUnicast() || ip.IsLoopback() || ip.IsLinkLocalUnicast())
}

func identificadorValido(valor string) bool {
	if len(valor) < 2 || len(valor) > 64 || valor != strings.ToLower(valor) || valor[0] < 'a' || valor[0] > 'z' {
		return false
	}
	for _, caracter := range valor[1:] {
		if (caracter < 'a' || caracter > 'z') && (caracter < '0' || caracter > '9') && caracter != '-' && caracter != '_' {
			return false
		}
	}
	return true
}

func bucketValido(valor string) bool {
	if len(valor) < 3 || len(valor) > 63 || valor != strings.ToLower(valor) || strings.Contains(valor, "..") {
		return false
	}
	for _, caracter := range valor {
		if (caracter < 'a' || caracter > 'z') && (caracter < '0' || caracter > '9') && caracter != '.' && caracter != '-' {
			return false
		}
	}
	return valor[0] != '.' && valor[0] != '-' && valor[len(valor)-1] != '.' && valor[len(valor)-1] != '-'
}

func textoTecnicoValido(valor string, maximo int) bool {
	return valor != "" && valor == strings.TrimSpace(valor) && len(valor) <= maximo &&
		!strings.ContainsAny(valor, "\r\n\x00")
}
