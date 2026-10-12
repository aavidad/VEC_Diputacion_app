package auditoria

import (
	"context"
	"errors"
	"testing"
	"time"
)

type capturaSesionPrueba struct{}

func (capturaSesionPrueba) CapturarLecturaAuditoria(context.Context, ContextoConsulta, Filtro, string, Busqueda) (CapturaLecturaSesion, error) {
	return CapturaLecturaSesion{ContextoCanonico: []byte("contexto"), VinculoCanonico: []byte("vinculo"),
		Orden: []byte(`{}`), OrganizacionRef: "organizacion:rrhh"}, nil
}

type lecturaSesionPrueba struct {
	filas     []Registro
	consultas int
}

func (l *lecturaSesionPrueba) ConsultarAuditoriaSesion(context.Context, ConsultaSesion) (PaginaFuente, error) {
	l.consultas++
	return PaginaFuente{Registros: l.filas}, nil
}
func (*lecturaSesionPrueba) BuscarObjetivos(context.Context, ConsultaSesion, Busqueda) ([]Objetivo, error) {
	return nil, nil
}
func (*lecturaSesionPrueba) RegistrarOpciones(context.Context, ConsultaSesion) error { return nil }

func TestLecturaAuditoriaExigeMotivoYNoPermiteReutilizarCursorConOtroMotivo(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	p := peticionAuditoriaIdentidadPrueba(t, ahora)
	p.Filtro.Limite = 1
	fuente := &lecturaSesionPrueba{filas: []Registro{
		{ID: "registro:dos", Fuente: "ct", ModuloID: "contratacion_temporal", Accion: "alta_o2", ActorRef: "persona:rrhh", Resultado: "confirmado", ExpedienteRef: p.Filtro.ExpedienteRef, OcurridoEn: ahora.Add(-time.Minute)},
		{ID: "registro:uno", Fuente: "ct", ModuloID: "contratacion_temporal", Accion: "alta_o2", ActorRef: "persona:rrhh", Resultado: "confirmado", ExpedienteRef: p.Filtro.ExpedienteRef, OcurridoEn: ahora.Add(-2 * time.Minute)},
	}}
	s, err := NuevoServicioLecturaSesion(capturaSesionPrueba{}, fuente)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Consultar(t.Context(), p); !errors.Is(err, ErrDenegada) || fuente.consultas != 0 {
		t.Fatalf("lectura sin motivo: %v, consultas=%d", err, fuente.consultas)
	}
	p.Filtro.MotivoConsulta = "Revisión de la tramitación"
	pagina, err := s.Consultar(t.Context(), p)
	if err != nil || len(pagina.Registros) != 1 || pagina.SiguienteCursor == "" {
		t.Fatalf("página: %+v, %v", pagina, err)
	}
	p.Cursor = pagina.SiguienteCursor
	p.Filtro.MotivoConsulta = "Revisión de otra incidencia"
	if _, err := s.Consultar(t.Context(), p); !errors.Is(err, ErrDenegada) || fuente.consultas != 1 {
		t.Fatalf("cursor con otro motivo: %v, consultas=%d", err, fuente.consultas)
	}
}

func TestCursorAuditoriaDeSesionLigaElPerfilQueConsulta(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	f := peticionAuditoriaIdentidadPrueba(t, ahora).Filtro
	f.MotivoConsulta = "Revisión administrativa"
	f.AutorLecturaRef = "persona:uno:perfil:rrhh"
	s, err := NuevoServicioLecturaSesion(capturaSesionPrueba{}, &lecturaSesionPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	cursor, err := s.codificarCursor(f, Posicion{OcurridoEn: ahora.Add(-time.Minute), Fuente: "ct", ID: "registro:uno"})
	if err != nil {
		t.Fatal(err)
	}
	f.AutorLecturaRef = "persona:uno:perfil:otro"
	if _, err := s.decodificarCursor(cursor, f); !errors.Is(err, ErrDenegada) {
		t.Fatalf("cursor de otro perfil admitido: %v", err)
	}
}
