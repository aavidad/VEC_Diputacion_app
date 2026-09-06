package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const referenciaOrganizacionSintetica = "org_0123456789abcdef"

func TestNuevaReferenciaOrganizacionAdmiteLimitesYRecuperaBytesExactos(t *testing.T) {
	t.Parallel()

	pruebas := []struct {
		nombre     string
		referencia string
	}{
		{nombre: "sufijo de dieciseis bytes", referencia: referenciaOrganizacionSintetica},
		{nombre: "sufijo de ochenta bytes", referencia: "org_" + strings.Repeat("a", 80)},
	}

	for _, prueba := range pruebas {
		prueba := prueba
		t.Run(prueba.nombre, func(t *testing.T) {
			t.Parallel()

			referencia, err := NuevaReferenciaOrganizacion(prueba.referencia)
			if err != nil {
				t.Fatalf("NuevaReferenciaOrganizacion() error = %v", err)
			}
			if err := referencia.Validar(); err != nil {
				t.Fatalf("Validar() error = %v", err)
			}
			recuperada, err := referencia.Referencia()
			if err != nil {
				t.Fatalf("Referencia() error = %v", err)
			}
			if recuperada != prueba.referencia {
				t.Fatalf("Referencia() = %q; esperado %q", recuperada, prueba.referencia)
			}
		})
	}
}

func TestNuevaReferenciaOrganizacionRechazaFormasNoCanonicas(t *testing.T) {
	t.Parallel()

	pruebas := []struct {
		nombre     string
		referencia string
	}{
		{nombre: "vacia"},
		{nombre: "solo prefijo", referencia: "org_"},
		{nombre: "sufijo de quince bytes", referencia: "org_" + strings.Repeat("a", 15)},
		{nombre: "sufijo de ochenta y un bytes", referencia: "org_" + strings.Repeat("a", 81)},
		{nombre: "prefijo distinto", referencia: "ent_0123456789abcdef"},
		{nombre: "prefijo en mayusculas", referencia: "ORG_0123456789abcdef"},
		{nombre: "mayuscula en carga", referencia: "org_0123456789abcdeF"},
		{nombre: "guion en carga", referencia: "org_0123456789abcde-"},
		{nombre: "subrayado en carga", referencia: "org_0123456789abcde_"},
		{nombre: "espacio inicial", referencia: " " + referenciaOrganizacionSintetica},
		{nombre: "espacio final", referencia: referenciaOrganizacionSintetica + " "},
		{nombre: "espacio interior", referencia: "org_01234567 89abcdef"},
		{nombre: "nul", referencia: "org_0123456789abcde\x00"},
		{nombre: "salto de linea", referencia: "org_0123456789abcde\n"},
		{nombre: "tabulador", referencia: "org_0123456789abcde\t"},
		{nombre: "control ascii", referencia: "org_0123456789abcde\x1f"},
		{nombre: "unicode compuesto", referencia: "org_0123456789abcdé"},
		{nombre: "unicode descompuesto", referencia: "org_0123456789abcde\u0301"},
		{nombre: "confusable cirilico", referencia: "org_0123456789abcdeа"},
		{nombre: "confusable griego", referencia: "org_0123456789abcdeο"},
	}

	for _, prueba := range pruebas {
		prueba := prueba
		t.Run(prueba.nombre, func(t *testing.T) {
			t.Parallel()

			referencia, err := NuevaReferenciaOrganizacion(prueba.referencia)
			if !errors.Is(err, ErrReferenciaOrganizacionInvalida) {
				t.Fatalf("NuevaReferenciaOrganizacion() error = %v; esperado centinela", err)
			}
			if referencia != (ReferenciaOrganizacion{}) {
				t.Fatalf("NuevaReferenciaOrganizacion() valor = %#v; esperado valor cero", referencia)
			}
			if err.Error() != "personal: referencia de organizacion invalida" {
				t.Fatalf("error = %q; esperado error opaco exacto", err)
			}
			if prueba.referencia != "" && strings.Contains(err.Error(), prueba.referencia) {
				t.Fatalf("el error filtra la entrada %q", prueba.referencia)
			}
		})
	}
}

