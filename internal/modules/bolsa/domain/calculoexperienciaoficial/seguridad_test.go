package calculoexperienciaoficial

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestTiposPrincipalesRedactanFormatoYLogs(t *testing.T) {
	clave, intencion, recibo := clavePrueba(t), intencionPrueba(t), reciboPrueba(t)
	var salida strings.Builder
	for _, valor := range []any{clave, intencion, recibo} {
		_, _ = fmt.Fprintf(&salida, "%v|%+v|%#v|%s|%q", valor, valor, valor, valor, valor)
	}
	registrador := slog.New(slog.NewJSONHandler(&salida, nil))
	registrador.Info("prueba", "clave", clave, "intencion", intencion, "recibo", recibo)
	texto := salida.String()
	for _, sensible := range []string{
		datosClavePrueba().SujetoPseudonimizado.Referencia,
		"Convocatoria/2026#1", huellaPrueba("1"),
	} {
		if strings.Contains(texto, sensible) {
			t.Fatalf("formato o log filtró material: %q", sensible)
		}
	}
	for _, marca := range []string{textoClaveOculta, textoIntencionOculta, textoReciboOculto} {
		if !strings.Contains(texto, marca) {
			t.Fatalf("falta marca de redacción %q", marca)
		}
	}
}

func TestCanonicosNoContienenCamposDeIdentidadDirectaNiContexto(t *testing.T) {
	artefactos := []interface{ RepresentacionCanonica() ([]byte, error) }{
		clavePrueba(t), intencionPrueba(t), reciboPrueba(t),
	}
	for _, artefacto := range artefactos {
		contenido, err := artefacto.RepresentacionCanonica()
		if err != nil {
			t.Fatal(err)
		}
		minusculas := strings.ToLower(string(contenido))
		for _, prohibido := range []string{
			`"dni"`, `"nif"`, `"nombre"`, `"apellidos"`, `"correo"`,
			`"actor"`, `"sesion"`, `"autorizacion"`, `"correlacion"`,
			`"instante"`, `"auditoria"`, "12345678z", "persona@example.test",
		} {
			if strings.Contains(minusculas, prohibido) {
				t.Fatalf("representación contiene %q: %s", prohibido, contenido)
			}
		}
	}
}

func TestGettersCopiasYSerializacionDeliberada(t *testing.T) {
	clave := clavePrueba(t)
	datos, err := clave.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if clave.SujetoPseudonimizado() != datos.SujetoPseudonimizado ||
		clave.Convocatoria() != datos.Convocatoria || clave.Reglas() != datos.Reglas ||
		clave.Entrada() != datos.Entrada || clave.Motor() != datos.Motor ||
		clave.HuellaPlanSHA256() != datos.HuellaPlanSHA256 || clave.Causa() != datos.Causa ||
		clave.Tipo() != datos.Tipo {
		t.Fatal("getters de clave divergentes")
	}
	if _, existe := clave.Predecesor(); existe {
		t.Fatal("cálculo inicial expuso predecesor")
	}
	datos.Convocatoria.Referencia = "Manipulada"
	if clave.Convocatoria().Referencia == "Manipulada" {
		t.Fatal("Datos compartió estado mutable")
	}
	intencion := intencionPrueba(t)
	recibo := reciboPrueba(t)
	if recibo.GeneracionClaveHMAC() != 3 || recibo.IndiceHMACSHA256() == "" ||
		recibo.HuellaClaveEfectoSHA256() == "" || recibo.HuellaIntencionSHA256() == "" ||
		recibo.HuellaResultadoSHA256() != intencion.HuellaResultadoSHA256() ||
		recibo.Estado() != ResultadoCompletado || recibo.Fase() != FaseCompletado {
		t.Fatal("getters de recibo divergentes")
	}
	for _, valor := range []any{clave, intencion, recibo} {
		contenido, err := json.Marshal(valor)
		if err != nil || !json.Valid(contenido) {
			t.Fatalf("MarshalJSON inválido: %v", err)
		}
	}
}

func TestDeserializacionJSONDirectaEstaProhibida(t *testing.T) {
	casos := []any{&ClaveEfectoV1{}, &IntencionResultadoV1{}, &ReciboV1{}}
	for _, destino := range casos {
		if err := json.Unmarshal([]byte(`{}`), destino); !errors.Is(err, ErrEntradaNoPermitida) {
			t.Fatalf("Unmarshal directo no rechazado para %T: %v", destino, err)
		}
	}
}

func TestErroresTipadosNoExponenValores(t *testing.T) {
	err := nuevoError("clave.motor", CodigoValorNoCanonico)
	var dominio *ErrorDominio
	if !errors.As(err, &dominio) || dominio.Codigo() != CodigoValorNoCanonico ||
		dominio.Campo() != "clave.motor" || !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("error no clasificable: %v", err)
	}
	if strings.Contains(err.Error(), "dato-secreto") || (&ErrorDominio{}).Error() == "" {
		t.Fatal("error inseguro")
	}
	var nulo *ErrorDominio
	if nulo.Error() == "" || nulo.Codigo() != "" || nulo.Campo() != "" || nulo.Is(err) {
		t.Fatal("receptor nulo no es defensivo")
	}
}

func TestJSONEstrictoRecorreColeccionesYLimites(t *testing.T) {
	if err := comprobarClavesJSONUnicas([]byte(`[{"a":1},{"b":{"c":2}}]`)); err != nil {
		t.Fatalf("JSON válido rechazado: %v", err)
	}
	if err := comprobarClavesJSONUnicas([]byte(`[{"a":1,"a":2}]`)); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("duplicado en colección aceptado: %v", err)
	}
	profundo := strings.Repeat(`[`, maximaProfundidadJSONV1+2) + "0" +
		strings.Repeat(`]`, maximaProfundidadJSONV1+2)
	if err := comprobarClavesJSONUnicas([]byte(profundo)); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("profundidad excesiva aceptada: %v", err)
	}
	campos := bytes.NewBufferString("{")
	for indice := 0; indice <= maximosCamposObjetoV1; indice++ {
		if indice > 0 {
			campos.WriteByte(',')
		}
		_, _ = fmt.Fprintf(campos, `"f%d":%d`, indice, indice)
	}
	campos.WriteByte('}')
	if err := comprobarClavesJSONUnicas(campos.Bytes()); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("demasiados campos aceptados: %v", err)
	}
}
