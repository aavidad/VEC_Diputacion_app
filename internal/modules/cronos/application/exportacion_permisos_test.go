package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type fuenteInformePermisosPrueba struct {
	datos    ports.PermisosExportables
	err      error
	llamadas int
	alLeer   func()
}

func (f *fuenteInformePermisosPrueba) LeerResumenPropioParaExportar(context.Context, ports.OrdenExportacionPermisos) (ports.PermisosExportables, error) {
	f.llamadas++
	if f.alLeer != nil {
		f.alLeer()
	}
	return f.datos, f.err
}

type preparadorInformePermisosPrueba struct {
	recibido   ports.ResumenPermisosInforme
	llamadas   int
	err        error
	alPreparar func()
}

func (p *preparadorInformePermisosPrueba) PrepararInformePermisos(_ context.Context, r ports.ResumenPermisosInforme) (ports.DocumentoPermisosPreparado, error) {
	p.llamadas++
	p.recibido = r
	if p.alPreparar != nil {
		p.alPreparar()
	}
	return ports.DocumentoPermisosPreparado{Contenido: []byte("%PDF-prueba"), CatalogoRef: "catalogo_ejemplo", CatalogoVersion: 1, CatalogoSHA256: strings.Repeat("b", 64)}, p.err
}

type registroInformePermisosPrueba struct {
	reloj       *relojExportacionPrueba
	llamadas    int
	err         error
	cambiar     func(*ports.ConfirmacionExportacionPermisos)
	alConfirmar func()
}

