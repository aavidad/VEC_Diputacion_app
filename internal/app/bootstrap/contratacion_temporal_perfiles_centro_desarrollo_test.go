package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// autoridadCentroPrueba cuenta cualquier intento de preparar o publicar y
// entrega como publicada la instantánea que se le indique.
type autoridadCentroPrueba struct {
	mu                     sync.Mutex
	preparadas, publicadas int
	consumos               int
	publicada              func(vecdomain.InstantaneaAutorizacion) (vecdomain.InstantaneaAutorizacion, error)
}

func (a *autoridadCentroPrueba) PrepararInstantanea(_ context.Context, i vecdomain.InstantaneaAutorizacion) (vecdomain.InstantaneaAutorizacion, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.preparadas++
	return i, nil
}

func (a *autoridadCentroPrueba) PublicarInstantanea(context.Context, vecdomain.InstantaneaAutorizacion) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.publicadas++
	return nil
}

func (a *autoridadCentroPrueba) consumirInstantaneaPublicada(_ context.Context, plantilla vecdomain.InstantaneaAutorizacion) (vecdomain.InstantaneaAutorizacion, error) {
	a.mu.Lock()
	a.consumos++
	a.mu.Unlock()
	return a.publicada(plantilla)
}

type registroCentroPrueba struct {
	mu                        sync.Mutex
	concesiones, denegaciones int
}

func (r *registroCentroPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	context.Context, puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.concesiones++
	return time.Now().UTC().Truncate(time.Microsecond), nil
}

func (r *registroCentroPrueba) RegistrarDenegacionAutorizacionLigadaV3(
	context.Context, puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.denegaciones++
	return nil
}

func principalCentroPrueba(sufijo, rol string) vecdomain.Principal {
	huella := sha256.Sum256([]byte("centro:" + sufijo))
	return vecdomain.Principal{ID: "centro_" + sufijo, Roles: []string{rol},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": hex.EncodeToString(huella[:])}}
}

// escenarioIdentidadCentroPrueba compone la identidad del centro por el mismo
// camino que la composición real, con la cancelación de la regla activa.
func escenarioIdentidadCentroPrueba(t *testing.T, sufijo string) (*identidadPeticionCentroDesarrollo, *registroCentroPrueba) {
	t.Helper()
	base, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	principal := principalCentroPrueba(sufijo, "solicitante_centro")
	cancelacion := &cancelacionCentroDesarrollo{piezas: &piezasCancelacionCTDesarrollo{}, roles: []string{"solicitante_centro"}}
	concesiones := append([]vecdomain.ConcesionRol{{Accion: ports.AccionConsultarPeticionCentro, ModuloID: ports.ModuloContratacion,
		TipoRecurso: ports.TipoRecursoPeticionCentro, Finalidades: []string{finalidadPeticionCentro}, GarantiaMinima: vecdomain.AuthAssuranceHigh}},
		cancelacion.concesiones("solicitante_centro")...)
	registro := &registroCentroPrueba{}
	id, err := nuevaIdentidadPeticionCentroDesarrollo(base.sello, registro, nil, principal,
		adscripcionCentroDesarrollo{CentroRef: "centro-520", PuestoRef: "rpt-520-735"}, concesiones,
		cancelacion.concesionCancelar("solicitante_centro"), base.reloj)
	if err != nil {
		t.Fatal(err)
	}
	return id, registro
}

func contextoRutaCentroPrueba(id *identidadPeticionCentroDesarrollo, ruta string) context.Context {
	ahora := id.soporte.reloj.Ahora()
	return context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{},
		capacidadConsultaContratacionTemporalDesarrollo{sello: id.soporte.sello, ruta: ruta, principal: id.principal,
			certificadoVerificadoEn: ahora.Add(-time.Second), certificadoValidoHasta: ahora.Add(time.Hour),
			contextoOperacion: &contextoOperacionCTDesarrollo{}})
}

