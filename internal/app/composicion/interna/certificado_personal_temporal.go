package interna

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var ErrCertificadoPersonalNoDisponible = errors.New("composicion interna: certificado personal no disponible")

const (
	proteccionClavePersonalPKCS11 = "pkcs11-pin-no-exportable"
	limiteRegistroCertificados    = 64 << 10
)

// El registro privado liga la credencial técnica a una cuenta. La persona, su
// perfil y sus ámbitos se obtienen después exclusivamente del F1 registrado.
// Se relee en cada petición para que una retirada tenga efecto sin reiniciar.
type registroCertificadosPersonales struct {
	ruta, rutaCRL string
}

type certificadoPersonalRegistrado struct {
	HuellaSHA256       string `json:"huella_sha256"`
	SujetoID           string `json:"sujeto_id"`
	CuentaID           string `json:"cuenta_id"`
	ProteccionClaveRef string `json:"proteccion_clave_ref"`
	Activo             bool   `json:"activo"`
}

type documentoCertificadosPersonales struct {
	Version      int                             `json:"version"`
	Certificados []certificadoPersonalRegistrado `json:"certificados"`
}

func nuevoRegistroCertificadosPersonales(ruta string) (*registroCertificadosPersonales, error) {
	if !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	r := &registroCertificadosPersonales{ruta: ruta}
	r.rutaCRL = filepath.Join(filepath.Dir(ruta), "clientes.crl")
	if _, err := r.leer(); err != nil {
		return nil, err
	}
	if _, err := r.leerCRL(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *registroCertificadosPersonales) leerCRL() (*x509.RevocationList, error) {
	if r == nil || r.rutaCRL == "" {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	contenido, err := leerArchivoPrivadoIdentidad(r.rutaCRL, limiteRegistroCertificados)
	if err != nil {
		return nil, err
	}
	defer clear(contenido)
	bloque, resto := pem.Decode(contenido)
	if bloque == nil || bloque.Type != "X509 CRL" || len(bloque.Headers) != 0 ||
		len(bytes.TrimSpace(resto)) != 0 {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	lista, err := x509.ParseRevocationList(bloque.Bytes)
	if err != nil || lista == nil || lista.ThisUpdate.IsZero() || lista.NextUpdate.IsZero() {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	return lista, nil
}

func (r *registroCertificadosPersonales) comprobarCRL(
	certificado, autoridad *x509.Certificate, ahora time.Time,
) error {
	if certificado == nil || autoridad == nil || ahora.Before(certificado.NotBefore) ||
		!ahora.Before(certificado.NotAfter) || ahora.Before(autoridad.NotBefore) ||
		!ahora.Before(autoridad.NotAfter) {
		return ErrCertificadoPersonalNoDisponible
	}
	lista, err := r.leerCRL()
	if err != nil || !bytes.Equal(lista.RawIssuer, autoridad.RawSubject) ||
		lista.CheckSignatureFrom(autoridad) != nil ||
		ahora.Before(lista.ThisUpdate) || !ahora.Before(lista.NextUpdate) {
		return ErrCertificadoPersonalNoDisponible
	}
	for _, entrada := range lista.RevokedCertificateEntries {
		if entrada.SerialNumber != nil && entrada.SerialNumber.Cmp(certificado.SerialNumber) == 0 {
			return ErrCertificadoPersonalNoDisponible
		}
	}
	return nil
}

func (r *registroCertificadosPersonales) leer() (map[string]certificadoPersonalRegistrado, error) {
	if r == nil || !filepath.IsAbs(r.ruta) {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	info, err := os.Lstat(r.ruta)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > limiteRegistroCertificados {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	f, err := os.Open(r.ruta)
	if err != nil {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil || !os.SameFile(info, actual) || !actual.Mode().IsRegular() || actual.Mode().Perm() != 0600 {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	contenido, err := io.ReadAll(io.LimitReader(f, limiteRegistroCertificados+1))
	if err != nil || len(contenido) == 0 || len(contenido) > limiteRegistroCertificados {
		clear(contenido)
		return nil, ErrCertificadoPersonalNoDisponible
	}
	defer clear(contenido)
	if rechazarClavesDuplicadasPools(contenido) != nil {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	var documento documentoCertificadosPersonales
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if decodificador.Decode(&documento) != nil || decodificador.Decode(new(any)) != io.EOF ||
		documento.Version != 1 || len(documento.Certificados) == 0 || len(documento.Certificados) > 64 {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	porHuella := make(map[string]certificadoPersonalRegistrado, len(documento.Certificados))
	cuentas := make(map[string]bool, len(documento.Certificados))
	for _, c := range documento.Certificados {
		if !huellaCertificadoPersonalValida(c.HuellaSHA256) ||
			!identificadorCertificadoPersonalValido(c.SujetoID, "per_") ||
			!identificadorCertificadoPersonalValido(c.CuentaID, "cta_") ||
			c.ProteccionClaveRef != proteccionClavePersonalPKCS11 ||
			porHuella[c.HuellaSHA256].HuellaSHA256 != "" || cuentas[c.CuentaID] {
			return nil, ErrCertificadoPersonalNoDisponible
		}
		porHuella[c.HuellaSHA256] = c
		cuentas[c.CuentaID] = true
	}
	return porHuella, nil
}

func huellaCertificadoPersonalValida(v string) bool {
	if !strings.HasPrefix(v, "sha256:") || len(v) != len("sha256:")+sha256.Size*2 {
		return false
	}
	for _, c := range v[len("sha256:"):] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func identificadorCertificadoPersonalValido(v, prefijo string) bool {
	if !strings.HasPrefix(v, prefijo) || len(v) < len(prefijo)+20 || len(v) > 128 {
		return false
	}
	for _, c := range v[len(prefijo):] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || c == '_') {
			return false
		}
	}
	return true
}

func (r *registroCertificadosPersonales) resolver(ctx context.Context, huella string) (certificadoPersonalRegistrado, error) {
	if ctx == nil || ctx.Err() != nil || !huellaCertificadoPersonalValida(huella) {
		return certificadoPersonalRegistrado{}, ErrCertificadoPersonalNoDisponible
	}
	todos, err := r.leer()
	if err != nil {
		return certificadoPersonalRegistrado{}, err
	}
	c, existe := todos[huella]
	if !existe || !c.Activo {
		return certificadoPersonalRegistrado{}, ErrCertificadoPersonalNoDisponible
	}
	return c, nil
}

// Esta fuente acredita que el certificado concreto fue emitido para una clave
// no exportable del token de desarrollo y sigue activo. El handshake aporta la
// prueba de posesión; la fuente no atribuye por ello garantía alta.
type acreditadorCertificadoPersonal struct {
	registro        *registroCertificadosPersonales
	emisor, claveID string
}

func (a acreditadorCertificadoPersonal) AcreditarIdentidadYFactores(ctx context.Context, identidad httpseguridad.AsercionProxyIdentidad) error {
	if len(identidad.Factores) != 1 || identidad.Factores[0].Metodo != httpseguridad.MetodoCertificado {
		return ErrCertificadoPersonalNoDisponible
	}
	f := identidad.Factores[0]
	huella := strings.TrimPrefix(f.CredencialRef, "cert:")
	c, err := a.registro.resolver(ctx, huella)
	if err != nil || f.CredencialRef != "cert:"+c.HuellaSHA256 ||
		c.SujetoID != identidad.SujetoID || c.CuentaID != identidad.Cuenta.ID ||
		!identificadorCertificadoPersonalValido(identidad.SesionID, "ses_") ||
		f.SujetoVinculadoID != c.SujetoID {
		return ErrCertificadoPersonalNoDisponible
	}
	return nil
}

func (a acreditadorCertificadoPersonal) ComprobarEmisorActivo(_ context.Context, emisor string) error {
	if emisor == "" || emisor != a.emisor {
		return ErrCertificadoPersonalNoDisponible
	}
	return nil
}

func (a acreditadorCertificadoPersonal) ComprobarClaveActiva(_ context.Context, emisor, claveID string) error {
	if emisor != a.emisor || claveID != a.claveID {
		return ErrCertificadoPersonalNoDisponible
	}
	return nil
}

func (a acreditadorCertificadoPersonal) ComprobarIdentidadActiva(ctx context.Context, consulta httpseguridad.ConsultaRevocacionPasarela) error {
	if consulta.Emisor != a.emisor || consulta.ClaveID != a.claveID ||
		len(consulta.Factores) != 1 || consulta.Factores[0].Metodo != httpseguridad.MetodoCertificado {
		return ErrCertificadoPersonalNoDisponible
	}
	huella := strings.TrimPrefix(consulta.Factores[0].CredencialRef, "cert:")
	c, err := a.registro.resolver(ctx, huella)
	if err != nil || c.CuentaID != consulta.CuentaID ||
		consulta.Factores[0].CredencialRef != "cert:"+c.HuellaSHA256 ||
		!identificadorCertificadoPersonalValido(consulta.SesionID, "ses_") {
		return ErrCertificadoPersonalNoDisponible
	}
	return nil
}

// El navegador no construye ni recibe aserciones. C4 aporta el estado de una
// conexión TLS directa y este adaptador firma una aserción local por petición.
type extractorCertificadoPersonalDirecto struct {
	emisor              *httpseguridad.EmisorAsercionPasarela
	registro            *registroCertificadosPersonales
	emisorID, audiencia string
	retirada            time.Time
}

type evaluadorCertificadoPersonal struct {
	registro                            *registroCertificadosPersonales
	emisor, politicaRef, huellaPolitica string
}

func (e evaluadorCertificadoPersonal) Evaluar(ctx context.Context, entrada httpseguridad.EntradaEvaluacionGarantia) (httpseguridad.ResultadoEvaluacionGarantia, error) {
	if e.registro == nil || ctx == nil || ctx.Err() != nil ||
		entrada.Emisor != e.emisor || entrada.Superficie != httpseguridad.SuperficieInternaCorporativa ||
		entrada.ACRVerificado != httpseguridad.ACRCertificadoPersonalDesarrolloProtegido ||
		entrada.MetodoPrimario != httpseguridad.MetodoCertificado || len(entrada.Factores) != 1 {
		return httpseguridad.ResultadoEvaluacionGarantia{}, ErrCertificadoPersonalNoDisponible
	}
	factor := entrada.Factores[0]
	huella := strings.TrimPrefix(factor.CredencialRef, "cert:")
	c, err := e.registro.resolver(ctx, huella)
	if err != nil || factor.Metodo != httpseguridad.MetodoCertificado ||
		factor.CredencialRef != "cert:"+c.HuellaSHA256 ||
		factor.SujetoVinculadoID != c.SujetoID ||
		entrada.SujetoID != c.SujetoID || entrada.CuentaID != c.CuentaID {
		return httpseguridad.ResultadoEvaluacionGarantia{}, ErrCertificadoPersonalNoDisponible
	}
	return httpseguridad.ResultadoEvaluacionGarantia{
		Garantia:    dominiovec.AuthAssuranceSubstantial,
		PoliticaRef: e.politicaRef, HuellaPolitica: e.huellaPolitica,
	}, nil
}

// El emisor y el verificador usan la misma política temporal y el mismo
// registro revocable. La clave Ed25519 firma transporte; no es credencial de
// la persona ni reemplaza el certificado del navegador.
func nuevaAsercionCertificadoPersonal(
	cfg Configuracion, claveID string, firmante crypto.Signer,
	registro *registroCertificadosPersonales, politicaRef, huellaPolitica string,
) (*extractorCertificadoPersonalDirecto, *httpseguridad.VerificadorAsercionPasarela,
	httpseguridad.EvaluadorGarantia, error) {
	if cfg.Validar() != nil || cfg.RetiradaPoliticaInternaEn.IsZero() ||
		registro == nil || firmante == nil ||
		!identificadorCertificadoPersonalValido(politicaRef, "pga_") ||
		!strings.HasPrefix(huellaPolitica, "sha256:") ||
		!huellaCertificadoPersonalValida(huellaPolitica) {
		return nil, nil, nil, ErrCertificadoPersonalNoDisponible
	}
	publica, ok := firmante.Public().(ed25519.PublicKey)
	if !ok || len(publica) != ed25519.PublicKeySize {
		return nil, nil, nil, ErrCertificadoPersonalNoDisponible
	}
	fuente := acreditadorCertificadoPersonal{registro: registro, emisor: cfg.EmisorIdentidad, claveID: claveID}
	superficie := cfg.configuracionSuperficie()
	emisor, err := httpseguridad.NuevoEmisorAsercionPasarela(superficie, claveID, firmante, fuente, fuente, nil)
	if err != nil {
		return nil, nil, nil, ErrCertificadoPersonalNoDisponible
	}
	verificador, err := httpseguridad.NuevoVerificadorAsercionPasarela(superficie,
		map[string]ed25519.PublicKey{claveID: publica}, fuente, nil)
	if err != nil {
		return nil, nil, nil, ErrCertificadoPersonalNoDisponible
	}
	extractor := &extractorCertificadoPersonalDirecto{
		emisor: emisor, registro: registro, emisorID: cfg.EmisorIdentidad,
		audiencia: cfg.Audiencia, retirada: cfg.RetiradaPoliticaInternaEn,
	}
	evaluador := evaluadorCertificadoPersonal{
		registro: registro, emisor: cfg.EmisorIdentidad,
		politicaRef: politicaRef, huellaPolitica: huellaPolitica,
	}
	return extractor, verificador, evaluador, nil
}

// montarServicioIdentidadCertificado sólo enlaza la autoridad común con el
// registro durable ya construido. La raíz que lo llame debe haber acreditado
// previamente los LOGIN y el conector HSM de seudonimización.
func montarServicioIdentidadCertificado(
	cfg Configuracion, claveID string, firmante crypto.Signer,
	rutaRegistro string, politicaRef, huellaPolitica string,
	sesiones httpseguridad.RegistroSesiones,
) (*httpseguridad.ServicioIdentidad, extractorAsercionInstitucional, error) {
	if interfazNulaIdentidadOffline(sesiones) {
		return nil, nil, ErrCertificadoPersonalNoDisponible
	}
	registro, err := nuevoRegistroCertificadosPersonales(rutaRegistro)
	if err != nil {
		return nil, nil, err
	}
	lista, err := registro.leerCRL()
	if err != nil {
		return nil, nil, err
	}
	caPEM, err := leerFicheroTLSSeguro(cfg.AutoridadClientesTLS, false)
	if err != nil {
		return nil, nil, ErrCertificadoPersonalNoDisponible
	}
	autoridades, err := certificadosPEMEstrictos(caPEM)
	limpiarBytesPropios(caPEM)
	if err != nil || len(autoridades) == 0 {
		return nil, nil, ErrCertificadoPersonalNoDisponible
	}
	ahora := time.Now().UTC()
	crlValida := false
	for _, autoridad := range autoridades {
		if autoridad != nil && bytes.Equal(lista.RawIssuer, autoridad.RawSubject) &&
			lista.CheckSignatureFrom(autoridad) == nil &&
			!ahora.Before(lista.ThisUpdate) && ahora.Before(lista.NextUpdate) {
			crlValida = true
		}
	}
	if !crlValida {
		return nil, nil, ErrCertificadoPersonalNoDisponible
	}
	extractor, verificador, evaluador, err := nuevaAsercionCertificadoPersonal(
		cfg, claveID, firmante, registro, politicaRef, huellaPolitica,
	)
	if err != nil {
		return nil, nil, err
	}
	servicio, err := httpseguridad.NuevoServicioIdentidad(
		cfg.configuracionSuperficie(), verificador, evaluador, sesiones, nil,
	)
	if err != nil {
		return nil, nil, ErrCertificadoPersonalNoDisponible
	}
	return servicio, extractor, nil
}

func (e *extractorCertificadoPersonalDirecto) ExtraerAsercionProtegida(r *http.Request) ([]byte, error) {
	if e == nil || e.emisor == nil || e.registro == nil || r == nil || r.URL == nil ||
		r.TLS == nil || r.Method != http.MethodGet || r.Body != http.NoBody ||
		r.ContentLength > 0 || len(r.TransferEncoding) != 0 ||
		!cabecerasCertificadoPersonalValidas(r.Header) {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	capacidad, ok := r.Context().Value(claveContextoCanalTLSInterno{}).(*capacidadCanalTLSInterno)
	if !ok || capacidad == nil || len(capacidad.estadoTLS.VerifiedChains) == 0 ||
		len(capacidad.estadoTLS.VerifiedChains[0]) == 0 ||
		len(r.TLS.VerifiedChains) == 0 || len(r.TLS.PeerCertificates) == 0 ||
		len(r.TLS.VerifiedChains[0]) < 2 {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	certificado := r.TLS.VerifiedChains[0][0]
	if certificado == nil || capacidad.estadoTLS.VerifiedChains[0][0] == nil || r.TLS.PeerCertificates[0] == nil ||
		!bytes.Equal(certificado.Raw, r.TLS.PeerCertificates[0].Raw) ||
		!bytes.Equal(certificado.Raw, capacidad.estadoTLS.VerifiedChains[0][0].Raw) ||
		!certificadoPersonalAdmisible(certificado) {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	if e.registro.comprobarCRL(certificado, r.TLS.VerifiedChains[0][1], ahora) != nil {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	huellaBytes := sha256.Sum256(certificado.Raw)
	huella := "sha256:" + hex.EncodeToString(huellaBytes[:])
	registro, err := e.registro.resolver(r.Context(), huella)
	if err != nil {
		return nil, err
	}
	canal, err := httpseguridad.ReferenciaCanalAsercionPasarela(*r.TLS, httpseguridad.SuperficieInternaCorporativa)
	if err != nil {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	expira := ahora.Add(2 * time.Minute)
	if !e.retirada.IsZero() && !expira.Before(e.retirada) {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	vinculo, err := httpseguridad.NuevoVinculoPeticionPasarela(r.Method, r.URL.RequestURI(), nil)
	if err != nil {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	var sesionAleatoria [24]byte
	if _, err := rand.Read(sesionAleatoria[:]); err != nil {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	sesionID := "ses_" + hex.EncodeToString(sesionAleatoria[:])
	clear(sesionAleatoria[:])
	identidad := httpseguridad.AsercionProxyIdentidad{
		Emisor: e.emisorID, Audiencia: e.audiencia,
		Superficie: httpseguridad.SuperficieInternaCorporativa,
		SujetoID:   registro.SujetoID,
		Cuenta:     httpseguridad.CuentaAcceso{ID: registro.CuentaID, SujetoVinculadoID: registro.SujetoID},
		SesionID:   sesionID, CanalVinculadoRef: canal,
		AutenticacionVerificadaEn: ahora, EmitidaEn: ahora, NoAntesDe: ahora,
		ExpiraEn: expira, MetodoPrimario: httpseguridad.MetodoCertificado,
		ACRVerificado: httpseguridad.ACRCertificadoPersonalDesarrolloProtegido,
		Factores: []httpseguridad.FactorAutenticacion{{
			Metodo: httpseguridad.MetodoCertificado, SujetoVinculadoID: registro.SujetoID,
			CredencialRef: "cert:" + huella, EvidenciaRef: "tls:verified:" + huella,
			GrupoCriptograficoRef: "key:" + huella, VerificadoEn: ahora,
		}},
	}
	protegida, err := e.emisor.Emitir(r.Context(), identidad, vinculo)
	if err != nil {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	return protegida, nil
}

func cabecerasCertificadoPersonalValidas(h http.Header) bool {
	for nombre := range h {
		n := strings.ToLower(nombre)
		if n == "authorization" || n == "proxy-authorization" || n == "cookie" ||
			n == strings.ToLower(httpseguridad.CabeceraAsercionPasarela) ||
			n == "forwarded" || n == "remote-user" || n == "x-remote-user" ||
			n == "x-authenticated-user" || n == "x-user" ||
			strings.HasPrefix(n, "x-forwarded-") || strings.HasPrefix(n, "x-auth-") ||
			strings.HasPrefix(n, "x-identity-") || strings.HasPrefix(n, "x-client-") ||
			strings.HasPrefix(n, "x-ssl-") {
			return false
		}
	}
	return true
}

func certificadoPersonalAdmisible(c *x509.Certificate) bool {
	if c == nil || c.IsCA || c.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		return false
	}
	for _, uso := range c.ExtKeyUsage {
		if uso == x509.ExtKeyUsageClientAuth {
			return true
		}
	}
	return false
}
