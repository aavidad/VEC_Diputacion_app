package httpseguridad

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"net/url"
	"strings"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrCanalProxyNoAutenticado  = errors.New("canal del proxy no autenticado")
	ErrAsercionAusente          = errors.New("asercion de identidad ausente")
	ErrAsercionDemasiadoGrande  = errors.New("asercion de identidad demasiado grande")
	ErrAsercionNoValida         = errors.New("asercion de identidad no valida")
	ErrSesionNoValida           = errors.New("sesion de identidad no valida o revocada")
	ErrVerificadorAusente       = errors.New("verificador de aserciones ausente")
	ErrEvaluadorGarantiaAusente = errors.New("evaluador de garantia ausente")
	ErrRegistroSesionesAusente  = errors.New("registro de sesiones ausente")
	ErrCredencialNoSerializable = errors.New("la credencial de identidad no se puede reconstruir desde una serializacion")
	ErrInicializacionIdentidad  = errors.New("no se pudo inicializar el servicio de identidad")
)

const (
	longitudMaximaAsercionProtegida = 64 * 1024
	longitudMaximaID                = 256
	longitudMaximaReferencia        = 512
)

// MetodoAutenticacion es un catalogo cerrado de mecanismos de identidad. Los
// roles y permisos no proceden de la asercion ni de este catalogo tecnico.
type MetodoAutenticacion string

const (
	MetodoKerberos    MetodoAutenticacion = "kerberos"
	MetodoCertificado MetodoAutenticacion = "certificado"
	MetodoDNIe        MetodoAutenticacion = "dnie"
	MetodoClave       MetodoAutenticacion = "clave"
	MetodoSSO         MetodoAutenticacion = "sso"
)

func (m MetodoAutenticacion) Valido() bool {
	switch m {
	case MetodoKerberos, MetodoCertificado, MetodoDNIe, MetodoClave, MetodoSSO:
		return true
	default:
		return false
	}
}

// TipoCanalProxy es cerrado. Por ahora solo mTLS tiene un constructor seguro;
// el socket Unix permanece cerrado hasta disponer de credenciales de par
// verificadas por plataforma.
type TipoCanalProxy string

const CanalProxyTLSMutuo TipoCanalProxy = "tls_mutuo"

// CanalProxyAutenticado no se puede fabricar mediante literales desde otros
// paquetes. Solo ServicioIdentidad.AutenticarCanalTLSMutuo puede emitirlo y lo
// liga a una instancia concreta del servicio.
type CanalProxyAutenticado struct {
	tipo         TipoCanalProxy
	identidadPar string
	evidenciaRef string
	superficie   Superficie
	instanciaRef [32]byte
	servicio     *ServicioIdentidad
}

func (c CanalProxyAutenticado) Tipo() TipoCanalProxy          { return c.tipo }
func (c CanalProxyAutenticado) IdentidadPar() string          { return c.identidadPar }
func (c CanalProxyAutenticado) Superficie() Superficie        { return c.superficie }
func (c CanalProxyAutenticado) ReferenciaVinculacion() string { return c.evidenciaRef }

func (c CanalProxyAutenticado) validar(servicio *ServicioIdentidad) error {
	if servicio == nil || c.tipo != CanalProxyTLSMutuo || c.superficie != servicio.configuracion.Superficie ||
		c.instanciaRef != servicio.instanciaRef || c.servicio != servicio ||
		strings.TrimSpace(c.identidadPar) == "" || strings.TrimSpace(c.evidenciaRef) == "" {
		return ErrCanalProxyNoAutenticado
	}
	return nil
}

// FactorAutenticacion conserva evidencias verificadas por el adaptador de
// identidad. GrupoCriptograficoRef identifica la credencial raiz real: un
// ticket Kerberos obtenido por PKINIT y el certificado de la misma tarjeta
// deben declarar el mismo grupo y, por tanto, no cuentan como dos factores
// independientes.
type FactorAutenticacion struct {
	Metodo                MetodoAutenticacion
	SujetoVinculadoID     string
	Principal             string
	CredencialRef         string
	EvidenciaRef          string
	GrupoCriptograficoRef string
	VerificadoEn          time.Time
}

// CuentaAcceso separa la persona de su cuenta tecnica. Los identificadores de
// cuenta se canonicalizan como identificadores ASCII sin distinguir caja.
type CuentaAcceso struct {
	ID                string
	SujetoVinculadoID string
	CuentaOrdinariaID string
	Privilegiada      bool
}

// AsercionProxyIdentidad solo puede proceder de VerificadorAsercionProtegida.
// ACRVerificado es la referencia autenticada del proveedor; nunca se acepta un
// nivel de garantia numerico declarado libremente por la asercion.
type AsercionProxyIdentidad struct {
	ID                string
	Emisor            string
	Audiencia         string
	Superficie        Superficie
	SujetoID          string
	Cuenta            CuentaAcceso
	SesionID          string
	CanalVinculadoRef string
	EmitidaEn         time.Time
	NoAntesDe         time.Time
	ExpiraEn          time.Time
	MetodoPrimario    MetodoAutenticacion
	ACRVerificado     string
	Factores          []FactorAutenticacion
}

// VerificadorAsercionProtegida verifica firma, algoritmo, claves, revocacion y
// formato antes de devolver datos tipados.
type VerificadorAsercionProtegida interface {
	Verificar(context.Context, []byte) (AsercionProxyIdentidad, error)
}

// EntradaEvaluacionGarantia contiene exclusivamente valores ya verificados y
// copias defensivas de los factores.
type EntradaEvaluacionGarantia struct {
	ACRVerificado  string
	Emisor         string
	Superficie     Superficie
	SujetoID       string
	CuentaID       string
	MetodoPrimario MetodoAutenticacion
	Factores       []FactorAutenticacion
}

// ResultadoEvaluacionGarantia identifica tanto el nivel calculado como la
// version exacta de la politica que lo calculo.
type ResultadoEvaluacionGarantia struct {
	Garantia       dominiovec.AuthAssurance
	PoliticaRef    string
	HuellaPolitica string
}

