package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/bootstrap"
	selector "vec-diputacion-granada/internal/vec/adapters/httpseguridad/adminperfiles"
	identidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	"vec-diputacion-granada/internal/vec/domain"
)

// escenario genera con el proveedor DEV publicado y claves sintéticas el
// mismo juego de archivos que deja el circuito de arranque. No implementa HMAC.
type escenario struct {
	dir, salida string
	archivos    map[string]any
	material    materialHMAC
	plan        domain.PlanFuentesInicialesAdminV1
	originales  originalesArranque
	extra       []string
}

func privado(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	if err := os.Chmod(d, 0700); err != nil {
		t.Fatal(err)
	}
	return d
}

func escribirJSON(t *testing.T, ruta string, v any, modo os.FileMode) []byte {
	t.Helper()
	b, ok := v.([]byte)
	if !ok {
		var err error
		if b, err = json.Marshal(v); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(ruta, b, modo); err != nil {
		t.Fatal(err)
	}
	return b
}

func proveedorSintetico(t *testing.T) proveedorHMAC {
	t.Helper()
	raiz := privado(t)
	dir := filepath.Join(raiz, "idempotencia")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	type generacion struct {
		Generacion                int    `json:"generacion"`
		ReferenciaLocalizador     string `json:"referencia_localizador"`
		ReferenciaHuellaSolicitud string `json:"referencia_huella_solicitud"`
	}
	config := struct {
		Version            int          `json:"version"`
		Esquema            string       `json:"esquema"`
		Autoridad          string       `json:"autoridad"`
		VersionEsquemaHMAC int          `json:"version_esquema_hmac"`
		Generaciones       []generacion `json:"generaciones"`
	}{Version: 1, Esquema: "vec.bolsa.convocatoria.idempotencia-hmac.desarrollo.v1", Autoridad: "no_autoritativo", VersionEsquemaHMAC: 2}
	for _, n := range []int{2, 1} {
		config.Generaciones = append(config.Generaciones, generacion{n, fmt.Sprintf("clave:hmac:convocatorias:localizador:desarrollo:v%d", n), fmt.Sprintf("clave:hmac:convocatorias:huella:desarrollo:v%d", n)})
		for i, dominio := range []string{"localizador", "huella-solicitud"} {
			escribirJSON(t, filepath.Join(dir, fmt.Sprintf("g%d-%s.bin", n, dominio)), bytes.Repeat([]byte{byte(n*2 + i)}, 32), 0600)
		}
	}
	rutaConfig := filepath.Join(dir, "configuracion.json")
	escribirJSON(t, rutaConfig, config, 0600)
	return proveedorHMAC{DirectorioMaterial: raiz, RutaConfiguracionHMAC: rutaConfig, EspacioIdentidad: "https://admin.example.invalid/identidad",
		DominioRef: "idh_0123456789abcdefghijklmn", EspacioClave: "vec.identidad.admin.prueba", DominioHMAC: "vec.identidad.admin.hmac.v1"}
}

// materialComoArranque reproduce el helper del arranque: proveedor sin cuenta
// ordinaria y la cuenta ordinaria calculada con el propósito "cuenta".
func materialComoArranque(t *testing.T, prov proveedorHMAC, o originalesArranque) materialHMAC {
	t.Helper()
	seud, cerrar, err := bootstrap.NuevoSeudonimizadorSesionDesdeArchivo(bootstrap.ConfiguracionSeudonimosSesionPrivada{DirectorioMaterial: prov.DirectorioMaterial,
		RutaConfiguracionHMAC: prov.RutaConfiguracionHMAC, EspacioIdentidad: prov.EspacioIdentidad, DominioRef: prov.DominioRef, EspacioClave: prov.EspacioClave, DominioHMAC: prov.DominioHMAC})
	if err != nil {
		t.Fatal(err)
	}
	defer cerrar()
	m := materialHMAC{Version: 1, FuenteRef: "prc_" + strings.Repeat("f", 32), FuenteVersion: 1}
	for i, p := range o.Personas {
		ids := identidad.IdentificadoresAlta{EspacioIdentidad: prov.EspacioIdentidad, AsercionID: p.SujetoOriginal, SesionID: p.SujetoOriginal, SujetoID: p.SujetoOriginal, CuentaID: p.CuentaOrdinariaOriginal}
		ord, err := seud.SeudonimizarAlta(context.Background(), ids)
		if err != nil {
			t.Fatal(err)
		}
		ids.CuentaID = p.CuentaPrivilegiadaOriginal
		adm, err := seud.SeudonimizarAlta(context.Background(), ids)
		if err != nil {
			t.Fatal(err)
		}
		m.Esquema, m.DominioRef, m.ClaveID, m.ClaveVersion = ord.Esquema, ord.DominioRef, ord.ClaveID, ord.ClaveVersion
		m.Personas[i] = materialHMACPersona{PersonaRef: p.PersonaRef, CuentaOrdinaria: hex.EncodeToString(ord.CuentaIDHMAC[:]),
			CuentaPrivilegiada: hex.EncodeToString(adm.CuentaIDHMAC[:]), Sujeto: hex.EncodeToString(ord.SujetoIDHMAC[:])}
	}
	return m
}

func nuevoEscenario(t *testing.T, alterar func(*materialHMAC)) *escenario {
	t.Helper()
	e := &escenario{dir: privado(t), archivos: map[string]any{}}
	e.salida = filepath.Join(privado(t), "identificadores.json")
	prov := proveedorSintetico(t)
	e.originales = originalesArranque{Entorno: "desarrollo", Alcance: "sintetico_declarado", OrganizacionRef: "org_" + strings.Repeat("a", 32)}
	for i, l := range []string{"a", "b"} {
		e.originales.Personas[i] = originalesPersona{Nombre: "Persona sintética " + l, PersonaRef: "per_" + strings.Repeat(l, 32),
			CuentaOrdinariaOriginal: "devord_" + strings.Repeat(l, 32), CuentaPrivilegiadaOriginal: "devadm_" + strings.Repeat(l, 32), SujetoOriginal: "devsuj_" + strings.Repeat(l, 32)}
	}
	e.material = materialComoArranque(t, prov, e.originales)
	if alterar != nil {
		alterar(&e.material)
	}
	materialRaw, err := json.Marshal(e.material)
	if err != nil {
		t.Fatal(err)
	}
	ca := strings.Repeat("c", 64)
	ahora := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	ev := domain.EvidenciaFuentesInicialesAdmin{Referencia: "prc_" + strings.Repeat("e", 32), Version: 1, HuellaSHA256: strings.Repeat("a", 64)}
	e.plan = domain.PlanFuentesInicialesAdminV1{Version: 1, OperacionRef: "pfi_" + strings.Repeat("z", 32), PreparadoEn: ahora, CaducaEn: ahora.Add(time.Hour),
		Entorno: "desarrollo", AlcanceFuente: "sintetico_declarado", Procedencia: ev,
		Organizacion: domain.OrganizacionFuentesInicialesAdmin{OrganizacionRef: e.originales.OrganizacionRef, VigenteHasta: ahora.Add(48 * time.Hour)},
		FuenteHMAC:   domain.EvidenciaFuentesInicialesAdmin{Referencia: e.material.FuenteRef, Version: 1, HuellaSHA256: huellaSHA256(materialRaw)},
		PoliticaADMIN: domain.PoliticaFuentesInicialesAdmin{PoliticaRef: "pga_" + strings.Repeat("p", 32), HostADMIN: "admin.example.invalid", CAHuellaSHA256: ca,
			HuellaAprobacionSHA256: strings.Repeat("d", 64), MaximaEdadRevocacionSegundos: 60, VigenteHasta: ahora.Add(24 * time.Hour)}}
	for i, l := range []string{"a", "b"} {
		e.plan.Personas[i] = domain.PersonaFuentesInicialesAdmin{PersonaRef: e.originales.Personas[i].PersonaRef, VigenteHasta: ahora.Add(24 * time.Hour),
			OperacionCuentaOrdinariaRef: "opr_" + strings.Repeat(l, 32) + "o", OperacionCuentaPrivilegiadaRef: "opr_" + strings.Repeat(l, 32) + "p", FuenteTitularidad: ev}
	}
	_, huellaPlan, err := e.plan.CanonicoYHuella()
	if err != nil {
		t.Fatal(err)
	}
	certs := certificadosArranque{Entorno: "desarrollo"}
	var ca1 datosCA
	var is1 datosIS
	ca1.OrganizacionRef, ca1.OrganizacionVersion, is1.PoliticaRef = e.plan.Organizacion.OrganizacionRef, 1, e.plan.PoliticaADMIN.PoliticaRef
	for i, l := range []string{"1", "2"} {
		ref := e.plan.Personas[i].PersonaRef
		certs.Certificados[i] = certificadoPersona{PersonaRef: ref, NombreSintetico: "Persona sintética", CertDERSHA256: strings.Repeat(l, 64), CADERSHA256: ca}
		ord, adm := "cta_ord"+strings.Repeat(l, 30), "cta_adm"+strings.Repeat(l, 30)
		ca1.Personas[i] = cuentaPersonaCA{ref, ord, adm, 1, 1, 1}
		is1.Personas[1-i] = cuentaPersonaIS{ref, ord, adm, 1}
	}
	fecha := "2026-10-04T12:00:01.123456Z"
	parcial := func(esquema string) (string, uint64, string, string, string, string, string, string) {
		return esquema, 1, "recibo:" + strings.Repeat("9", 32), e.plan.OperacionRef, huellaPlan, "aprobacion", "sintetico_declarado", fecha
	}
	var rca reciboParcial[datosCA]
	rca.Esquema, rca.Version, rca.ReciboRef, rca.OperacionRef, rca.PlanSHA256, rca.AprobacionRef, rca.AlcanceFuente, rca.RegistradaEn = parcial("vec.ca.fuentes-iniciales-admin.v1")
	rca.Datos, rca.HuellaSHA256 = ca1, strings.Repeat("7", 64)
	var ris reciboParcial[datosIS]
	ris.Esquema, ris.Version, ris.ReciboRef, ris.OperacionRef, ris.PlanSHA256, ris.AprobacionRef, ris.AlcanceFuente, ris.RegistradaEn = parcial("vec.is.fuentes-iniciales-admin.v1")
	ris.Datos, ris.HuellaSHA256 = is1, strings.Repeat("8", 64)
	acuse := acuseFuentes{Estado: "permitido", Recibo: &reciboFuentes{Esquema: "vec.admin.fuentes-confirmadas.v1", CA: rca, IS: ris, ReciboRef: "recibo_fuentes:" + strings.Repeat("1", 32),
		OperacionRef: e.plan.OperacionRef, PlanSHA256: huellaPlan, PreimagenSHA256: strings.Repeat("2", 64), ConfiguracionSHA256: strings.Repeat("3", 64), OperadorLogin: "operador",
		AprobacionRef: "aprobacion", AuditoriaRef: "aud_v3_f_" + strings.Repeat("4", 32), AuditoriaSecuencia: 1, AuditoriaHuellaSHA256: strings.Repeat("5", 64), RegistradaEn: fecha},
		AuditoriaIntento: auditoriaAcuse{AuditoriaRef: "aud_v3_fi_" + strings.Repeat("6", 32), Secuencia: 2, HuellaSHA256: strings.Repeat("6", 64), CorrelacionRef: "correlacion_" + strings.Repeat("6", 32), RegistradaEn: fecha}}
	prov.IncluirCuentaOrdinaria = false
	for nombre, v := range map[string]any{"originales": e.originales, "certificados": certs, "fuente": e.plan, "acuse": acuse, "material": materialRaw, "proveedor": prov} {
		escribirJSON(t, filepath.Join(e.dir, nombre+".json"), v, 0600)
		e.archivos[nombre] = v
	}
	return e
}

func (e *escenario) args(t *testing.T) []string {
	t.Helper()
	textos, err := filepath.Abs("../../web/static/textos/es/admin-identificadores-preparar.json")
	if err != nil {
		t.Fatal(err)
	}
	a := []string{"--textos", textos, "--salida", e.salida}
	for _, n := range []string{"originales", "certificados", "fuente", "acuse", "material", "proveedor"} {
		a = append(a, "--"+n, filepath.Join(e.dir, n+".json"))
	}
	return append(a, e.extra...)
}

func reescribirProveedor(t *testing.T, e *escenario, cambiar func(*proveedorHMAC)) {
	t.Helper()
	p := e.archivos["proveedor"].(proveedorHMAC)
	cambiar(&p)
	escribirJSON(t, filepath.Join(e.dir, "proveedor.json"), p, 0600)
}

// configuracionAdmin escribe una configuración privada de vec-admin reducida:
// otros bloques más el bloque identidad, que es lo único que se coteja.
func configuracionAdmin(t *testing.T, e *escenario, cambiar func(*proveedorHMAC)) {
	t.Helper()
	p := e.archivos["proveedor"].(proveedorHMAC)
	p.IncluirCuentaOrdinaria = true
	if cambiar != nil {
		cambiar(&p)
	}
	ruta := filepath.Join(e.dir, "vec-admin.json")
	escribirJSON(t, ruta, map[string]any{"pools": map[string]string{"cuentas_admin": "/privado/pool"}, "identidad": p}, 0600)
	e.extra = []string{"--configuracion-admin", ruta}
}

func TestPreparaConConfiguracionAdminCoincidente(t *testing.T) {
	e := nuevoEscenario(t, nil)
	configuracionAdmin(t, e, nil)
	var salida, errores bytes.Buffer
	if ejecutar(e.args(t), &salida, &errores) != 0 || !strings.Contains(salida.String(), `"identificadores_preparados"`) {
		t.Fatalf("rechazado: %s", errores.String())
	}
}

func TestSalidaRechazadaPorElCargador(t *testing.T) {
	original := cargarFuente
	t.Cleanup(func() { cargarFuente = original })
	for _, caso := range []struct {
		codigo   string
		bloquear bool
	}{{"salida_rechazada_retirada", false}, {"salida_rechazada_sin_retirar", true}} {
		t.Run(caso.codigo, func(t *testing.T) {
			e := nuevoEscenario(t, nil)
			dir := filepath.Dir(e.salida)
			t.Cleanup(func() { _ = os.Chmod(dir, 0700) })
			cargarFuente = func(string, string) (selector.FuenteIdentificadoresADMIN, error) {
				if caso.bloquear && os.Chmod(dir, 0500) != nil {
					t.Fatal("chmod")
				}
				return nil, os.ErrInvalid
			}
			var salida, errores bytes.Buffer
			if ejecutar(e.args(t), &salida, &errores) != 1 || salida.Len() != 0 || !strings.Contains(errores.String(), `"codigo":"`+caso.codigo+`"`) {
				t.Fatalf("resultado inesperado: %s", errores.String())
			}
			if _, err := os.Lstat(e.salida); os.IsNotExist(err) == caso.bloquear {
				t.Fatal("el estado del archivo no coincide con el mensaje")
			}
		})
	}
}

func TestPreparaArchivoQueAceptaElCargadorDelRuntime(t *testing.T) {
	e := nuevoEscenario(t, nil)
	var salida, errores bytes.Buffer
	if ejecutar(e.args(t), &salida, &errores) != 0 || errores.Len() != 0 {
		t.Fatalf("rechazado: %s", errores.String())
	}
	var d diagnostico
	if err := json.Unmarshal(salida.Bytes(), &d); err != nil || d.Codigo != "identificadores_preparados" || !d.Preparado {
		t.Fatal("diagnóstico inesperado")
	}
	for _, p := range e.originales.Personas {
		for _, v := range []string{p.SujetoOriginal, p.CuentaOrdinariaOriginal, p.CuentaPrivilegiadaOriginal} {
			if strings.Contains(salida.String(), v) {
				t.Fatal("el diagnóstico filtra un identificador original")
			}
		}
	}
	info, err := os.Stat(e.salida)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("salida sin permisos privados")
	}
	fuente, err := selector.NuevaFuenteIdentificadoresADMINDesdeArchivo(e.salida, d.IdentificadoresSHA256)
	if err != nil {
		t.Fatal("el cargador del runtime rechaza el archivo con el SHA impreso")
	}
	acuse := e.archivos["acuse"].(acuseFuentes)
	certs := e.archivos["certificados"].(certificadosArranque)
	for i, p := range e.plan.Personas {
		c := acuse.Recibo.CA.Datos.Personas[i]
		ids, err := fuente.ResolverIdentificadoresADMIN(context.Background(), selector.ReferenciaFuenteIdentificadoresADMIN{PersonaRef: p.PersonaRef, CuentaRef: c.CuentaPrivilegiadaRef,
			CuentaOrdinariaRef: c.CuentaOrdinariaRef, CertificadoSHA256: certs.Certificados[i].CertDERSHA256, CASHA256: e.plan.PoliticaADMIN.CAHuellaSHA256,
			EspacioIdentidad: "https://admin.example.invalid/identidad", DominioHMACRef: e.material.DominioRef, ClaveHMACID: e.material.ClaveID, ClaveHMACVersion: e.material.ClaveVersion,
			FuenteRef: e.plan.FuenteHMAC.Referencia, FuenteSHA256: e.plan.FuenteHMAC.HuellaSHA256})
		o := e.originales.Personas[i]
		if err != nil || ids.SujetoID != o.SujetoOriginal || ids.CuentaID != o.CuentaPrivilegiadaOriginal || ids.CuentaOrdinariaID != o.CuentaOrdinariaOriginal {
			t.Fatal("el archivo no resuelve los originales de la persona")
		}
	}
	// Una segunda ejecución nunca sobrescribe la salida.
	salida.Reset()
	if ejecutar(e.args(t), &salida, &errores) == 0 || !strings.Contains(errores.String(), `"salida_insegura"`) {
		t.Fatal("salida existente sobrescrita")
	}
}

