package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ServicioRegistroPropioV1 consume exclusivamente una acreditación obtenida
// de un proveedor institucional. El solicitante aún carece de contexto V3;
// quien autoriza este primer efecto es otro actor central ya registrado.
type ServicioRegistroPropioV1 struct {
	acreditador  ports.AcreditadorInstitucionalRegistroPropioV1
	equivalencia ports.EquivalenciaPersonaRegistroPropioV1
	autorizador  ports.AutorizadorRegistroPropioV1
	registro     ports.RegistroPropioV1
	ahora        func() time.Time
}

func NuevoServicioRegistroPropioV1(a ports.AcreditadorInstitucionalRegistroPropioV1, e ports.EquivalenciaPersonaRegistroPropioV1, z ports.AutorizadorRegistroPropioV1, r ports.RegistroPropioV1, ahora func() time.Time) (*ServicioRegistroPropioV1, error) {
	if nuloRegistroPropio(a) || nuloRegistroPropio(e) || nuloRegistroPropio(z) || nuloRegistroPropio(r) || ahora == nil {
		return nil, ports.ErrRegistroPropioNoDisponible
	}
	return &ServicioRegistroPropioV1{a, e, z, r, ahora}, nil
}

func (s *ServicioRegistroPropioV1) Registrar(ctx context.Context, q ports.SolicitudRegistroPropioV1) (domain.ReciboRegistroPropioV1, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || q.OperacionRef == "" || q.CredencialRef == "" ||
		q.ResultadoActor.Validar() != nil || q.VinculoActor.ValidarPara(q.ResultadoActor) != nil ||
		q.Correlacion.Validar() != nil || !domain.ReferenciaMotivoAutorizacionV2Valida(q.Motivo) {
		return domain.ReciboRegistroPropioV1{}, ports.ErrRegistroPropioNoDisponible
	}
	acreditacion, err := s.acreditador.AcreditarRegistroPropio(ctx, q.CredencialRef)
	if err != nil || acreditacion.CredencialRef != q.CredencialRef || acreditacion.ValidarEn(s.ahora()) != nil {
		return domain.ReciboRegistroPropioV1{}, ports.ErrRegistroPropioNoDisponible
	}
	equivalencia, err := s.equivalencia.ResolverEquivalenciaPersona(ctx, q.OperacionRef, acreditacion)
	if err != nil || !equivalenciaRegistroPropioValida(equivalencia, acreditacion) {
		return domain.ReciboRegistroPropioV1{}, ports.ErrRegistroPropioNoDisponible
	}
	actor := q.ResultadoActor.Contexto.PersonaRef
	if actor == equivalencia.PersonaRef {
		return domain.ReciboRegistroPropioV1{}, ports.ErrRegistroPropioNoDisponible
	}
	// La concesión V3 se confirma antes de iniciar la transacción de efecto.
	// Una instantánea SERIALIZABLE abierta antes no vería esta concesión.
	entrada, huella, err := entradaRegistroPropio(q.OperacionRef, acreditacion, equivalencia)
	if err != nil {
		return domain.ReciboRegistroPropioV1{}, ports.ErrRegistroPropioNoDisponible
	}
	recurso := domain.RecursoAutorizable{Referencia: q.OperacionRef, ModuloID: "vec", Tipo: "registro_propio", Ambitos: map[string]string{"sujeto_ref": acreditacion.SujetoRef}, Atributos: map[string]string{"entrada_sha256": huella, "equivalencia_ref": equivalencia.PruebaRef}}
	recursoCanonico, err := json.Marshal(struct {
		Ambitos   map[string]string `json:"ambitos"`
		Atributos map[string]string `json:"atributos"`
	}{recurso.Ambitos, recurso.Atributos})
	if err != nil {
		return domain.ReciboRegistroPropioV1{}, ports.ErrRegistroPropioNoDisponible
	}
	nominal, err := domain.NuevaSolicitudAutorizacionLigadaV3(domain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: q.VinculoActor, ReferenciaMotivo: q.Motivo, Accion: ports.AccionRegistroPropioV1, Recurso: recurso, Finalidad: ports.FinalidadRegistroPropioV1, Correlacion: q.Correlacion})
	if err != nil {
		return domain.ReciboRegistroPropioV1{}, ports.ErrRegistroPropioNoDisponible
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, nominal, q.ResultadoActor)
	if err != nil || nuloRegistroPropio(exportador) || decision.ValidarPara(nominal) != nil {
		return domain.ReciboRegistroPropioV1{}, ports.ErrRegistroPropioNoDisponible
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialRegistroPropioValido(material, nominal, decision, confirmacion, q.ResultadoActor, recurso, s.ahora()) {
		return domain.ReciboRegistroPropioV1{}, ports.ErrRegistroPropioNoDisponible
	}
	orden := ports.OrdenRegistroPropioV1{OperacionRef: q.OperacionRef, Acreditacion: acreditacion, Equivalencia: equivalencia, ActorRef: actor, EntradaCanonica: entrada, RecursoCanonico: recursoCanonico, Solicitud: nominal, Decision: decision, Confirmacion: confirmacion, Material: material}
	recibo, err := s.registro.RegistrarPropio(ctx, orden, func(bloqueado context.Context) error {
		if err := s.acreditador.RevalidarRegistroPropio(bloqueado, acreditacion); err != nil || acreditacion.ValidarEn(s.ahora()) != nil {
			return ports.ErrRegistroPropioNoDisponible
		}
		segunda, err := s.equivalencia.ResolverEquivalenciaPersona(bloqueado, q.OperacionRef, acreditacion)
		if err != nil || !reflect.DeepEqual(segunda, equivalencia) || bloqueado.Err() != nil {
			return ports.ErrRegistroPropioNoDisponible
		}
		return nil
	})
	if err != nil || recibo.ValidarPendiente() != nil || recibo.OperacionRef != q.OperacionRef || recibo.ProcedenciaRef != acreditacion.ProcedenciaRef || recibo.ProcedenciaVersion != acreditacion.ProcedenciaVersion || recibo.ProcedenciaSHA256 != acreditacion.ProcedenciaSHA256 || equivalencia.PersonaRef != "" && recibo.PersonaRef != equivalencia.PersonaRef {
		return domain.ReciboRegistroPropioV1{}, ports.ErrRegistroPropioNoDisponible
	}
	return recibo, nil
}