// EvaluadorGarantia calcula la garantia desde ACR/AMR y grupos criptograficos
// confiables. Es obligatorio y debe fallar cerrado ante combinaciones que no
// reconozca.
type EvaluadorGarantia interface {
	Evaluar(context.Context, EntradaEvaluacionGarantia) (ResultadoEvaluacionGarantia, error)
}

// AltaSesionAtomica es el dato que recibe el registro de reproduccion. No
// contiene el mensaje protegido.
type AltaSesionAtomica struct {
	AsercionID         string
	SesionID           string
	SujetoID           string
	CuentaID           string
	CuentaOrdinariaID  string
	CuentaPrivilegiada bool
	Superficie         Superficie
	EmitidaEn          time.Time
	ExpiraEn           time.Time
	PoliticaRef        string
	HuellaPolitica     string
}

// ConsultaSesionActiva identifica de forma completa la sesion que se proyecta.
type ConsultaSesionActiva struct {
	AsercionID         string
	SesionID           string
	SujetoID           string
	CuentaID           string
	CuentaOrdinariaID  string
	CuentaPrivilegiada bool
	Superficie         Superficie
	ExpiraEn           time.Time
}

// RegistroSesiones consume el identificador de asercion, comprueba que la
// cuenta de acceso y, en administracion, su cuenta ordinaria esten activas, y
// registra la sesion en una unica operacion atomica. Al proyectar debe volver a
// comprobar ambas cuentas activas y sesion no revocada. Una
// implementacion que separase esas operaciones abriria una carrera TOCTOU.
type RegistroSesiones interface {
	ConsumirAsercionYRegistrar(context.Context, AltaSesionAtomica) error
	ComprobarSesionYCuentaActivas(context.Context, ConsultaSesionActiva) error
}

type Reloj interface {
	Ahora() time.Time
}

type relojSistema struct{}

func (relojSistema) Ahora() time.Time { return time.Now() }

// CredencialProxy conserva la asercion en memoria privada y copia la entrada.
// No existe un getter de los bytes; solo ServicioIdentidad puede entregarlos al
// verificador mediante una copia adicional.
type CredencialProxy struct {
	asercionProtegida []byte
	canal             CanalProxyAutenticado
}

// NuevaCredencialProxy es la unica forma publica de aportar una credencial.
func NuevaCredencialProxy(asercionProtegida []byte, canal CanalProxyAutenticado) (CredencialProxy, error) {
	if len(asercionProtegida) == 0 {
		return CredencialProxy{}, ErrAsercionAusente
	}
	if len(asercionProtegida) > longitudMaximaAsercionProtegida {
		return CredencialProxy{}, ErrAsercionDemasiadoGrande
	}
	return CredencialProxy{
		asercionProtegida: append([]byte(nil), asercionProtegida...),
		canal:             canal,
	}, nil
}

const credencialProxyRedactada = "[CREDENCIAL DE IDENTIDAD CONFIDENCIAL]"

func (CredencialProxy) String() string   { return credencialProxyRedactada }
func (CredencialProxy) GoString() string { return credencialProxyRedactada }

func (CredencialProxy) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(credencialProxyRedactada))
}

func (CredencialProxy) MarshalJSON() ([]byte, error) {
	return []byte(`{"credencial_proxy":"[CONFIDENCIAL]"}`), nil
}

func (CredencialProxy) MarshalText() ([]byte, error) {
	return []byte(credencialProxyRedactada), nil
}

func (CredencialProxy) MarshalBinary() ([]byte, error) {
	return []byte(credencialProxyRedactada), nil
}

func (CredencialProxy) GobEncode() ([]byte, error) {
	return []byte(credencialProxyRedactada), nil
}

func (*CredencialProxy) GobDecode([]byte) error { return ErrCredencialNoSerializable }

func (CredencialProxy) LogValue() slog.Value {
	return slog.StringValue(credencialProxyRedactada)
}

// IdentidadSesion es un vale opaco e inmutable. Solo puede proyectarlo la
// misma instancia y politica de ServicioIdentidad que lo emitio.
type IdentidadSesion struct {
	estado              estadoIdentidadSesion
	instanciaRef        [32]byte
	huellaConfiguracion [32]byte
	servicio            *ServicioIdentidad
}

const identidadSesionRedactada = "[IDENTIDAD DE SESION VALIDADA]"

func (IdentidadSesion) String() string   { return identidadSesionRedactada }
func (IdentidadSesion) GoString() string { return identidadSesionRedactada }
func (IdentidadSesion) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(identidadSesionRedactada))
}

type estadoIdentidadSesion struct {
	asercionID          string
	emisor              string
	audiencia           string
	sujetoID            string
	cuenta              CuentaAcceso
	sesionID            string
	superficie          Superficie
	canalVinculadoRef   string
	emitidaEn           time.Time
	noAntesDe           time.Time
	expiraEn            time.Time
	metodoPrimario      MetodoAutenticacion
	acrVerificado       string
	garantia            dominiovec.AuthAssurance
	politicaGarantiaRef string
	huellaPolitica      string
	factores            []FactorAutenticacion
}

// ResumenFactorAuditoria no contiene secretos ni concede autorizacion.
type ResumenFactorAuditoria struct {
	Metodo                MetodoAutenticacion
	EvidenciaRef          string
	GrupoCriptograficoRef string
	VerificadoEn          time.Time
}

// ContextoAuditoriaAutenticada conserva la identidad humana, la cuenta, la
// sesion, la superficie y la politica verificadas. Deliberadamente no contiene
// roles, permisos ni atributos de autorizacion.
type ContextoAuditoriaAutenticada struct {
	asercionID          string
	emisor              string
	audiencia           string
	sujetoID            string
	cuentaID            string
	cuentaOrdinariaID   string
	cuentaPrivilegiada  bool
	sesionID            string
	superficie          Superficie
	metodoPrimario      MetodoAutenticacion
	garantia            dominiovec.AuthAssurance
	emitidaEn           time.Time
	noAntesDe           time.Time
	expiraEn            time.Time
	politicaGarantiaRef string
	huellaPolitica      string
	huellaConfiguracion string
	canalVinculadoRef   string
	factores            []ResumenFactorAuditoria
}

