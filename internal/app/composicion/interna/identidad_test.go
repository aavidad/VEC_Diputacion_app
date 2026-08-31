package interna

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/gob"
	"encoding/json"
	"errors"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const sanProxyIdentidadOfflinePrueba = "proxy-interno.identidad.test"

type relojIdentidadOfflinePrueba struct{ ahora time.Time }

func (r *relojIdentidadOfflinePrueba) Ahora() time.Time { return r.ahora }

type verificadorIdentidadOfflinePrueba struct {
	asercion httpseguridad.AsercionProxyIdentidad
	error    error
	llamadas int
}

func (v *verificadorIdentidadOfflinePrueba) Verificar(
	_ context.Context,
	_ []byte,
) (httpseguridad.AsercionProxyIdentidad, error) {
	v.llamadas++
	if v.error != nil {
		return httpseguridad.AsercionProxyIdentidad{}, v.error
	}
	resultado := v.asercion
	resultado.Factores = append([]httpseguridad.FactorAutenticacion(nil), v.asercion.Factores...)
	return resultado, nil
}

type evaluadorIdentidadOfflinePrueba struct {
	resultado httpseguridad.ResultadoEvaluacionGarantia
	error     error
	llamadas  int
}

func (e *evaluadorIdentidadOfflinePrueba) Evaluar(
	_ context.Context,
	_ httpseguridad.EntradaEvaluacionGarantia,
) (httpseguridad.ResultadoEvaluacionGarantia, error) {
	e.llamadas++
	return e.resultado, e.error
}

type registroIdentidadOfflinePrueba struct {
	errorAlta         error
	errorRevalidacion error
	adulterarAlta     func(*httpseguridad.ConfirmacionAltaSesion)
	altas             int
	revalidaciones    int
	confirmacion      httpseguridad.ConfirmacionAltaSesion
}

func (r *registroIdentidadOfflinePrueba) ConsumirAsercionYRegistrar(
	_ context.Context,
	alta httpseguridad.AltaSesionAtomica,
) (httpseguridad.ConfirmacionAltaSesion, error) {
	r.altas++
	if r.errorAlta != nil {
		return httpseguridad.ConfirmacionAltaSesion{}, r.errorAlta
	}
	confirmacion := httpseguridad.ConfirmacionAltaSesion{
		AutenticacionRef:          "aut_0123456789abcdefghijkl",
		AsercionRef:               "ase_0123456789abcdefghijkl",
		SesionRef:                 "ses_0123456789abcdefghijkl",
		ControlSesionRef:          "cse_0123456789abcdefghijkl",
		ControlSesionRevision:     1,
		ControlSesionEstado:       httpseguridad.EstadoControlSesionActiva,
		ControlSesionHuellaSHA256: strings.Repeat("c", 64),
		CuentaRef:                 "cta_0123456789abcdefghijkl",
		CuentaOrdinariaRef:        "cta_0123456789abcdefghijkl",
		SesionRevalidadaEn:        alta.SesionEmitidaEn,
		SesionValidaHasta:         alta.AsercionExpiraEn,
		AltaConfirmada:            alta,
	}
	if r.adulterarAlta != nil {
		r.adulterarAlta(&confirmacion)
	}
	r.confirmacion = confirmacion
	return confirmacion, nil
}

func (r *registroIdentidadOfflinePrueba) ComprobarSesionYCuentaActivas(
	_ context.Context,
	consulta httpseguridad.ConsultaSesionActiva,
) error {
	r.revalidaciones++
	if r.errorRevalidacion != nil {
		return r.errorRevalidacion
	}
	if consulta.Validar() != nil || r.confirmacion.Validar() != nil ||
		consulta.AutenticacionRef != r.confirmacion.AutenticacionRef ||
		consulta.SesionRef != r.confirmacion.SesionRef ||
		consulta.ControlSesionRef != r.confirmacion.ControlSesionRef {
		return errors.New("revalidacion no coincide")
	}
	return nil
}

type contextoIdentidadOfflineNulo struct{}

