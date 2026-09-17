package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestCatalogosAltaContratacionTemporalDesarrolloCompartenOrigenConAltaEnMTLSReal(
	t *testing.T,
) {
	cfg, rutas := generarMaterialDesarrolloConPostgreSQLPrueba(t)
	servidor, err := NewHTTPServerWithConfig(cfg)
	if err != nil {
		t.Fatalf("componer desarrollo: %v", err)
	}
	escucha, err := net.Listen("tcp", servidor.Addr)
	if err != nil {
		t.Fatalf("abrir listener: %v", err)
	}
	t.Cleanup(func() {
		_ = servidor.Close()
		_ = escucha.Close()
	})
	go func() {
		_ = servidor.ServeTLS(escucha, cfg.TLSCertFile, cfg.TLSKeyFile)
	}()
	baseURL := fmt.Sprintf("https://localhost:%d", escucha.Addr().(*net.TCPAddr).Port)
	ruta := baseURL + rutaCatalogosAltaContratacionTemporalDesarrollo

	t.Run("sin certificado no alcanza el catalogo", func(t *testing.T) {
		cliente := nuevoClienteSinCertificadoContratacionTemporalDesarrollo(t, rutas)
		peticion := nuevaPeticionCatalogosAltaContratacionTemporalDesarrolloPrueba(
			t, http.MethodGet, ruta,
		)
		respuesta, err := cliente.Do(peticion)
		if err == nil {
			contenido, _ := io.ReadAll(respuesta.Body)
			respuesta.Body.Close()
			t.Fatalf("el listener acepto catalogo sin certificado: %d %s", respuesta.StatusCode, contenido)
		}
	})

	cliente := nuevoClienteMTLSContratacionTemporalDesarrollo(t, rutas)
	anadirCadenaCompletaClienteMTLSContratacionTemporalDesarrollo(t, cliente, rutas)
	t.Cleanup(cliente.CloseIdleConnections)

	for _, caso := range []struct{ nombre, cabecera, valor string }{
		{"bearer", "Authorization", "Bearer autoridad-forjada"},
		{"cookie", "Cookie", "sesion=forjada"},
		{"principal cliente", "X-Vec-Principal", "administrador"},
		{"identidad cliente", "Identity", "administrador"},
		{"rol cliente", "X-Role", "administrador"},
	} {
		t.Run(caso.nombre+" no aporta autoridad", func(t *testing.T) {
			peticion := nuevaPeticionCatalogosAltaContratacionTemporalDesarrolloPrueba(
				t, http.MethodGet, ruta,
			)
			peticion.Header.Set(caso.cabecera, caso.valor)
			respuesta, contenido := ejecutarPeticionContratacionTemporalDesarrolloPrueba(
				t, cliente, peticion,
			)
			if respuesta.StatusCode == http.StatusOK ||
				bytes.Contains(contenido, []byte(esquemaCatalogosAltaContratacionTemporal)) ||
				len(respuesta.Header.Values("Set-Cookie")) != 0 {
				t.Fatalf("cabecera cliente alcanzo el catalogo: %d %s", respuesta.StatusCode, contenido)
			}
		})
	}

	peticion := nuevaPeticionCatalogosAltaContratacionTemporalDesarrolloPrueba(
		t, http.MethodGet, ruta,
	)
	respuesta, contenido := ejecutarPeticionContratacionTemporalDesarrolloPrueba(
		t, cliente, peticion,
	)
	if respuesta.StatusCode != http.StatusOK {
		t.Fatalf("catalogo=%d %s", respuesta.StatusCode, contenido)
	}
	if !strings.Contains(respuesta.Header.Get("Cache-Control"), "no-store") ||
		respuesta.Header.Get("Content-Type") != "application/json; charset=utf-8" ||
		len(respuesta.Header.Values("Set-Cookie")) != 0 {
		t.Fatalf("cabeceras inseguras: %v", respuesta.Header)
	}
	var sobreCrudo map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &sobreCrudo); err != nil ||
		len(sobreCrudo) != 1 || sobreCrudo["data"] == nil {
		t.Fatalf("sobre abierto o invalido: %v %s", err, contenido)
	}
	var datosCrudos map[string]json.RawMessage
	if err := json.Unmarshal(sobreCrudo["data"], &datosCrudos); err != nil ||
		len(datosCrudos) != 5 {
		t.Fatalf("datos abiertos o invalidos: %v %s", err, sobreCrudo["data"])
	}
	for _, campo := range []string{"esquema", "centros", "categorias", "motivos", "documentos"} {
		if datosCrudos[campo] == nil {
			t.Fatalf("falta campo cerrado %q: %s", campo, sobreCrudo["data"])
		}
	}

	var catalogos respuestaCatalogosAltaContratacionTemporalDesarrollo
	if err := json.Unmarshal(contenido, &catalogos); err != nil {
		t.Fatalf("decodificar catalogos: %v: %s", err, contenido)
	}
	if catalogos.Data.Esquema != esquemaCatalogosAltaContratacionTemporal ||
		len(catalogos.Data.Centros) != 1 ||
		len(catalogos.Data.Centros[0].Contactos) != 1 ||
		len(catalogos.Data.Categorias) != 1 ||
		len(catalogos.Data.Categorias[0].GruposSubgrupos) != 1 ||
		len(catalogos.Data.Motivos) != 1 ||
		catalogos.Data.Documentos == nil || len(catalogos.Data.Documentos) != 0 {
		t.Fatalf("catalogo no minimo: %+v", catalogos.Data)
	}
	centro := catalogos.Data.Centros[0]
	contacto := centro.Contactos[0]
	categoria := catalogos.Data.Categorias[0]
	grupo := categoria.GruposSubgrupos[0]
	motivo := catalogos.Data.Motivos[0]
	if centro.Referencia != centroAltaContratacionTemporalDesarrollo ||
		contacto.Referencia != contactoAltaContratacionTemporalDesarrollo ||
		categoria.Referencia != categoriaAltaContratacionTemporalDesarrollo ||
		grupo.Clave != grupoSubgrupoAltaContratacionTemporalDesarrollo ||
		motivo.Clave != string(motivoAltaContratacionTemporalDesarrollo) {
		t.Fatalf("referencias divergentes del alta: %+v", catalogos.Data)
	}
	for nombre, valores := range map[string][2]string{
		"centro":    {centro.Etiqueta, "Centro solicitante"},
		"contacto":  {contacto.Etiqueta, "Contacto del centro"},
		"categoria": {categoria.Etiqueta, "Categoría C2"},
		"grupo":     {grupo.Etiqueta, "Grupo C2"},
		"motivo":    {motivo.Etiqueta, "Sustitución temporal"},
	} {
		etiqueta, esperada := valores[0], valores[1]
		minusculas := strings.ToLower(etiqueta)
		if etiqueta != esperada || strings.Contains(minusculas, "desarrollo") ||
			strings.Contains(minusculas, "no autoritativ") ||
			strings.Contains(minusculas, "demo") {
			t.Fatalf("%s muestra una etiqueta no neutra: %q", nombre, etiqueta)
		}
	}

	cuerpoAlta := cuerpoAltaDesdeCatalogosContratacionTemporalDesarrolloPrueba(
		t, centro.Referencia, contacto.Referencia, categoria.Referencia,
		grupo.Clave, motivo.Clave,
	)
	peticionAlta := nuevaPeticionJSONContratacionTemporalDesarrolloPrueba(
		t, baseURL+httpinterno.RutaAltaSolicitudes, cuerpoAlta,
	)
	respuestaAlta, contenidoAlta := ejecutarPeticionContratacionTemporalDesarrolloPrueba(
		t, cliente, peticionAlta,
	)
	if respuestaAlta.StatusCode != http.StatusCreated ||
		!bytes.Contains(contenidoAlta, []byte(`"expediente_ref"`)) {
		t.Fatalf("las referencias catalogadas no alcanzan el alta: %d %s", respuestaAlta.StatusCode, contenidoAlta)
	}

	peticionHEAD := nuevaPeticionCatalogosAltaContratacionTemporalDesarrolloPrueba(
		t, http.MethodHead, ruta,
	)
	respuestaHEAD, contenidoHEAD := ejecutarPeticionContratacionTemporalDesarrolloPrueba(
		t, cliente, peticionHEAD,
	)
	if respuestaHEAD.StatusCode != http.StatusOK || len(contenidoHEAD) != 0 ||
		!strings.Contains(respuestaHEAD.Header.Get("Cache-Control"), "no-store") {
		t.Fatalf("HEAD invalido: %d %q %v", respuestaHEAD.StatusCode, contenidoHEAD, respuestaHEAD.Header)
	}

	for _, metodo := range []string{
		http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodOptions,
	} {
		t.Run("rechaza "+metodo, func(t *testing.T) {
			peticion := nuevaPeticionCatalogosAltaContratacionTemporalDesarrolloPrueba(
				t, metodo, ruta,
			)
			respuesta, contenido := ejecutarPeticionContratacionTemporalDesarrolloPrueba(
				t, cliente, peticion,
			)
			if respuesta.StatusCode != http.StatusMethodNotAllowed ||
				respuesta.Header.Get("Allow") != "GET, HEAD" ||
				bytes.Contains(contenido, []byte(esquemaCatalogosAltaContratacionTemporal)) {
				t.Fatalf("%s no fue rechazado: %d %s", metodo, respuesta.StatusCode, contenido)
			}
		})
	}
}

