package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	plantillashttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpapi/plantillascatalogo"
	plantillasapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestProveedorDocumentalPlantillasCTRutaYRecursoExactos(t *testing.T) {
	casos := []struct{ accion, ruta string }{
		{"contratacion_temporal.plantillas_documentos.documental_listar", plantillashttp.RutaBorradoresDisponibles},
		{"contratacion_temporal.plantillas_documentos.documental_descargar", plantillashttp.RutaBorradores},
	}
	base := vecdomain.RecursoAutorizable{
		Referencia: "expediente:ct:0001", ModuloID: plantillasapp.ModuloID, Tipo: tipoDocumentalPlantillasCT,
		Ambitos: map[string]string{
			"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
			"clase_ambito":     string(ctports.AmbitoOrganizacionRRHH),
			"ambito_ref":       organizacionAltaContratacionTemporalDesarrollo,
		},
		Atributos: map[string]string{"material_sha256": strings.Repeat("a", 64)},
	}
	for _, caso := range casos {
		if rutaAccionPlantillasDocumentalCT(caso.accion) != caso.ruta ||
			accionRutaPlantillasDocumentalCT(caso.ruta) != caso.accion ||
			!recursoPlantillasDocumentalCTValido(caso.accion, base) {
			t.Fatalf("CT133 válido rechazado: %s", caso.accion)
		}
	}
	if rutaAccionPlantillasDocumentalCT("contratacion_temporal.plantillas_documentos.consultar") != "" ||
		accionRutaPlantillasDocumentalCT(plantillashttp.RutaCatalogo) != "" ||
		recursoPlantillasDocumentalCTValido("contratacion_temporal.plantillas_documentos.editar", base) {
		t.Fatal("accion de gobierno aceptada por proveedor documental")
	}
	for nombre, mutar := range map[string]func(*vecdomain.RecursoAutorizable){
		"expediente ausente":            func(r *vecdomain.RecursoAutorizable) { r.Referencia = "" },
		"catalogo en vez de expediente": func(r *vecdomain.RecursoAutorizable) { r.Referencia = plantillasapp.CatalogoID },
		"modulo ajeno":                  func(r *vecdomain.RecursoAutorizable) { r.ModuloID = "otro_modulo" },
		"tipo ajeno":                    func(r *vecdomain.RecursoAutorizable) { r.Tipo = tipoCatalogoPlantillasCT },
		"organizacion ajena":            func(r *vecdomain.RecursoAutorizable) { r.Ambitos["organizacion_ref"] = "organizacion:ajena" },
		"clase centro":                  func(r *vecdomain.RecursoAutorizable) { r.Ambitos["clase_ambito"] = "centro" },
		"ambito ajeno":                  func(r *vecdomain.RecursoAutorizable) { r.Ambitos["ambito_ref"] = "organizacion:ajena" },
		"ambito adicional":              func(r *vecdomain.RecursoAutorizable) { r.Ambitos["expediente_ref"] = r.Referencia },
		"huella no canonica":            func(r *vecdomain.RecursoAutorizable) { r.Atributos["material_sha256"] = strings.Repeat("A", 64) },
		"atributo adicional":            func(r *vecdomain.RecursoAutorizable) { r.Atributos["formato"] = "pdf" },
	} {
		t.Run(nombre, func(t *testing.T) {
			r := base
			r.Ambitos = map[string]string{
				"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
				"clase_ambito":     string(ctports.AmbitoOrganizacionRRHH),
				"ambito_ref":       organizacionAltaContratacionTemporalDesarrollo,
			}
			r.Atributos = map[string]string{"material_sha256": strings.Repeat("a", 64)}
			mutar(&r)
			if recursoPlantillasDocumentalCTValido(casos[0].accion, r) ||
				recursoPlantillasDocumentalCTValido(casos[1].accion, r) {
				t.Fatal("recurso documental ampliado aceptado")
			}
		})
	}
}

func TestProveedorDocumentalPlantillasCTExigeDependenciasNominalesYContexto(t *testing.T) {
	soporte := &soporteAltaContratacionTemporalDesarrollo{}
	motivo := motivoDocumentalPlantillasCTDesarrollo()
	pdp := &aplicacionvec.ServicioAutorizacionSolicitudLigadaV3{}
	material := &proveedorMaterialAltaContratacionTemporalDesarrollo{soporte: soporte, motivo: motivo}
	proveedor, err := nuevoProveedorDocumentalPlantillasCT(soporte, pdp, material, motivo, relojContratacionTemporalDesarrollo{})
	if err != nil || proveedor == nil {
		t.Fatalf("proveedor nominal CT133 rechazado: %v", err)
	}
	if _, err := proveedor.ResolverContextoActor(context.Background()); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatalf("sin capacidad mTLS = %v", err)
	}
	if _, err := proveedor.AutorizarCatalogoDocumental(context.Background(), vecdomain.ContextoActor{},
		"contratacion_temporal.plantillas_documentos.documental_listar", vecdomain.RecursoAutorizable{}); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatalf("sin actor y recurso = %v", err)
	}
	for nombre, otro := range map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo{
		"sin material": nil,
		"otro soporte": {soporte: &soporteAltaContratacionTemporalDesarrollo{}, motivo: motivo},
		"otro motivo":  {soporte: soporte, motivo: motivoCatalogoPlantillasCTDesarrollo()},
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := nuevoProveedorDocumentalPlantillasCT(soporte, pdp, otro, motivo,
				relojContratacionTemporalDesarrollo{}); !errors.Is(err, plantillasapp.ErrNoDisponible) {
				t.Fatalf("dependencia no nominal = %v", err)
			}
		})
	}
	if _, err := nuevoProveedorDocumentalPlantillasCT(soporte, pdp, material,
		motivoCatalogoPlantillasCTDesarrollo(), relojContratacionTemporalDesarrollo{}); !errors.Is(err, plantillasapp.ErrNoDisponible) {
		t.Fatalf("motivo de gobierno aceptado: %v", err)
	}
}

