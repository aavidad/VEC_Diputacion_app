package datospersonales_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/fichero"
	"vec-diputacion-granada/internal/vec/datospersonales"
)

const (
	rutaPaqueteEjemplo = "../../../data/demo/reglas/aspirantes_datos_personales.ejemplo.demo.json"
	rutaTextosES       = "../../../web/static/textos/es/datos-personales.json"
)

type relojFijo struct{}

func (relojFijo) Ahora() time.Time { return time.Date(2026, 9, 29, 9, 0, 0, 0, time.UTC) }

func resolutorDesde(t *testing.T, ruta string) *datospersonales.Resolutor {
	t.Helper()
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		t.Fatalf("el adaptador rechaza el paquete: %v", err)
	}
	resolutor, err := datospersonales.NuevoResolutor(consulta, consulta, relojFijo{})
	if err != nil {
		t.Fatal(err)
	}
	return resolutor
}

func catalogoEjemplo(t *testing.T) datospersonales.Catalogo {
	t.Helper()
	catalogo, err := resolutorDesde(t, rutaPaqueteEjemplo).Catalogo(context.Background())
	if err != nil {
		t.Fatalf("el paquete de ejemplo no cumple el contrato: %v", err)
	}
	return catalogo
}

func TestPaqueteEjemploCumpleElContratoYEstaRotulado(t *testing.T) {
	catalogo := catalogoEjemplo(t)
	if !catalogo.PaqueteEjemplo || catalogo.CatalogoID != datospersonales.CatalogoDatosPersonales ||
		catalogo.Version != 1 || len(catalogo.HuellaCatalogo) != 64 {
		t.Fatalf("procedencia inesperada: %+v", catalogo)
	}
	for _, dato := range catalogo.Datos {
		if !dato.EsEjemplo() || dato.PendienteDe == "" {
			t.Errorf("%s no está rotulado como pendiente de RRHH/DPD", dato.Clave)
		}
	}
	for _, tipo := range datospersonales.TiposConvocatoria() {
		if len(catalogo.Para(tipo, datospersonales.MomentoInscripcion)) == 0 {
			t.Errorf("%s no pide nada en la inscripción", tipo)
		}
	}
}