func TestCatalogosAltaContratacionTemporalNoSeRegistranFueraDeDesarrollo(
	t *testing.T,
) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatalf("componer dependencias de prueba: %v", err)
	}
	resolvedor, err := composicion.ResolvedorIdentidad()
	if err != nil {
		t.Fatalf("resolver identidad de prueba: %v", err)
	}
	casos := []struct {
		nombre    string
		modificar func(*config.Config)
	}{
		{"produccion", func(actual *config.Config) {
			actual.ExecutionProfile = config.ExecutionProfileProduction
		}},
		{"sin segunda llave", func(actual *config.Config) {
			actual.DevelopmentGuard = ""
		}},
		{"red no loopback", func(actual *config.Config) {
			actual.Address = "0.0.0.0:0"
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			actual := cfg
			caso.modificar(&actual)
			rutas, autoridad, cerrar, err := nuevasRutasContratacionTemporalDesarrollo(
				actual, resolvedor, composicion.derivadorIdempotencia, io.Discard,
			)
			if !errors.Is(err, ErrActivacionDesarrolloInvalida) ||
				rutas != nil || autoridad != nil || cerrar != nil {
				t.Fatalf("configuracion invalida registro rutas CT: rutas=%v autoridad=%v error=%v", rutas, autoridad, err)
			}
		})
	}
}

