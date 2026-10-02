package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/personal/adapters/httpinterno"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/web"
)

const (
	esquemaConsultaNominal             = "vec.personal.comparacion-organizacion.consulta.v1"
	esquemaManifiestoConsultaNominal   = "vec.personal.comparacion-organizacion.consulta.manifiesto.v1"
	limiteConfiguracionConsultaNominal = 16 << 10
	tiempoConsultaNominal              = 2 * time.Minute
)

type configuracionConsultaNominal struct {
	Version            int    `json:"version"`
	Origen             string `json:"origen"`
	AutoridadCA        string `json:"autoridad_ca"`
	CertificadoCliente string `json:"certificado_cliente"`
	ClaveCliente       string `json:"clave_cliente"`
	MaximoBytesPagina  int64  `json:"maximo_bytes_pagina"`
	directorio         string
}

type entradaConsultaNominal struct {
	Esquema string                                 `json:"esquema"`
	Formato string                                 `json:"formato"`
	Antes   personal.SelectorOrganizacionHistorica `json:"antes"`
	Despues personal.SelectorOrganizacionHistorica `json:"despues"`
}

// La evidencia pertenece a cada lectura del servidor. No se genera recibo de
// comparación, ni se conserva otra copia de las colecciones recibidas.
type lecturaConsultaNominal struct {
	Selector            personal.SelectorOrganizacionHistorica               `json:"selector"`
	Evidencia           personalports.EvidenciaConsultaOrganizacionHistorica `json:"evidencia"`
	VersionRPTRef       string                                               `json:"version_rpt_ref"`
	VersionPlantillaRef string                                               `json:"version_plantilla_ref"`
	Cobertura           personalports.CoberturaFuentesOrganizacionHistorica  `json:"cobertura"`
	CursorSiguiente     string                                               `json:"cursor_siguiente"`
}

type manifiestoConsultaNominal struct {
	Esquema            string                                    `json:"esquema"`
	Modo               string                                    `json:"modo"`
	Comparacion        personal.ComparacionOrganizacionHistorica `json:"comparacion"`
	LecturasAntes      []lecturaConsultaNominal                  `json:"lecturas_antes"`
	LecturasDespues    []lecturaConsultaNominal                  `json:"lecturas_despues"`
	TotalHechosAntes   int                                       `json:"total_hechos_antes"`
	TotalHechosDespues int                                       `json:"total_hechos_despues"`
	PaginasAntes       int                                       `json:"paginas_antes"`
	PaginasDespues     int                                       `json:"paginas_despues"`
}

type salidaConsultaNominal struct {
	Manifiesto   manifiestoConsultaNominal `json:"manifiesto"`
	HuellaSHA256 string                    `json:"huella_sha256"`
}

type clienteConsultaNominal interface {
	Consultar(context.Context, personal.SelectorOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error)
	Cerrar()
}
type fabricaConsultaNominal func(configuracionConsultaNominal) (clienteConsultaNominal, error)

func nuevaConsultaNominal(c configuracionConsultaNominal) (clienteConsultaNominal, error) {
	return httpinterno.NuevoClienteOrganizacionHistorica(httpinterno.ConfiguracionOrganizacionHistorica{
		Origen: c.Origen, Directorio: c.directorio, AutoridadCA: c.AutoridadCA,
		CertificadoCliente: c.CertificadoCliente, ClaveCliente: c.ClaveCliente,
		MaximoBytesPagina: c.MaximoBytesPagina,
	})
}

func ejecutarConsultaNominal(ruta string, input io.Reader, output, errores io.Writer, fabrica fabricaConsultaNominal) int {
	catalogo, _, err := web.CatalogoComparacionOrganizacion()
	if err != nil {
		return codigoErrorSalida(err)
	}
	fallo := func(key string) int {
		if _, err := io.WriteString(errores, catalogo.T(catalogo.DefaultLocale(), key)+"\n"); err != nil {
			return codigoErrorSalida(err)
		}
		return 1
	}
	var entrada entradaConsultaNominal
	if decodificarConsultaNominal(input, limiteEntrada, &entrada) != nil || entrada.Esquema != esquemaConsultaNominal || entrada.Formato != "json" ||
		entrada.Antes.Validar() != nil || entrada.Despues.Validar() != nil || entrada.Antes.Cursor != "" || entrada.Despues.Cursor != "" ||
		entrada.Antes.OrganismoRef != entrada.Despues.OrganismoRef || entrada.Antes.UnidadClave != entrada.Despues.UnidadClave {
		return fallo("error_entrada")
	}
	cfg, err := leerConfiguracionConsultaNominal(ruta)
	if err != nil || fabrica == nil {
		return fallo("error_entrada")
	}
	cliente, err := fabrica(cfg)
	if err != nil || cliente == nil {
		return fallo("error_entrada")
	}
	defer cliente.Cerrar()
	ctx, cancel := context.WithTimeout(context.Background(), tiempoConsultaNominal)
	defer cancel()
	vistas := map[string]bool{}
	a, la, err := reunirConsultaNominal(ctx, cliente, entrada.Antes, vistas)
	if err != nil {
		return fallo("error_entrada")
	}
	d, ld, err := reunirConsultaNominal(ctx, cliente, entrada.Despues, vistas)
	if err != nil {
		return fallo("error_entrada")
	}
	comparacion, err := personal.CompararOrganizacionHistorica(a, d)
	if err != nil || ctx.Err() != nil {
		return fallo("error_entrada")
	}
	m := manifiestoConsultaNominal{
		Esquema: esquemaManifiestoConsultaNominal, Modo: "consulta_autorizada", Comparacion: comparacion,
		LecturasAntes: la, LecturasDespues: ld, TotalHechosAntes: totalHechosConsultaNominal(a), TotalHechosDespues: totalHechosConsultaNominal(d),
		PaginasAntes: len(la), PaginasDespues: len(ld),
	}
	canonico, err := json.Marshal(m)
	if err != nil {
		return fallo("error_salida")
	}
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(salidaConsultaNominal{Manifiesto: m, HuellaSHA256: huella(canonico)}); err != nil {
		return fallo("error_salida")
	}
	if ctx.Err() != nil {
		return fallo("error_entrada")
	}
	if n, err := output.Write(b.Bytes()); err != nil || n != b.Len() {
		return fallo("error_salida")
	}
	return 0
}