// Valores provisionales del apartado 2 del estudio que el paquete debe
// reflejar. Si RRHH o el DPD los cambian, cambia el paquete y esta prueba.
func TestPaqueteEjemploReflejaElEstudio(t *testing.T) {
	catalogo := catalogoEjemplo(t)
	casos := []struct {
		tipo           datospersonales.TipoConvocatoria
		momento        datospersonales.Momento
		dato           string
		obligatoriedad datospersonales.Obligatoriedad
		fuente         datospersonales.Fuente
		categoria      datospersonales.Categoria
	}{
		{datospersonales.TipoBolsa, datospersonales.MomentoInscripcion, "telefono", datospersonales.Obligatorio, datospersonales.FuentePersona, datospersonales.CategoriaOrdinaria},
		{datospersonales.TipoSelectivoLibre, datospersonales.MomentoInscripcion, "telefono", datospersonales.Voluntario, datospersonales.FuentePersona, datospersonales.CategoriaOrdinaria},
		{datospersonales.TipoSelectivoLibre, datospersonales.MomentoInscripcion, "domicilio_notificacion", datospersonales.Condicional, datospersonales.FuentePersona, datospersonales.CategoriaOrdinaria},
		{datospersonales.TipoBolsa, datospersonales.MomentoInscripcion, "nombre_apellidos", datospersonales.Obligatorio, datospersonales.FuenteIdentificacionElectronica, datospersonales.CategoriaOrdinaria},
		{datospersonales.TipoSelectivoLibre, datospersonales.MomentoInscripcion, "discapacidad_grado", datospersonales.Condicional, datospersonales.FuenteConsultaAdministracion, datospersonales.CategoriaEspecial},
		{datospersonales.TipoSelectivoLibre, datospersonales.MomentoBaremacion, "servicios_diputacion", datospersonales.Condicional, datospersonales.FuentePersonalDiputacion, datospersonales.CategoriaOrdinaria},
		{datospersonales.TipoPromocionInterna, datospersonales.MomentoInscripcion, "condicion_personal_diputacion", datospersonales.Obligatorio, datospersonales.FuentePersonalDiputacion, datospersonales.CategoriaOrdinaria},
		{datospersonales.TipoBolsa, datospersonales.MomentoContratacion, "aptitud_medica", datospersonales.Condicional, datospersonales.FuenteVigilanciaSalud, datospersonales.CategoriaEspecial},
		{datospersonales.TipoBolsa, datospersonales.MomentoContratacion, "certificado_delitos_sexuales", datospersonales.Condicional, datospersonales.FuenteConsultaAdministracion, datospersonales.CategoriaPenal},
	}
	for _, caso := range casos {
		dato, ok := catalogo.Permitido(caso.tipo, caso.momento, caso.dato)
		if !ok || dato.Obligatoriedad != caso.obligatoriedad || dato.Fuente != caso.fuente || dato.Categoria != caso.categoria {
			t.Errorf("%s/%s/%s: %+v (existe=%v)", caso.tipo, caso.momento, caso.dato, dato, ok)
		}
	}
	// Lo que el estudio dice que nunca se pide no figura en ningún tipo ni momento.
	for _, prohibido := range []string{"diagnostico_discapacidad", "contacto_emergencia", "domicilio_laboral_fiscal"} {
		if _, ok := catalogo.Permitido(datospersonales.TipoBolsa, datospersonales.MomentoInscripcion, prohibido); ok {
			t.Errorf("%s no debe pedirse en la inscripción", prohibido)
		}
	}
	for _, dato := range catalogo.Datos {
		if dato.Dato == "diagnostico_discapacidad" || dato.Dato == "contacto_emergencia" {
			t.Errorf("%s figura en el catálogo", dato.Clave)
		}
		// Ningún dato especial, penal o protegido es obligatorio en Aspirantes.
		if dato.Custodia == datospersonales.CustodiaAspirantes && dato.Categoria.EsSensible() && dato.Obligatoriedad == datospersonales.Obligatorio {
			t.Errorf("%s es sensible y obligatorio en Aspirantes", dato.Clave)
		}
		// Lo que se consulta ofrece la oposición y la alternativa de aportarlo.
		if dato.SeConsulta() && (dato.ConsultaRegimen != datospersonales.RegimenOposicion || dato.FuenteAlternativa != datospersonales.FuentePersona) {
			t.Errorf("%s se consulta sin oposición ni alternativa", dato.Clave)
		}
	}
	// Promoción interna y provisión son trámites de empleado: sin tasa,
	// llamamiento ni contratación en Aspirantes.
	for _, tipo := range []datospersonales.TipoConvocatoria{datospersonales.TipoPromocionInterna, datospersonales.TipoProvision} {
		for _, momento := range []datospersonales.Momento{datospersonales.MomentoPagoTasa, datospersonales.MomentoLlamamiento, datospersonales.MomentoContratacion} {
			if datos := catalogo.Para(tipo, momento); len(datos) != 0 {
				t.Errorf("%s pide datos en %s: %d", tipo, momento, len(datos))
			}
		}
	}
	for _, dato := range catalogo.Para(datospersonales.TipoBolsa, datospersonales.MomentoContratacion) {
		if dato.Custodia != datospersonales.CustodiaPersonal {
			t.Errorf("%s: los datos del contrato los guarda Personal", dato.Clave)
		}
	}
}

