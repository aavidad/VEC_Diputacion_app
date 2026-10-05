package simulacion

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/application"
)

func puntosPrueba(n int64) *int64 { return &n }

func TestCatalogoNotasSoloPruebasDelEjemplo(t *testing.T) {
	ejemplos, err := Ejemplos()
	if err != nil {
		t.Fatal(err)
	}
	for _, ejemplo := range ejemplos {
		entrada, _, err := Preparar(ejemplo.Referencia)
		if err != nil {
			t.Fatal(err)
		}
		esperadas := 0
		for _, fase := range ejemplo.Configuracion.Fases {
			if fase.Tipo == "prueba" {
				esperadas += len(entrada.Solicitudes)
			}
		}
		if ejemplo.NotasPrueba == nil || len(ejemplo.NotasPrueba) != esperadas {
			t.Fatalf("%s: catálogo incompleto", ejemplo.Referencia)
		}
		for _, nota := range ejemplo.NotasPrueba {
			encontrada := false
			for _, solicitud := range entrada.Solicitudes {
				if solicitud.Referencia == nota.SolicitudRef && solicitud.Nombre == nota.Nombre && reflect.DeepEqual(solicitud.Notas[nota.FaseRef], nota.PuntosMicropuntos) {
					encontrada = true
				}
			}
			if !encontrada {
				t.Fatalf("nota ajena al ejemplo: %+v", nota)
			}
		}
	}
}

func TestNotasEditadasSoloCambianElEnsayoYConservanBolsa(t *testing.T) {
	ejemplos, _ := Ejemplos()
	ejemplo := ejemplos[2]
	original, baremador, _ := Preparar(ejemplo.Referencia)
	rOriginal, err := application.Simular(ejemplo.Configuracion, original, baremador)
	if err != nil {
		t.Fatal(err)
	}
	propuestas := []NotaPrueba{{"solicitud_1", "ejercicio_1", nil}, {"solicitud_2", "ejercicio_1", puntosPrueba(10_000_000)}}
	entrada, b, err := PrepararConNotas(Solicitud{EjemploRef: ejemplo.Referencia, Configuracion: ejemplo.Configuracion, NotasPrueba: propuestas})
	if err != nil {
		t.Fatal(err)
	}
	*propuestas[1].PuntosMicropuntos = 0
	if *entrada.Solicitudes[1].Notas["ejercicio_1"] != 10_000_000 {
		t.Fatal("nota compartida con el cliente")
	}
	r, err := application.Simular(ejemplo.Configuracion, entrada, b)
	if err != nil {
		t.Fatal(err)
	}
	MarcarNotasEditadas(&r, propuestas)
	if r.Alcance != "ensayo_sintetico" || r.Estado != "indeterminado" || r.Solicitudes[0].TotalMicropuntos != nil || r.Solicitudes[0].Estado != "pendiente" {
		t.Fatal("null se convirtió en nota o clasificación")
	}
	for i, solicitud := range r.Solicitudes {
		if solicitud.Nombre != original.Solicitudes[i].Nombre || !reflect.DeepEqual(solicitud.AccesoDetalle, rOriginal.Solicitudes[i].AccesoDetalle) || !reflect.DeepEqual(solicitud.Fases[1], rOriginal.Solicitudes[i].Fases[1]) {
			t.Fatal("se cambiaron nombres, acceso o méritos del motor común")
		}
		esperado := "prueba_embebida"
		if i < 2 {
			esperado = "prueba_editada"
		}
		if solicitud.Fases[0].Origen != esperado {
			t.Fatalf("procedencia incorrecta: %+v", solicitud.Fases[0])
		}
	}
	recuperada, _, _ := PrepararConNotas(Solicitud{EjemploRef: ejemplo.Referencia, Configuracion: ejemplo.Configuracion})
	if !reflect.DeepEqual(original, recuperada) {
		t.Fatal("se alteró el siguiente ensayo sin propuestas")
	}
	*ejemplo.NotasPrueba[0].PuntosMicropuntos = 0
	nuevoCatalogo, _ := Ejemplos()
	if *nuevoCatalogo[2].NotasPrueba[0].PuntosMicropuntos != 8_000_000 {
		t.Fatal("catálogo compartido entre peticiones")
	}
}

