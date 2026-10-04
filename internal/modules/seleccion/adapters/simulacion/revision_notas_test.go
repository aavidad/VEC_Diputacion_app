package simulacion

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestRevisionNotasAcumulaCorreccionesYConservaBolsa(t *testing.T) {
	ejemplos, err := Ejemplos()
	if err != nil {
		t.Fatal(err)
	}
	e := ejemplos[2]
	s := SolicitudRevisionNotas{EjemploRef: e.Referencia, Configuracion: e.Configuracion,
		NotasAntecedente: []NotaPrueba{{"solicitud_1", "ejercicio_1", puntosPrueba(9_000_000)}},
		NotasPropuestas:  []NotaPrueba{{"solicitud_2", "ejercicio_1", puntosPrueba(10_000_000)}}}
	r, err := PrepararRevisionNotas(s)
	if err != nil {
		t.Fatal(err)
	}
	if *r.Antecedente.Solicitudes[0].Fases[0].PuntosMicropuntos != 9_000_000 || *r.Propuesta.Solicitudes[0].Fases[0].PuntosMicropuntos != 9_000_000 || *r.Propuesta.Solicitudes[1].Fases[0].PuntosMicropuntos != 10_000_000 {
		t.Fatal("la propuesta reinició otra edición del antecedente")
	}
	if len(r.Cambios) != 1 || r.Cambios[0].SolicitudRef != "solicitud_2" || r.Cambios[0].FaseRef != "ejercicio_1" || *r.Cambios[0].AnteriorMicropuntos != 7_000_000 || *r.Cambios[0].PropuestaMicropuntos != 10_000_000 {
		t.Fatalf("comparación incorrecta: %+v", r.Cambios)
	}
	for i, a := range r.Antecedente.Solicitudes {
		p := r.Propuesta.Solicitudes[i]
		if a.Nombre != p.Nombre || a.Referencia != p.Referencia || a.Acceso != p.Acceso || !reflect.DeepEqual(a.AccesoDetalle, p.AccesoDetalle) || !reflect.DeepEqual(a.Fases[1], p.Fases[1]) || a.Fases[1].Origen != "motor_bolsa" || a.Fases[1].HuellaMeritosSHA256 == "" {
			t.Fatal("se sustituyeron solicitudes, requisitos o trazas de Bolsa")
		}
	}
	if r.Antecedente.Alcance != "ensayo_sintetico" || r.Propuesta.Alcance != "ensayo_sintetico" || !reflect.DeepEqual(r.Configuracion, s.Configuracion) || !reflect.DeepEqual(r.Pendientes, []string{"revision_competente", "acto", "firma", "publicacion"}) {
		t.Fatal("se alteró la configuración o el alcance del borrador")
	}
	*s.NotasAntecedente[0].PuntosMicropuntos = 0
	s.Configuracion.Fases[0].Peso = 1
	if *r.NotasAntecedente[0].PuntosMicropuntos != 9_000_000 || r.Configuracion.Fases[0].Peso != 60 {
		t.Fatal("la salida comparte material mutable con la entrada")
	}
}

func TestRevisionNotasNullNoProduceAprobadoYNuevaNotaSustituyeAntecedente(t *testing.T) {
	ejemplos, _ := Ejemplos()
	e := ejemplos[0]
	s := SolicitudRevisionNotas{EjemploRef: e.Referencia, Configuracion: e.Configuracion,
		NotasAntecedente: []NotaPrueba{{"solicitud_1", "ejercicio_1", puntosPrueba(9_000_000)}, {"solicitud_2", "ejercicio_2", nil}},
		NotasPropuestas:  []NotaPrueba{{"solicitud_1", "ejercicio_1", nil}}}
	r, err := PrepararRevisionNotas(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Cambios) != 1 || r.Cambios[0].PropuestaMicropuntos != nil || r.Propuesta.Solicitudes[0].Fases[0].PuntosMicropuntos != nil || r.Propuesta.Solicitudes[1].Fases[1].PuntosMicropuntos != nil {
		t.Fatal("se perdió un null o se reinició la nota pendiente anterior")
	}
	if r.Propuesta.Estado != "indeterminado" || r.Propuesta.Solicitudes[0].Estado != "pendiente" || r.Propuesta.Solicitudes[0].TotalMicropuntos != nil || r.Propuesta.Solicitudes[0].Orden != nil || r.Propuesta.Solicitudes[0].Fases[0].Origen != "prueba_editada" {
		t.Fatal("la propuesta pendiente produjo calificación cerrada")
	}
}