// i18n puro: todo código del catálogo tiene su texto en el catálogo de textos.
func TestCodigosDelCatalogoTienenTexto(t *testing.T) {
	contenido, err := os.ReadFile(rutaTextosES)
	if err != nil {
		t.Fatal(err)
	}
	var textos map[string]map[string]any
	if err := json.Unmarshal(contenido, &textos); err != nil {
		t.Fatal(err)
	}
	exigir := func(seccion, codigo string) {
		if codigo == "" {
			return
		}
		if _, ok := textos[seccion][codigo]; !ok {
			t.Errorf("falta el texto %s.%s", seccion, codigo)
		}
	}
	for _, tipo := range datospersonales.TiposConvocatoria() {
		exigir("tipo_convocatoria", string(tipo))
	}
	for _, momento := range datospersonales.Momentos() {
		exigir("momento", string(momento))
	}
	for _, dato := range catalogoEjemplo(t).Datos {
		exigir("dato", dato.Dato)
		exigir("finalidad", dato.Finalidad)
		exigir("condicion", dato.Condicion)
		exigir("obligatoriedad", string(dato.Obligatoriedad))
		exigir("fuente", string(dato.Fuente))
		exigir("fuente", string(dato.FuenteAlternativa))
		exigir("consulta_servicio", dato.ConsultaServicio)
		exigir("consulta_regimen", string(dato.ConsultaRegimen))
		exigir("casilla_consulta", string(dato.ConsultaRegimen))
		exigir("categoria", string(dato.Categoria))
		exigir("custodia", string(dato.Custodia))
		exigir("pendiente_de", string(dato.PendienteDe))
	}
	exigir("aviso", "paquete_ejemplo")
}

func TestParaDevuelveCopiasYProcedencia(t *testing.T) {
	resolutor := resolutorDesde(t, rutaPaqueteEjemplo)
	datos, procedencia, err := resolutor.Para(context.Background(), datospersonales.TipoBolsa, datospersonales.MomentoInscripcion)
	if err != nil || len(datos) == 0 || procedencia.Datos != nil || !procedencia.PaqueteEjemplo || procedencia.HuellaCatalogo == "" {
		t.Fatalf("resultado inesperado: %d %+v %v", len(datos), procedencia, err)
	}
	for _, dato := range datos {
		if dato.Tipo != datospersonales.TipoBolsa || dato.Momento != datospersonales.MomentoInscripcion ||
			dato.ReferenciaEntrada.CatalogoHuellaSHA256 != procedencia.HuellaCatalogo {
			t.Fatalf("dato fuera de filtro o sin huella: %+v", dato)
		}
	}
	datos[0].BaseRGPD[0] = "alterado"
	otra, _, _ := resolutor.Para(context.Background(), datospersonales.TipoBolsa, datospersonales.MomentoInscripcion)
	if otra[0].BaseRGPD[0] == "alterado" {
		t.Fatal("la lista de bases no es una copia")
	}
	if _, _, err := resolutor.Para(context.Background(), "otro", datospersonales.MomentoInscripcion); !errors.Is(err, datospersonales.ErrConfiguracion) {
		t.Fatalf("tipo desconocido admitido: %v", err)
	}
	var nulo *datospersonales.Resolutor
	if _, err := nulo.Catalogo(context.Background()); !errors.Is(err, datospersonales.ErrCatalogoNoConfigurado) {
		t.Fatalf("sin catálogo: %v", err)
	}
	if _, err := datospersonales.NuevoResolutor(nil, nil, relojFijo{}); !errors.Is(err, datospersonales.ErrConfiguracion) {
		t.Fatalf("configuración nula admitida: %v", err)
	}
}

