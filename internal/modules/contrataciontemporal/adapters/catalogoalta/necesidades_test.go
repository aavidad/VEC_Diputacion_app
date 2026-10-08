package catalogoalta

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestNecesidadesEjemploSeparaCausasYReferencia(t *testing.T) {
	c, err := CargarNecesidades("")
	if err != nil {
		t.Fatal(err)
	}
	if !c.EsEjemplo || c.Version != 1 || c.JornadaReferenciaMinutos != 2250 ||
		len(c.Causas) != 4 || len(c.HuellaSHA256) != 64 {
		t.Fatalf("publicación inesperada: versión=%d causas=%d jornada=%d", c.Version, len(c.Causas), c.JornadaReferenciaMinutos)
	}
	if c.Causas[0].Clave != "vacante" || c.Causas[1].Clave != "sustitucion" ||
		c.Causas[2].Clave != "acumulacion_tareas" || c.Causas[3].Clave != "programa_temporal" {
		t.Fatal("las causas de la circular no coinciden")
	}
	una, preseleccion, err := c.CausasAdmitidas([]domain.ClaveCatalogo{"programa_temporal"})
	if err != nil || len(una) != 1 || preseleccion != "programa_temporal" {
		t.Fatal("preselección única incorrecta", err)
	}
	una[0].CamposPermitidos[0] = "adulterado"
	if c.Causas[3].CamposPermitidos[0] == "adulterado" {
		t.Fatal("la selección comparte el catálogo mutable")
	}
	varias, preseleccion, err := c.CausasAdmitidas([]domain.ClaveCatalogo{"vacante", "sustitucion"})
	if err != nil || len(varias) != 2 || preseleccion != "" {
		t.Fatal("se preseleccionó entre varias causas", err)
	}
	if _, _, err := c.CausasAdmitidas([]domain.ClaveCatalogo{"otra"}); err == nil {
		t.Fatal("causa ajena admitida")
	}
	if _, _, err := c.CausasAdmitidas([]domain.ClaveCatalogo{"vacante", "vacante"}); err == nil {
		t.Fatal("causa repetida admitida")
	}
	alterado := c
	alterado.HuellaSHA256 = ""
	if _, _, err := alterado.CausasAdmitidas([]domain.ClaveCatalogo{"vacante"}); err == nil {
		t.Fatal("catálogo sin huella admitido")
	}
}

