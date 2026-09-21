package canonico

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	dominiopublico "vec-diputacion-granada/internal/modules/bolsa/publico/dominio"
)

const (
	maximoBytesProyeccionPublicacionV2 = 256 * 1024 * 1024
	maximoBytesManifiestoPublicacionV2 = 256 * 1024 * 1024
	maximoBytesBolsasPublicacionV1     = 64 * 1024 * 1024
	maximoBolsasPublicacionV1          = 128
	maximoPosicionesPublicacionV1      = 1_000_000
)

type MaterialPublicacionV3 struct {
	ProyeccionV2          []byte
	BolsasV1              []byte
	Manifiesto            ManifiestoPublicoV3
	AnclaManifiestoSHA256 string
}

// PrepararMaterialPublicacionV3 deriva el manifiesto compacto desde el
// contrato SQL completo y lo confronta con el manifiesto externo. La V2 se
// entrega íntegra a PostgreSQL; sólo B10 se reserializa para fijar su instante
// textual exactamente igual al de fuente.actualizada_en.
func PrepararMaterialPublicacionV3(proyeccionV2, manifiestoV2, bolsasV1 []byte) (MaterialPublicacionV3, error) {
	if len(proyeccionV2) == 0 || len(proyeccionV2) > maximoBytesProyeccionPublicacionV2 || len(manifiestoV2) == 0 || len(manifiestoV2) > maximoBytesManifiestoPublicacionV2 || len(bolsasV1) == 0 || len(bolsasV1) > maximoBytesBolsasPublicacionV1 {
		return MaterialPublicacionV3{}, ErrManifiestoPublicoInvalido
	}
	var proyeccion proyeccionPublicacionV2
	if !decodificarJSONExacto(proyeccionV2, &proyeccion) {
		return MaterialPublicacionV3{}, ErrManifiestoPublicoInvalido
	}
	derivado, instanteTexto, err := proyeccion.manifiesto()
	if err != nil {
		return MaterialPublicacionV3{}, ErrManifiestoPublicoInvalido
	}
	var externo ManifiestoPublicoV2
	if !decodificarJSONExacto(manifiestoV2, &externo) || externo.Validar() != nil {
		return MaterialPublicacionV3{}, ErrManifiestoPublicoInvalido
	}
	huellaDerivada, err := derivado.HuellaSHA256()
	if err != nil {
		return MaterialPublicacionV3{}, ErrManifiestoPublicoInvalido
	}
	huellaExterna, err := externo.HuellaSHA256()
	if err != nil || huellaDerivada != huellaExterna {
		return MaterialPublicacionV3{}, ErrManifiestoPublicoInvalido
	}
	var bolsas BolsasManifiestoV1
	if !decodificarJSONExacto(bolsasV1, &bolsas) || bolsas.Bolsas == nil || len(bolsas.Bolsas) > maximoBolsasPublicacionV1 {
		return MaterialPublicacionV3{}, ErrManifiestoPublicoInvalido
	}
	total := 0
	for _, bolsa := range bolsas.Bolsas {
		if bolsa.Posiciones == nil || len(bolsa.Posiciones) > maximoPosicionesPublicacionV1-total {
			return MaterialPublicacionV3{}, ErrManifiestoPublicoInvalido
		}
		total += len(bolsa.Posiciones)
	}
	v3 := ManifiestoPublicoV3{Esquema: EsquemaManifiestoPublicoV3, Fuente: derivado.Fuente, Catalogos: derivado.Catalogos, Categorias: derivado.Categorias, Convocatorias: derivado.Convocatorias, BolsasV1: bolsas}
	v3, err = v3.canonico()
	if err != nil {
		return MaterialPublicacionV3{}, ErrManifiestoPublicoInvalido
	}
	ancla, err := v3.HuellaSHA256()
	if err != nil {
		return MaterialPublicacionV3{}, ErrManifiestoPublicoInvalido
	}
	transporte, err := json.Marshal(bolsasTransporteV1{GeneradoEn: instanteTexto, Bolsas: copiarBolsasTransporte(v3.BolsasV1.Bolsas)})
	if err != nil || len(transporte) > maximoBytesBolsasPublicacionV1/2 {
		return MaterialPublicacionV3{}, ErrManifiestoPublicoInvalido
	}
	return MaterialPublicacionV3{ProyeccionV2: append([]byte(nil), proyeccionV2...), BolsasV1: transporte, Manifiesto: v3, AnclaManifiestoSHA256: ancla}, nil
}

type bolsasTransporteV1 struct {
	GeneradoEn string              `json:"generado_en"`
	Bolsas     []BolsaManifiestoV1 `json:"bolsas"`
}

func copiarBolsasTransporte(origen []BolsaManifiestoV1) []BolsaManifiestoV1 {
	resultado := make([]BolsaManifiestoV1, len(origen))
	for i := range origen {
		resultado[i] = origen[i]
		resultado[i].Grupos = append([]string(nil), origen[i].Grupos...)
		resultado[i].Posiciones = make([]PosicionBolsaManifiestoV1, len(origen[i].Posiciones))
		copy(resultado[i].Posiciones, origen[i].Posiciones)
	}
	return resultado
}

