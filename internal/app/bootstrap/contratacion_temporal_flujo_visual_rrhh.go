package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sort"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const esquemaManifestFlujoVisualRRHH = "vec.contratacion_temporal.manifest_flujo_visual_rrhh.v1"

var patronHuellaFlujoVisualRRHH = regexp.MustCompile(`^[a-f0-9]{64}$`)

var ErrManifestFlujoVisualRRHHInvalido = errors.New(
	"bootstrap: manifest de flujo visual RRHH invalido",
)

// LectorFlujoVisualRRHH resuelve la representación del rail después de la
// lectura autorizada del detalle. No crea una segunda sesión, recibo o
// autorización: sólo coteja el triple inmovilizado por el expediente contra
// el manifiesto de presentación publicado.
type LectorFlujoVisualRRHH struct {
	origen domain.ReferenciaFlujo
	base   ports.PresentacionFlujoRRHH
	mapeos map[domain.ClaveFase]domain.ClaveFase
}

type manifestFlujoVisualRRHH struct {
	Esquema string `json:"esquema"`
	Fuente  struct {
		Referencia string `json:"referencia"`
		Huella     string `json:"huella_sha256"`
	} `json:"fuente"`
	FlujoOrigen struct {
		Referencia string `json:"referencia"`
		Version    uint64 `json:"version"`
		Huella     string `json:"huella_sha256"`
	} `json:"flujo_origen"`
	Presentacion ports.PresentacionFlujoRRHH `json:"presentacion"`
	Mapeos       []mapeoFaseOrigenVisualRRHH `json:"mapeos_fase_origen"`
}

type mapeoFaseOrigenVisualRRHH struct {
	Origen       domain.ClaveFase `json:"fase_origen"`
	Presentacion domain.ClaveFase `json:"fase_presentacion"`
}

// LeerLectorFlujoVisualRRHH admite un único manifiesto JSON de tamaño acotado.
// La fuente se identifica por referencia y SHA-256 del Word de RRHH; su texto
// nunca se carga ni se devuelve por la API.
func LeerLectorFlujoVisualRRHH(
	contenido io.Reader,
) (*LectorFlujoVisualRRHH, error) {
	if contenido == nil {
		return nil, ErrManifestFlujoVisualRRHHInvalido
	}
	decodificador := json.NewDecoder(io.LimitReader(contenido, 128<<10))
	decodificador.DisallowUnknownFields()
	var manifest manifestFlujoVisualRRHH
	if err := decodificador.Decode(&manifest); err != nil ||
		decodificador.Decode(new(any)) != io.EOF {
		return nil, ErrManifestFlujoVisualRRHHInvalido
	}
	origen := domain.ReferenciaFlujo{
		DefinicionRef: manifest.FlujoOrigen.Referencia,
		Version:       manifest.FlujoOrigen.Version,
		HuellaSHA256:  manifest.FlujoOrigen.Huella,
	}
	if manifest.Esquema != esquemaManifestFlujoVisualRRHH ||
		!domain.ReferenciaOpacaValida(manifest.Fuente.Referencia) ||
		!huellaFlujoVisualRRHHValida(manifest.Fuente.Huella) ||
		origen.Validar() != nil || manifest.Presentacion.FaseActual != "" {
		return nil, ErrManifestFlujoVisualRRHHInvalido
	}
	fases := make(map[domain.ClaveFase]struct{}, len(manifest.Presentacion.Fases))
	var ordenAnterior uint16
	for _, fase := range manifest.Presentacion.Fases {
		if fase.Orden <= ordenAnterior {
			return nil, ErrManifestFlujoVisualRRHHInvalido
		}
		ordenAnterior = fase.Orden
		fases[fase.Clave] = struct{}{}
	}
	mapeos := make(map[domain.ClaveFase]domain.ClaveFase, len(manifest.Mapeos))
	for _, mapeo := range manifest.Mapeos {
		if !mapeo.Origen.Valida() || !mapeo.Presentacion.Valida() {
			return nil, ErrManifestFlujoVisualRRHHInvalido
		}
		if _, existe := fases[mapeo.Presentacion]; !existe {
			return nil, ErrManifestFlujoVisualRRHHInvalido
		}
		if _, repetido := mapeos[mapeo.Origen]; repetido {
			return nil, ErrManifestFlujoVisualRRHHInvalido
		}
		mapeos[mapeo.Origen] = mapeo.Presentacion
	}
	vinculo, err := calcularHuellaVinculoFlujoVisualRRHH(manifest, mapeos)
	if err != nil || vinculo != manifest.Presentacion.VinculoHuella ||
		manifest.Presentacion.Validar() != nil {
		return nil, ErrManifestFlujoVisualRRHHInvalido
	}
	return &LectorFlujoVisualRRHH{
		origen: origen, base: clonarPresentacionFlujoRRHH(manifest.Presentacion),
		mapeos: mapeos,
	}, nil
}

