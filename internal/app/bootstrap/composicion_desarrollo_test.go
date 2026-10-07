package bootstrap

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	reglasbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/reglas"
	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestLecturaNominalRRHHBolsaRecibeIntentosAntesDeConfigurar(t *testing.T) {
	fuente := &fuenteConstituidaRRHHDesarrollo{}
	llamadas := 0
	configurar := func(recibida *fuenteConstituidaRRHHDesarrollo) error {
		llamadas++
		if recibida != fuente || recibida.intentos == nil {
			t.Fatal("la configuración recibió una fuente sin política de intentos")
		}
		return nil
	}
	if err := configurarLecturasNominalesPreparadasRRHHBolsa(fuente, configurar); !errors.Is(err, ErrComposicionDesarrolloIncompleta) || llamadas != 0 {
		t.Fatalf("configuración antes de parámetros: llamadas=%d error=%v", llamadas, err)
	}
	fuente.intentos = reglasbolsa.NuevosIntentosContacto(nil)
	if err := configurarLecturasNominalesPreparadasRRHHBolsa(fuente, configurar); err != nil || llamadas != 1 {
		t.Fatalf("configuración después de parámetros: llamadas=%d error=%v", llamadas, err)
	}
}

func TestComposicionLecturasRRHHBolsaAislaAccionesEnPDPComun(t *testing.T) {
	const perfil = "prf_bolsa_bback"
	fronteras, err := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo(perfil, false)
	if err != nil {
		t.Fatal(err)
	}
	fronteras, err = sustituirFronteraCandidatosB5PorNominalBolsa(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	nominales, err := descriptoresFronterasRRHHNominalBolsaDesarrollo(perfil)
	if err != nil {
		t.Fatal(err)
	}
	fronteras = append(fronteras, nominales...)
	for i := range fronteras {
		for j := 0; j < i; j++ {
			if colisionanFronterasComunDesarrollo(fronteras[i], fronteras[j]) {
				t.Fatalf("fronteras colisionan: %s y %s", fronteras[i].Clave, fronteras[j].Clave)
			}
		}
	}
	catalogoFronteras, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	politica := politicaDescriptoresBolsaPrueba(t)
	autorizaciones, err := descriptoresAutorizacionBorradorLlamamientoBolsaDesarrollo(politica, false)
	if err != nil {
		t.Fatal(err)
	}
	autorizaciones, err = sustituirAutorizacionCandidatosB5PorNominalBolsa(autorizaciones)
	if err != nil {
		t.Fatal(err)
	}
	autorizacionesNominales, err := descriptoresAutorizacionRRHHNominalBolsaDesarrollo(politica)
	if err != nil {
		t.Fatal(err)
	}
	autorizaciones = append(autorizaciones, autorizacionesNominales...)
	catalogo, err := nuevoCatalogoAutorizacionComunDesarrollo(catalogoFronteras, autorizaciones)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct{ accion, frontera, capacidad string }{
		{puertosbolsa.AccionRRHHBolsasConsultar, claveFronteraRRHHBolsasBolsa, claveCapacidadRRHHBolsasBolsa},
		{puertosbolsa.AccionRRHHEstadisticasConsultar, claveFronteraRRHHEstadisticasBolsa, claveCapacidadRRHHEstadisticasBolsa},
		{puertosbolsa.AccionRRHHCandidatosConsultar, claveFronteraRRHHCandidatosBolsa, claveCapacidadRRHHCandidatosBolsa},
	} {
		if _, ok := catalogo.politicaPara(caso.accion, caso.frontera, clavePoliticaRRHHNominalBolsa, caso.capacidad); !ok {
			t.Fatalf("acción nominal sin política propia: %s", caso.accion)
		}
		if _, ok := catalogo.politicaPara(caso.accion, claveFronteraCrearBorradorLlamamientoBolsa,
			clavePoliticaBorradorLlamamientoBolsaDesarrollo, claveCapacidadCrearBorradorLlamamientoBolsa); ok {
			t.Fatalf("acción nominal reutilizó frontera de escritura: %s", caso.accion)
		}
	}
	if _, ok := catalogo.politicaPara(puertosbolsa.AccionCrearBorradorLlamamientoInterno,
		claveFronteraCrearBorradorLlamamientoBolsa, clavePoliticaBorradorLlamamientoBolsaDesarrollo,
		claveCapacidadCrearBorradorLlamamientoBolsa); !ok {
		t.Fatal("el catálogo común perdió la acción de borrador existente")
	}
	rutaCandidatos := rutaBolsasRRHHDesarrollo + "/bolsa:01/candidatos"
	frontera, ok := catalogoFronteras.resolver(http.MethodGet, rutaCandidatos)
	if !ok || frontera.Clave != claveFronteraRRHHCandidatosBolsa ||
		frontera.ClaveCapacidad != claveCapacidadRRHHCandidatosBolsa {
		t.Fatalf("la ruta de candidatos conserva una autoridad indebida: %+v", frontera)
	}
	if _, ok := catalogo.politicaPara(puertosbolsa.AccionConsultarContactoParticipacion,
		claveFronteraConsultarContactosB5Bolsa, clavePoliticaBorradorLlamamientoBolsaDesarrollo,
		claveCapacidadConsultarContactosBolsa); ok {
		t.Fatal("el permiso histórico B5 sigue acreditando la lista nominal")
	}
	if _, ok := catalogo.politicaPara(puertosbolsa.AccionConsultarContactoParticipacion,
		claveFronteraConsultarContactosBolsa, clavePoliticaBorradorLlamamientoBolsaDesarrollo,
		claveCapacidadConsultarContactosBolsa); !ok {
		t.Fatal("la lectura específica de contactos perdió su acción")
	}
}

func TestPrepararHuellasProvisionRRHHBolsaSoloLeeAsignacionPublicada(t *testing.T) {
	politica, autoridad, _ := politicaProvisionBolsaPrueba(t, 6)
	composicion := &ComposicionSeguridadDesarrollo{politicaRRHHNominalBolsa: politica}
	huellas, err := composicion.PrepararHuellasProvisionLecturasRRHHBolsa(context.Background())
	if err != nil || huellas.PreimagenSHA256 == "" || huellas.ObjetivoSHA256 == "" ||
		huellas.PreimagenSHA256 == huellas.ObjetivoSHA256 {
		t.Fatalf("preparación de huellas inválida: %+v, %v", huellas, err)
	}
	if autoridad.publicadas != 0 || autoridad.leida.instantanea.VersionRol.Version != 6 {
		t.Fatal("preparar huellas alteró la asignación o publicó un permiso")
	}
	for _, concesion := range autoridad.leida.instantanea.VersionRol.Concesiones {
		for _, nominal := range []string{
			puertosbolsa.AccionRRHHBolsasConsultar,
			puertosbolsa.AccionRRHHEstadisticasConsultar,
			puertosbolsa.AccionRRHHCandidatosConsultar,
		} {
			if concesion.Accion == nominal {
				t.Fatalf("la versión base ya concedía la acción nominal: %s", nominal)
			}
		}
	}
	if _, err := (&ComposicionSeguridadDesarrollo{}).PrepararHuellasProvisionLecturasRRHHBolsa(context.Background()); !errors.Is(err, ErrComposicionDesarrolloIncompleta) {
		t.Fatalf("preparación sin política nominal = %v", err)
	}
}

func directorioTemporalFueraDeGitPrueba(t *testing.T) string {
	t.Helper()
	candidatos := []string{os.TempDir()}
	if cacheUsuario, err := os.UserCacheDir(); err == nil {
		candidatos = append(candidatos, cacheUsuario)
	}
	if runtime.GOOS != "windows" {
		candidatos = append(candidatos, "/var/tmp", "/dev/shm", "/tmp")
	}
	vistos := make(map[string]struct{}, len(candidatos))
	motivos := make([]string, 0, len(candidatos))

	for _, candidato := range candidatos {
		baseAbsoluta, err := filepath.Abs(candidato)
		if err != nil {
			motivos = append(motivos, candidato+": "+err.Error())
			continue
		}
		base, err := filepath.EvalSymlinks(baseAbsoluta)
		if err != nil {
			motivos = append(motivos, baseAbsoluta+": resolver ruta física: "+err.Error())
			continue
		}
		base = filepath.Clean(base)
		if _, repetido := vistos[base]; repetido {
			continue
		}
		vistos[base] = struct{}{}
		if dentroDeRepositorioGit(base) {
			motivos = append(motivos, base+": pertenece a un árbol Git")
			continue
		}

		directorio, err := os.MkdirTemp(base, "vec-bootstrap-prueba-")
		if err != nil {
			motivos = append(motivos, base+": "+err.Error())
			continue
		}
		if dentroDeRepositorioGit(directorio) {
			if err := os.RemoveAll(directorio); err != nil {
				t.Fatalf("retirar temporal inadecuado %q: %v", directorio, err)
			}
			motivos = append(motivos, directorio+": pertenece a un árbol Git")
			continue
		}

		t.Cleanup(func() {
			if err := os.RemoveAll(directorio); err != nil && !errors.Is(err, os.ErrNotExist) {
				t.Errorf("retirar temporal de prueba %q: %v", directorio, err)
			}
		})
		return directorio
	}

	t.Fatalf("no existe un directorio temporal ajeno a Git: %s", strings.Join(motivos, "; "))
	return ""
}

func TestDirectorioTemporalFueraDeGitPruebaResuelveTMPDIREnlazado(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("TMPDIR no selecciona el directorio temporal en Windows")
	}
	raiz := directorioTemporalFueraDeGitPrueba(t)
	destinoFisico := filepath.Join(raiz, "destino-fisico")
	if err := os.Mkdir(destinoFisico, 0o700); err != nil {
		t.Fatalf("crear destino físico: %v", err)
	}
	enlace := filepath.Join(raiz, "tmp-enlazado")
	if err := os.Symlink(destinoFisico, enlace); err != nil {
		t.Fatalf("crear TMPDIR enlazado: %v", err)
	}
	t.Setenv("TMPDIR", enlace)

	directorio := directorioTemporalFueraDeGitPrueba(t)
	if filepath.Dir(directorio) != destinoFisico {
		t.Fatalf("el temporal no usa la ruta física de TMPDIR: obtenido=%q esperado=%q", directorio, destinoFisico)
	}
}

