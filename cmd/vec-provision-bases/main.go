// vec-provision-bases consulta transcripciones públicas para preparar borradores.
// No registra solicitudes ni valida decisiones de RRHH.
package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"syscall"

	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/provision/domain"
)

//go:embed catalogo.json
var datosCatalogo []byte

type fuente struct {
	Tipo   string `json:"tipo"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

type apartado struct {
	Referencia        string `json:"referencia"`
	Estado            string `json:"estado"`
	FamiliaMotor      string `json:"familia_motor,omitempty"`
	MaximoMicropuntos string `json:"maximo_micropuntos,omitempty"`
	Dato              string `json:"dato"`
}

type procesoPublico struct {
	Modalidad                           string     `json:"modalidad"`
	Expediente                          string     `json:"expediente"`
	CVE                                 string     `json:"cve"`
	Estado                              string     `json:"estado"`
	Fuentes                             []fuente   `json:"fuentes"`
	FuentePlazoURL                      string     `json:"fuente_plazo_url,omitempty"`
	PlazoFinInclusivo                   string     `json:"plazo_fin_inclusivo,omitempty"`
	FechaCorteExclusiva                 string     `json:"fecha_corte_exclusiva,omitempty"`
	PuestosAnexoCodigo                  []string   `json:"puestos_anexo_codigo,omitempty"`
	PuestosExcluidosRectificacionCodigo []string   `json:"puestos_excluidos_rectificacion_codigo,omitempty"`
	Apartados                           []apartado `json:"apartados"`
	Contraste                           *contraste `json:"contraste,omitempty"`
}

type catalogo struct {
	SchemaVersion string           `json:"schema_version"`
	Procesos      []procesoPublico `json:"procesos"`
}

type diferencia struct {
	Codigo   string `json:"codigo"`
	Campo    string `json:"campo"`
	Esperado string `json:"esperado,omitempty"`
}

type contraste struct {
	Estado         string       `json:"estado"`
	CoberturaDatos string       `json:"cobertura_datos"`
	OfertaEstado   string       `json:"oferta_estado"`
	Diferencias    []diferencia `json:"diferencias"`
	Pendientes     []string     `json:"pendientes"`
}

type respuesta struct {
	SchemaVersion string           `json:"schema_version"`
	Procesos      []procesoPublico `json:"procesos"`
	Fuente        *verificacion    `json:"fuente,omitempty"`
}

type verificacion struct {
	Tipo   string `json:"tipo"`
	Estado string `json:"estado"`
	SHA256 string `json:"sha256"`
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr)) }

func ejecutar(args []string, out, errout io.Writer) int {
	flags := flag.NewFlagSet("vec-provision-bases", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	expediente := flags.String("expediente", "", "")
	borrador := flags.String("proceso", "", "")
	archivoFuente := flags.String("verificar-fuente", "", "")
	tipoFuente := flags.String("tipo-fuente", "", "")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || (*borrador != "" && *expediente == "") || (*archivoFuente == "") != (*tipoFuente == "") {
		return errorNominal(errout, "argumentos_invalidos")
	}
	var c catalogo
	if err := json.Unmarshal(datosCatalogo, &c); err != nil || c.SchemaVersion != "provision.bases_publicas.v1" {
		return errorNominal(errout, "catalogo_invalido")
	}
	r := respuesta{SchemaVersion: c.SchemaVersion, Procesos: []procesoPublico{}}
	for _, p := range c.Procesos {
		if *expediente == "" || *expediente == p.Expediente {
			r.Procesos = append(r.Procesos, p)
		}
	}
	if len(r.Procesos) == 0 {
		return errorNominal(errout, "expediente_desconocido")
	}
	if *borrador != "" {
		if len(r.Procesos) != 1 || r.Procesos[0].Modalidad != "concurso_general" {
			return errorNominal(errout, "contraste_no_disponible")
		}
		p, err := leerProceso(*borrador)
		if err != nil {
			return errorNominal(errout, "proceso_invalido")
		}
		r.Procesos[0].Contraste = contrastar(r.Procesos[0], p)
	}
	if *archivoFuente != "" {
		var esperada *fuente
		for _, p := range r.Procesos {
			for i := range p.Fuentes {
				if p.Fuentes[i].Tipo == *tipoFuente {
					if esperada != nil {
						return errorNominal(errout, "tipo_fuente_ambiguo")
					}
					esperada = &p.Fuentes[i]
				}
			}
		}
		if esperada == nil {
			return errorNominal(errout, "tipo_fuente_desconocido")
		}
		h, err := huellaArchivo(*archivoFuente)
		if err != nil {
			return errorNominal(errout, "fuente_invalida")
		}
		estado := "coincide"
		if h != esperada.SHA256 {
			estado = "difiere"
		}
		r.Fuente = &verificacion{Tipo: *tipoFuente, Estado: estado, SHA256: h}
	}
	if err := json.NewEncoder(out).Encode(r); err != nil {
		return errorNominal(errout, "salida_fallida")
	}
	return 0
}

func leerProceso(ruta string) (domain.ProcesoProvision, error) {
	var p domain.ProcesoProvision
	f, err := abrirRegular(ruta, simulacion.MaximoBytes)
	if err != nil {
		return p, err
	}
	defer func() { _ = f.Close() }()
	if err := simulacion.Decodificar(f, &p); err != nil {
		return p, err
	}
	if err := domain.ValidarProceso(p); err != nil {
		return p, err
	}
	return p, nil
}

func huellaArchivo(ruta string) (string, error) {
	const maximo int64 = 32 << 20
	f, err := abrirRegular(ruta, maximo)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, maximo+1))
	if err != nil {
		return "", err
	}
	if n > maximo {
		return "", errors.New("archivo_invalido")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func abrirRegular(ruta string, maximo int64) (*os.File, error) {
	// O_NONBLOCK evita quedar esperando a un emisor FIFO antes de comprobar fstat.
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, errors.New("archivo_invalido")
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maximo {
		_ = f.Close()
		return nil, errors.New("archivo_invalido")
	}
	return f, nil
}

func contrastar(c procesoPublico, p domain.ProcesoProvision) *contraste {
	r := &contraste{Estado: "cotejo_parcial", CoberturaDatos: "parcial", OfertaEstado: "no_cotejada", Diferencias: []diferencia{}, Pendientes: []string{
		"codigos_del_anexo_no_cotejados",
		"version_de_bases_y_rectificaciones_por_rrhh",
		"correspondencia_de_puestos_con_rpt_y_dotacion",
		"formulas_y_excepciones_de_cada_apartado",
		"fuentes_de_requisitos_y_meritos_al_corte",
		"adjudicacion_y_desempates_conformes_a_bases",
	}}
	agregar := func(codigo, campo, esperado string) {
		r.Diferencias = append(r.Diferencias, diferencia{Codigo: codigo, Campo: campo, Esperado: esperado})
	}
	if p.Referencia != c.Expediente {
		agregar("convocatoria_distinta", "referencia", c.Expediente)
	}
	if p.Configuracion.BasesRef != c.CVE {
		agregar("base_sin_vinculo_publico", "configuracion.bases_ref", c.CVE)
	}
	if p.Configuracion.FechaCorte.String() != c.FechaCorteExclusiva {
		agregar("fecha_corte_distinta", "configuracion.fecha_corte", c.FechaCorteExclusiva)
	}
	if p.Configuracion.MaximoTotal.String() != "100000000" {
		agregar("maximo_total_distinto", "configuracion.maximo_total", "100000000")
	}
	porFamilia := map[string][]domain.Regla{}
	for _, regla := range p.Configuracion.Reglas {
		porFamilia[string(regla.Familia)] = append(porFamilia[string(regla.Familia)], regla)
	}
	for _, apartado := range c.Apartados {
		if apartado.FamiliaMotor == "" {
			continue
		}
		reglas := porFamilia[apartado.FamiliaMotor]
		if len(reglas) != 1 {
			agregar("familia_sin_regla_unica", "configuracion.reglas."+apartado.FamiliaMotor, apartado.Referencia)
			continue
		}
		if reglas[0].Maximo.String() != apartado.MaximoMicropuntos {
			agregar("maximo_familia_distinto", "configuracion.reglas."+apartado.FamiliaMotor+".maximo", apartado.MaximoMicropuntos)
		}
		if reglas[0].ReferenciaBase != c.CVE+"#"+apartado.Referencia {
			agregar("regla_sin_vinculo_base", "configuracion.reglas."+apartado.FamiliaMotor+".referencia_base", c.CVE+"#"+apartado.Referencia)
		}
	}
	return r
}

func errorNominal(out io.Writer, codigo string) int {
	_ = json.NewEncoder(out).Encode(struct {
		Error struct {
			Codigo string `json:"codigo"`
		} `json:"error"`
	}{Error: struct {
		Codigo string `json:"codigo"`
	}{Codigo: codigo}})
	return 1
}
