package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type sellosReincorporacionPrueba struct{}

func (sellosReincorporacionPrueba) SellarAmbitoReincorporacionTitular(context.Context, ports.SolicitudSellarAmbitoIdempotencia) (ports.ColeccionSellosHMAC, error) {
	return ports.NuevaColeccionSellosHMAC("hmac-sha256:"+ports.DominioAmbitoReincorporacionTitular+"/v1:"+strings.Repeat("a", 64), nil)
}
func (sellosReincorporacionPrueba) DerivarHuellaReincorporacionTitular(context.Context, []byte) (ports.ColeccionSellosHMAC, error) {
	return ports.NuevaColeccionSellosHMAC("hmac-sha256:"+ports.DominioHuellaReincorporacionTitular+"/v1:"+strings.Repeat("b", 64), nil)
}

type repositorioReincorporacionPrueba struct {
	expediente                    domain.Expediente
	confirmada                    bool
	preparaciones, confirmaciones int
	recibo                        ports.ReciboReincorporacionTitular
}

func (r *repositorioReincorporacionPrueba) PrepararReincorporacionTitular(_ context.Context, m ports.MaterialReincorporacionTitular, s ports.SellosOperacionSeguimiento, refs ports.ReferenciasEfectoSeguimiento) (ports.PreparacionReincorporacionTitular, error) {
	r.preparaciones++
	a, _ := s.Ambitos.Datos()
	h, _ := s.Huellas.Datos()
	p := ports.PreparacionReincorporacionTitular{Expediente: r.expediente.Clonar(), Referencias: refs,
		AmbitoIdempotenciaHMAC: a.Activo.Valor, HuellaPeticionHMAC: h.Activo.Valor,
		CeseEventoRef: "evento:cese:prueba", CeseReciboRef: "recibo:cese:prueba", Confirmada: r.confirmada}
	if r.confirmada {
		p.Recibo = &r.recibo
	}
	return p, nil
}
func (r *repositorioReincorporacionPrueba) ConfirmarReincorporacionTitular(_ context.Context, o ports.OrdenConfirmarReincorporacionTitular) (ports.ReciboReincorporacionTitular, error) {
	r.confirmaciones++
	r.recibo = ports.ReciboReincorporacionTitular{Operacion: ports.OperacionRegistrarReincorporacionTitular,
		OrganizacionRef: o.Material.OrganizacionRef, ExpedienteRef: o.Material.ExpedienteRef, RelacionRef: o.Material.RelacionRef,
		FechaEfectiva: o.Material.FechaEfectiva.Format(time.DateOnly), CeseEventoRef: o.Preparacion.CeseEventoRef,
		CeseReciboRef: o.Preparacion.CeseReciboRef, VersionAnterior: o.Material.VersionEsperada,
		VersionResultante: o.Siguiente.Version, ReciboRef: o.Preparacion.Referencias.ReciboRef,
		AuditoriaRef: "aud_v3_" + strings.Repeat("0", 32), EventoRef: o.Preparacion.Referencias.EventoRef,
		ActorRef: o.Material.ActorRef, RegistradaEn: o.InstanteEfecto}
	r.confirmada = true
	return r.recibo, nil
}

func TestReincorporacionExigeCanalYReautorizaReplay(t *testing.T) {
	esc := nuevoEscenarioSeguimiento(t, 1)
	actual := esc.repo.expediente
	fecha := time.Date(2027, 2, 15, 0, 0, 0, 0, time.UTC)
	ref := "documento:ct:justificante:1"
	sha := strings.Repeat("b", 64)
	cesado, err := actual.RegistrarCese(actual.Version, domain.DatosCese{CausaClave: "fin_sustitucion", FechaEfecto: fecha,
		JustificanteTipo: "comunicacion_reincorporacion", JustificanteRef: ref, JustificanteSHA256: sha},
		domain.DatosActuacion{AccionClave: domain.AccionCesarNombramiento, ActorRef: "per_prueba", UnidadRef: actual.Asignacion.UnidadRef,
			ReciboRef: "recibo:cese:prueba", RealizadaEn: actual.ActualizadoEn.Add(time.Minute), FaseDestino: domain.FaseNombramiento,
			EstadoDestino: domain.EstadoEnCurso, DocumentosRef: []string{ref}})
	if err != nil {
		t.Fatal(err)
	}
	repo := &repositorioReincorporacionPrueba{expediente: cesado}
	s, err := NuevoServicioReincorporacionTitular(DependenciasReincorporacionTitular{Contextos: esc.servicio.contextos,
		Sellos: sellosReincorporacionPrueba{}, Repositorio: repo, Reglas: esc.reglas, Autorizador: esc.autorizador,
		Referencias: referenciasSeguimientoDoble{}, Reloj: esc.servicio.reloj})
	if err != nil {
		t.Fatal(err)
	}
	sol := SolicitudRegistrarReincorporacionTitular{Canal: esc.canal, ExpedienteRef: cesado.Referencia,
		RelacionRef: "relacion:personal:1", FechaEfectiva: fecha, DocumentoRef: ref, DocumentoSHA256: sha,
		VersionEsperada: cesado.Version, ClaveIdempotencia: "88888888-8888-4888-8888-888888888888"}
	if _, err = s.RegistrarReincorporacionTitular(context.Background(), sol); err != nil {
		t.Fatal(err)
	}
	if repo.confirmaciones != 1 || len(esc.autorizador.solicitudes) != 1 ||
		esc.autorizador.solicitudes[0].Audiencia != ports.AudienciaConsumoReincorporacionTitularV1 ||
		esc.autorizador.solicitudes[0].Accion != domain.AccionRegistrarReincorporacionTitular {
		t.Fatal("efecto o decisión V3 incorrectos")
	}
	if _, err = s.RegistrarReincorporacionTitular(context.Background(), sol); err != nil {
		t.Fatal(err)
	}
	if repo.confirmaciones != 1 || len(esc.autorizador.solicitudes) != 2 {
		t.Fatal("replay duplicó efecto o no reautorizó")
	}
	sol.Canal.AutenticacionRef = ""
	if _, err = s.RegistrarReincorporacionTitular(context.Background(), sol); !errors.Is(err, ErrSolicitudReincorporacionTitularInvalida) {
		t.Fatalf("canal: %v", err)
	}
}