func decodificarConsultaNominal(input io.Reader, limite int64, destino any) error {
	datos, err := io.ReadAll(io.LimitReader(input, limite+1))
	if err != nil || int64(len(datos)) > limite || !utf8.Valid(datos) || !json.Valid(datos) || clavesUnicas(datos) != nil || formaConsultaNominal(datos, destino) != nil {
		return personal.ErrConsultaOrganizacionHistoricaInvalida
	}
	d := json.NewDecoder(bytes.NewReader(datos))
	d.DisallowUnknownFields()
	if err := d.Decode(destino); err != nil {
		return personal.ErrConsultaOrganizacionHistoricaInvalida
	}
	return nil
}

// DisallowUnknownFields también admite variantes de mayúsculas que pueden
// colapsar dos claves distintas sobre el mismo campo. El esquema nominal usa
// únicamente los nombres literales publicados para cada objeto.
func formaConsultaNominal(datos []byte, destino any) error {
	var campos []string
	switch destino.(type) {
	case *configuracionConsultaNominal:
		campos = []string{"version", "origen", "autoridad_ca", "certificado_cliente", "clave_cliente", "maximo_bytes_pagina"}
	case *entradaConsultaNominal:
		campos = []string{"esquema", "formato", "antes", "despues"}
	default:
		return personal.ErrConsultaOrganizacionHistoricaInvalida
	}
	objeto, err := camposConsultaNominal(datos, campos)
	if err != nil {
		return err
	}
	if _, entrada := destino.(*entradaConsultaNominal); entrada {
		for _, corte := range []string{"antes", "despues"} {
			if _, err := camposConsultaNominal(objeto[corte], []string{"organismo_ref", "unidad_clave", "vigente_en", "conocido_en", "version_rpt_ref", "version_plantilla_ref", "limite", "cursor"}); err != nil {
				return err
			}
		}
	}
	return nil
}

func camposConsultaNominal(datos []byte, admitidos []string) (map[string]json.RawMessage, error) {
	var objeto map[string]json.RawMessage
	if json.Unmarshal(datos, &objeto) != nil || objeto == nil {
		return nil, personal.ErrConsultaOrganizacionHistoricaInvalida
	}
	permitidos := map[string]bool{}
	for _, clave := range admitidos {
		permitidos[clave] = true
	}
	for clave := range objeto {
		if !permitidos[clave] {
			return nil, personal.ErrConsultaOrganizacionHistoricaInvalida
		}
	}
	return objeto, nil
}

// El archivo se abre dentro de un directorio privado comprobado, sin seguir un
// enlace del archivo ni bloquearse al recibir una FIFO. Su contenido nunca se
// devuelve al invocante ni se incorpora al manifiesto.
func leerConfiguracionConsultaNominal(ruta string) (configuracionConsultaNominal, error) {
	var cfg configuracionConsultaNominal
	if !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta {
		return cfg, personal.ErrConsultaOrganizacionHistoricaInvalida
	}
	dir := filepath.Dir(ruta)
	real, err := filepath.EvalSymlinks(dir)
	if err != nil || real != dir {
		return cfg, personal.ErrConsultaOrganizacionHistoricaInvalida
	}
	for p := dir; ; p = filepath.Dir(p) {
		ancestro, err := os.OpenRoot(p)
		if err != nil {
			return cfg, personal.ErrConsultaOrganizacionHistoricaInvalida
		}
		_, gitErr := ancestro.Lstat(".git")
		closeErr := ancestro.Close()
		if !os.IsNotExist(gitErr) || closeErr != nil {
			return cfg, personal.ErrConsultaOrganizacionHistoricaInvalida
		}
		if p == filepath.Dir(p) {
			break
		}
	}
	raiz, err := os.OpenRoot(dir)
	if err != nil {
		return cfg, personal.ErrConsultaOrganizacionHistoricaInvalida
	}
	defer raiz.Close()
	info, err := raiz.Stat(".")
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return cfg, personal.ErrConsultaOrganizacionHistoricaInvalida
	}
	f, err := raiz.OpenFile(filepath.Base(ruta), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return cfg, personal.ErrConsultaOrganizacionHistoricaInvalida
	}
	defer f.Close()
	info, err = f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > limiteConfiguracionConsultaNominal {
		return cfg, personal.ErrConsultaOrganizacionHistoricaInvalida
	}
	if decodificarConsultaNominal(f, limiteConfiguracionConsultaNominal, &cfg) != nil || cfg.Version != 1 ||
		cfg.Origen == "" || cfg.AutoridadCA == "" || cfg.CertificadoCliente == "" || cfg.ClaveCliente == "" || cfg.MaximoBytesPagina < 1 || cfg.MaximoBytesPagina > math.MaxInt64-1 {
		return configuracionConsultaNominal{}, personal.ErrConsultaOrganizacionHistoricaInvalida
	}
	cfg.directorio = dir
	return cfg, nil
}

