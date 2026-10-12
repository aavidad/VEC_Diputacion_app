package mibolsa

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// LecturaPreferenciasBolsas evalúa la autorización actual de cada GET. No
// convierte la captura en una concesión reutilizable ni emite material V3.
type LecturaPreferenciasBolsas struct {
	lector  bolsa.LecturaPreferenciasBolsas
	auditor bolsa.AuditoriaIntentosPreferenciasBolsas
	fuente  puertosvec.FuenteAutorizacion
	reloj   puertosvec.Reloj
}

func NuevaLecturaPreferenciasBolsas(lector bolsa.LecturaPreferenciasBolsas, auditor bolsa.AuditoriaIntentosPreferenciasBolsas, fuente puertosvec.FuenteAutorizacion, reloj puertosvec.Reloj) (*LecturaPreferenciasBolsas, error) {
	if nula(lector) || nula(auditor) || nula(fuente) || nula(reloj) {
		return nil, bolsa.ErrPreferenciasNoDisponibles
	}
	return &LecturaPreferenciasBolsas{lector: lector, auditor: auditor, fuente: fuente, reloj: reloj}, nil
}

func (s *LecturaPreferenciasBolsas) ConsultarPreferencias(ctx context.Context, orden Orden) (bolsa.EstadoPreferenciasBolsas, error) {
	vacio := bolsa.EstadoPreferenciasBolsas{}
	if ctx == nil || s == nil || nula(s.lector) || nula(s.auditor) || nula(s.fuente) || nula(s.reloj) {
		return vacio, bolsa.ErrPreferenciasNoDisponibles
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	actor, candidato, err := validarOrden(orden, ahora)
	if err != nil {
		return vacio, err
	}
	datos, err := orden.Vinculo.Datos()
	if err != nil || datos.CuentaPrivilegiada || datos.Superficie != dominiovec.SuperficieAutenticacionExternaPersonalV1 ||
		datos.GarantiaObservada != dominiovec.AuthAssuranceHigh {
		return vacio, denegar(err)
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: "mi-bolsa:" + candidato, ModuloID: bolsa.ModuloMiBolsa,
		Tipo: bolsa.TipoRecursoMiBolsa, Ambitos: map[string]string{"candidato_ref": candidato},
		Atributos: map[string]string{"propiedad": "candidato"}}
	if recurso.Validar() != nil {
		return vacio, denegar(nil)
	}
	instantanea, err := s.fuente.ObtenerInstantaneaAutorizacion(ctx, datos.PrincipalID, datos.PerfilActivoRef)
	if err != nil || ctx.Err() != nil {
		return vacio, s.registrarFallo(ctx, orden, actor, candidato, datos, "error")
	}
	if instantanea.Validar() != nil {
		return vacio, s.registrarFallo(ctx, orden, actor, candidato, datos, "error")
	}
	if !concesionLecturaPreferenciasExacta(instantanea) {
		return vacio, s.registrarFallo(ctx, orden, actor, candidato, datos, "denegado")
	}
	ahora = s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	campos := bolsa.CamposLecturaPreferenciasBolsas()
	camposEvaluados := slices.Clone(campos)
	slices.Sort(camposEvaluados)
	evaluacion, err := dominiovec.EvaluarLecturaActualAutorizacionV3(orden.Vinculo, actor, instantanea,
		bolsa.AccionConsultarPreferenciasBolsas, recurso, bolsa.FinalidadPreferenciasBolsas, campos, ahora)
	if err != nil {
		return vacio, s.registrarFallo(ctx, orden, actor, candidato, datos, "error")
	}
	if !evaluacion.Permitido || !slices.Equal(evaluacion.CamposPermitidos, camposEvaluados) ||
		len(evaluacion.Obligaciones) > 1 || len(evaluacion.Obligaciones) == 1 && evaluacion.Obligaciones[0] != "auditar" {
		return vacio, s.registrarFallo(ctx, orden, actor, candidato, datos, "denegado")
	}
	if err := ctx.Err(); err != nil {
		return vacio, s.registrarFallo(ctx, orden, actor, candidato, datos, "error")
	}
	captura, err := capturarLecturaPreferencias(ctx, orden, actor, candidato, datos, evaluacion, s.reloj)
	if err != nil {
		return vacio, s.registrarFallo(ctx, orden, actor, candidato, datos, "error")
	}
	estado, err := s.lector.LeerPreferenciasBolsas(ctx, captura)
	if err != nil {
		resultado := "error"
		if errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
			resultado = "denegado"
		}
		return vacio, s.registrarFallo(ctx, orden, actor, candidato, datos, resultado)
	}
	return estado, nil
}