// exigirCentroPrueba reproduce autoridadCancelacionCentroDesarrollo.exigir
// sin la comprobación de pertenencia, que necesita la base.
func exigirCentroPrueba(ctx context.Context, p perfilCentroDesarrollo, accion, finalidad string, recurso vecdomain.RecursoAutorizable) error {
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return err
	}
	datos := vecdomain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: p.soporte.contexto.Vinculo, ReferenciaMotivo: motivoPeticionCentroDesarrollo(),
		Accion: accion, Recurso: recurso, Finalidad: finalidad, Correlacion: correlacion}
	ctx = context.WithValue(ctx, claveMaterialPeticionCentroDesarrollo{}, materialAutorizacionPeticionCentroDesarrollo{
		cancelacion: &recursoCancelacionCentroDesarrollo{accion: accion, finalidad: finalidad, recurso: recurso, actor: p.actor}})
	if !solicitudAutorizacionPeticionCentroDesarrolloValida(ctx, datos) {
		return errors.New("solicitud del centro no válida")
	}
	s, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return err
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	_, _, err = p.autorizador.ExigirSolicitudLigadaV3(ctx, s, p.soporte.contexto.Resultado)
	return err
}

func recursoConsultaCancelacionCentroPrueba(id *identidadPeticionCentroDesarrollo, expediente string) vecdomain.RecursoAutorizable {
	return vecdomain.RecursoAutorizable{Referencia: expediente, ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoCancelacion,
		Ambitos:   map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo, "centro_ref": id.actor.CentroRef},
		Atributos: map[string]string{"lectura": "cancelacion_opciones", "expediente_ref": expediente}}
}

func instantaneaPublicadaPrueba(plantilla vecdomain.InstantaneaAutorizacion, version int) vecdomain.InstantaneaAutorizacion {
	publicada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(plantilla)
	publicada.AsignacionPerfil.Version = version
	return publicada
}

func TestPerfilGeneralCentroConsumeSinPrepararNiPublicar(t *testing.T) {
	id, registro := escenarioIdentidadCentroPrueba(t, "consumo")
	autoridad := &autoridadCentroPrueba{publicada: func(p vecdomain.InstantaneaAutorizacion) (vecdomain.InstantaneaAutorizacion, error) {
		return instantaneaPublicadaPrueba(p, 7), nil
	}}
	id.soporte.autoridadAsignaciones = autoridad
	general := perfilGeneralCentroDesarrollo(id)
	var wg sync.WaitGroup
	errores := make(chan error, 30)
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := contextoRutaCentroPrueba(id, rutaCancelacionCentro)
			errores <- exigirCentroPrueba(ctx, general, accionConsultarCancelacionCTDesarrollo, finalidadPeticionCentro,
				recursoConsultaCancelacionCentroPrueba(id, "expediente:uno"))
		}()
	}
	wg.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("lectura del centro denegada: %v", err)
		}
	}
	if autoridad.preparadas != 0 || autoridad.publicadas != 0 {
		t.Fatalf("una lectura preparó (%d) o publicó (%d) la instantánea", autoridad.preparadas, autoridad.publicadas)
	}
	if autoridad.consumos != 30 || registro.concesiones != 30 {
		t.Fatalf("consumos %d, concesiones %d", autoridad.consumos, registro.concesiones)
	}
	if len(id.soporte.instantaneasPorSolicitud) != 0 {
		t.Fatal("el consumo no debe acumular instantáneas por petición")
	}
}

func TestPerfilGeneralCentroDeniegaSiLaPublicadaNoEsConsumible(t *testing.T) {
	id, registro := escenarioIdentidadCentroPrueba(t, "revocado")
	autoridad := &autoridadCentroPrueba{publicada: func(vecdomain.InstantaneaAutorizacion) (vecdomain.InstantaneaAutorizacion, error) {
		return vecdomain.InstantaneaAutorizacion{}, errPerfilCentroNoConsumible
	}}
	id.soporte.autoridadAsignaciones = autoridad
	ctx := contextoRutaCentroPrueba(id, rutaCancelacionCentro)
	if err := exigirCentroPrueba(ctx, perfilGeneralCentroDesarrollo(id), accionConsultarCancelacionCTDesarrollo, finalidadPeticionCentro,
		recursoConsultaCancelacionCentroPrueba(id, "expediente:uno")); err == nil {
		t.Fatal("una asignación no consumible concedió la lectura")
	}
	if autoridad.preparadas != 0 || autoridad.publicadas != 0 || registro.concesiones != 0 {
		t.Fatalf("la denegación preparó %d, publicó %d o concedió %d", autoridad.preparadas, autoridad.publicadas, registro.concesiones)
	}
}