func generarMaterialDesarrolloPrueba(t *testing.T) (config.Config, config.DevelopmentMaterialPaths) {
	t.Helper()
	raizRepositorio, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	generador := filepath.Join(raizRepositorio, "scripts", "generar_credenciales_desarrollo.sh")
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl no disponible")
	}
	destino := filepath.Join(directorioTemporalFueraDeGitPrueba(t), "credenciales")
	orden := exec.Command(generador, destino)
	if salida, err := orden.CombinedOutput(); err != nil {
		t.Fatalf("generar material: %v\n%s", err, salida)
	}
	cfg := config.Config{
		Address:                   "127.0.0.1:0",
		ExecutionProfile:          config.ExecutionProfileDevelopment,
		AuthMode:                  config.AuthModeDevelopment,
		DevelopmentGuard:          config.DevelopmentGuardAcknowledgement,
		DevelopmentMaterialDir:    destino,
		PersonalCatalogPath:       "memory",
		BolsaPublicSourcePath:     filepath.Join(raizRepositorio, config.DefaultBolsaPublicSourcePath),
		BolsaCategoriesSourcePath: filepath.Join(raizRepositorio, config.DefaultBolsaCategoriesSourcePath),
	}.Normalize()
	return cfg, cfg.DevelopmentPaths()
}