func concesionLecturaPreferenciasExacta(i dominiovec.InstantaneaAutorizacion) bool {
	if i.VersionRol.RolID != "candidato_bolsa_portal_historial_propio_desarrollo" ||
		i.AsignacionPerfil.EmitidaPor == "identidad:desarrollo:no-autoritativa" ||
		i.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		i.ControlVigenciaVersionRol.ActualizadoPor == "seguridad:desarrollo:no-autoritativa" {
		return false
	}
	n := 0
	for _, c := range i.VersionRol.Concesiones {
		if c.Accion != bolsa.AccionConsultarPreferenciasBolsas {
			continue
		}
		n++
		campos := slices.Clone(c.CamposPermitidos)
		slices.Sort(campos)
		esperados := bolsa.CamposLecturaPreferenciasBolsas()
		slices.Sort(esperados)
		if c.ModuloID != bolsa.ModuloMiBolsa || c.TipoRecurso != bolsa.TipoRecursoMiBolsa ||
			!slices.Equal(c.Finalidades, []string{bolsa.FinalidadPreferenciasBolsas}) ||
			!slices.Equal(campos, esperados) ||
			c.GarantiaMinima != dominiovec.AuthAssuranceHigh ||
			len(c.Obligaciones) > 1 || len(c.Obligaciones) == 1 && c.Obligaciones[0] != "auditar" {
			return false
		}
	}
	return n == 1
}

// La auditoría usa un plazo propio cuando la lectura agotó el contexto HTTP.
// SQL coteja otra vez la sesión vigente y el candidato; ningún dato sale en el fallo.
func (s *LecturaPreferenciasBolsas) registrarFallo(ctx context.Context, orden Orden,
	actor dominiovec.ResultadoContextoActorRegistradoV2, candidato string,
	datos dominiovec.DatosVinculoAutenticacionActorV2, resultado string) error {
	captura, err := capturarIntentoFallidoPreferencias(orden, actor, candidato, datos, resultado)
	if err != nil {
		return bolsa.ErrPreferenciasNoDisponibles
	}
	auditCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancelar()
	if err := s.auditor.RegistrarIntentoLecturaPreferenciasBolsas(auditCtx, captura); err != nil {
		return bolsa.ErrPreferenciasNoDisponibles
	}
	if resultado == "denegado" {
		return dominiovec.ErrAutorizacionDenegada
	}
	return bolsa.ErrPreferenciasNoDisponibles
}