func (*contextoIdentidadOfflineNulo) Deadline() (time.Time, bool) { return time.Time{}, false }
func (*contextoIdentidadOfflineNulo) Done() <-chan struct{}       { return nil }
func (*contextoIdentidadOfflineNulo) Err() error                  { return nil }
func (*contextoIdentidadOfflineNulo) Value(any) any               { return nil }

type entornoIdentidadOfflinePrueba struct {
	configuracion httpseguridad.ConfiguracionSuperficie
	estadoTLS     tls.ConnectionState
	servicio      *httpseguridad.ServicioIdentidad
	fachada       *FachadaIdentidadOffline
	verificador   *verificadorIdentidadOfflinePrueba
	evaluador     *evaluadorIdentidadOfflinePrueba
	registro      *registroIdentidadOfflinePrueba
	canal         httpseguridad.CanalProxyAutenticado
	ahora         time.Time
}

func TestFachadaIdentidadOfflineEmiteCapsulaRevalidadaYLigada(t *testing.T) {
	estadoTLS := estadoTLSMutuoIdentidadOfflinePrueba(t)
	entorno := nuevoEntornoIdentidadOfflinePrueba(t, estadoTLS)
	protegida := []byte("asercion-corporativa-protegida")

	capsula, err := entorno.fachada.Autenticar(context.Background(), estadoTLS, protegida)
	if err != nil {
		t.Fatalf("autenticar: %v", err)
	}
	if entorno.verificador.llamadas != 1 || entorno.evaluador.llamadas != 2 ||
		entorno.registro.altas != 1 || entorno.registro.revalidaciones != 1 {
		t.Fatalf("secuencia incompleta: verificador=%d evaluador=%d altas=%d revalidaciones=%d",
			entorno.verificador.llamadas, entorno.evaluador.llamadas,
			entorno.registro.altas, entorno.registro.revalidaciones)
	}

	otro := nuevoEntornoIdentidadOfflinePrueba(t, estadoTLS)
	if _, err = otro.servicio.VincularCapsulaIdentidadPeticion(
		context.Background(), capsula, otro.canal,
	); !errors.Is(err, httpseguridad.ErrSesionNoValida) {
		t.Fatalf("capsula cruzada entre instancias admitida: %v", err)
	}
	otroCanal, err := entorno.servicio.AutenticarCanalTLSMutuo(
		estadoTLSMutuoIdentidadOfflinePrueba(t),
	)
	if err != nil {
		t.Fatalf("segundo canal real: %v", err)
	}
	if _, err = entorno.servicio.VincularCapsulaIdentidadPeticion(
		context.Background(), capsula, otroCanal,
	); !errors.Is(err, httpseguridad.ErrSesionNoValida) {
		t.Fatalf("capsula cruzada entre canales admitida: %v", err)
	}

	ctx, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
		context.Background(), capsula, entorno.canal,
	)
	if err != nil {
		t.Fatalf("vincular capsula: %v", err)
	}
	cuenta, auditoria, err := entorno.servicio.ExtraerCapsulaIdentidadPeticion(ctx)
	if err != nil || cuenta.Validar() != nil ||
		cuenta.Garantia != dominiovec.AuthAssuranceHigh ||
		auditoria.CanalVinculadoRef() != entorno.canal.ReferenciaVinculacion() {
		t.Fatalf("extraer identidad ligada: cuenta=%+v auditoria=%v error=%v", cuenta, auditoria, err)
	}
}