func TestRevisionNotasRepetibleSinCambiosHuellaDeBytesJSONGo(t *testing.T) {
	ejemplos, _ := Ejemplos()
	for _, e := range ejemplos {
		s := SolicitudRevisionNotas{EjemploRef: e.Referencia, Configuracion: e.Configuracion}
		a, err := PrepararRevisionNotas(s)
		if err != nil {
			t.Fatal(err)
		}
		b, err := PrepararRevisionNotas(s)
		if err != nil || !reflect.DeepEqual(a, b) || len(a.Cambios) != 0 || !reflect.DeepEqual(a.Antecedente, a.Propuesta) {
			t.Fatal("revisión sin cambios no repetible")
		}
		exacto, _ := json.Marshal(a.Antecedente)
		h := sha256.Sum256(exacto)
		if a.HuellaAntecedenteSHA256 != hex.EncodeToString(h[:]) {
			t.Fatal("la huella no identifica los bytes JSON Go del antecedente")
		}
		if a.Esquema != "seleccion.revision_notas.v1" || a.Alcance != "preparacion_sintetica" || a.NotasAntecedente == nil || a.NotasPropuestas == nil || a.Cambios == nil {
			t.Fatal("contrato incompleto o arrays null")
		}
	}
}

func TestRevisionNotasRechazaEntradaAjenaAmbiguaONoAcotada(t *testing.T) {
	ejemplos, _ := Ejemplos()
	e := ejemplos[2]
	config, _ := json.Marshal(e.Configuracion)
	prefijo := `{"ejemplo_ref":"` + e.Referencia + `","configuracion":` + string(config)
	for _, campo := range []string{"notas_antecedente", "notas_propuestas"} {
		for _, lista := range []string{
			`null`, `[null]`, `[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1"}]`,
			`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":1.5}]`,
			`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":-1}]`,
			`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":10000001}]`,
			`[{"solicitud_ref":"otra","fase_ref":"ejercicio_1","puntos_micropuntos":0}]`,
			`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_2","puntos_micropuntos":0}]`,
			`[{"solicitud_ref":"solicitud_1","fase_ref":"meritos","puntos_micropuntos":0}]`,
			`[{"solicitud_ref":"ſolicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":0}]`,
			`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":0,"nombre":"otra"}]`,
			`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntoſ_micropuntos":0}]`,
			`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":0,"puntos_micropuntos":1}]`,
			`[{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":0},{"solicitud_ref":"solicitud_1","fase_ref":"ejercicio_1","puntos_micropuntos":1}]`,
		} {
			s, err := DecodificarRevisionNotas([]byte(prefijo + `,"` + campo + `":` + lista + `}`))
			if err == nil {
				r, preparacionErr := PrepararRevisionNotas(s)
				if preparacionErr == nil || r.Esquema != "" {
					t.Fatalf("lista inválida produjo salida: %s %s", campo, lista)
				}
			}
		}
	}
	for _, datos := range []string{prefijo + `,"persona_ref":"otra"}`, prefijo + `,"configuracion":{}}`, prefijo + `,"Notas_propuestas":[]}`, prefijo + `} {}`, strings.Repeat("x", MaximoBytes+1)} {
		if _, err := DecodificarRevisionNotas([]byte(datos)); err == nil {
			t.Fatalf("entrada ambigua admitida: %.100s", datos)
		}
	}
}
