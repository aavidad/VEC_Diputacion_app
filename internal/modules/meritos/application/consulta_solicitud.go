package application

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConsultaPropia    = "meritos.hecho.consultar_propio"
	FinalidadConsultaPropia = "consulta_hecho_propio"
	AudienciaConsultaPropia = "vec_meritos.hecho.consultar_propio.v1"
	EsquemaConsultaPropia   = "vec.meritos.hecho.consulta_propia.v1"
)

type SolicitudConsultaPropia struct {
	Vinculo     vec.VinculoAutenticacionActorV2
	Contexto    vec.ResultadoContextoActorRegistradoV2
	Correlacion vec.ReferenciaCorrelacionAutorizacionV2
	Motivo      vec.ReferenciaEntradaCatalogo
	HechoRef    string
}

type selectorConsultaPropia struct {
	Esquema    string `json:"esquema"`
	HechoRef   string `json:"hecho_ref"`
	PersonaRef string `json:"persona_ref"`
}

func canonSelectorConsultaPropia(hecho, persona string) ([]byte, error) {
	if !domain.ReferenciaValida(hecho) || !domain.ReferenciaValida(persona) {
		return nil, ErrSolicitud
	}
	return json.Marshal(selectorConsultaPropia{EsquemaConsultaPropia, hecho, persona})
}

func huellaConsulta(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func (s SolicitudConsultaPropia) validarContexto() error {
	if s.Contexto.Validar() != nil || s.Vinculo.ValidarPara(s.Contexto) != nil || s.Correlacion.Validar() != nil {
		return vec.ErrAutorizacionDenegada
	}
	return nil
}

func ordenConsultaPropia(s SolicitudConsultaPropia) (ports.OrdenConsultaPropia, error) {
	if !vec.ReferenciaMotivoAutorizacionV2Valida(s.Motivo) {
		return ports.OrdenConsultaPropia{}, ErrSolicitud
	}
	persona := s.Contexto.Contexto.PersonaRef
	b, err := canonSelectorConsultaPropia(s.HechoRef, persona)
	if err != nil {
		return ports.OrdenConsultaPropia{}, err
	}
	return ports.OrdenConsultaPropia{SelectorCanonico: b, HechoRef: s.HechoRef, PersonaRef: persona,
		HuellaConsultaSHA256: huellaConsulta(b), Motivo: s.Motivo}, nil
}

// ValidarOrdenConsultaPropia coteja selector y concesión antes del consumo.
// El repositorio conserva la revalidación central dentro de su transacción.
func ValidarOrdenConsultaPropia(o ports.OrdenConsultaPropia) error {
	b, err := canonSelectorConsultaPropia(o.HechoRef, o.PersonaRef)
	if err != nil || !bytes.Equal(b, o.SelectorCanonico) || o.HuellaConsultaSHA256 != huellaConsulta(b) ||
		!vec.ReferenciaMotivoAutorizacionV2Valida(o.Motivo) {
		return ports.ErrConsultaNoDisponible
	}
	a := o.Autorizacion
	d, err := a.Solicitud.Datos()
	if err != nil || a.Contexto.Validar() != nil || d.VinculoAutenticacionActor.ValidarPara(a.Contexto) != nil {
		return ports.ErrConsultaNoDisponible
	}
	v, err := d.VinculoAutenticacionActor.Datos()
	if err != nil || v.PrincipalID != o.PersonaRef || a.Contexto.Contexto.PersonaRef != o.PersonaRef ||
		d.Accion != AccionConsultaPropia || d.Finalidad != FinalidadConsultaPropia || d.ReferenciaMotivo != o.Motivo ||
		d.Recurso.Referencia != o.HechoRef || d.Recurso.ModuloID != "meritos" || d.Recurso.Tipo != "hecho" ||
		len(d.Recurso.Ambitos) != 2 || len(d.Recurso.Atributos) != 0 ||
		d.Recurso.Ambitos["persona_ref"] != o.PersonaRef || d.Recurso.Ambitos["huella_consulta_sha256"] != o.HuellaConsultaSHA256 ||
		exigirProyeccionConsulta(a) != nil ||
		!vecports.MaterialAtestadoLigadoV3(a.Solicitud, a.Decision, a.Confirmacion, a.Contexto, o.Motivo, a.Material, AudienciaConsultaPropia) {
		return ports.ErrConsultaNoDisponible
	}
	return nil
}

func exigirProyeccionConsulta(a ports.AutorizacionOperacion) error {
	if a.Decision.ExigirProyeccionPara(a.Solicitud, []string{"hecho_actual", "recibo_consulta"}, []string{"auditar"}) != nil {
		return vec.ErrAutorizacionDenegada
	}
	r, err := a.Decision.RestriccionesProyeccionPara(a.Solicitud)
	if err != nil || len(r.Obligaciones) != 1 || r.Obligaciones[0] != "auditar" {
		return vec.ErrAutorizacionDenegada
	}
	return nil
}
