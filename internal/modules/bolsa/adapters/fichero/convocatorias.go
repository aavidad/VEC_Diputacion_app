// Package fichero aporta únicamente una fuente local de demostración. No es
// un repositorio productivo ni persiste escrituras.
package fichero

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	versionEsquemaFichero = 1
	maximoBytesFichero    = 2 << 20
	maximoResultados      = 100
	maximoDesplazamiento  = 12_000
)

var (
	_                 puertosbolsa.ConsultaConvocatoriasPublicas = (*ConsultaConvocatorias)(nil)
	patronDNI                                                    = regexp.MustCompile(`(?i)(?:^|[^0-9])[0-9]{8}[A-Z](?:$|[^A-Z0-9])`)
	patronCorreo                                                 = regexp.MustCompile(`(?i)[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}`)
	patronTelefono                                               = regexp.MustCompile(`(?:^|[^0-9])(?:\+34[ .-]?)?[6789][0-9 .-]{7,12}[0-9](?:$|[^0-9])`)
	patronClaveFiltro                                            = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$`)
)

type archivoConvocatorias struct {
	VersionEsquema int                                       `json:"version_esquema"`
	Fuente         puertosbolsa.MetadatosFuenteConvocatorias `json:"fuente"`
	Catalogos      []puertosbolsa.CatalogoPublico            `json:"catalogos"`
	Convocatorias  []dominiobolsa.Convocatoria               `json:"convocatorias"`
}

// ConsultaConvocatorias mantiene una instantánea inmutable cargada al inicio.
// Cambiar a PostgreSQL/Oracle requiere otro adaptador del mismo puerto, no
// cambios en aplicación, HTTP o UI.
type ConsultaConvocatorias struct {
	convocatorias []dominiobolsa.Convocatoria
	porPublico    map[string]dominiobolsa.Convocatoria
	catalogos     []puertosbolsa.CatalogoPublico
	fuente        puertosbolsa.MetadatosFuenteConvocatorias
}

func NuevaConsultaConvocatorias(ruta string) (*ConsultaConvocatorias, error) {
	ruta = strings.TrimSpace(ruta)
	if ruta == "" {
		return nil, puertosbolsa.ErrFuenteConvocatoriasInvalida
	}
	fichero, err := os.Open(ruta)
	if err != nil {
		return nil, fmt.Errorf("%w: no se puede abrir la fuente DEMO", puertosbolsa.ErrFuenteConvocatoriasInvalida)
	}
	defer fichero.Close()
	contenido, err := io.ReadAll(io.LimitReader(fichero, maximoBytesFichero+1))
	if err != nil || len(contenido) == 0 || len(contenido) > maximoBytesFichero {
		return nil, fmt.Errorf("%w: tamaño no permitido", puertosbolsa.ErrFuenteConvocatoriasInvalida)
	}
	return nuevaConsultaDesdeJSON(contenido)
}

func nuevaConsultaDesdeJSON(contenido []byte) (*ConsultaConvocatorias, error) {
	if err := validarJSONSinClavesDuplicadasNiPII(contenido); err != nil {
		return nil, fmt.Errorf("%w: contenido rechazado", puertosbolsa.ErrFuenteConvocatoriasInvalida)
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var archivo archivoConvocatorias
	if err := decodificador.Decode(&archivo); err != nil {
		return nil, fmt.Errorf("%w: esquema no válido", puertosbolsa.ErrFuenteConvocatoriasInvalida)
	}
	if err := exigirFinJSON(decodificador); err != nil {
		return nil, fmt.Errorf("%w: más de un documento JSON", puertosbolsa.ErrFuenteConvocatoriasInvalida)
	}
	if err := validarArchivo(archivo); err != nil {
		return nil, err
	}
	consulta := &ConsultaConvocatorias{
		convocatorias: make([]dominiobolsa.Convocatoria, 0, len(archivo.Convocatorias)),
		porPublico:    make(map[string]dominiobolsa.Convocatoria, len(archivo.Convocatorias)),
		catalogos:     clonarCatalogos(archivo.Catalogos),
		fuente:        archivo.Fuente,
	}
	for _, convocatoria := range archivo.Convocatorias {
		clon := convocatoria.Clonar()
		consulta.convocatorias = append(consulta.convocatorias, clon)
		consulta.porPublico[clon.DatosPublicos.IdentificadorPublico] = clon
	}
	sort.Slice(consulta.convocatorias, func(i, j int) bool {
		a, b := consulta.convocatorias[i].DatosPublicos, consulta.convocatorias[j].DatosPublicos
		if a.ActualizadaEn.Equal(b.ActualizadaEn) {
			return a.IdentificadorPublico < b.IdentificadorPublico
		}
		return a.ActualizadaEn.After(b.ActualizadaEn)
	})
	return consulta, nil
}

func (c *ConsultaConvocatorias) BuscarPublicadas(ctx context.Context, filtro puertosbolsa.FiltroConvocatoriasPublicas) (puertosbolsa.PaginaConvocatorias, error) {
	if ctx == nil || c == nil {
		return puertosbolsa.PaginaConvocatorias{}, puertosbolsa.ErrConsultaConvocatoriasInvalida
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.PaginaConvocatorias{}, err
	}
	if !filtroValido(filtro) {
		return puertosbolsa.PaginaConvocatorias{}, puertosbolsa.ErrConsultaConvocatoriasInvalida
	}
	texto := strings.ToLower(filtro.Texto)
	coincidencias := make([]dominiobolsa.Convocatoria, 0, len(c.convocatorias))
	for _, convocatoria := range c.convocatorias {
		if err := ctx.Err(); err != nil {
			return puertosbolsa.PaginaConvocatorias{}, err
		}
		d := convocatoria.DatosPublicos
		if filtro.Tipo != "" && d.Tipo != filtro.Tipo || filtro.Estado != "" && string(convocatoria.Estado) != filtro.Estado ||
			filtro.Categoria != "" && !contiene(d.Categorias, filtro.Categoria) ||
			filtro.SoloPlazoAbierto && !tienePlazoAbierto(d.Plazos, filtro.Instante) ||
			texto != "" && !contieneTextoPublico(convocatoria, texto) {
			continue
		}
		coincidencias = append(coincidencias, convocatoria.Clonar())
	}
	total := len(coincidencias)
	inicio := filtro.Desplazamiento
	if inicio > total {
		inicio = total
	}
	fin := inicio + filtro.Limite
	if fin > total {
		fin = total
	}
	return puertosbolsa.PaginaConvocatorias{
		Convocatorias: clonarConvocatorias(coincidencias[inicio:fin]), Total: total,
		Catalogos: clonarCatalogos(c.catalogos), Fuente: c.fuente,
	}, nil
}

func (c *ConsultaConvocatorias) ObtenerPublicada(ctx context.Context, identificador string) (puertosbolsa.DetalleConvocatoria, error) {
	if ctx == nil || c == nil || identificador != strings.TrimSpace(identificador) || identificador == "" {
		return puertosbolsa.DetalleConvocatoria{}, puertosbolsa.ErrConsultaConvocatoriasInvalida
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.DetalleConvocatoria{}, err
	}
	convocatoria, existe := c.porPublico[identificador]
	if !existe {
		return puertosbolsa.DetalleConvocatoria{}, puertosbolsa.ErrConvocatoriaNoEncontrada
	}
	return puertosbolsa.DetalleConvocatoria{Convocatoria: convocatoria.Clonar(), Catalogos: clonarCatalogos(c.catalogos), Fuente: c.fuente}, nil
}

func validarArchivo(archivo archivoConvocatorias) error {
	if archivo.VersionEsquema != versionEsquemaFichero || !archivo.Fuente.Demostracion ||
		!strings.Contains(strings.ToUpper(archivo.Fuente.Aviso), "DEMOSTRACIÓN") || len(archivo.Convocatorias) == 0 {
		return fmt.Errorf("%w: el adaptador de fichero exige datos DEMO explícitos", puertosbolsa.ErrFuenteConvocatoriasInvalida)
	}
	indice, err := indiceCatalogos(archivo.Catalogos)
	if err != nil {
		return err
	}
	ids := make(map[string]struct{}, len(archivo.Convocatorias))
	publicos := make(map[string]struct{}, len(archivo.Convocatorias))
	for _, convocatoria := range archivo.Convocatorias {
		if err := convocatoria.ValidarPublicacion(); err != nil {
			return fmt.Errorf("%w: convocatoria no válida", puertosbolsa.ErrFuenteConvocatoriasInvalida)
		}
		if _, existe := ids[convocatoria.ID]; existe {
			return fmt.Errorf("%w: proceso duplicado", puertosbolsa.ErrFuenteConvocatoriasInvalida)
		}
		if _, existe := publicos[convocatoria.DatosPublicos.IdentificadorPublico]; existe {
			return fmt.Errorf("%w: identificador público duplicado", puertosbolsa.ErrFuenteConvocatoriasInvalida)
		}
		ids[convocatoria.ID] = struct{}{}
		publicos[convocatoria.DatosPublicos.IdentificadorPublico] = struct{}{}
		if !referenciasCatalogoValidas(convocatoria, indice) {
			return fmt.Errorf("%w: referencia de catálogo no publicable", puertosbolsa.ErrFuenteConvocatoriasInvalida)
		}
	}
	return nil
}

func indiceCatalogos(catalogos []puertosbolsa.CatalogoPublico) (map[string]map[string]bool, error) {
	indice := make(map[string]map[string]bool, len(catalogos))
	for _, catalogo := range catalogos {
		if catalogo.Referencia == "" || catalogo.Version < 1 || len(catalogo.Entradas) == 0 || indice[catalogo.Referencia] != nil {
			return nil, fmt.Errorf("%w: catálogo no válido", puertosbolsa.ErrFuenteConvocatoriasInvalida)
		}
		indice[catalogo.Referencia] = make(map[string]bool, len(catalogo.Entradas))
		for _, entrada := range catalogo.Entradas {
			if entrada.Clave == "" || entrada.Version != catalogo.Version || entrada.Etiqueta == "" || entrada.Semantica == "" || entrada.Orden < 1 {
				return nil, fmt.Errorf("%w: entrada de catálogo no válida", puertosbolsa.ErrFuenteConvocatoriasInvalida)
			}
			if _, existe := indice[catalogo.Referencia][entrada.Clave]; existe {
				return nil, fmt.Errorf("%w: entrada de catálogo duplicada", puertosbolsa.ErrFuenteConvocatoriasInvalida)
			}
			indice[catalogo.Referencia][entrada.Clave] = entrada.Publicable
		}
	}
	for _, requerido := range []string{puertosbolsa.CatalogoTiposConvocatoria, puertosbolsa.CatalogoEstadosConvocatoria, puertosbolsa.CatalogoCategoriasConvocatoria, puertosbolsa.CatalogoTiposPlazo, puertosbolsa.CatalogoTiposDocumento, puertosbolsa.CatalogoCategoriasAyuda} {
		if indice[requerido] == nil {
			return nil, fmt.Errorf("%w: falta catálogo requerido", puertosbolsa.ErrFuenteConvocatoriasInvalida)
		}
	}
	return indice, nil
}

func referenciasCatalogoValidas(c dominiobolsa.Convocatoria, indice map[string]map[string]bool) bool {
	d := c.DatosPublicos
	if !indice[puertosbolsa.CatalogoEstadosConvocatoria][string(c.Estado)] || !indice[puertosbolsa.CatalogoTiposConvocatoria][d.Tipo] {
		return false
	}
	for _, clave := range d.Categorias {
		if !indice[puertosbolsa.CatalogoCategoriasConvocatoria][clave] {
			return false
		}
	}
	for _, plazo := range d.Plazos {
		if !indice[puertosbolsa.CatalogoTiposPlazo][plazo.Tipo] {
			return false
		}
	}
	for _, documento := range d.Documentos {
		if !indice[puertosbolsa.CatalogoTiposDocumento][documento.Tipo] {
			return false
		}
	}
	for _, ayuda := range d.Ayuda {
		if !indice[puertosbolsa.CatalogoCategoriasAyuda][ayuda.Categoria] {
			return false
		}
	}
	return true
}

func filtroValido(f puertosbolsa.FiltroConvocatoriasPublicas) bool {
	return f.Texto == strings.TrimSpace(f.Texto) && len([]rune(f.Texto)) <= 100 && f.Limite >= 1 && f.Limite <= maximoResultados &&
		f.Desplazamiento >= 0 && f.Desplazamiento <= maximoDesplazamiento && !f.Instante.IsZero() &&
		f.Instante.Location() == time.UTC && f.Instante.Nanosecond()%1000 == 0 &&
		claveFiltroValida(f.Tipo) && claveFiltroValida(f.Categoria) && claveFiltroValida(f.Estado)
}

func claveFiltroValida(valor string) bool {
	return valor == "" || (valor == strings.TrimSpace(valor) && patronClaveFiltro.MatchString(valor))
}

func contiene(valores []string, buscado string) bool {
	for _, valor := range valores {
		if valor == buscado {
			return true
		}
	}
	return false
}

func tienePlazoAbierto(plazos []dominiobolsa.PlazoConvocatoria, ahora time.Time) bool {
	for _, plazo := range plazos {
		if !ahora.Before(plazo.AbreEn) && !ahora.After(plazo.CierraEn) {
			return true
		}
	}
	return false
}

func contieneTextoPublico(c dominiobolsa.Convocatoria, texto string) bool {
	d := c.DatosPublicos
	contenido := strings.ToLower(strings.Join([]string{d.Titulo, d.Resumen, d.Descripcion}, " "))
	return strings.Contains(contenido, texto)
}

func clonarConvocatorias(origen []dominiobolsa.Convocatoria) []dominiobolsa.Convocatoria {
	destino := make([]dominiobolsa.Convocatoria, 0, len(origen))
	for _, c := range origen {
		destino = append(destino, c.Clonar())
	}
	return destino
}

func clonarCatalogos(origen []puertosbolsa.CatalogoPublico) []puertosbolsa.CatalogoPublico {
	destino := make([]puertosbolsa.CatalogoPublico, len(origen))
	for i, catalogo := range origen {
		destino[i] = catalogo
		destino[i].Entradas = append([]puertosbolsa.EntradaCatalogoPublico(nil), catalogo.Entradas...)
	}
	return destino
}

func exigirFinJSON(decodificador *json.Decoder) error {
	var extra any
	err := decodificador.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return errors.New("contenido JSON adicional")
}

func validarJSONSinClavesDuplicadasNiPII(contenido []byte) error {
	if patronDNI.Match(contenido) || patronCorreo.Match(contenido) || patronTelefono.Match(contenido) {
		return errors.New("posible dato personal en fuente DEMO")
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	if err := validarValorJSON(decodificador); err != nil {
		return err
	}
	return exigirFinJSON(decodificador)
}

func validarValorJSON(decodificador *json.Decoder) error {
	token, err := decodificador.Token()
	if err != nil {
		return err
	}
	delimitador, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	switch delimitador {
	case '{':
		vistas := map[string]struct{}{}
		for decodificador.More() {
			claveToken, err := decodificador.Token()
			if err != nil {
				return err
			}
			clave, ok := claveToken.(string)
			if !ok {
				return errors.New("clave JSON no válida")
			}
			if _, existe := vistas[clave]; existe {
				return errors.New("clave JSON duplicada")
			}
			vistas[clave] = struct{}{}
			if clavePIIProhibida(clave) {
				return errors.New("clave de dato personal prohibida")
			}
			if err := validarValorJSON(decodificador); err != nil {
				return err
			}
		}
		_, err = decodificador.Token()
		return err
	case '[':
		for decodificador.More() {
			if err := validarValorJSON(decodificador); err != nil {
				return err
			}
		}
		_, err = decodificador.Token()
		return err
	default:
		return errors.New("delimitador JSON no válido")
	}
}

func clavePIIProhibida(clave string) bool {
	normalizada := strings.ToLower(strings.ReplaceAll(clave, "-", "_"))
	_, prohibida := map[string]struct{}{
		"dni": {}, "nif": {}, "nie": {}, "nombre": {}, "apellidos": {}, "correo": {}, "email": {}, "telefono": {},
		"direccion": {}, "fecha_nacimiento": {}, "candidate_id": {}, "candidato_id": {}, "sujeto_ref": {}, "persona_ref": {},
	}[normalizada]
	return prohibida
}
