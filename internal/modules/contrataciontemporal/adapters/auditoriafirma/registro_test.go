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

type registroPrueba struct {
	fallo   error
	cerrado bool
}

func (r *registroPrueba) RegistrarFirmaVerificadaV2(context.Context, ct.MaterialFirmaVerificadaV2, ct.CapacidadFirmaVerificadaV2) (ct.ReciboFirmaDocumento, error) {
	r.cerrado = true
	return ct.ReciboFirmaDocumento{ReciboRef: "recibo:original"}, r.fallo
}
func (r *registroPrueba) ConsultarFirmasAutorizadasV2(context.Context, ct.MaterialConsultaFirmasR5V2, ct.CapacidadConsultaFirmasR5V2) (ct.LecturaFirmasR5V2, error) {
	r.cerrado = true
	return ct.LecturaFirmasR5V2{LecturaFirmasR5: ct.LecturaFirmasR5{HistoriaRevision: 7}}, r.fallo
}

type fabricaPrueba struct {
	t            *testing.T
	registro     *registroPrueba
	alterar      bool
	invocaciones int
	ultima       ports.OrdenIntentoAuditoria
}

func (f *fabricaPrueba) CrearOrdenIntentoFirma(_ context.Context, intento, accion, recurso string, resultado domain.ResultadoIntentoAuditoria) (ports.OrdenIntentoAuditoria, error) {
	f.invocaciones++
	if !f.registro.cerrado {
		f.t.Fatal("auditoría antes de cerrar transacción original")
	}
	ahora := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	r, v, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		f.t.Fatal(err)
	}
	if f.alterar {
		recurso = "expediente:ajeno"
	}
	d := domain.DatosIntentoAuditoria{Accion: accion, ModuloID: ct.ModuloContratacion, RecursoRef: recurso,
		FinalidadRef: "contratacion_temporal", Resultado: resultado, Proceso: "vec-server", Canal: "interna_corporativa",
		CorrelacionRef: "correlacion_11111111111111111111111111111111", Motivo: domain.ReferenciaEntradaCatalogo{
			CatalogoID: "motivos_auditoria", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "acceso_denegado"}}
	f.ultima, err = ports.NuevaOrdenIntentoAuditoria(intento, r, v, d)
	return f.ultima, err
}

type intentosPrueba struct {
	fabrica           *fabricaPrueba
	invocaciones      int
	fallo, acuseAjeno bool
}

func (a *intentosPrueba) AppendIntentoAuditoria(_ context.Context, o ports.OrdenIntentoAuditoria) (ports.AcuseIntentoAuditoria, error) {
	a.invocaciones++
	d, err := o.Datos()
	if err != nil {
		a.fabrica.t.Fatal(err)
	}
	original, err := a.fabrica.ultima.Datos()
	if err != nil || d.IntentoRef != original.IntentoRef || d.ResultadoContexto.HuellaSHA256 != original.ResultadoContexto.HuellaSHA256 || d.Datos != original.Datos {
		a.fabrica.t.Fatal("identidad u orden sustituidas")
	}
	if a.fallo {
		return ports.AcuseIntentoAuditoria{}, errors.New("sin confirmar auditoria")
	}
	correlacion := d.Datos.CorrelacionRef
	if a.acuseAjeno {
		correlacion = "correlacion_22222222222222222222222222222222"
	}
	return ports.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_11111111111111111111111111111111", Secuencia: 1,
		HuellaSHA256: strings.Repeat("b", 64), CorrelacionRef: correlacion, RegistradaEn: time.Date(2026, 10, 3, 10, 0, 1, 0, time.UTC)}, nil
}