func (r *registroInformePermisosPrueba) ConfirmarExportacionPermisos(_ context.Context, _ ports.OrdenExportacionPermisos, e ports.EvidenciaExportacionPermisos) (ports.ConfirmacionExportacionPermisos, error) {
	r.llamadas++
	if r.alConfirmar != nil {
		r.alConfirmar()
	}
	c := ports.ConfirmacionExportacionPermisos{Evidencia: e, ReciboRef: "recibo_ejemplo", AuditoriaRef: "auditoria_ejemplo", ConfirmadaUTC: r.reloj.t}
	if r.cambiar != nil {
		r.cambiar(&c)
	}
	return c, r.err
}
func prepararInformePermisosPrueba(t *testing.T) (*ServicioExportacionPermisos, ports.OrdenExportacionPermisos, *fuenteInformePermisosPrueba, *preparadorInformePermisosPrueba, *registroInformePermisosPrueba) {
	t.Helper()
	actor, err := contexto(t).OrdenConsumo.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenExportacionPermisos(actor, 2026)
	if err != nil {
		t.Fatal(err)
	}
	refs, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil {
		t.Fatal(err)
	}
	reloj := &relojExportacionPrueba{time.Now().UTC().Truncate(time.Microsecond)}
	concedido := int64(3)
	fuente := &fuenteInformePermisosPrueba{datos: ports.PermisosExportables{EmpleadoRef: refs[0], Ejercicio: 2026, CorteUTC: reloj.t, FuenteRef: "fuente_ejemplo", FuenteVersion: 1, FuenteSHA256: strings.Repeat("a", 64), Politica: ports.PoliticaExportacionPermisos{Referencia: "politica_ejemplo", Version: 1, SHA256: strings.Repeat("c", 64), TiposPermitidos: []string{"tipo_ejemplo"}, CamposPermitidos: []string{"etiqueta", "unidad", "computo", "pendiente_resolver", "concedido", "restante", "conciliacion"}}, Filas: []ports.FilaPermisoExportable{{TipoRef: "tipo_ejemplo", Resumen: ports.FilaInformePermisos{Etiqueta: "Permiso de ejemplo", Unidad: domain.LeaveUnitDay, Computo: domain.ComputoLaborables, Concedido: &concedido, Conciliacion: ports.ConciliacionPermisosPendiente}}}}}
	p := &preparadorInformePermisosPrueba{}
	r := &registroInformePermisosPrueba{reloj: reloj}
	s, err := NuevaExportacionPermisos(fuente, p, r, reloj)
	if err != nil {
		t.Fatal(err)
	}
	return s, orden, fuente, p, r
}
func TestExportacionPermisosMinimizaYConfirmaDocumentoExacto(t *testing.T) {
	s, o, f, p, r := prepararInformePermisosPrueba(t)
	// La fuente ya ha excluido el segundo tipo. El renderer sólo ve la fila mínima;
	// nunca conoce el número de tipos permitidos, sus referencias ni las exclusiones.
	f.datos.Politica.TiposPermitidos = append(f.datos.Politica.TiposPermitidos, "tipo_excluido_del_informe")
	p.alPreparar = func() { *f.datos.Filas[0].Resumen.Concedido = 99 }
	resultado, err := s.ExportarPermisosPropios(context.Background(), o)
	if err != nil || len(resultado.Contenido) == 0 || p.llamadas != 1 || r.llamadas != 1 {
		t.Fatal(err, p.llamadas, r.llamadas)
	}
	if len(p.recibido.Filas) != 1 || p.recibido.Filas[0].Restante != nil || *p.recibido.Filas[0].Concedido != 3 {
		t.Fatal("proyección no mínima o sin copia")
	}
	e := resultado.Confirmacion.Evidencia
	if e.Accion != ports.AccionExportarPermisosPDF || e.PoliticaSHA256 != f.datos.Politica.SHA256 || e.FuenteSHA256 != f.datos.FuenteSHA256 || e.CorteUTC != f.datos.CorteUTC || e.Tamano != int64(len(resultado.Contenido)) || len(e.DocumentoSHA256) != 64 {
		t.Fatal("confirmación incompleta")
	}
}
func TestExportacionPermisosDenegacionesSinBytes(t *testing.T) {
	fallo := errors.New("fallo_ejemplo")
	casos := []struct {
		nombre  string
		alterar func(*ServicioExportacionPermisos, *fuenteInformePermisosPrueba, *preparadorInformePermisosPrueba, *registroInformePermisosPrueba)
	}{
		{"fuente_denegada", func(_ *ServicioExportacionPermisos, f *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, _ *registroInformePermisosPrueba) {
			f.err = fallo
		}},
		{"otra_persona", func(_ *ServicioExportacionPermisos, f *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, _ *registroInformePermisosPrueba) {
			f.datos.EmpleadoRef = "otra_persona"
		}},
		{"otro_ejercicio", func(_ *ServicioExportacionPermisos, f *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, _ *registroInformePermisosPrueba) {
			f.datos.Ejercicio++
		}},
		{"corte_futuro", func(_ *ServicioExportacionPermisos, f *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, _ *registroInformePermisosPrueba) {
			f.datos.CorteUTC = f.datos.CorteUTC.Add(time.Second)
		}},
		{"sin_politica", func(_ *ServicioExportacionPermisos, f *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, _ *registroInformePermisosPrueba) {
			f.datos.Politica = ports.PoliticaExportacionPermisos{}
		}},
		{"tipo_excluido", func(_ *ServicioExportacionPermisos, f *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, _ *registroInformePermisosPrueba) {
			f.datos.Filas[0].TipoRef = "tipo_no_permitido"
		}},
		{"tipo_ausente", func(_ *ServicioExportacionPermisos, f *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, _ *registroInformePermisosPrueba) {
			f.datos.Filas[0].TipoRef = ""
		}},
		{"fila_duplicada", func(_ *ServicioExportacionPermisos, f *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, _ *registroInformePermisosPrueba) {
			f.datos.Filas = append(f.datos.Filas, f.datos.Filas[0])
		}},
		{"politica_tipo_duplicado", func(_ *ServicioExportacionPermisos, f *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, _ *registroInformePermisosPrueba) {
			f.datos.Politica.TiposPermitidos = append(f.datos.Politica.TiposPermitidos, "tipo_ejemplo")
		}},
		{"sin_campos", func(_ *ServicioExportacionPermisos, f *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, _ *registroInformePermisosPrueba) {
			f.datos.Politica.CamposPermitidos = nil
		}},
		{"cantidad_sin_unidad", func(_ *ServicioExportacionPermisos, f *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, _ *registroInformePermisosPrueba) {
			f.datos.Politica.CamposPermitidos = []string{"etiqueta", "concedido"}
		}},
		{"campo_duplicado", func(_ *ServicioExportacionPermisos, f *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, _ *registroInformePermisosPrueba) {
			f.datos.Politica.CamposPermitidos = []string{"etiqueta", "etiqueta"}
		}},
		{"campo_sensible", func(_ *ServicioExportacionPermisos, f *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, _ *registroInformePermisosPrueba) {
			f.datos.Politica.CamposPermitidos[0] = "motivo"
		}},
		{"renderer_falla", func(_ *ServicioExportacionPermisos, _ *fuenteInformePermisosPrueba, p *preparadorInformePermisosPrueba, _ *registroInformePermisosPrueba) {
			p.err = fallo
		}},
		{"registro_falla", func(_ *ServicioExportacionPermisos, _ *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, r *registroInformePermisosPrueba) {
			r.err = fallo
		}},
		{"otro_pdf", func(_ *ServicioExportacionPermisos, _ *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, r *registroInformePermisosPrueba) {
			r.cambiar = func(c *ports.ConfirmacionExportacionPermisos) { c.Evidencia.DocumentoSHA256 = strings.Repeat("f", 64) }
		}},
		{"otra_politica", func(_ *ServicioExportacionPermisos, _ *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, r *registroInformePermisosPrueba) {
			r.cambiar = func(c *ports.ConfirmacionExportacionPermisos) { c.Evidencia.PoliticaVersion++ }
		}},
		{"otra_fuente", func(_ *ServicioExportacionPermisos, _ *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, r *registroInformePermisosPrueba) {
			r.cambiar = func(c *ports.ConfirmacionExportacionPermisos) { c.Evidencia.FuenteSHA256 = strings.Repeat("f", 64) }
		}},
		{"sin_auditoria", func(_ *ServicioExportacionPermisos, _ *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, r *registroInformePermisosPrueba) {
			r.cambiar = func(c *ports.ConfirmacionExportacionPermisos) { c.AuditoriaRef = "" }
		}},
		{"recibo_anterior", func(_ *ServicioExportacionPermisos, _ *fuenteInformePermisosPrueba, _ *preparadorInformePermisosPrueba, r *registroInformePermisosPrueba) {
			r.cambiar = func(c *ports.ConfirmacionExportacionPermisos) { c.ConfirmadaUTC = c.ConfirmadaUTC.Add(-time.Second) }
		}},
		{"vigencia_agotada", func(_ *ServicioExportacionPermisos, _ *fuenteInformePermisosPrueba, p *preparadorInformePermisosPrueba, r *registroInformePermisosPrueba) {
			p.alPreparar = func() { r.reloj.t = r.reloj.t.Add(24 * time.Hour) }
		}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			s, o, f, p, r := prepararInformePermisosPrueba(t)
			c.alterar(s, f, p, r)
			resultado, err := s.ExportarPermisosPropios(context.Background(), o)
			if err == nil || len(resultado.Contenido) != 0 {
				t.Fatal("fallo abierto", err, len(resultado.Contenido))
			}
			if c.nombre == "tipo_excluido" && (p.llamadas != 0 || r.llamadas != 0) {
				t.Fatal("tipo excluido dejó huella en el informe")
			}
		})
	}
}