func (c ContextoAuditoriaAutenticada) AsercionID() string                  { return c.asercionID }
func (c ContextoAuditoriaAutenticada) Emisor() string                      { return c.emisor }
func (c ContextoAuditoriaAutenticada) Audiencia() string                   { return c.audiencia }
func (c ContextoAuditoriaAutenticada) SujetoID() string                    { return c.sujetoID }
func (c ContextoAuditoriaAutenticada) CuentaID() string                    { return c.cuentaID }
func (c ContextoAuditoriaAutenticada) CuentaOrdinariaID() string           { return c.cuentaOrdinariaID }
func (c ContextoAuditoriaAutenticada) CuentaPrivilegiada() bool            { return c.cuentaPrivilegiada }
func (c ContextoAuditoriaAutenticada) SesionID() string                    { return c.sesionID }
func (c ContextoAuditoriaAutenticada) Superficie() Superficie              { return c.superficie }
func (c ContextoAuditoriaAutenticada) MetodoPrimario() MetodoAutenticacion { return c.metodoPrimario }
func (c ContextoAuditoriaAutenticada) Garantia() dominiovec.AuthAssurance  { return c.garantia }
func (c ContextoAuditoriaAutenticada) EmitidaEn() time.Time                { return c.emitidaEn }
func (c ContextoAuditoriaAutenticada) NoAntesDe() time.Time                { return c.noAntesDe }
func (c ContextoAuditoriaAutenticada) ExpiraEn() time.Time                 { return c.expiraEn }
func (c ContextoAuditoriaAutenticada) PoliticaGarantiaRef() string         { return c.politicaGarantiaRef }
func (c ContextoAuditoriaAutenticada) HuellaPolitica() string              { return c.huellaPolitica }
func (c ContextoAuditoriaAutenticada) HuellaConfiguracion() string         { return c.huellaConfiguracion }
func (c ContextoAuditoriaAutenticada) CanalVinculadoRef() string           { return c.canalVinculadoRef }
func (c ContextoAuditoriaAutenticada) Factores() []ResumenFactorAuditoria {
	return append([]ResumenFactorAuditoria(nil), c.factores...)
}

// ServicioIdentidad es la unica autoridad que crea y proyecta identidades.
type ServicioIdentidad struct {
	configuracion       ConfiguracionSuperficie
	verificador         VerificadorAsercionProtegida
	evaluadorGarantia   EvaluadorGarantia
	registroSesiones    RegistroSesiones
	reloj               Reloj
	instanciaRef        [32]byte
	huellaConfiguracion [32]byte
}

func NuevoServicioIdentidad(
	configuracion ConfiguracionSuperficie,
	verificador VerificadorAsercionProtegida,
	evaluadorGarantia EvaluadorGarantia,
	registroSesiones RegistroSesiones,
	reloj Reloj,
) (*ServicioIdentidad, error) {
	if err := configuracion.Validar(); err != nil {
		return nil, err
	}
	if configuracion.Superficie == SuperficiePublicaAnonima {
		return nil, fmt.Errorf("%w: la superficie anonima no resuelve sesiones", ErrConfiguracionSuperficie)
	}
	if verificador == nil {
		return nil, ErrVerificadorAusente
	}
	if evaluadorGarantia == nil {
		return nil, ErrEvaluadorGarantiaAusente
	}
	if registroSesiones == nil {
		return nil, ErrRegistroSesionesAusente
	}
	if reloj == nil {
		reloj = relojSistema{}
	}
	configuracion = copiarYNormalizarConfiguracion(configuracion)
	contenidoConfiguracion, err := json.Marshal(configuracion)
	if err != nil {
		return nil, ErrInicializacionIdentidad
	}
	servicio := &ServicioIdentidad{
		configuracion:       configuracion,
		verificador:         verificador,
		evaluadorGarantia:   evaluadorGarantia,
		registroSesiones:    registroSesiones,
		reloj:               reloj,
		huellaConfiguracion: sha256.Sum256(contenidoConfiguracion),
	}
	if _, err := rand.Read(servicio.instanciaRef[:]); err != nil {
		return nil, ErrInicializacionIdentidad
	}
	return servicio, nil
}

func copiarYNormalizarConfiguracion(c ConfiguracionSuperficie) ConfiguracionSuperficie {
	c.RedesPermitidas = append([]string(nil), c.RedesPermitidas...)
	c.MetodosAdmitidos = append([]MetodoAutenticacion(nil), c.MetodosAdmitidos...)
	c.FactoresRequeridos = append([]MetodoAutenticacion(nil), c.FactoresRequeridos...)
	c.HuellasProxyTLSPermitidas = append([]string(nil), c.HuellasProxyTLSPermitidas...)
	for indice, huella := range c.HuellasProxyTLSPermitidas {
		c.HuellasProxyTLSPermitidas[indice], _ = normalizarHuellaCertificado(huella)
	}
	c.IdentidadesSANProxyPermitidas = append([]string(nil), c.IdentidadesSANProxyPermitidas...)
	for indice, identidad := range c.IdentidadesSANProxyPermitidas {
		c.IdentidadesSANProxyPermitidas[indice], _ = normalizarIdentidadSAN(identidad)
	}
	return c
}

