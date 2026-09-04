package bootstrap

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"vec-diputacion-granada/config"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var ErrMaterialDesarrolloInvalido = errors.New("bootstrap: material criptografico de desarrollo invalido")

const tamanoMaximoFicheroMaterialDesarrollo = 256 << 10

type materialSeguridadDesarrollo struct {
	configuracionTLS             *tls.Config
	identidad                    *resolvedorIdentidadDesarrollo
	claveKMS                     [sha256.Size]byte
	firmaAtestacionKMS           ed25519.PrivateKey
	verificadorAtestacionKMS     ed25519.PublicKey
	huellaPublicaAtestacionKMS   [sha256.Size]byte
	firmaRevalidacionKMS         ed25519.PrivateKey
	verificadorRevalidacionKMS   ed25519.PublicKey
	huellaPublicaRevalidacionKMS [sha256.Size]byte
	claveTSA                     [sha256.Size]byte
	idempotencia                 materialIdempotenciaDesarrollo
}

type archivoIdentidadDesarrollo struct {
	Version           int      `json:"version"`
	Autoridad         string   `json:"autoridad"`
	CertificateSHA256 string   `json:"certificate_sha256"`
	Subject           string   `json:"subject"`
	DisplayName       string   `json:"display_name"`
	Roles             []string `json:"roles"`
}

type archivoManifiestoDesarrollo struct {
	Version                            int               `json:"version"`
	Perfil                             string            `json:"perfil"`
	Autoridad                          string            `json:"autoridad"`
	MigrableAProduccion                bool              `json:"migrable_a_produccion"`
	HuellaCASHA256                     string            `json:"huella_ca_sha256"`
	HuellaServidorSHA256               string            `json:"huella_servidor_sha256"`
	HuellaClienteSHA256                string            `json:"huella_cliente_sha256"`
	HuellaIntervencionSHA256           string            `json:"huella_intervencion_sha256"`
	HuellaPublicaAtestacionKMSSHA256   string            `json:"huella_publica_atestacion_kms_sha256"`
	HuellaPublicaRevalidacionKMSSHA256 string            `json:"huella_publica_revalidacion_kms_sha256"`
	Proveedores                        map[string]string `json:"proveedores"`
}

