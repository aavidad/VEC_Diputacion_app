package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type preparadorIntentosPrueba struct{}

func (preparadorIntentosPrueba) SoportaRegistroTelefonoServidor() bool { return true }

func (preparadorIntentosPrueba) PrepararSolicitudRegistrarContacto(_ context.Context, e EntradaRegistrarContactoParticipacion) (puertosbolsa.SolicitudRegistrarContactoParticipacion, error) {
	return puertosbolsa.SolicitudRegistrarContactoParticipacion{BolsaRef: e.BolsaRef, ParticipacionRef: e.ParticipacionRef, LlamamientoRef: e.LlamamientoRef, Canal: e.Canal, Resultado: e.Resultado, Anotacion: e.Anotacion, Instante: e.Instante, InstanteServidor: e.InstanteServidor}, nil
}
func (preparadorIntentosPrueba) PrepararConsultaContactos(_ context.Context, bolsa, participacion, cursor string, limite int) (puertosbolsa.ConsultaContactosParticipacion, error) {
	return puertosbolsa.ConsultaContactosParticipacion{BolsaRef: bolsa, ParticipacionRef: participacion, Cursor: cursor, Limite: limite}, nil
}
func (preparadorIntentosPrueba) PrepararConsultaContactosBolsa(context.Context, string, string, int) (puertosbolsa.ConsultaContactosBolsa, error) {
	return puertosbolsa.ConsultaContactosBolsa{}, nil
}

type operadorIntentosPrueba struct {
	errRegistro   error
	limite        int
	llamamiento   string
	completo      bool
	registroFinal *dominiobolsa.EstadoIntentosTelefonicos
	ultima        puertosbolsa.SolicitudRegistrarContactoParticipacion
	disponible    bool
}

func (o *operadorIntentosPrueba) RegistrarContactoParticipacion(_ context.Context, s puertosbolsa.SolicitudRegistrarContactoParticipacion) (puertosbolsa.RegistroContactoParticipacion, error) {
	o.ultima = s
	if o.errRegistro != nil {
		return puertosbolsa.RegistroContactoParticipacion{}, o.errRegistro
	}
	return puertosbolsa.RegistroContactoParticipacion{Contacto: dominiobolsa.ContactoParticipacion{ContactoRef: "contacto:1", Canal: s.Canal, Resultado: s.Resultado, Instante: s.Instante}, ReciboRef: "recibo:1", Intentos: o.registroFinal}, nil
}
func (o *operadorIntentosPrueba) ListarContactosParticipacion(_ context.Context, q puertosbolsa.ConsultaContactosParticipacion) (puertosbolsa.PaginaContactosParticipacion, error) {
	o.limite = q.Limite
	return puertosbolsa.PaginaContactosParticipacion{Contactos: []dominiobolsa.ContactoParticipacion{{ContactoRef: "contacto:1", Canal: "telefono", Resultado: "no_contesta", LlamamientoRef: "llamamiento:1", Instante: time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)}}, RegistroTelefonoDisponible: o.disponible}, nil
}
func (o *operadorIntentosPrueba) ListarContactosBolsa(context.Context, puertosbolsa.ConsultaContactosBolsa) (puertosbolsa.PaginaContactosParticipacion, error) {
	return puertosbolsa.PaginaContactosParticipacion{}, nil
}
func (o *operadorIntentosPrueba) EstadoIntentosTelefonicos(_ context.Context, llamamiento string, contactos []dominiobolsa.ContactoParticipacion, completo bool) (puertosbolsa.EstadoIntentosContacto, error) {
	o.llamamiento, o.completo = llamamiento, completo
	return puertosbolsa.EstadoIntentosContacto{
		Configurada: true, LlamamientoRef: llamamiento, Completo: completo, Ahora: time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC),
		Estado:   dominiobolsa.EstadoIntentosTelefonicos{SinContacto: len(contactos), Maximo: 4, Proceso: 1, Intento: 2, Avisos: []string{"antes_de_separacion"}},
		Politica: dominiobolsa.PoliticaIntentosTelefonicos{IntentosPorProceso: 2, Procesos: 2, SeparacionMinima: 2 * time.Hour, ControlSeparacion: "impedir", ResultadosSinContacto: []string{"no_contesta"}},
		Reglas:   []puertosbolsa.ReglaIntentosContacto{{Clave: "b02.intentos_contacto", Etiqueta: "Dos intentos", Referencia: "vec.bolsa.reglas:1:b02.intentos_contacto"}},
	}, nil
}

