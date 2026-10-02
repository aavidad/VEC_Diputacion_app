package bootstrap

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

func configuracionPerfilFirmaR5Prueba(t *testing.T) configuracionPerfilFijoFirmaR5CTDesarrollo {
	t.Helper()
	s, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	concesiones, err := concesionesPerfilFirmaR5CTDesarrollo(httpinterno.RutaRegistroFirmaVec)
	if err != nil {
		t.Fatal(err)
	}
	p, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, s.reloj.Ahora(), "firma-r5-propia-prueba",
		[]string{httpinterno.RutaRegistroFirmaVec}, func(actor, perfil string) (core.InstantaneaAutorizacion, error) {
			return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(actor, perfil, s.reloj.Ahora(),
				"rol-nominal-r5-prueba", "Rol nominal de prueba", "asignacion-r5-prueba", concesiones,
				[]core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
		})
	if err != nil {
		t.Fatal(err)
	}
	huella, err := p.plantilla.VersionRol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	return configuracionPerfilFijoFirmaR5CTDesarrollo{soporte: s, clave: "firma-r5-propia-prueba",
		actoAsignacion: "cargo_ct:0123456789abcdef0123456789abcdef", rutas: []string{httpinterno.RutaRegistroFirmaVec},
		descriptor: descriptorPerfilFirmaR5CTDesarrollo{perfilRef: "perfil:ct:alternativo", perfilActivoRef: p.perfilRef(),
			rolID: p.plantilla.VersionRol.RolID, versionRolRef: p.plantilla.VersionRol.Referencia(), huellaVersionRol: huella,
			catalogoRef: "catalogo-firma-prueba:2", huellaCatalogo: strings.Repeat("a", 64)},
		circuito: reglas.CircuitoFirma{CatalogoID: "catalogo-firma-prueba", Version: 2, HuellaCatalogo: strings.Repeat("a", 64),
			Documentos: []reglas.CircuitoDocumento{{Documento: "resolucion", Pasos: []reglas.PasoFirma{{PerfilRef: "perfil:ct:principal", PerfilesAlternativos: []string{"perfil:ct:alternativo"}}}}}},
		contexto: p.contexto, esperado: p.contexto.Resultado, sesion: proveedorSesionOperativaCTPrueba{contexto: p.contexto}, plantilla: p.plantilla}
}

func TestPerfilFijoFirmaR5LigaRolPerfilAlternativoYContextoExistente(t *testing.T) {
	c := configuracionPerfilFirmaR5Prueba(t)
	p, err := nuevoPerfilFijoFirmaR5CTDesarrollo(c)
	if err != nil || !p.soloConsumoCentral || p.actoAsignacionEsperado() != c.actoAsignacion || !p.atiendeMetodo(c.rutas[0], http.MethodPost) {
		t.Fatalf("perfil nominal no consumible: %v", err)
	}
	c.plantilla.VersionRol.Concesiones[0].Accion = "accion-ajena"
	if p.plantilla.VersionRol.Concesiones[0].Accion != ports.AccionRegistrarFirmaVec {
		t.Fatal("plantilla comparte estado mutable")
	}
	for nombre, modificar := range map[string]func(*configuracionPerfilFijoFirmaR5CTDesarrollo){
		"perfil ajeno al catálogo": func(c *configuracionPerfilFijoFirmaR5CTDesarrollo) { c.descriptor.perfilRef = "perfil:ct:ajeno" },
		"rol distinto":             func(c *configuracionPerfilFijoFirmaR5CTDesarrollo) { c.descriptor.rolID = "otro-rol" },
		"huella distinta": func(c *configuracionPerfilFijoFirmaR5CTDesarrollo) {
			c.descriptor.huellaVersionRol = strings.Repeat("b", 64)
		},
		"perfil activo distinto": func(c *configuracionPerfilFijoFirmaR5CTDesarrollo) {
			c.descriptor.perfilActivoRef = "perfil-activo-ajeno"
		},
		"catálogo distinto": func(c *configuracionPerfilFijoFirmaR5CTDesarrollo) {
			c.descriptor.catalogoRef = "catalogo-firma-prueba:1"
		},
		"sesión ausente": func(c *configuracionPerfilFijoFirmaR5CTDesarrollo) { c.sesion = nil },
		"ruta antigua": func(c *configuracionPerfilFijoFirmaR5CTDesarrollo) {
			c.rutas = []string{httpinterno.RutaFirmaDocumento}
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			c := configuracionPerfilFirmaR5Prueba(t)
			modificar(&c)
			if _, err := nuevoPerfilFijoFirmaR5CTDesarrollo(c); err == nil {
				t.Fatal("configuración divergente admitida")
			}
		})
	}
}

