package bootstrap

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type servicioIntentosPreparacionPrueba struct {
	cerrado bool
	err     error
	despues func()
}

func (s *servicioIntentosPreparacionPrueba) terminar() (bolsaports.ResultadoPreparacionBasesV3, error) {
	s.cerrado = true
	if s.despues != nil {
		s.despues()
	}
	return bolsaports.ResultadoPreparacionBasesV3{}, s.err
}
func (s *servicioIntentosPreparacionPrueba) Guardar(context.Context, bolsaports.SolicitudGuardarPreparacionBasesV3) (bolsaports.ResultadoPreparacionBasesV3, error) {
	return s.terminar()
}
func (s *servicioIntentosPreparacionPrueba) Consultar(context.Context, bolsaports.SolicitudConsultarPreparacionBasesV3) (bolsaports.ResultadoPreparacionBasesV3, error) {
	return s.terminar()
}

type registradorIntentosPreparacionPrueba struct {
	servicio      *servicioIntentosPreparacionPrueba
	ordenes       []vecports.DatosOrdenIntentoAuditoria
	err           error
	acuseInvalido bool
}

func (r *registradorIntentosPreparacionPrueba) AppendIntentoAuditoria(_ context.Context, o vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	if !r.servicio.cerrado {
		return vecports.AcuseIntentoAuditoria{}, errors.New("negocio todavía abierto")
	}
	d, err := o.Datos()
	if err != nil {
		return vecports.AcuseIntentoAuditoria{}, err
	}
	r.ordenes = append(r.ordenes, d)
	if r.err != nil {
		return vecports.AcuseIntentoAuditoria{}, r.err
	}
	if r.acuseInvalido {
		return vecports.AcuseIntentoAuditoria{}, nil
	}
	return vecports.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_sintetica", Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64),
		CorrelacionRef: d.Datos.CorrelacionRef, RegistradaEn: time.Now().UTC()}, nil
}

func TestPreparacionBasesIntentosNominalesDespuesDelRetorno(t *testing.T) {
	for i := range 2 {
		for _, fallo := range []error{bolsaports.ErrPreparacionBasesDenegada, bolsaports.ErrPreparacionBasesNoDisponible, context.Canceled,
			errors.Join(bolsaports.ErrPreparacionBasesDenegada, bolsaports.ErrPreparacionBasesNoDisponible)} {
			for _, auditor := range []string{"confirmado", "fallo", "acuse_invalido"} {
				t.Run(string(rune('A'+i))+fallo.Error()+auditor, func(t *testing.T) {
					b, ctx := brokerPreparacionBasesPrueba(t, i)
					z, _, err := b.contexto(ctx)
					if err != nil {
						t.Fatal(err)
					}
					h, err := b.ResolverContextoHTTP(httptest.NewRequest("POST", b.perfiles[i].ruta, nil).WithContext(ctx))
					if err != nil {
						t.Fatal(err)
					}
					servicio := &servicioIntentosPreparacionPrueba{err: fallo}
					// La sesión deja de ser resoluble al retornar. La auditoría debe usar
					// lo capturado al entrar, también después de revocación o cancelación.
					servicio.despues = func() { b.sesiones[i] = nil }
					r := &registradorIntentosPreparacionPrueba{servicio: servicio}
					if auditor == "fallo" {
						r.err = errors.New("auditor sintético no disponible")
					}
					r.acuseInvalido = auditor == "acuse_invalido"
					motivo := motivoCatalogoPlantillasCTDesarrollo()
					p, err := nuevoPreparadorBasesAuditadoV3(servicio, b, r, "vec-sintetico", motivo, motivo)
					if err != nil {
						t.Fatal(err)
					}
					if i == 0 {
						_, err = p.Guardar(ctx, bolsaports.SolicitudGuardarPreparacionBasesV3{Actor: h.Actor, Correlacion: h.Correlacion, Ambito: h.Ambito,
							Esperada: prep.Esperada{PreparacionRef: "preparacion:sintetica"}, Material: prep.Material{}, ClaveOperacion: "operacion:sintetica"})
					} else {
						_, err = p.Consultar(ctx, bolsaports.SolicitudConsultarPreparacionBasesV3{Actor: h.Actor, Correlacion: h.Correlacion, Ambito: h.Ambito,
							Selector: bolsaports.SelectorConsultaPreparacionBases{Modo: "actual", Exacta: prep.Esperada{PreparacionRef: "preparacion:sintetica"}}})
					}
					if auditor == "confirmado" {
						if !errors.Is(err, fallo) {
							t.Fatalf("resultado original perdido: %v", err)
						}
					} else {
						if !errors.Is(err, bolsaports.ErrPreparacionBasesNoDisponible) {
							t.Fatalf("auditoría sin acuse no cerró 503: %v", err)
						}
					}
					if len(r.ordenes) != 1 {
						t.Fatalf("intentos=%d", len(r.ordenes))
					}
					d := r.ordenes[0]
					corr, _ := h.Correlacion.ValorCanonico()
					resultado := core.ResultadoIntentoAuditoriaError
					if fallo == bolsaports.ErrPreparacionBasesDenegada {
						resultado = core.ResultadoIntentoAuditoriaDenegado
					}
					if d.ResultadoContexto.HuellaSHA256 != z.Resultado.HuellaSHA256 || d.Datos.Resultado != resultado || d.Datos.CorrelacionRef != corr ||
						d.Datos.RecursoRef != "preparacion:sintetica" || d.Datos.Accion != b.perfiles[i].accion || d.Datos.Proceso != "vec-sintetico" {
						t.Fatal("intento no conserva identidad histórica, recurso, acción o correlación")
					}
				})
			}
		}
	}
}

func TestPreparacionBasesResultadoAutorizadoNoDuplicaIntento(t *testing.T) {
	for _, errNegocio := range []error{nil, bolsaports.ErrPreparacionBasesConflicto, bolsaports.ErrPreparacionBasesClaveReutilizada, bolsaports.ErrPreparacionBasesNoEncontrada} {
		b, ctx := brokerPreparacionBasesPrueba(t, 1)
		h, err := b.ResolverContextoHTTP(httptest.NewRequest("POST", b.perfiles[1].ruta, nil).WithContext(ctx))
		if err != nil {
			t.Fatal(err)
		}
		servicio := &servicioIntentosPreparacionPrueba{err: errNegocio}
		r := &registradorIntentosPreparacionPrueba{servicio: servicio}
		motivo := motivoCatalogoPlantillasCTDesarrollo()
		p, err := nuevoPreparadorBasesAuditadoV3(servicio, b, r, "vec-sintetico", motivo, motivo)
		if err != nil {
			t.Fatal(err)
		}
		_, err = p.Consultar(ctx, bolsaports.SolicitudConsultarPreparacionBasesV3{Actor: h.Actor, Correlacion: h.Correlacion, Ambito: h.Ambito,
			Selector: bolsaports.SelectorConsultaPreparacionBases{Modo: "actual", Exacta: prep.Esperada{PreparacionRef: "preparacion:sintetica"}}})
		if err != errNegocio || len(r.ordenes) != 0 {
			t.Fatal("resultado autorizado generó otro intento o cambió error")
		}
	}
}
