package bootstrap

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	bolsapersonal "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type motivosLectorOpcionalPrueba struct{}

func (motivosLectorOpcionalPrueba) ResolverMotivoCuadroRRHH(context.Context, time.Time) (vecdomain.ReferenciaEntradaCatalogo, error) {
	return vecdomain.ReferenciaEntradaCatalogo{}, nil
}

func (motivosLectorOpcionalPrueba) ResolverMotivoDetalleRRHH(context.Context, time.Time) (vecdomain.ReferenciaEntradaCatalogo, error) {
	return vecdomain.ReferenciaEntradaCatalogo{}, nil
}

func TestFalloMontajeLectorRRHHConservaBaseYCierraConsultasOpcionales(t *testing.T) {
	huella := sha256.Sum256([]byte("certificado_lector_sintetico"))
	principal := vecdomain.Principal{ID: "lector:prueba", Roles: []string{"lector_rrhh"}}
	lector := identidadConsultaRRHHDesarrollo{identidad: identidadCertificadoDesarrollo{huella: huella, principal: principal}}
	cierresBase := 0
	base := dependenciasConsultasRRHHDesarrollo{cerrar: func() { cierresBase++ }}
	alta := &dependenciasAltaContratacionTemporalDesarrollo{soporte: &soporteAltaContratacionTemporalDesarrollo{sello: &selloConsultasContratacionTemporalDesarrollo{}}}
	_, err := nuevasDependenciasLectoresRRHHDesarrollo(context.Background(), config.Config{}, alta, nil,
		relojContratacionTemporalDesarrollo{}, []identidadConsultaRRHHDesarrollo{lector},
		&postgresct.PoolConsultasRRHHPostgreSQL{}, motivosLectorOpcionalPrueba{},
		vecdomain.ReferenciaEntradaCatalogo{}, vecdomain.ReferenciaEntradaCatalogo{}, base,
		catalogoFronterasComunDesarrollo{}, false)
	if !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) || cierresBase != 0 {
		t.Fatalf("el fallo del lector cerró la consulta base: err=%v cierres=%d", err, cierresBase)
	}
	resolvedor := &resolvedorIdentidadDesarrollo{
		porHuella:    map[[sha256.Size]byte]vecdomain.Principal{huella: principal},
		lectoresRRHH: map[[sha256.Size]byte]identidadConsultaRRHHDesarrollo{huella: lector},
	}
	cerrarConsultasLectoresRRHHDesarrollo(&base, resolvedor)
	if !resolvedor.lectoresRRHHInvalidos || len(resolvedor.lectoresConsultaRRHH()) != 0 ||
		len(resolvedor.porHuella) != 1 || base.cuadroHTTP == nil || base.detalleHTTP == nil {
		t.Fatal("el fallo operativo dejó lector habilitado o retiró la identidad base")
	}
	if _, err := base.cuadroHTTP.Consultar(context.Background(), ports.SolicitudCuadroRRHH{}); err == nil {
		t.Fatal("el cuadro usó el lector base tras un montaje opcional fallido")
	}
	base.cerrar()
	if cierresBase != 1 {
		t.Fatal("se perdió el cierre final de la consulta base")
	}
}

