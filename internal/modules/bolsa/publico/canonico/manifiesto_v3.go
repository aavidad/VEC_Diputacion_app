package canonico

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"time"
)

const EsquemaManifiestoPublicoV3 = "vec.bolsa.manifiesto-publico.canonico.v3"

var (
	patronReferenciaBolsaManifiesto = regexp.MustCompile(`^[a-z0-9][a-z0-9:._-]{2,159}$`)
	patronDocumentoBolsaManifiesto  = regexp.MustCompile(`^\*{3}[0-9]{4}\*{2}$`)
	patronCategoriaClaveManifiesto  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,79}$`)
	patronGrupoManifiesto           = regexp.MustCompile(`^[A-Z][0-9A-Z-]{0,7}$`)
)

// PosicionBolsaManifiestoV1 es la única representación de una candidatura
// que entra en el manifiesto externo: orden, documento enmascarado y estado.
type PosicionBolsaManifiestoV1 struct {
	Orden                int    `json:"orden"`
	DocumentoEnmascarado string `json:"documento_enmascarado"`
	EstadoClave          string `json:"estado_clave"`
}

// BolsaManifiestoV1 fija la lista pública ya minimizada, sin nombres,
// contactos ni referencias internas de participación.
type BolsaManifiestoV1 struct {
	BolsaRef       string                      `json:"bolsa_ref"`
	Categoria      string                      `json:"categoria"`
	CategoriaClave string                      `json:"categoria_clave"`
	Grupos         []string                    `json:"grupos"`
	TipoLista      string                      `json:"tipo_lista"`
	VigenteDesde   time.Time                   `json:"vigente_desde"`
	VigenteHasta   *time.Time                  `json:"vigente_hasta"`
	Total          int                         `json:"total"`
	Posiciones     []PosicionBolsaManifiestoV1 `json:"posiciones"`
}

type BolsasManifiestoV1 struct {
	GeneradoEn time.Time           `json:"generado_en"`
	Bolsas     []BolsaManifiestoV1 `json:"bolsas"`
}

// ManifiestoPublicoV3 prolonga V2 con las bolsas B10. Los campos V2 se
// conservan en el mismo orden para que ambas partes reconstruyan una única
// ancla y la extensión se añada al final.
type ManifiestoPublicoV3 struct {
	Esquema       string                            `json:"esquema"`
	Fuente        FuenteManifiestoPublicoV2         `json:"fuente"`
	Catalogos     []CatalogoManifiestoV2            `json:"catalogos"`
	Categorias    CategoriasManifiestoPublicoV2     `json:"categorias"`
	Convocatorias []ConvocatoriaManifiestoPublicoV2 `json:"convocatorias"`
	BolsasV1      BolsasManifiestoV1                `json:"bolsas_v1"`
}

func (m ManifiestoPublicoV3) HuellaSHA256() (string, error) {
	canonico, err := m.canonico()
	if err != nil {
		return "", err
	}
	contenido, err := json.Marshal(canonico)
	if err != nil {
		return "", ErrManifiestoPublicoInvalido
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func (m ManifiestoPublicoV3) Validar() error {
	_, err := m.canonico()
	return err
}

func (m ManifiestoPublicoV3) canonico() (ManifiestoPublicoV3, error) {
	base, err := (ManifiestoPublicoV2{
		Esquema: EsquemaManifiestoPublicoV2, Fuente: m.Fuente, Catalogos: m.Catalogos,
		Categorias: m.Categorias, Convocatorias: m.Convocatorias,
	}).canonico()
	if err != nil || m.Esquema != EsquemaManifiestoPublicoV3 || !instanteCanonico(m.BolsasV1.GeneradoEn) ||
		!m.BolsasV1.GeneradoEn.Equal(m.Fuente.ActualizadaEn) || len(m.BolsasV1.Bolsas) > 10000 {
		return ManifiestoPublicoV3{}, ErrManifiestoPublicoInvalido
	}
	resultado := ManifiestoPublicoV3{
		Esquema: EsquemaManifiestoPublicoV3, Fuente: base.Fuente, Catalogos: base.Catalogos,
		Categorias: base.Categorias, Convocatorias: base.Convocatorias,
		BolsasV1: BolsasManifiestoV1{
			GeneradoEn: m.BolsasV1.GeneradoEn.UTC().Truncate(time.Microsecond),
			Bolsas:     append([]BolsaManifiestoV1(nil), m.BolsasV1.Bolsas...),
		},
	}
	vistas := make(map[string]struct{}, len(resultado.BolsasV1.Bolsas))
	for indice := range resultado.BolsasV1.Bolsas {
		bolsa := &resultado.BolsasV1.Bolsas[indice]
		if !bolsaManifiestoValida(bolsa) {
			return ManifiestoPublicoV3{}, ErrManifiestoPublicoInvalido
		}
		if _, duplicada := vistas[bolsa.BolsaRef]; duplicada {
			return ManifiestoPublicoV3{}, ErrManifiestoPublicoInvalido
		}
		vistas[bolsa.BolsaRef] = struct{}{}
		bolsa.Grupos = append([]string(nil), bolsa.Grupos...)
		sort.Strings(bolsa.Grupos)
		bolsa.Posiciones = append([]PosicionBolsaManifiestoV1(nil), bolsa.Posiciones...)
		bolsa.VigenteDesde = bolsa.VigenteDesde.UTC().Truncate(time.Microsecond)
		if bolsa.VigenteHasta != nil {
			hasta := bolsa.VigenteHasta.UTC().Truncate(time.Microsecond)
			bolsa.VigenteHasta = &hasta
		}
	}
	sort.Slice(resultado.BolsasV1.Bolsas, func(i, j int) bool {
		return resultado.BolsasV1.Bolsas[i].BolsaRef < resultado.BolsasV1.Bolsas[j].BolsaRef
	})
	return resultado, nil
}

func bolsaManifiestoValida(bolsa *BolsaManifiestoV1) bool {
	if bolsa == nil || !patronReferenciaBolsaManifiesto.MatchString(bolsa.BolsaRef) ||
		!textoCanonico(bolsa.Categoria, 160, false) || !patronCategoriaClaveManifiesto.MatchString(bolsa.CategoriaClave) ||
		!patronCategoriaClaveManifiesto.MatchString(bolsa.TipoLista) || !instanteCanonico(bolsa.VigenteDesde) ||
		bolsa.Total < 0 || bolsa.Total > 100000 || len(bolsa.Grupos) == 0 || len(bolsa.Grupos) > 8 ||
		len(bolsa.Posiciones) != bolsa.Total || (bolsa.VigenteHasta != nil &&
		(!instanteCanonico(*bolsa.VigenteHasta) || !bolsa.VigenteDesde.Before(*bolsa.VigenteHasta))) {
		return false
	}
	grupos := make(map[string]struct{}, len(bolsa.Grupos))
	for _, grupo := range bolsa.Grupos {
		if !patronGrupoManifiesto.MatchString(grupo) {
			return false
		}
		if _, duplicado := grupos[grupo]; duplicado {
			return false
		}
		grupos[grupo] = struct{}{}
	}
	for indice, posicion := range bolsa.Posiciones {
		if posicion.Orden != indice+1 || !patronDocumentoBolsaManifiesto.MatchString(posicion.DocumentoEnmascarado) ||
			!estadoBolsaManifiestoValido(posicion.EstadoClave) {
			return false
		}
	}
	return true
}

func estadoBolsaManifiestoValido(estado string) bool {
	switch estado {
	case "disponible", "ocupado", "no_disponible", "excluido", "renuncia_pendiente":
		return true
	default:
		return false
	}
}
