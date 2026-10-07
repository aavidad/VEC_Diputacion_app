package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type autorizadorPreparacionCERPrueba struct {
	t       *testing.T
	denegar bool
	pasos   *[]string
}

func (a autorizadorPreparacionCERPrueba) AutorizarConsultaRegistroEmpleadoB2(_ context.Context, m domain.MaterialConsultaRegistroEmpleadoB2) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	*a.pasos = append(*a.pasos, "autorizacion")
	if a.denegar {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, domain.ErrRegistroEmpleadoB2Denegado
	}
	h, err := m.HuellaSHA256()
	if err != nil {
		a.t.Fatal(err)
	}
	ahora := m.Actor().ResueltoEn
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_cer_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_cer_prueba", strings.Repeat("c", 64), domain.AccionFichaEmpleadoB2, m.EmpleadoRef(), h, domain.AudienciaFichaEmpleadoB2, ahora, ahora.Add(3*time.Second))
	if err != nil {
		a.t.Fatal(err)
	}
	raiz, err := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if err != nil {
		a.t.Fatal(err)
	}
	actor := m.Actor()
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		a.t.Fatal(err)
	}
	valor, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), canon, actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		a.t.Fatal(err)
	}
	return valor, nil
}

type repositorioPreparacionCERPrueba struct {
	pasos    *[]string
	invalido string
}

func (r repositorioPreparacionCERPrueba) ConsultarFichaRRHH(_ context.Context, o ports.OrdenFichaEmpleadoB2) (ports.ResultadoFichaEmpleadoB2, error) {
	*r.pasos = append(*r.pasos, "lectura_v3_auditada")
	f := domain.FichaEmpleadoB2{EmpleadoRef: o.Material.EmpleadoRef(), OrganismoRef: o.Material.OrganismoRef(), PersonaRef: "per_" + strings.Repeat("p", 24), Version: 3, Corte: o.Material.Corte(), Servicios: []domain.ServicioReconocidoB2{}}
	x := o.Autorizacion.ResumenCapacidad()
	e := ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "recibo:cer", DecisionRef: x.DecisionRef(), EfectoRef: x.EfectoRef(), AuditoriaRef: "auditoria:cer", ConsumoHuellaSHA256: strings.Repeat("d", 64), ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)}
	if r.invalido == "evidencia" {
		e.DecisionRef = "dec:ajena"
	}
	if r.invalido == "ficha" {
		f.EmpleadoRef = "emp_" + strings.Repeat("z", 24)
	}
	if r.invalido == "firma" {
		f.FirmaOficial = true
	}
	// El repositorio no es autoridad de la preparación: la aplicación la deriva
	// de los hechos comprobados y nunca conserva una proyección inyectada.
	return ports.ResultadoFichaEmpleadoB2{Ficha: f, Evidencia: e, PreparacionServicios: &domain.PreparacionServiciosParaCertificados{Cobertura: "completa"}}, nil
}
func (r repositorioPreparacionCERPrueba) ListarVacantesRRHH(context.Context, ports.OrdenVacantesB2) (ports.ResultadoVacantesB2, error) {
	return ports.ResultadoVacantesB2{}, errors.New("consulta inesperada")
}

func TestRegistroB2PreparacionCERDespuesDeLecturaV3Validada(t *testing.T) {
	for _, caso := range []string{"valida", "denegada", "evidencia", "ficha", "firma"} {
		t.Run(caso, func(t *testing.T) {
			var pasos []string
			actor := solicitudP(t).Actor
			s, err := NuevoServicioRegistroEmpleadoB2(autorizadorPreparacionCERPrueba{t: t, denegar: caso == "denegada", pasos: &pasos}, repositorioPreparacionCERPrueba{pasos: &pasos, invalido: caso})
			if err != nil {
				t.Fatal(err)
			}
			resultado, err := s.ConsultarFicha(context.Background(), domain.SolicitudFichaEmpleadoB2{Actor: actor, OrganismoRef: "organismo:uno", EmpleadoRef: "emp_" + strings.Repeat("e", 24), Corte: domain.CorteEmpleadoB2{VigenteEn: "2026-09-25", ConocidoEn: actor.ResueltoEn}})
			if caso != "valida" {
				if err == nil || resultado.PreparacionServicios != nil || resultado.Ficha.EmpleadoRef != "" {
					t.Fatalf("filtró ficha o preparación: %+v, %v", resultado, err)
				}
				if caso == "denegada" && (len(pasos) != 1 || !errors.Is(err, domain.ErrRegistroEmpleadoB2Denegado)) {
					t.Fatalf("leyó tras denegación: %v, %v", pasos, err)
				}
				return
			}
			p := resultado.PreparacionServicios
			if err != nil || len(pasos) != 2 || pasos[0] != "autorizacion" || pasos[1] != "lectura_v3_auditada" || p == nil {
				t.Fatalf("preparación sin lectura V3: %v, %v", pasos, err)
			}
			if p.EmpleadoRef != resultado.Ficha.EmpleadoRef || p.Corte != resultado.Ficha.Corte || p.Version != resultado.Ficha.Version || p.Cobertura != "no_acreditada" || p.Estado != "preparacion_sintetica" || p.FirmaOficial || p.EficaciaAdministrativa {
				t.Fatalf("preparación ajena o eficaz: %+v", p)
			}
		})
	}
}