func TestReferenciaOrganizacionNoTransformaEntradas(t *testing.T) {
	t.Parallel()

	entradas := []string{
		" " + referenciaOrganizacionSintetica,
		referenciaOrganizacionSintetica + " ",
		"org_0123456789ABCDEF",
		"org_0123456789abcdé",
	}
	for _, entrada := range entradas {
		entrada := entrada
		t.Run(entrada, func(t *testing.T) {
			t.Parallel()

			valor, err := NuevaReferenciaOrganizacion(entrada)
			if !errors.Is(err, ErrReferenciaOrganizacionInvalida) {
				t.Fatalf("NuevaReferenciaOrganizacion(%q) error = %v; esperado centinela", entrada, err)
			}
			if valor != (ReferenciaOrganizacion{}) {
				t.Fatalf("NuevaReferenciaOrganizacion(%q) transformo la entrada", entrada)
			}
		})
	}
}

func TestReferenciaOrganizacionReacreditaYRechazaValorCero(t *testing.T) {
	t.Parallel()

	valida := ReferenciaOrganizacion{referencia: referenciaOrganizacionSintetica}
	if err := valida.Validar(); err != nil {
		t.Fatalf("Validar() de valor canonico error = %v", err)
	}

	corrupta := ReferenciaOrganizacion{referencia: referenciaOrganizacionSintetica + "-"}
	if err := corrupta.Validar(); !errors.Is(err, ErrReferenciaOrganizacionInvalida) {
		t.Fatalf("Validar() de valor corrupto error = %v; esperado centinela", err)
	}
	if recuperada, err := corrupta.Referencia(); recuperada != "" || !errors.Is(err, ErrReferenciaOrganizacionInvalida) {
		t.Fatalf("Referencia() de valor corrupto = %q, %v; esperado vacio y centinela", recuperada, err)
	}

	var cero ReferenciaOrganizacion
	if err := cero.Validar(); !errors.Is(err, ErrReferenciaOrganizacionInvalida) {
		t.Fatalf("Validar() del valor cero error = %v; esperado centinela", err)
	}
	if recuperada, err := cero.Referencia(); recuperada != "" || !errors.Is(err, ErrReferenciaOrganizacionInvalida) {
		t.Fatalf("Referencia() del valor cero = %q, %v; esperado vacio y centinela", recuperada, err)
	}
}

func TestReferenciaOrganizacionEsComparableInmutableYSinSuperficieAdicional(t *testing.T) {
	t.Parallel()

	primera, err := NuevaReferenciaOrganizacion(referenciaOrganizacionSintetica)
	if err != nil {
		t.Fatalf("NuevaReferenciaOrganizacion() error = %v", err)
	}
	segunda, err := NuevaReferenciaOrganizacion(referenciaOrganizacionSintetica)
	if err != nil {
		t.Fatalf("NuevaReferenciaOrganizacion() error = %v", err)
	}
	if primera != segunda {
		t.Fatal("dos referencias construidas desde los mismos bytes no son comparables como iguales")
	}
	conjunto := map[ReferenciaOrganizacion]struct{}{primera: {}}
	if _, existe := conjunto[segunda]; !existe {
		t.Fatal("ReferenciaOrganizacion no conserva semantica comparable como clave")
	}

	tipo := reflect.TypeOf(ReferenciaOrganizacion{})
	if tipo.NumField() != 1 {
		t.Fatalf("ReferenciaOrganizacion tiene %d campos; esperado uno", tipo.NumField())
	}
	campo := tipo.Field(0)
	if campo.Name != "referencia" || campo.Type.Kind() != reflect.String || campo.IsExported() {
		t.Fatalf("campo interno = %#v; esperado string privado referencia", campo)
	}
	metodosEsperados := []string{"Referencia", "Validar"}
	if tipo.NumMethod() != len(metodosEsperados) {
		t.Fatalf("ReferenciaOrganizacion expone %d metodos; esperado %d", tipo.NumMethod(), len(metodosEsperados))
	}
	for indice, esperado := range metodosEsperados {
		if metodo := tipo.Method(indice); metodo.Name != esperado {
			t.Fatalf("metodo exportado %d = %q; esperado %q", indice, metodo.Name, esperado)
		}
	}
}

