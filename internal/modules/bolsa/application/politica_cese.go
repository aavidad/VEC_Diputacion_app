package application

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ServicioConsultaPoliticaCese struct {
	lector      ports.ConsultaPoliticaCese
	autorizador ports.AutorizadorPoliticaCeseV3
	reloj       func() time.Time
}

func NuevoServicioConsultaPoliticaCese(lector ports.ConsultaPoliticaCese, autorizador ports.AutorizadorPoliticaCeseV3, reloj func() time.Time) (*ServicioConsultaPoliticaCese, error) {
	if dependenciaNulaPoliticaCese(lector) || dependenciaNulaPoliticaCese(autorizador) || reloj == nil {
		return nil, ports.ErrConsultaPoliticaCeseNoDisponible
	}
	return &ServicioConsultaPoliticaCese{lector: lector, autorizador: autorizador, reloj: reloj}, nil
}

// Consultar exige una concesión V3 nominal y confirmada para el recurso fijo
// antes de llamar al lector SQL B45. La decisión debe conceder únicamente el
// campo politica_cese y ninguna obligación que este caso no implemente.
func (s *ServicioConsultaPoliticaCese) Consultar(ctx context.Context, orden ports.OrdenConsultaPoliticaCese) (domain.PoliticaCese, error) {
	if s == nil || ctx == nil || dependenciaNulaPoliticaCese(s.lector) || dependenciaNulaPoliticaCese(s.autorizador) || s.reloj == nil || ctx.Err() != nil {
		return domain.PoliticaCese{}, ports.ErrConsultaPoliticaCeseNoDisponible
	}
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	actor, err := orden.ResultadoContexto.Clonar()
	vinculo, errVinculo := orden.Vinculo.Datos()
	if err != nil || errVinculo != nil || ahora.IsZero() ||
		actor.Contexto.Principal.AuthMethod == vd.AuthMethodDemo || vinculo.MetodoObservado == vd.AuthMethodDemo ||
		vinculo.Superficie != vd.SuperficieAutenticacionInternaCorporativaV1 ||
		orden.Vinculo.ValidarPara(actor) != nil || !orden.Vinculo.VigenteEn(ahora, actor) ||
		!vd.ReferenciaMotivoAutorizacionV2Valida(orden.Motivo) || orden.Correlacion.Validar() != nil {
		return domain.PoliticaCese{}, vd.ErrAutorizacionDenegada
	}
	recurso := vd.RecursoAutorizable{
		Referencia: ports.RecursoPoliticaCeseRef, ModuloID: "bolsa", Tipo: ports.TipoRecursoPoliticaCese,
		Ambitos: map[string]string{"modulo_ref": "bolsa"}, Atributos: map[string]string{},
	}
	solicitud, err := vd.NuevaSolicitudAutorizacionLigadaV3(vd.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: orden.Vinculo, ReferenciaMotivo: orden.Motivo,
		Accion: ports.AccionConsultarPoliticaCese, Recurso: recurso,
		Finalidad: ports.FinalidadConsultarPoliticaCese, Correlacion: orden.Correlacion,
	})
	if err != nil {
		return domain.PoliticaCese{}, vd.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, actor)
	if err != nil {
		if errors.Is(err, vecports.ErrFuenteAutorizacionNoDisponible) ||
			errors.Is(err, vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) ||
			errors.Is(err, vecports.ErrRegistroDecisionNoDisponible) {
			return domain.PoliticaCese{}, ports.ErrConsultaPoliticaCeseNoDisponible
		}
		return domain.PoliticaCese{}, vd.ErrAutorizacionDenegada
	}
	if dependenciaNulaPoliticaCese(exportador) {
		return domain.PoliticaCese{}, ports.ErrConsultaPoliticaCeseNoDisponible
	}
	ahora = s.reloj().UTC().Truncate(time.Microsecond)
	if ctx.Err() != nil || !concesionPoliticaCeseExacta(solicitud, decision, confirmacion, actor, ahora) ||
		!orden.Vinculo.VigenteEn(ahora, actor) {
		return domain.PoliticaCese{}, vd.ErrAutorizacionDenegada
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, actor, orden.Motivo, material, ports.AudienciaConsultarPoliticaCese) {
		return domain.PoliticaCese{}, vd.ErrAutorizacionDenegada
	}
	politica, err := s.lector.ConsultarPoliticaCese(ctx, material)
	if err != nil || politica.Validar() != nil {
		return domain.PoliticaCese{}, ports.ErrConsultaPoliticaCeseNoDisponible
	}
	return politica.Clonar(), nil
}

func concesionPoliticaCeseExacta(s vd.SolicitudAutorizacionLigadaV3, d vd.DecisionAutorizacionLigadaV3, c vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, actor vd.ResultadoContextoActorRegistradoV2, ahora time.Time) bool {
	if d.ValidarPara(s) != nil {
		return false
	}
	concedida, _, err := d.Resultado()
	switch {
	case err != nil, !concedida:
		return false
	}
	datos, err := s.Datos()
	switch {
	case err != nil:
		return false
	}
	registro, err := vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, datos.ReferenciaMotivo, actor)
	switch {
	case err != nil, c.ValidarPara(registro) != nil, !c.DentroDeVentanaEn(ahora):
		return false
	}
	canon, err := vd.RepresentacionCanonicaDecisionAutorizacionV3(d)
	var limites struct {
		Campos       []string `json:"campos_permitidos"`
		Obligaciones []string `json:"obligaciones"`
	}
	return err == nil && json.Unmarshal(canon, &limites) == nil &&
		slices.Equal(limites.Campos, []string{ports.CampoPoliticaCese}) && len(limites.Obligaciones) == 0
}

func dependenciaNulaPoliticaCese(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}