func TestLectorTecnicoRRHHRevalidaCuentaBaseSinCambiarPerfilFijo(t *testing.T) {
	alta, fijo, lector := escenarioLectorTecnicoFijoPrueba(t)
	soporte, _, err := nuevoSoporteLectorRRHHDesarrollo(alta, lector, relojContratacionTemporalDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	cuenta, err := soporteCuentaNominalConsultasRRHHDesarrollo(alta, soporte)
	if err != nil || cuenta != alta.soporte || soporte.contexto.Resultado.Contexto.PerfilActivoRef != fijo.perfilRef() {
		t.Fatal("el lector técnico no conservó cuenta base y perfil fijo exactos")
	}
	transacciones := &baseCuentaNominalDesarrolloPrueba{}
	seudonimos := seudonimosCuentaNominalDesarrolloPrueba()
	for i := 0; i < 2; i++ {
		if err := prepararCuentaNominalConsultasDesarrolloConTransaccion(context.Background(), transacciones, cuenta, seudonimos); err != nil {
			t.Fatalf("replay de cuenta base %d: %v", i, err)
		}
	}
	if transacciones.incorporaciones != 1 || transacciones.confirmaciones != 2 {
		t.Fatal("el lector técnico abrió otra cuenta o no revalidó el alias")
	}
	if err := prepararCuentaNominalConsultasDesarrolloConTransaccion(context.Background(), &baseCuentaNominalDesarrolloPrueba{}, soporte, seudonimos); err == nil {
		t.Fatal("el preparador genérico admitió el perfil fijo como dinámico")
	}
	for nombre, alterar := range map[string]func(*dependenciasAltaContratacionTemporalDesarrollo, *soporteAltaContratacionTemporalDesarrollo, *perfilFijoCTDesarrollo){
		"sujeto ajeno": func(_ *dependenciasAltaContratacionTemporalDesarrollo, s *soporteAltaContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo) {
			s.principalID = "sujeto:ajeno"
		},
		"certificado ajeno": func(_ *dependenciasAltaContratacionTemporalDesarrollo, s *soporteAltaContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo) {
			s.certificadoSHA256 = "certificado_ajeno"
		},
		"perfil genérico": func(a *dependenciasAltaContratacionTemporalDesarrollo, s *soporteAltaContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo) {
			s.contexto = a.soporte.contexto
		},
		"fijo ausente": func(a *dependenciasAltaContratacionTemporalDesarrollo, _ *soporteAltaContratacionTemporalDesarrollo, p *perfilFijoCTDesarrollo) {
			filtrados := make([]*perfilFijoCTDesarrollo, 0, len(a.soporte.perfilesFijos))
			for _, actual := range a.soporte.perfilesFijos {
				if actual != p {
					filtrados = append(filtrados, actual)
				}
			}
			a.soporte.perfilesFijos = filtrados
		},
		"fijo duplicado": func(a *dependenciasAltaContratacionTemporalDesarrollo, _ *soporteAltaContratacionTemporalDesarrollo, p *perfilFijoCTDesarrollo) {
			a.soporte.perfilesFijos = append(a.soporte.perfilesFijos, p)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			a, p, l := escenarioLectorTecnicoFijoPrueba(t)
			s, _, err := nuevoSoporteLectorRRHHDesarrollo(a, l, relojContratacionTemporalDesarrollo{})
			if err != nil {
				t.Fatal(err)
			}
			alterar(a, s, p)
			if elegido, err := soporteCuentaNominalConsultasRRHHDesarrollo(a, s); err == nil || elegido != nil {
				t.Fatal("una identidad o un perfil divergente llegó al replay de cuenta")
			}
		})
	}
}