func TestPerfilGeneralCentroNoCancelaConAmbitosDeExpediente(t *testing.T) {
	id, registro := escenarioIdentidadCentroPrueba(t, "sin-cancelar")
	autoridad := &autoridadCentroPrueba{publicada: func(p vecdomain.InstantaneaAutorizacion) (vecdomain.InstantaneaAutorizacion, error) {
		return instantaneaPublicadaPrueba(p, 1), nil
	}}
	id.soporte.autoridadAsignaciones = autoridad
	ctx := contextoRutaCentroPrueba(id, rutaCancelacionesCentro)
	if err := exigirCentroPrueba(ctx, perfilGeneralCentroDesarrollo(id), string(domain.AccionCancelarExpediente), ports.FinalidadCancelarExpediente,
		recursoCancelarCentroPrueba(id.actor.CentroRef)); err == nil || registro.concesiones != 0 {
		t.Fatal("el perfil general concedió una cancelación ligada al expediente")
	}
	if autoridad.preparadas != 0 || autoridad.publicadas != 0 {
		t.Fatal("el perfil general se estrechó por petición")
	}
}

func recursoCancelarCentroPrueba(centro string) vecdomain.RecursoAutorizable {
	return vecdomain.RecursoAutorizable{Referencia: "expediente:uno", ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoCancelacion,
		Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo, "expediente_ref": "expediente:uno",
			"fase_previa": "solicitud", "estado_previo": string(domain.EstadoEnCurso), "centro_ref": centro},
		Atributos: map[string]string{"canal": string(domain.CanalCancelacionCentro)}}
}

func TestPerfilCancelacionCentroEsPropioYSoloCancela(t *testing.T) {
	id, registro := escenarioIdentidadCentroPrueba(t, "cancela")
	if id.cancelacion == nil {
		t.Fatal("sin perfil de cancelación")
	}
	general, _ := id.soporte.contexto.Vinculo.Datos()
	propio, _ := id.cancelacion.soporte.contexto.Vinculo.Datos()
	if propio.PerfilActivoRef == general.PerfilActivoRef || propio.SesionRef == general.SesionRef ||
		propio.PrincipalID != general.PrincipalID || propio.CuentaRef != general.CuentaRef ||
		id.cancelacion.actor.PerfilRef != propio.PerfilActivoRef || id.cancelacion.actor.ActorRef != id.actor.ActorRef ||
		id.cancelacion.actor.CentroRef != id.actor.CentroRef {
		t.Fatal("el perfil de cancelación no es un perfil propio de la misma persona y centro")
	}
	concesiones := id.cancelacion.soporte.instantanea.VersionRol.Concesiones
	if len(concesiones) != 1 || concesiones[0].Accion != string(domain.AccionCancelarExpediente) ||
		id.cancelacion.soporte.instantanea.VersionRol.RolID != rolCancelacionCentroDesarrollo ||
		id.cancelacion.soporte.soloConsumePublicada || !id.cancelacion.soporte.perfilCancelacionCentro {
		t.Fatalf("rol del perfil de cancelación: %+v", id.cancelacion.soporte.instantanea.VersionRol)
	}
	comun := (&autoridadPostgreSQLContratacionTemporalDesarrollo{soporte: id.cancelacion.soporte}).autoridadComun()
	if !comun.exigirOrigenOperativo || comun.actoAsignacion != actoAsignacionCancelacionCentroDesarrollo ||
		comun.actoControlRol != actoControlRolCancelacionCentroDesarrollo {
		t.Fatal("el perfil de cancelación publica sin la guarda de origen operativo")
	}
	if generalComun := (&autoridadPostgreSQLContratacionTemporalDesarrollo{soporte: id.soporte}).autoridadComun(); generalComun.actoAsignacion != actoAsignacionCTDesarrollo {
		t.Fatal("el perfil general cambió de acto")
	}
	// La cancelación prepara y publica en su propio perfil.
	autoridad := &autoridadCentroPrueba{}
	id.cancelacion.soporte.autoridadAsignaciones = autoridad
	ctx := contextoRutaCentroPrueba(id, rutaCancelacionesCentro)
	if err := exigirCentroPrueba(ctx, *id.cancelacion, string(domain.AccionCancelarExpediente), ports.FinalidadCancelarExpediente,
		recursoCancelarCentroPrueba(id.actor.CentroRef)); err != nil {
		t.Fatalf("cancelación denegada en su perfil: %v", err)
	}
	if autoridad.preparadas != 1 || autoridad.publicadas != 1 || registro.concesiones != 1 {
		t.Fatalf("cancelación: preparadas %d, publicadas %d, concesiones %d", autoridad.preparadas, autoridad.publicadas, registro.concesiones)
	}
	// Y no sirve para leer: la consulta de opciones pertenece al perfil general.
	ctx = contextoRutaCentroPrueba(id, rutaCancelacionCentro)
	if err := exigirCentroPrueba(ctx, *id.cancelacion, accionConsultarCancelacionCTDesarrollo, finalidadPeticionCentro,
		recursoConsultaCancelacionCentroPrueba(id, "expediente:uno")); err == nil {
		t.Fatal("el perfil de cancelación concedió una lectura")
	}
	if autoridad.publicadas != 1 {
		t.Fatal("una lectura publicó en el perfil de cancelación")
	}
}

