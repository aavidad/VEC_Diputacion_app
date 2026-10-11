package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// SolicitudFirmantesMerito contiene material preparado internamente por RRHH.
// El transporte no puede construir Contexto/Vinculo ni designaciones válidas.
type SolicitudFirmantesMerito struct {
	Contexto              vec.ResultadoContextoActorRegistradoV2
	Vinculo               vec.VinculoAutenticacionActorV2
	Accion                string
	Material              domain.MaterialDecisionFirmantes
	VersionRosterEsperada uint64
	HuellaAnteriorSHA256  string
	ClaveIdempotencia     string
	Firmantes             []domain.FirmanteDesignado
	SustituyeRef          string
	Sustituto             domain.FirmanteDesignado
}

// ServicioFirmantesMerito prepara snapshots; no registra firmas ni modifica
// Hecho, puntos o ranking. Sin los tres puertos de autoridad no hay escritura.
type ServicioFirmantesMerito struct {
	catalogo      domain.CatalogoRolesFirmantes
	autorizador   ports.AutorizadorFirmantesMerito
	decisiones    ports.FuenteDecisionFirmantes
	designaciones ports.FuenteDesignacionesFirmantes
	repositorio   ports.RepositorioPlanFirmantes
	reloj         vecports.Reloj
}

func NuevoServicioFirmantesMerito(c domain.CatalogoRolesFirmantes, a ports.AutorizadorFirmantesMerito,
	f ports.FuenteDecisionFirmantes, d ports.FuenteDesignacionesFirmantes, r ports.RepositorioPlanFirmantes, reloj vecports.Reloj) (*ServicioFirmantesMerito, error) {
	if c.Validar() != nil || nulo(a) || nulo(f) || nulo(d) || nulo(r) || nulo(reloj) {
		return nil, ports.ErrPlanFirmantesNoDisponible
	}
	c.Roles = append([]domain.DefinicionRolFirmante(nil), c.Roles...)
	return &ServicioFirmantesMerito{c, a, f, d, r, reloj}, nil
}