func TestMaterialLectoresRRHHOpcionalNoDetieneIdentidadObligatoria(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	leer := func() *resolvedorIdentidadDesarrollo {
		t.Helper()
		material, err := cargarMaterialSeguridadDesarrollo(cfg)
		if err != nil {
			t.Fatalf("el material obligatorio debe componerse: %v", err)
		}
		if material.identidad == nil {
			t.Fatal("falta el resolvedor obligatorio")
		}
		return material.identidad
	}
	base := leer()
	if len(base.lectoresConsultaRRHH()) != 0 || base.lectoresRRHHInvalidos {
		t.Fatal("el lector RRHH se activó sin manifiesto")
	}
	tecnico, ok := base.principalConRolUnico(rolTecnicoRRHHContratacionTemporalDesarrollo)
	if !ok {
		t.Fatal("falta la identidad técnica obligatoria")
	}
	certificado, err := filepath.Rel(cfg.DevelopmentMaterialDir, rutas.ClientCertificate)
	if err != nil {
		t.Fatal(err)
	}
	identidad, err := filepath.Rel(cfg.DevelopmentMaterialDir, rutas.Identity)
	if err != nil {
		t.Fatal(err)
	}
	entrada := archivoEntradaConsultaRRHHDesarrollo{
		Certificate: certificado, Identity: identidad, Subject: tecnico.ID,
		PerfilRef: referenciaAltaContratacionTemporalDesarrollo("prf_", tecnico.ID+"\x00"+
			tecnico.Attributes["certificate_sha256"]+"\x00"+
			discriminadorPerfilFijoCTDesarrollo(clavePerfilFijoAltaCTDesarrollo).perfil),
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		ClaseAmbito:     string(ports.AmbitoOrganizacionRRHH),
		AmbitoRef:       organizacionAltaContratacionTemporalDesarrollo,
	}
	ruta := filepath.Join(cfg.DevelopmentMaterialDir, "identidad", nombreManifiestoConsultasRRHHDesarrollo)
	guardar := func(entrada archivoEntradaConsultaRRHHDesarrollo) {
		t.Helper()
		contenido, err := json.Marshal(archivoManifiestoConsultasRRHHDesarrollo{
			Version: 1, Autoridad: AutoridadNoAutoritativa,
			Entradas: []archivoEntradaConsultaRRHHDesarrollo{entrada},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	guardar(entrada)
	valida := leer()
	if lectores := valida.lectoresConsultaRRHH(); len(lectores) != 1 || lectores[0].perfilRef != entrada.PerfilRef || valida.lectoresRRHHInvalidos {
		t.Fatalf("el lector válido no se registró: %d", len(lectores))
	}
	if err := os.WriteFile(ruta, []byte(`{"version":`), 0o600); err != nil {
		t.Fatal(err)
	}
	invalida := leer()
	if len(invalida.lectoresConsultaRRHH()) != 0 || !invalida.lectoresRRHHInvalidos {
		t.Fatal("un manifiesto mal formado concedió lectura")
	}
	entrada.PerfilRef = referenciaAltaContratacionTemporalDesarrollo("prf_", "perfil-ajeno")
	guardar(entrada)
	invalida = leer()
	if len(invalida.lectoresConsultaRRHH()) != 0 || !invalida.lectoresRRHHInvalidos {
		t.Fatal("un perfil divergente concedió lectura")
	}
}

func TestManifiestoLectorInvalidoDeniegaConsultaSinQuitarTecnico(t *testing.T) {
	alta, _, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	raw := []byte("certificado_sintetico_rrhh_opcional")
	huella := sha256.Sum256(raw)
	principal.Attributes["certificate_sha256"] = hex.EncodeToString(huella[:])
	resolvedor, err := nuevoResolvedorIdentidadDesarrollo(identidadCertificadoDesarrollo{huella: huella, principal: principal})
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC()
	certificado := &x509.Certificate{Raw: raw, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour)}
	probar := func(ruta, metodo string, invalido, permite bool) {
		t.Helper()
		resolvedor.lectoresRRHHInvalidos = invalido
		peticion := httptest.NewRequest(metodo, "https://localhost"+ruta, nil)
		peticion.RemoteAddr = "127.0.0.1:41000"
		peticion.TLS = &tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13,
			PeerCertificates: []*x509.Certificate{certificado}, VerifiedChains: [][]*x509.Certificate{{certificado, {}}}}
		llamadas := 0
		middleware := &revalidadorConsultasContratacionTemporalDesarrollo{
			autoridad: &autoridadConsultasContratacionTemporalDesarrollo{sello: alta.soporte.sello, resolvedor: resolvedor},
			siguiente: http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				llamadas++
				capacidad, existe := r.Context().Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
				sellada := existe && capacidad.sello == alta.soporte.sello
				if sellada != permite {
					t.Fatalf("ruta=%s manifiesto inválido=%t capacidad sellada=%t", ruta, invalido, sellada)
				}
			}),
		}
		middleware.ServeHTTP(httptest.NewRecorder(), peticion)
		if llamadas != 1 {
			t.Fatal("la petición no continuó por la frontera")
		}
	}
	probar(httpinterno.RutaEstadisticasRRHH, http.MethodGet, false, true)
	probar(httpinterno.RutaEstadisticasRRHH, http.MethodGet, true, false)
	probar(httpinterno.RutaAltaSolicitudes, http.MethodPost, true, true)
}

func TestInscripcionRRHHOpcionalConFalloConservaRestoDeRutas(t *testing.T) {
	cfg := config.Config{BolsaInscripcionesEnabled: "valor_invalido"}
	seleccion, err := seleccionMaterialCTDesarrolloDesdeConfig(cfg)
	if err != nil || seleccion.inscripcionBolsa {
		t.Fatalf("el selector opcional bloqueó la selección general: %+v %v", seleccion, err)
	}
	for _, descriptor := range descriptoresMaterialSeleccionadosCTDesarrollo(seleccion) {
		for _, inscripcion := range descriptoresMaterialInscripcionRRHHDesarrollo() {
			if descriptor.Audiencia == inscripcion.Audiencia {
				t.Fatal("se publicó material V3 con el selector inválido")
			}
		}
	}
	if i := componerInscripcionRRHHOpcionalDesarrollo(cfg, nil, nil, materialInscripcionRRHHDesarrollo{}, nil); i != nil {
		t.Fatal("se montaron rutas con selector inválido")
	}
	base := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	compuesta := componerRaizConInscripcionRRHH(base, nil)
	respuesta := httptest.NewRecorder()
	compuesta.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, "/api/vec/bolsa/rrhh/inscripciones", nil))
	if respuesta.Code != http.StatusTeapot {
		t.Fatalf("la ruta general dejó de responder: %d", respuesta.Code)
	}
	activo := configInscripcionEncendidaPrueba(t)
	if i := componerInscripcionRRHHOpcionalDesarrollo(activo, nil, nil, materialInscripcionRRHHDesarrollo{}, nil); i != nil {
		t.Fatal("se montaron rutas sin identidad RRHH")
	}
}

