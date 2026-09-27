package auditoria

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/fichero"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

type relojPrueba struct{ instante time.Time }

func (r relojPrueba) Ahora() time.Time { return r.instante }

type validadorMotivoPrueba struct {
	recibido domain.ReferenciaEntradaCatalogo
}

func (v *validadorMotivoPrueba) ValidarReferenciaMotivoAutorizacionV2(_ context.Context, m domain.ReferenciaEntradaCatalogo, _ time.Time) error {
	v.recibido = m
	return m.Validar()
}

func TestOpcionesEjemploSeResuelvenDesdeCatalogoVigente(t *testing.T) {
	const ruta = "../../../data/demo/reglas/auditoria_consulta.ejemplo.demo.json"
	fuente, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		t.Fatalf("paquete de ejemplo invalido: %v", err)
	}
	ahora := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{Consulta: fuente,
		CatalogoID: "motivos_autorizacion_auditoria", ModuloID: "auditoria", Reloj: relojPrueba{ahora}})
	if err != nil {
		t.Fatalf("resolutor: %v", err)
	}
	validador := &validadorMotivoPrueba{}
	opciones, err := NuevoProveedorOpcionesCatalogo(resolutor, validador)
	if err != nil {
		t.Fatalf("proveedor: %v", err)
	}
	actuales, err := opciones.Actuales(context.Background())
	if err != nil {
		t.Fatalf("opciones: %v", err)
	}
	if !actuales.EsEjemplo || actuales.PermisoRequerido != AccionConsultar ||
		actuales.FinalidadRef != "revision_administrativa_auditoria_rrhh" ||
		actuales.MotivoRef != actuales.Motivo.Referencia() ||
		validador.recibido != actuales.Motivo {
		t.Fatalf("opciones no proceden de la referencia publicada: %+v", actuales)
	}
}

func TestFiltroExactoLigaPaginaFinalidadYMotivo(t *testing.T) {
	f := Filtro{Fuente: "ct", ExpedienteRef: "expediente:ct:1", Desde: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		Hasta: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), Limite: 50,
		FinalidadRef: "revision_administrativa_auditoria_rrhh", MotivoRef: "motivos_autorizacion_auditoria:1:motivo_a36f10964f684ec2bea55091667db47a"}
	h1, err := HuellaFiltro(f)
	if err != nil {
		t.Fatalf("filtro: %v", err)
	}
	f.ActorRef = "hmac-sha256:uno:" + strings.Repeat("a", 64)
	h2, err := HuellaFiltro(f)
	if err != nil || h1 == h2 {
		t.Fatalf("actor no ligado: %q %q %v", h1, h2, err)
	}
	f.ActorRef = ""
	f.Antes = Posicion{OcurridoEn: time.Date(2026, 9, 1, 1, 0, 0, 0, time.UTC), Fuente: "ct", ID: "evento:1"}
	h3, err := HuellaFiltro(f)
	if err != nil || h1 == h3 {
		t.Fatalf("pagina no ligada: %q %q %v", h1, h3, err)
	}
	f.ExpedienteRef = "*"
	if _, err := HuellaFiltro(f); err == nil {
		t.Fatal("comodin admitido")
	}
}

func TestProyeccionRechazaDatoPersonalFueraDeLista(t *testing.T) {
	f := Filtro{Fuente: "bolsa", ExpedienteRef: "participacion:1", Desde: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		Hasta: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), Limite: 50,
		FinalidadRef: "auditoria_rrhh", MotivoRef: "motivos_autorizacion_auditoria:1:motivo_a36f10964f684ec2bea55091667db47a"}
	r := Registro{ID: "evento:1", Fuente: "bolsa", ModuloID: "bolsa", Accion: "bolsa.participacion.cambiar",
		OcurridoEn: time.Date(2026, 9, 1, 1, 0, 0, 0, time.UTC), Resultado: "ok", ExpedienteRef: f.ExpedienteRef,
		Antes: map[string]string{"correo": "version:1"}, Despues: map[string]string{"correo": "version:2"}, DatosDisponibles: true}
	if !registroValido(r, f, "bolsa") {
		t.Fatal("version de contacto minimizada rechazada")
	}
	r.Despues["correo"] = "persona@example.org"
	if registroValido(r, f, "bolsa") {
		t.Fatal("correo claro admitido")
	}
	r.Despues = map[string]string{"dni": "00000000T"}
	if registroValido(r, f, "bolsa") {
		t.Fatal("campo personal fuera de lista admitido")
	}
}