func nuevaPeticionCatalogosAltaContratacionTemporalDesarrolloPrueba(
	t *testing.T,
	metodo string,
	ruta string,
) *http.Request {
	t.Helper()
	peticion, err := http.NewRequest(metodo, ruta, nil)
	if err != nil {
		t.Fatal(err)
	}
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func cuerpoAltaDesdeCatalogosContratacionTemporalDesarrolloPrueba(
	t *testing.T,
	centro string,
	contacto string,
	categoria string,
	grupo string,
	motivo string,
) string {
	t.Helper()
	ahora := time.Now().UTC()
	inicio := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, 1)
	fin := inicio.AddDate(0, 3, 0)
	cuerpo := map[string]any{
		"clave_idempotencia": "781972a8-59d6-4168-a877-d9c61c0ae8e4",
		"solicitud": map[string]any{
			"centro_ref": centro, "contacto_ref": contacto,
			"categoria_ref": categoria, "grupo_subgrupo": grupo,
			"motivo_clave": motivo,
			"detalle":      "Cobertura temporal en entorno local no autoritativo.",
			"periodo": map[string]string{
				"inicio": inicio.Format("2006-01-02T15:04:05Z"),
				"fin":    fin.Format("2006-01-02T15:04:05Z"),
			},
			"rc":                  map[string]bool{"existe": false},
			"documentos_adjuntos": []string{},
			"observaciones":       "",
		},
	}
	contenido, err := json.Marshal(cuerpo)
	if err != nil {
		t.Fatal(err)
	}
	return string(contenido)
}

func TestEtiquetasReferenciasCatalogosAltaDesarrolloNombranCentroCategoriaYContacto(t *testing.T) {
	t.Parallel()
	origen := nuevoOrigenConsultasContratacionTemporalDesarrollo()
	etiquetar := origen.etiquetasReferenciasCatalogosAlta()
	for referencia, esperado := range map[string]string{
		centroAltaContratacionTemporalDesarrollo:    "Centro solicitante",
		categoriaAltaContratacionTemporalDesarrollo: "Auxiliar administrativo/a",
		contactoAltaContratacionTemporalDesarrollo:  "Contacto del centro",
		"unidad:desarrollo:rrhh":                    "",
		"":                                          "",
	} {
		if obtenido := etiquetar(referencia); obtenido != esperado {
			t.Fatalf("%q: esperado %q, obtenido %q", referencia, esperado, obtenido)
		}
	}
	var nulo *origenConsultasContratacionTemporalDesarrollo
	if nulo.etiquetasReferenciasCatalogosAlta()(centroAltaContratacionTemporalDesarrollo) != "" {
		t.Fatal("un origen nulo no debe nombrar nada")
	}
}

