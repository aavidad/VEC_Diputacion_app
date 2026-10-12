package auditoria

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

type capturaSesionPrueba struct{ err error }

func (c capturaSesionPrueba) CapturarLecturaAuditoria(context.Context, ContextoConsulta, Filtro, string, Busqueda) (CapturaLecturaSesion, error) {
	if c.err != nil {
		return CapturaLecturaSesion{}, c.err
	}
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

func TestOpcionesAuditoriaDistinguenDenegacionDeIndisponibilidad(t *testing.T) {
	for _, caso := range []struct {
		nombre    string
		err       error
		codigo    int
		resultado domain.ResultadoIntentoAuditoria
	}{
		{"permiso", fmt.Errorf("permiso propio: %w", ErrDenegada), http.StatusForbidden, domain.ResultadoIntentoAuditoriaDenegado},
		{"dependencia", ErrNoDisponible, http.StatusServiceUnavailable, domain.ResultadoIntentoAuditoriaError},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			p := peticionAuditoriaIdentidadPrueba(t, time.Now().UTC().Truncate(time.Microsecond))
			s, err := NuevoServicioLecturaSesion(capturaSesionPrueba{err: caso.err}, &lecturaSesionPrueba{})
			if err != nil {
				t.Fatal(err)
			}
			r := &registradorIntentosConsultaPrueba{}
			if err := s.ConfigurarIntentos(ConfiguracionIntentos{Registrador: r, Proceso: "vec_server_ensayo", Canal: string(domain.SuperficieAutenticacionInternaCorporativaV1), Finalidad: p.Filtro.FinalidadRef, Motivo: p.Contexto.Motivo}); err != nil {
				t.Fatal(err)
			}
			o := &opcionesIntentosPrueba{valor: Opciones{FinalidadRef: p.Filtro.FinalidadRef, MotivoRef: p.Filtro.MotivoRef, Motivo: p.Contexto.Motivo, PermisoRequerido: AccionConsultar}}
			i := IdentidadResuelta{Vinculo: p.Contexto.Vinculo, Resultado: p.Contexto.Resultado, Correlacion: p.Contexto.Correlacion}
			h, err := NuevoManejador(s, o, identidadAuditoriaHTTPPrueba{i})
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaOpciones, nil))
			if w.Code != caso.codigo || len(r.ordenes) != 1 || w.Header().Get("X-Audit-Ref") == "" {
				t.Fatalf("estado=%d, esperado=%d, intentos=%d", w.Code, caso.codigo, len(r.ordenes))
			}
			datos, err := r.ordenes[0].Datos()
			if err != nil || datos.Datos.Resultado != caso.resultado {
				t.Fatalf("resultado=%v, esperado=%v, error=%v", datos.Datos.Resultado, caso.resultado, err)
			}
			w = httptest.NewRecorder()
			peticion := httptest.NewRequest(http.MethodPost, RutaConsulta, strings.NewReader(`{"fuente":"ct","operacion":"invalida"}`))
			peticion.Header.Set("Content-Type", "application/json")
			h.ServeHTTP(w, peticion)
			if w.Code != http.StatusBadRequest || len(r.ordenes) != 2 || w.Header().Get("X-Audit-Ref") == "" {
				t.Fatalf("operación inválida sin asiento: estado=%d, intentos=%d", w.Code, len(r.ordenes))
			}
		})
	}
}
