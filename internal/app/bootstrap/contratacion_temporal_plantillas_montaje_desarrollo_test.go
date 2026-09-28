package bootstrap

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	plantillashttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpapi/plantillascatalogo"
	cthttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	plantillasapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestPlantillasCatalogoCTSelectorYFronteras(t *testing.T) {
	cfg := configPlantillasDesarrolloPrueba("../../../" + rutaPlantillasCTEjemplo)
	if activo, err := plantillasCatalogoCTDesarrolloSolicitado(cfg); err != nil || activo {
		t.Fatalf("ausente debe estar apagado: %v %v", activo, err)
	}
	t.Setenv(envCTPlantillasGobiernoEnabled, "true")
	if activo, err := plantillasCatalogoCTDesarrolloSolicitado(cfg); err != nil || !activo {
		t.Fatalf("selector CT131 independiente de B-BACK = %v %v", activo, err)
	}
	cfg.ReglasEjemplo.CTPlantillasSourcePath = ""
	if _, err := plantillasCatalogoCTDesarrolloSolicitado(cfg); err == nil {
		t.Fatal("sin catálogo provisionable no se debe activar")
	}
	cfg = configPlantillasDesarrolloPrueba("../../../" + rutaPlantillasCTEjemplo)
	t.Setenv(envCTPlantillasGobiernoEnabled, "TRUE")
	if _, err := plantillasCatalogoCTDesarrolloSolicitado(cfg); err == nil {
		t.Fatal("selector no canónico admitido")
	}
	t.Setenv(envCTPlantillasGobiernoEnabled, "true")
	fronteras, err := descriptoresFronterasContratacionTemporalConPlantillasDesarrollo(
		"prf_ct_prueba", []string{"prf_ct_prueba"}, false, true, "prf_plantillas_prueba")
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	for _, par := range []struct{ metodo, ruta, accion string }{
		{http.MethodGet, plantillashttp.RutaCatalogo, "contratacion_temporal.plantillas_documentos.consultar"},
		{http.MethodPost, plantillashttp.RutaEntradas, "contratacion_temporal.plantillas_documentos.editar"},
		{http.MethodPost, plantillashttp.RutaPublicar, "contratacion_temporal.plantillas_documentos.publicar"},
	} {
		d, ok := catalogo.resolver(par.metodo, par.ruta)
		if !ok || d.ClaveCapacidad != par.accion || !d.admitePerfil("prf_plantillas_prueba") || d.admitePerfil("prf_ct_prueba") {
			t.Fatalf("frontera %s %s = %v %#v", par.metodo, par.ruta, ok, d)
		}
		if _, ok := catalogo.resolver(http.MethodHead, par.ruta); ok {
			t.Fatal("HEAD adquirió capacidad")
		}
	}
	if _, ok := catalogo.resolver(http.MethodGet, plantillashttp.RutaEntradas); ok {
		t.Fatal("GET adquirió escritura")
	}
	if _, ok := catalogo.resolver(http.MethodPost, cthttp.RutaConsultaDetalleRRHH); !ok {
		t.Fatal("detalle existente perdido")
	}
	if _, err := nuevoCatalogoAutorizacionComunDesarrollo(catalogo,
		descriptoresAutorizacionContratacionTemporalDesarrollo(politicaDescriptoresCTPrueba(t), false)); err != nil {
		t.Fatalf("PDP CT base alterado por plantillas: %v", err)
	}
	if _, err := descriptoresFronterasContratacionTemporalConPlantillasDesarrollo(
		"prf_ct_prueba", []string{"prf_ct_prueba"}, false, true, "prf_ct_prueba"); err == nil {
		t.Fatal("perfil CT base reutilizado para CT131")
	}
}