func TestFachadaIdentidadOfflineRechazaFronterasAdversas(t *testing.T) {
	estadoTLS := estadoTLSMutuoIdentidadOfflinePrueba(t)
	casos := []struct {
		nombre               string
		mutar                func(*entornoIdentidadOfflinePrueba)
		altasEsperadas       int
		revalidacionesEspera int
	}{
		{"emisor", func(e *entornoIdentidadOfflinePrueba) { e.verificador.asercion.Emisor = "https://emisor.ajeno.test" }, 0, 0},
		{"audiencia", func(e *entornoIdentidadOfflinePrueba) { e.verificador.asercion.Audiencia = "vec-ajena" }, 0, 0},
		{"superficie", func(e *entornoIdentidadOfflinePrueba) {
			e.verificador.asercion.Superficie = httpseguridad.SuperficieExternaPersonal
		}, 0, 0},
		{"canal", func(e *entornoIdentidadOfflinePrueba) {
			e.verificador.asercion.CanalVinculadoRef = "tls-exportador:sha256:" + strings.Repeat("d", 64)
		}, 0, 0},
		{"factor", func(e *entornoIdentidadOfflinePrueba) {
			e.verificador.asercion.Factores = e.verificador.asercion.Factores[:1]
		}, 0, 0},
		{"grupo criptografico", func(e *entornoIdentidadOfflinePrueba) {
			e.verificador.asercion.Factores[1].GrupoCriptograficoRef = e.verificador.asercion.Factores[0].GrupoCriptograficoRef
		}, 0, 0},
		{"garantia", func(e *entornoIdentidadOfflinePrueba) {
			e.evaluador.resultado.Garantia = dominiovec.AuthAssuranceSubstantial
		}, 0, 0},
		{"alta durable", func(e *entornoIdentidadOfflinePrueba) { e.registro.errorAlta = errors.New("alta rechazada") }, 1, 0},
		{"confirmacion adulterada", func(e *entornoIdentidadOfflinePrueba) {
			e.registro.adulterarAlta = func(c *httpseguridad.ConfirmacionAltaSesion) { c.ControlSesionRevision = 2 }
		}, 1, 0},
		{"revalidacion", func(e *entornoIdentidadOfflinePrueba) { e.registro.errorRevalidacion = errors.New("sesion revocada") }, 1, 1},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := nuevoEntornoIdentidadOfflinePrueba(t, estadoTLS)
			caso.mutar(entorno)
			capsula, err := entorno.fachada.Autenticar(
				context.Background(), estadoTLS, []byte("asercion-adversa"),
			)
			if err == nil {
				t.Fatal("frontera adversa aceptada")
			}
			if entorno.registro.altas != caso.altasEsperadas ||
				entorno.registro.revalidaciones != caso.revalidacionesEspera {
				t.Fatalf("efectos inesperados: altas=%d revalidaciones=%d",
					entorno.registro.altas, entorno.registro.revalidaciones)
			}
			if _, errVinculo := entorno.servicio.VincularCapsulaIdentidadPeticion(
				context.Background(), capsula, entorno.canal,
			); errVinculo == nil {
				t.Fatal("el rechazo devolvio una capsula utilizable")
			}
		})
	}
}

