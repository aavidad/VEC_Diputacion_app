package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type fuenteOriginalC8Prueba struct {
	original          ports.OriginalAcreditado
	err               error
	llamadas          int
	versionSolicitada uint64
}

func (f *fuenteOriginalC8Prueba) AcreditarOriginal(_ context.Context, _ ports.OrdenJustificacion, _ domain.SolicitudJustificable, _ domain.PoliticaJustificacion, _ string, version uint64) (ports.OriginalAcreditado, error) {
	f.llamadas++
	f.versionSolicitada = version
	return f.original, f.err
}

func TestC8RechazaOriginalSinOrigenPropietarioHuellaOVersion(t *testing.T) {
	for _, caso := range []string{"fuente", "modulo", "propietario", "aportante", "huella", "version", "admision", "tipo", "expediente"} {
		t.Run(caso, func(t *testing.T) {
			j, orden, preparacion, documentos, repo, _, peticion := escenarioJustificacion(t)
			actor, err := orden.ContextoActor()
			if err != nil {
				t.Fatal(err)
			}
			f := &fuenteOriginalC8Prueba{original: ports.OriginalAcreditado{
				Documento: peticion.Documento, ModuloID: "cronos", ExpedienteRef: preparacion.p.Solicitud.ExpedienteDocumentalRef,
				TipoRef: preparacion.p.Politica.TipoDocumentalRef, EmpleadoPropietario: preparacion.p.Solicitud.EmpleadoRef,
				PersonaAportante: actor.PersonaRef, EstadoAdmision: "admitido",
			}}
			switch caso {
			case "fuente":
				f.err = ports.ErrJustificacionNoDisponible
			case "modulo":
				f.original.ModuloID = "dietas"
			case "propietario":
				f.original.EmpleadoPropietario = "emp_otra"
			case "aportante":
				f.original.PersonaAportante = "per_otra"
			case "huella":
				f.original.Documento.SHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			case "version":
				f.original.Documento.Version++
			case "admision":
				f.original.EstadoAdmision = "cuarentena"
			case "tipo":
				f.original.TipoRef = peticion.Documento.ID
			case "expediente":
				f.original.ExpedienteRef = peticion.Documento.ID
			}
			s, err := NuevoServicioJustificantesC8(j, f)
			if err != nil {
				t.Fatal(err)
			}
			_, err = s.Aportar(context.Background(), orden, ports.PeticionAporteOriginal{
				SolicitudRef: peticion.SolicitudRef, ClaveOperacion: peticion.ClaveOperacion,
				DocumentoID: peticion.Documento.ID, DocumentoVersion: peticion.Documento.Version,
				SHA256: peticion.Documento.SHA256, VersionEsperada: peticion.VersionEsperada,
			})
			if err == nil || documentos.llamadas != 0 || repo.efectos != 0 {
				t.Fatalf("C8 aceptó original no acreditado: %v, registros=%d, efectos=%d", err, documentos.llamadas, repo.efectos)
			}
		})
	}
}

func TestC8NoRevisaSiDocumentosNoAcreditaOriginal(t *testing.T) {
	j, orden, preparacion, _, repo, _, peticion := escenarioJustificacion(t)
	preparacion.p.Actual = &domain.Justificacion{Vinculo: domain.VinculoJustificacion{
		SolicitudRef: peticion.SolicitudRef, EmpleadoRef: preparacion.p.Solicitud.EmpleadoRef,
		CatalogoVersionRef: preparacion.p.Solicitud.CatalogoVersionRef, PermisoRef: preparacion.p.Solicitud.PermisoRef,
		ExpedienteDocumentalRef: preparacion.p.Solicitud.ExpedienteDocumentalRef, Documento: peticion.Documento,
	}, Version: 1, Estado: domain.JustificacionPendiente}
	f := &fuenteOriginalC8Prueba{err: ports.ErrJustificacionNoDisponible}
	s, err := NuevoServicioJustificantesC8(j, f)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Revisar(context.Background(), orden, ports.PeticionRevisionOriginal{
		SolicitudRef: peticion.SolicitudRef, ClaveOperacion: peticion.ClaveOperacion,
		VersionEsperada: 1, Decision: domain.JustificacionAceptada, MotivoRef: "motivo:documentacion:conforme",
	})
	if !errors.Is(err, ports.ErrJustificacionNoDisponible) || repo.efectos != 0 || f.llamadas != 1 {
		t.Fatalf("revisión sin original: %v, efectos=%d, lecturas=%d", err, repo.efectos, f.llamadas)
	}
}

func TestC8AporteAcreditadoConservaReciboEIdempotencia(t *testing.T) {
	j, orden, preparacion, documentos, repo, _, peticion := escenarioJustificacion(t)
	actor, err := orden.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	f := &fuenteOriginalC8Prueba{original: ports.OriginalAcreditado{
		Documento: peticion.Documento, ModuloID: "cronos", ExpedienteRef: preparacion.p.Solicitud.ExpedienteDocumentalRef,
		TipoRef: preparacion.p.Politica.TipoDocumentalRef, EmpleadoPropietario: preparacion.p.Solicitud.EmpleadoRef,
		PersonaAportante: actor.PersonaRef, EstadoAdmision: "admitido",
	}}
	s, err := NuevoServicioJustificantesC8(j, f)
	if err != nil {
		t.Fatal(err)
	}
	in := ports.PeticionAporteOriginal{SolicitudRef: peticion.SolicitudRef, ClaveOperacion: peticion.ClaveOperacion,
		VersionEsperada: peticion.VersionEsperada, DocumentoID: peticion.Documento.ID,
		DocumentoVersion: peticion.Documento.Version, SHA256: peticion.Documento.SHA256}
	primero, err := s.Aportar(context.Background(), orden, in)
	if err != nil || primero.ReciboCronos == nil || primero.EnlacePendiente || documentos.llamadas != 1 || repo.efectos != 1 || f.versionSolicitada != in.DocumentoVersion {
		t.Fatalf("primer aporte: %v, documento=%d, efecto=%d", err, documentos.llamadas, repo.efectos)
	}
	segundo, err := s.Aportar(context.Background(), orden, in)
	if err != nil || segundo.ReciboCronos == nil || !segundo.ReciboCronos.Replay ||
		segundo.ReciboCronos.ReciboRef != primero.ReciboCronos.ReciboRef || documentos.llamadas != 1 || repo.efectos != 1 {
		t.Fatalf("repetición: %v, documento=%d, efecto=%d", err, documentos.llamadas, repo.efectos)
	}
}
