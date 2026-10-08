package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const MaximoInstantaneaCatalogoNecesidadesAltaBytes = 8192

var patronCodigoNecesidad = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,79}$`)

// ErrNecesidadAltaInvalida evita que una opción de ejemplo se interprete como
// acreditación de plaza, ocupación, financiación o modalidad contractual.
var ErrNecesidadAltaInvalida = errors.New("contratacion temporal: necesidad de alta invalida")

// CausaNecesidadAlta describe la petición del centro. La modalidad jurídica
// posterior es una decisión distinta de RRHH.
type CausaNecesidadAlta struct {
	Clave              ClaveCatalogo `json:"clave"`
	EtiquetaClave      string        `json:"etiqueta_clave"`
	FuenteRef          string        `json:"fuente_ref"`
	FuenteURL          string        `json:"fuente_url"`
	ReglaRef           string        `json:"regla_ref"`
	FechaFin           string        `json:"fecha_fin"`
	CausaFin           ClaveCatalogo `json:"causa_fin,omitempty"`
	MaximoMeses        uint8         `json:"maximo_meses"`
	VentanaMeses       uint8         `json:"ventana_meses,omitempty"`
	CamposPermitidos   []string      `json:"campos_permitidos"`
	CamposObligatorios []string      `json:"campos_obligatorios"`
	UnoDe              [][]string    `json:"uno_de,omitempty"`
}

// CatalogoNecesidadesAlta es una publicación de ejemplo, versionada y con
// procedencia. La huella del contenido se añade al cargarlo, sin confiar en
// una huella declarada dentro del propio fichero.
type CatalogoNecesidadesAlta struct {
	Esquema                  string               `json:"esquema"`
	Referencia               string               `json:"referencia"`
	Version                  uint64               `json:"version"`
	EsEjemplo                bool                 `json:"es_ejemplo"`
	FuenteRef                string               `json:"fuente_ref"`
	FuenteURL                string               `json:"fuente_url"`
	JornadaReferenciaMinutos uint16               `json:"jornada_referencia_minutos"`
	JornadaFuenteRef         string               `json:"jornada_fuente_ref"`
	Causas                   []CausaNecesidadAlta `json:"causas"`
	HuellaSHA256             string               `json:"-"`
	ContenidoCanonico        []byte               `json:"-"`
}

func (c CatalogoNecesidadesAlta) Validar() error {
	if c.Esquema != "vec.ct.necesidades_alta.v1" || !referenciaValida(c.Referencia) ||
		c.Version == 0 || c.Version > 1<<53-1 ||
		!huellaValida(c.HuellaSHA256) ||
		len(c.ContenidoCanonico) == 0 || len(c.ContenidoCanonico) > MaximoInstantaneaCatalogoNecesidadesAltaBytes ||
		huellaContenidoNecesidadesAlta(c.ContenidoCanonico) != c.HuellaSHA256 ||
		!referenciaValida(c.FuenteRef) ||
		!referenciaValida(c.JornadaFuenteRef) ||
		c.JornadaReferenciaMinutos == 0 || c.JornadaReferenciaMinutos > 7*24*60 ||
		len(c.Causas) == 0 || len(c.Causas) > 32 {
		return ErrNecesidadAltaInvalida
	}
	if err := validarFuentePublica(c.FuenteURL); err != nil {
		return err
	}
	vistas := make(map[ClaveCatalogo]bool, len(c.Causas))
	for _, causa := range c.Causas {
		if err := validarFuentePublica(causa.FuenteURL); err != nil {
			return err
		}
		if !causa.Clave.Valida() || vistas[causa.Clave] ||
			!referenciaValida(causa.FuenteRef) ||
			!referenciaValida(causa.ReglaRef) ||
			!ClaveCatalogo(causa.EtiquetaClave).Valida() ||
			causa.MaximoMeses == 0 || causa.MaximoMeses > 120 ||
			(causa.VentanaMeses != 0 && causa.VentanaMeses < causa.MaximoMeses) {
			return ErrNecesidadAltaInvalida
		}
		switch causa.FechaFin {
		case "obligatoria":
			if causa.CausaFin != "" {
				return ErrNecesidadAltaInvalida
			}
		case "opcional", "no_aplica":
			if !causa.CausaFin.Valida() {
				return ErrNecesidadAltaInvalida
			}
		default:
			return ErrNecesidadAltaInvalida
		}
		permitidos := make(map[string]bool, len(causa.CamposPermitidos))
		for _, campo := range causa.CamposPermitidos {
			if !campoNecesidadConocido(campo) || permitidos[campo] {
				return ErrNecesidadAltaInvalida
			}
			permitidos[campo] = true
		}
		obligatorios := make(map[string]bool, len(causa.CamposObligatorios))
		for _, campo := range causa.CamposObligatorios {
			if !permitidos[campo] || obligatorios[campo] {
				return ErrNecesidadAltaInvalida
			}
			obligatorios[campo] = true
		}
		for _, grupo := range causa.UnoDe {
			if len(grupo) < 2 || len(grupo) > 4 {
				return ErrNecesidadAltaInvalida
			}
			visto := map[string]bool{}
			for _, campo := range grupo {
				if !permitidos[campo] || visto[campo] || obligatorios[campo] {
					return ErrNecesidadAltaInvalida
				}
				visto[campo] = true
			}
		}
		vistas[causa.Clave] = true
	}
	return nil
}

func huellaContenidoNecesidadesAlta(contenido []byte) string {
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:])
}

// RestaurarCatalogoNecesidadesAlta verifica una instantánea conservada, sin
// consultar ni reinterpretar una publicación posterior.
func RestaurarCatalogoNecesidadesAlta(contenido []byte) (CatalogoNecesidadesAlta, error) {
	if len(contenido) == 0 || len(contenido) > MaximoInstantaneaCatalogoNecesidadesAltaBytes {
		return CatalogoNecesidadesAlta{}, ErrNecesidadAltaInvalida
	}
	var c CatalogoNecesidadesAlta
	lector := json.NewDecoder(bytes.NewReader(contenido))
	lector.DisallowUnknownFields()
	if lector.Decode(&c) != nil || lector.Decode(&struct{}{}) != io.EOF {
		return CatalogoNecesidadesAlta{}, ErrNecesidadAltaInvalida
	}
	c.ContenidoCanonico = append([]byte(nil), contenido...)
	c.HuellaSHA256 = huellaContenidoNecesidadesAlta(contenido)
	if c.Validar() != nil {
		return CatalogoNecesidadesAlta{}, ErrNecesidadAltaInvalida
	}
	return c, nil
}

// validarFuentePublica exige un enlace https sin credenciales, puerto, consulta
// ni fragmento a una fuente oficial admitida (BOE o Diputación de Granada).
func validarFuentePublica(valor string) error {
	u, err := url.Parse(valor)
	if err != nil {
		return fmt.Errorf("%w: fuente pública: %w", ErrNecesidadAltaInvalida, err)
	}
	if u.Scheme != "https" || u.User != nil || u.Port() != "" ||
		u.RawQuery != "" || u.Fragment != "" || u.Path == "" {
		return ErrNecesidadAltaInvalida
	}
	switch u.Hostname() {
	case "www.boe.es", "www.dipgra.es", "bop.dipgra.es":
		return nil
	}
	return ErrNecesidadAltaInvalida
}

// CausasAdmitidas filtra exclusivamente por claves que otro componente haya
// obtenido de una configuración admitida; no examina categoría, RPT o puesto.
// Devuelve preselección sólo si queda exactamente una causa.
func (c CatalogoNecesidadesAlta) CausasAdmitidas(claves []ClaveCatalogo) ([]CausaNecesidadAlta, ClaveCatalogo, error) {
	if c.Validar() != nil {
		return nil, "", ErrNecesidadAltaInvalida
	}
	admitidas := make(map[ClaveCatalogo]bool, len(claves))
	for _, clave := range claves {
		if !clave.Valida() || admitidas[clave] {
			return nil, "", ErrNecesidadAltaInvalida
		}
		admitidas[clave] = true
	}
	resultado := make([]CausaNecesidadAlta, 0, len(claves))
	for _, causa := range c.Causas {
		if admitidas[causa.Clave] {
			copia := causa
			copia.CamposPermitidos = append([]string(nil), causa.CamposPermitidos...)
			copia.CamposObligatorios = append([]string(nil), causa.CamposObligatorios...)
			copia.UnoDe = make([][]string, len(causa.UnoDe))
			for i, grupo := range causa.UnoDe {
				copia.UnoDe[i] = append([]string(nil), grupo...)
			}
			resultado = append(resultado, copia)
		}
	}
	if len(resultado) != len(claves) {
		return nil, "", ErrNecesidadAltaInvalida
	}
	if len(resultado) == 1 {
		return resultado, resultado[0].Clave, nil
	}
	return resultado, "", nil
}

// DatosNecesidadAlta recoge datos estructurados; las referencias nominales son
// opacas y su presencia no acredita por sí misma vacancia, titular ni crédito.
type DatosNecesidadAlta struct {
	Esquema              string            `json:"esquema"`
	CatalogoRef          string            `json:"catalogo_ref"`
	CatalogoVersion      uint64            `json:"catalogo_version"`
	CatalogoHuellaSHA256 string            `json:"catalogo_huella_sha256"`
	CausaClave           ClaveCatalogo     `json:"causa_clave"`
	Periodo              PeriodoPrevisto   `json:"periodo"`
	JornadaMinutos       uint16            `json:"jornada_minutos"`
	Campos               map[string]string `json:"campos"`
	CatalogoInstantanea  []byte            `json:"catalogo_instantanea"`
}

func (c CatalogoNecesidadesAlta) ValidarDatos(d DatosNecesidadAlta) error {
	if c.Validar() != nil || d.Esquema != "vec.ct.necesidad_alta.v1" ||
		d.CatalogoRef != c.Referencia ||
		d.CatalogoVersion != c.Version || d.CatalogoHuellaSHA256 != c.HuellaSHA256 ||
		(len(d.CatalogoInstantanea) != 0 && !bytes.Equal(d.CatalogoInstantanea, c.ContenidoCanonico)) ||
		!d.CausaClave.Valida() || d.Periodo.Validar() != nil ||
		d.JornadaMinutos == 0 || d.JornadaMinutos > 7*24*60 {
		return ErrNecesidadAltaInvalida
	}
	var causa *CausaNecesidadAlta
	for i := range c.Causas {
		if c.Causas[i].Clave == d.CausaClave {
			causa = &c.Causas[i]
			break
		}
	}
	if causa == nil {
		return ErrNecesidadAltaInvalida
	}
	politicaEsperada := PoliticaFin{
		ReglaRef: causa.ReglaRef, CatalogoVersion: c.Version,
		CatalogoHuellaSHA256: c.HuellaSHA256, FechaFin: causa.FechaFin,
		CausaFin: causa.CausaFin,
	}
	if d.Periodo.PoliticaFin != (PoliticaFin{}) && d.Periodo.PoliticaFin != politicaEsperada {
		return ErrNecesidadAltaInvalida
	}
	if d.Periodo.Fin.IsZero() {
		if causa.FechaFin == "obligatoria" || d.Periodo.CausaFin != causa.CausaFin {
			return ErrNecesidadAltaInvalida
		}
	} else {
		if causa.FechaFin == "no_aplica" ||
			!d.Periodo.Fin.Before(finExclusivoTrasMeses(d.Periodo.Inicio, int(causa.MaximoMeses))) {
			return ErrNecesidadAltaInvalida
		}
	}
	if programaFin := d.Campos["programa_fin"]; programaFin != "" {
		fecha, err := time.Parse("2006-01-02", programaFin)
		if err != nil || fecha.Before(d.Periodo.Inicio) || fecha.Before(d.Periodo.Fin) {
			return ErrNecesidadAltaInvalida
		}
	}
	permitidos := make(map[string]bool, len(causa.CamposPermitidos))
	for _, campo := range causa.CamposPermitidos {
		permitidos[campo] = true
	}
	for campo, valor := range d.Campos {
		if !permitidos[campo] || !valorCampoNecesidadValido(campo, valor) {
			return ErrNecesidadAltaInvalida
		}
	}
	for _, campo := range causa.CamposObligatorios {
		if d.Campos[campo] == "" {
			return ErrNecesidadAltaInvalida
		}
	}
	for _, grupo := range causa.UnoDe {
		cuenta := 0
		for _, campo := range grupo {
			if d.Campos[campo] != "" {
				cuenta++
			}
		}
		if cuenta != 1 {
			return ErrNecesidadAltaInvalida
		}
	}
	return nil
}

// SellarDatos adjunta al dato validado la publicación completa que regía la
// decisión. El cliente aporta datos; esta instantánea procede del servidor.
func (c CatalogoNecesidadesAlta) SellarDatos(d DatosNecesidadAlta) (DatosNecesidadAlta, error) {
	if len(d.CatalogoInstantanea) != 0 || c.ValidarDatos(d) != nil {
		return DatosNecesidadAlta{}, ErrNecesidadAltaInvalida
	}
	copia := d.clonar()
	if copia.Periodo.Fin.IsZero() && copia.Periodo.PoliticaFin == (PoliticaFin{}) {
		for _, causa := range c.Causas {
			if causa.Clave == copia.CausaClave {
				copia.Periodo.PoliticaFin = PoliticaFin{
					ReglaRef: causa.ReglaRef, CatalogoVersion: c.Version,
					CatalogoHuellaSHA256: c.HuellaSHA256, FechaFin: causa.FechaFin,
					CausaFin: causa.CausaFin,
				}
				break
			}
		}
	}
	copia.CatalogoInstantanea = append([]byte(nil), c.ContenidoCanonico...)
	if copia.ValidarInstantanea() != nil {
		return DatosNecesidadAlta{}, ErrNecesidadAltaInvalida
	}
	return copia, nil
}

func (d DatosNecesidadAlta) ValidarInstantanea() error {
	catalogo, err := RestaurarCatalogoNecesidadesAlta(d.CatalogoInstantanea)
	if err != nil || catalogo.ValidarDatos(d) != nil {
		return ErrNecesidadAltaInvalida
	}
	return nil
}

func (d DatosNecesidadAlta) clonar() DatosNecesidadAlta {
	copia := d
	copia.Campos = make(map[string]string, len(d.Campos))
	for clave, valor := range d.Campos {
		copia.Campos[clave] = valor
	}
	copia.CatalogoInstantanea = append([]byte(nil), d.CatalogoInstantanea...)
	return copia
}

func (d DatosNecesidadAlta) Clonar() (DatosNecesidadAlta, error) {
	if d.ValidarInstantanea() != nil {
		return DatosNecesidadAlta{}, ErrNecesidadAltaInvalida
	}
	return d.clonar(), nil
}

// finExclusivoTrasMeses aplica el límite a fechas civiles, sin el desborde de
// AddDate cuando el mes de destino carece del día inicial (p. ej. 31/05).
func finExclusivoTrasMeses(inicio time.Time, meses int) time.Time {
	primerDia := time.Date(inicio.Year(), inicio.Month()+time.Month(meses), 1, 0, 0, 0, 0, time.UTC)
	siguienteMes := primerDia.AddDate(0, 1, 0)
	ultimoDia := siguienteMes.AddDate(0, 0, -1).Day()
	if inicio.Day() > ultimoDia {
		return siguienteMes
	}
	return primerDia.AddDate(0, 0, inicio.Day()-1)
}

func campoNecesidadConocido(campo string) bool {
	switch campo {
	case "plaza_codigo", "puesto_codigo", "titular_ref", "justificacion_temporal",
		"numero_personas",
		"programa_denominacion", "programa_fin", "proyecto_codigo",
		"financiacion_ref", "rc_ref", "intervencion_ref", "vacancia_fuente_ref",
		"rpt_catalogo_ref", "rpt_catalogo_huella_sha256",
		"organica_codigo", "funcional_codigo", "proyecto_gasto_codigo",
		"porcentaje_financiacion":
		return true
	}
	return false
}

func valorCampoNecesidadValido(campo, valor string) bool {
	if valor == "" || strings.TrimSpace(valor) != valor {
		return false
	}
	switch campo {
	case "titular_ref", "financiacion_ref", "rc_ref", "intervencion_ref", "vacancia_fuente_ref", "rpt_catalogo_ref":
		return referenciaValida(valor)
	case "rpt_catalogo_huella_sha256":
		return huellaValida(valor)
	case "numero_personas":
		n, err := strconv.ParseUint(valor, 10, 32)
		return err == nil && n > 0 && strconv.FormatUint(n, 10) == valor
	case "porcentaje_financiacion":
		n, err := strconv.ParseUint(valor, 10, 8)
		return err == nil && n >= 1 && n <= 100 && strconv.FormatUint(n, 10) == valor
	case "programa_fin":
		fecha, err := time.Parse("2006-01-02", valor)
		return err == nil && fecha.Format("2006-01-02") == valor
	case "justificacion_temporal", "programa_denominacion":
		return textoValido(valor, 4000, false)
	default:
		return patronCodigoNecesidad.MatchString(valor)
	}
}
