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

type destinoIntentosRPTPrueba struct {
	ordenes        []vecports.OrdenIntentoAuditoria
	err            error
	preflightError error
	acuseInvalido  bool
	checks         int
}

func (d *destinoIntentosRPTPrueba) PreflightIntentoAuditoria(context.Context) error {
	d.checks++
	return d.preflightError
}
func (d *destinoIntentosRPTPrueba) AppendIntentoAuditoria(_ context.Context, o vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
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
	return vecports.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_prueba", Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: datos.Datos.CorrelacionRef, RegistradaEn: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)}, nil
}

func configuracionIntentosRPTPrueba() ConfiguracionIntentosLectorRPT {
	motivo := vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 3, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_" + strings.Repeat("b", 32)}
	invalido, noDisponible := motivo, motivo
	invalido.EntradaClave = "motivo_" + strings.Repeat("c", 32)
	noDisponible.EntradaClave = "motivo_" + strings.Repeat("d", 32)
	return ConfiguracionIntentosLectorRPT{Proceso: "vec-server-interno", RecursoEntradaInvalida: "personal:lector_relacion_rpt", Canal: string(vecdomain.SuperficieAutenticacionInternaCorporativaV1), MotivoDenegado: motivo, MotivoEntradaInvalida: invalido, MotivoNoDisponible: noDisponible}

}

func contextoIntentoRPTPrueba(t *testing.T, identidad IdentidadRegistradaLectorRelacionRPT) context.Context {
	t.Helper()
	ctx, err := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ref, err := vecports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		t.Fatal(err)
	}
	return context.WithValue(ctx, claveIntentoLectorRelacionRPT{}, &intentoLectorRelacionRPT{identidad: identidad, referencia: ref})
}

func TestLectorRPTIntentosCierranDestinoAusenteConfiguracionYPreflight(t *testing.T) {
	var typedNil *destinoIntentosRPTPrueba
	for _, destino := range []RegistradorIntentosLectorRPT{nil, typedNil} {
		if _, err := NuevoRegistroIntentosLectorRelacionRPT(destino, configuracionIntentosRPTPrueba()); !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) {
			t.Fatal("nil admitido", err)
		}
	}
	d := &destinoIntentosRPTPrueba{preflightError: errors.New("detalle privado")}
	if _, err := NuevoRegistroIntentosLectorRelacionRPT(d, ConfiguracionIntentosLectorRPT{}); !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) {
		t.Fatal("config ausente admitida", err)
	}
	r, err := NuevoRegistroIntentosLectorRelacionRPT(d, configuracionIntentosRPTPrueba())
	if err != nil {
		t.Fatal(err)
	}
	if err := r.VerificarRegistroRelacionRPT(context.Background()); !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) || strings.Contains(err.Error(), "privado") || d.checks != 1 {
		t.Fatal("preflight no cerrado", err)
	}
}

func TestLectorRPTIntentoSinCapturaNoInventaActor(t *testing.T) {
	d := &destinoIntentosRPTPrueba{}
	r, _ := NuevoRegistroIntentosLectorRelacionRPT(d, configuracionIntentosRPTPrueba())
	if err := r.RegistrarIntentoRelacionRPT(context.Background(), ports.IntentoLectorRelacionRPT{Motivo: "denegado"}); !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) || len(d.ordenes) != 0 {
		t.Fatal("identidad inventada", err)
	}
}

func TestLectorRPTIntentoConservaIdentidadOriginalAunqueActorEntradaSeaOtro(t *testing.T) {
	id := identidadIntentoRPTVigenciaPrueba(t, time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC), "")
	d := &destinoIntentosRPTPrueba{}
	r, _ := NuevoRegistroIntentosLectorRelacionRPT(d, configuracionIntentosRPTPrueba())
	ctx := contextoIntentoRPTPrueba(t, id)
	if err := r.RegistrarIntentoRelacionRPT(ctx, ports.IntentoLectorRelacionRPT{Motivo: "entrada_invalida", RelacionRef: "correo@privado"}); err != nil {
		t.Fatal(err)
	}
	o, _ := d.ordenes[0].Datos()
	if o.ResultadoContexto.Contexto.PersonaRef != id.Resultado.Contexto.PersonaRef || o.ResultadoContexto.Contexto.PerfilActivoRef != id.Resultado.Contexto.PerfilActivoRef || o.Datos.RecursoRef != "personal:lector_relacion_rpt" || o.Datos.Resultado != vecdomain.ResultadoIntentoAuditoriaError || o.Datos.Motivo != configuracionIntentosRPTPrueba().MotivoEntradaInvalida {
		t.Fatal("atribución o minimización incorrecta")
	}
}

