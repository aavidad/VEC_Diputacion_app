package application

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type lectorIntencionSingularPrueba struct {
	expediente ports.ExpedienteParaSeleccion
	llamadas   int
}

func (l *lectorIntencionSingularPrueba) LeerExpedienteParaSeleccion(
	_ context.Context, _, _ string, _ uint64,
) (ports.ExpedienteParaSeleccion, error) {
	l.llamadas++
	return l.expediente, nil
}

func TestProyectarIntencionSingularPendienteConVersionYAmbito(t *testing.T) {
	contenido, err := os.ReadFile("../domain/testdata/expediente_nombramiento_v7.json")
	if err != nil {
		t.Fatal(err)
	}
	var e domain.Expediente
	if err := json.Unmarshal(contenido, &e); err != nil {
		t.Fatal(err)
	}
	e.Actuaciones = e.Actuaciones[:6]
	e.Version = 6
	e.FaseActual = domain.FaseFiscalizacion
	e.ActualizadoEn = e.Actuaciones[5].RealizadaEn
	e.Solicitud.NumeroPersonas = 1
	e.Solicitud.JornadaMinutos = 420
	e.Solicitud.PuestoSolicitado = "puesto:administrativo:01"
	if err := e.Validar(); err != nil {
		t.Fatal(err)
	}
	const idempotenciaOriginal = "018f47a2-6b31-4c80-8a95-4d2e707c5a11"
	q := ports.SolicitudAutorizacionConsultaEstadoSeleccionOfertaCT{
		OrganizacionRef: e.OrganizacionRef, ExpedienteRef: e.Referencia, VersionObservada: e.Version,
	}
	i := ports.IntencionSingularPendienteCT{OrganizacionRef: e.OrganizacionRef,
		ExpedienteRef: e.Referencia, VersionExpediente: e.Version,
		ClaveIdempotencia: idempotenciaOriginal, Efecto: ports.EfectoSolicitarSeleccionLlamamiento,
		VentanaAbierta: true}
	estado := ports.EstadoSeleccionOfertaPorExpedienteCT{Encontrada: true, Singular: &i}
	lector := &lectorIntencionSingularPrueba{expediente: ports.ExpedienteParaSeleccion{
		Fiscalizado: e, VersionActual: e.Version}}
	d, err := proyectarIntencionSingularPendiente(t.Context(), q, estado, lector)
	if err != nil || d.Estado != "pendiente" || d.TipoSeleccion != "llamamiento" ||
		d.ClaveIdempotencia != idempotenciaOriginal || d.VersionEsperada != e.Version ||
		d.Efecto != "solicitar_llamamiento" || !d.VentanaAbierta || lector.llamadas != 1 {
		t.Fatalf("la intención original no quedó disponible: %+v, %v", d, err)
	}

	for _, caso := range []struct {
		nombre  string
		cambiar func()
	}{
		{"organización ajena", func() { q.OrganizacionRef = "organizacion:ajena" }},
		{"versión ajena", func() { q.VersionObservada++ }},
		{"fiscalización plural", func() { lector.expediente.Fiscalizado.Solicitud.NumeroPersonas = 2 }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			q = ports.SolicitudAutorizacionConsultaEstadoSeleccionOfertaCT{
				OrganizacionRef: e.OrganizacionRef, ExpedienteRef: e.Referencia, VersionObservada: e.Version}
			lector.expediente.Fiscalizado = e.Clonar()
			caso.cambiar()
			if _, err := proyectarIntencionSingularPendiente(t.Context(), q, estado, lector); !errors.Is(err, ErrResultadoSeleccionOfertaNoConfiable) {
				t.Fatalf("se expuso intención fuera de ámbito o fiscalización: %v", err)
			}
		})
	}
}
