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

var instanteServiciosCertificadosPrueba = time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)

func consultaServiciosCertificadosPrueba(t *testing.T, prefijo string) ports.ConsultaServiciosParaCertificadosV1 {
	t.Helper()
	b := solicitudFichaPropiaPrueba(t, prefijo)
	return ports.ConsultaServiciosParaCertificadosV1{Actor: b.Actor, EmpleadoRef: "emp_" + strings.Repeat("a", 24), OrganismoRef: "dipgra", Corte: b.Corte}
}

// Doble estructural: no acredita firma, consumo, SQL ni COMMIT real.
func atestacionServiciosCertificadosPrueba(t *testing.T, m domain.MaterialLectorServiciosCertificados, accion, audiencia string) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	h, _ := m.HuellaSHA256()
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), accion, m.EmpleadoRef(), h, audiencia, instanteServiciosCertificadosPrueba, instanteServiciosCertificadosPrueba.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	actor := m.Actor()
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), canon, 1, 1, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

type autorizadorServiciosCertificadosPrueba struct {
	t                 *testing.T
	accion, audiencia string
	err               error
	llamadas          int
}

func (a *autorizadorServiciosCertificadosPrueba) AutorizarServiciosParaCertificados(_ context.Context, m domain.MaterialLectorServiciosCertificados) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	if a.err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, a.err
	}
	accion, audiencia := ports.AccionServiciosParaCertificadosV1, ports.AudienciaServiciosParaCertificadosV1
	if a.accion != "" {
		accion, audiencia = a.accion, a.audiencia
	}
	return atestacionServiciosCertificadosPrueba(a.t, m, accion, audiencia), nil
}

func servicioCertificadosPrueba(ref, desde, hasta string, dias int64) ports.ServicioParaCertificadosV2 {
	return ports.ServicioParaCertificadosV2{DiasReconocidos: dias, ServicioParaCertificadosV1: ports.ServicioParaCertificadosV1{
		ServicioRef: "srv_" + ref + strings.Repeat("s", 22), RelacionRef: "rel_" + strings.Repeat("r", 24), Version: 1,
		Periodo: ports.PeriodoPersonalNominalV1{Desde: domain.FechaCivil(desde), Hasta: domain.FechaCivil(hasta)},
		Estado:  "reconocido", ClaseRef: "interinidad", ClaseVersion: 1,
		Procedencia: ports.ProcedenciaPersonalNominalV1{ActoRef: "acto:sintetico-1", FuenteRef: "fuente:registro-b2", FuenteVersion: "3", Certeza: ports.CertezaPersonalNoAcreditadaV1}}}
}

type repositorioServiciosCertificadosPrueba struct {
	err      error
	alterar  string
	llamadas int
}

func (r *repositorioServiciosCertificadosPrueba) ConsultarServiciosParaCertificados(_ context.Context, o ports.OrdenLectorServiciosCertificados) (ports.ResultadoServiciosParaCertificadosV2, error) {
	r.llamadas++
	if r.err != nil {
		return ports.ResultadoServiciosParaCertificadosV2{}, r.err
	}
	x := o.Autorizacion.ResumenCapacidad()
	huella := strings.Repeat("d", 64)
	out := ports.ResultadoServiciosParaCertificadosV2{EmpleadoRef: o.Material.EmpleadoRef(), OrganismoRef: o.Material.OrganismoRef(), Version: 3,
		Corte: o.Material.Corte(), Cobertura: ports.CoberturaPersonalNoAcreditadaV1,
		Servicios: []ports.ServicioParaCertificadosV2{servicioCertificadosPrueba("a", "2019-02-01", "2020-01-31", 364), servicioCertificadosPrueba("b", "2021-03-01", "2027-01-01", 0)},
		Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "aud_v3_" + huella[:32], DecisionRef: x.DecisionRef(), EfectoRef: x.EfectoRef(), ConsumoHuellaSHA256: huella, AuditoriaRef: "aud_v3_" + huella[:32], ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)}}
	switch r.alterar {
	case "cobertura":
		out.Cobertura = ports.CoberturaPersonalCompletaV1
	case "certeza":
		out.Servicios[0].Procedencia.Certeza = ports.CertezaPersonalAcreditadaV1
	case "orden":
		out.Servicios[0], out.Servicios[1] = out.Servicios[1], out.Servicios[0]
	case "duplicado":
		out.Servicios[1].ServicioRef = out.Servicios[0].ServicioRef
		out.Servicios[1].Periodo.Desde = out.Servicios[0].Periodo.Desde
	case "futuro":
		out.Servicios[1].Periodo.Desde = "2026-10-01"
	case "recibo":
		out.Evidencia.ReciboRef = "fichapropia:0f0e0d0c-0b0a-4908-8706-050403020100"
	case "empleado":
		out.EmpleadoRef = "emp_" + strings.Repeat("z", 24)
	case "organismo":
		out.OrganismoRef = "otro_organismo"
	case "corte":
		out.Corte.VigenteEn = "2026-01-01"
	case "version":
		out.Version = 1
	case "dias":
		out.Servicios[0].DiasReconocidos = -1
	}
	return out, nil
}