// Una sola entrada que incumple el contrato invalida el catálogo entero.
func TestEntradaFueraDeContratoInvalidaElCatalogo(t *testing.T) {
	casos := map[string]func(map[string]string){
		"condicional sin condicion": func(a map[string]string) { a["obligatoriedad"] = "condicional"; delete(a, "condicion") },
		"obligatorio con condicion": func(a map[string]string) { a["condicion"] = "si_puntua" },
		"momento desconocido":       func(a map[string]string) { a["momento"] = "siempre" },
		"tipo desconocido":          func(a map[string]string) { a["tipo_convocatoria"] = "oposicion" },
		"atributo desconocido":      func(a map[string]string) { a["texto_visible"] = "Nombre" },
		"consulta sin regimen":      func(a map[string]string) { a["fuente"] = "consulta_administracion"; a["consulta_servicio"] = "titulos" },
		"consulta sin alternativa": func(a map[string]string) {
			a["fuente"] = "consulta_administracion"
			a["consulta_servicio"] = "titulos"
			a["consulta_regimen"] = "oposicion"
			delete(a, "fuente_alternativa")
		},
		"servicio sin consulta":           func(a map[string]string) { a["consulta_servicio"] = "titulos" },
		"contrato guardado en aspirantes": func(a map[string]string) { a["momento"] = "contratacion" },
		"personal fuera del contrato":     func(a map[string]string) { a["custodia"] = "personal" },
		"penal en aspirantes":             func(a map[string]string) { a["categoria"] = "penal" },
		"comprobacion anterior":           func(a map[string]string) { a["momento"] = "admision"; a["momento_comprobacion"] = "inscripcion" },
		"aprobado en paquete de ejemplo": func(a map[string]string) {
			a["origen"] = "aprobado"
			a["aprobacion_ref"] = "acta"
			delete(a, "pendiente_de")
		},
		"ejemplo sin quien lo confirma":     func(a map[string]string) { delete(a, "pendiente_de") },
		"base rgpd mal formada":             func(a map[string]string) { a["base_rgpd"] = "art 6" },
		"sin norma":                         func(a map[string]string) { delete(a, "norma") },
		"categoria desconocida":             func(a map[string]string) { a["categoria"] = "sensible" },
		"fuente alternativa igual a fuente": func(a map[string]string) { a["fuente_alternativa"] = a["fuente"] },
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			ruta := paqueteMutado(t, func(entradas []map[string]any) []map[string]any {
				// La primera entrada es bolsa/inscripción/nombre_apellidos: obligatoria y de persona física.
				atributos := entradas[0]["atributos"].(map[string]any)
				planos := map[string]string{}
				for clave, valor := range atributos {
					planos[clave] = valor.(string)
				}
				mutar(planos)
				nuevos := map[string]any{}
				for clave, valor := range planos {
					nuevos[clave] = valor
				}
				entradas[0]["atributos"] = nuevos
				return entradas
			})
			consulta, err := fichero.NuevaConsultaCatalogos(ruta)
			if err != nil {
				t.Fatalf("el adaptador rechaza la mutación antes del contrato: %v", err)
			}
			resolutor, err := datospersonales.NuevoResolutor(consulta, consulta, relojFijo{})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := resolutor.Catalogo(context.Background()); !errors.Is(err, datospersonales.ErrCatalogoInvalido) {
				t.Fatalf("catálogo admitido: %v", err)
			}
		})
	}
	t.Run("dato repetido en el mismo tipo y momento", func(t *testing.T) {
		ruta := paqueteMutado(t, func(entradas []map[string]any) []map[string]any {
			copia := map[string]any{}
			for clave, valor := range entradas[0] {
				copia[clave] = valor
			}
			copia["clave"] = "bolsa.inscripcion.nombre_apellidos_repetido"
			return append(entradas, copia)
		})
		if _, err := resolutorDesde(t, ruta).Catalogo(context.Background()); !errors.Is(err, datospersonales.ErrCatalogoInvalido) {
			t.Fatalf("duplicado admitido: %v", err)
		}
	})
}

func paqueteMutado(t *testing.T, mutar func([]map[string]any) []map[string]any) string {
	t.Helper()
	contenido, err := os.ReadFile(rutaPaqueteEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	var paquete map[string]any
	if err := json.Unmarshal(contenido, &paquete); err != nil {
		t.Fatal(err)
	}
	catalogo := paquete["catalogo"].(map[string]any)
	crudas := catalogo["entradas"].([]any)
	entradas := make([]map[string]any, 0, len(crudas))
	for _, entrada := range crudas {
		entradas = append(entradas, entrada.(map[string]any))
	}
	catalogo["entradas"] = mutar(entradas)
	salida, err := json.Marshal(paquete)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "paquete.json")
	if err := os.WriteFile(ruta, salida, 0o600); err != nil {
		t.Fatal(err)
	}
	return ruta
}
