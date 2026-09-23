package mibolsa

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var ErrServicioMiBolsaInvalido = errors.New("bolsa: servicio mi bolsa invalido")

// Orden procede solamente de la frontera autenticada, registrada y
// revalidada. El cliente HTTP no puede completar ninguno de sus campos.
type Orden struct {
	ResultadoContexto dominiovec.ResultadoContextoActorRegistradoV2
	Vinculo           dominiovec.VinculoAutenticacionActorV2
	Motivo            dominiovec.ReferenciaEntradaCatalogo
	Correlacion       dominiovec.ReferenciaCorrelacionAutorizacionV2
}

type Servicio struct {
	consulta    puertosbolsa.ConsultaMiBolsa
	autorizador puertosvec.AutorizadorSolicitudLigadaV3
	proveedor   puertosbolsa.ProveedorMaterialMiBolsa
	reloj       puertosvec.Reloj
}

func Nuevo(consulta puertosbolsa.ConsultaMiBolsa, autorizador puertosvec.AutorizadorSolicitudLigadaV3, proveedor puertosbolsa.ProveedorMaterialMiBolsa, reloj puertosvec.Reloj) (*Servicio, error) {
	if nula(consulta) || nula(autorizador) || nula(proveedor) || nula(reloj) {
		return nil, ErrServicioMiBolsaInvalido
	}
	return &Servicio{consulta: consulta, autorizador: autorizador, proveedor: proveedor, reloj: reloj}, nil
}

func (s *Servicio) Consultar(ctx context.Context, orden Orden) (puertosbolsa.InstantaneaMiBolsa, error) {
	if ctx == nil || s == nil || nula(s.consulta) || nula(s.autorizador) || nula(s.proveedor) || nula(s.reloj) {
		return puertosbolsa.InstantaneaMiBolsa{}, ErrServicioMiBolsaInvalido
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, err
	}
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	resultadoActor, candidato, err := validarOrden(orden, ahora)
	if err != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, err
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: "mi-bolsa:" + candidato, ModuloID: puertosbolsa.ModuloMiBolsa, Tipo: puertosbolsa.TipoRecursoMiBolsa, Ambitos: map[string]string{"candidato_ref": candidato}, Atributos: map[string]string{"propiedad": "candidato"}}
	if recurso.Validar() != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, errors.Join(dominiovec.ErrAutorizacionDenegada, puertosbolsa.ErrConsultaMiBolsaInvalida)
	}
	nominal, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: orden.Vinculo, ReferenciaMotivo: orden.Motivo,
		Accion: puertosbolsa.AccionConsultarMiBolsa, Recurso: recurso,
		Finalidad: puertosbolsa.FinalidadMiBolsa, Correlacion: orden.Correlacion,
	})
	if err != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, denegar(err)
	}
	decision, confirmacion, err := s.autorizador.ExigirSolicitudLigadaV3(ctx, nominal, resultadoActor)
	if err != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, denegar(err)
	}
	ahora = s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ctx.Err() != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, ctx.Err()
	}
	if !decisionExacta(nominal, decision, confirmacion, resultadoActor, ahora) {
		return puertosbolsa.InstantaneaMiBolsa{}, denegar(nil)
	}
	exportador, err := s.proveedor.EmitirMaterialMiBolsa(ctx, nominal, resultadoActor, decision, confirmacion)
	if err != nil || nula(exportador) {
		return puertosbolsa.InstantaneaMiBolsa{}, errors.Join(puertosbolsa.ErrMaterialMiBolsaNoDisponible, err)
	}
	if ctx.Err() != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, ctx.Err()
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, errors.Join(puertosbolsa.ErrMaterialMiBolsaNoDisponible, err)
	}
	ahora = s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ctx.Err() != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, ctx.Err()
	}
	if _, _, err = validarOrden(orden, ahora); err != nil || !materialExacto(material, nominal, decision, confirmacion, resultadoActor, ahora) {
		return puertosbolsa.InstantaneaMiBolsa{}, denegar(err)
	}
	solicitud := puertosbolsa.SolicitudConsultaMiBolsa{CandidatoRef: candidato, Material: material, ConsultadaEn: ahora}
	resultado, err := s.consulta.ConsultarMiBolsa(ctx, solicitud)
	if err != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, err
	}
	if err := validarResultado(resultado, ahora); err != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, errors.Join(puertosbolsa.ErrResultadoMiBolsaInvalido, err)
	}
	return resultado, nil
}