func TestPerfilCentroSeEligePorLaRutaDelManejador(t *testing.T) {
	id, _ := escenarioIdentidadCentroPrueba(t, "ruta")
	general, err := perfilCentroPorRutaDesarrollo(id, rutaCancelacionCentro)
	if err != nil || general.soporte != id.soporte {
		t.Fatal("la consulta de opciones no usa el perfil general")
	}
	propio, err := perfilCentroPorRutaDesarrollo(id, rutaCancelacionesCentro)
	if err != nil || propio.soporte != id.cancelacion.soporte {
		t.Fatal("la cancelación no usa su perfil propio")
	}
	if _, err := perfilCentroPorRutaDesarrollo(id, rutaBandejaPeticionCentro); err == nil {
		t.Fatal("fuera de la cancelación no se elige perfil")
	}
	sinCancelar := *id
	sinCancelar.cancelacion = nil
	if _, err := perfilCentroPorRutaDesarrollo(&sinCancelar, rutaCancelacionesCentro); err == nil {
		t.Fatal("sin perfil de cancelación se cancelaría con el general")
	}
}

func TestConsultaCancelacionCentroUsaAmbitosGenerales(t *testing.T) {
	actor := domain.ActorPeticionCentro{ActorRef: "actor:centro", PerfilRef: "perfil:centro", CentroRef: "centro-520", PuestoRef: "puesto:1"}
	recurso := vecdomain.RecursoAutorizable{Referencia: "expediente:1", ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoCancelacion,
		Ambitos:   map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo, "centro_ref": "centro-520"},
		Atributos: map[string]string{"lectura": "cancelacion_opciones", "expediente_ref": "expediente:1"}}
	r := &recursoCancelacionCentroDesarrollo{accion: accionConsultarCancelacionCTDesarrollo, finalidad: finalidadPeticionCentro, recurso: recurso, actor: actor}
	d := vecdomain.DatosSolicitudAutorizacionLigadaV3{Accion: r.accion, Finalidad: r.finalidad, Recurso: recurso}
	if !r.validaPara("actor:centro", "perfil:centro", d) {
		t.Fatal("consulta con ámbitos generales rechazada")
	}
	antigua := recurso
	antigua.Ambitos = map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo, "centro_ref": "centro-520", "expediente_ref": "expediente:1"}
	antigua.Atributos = map[string]string{"lectura": "cancelacion_opciones"}
	r.recurso, d.Recurso = antigua, antigua
	if r.validaPara("actor:centro", "perfil:centro", d) {
		t.Fatal("la consulta ligada al expediente estrecharía el perfil general")
	}
	otro := recurso
	otro.Atributos = map[string]string{"lectura": "cancelacion_opciones", "expediente_ref": "expediente:2"}
	r.recurso, d.Recurso = otro, otro
	if r.validaPara("actor:centro", "perfil:centro", d) {
		t.Fatal("expediente del atributo distinto de la referencia")
	}
}

