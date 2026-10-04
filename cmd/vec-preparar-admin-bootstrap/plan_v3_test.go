package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func materialSinteticoV3() domain.PlanBootstrapAdministracionV3 {
	v2 := materialSintetico()
	p := domain.PlanBootstrapAdministracionV3{Version: 3, PreparadoEn: v2.PreparadoEn, CaducaEn: v2.CaducaEn,
		ControlContinuidadRevisionEsperada: v2.ControlContinuidadRevisionEsperada, BootstrapEstadoEsperado: v2.BootstrapEstadoEsperado,
		Rol: v2.Rol, FuenteIdentidad: v2.FuenteIdentidad, FuenteCA: v2.FuenteCA,
		FuenteRepartoAprobado: evidencia{Referencia: "fuente:reparto:sintetico", Version: 1, HuellaSHA256: strings.Repeat("9", 64)}}
	p.Rol.VersionRef = "rol:administracion_perfiles:v4"
	ambitos := func() []domain.AmbitoBootstrapAdministracionV3 {
		return []domain.AmbitoBootstrapAdministracionV3{{Dimension: "organizacion_ref", Valores: []string{"organizacion:sintetica"},
			Fuente: evidencia{Referencia: "fuente:organizacion:sintetica", Version: 1, HuellaSHA256: strings.Repeat("8", 64)}}}
	}
	for i, persona := range v2.Personas {
		p.Personas[i] = domain.PersonaBootstrapAdministracionV3{CuentaRef: persona.CuentaRef, CuentaVersion: persona.CuentaVersion,
			PersonaRef: persona.PersonaRef, PersonaVersion: persona.PersonaVersion, PerfilRef: persona.PerfilRef, VinculoRef: persona.VinculoRef,
			PreimagenHuellaSHA256: persona.PreimagenHuellaSHA256, Procedencia: persona.Procedencia, VigenteHasta: persona.VigenteHasta,
			Certificado: persona.Certificado, Ambitos: ambitos(), Sistemas: []domain.AsignacionSistemasBootstrapAdministracionV3{}}
		for _, s := range persona.Sistemas {
			p.Personas[i].Sistemas = append(p.Personas[i].Sistemas, domain.AsignacionSistemasBootstrapAdministracionV3{
				Rol: s.Rol, PerfilRef: s.PerfilRef, VinculoRef: s.VinculoRef, VigenteHasta: s.VigenteHasta, Ambitos: ambitos()})
		}
	}
	g := v2.Gobierno
	p.Gobierno = domain.GobiernoBootstrapAdministracionV3{AudienciaSelectorADMIN: g.AudienciaSelectorADMIN, AudienciaAdministrativa: g.AudienciaAdministrativa,
		PoliticaCertificadoRef: g.PoliticaCertificadoRef, PoliticaCertificadoHuellaSHA256: g.PoliticaCertificadoHuellaSHA256, Motivos: g.Motivos}
	for i, r := range g.Roles {
		version, rolID := uint64(1), "operador_plataforma"
		if i == 0 {
			r.VersionRef, version, rolID = p.Rol.VersionRef, 4, "administracion_perfiles"
		}
		p.Gobierno.Roles = append(p.Gobierno.Roles, domain.RolGobernadoBootstrapAdministracionV3{VersionRef: r.VersionRef, HuellaSHA256: r.HuellaSHA256,
			Clase: r.Clase, CategoriaAdmin: r.CategoriaAdmin, UnidadRequerida: false,
			AmbitosFijos: []domain.AmbitoFijoBootstrapAdministracion{{Clave: "organizacion_ref", Valores: []string{"organizacion:sintetica"}}},
			VigenteDesde: r.VigenteDesde, VigenteHasta: r.VigenteHasta, DuracionPropuestaSegundos: r.DuracionPropuestaSegundos,
			FuenteCategoria:   evidencia{Referencia: r.VersionRef, Version: version, HuellaSHA256: r.HuellaSHA256},
			DimensionesAmbito: []string{"organizacion_ref"}, RolID: rolID})
	}
	return p
}

func prepararFuenteV3(t *testing.T, p domain.PlanBootstrapAdministracionV3) (string, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "privado")
	if os.Mkdir(dir, 0700) != nil {
		t.Fatal("directorio privado no disponible")
	}
	fuente, plan := filepath.Join(dir, "fuente.json"), filepath.Join(dir, "plan.json")
	b, err := json.Marshal(p)
	if err != nil || os.WriteFile(fuente, b, 0600) != nil {
		t.Fatal("fuente privada no disponible")
	}
	return fuente, plan
}

func rutaTextosV3(t *testing.T, idioma string) string {
	t.Helper()
	ruta, err := filepath.Abs(filepath.Join("..", "..", "web", "static", "textos", idioma, "admin-comprobar-perfil.json"))
	if err != nil {
		t.Fatal(err)
	}
	return ruta
}