func reunirConsultaNominal(ctx context.Context, cliente clienteConsultaNominal, selector personal.SelectorOrganizacionHistorica, vistas map[string]bool) (personal.InstantaneaComparacionOrganizacion, []lecturaConsultaNominal, error) {
	var vacio personal.InstantaneaComparacionOrganizacion
	r, err := personal.NuevaReunionPaginasOrganizacion(selector)
	if err != nil {
		return vacio, nil, err
	}
	lecturas := []lecturaConsultaNominal{}
	for n := 0; n < personal.LimitePaginasComparacionOrganizacion; n++ {
		if err := ctx.Err(); err != nil {
			return vacio, nil, err
		}
		resultado, err := cliente.Consultar(ctx, selector)
		if err != nil {
			return vacio, nil, err
		}
		if err := ctx.Err(); err != nil {
			return vacio, nil, err
		}
		p := resultado.Pagina
		if !evidenciaConsultaNominalUnica(resultado.Evidencia, selector.OrganismoRef, vistas) {
			return vacio, nil, personal.ErrOrganizacionHistoricaNoDisponible
		}
		i := personal.InstantaneaComparacionOrganizacion{Selector: p.Selector, Cobertura: personal.CoberturaComparacionOrganizacion(p.Cobertura),
			Unidades: p.Unidades, PuestosTipo: p.PuestosTipo, Dotaciones: p.Dotaciones, Plazas: p.Plazas, PuestosIndividuales: p.PuestosIndividuales, Vinculos: p.Vinculos}
		if err := r.Agregar(personal.PaginaInstantaneaOrganizacion{Instantanea: i, VersionRPTRef: p.VersionRPTRef, VersionPlantillaRef: p.VersionPlantillaRef, CursorSiguiente: p.CursorSiguiente}); err != nil {
			return vacio, nil, err
		}
		lecturas = append(lecturas, lecturaConsultaNominal{Selector: p.Selector, Evidencia: resultado.Evidencia,
			VersionRPTRef: p.VersionRPTRef, VersionPlantillaRef: p.VersionPlantillaRef, Cobertura: p.Cobertura, CursorSiguiente: p.CursorSiguiente})
		if p.CursorSiguiente == "" {
			i, err := r.Finalizar()
			return i, lecturas, err
		}
		selector.Cursor = p.CursorSiguiente
	}
	return vacio, nil, personal.ErrOrganizacionHistoricaNoDisponible
}

func evidenciaConsultaNominalUnica(e personalports.EvidenciaConsultaOrganizacionHistorica, organismo string, vistas map[string]bool) bool {
	_, offset := e.ConsultadaEn.Zone()
	if e.EfectoRef != organismo || e.ConsultadaEn.IsZero() || offset != 0 || e.ConsultadaEn.Nanosecond()%1000 != 0 || len(e.ConsumoHuellaSHA256) != 64 {
		return false
	}
	for _, c := range e.ConsumoHuellaSHA256 {
		if !(c >= 'a' && c <= 'f' || c >= '0' && c <= '9') {
			return false
		}
	}
	valores := []string{e.ReciboRef, e.DecisionRef, e.AuditoriaRef, e.ConsumoHuellaSHA256}
	for n, v := range valores {
		if n < 3 {
			if len(v) < 3 || len(v) > 160 || v[0] < 'a' || v[0] > 'z' {
				return false
			}
			for _, c := range v {
				if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == ':' || c == '-') {
					return false
				}
			}
		}
		// El prefijo de campo evita confundir referencias de autoridades distintas.
		clave := string(rune('0'+n)) + ":" + v
		if vistas[clave] {
			return false
		}
	}
	for n, v := range valores {
		vistas[string(rune('0'+n))+":"+v] = true
	}
	return true
}

func totalHechosConsultaNominal(i personal.InstantaneaComparacionOrganizacion) int {
	return len(i.Unidades) + len(i.PuestosTipo) + len(i.Dotaciones) + len(i.Plazas) + len(i.PuestosIndividuales) + len(i.Vinculos)
}
