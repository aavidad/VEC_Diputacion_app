package memory

import (
	"context"
	"math"
	"sort"
	"sync"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// AlmacenAutorizacionMemoria es un adaptador de desarrollo y pruebas. Mantiene
// instantaneas inmutables y el registro de decisiones protegido para acceso
// concurrente; no sustituye al registro duradero y sellado de produccion.
type AlmacenAutorizacionMemoria struct {
	mu                sync.RWMutex
	roles             map[string]domain.VersionRol
	controlesRoles    map[string]domain.ControlVigenciaVersionRol
	asignaciones      map[string]domain.AsignacionPerfil
	perfilesActuales  map[string]string
	politicas         map[string]domain.PoliticaRestrictiva
	politicasActuales map[string]string
	revisionPoliticas uint64
	huellaPoliticas   string
	decisiones        map[string]domain.DecisionAutorizacion
	denegaciones      map[string]domain.DecisionAutorizacion
}

func NuevoAlmacenAutorizacionMemoria() *AlmacenAutorizacionMemoria {
	huellaPoliticas, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		panic("calcular huella del catalogo de autorizacion vacio: " + err.Error())
	}
	return &AlmacenAutorizacionMemoria{
		roles:             make(map[string]domain.VersionRol),
		controlesRoles:    make(map[string]domain.ControlVigenciaVersionRol),
		asignaciones:      make(map[string]domain.AsignacionPerfil),
		perfilesActuales:  make(map[string]string),
		politicas:         make(map[string]domain.PoliticaRestrictiva),
		politicasActuales: make(map[string]string),
		revisionPoliticas: 1,
		huellaPoliticas:   huellaPoliticas,
		decisiones:        make(map[string]domain.DecisionAutorizacion),
		denegaciones:      make(map[string]domain.DecisionAutorizacion),
	}
}

// SembrarVersionRol carga una instantanea de configuracion. No modifica una
// version ya existente, ni siquiera si el contenido coincide.
func (a *AlmacenAutorizacionMemoria) SembrarVersionRol(version domain.VersionRol) error {
	if err := version.Validar(); err != nil {
		return err
	}
	referencia := version.Referencia()
	control := domain.ControlVigenciaVersionRol{
		VersionRolRef:  referencia,
		Revision:       1,
		Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
		ActualizadoPor: version.PublicadaPor,
		ActualizadoEn:  version.PublicadaEn,
	}
	if version.Estado == domain.EstadoVersionRolRetirada {
		control.Estado = domain.EstadoControlVigenciaVersionRolRetirada
		control.ActualizadoPor = version.RetiradaPor
		control.ActualizadoEn = version.RetiradaEn
		control.ActoRef = version.RetiradaRef
		control.MotivoCodigo = version.MotivoRetiradaCodigo
	}
	if err := control.Validar(); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, existe := a.roles[referencia]; existe {
		return ports.ErrVersionAutorizacionYaExiste
	}
	a.roles[referencia] = clonarVersionRolAutorizacion(version)
	a.controlesRoles[referencia] = control
	return nil
}

// SembrarControlVigenciaVersionRol aplica una retirada global explicita sobre
// una version exacta. Una version retirada distinta nunca afecta por inferencia
// a las asignaciones que referencian otra version.
func (a *AlmacenAutorizacionMemoria) SembrarControlVigenciaVersionRol(control domain.ControlVigenciaVersionRol) error {
	if err := control.Validar(); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, existe := a.roles[control.VersionRolRef]; !existe {
		return ports.ErrVersionRolNoEncontrada
	}
	actual, existe := a.controlesRoles[control.VersionRolRef]
	if !existe {
		return domain.ErrConfiguracionAccesoInvalida
	}
	if actual.Revision == math.MaxUint64 || control.Revision != actual.Revision+1 {
		return ports.ErrSecuenciaVersionInvalida
	}
	if actual.Estado == domain.EstadoControlVigenciaVersionRolRetirada ||
		control.Estado != domain.EstadoControlVigenciaVersionRolRetirada ||
		control.ActualizadoEn.Before(actual.ActualizadoEn) {
		return domain.ErrConfiguracionAccesoInvalida
	}
	a.controlesRoles[control.VersionRolRef] = control
	return nil
}

