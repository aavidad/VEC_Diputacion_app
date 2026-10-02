package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func planBootstrapPGSintetico() domain.PlanBootstrapAdministracionV2 {
	h := func(s string) string { return strings.Repeat(s, 64) }
	ref := func(prefijo, c string) string { return prefijo + strings.Repeat(c, 24) }
	e := func(nombre, c string) domain.EvidenciaBootstrapAdministracion {
		return domain.EvidenciaBootstrapAdministracion{Referencia: nombre, Version: 1, HuellaSHA256: h(c)}
	}
	p := domain.PersonaBootstrapAdministracion{CuentaRef: ref("cta_", "a"), CuentaVersion: 2, PersonaRef: ref("per_", "a"), PersonaVersion: 3,
		PerfilRef: ref("prf_", "a"), VinculoRef: ref("vca_", "a"), PreimagenHuellaSHA256: h("a"), Procedencia: e("fuente:identidad:uno", "b"), VigenteHasta: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		Certificado: domain.CertificadoBootstrapAdministracion{PersonaRef: ref("per_", "a"), CuentaRef: ref("cta_", "a"), HuellaSHA256: h("c"), CAHuellaSHA256: h("d"), Acreditacion: e("cert:uno", "e")}}
	p2 := p
	p2.CuentaRef = ref("cta_", "z")
	p2.PersonaRef = ref("per_", "z")
	p2.PerfilRef = ref("prf_", "z")
	p2.VinculoRef = ref("vca_", "z")
	p2.Certificado.PersonaRef = p2.PersonaRef
	p2.Certificado.CuentaRef = p2.CuentaRef
	p2.Certificado.HuellaSHA256 = h("f")
	return domain.PlanBootstrapAdministracionV2{Version: 2, PreparadoEn: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC), CaducaEn: time.Date(2026, 10, 2, 13, 0, 0, 0, time.UTC),
		ControlContinuidadRevisionEsperada: 1, BootstrapEstadoEsperado: "pendiente", Rol: domain.RolBootstrapAdministracion{VersionRef: "rol:administracion_perfiles:v3", HuellaSHA256: h("1"), ControlRevision: 1, ControlHuellaSHA256: h("2")},
		FuenteIdentidad: e("autoridad:identidad", "3"), FuenteCA: e("autoridad:ca", "d"), Personas: [2]domain.PersonaBootstrapAdministracion{p, p2},
		Gobierno: domain.GobiernoBootstrapAdministracion{AudienciaAdministrativa: "admin:ensayo:v1", PoliticaCertificadoRef: "politica:admin:ejemplo", PoliticaCertificadoHuellaSHA256: h("4"),
			Roles: []domain.RolGobernadoBootstrapAdministracion{{VersionRef: "rol:administracion_perfiles:v3", HuellaSHA256: h("1"), Clase: domain.ClaseControlPerfilAdministrador,
				AmbitosFijos: []domain.AmbitoFijoBootstrapAdministracion{{Clave: "unidad", Valores: []string{"unidad:ejemplo"}}}, VigenteDesde: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
				VigenteHasta: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), DuracionPropuestaSegundos: 300}}}}
}

func TestBootstrapApruebaHuellaExactaAntesDeSQL(t *testing.T) {
	plan := planBootstrapPGSintetico()
	preimagen, err := plan.Preimagen()
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []string{"hash_ajeno", "plan_v1", "config_alterada", "pool_ajeno"} {
		t.Run(caso, func(t *testing.T) {
			pool := &poolFalso{fila: filaFalsa{dato: caso != "pool_ajeno"}}
			aprobacion := preimagen.HuellaPlanSHA256
			if caso == "hash_ajeno" {
				aprobacion = strings.Repeat("f", 64)
			}
			proveedor, e := nuevoProvisionadorBootstrap(context.Background(), pool, aprobacion, relojFijo(plan.PreparadoEn.Add(time.Minute)))
			if caso == "pool_ajeno" {
				if e == nil {
					t.Fatal("pool no acreditado aceptado")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			p := preimagen
			if caso == "plan_v1" {
				p.PlanV2 = nil
			}
			if caso == "config_alterada" {
				copia := plan
				copia.Gobierno.PoliticaCertificadoRef = "politica:ajena"
				p.PlanV2 = &copia
			}
			if _, e = proveedor.ProvisionarDosAdministradoresIniciales(context.Background(), p); e == nil || pool.comienzos != 0 {
				t.Fatal("plan no aprobado llega a SQL")
			}
		})
	}
}

func TestBootstrapReciboSeCompruebaAntesDeConfirmar(t *testing.T) {
	plan := planBootstrapPGSintetico()
	preimagen, err := plan.Preimagen()
	if err != nil {
		t.Fatal(err)
	}
	r := reciboBootstrapJSON{ActoRef: "acto_admin:" + strings.Repeat("a", 32), ReciboRef: "recibo_admin:" + strings.Repeat("b", 32),
		HuellaPlanSHA256: preimagen.HuellaPlanSHA256, PrimeraPersonaRef: preimagen.Primera.PersonaRef, SegundaPersonaRef: preimagen.Segunda.PersonaRef, ConfirmadoEn: plan.PreparadoEn.Add(time.Minute)}
	for _, caso := range []string{"confirmado", "recibo_ajeno", "commit_incierto", "replay_tras_caducidad"} {
		t.Run(caso, func(t *testing.T) {
			x := r
			if caso == "recibo_ajeno" {
				x.SegundaPersonaRef = preimagen.Primera.PersonaRef
			}
			b, _ := json.Marshal(x)
			tx := &txFalsa{fila: filaFalsa{dato: b}}
			if caso == "commit_incierto" {
				tx.falloCommit = errors.New("fallo privado")
			}
			pool := &poolFalso{tx: tx, fila: filaFalsa{dato: true}}
			ahora := plan.PreparadoEn.Add(time.Minute)
			if caso == "replay_tras_caducidad" {
				ahora = plan.CaducaEn.Add(time.Hour)
			}
			proveedor, e := nuevoProvisionadorBootstrap(context.Background(), pool, preimagen.HuellaPlanSHA256, relojFijo(ahora))
			if e != nil {
				t.Fatal(e)
			}
			resultado, e := proveedor.ProvisionarDosAdministradoresIniciales(context.Background(), preimagen)
			if (e == nil) != (caso == "confirmado" || caso == "replay_tras_caducidad") {
				t.Fatalf("err=%v", e)
			}
			if caso == "recibo_ajeno" && tx.commits != 0 {
				t.Fatal("confirma recibo ajeno")
			}
			if e != nil && resultado.ReciboRef != "" {
				t.Fatal("expone recibo sin confirmación")
			}
			if tx.rollbacks != 1 {
				t.Fatal("no cierra transacción")
			}
		})
	}
}