func TestExportacionPermisosSubconjuntoBorraCamposYConservaPolitica(t *testing.T) {
	s, o, f, p, r := prepararInformePermisosPrueba(t)
	f.datos.Politica.CamposPermitidos = []string{"etiqueta", "unidad", "concedido"}
	original := append([]string(nil), f.datos.Politica.CamposPermitidos...)
	p.alPreparar = func() { f.datos.Politica.CamposPermitidos[0] = "conciliacion" }
	resultado, err := s.ExportarPermisosPropios(context.Background(), o)
	if err != nil || r.llamadas != 1 || len(resultado.Contenido) == 0 {
		t.Fatal(err)
	}
	got := p.recibido
	if len(got.CamposPermitidos) != 3 || got.CamposPermitidos[0] != original[0] ||
		got.Filas[0].Etiqueta != "Permiso de ejemplo" || got.Filas[0].Unidad != domain.LeaveUnitDay || *got.Filas[0].Concedido != 3 ||
		got.Filas[0].Computo != "" || got.Filas[0].PendienteResolver != nil || got.Filas[0].Restante != nil || got.Filas[0].Conciliacion != "" {
		t.Fatal("el preparador recibió un campo excluido o lista mutable", got)
	}
	e := resultado.Confirmacion.Evidencia
	if e.PoliticaRef != "politica_ejemplo" || e.PoliticaVersion != 1 || e.PoliticaSHA256 != strings.Repeat("c", 64) || e.FuenteRef != "fuente_ejemplo" || e.FuenteSHA256 != strings.Repeat("a", 64) {
		t.Fatal("origen y política no conservados", e)
	}
}