// SembrarAsignacionPerfil conserva todas las instantaneas y avanza el puntero
// vigente de cada perfil. Una revocacion se expresa mediante otra version.
func (a *AlmacenAutorizacionMemoria) SembrarAsignacionPerfil(asignacion domain.AsignacionPerfil) error {
	if err := asignacion.Validar(); err != nil {
		return err
	}
	perfilRef := asignacion.PerfilActivoRef
	a.mu.Lock()
	defer a.mu.Unlock()
	if referenciaActual, existe := a.perfilesActuales[perfilRef]; existe {
		actual := a.asignaciones[referenciaActual]
		if asignacion.Version <= actual.Version {
			return ports.ErrSecuenciaVersionInvalida
		}
		if asignacion.AsignacionID != actual.AsignacionID || asignacion.PrincipalID != actual.PrincipalID {
			return domain.ErrConfiguracionAccesoInvalida
		}
	}
	referencia := asignacion.Referencia()
	if _, existe := a.asignaciones[referencia]; existe {
		return ports.ErrVersionAutorizacionYaExiste
	}
	a.asignaciones[referencia] = clonarAsignacionAutorizacion(asignacion)
	a.perfilesActuales[perfilRef] = referencia
	return nil
}

// SembrarPoliticaRestrictiva retiene cada instantanea y solo avanza la version
// actual con una posterior. Una version retirada deja de aplicarse.
func (a *AlmacenAutorizacionMemoria) SembrarPoliticaRestrictiva(politica domain.PoliticaRestrictiva) error {
	if err := politica.Validar(); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if referenciaActual, existe := a.politicasActuales[politica.PoliticaID]; existe {
		actual := a.politicas[referenciaActual]
		if politica.Version <= actual.Version {
			return ports.ErrSecuenciaVersionInvalida
		}
	}
	if a.revisionPoliticas == math.MaxUint64 {
		return ports.ErrSecuenciaVersionInvalida
	}
	referencia := politica.Referencia()
	if _, existe := a.politicas[referencia]; existe {
		return ports.ErrVersionAutorizacionYaExiste
	}
	politicasProspectivas := make([]domain.PoliticaRestrictiva, 0, len(a.politicasActuales)+1)
	reemplazada := false
	for politicaID, referenciaActual := range a.politicasActuales {
		if politicaID == politica.PoliticaID {
			politicasProspectivas = append(politicasProspectivas, clonarPoliticaAutorizacion(politica))
			reemplazada = true
			continue
		}
		politicasProspectivas = append(politicasProspectivas, clonarPoliticaAutorizacion(a.politicas[referenciaActual]))
	}
	if !reemplazada {
		politicasProspectivas = append(politicasProspectivas, clonarPoliticaAutorizacion(politica))
	}
	huellaProspectiva, err := domain.HuellaCatalogoPoliticasAutorizacion(politicasProspectivas)
	if err != nil {
		return err
	}
	a.politicas[referencia] = clonarPoliticaAutorizacion(politica)
	a.politicasActuales[politica.PoliticaID] = referencia
	a.revisionPoliticas++
	a.huellaPoliticas = huellaProspectiva
	return nil
}

