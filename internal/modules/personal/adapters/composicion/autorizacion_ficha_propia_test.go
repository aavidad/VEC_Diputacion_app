package composicion

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestClasificarErrorAutorizacionFichaPropia(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		ctx      context.Context
		err      error
		esperado error
	}{
		{"denegacion registrada", context.Background(), vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, personaldomain.ErrFichaPropiaDenegada},
		{"emisor caido", context.Background(), errors.New("emisor caido"), personaldomain.ErrFichaPropiaNoDisponible},
		{"registro de denegacion caido", context.Background(), errors.Join(vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible), personaldomain.ErrFichaPropiaNoDisponible},
		{"timeout mezclado", context.Background(), errors.Join(vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, context.DeadlineExceeded), personaldomain.ErrFichaPropiaNoDisponible},
		{"sin contexto", nil, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, personaldomain.ErrFichaPropiaNoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if got := clasificarErrorAutorizacionFichaPropia(caso.ctx, caso.err); !errors.Is(got, caso.esperado) {
				t.Fatalf("clasificación: %v", got)
			}
		})
	}
}

type identidadFichaPropiaPrueba struct{ err error }

func (i identidadFichaPropiaPrueba) ResolverIdentidadFichaPropia(context.Context) (IdentidadRegistradaFichaPropia, error) {
	return IdentidadRegistradaFichaPropia{}, i.err
}

type emisorFichaPropiaPrueba struct{ llamadas int }

func (e *emisorFichaPropiaPrueba) EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	return vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, errors.New("no debe emitirse")
}

func TestProveedorFichaPropiaNoEmiteSinIdentidadDeLaPeticion(t *testing.T) {
	if _, err := NuevoProveedorAutorizacionFichaPropia(identidadFichaPropiaPrueba{}, &emisorFichaPropiaPrueba{}, vecdomain.ReferenciaEntradaCatalogo{}); err == nil {
		t.Fatal("proveedor compuesto sin motivo válido")
	}
	emisor := &emisorFichaPropiaPrueba{}
	p := &ProveedorAutorizacionFichaPropia{identidad: identidadFichaPropiaPrueba{err: errors.New("sin sesión")}, emisor: emisor}
	if _, err := p.AutorizarFichaPropia(context.Background(), personaldomain.MaterialFichaPropia{}); !errors.Is(err, personaldomain.ErrFichaPropiaNoDisponible) || emisor.llamadas != 0 {
		t.Fatal("se emitió sin material o sin identidad", err)
	}
}

type emisorFichaCapturadaPrueba struct {
	llamadas  int
	solicitud vecdomain.SolicitudAutorizacionLigadaV3
	contexto  vecdomain.ResultadoContextoActorRegistradoV2
}

func (e *emisorFichaCapturadaPrueba) EmitirMaterialAutorizacionAtestadaV3(_ context.Context, s vecdomain.SolicitudAutorizacionLigadaV3, r vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	e.solicitud = s
	e.contexto = r
	return vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3
}
func TestProveedorFichaPropiaUsaIdentidadYCorrelacionOriginales(t *testing.T) {
	t0 := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	id := identidadIntentoFichaPropiaVigenciaPrueba(t, t0, "")
	ctx, resolver := contextoFichaCapturadaPrueba(t, id)
	resolver.identidad = IdentidadRegistradaFichaPropia{}
	m, err := personaldomain.NuevoMaterialFichaPropia(personaldomain.SolicitudFichaPropia{Actor: id.Resultado.Contexto, Corte: personaldomain.CorteEmpleadoB2{VigenteEn: "2026-10-03", ConocidoEn: t0.Add(-time.Second)}})
	if err != nil {
		t.Fatal(err)
	}
	emisor := &emisorFichaCapturadaPrueba{}
	p, err := NuevoProveedorAutorizacionFichaPropia(resolver, emisor, configuracionIntentosFichaPrueba().MotivoDenegado)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.AutorizarFichaPropia(ctx, m)
	if !errors.Is(err, personaldomain.ErrFichaPropiaDenegada) || resolver.llamadas != 1 || emisor.llamadas != 1 {
		t.Fatal("otra identidad o emisión", err)
	}
	datos, err := emisor.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	c, _ := datos.Correlacion.ValorCanonico()
	ref, _ := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	original, _ := ref.ValorCanonico()
	if c != original || !bytes.Equal(emisor.contexto.RepresentacionCanonica, id.Resultado.RepresentacionCanonica) {
		t.Fatal("sustituyó correlación/contexto")
	}
}

type consultorFichaCapturadaPrueba struct {
	llamadas  int
	identidad IdentidadRegistradaFichaPropia
}

func (c *consultorFichaCapturadaPrueba) Consultar(ctx context.Context, _ personaldomain.SolicitudFichaPropia) (personalports.ResultadoFichaPropia, error) {
	c.llamadas++
	c.identidad, _ = IdentidadOriginalFichaPropia(ctx)
	return personalports.ResultadoFichaPropia{}, nil
}
func TestConsultorFichaCapturaAntesDeInvocarYSinDuplicarCapturaHTTP(t *testing.T) {
	id := identidadIntentoFichaPropiaVigenciaPrueba(t, time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC), "")
	ctx, resolver := contextoFichaCapturadaPrueba(t, id)
	consulta := &consultorFichaCapturadaPrueba{}
	wrapper, err := NuevoConsultorFichaPropiaConIdentidad(consulta, resolver, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = wrapper.Consultar(ctx, personaldomain.SolicitudFichaPropia{}); err != nil || consulta.llamadas != 1 || resolver.llamadas != 1 || consulta.identidad.Resultado.HuellaSHA256 != id.Resultado.HuellaSHA256 {
		t.Fatal("captura duplicada", err)
	}
	if _, err = wrapper.Consultar(context.Background(), personaldomain.SolicitudFichaPropia{}); !errors.Is(err, personaldomain.ErrFichaPropiaNoDisponible) || consulta.llamadas != 1 {
		t.Fatal("consulta sin captura", err)
	}
}
