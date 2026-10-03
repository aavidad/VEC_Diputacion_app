package auditoria

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const maxSecuenciaNominal uint64 = 9007199254740991

func (f FiltroNominal) Validar() error {
	if (f.ActorRef == "" && f.RecursoRef == "" && f.Accion == "") ||
		(f.ActorRef != "" && !referenciaExacta(f.ActorRef, 512)) ||
		(f.RecursoRef != "" && !referenciaExacta(f.RecursoRef, 512)) ||
		(f.Accion != "" && !referenciaExacta(f.Accion, 160)) ||
		!instanteValido(f.Desde) || !instanteValido(f.Hasta) || !f.Hasta.After(f.Desde) ||
		f.Hasta.Sub(f.Desde) > MaximoIntervalo || f.Limite == 0 || f.Limite > MaximoRegistros ||
		f.AntesSecuencia > maxSecuenciaNominal || !referenciaExacta(f.FinalidadRef, 128) ||
		!referenciaExacta(f.MotivoRef, 128) {
		return ErrDenegada
	}
	return nil
}

// HuellaFiltroNominal binds every selector, the page and the catalogue refs.
// LF framing is unambiguous because references cannot contain controls.
func HuellaFiltroNominal(f FiltroNominal) (string, error) {
	if f.Validar() != nil {
		return "", ErrDenegada
	}
	preimagen := strings.Join([]string{"vec.auditoria.filtro.nominal.v1", f.ActorRef, f.RecursoRef,
		f.Accion, f.Desde.Format(formatoInstante), f.Hasta.Format(formatoInstante),
		strconv.FormatUint(uint64(f.Limite), 10), strconv.FormatUint(f.AntesSecuencia, 10),
		f.FinalidadRef, f.MotivoRef}, "\n")
	h := sha256.Sum256([]byte(preimagen))
	return hex.EncodeToString(h[:]), nil
}

func RecursoFiltroNominal(f FiltroNominal) (vecdomain.RecursoAutorizable, error) {
	h, err := HuellaFiltroNominal(f)
	if err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	r := vecdomain.RecursoAutorizable{Referencia: "consulta-auditoria:sha256:" + h,
		ModuloID: ModuloAutorizacion, Tipo: TipoRecursoNominal,
		Ambitos:   map[string]string{"fuente": "ad3_v3_interna"},
		Atributos: map[string]string{"filtro_sha256": h}}
	if r.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ErrDenegada
	}
	return r, nil
}

func ValidarConsultaNominalAutorizadaEn(q ConsultaNominalAutorizada, ahora time.Time) error {
	r, err := RecursoFiltroNominal(q.Filtro)
	datos, errDatos := q.Solicitud.Datos()
	if err != nil || errDatos != nil || !instanteValido(ahora) ||
		datos.Accion != AccionConsultarNominal || datos.Finalidad != q.Filtro.FinalidadRef ||
		datos.ReferenciaMotivo.Referencia() != q.Filtro.MotivoRef || !reflect.DeepEqual(datos.Recurso, r) ||
		q.ResultadoContexto.Validar() != nil || datos.VinculoAutenticacionActor.ValidarPara(q.ResultadoContexto) != nil ||
		!datos.VinculoAutenticacionActor.VigenteEn(ahora, q.ResultadoContexto) || q.Decision.ValidarPara(q.Solicitud) != nil {
		return ErrDenegada
	}
	concedida, _, err := q.Decision.Resultado()
	if err != nil || !concedida {
		return ErrDenegada
	}
	canon, err := vecdomain.RepresentacionCanonicaDecisionAutorizacionV3(q.Decision)
	var d decisionConsultaCanonica
	if err != nil || json.Unmarshal(canon, &d) != nil || !d.Concedida || d.Accion != AccionConsultarNominal ||
		d.ModuloID != ModuloAutorizacion || d.TipoRecurso != TipoRecursoNominal || d.RecursoRef != r.Referencia ||
		d.Finalidad != q.Filtro.FinalidadRef || !reflect.DeepEqual(d.CamposPermitidos, CamposPermitidosNominal()) || len(d.Obligaciones) != 0 ||
		!vecports.MaterialAtestadoLigadoV3(q.Solicitud, q.Decision, q.Confirmacion, q.ResultadoContexto,
			datos.ReferenciaMotivo, q.Material, AudienciaConsumoNominal) {
		return ErrDenegada
	}
	resumen := q.Material.ResumenCapacidad()
	if ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) {
		return ErrDenegada
	}
	return nil
}

func registroNominalValido(r RegistroNominal, f FiltroNominal) bool {
	return referenciaExacta(r.AuditoriaRef, 512) && r.Secuencia > 0 && r.Secuencia <= maxSecuenciaNominal &&
		referenciaExacta(r.ActorRef, 512) && referenciaExacta(r.PerfilActivoRef, 512) &&
		referenciaExacta(r.AsignacionRef, 512) && referenciaExacta(r.VersionRolRef, 512) &&
		referenciaExacta(r.ModuloID, 128) && referenciaExacta(r.Accion, 160) && referenciaExacta(r.RecursoRef, 512) &&
		referenciaExacta(r.FinalidadRef, 128) && referenciaExacta(r.CorrelacionRef, 512) && vecdomain.SuperficieAutenticacionActorV1(r.Canal).Valida() &&
		instanteValido(r.RegistradaEn) && !r.RegistradaEn.Before(f.Desde) && r.RegistradaEn.Before(f.Hasta) &&
		r.TipoRegistro == TipoRegistroConsumoConfirmado && (f.ActorRef == "" || r.ActorRef == f.ActorRef) &&
		(f.RecursoRef == "" || r.RecursoRef == f.RecursoRef) && (f.Accion == "" || r.Accion == f.Accion) &&
		(f.AntesSecuencia == 0 || r.Secuencia < f.AntesSecuencia)
}