func TestRegistroFirmaAuditaFallosDespuesDeCerrar(t *testing.T) {
	for _, consulta := range []bool{false, true} {
		for _, fallo := range []error{ct.ErrFirmaDocumentoDenegada, ct.ErrRegistroFirmaDocumentoNoDisponible, context.Canceled} {
			t.Run(fallo.Error()+map[bool]string{true: "_consulta", false: "_registro"}[consulta], func(t *testing.T) {
				base := &registroPrueba{fallo: fallo}
				f := &fabricaPrueba{t: t, registro: base}
				a := &intentosPrueba{fabrica: f}
				r, err := Nuevo(base, a, f)
				if err != nil {
					t.Fatal(err)
				}
				if consulta {
					l, e := r.ConsultarFirmasAutorizadasV2(context.Background(), ct.MaterialConsultaFirmasR5V2{MaterialConsultaFirmasR5: ct.MaterialConsultaFirmasR5{ExpedienteRef: "expediente:prueba"}}, ct.CapacidadConsultaFirmasR5V2{})
					err = e
					if l.HistoriaRevision != 0 {
						t.Fatal("fallo expuso historia")
					}
				} else {
					m := ct.MaterialFirmaVerificadaV2{MaterialFirmaExterna: ct.MaterialFirmaExterna{Via: ct.ViaFirmaCertificadoVEC, ClaveIdempotencia: "clave-prueba-000001"}}
					v, e := r.RegistrarFirmaVerificadaV2(context.Background(), m, ct.CapacidadFirmaVerificadaV2{})
					err = e
					if v.ReciboRef != "" {
						t.Fatal("fallo expuso recibo")
					}
				}
				if !errors.Is(err, fallo) || a.invocaciones != 1 || f.invocaciones != 1 {
					t.Fatalf("fallo no acreditado: %v", err)
				}
				d, e := f.ultima.Datos()
				if e != nil {
					t.Fatal(e)
				}
				resultado := domain.ResultadoIntentoAuditoriaError
				if errors.Is(fallo, ct.ErrFirmaDocumentoDenegada) {
					resultado = domain.ResultadoIntentoAuditoriaDenegado
				}
				if d.Datos.Resultado != resultado {
					t.Fatal("COMMIT/error clasificado como permiso")
				}
			})
		}
	}
}

func TestRegistroFirmaExigeOrdenYAcuseExactos(t *testing.T) {
	for _, caso := range []string{"orden_ajena", "auditoria_fallida", "acuse_ajeno"} {
		t.Run(caso, func(t *testing.T) {
			base := &registroPrueba{fallo: ct.ErrFirmaDocumentoDenegada}
			f := &fabricaPrueba{t: t, registro: base, alterar: caso == "orden_ajena"}
			a := &intentosPrueba{fabrica: f, fallo: caso == "auditoria_fallida", acuseAjeno: caso == "acuse_ajeno"}
			r, err := Nuevo(base, a, f)
			if err != nil {
				t.Fatal(err)
			}
			_, err = r.ConsultarFirmasAutorizadasV2(context.Background(), ct.MaterialConsultaFirmasR5V2{MaterialConsultaFirmasR5: ct.MaterialConsultaFirmasR5{ExpedienteRef: "expediente:prueba"}}, ct.CapacidadConsultaFirmasR5V2{})
			if !errors.Is(err, ct.ErrRegistroFirmaDocumentoNoDisponible) {
				t.Fatal("auditoría fallida presentada como denegación acreditada")
			}
			if caso == "orden_ajena" && a.invocaciones != 0 {
				t.Fatal("se persistió intento de recurso ajeno")
			}
		})
	}
}

func TestRegistroFirmaPermitidoConservaAuditoriaTransaccional(t *testing.T) {
	base := &registroPrueba{}
	f := &fabricaPrueba{t: t, registro: base}
	a := &intentosPrueba{fabrica: f}
	r, err := Nuevo(base, a, f)
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := r.RegistrarFirmaVerificadaV2(context.Background(), ct.MaterialFirmaVerificadaV2{}, ct.CapacidadFirmaVerificadaV2{})
	if err != nil || recibo.ReciboRef != "recibo:original" || f.invocaciones != 0 || a.invocaciones != 0 {
		t.Fatal("éxito duplicó auditoría")
	}
	var ausente *registroPrueba
	if _, err := Nuevo(ausente, a, f); !errors.Is(err, ct.ErrRegistroFirmaDocumentoNoDisponible) {
		t.Fatal("dependencia nula admitida")
	}
}
