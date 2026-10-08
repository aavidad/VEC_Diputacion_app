package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	bolsadominio "vec-diputacion-granada/internal/modules/bolsa/domain"
)

type canalesLlamamientoPrueba []bolsadominio.CanalLlamamiento

func (c canalesLlamamientoPrueba) CanalesLlamamientoActivos(context.Context) []bolsadominio.CanalLlamamiento {
	return c
}

type comprobadorTelefonoPrueba struct {
	instalado bool
	err       error
}

func (c comprobadorTelefonoPrueba) RegistroTelefonoInstalado(context.Context) (bool, error) {
	return c.instalado, c.err
}

func TestCanalesLlamamientoSoloPublicaTelefonoConB87(t *testing.T) {
	ctx := context.Background()
	for _, caso := range []struct {
		nombre    string
		telefono  comprobadorTelefonoPrueba
		esperados []string
	}{
		{"con B87", comprobadorTelefonoPrueba{instalado: true}, []string{"correo", "telefono"}},
		{"sin B87", comprobadorTelefonoPrueba{}, []string{"correo"}},
		{"lectura caída", comprobadorTelefonoPrueba{err: errors.New("caída")}, []string{"correo"}},
	} {
		registro, err := componerCanalesLlamamientoBolsaDesarrollo(true, caso.telefono)
		if err != nil {
			t.Fatalf("%s: %v", caso.nombre, err)
		}
		activos := registro.Activos(ctx)
		if len(activos) != len(caso.esperados) {
			t.Fatalf("%s: %+v", caso.nombre, activos)
		}
		for i, canal := range activos {
			if canal.Canal != caso.esperados[i] {
				t.Fatalf("%s: canal %d=%s", caso.nombre, i, canal.Canal)
			}
		}
		if faltan := registro.CanalesSinProveedor(); len(faltan) != 0 {
			t.Fatalf("%s: SMS/Telegram deben venir apagados en el catálogo: %v", caso.nombre, faltan)
		}
	}
}

func TestCandidatosPorLlamamientoYCanalesPublicados(t *testing.T) {
	datos := datosBolsasRRHHPrueba()
	fecha := time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC)
	llamamiento := "llamamiento:0123456789abcdef"
	contactos := []bolsadominio.ContactoParticipacion{
		{ContactoRef: "contacto:1", BolsaRef: datos.Bolsas[0].Referencia, ParticipacionRef: "participacion:002", LlamamientoRef: llamamiento, Canal: "correo", Resultado: bolsadominio.ResultadoContactoEnviado, Instante: fecha},
		{ContactoRef: "contacto:2", BolsaRef: datos.Bolsas[0].Referencia, ParticipacionRef: "participacion:002", LlamamientoRef: llamamiento, Canal: "telefono", Resultado: bolsadominio.ResultadoContactoComunica, Instante: fecha.Add(time.Hour)},
		{ContactoRef: "contacto:3", BolsaRef: datos.Bolsas[0].Referencia, ParticipacionRef: "participacion:001", LlamamientoRef: "llamamiento:otro-llamamiento", Canal: "correo", Resultado: bolsadominio.ResultadoContactoEnviado, Instante: fecha},
	}
	manejador := nuevoManejadorBolsasRRHHDesarrollo(func(context.Context) (datasetBolsasRRHHDesarrollo, error) { return datos, nil })
	manejador.contactos = lectorContactosTurnoPrueba{primera: contactos}
	manejador.canales = canalesLlamamientoPrueba{{Canal: "correo", Modo: "automatico", AlEmitir: true, Resultados: []string{"enviado"}, Activo: true}}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/bolsa:constituida:administrativo/candidatos?llamamiento="+llamamiento+"&limite=100", nil))
	if respuesta.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", respuesta.Code, respuesta.Body.String())
	}
	var salida struct {
		Data struct {
			Candidatos []struct {
				Referencia string `json:"participacion_ref"`
			} `json:"candidatos"`
			Canales []bolsadominio.CanalLlamamiento `json:"canales_llamamiento"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &salida); err != nil || len(salida.Data.Candidatos) != 1 || salida.Data.Candidatos[0].Referencia != "participacion:002" || len(salida.Data.Canales) != 1 || salida.Data.Canales[0].Canal != "correo" {
		t.Fatalf("filtro por llamamiento o canales: %+v err=%v", salida, err)
	}
	for _, consulta := range []string{"llamamiento=" + llamamiento + "&estado=disponible", "llamamiento=" + llamamiento + "&cursor=participacion:001", "llamamiento=otro", "llamamiento=llamamiento:con%20espacio"} {
		mala := httptest.NewRecorder()
		manejador.ServeHTTP(mala, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/bolsa:constituida:administrativo/candidatos?"+consulta, nil))
		if mala.Code != http.StatusBadRequest {
			t.Fatalf("%s: status=%d", consulta, mala.Code)
		}
	}
}