func TestCLIPlanV3PreparaCotejaSHARealYAprobacionExterna(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			p := materialSinteticoV3()
			fuente, plan := prepararFuenteV3(t, p)
			var salida, errores bytes.Buffer
			args := []string{"-fuente", fuente, "-plan", plan, "-textos", rutaTextosV3(t, idioma)}
			if ejecutar(args, &salida, &errores) != 0 {
				t.Fatal(errores.String())
			}
			_, esperada, err := p.CanonicoYHuella()
			if err != nil || strings.TrimSpace(salida.String()) != esperada {
				t.Fatal("SHA distinta del canon real")
			}
			var diagnostico diagnosticoBootstrapV3
			if json.Unmarshal(errores.Bytes(), &diagnostico) != nil || !diagnostico.Preparado || diagnostico.Aplicado || diagnostico.AprobacionCotejada || diagnostico.Mensaje == "" {
				t.Fatal("preparación presentada como aplicada o aprobada")
			}
			if strings.Contains(salida.String()+errores.String(), p.Personas[0].PersonaRef) {
				t.Fatal("salida contiene identidad de la fuente")
			}
			original, _ := os.ReadFile(plan)
			aprobacion := filepath.Join(filepath.Dir(fuente), "aprobacion.json")
			b, _ := json.Marshal(aprobacionPrivada{HuellaPlanSHA256: esperada})
			if os.WriteFile(aprobacion, b, 0600) != nil {
				t.Fatal("aprobación no disponible")
			}
			salida.Reset()
			errores.Reset()
			if ejecutar(append(args, "-cotejar", "-aprobacion", aprobacion), &salida, &errores) != 0 {
				t.Fatal(errores.String())
			}
			if json.Unmarshal(errores.Bytes(), &diagnostico) != nil || !diagnostico.Cotejado || !diagnostico.AprobacionCotejada || diagnostico.Aplicado {
				t.Fatal("cotejo de aprobación incorrecto")
			}
			posterior, _ := os.ReadFile(plan)
			if !bytes.Equal(original, posterior) {
				t.Fatal("cotejo reescribe el plan")
			}
			b, _ = json.Marshal(aprobacionPrivada{HuellaPlanSHA256: strings.Repeat("a", 64)})
			if os.WriteFile(aprobacion, b, 0600) != nil {
				t.Fatal("aprobación divergente no disponible")
			}
			salida.Reset()
			errores.Reset()
			if ejecutar(append(args, "-cotejar", "-aprobacion", aprobacion), &salida, &errores) == 0 || salida.Len() != 0 {
				t.Fatal("aprobación de otra huella admitida")
			}
			posterior, _ = os.ReadFile(plan)
			if !bytes.Equal(original, posterior) {
				t.Fatal("aprobación divergente reescribe el plan")
			}
		})
	}
}

func TestCLIPlanV3AplicarPermaneceCerradoSinEscribirPlanNiRecibo(t *testing.T) {
	fuente, plan := prepararFuenteV3(t, materialSinteticoV3())
	var salida, errores bytes.Buffer
	recibo := filepath.Join(filepath.Dir(plan), "recibo.json")
	args := []string{"-fuente", fuente, "-plan", plan, "-textos", rutaTextosV3(t, "es"), "-aplicar", "-conexion", "sin_proveedor", "-recibo", recibo}
	if ejecutar(args, &salida, &errores) == 0 || salida.Len() != 0 {
		t.Fatal("aplicación sin autoridad admitida")
	}
	var d diagnosticoBootstrapV3
	if json.Unmarshal(errores.Bytes(), &d) != nil || d.Codigo != "provision_no_confirmada" || d.Aplicado || d.Preparado {
		t.Fatal("resultado de aplicación engañoso")
	}
	for _, ruta := range []string{plan, recibo} {
		if _, err := os.Stat(ruta); !os.IsNotExist(err) {
			t.Fatal("aplicación cerrada escribió un artefacto")
		}
	}
}

func TestCLIPlanV3DeniegaSHAAusenteDuplicadosYActorLibre(t *testing.T) {
	for _, caso := range []string{"huella_null", "campo_libre", "duplicado"} {
		t.Run(caso, func(t *testing.T) {
			p := materialSinteticoV3()
			fuente, plan := prepararFuenteV3(t, p)
			b, _ := os.ReadFile(fuente)
			switch caso {
			case "huella_null":
				b = bytes.Replace(b, []byte(`"huella_sha256":"`+p.Rol.HuellaSHA256+`"`), []byte(`"huella_sha256":null`), 1)
			case "campo_libre":
				b = append([]byte(`{"operador_login":"login_sintetico",`), b[1:]...)
			case "duplicado":
				b = append([]byte(`{"version":3,`), b[1:]...)
			}
			if os.WriteFile(fuente, b, 0600) != nil {
				t.Fatal("fuente no disponible")
			}
			var salida, errores bytes.Buffer
			if ejecutar([]string{"-fuente", fuente, "-plan", plan, "-textos", rutaTextosV3(t, "es")}, &salida, &errores) == 0 || salida.Len() != 0 {
				t.Fatal("fuente incompleta o identidad libre aceptada")
			}
			if _, err := os.Stat(plan); !os.IsNotExist(err) {
				t.Fatal("fuente rechazada produjo plan")
			}
		})
	}
}

func TestCLIPlanV3FormatoNoSeSeleccionaPorCatalogoLinguistico(t *testing.T) {
	fuente, plan := prepararFuente(t, materialSintetico())
	var salida, errores bytes.Buffer
	if ejecutar([]string{"-fuente", fuente, "-plan", plan, "-textos", rutaTextosV3(t, "es")}, &salida, &errores) == 0 {
		t.Fatal("catálogo lingüístico selecciona V3 sobre una fuente V2")
	}
	fuente, plan = prepararFuenteV3(t, materialSinteticoV3())
	if ejecutar([]string{"-fuente", fuente, "-plan", plan, "-version-plan", "2", "-textos", rutaTextosV3(t, "es")}, &salida, &errores) == 0 {
		t.Fatal("versión explícita divergente aceptada")
	}
	if ejecutar([]string{"-fuente", fuente, "-plan", plan}, &salida, &errores) != 2 {
		t.Fatal("V3 sin catálogo produce diagnóstico inventado")
	}
}