func generarMaterialDesarrolloConPostgreSQLPrueba(
	t *testing.T,
) (config.Config, config.DevelopmentMaterialPaths) {
	t.Helper()
	variables := []string{
		config.EnvContratacionTemporalDatabaseURL,
		config.EnvContratacionTemporalGobiernoDatabaseURL,
		config.EnvContratacionTemporalConfirmadorDatabaseURL,
		config.EnvContratacionTemporalLectorResultadoDatabaseURL,
	}
	presentes := 0
	for _, variable := range variables {
		if strings.TrimSpace(os.Getenv(variable)) != "" {
			presentes++
		}
	}
	if presentes == 0 {
		t.Skip("PostgreSQL de contratación temporal no configurado")
	}
	if presentes != len(variables) {
		t.Fatalf(
			"PostgreSQL de contratación temporal incompleto: %d/%d conexiones",
			presentes,
			len(variables),
		)
	}
	postgresql := config.Load().ContratacionTemporalPostgreSQL
	if err := postgresql.Validar(); err != nil {
		t.Fatalf("PostgreSQL de contratación temporal inválido: %v", err)
	}
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	cfg.ContratacionTemporalPostgreSQL = postgresql
	return cfg, rutas
}

func TestProduccionRechazaTLSRealGeneradoPorT21(t *testing.T) {
	_, rutas := generarMaterialDesarrolloPrueba(t)
	servidor, err := NewHTTPServerWithConfig(config.Config{
		Address:             "127.0.0.1:0",
		PersonalCatalogPath: "memory",
		TLSCertFile:         rutas.ServerCertificate,
		TLSKeyFile:          rutas.ServerPrivateKey,
	})
	if servidor != nil || !errors.Is(err, ErrProveedorDesarrolloEnProduccion) {
		t.Fatalf("produccion acepto el TLS concreto de T21: servidor=%v error=%v", servidor, err)
	}
}

