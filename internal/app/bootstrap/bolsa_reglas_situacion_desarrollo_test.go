package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
)

type respuestaReglasSituacionPrueba struct {
	Data struct {
		Esquema      string              `json:"esquema"`
		Configuradas bool                `json:"configuradas"`
		Transiciones map[string][]string `json:"transiciones"`
		CausasBaja   []struct {
			Codigo      string `json:"codigo"`
			Procedencia struct {
				Articulo string `json:"articulo"`
			} `json:"procedencia"`
		} `json:"causas_baja"`
		Reposicion *struct {
			Modalidades []struct {
				Codigo string `json:"codigo"`
				Meses  int    `json:"meses"`
			} `json:"modalidades"`
			Propuesta *struct {
				FechaDisponible string `json:"fecha_disponible"`
				UltimoDia       string `json:"ultimo_dia_no_disponible"`
				Meses           int    `json:"meses"`
				Procedencia     struct {
					Articulo string `json:"articulo"`
				} `json:"procedencia"`
			} `json:"propuesta"`
		} `json:"reposicion"`
	} `json:"data"`
}

func consultarReglasSituacionPrueba(t *testing.T, manejador http.Handler, ruta string) (int, respuestaReglasSituacionPrueba) {
	t.Helper()
	grabadora := httptest.NewRecorder()
	manejador.ServeHTTP(grabadora, httptest.NewRequest(http.MethodGet, ruta, nil))
	var cuerpo respuestaReglasSituacionPrueba
	if grabadora.Code == http.StatusOK {
		if err := json.Unmarshal(grabadora.Body.Bytes(), &cuerpo); err != nil {
			t.Fatal(err)
		}
	}
	return grabadora.Code, cuerpo
}

func TestReglasSituacionBolsaSinCatalogoMantienenLaConductaActual(t *testing.T) {
	// El mutador real sin servicio no debe romper la composición.
	ruta, err := componerReglasSituacionBolsaDesarrollo(nil, &manejadorParticipacionBolsaDesarrollo{})
	if err != nil || ruta.Ruta != rutaReglasSituacionBolsaDesarrollo {
		t.Fatalf("ruta=%+v err=%v", ruta, err)
	}
	codigo, cuerpo := consultarReglasSituacionPrueba(t, ruta.Manejador, rutaReglasSituacionBolsaDesarrollo)
	if codigo != http.StatusOK || cuerpo.Data.Configuradas || cuerpo.Data.Reposicion != nil || len(cuerpo.Data.CausasBaja) != 0 ||
		!slices.Equal(cuerpo.Data.Transiciones["renuncia"], []string{"disponible", "excluido"}) {
		t.Fatalf("sin catálogo: %d %+v", codigo, cuerpo.Data)
	}
}

func TestReglasSituacionBolsaConCatalogoProponenLaReposicionDeFechaAFecha(t *testing.T) {
	compuestas, err := nuevasReglasEjemploDesarrollo(configuracionDesarrolloReglasEjemplo(rutaReglasBolsaEjemploPrueba, ""), nil, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	servicio := &aplicacionbolsa.ServicioSituacionParticipacion{}
	ruta, err := componerReglasSituacionBolsaDesarrollo(compuestas.bolsa, &manejadorParticipacionBolsaDesarrollo{servicioSituacion: servicio})
	if err != nil {
		t.Fatal(err)
	}
	codigo, cuerpo := consultarReglasSituacionPrueba(t, ruta.Manejador, rutaReglasSituacionBolsaDesarrollo)
	if codigo != http.StatusOK || !cuerpo.Data.Configuradas || cuerpo.Data.Esquema != "vec.bolsa.rrhh.reglas_situacion.v1" ||
		!slices.Equal(cuerpo.Data.Transiciones["renuncia"], []string{"excluido"}) ||
		!slices.Equal(cuerpo.Data.Transiciones["trabajando"], []string{"disponible", "excluido", "disponible_desde"}) ||
		len(cuerpo.Data.CausasBaja) != 6 || cuerpo.Data.CausasBaja[3].Codigo != "sin_contacto" ||
		cuerpo.Data.Reposicion == nil || cuerpo.Data.Reposicion.Propuesta != nil || len(cuerpo.Data.Reposicion.Modalidades) != 1 {
		t.Fatalf("con catálogo: %d %+v", codigo, cuerpo.Data)
	}
	casos := []struct {
		consulta, disponible, ultimo string
		meses                        int
	}{
		// 31 de enero + 5 meses: junio no tiene 31, el plazo acaba el 30.
		{"?fin_relacion=2026-01-31", "2026-06-30T22:00:00Z", "2026-06-30", 5},
		{"?fin_relacion=2026-01-31&modalidad=general", "2026-06-30T22:00:00Z", "2026-06-30", 5},
		// Acumulación de tareas: 9 meses; en noviembre ya rige el horario de invierno.
		{"?fin_relacion=2026-01-31&modalidad=acumulacion_tareas", "2026-10-31T23:00:00Z", "2026-10-31", 9},
	}
	for _, caso := range casos {
		codigo, cuerpo := consultarReglasSituacionPrueba(t, ruta.Manejador, rutaReglasSituacionBolsaDesarrollo+caso.consulta)
		propuesta := cuerpo.Data.Reposicion.Propuesta
		if codigo != http.StatusOK || propuesta == nil || propuesta.FechaDisponible != caso.disponible ||
			propuesta.UltimoDia != caso.ultimo || propuesta.Meses != caso.meses || propuesta.Procedencia.Articulo != "art. 9.1" {
			t.Fatalf("%s: %d %+v", caso.consulta, codigo, propuesta)
		}
	}
	for _, consulta := range []string{"?modalidad=general", "?fin_relacion=31-01-2026", "?fin_relacion=2026-01-31&otro=1", "?fin_relacion=2026-01-31&fin_relacion=2026-02-01"} {
		if codigo, _ := consultarReglasSituacionPrueba(t, ruta.Manejador, rutaReglasSituacionBolsaDesarrollo+consulta); codigo != http.StatusBadRequest {
			t.Errorf("%s: se esperaba 400 y llegó %d", consulta, codigo)
		}
	}
}
