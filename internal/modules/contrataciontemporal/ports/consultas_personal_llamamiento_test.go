package ports

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestConsultasPersonalLlamamientoConservaPaginasYRechazaDatosAjenos(t *testing.T) {
	original := `[ {"auditoria_ref":"auditoria:uno", "recibo_ref":"recibo:uno"}, {"recibo_ref":"recibo:dos","auditoria_ref":"auditoria:dos"} ]`
	var consultas ConsultasPersonalLlamamiento
	if err := json.Unmarshal([]byte(original), &consultas); err != nil {
		t.Fatal(err)
	}
	esperado := `[{"recibo_ref":"recibo:uno","auditoria_ref":"auditoria:uno"},{"recibo_ref":"recibo:dos","auditoria_ref":"auditoria:dos"}]`
	contenido, err := json.Marshal(consultas)
	if err != nil || string(contenido) != esperado {
		t.Fatalf("páginas perdidas o reordenadas: %s, %v", contenido, err)
	}
	for _, invalido := range []string{
		`null`, `[]`, `{}`, `[{"recibo_ref":"recibo:uno"}]`,
		`[{"recibo_ref":"recibo:uno","auditoria_ref":"auditoria:uno","puesto":"extra"}]`,
		`[{"recibo_ref":"recibo:uno","recibo_ref":"recibo:otro","auditoria_ref":"auditoria:uno"}]`,
		`[{"recibo_ref":"correo@example.test","auditoria_ref":"auditoria:uno"}]`,
		`[` + strings.Repeat(`{"recibo_ref":"recibo:uno","auditoria_ref":"auditoria:uno"},`, 64) + `{"recibo_ref":"recibo:fin","auditoria_ref":"auditoria:fin"}]`,
	} {
		var recibida ConsultasPersonalLlamamiento
		if err := json.Unmarshal([]byte(invalido), &recibida); err == nil {
			t.Fatalf("consulta inválida admitida: %s", invalido)
		}
	}
}

func TestConsultasPersonalLlamamientoQuedanFirmadasEnArtefactoDurable(t *testing.T) {
	base := instanteBolsaPrueba()
	ahora := base.Add(3 * time.Minute)
	sellador := selladorRespuestaBolsaPrueba()
	comando := comandoLlamamientoPrueba(t, base, sellador)
	recibo := reciboLlamamientoPrueba(t, comando, base)
	recibo.Procedencia.SinRetencionDeclarada = true
	recibo.Procedencia.Evidencia.RetenerHasta = time.Time{}
	recibo.RetencionSeleccion = ReferenciaVersionadaIntegracionBolsa{}
	consultas, err := NuevasConsultasPersonalLlamamiento([]ConsultaPersonalLlamamiento{
		{ReciboRef: "recibo:pagina:uno", AuditoriaRef: "auditoria:pagina:uno"},
		{ReciboRef: "recibo:pagina:dos", AuditoriaRef: "auditoria:pagina:dos"},
	})
	if err != nil {
		t.Fatal(err)
	}
	recibo.Procedencia.ConsultasPersonal = consultas
	firmarLlamamientoPrueba(t, sellador, comando, &recibo)
	verificador := verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba)
	comprobante, evidencia, err := verificador.VerificarReciboLlamamiento(t.Context(), comando, recibo, ahora)
	if err != nil {
		t.Fatal(err)
	}
	artefacto, err := NuevoArtefactoProbatorioLlamamientoBolsa(comando, recibo, evidencia, comprobante)
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(artefacto)
	if err != nil {
		t.Fatal(err)
	}
	recuperado, err := DecodificarArtefactoProbatorioLlamamientoBolsa(contenido)
	if err != nil || recuperado.Recibo != recibo || recuperado.Recibo.Procedencia.ConsultasPersonal != consultas {
		t.Fatalf("recuperación perdió procedencia: %v", err)
	}
	// Otra instancia del verificador vuelve a comprobar los bytes conservados.
	verificadorNuevo := verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba)
	if _, _, err := verificadorNuevo.VerificarReciboLlamamiento(t.Context(), comando, recuperado.Recibo, ahora); err != nil {
		t.Fatal(err)
	}
	alterado := recuperado.Recibo
	alterado.Procedencia.ConsultasPersonal = ConsultasPersonalLlamamiento(strings.Replace(string(consultas), "recibo:pagina:uno", "recibo:pagina:otro", 1))
	if _, _, err := verificadorNuevo.VerificarReciboLlamamiento(t.Context(), comando, alterado, ahora); err == nil {
		t.Fatal("el sello no protege el recibo de Personal")
	}
	legado := procedenciaBolsaPrueba(base)
	antes := nuevoCanonicoBolsa("procedencia")
	antes.procedencia(legado)
	legado.ConsultasPersonal = consultas
	despues := nuevoCanonicoBolsa("procedencia")
	despues.procedencia(legado)
	if !bytes.HasPrefix(despues.bytes(), antes.bytes()) || legado.consultasPersonalValidas() {
		t.Fatal("la extensión cambió el canon previo o se admitió fuera del circuito nativo")
	}
}