func TestServidorPublicoRechazaTLSYSelectoresRealesDeT21(t *testing.T) {
	cfgDesarrollo, rutas := generarMaterialDesarrolloPrueba(t)
	servidor, err := NewHTTPServerPublicoWithConfig(config.Config{
		Address:             "127.0.0.1:0",
		PersonalCatalogPath: "memory",
		TLSCertFile:         rutas.ServerCertificate,
		TLSKeyFile:          rutas.ServerPrivateKey,
	})
	if servidor != nil || !errors.Is(err, ErrProveedorDesarrolloEnProduccion) {
		t.Fatalf("servidor publico acepto TLS T21: servidor=%v error=%v", servidor, err)
	}

	cfgDesarrollo.TLSCertFile = ""
	cfgDesarrollo.TLSKeyFile = ""
	servidor, err = NewHTTPServerPublicoWithConfig(cfgDesarrollo)
	if servidor != nil || !errors.Is(err, ErrActivacionDesarrolloInvalida) {
		t.Fatalf("servidor publico degrado selectores T21: servidor=%v error=%v", servidor, err)
	}
}

func TestMaterialDesarrolloDetectaRepositorioDesdeLaPropiaRuta(t *testing.T) {
	repositorio := directorioTemporalFueraDeGitPrueba(t)
	if err := os.Mkdir(filepath.Join(repositorio, ".git"), 0o700); err != nil {
		t.Fatalf("crear marca Git: %v", err)
	}
	ruta := filepath.Join(repositorio, "estado", "credenciales")
	if !dentroDeRepositorioGit(ruta) {
		t.Fatal("la deteccion dependio del directorio de trabajo y acepto una ruta dentro de Git")
	}
	if dentroDeRepositorioGit(directorioTemporalFueraDeGitPrueba(t)) {
		t.Fatal("una ruta temporal ajena fue clasificada como repositorio")
	}
}

func TestComposicionDesarrolloOperaConTLSMutuoEIdentidadAlta(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloConPostgreSQLPrueba(t)
	var registro bytes.Buffer
	servidor, composicion, err := NewHTTPServerDesarrolloWithConfig(cfg, &registro)
	if err != nil {
		t.Fatalf("componer desarrollo: %v", err)
	}
	if servidor.TLSConfig == nil || servidor.TLSConfig.ClientAuth != tls.RequireAndVerifyClientCert ||
		servidor.TLSConfig.MinVersion != tls.VersionTLS13 {
		t.Fatalf("TLS de desarrollo no exige mTLS 1.3: %+v", servidor.TLSConfig)
	}
	metadatos, err := composicion.MetadatosComposicion()
	if err != nil || metadatos.Datos().Autoridad != AutoridadNoAutoritativa {
		t.Fatalf("marca del acto: %+v, %v", metadatos.Datos(), err)
	}
	procedencia, err := composicion.ProcedenciaActosBorrador()
	if err != nil || procedencia.Esquema == "" ||
		procedencia.Autoridad != gobiernoconvocatorias.AutoridadActoNoAutoritativa || procedencia.MigrableProduccion {
		t.Fatalf("procedencia durable: %+v, %v", procedencia, err)
	}
	if !strings.Contains(registro.String(), "credenciales_no_autoritativas") {
		t.Fatalf("arranque no fue ruidoso: %s", registro.String())
	}
	if _, err := composicion.CifradorBorradores(); err != nil {
		t.Fatalf("emisor KMS no compuesto: %v", err)
	}
	if _, err := composicion.RevalidadorKMSBorradores(); err != nil {
		t.Fatalf("revalidador KMS no compuesto: %v", err)
	}
	if _, err := composicion.VerificadorFirmasKMSBorradores(); err != nil {
		t.Fatalf("verificador publico KMS no compuesto: %v", err)
	}
	if _, err := composicion.DerivadorIdentidadesBorrador(); err != nil {
		t.Fatalf("derivador HMAC de idempotencia no compuesto: %v", err)
	}

	prueba := httptest.NewUnstartedServer(servidor.Handler)
	prueba.TLS = servidor.TLSConfig.Clone()
	prueba.StartTLS()
	t.Cleanup(prueba.Close)

	certificadoCliente, err := tls.LoadX509KeyPair(rutas.ClientCertificate, rutas.ClientPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	caPEM, err := os.ReadFile(rutas.CACertificate)
	if err != nil {
		t.Fatal(err)
	}
	raices := x509.NewCertPool()
	if !raices.AppendCertsFromPEM(caPEM) {
		t.Fatal("CA local no cargada")
	}
	clienteSinCertificado := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		RootCAs: raices, ServerName: "localhost", MinVersion: tls.VersionTLS13,
	}}}
	if respuestaSinCertificado, err := clienteSinCertificado.Get(prueba.URL + "/api/vec/session"); err == nil {
		respuestaSinCertificado.Body.Close()
		t.Fatal("el listener mTLS acepto un cliente sin certificado")
	}
	cliente := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		Certificates: []tls.Certificate{certificadoCliente}, RootCAs: raices,
		ServerName: "localhost", MinVersion: tls.VersionTLS13,
	}}}
	respuesta, err := cliente.Get(prueba.URL + "/api/vec/session")
	if err != nil {
		t.Fatalf("peticion mTLS: %v", err)
	}
	defer respuesta.Body.Close()
	contenido, _ := io.ReadAll(respuesta.Body)
	if respuesta.StatusCode != http.StatusOK || !bytes.Contains(contenido, []byte(`"autoridad":"no_autoritativo"`)) ||
		!bytes.Contains(contenido, []byte(`"auth_assurance":"alto"`)) {
		t.Fatalf("sesion mTLS = %d %s", respuesta.StatusCode, contenido)
	}
	respuestaPublica, err := cliente.Get(prueba.URL + "/api/publico/bolsa/convocatorias")
	if err != nil {
		t.Fatalf("consulta publica en servidor de desarrollo: %v", err)
	}
	contenidoPublico, err := io.ReadAll(respuestaPublica.Body)
	respuestaPublica.Body.Close()
	if err != nil || respuestaPublica.StatusCode != http.StatusOK ||
		!bytes.Contains(contenidoPublico, []byte(`"esquema":"vec.bolsa.publico.convocatorias.v1"`)) {
		t.Fatalf("consulta publica en desarrollo = %d %s, %v", respuestaPublica.StatusCode, contenidoPublico, err)
	}
	peticionSuplantada, err := http.NewRequest(http.MethodGet, prueba.URL+"/api/vec/session", nil)
	if err != nil {
		t.Fatal(err)
	}
	peticionSuplantada.Header.Set("X-Vec-Principal", "administrador")
	respuestaSuplantada, err := cliente.Do(peticionSuplantada)
	if err != nil {
		t.Fatal(err)
	}
	defer respuestaSuplantada.Body.Close()
	if respuestaSuplantada.StatusCode == http.StatusOK {
		t.Fatal("una cabecera de identidad suplanto al certificado mTLS")
	}
}