type intentosServiciosCertificadosPrueba struct {
	motivos []string
	err     error
}

func (i *intentosServiciosCertificadosPrueba) VerificarRegistroServiciosCertificados(context.Context) error {
	return nil
}
func (i *intentosServiciosCertificadosPrueba) RegistrarIntentoServiciosCertificados(_ context.Context, in ports.IntentoLectorServiciosCertificados) error {
	i.motivos = append(i.motivos, in.Motivo)
	return i.err
}

func servicioLectorCertificadosPrueba(t *testing.T, a *autorizadorServiciosCertificadosPrueba, r *repositorioServiciosCertificadosPrueba, i *intentosServiciosCertificadosPrueba) *ServicioLectorServiciosCertificados {
	t.Helper()
	s, err := NuevoServicioLectorServiciosCertificados(a, r, i, func() time.Time { return instanteServiciosCertificadosPrueba.Add(time.Second) })
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestLectorServiciosCertificadosDevuelveV2ConDiasYV1SinEllos(t *testing.T) {
	a := &autorizadorServiciosCertificadosPrueba{t: t}
	r := &repositorioServiciosCertificadosPrueba{}
	i := &intentosServiciosCertificadosPrueba{}
	s := servicioLectorCertificadosPrueba(t, a, r, i)
	v2, err := s.ConsultarServiciosParaCertificadosV2(context.Background(), consultaServiciosCertificadosPrueba(t, "pep_"))
	if err != nil || len(v2.Servicios) != 2 || v2.Servicios[0].DiasReconocidos != 364 || v2.Cobertura != ports.CoberturaPersonalNoAcreditadaV1 || len(i.motivos) != 0 {
		t.Fatalf("V2: %+v, %v", v2, err)
	}
	v1, err := s.ConsultarServiciosParaCertificados(context.Background(), consultaServiciosCertificadosPrueba(t, "pep_"))
	if err != nil || len(v1.Servicios) != 2 || v1.Servicios[0].ServicioRef != v2.Servicios[0].ServicioRef || v1.Servicios[1].Periodo.Hasta != "2027-01-01" {
		t.Fatalf("V1: %+v, %v", v1, err)
	}
	if a.llamadas != 2 || r.llamadas != 2 {
		t.Fatal("cada lectura necesita su propia concesión y transacción")
	}
}

func TestLectorServiciosCertificadosSoloAutoservicio(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		mutar   func(*ports.ConsultaServiciosParaCertificadosV1)
		prefijo string
		err     error
		motivo  string
	}{
		{"empleado_ajeno", func(c *ports.ConsultaServiciosParaCertificadosV1) { c.EmpleadoRef = "emp_" + strings.Repeat("b", 24) }, "pep_", domain.ErrLectorServiciosCertificadosDenegado, "denegado"},
		{"vinculo_heredado", func(*ports.ConsultaServiciosParaCertificadosV1) {}, "vin_", domain.ErrLectorServiciosCertificadosDenegado, "denegado"},
		{"organismo_invalido", func(c *ports.ConsultaServiciosParaCertificadosV1) { c.OrganismoRef = "Dipgra" }, "pep_", domain.ErrLectorServiciosCertificadosInvalido, "entrada_invalida"},
		{"corte_futuro", func(c *ports.ConsultaServiciosParaCertificadosV1) {
			c.Corte.ConocidoEn = instanteServiciosCertificadosPrueba.Add(time.Hour)
		}, "pep_", domain.ErrLectorServiciosCertificadosDenegado, "denegado"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			a := &autorizadorServiciosCertificadosPrueba{t: t}
			r := &repositorioServiciosCertificadosPrueba{}
			i := &intentosServiciosCertificadosPrueba{}
			c := consultaServiciosCertificadosPrueba(t, caso.prefijo)
			caso.mutar(&c)
			out, err := servicioLectorCertificadosPrueba(t, a, r, i).ConsultarServiciosParaCertificadosV2(context.Background(), c)
			if !errors.Is(err, caso.err) || out.Servicios != nil || a.llamadas != 0 || r.llamadas != 0 || len(i.motivos) != 1 || i.motivos[0] != caso.motivo {
				t.Fatalf("%v, autorizador=%d repositorio=%d intentos=%v", err, a.llamadas, r.llamadas, i.motivos)
			}
		})
	}
}

