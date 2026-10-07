package httpseguridad

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// ServicioPresentacionCertificado fija las autoridades de certificado y
// persistencia al constituirse. No toma una fuente ni un puerto del request.
// La reanudacion valida una presentacion NUEVA y conserva la sesion original.
type ServicioPresentacionCertificado struct {
	identidad    *ServicioIdentidad
	certificador CertificadorPresentacionActual
	registro     RegistroPresentacionesCertificado
}

func NuevoServicioPresentacionCertificado(
	identidad *ServicioIdentidad,
	certificador CertificadorPresentacionActual,
	registro RegistroPresentacionesCertificado,
) (*ServicioPresentacionCertificado, error) {
	if identidad == nil || interfazNulaPasarela(certificador) || interfazNulaPasarela(registro) ||
		identidad.configuracion.Superficie != SuperficieInternaCorporativa ||
		identidad.configuracion.PoliticaInterna != PoliticaInternaDesarrolloCertificadoPersonal ||
		identidad.verificador == nil || identidad.evaluadorGarantia == nil || identidad.reloj == nil {
		return nil, ErrPresentacionCertificadoNoValida
	}
	return &ServicioPresentacionCertificado{identidad: identidad, certificador: certificador, registro: registro}, nil
}

// AcreditarCertificadoActual solo acepta el estado de una conexion mTLS cuyo
// handshake completo emitio un exportador TLS valido. La composicion C4 debe
// consumir su capacidad de canal antes de pasar el estado; el certificador
// fijado coteja registro privado y CRL actuales y devuelve sus limites.
func (s *ServicioPresentacionCertificado) AcreditarCertificadoActual(
	ctx context.Context,
	estado tls.ConnectionState,
) (context.Context, CanalProxyAutenticado, PruebaCertificadoActual, error) {
	if s == nil || s.identidad == nil || interfazNulaPasarela(s.certificador) || ctx == nil ||
		len(estado.VerifiedChains) != 1 || len(estado.VerifiedChains[0]) < 2 ||
		len(estado.PeerCertificates) == 0 ||
		len(estado.PeerCertificates) > len(estado.VerifiedChains[0]) {
		return nil, CanalProxyAutenticado{}, PruebaCertificadoActual{}, ErrPresentacionCertificadoNoValida
	}
	if err := ctx.Err(); err != nil {
		return nil, CanalProxyAutenticado{}, PruebaCertificadoActual{}, err
	}
	for i, certificadoPar := range estado.PeerCertificates {
		if certificadoPar == nil || estado.VerifiedChains[0][i] == nil ||
			!bytes.Equal(certificadoPar.Raw, estado.VerifiedChains[0][i].Raw) {
			return nil, CanalProxyAutenticado{}, PruebaCertificadoActual{}, ErrPresentacionCertificadoNoValida
		}
	}
	canal, err := s.identidad.AutenticarCanalTLSMutuo(estado)
	if err != nil || canal.validar(s.identidad) != nil {
		return nil, CanalProxyAutenticado{}, PruebaCertificadoActual{}, ErrPresentacionCertificadoNoValida
	}
	cadena := make([]*x509.Certificate, len(estado.VerifiedChains[0]))
	cadenaParaFuente := make([]*x509.Certificate, len(cadena))
	for i, certificadoTLS := range estado.VerifiedChains[0] {
		if certificadoTLS == nil || len(certificadoTLS.Raw) == 0 {
			return nil, CanalProxyAutenticado{}, PruebaCertificadoActual{}, ErrPresentacionCertificadoNoValida
		}
		var errParse error
		cadena[i], errParse = x509.ParseCertificate(bytes.Clone(certificadoTLS.Raw))
		if errParse != nil {
			return nil, CanalProxyAutenticado{}, PruebaCertificadoActual{}, ErrPresentacionCertificadoNoValida
		}
		cadenaParaFuente[i], errParse = x509.ParseCertificate(bytes.Clone(certificadoTLS.Raw))
		if errParse != nil {
			return nil, CanalProxyAutenticado{}, PruebaCertificadoActual{}, ErrPresentacionCertificadoNoValida
		}
		if i > 0 && cadena[i-1].CheckSignatureFrom(cadena[i]) != nil {
			return nil, CanalProxyAutenticado{}, PruebaCertificadoActual{}, ErrPresentacionCertificadoNoValida
		}
	}
	certificado, ca := cadena[0], cadena[1]
	if certificado.IsCA || certificado.KeyUsage&x509.KeyUsageDigitalSignature == 0 ||
		!certificadoAdmiteAutenticacionCliente(certificado) {
		return nil, CanalProxyAutenticado{}, PruebaCertificadoActual{}, ErrPresentacionCertificadoNoValida
	}
	ahora := s.identidad.reloj.Ahora().UTC().Truncate(time.Microsecond)
	certificadoHasta := certificado.NotAfter.UTC().Truncate(time.Microsecond)
	cadenaHasta := ca.NotAfter.UTC().Truncate(time.Microsecond)
	if ahora.IsZero() || ahora.Before(certificado.NotBefore) || !ahora.Before(certificadoHasta) {
		return nil, CanalProxyAutenticado{}, PruebaCertificadoActual{}, ErrPresentacionCertificadoNoValida
	}
	for _, autoridad := range cadena[1:] {
		if ahora.Before(autoridad.NotBefore) || !ahora.Before(autoridad.NotAfter) {
			return nil, CanalProxyAutenticado{}, PruebaCertificadoActual{}, ErrPresentacionCertificadoNoValida
		}
		if autoridad.NotAfter.Before(cadenaHasta) {
			cadenaHasta = autoridad.NotAfter.UTC().Truncate(time.Microsecond)
		}
	}
	huellaCertificado := sha256.Sum256(certificado.Raw)
	huellaCA := sha256.Sum256(ca.Raw)
	certificadoSHA256 := "sha256:" + hex.EncodeToString(huellaCertificado[:])
	caSHA256 := "sha256:" + hex.EncodeToString(huellaCA[:])
	inicioComprobacion := ahora
	acreditacion, err := s.certificador.AcreditarCertificadoActual(ctx, cadenaParaFuente, ahora)
	if err != nil {
		return nil, CanalProxyAutenticado{}, PruebaCertificadoActual{}, falloPresentacionCertificado(ctx, err)
	}
	ahora = s.identidad.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if err := ctx.Err(); err != nil {
		return nil, CanalProxyAutenticado{}, PruebaCertificadoActual{}, err
	}
	if ahora.IsZero() || !instanteSesionCanonico(acreditacion.CRLSiguienteEn) ||
		!instanteSesionCanonico(acreditacion.RevocacionVerificadaEn) ||
		acreditacion.CertificadoSHA256 != certificadoSHA256 || acreditacion.CASHA256 != caSHA256 ||
		!acreditacion.CertificadoValidoHasta.Equal(certificadoHasta) ||
		!acreditacion.CAValidaHasta.Equal(cadenaHasta) ||
		acreditacion.RevocacionVerificadaEn.Before(inicioComprobacion) ||
		acreditacion.RevocacionVerificadaEn.After(ahora) ||
		!ahora.Before(acreditacion.CRLSiguienteEn) ||
		!ahora.Before(certificadoHasta) || !ahora.Before(cadenaHasta) {
		return nil, CanalProxyAutenticado{}, PruebaCertificadoActual{}, ErrPresentacionCertificadoNoValida
	}
	sujeto, errSujeto := canonicalizarID(acreditacion.SujetoID, longitudMaximaID, false)
	cuenta, errCuenta := canonicalizarID(acreditacion.CuentaID, longitudMaximaID, true)
	if errSujeto != nil || errCuenta != nil || sujeto != acreditacion.SujetoID ||
		sujeto == cuenta {
		return nil, CanalProxyAutenticado{}, PruebaCertificadoActual{}, ErrPresentacionCertificadoNoValida
	}
	marcador := &marcaPeticionCertificadoActual{}
	ctxAcreditado := context.WithValue(ctx, clavePeticionCertificadoActual{}, marcador)
	return ctxAcreditado, canal, PruebaCertificadoActual{datos: &datosPruebaCertificadoActual{
		marcador: marcador,
		servicio: s, canalRef: canal.ReferenciaVinculacion(),
		certificadoSHA256: certificadoSHA256,
		caSHA256:          caSHA256,
		sujetoID:          sujeto, cuentaID: cuenta, verificadaEn: acreditacion.RevocacionVerificadaEn,
		certificadoHasta: certificadoHasta,
		caHasta:          cadenaHasta,
		crlSiguienteEn:   acreditacion.CRLSiguienteEn,
	}}, nil
}