func catalogoOrganizacionPrueba() vecdomain.CatalogoConfigurable {
	f := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return vecdomain.CatalogoConfigurable{ID: "estructura-organizativa-dipgra", Version: 1, Revision: 1, ModuloID: "personal", Nombre: "Estructura", Descripcion: "Estructura", FuenteRef: "datosreserva", MotivoCreacion: "fixture", Estado: vecdomain.EstadoCatalogoBorrador, CreadoPor: "tecnico-estructura-1", CreadoEn: f, Entradas: []vecdomain.EntradaCatalogoConfigurable{{Clave: "director", Etiqueta: "Director", Orden: 0, VigenteDesde: f, Atributos: map[string]string{"tipo": "puesto_responsabilidad"}}}}
}

func TestPrepararCambioOrganizacionEditaSinMutarYConservaProcedencia(t *testing.T) {
	a := catalogoOrganizacionPrueba()
	h, _ := a.HuellaSHA256()
	ahora := a.CreadoEn.Add(time.Hour)
	r, err := PrepararCambioEstructuraOrganizativa(a, a.Revision, h, "tecnico-estructura-2", "ajuste de etiqueta", CambioUnidadOrganizativa{Clave: "director", Etiqueta: "Dirección", Tipo: "centro"}, ahora)
	if err != nil || r.Revision != a.Revision+1 || r.Entradas[0].Atributos["modificada_localmente"] != "si" || r.Entradas[0].VigenteDesde != a.Entradas[0].VigenteDesde {
		t.Fatalf("resultado=%#v err=%v", r, err)
	}
	if a.Entradas[0].Etiqueta != "Director" {
		t.Fatal("original mutado")
	}
}

func TestPrepararCambioOrganizacionAltaDirectorSinNivelFijo(t *testing.T) {
	a := catalogoOrganizacionPrueba()
	h, _ := a.HuellaSHA256()
	r, err := PrepararCambioEstructuraOrganizativa(a, a.Revision, h, "tecnico-estructura-2", "alta unidad", CambioUnidadOrganizativa{Clave: "local-01234567-89ab-4cde-8fab-0123456789ab", Etiqueta: "Unidad nueva", Tipo: "delegacion", AdscripcionClave: "director"}, a.CreadoEn.Add(time.Hour))
	if err != nil || len(r.Entradas) != 2 || r.Entradas[1].Atributos["codigo_fuente"] != "" || r.Entradas[1].Atributos["pagina_fuente"] != "" || r.Entradas[1].Atributos["adscripcion_clave"] != "director" {
		t.Fatalf("resultado=%#v err=%v", r, err)
	}
}

func TestPrepararCambioOrganizacionConflictoPadreCicloYHash(t *testing.T) {
	a := catalogoOrganizacionPrueba()
	h, _ := a.HuellaSHA256()
	base := CambioUnidadOrganizativa{Clave: "director", Etiqueta: "Director", Tipo: "centro", AdscripcionClave: "no-existe"}
	if _, err := PrepararCambioEstructuraOrganizativa(a, a.Revision, "bad", "tecnico-estructura-2", "motivo", base, a.CreadoEn.Add(time.Hour)); !errors.Is(err, ErrRevisionOrganizacionEnConflicto) {
		t.Fatalf("hash err=%v", err)
	}
	if _, err := PrepararCambioEstructuraOrganizativa(a, a.Revision, h, "tecnico-estructura-2", "motivo", base, a.CreadoEn.Add(time.Hour)); !errors.Is(err, ErrCambioOrganizacionInvalido) {
		t.Fatalf("padre err=%v", err)
	}
}
