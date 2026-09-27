package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"

	publicacion "vec-diputacion-granada/internal/modules/bolsa/publico/aplicacion"
	"vec-diputacion-granada/internal/modules/bolsa/publico/canonico"
)

var ErrFuenteCeseB10NoDisponible = errors.New("bolsa: fuente de publicacion B10 no disponible")

var patronDocumentoFuenteCeseB10 = regexp.MustCompile(`^\*{3}[0-9]{4}\*{2}$`)

// ParticipacionFuenteCeseB10 es material privado de una lectura de Bolsa. La
// referencia sólo comprueba inclusión y unicidad; jamás entra en BolsasV1.
// EstadoEfectivo procede de B6/B45 en el instante Corte, no de la situación
// histórica B2. FechaDisponible es el instante UTC de medianoche civil de
// Granada calculado por B45, no un DATE escaneado sin zona.
type ParticipacionFuenteCeseB10 struct {
	ParticipacionRef string
	Orden            int
	Documento        string
	EstadoEfectivo   string
	FechaDisponible  *time.Time
}

type BolsaFuenteCeseB10 struct {
	BolsaRef        string
	Categoria       string
	CategoriaClave  string
	Grupos          []string
	TipoLista       string
	VigenteDesde    time.Time
	VigenteHasta    *time.Time
	Total           int
	Participaciones []ParticipacionFuenteCeseB10
}

// CapturaFuenteCeseB10 debe ser una instantánea durable y consistente para el
// evento B49. El capturador verifica que EventoRef lleva a ParticipacionOrigen
// en BolsaRef y que todas las participaciones de todas las bolsas están en el
// mismo corte; al reintentar devuelve los mismos bytes y filas. La proyección
// V2 y su manifiesto conservan la autoridad externa de convocatorias.
type CapturaFuenteCeseB10 struct {
	OrigenPosicion        int64
	OrigenRef             string
	EventoRef             string
	Fase                  string
	BolsaRef              string
	ParticipacionOrigen   string
	Corte                 time.Time
	AnteriorActualizadaEn time.Time
	ProyeccionV2          []byte
	ManifiestoV2          []byte
	Bolsas                []BolsaFuenteCeseB10
}

// CapturadorFuenteCeseB10 pertenece a las autoridades de Bolsa y de la
// publicación V2. No se sustituye por la caché RRHH ni por lecturas públicas:
// estas no pueden probar inclusión de evento ni un corte único tras B45.
type CapturadorFuenteCeseB10 interface {
	CapturarCeseB10(context.Context, publicacion.CesePendienteB10) (CapturaFuenteCeseB10, error)
}

type fuenteInstantaneaCeseB10 struct{ capturador CapturadorFuenteCeseB10 }

var _ publicacion.FuenteInstantaneaCeseB10 = (*fuenteInstantaneaCeseB10)(nil)

func NuevaFuenteInstantaneaCeseB10(c CapturadorFuenteCeseB10) (publicacion.FuenteInstantaneaCeseB10, error) {
	if c == nil {
		return nil, ErrFuenteCeseB10NoDisponible
	}
	return &fuenteInstantaneaCeseB10{capturador: c}, nil
}

func (f *fuenteInstantaneaCeseB10) PrepararInstantaneaCeseB10(ctx context.Context, evento publicacion.CesePendienteB10) (publicacion.InstantaneaCeseB10, error) {
	if ctx == nil || f == nil || f.capturador == nil || ctx.Err() != nil {
		return publicacion.InstantaneaCeseB10{}, ErrFuenteCeseB10NoDisponible
	}
	captura, err := f.capturador.CapturarCeseB10(ctx, evento)
	if err != nil || captura.OrigenPosicion != evento.OrigenPosicion || captura.OrigenRef != evento.OrigenRef ||
		captura.EventoRef != evento.EventoRef || captura.Fase != evento.Fase || captura.BolsaRef != evento.BolsaRef ||
		captura.ParticipacionOrigen == "" || captura.Corte.IsZero() || captura.Corte.Location() != time.UTC ||
		!captura.Corte.Equal(captura.Corte.Truncate(time.Microsecond)) ||
		!captura.Corte.After(captura.AnteriorActualizadaEn) {
		return publicacion.InstantaneaCeseB10{}, ErrFuenteCeseB10NoDisponible
	}
	var v2 struct {
		Fuente struct {
			ActualizadaEn time.Time `json:"actualizada_en"`
		} `json:"fuente"`
	}
	var manifiesto canonico.ManifiestoPublicoV2
	if json.Unmarshal(captura.ProyeccionV2, &v2) != nil || json.Unmarshal(captura.ManifiestoV2, &manifiesto) != nil ||
		!v2.Fuente.ActualizadaEn.Equal(captura.Corte) || !manifiesto.Fuente.ActualizadaEn.Equal(captura.Corte) {
		return publicacion.InstantaneaCeseB10{}, ErrFuenteCeseB10NoDisponible
	}
	bolsas, err := proyectarBolsasCeseB10(captura)
	if err != nil {
		return publicacion.InstantaneaCeseB10{}, err
	}
	contenido, err := json.Marshal(bolsas)
	if err != nil {
		return publicacion.InstantaneaCeseB10{}, ErrFuenteCeseB10NoDisponible
	}
	// El validador canónico coteja íntegramente V2 con el manifiesto externo,
	// y B10 con ambos. No se publica nada al fallar cualquiera de las tres piezas.
	material, err := canonico.PrepararMaterialPublicacionV3(captura.ProyeccionV2, captura.ManifiestoV2, contenido)
	if err != nil {
		return publicacion.InstantaneaCeseB10{}, ErrFuenteCeseB10NoDisponible
	}
	return publicacion.InstantaneaCeseB10{
		OrigenPosicionIncluida: captura.OrigenPosicion, OrigenRefIncluido: captura.OrigenRef,
		FaseIncluida: captura.Fase, ProyeccionV2: material.ProyeccionV2,
		ManifiestoV2: append([]byte(nil), captura.ManifiestoV2...), BolsasV1: material.BolsasV1,
	}, nil
}