func TestPortalExternoOpcionalInvalidoNoActivaPerfilInterno(t *testing.T) {
	t.Setenv(config.EnvBolsaLlamamientosDatabaseURL, "postgres://bolsa@localhost/vec")
	cfg := config.Config{
		BolsaInscripcionesEnabled:      "valor_invalido",
		DevelopmentMaterialDir:         t.TempDir(),
		ContratacionTemporalPostgreSQL: config.Load().ContratacionTemporalPostgreSQL,
	}
	identidades := filepath.Join(cfg.DevelopmentMaterialDir, "identidad")
	if err := os.Mkdir(identidades, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(identidades, "bolsa-candidato.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !debeComponerMiBolsaDesarrollo(cfg) || !miBolsaExternaSeleccionadaEnCombinado(cfg) {
		t.Fatal("el selector inválido no retuvo la selección externa")
	}
	seleccion, err := seleccionMaterialCTDesarrolloDesdeConfig(cfg)
	if err != nil || seleccion.miBolsa || seleccion.portalCandidato {
		t.Fatalf("se seleccionó la autoridad interna: %+v %v", seleccion, err)
	}
	bolsa, inscripcion, cerrar := componerBolsaPersonalExternaOpcionalDesarrollo(cfg, nil, nil)
	if bolsa != nil || inscripcion != nil || cerrar == nil {
		t.Fatal("el selector inválido montó rutas externas")
	}
	cerrar()
}

func TestFalloInscripcionExternaConservaMiBolsaYLiberaSoloSuCierre(t *testing.T) {
	cerradas := 0
	inscripcion, cerrar := aislarFalloInscripcionExterna(nil, func() { cerradas++ }, errors.New("fallo sintético"))
	if inscripcion != nil || cerrar == nil || cerradas != 1 {
		t.Fatal("falló el cierre aislado de inscripción")
	}
	bolsa := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) })
	base := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	raiz := componerRaizConBolsaPersonalExterna(base, bolsa, inscripcion)
	respuesta := httptest.NewRecorder()
	raiz.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, bolsapersonal.RutaMiBolsa, nil))
	if respuesta.Code != http.StatusForbidden {
		t.Fatalf("Mi Bolsa válida se perdió al fallar inscripción: %d", respuesta.Code)
	}
	cerrar()
	if cerradas != 1 {
		t.Fatal("el cierre devuelto cerró dos veces inscripción")
	}
}

func TestInscripcionRRHHCuentasNoSustituyePerfil(t *testing.T) {
	huella := sha256.Sum256([]byte("certificado_sintetico"))
	certificado := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	lector := identidadConsultaRRHHDesarrollo{identidad: identidadCertificadoDesarrollo{huella: huella}, perfilRef: "prf_correcto"}
	lector.identidad.principal.ID = "tecnico:prueba"
	lector.identidad.principal.Attributes = map[string]string{"certificate_sha256": certificado}
	resolvedor := &resolvedorIdentidadDesarrollo{lectoresRRHH: map[[sha256.Size]byte]identidadConsultaRRHHDesarrollo{huella: lector}}
	cuenta := cuentaRutasDietasDesarrollo{CertificadoSHA256: certificado, Sujeto: "tecnico:prueba", PerfilRef: "prf_ajeno", CuentaRef: "cta_prueba"}
	if lectores, err := cuentasRRHHInscripcionDesarrollo(resolvedor, []cuentaRutasDietasDesarrollo{cuenta}); err == nil || lectores != nil {
		t.Fatal("se admitió una cuenta de inscripción con otro perfil")
	}
}
