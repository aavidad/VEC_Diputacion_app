package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"reflect"
	"regexp"
	"sync"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominioct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible = errors.New(
	"bolsa: politica de borrador de llamamiento de desarrollo no disponible",
)

var aprobacionDocumentalBolsaValida = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{7,127}$`)
var huellaDocumentalBolsaValida = regexp.MustCompile(`^[a-f0-9]{64}$`)

func huellaDocumentalBolsaVisible(valor string) string {
	if huellaDocumentalBolsaValida.MatchString(valor) {
		return valor
	}
	return "formato_invalido"
}

// autoridadInicialBorradorLlamamientoBolsaDesarrollo conserva la preparación
// inicial separada. Sólo la composición puede aportar la autoridad PostgreSQL
// común; la política no abre conexiones ni publica por su cuenta al construirse.
type autoridadInicialBorradorLlamamientoBolsaDesarrollo interface {
	prepararInstantanea(context.Context, dominiovec.InstantaneaAutorizacion, bool) (dominiovec.InstantaneaAutorizacion, error)
	publicarInstantaneaDesdePreimagen(context.Context, dominiovec.InstantaneaAutorizacion, dominiovec.InstantaneaAutorizacion) error
}

// politicaBorradorLlamamientoBolsaDesarrollo es la fuente nominal exclusiva de
// B-BACK. No contiene concesiones de Contratación ni interpreta datos del
// transporte: principal, perfil y ámbitos proceden del contexto Bolsa creado
// antes de atender peticiones.
type politicaBorradorLlamamientoBolsaDesarrollo struct {
	soporte     *soporteSesionBorradorBolsaDesarrollo
	autoridad   autoridadInicialBorradorLlamamientoBolsaDesarrollo
	registro    registroDecisionesAnalisisContratacionTemporalDesarrollo
	reloj       relojContratacionTemporalDesarrollo
	mu          sync.RWMutex
	publicada   bool
	instantanea dominiovec.InstantaneaAutorizacion
}

func nuevaPoliticaBorradorLlamamientoBolsaDesarrollo(
	soporte *soporteSesionBorradorBolsaDesarrollo,
	autoridad autoridadInicialBorradorLlamamientoBolsaDesarrollo,
	registro registroDecisionesAnalisisContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (*politicaBorradorLlamamientoBolsaDesarrollo, error) {
	if soporte == nil || soporte.soporteCanal == nil || dependenciaAutorizacionComunDesarrolloNula(autoridad) || dependenciaAutorizacionComunDesarrolloNula(registro) ||
		soporte.unidadRef == "" || soporte.ambitoRef == "" ||
		soporte.soporteCanal.contexto.Resultado.Validar() != nil ||
		soporte.soporteCanal.contexto.Vinculo.ValidarPara(soporte.soporteCanal.contexto.Resultado) != nil {
		return nil, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	return &politicaBorradorLlamamientoBolsaDesarrollo{
		soporte: soporte, autoridad: autoridad, registro: registro, reloj: reloj,
	}, nil
}

// PublicarInicial prepara con permitirInicial=true y sólo conserva la copia
// después de que la autoridad durable la acepte. Debe invocarse una vez durante
// la composición, antes de exponer rutas.
func (p *politicaBorradorLlamamientoBolsaDesarrollo) PublicarInicial(ctx context.Context) error {
	return p.publicarInicial(ctx, false, false)
}

// PublicarInicialConPoliticaOfertas amplía conservadoramente la publicación
// sintética B5 a la versión 6. La versión 5 se mantiene intacta cuando B47 no
// se ha solicitado: activar la ruta no debe conceder la edición por omisión.
func (p *politicaBorradorLlamamientoBolsaDesarrollo) PublicarInicialConPoliticaOfertas(ctx context.Context) error {
	return p.publicarInicial(ctx, true, false)
}

// PublicarInicialConReincorporacion conserva la rama de rol publicada: v5/v6
// son las preimágenes exactas de v7/v8. La consulta B55 no concede cambios.
func (p *politicaBorradorLlamamientoBolsaDesarrollo) PublicarInicialConReincorporacion(ctx context.Context, incluirPoliticaOfertas bool) error {
	return p.publicarInicial(ctx, incluirPoliticaOfertas, true)
}

func (p *politicaBorradorLlamamientoBolsaDesarrollo) publicarInicial(ctx context.Context, incluirPoliticaOfertas, incluirReincorporacion bool) error {
	if p == nil || ctx == nil || ctx.Err() != nil || p.soporte == nil || p.soporte.soporteCanal == nil ||
		dependenciaAutorizacionComunDesarrolloNula(p.autoridad) {
		return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.publicada {
		return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	// B1 todavía no tiene una plantilla de rol acordada en la autoridad V3.
	// Estos ajustes históricos no pueden convertir el arranque en publicador.
	if os.Getenv(envCargaConvocaAprobacion) != "" || os.Getenv(envCargaConvocaPreimagen) != "" || os.Getenv(envCargaConvocaObjetivo) != "" {
		return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	datos, err := p.soporte.soporteCanal.contexto.Vinculo.Datos()
	if err != nil {
		return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	ahora := p.reloj.Ahora()
	versionRol := 5
	if incluirPoliticaOfertas {
		versionRol = 6
	}
	versionPreimagen := versionRol - 1
	if incluirReincorporacion {
		versionPreimagen = versionRol
		versionRol += 2 // v5→v7 y v6→v8, sin colisión entre ramas.
	}
	// Las concesiones documental y de consulta completa de datos de contacto
	// solo se consumen si una provisión aprobada fuera de la petición las
	// publicó por CAS. El arranque sin aprobación no las publica.
	if lector, ok := p.autoridad.(lectorAsignacionPublicadaCTDesarrollo); ok {
		publicada, encontrada, err := lector.leerAsignacionPublicada(ctx, datos.PerfilActivoRef)
		if err != nil {
			return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
		}
		version := publicada.instantanea.VersionRol.Version
		// Una nueva versión gobernada del mismo rol puede añadir B1 sin que el
		// arranque publique ni reconstruya la asignación. Se consume la copia
		// central y se mantiene íntegra su identidad y su historia.
		if encontrada && instantaneaBolsaCargaConvocaCompatible(publicada.instantanea, datos,
			p.soporte, ahora, versionRol) && procedenciaCargaConvocaGobernada(publicada, p.autoridad) {
			p.instantanea = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(publicada.instantanea)
			p.publicada = true
			return nil
		}
		objetivoVersion := versionRol + saltoProvisionDatosContactoBolsa
		const envPreimagen = "VEC_BOLSA_DOCUMENTAL_PROVISION_PREIMAGEN_SHA256"
		const envObjetivo = "VEC_BOLSA_DOCUMENTAL_PROVISION_OBJETIVO_SHA256"
		const envAprobacion = "VEC_BOLSA_DOCUMENTAL_PROVISION_APROBACION_REF"
		aprobacion := os.Getenv(envAprobacion)
		preimagenIndicada, objetivoIndicado := os.Getenv(envPreimagen), os.Getenv(envObjetivo)
		sinAprobacion := aprobacion == "" && preimagenIndicada == "" && objetivoIndicado == ""
		documental := encontrada && version == versionRol+saltoProvisionDocumentalBolsa
		// Con la documental ya aplicada, una aprobación cuya preimagen no es la
		// asignación publicada es la de aquella provisión: no concede nada nuevo
		// y no debe impedir el arranque. Se ignora con aviso.
		if documental && !sinAprobacion {
			if preimagenPublicada, err := publicada.instantanea.AsignacionPerfil.HuellaSHA256(); err != nil || preimagenIndicada != preimagenPublicada {
				slog.Warn("aprobación de provisión RRHH Bolsa anterior ignorada", "perfil_ref", datos.PerfilActivoRef,
					"recibido_preimagen", huellaDocumentalBolsaVisible(preimagenIndicada))
				sinAprobacion = true
			}
		}
		if encontrada && version >= 9 && !documental && version != objetivoVersion {
			return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
		}
		// Ya provisionada: la versión completa, o la documental anterior
		// mientras no haya otra aprobación para ampliarla.
		if encontrada && (version == objetivoVersion || documental && sinAprobacion) {
			esperada, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
				datos.PrincipalID, datos.PerfilActivoRef, p.soporte.unidadRef, p.soporte.ambitoRef, ahora, version)
			if err != nil {
				return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
			}
			esperada = instantaneaMiBolsaEsperada(esperada, publicada.instantanea)
			if publicada.instantanea.Validar() != nil || !reflect.DeepEqual(publicada.instantanea, esperada) ||
				!publicada.instantanea.AsignacionPerfil.VigenteEn(ahora) ||
				publicada.instantanea.ControlVigenciaVersionRol.Estado != dominiovec.EstadoControlVigenciaVersionRolHabilitada {
				return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
			}
			if propia, ok := p.autoridad.(*autoridadPostgreSQLDesarrollo); ok &&
				(publicada.actoAsignacion != propia.actoAsignacion || publicada.actoControl != propia.actoControlRol ||
					publicada.actualizadaPor != esperada.AsignacionPerfil.EmitidaPor) {
				return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
			}
			if documental {
				if preimagenSHA, objetivoSHA, err := huellasProvisionRRHHBolsa(publicada.instantanea, datos, p.soporte, ahora, objetivoVersion); err == nil {
					slog.Warn("consulta de datos de contacto Bolsa pendiente de provisión", "perfil_ref", datos.PerfilActivoRef,
						"estado", "pendiente_provision", "preimagen_sha256", preimagenSHA, "objetivo_sha256", objetivoSHA)
				}
			}
			p.instantanea = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(publicada.instantanea)
			p.publicada = true
			return nil
		}
		if encontrada && (version == versionRol || documental) {
			preimagenSHA, objetivoSHA, err := huellasProvisionRRHHBolsa(publicada.instantanea, datos, p.soporte, ahora, objetivoVersion)
			if err != nil {
				return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
			}
			if sinAprobacion {
				slog.Warn("solicitudes documentales y consulta de datos de contacto Bolsa pendientes de provisión", "perfil_ref", datos.PerfilActivoRef,
					"estado", "pendiente_provision", "preimagen_sha256", preimagenSHA, "objetivo_sha256", objetivoSHA)
			} else {
				if !aprobacionDocumentalBolsaValida.MatchString(aprobacion) {
					slog.Error("PARO provisión RRHH Bolsa: aprobación inválida", "clave", envAprobacion,
						"esperado", "referencia_opaca_8_a_128", "recibido_longitud", len(aprobacion))
					return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
				}
				if preimagenIndicada != preimagenSHA || objetivoIndicado != objetivoSHA {
					slog.Error("PARO provisión RRHH Bolsa: huella distinta", "clave_preimagen", envPreimagen,
						"esperado_preimagen", preimagenSHA, "recibido_preimagen", huellaDocumentalBolsaVisible(preimagenIndicada),
						"clave_objetivo", envObjetivo, "esperado_objetivo", objetivoSHA, "recibido_objetivo", huellaDocumentalBolsaVisible(objetivoIndicado))
					return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
				}
				objetivo, err := objetivoProvisionRRHHBolsa(publicada.instantanea, datos, p.soporte, ahora, objetivoVersion)
				if err != nil {
					return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
				}
				preparada, errPreparar := p.autoridad.prepararInstantanea(ctx, objetivo, false)
				if errPreparar != nil || preparada.VersionRol.Version != objetivoVersion || !reflect.DeepEqual(preparada, objetivo) {
					slog.Error("PARO provisión RRHH Bolsa: rol preparado distinto", "clave", "version_rol",
						"esperado", objetivoVersion, "recibido", preparada.VersionRol.Version)
					return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
				}
				if err := p.autoridad.publicarInstantaneaDesdePreimagen(ctx, preparada, publicada.instantanea); err != nil {
					return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
				}
				p.instantanea = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(preparada)
				p.publicada = true
				slog.Info("provisión RRHH Bolsa aplicada", "aprobacion_ref", aprobacion, "perfil_ref", datos.PerfilActivoRef,
					"preimagen_sha256", preimagenSHA, "objetivo_sha256", objetivoSHA, "version", preparada.AsignacionPerfil.Version)
				return nil
			}
		}
	}
	semilla, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
		datos.PrincipalID, datos.PerfilActivoRef, p.soporte.unidadRef, p.soporte.ambitoRef, ahora, versionRol,
	)
	if err != nil {
		return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	preimagen, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
		datos.PrincipalID, datos.PerfilActivoRef, p.soporte.unidadRef, p.soporte.ambitoRef, ahora, versionPreimagen,
	)
	if err != nil {
		return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	preparada, err := p.autoridad.prepararInstantanea(ctx, semilla, true)
	if preparada.AsignacionPerfil.Version > 1 {
		preimagen.AsignacionPerfil.Version = preparada.AsignacionPerfil.Version - 1
	}
	esperada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
	esperada.AsignacionPerfil.Version = preparada.AsignacionPerfil.Version
	versionAdmitida := preparada.AsignacionPerfil.Version >= 1 && preparada.AsignacionPerfil.Version <= 8 && preparada.VersionRol.Version == versionRol
	if err != nil || preparada.Validar() != nil || !versionAdmitida || !reflect.DeepEqual(preparada, esperada) ||
		p.autoridad.publicarInstantaneaDesdePreimagen(ctx, preparada, preimagen) != nil {
		return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	p.instantanea = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(preparada)
	p.publicada = true
	return nil
}

func nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrollo(
	principalID, perfilRef, unidadRef, ambitoRef string, ahora time.Time,
) (dominiovec.InstantaneaAutorizacion, error) {
	return nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(principalID, perfilRef, unidadRef, ambitoRef, ahora, 5)
}

const (
	envCargaConvocaAprobacion = "VEC_BOLSA_CARGA_CONVOCA_PROVISION_APROBACION_REF"
	envCargaConvocaPreimagen  = "VEC_BOLSA_CARGA_CONVOCA_PROVISION_PREIMAGEN_SHA256"
	envCargaConvocaObjetivo   = "VEC_BOLSA_CARGA_CONVOCA_PROVISION_OBJETIVO_SHA256"
)

func nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
	principalID, perfilRef, unidadRef, ambitoRef string, ahora time.Time, versionRol int,
) (dominiovec.InstantaneaAutorizacion, error) {
	desde, hasta, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(ahora)
	if !vigente || principalID == "" || perfilRef == "" || unidadRef == "" || ambitoRef == "" || versionRol < 1 || versionRol > 16 {
		return dominiovec.InstantaneaAutorizacion{}, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	versionBase := versionRol
	concesion := func(accion, finalidad string) dominiovec.ConcesionRol {
		tipoRecurso := puertosbolsa.TipoRecursoBorradorLlamamiento
		if accion == puertosbolsa.AccionCambiarSituacionParticipacion || accion == puertosbolsa.AccionConsultarSolicitudesDocumentalesRRHH || accion == puertosbolsa.AccionRegistrarContactoParticipacion || accion == puertosbolsa.AccionConsultarContactoParticipacion || accion == puertosbolsa.AccionRegistrarDatosContactoParticipacion || accion == puertosbolsa.AccionConsultarDatosContactoParticipacion || accion == puertosbolsa.AccionEmitirLlamamiento {
			tipoRecurso = puertosbolsa.TipoRecursoSituacionParticipacion
			if accion == puertosbolsa.AccionEmitirLlamamiento {
				tipoRecurso = puertosbolsa.TipoRecursoEmision
			}
		}
		return dominiovec.ConcesionRol{
			Accion: accion, ModuloID: puertosbolsa.ModuloBorradorLlamamiento,
			TipoRecurso: tipoRecurso,
			Finalidades: []string{finalidad}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
		}
	}
	concesiones := []dominiovec.ConcesionRol{
		concesion(puertosbolsa.AccionCrearBorradorLlamamientoInterno, puertosbolsa.FinalidadCrearBorradorLlamamientoInterno),
		concesion(puertosbolsa.AccionConsultarBorradorLlamamientoInterno, puertosbolsa.FinalidadConsultarBorradorLlamamientoInterno),
	}
	if versionBase >= 2 {
		concesiones = append(concesiones, concesion(puertosbolsa.AccionCambiarSituacionParticipacion, puertosbolsa.FinalidadCambiarSituacionParticipacion))
	}
	if versionBase >= 3 {
		concesiones = append(concesiones, concesion(puertosbolsa.AccionRegistrarContactoParticipacion, puertosbolsa.FinalidadRegistrarContactoParticipacion))
	}
	if versionBase >= 4 {
		concesiones = append(concesiones, concesion(puertosbolsa.AccionConsultarContactoParticipacion, puertosbolsa.FinalidadConsultarContactoParticipacion))
		concesiones = append(concesiones, concesion(puertosbolsa.AccionEmitirLlamamiento, puertosbolsa.FinalidadEmitirLlamamiento))
	}
	if versionBase >= 5 {
		concesiones = append(concesiones, concesion(puertosbolsa.AccionRegistrarDatosContactoParticipacion, puertosbolsa.FinalidadRegistrarDatosContactoParticipacion))
	}
	if versionBase >= 9 {
		concesiones = append(concesiones, concesion(puertosbolsa.AccionConsultarSolicitudesDocumentalesRRHH, puertosbolsa.FinalidadCambiarSituacionParticipacion))
	}
	if versionBase >= 9+saltoProvisionDocumentalBolsa {
		// Correo y teléfonos completos: acción y finalidad propias (AD197/B78).
		concesiones = append(concesiones, concesion(puertosbolsa.AccionConsultarDatosContactoParticipacion, puertosbolsa.FinalidadConsultarDatosContactoParticipacion))
	}
	if versionRolBolsaConPoliticaOfertas(versionBase) {
		concesiones = append(concesiones, dominiovec.ConcesionRol{
			Accion: puertosbolsa.AccionPublicarPoliticaOfertas, ModuloID: "bolsa",
			TipoRecurso: "bolsa_constituida", Finalidades: []string{puertosbolsa.FinalidadPoliticaOfertas},
			GarantiaMinima: dominiovec.AuthAssuranceHigh,
		})
		concesiones = append(concesiones, dominiovec.ConcesionRol{
			Accion: puertosbolsa.AccionConsultarPoliticaOfertas, ModuloID: "bolsa",
			TipoRecurso: "bolsa_constituida", Finalidades: []string{puertosbolsa.FinalidadConsultarPoliticaOfertas},
			GarantiaMinima: dominiovec.AuthAssuranceHigh, CamposPermitidos: []string{puertosbolsa.CampoConsultarPoliticaOfertas},
		})
	}
	if versionRolBolsaConReincorporacion(versionBase) {
		concesiones = append(concesiones, dominiovec.ConcesionRol{
			Accion: puertosbolsa.AccionConsultarReincorporacionTitular, ModuloID: puertosbolsa.ModuloSituacionParticipacion,
			TipoRecurso:      puertosbolsa.TipoRecursoSituacionParticipacion,
			Finalidades:      []string{puertosbolsa.FinalidadConsultarReincorporacionTitular},
			GarantiaMinima:   dominiovec.AuthAssuranceHigh,
			CamposPermitidos: []string{puertosbolsa.CampoConsultarReincorporacionTitular},
		})
	}
	version := dominiovec.VersionRol{
		RolID: "tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo", Version: versionRol,
		Nombre:       "Tecnico RRHH de borradores de llamamiento de desarrollo",
		Estado:       dominiovec.EstadoVersionRolPublicada,
		Concesiones:  concesiones,
		PublicadaPor: "seguridad:desarrollo:no-autoritativa", PublicadaEn: desde,
	}
	asignacion := dominiovec.AsignacionPerfil{
		AsignacionID: referenciaAltaContratacionTemporalDesarrollo("asg_", principalID+"\x00"+perfilRef+"\x00bolsa-bback-v1"),
		Version:      1, PerfilActivoRef: perfilRef, PrincipalID: principalID, VersionRolRef: version.Referencia(),
		Estado:       dominiovec.EstadoAsignacionPerfilActiva,
		Ambitos:      []dominiovec.AmbitoPerfil{{Clave: "unidad_ref", Valores: []string{unidadRef}}, {Clave: "ambito_ref", Valores: []string{ambitoRef}}},
		VigenteDesde: desde, VigenteHasta: hasta,
		EmitidaPor: "identidad:desarrollo:no-autoritativa", EmitidaEn: desde,
	}
	huella, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		return dominiovec.InstantaneaAutorizacion{}, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	instantanea := dominiovec.InstantaneaAutorizacion{
		AsignacionPerfil: asignacion, VersionRol: version,
		ControlVigenciaVersionRol: dominiovec.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1, Estado: dominiovec.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: desde,
		},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella,
	}
	if instantanea.Validar() != nil {
		return dominiovec.InstantaneaAutorizacion{}, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	return instantanea, nil
}

func (p *politicaBorradorLlamamientoBolsaDesarrollo) ObtenerInstantaneaAutorizacion(
	ctx context.Context, principalID, perfilRef string,
) (dominiovec.InstantaneaAutorizacion, error) {
	if p == nil || ctx == nil || ctx.Err() != nil {
		return dominiovec.InstantaneaAutorizacion{}, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if !p.publicada || p.instantanea.Validar() != nil ||
		principalID != p.instantanea.AsignacionPerfil.PrincipalID || perfilRef != p.instantanea.AsignacionPerfil.PerfilActivoRef {
		return dominiovec.InstantaneaAutorizacion{}, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	return clonarInstantaneaAutorizacionPostgreSQLDesarrollo(p.instantanea), nil
}

// La composición sólo puede montar B1 si la asignación publicada y contrastada
// contiene exactamente su concesión; la semilla histórica no la añade.
func (p *politicaBorradorLlamamientoBolsaDesarrollo) permiteCargaConvoca() bool {
	if p == nil {
		return false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if !p.publicada || p.instantanea.Validar() != nil ||
		!p.instantanea.AsignacionPerfil.VigenteEn(p.reloj.Ahora()) ||
		p.instantanea.VersionRol.Estado != dominiovec.EstadoVersionRolPublicada ||
		p.instantanea.ControlVigenciaVersionRol.Estado != dominiovec.EstadoControlVigenciaVersionRolHabilitada {
		return false
	}
	return concesionCargaConvocaExacta(p.instantanea.VersionRol.Concesiones)
}

func (p *politicaBorradorLlamamientoBolsaDesarrollo) ValidarReferenciaMotivoAutorizacionV2(
	ctx context.Context, referencia dominiovec.ReferenciaEntradaCatalogo, instante time.Time,
) error {
	if p == nil || ctx == nil || ctx.Err() != nil || !dominioct.InstanteUTCCanonico(instante) {
		return dominiovec.ErrSolicitudAutorizacionInvalida
	}
	p.mu.RLock()
	publicada, instantanea := p.publicada, p.instantanea
	p.mu.RUnlock()
	if !publicada || instantanea.Validar() != nil || !instantanea.AsignacionPerfil.VigenteEn(instante) {
		return dominiovec.ErrSolicitudAutorizacionInvalida
	}
	if referencia == motivoPublicarPoliticaOfertasBolsaDesarrollo() || referencia == motivoConsultarPoliticaOfertasBolsaDesarrollo() {
		if !versionRolBolsaConPoliticaOfertas(instantanea.VersionRol.Version) {
			return dominiovec.ErrSolicitudAutorizacionInvalida
		}
		return nil
	}
	if referencia == motivoConsultarReincorporacionTitularBolsaDesarrollo() {
		if !versionRolBolsaConReincorporacion(instantanea.VersionRol.Version) {
			return dominiovec.ErrSolicitudAutorizacionInvalida
		}
		return nil
	}
	// El motivo de la consulta completa vale siempre: sin la concesión, el PDP
	// la deniega y la denegación queda registrada (403, no indisponibilidad).
	if referencia == motivoConsultarDatosContactoParticipacionBolsaDesarrollo() {
		return nil
	}
	if referencia == motivoConfirmarCargaConvocaBolsaDesarrollo() {
		return nil // La concesión positiva se evalúa en el PDP sobre la versión vigente.
	}
	if referencia != motivoCrearBorradorLlamamientoBolsaDesarrollo() && referencia != motivoConsultarBorradorLlamamientoBolsaDesarrollo() && referencia != motivoCambiarSituacionParticipacionBolsaDesarrollo() && referencia != motivoRegistrarContactoParticipacionBolsaDesarrollo() && referencia != motivoConsultarContactoParticipacionBolsaDesarrollo() && referencia != motivoRegistrarDatosContactoParticipacionBolsaDesarrollo() && referencia != motivoEmitirLlamamientoBolsaDesarrollo() {
		return dominiovec.ErrSolicitudAutorizacionInvalida
	}
	return nil
}

func (p *politicaBorradorLlamamientoBolsaDesarrollo) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	ctx context.Context, orden puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	if !p.ordenRegistroBorradorLlamamientoValida(ctx, orden, true) ||
		dependenciaAutorizacionComunDesarrolloNula(p.registro) {
		return time.Time{}, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible
	}
	return p.registro.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden)
}

func (p *politicaBorradorLlamamientoBolsaDesarrollo) RegistrarDenegacionAutorizacionLigadaV3(
	ctx context.Context, orden puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	if !p.ordenRegistroBorradorLlamamientoValida(ctx, orden, false) ||
		dependenciaAutorizacionComunDesarrolloNula(p.registro) {
		return puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	return p.registro.RegistrarDenegacionAutorizacionLigadaV3(ctx, orden)
}

func (p *politicaBorradorLlamamientoBolsaDesarrollo) ordenRegistroBorradorLlamamientoValida(
	ctx context.Context,
	orden interface {
		Datos() (puertosvec.DatosOrdenRegistroAutorizacionLigadaV3, error)
	},
	concesion bool,
) bool {
	if p == nil || ctx == nil || ctx.Err() != nil || orden == nil {
		return false
	}
	datos, err := orden.Datos()
	if err != nil || datos.ResultadoContexto.Validar() != nil {
		return false
	}
	p.mu.RLock()
	publicada, instantanea := p.publicada, p.instantanea
	p.mu.RUnlock()
	if !publicada || instantanea.Validar() != nil ||
		datos.ResultadoContexto.Contexto.Principal.ID != instantanea.AsignacionPerfil.PrincipalID ||
		datos.ResultadoContexto.Contexto.PerfilActivoRef != instantanea.AsignacionPerfil.PerfilActivoRef {
		return false
	}
	solicitud, err := datos.Solicitud.Datos()
	if err != nil || !motivoBorradorLlamamientoCorresponde(solicitud.Accion, datos.ReferenciaMotivo) ||
		solicitud.ReferenciaMotivo != datos.ReferenciaMotivo {
		return false
	}
	return !concesion || datos.Decision.ValidarPara(datos.Solicitud) == nil
}

func motivoBorradorLlamamientoCorresponde(accion string, motivo dominiovec.ReferenciaEntradaCatalogo) bool {
	switch accion {
	case puertosbolsa.AccionCrearBorradorLlamamientoInterno:
		return motivo == motivoCrearBorradorLlamamientoBolsaDesarrollo()
	case puertosbolsa.AccionConsultarBorradorLlamamientoInterno:
		return motivo == motivoConsultarBorradorLlamamientoBolsaDesarrollo()
	case puertosbolsa.AccionCambiarSituacionParticipacion:
		return motivo == motivoCambiarSituacionParticipacionBolsaDesarrollo()
	case puertosbolsa.AccionConsultarSolicitudesDocumentalesRRHH:
		return motivo == motivoCambiarSituacionParticipacionBolsaDesarrollo()
	case puertosbolsa.AccionRegistrarContactoParticipacion:
		return motivo == motivoRegistrarContactoParticipacionBolsaDesarrollo()
	case puertosbolsa.AccionConsultarContactoParticipacion:
		return motivo == motivoConsultarContactoParticipacionBolsaDesarrollo()
	case puertosbolsa.AccionRegistrarDatosContactoParticipacion:
		return motivo == motivoRegistrarDatosContactoParticipacionBolsaDesarrollo()
	case puertosbolsa.AccionConsultarDatosContactoParticipacion:
		return motivo == motivoConsultarDatosContactoParticipacionBolsaDesarrollo()
	case puertosbolsa.AccionEmitirLlamamiento:
		return motivo == motivoEmitirLlamamientoBolsaDesarrollo()
	case puertosbolsa.AccionPublicarPoliticaOfertas:
		return motivo == motivoPublicarPoliticaOfertasBolsaDesarrollo()
	case puertosbolsa.AccionConsultarPoliticaOfertas:
		return motivo == motivoConsultarPoliticaOfertasBolsaDesarrollo()
	case puertosbolsa.AccionConsultarReincorporacionTitular:
		return motivo == motivoConsultarReincorporacionTitularBolsaDesarrollo()
	case puertosbolsa.AccionConfirmarCargaConvoca:
		return motivo == motivoConfirmarCargaConvocaBolsaDesarrollo()
	default:
		return false
	}
}

func motivoCambiarSituacionParticipacionBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_situacion_participacion_bolsa", CatalogoVersion: 1, CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-bolsa-b2-v1"), EntradaClave: referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-b2-situacion-cambiar")}
}

func motivoRegistrarContactoParticipacionBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_contacto_participacion_bolsa_v2", CatalogoVersion: 1, CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-bolsa-b3-v2"), EntradaClave: referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-b3-contacto-registrar")}
}

func motivoConsultarContactoParticipacionBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_contacto_participacion_bolsa_v2", CatalogoVersion: 1, CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-bolsa-b3-v2"), EntradaClave: referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-b3-contacto-consultar")}
}

func motivoRegistrarDatosContactoParticipacionBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_datos_contacto_participacion_bolsa", CatalogoVersion: 1, CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-bolsa-b4-v1"), EntradaClave: referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-b4-datos-contacto-registrar")}
}

func motivoEmitirLlamamientoBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_emision_llamamiento_bolsa", CatalogoVersion: 1, CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-bolsa-b7-v1"), EntradaClave: referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-b7-llamamiento-emitir")}
}

func motivoPublicarPoliticaOfertasBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_politica_ofertas_bolsa", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-bolsa-b47-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-b47-politica-ofertas-publicar")}
}

func motivoConsultarPoliticaOfertasBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_politica_ofertas_bolsa", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-bolsa-b47-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-b51-politica-ofertas-consultar")}
}

func motivoConsultarReincorporacionTitularBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_reincorporacion_titular_bolsa", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-bolsa-b55-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-b55-reincorporacion-titular-consultar")}
}

func motivoCrearBorradorLlamamientoBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_borrador_llamamiento_bolsa", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-bolsa-bback-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-bback-crear")}
}

func motivoConsultarBorradorLlamamientoBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_borrador_llamamiento_bolsa", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-bolsa-bback-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-bback-consultar")}
}

func motivoConfirmarCargaConvocaBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_carga_convoca_bolsa", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-bolsa-b1-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-b1-carga-convoca-confirmar")}
}

var _ puertosvec.FuenteAutorizacion = (*politicaBorradorLlamamientoBolsaDesarrollo)(nil)
var _ puertosvec.RegistroConcesionesCandidatasAutorizacionLigadaV3 = (*politicaBorradorLlamamientoBolsaDesarrollo)(nil)
var _ puertosvec.RegistroDenegacionesAutorizacionLigadaV3 = (*politicaBorradorLlamamientoBolsaDesarrollo)(nil)
var _ puertosvec.ValidadorReferenciaMotivoAutorizacionV2 = (*politicaBorradorLlamamientoBolsaDesarrollo)(nil)
