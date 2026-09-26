package bootstrap

import (
	"context"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestCancelacionCentroInactivaSinSelectorNiBandeja(t *testing.T) {
	if c := nuevaCancelacionCentroDesarrollo(config.Config{}, &dependenciasAltaContratacionTemporalDesarrollo{}, &incorporacionCentroDesarrollo{activo: true}); c.activa() {
		t.Fatal("sin selector el centro no cancela")
	}
	var c *cancelacionCentroDesarrollo
	if c.concesiones("solicitante_centro") != nil {
		t.Fatal("inactiva no concede nada")
	}
	if rutas, err := (&cancelacionCentroDesarrollo{}).rutas(nil); err != nil || rutas != nil {
		t.Fatalf("inactiva no monta rutas: %v %v", rutas, err)
	}
	if !rutaPeticionCentroDesarrollo(rutaCancelacionesCentro) || !rutaPeticionCentroDesarrollo(rutaCancelacionCentro) {
		t.Fatal("las rutas del centro pertenecen al circuito de peticiones")
	}
}

func TestCancelacionCentroConcedeSoloALosPerfilesDeLaRegla(t *testing.T) {
	c := &cancelacionCentroDesarrollo{piezas: &piezasCancelacionCTDesarrollo{}, roles: []string{"solicitante_centro"}}
	if got := c.concesiones("ratificador_centro"); got != nil {
		t.Fatalf("perfil fuera de la regla: %v", got)
	}
	got := c.concesiones("solicitante_centro")
	if len(got) != 2 || got[0].Accion != string(domain.AccionCancelarExpediente) || got[0].Finalidades[0] != ports.FinalidadCancelarExpediente ||
		got[1].Accion != accionConsultarCancelacionCTDesarrollo || got[0].TipoRecurso != ports.TipoRecursoCancelacion {
		t.Fatalf("concesiones: %+v", got)
	}
}

func TestRecursoCancelacionCentroLigaElCentroDelActor(t *testing.T) {
	actor := domain.ActorPeticionCentro{ActorRef: "actor:centro", PerfilRef: "perfil:centro", CentroRef: "centro-520", PuestoRef: "puesto:1"}
	recurso := vecdomain.RecursoAutorizable{Referencia: "expediente:1", ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoCancelacion,
		Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo, "expediente_ref": "expediente:1",
			"fase_previa": "solicitud", "estado_previo": string(domain.EstadoEnCurso), "centro_ref": "centro-520"},
		Atributos: map[string]string{"canal": string(domain.CanalCancelacionCentro)}}
	r := &recursoCancelacionCentroDesarrollo{accion: string(domain.AccionCancelarExpediente), finalidad: ports.FinalidadCancelarExpediente, recurso: recurso, actor: actor}
	d := vecdomain.DatosSolicitudAutorizacionLigadaV3{Accion: r.accion, Finalidad: r.finalidad, Recurso: recurso}
	if !r.validaPara("actor:centro", "perfil:centro", d) {
		t.Fatal("recurso exacto del centro rechazado")
	}
	if len(r.ambitos()) != 5 {
		t.Fatalf("ámbitos: %v", r.ambitos())
	}
	ajeno := d
	ajeno.Recurso.Ambitos = map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo, "expediente_ref": "expediente:1",
		"fase_previa": "solicitud", "estado_previo": string(domain.EstadoEnCurso), "centro_ref": "centro-999"}
	otro := *r
	otro.recurso = ajeno.Recurso
	if otro.validaPara("actor:centro", "perfil:centro", ajeno) {
		t.Fatal("expediente de otro centro admitido")
	}
	if r.validaPara("actor:otro", "perfil:centro", d) {
		t.Fatal("otro actor admitido")
	}
	rrhh := d
	rrhh.Recurso.Atributos = map[string]string{"canal": string(domain.CanalCancelacionRRHH)}
	r2 := *r
	r2.recurso = rrhh.Recurso
	if r2.validaPara("actor:centro", "perfil:centro", rrhh) {
		t.Fatal("canal de RRHH admitido en el circuito del centro")
	}
}

func TestAutoridadCancelacionCentroNiegaSinCapacidad(t *testing.T) {
	a := &autoridadCancelacionCentroDesarrollo{proveedor: &proveedorPeticionCentroDesarrollo{}, roles: []string{"solicitante_centro"}}
	if _, err := a.ResolverContextoCanalSeguimiento(context.Background()); err == nil {
		t.Fatal("sin capacidad mTLS de la ruta no hay canal")
	}
	if err := a.AutorizarLecturaSeguimiento(context.Background(), organizacionAltaContratacionTemporalDesarrollo, "expediente:1"); err == nil {
		t.Fatal("lectura sin identidad admitida")
	}
	if _, err := a.AutorizarOperacionSeguimiento(context.Background(), ports.SolicitudAutorizarOperacionSeguimiento{}); err == nil {
		t.Fatal("operación sin identidad admitida")
	}
}