type fuentePublicacionV2 struct {
	Revision      string `json:"revision"`
	ActualizadaEn string `json:"actualizada_en"`
}
type proyeccionPublicacionV2 struct {
	Fuente        fuentePublicacionV2         `json:"fuente"`
	Catalogos     []CatalogoManifiestoV2      `json:"catalogos"`
	Categorias    categoriasPublicacionV2     `json:"categorias"`
	Convocatorias []convocatoriaPublicacionV2 `json:"convocatorias"`
}
type categoriasPublicacionV2 struct {
	Actual    ReferenciaCatalogoCategoriasManifiestoV2 `json:"actual"`
	Snapshots []snapshotCategoriasPublicacionV2        `json:"snapshots"`
}
type snapshotCategoriasPublicacionV2 struct {
	CatalogoID                    string                `json:"catalogo_id"`
	Version                       int                   `json:"version"`
	HuellaGobernadaSHA256         string                `json:"huella_gobernada_sha256"`
	HuellaProyeccionPublicaSHA256 string                `json:"huella_proyeccion_publica_sha256"`
	Categorias                    []CategoriaCatalogoV1 `json:"categorias"`
}
type convocatoriaPublicacionV2 struct {
	IdentificadorPublico       string                         `json:"identificador_publico"`
	VersionPublica             string                         `json:"version_publica"`
	Estado                     string                         `json:"estado"`
	Tipo                       string                         `json:"tipo"`
	HuellaPublicaSHA256        string                         `json:"huella_publica_sha256"`
	HuellaResumenPublicoSHA256 string                         `json:"huella_resumen_publico_sha256"`
	Titulo                     string                         `json:"titulo"`
	Resumen                    string                         `json:"resumen"`
	Descripcion                string                         `json:"descripcion"`
	PublicadaEn                time.Time                      `json:"publicada_en"`
	ActualizadaEn              time.Time                      `json:"actualizada_en"`
	CatalogoCategorias         ReferenciaCatalogoCategoriasV2 `json:"catalogo_categorias"`
	Categorias                 []string                       `json:"categorias"`
	Plazos                     []PlazoConvocatoriaV1          `json:"plazos"`
	Requisitos                 []RequisitoConvocatoriaV1      `json:"requisitos"`
	Documentos                 []DocumentoConvocatoriaV1      `json:"documentos"`
	Ayuda                      []AyudaConvocatoriaV1          `json:"ayuda"`
}

