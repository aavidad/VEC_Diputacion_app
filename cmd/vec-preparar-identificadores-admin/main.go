// Command vec-preparar-identificadores-admin prepara el archivo privado v1 de
// identificadores originales que exige vec-admin (fuente_identificadores_archivo).
//
// Reúne lo que el circuito de arranque ya produjo: preimágenes originales,
// certificados, plan de fuentes, acuse confirmado de su aplicación, material
// HMAC oficial y configuración del proveedor. Antes de escribir recalcula las
// HMAC con la misma función y proveedor que usa el runtime y las coteja con el
// material confirmado; ante cualquier diferencia no escribe nada. No conecta
// con PostgreSQL ni imprime identificadores: sólo un código y el SHA256.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"slices"

	"vec-diputacion-granada/internal/shared/i18n"
	selector "vec-diputacion-granada/internal/vec/adapters/httpseguridad/adminperfiles"
	"vec-diputacion-granada/internal/vec/domain"
)

const (
	limiteDocumento = 64 << 10
	// Mismo límite que aplica vec-admin a su configuración privada.
	limiteConfiguracionAdmin = 256 << 10
)

// Ganchos sustituibles solo en pruebas.
var (
	cargarFuente  = selector.NuevaFuenteIdentificadoresADMINDesdeArchivo
	retirarSalida = retirar
)

type diagnostico struct {
	Codigo                string `json:"codigo"`
	Mensaje               string `json:"mensaje"`
	Limite                string `json:"limite"`
	Preparado             bool   `json:"preparado"`
	IdentificadoresSHA256 string `json:"identificadores_sha256,omitempty"`
}

var clavesTextos = []string{"limite", "uso_invalido", "entrada_insegura", "entrada_invalida", "entradas_divergentes", "proveedor_no_disponible",
	"configuracion_admin_divergente", "coordenadas_divergentes", "sujeto_divergente", "cuenta_divergente", "alias_ordinario_divergente", "salida_insegura",
	"salida_rechazada_retirada", "salida_rechazada_sin_retirar", "identificadores_preparados"}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr)) }

