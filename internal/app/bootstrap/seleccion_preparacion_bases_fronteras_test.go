package bootstrap

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	core "vec-diputacion-granada/internal/vec/domain"
)

func perfilesPreparacionBasesPrueba(t *testing.T) [2]*perfilPreparacionBasesV3 {
	t.Helper()
	base, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	var ps [2]*perfilPreparacionBasesV3
	for i := range ps {
		p, err := nuevoPerfilPreparacionBasesV3(base, configuracionPreparacionBasesPrueba(), i == 0, time.Now().UTC().Truncate(time.Microsecond))
		if err != nil {
			t.Fatal(err)
		}
		ps[i] = p
	}
	return ps
}

func TestPreparacionBasesPerfilesNominalesNoSumanCampos(t *testing.T) {
	ps := perfilesPreparacionBasesPrueba(t)
	if ps[0].perfilRef() == ps[1].perfilRef() || ps[0].soporte == ps[1].soporte ||
		ps[0].soporte.contexto.Resultado.Contexto.PersonaRef != ps[1].soporte.contexto.Resultado.Contexto.PersonaRef {
		t.Fatal("perfiles no segregados de la misma persona")
	}
	for i, p := range ps {
		if p.plantilla.Validar() != nil || len(p.plantilla.VersionRol.Concesiones) != 1 || len(p.plantilla.AsignacionPerfil.Ambitos) != 2 {
			t.Fatal("semilla no nominal")
		}
		concesion := p.plantilla.VersionRol.Concesiones[0]
		campos := []string{"material_preparacion", "recibo_preparacion"}
		if i == 0 {
			campos = []string{"auditoria", "evento_outbox", "historia", "material_preparacion", "recibo_preparacion"}
		}
		if concesion.Accion != p.accion || !slices.Equal(campos, concesion.CamposPermitidos) || p.plantilla.VersionRol.RolID != []string{"guardar_preparacion_bases_bolsa", "consultar_preparacion_bases_bolsa"}[i] {
			t.Fatal("campos o rol fuera del contrato nominal")
		}
		tipo := core.RecursoAutorizable{Referencia: "preparacion:prueba", ModuloID: "bolsa", Tipo: "preparacion_bases",
			Ambitos: map[string]string{"organizacion_ref": p.ambito.OrganizacionRef(), "unidad_gestion_ref": p.ambito.UnidadGestionRef()}, Atributos: map[string]string{"material_sha256": strings.Repeat("a", 64)}}
		c, err := core.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), seguridad.GeneradorReferenciasCriptograficas{})
		if err != nil {
			t.Fatal(err)
		}
		s, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: p.soporte.contexto.Vinculo,
			ReferenciaMotivo: p.motivo, Accion: p.accion, Recurso: tipo, Finalidad: "preparacion_bases", Correlacion: c})
		if err != nil {
			t.Fatal(err)
		}
		ahora := time.Now().UTC().Truncate(time.Microsecond)
		e, err := core.NuevaEvidenciaEvaluacionAutorizacionV3(s, p.plantilla, "decision:s2-prueba", ahora, ahora.Add(time.Second))
		if err != nil {
			t.Fatal(err)
		}
		d, err := core.NuevaDecisionAutorizacionLigadaV3(s, e)
		if err != nil || d.ExigirProyeccionPara(s, campos, nil) != nil {
			t.Fatal("PDP no emite campos nominales")
		}
		if d.ExigirProyeccionPara(s, []string{"material_preparacion"}, nil) == nil {
			t.Fatal("PDP aceptó proyección parcial")
		}
	}
}