const rutaContactosPrueba = RutaBolsasGestion + "/bolsa:1/candidatos/participacion:1/contactos"

func pedirContactos(t *testing.T, h http.Handler, metodo, destino, cuerpo string) (int, map[string]any) {
	t.Helper()
	var r *http.Request
	if cuerpo == "" {
		r = httptest.NewRequest(metodo, destino, nil)
	} else {
		r = httptest.NewRequest(metodo, destino, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "clave-1")
	}
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var salida map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &salida); err != nil {
		t.Fatalf("respuesta no JSON: %s", w.Body.String())
	}
	return w.Code, salida
}

func TestConsultaContactosConLlamamientoAnadeControlDeIntentos(t *testing.T) {
	operador := &operadorIntentosPrueba{}
	h, err := NuevoHandlerContactoParticipacion(preparadorIntentosPrueba{}, operador)
	if err != nil {
		t.Fatal(err)
	}
	codigo, salida := pedirContactos(t, h, http.MethodGet, rutaContactosPrueba+"?llamamiento_ref=llamamiento:1", "")
	datos, _ := salida["data"].(map[string]any)
	intentos, _ := datos["intentos"].(map[string]any)
	if codigo != 200 || intentos["configurado"] != true || intentos["proceso"] != float64(1) || intentos["intento"] != float64(2) ||
		operador.limite != limiteHistoricoIntentos || operador.llamamiento != "llamamiento:1" || !operador.completo {
		t.Fatalf("codigo=%d intentos=%v limite=%d", codigo, intentos, operador.limite)
	}
	if codigo, salida := pedirContactos(t, h, http.MethodGet, rutaContactosPrueba, ""); codigo != 200 || salida["data"].(map[string]any)["intentos"] != nil {
		t.Fatalf("sin llamamiento no se evalúa: %d %v", codigo, salida)
	}
	for _, consulta := range []string{"?llamamiento_ref=", "?llamamiento_ref=a&llamamiento_ref=b", "?llamamiento_ref=a%20b"} {
		if codigo, _ := pedirContactos(t, h, http.MethodGet, rutaContactosPrueba+consulta, ""); codigo != 400 {
			t.Errorf("%s: codigo=%d", consulta, codigo)
		}
	}
}

func TestRegistroDeIntentoDevuelveEstadoYRechazosDeLasReglas(t *testing.T) {
	cuerpo := `{"canal":"telefono","instante":"2026-09-28T08:00:00Z","resultado":"no_contesta","anotacion":"Sin respuesta","llamamiento_ref":"llamamiento:1"}`
	operador := &operadorIntentosPrueba{registroFinal: &dominiobolsa.EstadoIntentosTelefonicos{SinContacto: 4, Maximo: 4, BajaPropuesta: true, Avisos: []string{"fuera_de_franja"}}}
	h, _ := NuevoHandlerContactoParticipacion(preparadorIntentosPrueba{}, operador)
	codigo, salida := pedirContactos(t, h, http.MethodPost, rutaContactosPrueba, cuerpo)
	intentos, _ := salida["data"].(map[string]any)["intentos"].(map[string]any)
	if codigo != 201 || intentos["baja_propuesta"] != true {
		t.Fatalf("codigo=%d salida=%v", codigo, salida)
	}
	for err, esperado := range map[error]string{
		dominiobolsa.ErrIntentoAntesDeSeparacion: "intento_antes_de_separacion",
		dominiobolsa.ErrIntentoFueraDeFranja:     "intento_fuera_de_franja",
		dominiobolsa.ErrIntentosContactoAgotados: "intentos_agotados",
	} {
		operador.errRegistro = err
		codigo, salida := pedirContactos(t, h, http.MethodPost, rutaContactosPrueba, cuerpo)
		if codigo != 409 || salida["error"].(map[string]any)["codigo"] != esperado {
			t.Errorf("%v: codigo=%d salida=%v", err, codigo, salida)
		}
	}
}