func ejecutar(args []string, salida, errores io.Writer) int {
	f := flag.NewFlagSet("vec-preparar-identificadores-admin", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var r rutas
	f.StringVar(&r.originales, "originales", "", "")
	f.StringVar(&r.certificados, "certificados", "", "")
	f.StringVar(&r.fuente, "fuente", "", "")
	f.StringVar(&r.acuse, "acuse", "", "")
	f.StringVar(&r.material, "material", "", "")
	f.StringVar(&r.proveedor, "proveedor", "", "")
	f.StringVar(&r.salida, "salida", "", "")
	f.StringVar(&r.configuracionAdmin, "configuracion-admin", "", "")
	var rutaTextos string
	f.StringVar(&rutaTextos, "textos", "", "")
	parseErr := f.Parse(args)
	textos, idioma, err := cargarTextos(rutaTextos)
	if err != nil {
		return informarFalloCatalogo(errores, err)
	}
	emitir := func(w io.Writer, d diagnostico) int {
		d.Mensaje = textos.T(idioma, d.Codigo)
		d.Limite = textos.T(idioma, "limite")
		if json.NewEncoder(w).Encode(d) != nil {
			return 2
		}
		if !d.Preparado {
			return 1
		}
		return 0
	}
	if parseErr != nil || f.NArg() != 0 || !r.distintas(rutaTextos) {
		return emitir(errores, diagnostico{Codigo: "uso_invalido"})
	}
	doc, err := preparar(r)
	defer clear(doc)
	if err != nil {
		return emitir(errores, diagnostico{Codigo: codigoDe(err)})
	}
	if crearExclusivo(r.salida, doc) != nil {
		return emitir(errores, diagnostico{Codigo: "salida_insegura"})
	}
	// El archivo sólo se da por preparado si lo acepta el cargador del runtime.
	sha := huellaSHA256(doc)
	if _, err := cargarFuente(r.salida, sha); err != nil {
		if retirarSalida(r.salida) != nil {
			return emitir(errores, diagnostico{Codigo: "salida_rechazada_sin_retirar"})
		}
		return emitir(errores, diagnostico{Codigo: "salida_rechazada_retirada"})
	}
	return emitir(salida, diagnostico{Codigo: "identificadores_preparados", Preparado: true, IdentificadoresSHA256: sha})
}

type rutas struct {
	originales, certificados, fuente, acuse, material, proveedor, salida string
	configuracionAdmin                                                   string // opcional
}

func (r rutas) distintas(textos string) bool {
	todas := []string{r.originales, r.certificados, r.fuente, r.acuse, r.material, r.proveedor, r.salida, textos}
	if r.configuracionAdmin != "" {
		todas = append(todas, r.configuracionAdmin)
	}
	for i, x := range todas {
		if x == "" || slices.Contains(todas[:i], x) {
			return false
		}
	}
	return true
}

// preparar lee y coteja todas las entradas; devuelve el documento v1 o el
// código del primer rechazo.
func preparar(r rutas) ([]byte, error) {
	leidos := map[string][]byte{}
	defer func() {
		for _, b := range leidos {
			clear(b)
		}
	}()
	for _, ruta := range []string{r.originales, r.certificados, r.fuente, r.acuse, r.material, r.proveedor} {
		b, err := leerPrivado(ruta)
		if err != nil {
			return nil, fallo("entrada_insegura", err)
		}
		leidos[ruta] = b
	}
	var (
		orig  originalesArranque
		certs certificadosArranque
		plan  domain.PlanFuentesInicialesAdminV1
		acuse acuseFuentes
		mat   materialHMAC
		prov  proveedorHMAC
	)
	for ruta, destino := range map[string]any{r.originales: &orig, r.certificados: &certs, r.fuente: &plan, r.acuse: &acuse, r.material: &mat, r.proveedor: &prov} {
		if err := decodificarEstricto(leidos[ruta], destino); err != nil {
			return nil, fallo("entrada_invalida", err)
		}
	}
	_, huellaPlan, err := plan.CanonicoYHuella()
	if err != nil {
		return nil, fallo("entrada_invalida", err)
	}
	e := entradasCotejadas{plan: plan, originales: orig, certificados: certs, material: mat, proveedor: prov,
		materialSHA256: huellaSHA256(leidos[r.material])}
	cuentas, ok := cuentasDesdeAcuse(acuse, plan, huellaPlan)
	if !ok {
		return nil, fallo("entradas_divergentes", nil)
	}
	e.cuentas = cuentas
	if r.configuracionAdmin != "" {
		if err := cotejarConfiguracionAdmin(r.configuracionAdmin, prov); err != nil {
			return nil, err
		}
	}
	return e.documento()
}

// cotejarConfiguracionAdmin lee solo el bloque "identidad" de la configuración
// privada de vec-admin (sin claves repetidas) y exige el mismo proveedor que
// proveedor.json. Así el archivo se coteja con lo que usará el runtime.
func cotejarConfiguracionAdmin(ruta string, prov proveedorHMAC) error {
	b, err := leerPrivadoHasta(ruta, limiteConfiguracionAdmin)
	if err != nil {
		return fallo("entrada_insegura", err)
	}
	defer clear(b)
	var cfg struct {
		Identidad json.RawMessage `json:"identidad"`
	}
	var admin proveedorHMAC
	if err := errors.Join(clavesUnicas(b), json.Unmarshal(b, &cfg)); err != nil {
		return fallo("entrada_invalida", err)
	}
	if len(cfg.Identidad) == 0 {
		return fallo("entrada_invalida", nil)
	}
	if err := decodificarEstricto(cfg.Identidad, &admin); err != nil {
		return fallo("entrada_invalida", err)
	}
	if !mismoProveedor(admin, prov) {
		return fallo("configuracion_admin_divergente", nil)
	}
	return nil
}

// falloPreparacion lleva el código público del catálogo y conserva la causa
// para quien la inspeccione con errors.Is/As. El diagnóstico solo muestra el
// código: la causa puede contener rutas o contenido privado.
type falloPreparacion struct {
	codigo string
	causa  error
}

func (f *falloPreparacion) Error() string { return f.codigo }
func (f *falloPreparacion) Unwrap() error { return f.causa }

func fallo(codigo string, causa error) error { return &falloPreparacion{codigo: codigo, causa: causa} }

func codigoDe(err error) string {
	var f *falloPreparacion
	if errors.As(err, &f) {
		return f.codigo
	}
	return "entrada_invalida"
}

type datosTextos struct {
	Idioma   string            `json:"idioma"`
	Mensajes map[string]string `json:"mensajes"`
}

func cargarTextos(ruta string) (*i18n.Catalog, string, error) {
	if ruta == "" {
		return nil, "", os.ErrInvalid
	}
	// El catálogo es público, pero se rechazan enlaces y archivos no regulares.
	raiz, err := os.OpenRoot(filepath.Dir(ruta))
	if err != nil {
		return nil, "", err
	}
	defer raiz.Close()
	f, err := raiz.OpenFile(filepath.Base(ruta), os.O_RDONLY|noSeguirEnlaces, 0)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > limiteDocumento {
		return nil, "", os.ErrInvalid
	}
	b, err := io.ReadAll(io.LimitReader(f, limiteDocumento+1))
	if err != nil || len(b) > limiteDocumento {
		return nil, "", os.ErrInvalid
	}
	var datos datosTextos
	if decodificarEstricto(b, &datos) != nil || len(datos.Mensajes) != len(clavesTextos) {
		return nil, "", os.ErrInvalid
	}
	for _, clave := range clavesTextos {
		if datos.Mensajes[clave] == "" {
			return nil, "", os.ErrInvalid
		}
	}
	catalogo, err := i18n.New(datos.Idioma, map[string]map[string]string{datos.Idioma: datos.Mensajes})
	return catalogo, datos.Idioma, err
}