// AutenticarCanalTLSMutuo crea un canal solo desde un handshake mTLS
// terminado y una cadena que la biblioteca TLS ya haya verificado.
func (s *ServicioIdentidad) AutenticarCanalTLSMutuo(estado tls.ConnectionState) (CanalProxyAutenticado, error) {
	if s == nil || !estado.HandshakeComplete ||
		(estado.Version != tls.VersionTLS12 && estado.Version != tls.VersionTLS13) ||
		!suiteCifradoTLSAdmitida(estado.CipherSuite) ||
		len(estado.VerifiedChains) == 0 || len(estado.VerifiedChains[0]) == 0 ||
		len(estado.PeerCertificates) == 0 {
		return CanalProxyAutenticado{}, ErrCanalProxyNoAutenticado
	}
	certificado := estado.VerifiedChains[0][0]
	if certificado == nil || len(certificado.Raw) == 0 || estado.PeerCertificates[0] == nil ||
		!bytes.Equal(certificado.Raw, estado.PeerCertificates[0].Raw) || !certificadoAdmiteAutenticacionCliente(certificado) {
		return CanalProxyAutenticado{}, ErrCanalProxyNoAutenticado
	}
	sumaCertificado := sha256.Sum256(certificado.Raw)
	huella := "sha256:" + hex.EncodeToString(sumaCertificado[:])
	coincideHuella := len(s.configuracion.HuellasProxyTLSPermitidas) == 0 ||
		contieneCadena(s.configuracion.HuellasProxyTLSPermitidas, huella)
	identidadSAN, coincideSAN := identidadSANPermitida(certificado, s.configuracion.IdentidadesSANProxyPermitidas)
	if len(s.configuracion.IdentidadesSANProxyPermitidas) == 0 {
		coincideSAN = true
	}
	if !coincideHuella || !coincideSAN {
		return CanalProxyAutenticado{}, ErrCanalProxyNoAutenticado
	}
	materialExportado, err := exportarMaterialCanalTLS(
		estado,
		"VEC-Diputacion-Canal-Identidad-v1",
		[]byte(s.configuracion.Superficie),
		sha256.Size,
	)
	if err != nil || len(materialExportado) != sha256.Size {
		return CanalProxyAutenticado{}, ErrCanalProxyNoAutenticado
	}
	huellaConexion := sha256.Sum256(materialExportado)
	for indice := range materialExportado {
		materialExportado[indice] = 0
	}
	identidadPar := huella
	if identidadSAN != "" {
		identidadPar = identidadSAN
	}
	return CanalProxyAutenticado{
		tipo:         CanalProxyTLSMutuo,
		identidadPar: identidadPar,
		evidenciaRef: "tls-exportador:sha256:" + hex.EncodeToString(huellaConexion[:]),
		superficie:   s.configuracion.Superficie,
		instanciaRef: s.instanciaRef,
		servicio:     s,
	}, nil
}

func exportarMaterialCanalTLS(
	estado tls.ConnectionState,
	etiqueta string,
	contexto []byte,
	longitud int,
) (material []byte, err error) {
	// ConnectionState contiene internamente la funcion exportadora solo cuando
	// procede de una conexion real. Una estructura fabricada puede provocar un
	// panic en la biblioteca estandar; se convierte siempre en denegacion.
	defer func() {
		if recover() != nil {
			material = nil
			err = ErrCanalProxyNoAutenticado
		}
	}()
	return estado.ExportKeyingMaterial(etiqueta, contexto, longitud)
}

func suiteCifradoTLSAdmitida(suite uint16) bool {
	switch suite {
	case tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256:
		return true
	default:
		return false
	}
}

func certificadoAdmiteAutenticacionCliente(certificado *x509.Certificate) bool {
	if len(certificado.ExtKeyUsage) == 0 {
		return true
	}
	for _, uso := range certificado.ExtKeyUsage {
		if uso == x509.ExtKeyUsageClientAuth || uso == x509.ExtKeyUsageAny {
			return true
		}
	}
	return false
}

func identidadSANPermitida(certificado *x509.Certificate, permitidas []string) (string, bool) {
	if len(permitidas) == 0 {
		return "", true
	}
	candidatas := make([]string, 0, len(certificado.DNSNames)+len(certificado.EmailAddresses)+len(certificado.IPAddresses)+len(certificado.URIs))
	for _, nombre := range certificado.DNSNames {
		candidatas = append(candidatas, "dns:"+strings.ToLower(strings.TrimSpace(nombre)))
	}
	for _, correo := range certificado.EmailAddresses {
		candidatas = append(candidatas, "email:"+strings.ToLower(strings.TrimSpace(correo)))
	}
	for _, direccion := range certificado.IPAddresses {
		candidatas = append(candidatas, "ip:"+direccion.String())
	}
	for _, uri := range certificado.URIs {
		candidatas = append(candidatas, "uri:"+uri.String())
	}
	for _, permitida := range permitidas {
		if contieneCadena(candidatas, permitida) {
			return permitida, true
		}
	}
	return "", false
}

// Resolver verifica canal, asercion, factores y garantia antes de consumir el
// identificador de asercion y registrar la sesion atomicamente.
func (s *ServicioIdentidad) Resolver(ctx context.Context, credencial CredencialProxy) (IdentidadSesion, error) {
	if s == nil || s.verificador == nil || s.evaluadorGarantia == nil || s.registroSesiones == nil || s.reloj == nil {
		return IdentidadSesion{}, ErrVerificadorAusente
	}
	if err := validarContexto(ctx); err != nil {
		return IdentidadSesion{}, err
	}
	if err := credencial.canal.validar(s); err != nil {
		return IdentidadSesion{}, err
	}
	if len(credencial.asercionProtegida) == 0 {
		return IdentidadSesion{}, ErrAsercionAusente
	}
	if len(credencial.asercionProtegida) > longitudMaximaAsercionProtegida {
		return IdentidadSesion{}, ErrAsercionDemasiadoGrande
	}
	asercion, err := s.verificador.Verificar(ctx, append([]byte(nil), credencial.asercionProtegida...))
	if err != nil {
		return IdentidadSesion{}, ErrAsercionNoValida
	}
	ahora := s.reloj.Ahora()
	if ahora.IsZero() {
		return IdentidadSesion{}, ErrAsercionNoValida
	}
	estado, err := normalizarYValidarAsercion(asercion, s.configuracion, credencial.canal.evidenciaRef, ahora)
	if err != nil {
		return IdentidadSesion{}, ErrAsercionNoValida
	}
	resultado, err := s.evaluarGarantia(ctx, estado)
	if err != nil {
		return IdentidadSesion{}, ErrAsercionNoValida
	}
	if err := ctx.Err(); err != nil {
		return IdentidadSesion{}, err
	}
	estado.garantia = resultado.Garantia
	estado.politicaGarantiaRef = resultado.PoliticaRef
	estado.huellaPolitica = resultado.HuellaPolitica
	if err := s.registroSesiones.ConsumirAsercionYRegistrar(ctx, altaSesion(estado)); err != nil {
		if ctx.Err() != nil {
			return IdentidadSesion{}, ctx.Err()
		}
		return IdentidadSesion{}, ErrSesionNoValida
	}
	if err := ctx.Err(); err != nil {
		return IdentidadSesion{}, err
	}
	ahoraFinal := s.reloj.Ahora()
	if ahoraFinal.IsZero() || !ahoraFinal.Before(estado.expiraEn) {
		return IdentidadSesion{}, ErrSesionNoValida
	}
	return IdentidadSesion{
		estado:              copiarEstado(estado),
		instanciaRef:        s.instanciaRef,
		huellaConfiguracion: s.huellaConfiguracion,
		servicio:            s,
	}, nil
}

