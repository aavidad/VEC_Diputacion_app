package auditoriafirma

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type lectorRecuperacionPrueba struct {
	fallo    error
	cerrado  bool
	cancelar context.CancelFunc
}

func (l *lectorRecuperacionPrueba) RecuperarFirmasAutorizadasV2(context.Context, ct.MaterialConsultaFirmasR5V2, ct.CapacidadRecuperacionFirmasV2) (ct.LecturaRecuperacionFirmasV2, error) {
	l.cerrado = true
	if l.cancelar != nil {
		l.cancelar()
	}
	return ct.LecturaRecuperacionFirmasV2{LecturaFirmasR5V2: ct.LecturaFirmasR5V2{
		LecturaFirmasR5: ct.LecturaFirmasR5{HistoriaRevision: 7}}}, l.fallo
}

type fabricaRecuperacionPrueba struct {
	t        *testing.T
	lector   *lectorRecuperacionPrueba
	alterar  string
	llamadas int
}

func (f *fabricaRecuperacionPrueba) CrearOrdenIntentoFirma(ctx context.Context, intento, accion, recurso string, resultado domain.ResultadoIntentoAuditoria) (ports.OrdenIntentoAuditoria, error) {
	f.llamadas++
	if !f.lector.cerrado {
		f.t.Fatal("auditoría antes de cerrar lector")
	}
	ref, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return ports.OrdenIntentoAuditoria{}, err
	}
	correlacion, err := ref.ValorCanonico()
	if err != nil {
		return ports.OrdenIntentoAuditoria{}, err
	}
	if f.alterar == "correlacion" {
		correlacion = "correlacion_22222222222222222222222222222222"
	}
	if f.alterar == "recurso" {
		recurso = "expediente:ajeno"
	}
	ahora := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	r, v, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		return ports.OrdenIntentoAuditoria{}, err
	}
	return ports.NuevaOrdenIntentoAuditoria(intento, r, v, domain.DatosIntentoAuditoria{
		Accion: accion, ModuloID: ct.ModuloContratacion, RecursoRef: recurso,
		FinalidadRef: "contratacion_temporal", Resultado: resultado, Proceso: "vec-server", Canal: "interna_corporativa",
		CorrelacionRef: correlacion, Motivo: domain.ReferenciaEntradaCatalogo{
			CatalogoID: "motivos_auditoria", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "acceso_denegado"}})
}

type intentosRecuperacionPrueba struct {
	llamadas   int
	acuseAjeno bool
}

func (a *intentosRecuperacionPrueba) AppendIntentoAuditoria(_ context.Context, orden ports.OrdenIntentoAuditoria) (ports.AcuseIntentoAuditoria, error) {
	a.llamadas++
	d, err := orden.Datos()
	if err != nil {
		return ports.AcuseIntentoAuditoria{}, err
	}
	correlacion := d.Datos.CorrelacionRef
	if a.acuseAjeno {
		correlacion = "correlacion_22222222222222222222222222222222"
	}
	return ports.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_11111111111111111111111111111111", Secuencia: 1,
		HuellaSHA256: strings.Repeat("b", 64), CorrelacionRef: correlacion,
		RegistradaEn: time.Date(2026, 10, 3, 10, 0, 1, 0, time.UTC)}, nil
}

func TestRecuperacionAuditaDenegadoYErrorDespuesLector(t *testing.T) {
	for _, fallo := range []error{ct.ErrFirmaDocumentoDenegada, ct.ErrRegistroFirmaDocumentoNoDisponible, context.Canceled} {
		t.Run(fallo.Error(), func(t *testing.T) {
			ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			l := &lectorRecuperacionPrueba{fallo: fallo}
			if errors.Is(fallo, context.Canceled) {
				var cancelar context.CancelFunc
				ctx, cancelar = context.WithCancel(ctx)
				defer cancelar()
				l.cancelar = cancelar
			}
			f := &fabricaRecuperacionPrueba{t: t, lector: l}
			a := &intentosRecuperacionPrueba{}
			r, err := NuevaRecuperacion(l, a, f)
			if err != nil {
				t.Fatal(err)
			}
			lectura, err := r.RecuperarFirmasAutorizadasV2(ctx, ct.MaterialConsultaFirmasR5V2{
				MaterialConsultaFirmasR5: ct.MaterialConsultaFirmasR5{ExpedienteRef: "expediente:prueba"}}, ct.CapacidadRecuperacionFirmasV2{})
			if !errors.Is(err, fallo) || lectura.HistoriaRevision != 0 || !l.cerrado || f.llamadas != 1 || a.llamadas != 1 {
				t.Fatal("fallo expuso lectura o perdió auditoría poslector")
			}
		})
	}
}