func TestRechazosSinEscribirSalida(t *testing.T) {
	for _, caso := range []struct {
		nombre, codigo string
		alterar        func(*materialHMAC)
		cambiar        func(*testing.T, *escenario)
	}{
		{"originales no privados", "entrada_insegura", nil, func(t *testing.T, e *escenario) {
			if os.Chmod(filepath.Join(e.dir, "originales.json"), 0640) != nil {
				t.Fatal("chmod")
			}
		}},
		{"campo extra en originales", "entrada_invalida", nil, func(t *testing.T, e *escenario) {
			ruta := filepath.Join(e.dir, "originales.json")
			b, _ := os.ReadFile(ruta)
			escribirJSON(t, ruta, append(bytes.TrimSuffix(b, []byte("}")), []byte(`,"extra":1}`)...), 0600)
		}},
		{"salida dentro de Git", "salida_insegura", nil, func(t *testing.T, e *escenario) {
			git := filepath.Join(filepath.Dir(e.salida), ".git")
			if os.Mkdir(git, 0700) != nil || os.WriteFile(filepath.Join(git, "HEAD"), []byte("ref: refs/heads/sintetica"), 0600) != nil {
				t.Fatal("git")
			}
		}},
		{"directorio de salida no privado", "salida_insegura", nil, func(t *testing.T, e *escenario) {
			if os.Chmod(filepath.Dir(e.salida), 0755) != nil {
				t.Fatal("chmod")
			}
		}},
		{"acuse con otras cuentas en IS", "entradas_divergentes", nil, func(t *testing.T, e *escenario) {
			a := e.archivos["acuse"].(acuseFuentes)
			r := *a.Recibo
			r.IS.Datos.Personas[0].CuentaOrdinariaRef = "cta_" + strings.Repeat("x", 32)
			a.Recibo = &r
			escribirJSON(t, filepath.Join(e.dir, "acuse.json"), a, 0600)
		}},
		{"material distinto del plan", "entradas_divergentes", nil, func(t *testing.T, e *escenario) {
			m := e.material
			m.ClaveID += "x"
			escribirJSON(t, filepath.Join(e.dir, "material.json"), m, 0600)
		}},
		{"persona repetida en IS", "entradas_divergentes", nil, func(t *testing.T, e *escenario) {
			a := e.archivos["acuse"].(acuseFuentes)
			r := *a.Recibo
			r.IS.Datos.Personas[1] = r.IS.Datos.Personas[0]
			a.Recibo = &r
			escribirJSON(t, filepath.Join(e.dir, "acuse.json"), a, 0600)
		}},
		{"configuración de vec-admin con otro proveedor", "configuracion_admin_divergente", nil, func(t *testing.T, e *escenario) {
			configuracionAdmin(t, e, func(p *proveedorHMAC) { p.DominioHMAC += ".otro" })
		}},
		{"proveedor no disponible", "proveedor_no_disponible", nil, func(t *testing.T, e *escenario) {
			reescribirProveedor(t, e, func(p *proveedorHMAC) { p.RutaConfiguracionHMAC = filepath.Join(e.dir, "no-existe.json") })
		}},
		{"coordenadas divergentes", "coordenadas_divergentes", nil, func(t *testing.T, e *escenario) {
			reescribirProveedor(t, e, func(p *proveedorHMAC) { p.EspacioClave += ".otro" })
		}},
		{"sujeto divergente", "sujeto_divergente", func(m *materialHMAC) { m.Personas[0].Sujeto = strings.Repeat("ef", 32) }, nil},
		{"alias ordinario divergente", "alias_ordinario_divergente", func(m *materialHMAC) { m.Personas[1].CuentaOrdinaria = strings.Repeat("ab", 32) }, nil},
		{"cuenta divergente", "cuenta_divergente", func(m *materialHMAC) { m.Personas[0].CuentaPrivilegiada = strings.Repeat("cd", 32) }, nil},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenario(t, caso.alterar)
			if caso.cambiar != nil {
				caso.cambiar(t, e)
			}
			var salida, errores bytes.Buffer
			if ejecutar(e.args(t), &salida, &errores) != 1 || salida.Len() != 0 || !strings.Contains(errores.String(), `"codigo":"`+caso.codigo+`"`) {
				t.Fatalf("resultado inesperado: %s", errores.String())
			}
			if _, err := os.Lstat(e.salida); !os.IsNotExist(err) {
				t.Fatal("se escribió la salida pese al rechazo")
			}
		})
	}
}

func TestCatalogosCubrenLosMismosCodigos(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		ruta, _ := filepath.Abs("../../web/static/textos/" + idioma + "/admin-identificadores-preparar.json")
		if _, got, err := cargarTextos(ruta); err != nil || got != idioma {
			t.Fatalf("catálogo %s incompleto", idioma)
		}
	}
	var errores bytes.Buffer
	if ejecutar([]string{"--textos", "/no/existe.json"}, &bytes.Buffer{}, &errores) != 2 || strings.TrimSpace(errores.String()) != `{"codigo":"catalogo_no_disponible"}` {
		t.Fatal("sin catálogo debe emitir solo el código de protocolo")
	}
}