func validarOrden(o Orden, ahora time.Time) (dominiovec.ResultadoContextoActorRegistradoV2, string, error) {
	r, ea := o.ResultadoContexto.Clonar()
	a := r.Contexto
	d, ev := o.Vinculo.Datos()
	if ahora.IsZero() || ea != nil || ev != nil || a.Principal.AuthMethod == dominiovec.AuthMethodDemo || d.MetodoObservado == dominiovec.AuthMethodDemo || d.Superficie != dominiovec.SuperficieAutenticacionExternaPersonalV1 || o.Vinculo.ValidarPara(r) != nil || !o.Vinculo.VigenteEn(ahora, r) || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(o.Motivo) || o.Correlacion.Validar() != nil {
		return dominiovec.ResultadoContextoActorRegistradoV2{}, "", errors.Join(dominiovec.ErrAutorizacionDenegada, puertosbolsa.ErrConsultaMiBolsaInvalida)
	}
	var candidatos []string
	for _, v := range a.Instantanea.Vinculos {
		if v.Tipo == dominiovec.TipoReferenciaContextoActorCandidato && v.VigenteEn(ahora) {
			candidatos = append(candidatos, v.Referencia)
		}
	}
	if len(candidatos) != 1 {
		return dominiovec.ResultadoContextoActorRegistradoV2{}, "", errors.Join(dominiovec.ErrAutorizacionDenegada, puertosbolsa.ErrConsultaMiBolsaInvalida)
	}
	return r, candidatos[0], nil
}
func validarResultado(r puertosbolsa.InstantaneaMiBolsa, ahora time.Time) error {
	if !r.ConsultadaEn.Equal(ahora) {
		return puertosbolsa.ErrResultadoMiBolsaInvalido
	}
	for _, p := range r.Participaciones {
		if p.Bolsa == "" || p.Categoria == "" || p.Version == 0 || p.OrdenInicial == 0 || p.TotalInstantanea < p.OrdenInicial || p.EstadoBolsa == "" || p.VigenteDesde.IsZero() || !p.VigenteDesde.Before(ahora) && !p.VigenteDesde.Equal(ahora) || p.VigenteHasta != nil && !p.VigenteHasta.After(p.VigenteDesde) {
			return puertosbolsa.ErrResultadoMiBolsaInvalido
		}
		if s := p.SituacionActual; s != nil {
			valida := false
			for _, estado := range dominiobolsa.SituacionesParticipacion() {
				if s.Estado == estado {
					valida = true
					break
				}
			}
			if !valida || s.Desde.IsZero() || s.Desde.After(ahora) || s.Hasta != nil && s.Hasta.Before(s.Desde) ||
				(s.Estado == dominiobolsa.SituacionDisponibleDesde) != (s.FechaDisponible != nil) ||
				s.FechaDisponible != nil && !s.FechaDisponible.After(s.Desde) {
				return puertosbolsa.ErrResultadoMiBolsaInvalido
			}
		}
		if l := p.UltimoLlamamiento; l != nil {
			if l.EmitidoEn.IsZero() || l.EmitidoEn.After(ahora) || l.Canal != "correo" || (l.Resultado != "enviado" && l.Resultado != "no_enviado") {
				return puertosbolsa.ErrResultadoMiBolsaInvalido
			}
		}
	}
	return nil
}
func nula(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return x.IsNil()
	}
	return false
}