type sesionDocumentalPlantillasCTPrueba struct {
	contexto contextoSeguridadComunDesarrollo
	llamadas int
}

func (s *sesionDocumentalPlantillasCTPrueba) ResolverContexto(context.Context) (contextoSeguridadComunDesarrollo, error) {
	s.llamadas++
	return s.contexto, nil
}

func TestProveedorDocumentalPlantillasCTMismoHolderYFronteraExacta(t *testing.T) {
	reloj := relojContratacionTemporalDesarrollo{}
	ahora := reloj.Ahora()
	principal := vecdomain.Principal{ID: "rrhh:ct:desarrollo", Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa,
			"perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": strings.Repeat("a", 64)}}
	baseContexto, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	base := &soporteAltaContratacionTemporalDesarrollo{sello: &selloConsultasContratacionTemporalDesarrollo{},
		principalID: principal.ID, certificadoSHA256: principal.Attributes["certificate_sha256"],
		contexto: baseContexto, reloj: reloj}
	soporte, perfil, err := nuevoSoportePlantillasDocumentalCTDesdeBaseDesarrollo(base, ahora)
	if err != nil {
		t.Fatal(err)
	}
	soporte.legadoDisponible = true
	soporte.contextoEsperadoRegistrado = soporte.contexto.Resultado
	sesion := &sesionDocumentalPlantillasCTPrueba{contexto: contextoSeguridadComunDesarrollo{
		Vinculo: soporte.contexto.Vinculo, Resultado: soporte.contexto.Resultado}}
	soporte.sesionOperativa = sesion
	fronteras, err := nuevoCatalogoFronterasComunDesarrollo(descriptoresFronterasPlantillasDocumentalCTDesarrollo(perfil))
	if err != nil {
		t.Fatal(err)
	}
	proveedor := &proveedorDocumentalPlantillasCT{soporte: soporte,
		pdp:    &aplicacionvec.ServicioAutorizacionSolicitudLigadaV3{},
		motivo: motivoDocumentalPlantillasCTDesarrollo(), reloj: reloj}
	crearContexto := func(metodo, ruta string) context.Context {
		capacidad := capacidadConsultaContratacionTemporalDesarrollo{sello: soporte.sello, ruta: ruta,
			principal: principal, certificadoVerificadoEn: ahora, certificadoValidoHasta: ahora.Add(time.Hour),
			contextoOperacion: &contextoOperacionCTDesarrollo{}}
		ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
		return context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
			metodo: metodo, ruta: ruta, superficie: superficieInternaSeguridadComunDesarrollo, catalogo: fronteras})
	}
	for _, caso := range []struct{ ruta, accion string }{
		{plantillashttp.RutaBorradoresDisponibles, "contratacion_temporal.plantillas_documentos.documental_listar"},
		{plantillashttp.RutaBorradores, "contratacion_temporal.plantillas_documentos.documental_descargar"},
	} {
		ctx := crearContexto(http.MethodPost, caso.ruta)
		actor, err := proveedor.ResolverContextoActor(ctx)
		if err != nil || actor.Validar() != nil {
			t.Fatalf("actor documental %s: %v", caso.ruta, err)
		}
		if _, err := proveedor.contextoVigente(ctx, caso.accion); err != nil {
			t.Fatalf("contexto documental %s: %v", caso.ruta, err)
		}
		if sesion.llamadas != 1 {
			t.Fatalf("holder resolvio %d sesiones, se esperaba una", sesion.llamadas)
		}
		if _, err := proveedor.contextoVigente(ctx, "contratacion_temporal.plantillas_documentos.consultar"); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
			t.Fatalf("accion de gobierno admitida: %v", err)
		}
		sesion.llamadas = 0
	}
	if _, err := proveedor.contextoVigente(crearContexto(http.MethodGet, plantillashttp.RutaBorradores),
		"contratacion_temporal.plantillas_documentos.documental_descargar"); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatalf("GET documental admitido: %v", err)
	}
	if _, err := proveedor.contextoVigente(crearContexto(http.MethodPost, plantillashttp.RutaBorradores),
		"contratacion_temporal.plantillas_documentos.documental_listar"); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatalf("descarga convertida en listado: %v", err)
	}
}