func cargarMaterialSeguridadDesarrollo(cfg config.Config) (materialSeguridadDesarrollo, error) {
	cfg = cfg.Normalize()
	if !cfg.DevelopmentEnabledByDoubleKey() || !filepath.IsAbs(cfg.DevelopmentMaterialDir) {
		return materialSeguridadDesarrollo{}, ErrActivacionDesarrolloInvalida
	}
	evaluada, err := filepath.EvalSymlinks(cfg.DevelopmentMaterialDir)
	if err != nil || evaluada != filepath.Clean(cfg.DevelopmentMaterialDir) {
		return materialSeguridadDesarrollo{}, fmt.Errorf("%w: ruta enlazada o no canonica", ErrMaterialDesarrolloInvalido)
	}
	if dentroDeRepositorioGit(cfg.DevelopmentMaterialDir) {
		return materialSeguridadDesarrollo{}, fmt.Errorf("%w: el directorio pertenece al repositorio", ErrMaterialDesarrolloInvalido)
	}
	if err := validarArbolMaterialDesarrollo(cfg.DevelopmentMaterialDir); err != nil {
		return materialSeguridadDesarrollo{}, err
	}
	rutas := cfg.DevelopmentPaths()
	for _, ruta := range []string{
		rutas.CACertificate, rutas.CAPrivateKey, rutas.ServerCertificate, rutas.ServerPrivateKey,
		rutas.ClientCertificate, rutas.ClientPrivateKey, rutas.KMSSecret,
		rutas.IntervencionCertificate, rutas.IntervencionPrivateKey, rutas.IntervencionIdentity,
		rutas.KMSAttestationKey, rutas.KMSAttestationPublic, rutas.TSASecret, rutas.Identity,
		rutas.KMSRevalidationKey, rutas.KMSRevalidationPublic,
		rutas.IdempotencyHMACConfig,
		filepath.Join(cfg.DevelopmentMaterialDir, "ca", "serie"),
		filepath.Join(cfg.DevelopmentMaterialDir, "manifiesto.json"),
		filepath.Join(cfg.DevelopmentMaterialDir, "desarrollo.env"),
	} {
		if err := validarFicheroMaterialPresente(ruta); err != nil {
			return materialSeguridadDesarrollo{}, err
		}
	}
	if cfg.TLSCertFile != rutas.ServerCertificate || cfg.TLSKeyFile != rutas.ServerPrivateKey {
		return materialSeguridadDesarrollo{}, fmt.Errorf("%w: TLS no corresponde al material del perfil", ErrMaterialDesarrolloInvalido)
	}

	caPEM, err := leerFicheroMaterialSeguro(rutas.CACertificate, tamanoMaximoFicheroMaterialDesarrollo)
	if err != nil {
		return materialSeguridadDesarrollo{}, err
	}
	servidorPEM, err := leerFicheroMaterialSeguro(rutas.ServerCertificate, tamanoMaximoFicheroMaterialDesarrollo)
	if err != nil {
		return materialSeguridadDesarrollo{}, err
	}
	servidorClavePEM, err := leerFicheroMaterialSeguro(rutas.ServerPrivateKey, tamanoMaximoFicheroMaterialDesarrollo)
	if err != nil {
		return materialSeguridadDesarrollo{}, err
	}
	clientePEM, err := leerFicheroMaterialSeguro(rutas.ClientCertificate, tamanoMaximoFicheroMaterialDesarrollo)
	if err != nil {
		return materialSeguridadDesarrollo{}, err
	}
	intervencionPEM, err := leerFicheroMaterialSeguro(
		rutas.IntervencionCertificate,
		tamanoMaximoFicheroMaterialDesarrollo,
	)
	if err != nil {
		return materialSeguridadDesarrollo{}, err
	}
	defer borrarBytes(servidorClavePEM)

	ca, err := decodificarCertificadoUnico(caPEM)
	if err != nil || !ca.IsCA || ca.KeyUsage&x509.KeyUsageCertSign == 0 {
		return materialSeguridadDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	parServidor, err := tls.X509KeyPair(servidorPEM, servidorClavePEM)
	if err != nil {
		return materialSeguridadDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	certificadoServidor, err := decodificarCertificadoUnico(servidorPEM)
	if err != nil {
		return materialSeguridadDesarrollo{}, err
	}
	parServidor.Leaf = certificadoServidor
	certificadoCliente, err := decodificarCertificadoUnico(clientePEM)
	if err != nil {
		return materialSeguridadDesarrollo{}, err
	}
	certificadoIntervencion, err := decodificarCertificadoUnico(intervencionPEM)
	if err != nil {
		return materialSeguridadDesarrollo{}, err
	}

	raices := x509.NewCertPool()
	raices.AddCert(ca)
	if _, err := certificadoServidor.Verify(x509.VerifyOptions{
		DNSName: "localhost", Roots: raices, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}); err != nil {
		return materialSeguridadDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	if _, err := certificadoCliente.Verify(x509.VerifyOptions{
		Roots: raices, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}); err != nil {
		return materialSeguridadDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	if _, err := certificadoIntervencion.Verify(x509.VerifyOptions{
		Roots: raices, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}); err != nil {
		return materialSeguridadDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	claveKMS, err := leerSecreto32Desarrollo(rutas.KMSSecret)
	if err != nil {
		return materialSeguridadDesarrollo{}, err
	}
	defer borrarBytes(claveKMS[:])
	claveTSA, err := leerSecreto32Desarrollo(rutas.TSASecret)
	if err != nil {
		return materialSeguridadDesarrollo{}, err
	}
	defer borrarBytes(claveTSA[:])
	materialIdempotencia, err := cargarMaterialIdempotenciaDesarrollo(
		cfg.DevelopmentMaterialDir, rutas.IdempotencyHMACConfig,
	)
	if err != nil {
		return materialSeguridadDesarrollo{}, err
	}
	materialIdempotenciaEntregado := false
	defer func() {
		if !materialIdempotenciaEntregado {
			materialIdempotencia.borrar()
		}
	}()
	if !materialIdempotencia.separadoDe(&claveKMS, &claveTSA) {
		return materialSeguridadDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	identidadRRHH, err := cargarIdentidadDesarrollo(
		rutas.Identity,
		certificadoCliente,
		"tecnico_rrhh",
	)
	if err != nil {
		return materialSeguridadDesarrollo{}, err
	}
	identidadIntervencion, err := cargarIdentidadDesarrollo(
		rutas.IntervencionIdentity,
		certificadoIntervencion,
		"intervencion",
	)
	if err != nil {
		return materialSeguridadDesarrollo{}, err
	}
	identidad, err := nuevoResolvedorIdentidadDesarrollo(
		identidadRRHH,
		identidadIntervencion,
	)
	if err != nil {
		return materialSeguridadDesarrollo{}, err
	}

	firmaAtestacionKMS, verificadorAtestacionKMS, huellaPublicaAtestacionKMS, err := cargarFirmaKMSDesarrollo(
		rutas.KMSAttestationKey, rutas.KMSAttestationPublic,
	)
	if err != nil {
		return materialSeguridadDesarrollo{}, err
	}
	defer borrarBytes(firmaAtestacionKMS)
	firmaRevalidacionKMS, verificadorRevalidacionKMS, huellaPublicaRevalidacionKMS, err := cargarFirmaKMSDesarrollo(
		rutas.KMSRevalidationKey, rutas.KMSRevalidationPublic,
	)
	if err != nil {
		return materialSeguridadDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	defer borrarBytes(firmaRevalidacionKMS)
	if subtle.ConstantTimeCompare(huellaPublicaAtestacionKMS[:], huellaPublicaRevalidacionKMS[:]) == 1 {
		return materialSeguridadDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	if err := validarManifiestoDesarrollo(
		filepath.Join(cfg.DevelopmentMaterialDir, "manifiesto.json"), ca, certificadoServidor,
		certificadoCliente, certificadoIntervencion,
		huellaPublicaAtestacionKMS, huellaPublicaRevalidacionKMS,
	); err != nil {
		return materialSeguridadDesarrollo{}, err
	}

	resultado := materialSeguridadDesarrollo{
		configuracionTLS: &tls.Config{
			Certificates: []tls.Certificate{parServidor},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    raices,
			MinVersion:   tls.VersionTLS13,
			MaxVersion:   tls.VersionTLS13,
		},
		identidad:                    identidad,
		claveKMS:                     claveKMS,
		firmaAtestacionKMS:           append(ed25519.PrivateKey(nil), firmaAtestacionKMS...),
		verificadorAtestacionKMS:     append(ed25519.PublicKey(nil), verificadorAtestacionKMS...),
		huellaPublicaAtestacionKMS:   huellaPublicaAtestacionKMS,
		firmaRevalidacionKMS:         append(ed25519.PrivateKey(nil), firmaRevalidacionKMS...),
		verificadorRevalidacionKMS:   append(ed25519.PublicKey(nil), verificadorRevalidacionKMS...),
		huellaPublicaRevalidacionKMS: huellaPublicaRevalidacionKMS,
		claveTSA:                     claveTSA,
		idempotencia:                 materialIdempotencia,
	}
	materialIdempotenciaEntregado = true
	return resultado, nil
}

func validarFicheroMaterialPresente(ruta string) error {
	informacion, err := os.Lstat(ruta)
	if err != nil || !informacion.Mode().IsRegular() || informacion.Mode()&os.ModeSymlink != 0 ||
		informacion.Mode().Perm()&0o077 != 0 || informacion.Size() < 1 {
		return ErrMaterialDesarrolloInvalido
	}
	return nil
}

func validarArbolMaterialDesarrollo(raiz string) error {
	informacion, err := os.Lstat(raiz)
	if err != nil || !informacion.IsDir() || informacion.Mode()&os.ModeSymlink != 0 ||
		informacion.Mode().Perm()&0o077 != 0 {
		return ErrMaterialDesarrolloInvalido
	}
	uid := uint32(os.Geteuid())
	err = filepath.WalkDir(raiz, func(ruta string, entrada fs.DirEntry, err error) error {
		if err != nil || entrada == nil || entrada.Type()&os.ModeSymlink != 0 {
			return ErrMaterialDesarrolloInvalido
		}
		info, err := entrada.Info()
		if err != nil || info.Mode().Perm()&0o077 != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
			return ErrMaterialDesarrolloInvalido
		}
		if estado, ok := info.Sys().(*syscall.Stat_t); !ok || estado.Uid != uid {
			return ErrMaterialDesarrolloInvalido
		}
		return nil
	})
	if err != nil {
		return errors.Join(ErrMaterialDesarrolloInvalido, err)
	}
	return nil
}

func leerFicheroMaterialSeguro(ruta string, limite int64) ([]byte, error) {
	inicial, err := os.Lstat(ruta)
	if err != nil || !inicial.Mode().IsRegular() || inicial.Mode()&os.ModeSymlink != 0 ||
		inicial.Mode().Perm()&0o077 != 0 || inicial.Size() < 1 || inicial.Size() > limite {
		return nil, ErrMaterialDesarrolloInvalido
	}
	fichero, err := os.Open(ruta)
	if err != nil {
		return nil, ErrMaterialDesarrolloInvalido
	}
	defer fichero.Close()
	abierto, err := fichero.Stat()
	if err != nil || !os.SameFile(inicial, abierto) {
		return nil, ErrMaterialDesarrolloInvalido
	}
	contenido, err := io.ReadAll(io.LimitReader(fichero, limite+1))
	if err != nil || len(contenido) < 1 || int64(len(contenido)) > limite || int64(len(contenido)) != abierto.Size() {
		return nil, ErrMaterialDesarrolloInvalido
	}
	return contenido, nil
}

func leerSecreto32Desarrollo(ruta string) ([sha256.Size]byte, error) {
	contenido, err := leerFicheroMaterialSeguro(ruta, sha256.Size)
	if err != nil || len(contenido) != sha256.Size {
		borrarBytes(contenido)
		return [sha256.Size]byte{}, ErrMaterialDesarrolloInvalido
	}
	var resultado [sha256.Size]byte
	copy(resultado[:], contenido)
	borrarBytes(contenido)
	return resultado, nil
}

func decodificarCertificadoUnico(contenido []byte) (*x509.Certificate, error) {
	bloque, resto := pem.Decode(contenido)
	if bloque == nil || bloque.Type != "CERTIFICATE" || len(bytes.TrimSpace(resto)) != 0 {
		return nil, ErrMaterialDesarrolloInvalido
	}
	certificado, err := x509.ParseCertificate(bloque.Bytes)
	if err != nil {
		return nil, ErrMaterialDesarrolloInvalido
	}
	return certificado, nil
}

func cargarIdentidadDesarrollo(
	ruta string,
	certificado *x509.Certificate,
	rolEsperado string,
) (identidadCertificadoDesarrollo, error) {
	contenido, err := leerFicheroMaterialSeguro(ruta, 64<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return identidadCertificadoDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var archivo archivoIdentidadDesarrollo
	if err := decodificador.Decode(&archivo); err != nil {
		return identidadCertificadoDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	var sobrante any
	if err := decodificador.Decode(&sobrante); !errors.Is(err, io.EOF) {
		return identidadCertificadoDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	huella := sha256.Sum256(certificado.Raw)
	huellaEsperada, err := hex.DecodeString(archivo.CertificateSHA256)
	if err != nil || len(huellaEsperada) != sha256.Size || subtle.ConstantTimeCompare(huella[:], huellaEsperada) != 1 ||
		archivo.Version != 1 || archivo.Autoridad != AutoridadNoAutoritativa || len(archivo.Roles) != 1 ||
		archivo.Roles[0] != rolEsperado {
		return identidadCertificadoDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	principal := vecdomain.Principal{
		ID: archivo.Subject, DisplayName: archivo.DisplayName,
		Roles:      append([]string(nil), archivo.Roles...),
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{
			"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": hex.EncodeToString(huella[:]),
		},
	}
	if principal.Validate() != nil {
		return identidadCertificadoDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	return identidadCertificadoDesarrollo{
		huella: huella, principal: principal,
	}, nil
}

func validarManifiestoDesarrollo(
	ruta string,
	ca, servidor, cliente, intervencion *x509.Certificate,
	huellaPublicaAtestacionKMS, huellaPublicaRevalidacionKMS [sha256.Size]byte,
) error {
	contenido, err := leerFicheroMaterialSeguro(ruta, 64<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return ErrMaterialDesarrolloInvalido
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var manifiesto archivoManifiestoDesarrollo
	if err := decodificador.Decode(&manifiesto); err != nil {
		return ErrMaterialDesarrolloInvalido
	}
	var sobrante any
	if err := decodificador.Decode(&sobrante); !errors.Is(err, io.EOF) {
		return ErrMaterialDesarrolloInvalido
	}
	huellas := []struct {
		declarada string
		cert      *x509.Certificate
	}{
		{manifiesto.HuellaCASHA256, ca},
		{manifiesto.HuellaServidorSHA256, servidor},
		{manifiesto.HuellaClienteSHA256, cliente},
		{manifiesto.HuellaIntervencionSHA256, intervencion},
	}
	for _, dato := range huellas {
		huella := sha256.Sum256(dato.cert.Raw)
		declarada, err := hex.DecodeString(strings.ToLower(dato.declarada))
		if err != nil || len(declarada) != sha256.Size || subtle.ConstantTimeCompare(declarada, huella[:]) != 1 {
			return ErrMaterialDesarrolloInvalido
		}
	}
	declaradaPublicaAtestacionKMS, err := hex.DecodeString(strings.ToLower(manifiesto.HuellaPublicaAtestacionKMSSHA256))
	if err != nil || len(declaradaPublicaAtestacionKMS) != sha256.Size ||
		subtle.ConstantTimeCompare(declaradaPublicaAtestacionKMS, huellaPublicaAtestacionKMS[:]) != 1 {
		return ErrMaterialDesarrolloInvalido
	}
	declaradaPublicaRevalidacionKMS, err := hex.DecodeString(strings.ToLower(manifiesto.HuellaPublicaRevalidacionKMSSHA256))
	if err != nil || len(declaradaPublicaRevalidacionKMS) != sha256.Size ||
		subtle.ConstantTimeCompare(declaradaPublicaRevalidacionKMS, huellaPublicaRevalidacionKMS[:]) != 1 {
		return ErrMaterialDesarrolloInvalido
	}
	esperados := map[string]string{
		"identidad":              "identidad-mtls-local-v1",
		"idempotencia_hmac":      referenciaProveedorIdempotenciaDesarrollo,
		"kms_emisor":             "kms-emisor-fichero-local-v2",
		"kms_revalidador":        "kms-revalidador-ed25519-local-v1",
		"kms_verificador_recibo": "kms-verificador-publico-local-v1",
		"tsa":                    "tsa-determinista-local-v1", "tls": "tls-ca-local-v1",
	}
	if manifiesto.Version != 4 || manifiesto.Perfil != config.ExecutionProfileDevelopment ||
		manifiesto.Autoridad != AutoridadNoAutoritativa || manifiesto.MigrableAProduccion ||
		len(manifiesto.Proveedores) != len(esperados) {
		return ErrMaterialDesarrolloInvalido
	}
	for tipo, referencia := range esperados {
		if manifiesto.Proveedores[tipo] != referencia {
			return ErrMaterialDesarrolloInvalido
		}
	}
	return nil
}

func cargarFirmaKMSDesarrollo(
	rutaPrivada, rutaPublica string,
) (ed25519.PrivateKey, ed25519.PublicKey, [sha256.Size]byte, error) {
	privadaPEM, err := leerFicheroMaterialSeguro(rutaPrivada, 64<<10)
	if err != nil {
		return nil, nil, [sha256.Size]byte{}, err
	}
	defer borrarBytes(privadaPEM)
	publicaPEM, err := leerFicheroMaterialSeguro(rutaPublica, 64<<10)
	if err != nil {
		return nil, nil, [sha256.Size]byte{}, err
	}
	bloquePrivado, resto := pem.Decode(privadaPEM)
	if bloquePrivado == nil || bloquePrivado.Type != "PRIVATE KEY" || len(bytes.TrimSpace(resto)) != 0 {
		return nil, nil, [sha256.Size]byte{}, ErrMaterialDesarrolloInvalido
	}
	valorPrivado, err := x509.ParsePKCS8PrivateKey(bloquePrivado.Bytes)
	privada, valida := valorPrivado.(ed25519.PrivateKey)
	if err != nil || !valida || len(privada) != ed25519.PrivateKeySize {
		return nil, nil, [sha256.Size]byte{}, ErrMaterialDesarrolloInvalido
	}
	defer borrarBytes(privada)
	bloquePublico, resto := pem.Decode(publicaPEM)
	if bloquePublico == nil || bloquePublico.Type != "PUBLIC KEY" || len(bytes.TrimSpace(resto)) != 0 {
		return nil, nil, [sha256.Size]byte{}, ErrMaterialDesarrolloInvalido
	}
	valorPublico, err := x509.ParsePKIXPublicKey(bloquePublico.Bytes)
	publica, valida := valorPublico.(ed25519.PublicKey)
	if err != nil || !valida || len(publica) != ed25519.PublicKeySize ||
		subtle.ConstantTimeCompare(privada.Public().(ed25519.PublicKey), publica) != 1 {
		return nil, nil, [sha256.Size]byte{}, ErrMaterialDesarrolloInvalido
	}
	huella := sha256.Sum256(bloquePublico.Bytes)
	return append(ed25519.PrivateKey(nil), privada...), append(ed25519.PublicKey(nil), publica...), huella, nil
}

func borrarBytes(contenido []byte) {
	for indice := range contenido {
		contenido[indice] = 0
	}
}

func dentroDeRepositorioGit(ruta string) bool {
	actual := filepath.Clean(ruta)
	for {
		if _, err := os.Stat(filepath.Join(actual, ".git")); err == nil {
			return true
		}
		padre := filepath.Dir(actual)
		if padre == actual {
			return false
		}
		actual = padre
	}
}

// rechazarTLSDesarrolloEnProduccion inspecciona el proveedor TLS realmente
// configurado, no un descriptor aportado por quien compone. El certificado
// generado por T21 conserva marcas X.509 estructurales aunque se copie o se
// renombre, por lo que nunca puede entrar por las variables TLS productivas.
func rechazarTLSDesarrolloEnProduccion(cfg config.Config) error {
	cfg = cfg.Normalize()
	if cfg.ExecutionProfile == config.ExecutionProfileDevelopment || cfg.TLSCertFile == "" {
		return nil
	}
	contenido, err := leerCertificadoParaClasificacion(cfg.TLSCertFile)
	if err != nil {
		// La carga autoritativa de net/http rechazará después un fichero ausente
		// o inválido. Aquí sólo clasificamos material T21 reconocible.
		return nil
	}
	certificado, err := decodificarPrimerCertificado(contenido)
	if err != nil {
		return nil
	}
	if nombreX509Contiene(certificado.Subject.Organization, "VEC Desarrollo") ||
		nombreX509Contiene(certificado.Subject.OrganizationalUnit, "NO AUTORITATIVO") ||
		nombreX509Contiene(certificado.Issuer.Organization, "VEC Desarrollo") ||
		nombreX509Contiene(certificado.Issuer.OrganizationalUnit, "NO AUTORITATIVO") {
		return ErrProveedorDesarrolloEnProduccion
	}
	return nil
}

func leerCertificadoParaClasificacion(ruta string) ([]byte, error) {
	fichero, err := os.Open(ruta)
	if err != nil {
		return nil, err
	}
	defer fichero.Close()
	contenido, err := io.ReadAll(io.LimitReader(fichero, tamanoMaximoFicheroMaterialDesarrollo+1))
	if err != nil || len(contenido) == 0 || len(contenido) > tamanoMaximoFicheroMaterialDesarrollo {
		return nil, ErrMaterialDesarrolloInvalido
	}
	return contenido, nil
}

func decodificarPrimerCertificado(contenido []byte) (*x509.Certificate, error) {
	bloque, _ := pem.Decode(contenido)
	if bloque == nil || bloque.Type != "CERTIFICATE" {
		return nil, ErrMaterialDesarrolloInvalido
	}
	certificado, err := x509.ParseCertificate(bloque.Bytes)
	if err != nil {
		return nil, ErrMaterialDesarrolloInvalido
	}
	return certificado, nil
}

func nombreX509Contiene(valores []string, esperado string) bool {
	for _, valor := range valores {
		if subtle.ConstantTimeCompare([]byte(valor), []byte(esperado)) == 1 {
			return true
		}
	}
	return false
}
