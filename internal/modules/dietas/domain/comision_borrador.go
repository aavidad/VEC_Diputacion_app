package domain

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const ProcedenciaRutaDeclarada = "declarada_no_verificada"

var ErrBorradorComisionInvalido = errors.New("dietas: borrador de comision invalido")
var referenciaDieta = regexp.MustCompile(`^[a-z][A-Za-z0-9_:-]{2,180}$`)
var codigoRutaDieta = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{1,63}$`)
var dineroDieta = regexp.MustCompile(`^(0|[1-9][0-9]{0,8})\.[0-9]{2}$`)
var cantidadDieta = regexp.MustCompile(`^(0|[1-9][0-9]{0,4})(\.[0-9]{1,4})?$`)
var tarifaDieta = regexp.MustCompile(`^(0|[1-9][0-9]{0,3})\.[0-9]{4}$`)

// El borrador conserva declaraciones del empleado, no una orden autorizada ni
// una liquidación. Los importes son decimales exactos; nunca float64.
type BorradorComision struct {
	PersonaRef                string              `json:"persona_ref"`
	Objeto                    string              `json:"objeto"`
	FechaInicio               string              `json:"fecha"`
	FechaFin                  string              `json:"fecha_fin"`
	HoraInicio                string              `json:"hora_inicio"`
	HoraFin                   string              `json:"hora_fin"`
	ZonaHoraria               string              `json:"zona_horaria"`
	Inicio                    time.Time           `json:"inicio"`
	Fin                       time.Time           `json:"fin"`
	VehiculoPropio            bool                `json:"vehiculo_propio"`
	Ruta                      RutaVersionada      `json:"ruta"`
	Gastos                    GastosComision      `json:"gastos"`
	PoliticaKilometraje       PoliticaKilometraje `json:"politica_kilometraje"`
	Desglose                  DesgloseComision    `json:"desglose"`
	Estado                    string              `json:"estado"`
	ContextoPersonal          string              `json:"contexto_personal"`
	Liquidable                bool                `json:"liquidable"`
	RutaEtiquetas             []string            `json:"ruta_etiquetas"`
	ProcedenciaRuta           string              `json:"procedencia_ruta"`
	RevalidacionRutaRequerida bool                `json:"revalidacion_ruta_requerida"`
}
type GastosComision struct {
	ManutencionEUR string `json:"manutencion_eur"`
	AlojamientoEUR string `json:"alojamiento_eur"`
	OtrosEUR       string `json:"otros_eur"`
}
type PoliticaKilometraje struct {
	Referencia     string `json:"referencia"`
	Version        string `json:"version"`
	TarifaEURPorKM string `json:"tarifa_eur_km"`
}
type DesgloseComision struct {
	KilometrajeEUR string `json:"kilometraje_eur"`
	ManutencionEUR string `json:"manutencion_eur"`
	AlojamientoEUR string `json:"alojamiento_eur"`
	OtrosEUR       string `json:"otros_eur"`
	TotalEUR       string `json:"total_eur"`
}

// RutaVersionada conserva una declaración coherente del cliente. Fuente,
// grafo, catálogo y cálculo NO son evidencia verificada por el servidor.
type RutaVersionada struct {
	Fuente            string       `json:"fuente"`
	Version           string       `json:"version"`
	Referencia        string       `json:"referencia"`
	CatalogoVersion   string       `json:"catalogo_version"`
	AlternativaRef    string       `json:"alternativa_ref"`
	Recomendada       bool         `json:"recomendada"`
	MotivoAlternativa string       `json:"motivo_alternativa"`
	Kilometros        string       `json:"kilometros"`
	Paradas           []ParadaRuta `json:"paradas,omitempty"`
	Tramos            []TramoRuta  `json:"tramos,omitempty"`
	Trazado           [][2]float64 `json:"trazado,omitempty"`
	Liquidable        bool         `json:"liquidable"`
}
type ParadaRuta struct {
	Codigo   string  `json:"codigo"`
	Nombre   string  `json:"nombre"`
	Latitud  float64 `json:"latitud"`
	Longitud float64 `json:"longitud"`
}
type TramoRuta struct {
	OrigenCodigo     string `json:"origen_codigo"`
	DestinoCodigo    string `json:"destino_codigo"`
	Kilometros       string `json:"kilometros"`
	DuracionMinutos  int    `json:"duracion_minutos"`
	AjusteKilometros string `json:"ajuste_kilometros"`
	MotivoAjuste     string `json:"motivo_ajuste"`
}

func (p PoliticaKilometraje) Validar() error {
	n, ok := decimalEscalado(p.TarifaEURPorKM, 4, tarifaDieta, 1000*10000)
	if !referenciaDieta.MatchString(p.Referencia) || !texto(p.Version, 160) || !ok || n < 0 {
		return ErrBorradorComisionInvalido
	}
	return nil
}
func (g GastosComision) Validar() error {
	for _, v := range []string{g.ManutencionEUR, g.AlojamientoEUR, g.OtrosEUR} {
		if _, ok := decimalEscalado(v, 2, dineroDieta, 99999999999); !ok {
			return ErrBorradorComisionInvalido
		}
	}
	return nil
}

func (r RutaVersionada) kilometrosCalculados() (int64, error) {
	if r.Fuente != "osrm_interno" || !texto(r.Version, 160) || !texto(r.CatalogoVersion, 160) || !codigoRutaDieta.MatchString(r.Referencia) || !codigoRutaDieta.MatchString(r.AlternativaRef) || r.Liquidable || len(r.Paradas) < 2 || len(r.Paradas) > 12 || len(r.Tramos) != len(r.Paradas)-1 || len(r.Trazado) < 2 || len(r.Trazado) > 2000 {
		return 0, ErrBorradorComisionInvalido
	}
	if (r.Recomendada && r.MotivoAlternativa != "") || (!r.Recomendada && !texto(r.MotivoAlternativa, 500)) {
		return 0, ErrBorradorComisionInvalido
	}
	for _, p := range r.Paradas {
		if !codigoRutaDieta.MatchString(p.Codigo) || !texto(p.Nombre, 100) || !coordenada(p.Latitud, 90) || !coordenada(p.Longitud, 180) {
			return 0, ErrBorradorComisionInvalido
		}
	}
	for _, p := range r.Trazado {
		if !coordenada(p[0], 90) || !coordenada(p[1], 180) {
			return 0, ErrBorradorComisionInvalido
		}
	}
	var total int64
	for i, t := range r.Tramos {
		km, ok := decimalEscalado(t.Kilometros, 4, cantidadDieta, 10000*10000)
		ajuste, okA := decimalEscalado(t.AjusteKilometros, 4, cantidadDieta, 1000*10000)
		if !ok || !okA || km <= 0 || t.OrigenCodigo != r.Paradas[i].Codigo || t.DestinoCodigo != r.Paradas[i+1].Codigo || t.OrigenCodigo == t.DestinoCodigo || t.DuracionMinutos < 1 || t.DuracionMinutos > 20000 || (ajuste == 0 && t.MotivoAjuste != "") || (ajuste > 0 && !texto(t.MotivoAjuste, 500)) {
			return 0, ErrBorradorComisionInvalido
		}
		total += km + ajuste
	}
	if total > 10000*10000 {
		return 0, ErrBorradorComisionInvalido
	}
	return total, nil
}
func (r RutaVersionada) Validar() error {
	n, e := r.kilometrosCalculados()
	v, ok := decimalEscalado(r.Kilometros, 4, cantidadDieta, 10000*10000)
	if e != nil || !ok || n != v {
		return ErrBorradorComisionInvalido
	}
	return nil
}

// PrepararBorrador resuelve fechas civiles y recalcula el desglose usando una
// política obtenida por el servidor. No acepta el total declarado por el cliente.
func PrepararBorrador(b BorradorComision, p PoliticaKilometraje, zona *time.Location) (BorradorComision, error) {
	if zona == nil || p.Validar() != nil || b.Gastos.Validar() != nil || !referenciaDieta.MatchString(b.PersonaRef) || !texto(b.Objeto, 500) {
		return BorradorComision{}, ErrBorradorComisionInvalido
	}
	inicio, e := ResolverInstanteCivil(b.FechaInicio, b.HoraInicio, zona)
	if e != nil {
		return BorradorComision{}, e
	}
	fin, e := ResolverInstanteCivil(b.FechaFin, b.HoraFin, zona)
	if e != nil || fin.Before(inicio) {
		return BorradorComision{}, ErrBorradorComisionInvalido
	}
	km, e := b.Ruta.kilometrosCalculados()
	if e != nil {
		return BorradorComision{}, e
	}
	// El total de ruta enviado debe concordar: nunca esconder un ajuste cambiado.
	declarado, ok := decimalEscalado(b.Ruta.Kilometros, 4, cantidadDieta, 10000*10000)
	if !ok || declarado != km {
		return BorradorComision{}, ErrBorradorComisionInvalido
	}
	b = b.Clonar()
	b.Inicio = inicio
	b.Fin = fin
	b.ZonaHoraria = zona.String()
	b.Ruta.Kilometros = decimalFormateado(km, 4)
	for i, t := range b.Ruta.Tramos {
		base, _ := decimalEscalado(t.Kilometros, 4, cantidadDieta, 10000*10000)
		ajuste, _ := decimalEscalado(t.AjusteKilometros, 4, cantidadDieta, 1000*10000)
		b.Ruta.Tramos[i].Kilometros = decimalFormateado(base, 4)
		b.Ruta.Tramos[i].AjusteKilometros = decimalFormateado(ajuste, 4)
	}
	b.PoliticaKilometraje = p
	b.Estado = "borrador"
	b.ContextoPersonal = "contexto_personal_pendiente"
	b.Liquidable = false
	b.RutaEtiquetas = make([]string, len(b.Ruta.Paradas))
	for i, p := range b.Ruta.Paradas {
		b.RutaEtiquetas[i] = p.Nombre
	}
	b.ProcedenciaRuta = ProcedenciaRutaDeclarada
	b.RevalidacionRutaRequerida = true
	b.Desglose = calcularDesglose(b.Gastos, km, p, b.VehiculoPropio)
	if !dineroDieta.MatchString(b.Desglose.TotalEUR) {
		return BorradorComision{}, ErrBorradorComisionInvalido
	}
	return b, nil
}
func calcularDesglose(g GastosComision, km int64, p PoliticaKilometraje, vehiculo bool) DesgloseComision {
	tarifa, _ := decimalEscalado(p.TarifaEURPorKM, 4, tarifaDieta, 1000*10000)
	var kilometraje int64
	// Escalas 10^4*10^4 -> céntimos; mitad hacia arriba, sin binarios flotantes.
	if vehiculo {
		kilometraje = (km*tarifa + 500000) / 1000000
	}
	m, _ := decimalEscalado(g.ManutencionEUR, 2, dineroDieta, 99999999999)
	a, _ := decimalEscalado(g.AlojamientoEUR, 2, dineroDieta, 99999999999)
	o, _ := decimalEscalado(g.OtrosEUR, 2, dineroDieta, 99999999999)
	return DesgloseComision{decimalFormateado(kilometraje, 2), g.ManutencionEUR, g.AlojamientoEUR, g.OtrosEUR, decimalFormateado(kilometraje+m+a+o, 2)}
}
func (b BorradorComision) Validar() error {
	z, e := time.LoadLocation(b.ZonaHoraria)
	if e != nil || b.ZonaHoraria == "" || b.ZonaHoraria == "Local" || b.Estado != "borrador" || b.ContextoPersonal != "contexto_personal_pendiente" || b.Liquidable || b.ProcedenciaRuta != ProcedenciaRutaDeclarada || !b.RevalidacionRutaRequerida || !utc(b.Inicio) || !utc(b.Fin) {
		return ErrBorradorComisionInvalido
	}
	esperado, e := PrepararBorrador(b, b.PoliticaKilometraje, z)
	if e != nil || !b.Inicio.Equal(esperado.Inicio) || !b.Fin.Equal(esperado.Fin) || b.Desglose != esperado.Desglose || !slices.Equal(b.RutaEtiquetas, esperado.RutaEtiquetas) {
		return ErrBorradorComisionInvalido
	}
	return nil
}

// Resumen excluye geometría, coordenadas y tramos del listado paginado.
// El detalle completo sólo se recupera con una nueva lectura autorizada.
func (b BorradorComision) Resumen() BorradorComision {
	b.RutaEtiquetas = append([]string(nil), b.RutaEtiquetas...)
	b.Ruta.Paradas, b.Ruta.Tramos, b.Ruta.Trazado = nil, nil, nil
	return b
}

// ValidarResumen valida exclusivamente la proyección ligera. No acredita
// trazado, origen OSRM ni un importe liquidable.
func (b BorradorComision) ValidarResumen() error {
	r := b.Ruta
	if len(b.RutaEtiquetas) < 2 || len(b.RutaEtiquetas) > 12 {
		return ErrBorradorComisionInvalido
	}
	for _, nombre := range b.RutaEtiquetas {
		if !texto(nombre, 100) {
			return ErrBorradorComisionInvalido
		}
	}
	if len(r.Paradas) != 0 || len(r.Tramos) != 0 || len(r.Trazado) != 0 ||
		r.Fuente != "osrm_interno" || !texto(r.Version, 160) || !texto(r.CatalogoVersion, 160) ||
		!codigoRutaDieta.MatchString(r.Referencia) || !codigoRutaDieta.MatchString(r.AlternativaRef) ||
		r.Liquidable || (r.Recomendada && r.MotivoAlternativa != "") || (!r.Recomendada && !texto(r.MotivoAlternativa, 500)) ||
		!referenciaDieta.MatchString(b.PersonaRef) || !texto(b.Objeto, 500) ||
		b.Estado != "borrador" || b.ContextoPersonal != "contexto_personal_pendiente" ||
		b.Liquidable || b.ProcedenciaRuta != ProcedenciaRutaDeclarada || !b.RevalidacionRutaRequerida ||
		b.Gastos.Validar() != nil || b.PoliticaKilometraje.Validar() != nil ||
		!utc(b.Inicio) || !utc(b.Fin) || b.ZonaHoraria == "" || b.ZonaHoraria == "Local" {
		return ErrBorradorComisionInvalido
	}
	km, ok := decimalEscalado(r.Kilometros, 4, cantidadDieta, 10000*10000)
	z, e := time.LoadLocation(b.ZonaHoraria)
	if !ok || km <= 0 || e != nil {
		return ErrBorradorComisionInvalido
	}
	inicio, e := ResolverInstanteCivil(b.FechaInicio, b.HoraInicio, z)
	fin, f := ResolverInstanteCivil(b.FechaFin, b.HoraFin, z)
	if e != nil || f != nil || fin.Before(inicio) || !inicio.Equal(b.Inicio) || !fin.Equal(b.Fin) ||
		b.Desglose != calcularDesglose(b.Gastos, km, b.PoliticaKilometraje, b.VehiculoPropio) {
		return ErrBorradorComisionInvalido
	}
	return nil
}
func (b BorradorComision) Clonar() BorradorComision {
	b.RutaEtiquetas = append([]string(nil), b.RutaEtiquetas...)
	b.Ruta.Paradas = append([]ParadaRuta(nil), b.Ruta.Paradas...)
	b.Ruta.Tramos = append([]TramoRuta(nil), b.Ruta.Tramos...)
	b.Ruta.Trazado = append([][2]float64(nil), b.Ruta.Trazado...)
	return b
}

// ResolverInstanteCivil rechaza horas inexistentes y horas que identifican dos
// instantes durante el cambio de huso. No elige silenciosamente uno de ellos.
func ResolverInstanteCivil(fecha, hora string, zona *time.Location) (time.Time, error) {
	if zona == nil || zona.String() == "Local" || !texto(zona.String(), 100) {
		return time.Time{}, ErrBorradorComisionInvalido
	}
	civil := fecha + "T" + hora
	p, e := time.Parse("2006-01-02T15:04", civil)
	if e != nil || p.Year() < 1 || p.Format("2006-01-02T15:04") != civil || len(fecha) != 10 || len(hora) != 5 {
		return time.Time{}, ErrBorradorComisionInvalido
	}
	offsets := map[int]bool{}
	for h := -48; h <= 48; h++ {
		_, off := p.Add(time.Duration(h) * time.Hour).In(zona).Zone()
		offsets[off] = true
	}
	var resultado time.Time
	coincidencias := 0
	for off := range offsets {
		candidato := p.Add(-time.Duration(off) * time.Second)
		if candidato.In(zona).Format("2006-01-02T15:04") == civil {
			resultado = candidato.UTC()
			coincidencias++
		}
	}
	if coincidencias != 1 {
		return time.Time{}, ErrBorradorComisionInvalido
	}
	return resultado, nil
}
func decimalEscalado(s string, escala int, re *regexp.Regexp, maximo int64) (int64, bool) {
	if !re.MatchString(s) {
		return 0, false
	}
	p := strings.Split(s, ".")
	fraccion := ""
	if len(p) == 2 {
		fraccion = p[1]
	}
	if len(fraccion) > escala {
		return 0, false
	}
	n, e := strconv.ParseInt(p[0]+fraccion+strings.Repeat("0", escala-len(fraccion)), 10, 64)
	return n, e == nil && n <= maximo
}
func decimalFormateado(n int64, escala int) string {
	factor := int64(1)
	for i := 0; i < escala; i++ {
		factor *= 10
	}
	return fmt.Sprintf("%d.%0*d", n/factor, escala, n%factor)
}
func texto(s string, n int) bool {
	return len(s) > 0 && len(s) <= n && s == strings.TrimSpace(s) && !strings.ContainsFunc(s, unicode.IsControl)
}
func coordenada(v, limite float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= -limite && v <= limite
}
func utc(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Nanosecond()%1000 == 0
}