// Resolver exige la identidad completa del flujo administrativo ya leído. Si
// la fase actual no tiene una equivalencia acreditada, deja FaseActual vacía:
// el cliente conserva el rail, pero no destaca ni completa ningún paso.
func (l *LectorFlujoVisualRRHH) Resolver(
	ctx context.Context,
	origen domain.ReferenciaFlujo,
	faseActual domain.ClaveFase,
) (ports.PresentacionFlujoRRHH, error) {
	if ctx == nil {
		return ports.PresentacionFlujoRRHH{}, ErrManifestFlujoVisualRRHHInvalido
	}
	if err := ctx.Err(); err != nil {
		return ports.PresentacionFlujoRRHH{}, err
	}
	if l == nil || origen.Validar() != nil || l.origen != origen {
		return ports.PresentacionFlujoRRHH{}, ErrManifestFlujoVisualRRHHInvalido
	}
	resultado := clonarPresentacionFlujoRRHH(l.base)
	resultado.FaseActual = ""
	if visual, existe := l.mapeos[faseActual]; existe {
		resultado.FaseActual = visual
	}
	if resultado.Validar() != nil {
		return ports.PresentacionFlujoRRHH{}, ErrManifestFlujoVisualRRHHInvalido
	}
	return resultado, nil
}

func calcularHuellaVinculoFlujoVisualRRHH(m manifestFlujoVisualRRHH, mapeos map[domain.ClaveFase]domain.ClaveFase) (string, error) {
	type fuente struct {
		Referencia string `json:"referencia"`
		Huella     string `json:"huella_sha256"`
	}
	type origen struct {
		Referencia string `json:"referencia"`
		Version    uint64 `json:"version"`
		Huella     string `json:"huella_sha256"`
	}
	ordenados := make([]mapeoFaseOrigenVisualRRHH, 0, len(mapeos))
	for origen, presentacion := range mapeos {
		ordenados = append(ordenados, mapeoFaseOrigenVisualRRHH{Origen: origen, Presentacion: presentacion})
	}
	sort.Slice(ordenados, func(i, j int) bool { return ordenados[i].Origen < ordenados[j].Origen })
	payload := struct {
		Fuente fuente                      `json:"fuente"`
		Origen origen                      `json:"flujo_origen"`
		Mapeos []mapeoFaseOrigenVisualRRHH `json:"mapeos_fase_origen"`
	}{
		Fuente: fuente{m.Fuente.Referencia, m.Fuente.Huella}, Origen: origen{m.FlujoOrigen.Referencia, m.FlujoOrigen.Version, m.FlujoOrigen.Huella}, Mapeos: ordenados,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", ErrManifestFlujoVisualRRHHInvalido
	}
	suma := sha256.Sum256(append([]byte("VEC-CT-VINCULO-FLUJO-VISUAL-RRHH-V1\x00"), b...))
	return hex.EncodeToString(suma[:]), nil
}

func clonarPresentacionFlujoRRHH(
	origen ports.PresentacionFlujoRRHH,
) ports.PresentacionFlujoRRHH {
	copia := origen
	copia.Fases = append([]ports.FasePresentacionFlujoRRHH(nil), origen.Fases...)
	return copia
}

func huellaFlujoVisualRRHHValida(valor string) bool {
	return patronHuellaFlujoVisualRRHH.MatchString(valor) &&
		valor != strings.Repeat("0", 64)
}