func (s *ServicioFirmantesMerito) Preparar(ctx context.Context, q SolicitudFirmantesMerito) (ports.ResultadoGuardarPlanFirmantes, error) {
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.decisiones) || nulo(s.designaciones) || nulo(s.repositorio) || nulo(s.reloj) {
		return ports.ResultadoGuardarPlanFirmantes{}, ports.ErrPlanFirmantesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoGuardarPlanFirmantes{}, err
	}
	if q.Contexto.Validar() != nil || q.Vinculo.ValidarPara(q.Contexto) != nil || q.Material.Validar() != nil ||
		!domain.ReferenciaValida(q.ClaveIdempotencia) || !solicitudFirmantesCoherente(q) {
		return ports.ResultadoGuardarPlanFirmantes{}, domain.ErrFirmantesMerito
	}
	preimagen, err := preimagenFirmantes(q)
	if err != nil {
		return ports.ResultadoGuardarPlanFirmantes{}, domain.ErrFirmantesMerito
	}
	h := sha256.Sum256(preimagen)
	solicitudAutoridad := ports.SolicitudAutoridadFirmantes{Accion: q.Accion, HechoRef: q.Material.HechoRef,
		DecisionRef: q.Material.DecisionRef, VersionRosterEsperada: q.VersionRosterEsperada,
		HuellaSolicitudSHA256: hex.EncodeToString(h[:])}
	aut, err := s.autorizador.AutorizarPreparacionFirmantes(ctx, solicitudAutoridad)
	if err != nil {
		return ports.ResultadoGuardarPlanFirmantes{}, err
	}
	ahora := s.reloj.Ahora().UTC()
	if !autoridadFirmantesCoincide(aut, solicitudAutoridad, q.Contexto.Contexto, ahora) {
		return ports.ResultadoGuardarPlanFirmantes{}, ports.ErrPlanFirmantesNoDisponible
	}
	recuperada, existe, err := s.repositorio.RecuperarConfirmadaAutorizada(ctx, aut, q.ClaveIdempotencia)
	if err != nil {
		return ports.ResultadoGuardarPlanFirmantes{}, err
	}
	if existe {
		if !recuperacionFirmantesCoincide(recuperada, aut, q, s.catalogo) {
			return ports.ResultadoGuardarPlanFirmantes{}, ports.ErrPlanFirmantesConflicto
		}
		out := recuperada.Resultado
		out.Plan = clonarPlanFirmantes(out.Plan)
		return out, nil
	}
	materialAutorizado, err := s.decisiones.ObtenerMaterialExactoAutorizado(ctx, aut, q.Material)
	if err != nil {
		return ports.ResultadoGuardarPlanFirmantes{}, err
	}
	if materialAutorizado.Validar() != nil || !materialesSolicitudFirmantesIguales(materialAutorizado, q.Material) {
		return ports.ResultadoGuardarPlanFirmantes{}, ports.ErrPlanFirmantesNoDisponible
	}
	var plan domain.PlanFirmantesMerito
	if q.Accion == ports.AccionPrepararPlanFirmantes {
		plan, err = domain.PrepararPlanFirmantesMerito(q.Material, s.catalogo, 1, q.Firmantes)
	} else {
		anterior, leerErr := s.repositorio.ObtenerActualAutorizado(ctx, aut, q.Material.DecisionRef)
		if leerErr != nil {
			return ports.ResultadoGuardarPlanFirmantes{}, leerErr
		}
		if anterior.Material.Validar() != nil || anterior.VersionRoster != q.VersionRosterEsperada ||
			anterior.HuellaRosterSHA256 != q.HuellaAnteriorSHA256 || anterior.Material.HechoRef != q.Material.HechoRef {
			return ports.ResultadoGuardarPlanFirmantes{}, ports.ErrPlanFirmantesConflicto
		}
		switch q.Accion {
		case ports.AccionCambiarPlanFirmantes:
			plan, err = anterior.NuevaVersionContenido(q.Material, s.catalogo, q.Firmantes)
		case ports.AccionSustituirPlanFirmantes:
			if !materialesSolicitudFirmantesIguales(q.Material, anterior.Material) {
				return ports.ResultadoGuardarPlanFirmantes{}, ports.ErrPlanFirmantesConflicto
			}
			plan, err = anterior.SustituirFirmante(s.catalogo, q.SustituyeRef, q.Sustituto)
		}
	}
	if err != nil || plan.HuellaRosterSHA256 == "" {
		return ports.ResultadoGuardarPlanFirmantes{}, domain.ErrFirmantesMerito
	}
	designaciones, err := s.designaciones.ComprobarDesignacionesExactas(ctx, clonarPlanFirmantes(plan))
	if err != nil {
		return ports.ResultadoGuardarPlanFirmantes{}, err
	}
	if designaciones.HuellaRosterSHA256 != plan.HuellaRosterSHA256 || !domain.ReferenciaValida(designaciones.FuenteRef) ||
		designaciones.FuenteVersion == 0 || !domain.ReferenciaValida(designaciones.CompetenciaRef) ||
		designaciones.VigenteHasta.IsZero() || !ahora.Before(designaciones.VigenteHasta) {
		return ports.ResultadoGuardarPlanFirmantes{}, ports.ErrPlanFirmantesNoDisponible
	}
	orden := ports.OrdenGuardarPlanFirmantes{Plan: clonarPlanFirmantes(plan), VersionRosterEsperada: q.VersionRosterEsperada,
		HuellaAnteriorSHA256: q.HuellaAnteriorSHA256, ClaveIdempotencia: q.ClaveIdempotencia,
		Autoridad: aut, Designaciones: designaciones}
	resultado, err := s.repositorio.GuardarVersion(ctx, orden)
	if err != nil {
		return ports.ResultadoGuardarPlanFirmantes{}, err
	}
	if !domain.ReferenciaValida(resultado.ReciboRef) || !reflect.DeepEqual(resultado.Plan, clonarPlanFirmantes(plan)) {
		return ports.ResultadoGuardarPlanFirmantes{}, ports.ErrPlanFirmantesNoDisponible
	}
	resultado.Plan = clonarPlanFirmantes(resultado.Plan)
	return resultado, nil
}