func TestCatalogosAltaDesarrolloConFicheroRPTPublica(t *testing.T) {
	t.Parallel()
	rutaRPT := "../../../data/catalogos/estructura-organizativa/v1.rpt-publica.json"
	origen := nuevoOrigenConsultasContratacionTemporalDesarrollo(rutaRPT)
	catalogos, err := origen.catalogosAlta()
	if err != nil {
		t.Fatalf("catalogosAlta con RPT fallo: %v", err)
	}
	if len(catalogos.Centros) != 41 {
		t.Fatalf("esperados 41 centros de la RPT, obtenidos %d", len(catalogos.Centros))
	}
	if len(catalogos.Categorias) != 6 {
		t.Fatalf("esperadas 6 categorias sinteticas, obtenidas %d", len(catalogos.Categorias))
	}
	if !origen.centroDeCatalogo("centro:rpt:101") {
		t.Fatal("centro:rpt:101 debe pertenecer al catalogo")
	}
	if !origen.centroDeCatalogo(centroAltaContratacionTemporalDesarrollo) {
		t.Fatal("centro:desarrollo:001 debe seguir aceptado")
	}
	if origen.centroDeCatalogo("centro:ajeno") {
		t.Fatal("centro:ajeno no debe pertenecer al catalogo")
	}
	if !origen.categoriaDeCatalogo("categoria:desarrollo:a2") {
		t.Fatal("categoria:desarrollo:a2 debe pertenecer al catalogo")
	}
	if !origen.categoriaDeCatalogo(categoriaAltaContratacionTemporalDesarrollo) {
		t.Fatal("categoria:desarrollo:c2 debe pertenecer al catalogo")
	}
	if origen.categoriaDeCatalogo("categoria:ajena") {
		t.Fatal("categoria:ajena no debe pertenecer al catalogo")
	}
	if !categoriaDeCatalogoDesarrollo("categoria:desarrollo:a2") {
		t.Fatal("categoriaDeCatalogoDesarrollo debe aceptar categoria:desarrollo:a2")
	}
	if !grupoSubgrupoDeCatalogoValido("categoria:desarrollo:a2", "A2") {
		t.Fatal("grupoSubgrupoDeCatalogoValido debe aceptar A2 para categoria:desarrollo:a2")
	}
	if grupoSubgrupoDeCatalogoValido("categoria:desarrollo:a2", "C1") {
		t.Fatal("grupoSubgrupoDeCatalogoValido no debe aceptar C1 para categoria:desarrollo:a2")
	}

	etiquetar := origen.etiquetasReferenciasCatalogosAlta()
	if obtenido := etiquetar("centro:rpt:101"); obtenido != "GABINETE DE PRESIDENCIA" {
		t.Fatalf("etiqueta centro:rpt:101 esperada 'GABINETE DE PRESIDENCIA', obtenida %q", obtenido)
	}
	if obtenido := etiquetar("categoria:desarrollo:a2"); obtenido != "Técnico/a medio/a" {
		t.Fatalf("etiqueta categoria:desarrollo:a2 esperada 'Técnico/a medio/a', obtenida %q", obtenido)
	}
	if obtenido := etiquetar(centroAltaContratacionTemporalDesarrollo); obtenido != "Centro solicitante" {
		t.Fatalf("etiqueta centro:desarrollo:001 esperada 'Centro solicitante', obtenida %q", obtenido)
	}
}

