package httpseguridad

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	CabeceraAsercionPasarela = "X-VEC-Asercion-Pasarela"
	versionAsercionPasarela  = "vec-pasarela-v1"
	limiteCuerpoPasarela     = 1 << 20
)

var ErrAsercionPasarela = errors.New("asercion de pasarela no valida")

type claveContextoPeticionPasarela struct{}

// VinculoPeticionPasarela representa los bytes que vera el servidor despues
// del proxy. La huella corresponde al cuerpo transferido al caso de uso.
type VinculoPeticionPasarela struct {
	Metodo       string `json:"metodo"`
	Ruta         string `json:"ruta"`
	CuerpoSHA256 string `json:"cuerpo_sha256"`
}

// NuevoVinculoPeticionPasarela se usa en la pasarela sobre la peticion que
// reenviara. No acepta rutas normalizadas que puedan cambiar en el destino.
func NuevoVinculoPeticionPasarela(metodo, ruta string, cuerpo []byte) (VinculoPeticionPasarela, error) {
	if !metodoPasarelaValido(metodo) || !rutaPasarelaValida(ruta) || len(cuerpo) > limiteCuerpoPasarela {
		return VinculoPeticionPasarela{}, ErrAsercionPasarela
	}
	huella := sha256.Sum256(cuerpo)
	return VinculoPeticionPasarela{Metodo: metodo, Ruta: ruta, CuerpoSHA256: hex.EncodeToString(huella[:])}, nil
}

// PrepararPeticionAsercionPasarela fija una copia privada del vinculo en el
// contexto. Consume y repone el cuerpo una sola vez, antes de autenticar y de
// ejecutar la ruta. El limite nunca puede superar 1 MiB, aunque la ruta
// admita cuerpos mayores: ese caso permanece cerrado para esta pasarela.
func PrepararPeticionAsercionPasarela(r *http.Request, limiteCuerpo int64) (*http.Request, error) {
	if r == nil || r.Context().Err() != nil || r.URL == nil || limiteCuerpo < 0 || limiteCuerpo > limiteCuerpoPasarela ||
		!metodoPasarelaValido(r.Method) || r.URL.IsAbs() || r.URL.Opaque != "" || r.URL.User != nil ||
		r.URL.Fragment != "" || (r.RequestURI != "" && r.RequestURI != r.URL.RequestURI()) {
		return nil, ErrAsercionPasarela
	}
	ruta := r.URL.RequestURI()
	if !rutaPasarelaValida(ruta) {
		return nil, ErrAsercionPasarela
	}
	var cuerpo []byte
	if r.Body != nil && r.Body != http.NoBody {
		leido, err := io.ReadAll(io.LimitReader(r.Body, limiteCuerpo+1))
		_ = r.Body.Close()
		if err != nil || int64(len(leido)) > limiteCuerpo {
			clear(leido)
			return nil, ErrAsercionPasarela
		}
		cuerpo = leido
	}
	if r.ContentLength >= 0 && r.ContentLength != int64(len(cuerpo)) {
		clear(cuerpo)
		return nil, ErrAsercionPasarela
	}
	vinculo, err := NuevoVinculoPeticionPasarela(r.Method, ruta, cuerpo)
	if err != nil {
		clear(cuerpo)
		return nil, err
	}
	preparada := r.Clone(context.WithValue(r.Context(), claveContextoPeticionPasarela{}, vinculo))
	if len(cuerpo) == 0 {
		preparada.Body = http.NoBody
	} else {
		preparada.Body = io.NopCloser(bytes.NewReader(cuerpo))
	}
	return preparada, nil
}

