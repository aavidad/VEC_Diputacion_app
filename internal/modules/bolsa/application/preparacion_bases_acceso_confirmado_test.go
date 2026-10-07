package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/ports"
)

type autorizadorAccesoBasesPrueba struct {
	material core.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (a autorizadorAccesoBasesPrueba) AutorizarGuardadoPreparacionBases(context.Context, ports.SolicitudGuardarPreparacionBasesV3) (core.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return a.material, nil
}
func (a autorizadorAccesoBasesPrueba) AutorizarConsultaPreparacionBases(context.Context, ports.SolicitudConsultarPreparacionBasesV3) (core.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return a.material, nil
}

type repositorioAccesoBasesPrueba struct {
	resultado ports.ResultadoPreparacionBasesV3
	err       error
}

func (r repositorioAccesoBasesPrueba) GuardarPreparacionBasesV3(context.Context, ports.OrdenGuardarPreparacionBasesV3) (ports.ResultadoPreparacionBasesV3, error) {
	return r.resultado, r.err
}
func (r repositorioAccesoBasesPrueba) ConsultarPreparacionBasesV3(context.Context, ports.OrdenConsultarPreparacionBasesV3) (ports.ResultadoPreparacionBasesV3, error) {
	return r.resultado, r.err
}

func TestPreparacionBasesMarcaAccesoSoloConReciboConfirmado(t *testing.T) {
	for _, estado := range []string{"version_en_conflicto", "clave_reutilizada", "no_encontrada"} {
		for _, confirmado := range []bool{true, false} {
			q := solicitudBasesV3Prueba(t)
			consulta := ports.SolicitudConsultarPreparacionBasesV3{Actor: q.Actor, Correlacion: q.Correlacion, Ambito: q.Ambito, Selector: ports.SelectorConsultaPreparacionBases{Modo: "actual", Exacta: q.Esperada}}
			preparacion, err := PrepararGuardadoPreparacionBasesV3(q)
			if err != nil {
				t.Fatal(err)
			}
			accion, audiencia := ports.AccionGuardarPreparacionBases, AudienciaGuardarPreparacionBasesV3
			esperado := ports.ErrPreparacionBasesConflicto
			if estado == "clave_reutilizada" {
				esperado = ports.ErrPreparacionBasesClaveReutilizada
			}
			if estado == "no_encontrada" {
				preparacion, err = PrepararConsultaPreparacionBasesV3(consulta)
				if err != nil {
					t.Fatal(err)
				}
				accion, audiencia = ports.AccionConsultarPreparacionBases, AudienciaConsultarPreparacionBasesV3
				esperado = ports.ErrPreparacionBasesNoEncontrada
			}
			material := exportacionEstructuralBasesPrueba(t, preparacion, q.Actor, accion, audiencia)
			corr, _ := q.Correlacion.ValorCanonico()
			r := repositorioAccesoBasesPrueba{resultado: ports.ResultadoPreparacionBasesV3{Estado: estado, Acceso: ports.EvidenciaAccesoPreparacionBasesV3{DecisionRef: material.ResumenCapacidad().DecisionRef(),
				ConsumoHuellaSHA256: strings.Repeat("d", 64), AuditoriaRef: "auditoria:acceso", ReciboRef: "recibo:acceso", CorrelacionRef: corr, AccedidaEn: q.Actor.ResueltoEn.Add(time.Second)}}}
			if !confirmado {
				r.err = esperado
			}
			s, err := NuevoServicioPreparacionBasesV3(autorizadorAccesoBasesPrueba{material}, r)
			if err != nil {
				t.Fatal(err)
			}
			if estado == "no_encontrada" {
				_, err = s.Consultar(context.Background(), consulta)
			} else {
				_, err = s.Guardar(context.Background(), q)
			}
			if !errors.Is(err, esperado) || AccesoConfirmadoPreparacionBasesV3(err) != confirmado {
				t.Fatalf("estado=%s confirmado=%t err=%v", estado, confirmado, err)
			}
		}
	}
}
