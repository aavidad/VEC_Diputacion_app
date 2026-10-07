package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

type repositorioPreparacionCarreraPrueba struct {
	repositorioPreparacionRPTPrueba
}

func (r repositorioPreparacionCarreraPrueba) ConsultarFichaRRHH(ctx context.Context, o ports.OrdenFichaEmpleadoB2) (ports.ResultadoFichaEmpleadoB2, error) {
	resultado, err := r.repositorioPreparacionRPTPrueba.ConsultarFichaRRHH(ctx, o)
	if err != nil {
		return resultado, err
	}
	// El orden de versiones difiere del orden de conocimiento. La relación
	// finalizada conocida más tarde debe prevalecer sobre la antigua v9.
	resultado.Ficha.Relaciones[1].Traza.Version = 9
	traza := resultado.Ficha.Relaciones[0].Traza
	traza.Hasta = ""
	traza.Version = 9
	traza.RegistradaEn = o.Material.Corte().ConocidoEn.Add(-3 * time.Hour)
	anterior := domain.ServicioReconocidoB2{ServicioRef: "servicio:carrera", RelacionRef: resultado.Ficha.Relaciones[0].RelacionRef, Estado: "reconocido", ClaseRef: "clase:uno", PeriodoDesde: "2020-01-01", PeriodoHasta: "2020-01-05", DiasReconocidos: 5, Traza: traza,
		CatalogoSnapshot: domain.SnapshotCatalogoEmpleadoB2{ClaseServicio: &domain.SnapshotEntradaCatalogoEmpleadoB2{OrganismoRef: o.Material.OrganismoRef(), Tipo: "clase_servicio", Ref: "clase:uno", Version: 1, Revision: 1, Denominacion: "Servicio sintético", HuellaSHA256: strings.Repeat("a", 64), VigenteDesde: "2020-01-01", Estado: "publicada"}}}
	ultima := anterior
	ultima.Estado = "declarado"
	ultima.Traza.Version = 1
	ultima.Traza.RegistradaEn = o.Material.Corte().ConocidoEn.Add(-2 * time.Hour)
	resultado.Ficha.Servicios = []domain.ServicioReconocidoB2{anterior, ultima}
	if r.invalido == "proyeccion" {
		resultado.Ficha.Servicios[0].RelacionRef = "rel_" + strings.Repeat("z", 24)
	}
	resultado.PreparacionCarrera = &domain.PreparacionAntecedentesCarrera{Alcance: "acreditado"}
	return resultado, nil
}

func TestRegistroB2PreparacionCarreraTrasV3SinPromocionarAntecedentes(t *testing.T) {
	for _, caso := range []string{"valida", "denegada", "evidencia", "ficha", "firma", "proyeccion"} {
		t.Run(caso, func(t *testing.T) {
			var pasos []string
			actor := solicitudP(t).Actor
			repo := repositorioPreparacionCarreraPrueba{repositorioPreparacionRPTPrueba{repositorioPreparacionCERPrueba{pasos: &pasos, invalido: caso}}}
			s, err := NuevoServicioRegistroEmpleadoB2(autorizadorPreparacionCERPrueba{t: t, denegar: caso == "denegada", pasos: &pasos}, repo)
			if err != nil {
				t.Fatal(err)
			}
			resultado, err := s.ConsultarFicha(context.Background(), domain.SolicitudFichaEmpleadoB2{Actor: actor, OrganismoRef: "organismo:uno", EmpleadoRef: "emp_" + strings.Repeat("e", 24), Corte: domain.CorteEmpleadoB2{VigenteEn: "2026-09-25", ConocidoEn: actor.ResueltoEn}})
			if caso != "valida" {
				if err == nil || resultado.PreparacionCarrera != nil || resultado.PreparacionServicios != nil || resultado.PreparacionRPT != nil || resultado.Ficha.EmpleadoRef != "" {
					t.Fatalf("datos tras fallo: %+v,%v", resultado, err)
				}
				if caso == "denegada" && (len(pasos) != 1 || !errors.Is(err, domain.ErrRegistroEmpleadoB2Denegado)) {
					t.Fatalf("lectura tras denegación: %v,%v", pasos, err)
				}
				return
			}
			p := resultado.PreparacionCarrera
			if err != nil || p == nil || resultado.PreparacionServicios == nil || resultado.PreparacionRPT == nil || len(pasos) != 2 || pasos[1] != "lectura_v3_auditada" {
				t.Fatalf("preparación sin lectura: %+v,%v", resultado, err)
			}
			if p.EmpleadoRef != resultado.Ficha.EmpleadoRef || p.Version != resultado.Ficha.Version || p.Corte != resultado.Ficha.Corte || p.Alcance != "preparacion" || p.Esquema != domain.EsquemaAntecedentesCarrera || len(p.Pendientes) < 3 {
				t.Fatalf("vínculo o límites: %+v", p)
			}
			if len(p.Relaciones) != 1 || p.Relaciones[0].Estado != "finalizada" || p.Relaciones[0].Traza.Version != 2 || len(p.Relaciones[0].Historia) != 2 {
				t.Fatalf("revivió relación antigua: %+v", p.Relaciones)
			}
			r := p.Relaciones[0]
			if len(r.Servicios) != 2 || r.Servicios[0].UltimaRevisionConocida || !r.Servicios[1].UltimaRevisionConocida || r.Servicios[1].Estado != "declarado" || r.Servicios[0].DiasReconocidos != 5 {
				t.Fatalf("promovió o sumó servicios: %+v", r.Servicios)
			}
			pendiente := false
			for _, clave := range r.Pendientes {
				if clave == "servicios_no_reconocidos" {
					pendiente = true
				}
			}
			if !pendiente {
				t.Fatal("desapareció pendiente de reconocimiento")
			}
		})
	}
}
