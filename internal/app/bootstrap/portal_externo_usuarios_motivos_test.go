package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func solicitudMotivosUsuariosPrueba() SolicitudMotivosUsuariosExterno {
	var entradas []EntradaMotivosUsuariosExterno
	for _, tipo := range []string{"consulta", "actualizacion"} {
		entradas = append(entradas, EntradaMotivosUsuariosExterno{Tipo: tipo, Clave: claveEntradaMotivosUsuariosExterno(tipo),
			ModuloID: "usuarios", Acciones: accionesEntradaMotivosUsuariosExterno(tipo), Etiquetas: map[string]string{"xx": "etiqueta de prueba"}})
	}
	return SolicitudMotivosUsuariosExterno{AprobacionRef: "aprobacion:usuarios:motivos:sintetica", SecuenciaEsperada: 37,
		Catalogo: CatalogoMotivosUsuariosExterno{CatalogoID: catalogoMotivosUsuariosExterno, Version: 1,
			PublicadoEn: time.Date(2026, 9, 30, 1, 0, 0, 0, time.UTC), Entradas: entradas}}
}

func TestMotivosUsuariosExternoSoloCatalogoDosEntradasDiezAcciones(t *testing.T) {
	s := solicitudMotivosUsuariosPrueba()
	r, err := s.Resumen()
	if err != nil || r.Secuencia != 38 || r.MotivoConsulta.CatalogoID != catalogoMotivosUsuariosExterno || r.MotivoConsulta.EntradaClave == r.MotivoActualizacion.EntradaClave {
		t.Fatalf("plan propio rechazado: %v", err)
	}
	for _, cambiar := range []func(*SolicitudMotivosUsuariosExterno){
		func(s *SolicitudMotivosUsuariosExterno) { s.Catalogo.CatalogoID = "motivos_otro_modulo" },
		func(s *SolicitudMotivosUsuariosExterno) { s.Catalogo.Entradas[0].Clave = s.Catalogo.Entradas[1].Clave },
		func(s *SolicitudMotivosUsuariosExterno) { s.Catalogo.Entradas[0].ModuloID = "bolsa" },
		func(s *SolicitudMotivosUsuariosExterno) {
			s.Catalogo.Entradas[0].Acciones = append(s.Catalogo.Entradas[0].Acciones, "accion_ajena")
		},
		func(s *SolicitudMotivosUsuariosExterno) { s.Catalogo.Entradas = s.Catalogo.Entradas[:1] },
		func(s *SolicitudMotivosUsuariosExterno) { s.Catalogo.Entradas[0].Etiquetas = nil },
		func(s *SolicitudMotivosUsuariosExterno) { s.SecuenciaEsperada = math.MaxInt64 },
		func(s *SolicitudMotivosUsuariosExterno) { s.AprobacionRef = "" },
	} {
		otra := solicitudMotivosUsuariosPrueba()
		cambiar(&otra)
		if _, err := otra.HuellaSHA256(); err == nil {
			t.Fatal("plan de otra autoridad admitido")
		}
	}
}

func TestMotivosUsuariosExternoHuellaLigaContenidoPreimagenYAprobacion(t *testing.T) {
	s := solicitudMotivosUsuariosPrueba()
	h, err := s.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	for _, cambiar := range []func(*SolicitudMotivosUsuariosExterno){
		func(s *SolicitudMotivosUsuariosExterno) { s.SecuenciaEsperada++ },
		func(s *SolicitudMotivosUsuariosExterno) { s.AprobacionRef += ":otra" },
		func(s *SolicitudMotivosUsuariosExterno) { s.Catalogo.Version++ },
		func(s *SolicitudMotivosUsuariosExterno) { s.Catalogo.Entradas[0].Etiquetas["xx"] += " modificada" },
	} {
		otra := solicitudMotivosUsuariosPrueba()
		cambiar(&otra)
		nueva, err := otra.HuellaSHA256()
		if err != nil || nueva == h {
			t.Fatal("material modificado conserva la aprobación", err)
		}
	}
	if _, err := PublicarMotivosUsuariosExterno(t.Context(), nil, s, h, "otra_aprobacion"); err == nil {
		t.Fatal("publicación sin aprobación exacta admitida")
	}
}

func TestMotivosUsuariosExternoFuenteNoAceptaIdentidadNiJSONResidual(t *testing.T) {
	b, err := json.Marshal(solicitudMotivosUsuariosPrueba())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LeerSolicitudMotivosUsuariosExterno(b); err != nil {
		t.Fatal(err)
	}
	for _, alterada := range [][]byte{
		append(append([]byte(nil), b...), []byte(" resto")...),
		append([]byte(`{"cuenta_ref":"cta_personal_ajena",`), b[1:]...),
		append([]byte(`{"aprobacion_ref":"otra",`), b[1:]...),
	} {
		if _, err := LeerSolicitudMotivosUsuariosExterno(alterada); err == nil {
			t.Fatal("fuente con identidad o residual admitida")
		}
	}
}

type txMotivosUsuariosPrueba struct {
	args     [][]any
	aceptada bool
	fallo    error
	commit   int
}

func (t *txMotivosUsuariosPrueba) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	t.args = append(t.args, append([]any(nil), args...))
	return filaProvisionUsuariosPrueba{valores: []any{t.aceptada}, err: t.fallo}
}
func (t *txMotivosUsuariosPrueba) Commit(context.Context) error { t.commit++; return nil }

func TestMotivosUsuariosExternoCASYReplayConservanEventoSecuenciaYContenido(t *testing.T) {
	s := solicitudMotivosUsuariosPrueba()
	r, err := s.Resumen()
	if err != nil {
		t.Fatal(err)
	}
	tx := &txMotivosUsuariosPrueba{aceptada: true}
	for i := 0; i < 2; i++ {
		obtenido, err := publicarMotivosUsuariosEnTransaccion(t.Context(), tx, s, r)
		if err != nil || obtenido.HuellaPlan != r.HuellaPlan || obtenido.Estado != "confirmado" {
			t.Fatal("recibo distinto", err)
		}
	}
	if len(tx.args) != 2 || !reflect.DeepEqual(tx.args[0], tx.args[1]) || tx.args[0][1] != int64(38) || tx.args[0][3] != catalogoMotivosUsuariosExterno {
		t.Fatal("replay con otro contenido o secuencia")
	}
	var entradas []map[string]any
	if json.Unmarshal(tx.args[0][7].([]byte), &entradas) != nil || len(entradas) != 2 {
		t.Fatal("no publicó juntas ambas entradas")
	}
	for _, e := range entradas {
		if len(e) != 3 {
			t.Fatal("proyección contiene datos fuera del contrato")
		}
	}
	rechazada := &txMotivosUsuariosPrueba{}
	if _, err := publicarMotivosUsuariosEnTransaccion(t.Context(), rechazada, s, r); err == nil || rechazada.commit != 0 {
		t.Fatal("ignoró rechazo CAS")
	}
	falla := &txMotivosUsuariosPrueba{fallo: errors.New("detalle_privado")}
	if _, err := publicarMotivosUsuariosEnTransaccion(t.Context(), falla, s, r); err != errMotivosUsuariosExterno || falla.commit != 0 {
		t.Fatal("filtró error o confirmó publicación fallida")
	}
}
