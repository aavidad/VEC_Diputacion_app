package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type formatosExportPrueba struct {
	formato domain.FormatoExportacionServiciosPropios
	err     error
}

func (f formatosExportPrueba) FormatoParaIdioma(context.Context, string) (domain.FormatoExportacionServiciosPropios, error) {
	return f.formato, f.err
}

type autorizadorExportPrueba struct {
	t        *testing.T
	err      error
	llamadas int
	consulta bool
}

func (a *autorizadorExportPrueba) AutorizarExportacionServiciosPropios(_ context.Context, m domain.MaterialExportacionServiciosPropios) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	if a.err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, a.err
	}
	base, _ := domain.NuevoMaterialFichaPropia(domain.SolicitudFichaPropia{Actor: m.Actor(), Corte: m.Corte()})
	b := atestacionFichaPropiaPrueba(a.t, base, domain.AccionFichaPropia)
	if a.consulta {
		return b, nil
	}
	x := b.ResumenCapacidad()
	h, _ := m.HuellaSHA256()
	r, e := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(x.DecisionRef(), strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), domain.AccionExportacionServiciosPropios, m.EmpleadoRef(), h, domain.AudienciaExportacionServiciosPropios, x.EmitidaEn(), x.ExpiraEn())
	if e != nil {
		a.t.Fatal(e)
	}
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(b.CapacidadCanonica(), r, b.DecisionCanonica(), b.MotivoCanonico(), b.ContextoActorCanonico(), b.PersonaVersion(), b.PerfilVersion(), b.PayloadVECAD3(), b.SobreCOSESign1(), b.EvidenciaVerificacion(), b.RaizPublicaSPKI())
}

type repositorioExportPrueba struct {
	err      error
	llamadas int
	alterar  bool
}

func (r *repositorioExportPrueba) ExportarServiciosPropios(_ context.Context, o ports.OrdenExportacionServiciosPropios) (ports.ResultadoExportacionServiciosPropios, error) {
	r.llamadas++
	if r.err != nil {
		return ports.ResultadoExportacionServiciosPropios{}, r.err
	}
	b := []byte("c1,c2,c3,c4,c5\n")
	h := sha256.Sum256(b)
	x := o.Autorizacion.ResumenCapacidad()
	ref := o.Material.Solicitud().ReciboRef
	if r.alterar {
		ref = "fichapropia:11111111-1111-1111-1111-111111111111"
	}
	return ports.ResultadoExportacionServiciosPropios{Corte: o.Material.Corte(), ContenidoCSV: b, ContenidoSHA256: hex.EncodeToString(h[:]), NombreArchivo: o.Material.Formato().Datos().NombreArchivo, Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: ref, DecisionRef: x.DecisionRef(), EfectoRef: x.EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("d", 64), AuditoriaRef: "auditoria:export", ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)}}, nil
}

type intentosExportPrueba struct {
	err      error
	intentos []ports.IntentoFichaPropia
	ctxErr   error
}