func capturarIntentoFallidoPreferencias(orden Orden, actor dominiovec.ResultadoContextoActorRegistradoV2,
	candidato string, datos dominiovec.DatosVinculoAutenticacionActorV2, resultado string) (bolsa.CapturaLecturaPreferencias, error) {
	var aleatorio [16]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return bolsa.CapturaLecturaPreferencias{}, err
	}
	correlacion, err := orden.Correlacion.ValorCanonico()
	if err != nil {
		return bolsa.CapturaLecturaPreferencias{}, err
	}
	contexto := slices.Clone(actor.RepresentacionCanonica)
	vinculo, err := json.Marshal(orden.Vinculo)
	if err != nil || len(contexto) == 0 {
		return bolsa.CapturaLecturaPreferencias{}, bolsa.ErrPreferenciasNoDisponibles
	}
	hContexto := sha256.Sum256(contexto)
	hRecurso := sha256.Sum256([]byte("mi-bolsa:" + candidato))
	ordenJSON, err := json.Marshal(map[string]string{
		"intento_ref":           "intento_" + hex.EncodeToString(aleatorio[:]),
		"registro_contexto_ref": datos.RegistroContextoRef,
		"contexto_sha256":       hex.EncodeToString(hContexto[:]),
		"procedencia_sha256":    datos.ManifiestoProcedenciaHuellaSHA256,
		"autenticacion_ref":     datos.AutenticacionRef, "sesion_ref": datos.SesionRef,
		"autenticacion_sha256": datos.AutenticacionHuellaSHA256,
		"accion":               bolsa.AccionConsultarPreferenciasBolsas, "modulo_id": bolsa.ModuloMiBolsa,
		"recurso_ref":   "mi-bolsa:" + hex.EncodeToString(hRecurso[:]),
		"finalidad_ref": bolsa.FinalidadPreferenciasBolsas,
		"resultado":     resultado, "motivo_ref": "lectura_preferencias_bolsas_" + resultado,
		"proceso": "vec-server", "canal": string(dominiovec.SuperficieAutenticacionExternaPersonalV1),
		"correlacion_ref": correlacion,
	})
	if err != nil {
		return bolsa.CapturaLecturaPreferencias{}, err
	}
	return bolsa.NuevaCapturaLecturaPreferencias(candidato, correlacion, contexto, vinculo, ordenJSON, datos.SesionValidaHasta), nil
}