func TestNecesidadRechazaDatosIncompatiblesYDuracion(t *testing.T) {
	c, err := CargarNecesidades("")
	if err != nil {
		t.Fatal(err)
	}
	inicio := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	base := map[string]string{"organica_codigo": "100", "funcional_codigo": "200", "proyecto_gasto_codigo": "300", "porcentaje_financiacion": "100"}
	ligar := func(d *domain.DatosNecesidadAlta) {
		d.Esquema = "vec.ct.necesidad_alta.v1"
		d.CatalogoRef, d.CatalogoVersion, d.CatalogoHuellaSHA256 = c.Referencia, c.Version, c.HuellaSHA256
	}
	campos := func(extra map[string]string) map[string]string {
		m := make(map[string]string, len(base)+len(extra))
		for k, v := range base {
			m[k] = v
		}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}
	periodo := func(fin time.Time) domain.PeriodoPrevisto { return domain.PeriodoPrevisto{Inicio: inicio, Fin: fin} }
	vacante := domain.DatosNecesidadAlta{CausaClave: "vacante", Periodo: periodo(inicio.AddDate(1, 0, -1)), JornadaMinutos: 2250,
		Campos: campos(map[string]string{"plaza_codigo": "1201", "puesto_codigo": "3388",
			"rpt_catalogo_ref":           "rpt:dipgra:2026",
			"rpt_catalogo_huella_sha256": strings.Repeat("a", 64)})}
	ligar(&vacante)
	if err := c.ValidarDatos(vacante); err != nil {
		t.Fatal(err)
	}
	delete(vacante.Campos, "plaza_codigo")
	if err := c.ValidarDatos(vacante); err != nil {
		t.Fatal("plaza aún no cargada debe ser opcional", err)
	}
	vacante.Campos["plaza_codigo"] = "1201"
	vacante.CatalogoVersion++
	if c.ValidarDatos(vacante) == nil {
		t.Fatal("otra versión de reglas admitida")
	}
	ligar(&vacante)
	vacante.Campos["titular_ref"] = "persona:opaca:1"
	if c.ValidarDatos(vacante) == nil {
		t.Fatal("titular incompatible con vacante")
	}
	delete(vacante.Campos, "titular_ref")
	vacante.Campos["rpt_catalogo_version"] = "1"
	if c.ValidarDatos(vacante) == nil {
		t.Fatal("versión de esquema RPT confundida con publicación")
	}
	delete(vacante.Campos, "rpt_catalogo_version")
	vacante.JornadaMinutos = 2251
	if err := c.ValidarDatos(vacante); err != nil {
		t.Fatal("la jornada de referencia es precarga editable, no máximo", err)
	}
	vacante.JornadaMinutos = 10081
	if c.ValidarDatos(vacante) == nil {
		t.Fatal("jornada superior a una semana admitida")
	}
	acumulacion := domain.DatosNecesidadAlta{CausaClave: "acumulacion_tareas", Periodo: periodo(inicio.AddDate(0, 9, 0)), JornadaMinutos: 1800,
		Campos: campos(map[string]string{"justificacion_temporal": "Refuerzo limitado de un servicio."})}
	ligar(&acumulacion)
	if c.ValidarDatos(acumulacion) == nil {
		t.Fatal("exceso de nueve meses admitido")
	}
	acumulacion.Periodo.Fin = inicio.AddDate(0, 9, -1)
	if err := c.ValidarDatos(acumulacion); err != nil {
		t.Fatal(err)
	}
	acumulacion.Periodo.Inicio = time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	acumulacion.Periodo.Fin = time.Date(2027, 2, 28, 0, 0, 0, 0, time.UTC)
	if err := c.ValidarDatos(acumulacion); err != nil {
		t.Fatal("fin de febrero válido", err)
	}
	acumulacion.Periodo.Fin = time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	if c.ValidarDatos(acumulacion) == nil {
		t.Fatal("mes normalizado de Go excedió el límite")
	}
	programa := domain.DatosNecesidadAlta{CausaClave: "programa_temporal", Periodo: periodo(inicio.AddDate(1, 0, -1)), JornadaMinutos: 1800,
		Campos: campos(map[string]string{"programa_denominacion": "Programa de refuerzo temporal", "programa_fin": "2027-10-01", "proyecto_codigo": "P01", "financiacion_ref": "financiacion:opaca:1", "rc_ref": "rc:opaca:1"})}
	ligar(&programa)
	if err := c.ValidarDatos(programa); err != nil {
		t.Fatal(err)
	}
	programa.Campos["intervencion_ref"] = "intervencion:opaca:1"
	if c.ValidarDatos(programa) == nil {
		t.Fatal("dos vías de financiación admitidas")
	}
	delete(programa.Campos, "intervencion_ref")
	delete(programa.Campos, "rc_ref")
	if c.ValidarDatos(programa) == nil {
		t.Fatal("sin RC o referencia de Intervención")
	}
	programa.Campos["intervencion_ref"] = "intervencion:opaca:1"
	programa.Campos["programa_fin"] = "2027-01-01"
	if c.ValidarDatos(programa) == nil {
		t.Fatal("programa más corto que la cobertura")
	}
	sustitucion := domain.DatosNecesidadAlta{CausaClave: "sustitucion", Periodo: domain.PeriodoPrevisto{Inicio: inicio, CausaFin: "reincorporacion_titular"}, JornadaMinutos: 2250, Campos: campos(map[string]string{"puesto_codigo": "3388",
		"rpt_catalogo_ref":           "rpt:dipgra:2026",
		"rpt_catalogo_huella_sha256": strings.Repeat("a", 64)})}
	ligar(&sustitucion)
	if err := c.ValidarDatos(sustitucion); err != nil {
		t.Fatal("fin por reincorporación rechazado", err)
	}
	selladaAbierta, err := c.SellarDatos(sustitucion)
	if err != nil || selladaAbierta.Periodo.PoliticaFin.ReglaRef != c.Causas[1].ReglaRef ||
		selladaAbierta.Periodo.PoliticaFin.CatalogoHuellaSHA256 != c.HuellaSHA256 {
		t.Fatalf("fin abierto sin política versionada de necesidad: %v", err)
	}
	delete(sustitucion.Campos, "puesto_codigo")
	if c.ValidarDatos(sustitucion) == nil {
		t.Fatal("sustitución sin puesto RPT")
	}
	sustitucion.Campos["puesto_codigo"] = "3388"
	sustitucion.Campos["plaza_codigo"] = "1201"
	sustitucion.Campos["titular_ref"] = "persona:opaca:1"
	if err := c.ValidarDatos(sustitucion); err != nil {
		t.Fatal("puesto, plaza y titular nominal opcional", err)
	}
}

