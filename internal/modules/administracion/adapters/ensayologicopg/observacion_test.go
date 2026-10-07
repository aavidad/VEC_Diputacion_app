package ensayologicopg

import (
	"context"
	"errors"
	"strings"
	"testing"

	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ensayofisicopg"
)

type observadorLecturaReal struct {
	llamado bool
	fallar  bool
}

func (o *observadorLecturaReal) Observar(ctx context.Context, e puertos.Entorno) (puertos.Observacion, error) {
	antes, err := e.PostgreSQL.ComprobarExclusion(ctx)
	if err != nil {
		return puertos.Observacion{}, err
	}
	b, err := e.PostgreSQL.EjecutarPostgreSQL(ctx, "psql", []string{"-X", "-At", "-U", "cs06_destino", "-d", "cs06l_sintetica", "-c", "SELECT count(*) FROM cs06l_sintetico.registros"}, nil, 4096)
	if err != nil || strings.TrimSpace(string(b)) != "1" {
		return puertos.Observacion{}, errors.New("contenido_no_comprobable")
	}
	despues, err := e.PostgreSQL.ComprobarExclusion(ctx)
	if err != nil || despues != antes {
		return puertos.Observacion{}, errors.New("aislamiento_cambiado")
	}
	o.llamado = true
	if o.fallar {
		return puertos.Observacion{}, errors.New("observador_fallo_sintetico")
	}
	return puertos.Observacion{ContrasteEstado: "no_comprobable", ArranqueEstado: "no_comprobable"}, nil
}
func verificarObservadores(t *testing.T, ctx context.Context, c Configuracion, dump, globals Archivo) {
	t.Helper()
	o := &observadorLecturaReal{}
	r := (Ensayador{Configuracion: c, Observador: o}).Ensayar(ctx, Solicitud{Sintetica: true, Dump: dump, Globals: globals})
	if !o.llamado || r.Estado != "restauracion_logica_completada" || !r.LimpiezaCompletada || r.HabilitaRestauracion || r.Observacion.ArranqueEstado != "no_comprobable" {
		t.Fatalf("hook real: %+v llamado %v", r, o.llamado)
	}
	o = &observadorLecturaReal{fallar: true}
	r = (Ensayador{Configuracion: c, Observador: o}).Ensayar(ctx, Solicitud{Sintetica: true, Dump: dump, Globals: globals})
	if !o.llamado || r.Etapa != "observacion" || r.Estado != "restauracion_logica_fallida" || !r.LimpiezaCompletada {
		t.Fatalf("fallo callback cleanup: %+v llamado %v", r, o.llamado)
	}
}
