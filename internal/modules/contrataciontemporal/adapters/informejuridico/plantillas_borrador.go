package informejuridico

import (
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Identidad del catálogo de plantillas de los borradores de RRHH. El texto,
// los campos que se combinan, las modalidades y los firmantes viven en el
// catálogo versionado; aquí sólo queda la gramática y la lista cerrada de
// campos que el detalle autorizado puede aportar.
const (
	CatalogoPlantillasBorradorID = "vec.contratacion_temporal.plantillas_documentos"
	ModuloPlantillasBorrador     = "contratacion_temporal"

	claveEntradaEtiquetas = "etiquetas"

	maximoPlantillasCatalogo    = 32
	maximoParrafosPlantilla     = 64
	maximoBytesTituloPlantilla  = 256
	maximoBytesPlantilla        = 64 * 1024
	maximoBytesDocumento        = 256 * 1024
	maximoFirmantesPlantilla    = 8
	maximoBytesFirmante         = 160
	maximoEtiquetasCatalogo     = 96
	maximoBytesEtiquetaCatalogo = 160
)

// ErrPlantillasBorradorInvalidas indica que el catálogo no cumple la
// gramática o usa un campo, atributo o tipo desconocido. Se detecta al cargar,
// nunca al generar un documento.
var ErrPlantillasBorradorInvalidas = errors.New("contratacion temporal: catalogo de plantillas de borrador invalido")

// TiposBorradorConocidos es la lista cerrada de documentos que la API sabe
// servir. Un catálogo puede omitir alguno (entonces no está disponible), pero
// no puede inventar otro.
var TiposBorradorConocidos = []ports.TipoBorradorRRHH{
	ports.BorradorInformeDefinitivo, ports.BorradorResolucion, ports.BorradorDiligencia,
	ports.BorradorTomaPosesion, ports.BorradorNotificacion, ports.BorradorComunicacionCentro,
	ports.BorradorContratoLaboral, ports.BorradorNombramiento, ports.BorradorCese,
	ports.BorradorModificacionNombramiento,
}

var (
	marcaPlantilla      = regexp.MustCompile(`\{\{([?!/]?)([a-z][a-z0-9_]{0,63})\}\}`)
	patronAccionRequida = regexp.MustCompile(`^[a-z][a-z0-9_]{0,95}$`)
	prefijosEtiqueta    = []string{"modalidad.", "causa.", "comprobacion.", "resultado."}
)

// PlantillaBorrador es una plantilla ya validada. Sus textos sólo contienen
// campos de la lista cerrada y bloques condicionales bien formados.
type PlantillaBorrador struct {
	Tipo           ports.TipoBorradorRRHH
	Referencia     string
	Nombre         string
	Titulo         string
	Parrafos       []string
	Modalidades    []domain.ClaveCatalogo
	Firmantes      []string
	RequiereAccion string
}

// AdmiteModalidad aplica la lista de modalidades del catálogo; vacía = todas.
func (p PlantillaBorrador) AdmiteModalidad(modalidad domain.ClaveCatalogo) bool {
	if len(p.Modalidades) == 0 {
		return true
	}
	for _, admitida := range p.Modalidades {
		if admitida == modalidad {
			return true
		}
	}
	return false
}

// PlantillasBorrador es la instantánea inmutable cargada al arrancar.
type PlantillasBorrador struct {
	plantillas map[ports.TipoBorradorRRHH]PlantillaBorrador
	etiquetas  map[string]string
	referencia string
	huella     string
}

// Plantilla devuelve la plantilla vigente de un tipo; un receptor nulo no
// tiene ninguna.
func (p *PlantillasBorrador) Plantilla(tipo ports.TipoBorradorRRHH) (PlantillaBorrador, bool) {
	if p == nil {
		return PlantillaBorrador{}, false
	}
	plantilla, ok := p.plantillas[tipo]
	if !ok {
		return PlantillaBorrador{}, false
	}
	plantilla.Parrafos = append([]string(nil), plantilla.Parrafos...)
	plantilla.Modalidades = append([]domain.ClaveCatalogo(nil), plantilla.Modalidades...)
	plantilla.Firmantes = append([]string(nil), plantilla.Firmantes...)
	return plantilla, true
}

// Tipos lista los documentos disponibles en el orden de la lista cerrada.
func (p *PlantillasBorrador) Tipos() []ports.TipoBorradorRRHH {
	if p == nil {
		return nil
	}
	tipos := make([]ports.TipoBorradorRRHH, 0, len(p.plantillas))
	for _, tipo := range TiposBorradorConocidos {
		if _, ok := p.plantillas[tipo]; ok {
			tipos = append(tipos, tipo)
		}
	}
	return tipos
}

// Referencia identifica catálogo y versión; Huella es su SHA-256 canónico.
func (p *PlantillasBorrador) Referencia() string {
	if p == nil {
		return ""
	}
	return p.referencia
}

func (p *PlantillasBorrador) Huella() string {
	if p == nil {
		return ""
	}
	return p.huella
}

func (p *PlantillasBorrador) etiqueta(prefijo, clave string) string {
	if p != nil {
		if valor, ok := p.etiquetas[prefijo+clave]; ok {
			return valor
		}
	}
	return clave
}

// NuevasPlantillasBorrador valida el catálogo completo con las entradas
// vigentes en instante. Cualquier atributo, campo o tipo desconocido, bloque
// mal cerrado o texto fuera de límites invalida el catálogo entero.
func NuevasPlantillasBorrador(catalogo vecdomain.CatalogoConfigurable, instante time.Time) (*PlantillasBorrador, error) {
	canonico, err := catalogo.ClonarCanonico()
	if err != nil || canonico.ID != CatalogoPlantillasBorradorID || canonico.ModuloID != ModuloPlantillasBorrador ||
		canonico.Estado != vecdomain.EstadoCatalogoPublicado || instante.IsZero() ||
		len(canonico.Entradas) > maximoPlantillasCatalogo+1 {
		return nil, ErrPlantillasBorradorInvalidas
	}
	huella, err := canonico.HuellaSHA256()
	if err != nil {
		return nil, ErrPlantillasBorradorInvalidas
	}
	resultado := &PlantillasBorrador{
		plantillas: make(map[ports.TipoBorradorRRHH]PlantillaBorrador),
		etiquetas:  map[string]string{},
		referencia: canonico.Referencia(), huella: huella,
	}
	conocidos := make(map[ports.TipoBorradorRRHH]struct{}, len(TiposBorradorConocidos))
	for _, tipo := range TiposBorradorConocidos {
		conocidos[tipo] = struct{}{}
	}
	for _, entrada := range canonico.Entradas {
		if entrada.Clave == claveEntradaEtiquetas {
			if !entrada.VigenteEn(instante) {
				continue
			}
			if err := resultado.cargarEtiquetas(entrada.Atributos); err != nil {
				return nil, err
			}
			continue
		}
		tipo := ports.TipoBorradorRRHH(entrada.Clave)
		if _, ok := conocidos[tipo]; !ok {
			return nil, ErrPlantillasBorradorInvalidas
		}
		// Una entrada no vigente se valida igual: un error latente no espera
		// a la fecha en que entraría en vigor.
		plantilla, err := plantillaDesdeEntrada(canonico, tipo, entrada)
		if err != nil {
			return nil, err
		}
		if entrada.VigenteEn(instante) {
			resultado.plantillas[tipo] = plantilla
		}
	}
	if len(resultado.plantillas) == 0 {
		return nil, ErrPlantillasBorradorInvalidas
	}
	return resultado, nil
}

func (p *PlantillasBorrador) cargarEtiquetas(atributos map[string]string) error {
	if len(atributos) > maximoEtiquetasCatalogo {
		return ErrPlantillasBorradorInvalidas
	}
	for clave, valor := range atributos {
		admitida := false
		for _, prefijo := range prefijosEtiqueta {
			if strings.HasPrefix(clave, prefijo) && len(clave) > len(prefijo) {
				admitida = true
			}
		}
		if !admitida || len(valor) > maximoBytesEtiquetaCatalogo || strings.ContainsAny(valor, "\n\r{}") {
			return ErrPlantillasBorradorInvalidas
		}
		p.etiquetas[clave] = valor
	}
	return nil
}

func plantillaDesdeEntrada(
	catalogo vecdomain.CatalogoConfigurable, tipo ports.TipoBorradorRRHH, entrada vecdomain.EntradaCatalogoConfigurable,
) (PlantillaBorrador, error) {
	atributos := entrada.Atributos
	plantilla := PlantillaBorrador{
		Tipo: tipo, Nombre: entrada.Etiqueta, Titulo: atributos["titulo"],
		Referencia: catalogo.Referencia() + ":" + entrada.Clave,
	}
	usados := 1
	if plantilla.Titulo == "" || len(plantilla.Titulo) > maximoBytesTituloPlantilla ||
		strings.ContainsAny(plantilla.Titulo, "\n\r") || validarTextoPlantilla(plantilla.Titulo, false) != nil {
		return PlantillaBorrador{}, ErrPlantillasBorradorInvalidas
	}
	total := len(plantilla.Titulo)
	for indice := 1; ; indice++ {
		parrafo, ok := atributos["parrafo."+dosCifras(indice)]
		if !ok {
			break
		}
		usados++
		total += len(parrafo)
		if indice > maximoParrafosPlantilla || total > maximoBytesPlantilla || validarTextoPlantilla(parrafo, true) != nil {
			return PlantillaBorrador{}, ErrPlantillasBorradorInvalidas
		}
		plantilla.Parrafos = append(plantilla.Parrafos, parrafo)
	}
	if len(plantilla.Parrafos) == 0 {
		return PlantillaBorrador{}, ErrPlantillasBorradorInvalidas
	}
	if valor, ok := atributos["modalidades"]; ok {
		usados++
		if valor != "*" {
			for _, parte := range strings.Split(valor, ",") {
				modalidad := domain.ClaveCatalogo(strings.TrimSpace(parte))
				if !modalidad.Valida() {
					return PlantillaBorrador{}, ErrPlantillasBorradorInvalidas
				}
				plantilla.Modalidades = append(plantilla.Modalidades, modalidad)
			}
		}
	}
	if valor, ok := atributos["firmantes"]; ok {
		usados++
		for _, parte := range strings.Split(valor, ";") {
			firmante := strings.TrimSpace(parte)
			if firmante == "" || len(firmante) > maximoBytesFirmante || strings.ContainsAny(firmante, "\n\r{}") {
				return PlantillaBorrador{}, ErrPlantillasBorradorInvalidas
			}
			plantilla.Firmantes = append(plantilla.Firmantes, firmante)
		}
		if len(plantilla.Firmantes) > maximoFirmantesPlantilla {
			return PlantillaBorrador{}, ErrPlantillasBorradorInvalidas
		}
	}
	if valor, ok := atributos["requiere_accion"]; ok {
		usados++
		if !patronAccionRequida.MatchString(valor) {
			return PlantillaBorrador{}, ErrPlantillasBorradorInvalidas
		}
		plantilla.RequiereAccion = valor
	}
	// Atributos descriptivos del paquete de ejemplo: no alteran el texto.
	for _, informativo := range []string{"origen", "norma", "duda"} {
		if _, ok := atributos[informativo]; ok {
			usados++
		}
	}
	if usados != len(atributos) {
		return PlantillaBorrador{}, ErrPlantillasBorradorInvalidas
	}
	return plantilla, nil
}

func dosCifras(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// validarTextoPlantilla comprueba la gramática: {{campo}}, y, si se admiten,
// bloques {{?campo}}…{{/campo}} (se incluye si el campo tiene valor) y
// {{!campo}}…{{/campo}} (si está vacío), sin anidar.
func validarTextoPlantilla(texto string, condicionales bool) error {
	if !utf8.ValidString(texto) {
		return ErrPlantillasBorradorInvalidas
	}
	abierto := ""
	restante := texto
	for {
		indices := marcaPlantilla.FindStringSubmatchIndex(restante)
		prefijo := restante
		if indices != nil {
			prefijo = restante[:indices[0]]
		}
		if strings.Contains(prefijo, "{{") || strings.Contains(prefijo, "}}") {
			return ErrPlantillasBorradorInvalidas
		}
		if indices == nil {
			break
		}
		marca, campo := restante[indices[2]:indices[3]], restante[indices[4]:indices[5]]
		if _, ok := camposBorrador[campo]; !ok {
			return ErrPlantillasBorradorInvalidas
		}
		switch {
		case marca == "":
		case !condicionales:
			return ErrPlantillasBorradorInvalidas
		case marca == "/":
			if abierto != campo {
				return ErrPlantillasBorradorInvalidas
			}
			abierto = ""
		default:
			if abierto != "" {
				return ErrPlantillasBorradorInvalidas
			}
			abierto = campo
		}
		restante = restante[indices[1]:]
	}
	if abierto != "" {
		return ErrPlantillasBorradorInvalidas
	}
	return nil
}

// Rellenar sustituye los campos una sola vez: un valor nunca se interpreta
// como plantilla. Un párrafo que queda vacío se omite.
func (p PlantillaBorrador) Rellenar(valores map[string]string) (vecdomain.ContenidoDocumento, error) {
	total := 0
	titulo, err := rellenarTexto(p.Titulo, valores, &total)
	if err != nil || strings.TrimSpace(titulo) == "" {
		return vecdomain.ContenidoDocumento{}, ports.ErrBorradorRRHHNoDisponible
	}
	contenido := vecdomain.ContenidoDocumento{Titulo: titulo, Parrafos: make([]string, 0, len(p.Parrafos))}
	for _, parrafo := range p.Parrafos {
		texto, err := rellenarTexto(parrafo, valores, &total)
		if err != nil {
			return vecdomain.ContenidoDocumento{}, ports.ErrBorradorRRHHNoDisponible
		}
		if texto != "" {
			contenido.Parrafos = append(contenido.Parrafos, texto)
		}
	}
	if len(contenido.Parrafos) == 0 {
		return vecdomain.ContenidoDocumento{}, ports.ErrBorradorRRHHNoDisponible
	}
	return contenido, nil
}

func rellenarTexto(texto string, valores map[string]string, total *int) (string, error) {
	var salida strings.Builder
	incluir := true
	restante := texto
	escribir := func(parte string) error {
		if !incluir {
			return nil
		}
		if *total += len(parte); *total > maximoBytesDocumento {
			return ErrPlantillasBorradorInvalidas
		}
		salida.WriteString(parte)
		return nil
	}
	for {
		indices := marcaPlantilla.FindStringSubmatchIndex(restante)
		if indices == nil {
			if err := escribir(restante); err != nil {
				return "", err
			}
			return salida.String(), nil
		}
		if err := escribir(restante[:indices[0]]); err != nil {
			return "", err
		}
		marca, campo := restante[indices[2]:indices[3]], restante[indices[4]:indices[5]]
		switch marca {
		case "?":
			incluir = valores[campo] != ""
		case "!":
			incluir = valores[campo] == ""
		case "/":
			incluir = true
		default:
			if err := escribir(valores[campo]); err != nil {
				return "", err
			}
		}
		restante = restante[indices[1]:]
	}
}

// CamposBorradorDisponibles devuelve la lista cerrada ordenada, útil para
// documentar el catálogo.
func CamposBorradorDisponibles() []string {
	campos := make([]string, 0, len(camposBorrador))
	for campo := range camposBorrador {
		campos = append(campos, campo)
	}
	sort.Strings(campos)
	return campos
}