// ProyectarPrincipal es la unica conversion al principal del nucleo. Revalida
// politica, vigencia, version de garantia, sesion y cuenta activa.
func (s *ServicioIdentidad) ProyectarPrincipal(
	ctx context.Context,
	identidad IdentidadSesion,
) (dominiovec.Principal, ContextoAuditoriaAutenticada, error) {
	if s == nil || s.evaluadorGarantia == nil || s.registroSesiones == nil || s.reloj == nil ||
		identidad.servicio != s ||
		identidad.instanciaRef != s.instanciaRef || identidad.huellaConfiguracion != s.huellaConfiguracion {
		return dominiovec.Principal{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	if err := validarContexto(ctx); err != nil {
		return dominiovec.Principal{}, ContextoAuditoriaAutenticada{}, err
	}
	ahora := s.reloj.Ahora()
	if ahora.IsZero() || validarEstadoSesion(identidad.estado, s.configuracion, ahora) != nil {
		return dominiovec.Principal{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	resultado, err := s.evaluarGarantia(ctx, identidad.estado)
	if err != nil || resultado.Garantia != identidad.estado.garantia ||
		resultado.PoliticaRef != identidad.estado.politicaGarantiaRef ||
		resultado.HuellaPolitica != identidad.estado.huellaPolitica {
		return dominiovec.Principal{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	if err := ctx.Err(); err != nil {
		return dominiovec.Principal{}, ContextoAuditoriaAutenticada{}, err
	}
	consulta := ConsultaSesionActiva{
		AsercionID:         identidad.estado.asercionID,
		SesionID:           identidad.estado.sesionID,
		SujetoID:           identidad.estado.sujetoID,
		CuentaID:           identidad.estado.cuenta.ID,
		CuentaOrdinariaID:  identidad.estado.cuenta.CuentaOrdinariaID,
		CuentaPrivilegiada: identidad.estado.cuenta.Privilegiada,
		Superficie:         identidad.estado.superficie,
		ExpiraEn:           identidad.estado.expiraEn,
	}
	if err := s.registroSesiones.ComprobarSesionYCuentaActivas(ctx, consulta); err != nil {
		if ctx.Err() != nil {
			return dominiovec.Principal{}, ContextoAuditoriaAutenticada{}, ctx.Err()
		}
		return dominiovec.Principal{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	if err := ctx.Err(); err != nil {
		return dominiovec.Principal{}, ContextoAuditoriaAutenticada{}, err
	}
	metodo, valido := metodoAutenticacionDominio(identidad.estado.metodoPrimario)
	if !valido {
		return dominiovec.Principal{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	principal := dominiovec.Principal{
		ID:            identidad.estado.cuenta.ID,
		AuthMethod:    metodo,
		AuthAssurance: identidad.estado.garantia,
	}
	if err := principal.Validate(); err != nil {
		return dominiovec.Principal{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	ahoraFinal := s.reloj.Ahora()
	if ahoraFinal.IsZero() || !ahoraFinal.Before(identidad.estado.expiraEn) {
		return dominiovec.Principal{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	return principal, nuevoContextoAuditoria(identidad.estado, s.huellaConfiguracion), nil
}

func validarContexto(ctx context.Context) error {
	if ctx == nil {
		return ErrAsercionNoValida
	}
	return ctx.Err()
}

func normalizarYValidarAsercion(
	a AsercionProxyIdentidad,
	c ConfiguracionSuperficie,
	canalVinculadoRef string,
	ahora time.Time,
) (estadoIdentidadSesion, error) {
	var estado estadoIdentidadSesion
	var err error
	if a.Emisor != c.EmisorIdentidad || a.Audiencia != c.Audiencia || a.Superficie != c.Superficie ||
		!a.MetodoPrimario.Valido() {
		return estado, ErrAsercionNoValida
	}
	if estado.asercionID, err = canonicalizarID(a.ID, longitudMaximaID, false); err != nil {
		return estado, err
	}
	if estado.sujetoID, err = canonicalizarID(a.SujetoID, longitudMaximaID, false); err != nil {
		return estado, err
	}
	if estado.sesionID, err = canonicalizarID(a.SesionID, longitudMaximaID, false); err != nil {
		return estado, err
	}
	if estado.canalVinculadoRef, err = canonicalizarID(a.CanalVinculadoRef, longitudMaximaReferencia, true); err != nil ||
		estado.canalVinculadoRef != canalVinculadoRef {
		return estado, ErrAsercionNoValida
	}
	if estado.acrVerificado, err = canonicalizarID(a.ACRVerificado, longitudMaximaReferencia, false); err != nil {
		return estado, err
	}
	if estado.cuenta, err = normalizarCuenta(a.Cuenta, estado.sujetoID); err != nil {
		return estado, err
	}
	factores, err := normalizarYValidarFactores(a.Factores, c, estado.sujetoID, a.EmitidaEn, a.ExpiraEn, ahora)
	if err != nil || !factoresContienenMetodo(factores, a.MetodoPrimario) {
		return estado, ErrAsercionNoValida
	}
	estado.emisor = a.Emisor
	estado.audiencia = a.Audiencia
	estado.superficie = a.Superficie
	estado.emitidaEn = a.EmitidaEn
	estado.noAntesDe = a.NoAntesDe
	estado.expiraEn = a.ExpiraEn
	estado.metodoPrimario = a.MetodoPrimario
	estado.factores = factores
	if err := validarTiempos(estado, c, ahora); err != nil || validarPoliticaCuenta(estado.cuenta, c) != nil {
		return estadoIdentidadSesion{}, ErrAsercionNoValida
	}
	return estado, nil
}

func validarEstadoSesion(estado estadoIdentidadSesion, c ConfiguracionSuperficie, ahora time.Time) error {
	if estado.emisor != c.EmisorIdentidad || estado.audiencia != c.Audiencia || estado.superficie != c.Superficie ||
		!estado.metodoPrimario.Valido() || !estado.garantia.Valida() ||
		!dominiovec.CumpleGarantiaAutenticacion(estado.garantia, c.GarantiaMinima) ||
		strings.TrimSpace(estado.politicaGarantiaRef) == "" || validarHuellaPolitica(estado.huellaPolitica) != nil {
		return ErrSesionNoValida
	}
	for _, valor := range []struct {
		texto      string
		maximo     int
		minusculas bool
	}{
		{estado.asercionID, longitudMaximaID, false},
		{estado.sujetoID, longitudMaximaID, false},
		{estado.sesionID, longitudMaximaID, false},
		{estado.canalVinculadoRef, longitudMaximaReferencia, true},
		{estado.acrVerificado, longitudMaximaReferencia, false},
	} {
		canonico, err := canonicalizarID(valor.texto, valor.maximo, valor.minusculas)
		if err != nil || canonico != valor.texto {
			return ErrSesionNoValida
		}
	}
	cuenta, err := normalizarCuenta(estado.cuenta, estado.sujetoID)
	if err != nil || cuenta != estado.cuenta || validarPoliticaCuenta(cuenta, c) != nil ||
		validarTiempos(estado, c, ahora) != nil {
		return ErrSesionNoValida
	}
	factores, err := normalizarYValidarFactores(estado.factores, c, estado.sujetoID, estado.emitidaEn, estado.expiraEn, ahora)
	if err != nil || !factoresContienenMetodo(factores, estado.metodoPrimario) || !factoresIguales(factores, estado.factores) {
		return ErrSesionNoValida
	}
	return nil
}

func validarTiempos(estado estadoIdentidadSesion, c ConfiguracionSuperficie, ahora time.Time) error {
	if estado.emitidaEn.IsZero() || estado.noAntesDe.IsZero() || estado.expiraEn.IsZero() ||
		!estado.expiraEn.After(estado.emitidaEn) ||
		estado.noAntesDe.Before(estado.emitidaEn.Add(-c.ToleranciaReloj)) ||
		estado.noAntesDe.After(estado.expiraEn) ||
		estado.expiraEn.Sub(estado.emitidaEn) > c.DuracionMaximaAsercion ||
		estado.emitidaEn.After(ahora.Add(c.ToleranciaReloj)) ||
		estado.noAntesDe.After(ahora.Add(c.ToleranciaReloj)) || !ahora.Before(estado.expiraEn) {
		return ErrAsercionNoValida
	}
	return nil
}

func normalizarCuenta(cuenta CuentaAcceso, sujetoID string) (CuentaAcceso, error) {
	id, err := canonicalizarID(cuenta.ID, longitudMaximaID, true)
	if err != nil {
		return CuentaAcceso{}, err
	}
	sujeto, err := canonicalizarID(cuenta.SujetoVinculadoID, longitudMaximaID, false)
	if err != nil || sujeto != sujetoID {
		return CuentaAcceso{}, ErrAsercionNoValida
	}
	ordinaria := ""
	if strings.TrimSpace(cuenta.CuentaOrdinariaID) != "" {
		ordinaria, err = canonicalizarID(cuenta.CuentaOrdinariaID, longitudMaximaID, true)
		if err != nil {
			return CuentaAcceso{}, err
		}
	}
	return CuentaAcceso{ID: id, SujetoVinculadoID: sujeto, CuentaOrdinariaID: ordinaria, Privilegiada: cuenta.Privilegiada}, nil
}

func validarPoliticaCuenta(cuenta CuentaAcceso, c ConfiguracionSuperficie) error {
	if c.RequiereCuentaPrivilegiada {
		if !cuenta.Privilegiada || cuenta.CuentaOrdinariaID == "" || cuenta.CuentaOrdinariaID == cuenta.ID {
			return ErrAsercionNoValida
		}
		return nil
	}
	if cuenta.Privilegiada || cuenta.CuentaOrdinariaID != "" {
		return ErrAsercionNoValida
	}
	return nil
}

func normalizarYValidarFactores(
	factores []FactorAutenticacion,
	c ConfiguracionSuperficie,
	sujetoID string,
	emitidaEn time.Time,
	expiraEn time.Time,
	ahora time.Time,
) ([]FactorAutenticacion, error) {
	porMetodo := make(map[MetodoAutenticacion]struct{}, len(factores))
	evidencias := make(map[string]struct{}, len(factores))
	grupos := make(map[string]struct{}, len(factores))
	normalizados := make([]FactorAutenticacion, 0, len(factores))
	for _, factor := range factores {
		normalizado, err := normalizarFactor(factor, sujetoID)
		if err != nil || normalizado.VerificadoEn.Before(emitidaEn.Add(-c.ToleranciaReloj)) ||
			normalizado.VerificadoEn.After(emitidaEn.Add(c.ToleranciaReloj)) ||
			normalizado.VerificadoEn.After(expiraEn) || normalizado.VerificadoEn.After(ahora.Add(c.ToleranciaReloj)) ||
			!contieneMetodo(c.MetodosAdmitidos, normalizado.Metodo) {
			return nil, ErrAsercionNoValida
		}
		if _, existe := porMetodo[normalizado.Metodo]; existe {
			return nil, ErrAsercionNoValida
		}
		if _, existe := evidencias[normalizado.EvidenciaRef]; existe {
			return nil, ErrAsercionNoValida
		}
		porMetodo[normalizado.Metodo] = struct{}{}
		evidencias[normalizado.EvidenciaRef] = struct{}{}
		grupos[normalizado.GrupoCriptograficoRef] = struct{}{}
		normalizados = append(normalizados, normalizado)
	}
	for _, requerido := range c.FactoresRequeridos {
		if _, existe := porMetodo[requerido]; !existe {
			return nil, ErrAsercionNoValida
		}
	}
	if len(porMetodo) < c.MinimoFactoresVerificados || len(grupos) < c.MinimoGruposCriptograficosDistintos {
		return nil, ErrAsercionNoValida
	}
	return normalizados, nil
}

func normalizarFactor(factor FactorAutenticacion, sujetoID string) (FactorAutenticacion, error) {
	if !factor.Metodo.Valido() || factor.VerificadoEn.IsZero() {
		return FactorAutenticacion{}, ErrAsercionNoValida
	}
	sujeto, err := canonicalizarID(factor.SujetoVinculadoID, longitudMaximaID, false)
	if err != nil || sujeto != sujetoID {
		return FactorAutenticacion{}, ErrAsercionNoValida
	}
	evidencia, err := canonicalizarID(factor.EvidenciaRef, longitudMaximaReferencia, true)
	if err != nil {
		return FactorAutenticacion{}, err
	}
	grupo, err := canonicalizarID(factor.GrupoCriptograficoRef, longitudMaximaReferencia, true)
	if err != nil {
		return FactorAutenticacion{}, err
	}
	principal, err := canonicalizarOpcional(factor.Principal, longitudMaximaReferencia, false)
	if err != nil {
		return FactorAutenticacion{}, err
	}
	credencial, err := canonicalizarOpcional(factor.CredencialRef, longitudMaximaReferencia, true)
	if err != nil {
		return FactorAutenticacion{}, err
	}
	switch factor.Metodo {
	case MetodoKerberos:
		if principal == "" {
			return FactorAutenticacion{}, ErrAsercionNoValida
		}
	case MetodoCertificado, MetodoDNIe:
		if credencial == "" {
			return FactorAutenticacion{}, ErrAsercionNoValida
		}
	}
	return FactorAutenticacion{
		Metodo: factor.Metodo, SujetoVinculadoID: sujeto, Principal: principal,
		CredencialRef: credencial, EvidenciaRef: evidencia,
		GrupoCriptograficoRef: grupo, VerificadoEn: factor.VerificadoEn,
	}, nil
}

func factoresContienenMetodo(factores []FactorAutenticacion, metodo MetodoAutenticacion) bool {
	for _, factor := range factores {
		if factor.Metodo == metodo {
			return true
		}
	}
	return false
}

func factoresIguales(primero, segundo []FactorAutenticacion) bool {
	if len(primero) != len(segundo) {
		return false
	}
	for indice := range primero {
		if primero[indice] != segundo[indice] {
			return false
		}
	}
	return true
}

func (s *ServicioIdentidad) evaluarGarantia(ctx context.Context, estado estadoIdentidadSesion) (ResultadoEvaluacionGarantia, error) {
	entrada := EntradaEvaluacionGarantia{
		ACRVerificado: estado.acrVerificado, Emisor: estado.emisor, Superficie: estado.superficie,
		SujetoID: estado.sujetoID, CuentaID: estado.cuenta.ID, MetodoPrimario: estado.metodoPrimario,
		Factores: append([]FactorAutenticacion(nil), estado.factores...),
	}
	resultado, err := s.evaluadorGarantia.Evaluar(ctx, entrada)
	if err != nil {
		return ResultadoEvaluacionGarantia{}, ErrAsercionNoValida
	}
	if !resultado.Garantia.Valida() ||
		!dominiovec.CumpleGarantiaAutenticacion(resultado.Garantia, s.configuracion.GarantiaMinima) {
		return ResultadoEvaluacionGarantia{}, ErrAsercionNoValida
	}
	politica, err := canonicalizarID(resultado.PoliticaRef, longitudMaximaReferencia, true)
	if err != nil {
		return ResultadoEvaluacionGarantia{}, ErrAsercionNoValida
	}
	huella, err := normalizarHuellaCertificado(resultado.HuellaPolitica)
	if err != nil {
		return ResultadoEvaluacionGarantia{}, ErrAsercionNoValida
	}
	return ResultadoEvaluacionGarantia{Garantia: resultado.Garantia, PoliticaRef: politica, HuellaPolitica: huella}, nil
}

func altaSesion(estado estadoIdentidadSesion) AltaSesionAtomica {
	return AltaSesionAtomica{
		AsercionID: estado.asercionID, SesionID: estado.sesionID, SujetoID: estado.sujetoID,
		CuentaID: estado.cuenta.ID, CuentaOrdinariaID: estado.cuenta.CuentaOrdinariaID,
		CuentaPrivilegiada: estado.cuenta.Privilegiada, Superficie: estado.superficie, EmitidaEn: estado.emitidaEn,
		ExpiraEn: estado.expiraEn, PoliticaRef: estado.politicaGarantiaRef, HuellaPolitica: estado.huellaPolitica,
	}
}

func copiarEstado(estado estadoIdentidadSesion) estadoIdentidadSesion {
	estado.factores = append([]FactorAutenticacion(nil), estado.factores...)
	return estado
}

func nuevoContextoAuditoria(estado estadoIdentidadSesion, huellaConfiguracion [32]byte) ContextoAuditoriaAutenticada {
	factores := make([]ResumenFactorAuditoria, 0, len(estado.factores))
	for _, factor := range estado.factores {
		factores = append(factores, ResumenFactorAuditoria{
			Metodo: factor.Metodo, EvidenciaRef: factor.EvidenciaRef,
			GrupoCriptograficoRef: factor.GrupoCriptograficoRef, VerificadoEn: factor.VerificadoEn,
		})
	}
	return ContextoAuditoriaAutenticada{
		asercionID: estado.asercionID, emisor: estado.emisor, audiencia: estado.audiencia,
		sujetoID: estado.sujetoID, cuentaID: estado.cuenta.ID,
		cuentaOrdinariaID: estado.cuenta.CuentaOrdinariaID, cuentaPrivilegiada: estado.cuenta.Privilegiada,
		sesionID: estado.sesionID, superficie: estado.superficie, metodoPrimario: estado.metodoPrimario,
		garantia: estado.garantia, emitidaEn: estado.emitidaEn, noAntesDe: estado.noAntesDe, expiraEn: estado.expiraEn,
		politicaGarantiaRef: estado.politicaGarantiaRef, huellaPolitica: estado.huellaPolitica,
		huellaConfiguracion: "sha256:" + hex.EncodeToString(huellaConfiguracion[:]),
		canalVinculadoRef:   estado.canalVinculadoRef, factores: factores,
	}
}

func metodoAutenticacionDominio(metodo MetodoAutenticacion) (dominiovec.AuthMethod, bool) {
	switch metodo {
	case MetodoKerberos:
		return dominiovec.AuthMethodKerberos, true
	case MetodoCertificado:
		return dominiovec.AuthMethodCertificate, true
	case MetodoDNIe:
		return dominiovec.AuthMethodDNIe, true
	case MetodoClave:
		return dominiovec.AuthMethodClave, true
	case MetodoSSO:
		return dominiovec.AuthMethodSSO, true
	default:
		return "", false
	}
}

func validarConfianzaProxyTLS(c ConfiguracionSuperficie) error {
	if len(c.HuellasProxyTLSPermitidas) == 0 && len(c.IdentidadesSANProxyPermitidas) == 0 {
		return fmt.Errorf("%w: se requiere una identidad mTLS explicita para el proxy", ErrConfiguracionSuperficie)
	}
	vistas := make(map[string]struct{}, len(c.HuellasProxyTLSPermitidas)+len(c.IdentidadesSANProxyPermitidas))
	for _, valor := range c.HuellasProxyTLSPermitidas {
		normalizado, err := normalizarHuellaCertificado(valor)
		if err != nil {
			return fmt.Errorf("%w: huella TLS no valida", ErrConfiguracionSuperficie)
		}
		if _, existe := vistas["huella:"+normalizado]; existe {
			return fmt.Errorf("%w: huella TLS duplicada", ErrConfiguracionSuperficie)
		}
		vistas["huella:"+normalizado] = struct{}{}
	}
	for _, valor := range c.IdentidadesSANProxyPermitidas {
		normalizado, err := normalizarIdentidadSAN(valor)
		if err != nil {
			return fmt.Errorf("%w: identidad SAN no valida", ErrConfiguracionSuperficie)
		}
		if _, existe := vistas["san:"+normalizado]; existe {
			return fmt.Errorf("%w: identidad SAN duplicada", ErrConfiguracionSuperficie)
		}
		vistas["san:"+normalizado] = struct{}{}
	}
	return nil
}

func validarAudienciaConfigurada(valor string) error {
	canonico, err := canonicalizarID(valor, longitudMaximaID, false)
	if err != nil || canonico != valor {
		return ErrConfiguracionSuperficie
	}
	return nil
}

func validarEmisorConfigurado(valor string) error {
	canonico, err := canonicalizarID(valor, longitudMaximaReferencia, false)
	if err != nil || canonico != valor {
		return ErrConfiguracionSuperficie
	}
	emisor, err := url.Parse(canonico)
	if err != nil || !strings.EqualFold(emisor.Scheme, "https") || emisor.Host == "" ||
		emisor.User != nil || emisor.Fragment != "" || emisor.RawQuery != "" {
		return ErrConfiguracionSuperficie
	}
	return nil
}

func normalizarHuellaCertificado(valor string) (string, error) {
	valor = strings.ToLower(strings.TrimSpace(valor))
	if !strings.HasPrefix(valor, "sha256:") {
		return "", ErrAsercionNoValida
	}
	contenido := strings.TrimPrefix(valor, "sha256:")
	bytesHuella, err := hex.DecodeString(contenido)
	if err != nil || len(bytesHuella) != sha256.Size || hex.EncodeToString(bytesHuella) != contenido {
		return "", ErrAsercionNoValida
	}
	return valor, nil
}

func validarHuellaPolitica(valor string) error {
	_, err := normalizarHuellaCertificado(valor)
	return err
}

func normalizarIdentidadSAN(valor string) (string, error) {
	valor = strings.TrimSpace(valor)
	partes := strings.SplitN(valor, ":", 2)
	if len(partes) != 2 || partes[1] == "" {
		return "", ErrAsercionNoValida
	}
	tipo := strings.ToLower(partes[0])
	contenido, err := canonicalizarID(partes[1], longitudMaximaReferencia, false)
	if err != nil {
		return "", err
	}
	switch tipo {
	case "dns", "email":
		contenido = strings.ToLower(contenido)
	case "ip":
		direccion, err := netip.ParseAddr(contenido)
		if err != nil {
			return "", ErrAsercionNoValida
		}
		contenido = direccion.Unmap().String()
	case "uri":
		direccion, err := url.Parse(contenido)
		if err != nil || !direccion.IsAbs() || direccion.Host == "" {
			return "", ErrAsercionNoValida
		}
	default:
		return "", ErrAsercionNoValida
	}
	return tipo + ":" + contenido, nil
}

func canonicalizarID(valor string, maximo int, minusculas bool) (string, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" || len(valor) > maximo {
		return "", ErrAsercionNoValida
	}
	for indice := 0; indice < len(valor); indice++ {
		if valor[indice] <= 0x20 || valor[indice] >= 0x7f {
			return "", ErrAsercionNoValida
		}
	}
	if minusculas {
		valor = strings.ToLower(valor)
	}
	return valor, nil
}

func canonicalizarOpcional(valor string, maximo int, minusculas bool) (string, error) {
	if strings.TrimSpace(valor) == "" {
		return "", nil
	}
	return canonicalizarID(valor, maximo, minusculas)
}

func contieneCadena(valores []string, buscado string) bool {
	for _, valor := range valores {
		if valor == buscado {
			return true
		}
	}
	return false
}
