package ports_test

import (
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func entradaDetalleDerivadoPrueba(t *testing.T) (ports.EntradaDetalleExpedienteRRHHMinimizada, ports.ReciboLecturaRRHH) {
	t.Helper()
	d := datosDetalleMinimizadoPrueba(3)
	informeEn := d.hitos[3].RealizadaEn.Add(time.Minute)
	d.hitos = append(d.hitos,
		ports.HitoExpedienteRRHH{Secuencia: 5, VersionExpediente: 5,
			AccionClave: domain.AccionEmitirInformeJuridico, RealizadaEn: informeEn,
			FaseOrigen: d.hitos[3].FaseDestino, FaseDestino: domain.FaseInformeJuridico,
			EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoEnCurso},
		ports.HitoExpedienteRRHH{Secuencia: 6, VersionExpediente: 6,
			AccionClave: domain.AccionRegistrarFiscalizacion, RealizadaEn: informeEn.Add(time.Minute),
			FaseOrigen: domain.FaseInformeJuridico, FaseDestino: domain.FaseFiscalizacion,
			EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoEnCurso},
	)
	padreHitos := append([]ports.HitoExpedienteRRHH(nil), d.hitos...)
	padreRef, numeroPadre, creadoPadre := d.resumen.ExpedienteRef, d.resumen.NumeroVisible, d.resumen.CreadoEn
	creadoHijo := d.hitos[5].RealizadaEn.Add(time.Minute)
	d.resumen.ExpedienteRef, d.resumen.NumeroVisible = "expediente:rrhh:hijo:01", numeroPadre+"-P1"
	d.resumen.Version, d.resumen.CreadoEn, d.resumen.ActualizadoEn = 6, creadoHijo, creadoHijo
	d.resumen.FaseClave, d.resumen.EstadoClave = domain.FaseFiscalizacion, domain.EstadoEnCurso
	d.hitos = []ports.HitoExpedienteRRHH{{Secuencia: 6, VersionExpediente: 6,
		AccionClave: domain.AccionDerivarNombramiento, RealizadaEn: creadoHijo,
		FaseOrigen: domain.FaseFiscalizacion, FaseDestino: domain.FaseFiscalizacion,
		EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoEnCurso}}
	refFiscal, _ := ports.NuevaReferenciaHitoFiscalizacionRRHH(6)
	entrada, err := ports.NuevaEntradaDetalleExpedienteRRHHMinimizadaDerivadaV1(
		d.resumen, d.solicitud, d.analisis, referenciaAnalisisMinimizadaPrueba(t, 2),
		d.cobertura, referenciaCoberturaMinimizadaPrueba(t, 3),
		d.asignacion, referenciaAsignacionMinimizadaPrueba(t, 4),
		&ports.FiscalizacionOperativaRRHH{ResultadoClave: domain.FiscalizacionFavorable,
			RegistradaEn: padreHitos[5].RealizadaEn},
		refFiscal, ports.ReferenciaHitoSubsanacionFiscalizacionRRHH{}, d.hitos,
		ports.InstruccionCompartidaRRHH{
			DerivacionRef: "derivacion:ct:01", PadreRef: padreRef, NumeroVisiblePadre: numeroPadre,
			OfertaRef: "oferta:ct:01", NumeroPlaza: 1, ReciboDerivacionRef: "recibo:ct:01",
			HuellaPadreSHA256: strings.Repeat("a", 64), CreadoEnPadre: creadoPadre,
			ActualizadoEnPadre: padreHitos[5].RealizadaEn, HitosPadre: padreHitos,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return entrada, reciboDetalleMinimizadoPrueba(t, d.resumen.ExpedienteRef, 6, creadoHijo)
}

func TestDetalleDerivadoConservaDosCronologiasYCanonPropio(t *testing.T) {
	entrada, lectura := entradaDetalleDerivadoPrueba(t)
	detalle, err := ports.NuevoDetalleExpedienteRRHHMinimizado(entrada, lectura)
	if err != nil {
		t.Fatal(err)
	}
	if len(detalle.Hitos) != 1 || detalle.Hitos[0].Secuencia != 6 ||
		detalle.InstruccionCompartida == nil || len(detalle.InstruccionCompartida.HitosPadre) != 6 {
		t.Fatal("se mezclaron los actos propios y los originales")
	}
	canon, err := entrada.ExportarContenidoCanonicoParaSQL(detalle.Resumen.ActualizadoEn.Add(time.Minute))
	if err != nil || !strings.HasPrefix(string(canon.BytesCanonicos()), "VEC-CT-CONTENIDO-DETALLE-RRHH-V5\n") {
		t.Fatalf("canon derivado: %v", err)
	}
	segundo, err := detalle.ExportarContenidoCanonicoParaSQL(detalle.Resumen.ActualizadoEn.Add(time.Minute))
	if err != nil || segundo.HuellaSHA256() != canon.HuellaSHA256() {
		t.Fatalf("canon no reproducible: %v", err)
	}
}

func TestDetalleDerivadoRechazaPadreReciboYCronologiaMezclada(t *testing.T) {
	entrada, lectura := entradaDetalleDerivadoPrueba(t)
	nominal, err := ports.NuevoDetalleExpedienteRRHHMinimizado(entrada, lectura)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := ports.NuevaSolicitudDetalleRRHH(nominal.Resumen.ExpedienteRef, nominal.Resumen.Version)
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string]func(*ports.DetalleExpedienteRRHH){
		"hito padre adulterado": func(d *ports.DetalleExpedienteRRHH) {
			d.InstruccionCompartida.HitosPadre[3].FaseDestino = "otra_fase"
		},
		"recibo ausente": func(d *ports.DetalleExpedienteRRHH) {
			d.InstruccionCompartida.ReciboDerivacionRef = ""
		},
		"hito padre como propio": func(d *ports.DetalleExpedienteRRHH) {
			d.Hitos[0] = d.InstruccionCompartida.HitosPadre[5]
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			d := nominal.Clonar()
			alterar(&d)
			if d.ValidarContenidoPublicablePara(solicitud) == nil {
				t.Fatal("se admitió un detalle adulterado")
			}
		})
	}
}