func TestNotasRechazaReferenciasDuplicadasMeritosYRango(t *testing.T) {
	ejemplos, _ := Ejemplos()
	e := ejemplos[2]
	for nombre, notas := range map[string][]NotaPrueba{
		"solicitud ajena":      {{"solicitud_ajena", "ejercicio_1", puntosPrueba(1)}},
		"fase desconocida":     {{"solicitud_1", "inventada", puntosPrueba(1)}},
		"fase de otro ejemplo": {{"solicitud_1", "ejercicio_2", puntosPrueba(1)}},
		"meritos":              {{"solicitud_1", "meritos", puntosPrueba(1)}},
		"negativa":             {{"solicitud_1", "ejercicio_1", puntosPrueba(-1)}},
		"superior maximo":      {{"solicitud_1", "ejercicio_1", puntosPrueba(10_000_001)}},
		"duplicada":            {{"solicitud_1", "ejercicio_1", nil}, {"solicitud_1", "ejercicio_1", puntosPrueba(1)}},
		"alias unicode":        {{"ſolicitud_1", "ejercicio_1", puntosPrueba(1)}},
	} {
		t.Run(nombre, func(t *testing.T) {
			r, _, err := PrepararConNotas(Solicitud{EjemploRef: e.Referencia, Configuracion: e.Configuracion, NotasPrueba: notas})
			if !errors.Is(err, ErrNotas) || len(r.Solicitudes) != 0 {
				t.Fatalf("nota inválida produjo entrada: %+v %v", r, err)
			}
		})
	}
	e.Configuracion.Fases[0].MaximoMicropuntos = 6_000_000
	_, _, err := PrepararConNotas(Solicitud{EjemploRef: e.Referencia, Configuracion: e.Configuracion, NotasPrueba: []NotaPrueba{{"solicitud_1", "ejercicio_1", puntosPrueba(6_000_001)}}})
	if !errors.Is(err, ErrNotas) {
		t.Fatal("se usó el máximo embebido en lugar del configurado")
	}
	e.Configuracion.Fases[0].Referencia = "otra_prueba"
	e.Configuracion.Desempates = nil
	_, _, err = PrepararConNotas(Solicitud{EjemploRef: e.Referencia, Configuracion: e.Configuracion, NotasPrueba: []NotaPrueba{{"solicitud_1", "ejercicio_1", nil}}})
	if !errors.Is(err, ErrNotas) {
		t.Fatal("se editó una fase ausente de la configuración enviada")
	}
}

func TestDecodificarNotasExigeCamposExactosYEnteroONull(t *testing.T) {
	prefijo := `{"ejemplo_ref":"seleccion_oposicion_v1","configuracion":{},"notas_prueba":`
	for _, lista := range []string{
		`null`, `{}`, `[null]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1"}]`,
		`[{"fase_ref":"ejercicio_1","puntos_micropuntos":null}]`,
		`[{"solicitud_ref":"solicitud_1","puntos_micropuntos":null}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":1.5}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":1e6}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":"1"}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":true}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":9223372036854775808}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":0,"nombre":"Otra"}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":0,"persona_ref":"otra"}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":0,"Puntos_micropuntos":1}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntoſ_micropuntos":0}]`,
		`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":0,"puntos_micropuntos":1}]`,
	} {
		if _, err := Decodificar([]byte(prefijo + lista + `}`)); !errors.Is(err, ErrSolicitud) {
			t.Fatalf("lista mal formada admitida: %s (%v)", lista, err)
		}
	}
	for _, lista := range []string{`[]`, `[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":null}]`, `[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":0}]`} {
		s, err := Decodificar([]byte(prefijo + lista + `}`))
		if err != nil {
			t.Fatalf("lista permitida rechazada: %s %v", lista, err)
		}
		canonico, _ := json.Marshal(s)
		if len(s.NotasPrueba) > 0 && !strings.Contains(string(canonico), `"puntos_micropuntos":`) {
			t.Fatal("se perdió la nota o el null explícito")
		}
	}
}
