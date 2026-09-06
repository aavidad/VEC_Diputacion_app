package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestEntregaPeticionDesarrolloLigaIdentidadMaterialYAltaOriginal(t *testing.T) {
	p := vecdomain.Principal{ID: "desarrollo:rrhh", DisplayName: "RRHH sintético", Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo}, AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": strings.Repeat("a", 64)}}
	ahora := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	c, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(p, ahora)
	if err != nil {
		t.Fatal(err)
	}
	v, err := c.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	m := ports.MaterialEntregaPeticionCentro{Modo: "bandeja", ActorRef: v.PrincipalID, PerfilRef: v.PerfilActivoRef}
	r, err := postgresct.RecursoEntregaPeticionCentro(m)
	if err != nil {
		t.Fatal(err)
	}
	d := vecdomain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: c.Vinculo, ReferenciaMotivo: motivoEntregaPeticionDesarrollo(), Accion: ports.AccionConsultarPeticionesRRHH, Recurso: r, Finalidad: ports.FinalidadEntregaPeticionCentro}
	ctx := context.WithValue(context.Background(), claveMaterialEntregaPeticionDesarrollo{}, m)
	if !solicitudAutorizacionEntregaPeticionValida(ctx, d) {
		t.Fatal("consulta propia denegada")
	}
	d.Recurso.Atributos["material_sha256"] = strings.Repeat("b", 64)
	if solicitudAutorizacionEntregaPeticionValida(ctx, d) {
		t.Fatal("material alterado autorizado")
	}
	if solicitudAutorizacionEntregaPeticionValida(context.Background(), d) {
		t.Fatal("cliente sin material de servidor autorizado")
	}
	for _, rol := range []string{"solicitante_centro", "ratificador_centro", "intervencion"} {
		p.Roles = []string{rol}
		if principalContratacionTemporalDesarrolloValidoParaRuta(p, rutaEntregaPeticionCentro) {
			t.Fatal("rol ajeno obtiene recepción RRHH", rol)
		}
	}
	e := ports.EntregaPeticionCentro{EstadoEntrega: "preparada", ClaveAlta: "c60518b7-f8b4-4fe7-b17e-635c46ac2e11", AmbitoAltaHMAC: "hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:" + strings.Repeat("a", 64), ActorRef: v.PrincipalID, PerfilRef: v.PerfilActivoRef,
		Peticion: domain.DatosPeticionCentro{Referencia: "peticion:centro:001", Version: 2, Estado: "ratificada", CreadaEn: ahora, RatificadaEn: ahora.Add(time.Second), MotivoRatificacion: "Revisión sintética",
			Configuracion: domain.ConfiguracionPeticionCentro{Referencia: "config:001", Version: 1, Solicitante: domain.ActorPeticionCentro{ActorRef: "actor:solicita", PerfilRef: "perfil:centro", CentroRef: "centro-520", PuestoRef: "puesto:001"}, Ratificador: domain.ActorPeticionCentro{ActorRef: "actor:ratifica", PerfilRef: "perfil:centro", CentroRef: "centro-520", PuestoRef: "puesto:002"}},
			Solicitud:     domain.SolicitudCentro{CentroRef: "centro-520", ContactoRef: "contacto:001", CategoriaRef: categoriaAltaContratacionTemporalDesarrollo, GrupoSubgrupo: "C2", MotivoClave: "sustitucion", Detalle: "Necesidad sintética", Periodo: domain.PeriodoPrevisto{Inicio: time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC), Fin: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)}}}}
	if e.ValidarReserva() != nil {
		t.Fatal("fixture de reserva inválido")
	}
	d.Accion, d.Finalidad = ports.AccionCrearSolicitud, ports.FinalidadCrearSolicitud
	d.Recurso = vecdomain.RecursoAutorizable{Referencia: "expediente:001", ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoExpediente, Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo, "centro_ref": "centro-520", "categoria_ref": categoriaAltaContratacionTemporalDesarrollo}}
	ctx = context.WithValue(context.Background(), claveAltaDePeticionDesarrollo{}, e)
	if !solicitudAutorizacionAltaDePeticionValida(ctx, d) {
		t.Fatal("alta de original ratificado denegada")
	}
	d.Recurso.Ambitos["centro_ref"] = "centro:otro"
	if solicitudAutorizacionAltaDePeticionValida(ctx, d) {
		t.Fatal("alta de otro centro autorizada")
	}
	if solicitudAutorizacionAltaDePeticionValida(context.Background(), d) {
		t.Fatal("alta sin reserva confiable autorizada")
	}
}

type selladorEntregaPeticionPrueba struct{ clave string }

func (s *selladorEntregaPeticionPrueba) SellarAmbitoIdempotencia(_ context.Context, m ports.SolicitudSellarAmbitoIdempotencia) (ports.ColeccionSellosHMAC, error) {
	s.clave = m.ClaveIdempotencia
	return ports.NuevaColeccionSellosHMAC("hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:"+strings.Repeat("a", 64), nil)
}

func TestEntregaPeticionDesarrolloRechazaSelloDeOtraClaveAntesDeAutorizar(t *testing.T) {
	p := vecdomain.Principal{ID: "desarrollo:rrhh", DisplayName: "RRHH sintético", Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo}, AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": strings.Repeat("a", 64)}}
	ahora := relojContratacionTemporalDesarrollo{}.Ahora()
	c, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(p, ahora)
	if err != nil {
		t.Fatal(err)
	}
	v, err := c.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	sello := &selloConsultasContratacionTemporalDesarrollo{}
	hmac := &selladorEntregaPeticionPrueba{}
	soporte := &soporteAltaContratacionTemporalDesarrollo{sello: sello, principalID: p.ID, certificadoSHA256: p.Attributes["certificate_sha256"], contexto: c, ambitos: hmac}
	proveedor := &proveedorEntregaPeticionDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: soporte}}
	ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidadConsultaContratacionTemporalDesarrollo{sello: sello, ruta: rutaEntregaPeticionCentro, principal: p, certificadoVerificadoEn: ahora, certificadoValidoHasta: ahora.Add(time.Hour)})
	clave, selloCorrecto, err := proveedor.NuevaClaveAltaDePeticion(ctx)
	if err != nil || !ports.ClaveIdempotenciaValida(clave) || hmac.clave != clave {
		t.Fatal("clave y sello no ligados", err)
	}
	m := ports.MaterialEntregaPeticionCentro{Modo: "preparar", ActorRef: v.PrincipalID, PerfilRef: v.PerfilActivoRef, PeticionRef: "peticion:001", VersionEsperada: 2, ClaveAltaCandidata: clave, AmbitoAltaHMAC: strings.Replace(selloCorrecto, strings.Repeat("a", 64), strings.Repeat("b", 64), 1)}
	if _, err := proveedor.AutorizarEntregaPeticionCentro(ctx, m); err != ports.ErrAutorizacionDenegada {
		t.Fatal("sello cruzado alcanzó autorización", err)
	}
}