func (a *AlmacenAutorizacionMemoria) ObtenerInstantaneaAutorizacion(
	ctx context.Context,
	principalID, perfilActivoRef string,
) (domain.InstantaneaAutorizacion, error) {
	if err := ctx.Err(); err != nil {
		return domain.InstantaneaAutorizacion{}, err
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	referencia, existe := a.perfilesActuales[perfilActivoRef]
	if !existe {
		return domain.InstantaneaAutorizacion{}, ports.ErrAsignacionPerfilNoEncontrada
	}
	asignacion, existe := a.asignaciones[referencia]
	if !existe || asignacion.PrincipalID != principalID {
		// No se distingue entre perfil inexistente y perfil de otro sujeto.
		return domain.InstantaneaAutorizacion{}, ports.ErrAsignacionPerfilNoEncontrada
	}
	version, existe := a.roles[asignacion.VersionRolRef]
	if !existe {
		return domain.InstantaneaAutorizacion{}, ports.ErrVersionRolNoEncontrada
	}
	controlVigenciaRol, existe := a.controlesRoles[asignacion.VersionRolRef]
	if !existe {
		return domain.InstantaneaAutorizacion{}, domain.ErrConfiguracionAccesoInvalida
	}
	politicas := make([]domain.PoliticaRestrictiva, 0, len(a.politicasActuales))
	for _, referenciaPolitica := range a.politicasActuales {
		politica, encontrada := a.politicas[referenciaPolitica]
		if !encontrada {
			return domain.InstantaneaAutorizacion{}, domain.ErrConfiguracionAccesoInvalida
		}
		politicas = append(politicas, clonarPoliticaAutorizacion(politica))
	}
	sort.Slice(politicas, func(i, j int) bool {
		return politicas[i].Referencia() < politicas[j].Referencia()
	})
	instantanea := domain.InstantaneaAutorizacion{
		AsignacionPerfil:              clonarAsignacionAutorizacion(asignacion),
		VersionRol:                    clonarVersionRolAutorizacion(version),
		ControlVigenciaVersionRol:     controlVigenciaRol,
		Politicas:                     politicas,
		RevisionCatalogoPoliticas:     a.revisionPoliticas,
		CatalogoPoliticasHuellaSHA256: a.huellaPoliticas,
	}
	if err := instantanea.Validar(); err != nil {
		return domain.InstantaneaAutorizacion{}, err
	}
	return instantanea, nil
}

func (a *AlmacenAutorizacionMemoria) RegistrarDecisionSiInstantaneaVigente(
	ctx context.Context,
	decision domain.DecisionAutorizacion,
) error {
	if !decision.Concedida || decision.Codigo != "concedida" {
		return ports.ErrRegistroDecisionNoDisponible
	}
	return a.registrarDecisionSiInstantaneaVigente(ctx, decision, true)
}

func (a *AlmacenAutorizacionMemoria) RegistrarDenegacionAutorizacion(
	ctx context.Context,
	decision domain.DecisionAutorizacion,
) error {
	if decision.Concedida || decision.Codigo == "concedida" {
		return ports.ErrRegistroDenegacionNoDisponible
	}
	return a.registrarDecisionSiInstantaneaVigente(ctx, decision, false)
}

func (a *AlmacenAutorizacionMemoria) registrarDecisionSiInstantaneaVigente(
	ctx context.Context,
	decision domain.DecisionAutorizacion,
	esConcesion bool,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := decision.ValidarEvidenciaInstantanea(); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	referenciaActual, existe := a.perfilesActuales[decision.PerfilActivoRef]
	if !existe {
		return ports.ErrInstantaneaAutorizacionObsoleta
	}
	asignacionActual, existe := a.asignaciones[referenciaActual]
	if !existe || asignacionActual.PrincipalID != decision.PrincipalID ||
		referenciaActual != decision.AsignacionRef {
		return ports.ErrInstantaneaAutorizacionObsoleta
	}
	huellaAsignacion, err := asignacionActual.HuellaSHA256()
	if err != nil || huellaAsignacion != decision.AsignacionHuellaSHA256 {
		return ports.ErrInstantaneaAutorizacionObsoleta
	}
	versionRol, existe := a.roles[asignacionActual.VersionRolRef]
	if !existe || versionRol.Referencia() != decision.VersionRolRef {
		return ports.ErrInstantaneaAutorizacionObsoleta
	}
	huellaRol, err := versionRol.HuellaSHA256()
	if err != nil || huellaRol != decision.VersionRolHuellaSHA256 {
		return ports.ErrInstantaneaAutorizacionObsoleta
	}
	controlVigenciaRol, existe := a.controlesRoles[asignacionActual.VersionRolRef]
	if !existe || controlVigenciaRol.VersionRolRef != decision.ControlVigenciaVersionRolRef ||
		controlVigenciaRol.Revision != decision.ControlVigenciaVersionRolRevision {
		return ports.ErrInstantaneaAutorizacionObsoleta
	}
	huellaControlVigenciaRol, err := controlVigenciaRol.HuellaSHA256()
	if err != nil || huellaControlVigenciaRol != decision.ControlVigenciaVersionRolHuellaSHA256 {
		return ports.ErrInstantaneaAutorizacionObsoleta
	}
	if decision.RevisionCatalogoPoliticas != a.revisionPoliticas ||
		decision.CatalogoPoliticasHuellaSHA256 != a.huellaPoliticas {
		return ports.ErrInstantaneaAutorizacionObsoleta
	}
	if _, existe := a.decisiones[decision.DecisionRef]; existe {
		return ports.ErrVersionAutorizacionYaExiste
	}
	if _, existe := a.denegaciones[decision.DecisionRef]; existe {
		return ports.ErrVersionAutorizacionYaExiste
	}
	if esConcesion {
		a.decisiones[decision.DecisionRef] = clonarDecisionAutorizacion(decision)
	} else {
		a.denegaciones[decision.DecisionRef] = clonarDecisionAutorizacion(decision)
	}
	return nil
}

func (a *AlmacenAutorizacionMemoria) ObtenerDecision(ctx context.Context, referencia string) (domain.DecisionAutorizacion, error) {
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	decision, existe := a.decisiones[referencia]
	if !existe {
		return domain.DecisionAutorizacion{}, ports.ErrDecisionAutorizacionNoEncontrada
	}
	return clonarDecisionAutorizacion(decision), nil
}

// ObtenerDenegacion existe solo en este adaptador de pruebas para comprobar
// la separacion fisica entre trazas negativas y capacidades ejecutables.
func (a *AlmacenAutorizacionMemoria) ObtenerDenegacion(
	ctx context.Context,
	referencia string,
) (domain.DecisionAutorizacion, error) {
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	decision, existe := a.denegaciones[referencia]
	if !existe {
		return domain.DecisionAutorizacion{}, ports.ErrDecisionAutorizacionNoEncontrada
	}
	return clonarDecisionAutorizacion(decision), nil
}

func clonarVersionRolAutorizacion(version domain.VersionRol) domain.VersionRol {
	version.Concesiones = append([]domain.ConcesionRol(nil), version.Concesiones...)
	for indice := range version.Concesiones {
		version.Concesiones[indice].Finalidades = append([]string(nil), version.Concesiones[indice].Finalidades...)
		version.Concesiones[indice].CamposPermitidos = append([]string(nil), version.Concesiones[indice].CamposPermitidos...)
		version.Concesiones[indice].Obligaciones = append([]string(nil), version.Concesiones[indice].Obligaciones...)
	}
	return version
}

func clonarAsignacionAutorizacion(asignacion domain.AsignacionPerfil) domain.AsignacionPerfil {
	asignacion.Ambitos = append([]domain.AmbitoPerfil(nil), asignacion.Ambitos...)
	for indice := range asignacion.Ambitos {
		asignacion.Ambitos[indice].Valores = append([]string(nil), asignacion.Ambitos[indice].Valores...)
	}
	return asignacion
}

func clonarPoliticaAutorizacion(politica domain.PoliticaRestrictiva) domain.PoliticaRestrictiva {
	politica.Acciones = append([]string(nil), politica.Acciones...)
	politica.Modulos = append([]string(nil), politica.Modulos...)
	politica.TiposRecurso = append([]string(nil), politica.TiposRecurso...)
	politica.FinalidadesPermitidas = append([]string(nil), politica.FinalidadesPermitidas...)
	politica.Restricciones = append([]domain.RestriccionAtributoRecurso(nil), politica.Restricciones...)
	for indice := range politica.Restricciones {
		politica.Restricciones[indice].ValoresPermitidos = append([]string(nil), politica.Restricciones[indice].ValoresPermitidos...)
	}
	politica.CamposPermitidos = append([]string(nil), politica.CamposPermitidos...)
	politica.Obligaciones = append([]string(nil), politica.Obligaciones...)
	return politica
}

func clonarDecisionAutorizacion(decision domain.DecisionAutorizacion) domain.DecisionAutorizacion {
	decision.PoliticasEvaluadasRefs = append([]string(nil), decision.PoliticasEvaluadasRefs...)
	decision.PoliticasRefs = append([]string(nil), decision.PoliticasRefs...)
	decision.CamposPermitidos = append([]string(nil), decision.CamposPermitidos...)
	decision.Obligaciones = append([]string(nil), decision.Obligaciones...)
	if decision.PoliticasHuellasSHA256 != nil {
		origen := decision.PoliticasHuellasSHA256
		decision.PoliticasHuellasSHA256 = make(map[string]string, len(origen))
		for referencia, huella := range origen {
			decision.PoliticasHuellasSHA256[referencia] = huella
		}
	}
	if decision.PoliticasEvaluadasHuellasSHA256 != nil {
		origen := decision.PoliticasEvaluadasHuellasSHA256
		decision.PoliticasEvaluadasHuellasSHA256 = make(map[string]string, len(origen))
		for referencia, huella := range origen {
			decision.PoliticasEvaluadasHuellasSHA256[referencia] = huella
		}
	}
	return decision
}

var _ ports.FuenteAutorizacion = (*AlmacenAutorizacionMemoria)(nil)
var _ ports.RegistroDecisionesAutorizacion = (*AlmacenAutorizacionMemoria)(nil)
var _ ports.RegistroDenegacionesAutorizacion = (*AlmacenAutorizacionMemoria)(nil)