func TestAltaConCentroRPTeYCategoriaA2RegistraYCentroAjenoSeRechaza(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloConPostgreSQLPrueba(t)
	cfg.PersonalOrganizacionSourcePath = "../../../data/catalogos/estructura-organizativa/v1.rpt-publica.json"
	servidor, err := NewHTTPServerWithConfig(cfg)
	if err != nil {
		t.Fatalf("componer desarrollo: %v", err)
	}
	escucha, err := net.Listen("tcp", servidor.Addr)
	if err != nil {
		t.Fatalf("abrir listener: %v", err)
	}
	t.Cleanup(func() {
		_ = servidor.Close()
		_ = escucha.Close()
	})
	go func() {
		_ = servidor.ServeTLS(escucha, cfg.TLSCertFile, cfg.TLSKeyFile)
	}()
	baseURL := fmt.Sprintf("https://localhost:%d", escucha.Addr().(*net.TCPAddr).Port)

	cliente := nuevoClienteMTLSContratacionTemporalDesarrollo(t, rutas)
	anadirCadenaCompletaClienteMTLSContratacionTemporalDesarrollo(t, cliente, rutas)
	t.Cleanup(cliente.CloseIdleConnections)

	t.Run("alta con centro RPT y categoria A2 registra correctamente", func(t *testing.T) {
		cuerpo := cuerpoAltaDesdeCatalogosContratacionTemporalDesarrolloPrueba(
			t, "centro:rpt:101", "contacto:rpt:101", "categoria:desarrollo:a2",
			"A2", "sustitucion",
		)
		peticion := nuevaPeticionJSONContratacionTemporalDesarrolloPrueba(
			t, baseURL+httpinterno.RutaAltaSolicitudes, cuerpo,
		)
		respuesta, contenido := ejecutarPeticionContratacionTemporalDesarrolloPrueba(
			t, cliente, peticion,
		)
		if respuesta.StatusCode != http.StatusCreated || !bytes.Contains(contenido, []byte(`"expediente_ref"`)) {
			t.Fatalf("el alta con centro RPT y categoria A2 debio registrar: %d %s", respuesta.StatusCode, contenido)
		}
	})

	t.Run("alta con centro ajeno es rechazada", func(t *testing.T) {
		cuerpo := cuerpoAltaDesdeCatalogosContratacionTemporalDesarrolloPrueba(
			t, "centro:ajeno", "contacto:desarrollo:001", "categoria:desarrollo:a2",
			"A2", "sustitucion",
		)
		peticion := nuevaPeticionJSONContratacionTemporalDesarrolloPrueba(
			t, baseURL+httpinterno.RutaAltaSolicitudes, cuerpo,
		)
		respuesta, contenido := ejecutarPeticionContratacionTemporalDesarrolloPrueba(
			t, cliente, peticion,
		)
		if respuesta.StatusCode != http.StatusForbidden {
			t.Fatalf("el alta con centro ajeno debio ser rechazada con 403: %d %s", respuesta.StatusCode, contenido)
		}
	})
}