func instantaneaCentroPlantillaPrueba(t *testing.T) (vecdomain.InstantaneaAutorizacion, time.Time) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	plantilla, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo("centro_principal", "prf_centro", ahora, "solicitante_centro",
		"Petición de centro de desarrollo", "peticion-centro-desarrollo-prueba",
		[]vecdomain.ConcesionRol{{Accion: ports.AccionConsultarPeticionCentro, ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoPeticionCentro,
			Finalidades: []string{finalidadPeticionCentro}, GarantiaMinima: vecdomain.AuthAssuranceHigh}},
		[]vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
			{Clave: "centro_ref", Valores: []string{"centro-520"}}})
	if err != nil {
		t.Fatal(err)
	}
	return plantilla, ahora
}

// publicadaDesdeJSONPrueba simula la rehidratación desde los documentos.
func publicadaDesdeJSONPrueba(t *testing.T, i vecdomain.InstantaneaAutorizacion, actoAsignacion, actoControl string) instantaneaPublicadaDesarrollo {
	t.Helper()
	var copia vecdomain.InstantaneaAutorizacion
	for _, par := range []struct{ origen, destino any }{
		{i.AsignacionPerfil, &copia.AsignacionPerfil}, {i.VersionRol, &copia.VersionRol}, {i.ControlVigenciaVersionRol, &copia.ControlVigenciaVersionRol},
	} {
		documento, err := json.Marshal(par.origen)
		if err != nil || json.Unmarshal(documento, par.destino) != nil {
			t.Fatal("documento no rehidratable")
		}
	}
	copia.RevisionCatalogoPoliticas, copia.CatalogoPoliticasHuellaSHA256 = i.RevisionCatalogoPoliticas, i.CatalogoPoliticasHuellaSHA256
	return instantaneaPublicadaDesarrollo{instantanea: copia, actoAsignacion: actoAsignacion,
		actualizadaPor: copia.AsignacionPerfil.EmitidaPor, actoControl: actoControl}
}

func conVersionesPrueba(i vecdomain.InstantaneaAutorizacion, rol, asignacion int) vecdomain.InstantaneaAutorizacion {
	i = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(i)
	i.VersionRol.Version = rol
	i.AsignacionPerfil.VersionRolRef = i.VersionRol.Referencia()
	i.ControlVigenciaVersionRol.VersionRolRef = i.VersionRol.Referencia()
	i.AsignacionPerfil.Version = asignacion
	return i
}

func TestInstantaneaConsumibleExigeLaPlantillaExactaYOperativa(t *testing.T) {
	plantilla, ahora := instantaneaCentroPlantillaPrueba(t)
	exacta := conVersionesPrueba(plantilla, 3, 12)
	consumida, ok := instantaneaConsumible(publicadaDesdeJSONPrueba(t, exacta, actoAsignacionCTDesarrollo, actoControlRolCTDesarrollo), plantilla, ahora)
	if !ok || !mismasHuellasInstantaneaDesarrollo(consumida, exacta) {
		t.Fatal("la asignación publicada exacta no se consume")
	}
	casos := map[string]func(*vecdomain.InstantaneaAutorizacion){
		"revocada": func(i *vecdomain.InstantaneaAutorizacion) {
			i.AsignacionPerfil.Estado = vecdomain.EstadoAsignacionPerfilRevocada
			i.AsignacionPerfil.RevocadaEn = i.AsignacionPerfil.EmitidaEn.Add(time.Second)
			i.AsignacionPerfil.RevocadaPor = "seguridad:prueba"
			i.AsignacionPerfil.RevocacionRef = "revocacion:prueba"
		},
		"rol_retirado": func(i *vecdomain.InstantaneaAutorizacion) {
			i.ControlVigenciaVersionRol.Estado = vecdomain.EstadoControlVigenciaVersionRolRetirada
			i.ControlVigenciaVersionRol.Revision = 2
		},
		"estrechada": func(i *vecdomain.InstantaneaAutorizacion) {
			i.AsignacionPerfil.Ambitos = append(i.AsignacionPerfil.Ambitos, vecdomain.AmbitoPerfil{Clave: "expediente_ref", Valores: []string{"expediente:uno"}})
		},
		"otro_centro": func(i *vecdomain.InstantaneaAutorizacion) {
			i.AsignacionPerfil.Ambitos[1].Valores = []string{"centro-999"}
		},
		"otras_concesiones": func(i *vecdomain.InstantaneaAutorizacion) {
			i.VersionRol.Concesiones[0].Accion = ports.AccionPresentarPeticionCentro
			i.AsignacionPerfil.VersionRolRef = i.VersionRol.Referencia()
			i.ControlVigenciaVersionRol.VersionRolRef = i.VersionRol.Referencia()
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			distinta := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(exacta)
			alterar(&distinta)
			if _, ok := instantaneaConsumible(publicadaDesdeJSONPrueba(t, distinta, actoAsignacionCTDesarrollo, actoControlRolCTDesarrollo), plantilla, ahora); ok {
				t.Fatal("se consumiría una asignación distinta de la del centro")
			}
		})
	}
}