func TestCatalogoDeclaradoAusenteOAlteradoNoUsaEjemplo(t *testing.T) {
	if _, err := CargarNecesidades(filepath.Join(t.TempDir(), "ausente.json")); err == nil {
		t.Fatal("se recuperó el ejemplo")
	}
	ruta := filepath.Join(t.TempDir(), "modificado.json")
	if err := os.WriteFile(ruta, []byte(strings.Replace(string(necesidadesEjemplo), "2250", "0", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarNecesidades(ruta); err == nil {
		t.Fatal("jornada inválida admitida")
	}
	if err := os.WriteFile(ruta, []byte(strings.Replace(string(necesidadesEjemplo), "https://www.dipgra.es/", "https://otro.example/", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarNecesidades(ruta); err == nil {
		t.Fatal("fuente ajena admitida")
	}
}

func TestNecesidadSelladaConservaCatalogoYClonNoComparteDatos(t *testing.T) {
	c, err := CargarNecesidades("")
	if err != nil {
		t.Fatal(err)
	}
	inicio := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	d := domain.DatosNecesidadAlta{
		Esquema: "vec.ct.necesidad_alta.v1", CatalogoRef: c.Referencia,
		CatalogoVersion: c.Version, CatalogoHuellaSHA256: c.HuellaSHA256,
		CausaClave: "vacante", Periodo: domain.PeriodoPrevisto{Inicio: inicio, Fin: inicio.AddDate(1, 0, -1)},
		JornadaMinutos: 2250,
		Campos: map[string]string{"plaza_codigo": "1201", "puesto_codigo": "3388", "organica_codigo": "100",
			"rpt_catalogo_ref":           "rpt:dipgra:2026",
			"rpt_catalogo_huella_sha256": strings.Repeat("a", 64),
			"funcional_codigo":           "200", "proyecto_gasto_codigo": "300", "porcentaje_financiacion": "100"},
	}
	sellada, err := c.SellarDatos(d)
	if err != nil || len(sellada.CatalogoInstantanea) != len(c.ContenidoCanonico) {
		t.Fatal("sellado de snapshot", err)
	}
	s := domain.SolicitudCentro{
		CentroRef: "centro:prueba", ContactoRef: "contacto:prueba", CategoriaRef: "categoria:prueba",
		GrupoSubgrupo: "C2", MotivoClave: d.CausaClave, Detalle: "Necesidad temporal declarada.",
		Periodo: d.Periodo, DocumentosAdjuntos: []string{}, Necesidad: &sellada,
	}
	clon, err := s.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	clon.Necesidad.Campos["puesto_codigo"] = "9999"
	clon.Necesidad.CatalogoInstantanea[0] ^= 1
	if s.Necesidad.Campos["puesto_codigo"] != "3388" || s.Necesidad.ValidarInstantanea() != nil {
		t.Fatal("el clon modificó la solicitud original")
	}
	if clon.Validar() == nil {
		t.Fatal("instantánea adulterada admitida")
	}
}
