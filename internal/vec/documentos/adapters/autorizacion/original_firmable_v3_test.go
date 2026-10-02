package autorizacion

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type emisorOriginalNoConcede struct{ llamadas int }

func (*emisorOriginalNoConcede) SeudonimosLecturaOriginal(context.Context, docports.AutorizacionV3) (DatosSeudonimosLectura, error) {
	panic("seudonimos solicitados sin reserva autorizada")
}
func (e *emisorOriginalNoConcede) EmitirConcesionAlmacenV3(context.Context, SolicitudConcesionAlmacenV3) (ConcesionAlmacenV3, error) {
	e.llamadas++
	return ConcesionAlmacenV3{}, errors.New("PDP sin concesion")
}

type emisorOperacionNoConcede struct{ llamadas int }

func (e *emisorOperacionNoConcede) AutorizarOperacionOriginalFirmable(context.Context, string, []byte, string, string) (docports.AutorizacionV3, error) {
	e.llamadas++
	return docports.AutorizacionV3{}, errors.New("PDP sin concesion")
}

func TestOriginalFirmableV3DeniegaSinDecisionSQLNiConcesionAlmacen(t *testing.T) {
	operaciones := &emisorOperacionNoConcede{}
	almacen := &emisorOriginalNoConcede{}
	a, err := NuevaAutoridadOriginalFirmableV3(operaciones, almacen, relojFijo{time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.AutorizarReservaOriginal(context.Background(), []byte(`{"accion":"reserva"}`),
		"ref:"+strings.Repeat("1", 64), "ref:"+strings.Repeat("2", 64)); !errors.Is(err, vecports.ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("reserva sin PDP concedida: %v", err)
	}
	if _, err := a.ContextoEscrituraOriginal(context.Background(), docports.ReservaOriginalFirmable{}, docports.IntentoOriginalFirmable{}); !errors.Is(err, vecports.ErrAutorizacionAlmacenInvalida) || almacen.llamadas != 0 || operaciones.llamadas != 1 {
		t.Fatalf("almacen sin reserva valida: %v, llamadas=%d/%d", err, operaciones.llamadas, almacen.llamadas)
	}
}

func TestRecursoEscrituraOriginalLigaIntentoYHuella(t *testing.T) {
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	r := docports.ReservaOriginalFirmable{ID: ref("1"), ExpedienteRef: ref("2"), TipoRef: ref("3"), Version: 4,
		HuellaSHA256: strings.Repeat("a", 64)}
	i := docports.IntentoOriginalFirmable{ReservaRef: ref("4"), Numero: 2, ClaveAlmacenRef: ref("5")}
	v := vecports.VinculosOperacionAlmacen{OperacionRef: "operacion:ensayo", CargaRef: i.ClaveAlmacenRef,
		Clasificacion: "conservacion", EfectoRef: r.ID,
		SujetoSeudonimoHMAC: "hmac-sha256:sujeto_v1:" + strings.Repeat("b", 64),
		HuellaSolicitudHMAC: "hmac-sha256:solicitud_v1:" + strings.Repeat("c", 64)}
	recurso := RecursoEscrituraOriginalFirmableV3(r, i, v)
	if recurso.Referencia != r.ID || recurso.ModuloID != "documentos" ||
		recurso.Atributos["documentos_original_reserva_ref"] != i.ReservaRef ||
		recurso.Atributos["documentos_original_intento_num"] != "2" ||
		recurso.Atributos["documentos_original_clave_almacen_ref"] != i.ClaveAlmacenRef ||
		recurso.Atributos["documentos_original_huella_sha256"] != r.HuellaSHA256 {
		t.Fatalf("recurso PDP no compromete reserva e intento: %+v", recurso)
	}
	i.ClaveAlmacenRef = ref("6")
	otro := RecursoEscrituraOriginalFirmableV3(r, i, v)
	if otro.Atributos["documentos_original_clave_almacen_ref"] != i.ClaveAlmacenRef ||
		otro.Atributos["documentos_original_clave_almacen_ref"] == recurso.Atributos["documentos_original_clave_almacen_ref"] {
		t.Fatal("otra clave no cambia el recurso PDP")
	}
}