func TestPreimagenAutoproducidaSoloReconoceElEstrechamientoPropio(t *testing.T) {
	plantilla, ahora := instantaneaCentroPlantillaPrueba(t)
	estrechada := conVersionesPrueba(plantilla, 3, 12)
	estrechada.AsignacionPerfil.Ambitos = append(estrechada.AsignacionPerfil.Ambitos,
		vecdomain.AmbitoPerfil{Clave: "expediente_ref", Valores: []string{"expediente:uno"}})
	if !preimagenAutoproducidaCentroDesarrollo(publicadaDesdeJSONPrueba(t, estrechada, actoAsignacionCTDesarrollo, actoControlRolCTDesarrollo), plantilla, ahora) {
		t.Fatal("el estrechamiento por petición del binario anterior no se reconoce")
	}
	if preimagenAutoproducidaCentroDesarrollo(publicadaDesdeJSONPrueba(t, estrechada, "acto:seguridad:restriccion", actoControlRolCTDesarrollo), plantilla, ahora) {
		t.Fatal("una restricción de otro acto se trataría como propia")
	}
	if preimagenAutoproducidaCentroDesarrollo(publicadaDesdeJSONPrueba(t, estrechada, actoAsignacionCTDesarrollo, "acto:seguridad:control"), plantilla, ahora) {
		t.Fatal("un control movido por otro acto se trataría como propio")
	}
	revocada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(estrechada)
	revocada.AsignacionPerfil.Estado = vecdomain.EstadoAsignacionPerfilRevocada
	revocada.AsignacionPerfil.RevocadaEn = revocada.AsignacionPerfil.EmitidaEn.Add(time.Second)
	revocada.AsignacionPerfil.RevocadaPor = "seguridad:prueba"
	revocada.AsignacionPerfil.RevocacionRef = "revocacion:prueba"
	if preimagenAutoproducidaCentroDesarrollo(publicadaDesdeJSONPrueba(t, revocada, actoAsignacionCTDesarrollo, actoControlRolCTDesarrollo), plantilla, ahora) {
		t.Fatal("una asignación revocada se reactivaría")
	}
	retirada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(estrechada)
	retirada.ControlVigenciaVersionRol.Estado = vecdomain.EstadoControlVigenciaVersionRolRetirada
	retirada.ControlVigenciaVersionRol.Revision = 2
	if preimagenAutoproducidaCentroDesarrollo(publicadaDesdeJSONPrueba(t, retirada, actoAsignacionCTDesarrollo, actoControlRolCTDesarrollo), plantilla, ahora) {
		t.Fatal("un rol retirado se reactivaría")
	}
	otroCentro := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(estrechada)
	otroCentro.AsignacionPerfil.Ambitos[1].Valores = []string{"centro-999"}
	if preimagenAutoproducidaCentroDesarrollo(publicadaDesdeJSONPrueba(t, otroCentro, actoAsignacionCTDesarrollo, actoControlRolCTDesarrollo), plantilla, ahora) {
		t.Fatal("un estrechamiento de otro centro se reconocería")
	}
	sinCentro := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(estrechada)
	sinCentro.AsignacionPerfil.Ambitos = []vecdomain.AmbitoPerfil{sinCentro.AsignacionPerfil.Ambitos[0], sinCentro.AsignacionPerfil.Ambitos[2]}
	if preimagenAutoproducidaCentroDesarrollo(publicadaDesdeJSONPrueba(t, sinCentro, actoAsignacionCTDesarrollo, actoControlRolCTDesarrollo), plantilla, ahora) {
		t.Fatal("una asignación sin el centro de la plantilla se ampliaría")
	}
	otroRol := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(estrechada)
	otroRol.VersionRol.RolID = "ratificador_centro"
	otroRol.AsignacionPerfil.VersionRolRef = otroRol.VersionRol.Referencia()
	otroRol.ControlVigenciaVersionRol.VersionRolRef = otroRol.VersionRol.Referencia()
	if preimagenAutoproducidaCentroDesarrollo(publicadaDesdeJSONPrueba(t, otroRol, actoAsignacionCTDesarrollo, actoControlRolCTDesarrollo), plantilla, ahora) {
		t.Fatal("un rol distinto se transformaría en el del centro")
	}
}

