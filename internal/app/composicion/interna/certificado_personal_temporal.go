package interna

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"vec-diputacion-granada/internal/app/composicion/identidadcertificado"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var ErrCertificadoPersonalNoDisponible = identidadcertificado.ErrNoDisponible

const proteccionClavePersonalPKCS11 = identidadcertificado.ProteccionClavePKCS11

type registroCertificadosPersonales = identidadcertificado.Registro
type certificadoPersonalRegistrado = identidadcertificado.CertificadoRegistrado
type documentoCertificadosPersonales = identidadcertificado.DocumentoRegistro

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
	c, err := a.registro.Resolver(ctx, huella)
	if err != nil || f.CredencialRef != "cert:"+c.HuellaSHA256 ||
		c.SujetoID != identidad.SujetoID || c.CuentaID != identidad.Cuenta.ID ||
		!identidadcertificado.IdentificadorValido(identidad.SesionID, "ses_") ||
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
	c, err := a.registro.Resolver(ctx, huella)
	if err != nil || c.CuentaID != consulta.CuentaID ||
		consulta.Factores[0].CredencialRef != "cert:"+c.HuellaSHA256 ||
		!identidadcertificado.IdentificadorValido(consulta.SesionID, "ses_") {
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
	c, err := e.registro.Resolver(ctx, huella)
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
		!identidadcertificado.IdentificadorValido(politicaRef, "pga_") ||
		!strings.HasPrefix(huellaPolitica, "sha256:") ||
		!identidadcertificado.HuellaValida(huellaPolitica) {
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
) (*httpseguridad.ServicioIdentidad, extractorAsercionInstitucional, *registroCertificadosPersonales, error) {
	if interfazNulaIdentidadOffline(sesiones) {
		return nil, nil, nil, ErrCertificadoPersonalNoDisponible
	}
	registro, err := identidadcertificado.NuevoRegistro(rutaRegistro)
	if err != nil {
		return nil, nil, nil, err
	}
	lista, err := registro.LeerCRLActual()
	if err != nil {
		return nil, nil, nil, err
	}
	caPEM, err := leerFicheroTLSSeguro(cfg.AutoridadClientesTLS, false)
	if err != nil {
		return nil, nil, nil, ErrCertificadoPersonalNoDisponible
	}
	autoridades, err := certificadosPEMEstrictos(caPEM)
	limpiarBytesPropios(caPEM)
	if err != nil || len(autoridades) == 0 {
		return nil, nil, nil, ErrCertificadoPersonalNoDisponible
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
		return nil, nil, nil, ErrCertificadoPersonalNoDisponible
	}
	extractor, verificador, evaluador, err := nuevaAsercionCertificadoPersonal(
		cfg, claveID, firmante, registro, politicaRef, huellaPolitica,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	servicio, err := httpseguridad.NuevoServicioIdentidad(
		cfg.configuracionSuperficie(), verificador, evaluador, sesiones, nil,
	)
	if err != nil {
		return nil, nil, nil, ErrCertificadoPersonalNoDisponible
	}
	return servicio, extractor, registro, nil
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
		!identidadcertificado.CertificadoAdmisible(certificado) {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	acreditacion, err := e.registro.AcreditarCadenaActual(r.Context(), r.TLS.VerifiedChains[0], ahora)
	if err != nil {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	huella := acreditacion.CertificadoSHA256
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
		SujetoID:   acreditacion.SujetoID,
		Cuenta:     httpseguridad.CuentaAcceso{ID: acreditacion.CuentaID, SujetoVinculadoID: acreditacion.SujetoID},
		SesionID:   sesionID, CanalVinculadoRef: canal,
		AutenticacionVerificadaEn: ahora, EmitidaEn: ahora, NoAntesDe: ahora,
		ExpiraEn: expira, MetodoPrimario: httpseguridad.MetodoCertificado,
		ACRVerificado: httpseguridad.ACRCertificadoPersonalDesarrolloProtegido,
		Factores: []httpseguridad.FactorAutenticacion{{
			Metodo: httpseguridad.MetodoCertificado, SujetoVinculadoID: acreditacion.SujetoID,
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