func TestFachadaIdentidadOfflineCierraNulosTLSYSuperficieAjena(t *testing.T) {
	var servicioNulo *httpseguridad.ServicioIdentidad
	if _, err := NuevaFachadaIdentidadOffline(nil); !errors.Is(err, ErrIdentidadOfflineNoDisponible) {
		t.Fatalf("servicio nulo admitido: %v", err)
	}
	if _, err := NuevaFachadaIdentidadOffline(servicioNulo); !errors.Is(err, ErrIdentidadOfflineNoDisponible) {
		t.Fatalf("servicio tipado nulo admitido: %v", err)
	}
	var fachadaNula *FachadaIdentidadOffline
	if _, err := fachadaNula.Autenticar(context.Background(), tls.ConnectionState{}, []byte("x")); !errors.Is(err, ErrIdentidadOfflineNoDisponible) {
		t.Fatalf("fachada nula admitida: %v", err)
	}

	estadoTLS := estadoTLSMutuoIdentidadOfflinePrueba(t)
	entorno := nuevoEntornoIdentidadOfflinePrueba(t, estadoTLS)
	if _, err := entorno.fachada.Autenticar(nil, estadoTLS, []byte("x")); !errors.Is(err, ErrIdentidadOfflineNoDisponible) {
		t.Fatalf("contexto nulo admitido: %v", err)
	}
	var contextoNulo *contextoIdentidadOfflineNulo
	if _, err := entorno.fachada.Autenticar(contextoNulo, estadoTLS, []byte("x")); !errors.Is(err, ErrIdentidadOfflineNoDisponible) {
		t.Fatalf("contexto tipado nulo admitido: %v", err)
	}
	if _, err := entorno.fachada.Autenticar(context.Background(), tls.ConnectionState{}, []byte("x")); !errors.Is(err, httpseguridad.ErrCanalProxyNoAutenticado) {
		t.Fatalf("TLS fabricado admitido: %v", err)
	}
	if _, err := entorno.fachada.Autenticar(
		context.Background(), estadoTLSSinClienteIdentidadOfflinePrueba(t), []byte("x"),
	); !errors.Is(err, httpseguridad.ErrCanalProxyNoAutenticado) {
		t.Fatalf("TLS real sin autenticacion de cliente admitido: %v", err)
	}
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := entorno.fachada.Autenticar(ctxCancelado, estadoTLS, []byte("x")); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion perdida: %v", err)
	}
	var protegidaNula []byte
	if _, err := entorno.fachada.Autenticar(context.Background(), estadoTLS, protegidaNula); !errors.Is(err, httpseguridad.ErrAsercionAusente) {
		t.Fatalf("asercion tipada nula admitida: %v", err)
	}

	configuracionExterna := entorno.configuracion
	configuracionExterna.Superficie = httpseguridad.SuperficieExternaPersonal
	configuracionExterna.ZonaRed = httpseguridad.ZonaRedPublica
	configuracionExterna.DireccionEscucha = "127.0.0.1:9443"
	configuracionExterna.RedesPermitidas = []string{"127.0.0.0/8"}
	configuracionExterna.MetodosAdmitidos = []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoCertificado}
	configuracionExterna.FactoresRequeridos = nil
	configuracionExterna.MinimoFactoresVerificados = 1
	configuracionExterna.MinimoGruposCriptograficosDistintos = 1
	configuracionExterna.GarantiaMinima = dominiovec.AuthAssuranceSubstantial
	servicioExterno := debeServicioIdentidadOfflinePrueba(
		t, configuracionExterna, &verificadorIdentidadOfflinePrueba{},
		&evaluadorIdentidadOfflinePrueba{}, &registroIdentidadOfflinePrueba{}, entorno.ahora,
	)
	fachadaExterna, err := NuevaFachadaIdentidadOffline(servicioExterno)
	if err != nil {
		t.Fatalf("crear fachada externa para rechazo: %v", err)
	}
	if _, err = fachadaExterna.Autenticar(context.Background(), estadoTLS, []byte("x")); !errors.Is(err, httpseguridad.ErrCanalProxyNoAutenticado) {
		t.Fatalf("superficie externa admitida: %v", err)
	}
}

func TestCapsulaEmitidaPorFachadaIdentidadOfflineNoSeSerializa(t *testing.T) {
	estadoTLS := estadoTLSMutuoIdentidadOfflinePrueba(t)
	entorno := nuevoEntornoIdentidadOfflinePrueba(t, estadoTLS)
	capsula, err := entorno.fachada.Autenticar(
		context.Background(), estadoTLS, []byte("material-que-no-debe-serializarse"),
	)
	if err != nil {
		t.Fatalf("autenticar: %v", err)
	}
	if _, err = json.Marshal(capsula); !errors.Is(err, httpseguridad.ErrIdentidadNoSerializable) {
		t.Fatalf("JSON admitido: %v", err)
	}
	if _, err = capsula.MarshalText(); !errors.Is(err, httpseguridad.ErrIdentidadNoSerializable) {
		t.Fatalf("texto admitido: %v", err)
	}
	if _, err = capsula.MarshalBinary(); !errors.Is(err, httpseguridad.ErrIdentidadNoSerializable) {
		t.Fatalf("binario admitido: %v", err)
	}
	var salida bytes.Buffer
	if err = gob.NewEncoder(&salida).Encode(capsula); !errors.Is(err, httpseguridad.ErrIdentidadNoSerializable) {
		t.Fatalf("gob admitido: %v", err)
	}
}

