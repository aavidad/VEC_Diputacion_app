package application

import (
	"context"
	"strings"
	"testing"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/ports"
)

// Estas pruebas reutilizan exclusivamente la fixture estructural Go anterior;
// no representan una concesion central ni hacen llamadas PostgreSQL.
type flujoCopiasPreparacionV3Prueba struct {
	t        *testing.T
	retenido ports.ResultadoPreparacionBasesV3
}

func (f *flujoCopiasPreparacionV3Prueba) AutorizarGuardadoPreparacionBases(_ context.Context, q ports.SolicitudGuardarPreparacionBasesV3) (core.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p, _ := PrepararGuardadoPreparacionBasesV3(q)
	return exportacionEstructuralBasesPrueba(f.t, p, q.Actor, ports.AccionGuardarPreparacionBases, AudienciaGuardarPreparacionBasesV3), nil
}
func (f *flujoCopiasPreparacionV3Prueba) AutorizarConsultaPreparacionBases(_ context.Context, q ports.SolicitudConsultarPreparacionBasesV3) (core.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p, _ := PrepararConsultaPreparacionBasesV3(q)
	return exportacionEstructuralBasesPrueba(f.t, p, q.Actor, ports.AccionConsultarPreparacionBases, AudienciaConsultarPreparacionBasesV3), nil
}
func (f *flujoCopiasPreparacionV3Prueba) GuardarPreparacionBasesV3(_ context.Context, o ports.OrdenGuardarPreparacionBasesV3) (ports.ResultadoPreparacionBasesV3, error) {
	return f.acceso(o.Autorizacion, o.Solicitud.Actor.ResueltoEn), nil
}
func (f *flujoCopiasPreparacionV3Prueba) ConsultarPreparacionBasesV3(_ context.Context, o ports.OrdenConsultarPreparacionBasesV3) (ports.ResultadoPreparacionBasesV3, error) {
	r := f.acceso(o.Autorizacion, o.Solicitud.Actor.ResueltoEn)
	r.Estado = "obtenida"
	return r, nil
}
func (f *flujoCopiasPreparacionV3Prueba) acceso(e core.ExportacionMaterialConsumoAutorizacionAtestadaV3, instante time.Time) ports.ResultadoPreparacionBasesV3 {
	r := f.retenido
	r.Acceso = ports.EvidenciaAccesoPreparacionBasesV3{DecisionRef: e.ResumenCapacidad().DecisionRef(), ConsumoHuellaSHA256: strings.Repeat("b", 64), ReciboRef: "recibo:acceso", AuditoriaRef: "auditoria:acceso", CorrelacionRef: "correlacion_" + strings.Repeat("a", 32), AccedidaEn: instante.Add(time.Second)}
	return r
}

func TestPreparacionBasesV3NoComparteSlicesAlGuardarNiConsultar(t *testing.T) {
	q := solicitudBasesV3Prueba(t)
	q.Material = prep.Material{Contenido: bolsa.ContenidoPublicableConvocatoria{Categorias: []string{"auxiliar"}, Documentos: []bolsa.DocumentoPublicableConvocatoria{{Titulo: "Propuesta sintética"}}}, Referencias: []prep.ReferenciaPropuesta{{Campo: "plaza", Referencia: bolsa.ReferenciaConfiguracionConvocatoria{ID: "plaza:sintetica"}}}}
	h, _ := q.Material.HuellaSHA256()
	i, _ := prep.HuellaIntencion(q.Esperada, q.Material, q.Ambito)
	f := &flujoCopiasPreparacionV3Prueba{t: t, retenido: ports.ResultadoPreparacionBasesV3{Estado: "guardada", Version: prep.Version{Ambito: q.Ambito, Estado: prep.Esperada{PreparacionRef: q.Esperada.PreparacionRef, Revision: 1, HuellaMaterialSHA256: h}, Material: q.Material}, Recibo: ports.ReciboPreparacionBases{ReciboRef: "recibo:original", HistoriaRef: "historia:original", AuditoriaRef: "auditoria:efecto", EventoRef: "evento:original", HuellaIntencionSHA256: i, ConfirmadaEn: q.Actor.ResueltoEn}}}
	s, err := NuevoServicioPreparacionBasesV3(f, f)
	if err != nil {
		t.Fatal(err)
	}
	guardado, err := s.Guardar(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	consulta := ports.SolicitudConsultarPreparacionBasesV3{Actor: q.Actor, Correlacion: q.Correlacion, Ambito: q.Ambito, Selector: ports.SelectorConsultaPreparacionBases{Modo: "actual", Exacta: prep.Esperada{PreparacionRef: q.Esperada.PreparacionRef}}}
	leido, err := s.Consultar(context.Background(), consulta)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []ports.ResultadoPreparacionBasesV3{guardado, leido} {
		r.Version.Material.Contenido.Categorias[0] = "otra"
		r.Version.Material.Contenido.Documentos[0].Titulo = "cambiado"
		r.Version.Material.Referencias[0].Referencia.ID = "plaza:otra"
		original, _ := f.retenido.Version.Material.HuellaSHA256()
		if original != h || f.retenido.Version.Validar() != nil || f.retenido.Recibo != guardado.Recibo {
			t.Fatal("mutacion del consumidor altera version o recibo retenidos")
		}
	}
}