func (i *intentosExportPrueba) VerificarRegistroExportacionServiciosPropios(context.Context) error {
	return nil
}
func (i *intentosExportPrueba) RegistrarIntentoExportacionServiciosPropios(ctx context.Context, in ports.IntentoFichaPropia) error {
	i.intentos = append(i.intentos, in)
	i.ctxErr = ctx.Err()
	return i.err
}
func solicitudYFormatoExportPrueba(t *testing.T) (domain.SolicitudExportacionServiciosPropios, formatosExportPrueba) {
	t.Helper()
	s := solicitudFichaPropiaPrueba(t, "pep_")
	f, e := domain.NuevoFormatoExportacionServiciosPropios(domain.DatosFormatoExportacionServiciosPropios{Referencia: "personal:servicios_propios:csv", Version: 1, Idioma: "xx", CatalogoSHA256: strings.Repeat("a", 64), NombreArchivo: "servicios.csv", Cabeceras: []string{"c1", "c2", "c3", "c4", "c5"}, Estados: map[string]string{"declarado": "e1", "comprobado": "e2", "reconocido": "e3"}})
	if e != nil {
		t.Fatal(e)
	}
	return domain.SolicitudExportacionServiciosPropios{Actor: s.Actor, Corte: s.Corte, ReciboRef: "fichapropia:0f0e0d0c-0b0a-4908-8706-050403020100", Idioma: "xx"}, formatosExportPrueba{formato: f}
}
func TestExportacionServiciosPropiosRequiereV3IndependienteYResultadoExacto(t *testing.T) {
	for _, caso := range []string{"permitido", "consulta", "recibo_ajeno"} {
		t.Run(caso, func(t *testing.T) {
			in, f := solicitudYFormatoExportPrueba(t)
			a := &autorizadorExportPrueba{t: t, consulta: caso == "consulta"}
			r := &repositorioExportPrueba{alterar: caso == "recibo_ajeno"}
			i := &intentosExportPrueba{}
			s, e := NuevoServicioExportacionServiciosPropios(a, r, f, i)
			if e != nil {
				t.Fatal(e)
			}
			out, e := s.Exportar(context.Background(), in)
			if caso == "permitido" {
				if e != nil || len(out.ContenidoCSV) == 0 || r.llamadas != 1 || len(i.intentos) != 0 {
					t.Fatal("export no servido", e)
				}
			} else {
				if !errors.Is(e, domain.ErrExportacionServiciosPropiosNoDisponible) || len(out.ContenidoCSV) != 0 || len(i.intentos) != 1 {
					t.Fatal("fallo no cerrado", e)
				}
				if caso == "consulta" && r.llamadas != 0 {
					t.Fatal("consultó con permiso de lectura")
				}
			}
		})
	}
}
func TestExportacionServiciosPropiosFalloRegistraYSinAcuseDevuelveIndisponible(t *testing.T) {
	for _, fallo := range []bool{false, true} {
		t.Run(strings.ToLower(map[bool]string{false: "registrado", true: "fallido"}[fallo]), func(t *testing.T) {
			in, f := solicitudYFormatoExportPrueba(t)
			a := &autorizadorExportPrueba{t: t}
			r := &repositorioExportPrueba{err: domain.ErrExportacionServiciosPropiosDenegada}
			i := &intentosExportPrueba{}
			if fallo {
				i.err = errors.New("privado")
			}
			s, _ := NuevoServicioExportacionServiciosPropios(a, r, f, i)
			out, e := s.Exportar(context.Background(), in)
			esperado := domain.ErrExportacionServiciosPropiosDenegada
			if fallo {
				esperado = domain.ErrExportacionServiciosPropiosNoDisponible
			}
			if !errors.Is(e, esperado) || len(out.ContenidoCSV) != 0 || len(i.intentos) != 1 || i.intentos[0].Motivo != "denegado" {
				t.Fatal(e)
			}
		})
	}
}
func TestExportacionServiciosPropiosCanceladaAuditaSinLlegarAFuente(t *testing.T) {
	in, f := solicitudYFormatoExportPrueba(t)
	a := &autorizadorExportPrueba{t: t}
	r := &repositorioExportPrueba{}
	i := &intentosExportPrueba{}
	s, _ := NuevoServicioExportacionServiciosPropios(a, r, f, i)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e := s.Exportar(ctx, in)
	if !errors.Is(e, context.Canceled) || a.llamadas != 0 || r.llamadas != 0 || len(i.intentos) != 1 || i.ctxErr != nil {
		t.Fatal("cancelación no conservó intento", e)
	}
	if _, e := NuevoServicioExportacionServiciosPropios(a, r, f, nil); !errors.Is(e, domain.ErrExportacionServiciosPropiosNoDisponible) {
		t.Fatal("registro opcional")
	}
}