func equivalenciaRegistroPropioValida(e ports.ResultadoEquivalenciaPersonaRegistroPropioV1, a domain.AcreditacionInstitucionalRegistroPropioV1) bool {
	return e.SujetoRef == a.SujetoRef && e.PruebaRef == a.EquivalenciaRef && e.PruebaVersion > 0 && len(e.PruebaSHA256) == 64 &&
		(e.Nueva && e.PersonaRef == "" || !e.Nueva && e.PersonaRef != "")
}

func entradaRegistroPropio(operacion string, a domain.AcreditacionInstitucionalRegistroPropioV1, e ports.ResultadoEquivalenciaPersonaRegistroPropioV1) ([]byte, string, error) {
	b, err := json.Marshal(struct {
		Operacion    string
		Acreditacion domain.AcreditacionInstitucionalRegistroPropioV1
		Equivalencia ports.ResultadoEquivalenciaPersonaRegistroPropioV1
	}{operacion, a, e})
	if err != nil {
		return nil, "", err
	}
	h := sha256.Sum256(b)
	return b, hex.EncodeToString(h[:]), nil
}

func materialRegistroPropioValido(m ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, s domain.SolicitudAutorizacionLigadaV3, d domain.DecisionAutorizacionLigadaV3, c ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, resultado domain.ResultadoContextoActorRegistradoV2, recurso domain.RecursoAutorizable, ahora time.Time) bool {
	if m.ValidarEstructura() != nil || d.ValidarPara(s) != nil || resultado.Validar() != nil {
		return false
	}
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, func() domain.ReferenciaEntradaCatalogo { x, _ := s.Datos(); return x.ReferenciaMotivo }(), resultado)
	if err != nil || c.ValidarPara(orden) != nil {
		return false
	}
	cd, err := c.Datos()
	if err != nil || !c.DentroDeVentanaEn(cd.RegistradaEn) {
		return false
	}
	datos, err := s.Datos()
	if err != nil || datos.Accion != ports.AccionRegistroPropioV1 || datos.Finalidad != ports.FinalidadRegistroPropioV1 || !reflect.DeepEqual(datos.Recurso, recurso) {
		return false
	}
	decisionCanonica, err := domain.RepresentacionCanonicaDecisionAutorizacionV3(d)
	if err != nil {
		return false
	}
	motivoCanonico, err := domain.RepresentacionCanonicaMotivoAutorizacionV2(datos.ReferenciaMotivo)
	if err != nil {
		return false
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return false
	}
	hd := sha256.Sum256(decisionCanonica)
	hm := sha256.Sum256(motivoCanonico)
	r := m.ResumenCapacidad()
	return bytes.Equal(m.DecisionCanonica(), decisionCanonica) && bytes.Equal(m.MotivoCanonico(), motivoCanonico) &&
		bytes.Equal(m.ContextoActorCanonico(), resultado.RepresentacionCanonica) && r.DecisionRef() == cd.DecisionRef &&
		r.DecisionHuellaSHA256() == hex.EncodeToString(hd[:]) && r.MotivoHuellaSHA256() == hex.EncodeToString(hm[:]) &&
		r.Operacion() == ports.AccionRegistroPropioV1 && r.AudienciaConsumo() == ports.AudienciaRegistroPropioV1 &&
		r.EfectoRef() == recurso.Referencia && r.EfectoHuellaSHA256() == huellaRecurso &&
		r.ContextoRef() == resultado.RegistroContextoRef && r.ContextoHuellaSHA256() == resultado.HuellaSHA256 &&
		m.PersonaVersion() == resultado.Contexto.Instantanea.PersonaVersion && m.PerfilVersion() == resultado.Contexto.Instantanea.PerfilVersion &&
		!ahora.Before(r.EmitidaEn()) && ahora.Before(r.ExpiraEn())
}

func nuloRegistroPropio(x any) bool {
	if x == nil {
		return true
	}
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}