func TestExportacionPermisosAdmiteFuenteYaMinimizada(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		campos []string
		fila   ports.FilaInformePermisos
		valido bool
	}{
		{"solo_etiqueta", []string{"etiqueta"}, ports.FilaInformePermisos{Etiqueta: "Permiso de ejemplo"}, true},
		{"solo_computo", []string{"computo"}, ports.FilaInformePermisos{Computo: domain.ComputoLaborables}, true},
		{"restante_con_metadato_interno", []string{"unidad", "restante"}, ports.FilaInformePermisos{Unidad: domain.LeaveUnitDay, Restante: cantidadPermisosPrueba(4), Conciliacion: ports.ConciliacionPermisosConfirmada}, true},
		{"restante_sin_conciliacion", []string{"unidad", "restante"}, ports.FilaInformePermisos{Unidad: domain.LeaveUnitDay, Restante: cantidadPermisosPrueba(4)}, false},
		{"restante_no_conciliado", []string{"unidad", "restante"}, ports.FilaInformePermisos{Unidad: domain.LeaveUnitDay, Restante: cantidadPermisosPrueba(4), Conciliacion: ports.ConciliacionPermisosPendiente}, false},
		{"restante_desconocido_sin_metadato", []string{"unidad", "restante"}, ports.FilaInformePermisos{Unidad: domain.LeaveUnitDay}, true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			s, o, fuente, preparador, registro := prepararInformePermisosPrueba(t)
			fuente.datos.Politica.CamposPermitidos = caso.campos
			fuente.datos.Filas[0].Resumen = caso.fila
			resultado, err := s.ExportarPermisosPropios(context.Background(), o)
			if caso.valido {
				if err != nil || preparador.llamadas != 1 || registro.llamadas != 1 || len(resultado.Contenido) == 0 {
					t.Fatal("fuente filtrada rechazada", err)
				}
				if !contieneCampoPermisos(caso.campos, "conciliacion") && preparador.recibido.Filas[0].Conciliacion != "" {
					t.Fatal("metadato interno llegó al preparador")
				}
			} else if err == nil || preparador.llamadas != 0 || registro.llamadas != 0 || len(resultado.Contenido) != 0 {
				t.Fatal("fuente inconsistente aceptada", err)
			}
		})
	}
}

func cantidadPermisosPrueba(v int64) *int64 { return &v }
func contieneCampoPermisos(campos []string, buscado string) bool {
	for _, campo := range campos {
		if campo == buscado {
			return true
		}
	}
	return false
}

func TestExportacionPermisosValidaOriginalAntesDeMinimizar(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		alterar func(*ports.FilaInformePermisos)
	}{
		{"estado_excluido_invalido", func(f *ports.FilaInformePermisos) { f.Conciliacion = "estado_inventado" }},
		{"cantidad_excluida_negativa", func(f *ports.FilaInformePermisos) { n := int64(-1); f.PendienteResolver = &n }},
		{"restante_incompatible_excluido", func(f *ports.FilaInformePermisos) { n := int64(2); f.Restante = &n }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			s, o, fuente, p, r := prepararInformePermisosPrueba(t)
			fuente.datos.Politica.CamposPermitidos = []string{"etiqueta"}
			caso.alterar(&fuente.datos.Filas[0].Resumen)
			resultado, err := s.ExportarPermisosPropios(context.Background(), o)
			if err == nil || len(resultado.Contenido) != 0 || p.llamadas != 0 || r.llamadas != 0 {
				t.Fatal("dato original inválido pasó por proyección", err)
			}
		})
	}
}
func TestExportacionPermisosSinAutoridadesNiOrden(t *testing.T) {
	s, o, f, p, r := prepararInformePermisosPrueba(t)
	var fuente *fuenteInformePermisosPrueba
	if _, err := NuevaExportacionPermisos(fuente, p, r, s.reloj); err == nil {
		t.Fatal("fuente nula admitida")
	}
	if _, err := NuevaExportacionPermisos(f, p, nil, s.reloj); err == nil {
		t.Fatal("sin registro admitido")
	}
	for _, orden := range []ports.OrdenExportacionPermisos{{}, o} {
		if orden.Ejercicio() != 0 {
			s = &ServicioExportacionPermisos{}
		}
		resultado, err := s.ExportarPermisosPropios(context.Background(), orden)
		if err == nil || len(resultado.Contenido) != 0 {
			t.Fatal("orden/servicio cero")
		}
	}
	if f.llamadas != 0 {
		t.Fatal("se consultó sin orden")
	}
}
func TestExportacionPermisosCanceladaSinBytes(t *testing.T) {
	for _, fase := range []string{"inicio", "fuente", "renderer", "registro"} {
		t.Run(fase, func(t *testing.T) {
			s, o, f, p, r := prepararInformePermisosPrueba(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch fase {
			case "inicio":
				cancel()
			case "fuente":
				f.alLeer = cancel
			case "renderer":
				p.alPreparar = cancel
			case "registro":
				r.alConfirmar = cancel
			}
			resultado, err := s.ExportarPermisosPropios(ctx, o)
			if !errors.Is(err, context.Canceled) || len(resultado.Contenido) != 0 {
				t.Fatal("cancelación abierta", err)
			}
		})
	}
}