func TestRegistroTelefonoActualNoAceptaInstanteDelCliente(t *testing.T) {
	o := &operadorIntentosPrueba{}
	h, err := NuevoHandlerContactoParticipacion(preparadorIntentosPrueba{}, o)
	if err != nil {
		t.Fatal(err)
	}
	actual := `{"canal":"telefono","resultado":"comunica","llamamiento_ref":"llamamiento:1"}`
	if estado, _ := pedirContactos(t, h, http.MethodPost, rutaContactosPrueba, actual); estado != 201 || !o.ultima.InstanteServidor || !o.ultima.Instante.IsZero() || o.ultima.Anotacion != "" {
		t.Fatalf("modo servidor perdido: %d %+v", estado, o.ultima)
	}
	for _, cuerpo := range []string{
		`{"canal":"telefono","instante":null,"resultado":"comunica","llamamiento_ref":"llamamiento:1"}`,
		`{"canal":"correo","resultado":"comunica","llamamiento_ref":"llamamiento:1"}`,
	} {
		if estado, _ := pedirContactos(t, h, http.MethodPost, rutaContactosPrueba, cuerpo); estado != 400 {
			t.Errorf("entrada inválida admitida: %d", estado)
		}
	}
	legado := `{"canal":"telefono","instante":"2026-09-28T08:00:00Z","resultado":"no_contesta","anotacion":"Sin respuesta","llamamiento_ref":"llamamiento:1"}`
	if estado, _ := pedirContactos(t, h, http.MethodPost, rutaContactosPrueba, legado); estado != 201 || o.ultima.InstanteServidor || o.ultima.Instante.IsZero() {
		t.Fatalf("legado alterado: %d %+v", estado, o.ultima)
	}
}

func TestRegistroTelefonoSoloSePublicaConB87YPreparador(t *testing.T) {
	o := &operadorIntentosPrueba{}
	h, _ := NuevoHandlerContactoParticipacion(preparadorIntentosPrueba{}, o)
	destino := rutaContactosPrueba + "?llamamiento_ref=llamamiento:1"
	_, sin := pedirContactos(t, h, http.MethodGet, destino, "")
	if sin["data"].(map[string]any)["intentos"].(map[string]any)["registro_telefono"] != nil {
		t.Fatal("capacidad publicada sin SQL")
	}
	o.disponible = true
	_, con := pedirContactos(t, h, http.MethodGet, destino, "")
	registro := con["data"].(map[string]any)["intentos"].(map[string]any)["registro_telefono"].(map[string]any)
	// Las reglas de la prueba no cuentan «comunica» como intento sin contacto.
	if registro["instante_servidor"] != true || registro["anotacion_opcional"] != true || len(registro["resultados"].([]any)) != 7 || slices.Contains(registro["resultados"].([]any), any("comunica")) {
		t.Fatalf("contrato de capacidad: %v", registro)
	}
}

func TestComunicaSoloSiLasReglasLoCuentanSinContacto(t *testing.T) {
	for _, caso := range []struct {
		intentos map[string]any
		comunica bool
	}{
		{map[string]any{"configurado": false}, true},
		{map[string]any{"configurado": true, "resultados_sin_contacto": []string{"no_contesta", "comunica"}}, true},
		{map[string]any{"configurado": true, "resultados_sin_contacto": []string{"no_contesta"}}, false},
	} {
		if got := slices.Contains(resultadosRegistroTelefono(caso.intentos), "comunica"); got != caso.comunica {
			t.Errorf("%v: comunica=%v", caso.intentos, got)
		}
	}
}