func TestRecuperacionCierraAnteOrdenOAcuseAjeno(t *testing.T) {
	for _, caso := range []string{"correlacion", "recurso", "acuse"} {
		t.Run(caso, func(t *testing.T) {
			ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			l := &lectorRecuperacionPrueba{fallo: ct.ErrFirmaDocumentoDenegada}
			f := &fabricaRecuperacionPrueba{t: t, lector: l, alterar: caso}
			a := &intentosRecuperacionPrueba{acuseAjeno: caso == "acuse"}
			r, err := NuevaRecuperacion(l, a, f)
			if err != nil {
				t.Fatal(err)
			}
			_, err = r.RecuperarFirmasAutorizadasV2(ctx, ct.MaterialConsultaFirmasR5V2{
				MaterialConsultaFirmasR5: ct.MaterialConsultaFirmasR5{ExpedienteRef: "expediente:prueba"}}, ct.CapacidadRecuperacionFirmasV2{})
			if !errors.Is(err, ct.ErrRegistroFirmaDocumentoNoDisponible) || f.llamadas != 1 {
				t.Fatal("orden o acuse ajeno admitido")
			}
			if caso != "acuse" && a.llamadas != 0 {
				t.Fatal("orden ajena persistida")
			}
		})
	}
}

func TestRecuperacionConservaExitoYRechazaDependenciasNulas(t *testing.T) {
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	l := &lectorRecuperacionPrueba{}
	f := &fabricaRecuperacionPrueba{t: t, lector: l}
	a := &intentosRecuperacionPrueba{}
	r, err := NuevaRecuperacion(l, a, f)
	if err != nil {
		t.Fatal(err)
	}
	lectura, err := r.RecuperarFirmasAutorizadasV2(ctx, ct.MaterialConsultaFirmasR5V2{}, ct.CapacidadRecuperacionFirmasV2{})
	if err != nil || lectura.HistoriaRevision != 7 || f.llamadas != 0 || a.llamadas != 0 {
		t.Fatal("éxito alterado o auditoría duplicada")
	}
	l.cerrado = false
	if _, err := r.RecuperarFirmasAutorizadasV2(context.Background(), ct.MaterialConsultaFirmasR5V2{}, ct.CapacidadRecuperacionFirmasV2{}); !errors.Is(err, ct.ErrRegistroFirmaDocumentoNoDisponible) || l.cerrado {
		t.Fatal("sin correlación se leyó")
	}
	var nulo *lectorRecuperacionPrueba
	if _, err := NuevaRecuperacion(nulo, a, f); !errors.Is(err, ct.ErrRegistroFirmaDocumentoNoDisponible) {
		t.Fatal("lector nulo admitido")
	}
	if _, err := (&Recuperacion{}).RecuperarFirmasAutorizadasV2(ctx, ct.MaterialConsultaFirmasR5V2{}, ct.CapacidadRecuperacionFirmasV2{}); !errors.Is(err, ct.ErrRegistroFirmaDocumentoNoDisponible) {
		t.Fatal("valor cero del wrapper no cerró")
	}
}

// Una versión inexistente llega del lector como «no encontrado» después de
// confirmar el consumo de la decisión con su auditoría. No se añade otro
// intento «error»: el acceso ya está registrado y no es un fallo técnico.
func TestLecturaNoEncontradaNoDuplicaAuditoria(t *testing.T) {
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	l := &lectorRecuperacionPrueba{fallo: ct.ErrExpedienteConsultaFirmasNoEncontrado}
	f := &fabricaRecuperacionPrueba{t: t, lector: l}
	a := &intentosRecuperacionPrueba{}
	r, err := NuevaRecuperacion(l, a, f)
	if err != nil {
		t.Fatal(err)
	}
	m := ct.MaterialConsultaFirmasR5V2{MaterialConsultaFirmasR5: ct.MaterialConsultaFirmasR5{ExpedienteRef: "expediente:prueba"}}
	lectura, err := r.RecuperarFirmasAutorizadasV2(ctx, m, ct.CapacidadRecuperacionFirmasV2{})
	if !errors.Is(err, ct.ErrExpedienteConsultaFirmasNoEncontrado) || lectura.HistoriaRevision != 0 || !l.cerrado ||
		f.llamadas != 0 || a.llamadas != 0 {
		t.Fatalf("recuperación no encontrada auditada otra vez: %v, órdenes=%d, intentos=%d", err, f.llamadas, a.llamadas)
	}

	base := &registroPrueba{fallo: ct.ErrExpedienteConsultaFirmasNoEncontrado}
	fc := &fabricaPrueba{t: t, registro: base}
	ac := &intentosPrueba{fabrica: fc}
	consulta, err := Nuevo(base, ac, fc)
	if err != nil {
		t.Fatal(err)
	}
	vista, err := consulta.ConsultarFirmasAutorizadasV2(ctx, m, ct.CapacidadConsultaFirmasR5V2{})
	if !errors.Is(err, ct.ErrExpedienteConsultaFirmasNoEncontrado) || vista.HistoriaRevision != 0 || !base.cerrado ||
		fc.invocaciones != 0 || ac.invocaciones != 0 {
		t.Fatalf("consulta no encontrada auditada otra vez: %v, órdenes=%d, intentos=%d", err, fc.invocaciones, ac.invocaciones)
	}
}