func TestPlantillasCatalogoCTSolicitudesLimitadasACanalYMotivo(t *testing.T) {
	const sha = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	base := vecdomain.DatosSolicitudAutorizacionLigadaV3{
		Accion: "contratacion_temporal.plantillas_documentos.editar", Finalidad: finalidadCatalogoPlantillasCT,
		ReferenciaMotivo: motivoCatalogoPlantillasCTDesarrollo(),
		Recurso: vecdomain.RecursoAutorizable{Referencia: "vec.contratacion_temporal.plantillas_documentos",
			ModuloID: "contratacion_temporal", Tipo: tipoCatalogoPlantillasCT,
			Ambitos:   map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo},
			Atributos: map[string]string{"material_sha256": sha}},
	}
	if !solicitudAutorizacionPlantillasCTDesarrolloValida(plantillashttp.RutaEntradas, base) {
		t.Fatal("edición nominal CT131 rechazada")
	}
	if solicitudAutorizacionPlantillasCTDesarrolloValida(plantillashttp.RutaPublicar, base) {
		t.Fatal("edición aceptada en publicación")
	}
	base.Recurso.Ambitos["organizacion_ref"] = "organizacion:ajena"
	if solicitudAutorizacionPlantillasCTDesarrolloValida(plantillashttp.RutaEntradas, base) {
		t.Fatal("organización ajena autorizada")
	}
}

func TestPlantillasCatalogoCTPerfilDedicadoYConcesionesLimitadas(t *testing.T) {
	ahora := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	principal := vecdomain.Principal{ID: "rrhh:ct:desarrollo", Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": strings.Repeat("a", 64)}}
	base, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	soporte := &soporteAltaContratacionTemporalDesarrollo{sello: &selloConsultasContratacionTemporalDesarrollo{},
		principalID: principal.ID, certificadoSHA256: principal.Attributes["certificate_sha256"], contexto: base,
		reloj: relojContratacionTemporalDesarrollo{}}
	dedicado, perfil, err := nuevoSoportePlantillasCatalogoCTDesdeBaseDesarrollo(soporte, ahora)
	if err != nil || dedicado == nil || perfil == base.Resultado.Contexto.PerfilActivoRef ||
		dedicado.contexto.Resultado.Contexto.PersonaRef != base.Resultado.Contexto.PersonaRef ||
		dedicado.contexto.Resultado.Contexto.Instantanea.CuentaRef != base.Resultado.Contexto.Instantanea.CuentaRef {
		t.Fatalf("perfil CT131 no segregado: %v", err)
	}
	semilla, err := instantaneaInicialPlantillasCatalogoCTDesarrollo(dedicado.contexto.Resultado.Contexto.Principal.ID, perfil, ahora)
	if err != nil || semilla.Validar() != nil || len(semilla.VersionRol.Concesiones) != 3 ||
		len(semilla.AsignacionPerfil.Ambitos) != 1 || semilla.AsignacionPerfil.Ambitos[0].Clave != "organizacion_ref" ||
		semilla.AsignacionPerfil.Ambitos[0].Valores[0] != organizacionAltaContratacionTemporalDesarrollo {
		t.Fatalf("semilla CT131 amplía perfil: %v", err)
	}
	for _, concesion := range semilla.VersionRol.Concesiones {
		if strings.Contains(concesion.Accion, "documental") || concesion.TipoRecurso != tipoCatalogoPlantillasCT ||
			len(concesion.Obligaciones) != 0 {
			t.Fatalf("concesión CT133 o ajena en CT131: %#v", concesion)
		}
	}
}

type fuentePerfilPlantillasCTPrueba struct {
	actual    vecdomain.InstantaneaAutorizacion
	err       error
	consultas int
}

func (f *fuentePerfilPlantillasCTPrueba) ObtenerInstantaneaAutorizacion(_ context.Context, _, _ string) (vecdomain.InstantaneaAutorizacion, error) {
	f.consultas++
	return f.actual, f.err
}

type publicadorPerfilPlantillasCTPrueba struct {
	publicaciones int
	err           error
}

func (p *publicadorPerfilPlantillasCTPrueba) PublicarInicial(_ context.Context, _ vecdomain.InstantaneaAutorizacion) error {
	p.publicaciones++
	return p.err
}

func TestPlantillasCatalogoCTSoloInicialConservaRevocacionRestriccionYReinicio(t *testing.T) {
	ahora := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	semilla, err := instantaneaInicialPlantillasCatalogoCTDesarrollo("per_ct_plantillas", "prf_ct_plantillas", ahora)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre              string
		actual              vecdomain.InstantaneaAutorizacion
		errFuente           error
		errPublicar         error
		quierePublicaciones int
		quiereError         bool
	}{
		{nombre: "instalacion_inicial", errFuente: vecports.ErrAsignacionPerfilNoEncontrada, quierePublicaciones: 1},
		{nombre: "reinicio", actual: semilla},
		{nombre: "restriccion", actual: func() vecdomain.InstantaneaAutorizacion {
			x := semilla
			x.VersionRol.Concesiones = append([]vecdomain.ConcesionRol(nil), x.VersionRol.Concesiones[:2]...)
			return x
		}()},
		{nombre: "revocacion", actual: func() vecdomain.InstantaneaAutorizacion {
			x := semilla
			x.AsignacionPerfil.Estado = vecdomain.EstadoAsignacionPerfilRevocada
			x.AsignacionPerfil.RevocadaPor = "seguridad:desarrollo"
			x.AsignacionPerfil.RevocadaEn = ahora.Add(time.Minute)
			x.AsignacionPerfil.RevocacionRef = "revocacion:ct:plantillas"
			return x
		}()},
		{nombre: "carrera_cas", errFuente: vecports.ErrAsignacionPerfilNoEncontrada, errPublicar: errors.New("concurrencia"), quierePublicaciones: 1, quiereError: true},
		{nombre: "fuente_no_disponible", errFuente: vecports.ErrFuenteAutorizacionNoDisponible, quiereError: true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			fuente := &fuentePerfilPlantillasCTPrueba{actual: caso.actual, err: caso.errFuente}
			publicador := &publicadorPerfilPlantillasCTPrueba{err: caso.errPublicar}
			err := asegurarPerfilPlantillasCatalogoCTSoloInicial(context.Background(), fuente, publicador, semilla)
			if (err != nil) != caso.quiereError || fuente.consultas != 1 || publicador.publicaciones != caso.quierePublicaciones {
				t.Fatalf("CAS inicial = err %v, consultas %d, publicaciones %d", err, fuente.consultas, publicador.publicaciones)
			}
		})
	}
}

