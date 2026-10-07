package domain_test

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	d "vec-diputacion-granada/internal/modules/provision/domain"
)

func ejemploBOP2026(t *testing.T) (d.Configuracion, d.Entrada) {
	t.Helper()
	var c d.Configuracion
	var e d.Entrada
	for _, archivo := range []struct {
		ruta    string
		destino any
	}{
		{"../../../../cmd/vec-provision-bases/testdata/reglas_concurso_2026_ensayo.json", &c},
		{"../../../../cmd/vec-provision-bases/testdata/entrada_concurso_2026_sintetica.json", &e},
	} {
		datos, err := os.ReadFile(archivo.ruta)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(datos, archivo.destino); err != nil {
			t.Fatal(err)
		}
	}
	return c, e
}

func reglaBOP2026(t *testing.T, c d.Configuracion, familia d.Familia) d.Regla {
	t.Helper()
	for _, r := range c.Reglas {
		if r.Familia == familia {
			return r
		}
	}
	t.Fatalf("falta regla %s", familia)
	return d.Regla{}
}

func entradaTemporalBOP(t *testing.T, periodos ...d.Periodo) d.Entrada {
	t.Helper()
	_, e := ejemploBOP2026(t)
	e.Disponibles = []d.Familia{d.ValoracionTrabajo}
	e.GradoPersonal = nil
	e.Periodos = periodos
	e.Cursos = []d.Curso{}
	e.Titulaciones = []d.Titulo{}
	return e
}

func TestBOP2026CincoFamiliasCalculadasTitulacionPendiente(t *testing.T) {
	c, e := ejemploBOP2026(t)
	r := concCalcular(t, c, e)
	if c.BasesRef != "BOP-GRA-2026-060006" || c.FechaCorte.String() != "2026-05-05" || c.CoberturaRequerida != "seis_familias_concurso_v1" {
		t.Fatal("versión de bases incorrecta")
	}
	if r.Completo || r.Total != nil || len(r.Incidencias) != 1 || r.Incidencias[0] != "regla_no_configurada:titulaciones" {
		t.Fatal("suma parcial presentada como total", r)
	}
	if parte := concDesglose(t, r, d.Titulaciones); parte.Estado != "pendiente_regla" {
		t.Fatal(parte)
	}
	for familia, puntos := range map[d.Familia]int64{d.ValoracionTrabajo: 2_050_000, d.Grado: 15_000_000, d.Antiguedad: 1_020_000, d.Permanencia: 2_500_000, d.Cursos: 480_000} {
		parte := concDesglose(t, r, familia)
		if parte.Estado != "calculado" || parte.Resultado.Micropuntos() != puntos {
			t.Fatalf("%s: %s %d", familia, parte.Estado, parte.Resultado.Micropuntos())
		}
	}
}

func TestBOP2026ExperienciaAgrupaFraccionesSoloMismoNivel(t *testing.T) {
	c, _ := ejemploBOP2026(t)
	c.Reglas = []d.Regla{reglaBOP2026(t, c, d.ValoracionTrabajo)}
	c.CoberturaRequerida = ""
	crear := func(id, desde, hasta string, nivel int) d.Periodo {
		p := concPeriodo(t, id, desde, hasta, d.ValoracionTrabajo)
		p.Nivel = nivel
		return p
	}
	e := entradaTemporalBOP(t, crear("uno", "2023-01-01", "2023-08-01", 24), crear("dos", "2023-08-01", "2024-03-01", 24))
	concTotal(t, concCalcular(t, c, e), 2_050_000) // 14 meses del mismo nivel: un año.
	e.Periodos = []d.Periodo{crear("seis", "2023-01-01", "2023-07-01", 24)}
	concTotal(t, concCalcular(t, c, e), 0) // La fracción igual a seis meses no supera el umbral.
	e.Periodos = []d.Periodo{crear("siete", "2023-01-01", "2023-08-01", 24)}
	concTotal(t, concCalcular(t, c, e), 2_050_000)
	e.Periodos = []d.Periodo{crear("uno", "2023-01-01", "2023-08-01", 24), crear("dos", "2023-08-01", "2024-03-01", 24)}
	e.Periodos[0].Nivel = 25
	e.Periodos[1].Nivel = 26
	concTotal(t, concCalcular(t, c, e), 4_400_000) // Dos niveles superiores: dos fracciones >6.
}

