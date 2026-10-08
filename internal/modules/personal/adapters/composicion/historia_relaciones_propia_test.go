package composicion

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func configuracionIntentosHistoriaRelacionesPrueba() ConfiguracionIntentosHistoriaRelacionesPropia {
	c := configuracionIntentosFichaPrueba()
	return ConfiguracionIntentosHistoriaRelacionesPropia{Proceso: c.Proceso, Canal: c.Canal, RecursoEntradaInvalida: "personal:historia_relaciones_propias", MotivoDenegado: c.MotivoDenegado, MotivoEntradaInvalida: c.MotivoEntradaInvalida, MotivoNoDisponible: c.MotivoNoDisponible}
}

func materialHistoriaRelacionesComposicionPrueba(t *testing.T, id IdentidadRegistradaFichaPropia) domain.MaterialHistoriaRelacionesPropia {
	t.Helper()
	m, err := domain.NuevoMaterialHistoriaRelacionesPropia(domain.SolicitudHistoriaRelacionesPropia{Actor: id.Resultado.Contexto, Corte: domain.CorteHistoriaRelacionesPropia{Desde: "2020-01-01", Hasta: "2027-01-01", ConocidoEn: time.Date(2026, 10, 3, 11, 59, 59, 0, time.UTC)}})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestHistoriaRelacionesProveedorEmiteAccionPropiaConCapturaOriginal(t *testing.T) {
	t0 := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	id := identidadIntentoFichaPropiaVigenciaPrueba(t, t0, "")
	ctx, resolver := contextoFichaCapturadaPrueba(t, id)
	resolver.identidad = IdentidadRegistradaFichaPropia{}
	emisor := &emisorFichaCapturadaPrueba{}
	p, err := NuevoProveedorAutorizacionHistoriaRelacionesPropia(resolver, emisor, configuracionIntentosFichaPrueba().MotivoDenegado)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.AutorizarHistoriaRelacionesPropia(ctx, materialHistoriaRelacionesComposicionPrueba(t, id))
	if !errors.Is(err, domain.ErrHistoriaRelacionesPropiaDenegada) || resolver.llamadas != 1 || emisor.llamadas != 1 {
		t.Fatal("perdió identidad o denegación registrada", err)
	}
	datos, err := emisor.solicitud.Datos()
	if err != nil || datos.Accion != domain.AccionHistoriaRelacionesPropia || datos.Finalidad != domain.FinalidadHistoriaRelacionesPropia || datos.Recurso.Tipo != domain.TipoRecursoHistoriaRelacionesPropia || datos.Accion == domain.AccionFichaPropia || datos.Accion == domain.AccionHistoriaServiciosPropia {
		t.Fatal("acción de otra consulta", err)
	}
	ref, _ := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	original, _ := ref.ValorCanonico()
	corr, _ := datos.Correlacion.ValorCanonico()
	if corr != original || emisor.contexto.HuellaSHA256 != id.Resultado.HuellaSHA256 {
		t.Fatal("correlación o contexto sustituidos")
	}
}

func TestHistoriaRelacionesIntentoComunConservaAccionRecursoYReplay(t *testing.T) {
	id := identidadIntentoFichaPropiaVigenciaPrueba(t, time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC), "")
	ctx, resolver := contextoFichaCapturadaPrueba(t, id)
	d := &destinoIntentosFichaPrueba{err: errors.New("commit ambiguo")}
	r, err := NuevoRegistroIntentosHistoriaRelacionesPropia(d, configuracionIntentosHistoriaRelacionesPrueba())
	if err != nil {
		t.Fatal(err)
	}
	in := ports.IntentoHistoriaRelacionesPropia{Motivo: "denegado"}
	if err := r.RegistrarIntentoHistoriaRelacionesPropia(ctx, in); !errors.Is(err, domain.ErrHistoriaRelacionesPropiaNoDisponible) || len(d.ordenes) != 2 {
		t.Fatal("fallo de auditoría confirmado", err)
	}
	d.err = nil
	if err := r.RegistrarIntentoHistoriaRelacionesPropia(ctx, in); err != nil || len(d.ordenes) != 3 {
		t.Fatal("replay sin acuse", err)
	}
	primera, _ := d.ordenes[0].Datos()
	for _, orden := range d.ordenes[1:] {
		datos, _ := orden.Datos()
		if datos.IntentoRef != primera.IntentoRef || datos.Datos != primera.Datos || datos.ResultadoContexto.HuellaSHA256 != primera.ResultadoContexto.HuellaSHA256 {
			t.Fatal("replay cambió orden o identidad")
		}
	}
	if primera.Datos.Accion != domain.AccionHistoriaRelacionesPropia || primera.Datos.FinalidadRef != domain.FinalidadHistoriaRelacionesPropia || !strings.HasPrefix(primera.Datos.RecursoRef, "personal:historia_relaciones_propias:sha256:") || primera.Datos.Resultado != vecdomain.ResultadoIntentoAuditoriaDenegado || resolver.llamadas != 1 {
		t.Fatal("intento común sin atribución propia")
	}
	if err := r.RegistrarIntentoHistoriaRelacionesPropia(ctx, in); err != nil || len(d.ordenes) != 3 {
		t.Fatal("acuse no retenido", err)
	}
}

func TestHistoriaRelacionesCancelacionYPreflightFallidoNoRevelanDatos(t *testing.T) {
	id := identidadIntentoFichaPropiaVigenciaPrueba(t, time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC), "")
	ctx, _ := contextoFichaCapturadaPrueba(t, id)
	d := &destinoIntentosFichaPrueba{preflightError: errors.New("ACL privada")}
	r, _ := NuevoRegistroIntentosHistoriaRelacionesPropia(d, configuracionIntentosHistoriaRelacionesPrueba())
	if err := r.VerificarRegistroHistoriaRelacionesPropia(ctx); !errors.Is(err, domain.ErrHistoriaRelacionesPropiaNoDisponible) || strings.Contains(err.Error(), "ACL") {
		t.Fatal("preflight filtrado", err)
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	d.preflightError = nil
	if err := r.RegistrarIntentoHistoriaRelacionesPropia(context.WithoutCancel(ctx), ports.IntentoHistoriaRelacionesPropia{Motivo: "no_disponible"}); err != nil || len(d.ordenes) != 1 {
		t.Fatal("cancelación borró captura", err)
	}
	o, _ := d.ordenes[0].Datos()
	if o.ResultadoContexto.HuellaSHA256 != id.Resultado.HuellaSHA256 || strings.Contains(o.Datos.RecursoRef, "emp_") {
		t.Fatal("auditoría reveló empleado")
	}
}
