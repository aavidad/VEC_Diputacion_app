// Package simulacion conecta ejemplos sintéticos con el motor común de Bolsa.
package simulacion

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"vec-diputacion-granada/internal/modules/bolsa/application/simulacionbaremo"
	"vec-diputacion-granada/internal/modules/bolsa/domain/calculomeritos"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

var ErrEjemplo = errors.New("ejemplo_no_admitido")

//go:embed ejemplos.json
var ejemplosJSON []byte

type Ejemplo struct {
	Referencia      string               `json:"referencia"`
	TituloClave     string               `json:"titulo_clave"`
	ConvocatoriaRef string               `json:"convocatoria_ref"`
	BasesVersion    int                  `json:"bases_version"`
	Configuracion   domain.Configuracion `json:"configuracion"`
	NotasPrueba     []NotaEditable       `json:"notas_prueba"`
}
type fixture struct {
	Ejemplo
	Entrada domain.Entrada             `json:"entrada"`
	Baremo  json.RawMessage            `json:"baremo"`
	Meritos map[string]json.RawMessage `json:"meritos"`
}

func leer() ([]fixture, error) {
	var datos struct {
		Ejemplos []fixture `json:"ejemplos"`
	}
	if err := json.Unmarshal(ejemplosJSON, &datos); err != nil {
		return nil, err
	}
	for _, e := range datos.Ejemplos {
		if err := e.Configuracion.Validar(); err != nil {
			return nil, err
		}
		if e.ConvocatoriaRef != e.Entrada.ConvocatoriaRef || e.BasesVersion != e.Entrada.BasesVersion {
			return nil, ErrEjemplo
		}
	}
	return datos.Ejemplos, nil
}

// Cada lectura reconstruye las instantáneas; ningún consumidor puede alterar
// los datos embebidos que utilizará la siguiente simulación.
func Ejemplos() ([]Ejemplo, error) {
	fixtures, err := leer()
	if err != nil {
		return nil, err
	}
	e := make([]Ejemplo, 0, len(fixtures))
	for _, f := range fixtures {
		ejemplo := f.Ejemplo
		ejemplo.NotasPrueba = catalogoNotas(f)
		e = append(e, ejemplo)
	}
	return e, nil
}
func Preparar(ref string) (domain.Entrada, BaremadorBolsa, error) {
	fixtures, err := leer()
	if err != nil {
		return domain.Entrada{}, BaremadorBolsa{}, err
	}
	for _, f := range fixtures {
		if f.Referencia == ref {
			return f.Entrada, BaremadorBolsa{f.ConvocatoriaRef, f.BasesVersion, f.Baremo, f.Meritos}, nil
		}
	}
	return domain.Entrada{}, BaremadorBolsa{}, ErrEjemplo
}

// BaremadorBolsa conserva los DTO y la restauración del propietario del baremo
// en este adaptador; domain y application reciben únicamente puntos y trazas.
type BaremadorBolsa struct {
	convocatoriaRef string
	basesVersion    int
	reglas          json.RawMessage
	entradas        map[string]json.RawMessage
}

func (b BaremadorBolsa) Valorar(convocatoriaRef string, basesVersion int, solicitudRef string) (domain.MeritosValorados, error) {
	entrada, ok := b.entradas[solicitudRef]
	if !ok || convocatoriaRef != b.convocatoriaRef || basesVersion != b.basesVersion {
		return domain.MeritosValorados{}, ErrEjemplo
	}
	// El catálogo embebido es legible para revisión. Bolsa exige la forma
	// canónica de sus DTO antes de restaurar las instantáneas y sus huellas.
	var conjunto calculomeritos.Conjunto
	reglasCanonicas, err := canonizar(b.reglas, &conjunto)
	if err != nil {
		return domain.MeritosValorados{}, err
	}
	var meritosEntrada calculomeritos.Entrada
	entradaCanonica, err := canonizar(entrada, &meritosEntrada)
	if err != nil {
		return domain.MeritosValorados{}, err
	}
	s, err := (simulacionbaremo.ServicioMeritos{}).SimularMeritos(simulacionbaremo.Solicitud{ConjuntoCanonico: reglasCanonicas, HuellaConjuntoSHA256: huella(reglasCanonicas), EntradaCanonica: entradaCanonica, HuellaEntradaSHA256: huella(entradaCanonica)})
	if err != nil {
		return domain.MeritosValorados{}, err
	}
	var sobre struct {
		ConvocatoriaRef string                   `json:"convocatoria_ref"`
		Resultado       calculomeritos.Resultado `json:"resultado"`
	}
	if err := json.Unmarshal(s.RepresentacionCanonica(), &sobre); err != nil {
		return domain.MeritosValorados{}, err
	}
	if sobre.ConvocatoriaRef != convocatoriaRef {
		return domain.MeritosValorados{}, ErrEjemplo
	}
	r := domain.MeritosValorados{Reglas: []domain.ReglaValorada{}, HuellaSHA256: s.HuellaResultadoSHA256()}
	if sobre.Resultado.Estado != "completado" || sobre.Resultado.Total == nil {
		return r, nil
	}
	total := sobre.Resultado.Total.Micropuntos()
	r.PuntosMicropuntos = &total
	for _, regla := range sobre.Resultado.Reglas {
		r.Reglas = append(r.Reglas, domain.ReglaValorada{Referencia: regla.Clave, PuntosMicropuntos: regla.Puntos.Micropuntos()})
	}
	return r, nil
}
func huella(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func canonizar(b []byte, destino any) ([]byte, error) {
	if err := json.Unmarshal(b, destino); err != nil {
		return nil, err
	}
	return json.Marshal(destino)
}
