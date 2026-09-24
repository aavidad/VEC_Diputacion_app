package lecturaincorporacion

import (
	"strings"
	"testing"
	"time"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

func TestMaterialV2CertificadoDesarrolloExigePoliticaExactaYVigente(t *testing.T) {
	ahora := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	r, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora,
		"per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl",
		core.AuthMethodCertificate, core.AuthAssuranceSubstantial)
	if err != nil {
		t.Fatal(err)
	}
	v, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ctx := ct.ContextoAutorizacionAltaV3{Vinculo: vinculo, Resultado: r}
	s := Selector{OrganizacionRef: "org_sintetica", SolicitudRef: "sol_sintetica", ExpedienteRef: "exp_sintetico",
		VersionExpediente: 1, ResultadoRef: "resultado_sintetico", ReciboRef: "recibo_sintetico",
		RelacionRef: "relacion_sintetica", OcupacionRef: "ocupacion_sintetica", MaterialSHA256: strings.Repeat("a", 64)}
	p := PoliticaConsultaV2{Tipo: httpseguridad.PoliticaInternaDesarrolloCertificadoPersonal,
		Referencia: v.PoliticaGarantiaRef, HuellaSHA256: v.PoliticaGarantiaHuellaSHA256,
		RetiradaEn: ahora.Add(time.Hour)}
	if _, err := NuevoMaterialV2(s, "unidad:rrhh", ctx, ahora); err == nil {
		t.Fatal("constructor histórico admitió garantía sustancial")
	}
	m, err := NuevoMaterialV2ConPolitica(s, "unidad:rrhh", ctx, ahora, p)
	if err != nil {
		t.Fatalf("política exacta rechazada: %v", err)
	}
	if !garantiaLecturaAdmitida(v, AudienciaV2, m.politica, ahora) ||
		garantiaLecturaAdmitida(v, Audiencia, m.politica, ahora) ||
		garantiaLecturaAdmitida(v, AudienciaV2, m.politica, p.RetiradaEn) {
		t.Fatal("alcance o retirada de la política incorrectos")
	}
	for nombre, alterar := range map[string]func(*PoliticaConsultaV2){
		"tipo":       func(p *PoliticaConsultaV2) { p.Tipo = "otra" },
		"referencia": func(p *PoliticaConsultaV2) { p.Referencia = "pga_otra23456789abcdefghijkl" },
		"huella":     func(p *PoliticaConsultaV2) { p.HuellaSHA256 = strings.Repeat("b", 64) },
		"retirada":   func(p *PoliticaConsultaV2) { p.RetiradaEn = ahora },
	} {
		t.Run(nombre, func(t *testing.T) {
			copia := p
			alterar(&copia)
			if _, err := NuevoMaterialV2ConPolitica(s, "unidad:rrhh", ctx, ahora, copia); err == nil {
				t.Fatal("política alterada admitida")
			}
		})
	}
}