func nuevoEntornoIdentidadOfflinePrueba(
	t *testing.T,
	estadoTLS tls.ConnectionState,
) *entornoIdentidadOfflinePrueba {
	t.Helper()
	ahora := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	configuracion := configuracionIdentidadOfflinePrueba()
	verificador := &verificadorIdentidadOfflinePrueba{}
	evaluador := &evaluadorIdentidadOfflinePrueba{resultado: httpseguridad.ResultadoEvaluacionGarantia{
		Garantia:       dominiovec.AuthAssuranceHigh,
		PoliticaRef:    "pga_0123456789abcdefghijkl",
		HuellaPolitica: "sha256:" + strings.Repeat("a", 64),
	}}
	registro := &registroIdentidadOfflinePrueba{}
	servicio := debeServicioIdentidadOfflinePrueba(
		t, configuracion, verificador, evaluador, registro, ahora,
	)
	canal, err := servicio.AutenticarCanalTLSMutuo(estadoTLS)
	if err != nil {
		t.Fatalf("autenticar canal de prueba: %v", err)
	}
	verificador.asercion = asercionIdentidadOfflinePrueba(ahora, configuracion, canal)
	fachada, err := NuevaFachadaIdentidadOffline(servicio)
	if err != nil {
		t.Fatalf("crear fachada: %v", err)
	}
	return &entornoIdentidadOfflinePrueba{
		configuracion: configuracion, estadoTLS: estadoTLS, servicio: servicio,
		fachada: fachada, verificador: verificador, evaluador: evaluador,
		registro: registro, canal: canal, ahora: ahora,
	}
}

func debeServicioIdentidadOfflinePrueba(
	t *testing.T,
	configuracion httpseguridad.ConfiguracionSuperficie,
	verificador httpseguridad.VerificadorAsercionProtegida,
	evaluador httpseguridad.EvaluadorGarantia,
	registro httpseguridad.RegistroSesiones,
	ahora time.Time,
) *httpseguridad.ServicioIdentidad {
	t.Helper()
	servicio, err := httpseguridad.NuevoServicioIdentidad(
		configuracion, verificador, evaluador, registro,
		&relojIdentidadOfflinePrueba{ahora: ahora},
	)
	if err != nil {
		t.Fatalf("crear servicio de identidad: %v", err)
	}
	return servicio
}

func configuracionIdentidadOfflinePrueba() httpseguridad.ConfiguracionSuperficie {
	return httpseguridad.ConfiguracionSuperficie{
		Superficie:                          httpseguridad.SuperficieInternaCorporativa,
		ZonaRed:                             httpseguridad.ZonaRedInterna,
		DireccionEscucha:                    "127.0.0.1:8443",
		Audiencia:                           "vec-interna",
		EmisorIdentidad:                     "https://idp.identidad.test",
		RedesPermitidas:                     []string{"127.0.0.0/8"},
		IdentidadesSANProxyPermitidas:       []string{"dns:" + sanProxyIdentidadOfflinePrueba},
		DuracionMaximaAsercion:              3 * time.Minute,
		EdadMaximaAutenticacion:             15 * time.Minute,
		ToleranciaReloj:                     20 * time.Second,
		MetodosAdmitidos:                    []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoKerberos, httpseguridad.MetodoCertificado},
		FactoresRequeridos:                  []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoKerberos, httpseguridad.MetodoCertificado},
		MinimoFactoresVerificados:           2,
		MinimoGruposCriptograficosDistintos: 2,
		GarantiaMinima:                      dominiovec.AuthAssuranceHigh,
	}
}

