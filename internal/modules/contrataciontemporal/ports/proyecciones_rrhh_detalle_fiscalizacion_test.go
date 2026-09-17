package ports_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestDetalleRRHHMinimizadoV3FiscalizacionEnlazaHitosYConservaV2(t *testing.T) {
	t.Parallel()
	datos := datosDetalleMinimizadoPrueba(3)
	fiscalizadaEn := datos.hitos[len(datos.hitos)-1].RealizadaEn.Add(time.Minute)
	datos.hitos = append(datos.hitos,
		ports.HitoExpedienteRRHH{
			Secuencia: 5, VersionExpediente: 5,
			AccionClave: domain.AccionRegistrarFiscalizacion,
			RealizadaEn: fiscalizadaEn,
			FaseOrigen:  datos.hitos[3].FaseDestino, FaseDestino: domain.FaseSubsanacionUnidad,
			EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoIncidencia,
		},
		ports.HitoExpedienteRRHH{
			Secuencia: 6, VersionExpediente: 6,
			AccionClave: domain.AccionRegistrarSubsanacionReparo,
			RealizadaEn: fiscalizadaEn.Add(time.Minute),
			FaseOrigen:  domain.FaseSubsanacionUnidad, FaseDestino: domain.FaseSubsanacionUnidad,
			EstadoOrigen: domain.EstadoIncidencia, EstadoDestino: domain.EstadoIncidencia,
		},
	)
	datos.resumen.Version = 6
	datos.resumen.FaseClave = domain.FaseSubsanacionUnidad
	datos.resumen.EstadoClave = domain.EstadoIncidencia
	datos.resumen.ActualizadoEn = datos.hitos[5].RealizadaEn

	refFiscalizacion, err := ports.NuevaReferenciaHitoFiscalizacionRRHH(5)
	if err != nil {
		t.Fatal(err)
	}
	refSubsanacion, err := ports.NuevaReferenciaHitoSubsanacionFiscalizacionRRHH(6)
	if err != nil {
		t.Fatal(err)
	}
	entrada, err := ports.NuevaEntradaDetalleExpedienteRRHHMinimizadaV3(
		datos.resumen, datos.solicitud, datos.analisis,
		referenciaAnalisisMinimizadaPrueba(t, 2), datos.cobertura,
		referenciaCoberturaMinimizadaPrueba(t, 3), datos.asignacion,
		referenciaAsignacionMinimizadaPrueba(t, 4),
		&ports.FiscalizacionOperativaRRHH{
			ResultadoClave: domain.FiscalizacionDesfavorable,
			Reparos: []ports.ReparoFiscalizacionOperativaRRHH{{
				Clave: "observaciones_fiscalizacion", Texto: "Falta justificar el coste.",
			}},
			RegistradaEn: fiscalizadaEn,
			Subsanacion: &ports.SubsanacionFiscalizacionOperativaRRHH{
				RegistradaEn: fiscalizadaEn.Add(time.Minute), Texto: "Justificación aportada.",
			},
		}, refFiscalizacion, refSubsanacion, datos.hitos,
	)
	if err != nil {
		t.Fatalf("crear V3: %v", err)
	}
	lectura := reciboDetalleMinimizadoPrueba(t, datos.resumen.ExpedienteRef, 6, datos.resumen.ActualizadoEn)
	detalle, err := ports.NuevoDetalleExpedienteRRHHMinimizado(entrada, lectura)
	if err != nil {
		t.Fatalf("reconstruir V3: %v", err)
	}
	canon, err := entrada.ExportarContenidoCanonicoParaSQL(datos.resumen.ActualizadoEn.Add(time.Minute))
	if err != nil || !strings.HasPrefix(string(canon.BytesCanonicos()), "VEC-CT-CONTENIDO-DETALLE-RRHH-V3\n") {
		t.Fatalf("canon V3 ausente: %v %q", err, canon.BytesCanonicos())
	}
	if detalle.Fiscalizacion == nil || detalle.Fiscalizacion.Subsanacion == nil ||
		detalle.Fiscalizacion.RegistradaEn != fiscalizadaEn ||
		detalle.Fiscalizacion.Subsanacion.RegistradaEn != fiscalizadaEn.Add(time.Minute) {
		t.Fatalf("bloque V3 perdido: %#v", detalle.Fiscalizacion)
	}
	contenido, err := json.Marshal(detalle)
	if err != nil {
		t.Fatal(err)
	}
	for _, esperado := range []string{
		`"fiscalizacion"`, `"resultado_clave":"desfavorable"`,
		`"clave":"observaciones_fiscalizacion"`, `"Justificación aportada."`,
	} {
		if !strings.Contains(string(contenido), esperado) {
			t.Fatalf("falta %s: %s", esperado, contenido)
		}
	}
	// La copia no comparte reparos ni subsanación con el detalle original.
	copia := detalle.Clonar()
	copia.Fiscalizacion.Reparos[0].Texto = "mutado"
	copia.Fiscalizacion.Subsanacion.Texto = "mutado"
	if detalle.Fiscalizacion.Reparos[0].Texto == "mutado" || detalle.Fiscalizacion.Subsanacion.Texto == "mutado" {
		t.Fatal("Clonar comparte el bloque fiscalización")
	}
}

func TestDetalleRRHHMinimizadoV3RechazaSubsanacionSinResultadoDesfavorable(t *testing.T) {
	t.Parallel()
	datos := datosDetalleMinimizadoPrueba(3)
	fiscalizadaEn := datos.hitos[3].RealizadaEn.Add(time.Minute)
	datos.hitos = append(datos.hitos, ports.HitoExpedienteRRHH{Secuencia: 5, VersionExpediente: 5,
		AccionClave: domain.AccionRegistrarFiscalizacion, RealizadaEn: fiscalizadaEn,
		FaseOrigen: datos.hitos[3].FaseDestino, FaseDestino: domain.FaseFiscalizacion,
		EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoEnCurso})
	datos.resumen.Version, datos.resumen.FaseClave, datos.resumen.ActualizadoEn = 5, domain.FaseFiscalizacion, fiscalizadaEn
	refFiscalizacion, _ := ports.NuevaReferenciaHitoFiscalizacionRRHH(5)
	refSubsanacion, _ := ports.NuevaReferenciaHitoSubsanacionFiscalizacionRRHH(5)
	_, err := ports.NuevaEntradaDetalleExpedienteRRHHMinimizadaV3(datos.resumen, datos.solicitud, datos.analisis,
		referenciaAnalisisMinimizadaPrueba(t, 2), datos.cobertura, referenciaCoberturaMinimizadaPrueba(t, 3), datos.asignacion,
		referenciaAsignacionMinimizadaPrueba(t, 4), &ports.FiscalizacionOperativaRRHH{
			ResultadoClave: domain.FiscalizacionFavorableConObservaciones,
			Reparos:        []ports.ReparoFiscalizacionOperativaRRHH{{Clave: "observaciones_fiscalizacion", Texto: "Observación."}},
			RegistradaEn:   fiscalizadaEn,
			Subsanacion:    &ports.SubsanacionFiscalizacionOperativaRRHH{RegistradaEn: fiscalizadaEn.Add(time.Second), Texto: "No procede."},
		}, refFiscalizacion, refSubsanacion, datos.hitos)
	if err == nil {
		t.Fatal("aceptó subsanación de resultado no desfavorable")
	}
}
