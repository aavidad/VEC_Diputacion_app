package application

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestNuevoServicioConsultaInternaAutoridadRechazaNulosTipados(t *testing.T) {
	repositorio := &consultaInternaGobernadaAutoridadPrueba{}
	exigidor := &exigidorConsultaInternaAutoridadPrueba{error: domain.ErrAutorizacionDenegada}
	reloj := relojUsoAutorizacionPrueba{ahora: instanteConsultaInternaAutoridadPrueba}
	var repositorioNulo *consultaInternaGobernadaAutoridadPrueba
	var exigidorNulo *exigidorConsultaInternaAutoridadPrueba
	var relojNulo *relojSecuencialFachadaPrueba
	casos := []struct {
		nombre      string
		repositorio ports.ConsultaInternaGobernadaFuentesAutoridad
		exigidor    ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
		reloj       ports.Reloj
	}{
		{"repositorio nulo", nil, exigidor, reloj},
		{"repositorio nulo tipado", repositorioNulo, exigidor, reloj},
		{"exigidor nulo", repositorio, nil, reloj},
		{"exigidor nulo tipado", repositorio, exigidorNulo, reloj},
		{"reloj nulo", repositorio, exigidor, nil},
		{"reloj nulo tipado", repositorio, exigidor, relojNulo},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio, err := NuevoServicioConsultaInternaFuentesAutoridad(
				caso.repositorio, caso.exigidor, caso.reloj,
			)
			if servicio != nil || !errors.Is(err, ErrDependenciaConsultaInternaFuenteAutoridadRequerida) {
				t.Fatalf("dependencia nula aceptada: servicio=%v error=%v", servicio, err)
			}
		})
	}
}

func TestOrdenConsultaInternaAutoridadNoSeSerializaNiExponeCapacidades(t *testing.T) {
	orden := ordenConsultaInternaAutoridadPrueba()
	serializaciones := []func() error{
		func() error { _, err := json.Marshal(orden); return err },
		func() error { _, err := xml.Marshal(orden); return err },
		func() error { var b bytes.Buffer; return gob.NewEncoder(&b).Encode(orden) },
		func() error { _, err := orden.MarshalText(); return err },
		func() error { _, err := orden.MarshalBinary(); return err },
		func() error { _, err := orden.MarshalCBOR(); return err },
		func() error { _, err := orden.MarshalYAML(); return err },
	}
	for indice, serializar := range serializaciones {
		if err := serializar(); !errors.Is(err, ErrSerializacionOrdenConsultaInternaAutoridad) {
			t.Fatalf("serializacion %d aceptada: %v", indice, err)
		}
	}
	var reconstruida OrdenConsultaInternaExactaFuenteAutoridad
	if err := json.Unmarshal([]byte(`{}`), &reconstruida); !errors.Is(err, ErrSerializacionOrdenConsultaInternaAutoridad) {
		t.Fatalf("reconstruccion JSON aceptada: %v", err)
	}
	if err := reconstruida.UnmarshalCBOR([]byte{0xa0}); !errors.Is(err, ErrSerializacionOrdenConsultaInternaAutoridad) {
		t.Fatalf("reconstruccion CBOR aceptada: %v", err)
	}
	if err := reconstruida.UnmarshalYAML(func(any) error { return nil }); !errors.Is(err, ErrSerializacionOrdenConsultaInternaAutoridad) {
		t.Fatalf("reconstruccion YAML aceptada: %v", err)
	}
	texto := fmt.Sprintf("%v %+v %#v", orden, orden, orden)
	correlacionRef, err := orden.Correlacion.ValorCanonico()
	if err != nil {
		t.Fatal(err)
	}
	for _, sensible := range []string{
		orden.Selector.FuenteID, orden.MotivoCatalogo.EntradaClave, orden.MotivoCatalogo.CatalogoID,
		orden.MotivoCatalogo.CatalogoHuellaSHA256, correlacionRef,
	} {
		if strings.Contains(texto, sensible) {
			t.Fatalf("formato expuso %q: %s", sensible, texto)
		}
	}
}

func TestConsultaInternaAutoridadRechazaResultadoSinReciboExacto(t *testing.T) {
	casos := []struct {
		nombre  string
		alterar func(*ports.ResultadoConsultaInternaFuenteAutoridad)
	}{
		{"recibo cero", func(r *ports.ResultadoConsultaInternaFuenteAutoridad) {
			r.Recibo = ports.ReciboConsultaInternaFuenteAutoridad{}
		}},
		{"fuente ausente declarada encontrada", func(r *ports.ResultadoConsultaInternaFuenteAutoridad) {
			r.Fuente = domain.FuenteAutoridadVersionada{}
			r.Estado = ports.ReferenciaEstadoFuenteAutoridad{}
		}},
		{"estado divergente del recibo", func(r *ports.ResultadoConsultaInternaFuenteAutoridad) {
			r.Estado.HuellaEstadoSHA256 = strings.Repeat("f", 64)
		}},
		{"bandera contradictoria", func(r *ports.ResultadoConsultaInternaFuenteAutoridad) {
			r.Encontrada = false
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio, repositorio, _, orden := escenarioConsultaInternaAutoridadPrueba(t, true)
			repositorio.alterar = caso.alterar
			resultado, err := servicio.ConsultarExacta(context.Background(), orden)
			if !errors.Is(err, ErrResultadoConsultaInternaFuenteAutoridadInvalido) ||
				resultado.Encontrada || repositorio.llamadas != 1 {
				t.Fatalf("resultado invalido aceptado: resultado=%v llamadas=%d error=%v",
					resultado, repositorio.llamadas, err)
			}
		})
	}
}