func TestPerfilFijoFirmaR5SoloConsumeActoCentralExacto(t *testing.T) {
	c := configuracionPerfilFirmaR5Prueba(t)
	p, err := nuevoPerfilFijoFirmaR5CTDesarrollo(c)
	if err != nil {
		t.Fatal(err)
	}
	autoridad := c.soporte.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	autoridad.asignaciones = map[string]instantaneaPublicadaDesarrollo{p.perfilRef(): {instantanea: p.plantilla, actoAsignacion: c.actoAsignacion}}
	if _, ok := c.soporte.consumirPerfilFijoCTDesarrollo(context.Background(), p); !ok {
		t.Fatal("no consumió publicación central exacta")
	}
	publicada := autoridad.asignaciones[p.perfilRef()]
	publicada.actoAsignacion = "cargo_ct:abcdef0123456789abcdef0123456789"
	autoridad.asignaciones[p.perfilRef()] = publicada
	if _, ok := c.soporte.consumirPerfilFijoCTDesarrollo(context.Background(), p); ok {
		t.Fatal("consumió otro acto con mismo prefijo")
	}
	if autoridad.preparadas != 0 || autoridad.publicadas != 0 {
		t.Fatal("consumo provisionó permisos")
	}
}

func TestRutasR5YConcesionesSegregadas(t *testing.T) {
	for _, ruta := range []string{httpinterno.RutaRegistroFirmaVec, httpinterno.RutaRegistroFirmaExterna} {
		if !rutaContextoAutorizacionContratacionTemporalDesarrollo(ruta) || !rutaSesionOperativaCTDesarrollo(ruta) {
			t.Fatal("ruta sin contexto y sesión")
		}
		metodos := inventarioRutasCTDesarrollo()[ruta]
		if len(metodos) != 1 || metodos[0] != pdpCT(http.MethodPost) {
			t.Fatal("ruta fuera del inventario nominal")
		}
		concesiones, err := concesionesPerfilFirmaR5CTDesarrollo(ruta)
		if err != nil {
			t.Fatal(err)
		}
		for _, concesion := range concesiones {
			if concesion.Validar() != nil {
				t.Fatal("concesión no válida")
			}
		}
		otra := ports.AccionRegistrarFirmaVec
		if ruta == httpinterno.RutaRegistroFirmaVec {
			otra = ports.AccionRegistrarFirmaExterna
		}
		for _, concesion := range concesiones {
			if concesion.Accion == otra {
				t.Fatal("perfil sumó otra vía de registro")
			}
		}
	}
}

func TestDescriptoresR5NoCruzanRegistroNiPreflight(t *testing.T) {
	fronteras := descriptoresFronterasFirmaR5CTDesarrollo("prf_r5_vec", "prf_r5_externa")
	fronteras = append(fronteras, descriptorFronteraPreflightFirmaR5CTDesarrollo([]string{"prf_r5_lector"}))
	catalogoFronteras, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	politica := politicaDescriptoresCTPrueba(t)
	autorizaciones := descriptoresAutorizacionFirmaR5CTDesarrollo(politica)
	autorizaciones = append(autorizaciones, descriptoresAutorizacionPreflightFirmaR5CTDesarrollo(politica)...)
	catalogo, err := nuevoCatalogoAutorizacionComunDesarrollo(catalogoFronteras, autorizaciones)
	if err != nil {
		t.Fatal(err)
	}
	for _, frontera := range fronteras {
		if _, ok := catalogo.politicaPara(ports.AccionConsultarFirmasR5, frontera.Clave, frontera.ClavePolitica, frontera.ClaveCapacidad); ok {
			t.Fatal("una frontera V2 concedió la consulta V1")
		}
		if _, ok := catalogo.politicaPara(ports.AccionConsultarFirmasR5V2, frontera.Clave, frontera.ClavePolitica, frontera.ClaveCapacidad); !ok {
			t.Fatal("consulta R5 anidada sin política")
		}
		for _, accion := range []string{ports.AccionRegistrarFirmaVec, ports.AccionRegistrarFirmaExterna} {
			_, ok := catalogo.politicaPara(accion, frontera.Clave, frontera.ClavePolitica, frontera.ClaveCapacidad)
			if ok != (accion == frontera.ClaveCapacidad) {
				t.Fatal("frontera sumó otra vía de registro")
			}
		}
	}
	concesiones, err := concesionesPerfilFirmaR5CTDesarrollo(httpinterno.RutaPreflightFirmaR5)
	if err != nil || len(concesiones) != 2 {
		t.Fatal("preflight no conserva lectura separada")
	}
}
