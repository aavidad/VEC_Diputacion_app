package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestSeudonimosDelPortalExternoUsanSuPropioEspacioDeClave(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	raiz := cfg.DevelopmentMaterialDir
	cargar := func() *derivadorIdentidadOperacionDesarrollo {
		idempotencia, err := cargarMaterialIdempotenciaDesarrollo(raiz, filepath.Join(raiz, config.DevelopmentIdempotencyHMACConfigRelativePath))
		if err != nil {
			t.Fatal(err)
		}
		d, err := nuevoDerivadorIdentidadOperacionDesarrollo(&idempotencia)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(d.borrar)
		return d
	}
	ids := postgresidentidad.IdentificadoresAlta{EspacioIdentidad: espacioIdentidadSesionDesarrollo,
		AsercionID: "a", SesionID: "s", CuentaID: "desarrollo:cta_sintetica", SujetoID: "desarrollo:sintetico"}
	interno, err := (&seudonimizadorSesionDesarrollo{derivador: cargar()}).SeudonimizarAlta(context.Background(), ids)
	if err != nil || !strings.HasPrefix(interno.ClaveID, "vec.identidad.desarrollo.g") {
		t.Fatalf("el espacio historico debe conservarse: %q %v", interno.ClaveID, err)
	}
	d := cargar()
	d.espacioSeudonimos = espacioSeudonimosPortalExterno
	externo, err := (&seudonimizadorSesionDesarrollo{derivador: d}).SeudonimizarAlta(context.Background(), ids)
	if err != nil || !strings.HasPrefix(externo.ClaveID, espacioSeudonimosPortalExterno+".g") || externo.ClaveID == interno.ClaveID {
		t.Fatalf("el externo debe usar su propio espacio de clave: %q %v", externo.ClaveID, err)
	}
}

func seudonimosValidosPrueba() seudonimosPortalExterno {
	return seudonimosPortalExterno{Version: 1, Cuentas: []seudonimoCuentaPortalExterno{{
		CuentaRef: "cta_sintetica_0123456789abcdef", Esquema: postgresidentidad.EsquemaHMACSHA256V1,
		DominioRef: dominioIdentidadSesionDesarrollo, ClaveID: espacioSeudonimosPortalExterno + ".g2", ClaveVersion: 2,
		CuentaHMAC: strings.Repeat("ab", 32), SujetoHMAC: strings.Repeat("cd", 32),
	}}}
}

