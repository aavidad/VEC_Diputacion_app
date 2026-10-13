package bootstrap

import (
	"context"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type clavePersonaNativaB2 struct {
	fuente   *FuentesContextoIncorporacionB2
	selector bp.SelectorAceptacionNativaCT
}

type identidadPersonaNativaB2 struct {
	autenticacion, sesion, control, principal, perfil, contexto string
	revisionControl, versionContexto                            uint64
	huellaContexto, huellaManifiesto                            string
}

type lecturaPersonaNativaB2 struct {
	resultado bp.ResultadoPersonaAceptacionNativaCT
	identidad identidadPersonaNativaB2
}

func identidadConsultaNativaB2(actor bp.ActorConfiablePersonaAceptacionCT) (identidadPersonaNativaB2, error) {
	var cero identidadPersonaNativaB2
	if actor.Resultado.Validar() != nil || actor.Vinculo.ValidarPara(actor.Resultado) != nil {
		return cero, ct.ErrAutorizacionDenegada
	}
	d, err := actor.Vinculo.Datos()
	if err != nil {
		return cero, ct.ErrAutorizacionDenegada
	}
	return identidadPersonaNativaB2{d.AutenticacionRef, d.SesionRef, d.ControlSesionRef,
		d.PrincipalID, d.PerfilActivoRef, d.ContextoActorRef, d.ControlSesionRevision,
		d.ContextoActorVersion, actor.Resultado.HuellaSHA256,
		actor.Resultado.ManifiestoProcedenciaHuellaSHA256}, nil
}

func clonarResultadoPersonaNativaB2(r bp.ResultadoPersonaAceptacionNativaCT) bp.ResultadoPersonaAceptacionNativaCT {
	copia := r
	if r.Aceptacion != nil {
		a := *r.Aceptacion
		copia.Aceptacion = &a
	}
	if r.Persona != nil {
		p := *r.Persona
		copia.Persona = &p
	}
	if r.Vinculo != nil {
		v := *r.Vinculo
		copia.Vinculo = &v
	}
	return copia
}

// Traducción literal entre dos contratos propietarios. Ni CT ni Personal
// asignan participación, candidatura u orden: B113 los deriva de B110.
func selectorAceptacionNativaB2(x dom.SelectorBolsaPlanB2Nativo) bp.SelectorAceptacionNativaCT {
	return bp.SelectorAceptacionNativaCT{
		OrganizacionRef: x.OrganizacionRef, ExpedienteRef: x.ExpedienteRef,
		ClaveCT: x.ClaveCT, UnidadRef: x.UnidadRef, CategoriaRef: x.CategoriaRef,
		NecesidadRef: x.NecesidadRef, BolsaRef: x.BolsaRef,
		LlamamientoRef: x.LlamamientoRef, ReciboEmisionRef: x.ReciboEmisionRef,
		ReciboRespuestaRef: x.ReciboRespuestaRef,
	}
}

func anclajeNativoB113Valido(a *bp.AnclajeAceptacionNativaCT, s bp.SelectorAceptacionNativaCT) bool {
	if a == nil || a.SelectorAceptacionNativaCT != s || a.VersionExpediente == 0 ||
		a.VersionBolsa == 0 || a.VersionInstantanea == 0 || a.VersionPolitica == 0 ||
		a.OrdenSeleccionado == 0 || !dom.InstanteUTCCanonico(a.RespondidaEn) {
		return false
	}
	for _, ref := range []string{a.ActaRef, a.InstantaneaRef, a.PoliticaRef,
		a.ParticipacionRef, a.CandidatoRef, a.RespuestaRef} {
		if !bp.ReferenciaOpacaLlamamientoValida(ref) {
			return false
		}
	}
	return dom.HuellaPlanPersonalB2Valida(a.HuellaBolsaSHA256) &&
		dom.HuellaPlanPersonalB2Valida(a.HuellaInstantaneaSHA256) &&
		dom.HuellaPlanPersonalB2Valida(a.HuellaPoliticaSHA256)
}

func (f *FuentesContextoIncorporacionB2) consultarPersonaBolsaB2Nativa(
	ctx context.Context, x dom.SelectorBolsaPlanB2Nativo,
) (bp.ResultadoPersonaAceptacionNativaCT, error) {
	var cero bp.ResultadoPersonaAceptacionNativaCT
	if f == nil || ctx == nil || ctx.Err() != nil || !x.Valido() ||
		dependenciaBootstrapNula(f.c.BolsaNativa) || f.c.ActorBolsa == nil {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	holder, _ := ctx.Value(claveContextosNominalesIncorporacion{}).(*capturaContextosNominalesIncorporacion)
	if holder != nil {
		holder.mu.Lock()
		if holder.muPersonasB2 == nil {
			holder.muPersonasB2 = make(chan struct{}, 1)
		}
		mutex := holder.muPersonasB2
		holder.mu.Unlock()
		select {
		case mutex <- struct{}{}:
			defer func() { <-mutex }()
		case <-ctx.Done():
			return cero, ctx.Err()
		}
	}
	actor, err := f.c.ActorBolsa(ctx)
	if err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	selector := selectorAceptacionNativaB2(x)
	var propietario *perfilesNominalesIncorporacion
	var identidad identidadPersonaNativaB2
	if holder != nil {
		holder.mu.Lock()
		propietario = holder.propietario
		holder.mu.Unlock()
		if propietario != nil && propietario.soporte != nil {
			identidad, err = identidadConsultaNativaB2(actor)
			if err != nil {
				return cero, err
			}
			perfil := propietario.b2[bp.AccionConsultaPersonaAceptacionNativaCT]
			if perfil == nil || identidad.perfil != perfil.perfilRef() {
				return cero, ct.ErrAutorizacionDenegada
			}
			clave := clavePersonaNativaB2{f, selector}
			holder.mu.Lock()
			lectura, existe := holder.personasNativasB2[clave]
			holder.mu.Unlock()
			if existe {
				if identidad != lectura.identidad ||
					actor.Resultado.Contexto.PerfilActivoRef != perfil.perfilRef() {
					return cero, ct.ErrAutorizacionDenegada
				}
				if err := propietario.consumirInicialB2(ctx, perfil); err != nil {
					return cero, err
				}
				if lectura.resultado.Vinculo != nil &&
					propietario.soporte.reloj.Ahora().Before(lectura.resultado.Vinculo.VigenteHasta) {
					return clonarResultadoPersonaNativaB2(lectura.resultado), nil
				}
				holder.mu.Lock()
				delete(holder.personasNativasB2, clave)
				holder.mu.Unlock()
			}
		}
	}
	r, err := f.c.BolsaNativa.ConsultarPersonaAceptacionNativaCT(ctx,
		bp.SolicitudConsultaAceptacionNativaCT{Selector: selector, ActorConfiable: actor})
	if err != nil || ctx.Err() != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, err
	}
	if r.Estado == "pendiente" || r.Estado == "no_encontrada" {
		return r, nil
	}
	if r.Estado != "acreditado" || r.Aceptacion == nil || r.Persona == nil || r.Vinculo == nil ||
		!anclajeNativoB113Valido(r.Aceptacion, selector) || r.Persona.Version < 1 ||
		r.Vinculo.ProcedenciaVersion < 1 ||
		!dom.ReferenciaOpacaValida(r.Evidencia.DecisionRef) ||
		!dom.ReferenciaOpacaValida(r.Evidencia.AuditoriaRef) ||
		!dom.HuellaPlanPersonalB2Valida(r.Evidencia.ConsumoHuellaSHA256) ||
		!dom.InstanteUTCCanonico(r.Evidencia.ConsultadaEn) {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	if holder != nil && propietario != nil && propietario.soporte != nil &&
		propietario.soporte.reloj.Ahora().Before(r.Vinculo.VigenteHasta) {
		clave := clavePersonaNativaB2{f, selector}
		holder.mu.Lock()
		if holder.personasNativasB2 == nil {
			holder.personasNativasB2 = make(map[clavePersonaNativaB2]lecturaPersonaNativaB2)
		}
		holder.personasNativasB2[clave] = lecturaPersonaNativaB2{
			resultado: clonarResultadoPersonaNativaB2(r), identidad: identidad,
		}
		holder.mu.Unlock()
	}
	return r, nil
}

func (f *fuentesIncorporacionPersonalB2) acreditarAnclajeNativoB2(
	ctx context.Context, a ct.AntecedentesPlanNominalB2,
) (ct.AntecedentesPlanNominalB2, error) {
	if f == nil || ctx == nil || ctx.Err() != nil || a.BolsaNativa.IsZero() ||
		!a.Bolsa.IsZero() || dependenciaBootstrapNula(f.anclajeNativo) {
		return ct.AntecedentesPlanNominalB2{}, ct.ErrPlanNominalB2NoDisponible
	}
	actual, err := f.autoridad.contexto(ctx, bp.AccionConsultaAnclajeAceptacionNativaCT)
	if err != nil {
		return ct.AntecedentesPlanNominalB2{}, err
	}
	selector := selectorAceptacionNativaB2(a.BolsaNativa)
	r, err := f.anclajeNativo.ConsultarAnclajeAceptacionNativaCT(ctx,
		bp.SolicitudConsultaAceptacionNativaCT{Selector: selector,
			ActorConfiable: bp.ActorConfiablePersonaAceptacionCT{Vinculo: actual.Vinculo, Resultado: actual.Resultado}})
	if err != nil || ctx.Err() != nil {
		if ctx.Err() != nil {
			return ct.AntecedentesPlanNominalB2{}, ctx.Err()
		}
		return ct.AntecedentesPlanNominalB2{}, err
	}
	if r.Estado == "no_encontrada" {
		return ct.AntecedentesPlanNominalB2{}, ct.ErrPreparacionIncorporacionPendiente
	}
	if r.Estado != "acreditado" || r.Anclaje == nil ||
		!anclajeNativoB113Valido(r.Anclaje, selector) ||
		r.Anclaje.VersionExpediente > a.VersionExpediente ||
		a.OrganizacionRef != selector.OrganizacionRef || a.ExpedienteRef != selector.ExpedienteRef ||
		a.CategoriaRef != selector.CategoriaRef ||
		!dom.ReferenciaOpacaValida(r.Evidencia.DecisionRef) ||
		!dom.ReferenciaOpacaValida(r.Evidencia.AuditoriaRef) ||
		!dom.HuellaPlanPersonalB2Valida(r.Evidencia.ConsumoHuellaSHA256) ||
		!dom.InstanteUTCCanonico(r.Evidencia.ConsultadaEn) {
		return ct.AntecedentesPlanNominalB2{}, ct.ErrPlanNominalB2NoDisponible
	}
	return a, nil
}

func (f *fuentesIncorporacionPersonalB2) resolverPersonaInicialNativaB2(
	ctx context.Context, c inc.ContratoPlanNominal,
) (inc.PersonaSeleccionadaBolsa, error) {
	var cero inc.PersonaSeleccionadaBolsa
	x := c.SelectorBolsaNativa
	if !x.Valido() || !c.SelectorBolsa.IsZero() || x.OrganizacionRef != c.OrganizacionRef ||
		x.ExpedienteRef != c.ExpedienteRef {
		return cero, ct.ErrPlanNominalB2Conflicto
	}
	r, err := f.externas.consultarPersonaBolsaB2Nativa(ctx, x)
	if err != nil {
		return cero, err
	}
	if r.Estado != "acreditado" || r.Persona == nil || r.Vinculo == nil || r.Aceptacion == nil {
		return cero, ct.ErrPreparacionIncorporacionPendiente
	}
	version, ok := versionB2DesdeInt64(r.Persona.Version)
	if !ok || !anclajeNativoB113Valido(r.Aceptacion, selectorAceptacionNativaB2(x)) {
		return cero, ct.ErrPlanNominalB2NoDisponible
	}
	return inc.PersonaSeleccionadaBolsa{
		OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
		AceptacionRef: c.AceptacionRef, LlamamientoRef: x.LlamamientoRef,
		PersonaRef: r.Persona.Ref, PersonaVersion: version,
		ReciboRef: x.ReciboRespuestaRef,
		FuenteRef: r.Vinculo.ProcedenciaRef, FuenteVersion: r.Vinculo.ProcedenciaVersion,
		FuenteSHA256: r.Vinculo.ProcedenciaSHA256,
	}, nil
}

func (f *FuentesContextoIncorporacionB2) leerPersonaSeleccionadaNativaB2(
	ctx context.Context, c inc.ContratoPlanNominal,
) (inc.PersonaSeleccionadaBolsa, error) {
	var cero inc.PersonaSeleccionadaBolsa
	x := c.SelectorBolsaNativa
	if f == nil || ctx == nil || ctx.Err() != nil || c.Protocolo != inc.ProtocoloPersonalB2V1 ||
		!x.Valido() || !c.SelectorBolsa.IsZero() ||
		x.OrganizacionRef != c.OrganizacionRef || x.ExpedienteRef != c.ExpedienteRef ||
		x.CategoriaRef != c.CategoriaRef || x.LlamamientoRef != c.LlamamientoRef ||
		x.ReciboRespuestaRef != c.SeleccionReciboRef {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	r, err := f.consultarPersonaBolsaB2Nativa(ctx, x)
	if err != nil {
		return cero, errorFuenteContextoIncorporacionB2(ctx, err)
	}
	if r.Estado == "pendiente" || r.Estado == "no_encontrada" {
		return cero, ct.ErrPreparacionIncorporacionPendiente
	}
	if r.Estado != "acreditado" || r.Aceptacion == nil || r.Persona == nil || r.Vinculo == nil ||
		r.Persona.Version < 1 || r.Vinculo.ProcedenciaVersion < 1 ||
		!anclajeNativoB113Valido(r.Aceptacion, selectorAceptacionNativaB2(x)) {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	version, ok := versionB2DesdeInt64(r.Persona.Version)
	if !ok {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	return inc.PersonaSeleccionadaBolsa{
		OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
		AceptacionRef: c.AceptacionRef, LlamamientoRef: x.LlamamientoRef,
		SeleccionRef: c.SeleccionRef, VersionSeleccion: c.VersionSeleccion,
		ReciboRef:  x.ReciboRespuestaRef,
		PersonaRef: r.Persona.Ref, PersonaVersion: version,
		FuenteRef: r.Vinculo.ProcedenciaRef, FuenteVersion: r.Vinculo.ProcedenciaVersion,
		FuenteSHA256: r.Vinculo.ProcedenciaSHA256,
	}, nil
}
