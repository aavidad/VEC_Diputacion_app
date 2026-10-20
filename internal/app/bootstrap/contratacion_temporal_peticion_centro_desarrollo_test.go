package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestPuestoPeticionCentroExigeCategoriaYCentroPublicados(t *testing.T) {
	organizacion := "../../../data/catalogos/estructura-organizativa/v1.rpt-publica.json"
	rpt := "../../../data/catalogos/rpt/v1.rpt-2026.json"
	opciones, err := nuevoCatalogoDesarrollo(organizacion, rpt)
	if err != nil {
		t.Fatal(err)
	}
	lector, err := fichero.NuevaConsultaCatalogos(organizacion)
	if err != nil {
		t.Fatal(err)
	}
	proveedor := &proveedorPeticionCentroDesarrollo{catalogoAlta: opciones, catalogos: lector,
		version: 1}
	actor := domain.ActorPeticionCentro{ActorRef: "actor:centro:520", PerfilRef: "perfil:centro:520",
		CentroRef: "centro-520", PuestoRef: "rpt-520-735"}
	solicitud := domain.SolicitudCentro{CentroRef: actor.CentroRef, CategoriaRef: "categoria:rpt:administrativo"}
	puesto := ports.PuestoRPTAltaVerificado{ExisteEnPublicacion: true, PuestoCodigo: "9",
		CategoriaClave: "administrativo", CentroCodigo: "520"}
	if err := proveedor.ValidarPuestoRPTPeticionCentro(t.Context(), actor, solicitud, puesto); err != nil {
		t.Fatalf("código 9 publicado para centro 520/administrativo: %v", err)
	}
	cruzado := puesto
	cruzado.CentroCodigo = "101"
	if err := proveedor.ValidarPuestoRPTPeticionCentro(t.Context(), actor, solicitud, cruzado); err == nil {
		t.Fatal("puesto de otro centro admitido")
	}
	cruzado = puesto
	cruzado.CategoriaClave = "auxiliar"
	if err := proveedor.ValidarPuestoRPTPeticionCentro(t.Context(), actor, solicitud, cruzado); err == nil {
		t.Fatal("puesto de otra categoría admitido")
	}
}

func TestPeticionCentroDesarrolloSeparaRolesYMaterial(t *testing.T) {
	p := vecdomain.Principal{ID: "desarrollo:solicitante-centro", DisplayName: "Solicitante sintético", Roles: []string{"solicitante_centro"}, AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": strings.Repeat("a", 64)}}
	for _, rol := range []string{"solicitante_centro", "ratificador_centro", "tecnico_rrhh", "intervencion"} {
		p.Roles = []string{rol}
		centro := rol == "solicitante_centro" || rol == "ratificador_centro"
		for _, ruta := range []string{rutaOperacionesPeticionCentro, rutaBandejaPeticionCentro, rutaContextoPeticionCentro} {
			if principalContratacionTemporalDesarrolloValidoParaRuta(p, ruta) != centro {
				t.Fatalf("cruce de rol %s en %s", rol, ruta)
			}
		}
		if centro && principalContratacionTemporalDesarrolloValidoParaRuta(p, httpinterno.RutaAltaSolicitudes) {
			t.Fatal("centro obtiene alta RRHH")
		}
	}
	p.Roles = []string{"solicitante_centro"}
	ca, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(p, time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	v, err := ca.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	q := ports.ConsultaPeticionCentro{Modo: "bandeja", Referencia: "centro-520", Actor: domain.ActorPeticionCentro{ActorRef: v.PrincipalID, PerfilRef: v.PerfilActivoRef, CentroRef: "centro-520", PuestoRef: "rpt-520-735"}}
	r, err := postgresct.RecursoConsultaPeticionCentro(q, organizacionAltaContratacionTemporalDesarrollo)
	if err != nil {
		t.Fatal(err)
	}
	d := vecdomain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: ca.Vinculo, ReferenciaMotivo: motivoPeticionCentroDesarrollo(), Accion: ports.AccionConsultarPeticionCentro, Recurso: r, Finalidad: finalidadPeticionCentro}
	ctx := context.WithValue(context.Background(), claveMaterialPeticionCentroDesarrollo{}, materialAutorizacionPeticionCentroDesarrollo{consulta: &q})
	if !solicitudAutorizacionPeticionCentroDesarrolloValida(ctx, d) {
		t.Fatal("material propio rechazado")
	}
	q.Actor.PuestoRef = "puesto:otro"
	if solicitudAutorizacionPeticionCentroDesarrolloValida(ctx, d) {
		t.Fatal("autorización usada para otro puesto")
	}
}

func TestPeticionCentroDesarrolloAdscripcionVigenteParaEtiquetas(t *testing.T) {
	a := domain.ActorPeticionCentro{CentroRef: "centro-520", PuestoRef: "puesto:001"}
	c := vecdomain.CatalogoConfigurable{Entradas: []vecdomain.EntradaCatalogoConfigurable{
		{Clave: "centro-520", Atributos: map[string]string{"tipo": "centro"}},
		{Clave: "centro-otro", Atributos: map[string]string{"tipo": "centro"}},
		{Clave: "puesto:001", Atributos: map[string]string{"tipo": "puesto_responsabilidad", "adscripcion_clave": "centro-520"}},
	}}
	if !actorPeticionCentroPerteneceCatalogo(c, a) {
		t.Fatal("puesto propio omitido")
	}
	c.Entradas[2].Atributos["adscripcion_clave"] = "centro-otro"
	if actorPeticionCentroPerteneceCatalogo(c, a) {
		t.Fatal("etiqueta de un puesto trasladado sigue visible")
	}
}
