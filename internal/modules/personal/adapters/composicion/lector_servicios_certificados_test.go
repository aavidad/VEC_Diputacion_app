package composicion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func identidadServiciosCertificadosPrueba(t *testing.T) IdentidadRegistradaLectorServiciosCertificados {
	t.Helper()
	id := identidadIntentoRPTVigenciaPrueba(t, time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC), "")
	return IdentidadRegistradaLectorServiciosCertificados{Vinculo: id.Vinculo, Resultado: id.Resultado}
}

func contextoIntentoServiciosCertificadosPrueba(t *testing.T, id IdentidadRegistradaLectorServiciosCertificados) context.Context {
	t.Helper()
	ctx, err := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ref, err := vecports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		t.Fatal(err)
	}
	return context.WithValue(ctx, claveIntentoLectorServiciosCertificados{}, &intentoLectorServiciosCertificados{identidad: id, referencia: ref})
}

func configuracionIntentosServiciosCertificadosPrueba() ConfiguracionIntentosLectorServiciosCertificados {
	c := configuracionIntentosRPTPrueba()
	return ConfiguracionIntentosLectorServiciosCertificados{Proceso: c.Proceso, Canal: c.Canal, RecursoEntradaInvalida: "personal:servicios_certificados",
		MotivoDenegado: c.MotivoDenegado, MotivoEntradaInvalida: c.MotivoEntradaInvalida, MotivoNoDisponible: c.MotivoNoDisponible}
}

type emisorServiciosCertificadosPrueba struct {
	llamadas  int
	solicitud vecdomain.SolicitudAutorizacionLigadaV3
	err       error
}

func (e *emisorServiciosCertificadosPrueba) EmitirMaterialAutorizacionAtestadaV3(_ context.Context, s vecdomain.SolicitudAutorizacionLigadaV3, _ vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	e.solicitud = s
	if e.err == nil {
		e.err = errors.New("doble: sin material firmado")
	}
	return vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, e.err
}

func motivoServiciosCertificadosPrueba() vecdomain.ReferenciaEntradaCatalogo {
	return configuracionIntentosRPTPrueba().MotivoDenegado
}

func TestProveedorServiciosCertificadosPideAccionPropiaParaLaIdentidadCapturada(t *testing.T) {
	id := identidadServiciosCertificadosPrueba(t)
	actor := id.Resultado.Contexto
	m, err := domain.NuevoMaterialLectorServiciosCertificados(domain.SolicitudLectorServiciosCertificados{Actor: actor, EmpleadoRef: "emp_0123456789abcdefghijkl", OrganismoRef: "dipgra",
		Corte: domain.CorteEmpleadoB2{VigenteEn: "2026-10-02", ConocidoEn: time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC)}})
	if err != nil {
		t.Fatal(err)
	}
	emisor := &emisorServiciosCertificadosPrueba{err: vecports.ErrDenegacionExplicitaAutorizacionLigadaV3}
	p, err := NuevoProveedorAutorizacionLectorServiciosCertificados(emisor, motivoServiciosCertificadosPrueba())
	if err != nil {
		t.Fatal(err)
	}
	// Sin identidad capturada en la frontera no se pide concesión.
	if _, err := p.AutorizarServiciosParaCertificados(context.Background(), m); !errors.Is(err, domain.ErrLectorServiciosCertificadosNoDisponible) || emisor.llamadas != 0 {
		t.Fatal("emitió sin identidad de la petición", err)
	}
	_, err = p.AutorizarServiciosParaCertificados(contextoIntentoServiciosCertificadosPrueba(t, id), m)
	datos, errDatos := emisor.solicitud.Datos()
	if !errors.Is(err, domain.ErrLectorServiciosCertificadosDenegado) || emisor.llamadas != 1 || errDatos != nil ||
		datos.Accion != ports.AccionServiciosParaCertificadosV1 || datos.Finalidad != domain.FinalidadLectorServiciosCertificados || datos.Recurso.Tipo != domain.TipoRecursoLectorServiciosCertificados {
		t.Fatalf("solicitud no nominal: %v", err)
	}
	// Un actor distinto del capturado no obtiene concesión prestada.
	otro := identidadIntentoRPTVigenciaPrueba(t, time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC), "contexto")
	emisor.llamadas = 0
	if _, err := p.AutorizarServiciosParaCertificados(contextoIntentoServiciosCertificadosPrueba(t, IdentidadRegistradaLectorServiciosCertificados{Vinculo: otro.Vinculo, Resultado: otro.Resultado}), m); !errors.Is(err, domain.ErrLectorServiciosCertificadosDenegado) || emisor.llamadas != 0 {
		t.Fatal("actor ajeno admitido", err)
	}
}