func TestBOP2026PermanenciaCorrectorYMezclaPendiente(t *testing.T) {
	grupo := func(t *testing.T, r d.Resultado) d.Detalle {
		t.Helper()
		for _, detalle := range concDesglose(t, r, d.Permanencia).Detalles {
			if detalle.HechoID == "grupo:permanencia" {
				return detalle
			}
		}
		t.Fatal("falta grupo de permanencia")
		return d.Detalle{}
	}
	c, e := ejemploBOP2026(t)
	c.Reglas = []d.Regla{reglaBOP2026(t, c, d.Permanencia)}
	c.CoberturaRequerida = ""
	e.Disponibles = []d.Familia{d.Permanencia}
	e.GradoPersonal = nil
	e.Cursos, e.Titulaciones = []d.Curso{}, []d.Titulo{}
	e.Periodos = []d.Periodo{concPeriodo(t, "provisional", "2023-01-01", "2023-09-01", d.Permanencia)}
	e.Periodos[0].Tipo = "provisional"
	r := concCalcular(t, c, e)
	concTotal(t, r, 1_250_000)
	if detalle := grupo(t, r); detalle.FactorJornada.String() != "1/1" || detalle.CorrectorProvisional != "1/2" || detalle.Motivo != "corrector_provisional_y_resto" {
		t.Fatal("corrector provisional atribuido a jornada", detalle)
	}
	e.Periodos = []d.Periodo{concPeriodo(t, "definitivo", "2023-01-01", "2024-03-01", d.Permanencia)}
	e.Periodos[0].Tipo = "definitivo"
	r = concCalcular(t, c, e)
	concTotal(t, r, 2_500_000) // Dos meses no computables.
	if detalle := grupo(t, r); detalle.FactorJornada.String() != "1/1" || detalle.CorrectorProvisional != "" || detalle.Motivo != "resto_sin_corrector" {
		t.Fatal("se informó un corrector no aplicado", detalle)
	}
	e.Periodos = []d.Periodo{
		concPeriodo(t, "definitivo", "2023-01-01", "2023-07-01", d.Permanencia),
		concPeriodo(t, "provisional", "2023-07-01", "2024-03-01", d.Permanencia),
	}
	e.Periodos[0].Tipo, e.Periodos[1].Tipo = "definitivo", "provisional"
	r = concCalcular(t, c, e)
	if r.Completo || r.Total != nil || len(r.Incidencias) != 1 || !strings.HasPrefix(r.Incidencias[0], "politica_no_admitida:") || concDesglose(t, r, d.Permanencia).Estado != "pendiente_politica" {
		t.Fatal("mezcla produjo puntuación aceptada", r)
	}
	if detalle := grupo(t, r); detalle.FactorJornada.String() != "1/1" || detalle.CorrectorProvisional != "" || detalle.Resultado.Micropuntos() != 0 {
		t.Fatal("la mezcla pendiente aparentó aplicar el corrector", detalle)
	}
	completa, entrada := ejemploBOP2026(t)
	entrada.Periodos = e.Periodos
	for i := range entrada.Periodos {
		entrada.Periodos[i].Familias = []d.Familia{d.Permanencia}
	}
	r = concCalcular(t, completa, entrada)
	if r.Total != nil || concDesglose(t, r, d.Permanencia).Estado != "pendiente_politica" || concDesglose(t, r, d.Grado).Resultado.Micropuntos() != 15_000_000 {
		t.Fatal("mezcla perdió los demás desgloses o emitió total", r)
	}
}