func falloPresentacionCertificado(ctx context.Context, causa error) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return errors.Join(ErrPresentacionCertificadoNoValida, err)
		}
	}
	// El error arbitrario del certificador no atraviesa esta frontera publica.
	_ = causa
	return ErrPresentacionCertificadoNoValida
}

// IniciarYConsumirPresentacion deja a UNA transaccion durable decidir si
// abre, reanuda o renueva. Revocada nunca se convierte en otra alta.
func (s *ServicioPresentacionCertificado) IniciarYConsumirPresentacion(ctx context.Context, credencial CredencialProxy, prueba PruebaCertificadoActual) (CapsulaPresentacionCertificado, error) {
	return s.consumirPresentacion(ctx, credencial, prueba, true)
}

// ReanudarYConsumirPresentacion es la unica entrada de operaciones ordinarias.
// Ausencia, expiracion o denegacion no disparan apertura ni renovacion.
func (s *ServicioPresentacionCertificado) ReanudarYConsumirPresentacion(ctx context.Context, credencial CredencialProxy, prueba PruebaCertificadoActual) (CapsulaPresentacionCertificado, error) {
	return s.consumirPresentacion(ctx, credencial, prueba, false)
}

func (s *ServicioPresentacionCertificado) consumirPresentacion(
	ctx context.Context, credencial CredencialProxy, prueba PruebaCertificadoActual, inicio bool,
) (CapsulaPresentacionCertificado, error) {
	var vacia CapsulaPresentacionCertificado
	if s == nil || s.identidad == nil || interfazNulaPasarela(s.registro) || ctx == nil ||
		prueba.datos == nil || prueba.datos.servicio != s ||
		ctx.Value(clavePeticionCertificadoActual{}) != prueba.datos.marcador ||
		credencial.canal.validar(s.identidad) != nil ||
		subtle.ConstantTimeCompare([]byte(prueba.datos.canalRef), []byte(credencial.canal.ReferenciaVinculacion())) != 1 ||
		len(credencial.asercionProtegida) == 0 || len(credencial.asercionProtegida) > longitudMaximaAsercionProtegida {
		return vacia, ErrPresentacionCertificadoNoValida
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	ahora := s.identidad.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !pruebaVigentePresentacionCertificado(prueba, ahora) ||
		!prueba.datos.consumida.CompareAndSwap(false, true) {
		return vacia, ErrPresentacionCertificadoNoValida
	}
	huellaActual := sha256.Sum256(credencial.asercionProtegida)
	protegida := bytes.Clone(credencial.asercionProtegida)
	asercion, err := s.identidad.verificador.Verificar(ctx, protegida)
	clear(protegida)
	if err != nil {
		return vacia, falloPresentacionCertificado(ctx, err)
	}
	estado, err := normalizarYValidarAsercion(
		asercion, s.identidad.configuracion, credencial.canal.ReferenciaVinculacion(), ahora,
	)
	if err != nil {
		return vacia, ErrPresentacionCertificadoNoValida
	}
	garantia, err := s.identidad.evaluarGarantia(ctx, estado)
	if err != nil {
		return vacia, falloPresentacionCertificado(ctx, err)
	}
	estado.garantia = garantia.Garantia
	estado.politicaGarantiaRef = garantia.PoliticaRef
	estado.huellaPolitica = garantia.HuellaPolitica
	estado.autenticacionHuellaSHA256 = hex.EncodeToString(huellaActual[:])
	metodo, valido := metodoAutenticacionDominio(estado.metodoPrimario)
	if !valido {
		return vacia, ErrPresentacionCertificadoNoValida
	}
	estado.metodoObservado = metodo
	if validarEstadoSesion(estado, s.identidad.configuracion, ahora) != nil ||
		estado.sujetoID != prueba.datos.sujetoID || estado.cuenta.ID != prueba.datos.cuentaID ||
		estado.cuenta.Privilegiada || estado.superficie != SuperficieInternaCorporativa ||
		metodo != domain.AuthMethodCertificate || estado.garantia != domain.AuthAssuranceSubstantial ||
		len(estado.factores) != 1 || estado.factores[0].Metodo != MetodoCertificado ||
		estado.factores[0].CredencialRef != "cert:"+prueba.datos.certificadoSHA256 {
		return vacia, ErrPresentacionCertificadoNoValida
	}
	operacion, err := nuevaOperacionPresentacionCertificado()
	if err != nil {
		return vacia, ErrPresentacionCertificadoNoValida
	}
	canal := sha256.Sum256([]byte(prueba.datos.canalRef))
	presentacion := DatosPresentacionCertificado{
		OperacionRef: operacion, AsercionID: estado.asercionID,
		SesionIDAfirmada: estado.sesionID, SujetoID: estado.sujetoID, CuentaID: estado.cuenta.ID,
		CertificadoSHA256: prueba.datos.certificadoSHA256, CASHA256: prueba.datos.caSHA256,
		CanalRef: prueba.datos.canalRef, CanalSHA256: hex.EncodeToString(canal[:]),
		Superficie: estado.superficie, Emisor: estado.emisor, Audiencia: estado.audiencia,
		ACRVerificado:   estado.acrVerificado,
		MetodoObservado: metodo, GarantiaObservada: estado.garantia,
		PoliticaGarantiaRef:          estado.politicaGarantiaRef,
		PoliticaGarantiaHuellaSHA256: strings.TrimPrefix(estado.huellaPolitica, "sha256:"),
		AsercionActualHuellaSHA256:   estado.autenticacionHuellaSHA256,
		AsercionActualEmitidaEn:      estado.emitidaEn, AsercionActualExpiraEn: estado.expiraEn,
		CertificadoVerificadoEn: prueba.datos.verificadaEn,
		CertificadoValidoHasta:  prueba.datos.certificadoHasta,
		CAValidaHasta:           prueba.datos.caHasta, CRLSiguienteActualizacion: prueba.datos.crlSiguienteEn,
		PoliticaRetiradaEn: s.identidad.configuracion.RetiradaPoliticaInternaEn,
	}
	if validarDatosPresentacionCertificado(presentacion) != nil || ctx.Err() != nil {
		return vacia, falloPresentacionCertificado(ctx, ErrPresentacionCertificadoNoValida)
	}
	var resultado ResultadoRegistroPresentacionCertificado
	if inicio {
		alta := altaSesion(estado)
		if alta.Validar() != nil {
			return vacia, ErrPresentacionCertificadoNoValida
		}
		resultado, err = s.registro.IniciarYConsumirPresentacion(ctx, OrdenInicioCertificado{datos: &datosOrdenInicioCertificado{alta: alta, presentacion: presentacion}})
	} else {
		resultado, err = s.registro.ReanudarYConsumirPresentacion(ctx, OrdenReanudacionCertificado{datos: &presentacion})
	}
	if err != nil {
		return vacia, falloPresentacionCertificado(ctx, err)
	}
	ahoraFinal := s.identidad.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if validarResultadoPresentacionCertificado(resultado, presentacion, estado, inicio, ahoraFinal) != nil {
		return vacia, ErrPresentacionCertificadoNoValida
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	return CapsulaPresentacionCertificado{datos: &datosCapsulaPresentacionCertificado{
		servicio: s, marcador: prueba.datos.marcador,
		canalRef: prueba.datos.canalRef, estadoActual: copiarEstado(estado),
		resultado: resultado, presentacion: presentacion,
	}}, nil
}

func nuevaOperacionPresentacionCertificado() (string, error) {
	var contenido [24]byte
	if _, err := rand.Read(contenido[:]); err != nil {
		return "", err
	}
	referencia := "opr_" + hex.EncodeToString(contenido[:])
	clear(contenido[:])
	return referencia, nil
}

func pruebaVigentePresentacionCertificado(prueba PruebaCertificadoActual, ahora time.Time) bool {
	if prueba.datos == nil || !instanteSesionCanonico(ahora) ||
		prueba.datos.certificadoSHA256 == "" || prueba.datos.caSHA256 == "" ||
		prueba.datos.canalRef == "" || prueba.datos.cuentaID == "" || prueba.datos.sujetoID == "" ||
		!ahora.Before(prueba.datos.certificadoHasta) || !ahora.Before(prueba.datos.caHasta) ||
		!ahora.Before(prueba.datos.crlSiguienteEn) || prueba.datos.verificadaEn.After(ahora) {
		return false
	}
	return true
}

func validarDatosPresentacionCertificado(d DatosPresentacionCertificado) error {
	if !referenciaOpacaSesionValida(d.OperacionRef, "opr_") || d.AsercionID == "" ||
		d.SesionIDAfirmada == "" || d.SujetoID == "" || d.CuentaID == "" ||
		d.CertificadoSHA256 == "" || d.CASHA256 == "" || d.CanalRef == "" ||
		!huellaSHA256SesionValida(d.CanalSHA256) ||
		!huellaSHA256SesionValida(d.AsercionActualHuellaSHA256) ||
		d.Superficie != SuperficieInternaCorporativa || d.MetodoObservado != domain.AuthMethodCertificate ||
		d.GarantiaObservada != domain.AuthAssuranceSubstantial ||
		d.ACRVerificado != ACRCertificadoPersonalDesarrolloProtegido ||
		!referenciaOpacaSesionValida(d.PoliticaGarantiaRef, "pga_") ||
		!huellaSHA256SesionValida(d.PoliticaGarantiaHuellaSHA256) ||
		!instanteSesionCanonico(d.AsercionActualEmitidaEn) ||
		!instanteSesionCanonico(d.AsercionActualExpiraEn) ||
		!instanteSesionCanonico(d.CertificadoVerificadoEn) ||
		!instanteSesionCanonico(d.CertificadoValidoHasta) ||
		!instanteSesionCanonico(d.CAValidaHasta) ||
		!instanteSesionCanonico(d.CRLSiguienteActualizacion) ||
		!instanteSesionCanonico(d.PoliticaRetiradaEn) ||
		!d.AsercionActualEmitidaEn.Before(d.AsercionActualExpiraEn) {
		return ErrPresentacionCertificadoNoValida
	}
	if normalizada, err := normalizarHuellaCertificado(d.CertificadoSHA256); err != nil || normalizada != d.CertificadoSHA256 {
		return ErrPresentacionCertificadoNoValida
	}
	if normalizada, err := normalizarHuellaCertificado(d.CASHA256); err != nil || normalizada != d.CASHA256 {
		return ErrPresentacionCertificadoNoValida
	}
	return nil
}

func validarResultadoPresentacionCertificado(r ResultadoRegistroPresentacionCertificado, d DatosPresentacionCertificado, estado estadoIdentidadSesion, inicio bool, ahora time.Time) error {
	a := r.SesionOriginal
	c := r.Recibo
	if a.Validar() != nil || !referenciaOpacaSesionValida(c.PresentacionRef, "prs_") ||
		c.OperacionRef != d.OperacionRef || c.Generacion == 0 || c.SesionGeneracion == 0 ||
		!huellaSHA256SesionValida(c.HuellaSHA256) ||
		!instanteSesionCanonico(c.RegistradaEn) || !instanteSesionCanonico(c.ValidaHasta) ||
		c.CanalSHA256 != d.CanalSHA256 || c.AsercionActualHuellaSHA256 != d.AsercionActualHuellaSHA256 ||
		!c.ValidaHasta.After(c.RegistradaEn) || c.ValidaHasta.After(c.RegistradaEn.Add(5*time.Minute)) ||
		ahora.Before(c.RegistradaEn) || c.RegistradaEn.Before(d.AsercionActualEmitidaEn) ||
		c.RegistradaEn.Before(d.CertificadoVerificadoEn) ||
		c.ValidaHasta.After(a.SesionValidaHasta) || c.ValidaHasta.After(d.AsercionActualExpiraEn) ||
		c.ValidaHasta.After(d.CertificadoValidoHasta) || c.ValidaHasta.After(d.CAValidaHasta) ||
		c.ValidaHasta.After(d.CRLSiguienteActualizacion) || c.ValidaHasta.After(d.PoliticaRetiradaEn) ||
		!ahora.Before(c.ValidaHasta) ||
		a.CuentaPrivilegiada || a.CuentaRef != a.CuentaOrdinariaRef ||
		a.Superficie != domain.SuperficieAutenticacionInternaCorporativaV1 ||
		a.MetodoObservado != d.MetodoObservado || a.GarantiaObservada != d.GarantiaObservada ||
		a.PoliticaGarantiaRef != d.PoliticaGarantiaRef ||
		a.PoliticaGarantiaHuellaSHA256 != d.PoliticaGarantiaHuellaSHA256 {
		return ErrPresentacionCertificadoNoValida
	}
	if (!inicio && c.ModoInicio != "reanudada") || (inicio && c.ModoInicio != "abierta" && c.ModoInicio != "reanudada" && c.ModoInicio != "renovada") {
		return ErrPresentacionCertificadoNoValida
	}
	if (c.ModoInicio == "abierta" || c.ModoInicio == "renovada") && (a.AutenticacionHuellaSHA256 != d.AsercionActualHuellaSHA256 || !a.SesionEmitidaEn.Equal(estado.emitidaEn) || !a.AutenticacionVerificadaEn.Equal(estado.autenticacionVerificadaEn)) {
		return ErrPresentacionCertificadoNoValida
	}
	return nil
}