func TestIntentosServiciosCertificadosRegistranHuellaDelEmpleadoPropio(t *testing.T) {
	id := identidadServiciosCertificadosPrueba(t)
	d := &destinoIntentosRPTPrueba{}
	r, err := NuevoRegistroIntentosLectorServiciosCertificados(d, configuracionIntentosServiciosCertificadosPrueba())
	if err != nil {
		t.Fatal(err)
	}
	if err := r.RegistrarIntentoServiciosCertificados(context.Background(), ports.IntentoLectorServiciosCertificados{Motivo: "denegado"}); !errors.Is(err, domain.ErrLectorServiciosCertificadosNoDisponible) || len(d.ordenes) != 0 {
		t.Fatal("identidad inventada", err)
	}
	if err := r.RegistrarIntentoServiciosCertificados(contextoIntentoServiciosCertificadosPrueba(t, id), ports.IntentoLectorServiciosCertificados{Motivo: "denegado"}); err != nil || len(d.ordenes) != 1 {
		t.Fatal(err)
	}
	datos, _ := d.ordenes[0].Datos()
	h := sha256.Sum256([]byte("emp_0123456789abcdefghijkl"))
	if datos.Datos.RecursoRef != "personal:servicios_certificados:sha256:"+hex.EncodeToString(h[:]) || datos.Datos.Resultado != vecdomain.ResultadoIntentoAuditoriaDenegado ||
		datos.Datos.Accion != ports.AccionServiciosParaCertificadosV1 || datos.ResultadoContexto.Contexto.PersonaRef != id.Resultado.Contexto.PersonaRef {
		t.Fatalf("intento mal atribuido: %+v", datos.Datos)
	}
	d2 := &destinoIntentosRPTPrueba{}
	r2, _ := NuevoRegistroIntentosLectorServiciosCertificados(d2, configuracionIntentosServiciosCertificadosPrueba())
	if err := r2.RegistrarIntentoServiciosCertificados(contextoIntentoServiciosCertificadosPrueba(t, id), ports.IntentoLectorServiciosCertificados{Motivo: "entrada_invalida"}); err != nil {
		t.Fatal(err)
	}
	datos, _ = d2.ordenes[0].Datos()
	if datos.Datos.RecursoRef != "personal:servicios_certificados" {
		t.Fatal("una entrada inválida no debe nombrar al empleado")
	}
}

func TestComponerLectorServiciosCertificadosCierraSinDependencias(t *testing.T) {
	if _, err := ComponerLectorServiciosCertificados(DependenciasLectorServiciosCertificados{}); !errors.Is(err, domain.ErrLectorServiciosCertificadosNoDisponible) {
		t.Fatal("compuesto sin dependencias", err)
	}
	if _, err := NuevoLectorServiciosCertificadosConIdentidad(nil, nil, time.Second); err == nil {
		t.Fatal("frontera sin lector admitida")
	}
	if strings.Contains(clasificarErrorAutorizacionLectorServiciosCertificados(context.Background(), errors.New("x")).Error(), "x") {
		t.Fatal("filtra el detalle")
	}
}