func asercionIdentidadOfflinePrueba(
	ahora time.Time,
	configuracion httpseguridad.ConfiguracionSuperficie,
	canal httpseguridad.CanalProxyAutenticado,
) httpseguridad.AsercionProxyIdentidad {
	emitida := ahora.Add(-time.Minute)
	return httpseguridad.AsercionProxyIdentidad{
		ID: "asercion-interna-001", Emisor: configuracion.EmisorIdentidad,
		Audiencia: configuracion.Audiencia, Superficie: configuracion.Superficie,
		SujetoID: "persona-interna-001",
		Cuenta: httpseguridad.CuentaAcceso{
			ID: "cuenta-interna-001", SujetoVinculadoID: "persona-interna-001",
		},
		SesionID: "sesion-interna-001", CanalVinculadoRef: canal.ReferenciaVinculacion(),
		AutenticacionVerificadaEn: emitida, EmitidaEn: emitida, NoAntesDe: emitida,
		ExpiraEn: ahora.Add(time.Minute), MetodoPrimario: httpseguridad.MetodoCertificado,
		ACRVerificado: "urn:vec:acr:alto",
		Factores: []httpseguridad.FactorAutenticacion{
			{
				Metodo: httpseguridad.MetodoKerberos, SujetoVinculadoID: "persona-interna-001",
				Principal: "cuenta-interna@IDENTIDAD.TEST", EvidenciaRef: "krb:ticket:001",
				GrupoCriptograficoRef: "grupo:kerberos:001", VerificadoEn: emitida,
			},
			{
				Metodo: httpseguridad.MetodoCertificado, SujetoVinculadoID: "persona-interna-001",
				CredencialRef: "cert:sha256:001", EvidenciaRef: "cert:validacion:001",
				GrupoCriptograficoRef: "grupo:certificado:001", VerificadoEn: emitida,
			},
		},
	}
}

func estadoTLSMutuoIdentidadOfflinePrueba(t *testing.T) tls.ConnectionState {
	return estadoTLSIdentidadOfflinePrueba(t, true)
}

func estadoTLSSinClienteIdentidadOfflinePrueba(t *testing.T) tls.ConnectionState {
	return estadoTLSIdentidadOfflinePrueba(t, false)
}

func estadoTLSIdentidadOfflinePrueba(t *testing.T, conCliente bool) tls.ConnectionState {
	t.Helper()
	ahora := time.Now()
	_, claveCA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generar CA: %v", err)
	}
	plantillaCA := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CA identidad offline"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), IsCA: true,
		BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	derCA, err := x509.CreateCertificate(rand.Reader, plantillaCA, plantillaCA, claveCA.Public(), claveCA)
	if err != nil {
		t.Fatalf("crear CA: %v", err)
	}
	certificadoCA, err := x509.ParseCertificate(derCA)
	if err != nil {
		t.Fatalf("parsear CA: %v", err)
	}
	crearCertificado := func(serial int64, nombre string, uso x509.ExtKeyUsage) tls.Certificate {
		publica, privada, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generar clave: %v", err)
		}
		plantilla := &x509.Certificate{
			SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: nombre},
			DNSNames: []string{nombre}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour),
			KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{uso},
		}
		der, err := x509.CreateCertificate(rand.Reader, plantilla, certificadoCA, publica, claveCA)
		if err != nil {
			t.Fatalf("crear certificado: %v", err)
		}
		return tls.Certificate{Certificate: [][]byte{der, derCA}, PrivateKey: privada}
	}
	raices := x509.NewCertPool()
	raices.AddCert(certificadoCA)
	autenticacionCliente := tls.NoClientCert
	if conCliente {
		autenticacionCliente = tls.RequireAndVerifyClientCert
	}
	servidorConfig := &tls.Config{
		Certificates: []tls.Certificate{crearCertificado(2, "servidor.identidad.test", x509.ExtKeyUsageServerAuth)},
		ClientAuth:   autenticacionCliente, ClientCAs: raices,
		MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13,
	}
	clienteConfig := &tls.Config{
		RootCAs: raices, ServerName: "servidor.identidad.test",
		MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13,
	}
	if conCliente {
		clienteConfig.Certificates = []tls.Certificate{
			crearCertificado(3, sanProxyIdentidadOfflinePrueba, x509.ExtKeyUsageClientAuth),
		}
	}
	parServidor, parCliente := net.Pipe()
	servidor := tls.Server(parServidor, servidorConfig)
	cliente := tls.Client(parCliente, clienteConfig)
	errores := make(chan error, 2)
	go func() { errores <- servidor.Handshake() }()
	go func() { errores <- cliente.Handshake() }()
	for range 2 {
		if err := <-errores; err != nil {
			_ = parServidor.Close()
			_ = parCliente.Close()
			t.Fatalf("handshake mTLS: %v", err)
		}
	}
	estado := servidor.ConnectionState()
	_ = parServidor.Close()
	_ = parCliente.Close()
	return estado
}