func TestPreparacionBasesFronterasNoCruzanPerfilNiCatalogo(t *testing.T) {
	ps := perfilesPreparacionBasesPrueba(t)
	ds, err := fronterasPreparacionBasesHTTPV3(ps[0].perfilRef(), ps[1].perfilRef())
	if err != nil {
		t.Fatal(err)
	}
	cat, err := nuevoCatalogoFronterasComunDesarrollo(ds)
	if err != nil {
		t.Fatal(err)
	}
	for i, p := range ps {
		sesion := &proveedorSesionConsultaRRHHDesarrollo{soporte: p.soporte, fronteras: cat, base: p.soporte.contexto.Resultado}
		principal := core.Principal{ID: p.soporte.principalID, Roles: []string{"tecnico_rrhh"}, AuthMethod: core.AuthMethodCertificate, AuthAssurance: core.AuthAssuranceHigh,
			Attributes: map[string]string{"certificate_sha256": p.soporte.certificadoSHA256}}
		capacidad := capacidadConsultaContratacionTemporalDesarrollo{sello: p.soporte.sello, ruta: p.ruta, metodo: http.MethodPost, principal: principal}
		f := fronteraSeguridadComunDesarrollo{metodo: http.MethodPost, ruta: p.ruta, superficie: superficieInternaSeguridadComunDesarrollo, catalogo: cat, descriptor: ds[i]}
		crear := func(c capacidadConsultaContratacionTemporalDesarrollo, f fronteraSeguridadComunDesarrollo) context.Context {
			ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
			return context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, f)
		}
		if !sesion.sesionPreparacionBasesHTTPV3(crear(capacidad, f), p.ruta) {
			t.Fatal("frontera legítima rechazada")
		}
		otro, _ := nuevoCatalogoFronterasComunDesarrollo(ds)
		cruzada := f
		cruzada.catalogo = otro
		if sesion.sesionPreparacionBasesHTTPV3(crear(capacidad, cruzada), p.ruta) {
			t.Fatal("otro catálogo aceptado")
		}
		base := sesion.base
		sesion.base = ps[1-i].soporte.contexto.Resultado
		if sesion.sesionPreparacionBasesHTTPV3(crear(capacidad, f), p.ruta) {
			t.Fatal("perfil cruzado aceptado")
		}
		sesion.base = base
		descriptor := cat.porClave[ds[i].Clave]
		modificado := descriptor
		modificado.ClavePolitica = "politica-ajena"
		cat.porClave[ds[i].Clave] = modificado
		if sesion.sesionPreparacionBasesHTTPV3(crear(capacidad, f), p.ruta) {
			t.Fatal("política cruzada aceptada")
		}
		cat.porClave[ds[i].Clave] = descriptor
		capacidad.metodo = http.MethodGet
		if sesion.sesionPreparacionBasesHTTPV3(crear(capacidad, f), p.ruta) {
			t.Fatal("otro método aceptado")
		}
	}
	if _, err := fronterasPreparacionBasesHTTPV3(ps[0].perfilRef(), ps[0].perfilRef()); err == nil {
		t.Fatal("unión de perfiles aceptada")
	}
}

func TestPreparacionBasesAudienciasCerradasYMaterialSeparado(t *testing.T) {
	ds := DescriptoresMaterialPreparacionBasesV3()
	if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(ds[:]); err != nil {
		t.Fatal(err)
	}
	base := materialRenovableCTPrueba(t, time.Now().UTC().Truncate(time.Microsecond))
	guardar, err := derivarMaterialConsumidorV3Desarrollo(base, ds[0])
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(guardar.claveHMAC)
	consultar, err := derivarMaterialConsumidorV3Desarrollo(base, ds[1])
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(consultar.claveHMAC)
	if slices.Equal(guardar.claveHMAC, consultar.claveHMAC) || guardar.claveHMACID == consultar.claveHMACID {
		t.Fatal("material compartido entre operaciones")
	}
	for _, d := range ds {
		if !slices.Contains(audienciasConsumoGobiernoCTDesarrollo(), d.Audiencia) || !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(d.Audiencia) {
			t.Fatal("audiencia propia rechazada")
		}
		for _, ajena := range []string{d.Audiencia + ".otra", strings.TrimSuffix(d.Audiencia, "v1") + "v2", "vec_bolsa_convocatorias.preparacion_bases.*"} {
			if audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(ajena) {
				t.Fatal("audiencia no nominal aceptada")
			}
		}
	}
}

func TestPreparacionBasesProvisionNoRestauraRevocacion(t *testing.T) {
	p := perfilesPreparacionBasesPrueba(t)[0]
	a := instantaneaPublicadaDesarrollo{instantanea: clonarInstantaneaAutorizacionPostgreSQLDesarrollo(p.plantilla), actoAsignacion: p.actoAsignacion, actoControl: p.actoControl,
		actualizadaPor: p.plantilla.AsignacionPerfil.EmitidaPor}
	h, err := a.instantanea.AsignacionPerfil.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	p.provision = provisionPreparacionBasesV3{AprobacionRef: "aprobacion:s2-prueba", PreimagenSHA256: h}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	if !preimagenPreparacionBasesProvisionable(a, p, ahora) {
		t.Fatal("preimagen propia rechazada")
	}
	a.instantanea.ControlVigenciaVersionRol.Estado = "retirada"
	if preimagenPreparacionBasesProvisionable(a, p, ahora) {
		t.Fatal("rol retirado restaurable")
	}
	a.instantanea = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(p.plantilla)
	a.instantanea.VersionRol.Concesiones[0].CamposPermitidos = []string{"material_preparacion"}
	if preimagenPreparacionBasesProvisionable(a, p, ahora) {
		t.Fatal("campos recortados restaurables")
	}
}