func metodoPasarelaValido(valor string) bool {
	if len(valor) == 0 || len(valor) > 16 {
		return false
	}
	for _, c := range valor {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}

func rutaPasarelaValida(valor string) bool {
	if len(valor) == 0 || len(valor) > 2048 || valor[0] != '/' || strings.HasPrefix(valor, "//") ||
		strings.ContainsAny(valor, "\\\r\n#") {
		return false
	}
	u, err := url.ParseRequestURI(valor)
	if err != nil || u.IsAbs() || u.Opaque != "" || u.User != nil || u.Fragment != "" || u.RequestURI() != valor {
		return false
	}
	if strings.Contains(strings.ToLower(u.EscapedPath()), "%2f") || strings.Contains(u.Path, "//") ||
		strings.ContainsAny(u.Path, "\\\x00") {
		return false
	}
	for _, segmento := range strings.Split(u.Path, "/") {
		if segmento == "." || segmento == ".." {
			return false
		}
	}
	return true
}

// ExtractorAsercionPasarela solo transporta el documento firmado. El canal
// se autentica despues mediante ServicioIdentidad; no confia en identidad
// declarada en cabeceras libres.
type ExtractorAsercionPasarela struct{}

func (ExtractorAsercionPasarela) ExtraerAsercionProtegida(r *http.Request) ([]byte, error) {
	if r == nil || r.Header == nil || r.Context().Value(claveContextoPeticionPasarela{}) == nil {
		return nil, ErrAsercionPasarela
	}
	var valores []string
	for nombre, entradas := range r.Header {
		n := strings.ToLower(nombre)
		if n == "authorization" || n == "proxy-authorization" || n == "cookie" ||
			n == "x-forwarded-user" || n == "x-remote-user" || n == "remote-user" ||
			n == "x-authenticated-user" || n == "x-client-cert" || strings.HasPrefix(n, "x-ssl-client-") {
			return nil, ErrAsercionPasarela
		}
		if n == strings.ToLower(CabeceraAsercionPasarela) {
			valores = append(valores, entradas...)
		}
	}
	if len(valores) != 1 || len(valores[0]) == 0 || len(valores[0]) > base64.RawURLEncoding.EncodedLen(longitudMaximaAsercionProtegida) {
		return nil, ErrAsercionPasarela
	}
	protegida, err := base64.RawURLEncoding.DecodeString(valores[0])
	if err != nil || len(protegida) == 0 || len(protegida) > longitudMaximaAsercionProtegida ||
		base64.RawURLEncoding.EncodeToString(protegida) != valores[0] {
		clear(protegida)
		return nil, ErrAsercionPasarela
	}
	return protegida, nil
}

// ReferenciaCanalAsercionPasarela debe calcularse en el extremo emisor sobre
// la MISMA conexion TLS proxy-servicio. El servicio calcula la misma referencia
// al autenticar el handshake mTLS; no se obtiene de una cabecera HTTP.
func ReferenciaCanalAsercionPasarela(estado tls.ConnectionState, superficie Superficie) (string, error) {
	if !superficie.Valida() || !estado.HandshakeComplete ||
		(estado.Version != tls.VersionTLS12 && estado.Version != tls.VersionTLS13) ||
		!suiteCifradoTLSAdmitida(estado.CipherSuite) {
		return "", ErrCanalProxyNoAutenticado
	}
	material, err := exportarMaterialCanalTLS(estado, "VEC-Diputacion-Canal-Identidad-v1", []byte(superficie), sha256.Size)
	if err != nil || len(material) != sha256.Size {
		return "", ErrCanalProxyNoAutenticado
	}
	huella := sha256.Sum256(material)
	clear(material)
	return "tls-exportador:sha256:" + hex.EncodeToString(huella[:]), nil
}

type cargaAsercionPasarela struct {
	Identidad AsercionProxyIdentidad  `json:"identidad"`
	Peticion  VinculoPeticionPasarela `json:"peticion"`
}

type sobreAsercionPasarela struct {
	Version string `json:"v"`
	ClaveID string `json:"kid"`
	Carga   string `json:"payload"`
	Firma   string `json:"sig"`
}

func materialFirmaPasarela(claveID string, carga []byte) []byte {
	material := make([]byte, 0, len(versionAsercionPasarela)+len(claveID)+len(carga)+2)
	material = append(material, versionAsercionPasarela...)
	material = append(material, 0)
	material = append(material, claveID...)
	material = append(material, 0)
	return append(material, carga...)
}

func claveIDPasarelaValida(id string) bool {
	if len(id) < 8 || len(id) > 128 {
		return false
	}
	for _, c := range id {
		if c != '-' && c != '_' && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// ConsultaRevocacionPasarela obliga al proveedor de estado a comprobar emisor,
// clave, asercion, sesion, cuenta y evidencias de factores; un fallo deniega.
type ConsultaRevocacionPasarela struct {
	Emisor, ClaveID, AsercionID, SesionID, CuentaID string
	Factores                                        []FactorAutenticacion
}

type FuenteRevocacionPasarela interface {
	ComprobarEmisorActivo(context.Context, string) error
	ComprobarClaveActiva(context.Context, string, string) error
	ComprobarIdentidadActiva(context.Context, ConsultaRevocacionPasarela) error
}

func comprobarRevocacionPasarela(ctx context.Context, fuente FuenteRevocacionPasarela, consulta ConsultaRevocacionPasarela) error {
	if interfazNulaPasarela(ctx) || interfazNulaPasarela(fuente) ||
		fuente.ComprobarEmisorActivo(ctx, consulta.Emisor) != nil ||
		fuente.ComprobarClaveActiva(ctx, consulta.Emisor, consulta.ClaveID) != nil ||
		fuente.ComprobarIdentidadActiva(ctx, consulta) != nil || ctx.Err() != nil {
		return ErrAsercionPasarela
	}
	return nil
}

func interfazNulaPasarela(valor any) bool {
	if valor == nil {
		return true
	}
	vista := reflect.ValueOf(valor)
	switch vista.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return vista.IsNil()
	default:
		return false
	}
}

// AcreditadorEmisionPasarela es la autoridad institucional que verifica las
// evidencias de factores antes de que el emisor firme. El constructor exige
// esta fuente; el adaptador no fabrica Kerberos, certificados ni ACR.
type AcreditadorEmisionPasarela interface {
	AcreditarIdentidadYFactores(context.Context, AsercionProxyIdentidad) error
}

type EmisorAsercionPasarela struct {
	configuracion ConfiguracionSuperficie
	claveID       string
	firmante      crypto.Signer
	acreditador   AcreditadorEmisionPasarela
	revocacion    FuenteRevocacionPasarela
	reloj         Reloj
}

func NuevoEmisorAsercionPasarela(c ConfiguracionSuperficie, claveID string, firmante crypto.Signer,
	acreditador AcreditadorEmisionPasarela, revocacion FuenteRevocacionPasarela, reloj Reloj,
) (*EmisorAsercionPasarela, error) {
	if c.Validar() != nil || !claveIDPasarelaValida(claveID) || interfazNulaPasarela(firmante) ||
		interfazNulaPasarela(acreditador) || interfazNulaPasarela(revocacion) {
		return nil, ErrAsercionPasarela
	}
	if len(clavePublicaEd25519(firmante.Public())) != ed25519.PublicKeySize {
		return nil, ErrAsercionPasarela
	}
	if reloj == nil {
		reloj = relojSistema{}
	} else if interfazNulaPasarela(reloj) {
		return nil, ErrAsercionPasarela
	}
	return &EmisorAsercionPasarela{configuracion: copiarYNormalizarConfiguracion(c), claveID: claveID,
		firmante: firmante, acreditador: acreditador, revocacion: revocacion, reloj: reloj}, nil
}

func clavePublicaEd25519(clave crypto.PublicKey) ed25519.PublicKey {
	valor, _ := clave.(ed25519.PublicKey)
	return valor
}

// Emitir crea un ID aleatorio por peticion. El ID se consume en el registro
// durable de ServicioIdentidad; un replay conserva firma valida pero falla.
func (e *EmisorAsercionPasarela) Emitir(ctx context.Context, identidad AsercionProxyIdentidad,
	peticion VinculoPeticionPasarela) ([]byte, error) {
	if e == nil || interfazNulaPasarela(ctx) || ctx.Err() != nil ||
		interfazNulaPasarela(e.acreditador) || interfazNulaPasarela(e.revocacion) ||
		!metodoPasarelaValido(peticion.Metodo) || !rutaPasarelaValida(peticion.Ruta) ||
		!huellaSHA256SesionValida(peticion.CuerpoSHA256) {
		return nil, ErrAsercionPasarela
	}
	nonce := make([]byte, 24)
	if _, err := rand.Read(nonce); err != nil {
		return nil, ErrAsercionPasarela
	}
	identidad.ID = "nonce_" + base64.RawURLEncoding.EncodeToString(nonce)
	clear(nonce)
	identidad.Factores = append([]FactorAutenticacion(nil), identidad.Factores...)
	ahora := e.reloj.Ahora()
	if _, err := normalizarYValidarAsercion(identidad, e.configuracion, identidad.CanalVinculadoRef, ahora); err != nil {
		return nil, ErrAsercionPasarela
	}
	paraAcreditar := identidad
	paraAcreditar.Factores = append([]FactorAutenticacion(nil), identidad.Factores...)
	if err := e.acreditador.AcreditarIdentidadYFactores(ctx, paraAcreditar); err != nil {
		return nil, ErrAsercionPasarela
	}
	consulta := consultaRevocacionPasarela(identidad, e.claveID)
	if err := comprobarRevocacionPasarela(ctx, e.revocacion, consulta); err != nil {
		return nil, ErrAsercionPasarela
	}
	carga, err := json.Marshal(cargaAsercionPasarela{Identidad: identidad, Peticion: peticion})
	if err != nil || len(carga) > longitudMaximaAsercionProtegida/2 {
		return nil, ErrAsercionPasarela
	}
	firma, err := e.firmante.Sign(rand.Reader, materialFirmaPasarela(e.claveID, carga), crypto.Hash(0))
	if err != nil || len(firma) != ed25519.SignatureSize {
		return nil, ErrAsercionPasarela
	}
	protegida, err := json.Marshal(sobreAsercionPasarela{Version: versionAsercionPasarela, ClaveID: e.claveID,
		Carga: base64.RawURLEncoding.EncodeToString(carga), Firma: base64.RawURLEncoding.EncodeToString(firma)})
	if err != nil || len(protegida) > longitudMaximaAsercionProtegida {
		return nil, ErrAsercionPasarela
	}
	return protegida, nil
}

func consultaRevocacionPasarela(a AsercionProxyIdentidad, claveID string) ConsultaRevocacionPasarela {
	return ConsultaRevocacionPasarela{Emisor: a.Emisor, ClaveID: claveID, AsercionID: a.ID,
		SesionID: a.SesionID, CuentaID: a.Cuenta.ID, Factores: append([]FactorAutenticacion(nil), a.Factores...)}
}

type VerificadorAsercionPasarela struct {
	configuracion ConfiguracionSuperficie
	claves        map[string]ed25519.PublicKey
	revocacion    FuenteRevocacionPasarela
	reloj         Reloj
}

func NuevoVerificadorAsercionPasarela(c ConfiguracionSuperficie, claves map[string]ed25519.PublicKey,
	revocacion FuenteRevocacionPasarela, reloj Reloj,
) (*VerificadorAsercionPasarela, error) {
	if c.Validar() != nil || len(claves) == 0 || interfazNulaPasarela(revocacion) {
		return nil, ErrAsercionPasarela
	}
	copia := make(map[string]ed25519.PublicKey, len(claves))
	for id, clave := range claves {
		if !claveIDPasarelaValida(id) || len(clave) != ed25519.PublicKeySize {
			return nil, ErrAsercionPasarela
		}
		copia[id] = append(ed25519.PublicKey(nil), clave...)
	}
	if reloj == nil {
		reloj = relojSistema{}
	} else if interfazNulaPasarela(reloj) {
		return nil, ErrAsercionPasarela
	}
	return &VerificadorAsercionPasarela{configuracion: copiarYNormalizarConfiguracion(c), claves: copia,
		revocacion: revocacion, reloj: reloj}, nil
}

func (v *VerificadorAsercionPasarela) Verificar(ctx context.Context, protegida []byte) (AsercionProxyIdentidad, error) {
	if v == nil || interfazNulaPasarela(ctx) || ctx.Err() != nil || len(protegida) == 0 ||
		len(protegida) > longitudMaximaAsercionProtegida || interfazNulaPasarela(v.revocacion) {
		return AsercionProxyIdentidad{}, ErrAsercionPasarela
	}
	peticion, ok := ctx.Value(claveContextoPeticionPasarela{}).(VinculoPeticionPasarela)
	if !ok {
		return AsercionProxyIdentidad{}, ErrAsercionPasarela
	}
	var sobre sobreAsercionPasarela
	if json.Unmarshal(protegida, &sobre) != nil || sobre.Version != versionAsercionPasarela ||
		!claveIDPasarelaValida(sobre.ClaveID) || v.claves[sobre.ClaveID] == nil {
		return AsercionProxyIdentidad{}, ErrAsercionPasarela
	}
	canonico, _ := json.Marshal(sobre)
	if !bytes.Equal(canonico, protegida) {
		return AsercionProxyIdentidad{}, ErrAsercionPasarela
	}
	carga, err := base64.RawURLEncoding.DecodeString(sobre.Carga)
	if err != nil || len(carga) == 0 || len(carga) > longitudMaximaAsercionProtegida/2 ||
		base64.RawURLEncoding.EncodeToString(carga) != sobre.Carga {
		return AsercionProxyIdentidad{}, ErrAsercionPasarela
	}
	firma, err := base64.RawURLEncoding.DecodeString(sobre.Firma)
	if err != nil || len(firma) != ed25519.SignatureSize ||
		!ed25519.Verify(v.claves[sobre.ClaveID], materialFirmaPasarela(sobre.ClaveID, carga), firma) {
		return AsercionProxyIdentidad{}, ErrAsercionPasarela
	}
	var datos cargaAsercionPasarela
	if json.Unmarshal(carga, &datos) != nil {
		return AsercionProxyIdentidad{}, ErrAsercionPasarela
	}
	cargaCanonica, _ := json.Marshal(datos)
	if !bytes.Equal(cargaCanonica, carga) || datos.Peticion != peticion ||
		!metodoPasarelaValido(datos.Peticion.Metodo) || !rutaPasarelaValida(datos.Peticion.Ruta) ||
		!huellaSHA256SesionValida(datos.Peticion.CuerpoSHA256) {
		return AsercionProxyIdentidad{}, ErrAsercionPasarela
	}
	a := datos.Identidad
	if a.Emisor != v.configuracion.EmisorIdentidad || a.Audiencia != v.configuracion.Audiencia ||
		a.Superficie != v.configuracion.Superficie || a.ID == "" ||
		!strings.HasPrefix(a.ID, "nonce_") ||
		!instanteSesionCanonico(a.EmitidaEn) || !instanteSesionCanonico(a.NoAntesDe) ||
		!instanteSesionCanonico(a.ExpiraEn) ||
		!a.ExpiraEn.After(a.EmitidaEn) || a.ExpiraEn.Sub(a.EmitidaEn) > v.configuracion.DuracionMaximaAsercion {
		return AsercionProxyIdentidad{}, ErrAsercionPasarela
	}
	nonce, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(a.ID, "nonce_"))
	if err != nil || len(nonce) != 24 ||
		base64.RawURLEncoding.EncodeToString(nonce) != strings.TrimPrefix(a.ID, "nonce_") {
		return AsercionProxyIdentidad{}, ErrAsercionPasarela
	}
	ahora := v.reloj.Ahora()
	if _, err := normalizarYValidarAsercion(a, v.configuracion, a.CanalVinculadoRef, ahora); err != nil {
		return AsercionProxyIdentidad{}, ErrAsercionPasarela
	}
	if err := comprobarRevocacionPasarela(ctx, v.revocacion, consultaRevocacionPasarela(a, sobre.ClaveID)); err != nil {
		return AsercionProxyIdentidad{}, ErrAsercionPasarela
	}
	return a, nil
}

// DictamenGarantiaPasarela vincula la garantia acreditada a la politica y a
// los factores exactos. El evaluador coteja todos estos campos sin completar
// omisiones ni interpretar un ACR como garantia por si solo.
type DictamenGarantiaPasarela struct {
	Garantia             dominiovec.AuthAssurance
	PoliticaRef          string
	HuellaPoliticaSHA256 string
	FactoresAcreditados  []FactorAutenticacion
}

// FuenteGarantiaAltaAcreditada debe comprobar fuera del texto firmado la
// independencia de los factores y la proteccion exigida por la politica.
// Sin esa fuente no existe evaluador productivo de garantia alta.
type FuenteGarantiaAltaAcreditada interface {
	AcreditarGarantiaAlta(context.Context, EntradaEvaluacionGarantia) (DictamenGarantiaPasarela, error)
}

type EvaluadorGarantiaPasarela struct {
	fuente      FuenteGarantiaAltaAcreditada
	politicaRef string
	huella      string
}

func NuevoEvaluadorGarantiaPasarela(fuente FuenteGarantiaAltaAcreditada, politicaRef, huellaSHA256 string) (*EvaluadorGarantiaPasarela, error) {
	if interfazNulaPasarela(fuente) || !referenciaOpacaSesionValida(politicaRef, "pga_") ||
		!huellaSHA256SesionValida(huellaSHA256) {
		return nil, ErrEvaluadorGarantiaAusente
	}
	return &EvaluadorGarantiaPasarela{fuente: fuente, politicaRef: politicaRef, huella: huellaSHA256}, nil
}

func (e *EvaluadorGarantiaPasarela) Evaluar(ctx context.Context, entrada EntradaEvaluacionGarantia) (ResultadoEvaluacionGarantia, error) {
	if e == nil || interfazNulaPasarela(e.fuente) || interfazNulaPasarela(ctx) || ctx.Err() != nil ||
		(entrada.Superficie != SuperficieInternaCorporativa && entrada.Superficie != SuperficieAdministracionPrivilegiada) ||
		entrada.Emisor == "" || entrada.SujetoID == "" || entrada.CuentaID == "" || entrada.ACRVerificado == "" ||
		len(entrada.Factores) != 2 {
		return ResultadoEvaluacionGarantia{}, ErrAsercionNoValida
	}
	var kerberos, certificado bool
	grupos := make(map[string]struct{}, 2)
	for _, factor := range entrada.Factores {
		if factor.SujetoVinculadoID != entrada.SujetoID || factor.EvidenciaRef == "" || factor.GrupoCriptograficoRef == "" {
			return ResultadoEvaluacionGarantia{}, ErrAsercionNoValida
		}
		grupos[factor.GrupoCriptograficoRef] = struct{}{}
		switch factor.Metodo {
		case MetodoKerberos:
			kerberos = true
		case MetodoCertificado:
			certificado = true
		default:
			return ResultadoEvaluacionGarantia{}, ErrAsercionNoValida
		}
	}
	if !kerberos || !certificado || len(grupos) != 2 ||
		(entrada.MetodoPrimario != MetodoKerberos && entrada.MetodoPrimario != MetodoCertificado) {
		return ResultadoEvaluacionGarantia{}, ErrAsercionNoValida
	}
	copia := entrada
	copia.Factores = append([]FactorAutenticacion(nil), entrada.Factores...)
	dictamen, err := e.fuente.AcreditarGarantiaAlta(ctx, copia)
	if err != nil || ctx.Err() != nil || dictamen.Garantia != dominiovec.AuthAssuranceHigh ||
		dictamen.PoliticaRef != e.politicaRef || dictamen.HuellaPoliticaSHA256 != e.huella ||
		!factoresAcreditadosCoinciden(entrada.Factores, dictamen.FactoresAcreditados) {
		return ResultadoEvaluacionGarantia{}, ErrAsercionNoValida
	}
	return ResultadoEvaluacionGarantia{Garantia: dominiovec.AuthAssuranceHigh,
		PoliticaRef: e.politicaRef, HuellaPolitica: "sha256:" + e.huella}, nil
}

func factoresAcreditadosCoinciden(declarados, acreditados []FactorAutenticacion) bool {
	if len(declarados) != len(acreditados) {
		return false
	}
	porEvidencia := make(map[string]FactorAutenticacion, len(acreditados))
	for _, factor := range acreditados {
		if factor.EvidenciaRef == "" {
			return false
		}
		if _, duplicado := porEvidencia[factor.EvidenciaRef]; duplicado {
			return false
		}
		porEvidencia[factor.EvidenciaRef] = factor
	}
	for _, declarado := range declarados {
		acreditado, existe := porEvidencia[declarado.EvidenciaRef]
		if !existe || acreditado.Metodo != declarado.Metodo ||
			acreditado.SujetoVinculadoID != declarado.SujetoVinculadoID ||
			acreditado.Principal != declarado.Principal || acreditado.CredencialRef != declarado.CredencialRef ||
			acreditado.GrupoCriptograficoRef != declarado.GrupoCriptograficoRef ||
			!acreditado.VerificadoEn.Equal(declarado.VerificadoEn) {
			return false
		}
	}
	return true
}

var _ VerificadorAsercionProtegida = (*VerificadorAsercionPasarela)(nil)
var _ EvaluadorGarantia = (*EvaluadorGarantiaPasarela)(nil)
