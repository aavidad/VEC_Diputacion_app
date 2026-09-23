package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

type consultaAvisosPrueba struct{ cortes []time.Time }

func (c *consultaAvisosPrueba) ContarAvisosRRHH(_ context.Context, corte time.Time) (map[string]int, error) {
	c.cortes = append(c.cortes, corte)
	return map[string]int{dominiobolsa.AvisoSaltoOrden: 1, dominiobolsa.AvisoTresAnos: 1}, nil
}

func TestAvisosRechazanCursoresManipuladosOFueraDeVigencia(t *testing.T) {
	instante := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	servicio, err := NuevoServicioAvisosRRHH(&consultaAvisosPrueba{}, func() time.Time { return instante })
	if err != nil {
		t.Fatal(err)
	}
	casos := []cursorAvisos{
		{Corte: instante.Add(time.Second), Offset: 1},
		{Corte: instante.Add(-vigenciaCursorAvisos - time.Microsecond), Offset: 1},
		{Corte: instante, Offset: maximoOffsetAvisos + 1},
	}
	for _, caso := range casos {
		contenido, err := json.Marshal(caso)
		if err != nil {
			t.Fatal(err)
		}
		cursor := base64.RawURLEncoding.EncodeToString(contenido)
		if _, err := servicio.Consultar(context.Background(), ConsultaAvisos{Limite: 1, Cursor: cursor}); err == nil {
			t.Fatalf("se aceptó el cursor manipulado: %#v", caso)
		}
	}
}

func (c *consultaAvisosPrueba) ListarAvisosRRHH(_ context.Context, corte time.Time, offset, limite int) ([]dominiobolsa.AvisoRRHH, error) {
	c.cortes = append(c.cortes, corte)
	filas := []dominiobolsa.AvisoRRHH{
		{Tipo: dominiobolsa.AvisoSaltoOrden, BolsaRef: "bolsa:1", Referencia: "aviso:1", Fecha: corte, Detalle: map[string]any{"participacion_ref": "participacion:1"}},
		{Tipo: dominiobolsa.AvisoTresAnos, BolsaRef: "bolsa:1", Referencia: "aviso:2", Fecha: corte, Detalle: map[string]any{"participacion_ref": "participacion:2"}},
	}
	if offset >= len(filas) {
		return nil, nil
	}
	fin := offset + limite
	if fin > len(filas) {
		fin = len(filas)
	}
	return filas[offset:fin], nil
}

func TestAvisosPaginanConElMismoCorteTemporal(t *testing.T) {
	instante := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	consulta := &consultaAvisosPrueba{}
	servicio, err := NuevoServicioAvisosRRHH(consulta, func() time.Time { return instante })
	if err != nil {
		t.Fatal(err)
	}
	primera, err := servicio.Consultar(context.Background(), ConsultaAvisos{Limite: 1})
	if err != nil || len(primera.Avisos) != 1 || primera.CursorSiguiente == "" {
		t.Fatalf("primera pagina inesperada: %#v, %v", primera, err)
	}
	segunda, err := servicio.Consultar(context.Background(), ConsultaAvisos{Limite: 1, Cursor: primera.CursorSiguiente})
	if err != nil || len(segunda.Avisos) != 1 || segunda.Avisos[0].Referencia != "aviso:2" {
		t.Fatalf("segunda pagina inesperada: %#v, %v", segunda, err)
	}
	if len(consulta.cortes) != 4 || !consulta.cortes[0].Equal(consulta.cortes[1]) || !consulta.cortes[0].Equal(consulta.cortes[2]) || !consulta.cortes[0].Equal(consulta.cortes[3]) {
		t.Fatalf("el corte temporal debe conservarse: %#v", consulta.cortes)
	}
}