func TestLectorRPTIntentoRecursoConservaCaseSinCopiarEntradaLibre(t *testing.T) {
	id := identidadIntentoRPTVigenciaPrueba(t, time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC), "")
	recursos := []string{}
	for _, ref := range []string{"rel_" + strings.Repeat("a", 24), "rel_" + strings.Repeat("A", 24)} {
		d := &destinoIntentosRPTPrueba{}
		r, _ := NuevoRegistroIntentosLectorRelacionRPT(d, configuracionIntentosRPTPrueba())
		if err := r.RegistrarIntentoRelacionRPT(contextoIntentoRPTPrueba(t, id), ports.IntentoLectorRelacionRPT{Actor: id.Resultado.Contexto, Motivo: "denegado", RelacionRef: ref}); err != nil {
			t.Fatal(err)
		}
		datos, _ := d.ordenes[0].Datos()
		if datos.Datos.Validar() != nil || !strings.HasPrefix(datos.Datos.RecursoRef, "personal:relacion_rpt:sha256:") {
			t.Fatal("recurso inválido")
		}
		recursos = append(recursos, datos.Datos.RecursoRef)
	}
	if recursos[0] == recursos[1] {
		t.Fatal("normalizó la referencia")
	}
}

func TestLectorRPTIntentosRecuperanMismaOrdenSinRepetirOperacion(t *testing.T) {
	id := identidadIntentoRPTVigenciaPrueba(t, time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC), "")
	d := &destinoIntentosRPTPrueba{err: errors.New("commit ambiguo privado")}
	r, _ := NuevoRegistroIntentosLectorRelacionRPT(d, configuracionIntentosRPTPrueba())
	ctx := contextoIntentoRPTPrueba(t, id)
	in := ports.IntentoLectorRelacionRPT{Actor: id.Resultado.Contexto, Motivo: "no_disponible", RelacionRef: "rel_" + strings.Repeat("a", 24)}
	if err := r.RegistrarIntentoRelacionRPT(ctx, in); !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) || len(d.ordenes) != 2 {
		t.Fatal("fallo sin respuesta cerrada", err)
	}
	d.err = nil
	if err := r.RegistrarIntentoRelacionRPT(ctx, in); err != nil || len(d.ordenes) != 3 {
		t.Fatal("replay falló", err)
	}
	first, _ := d.ordenes[0].Datos()
	for _, o := range d.ordenes[1:] {
		x, _ := o.Datos()
		if x.IntentoRef != first.IntentoRef || x.Datos != first.Datos || x.ResultadoContexto.HuellaSHA256 != first.ResultadoContexto.HuellaSHA256 {
			t.Fatal("replay reemplazó material")
		}
	}
	if err := r.RegistrarIntentoRelacionRPT(ctx, in); err != nil || len(d.ordenes) != 3 {
		t.Fatal("acuse no retenido", err)
	}
	in.Motivo = "denegado"
	if err := r.RegistrarIntentoRelacionRPT(ctx, in); !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) || len(d.ordenes) != 3 {
		t.Fatal("reutilizó clave con otro material", err)
	}
}

func TestLectorRPTIntentosNoConfirmanAcuseInvalidoNiCanalAjeno(t *testing.T) {
	id := identidadIntentoRPTVigenciaPrueba(t, time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC), "")
	for _, caso := range []string{"acuse", "canal"} {
		t.Run(caso, func(t *testing.T) {
			d := &destinoIntentosRPTPrueba{acuseInvalido: caso == "acuse"}
			c := configuracionIntentosRPTPrueba()
			if caso == "canal" {
				c.Canal = "administrativa"
			}
			r, _ := NuevoRegistroIntentosLectorRelacionRPT(d, c)
			if err := r.RegistrarIntentoRelacionRPT(contextoIntentoRPTPrueba(t, id), ports.IntentoLectorRelacionRPT{Actor: id.Resultado.Contexto, Motivo: "denegado"}); !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) {
				t.Fatal("confirmó fallo", err)
			}
			if caso == "canal" && len(d.ordenes) != 0 {
				t.Fatal("relabel del canal")
			}
		})
	}
}
