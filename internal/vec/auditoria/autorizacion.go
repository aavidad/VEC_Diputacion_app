package auditoria

import (
	"encoding/json"
	"reflect"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type decisionConsultaCanonica struct {
	Concedida        bool     `json:"concedida"`
	Accion           string   `json:"accion"`
	ModuloID         string   `json:"modulo_id"`
	TipoRecurso      string   `json:"tipo_recurso"`
	RecursoRef       string   `json:"recurso_ref"`
	Finalidad        string   `json:"finalidad"`
	CamposPermitidos []string `json:"campos_permitidos"`
	Obligaciones     []string `json:"obligaciones"`
}

// ValidarConsultaAutorizadaEn permite a cada propietario cotejar la capacidad
// antes de entrar a SQL. El consumidor AD3 nominal vuelve a verificar firma,
// gobierno, revocacion y consumo dentro de la transaccion de lectura.
func ValidarConsultaAutorizadaEn(q ConsultaAutorizada, ahora time.Time) error {
	if q.Filtro.Validar() != nil || !instanteValido(ahora.UTC().Truncate(time.Microsecond)) {
		return ErrDenegada
	}
	datos, err := q.Solicitud.Datos()
	if err != nil || datos.Accion != AccionConsultar || datos.Finalidad != q.Filtro.FinalidadRef ||
		datos.ReferenciaMotivo.Referencia() != q.Filtro.MotivoRef ||
		validarRecursoConsultaLigadoAlFiltro(datos.Recurso, q.Filtro) != nil ||
		q.ResultadoContexto.Validar() != nil ||
		datos.VinculoAutenticacionActor.ValidarPara(q.ResultadoContexto) != nil ||
		!datos.VinculoAutenticacionActor.VigenteEn(ahora, q.ResultadoContexto) ||
		q.Decision.ValidarPara(q.Solicitud) != nil {
		return ErrDenegada
	}
	concedida, _, err := q.Decision.Resultado()
	if err != nil || !concedida {
		return ErrDenegada
	}
	canon, err := vecdomain.RepresentacionCanonicaDecisionAutorizacionV3(q.Decision)
	if err != nil {
		return ErrDenegada
	}
	var d decisionConsultaCanonica
	if json.Unmarshal(canon, &d) != nil || !d.Concedida || d.Accion != AccionConsultar ||
		d.ModuloID != ModuloAutorizacion || d.TipoRecurso != TipoRecurso ||
		d.RecursoRef != q.Filtro.ExpedienteRef || d.Finalidad != q.Filtro.FinalidadRef ||
		!reflect.DeepEqual(d.CamposPermitidos, camposPermitidos) || len(d.Obligaciones) != 0 {
		return ErrDenegada
	}
	if !vecports.MaterialAtestadoLigadoV3(q.Solicitud, q.Decision, q.Confirmacion,
		q.ResultadoContexto, datos.ReferenciaMotivo, q.Material, AudienciaConsumo) {
		return ErrDenegada
	}
	resumen := q.Material.ResumenCapacidad()
	if ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) {
		return ErrDenegada
	}
	return nil
}

func ValidarConsultaAutorizada(q ConsultaAutorizada) error {
	return ValidarConsultaAutorizadaEn(q, time.Now().UTC().Truncate(time.Microsecond))
}