func TestAsignacionActualOperativaDeLaGuarda(t *testing.T) {
	plantilla, ahora := instantaneaCentroPlantillaPrueba(t)
	i := conVersionesPrueba(plantilla, 1, 4)
	documento, _ := json.Marshal(i.AsignacionPerfil)
	huella, _ := i.AsignacionPerfil.HuellaSHA256()
	actual := asignacionActualPostgreSQLDesarrollo{referencia: i.AsignacionPerfil.Referencia(), identificador: i.AsignacionPerfil.AsignacionID,
		version: 4, perfilRef: i.AsignacionPerfil.PerfilActivoRef, principalID: i.AsignacionPerfil.PrincipalID,
		versionRolRef: i.VersionRol.Referencia(), huella: huella, documento: documento, actoRef: actoAsignacionCancelacionCentroDesarrollo,
		actualizadaPor: i.AsignacionPerfil.EmitidaPor}
	if !asignacionActualOperativaPostgreSQLDesarrollo(actual, "habilitada", ahora) {
		t.Fatal("asignación operativa rechazada")
	}
	if asignacionActualOperativaPostgreSQLDesarrollo(actual, "retirada", ahora) {
		t.Fatal("rol retirado admitido")
	}
	if asignacionActualOperativaPostgreSQLDesarrollo(actual, "habilitada", i.AsignacionPerfil.VigenteHasta) {
		t.Fatal("asignación caducada admitida")
	}
	otra := actual
	otra.actualizadaPor = "seguridad:otra"
	if asignacionActualOperativaPostgreSQLDesarrollo(otra, "habilitada", ahora) {
		t.Fatal("puntero movido por otro emisor admitido")
	}
	revocada := i
	revocada.AsignacionPerfil.Estado = vecdomain.EstadoAsignacionPerfilRevocada
	revocada.AsignacionPerfil.RevocadaEn = i.AsignacionPerfil.EmitidaEn.Add(time.Second)
	revocada.AsignacionPerfil.RevocadaPor = "seguridad:prueba"
	revocada.AsignacionPerfil.RevocacionRef = "revocacion:prueba"
	actualRevocada := actual
	actualRevocada.documento, _ = json.Marshal(revocada.AsignacionPerfil)
	actualRevocada.huella, _ = revocada.AsignacionPerfil.HuellaSHA256()
	actualRevocada.referencia = revocada.AsignacionPerfil.Referencia()
	if asignacionActualOperativaPostgreSQLDesarrollo(actualRevocada, "habilitada", ahora) {
		t.Fatal("asignación revocada admitida")
	}
}

func TestAprobacionProvisionPerfilesCentroSoloEnDesarrolloYConForma(t *testing.T) {
	if (config.Config{CTAprobacionPerfilesCentro: "aprobacion:ct:centro:20260929"}).CTProvisionPerfilesCentroAprobacionRef() != "" {
		t.Fatal("fuera de desarrollo no hay provisión")
	}
	desarrollo := config.Config{ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment,
		DevelopmentGuard: config.DevelopmentGuardAcknowledgement}
	for valor, esperado := range map[string]string{
		"aprobacion:ct:centro:20260929": "aprobacion:ct:centro:20260929",
		"corta":                         "",
		"con espacios internos":         "",
		"aprobación:acentuada:1":        "",
		"":                              "",
	} {
		desarrollo.CTAprobacionPerfilesCentro = valor
		if got := desarrollo.CTProvisionPerfilesCentroAprobacionRef(); got != esperado {
			t.Fatalf("%q: %q", valor, got)
		}
	}
}