type sesionPlantillasCTPrueba struct {
	contexto contextoSeguridadComunDesarrollo
}

func (s sesionPlantillasCTPrueba) ResolverContexto(context.Context) (contextoSeguridadComunDesarrollo, error) {
	return s.contexto, nil
}

type pdpIndicadorPlantillasCTPrueba struct{ preparaciones, exigencias int }

func (p *pdpIndicadorPlantillasCTPrueba) ExigirSolicitudLigadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3,
	vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3,
	vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	p.exigencias++
	return vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, vecdomain.ErrAutorizacionDenegada
}

func (p *pdpIndicadorPlantillasCTPrueba) PrepararRegistroCompuestoSolicitudLigadaV3(context.Context,
	vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2,
	vecports.GeneradorReferenciaDecisionAutorizacion) (vecdomain.DecisionAutorizacionLigadaV3,
	vecports.CandidataRegistroDecisionAutorizacionLigadaV3, error) {
	p.preparaciones++
	return vecdomain.DecisionAutorizacionLigadaV3{}, vecports.CandidataRegistroDecisionAutorizacionLigadaV3{}, vecdomain.ErrAutorizacionDenegada
}

func TestPlantillasCatalogoCTIndicadorSoloDesdeGETAutenticado(t *testing.T) {
	ahora := relojContratacionTemporalDesarrollo{}.Ahora()
	principal := vecdomain.Principal{ID: "rrhh:ct:desarrollo", Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": strings.Repeat("a", 64)}}
	baseContexto, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	base := &soporteAltaContratacionTemporalDesarrollo{sello: &selloConsultasContratacionTemporalDesarrollo{},
		principalID: principal.ID, certificadoSHA256: principal.Attributes["certificate_sha256"], contexto: baseContexto,
		reloj: relojContratacionTemporalDesarrollo{}}
	soporte, perfil, err := nuevoSoportePlantillasCatalogoCTDesdeBaseDesarrollo(base, ahora)
	if err != nil {
		t.Fatal(err)
	}
	soporte.contextoEsperadoRegistrado = soporte.contexto.Resultado
	soporte.sesionOperativa = sesionPlantillasCTPrueba{contextoSeguridadComunDesarrollo{
		Vinculo: soporte.contexto.Vinculo, Resultado: soporte.contexto.Resultado}}
	fronteras, err := nuevoCatalogoFronterasComunDesarrollo(descriptoresFronterasPlantillasCTDesarrollo(perfil))
	if err != nil {
		t.Fatal(err)
	}
	descriptor, ok := fronteras.resolver(http.MethodGet, plantillashttp.RutaCatalogo)
	if !ok {
		t.Fatal("GET no registrado")
	}
	capacidad := capacidadConsultaContratacionTemporalDesarrollo{sello: soporte.sello, ruta: plantillashttp.RutaCatalogo,
		principal: principal, certificadoVerificadoEn: ahora, certificadoValidoHasta: ahora.Add(time.Hour),
		contextoOperacion: &contextoOperacionCTDesarrollo{}}
	ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	ctx = context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: http.MethodGet, ruta: plantillashttp.RutaCatalogo, superficie: superficieInternaSeguridadComunDesarrollo,
		catalogo: fronteras, descriptor: descriptor})
	pdp := &pdpIndicadorPlantillasCTPrueba{}
	p := &proveedorCatalogoPlantillasCT{soporte: soporte, pdp: pdp,
		motivo: motivoCatalogoPlantillasCTDesarrollo(), reloj: relojContratacionTemporalDesarrollo{}}
	if _, err := p.contextoVigente(ctx, "contratacion_temporal.plantillas_documentos.editar", true); err != nil {
		t.Fatalf("indicador GET legítimo denegado: %v", err)
	}
	if _, err := p.contextoVigente(ctx, "contratacion_temporal.plantillas_documentos.editar", false); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatalf("GET convertido en efecto POST: %v", err)
	}
	actor, err := p.ResolverContextoActor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	indicador := vecdomain.RecursoAutorizable{Referencia: plantillasapp.CatalogoID, ModuloID: plantillasapp.ModuloID,
		Tipo: tipoCatalogoPlantillasCT, Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo},
		Atributos: map[string]string{"operacion": "editar", "estado": "publicado", "version": "1", "revision": "0"}}
	puede, err := p.ComprobarCapacidadCatalogoPlantillas(ctx, actor, "contratacion_temporal.plantillas_documentos.editar", indicador)
	if err != nil || puede || pdp.preparaciones != 1 || pdp.exigencias != 0 {
		t.Fatalf("indicador denegado exigió concesión: puede=%v err=%v preparaciones=%d exigencias=%d",
			puede, err, pdp.preparaciones, pdp.exigencias)
	}
	descriptorPOST, ok := fronteras.resolver(http.MethodPost, plantillashttp.RutaEntradas)
	if !ok {
		t.Fatal("POST edición no registrado")
	}
	capacidad.ruta = plantillashttp.RutaEntradas
	capacidad.contextoOperacion = &contextoOperacionCTDesarrollo{}
	ctxPOST := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	ctxPOST = context.WithValue(ctxPOST, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: http.MethodPost, ruta: plantillashttp.RutaEntradas, superficie: superficieInternaSeguridadComunDesarrollo,
		catalogo: fronteras, descriptor: descriptorPOST})
	if _, err := p.contextoVigente(ctxPOST, "contratacion_temporal.plantillas_documentos.consultar", false); err != nil {
		t.Fatalf("lectura V3 previa al POST edición denegada: %v", err)
	}
	if _, err := p.contextoVigente(ctxPOST, "contratacion_temporal.plantillas_documentos.publicar", false); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatalf("POST edición adquirió publicación: %v", err)
	}
}