func TestLadoInternoSoloRegistraAliasDelEspacioExterno(t *testing.T) {
	codificar := func(s seudonimosPortalExterno) []byte {
		b, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	if _, err := leerSeudonimosPortalExterno(codificar(seudonimosValidosPrueba())); err != nil {
		t.Fatalf("alias externos validos rechazados: %v", err)
	}
	for nombre, cambiar := range map[string]func(*seudonimoCuentaPortalExterno){
		"espacio interno":  func(c *seudonimoCuentaPortalExterno) { c.ClaveID = "vec.identidad.desarrollo.g2" },
		"huellas iguales":  func(c *seudonimoCuentaPortalExterno) { c.SujetoHMAC = c.CuentaHMAC },
		"huella corta":     func(c *seudonimoCuentaPortalExterno) { c.CuentaHMAC = "ab" },
		"cuenta sin cta_":  func(c *seudonimoCuentaPortalExterno) { c.CuentaRef = "per_sintetica" },
		"dominio ajeno":    func(c *seudonimoCuentaPortalExterno) { c.DominioRef = "idh_otro" },
		"esquema ajeno":    func(c *seudonimoCuentaPortalExterno) { c.Esquema = "otro" },
		"version invalida": func(c *seudonimoCuentaPortalExterno) { c.ClaveVersion = 0 },
	} {
		s := seudonimosValidosPrueba()
		cambiar(&s.Cuentas[0])
		if _, err := leerSeudonimosPortalExterno(codificar(s)); !errors.Is(err, ErrSeudonimosPortalExternoInvalidos) {
			t.Fatalf("%s aceptado: %v", nombre, err)
		}
	}
	if _, err := leerSeudonimosPortalExterno([]byte(`{"version":1,"cuentas":[],"extra":1}`)); err == nil {
		t.Fatal("campo desconocido aceptado")
	}
}

func TestExportarSeudonimosSoloEnElProcesoExterno(t *testing.T) {
	vaciarConexionesDelEntorno(t)
	m := generarMaterialPortalExternoPrueba(t)
	motivo := core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 2,
		CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_11111111111111111111111111111111"}
	identidad, err := os.ReadFile(filepath.Join(m.cfg.DevelopmentMaterialDir, "identidad", "candidato.json"))
	if err != nil {
		t.Fatal(err)
	}
	var candidato struct {
		Huella  string `json:"certificate_sha256"`
		Subject string `json:"subject"`
	}
	if json.Unmarshal(identidad, &candidato) != nil {
		t.Fatal("identidad del candidato ilegible")
	}
	c := configuracionUsuariosPreferenciasDesarrollo{Version: 1, Autoridad: AutoridadNoAutoritativa,
		Superficie: core.SuperficieAutenticacionExternaPersonalV1, MotivoConsulta: motivo, MotivoActualizacion: motivo,
		Cuentas: []cuentaUsuariosPreferenciasDesarrollo{{cuentaRutasDietasDesarrollo: cuentaRutasDietasDesarrollo{
			CertificadoSHA256: candidato.Huella, Sujeto: candidato.Subject,
			CuentaRef: "cta_externa_0123456789abcdefghijkl", PerfilRef: "prf_externa_0123456789abcdefghijkl"}}}}
	escribirJSONExternoPrueba(t, filepath.Join(m.cfg.DevelopmentMaterialDir, "identidad", "usuarios-preferencias-externa.json"), c)
	contenido, err := ExportarSeudonimosPortalExterno(m.cfg)
	if err != nil {
		t.Fatalf("exportacion en el externo: %v", err)
	}
	s, err := leerSeudonimosPortalExterno(contenido)
	if err != nil || len(s.Cuentas) != 1 || s.Cuentas[0].CuentaRef != c.Cuentas[0].CuentaRef ||
		strings.Contains(string(contenido), candidato.Subject) {
		t.Fatalf("exportacion inesperada o con identificadores en claro: %s %v", contenido, err)
	}
	for _, portal := range []string{"", "interno"} {
		cfg := m.cfg
		cfg.PortalProceso = portal
		if _, err := ExportarSeudonimosPortalExterno(cfg); err == nil {
			t.Fatalf("exportacion aceptada fuera del externo (%q)", portal)
		}
	}
}

func TestAliasSoloParaCuentasAutorizadasYNoInternas(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	s := seudonimosValidosPrueba()
	contenido, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	cuenta := s.Cuentas[0].CuentaRef
	if _, err := seudonimosAutorizadosPortalExterno(cfg, contenido, []string{cuenta}); err != nil {
		t.Fatalf("cuenta autorizada rechazada: %v", err)
	}
	for nombre, autorizadas := range map[string][]string{
		"sin lista":       nil,
		"otra cuenta":     {"cta_otra_0123456789abcdef"},
		"sin prefijo cta": {strings.TrimPrefix(cuenta, "cta_")},
	} {
		if _, err := seudonimosAutorizadosPortalExterno(cfg, contenido, autorizadas); !errors.Is(err, ErrSeudonimosPortalExternoInvalidos) {
			t.Fatalf("%s: alias aceptado: %v", nombre, err)
		}
	}
	// Una cuenta de la superficie corporativa de este proceso nunca recibe un
	// alias del espacio externo, aunque el operador la incluya por error.
	motivo := core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 2,
		CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_11111111111111111111111111111111"}
	interna := configuracionUsuariosPreferenciasDesarrollo{Version: 1, Autoridad: AutoridadNoAutoritativa,
		Superficie: core.SuperficieAutenticacionInternaCorporativaV1, MotivoConsulta: motivo, MotivoActualizacion: motivo,
		Cuentas: []cuentaUsuariosPreferenciasDesarrollo{{cuentaRutasDietasDesarrollo: cuentaRutasDietasDesarrollo{
			CertificadoSHA256: strings.Repeat("0", 64), Sujeto: "desarrollo:rrhh", CuentaRef: cuenta, PerfilRef: "prf_interna_0123456789abcdef"}}}}
	escribirJSONExternoPrueba(t, filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "usuarios-preferencias-interna.json"), interna)
	if _, err := seudonimosAutorizadosPortalExterno(cfg, contenido, []string{cuenta}); !errors.Is(err, ErrSeudonimosPortalExternoInvalidos) {
		t.Fatalf("alias externo para una cuenta interna aceptado: %v", err)
	}
}
