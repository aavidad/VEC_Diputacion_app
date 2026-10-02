package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"vec-diputacion-granada/internal/modules/carrera/domain"
	"vec-diputacion-granada/internal/modules/carrera/ports"
)

var ErrCotejoPromocionNoDisponible = errors.New("carrera.cotejo.error.no_disponible")
var ErrConsultaPromocion = errors.New("carrera.cotejo.error.consulta_invalida")

type ResultadoRequisitoPromocion struct {
	RequisitoRef      string `json:"requisito_ref"`
	EstadoMostrado    string `json:"estado_mostrado"`
	MotivoGuardaClave string `json:"motivo_guarda_clave"`
}

type CotejoPromocionSintetico struct {
	Resultados       []ResultadoRequisitoPromocion    `json:"resultados"`
	EstadoGlobal     string                           `json:"estado_global"`
	EtiquetaClave    string                           `json:"etiqueta_clave"`
	BasesVerificadas bool                             `json:"bases_verificadas"`
	Consulta         ports.ConsultaPromocionSintetica `json:"consulta"`
	Dictamen         ports.DictamenPromocionSintetico `json:"dictamen_aportado"`
	Pendientes       []string                         `json:"pendientes"`
}

// HuellaConsultaPromocion identifica solo el material local de preparación.
// No verifica la huella de bases aportada ni acredita firma o aprobación.
func HuellaConsultaPromocion(q ports.ConsultaPromocionSintetica) (string, error) {
	if !consultaPromocionValida(q) {
		return "", ErrConsultaPromocion
	}
	b, err := json.Marshal(q)
	if err != nil {
		return "", ErrConsultaPromocion
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// CotejarPromocionSintetica consume el dictamen del productor del ensayo. No
// interpreta bases, evalúa predicados ni convierte declaraciones en hechos.
func (Servicio) CotejarPromocionSintetica(ctx context.Context, q ports.ConsultaPromocionSintetica, l ports.LectorCotejoProcesoSelectivoSintetico) (CotejoPromocionSintetico, error) {
	huella, err := HuellaConsultaPromocion(q)
	if err != nil {
		return CotejoPromocionSintetico{}, err
	}
	if ctx == nil || ctx.Err() != nil || l == nil {
		return CotejoPromocionSintetico{}, ErrCotejoPromocionNoDisponible
	}
	q = copiarConsultaPromocion(q)
	d, err := l.ConsultarCotejoPromocionSintetico(ctx, copiarConsultaPromocion(q))
	if err != nil || ctx.Err() != nil || !dictamenPromocionValido(q, d, huella) {
		return CotejoPromocionSintetico{}, ErrCotejoPromocionNoDisponible
	}
	return CotejoPromocionSintetico{Resultados: resultadosPromocion(q, d), EstadoGlobal: "pendiente", EtiquetaClave: "carrera.cotejo.dictamen_ensayo", Consulta: q, Dictamen: copiarDictamenPromocion(d), Pendientes: []string{"carrera.cotejo.pendiente.seleccion_nominal", "carrera.cotejo.pendiente.lector_personal_rum", "carrera.cotejo.pendiente.h08", "carrera.cotejo.pendiente.admision"}}, nil
}

func tokenPromocion(s string) bool { return len(s) > 0 && len(s) <= 256 && strings.TrimSpace(s) == s }
func shaPromocion(s string) bool {
	if len(s) != 64 {
		return false
	}
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 32 && strings.ToLower(s) == s
}
func consultaPromocionValida(q ports.ConsultaPromocionSintetica) bool {
	if !instantePromocion(q.InstanteReferencia) || q.Alcance != domain.AlcanceSintetico || !shaPromocion(q.HuellaBasesAportada) || len(q.Requisitos) == 0 || len(q.Requisitos) > 64 || len(q.Hechos) > 128 {
		return false
	}
	for _, s := range []string{q.CasoRef, q.PersonaRef, q.ProcesoRef, q.ProcesoVersion, q.BasesRef, q.BasesVersion, q.InstantaneaHechosRef, q.InstantaneaHechosVersion} {
		if !tokenPromocion(s) {
			return false
		}
	}
	vistos := map[string]bool{}
	for _, r := range q.Requisitos {
		if !tokenPromocion(r.Referencia) || !tokenPromocion(r.Version) || !tokenPromocion(r.Hito) || !fechaPromocion(r.HitoFecha) || !tokenPromocion(r.Fuente) || !tokenPromocion(r.FuenteVersion) || len(r.ReglaRef) > 256 || (r.Representacion != "estructurada" && r.Representacion != "texto_libre") || vistos[r.Referencia] {
			return false
		}
		vistos[r.Referencia] = true
	}
	vistos = map[string]bool{}
	for _, h := range q.Hechos {
		for _, s := range []string{h.Referencia, h.Version, h.EstadoAportado, h.Fuente, h.FuenteVersion} {
			if !tokenPromocion(s) {
				return false
			}
		}
		if (h.EstadoAportado != "declarado" && h.EstadoAportado != "pendiente" && h.EstadoAportado != "acreditado" && h.EstadoAportado != "rechazado") || !fechaPromocion(h.VigenteDesde) || (h.VigenteHasta != "" && (!fechaPromocion(h.VigenteHasta) || h.VigenteHasta < h.VigenteDesde)) || len(h.Evidencia) > 256 || vistos[h.Referencia] {
			return false
		}
		vistos[h.Referencia] = true
	}
	return true
}
func dictamenPromocionValido(q ports.ConsultaPromocionSintetica, d ports.DictamenPromocionSintetico, huella string) bool {
	if !tokenPromocion(d.Referencia) || !tokenPromocion(d.Version) || !instantePromocion(d.EvaluadoEn) || d.Alcance != domain.AlcanceSintetico || d.HuellaConsultaSHA256 != huella || !tokenPromocion(d.EvaluadorRef) || !tokenPromocion(d.EvaluadorVersion) || len(d.Comprobaciones) != len(q.Requisitos) {
		return false
	}
	reqs := map[string]ports.RequisitoPromocionSintetico{}
	for _, r := range q.Requisitos {
		reqs[r.Referencia] = r
	}
	hechos := map[string]bool{}
	for _, h := range q.Hechos {
		hechos[h.Referencia] = true
	}
	vistos := map[string]bool{}
	for _, c := range d.Comprobaciones {
		r, ok := reqs[c.RequisitoRef]
		if !ok || vistos[c.RequisitoRef] || c.RequisitoVersion != r.Version || c.Hito != r.Hito || c.HitoFecha != r.HitoFecha || !tokenPromocion(c.MotivoClave) || !tokenPromocion(c.Fuente) || !tokenPromocion(c.FuenteVersion) || len(c.HechosReferencias) > 128 {
			return false
		}
		if c.EstadoAportado != "cumple" && c.EstadoAportado != "no_cumple" && c.EstadoAportado != "pendiente" {
			return false
		}
		refs := map[string]bool{}
		for _, ref := range c.HechosReferencias {
			if !hechos[ref] || refs[ref] {
				return false
			}
			refs[ref] = true
		}
		vistos[c.RequisitoRef] = true
	}
	return true
}
func copiarConsultaPromocion(q ports.ConsultaPromocionSintetica) ports.ConsultaPromocionSintetica {
	q.Requisitos = append([]ports.RequisitoPromocionSintetico(nil), q.Requisitos...)
	q.Hechos = append([]ports.HechoPromocionSintetico(nil), q.Hechos...)
	return q
}
func copiarDictamenPromocion(d ports.DictamenPromocionSintetico) ports.DictamenPromocionSintetico {
	d.Comprobaciones = append([]ports.ComprobacionPromocionSintetica(nil), d.Comprobaciones...)
	for i := range d.Comprobaciones {
		d.Comprobaciones[i].HechosReferencias = append([]string(nil), d.Comprobaciones[i].HechosReferencias...)
	}
	return d
}

func fechaPromocion(s string) bool {
	d, e := time.Parse(time.DateOnly, s)
	return e == nil && d.Year() > 0
}
func instantePromocion(s string) bool {
	d, e := time.Parse(time.RFC3339Nano, s)
	return e == nil && d.Year() > 0
}

// La guarda preserva el estado recibido y solo impide mostrar una comprobación
// concluyente sin estructura y soporte probatorio identificados. No interpreta
// la condición, resuelve equivalencias ni evalúa vigencia contra hitos.
func resultadosPromocion(q ports.ConsultaPromocionSintetica, d ports.DictamenPromocionSintetico) []ResultadoRequisitoPromocion {
	reqs := map[string]ports.RequisitoPromocionSintetico{}
	for _, r := range q.Requisitos {
		reqs[r.Referencia] = r
	}
	hechos := map[string]ports.HechoPromocionSintetico{}
	for _, h := range q.Hechos {
		hechos[h.Referencia] = h
	}
	out := make([]ResultadoRequisitoPromocion, 0, len(d.Comprobaciones))
	for _, c := range d.Comprobaciones {
		r := reqs[c.RequisitoRef]
		estado, motivo := c.EstadoAportado, ""
		if estado != "pendiente" {
			if r.Representacion == "texto_libre" {
				motivo = "carrera.cotejo.guarda.texto_libre"
			} else if !tokenPromocion(r.ReglaRef) {
				motivo = "carrera.cotejo.guarda.regla_ausente"
			} else {
				if len(c.HechosReferencias) == 0 {
					motivo = "carrera.cotejo.guarda.soporte_ausente"
				}
				for _, ref := range c.HechosReferencias {
					h := hechos[ref]
					if h.EstadoAportado != "acreditado" || strings.TrimSpace(h.Evidencia) == "" {
						motivo = "carrera.cotejo.guarda.soporte_ausente"
					}
				}
			}
			if motivo != "" {
				estado = "pendiente"
			}
		}
		out = append(out, ResultadoRequisitoPromocion{RequisitoRef: c.RequisitoRef, EstadoMostrado: estado, MotivoGuardaClave: motivo})
	}
	return out
}