func (p proyeccionPublicacionV2) manifiesto() (ManifiestoPublicoV2, string, error) {
	if p.Convocatorias == nil {
		return ManifiestoPublicoV2{}, "", ErrManifiestoPublicoInvalido
	}
	instante, err := time.Parse(time.RFC3339Nano, p.Fuente.ActualizadaEn)
	if err != nil {
		return ManifiestoPublicoV2{}, "", err
	}
	fuente := FuenteManifiestoPublicoV2{Revision: p.Fuente.Revision, ActualizadaEn: instante.UTC()}
	categorias := CategoriasManifiestoPublicoV2{Actual: p.Categorias.Actual, Snapshots: make([]SnapshotCategoriasManifiestoV2, len(p.Categorias.Snapshots))}
	for i, s := range p.Categorias.Snapshots {
		categorias.Snapshots[i] = SnapshotCategoriasManifiestoV2{HuellaGobernadaSHA256: s.HuellaGobernadaSHA256, HuellaProyeccionSHA256: s.HuellaProyeccionPublicaSHA256, Catalogo: CatalogoCategoriasV1{Esquema: EsquemaCatalogoCategoriasV1, CatalogoID: s.CatalogoID, Version: s.Version, Categorias: s.Categorias}}
	}
	entradas := make([]ConvocatoriaManifiestoPublicoV2, len(p.Convocatorias))
	for i, c := range p.Convocatorias {
		if c.Categorias == nil || c.Plazos == nil || c.Requisitos == nil || c.Documentos == nil || c.Ayuda == nil {
			return ManifiestoPublicoV2{}, "", ErrManifiestoPublicoInvalido
		}
		d := &dominiopublico.DatosPublicosConvocatoria{IdentificadorPublico: c.IdentificadorPublico, Tipo: c.Tipo, CatalogoCategorias: dominiopublico.ReferenciaCatalogoCategorias{CatalogoID: c.CatalogoCategorias.CatalogoID, CatalogoVersion: c.CatalogoCategorias.CatalogoVersion, CatalogoHuellaSHA256: c.CatalogoCategorias.CatalogoHuellaSHA256, CatalogoHuellaProyeccionSHA256: c.CatalogoCategorias.CatalogoHuellaProyeccionSHA256}, Categorias: c.Categorias, Titulo: c.Titulo, Resumen: c.Resumen, Descripcion: c.Descripcion, PublicadaEn: c.PublicadaEn, ActualizadaEn: c.ActualizadaEn, Plazos: convertirPlazos(c.Plazos), Requisitos: convertirRequisitos(c.Requisitos), Documentos: convertirDocumentos(c.Documentos), Ayuda: convertirAyuda(c.Ayuda)}
		conv := dominiopublico.Convocatoria{Version: c.VersionPublica, Estado: dominiopublico.EstadoConvocatoria(c.Estado), HuellaSHA256: c.HuellaPublicaSHA256, DatosPublicos: d}
		completa, e := HuellaConvocatoriaV2(conv)
		if e != nil || completa != c.HuellaPublicaSHA256 {
			return ManifiestoPublicoV2{}, "", ErrManifiestoPublicoInvalido
		}
		resumen, e := ResumenDesdeConvocatoriaV2(conv)
		if e != nil {
			return ManifiestoPublicoV2{}, "", e
		}
		corta, e := HuellaResumenConvocatoriaV2(resumen)
		if e != nil || corta != c.HuellaResumenPublicoSHA256 {
			return ManifiestoPublicoV2{}, "", ErrManifiestoPublicoInvalido
		}
		entradas[i] = ConvocatoriaManifiestoPublicoV2{IdentificadorPublico: c.IdentificadorPublico, HuellaCompletaSHA256: completa, HuellaResumenSHA256: corta}
	}
	return ManifiestoPublicoV2{Esquema: EsquemaManifiestoPublicoV2, Fuente: fuente, Catalogos: p.Catalogos, Categorias: categorias, Convocatorias: entradas}, p.Fuente.ActualizadaEn, nil
}
func convertirPlazos(x []PlazoConvocatoriaV1) []dominiopublico.PlazoConvocatoria {
	r := make([]dominiopublico.PlazoConvocatoria, len(x))
	for i := range x {
		r[i] = dominiopublico.PlazoConvocatoria(x[i])
	}
	return r
}
func convertirRequisitos(x []RequisitoConvocatoriaV1) []dominiopublico.RequisitoConvocatoria {
	r := make([]dominiopublico.RequisitoConvocatoria, len(x))
	for i := range x {
		r[i] = dominiopublico.RequisitoConvocatoria(x[i])
	}
	return r
}
func convertirDocumentos(x []DocumentoConvocatoriaV1) []dominiopublico.DocumentoConvocatoria {
	r := make([]dominiopublico.DocumentoConvocatoria, len(x))
	for i := range x {
		r[i] = dominiopublico.DocumentoConvocatoria(x[i])
	}
	return r
}
func convertirAyuda(x []AyudaConvocatoriaV1) []dominiopublico.AyudaConvocatoria {
	r := make([]dominiopublico.AyudaConvocatoria, len(x))
	for i := range x {
		r[i] = dominiopublico.AyudaConvocatoria(x[i])
	}
	return r
}
func decodificarJSONExacto(x []byte, dest any) bool {
	if !sinDuplicadosNiAliases(x) {
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(x))
	dec.DisallowUnknownFields()
	if dec.Decode(dest) != nil {
		return false
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return false
	}
	normalizado, err := json.Marshal(dest)
	return err == nil && mismaFormaJSON(x, normalizado)
}

func mismaFormaJSON(origen, normalizado []byte) bool {
	var a, b any
	if json.Unmarshal(origen, &a) != nil || json.Unmarshal(normalizado, &b) != nil {
		return false
	}
	return mismaFormaJSONValor(a, b)
}

func mismaFormaJSONValor(a, b any) bool {
	switch izquierda := a.(type) {
	case map[string]any:
		derecha, ok := b.(map[string]any)
		if !ok || len(izquierda) != len(derecha) {
			return false
		}
		for clave, valor := range izquierda {
			otro, existe := derecha[clave]
			if !existe || !mismaFormaJSONValor(valor, otro) {
				return false
			}
		}
		return true
	case []any:
		derecha, ok := b.([]any)
		if !ok || len(izquierda) != len(derecha) {
			return false
		}
		for i := range izquierda {
			if !mismaFormaJSONValor(izquierda[i], derecha[i]) {
				return false
			}
		}
		return true
	case nil:
		return b == nil
	default:
		return tipoJSON(izquierda) == tipoJSON(b)
	}
}

func tipoJSON(valor any) string {
	switch valor.(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	case float64:
		return "number"
	default:
		return "otro"
	}
}

func sinDuplicadosNiAliases(x []byte) bool {
	d := json.NewDecoder(bytes.NewReader(x))
	var leer func() bool
	leer = func() bool {
		tok, e := d.Token()
		if e != nil {
			return false
		}
		z, ok := tok.(json.Delim)
		if !ok {
			return true
		}
		if z == '{' {
			v := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return false
				}
				s, ok := k.(string)
				if !ok || v[s] || s != strings.ToLower(s) {
					return false
				}
				v[s] = true
				if !leer() {
					return false
				}
			}
			_, e = d.Token()
			return e == nil
		}
		if z == '[' {
			for d.More() {
				if !leer() {
					return false
				}
			}
			_, e = d.Token()
			return e == nil
		}
		return false
	}
	if !leer() {
		return false
	}
	_, e := d.Token()
	return e == io.EOF
}