func TestBOP2026VentanaExperienciaNoRecortaAntiguedad(t *testing.T) {
	c, e := ejemploBOP2026(t)
	p := concPeriodo(t, "anterior", "2010-01-01", "2011-01-01", d.Antiguedad)
	p.Tipo = "definitivo"
	e.Periodos = append(e.Periodos, p)
	r := concCalcular(t, c, e)
	if parte := concDesglose(t, r, d.Antiguedad); parte.Resultado.Micropuntos() != 2_040_000 {
		t.Fatal("la ventana de experiencia recortó antigüedad", parte)
	}
	if parte := concDesglose(t, r, d.ValoracionTrabajo); parte.Resultado.Micropuntos() != 2_050_000 {
		t.Fatal("periodo anterior entró en experiencia", parte)
	}
}

func TestBOP2026MesCivilParcialNoSeInventa(t *testing.T) {
	c, _ := ejemploBOP2026(t)
	c.Reglas = []d.Regla{reglaBOP2026(t, c, d.ValoracionTrabajo)}
	c.CoberturaRequerida = ""
	e := entradaTemporalBOP(t, concPeriodo(t, "parcial", "2023-01-01", "2023-01-20", d.ValoracionTrabajo))
	_, err := d.Calcular(c, e)
	var nominal *d.Error
	if !errors.As(err, &nominal) || nominal.Codigo != "meses_no_acreditados" || nominal.Campo != "periodos" {
		t.Fatal("se convirtió un mes parcial sin fuente", err)
	}
}

func TestBOP2026ComparadorIncluyeCoberturaVentanaYCorrector(t *testing.T) {
	anterior, _ := ejemploBOP2026(t)
	nueva := anterior
	nueva.Version = "BOP-GRA-2026-060006:ensayo-v2"
	nueva.CoberturaRequerida = ""
	nueva.Reglas = append([]d.Regla(nil), anterior.Reglas...)
	for i := range nueva.Reglas {
		switch nueva.Reglas[i].Familia {
		case d.ValoracionTrabajo:
			nueva.Reglas[i].VentanaDesde = "2017-05-05"
		case d.Permanencia:
			nueva.Reglas[i].FactorProvisionalNumerador = 1
			nueva.Reglas[i].FactorProvisionalDenominador = 3
		}
	}
	diferencia, err := d.CompararConfiguraciones(anterior, nueva)
	if err != nil {
		t.Fatal(err)
	}
	cambios := map[string]bool{}
	for _, cambio := range diferencia.Cambios {
		cambios[cambio.Campo] = true
	}
	for _, campo := range []string{"cobertura_requerida", "ventana_desde", "factor_provisional_denominador"} {
		if !cambios[campo] {
			t.Fatal("cambio omitido del comparador", campo)
		}
	}
}

func TestBOP2026ConfiguracionRechazaPoliticasYVentanasInvalidas(t *testing.T) {
	base, _ := ejemploBOP2026(t)
	for _, caso := range []struct {
		nombre string
		mutar  func(*d.Configuracion)
	}{
		{"cobertura", func(c *d.Configuracion) { c.CoberturaRequerida = "desconocida" }},
		{"ventana", func(c *d.Configuracion) { c.Reglas[0].VentanaDesde = "fecha-sin-formato" }},
		{"factor", func(c *d.Configuracion) { c.Reglas[3].FactorProvisionalNumerador = 3 }},
		{"politica_ajena", func(c *d.Configuracion) { c.Reglas[3].PermanenciaPolitica = "otra" }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			c := base
			c.Reglas = append([]d.Regla(nil), base.Reglas...)
			caso.mutar(&c)
			if err := d.ValidarConfiguracion(c); err == nil {
				t.Fatal("configuración inválida aceptada")
			}
		})
	}
}