func TestLectorServiciosCertificadosNoReutilizaOtraConcesion(t *testing.T) {
	for _, caso := range []struct{ nombre, accion, audiencia string }{
		{"ficha_propia", domain.AccionFichaPropia, domain.AudienciaFichaPropia},
		{"historia_servicios", domain.AccionHistoriaServiciosPropia, domain.AudienciaHistoriaServiciosPropia},
		{"audiencia_ajena", ports.AccionServiciosParaCertificadosV1, domain.AudienciaFichaPropia},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			a := &autorizadorServiciosCertificadosPrueba{t: t, accion: caso.accion, audiencia: caso.audiencia}
			r := &repositorioServiciosCertificadosPrueba{}
			i := &intentosServiciosCertificadosPrueba{}
			_, err := servicioLectorCertificadosPrueba(t, a, r, i).ConsultarServiciosParaCertificadosV2(context.Background(), consultaServiciosCertificadosPrueba(t, "pep_"))
			if !errors.Is(err, domain.ErrLectorServiciosCertificadosNoDisponible) || r.llamadas != 0 || len(i.motivos) != 1 {
				t.Fatalf("%v, repositorio=%d", err, r.llamadas)
			}
		})
	}
}

func TestLectorServiciosCertificadosRechazaRespuestaNoLigada(t *testing.T) {
	for _, caso := range []string{"cobertura", "certeza", "orden", "duplicado", "futuro", "recibo", "empleado", "organismo", "corte", "version", "dias"} {
		t.Run(caso, func(t *testing.T) {
			i := &intentosServiciosCertificadosPrueba{}
			out, err := servicioLectorCertificadosPrueba(t, &autorizadorServiciosCertificadosPrueba{t: t}, &repositorioServiciosCertificadosPrueba{alterar: caso}, i).
				ConsultarServiciosParaCertificadosV2(context.Background(), consultaServiciosCertificadosPrueba(t, "pep_"))
			if !errors.Is(err, domain.ErrLectorServiciosCertificadosNoDisponible) || out.Servicios != nil || len(i.motivos) != 1 || i.motivos[0] != "no_disponible" {
				t.Fatalf("%v, %v", err, i.motivos)
			}
		})
	}
}

func TestLectorServiciosCertificadosErroresNominalesYAcuse(t *testing.T) {
	for _, caso := range []struct {
		nombre      string
		repo, pdp   error
		acuse       error
		esperado    error
		motivo      string
		repoLlamado bool
	}{
		{"pdp_deniega", nil, domain.ErrLectorServiciosCertificadosDenegado, nil, domain.ErrLectorServiciosCertificadosDenegado, "denegado", false},
		{"pdp_caido", nil, errors.New("detalle privado"), nil, domain.ErrLectorServiciosCertificadosNoDisponible, "no_disponible", false},
		{"fuente_deniega", domain.ErrLectorServiciosCertificadosDenegado, nil, nil, domain.ErrLectorServiciosCertificadosDenegado, "denegado", true},
		{"excede", domain.ErrLectorServiciosCertificadosExcedeLimite, nil, nil, domain.ErrLectorServiciosCertificadosExcedeLimite, "no_disponible", true},
		{"sin_acuse", domain.ErrLectorServiciosCertificadosDenegado, nil, errors.New("privado"), domain.ErrLectorServiciosCertificadosNoDisponible, "denegado", true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			a := &autorizadorServiciosCertificadosPrueba{t: t, err: caso.pdp}
			r := &repositorioServiciosCertificadosPrueba{err: caso.repo}
			i := &intentosServiciosCertificadosPrueba{err: caso.acuse}
			_, err := servicioLectorCertificadosPrueba(t, a, r, i).ConsultarServiciosParaCertificadosV2(context.Background(), consultaServiciosCertificadosPrueba(t, "pep_"))
			if !errors.Is(err, caso.esperado) || (r.llamadas == 1) != caso.repoLlamado || len(i.motivos) != 1 || i.motivos[0] != caso.motivo {
				t.Fatalf("%v, repositorio=%d intentos=%v", err, r.llamadas, i.motivos)
			}
		})
	}
}

func TestLectorServiciosCertificadosV1NoComparteMemoria(t *testing.T) {
	r := ports.ResultadoServiciosParaCertificadosV2{Servicios: []ports.ServicioParaCertificadosV2{servicioCertificadosPrueba("a", "2019-02-01", "2020-01-31", 10)}}
	v1 := r.V1()
	v1.Servicios[0].Estado = "declarado"
	if r.Servicios[0].Estado != "reconocido" {
		t.Fatal("V1 comparte la lista de V2")
	}
}