func TestConsultaInternaAutoridadPropagaFalloUnicoDelRepositorio(t *testing.T) {
	servicio, repositorio, _, orden := escenarioConsultaInternaAutoridadPrueba(t, true)
	errorRepositorio := errors.New("fallo durable de prueba")
	repositorio.error = errorRepositorio
	resultado, err := servicio.ConsultarExacta(context.Background(), orden)
	if !errors.Is(err, errorRepositorio) || resultado.Encontrada || repositorio.llamadas != 1 {
		t.Fatalf("fallo durable ocultado: resultado=%v llamadas=%d error=%v",
			resultado, repositorio.llamadas, err)
	}
}

func TestConsultaInternaAutoridadCortaCancelacionDuranteElPEP(t *testing.T) {
	repositorio := &consultaInternaGobernadaAutoridadPrueba{
		encontrada: true, fuente: fuenteConsultaInternaAutoridadPrueba(t),
	}
	ctx, cancelar := context.WithCancel(context.Background())
	autorizador := &autorizadorConsultaInternaAutoridadV2Prueba{
		ahora:          instanteConsultaInternaAutoridadPrueba,
		campos:         []string{ports.CampoConsultaInternaFuenteAutoridad},
		garantiaMinima: domain.AuthAssuranceHigh,
		despues:        cancelar,
	}
	fachada := nuevaFachadaUsoAutorizacionV2AutoridadPrueba(
		t, autorizador, relojUsoAutorizacionPrueba{ahora: instanteConsultaInternaAutoridadPrueba},
	)
	servicio, err := NuevoServicioConsultaInternaFuentesAutoridad(
		repositorio, fachada, relojUsoAutorizacionPrueba{ahora: instanteConsultaInternaAutoridadPrueba},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.ConsultarExacta(ctx, ordenConsultaInternaAutoridadPrueba())
	if !errors.Is(err, context.Canceled) || repositorio.llamadas != 0 || autorizador.llamadas != 1 {
		t.Fatalf("cancelacion no cerrada: repo=%d PDP=%d error=%v",
			repositorio.llamadas, autorizador.llamadas, err)
	}
}

func TestConsultaInternaAutoridadExigeSuperficieInternaAlta(t *testing.T) {
	for _, caso := range []struct {
		nombre     string
		superficie domain.SuperficieAutenticacionActorV1
		garantia   domain.AuthAssurance
	}{
		{"superficie externa", domain.SuperficieAutenticacionExternaPersonalV1, domain.AuthAssuranceHigh},
		{"garantia sustancial", domain.SuperficieAutenticacionInternaCorporativaV1, domain.AuthAssuranceSubstantial},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			repositorio := &consultaInternaGobernadaAutoridadPrueba{
				encontrada: true, fuente: fuenteConsultaInternaAutoridadPrueba(t),
			}
			autorizador := &autorizadorConsultaInternaAutoridadV2Prueba{
				ahora:          instanteConsultaInternaAutoridadPrueba,
				campos:         []string{ports.CampoConsultaInternaFuenteAutoridad},
				garantiaMinima: caso.garantia,
			}
			fachada := nuevaFachadaUsoAutorizacionV2AutoridadPrueba(
				t, autorizador, relojUsoAutorizacionPrueba{ahora: instanteConsultaInternaAutoridadPrueba},
			)
			servicio, err := NuevoServicioConsultaInternaFuentesAutoridad(
				repositorio, fachada, relojUsoAutorizacionPrueba{ahora: instanteConsultaInternaAutoridadPrueba},
			)
			if err != nil {
				t.Fatal(err)
			}
			actor, vinculo := nuevasCredencialesFachadaUsoAutorizacionPrueba(
				t, instanteConsultaInternaAutoridadPrueba, caso.superficie, caso.garantia, 30*time.Minute,
			)
			orden := ordenConsultaInternaAutoridadPrueba()
			orden.ContextoActor, orden.VinculoAutenticacionActor = actor, vinculo
			_, err = servicio.ConsultarExacta(context.Background(), orden)
			if !errors.Is(err, domain.ErrAutorizacionDenegada) || repositorio.llamadas != 0 {
				t.Fatalf("perfil insuficiente aceptado: repo=%d error=%v", repositorio.llamadas, err)
			}
		})
	}
}