func TestSoporteAltaResolverFlujoYAutorizacionConCatalogoRPT(t *testing.T) {
	t.Parallel()
	rutaRPT := "../../../data/catalogos/estructura-organizativa/v1.rpt-publica.json"
	origen := nuevoOrigenConsultasContratacionTemporalDesarrollo(rutaRPT)
	flujoEsperado := ports.ConfiguracionAltaFlujo{
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo:ct:desarrollo",
			Version:       1,
			HuellaSHA256:  huellaAltaContratacionTemporalDesarrollo("flujo"),
		},
		FaseInicial:      domain.ClaveFase("solicitud"),
		UnidadInicialRef: "unidad:desarrollo:rrhh",
		AccionInicial:    domain.ClaveCatalogo("alta"),
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	sello := &selloConsultasContratacionTemporalDesarrollo{}
	p := dominiovec.Principal{
		ID:            "desarrollo:rrhh",
		DisplayName:   "RRHH sintético",
		Roles:         []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod:    dominiovec.AuthMethodCertificate,
		AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{
			"autoridad":          AutoridadNoAutoritativa,
			"perfil_ejecucion":   config.ExecutionProfileDevelopment,
			"certificate_sha256": strings.Repeat("a", 64),
		},
	}
	soporte := &soporteAltaContratacionTemporalDesarrollo{
		origen:            origen,
		flujo:             flujoEsperado,
		sello:             sello,
		principalID:       p.ID,
		certificadoSHA256: p.Attributes["certificate_sha256"],
	}
	ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidadConsultaContratacionTemporalDesarrollo{
		sello:                   sello,
		ruta:                    httpinterno.RutaAltaSolicitudes,
		principal:               p,
		certificadoVerificadoEn: ahora,
		certificadoValidoHasta:  ahora.Add(time.Hour),
	})

	t.Run("ResolverFlujoAlta acepta centro RPT y categoria A2", func(t *testing.T) {
		solicitud := ports.SolicitudResolverFlujo{
			OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
			CentroRef:       "centro:rpt:101",
			CategoriaRef:    "categoria:desarrollo:a2",
			MotivoClave:     motivoAltaContratacionTemporalDesarrollo,
			Instante:        ahora,
		}
		flujo, err := soporte.ResolverFlujoAlta(ctx, solicitud)
		if err != nil {
			t.Fatalf("esperado flujo disponible para centro RPT y categoria A2, obtenido error: %v", err)
		}
		if flujo.Flujo.DefinicionRef != flujoEsperado.Flujo.DefinicionRef {
			t.Fatalf("flujo inesperado: %+v", flujo)
		}
	})

	t.Run("ResolverFlujoAlta rechaza centro ajeno", func(t *testing.T) {
		solicitud := ports.SolicitudResolverFlujo{
			OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
			CentroRef:       "centro:ajeno",
			CategoriaRef:    "categoria:desarrollo:a2",
			MotivoClave:     motivoAltaContratacionTemporalDesarrollo,
			Instante:        ahora,
		}
		_, err := soporte.ResolverFlujoAlta(ctx, solicitud)
		if !errors.Is(err, ports.ErrFlujoNoDisponible) {
			t.Fatalf("esperado ErrFlujoNoDisponible para centro ajeno, obtenido: %v", err)
		}
	})

	t.Run("ResolverFlujoAlta rechaza categoria ajena", func(t *testing.T) {
		solicitud := ports.SolicitudResolverFlujo{
			OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
			CentroRef:       "centro:rpt:101",
			CategoriaRef:    "categoria:ajena",
			MotivoClave:     motivoAltaContratacionTemporalDesarrollo,
			Instante:        ahora,
		}
		_, err := soporte.ResolverFlujoAlta(ctx, solicitud)
		if !errors.Is(err, ports.ErrFlujoNoDisponible) {
			t.Fatalf("esperado ErrFlujoNoDisponible para categoria ajena, obtenido: %v", err)
		}
	})

	t.Run("solicitudAutorizacionAlta valida centro RPT y categoria A2", func(t *testing.T) {
		datosValidos := dominiovec.DatosSolicitudAutorizacionLigadaV3{
			Accion:    ports.AccionCrearSolicitud,
			Finalidad: ports.FinalidadCrearSolicitud,
			Recurso: dominiovec.RecursoAutorizable{
				ModuloID: ports.ModuloContratacion,
				Tipo:     ports.TipoRecursoExpediente,
				Ambitos: map[string]string{
					"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
					"centro_ref":       "centro:rpt:101",
					"categoria_ref":    "categoria:desarrollo:a2",
				},
			},
		}
		if !soporte.solicitudAutorizacionAltaContratacionTemporalDesarrolloValida(httpinterno.RutaAltaSolicitudes, datosValidos) {
			t.Fatal("datos validos con centro RPT y categoria A2 deben ser validos")
		}

		datosCentroAjeno := datosValidos
		datosCentroAjeno.Recurso.Ambitos = map[string]string{
			"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
			"centro_ref":       "centro:ajeno",
			"categoria_ref":    "categoria:desarrollo:a2",
		}
		if soporte.solicitudAutorizacionAltaContratacionTemporalDesarrolloValida(httpinterno.RutaAltaSolicitudes, datosCentroAjeno) {
			t.Fatal("centro ajeno debe ser rechazado en solicitudAutorizacionAlta")
		}
	})

	t.Run("AmbitoPerfil lista todos los centros de la RPT", func(t *testing.T) {
		inst, err := nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
			"principal-1", "perfil-1", time.Now().UTC(), origen,
		)
		if err != nil {
			t.Fatalf("crear instantanea con origen RPT: %v", err)
		}
		var valoresCentros []string
		var valoresCategorias []string
		for _, a := range inst.AsignacionPerfil.Ambitos {
			if a.Clave == "centro_ref" {
				valoresCentros = a.Valores
			}
			if a.Clave == "categoria_ref" {
				valoresCategorias = a.Valores
			}
		}
		if len(valoresCentros) < 41 {
			t.Fatalf("esperados al menos 41 centros en AmbitoPerfil, obtenidos %d", len(valoresCentros))
		}
		if len(valoresCategorias) < 5 {
			t.Fatalf("esperadas al menos 5 categorias en AmbitoPerfil, obtenidas %d", len(valoresCategorias))
		}
	})
}