type filaACLPlantillasCTPrueba struct{ valida bool }

func (f filaACLPlantillasCTPrueba) Scan(destinos ...any) error {
	*(destinos[0].(*bool)) = f.valida
	return nil
}

type consultaACLPlantillasCTPrueba struct {
	valida     bool
	sql        string
	argumentos []any
}

func (c *consultaACLPlantillasCTPrueba) QueryRow(_ context.Context, sql string, argumentos ...any) pgx.Row {
	c.sql = sql
	c.argumentos = append([]any(nil), argumentos...)
	return filaACLPlantillasCTPrueba{c.valida}
}

func TestPlantillasCatalogoCTPreflightDeniegaRolesYACLIncompletas(t *testing.T) {
	if pool, err := abrirPoolAutorizacionRRHHDesarrollo(context.Background(), "", "vec_autorizacion_propietario", "plantillas"); pool != nil || !errors.Is(err, plantillasapp.ErrNoDisponible) {
		t.Fatalf("rol propietario admitido como LOGIN de lectura: %v", err)
	}
	fuente := &consultaACLPlantillasCTPrueba{valida: true}
	motivos := &consultaACLPlantillasCTPrueba{valida: true}
	if err := comprobarPreflightAutoridadesPlantillasCT(context.Background(), fuente, motivos); err != nil {
		t.Fatalf("ACL nominal positiva = %v", err)
	}
	for _, c := range []*consultaACLPlantillasCTPrueba{fuente, motivos} {
		for _, fragmento := range []string{"current_user=session_user", "pg_has_role(session_user,$3::regrole,'USAGE')",
			"has_schema_privilege(session_user,'vec_autorizacion','USAGE')",
			"has_schema_privilege($3::text,'vec_autorizacion','USAGE')",
			"NOT pg_catalog.has_schema_privilege(session_user,'vec_autorizacion','CREATE')",
			"to_regrole('vec_contratacion_temporal_ejecutor')", "to_regrole('vec_bolsa_llamamientos_ejecutor')",
			"to_regnamespace('vec_bolsa_llamamientos')", "consultar_auditoria_ct_atestada_v1",
			"registrar_auditoria_frontera_auditoria_v1",
			"has_table_privilege", "has_any_column_privilege", "NOT coalesce(pg_catalog.has_function_privilege"} {
			if !strings.Contains(c.sql, fragmento) {
				t.Fatalf("ACL omitida: %s", fragmento)
			}
		}
		if len(c.argumentos) != 3 {
			t.Fatalf("sin rol/función nominal: %v", c.argumentos)
		}
	}
	if fuente.argumentos[2] != config.RolAutorizacionFuenteRRHH || motivos.argumentos[2] != config.RolAutorizacionMotivosEvaluadorRRHH {
		t.Fatal("rol de fuente/evaluador cruzado")
	}
	fuente.valida = false
	if err := comprobarPreflightAutoridadesPlantillasCT(context.Background(), fuente, motivos); !errors.Is(err, plantillasapp.ErrNoDisponible) {
		t.Fatalf("fuente con ACL ajena admitida: %v", err)
	}
	fuente.valida = true
	motivos.valida = false
	if err := comprobarPreflightAutoridadesPlantillasCT(context.Background(), fuente, motivos); !errors.Is(err, plantillasapp.ErrNoDisponible) {
		t.Fatalf("evaluador con ACL ajena admitido: %v", err)
	}
}
