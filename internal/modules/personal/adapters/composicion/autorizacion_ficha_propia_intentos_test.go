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

type destinoIntentosFichaPrueba struct {
	ordenes             []vecports.OrdenIntentoAuditoria
	err, preflightError error
	acuseInvalido       bool
}

func (d *destinoIntentosFichaPrueba) PreflightIntentoAuditoria(context.Context) error {
	return d.preflightError
}
func (d *destinoIntentosFichaPrueba) AppendIntentoAuditoria(_ context.Context, o vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	d.ordenes = append(d.ordenes, o)
	if d.err != nil {
		return vecports.AcuseIntentoAuditoria{}, d.err
	}
	if d.acuseInvalido {
		return vecports.AcuseIntentoAuditoria{}, nil
	}
	datos, err := o.Datos()
	if err != nil {
		return vecports.AcuseIntentoAuditoria{}, err
	}
	return vecports.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_prueba", Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: datos.Datos.CorrelacionRef, RegistradaEn: time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)}, nil
}
func configuracionIntentosFichaPrueba() ConfiguracionIntentosFichaPropia {
	motivo := vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 3, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_" + strings.Repeat("b", 32)}
	invalido, caido := motivo, motivo
	invalido.EntradaClave = "motivo_" + strings.Repeat("c", 32)
	caido.EntradaClave = "motivo_" + strings.Repeat("d", 32)
	return ConfiguracionIntentosFichaPropia{Proceso: "vec-server-interno", Canal: string(vecdomain.SuperficieAutenticacionInternaCorporativaV1), RecursoEntradaInvalida: "personal:ficha_propia", MotivoDenegado: motivo, MotivoEntradaInvalida: invalido, MotivoNoDisponible: caido}
}
func contextoFichaCapturadaPrueba(t *testing.T, id IdentidadRegistradaFichaPropia) (context.Context, *resolutorIntentoFichaPropiaVigenciaPrueba) {
	t.Helper()
	ctx, err := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	resolver := &resolutorIntentoFichaPropiaVigenciaPrueba{identidad: id}
	ctx, err = PrepararContextoIntentoFichaPropia(ctx, resolver, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, resolver
}
func TestFichaPropiaIntentosRequierenDestinoConfiguracionYCaptura(t *testing.T) {
	var nulo *destinoIntentosFichaPrueba
	for _, d := range []RegistradorIntentosFichaPropia{nil, nulo} {
		if _, err := NuevoRegistroIntentosFichaPropia(d, configuracionIntentosFichaPrueba()); !errors.Is(err, domain.ErrFichaPropiaNoDisponible) {
			t.Fatal("nil admitido", err)
		}
	}
	d := &destinoIntentosFichaPrueba{preflightError: errors.New("ACL privada")}
	if _, err := NuevoRegistroIntentosFichaPropia(d, ConfiguracionIntentosFichaPropia{}); !errors.Is(err, domain.ErrFichaPropiaNoDisponible) {
		t.Fatal("config ausente admitida", err)
	}
	r, _ := NuevoRegistroIntentosFichaPropia(d, configuracionIntentosFichaPrueba())
	if err := r.VerificarRegistroFichaPropia(context.Background()); !errors.Is(err, domain.ErrFichaPropiaNoDisponible) {
		t.Fatal("preflight abierto", err)
	}
	if err := r.RegistrarIntentoFichaPropia(context.Background(), ports.IntentoFichaPropia{Motivo: "denegado"}); !errors.Is(err, domain.ErrFichaPropiaNoDisponible) || len(d.ordenes) != 0 {
		t.Fatal("inventó actor", err)
	}
}
func TestFichaPropiaCapturaConservaPerfilYCorrelacionSinSegundaResolucion(t *testing.T) {
	id := identidadIntentoFichaPropiaVigenciaPrueba(t, time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC), "")
	ctx, resolver := contextoFichaCapturadaPrueba(t, id)
	ref, _ := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	corr, _ := ref.ValorCanonico()
	resolver.identidad = IdentidadRegistradaFichaPropia{}
	nuevo, err := PrepararContextoIntentoFichaPropia(ctx, resolver, time.Second)
	if err != nil || resolver.llamadas != 1 {
		t.Fatal("resolvió otra identidad", err)
	}
	original, err := IdentidadOriginalFichaPropia(nuevo)
	if err != nil || original.Resultado.HuellaSHA256 != id.Resultado.HuellaSHA256 {
		t.Fatal("sustituyó contexto", err)
	}
	original.Resultado.RepresentacionCanonica[0] = 'x'
	d := &destinoIntentosFichaPrueba{}
	r, _ := NuevoRegistroIntentosFichaPropia(d, configuracionIntentosFichaPrueba())
	if err := r.RegistrarIntentoFichaPropia(ctx, ports.IntentoFichaPropia{Motivo: "denegado"}); err != nil {
		t.Fatal(err)
	}
	o, _ := d.ordenes[0].Datos()
	if o.ResultadoContexto.HuellaSHA256 != id.Resultado.HuellaSHA256 || o.ResultadoContexto.Contexto.PerfilActivoRef != id.Resultado.Contexto.PerfilActivoRef || o.Datos.CorrelacionRef != corr || o.Datos.Accion != domain.AccionFichaPropia || o.Datos.FinalidadRef != domain.FinalidadFichaPropia || !strings.HasPrefix(o.Datos.RecursoRef, "personal:ficha_propia:sha256:") {
		t.Fatal("perdió atribución nominal")
	}
}
func TestFichaPropiaRecuperaMismaOrdenSinNuevaLectura(t *testing.T) {
	id := identidadIntentoFichaPropiaVigenciaPrueba(t, time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC), "")
	ctx, resolver := contextoFichaCapturadaPrueba(t, id)
	d := &destinoIntentosFichaPrueba{err: errors.New("commit ambiguo privado")}
	r, _ := NuevoRegistroIntentosFichaPropia(d, configuracionIntentosFichaPrueba())
	in := ports.IntentoFichaPropia{Motivo: "no_disponible"}
	if err := r.RegistrarIntentoFichaPropia(ctx, in); !errors.Is(err, domain.ErrFichaPropiaNoDisponible) || len(d.ordenes) != 2 {
		t.Fatal("no cerró ambiguo", err)
	}
	d.err = nil
	if err := r.RegistrarIntentoFichaPropia(ctx, in); err != nil || len(d.ordenes) != 3 {
		t.Fatal("replay falló", err)
	}
	primera, _ := d.ordenes[0].Datos()
	for _, o := range d.ordenes[1:] {
		datos, _ := o.Datos()
		if datos.IntentoRef != primera.IntentoRef || datos.Datos != primera.Datos || datos.ResultadoContexto.HuellaSHA256 != primera.ResultadoContexto.HuellaSHA256 {
			t.Fatal("replay sustituyó material")
		}
	}
	if err := r.RegistrarIntentoFichaPropia(ctx, in); err != nil || len(d.ordenes) != 3 || resolver.llamadas != 1 {
		t.Fatal("acuse no retenido", err)
	}
	if err := r.RegistrarIntentoFichaPropia(ctx, ports.IntentoFichaPropia{Motivo: "denegado"}); !errors.Is(err, domain.ErrFichaPropiaNoDisponible) || len(d.ordenes) != 3 {
		t.Fatal("misma clave otro material", err)
	}
}
func TestFichaPropiaIntentosNoConfirmanAcuseInvalidoNiCanalAjeno(t *testing.T) {
	id := identidadIntentoFichaPropiaVigenciaPrueba(t, time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC), "")
	for _, caso := range []string{"acuse", "canal"} {
		t.Run(caso, func(t *testing.T) {
			ctx, _ := contextoFichaCapturadaPrueba(t, id)
			d := &destinoIntentosFichaPrueba{acuseInvalido: caso == "acuse"}
			config := configuracionIntentosFichaPrueba()
			if caso == "canal" {
				config.Canal = "administrativa"
			}
			r, _ := NuevoRegistroIntentosFichaPropia(d, config)
			if err := r.RegistrarIntentoFichaPropia(ctx, ports.IntentoFichaPropia{Motivo: "denegado"}); !errors.Is(err, domain.ErrFichaPropiaNoDisponible) {
				t.Fatal("confirmó fallo", err)
			}
			if caso == "canal" && len(d.ordenes) != 0 {
				t.Fatal("reemplazó canal")
			}
		})
	}
}
func TestFichaPropiaCaducidadOCancelacionNoBorranIdentidadHistorica(t *testing.T) {
	for _, caduca := range []string{"sesion", "contexto", "enlace_empleado"} {
		t.Run(caduca, func(t *testing.T) {
			t0 := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
			id := identidadIntentoFichaPropiaVigenciaPrueba(t, t0, caduca)
			if id.Vinculo.VigenteEn(t0.Add(time.Second), id.Resultado) {
				t.Fatal("fixture no caducó")
			}
			ctx, resolver := contextoFichaCapturadaPrueba(t, id)
			ctx, cancel := context.WithCancel(ctx)
			cancel()
			d := &destinoIntentosFichaPrueba{}
			r, _ := NuevoRegistroIntentosFichaPropia(d, configuracionIntentosFichaPrueba())
			if err := r.RegistrarIntentoFichaPropia(context.WithoutCancel(ctx), ports.IntentoFichaPropia{Motivo: "denegado"}); err != nil {
				t.Fatal(err)
			}
			o, _ := d.ordenes[0].Datos()
			if o.ResultadoContexto.HuellaSHA256 != id.Resultado.HuellaSHA256 || resolver.llamadas != 1 {
				t.Fatal("perdió identidad histórica")
			}
		})
	}
}
func TestFichaPropiaCapturaNoInventadaSinCorrelacion(t *testing.T) {
	resolver := &resolutorIntentoFichaPropiaVigenciaPrueba{}
	if _, err := PrepararContextoIntentoFichaPropia(context.Background(), resolver, time.Second); !errors.Is(err, domain.ErrFichaPropiaNoDisponible) || resolver.llamadas != 0 {
		t.Fatal("generó correlación paralela", err)
	}
}