func proyectarBolsasCeseB10(c CapturaFuenteCeseB10) (canonico.BolsasManifiestoV1, error) {
	if len(c.Bolsas) > 128 {
		return canonico.BolsasManifiestoV1{}, ErrFuenteCeseB10NoDisponible
	}
	resultado := canonico.BolsasManifiestoV1{GeneradoEn: c.Corte, Bolsas: make([]canonico.BolsaManifiestoV1, 0, len(c.Bolsas))}
	var origenIncluido bool
	vistas := make(map[string]struct{}, len(c.Bolsas))
	participacionesGlobales := make(map[string]struct{})
	for _, b := range c.Bolsas {
		if _, duplicada := vistas[b.BolsaRef]; duplicada || b.BolsaRef == "" || b.Total != len(b.Participaciones) || b.Total > 100000 || b.Total < 0 {
			return canonico.BolsasManifiestoV1{}, ErrFuenteCeseB10NoDisponible
		}
		if b.VigenteDesde.After(c.Corte) || (b.VigenteHasta != nil && !b.VigenteHasta.After(c.Corte)) {
			return canonico.BolsasManifiestoV1{}, ErrFuenteCeseB10NoDisponible
		}
		vistas[b.BolsaRef] = struct{}{}
		publica := canonico.BolsaManifiestoV1{
			BolsaRef: b.BolsaRef, Categoria: b.Categoria, CategoriaClave: b.CategoriaClave,
			Grupos: append([]string(nil), b.Grupos...), TipoLista: b.TipoLista,
			VigenteDesde: b.VigenteDesde, VigenteHasta: b.VigenteHasta, Total: b.Total,
			Posiciones: make([]canonico.PosicionBolsaManifiestoV1, 0, b.Total),
		}
		filas := append([]ParticipacionFuenteCeseB10(nil), b.Participaciones...)
		sort.Slice(filas, func(i, j int) bool { return filas[i].Orden < filas[j].Orden })
		participaciones := make(map[string]struct{}, len(filas))
		for indice, p := range filas {
			if p.ParticipacionRef == "" || p.Orden != indice+1 || !patronDocumentoFuenteCeseB10.MatchString(p.Documento) {
				return canonico.BolsasManifiestoV1{}, ErrFuenteCeseB10NoDisponible
			}
			if _, duplicada := participaciones[p.ParticipacionRef]; duplicada {
				return canonico.BolsasManifiestoV1{}, ErrFuenteCeseB10NoDisponible
			}
			if _, duplicada := participacionesGlobales[p.ParticipacionRef]; duplicada {
				return canonico.BolsasManifiestoV1{}, ErrFuenteCeseB10NoDisponible
			}
			participaciones[p.ParticipacionRef] = struct{}{}
			participacionesGlobales[p.ParticipacionRef] = struct{}{}
			if b.BolsaRef == c.BolsaRef && p.ParticipacionRef == c.ParticipacionOrigen {
				origenIncluido = true
			}
			estado, err := estadoEfectivoPublicoCeseB10(p, c.Corte)
			if err != nil {
				return canonico.BolsasManifiestoV1{}, err
			}
			publica.Posiciones = append(publica.Posiciones, canonico.PosicionBolsaManifiestoV1{
				Orden: p.Orden, DocumentoEnmascarado: p.Documento, EstadoClave: estado,
			})
		}
		resultado.Bolsas = append(resultado.Bolsas, publica)
	}
	if !origenIncluido {
		return canonico.BolsasManifiestoV1{}, ErrFuenteCeseB10NoDisponible
	}
	return resultado, nil
}

func estadoEfectivoPublicoCeseB10(p ParticipacionFuenteCeseB10, corte time.Time) (string, error) {
	switch strings.TrimSpace(p.EstadoEfectivo) {
	case "disponible", "no_disponible", "excluido", "ocupado", "renuncia_pendiente":
		return p.EstadoEfectivo, nil
	case "trabajando", "pendiente_incorporacion":
		return "ocupado", nil
	case "renuncia":
		return "renuncia_pendiente", nil
	case "disponible_desde":
		if p.FechaDisponible == nil || p.FechaDisponible.IsZero() {
			return "", ErrFuenteCeseB10NoDisponible
		}
		if !p.FechaDisponible.After(corte) {
			return "disponible", nil
		}
		return "no_disponible", nil
	default:
		return "", ErrFuenteCeseB10NoDisponible
	}
}