func recuperacionFirmantesCoincide(r ports.RecuperacionPlanFirmantes, a ports.AutoridadFirmantes, q SolicitudFirmantesMerito, c domain.CatalogoRolesFirmantes) bool {
	p := r.Resultado.Plan
	if r.Accion != q.Accion || r.ActorRef != a.ActorRef || r.ClaveIdempotencia != q.ClaveIdempotencia ||
		r.HuellaSolicitudSHA256 != a.HuellaSolicitudSHA256 || r.VersionRosterEsperada != q.VersionRosterEsperada ||
		!r.Resultado.Repetida || !domain.ReferenciaValida(r.Resultado.ReciboRef) ||
		p.Material.Validar() != nil || !materialesSolicitudFirmantesIguales(p.Material, q.Material) ||
		p.VersionRoster != q.VersionRosterEsperada+1 || !huellaSolicitudFirmantesValida(p.HuellaRosterSHA256) ||
		p.Estado != "preparado_congelado" || len(p.Firmas) != 0 {
		return false
	}
	if q.Accion == ports.AccionPrepararPlanFirmantes || q.Accion == ports.AccionCambiarPlanFirmantes {
		esperado, err := domain.PrepararPlanFirmantesMerito(q.Material, c, q.VersionRosterEsperada+1, q.Firmantes)
		if err != nil || p.HuellaRosterSHA256 != esperado.HuellaRosterSHA256 {
			return false
		}
	}
	if q.Accion == ports.AccionSustituirPlanFirmantes {
		encontrado := false
		for _, f := range p.Firmantes {
			if f.DesignacionRef == q.SustituyeRef {
				return false
			}
			if f.DesignacionRef == q.Sustituto.DesignacionRef && f != q.Sustituto {
				return false
			}
			if f == q.Sustituto {
				encontrado = true
			}
		}
		if !encontrado {
			return false
		}
	}
	return true
}

func solicitudFirmantesCoherente(q SolicitudFirmantesMerito) bool {
	switch q.Accion {
	case ports.AccionPrepararPlanFirmantes:
		return q.VersionRosterEsperada == 0 && q.HuellaAnteriorSHA256 == "" && q.SustituyeRef == "" &&
			q.Sustituto == (domain.FirmanteDesignado{}) && len(q.Firmantes) > 0
	case ports.AccionCambiarPlanFirmantes:
		return q.VersionRosterEsperada > 0 && huellaSolicitudFirmantesValida(q.HuellaAnteriorSHA256) &&
			q.SustituyeRef == "" && q.Sustituto == (domain.FirmanteDesignado{}) && len(q.Firmantes) > 0
	case ports.AccionSustituirPlanFirmantes:
		return q.VersionRosterEsperada > 0 && huellaSolicitudFirmantesValida(q.HuellaAnteriorSHA256) &&
			domain.ReferenciaValida(q.SustituyeRef) && q.Sustituto.SustituyeRef == q.SustituyeRef && len(q.Firmantes) == 0
	default:
		return false
	}
}

func huellaSolicitudFirmantesValida(v string) bool {
	if len(v) != 64 {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil && v != "0000000000000000000000000000000000000000000000000000000000000000"
}

func preimagenFirmantes(q SolicitudFirmantesMerito) ([]byte, error) {
	return json.Marshal(struct {
		Accion    string                           `json:"accion"`
		Material  domain.MaterialDecisionFirmantes `json:"material"`
		Esperada  uint64                           `json:"version_roster_esperada"`
		Anterior  string                           `json:"huella_anterior_sha256"`
		Clave     string                           `json:"clave_idempotencia"`
		Firmantes []domain.FirmanteDesignado       `json:"firmantes"`
		Sustituye string                           `json:"sustituye_ref"`
		Sustituto domain.FirmanteDesignado         `json:"sustituto"`
	}{q.Accion, q.Material, q.VersionRosterEsperada, q.HuellaAnteriorSHA256, q.ClaveIdempotencia,
		q.Firmantes, q.SustituyeRef, q.Sustituto})
}

func autoridadFirmantesCoincide(a ports.AutoridadFirmantes, q ports.SolicitudAutoridadFirmantes, actor vec.ContextoActor, ahora time.Time) bool {
	return a.Accion == q.Accion && a.ActorRef == actor.PersonaRef && a.PerfilRef == actor.PerfilActivoRef &&
		a.HechoRef == q.HechoRef && a.DecisionRef == q.DecisionRef &&
		a.VersionRosterEsperada == q.VersionRosterEsperada && a.HuellaSolicitudSHA256 == q.HuellaSolicitudSHA256 &&
		domain.ReferenciaValida(a.PermisoRef) && !a.VigenteHasta.IsZero() && ahora.Before(a.VigenteHasta)
}

func materialesSolicitudFirmantesIguales(a, b domain.MaterialDecisionFirmantes) bool {
	x, ex := json.Marshal(a)
	y, ey := json.Marshal(b)
	return ex == nil && ey == nil && string(x) == string(y)
}

func clonarPlanFirmantes(p domain.PlanFirmantesMerito) domain.PlanFirmantesMerito {
	p.Material.Evidencias = append([]vec.ReferenciaDocumento(nil), p.Material.Evidencias...)
	p.Firmantes = append([]domain.FirmanteDesignado(nil), p.Firmantes...)
	p.Firmas = append([]domain.FirmaConstatada(nil), p.Firmas...)
	return p
}