func capturarLecturaPreferencias(ctx context.Context, orden Orden,
	actor dominiovec.ResultadoContextoActorRegistradoV2, candidato string,
	datos dominiovec.DatosVinculoAutenticacionActorV2,
	e dominiovec.ResultadoLecturaActualAutorizacionV3, reloj puertosvec.Reloj,
) (bolsa.CapturaLecturaPreferencias, error) {
	vacia := bolsa.CapturaLecturaPreferencias{}
	var aleatorio [16]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return vacia, bolsa.ErrPreferenciasNoDisponibles
	}
	correlacion, err := orden.Correlacion.ValorCanonico()
	if err != nil {
		return vacia, denegar(err)
	}
	ahora := reloj.Ahora().UTC().Truncate(time.Microsecond)
	limite := ahora.Add(30 * time.Second)
	if e.ValidaHasta.Before(limite) {
		limite = e.ValidaHasta
	}
	if datos.SesionValidaHasta.Before(limite) {
		limite = datos.SesionValidaHasta
	}
	if ctx.Err() != nil || !limite.After(ahora) || !orden.Vinculo.VigenteEn(ahora, actor) {
		return vacia, denegar(ctx.Err())
	}
	contexto := slices.Clone(actor.RepresentacionCanonica)
	vinculo, err := json.Marshal(orden.Vinculo)
	if err != nil || len(contexto) == 0 {
		return vacia, denegar(err)
	}
	huella := sha256.Sum256(contexto)
	ordenJSON, err := json.Marshal(struct {
		IntentoRef                    string                                               `json:"intento_ref"`
		RegistroContextoRef           string                                               `json:"registro_contexto_ref"`
		ContextoSHA256                string                                               `json:"contexto_sha256"`
		ProcedenciaSHA256             string                                               `json:"procedencia_sha256"`
		AutenticacionRef              string                                               `json:"autenticacion_ref"`
		SesionRef                     string                                               `json:"sesion_ref"`
		AutenticacionSHA256           string                                               `json:"autenticacion_sha256"`
		Accion                        string                                               `json:"accion"`
		ModuloID                      string                                               `json:"modulo_id"`
		RecursoRef                    string                                               `json:"recurso_ref"`
		CandidatoRef                  string                                               `json:"candidato_ref"`
		TipoRecurso                   string                                               `json:"tipo_recurso"`
		FinalidadRef                  string                                               `json:"finalidad_ref"`
		CamposPermitidos              []string                                             `json:"campos_permitidos"`
		Resultado                     string                                               `json:"resultado"`
		MotivoRef                     string                                               `json:"motivo_ref"`
		Proceso                       string                                               `json:"proceso"`
		Canal                         string                                               `json:"canal"`
		CorrelacionRef                string                                               `json:"correlacion_ref"`
		LecturaRevisionPermisos       string                                               `json:"lectura_revision_permisos"`
		LecturaInstantaneaSHA256      string                                               `json:"lectura_instantanea_sha256"`
		AsignacionRef                 string                                               `json:"asignacion_ref"`
		AsignacionVersion             int                                                  `json:"asignacion_version"`
		AsignacionHuellaSHA256        string                                               `json:"asignacion_huella_sha256"`
		VersionRolRef                 string                                               `json:"version_rol_ref"`
		VersionRolVersion             int                                                  `json:"version_rol_version"`
		VersionRolHuellaSHA256        string                                               `json:"version_rol_huella_sha256"`
		ControlVersionRolRef          string                                               `json:"control_version_rol_ref"`
		RevisionControlVersionRol     uint64                                               `json:"revision_control_version_rol"`
		ControlVersionRolHuellaSHA256 string                                               `json:"control_version_rol_huella_sha256"`
		RevisionCatalogoPoliticas     uint64                                               `json:"revision_catalogo_politicas"`
		CatalogoPoliticasHuellaSHA256 string                                               `json:"catalogo_politicas_huella_sha256"`
		PoliticasEvaluadas            []dominiovec.ReferenciaHuellaPoliticaLecturaActualV3 `json:"politicas_evaluadas"`
		EmitidaEn                     time.Time                                            `json:"emitida_en"`
		ValidaHasta                   time.Time                                            `json:"valida_hasta"`
	}{
		IntentoRef:          "lectura_" + hex.EncodeToString(aleatorio[:]),
		RegistroContextoRef: datos.RegistroContextoRef, ContextoSHA256: hex.EncodeToString(huella[:]),
		ProcedenciaSHA256: datos.ManifiestoProcedenciaHuellaSHA256,
		AutenticacionRef:  datos.AutenticacionRef, SesionRef: datos.SesionRef,
		AutenticacionSHA256: datos.AutenticacionHuellaSHA256,
		Accion:              bolsa.AccionConsultarPreferenciasBolsas, ModuloID: bolsa.ModuloMiBolsa,
		RecursoRef: "mi-bolsa:" + candidato, CandidatoRef: candidato, TipoRecurso: bolsa.TipoRecursoMiBolsa,
		FinalidadRef: bolsa.FinalidadPreferenciasBolsas, CamposPermitidos: bolsa.CamposLecturaPreferenciasBolsas(),
		Resultado: "obtenida", MotivoRef: "lectura_preferencias_bolsas_correcta", Proceso: "vec-server",
		Canal: string(dominiovec.SuperficieAutenticacionExternaPersonalV1), CorrelacionRef: correlacion,
		LecturaRevisionPermisos: strconv.FormatUint(e.RevisionPermisos, 10), LecturaInstantaneaSHA256: e.HuellaInstantaneaSHA256,
		AsignacionRef: e.AsignacionRef, AsignacionVersion: e.AsignacionVersion, AsignacionHuellaSHA256: e.AsignacionHuellaSHA256,
		VersionRolRef: e.VersionRolRef, VersionRolVersion: e.VersionRolVersion, VersionRolHuellaSHA256: e.VersionRolHuellaSHA256,
		ControlVersionRolRef: e.ControlVersionRolRef, RevisionControlVersionRol: e.RevisionControlVersionRol,
		ControlVersionRolHuellaSHA256: e.ControlVersionRolHuellaSHA256, RevisionCatalogoPoliticas: e.RevisionCatalogoPoliticas,
		CatalogoPoliticasHuellaSHA256: e.CatalogoPoliticasHuellaSHA256,
		PoliticasEvaluadas:            append([]dominiovec.ReferenciaHuellaPoliticaLecturaActualV3{}, e.PoliticasEvaluadas...),
		EmitidaEn:                     e.EvaluadaEn, ValidaHasta: limite,
	})
	if err != nil {
		return vacia, bolsa.ErrPreferenciasNoDisponibles
	}
	return bolsa.NuevaCapturaLecturaPreferencias(candidato, correlacion, contexto, vinculo, ordenJSON, limite), nil
}