func TestRaizHTTPComponePerfilDesarrolloSoloConDobleLlaveCompleta(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloConPostgreSQLPrueba(t)
	servidor, err := NewHTTPServerWithConfig(cfg)
	if err != nil {
		t.Fatalf("raiz HTTP desarrollo: %v", err)
	}
	if servidor == nil || servidor.TLSConfig == nil ||
		servidor.TLSConfig.ClientAuth != tls.RequireAndVerifyClientCert ||
		servidor.TLSConfig.MinVersion != tls.VersionTLS13 {
		t.Fatalf("raiz HTTP no selecciono mTLS de desarrollo: %+v", servidor)
	}
}

func TestComposicionDesarrolloRechazaPermisosAmplios(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	if err := os.Chmod(rutas.KMSSecret, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard); err == nil {
		t.Fatal("se acepto un secreto legible por el grupo")
	}
}

func TestComposicionDesarrolloRechazaReutilizarClaveDeAtestacion(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	for origen, destino := range map[string]string{
		rutas.KMSAttestationKey:    rutas.KMSRevalidationKey,
		rutas.KMSAttestationPublic: rutas.KMSRevalidationPublic,
	} {
		contenido, err := os.ReadFile(origen)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destino, contenido, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard); err == nil {
		t.Fatal("se acepto la misma pareja para atestacion y revalidacion")
	}
}

func TestTSADesarrolloEsDeterministaYQuedaMarcada(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	tsa, err := composicion.SelladorTiempo()
	if err != nil {
		t.Fatal(err)
	}
	solicitud := vecports.InteropRequest{
		Operation: "sellar-huella", Subject: "documento:123",
		Payload: map[string]string{"huella_sha256": strings.Repeat("a", 64), "perfil": "PAdES-B-T"},
	}
	primero, err := tsa.Timestamp(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := tsa.Timestamp(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if primero.Reference != segundo.Reference || primero.Status != AutoridadNoAutoritativa ||
		primero.Payload["migrable_a_produccion"] != "false" {
		t.Fatalf("sello local no determinista o sin marca: %+v / %+v", primero, segundo)
	}
	solicitud.Payload["perfil"] = "PAdES-B-LTA"
	tercero, err := tsa.Timestamp(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if tercero.Reference == primero.Reference {
		t.Fatal("una carga distinta produjo el mismo sello")
	}
}