func denegar(err error) error {
	return errors.Join(dominiovec.ErrAutorizacionDenegada, puertosbolsa.ErrConsultaMiBolsaInvalida, err)
}

func decisionExacta(s dominiovec.SolicitudAutorizacionLigadaV3, d dominiovec.DecisionAutorizacionLigadaV3, c puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, r dominiovec.ResultadoContextoActorRegistradoV2, ahora time.Time) bool {
	if d.ValidarPara(s) != nil {
		return false
	}
	datos, err := s.Datos()
	if err != nil {
		return false
	}
	orden, err := puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, datos.ReferenciaMotivo, r)
	if err != nil || c.ValidarPara(orden) != nil || !c.DentroDeVentanaEn(ahora) {
		return false
	}
	// Esta proyección solo restringe una decisión nominal ya cotejada: no crea
	// capacidades desde JSON ni acepta una lista más amplia o una obligación ignota.
	canonica, err := dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(d)
	var limites struct {
		Campos       []string `json:"campos_permitidos"`
		Obligaciones []string `json:"obligaciones"`
	}
	return err == nil && json.Unmarshal(canonica, &limites) == nil && len(limites.Campos) == 1 && limites.Campos[0] == puertosbolsa.CampoMiBolsa && len(limites.Obligaciones) == 0
}

func materialExacto(m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, s dominiovec.SolicitudAutorizacionLigadaV3, d dominiovec.DecisionAutorizacionLigadaV3, c puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, r dominiovec.ResultadoContextoActorRegistradoV2, ahora time.Time) bool {
	if m.ValidarEstructura() != nil || d.ValidarPara(s) != nil {
		return false
	}
	datos, err := s.Datos()
	if err != nil {
		return false
	}
	dc, ed := dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(d)
	mc, em := dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(datos.ReferenciaMotivo)
	huella, eh := datos.Recurso.HuellaContextoAutorizacionSHA256()
	desde, hasta, ev := d.VentanaValidez()
	resumen := m.ResumenCapacidad()
	confirmacion, ec := c.Datos()
	hd, hm := sha256.Sum256(dc), sha256.Sum256(mc)
	proyeccion, ep := dominiovec.ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(m.PayloadVECAD3())
	if ep != nil {
		return false
	}
	cabecera, ep := proyeccion.Cabecera()
	if ep != nil {
		return false
	}
	payload, ep := dominiovec.SerializarMensajeAtestacionAutorizacionV3(cabecera, d, datos.ReferenciaMotivo, r)
	if ep != nil || !bytes.Equal(payload, m.PayloadVECAD3()) {
		return false
	}
	return ec == nil && resumen.DecisionRef() == confirmacion.DecisionRef && resumen.DecisionHuellaSHA256() == hex.EncodeToString(hd[:]) && resumen.MotivoHuellaSHA256() == hex.EncodeToString(hm[:]) && ed == nil && em == nil && eh == nil && ev == nil && !ahora.Before(desde) && ahora.Before(hasta) &&
		bytes.Equal(dc, m.DecisionCanonica()) && bytes.Equal(mc, m.MotivoCanonico()) && bytes.Equal(r.RepresentacionCanonica, m.ContextoActorCanonico()) &&
		resumen.Operacion() == puertosbolsa.AccionConsultarMiBolsa && resumen.EfectoRef() == datos.Recurso.Referencia && resumen.EfectoHuellaSHA256() == huella && resumen.AudienciaConsumo() == puertosbolsa.AudienciaMiBolsa &&
		resumen.ContextoRef() == r.RegistroContextoRef && resumen.ContextoHuellaSHA256() == r.HuellaSHA256 && m.PersonaVersion() == r.Contexto.Instantanea.PersonaVersion && m.PerfilVersion() == r.Contexto.Instantanea.PerfilVersion && !ahora.Before(resumen.EmitidaEn()) && ahora.Before(resumen.ExpiraEn())
}
