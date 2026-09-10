package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

const (
	organizacionSeguimientoExistente = "organizacion:desarrollo:dipgra"
	expedienteSeguimientoExistente   = "expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7"
)

// Valores medidos antes del parche sobre los fixtures existentes (2026-09-10).
// seguimiento.go SHA256 318e9d682e157766fc6200adc7a8020c9ae0c2d09db2f4d4f53ea74a1797218a.
// seguimiento_canon.go SHA256 bfcea7d70db69bcb074a2e96d66611e0823c2bec74cf1dcaab79001274f3c5f9.
func TestSeguimientoReferenciasLegacyPreservaCanon(t *testing.T) {
	definicion := definicionSeguimientoValida(t, false)
	if definicion.Referencia().HuellaSHA256 != "6fa7ce07a51c966667b5de2151ad8e4fefeae5559d80b559f07c3648e4d00055" {
		t.Fatal("cambió la huella previa de definición")
	}
	for _, caso := range []struct {
		nombre      string
		seguimiento Seguimiento
		longitud    int
		huella      string
	}{
		{"inicial", seguimientoNuevoValido(t, definicion), 648, "638f62c09923a45b774a5347766fa0a87a53bb08a3d739c175d2ccaba4036c77"},
		{"incorporado", seguimientoIncorporado(t, definicion), 1763, "36e61167d72c7728682a20872924ea5d94dc19d3047ddd668d241c8671e17f56"},
		{"completo", seguimientoCompletoParaRehidratar(t, definicion), 5128, "85a0b5e4d0034eba2479c82571381bf91d9c1e4825635ad5d898e804e4d0f152"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			material := comprobarRestauracionReferenciasSeguimiento(t, definicion, caso.seguimiento)
			if len(material) != caso.longitud || resumenSeguimiento(material) != caso.huella {
				t.Fatal("cambiaron los bytes canónicos previos, incluidas las huellas de actuaciones")
			}
			if caso.seguimiento.Estado().HuellaRaizSHA256 != "61cf5ba80f07e770ae1ddd563505ac590a6d5ce781e9c5db2707e9e919bdb35a" {
				t.Fatal("cambió la huella previa de raíz")
			}
		})
	}
}

func TestSeguimientoReferenciasExpedienteExistenteConservaIdentificadores(t *testing.T) {
	definicion := definicionSeguimientoValida(t, false)
	alta := altaReferenciasSeguimientoPrueba(t, definicion)
	alta.OrganizacionRef = organizacionSeguimientoExistente
	alta.ExpedienteRef = expedienteSeguimientoExistente
	inicial, err := NuevoSeguimiento(definicion, alta)
	if err != nil {
		t.Fatal(err)
	}
	datos := seguimientoIncorporado(t, definicion).Actuaciones()[0].datos()
	aplicado, err := inicial.Aplicar(definicion, 0, datos)
	if err != nil {
		t.Fatal(err)
	}
	for _, seguimiento := range []Seguimiento{inicial, aplicado} {
		estado := seguimiento.Estado()
		if estado.OrganizacionRef != alta.OrganizacionRef || estado.ExpedienteRef != alta.ExpedienteRef {
			t.Fatal("se sustituyeron identificadores existentes")
		}
		material := comprobarRestauracionReferenciasSeguimiento(t, definicion, seguimiento)
		for _, referencia := range []string{alta.OrganizacionRef, alta.ExpedienteRef} {
			if !bytes.Contains(material, []byte(referencia)) {
				t.Fatal("el material canónico no conserva la referencia exacta")
			}
		}
	}
	if aplicado.Version() != 1 || len(aplicado.Actuaciones()) != 1 ||
		aplicado.Estado().HuellaRaizSHA256 != inicial.Estado().HuellaRaizSHA256 {
		t.Fatal("la incorporación alteró la raíz o duplicó la actuación")
	}
}

func TestSeguimientoReferenciasExpedienteRechazaFormatosAjenos(t *testing.T) {
	definicion := definicionSeguimientoValida(t, false)
	base := seguimientoNuevoValido(t, definicion).Estado()
	cero := strings.Repeat("0", 64)
	for _, campo := range []struct {
		nombre        string
		invalidos     []string
		asignarAlta   func(*AltaSeguimiento, string)
		asignarEstado func(*EstadoPersistidoSeguimiento, string)
	}{
		{"organizacion", []string{"", "12345678Z", "organizacion:", "organización:dipgra", "organizacion:con espacio", " organizacion:dipgra", "organizacion:dipgra\n", "Organizacion:dipgra", "ref:" + cero, "organizacion:" + strings.Repeat("a", 148), expedienteSeguimientoExistente},
			func(a *AltaSeguimiento, v string) { a.OrganizacionRef = v },
			func(e *EstadoPersistidoSeguimiento, v string) { e.OrganizacionRef = v }},
		{"expediente", []string{"", "12345678Z", "expediente:ct:", "expediente:ct:" + cero, "ref:" + cero, "expediente:otro:" + strings.Repeat("a", 64), "expediente:ct:" + strings.Repeat("A", 64), "expediente:ct:" + strings.Repeat("g", 64), expedienteSeguimientoExistente[:76], expedienteSeguimientoExistente + "a", organizacionSeguimientoExistente},
			func(a *AltaSeguimiento, v string) { a.ExpedienteRef = v },
			func(e *EstadoPersistidoSeguimiento, v string) { e.ExpedienteRef = v }},
	} {
		t.Run(campo.nombre, func(t *testing.T) {
			for _, invalido := range campo.invalidos {
				alta := altaReferenciasSeguimientoPrueba(t, definicion)
				campo.asignarAlta(&alta, invalido)
				if _, err := NuevoSeguimiento(definicion, alta); !errors.Is(err, ErrSeguimientoInvalido) {
					t.Fatalf("alta aceptó formato ajeno: %v", err)
				}
				estado := base.clonar()
				campo.asignarEstado(&estado, invalido)
				if _, err := calcularHuellaRaizSeguimiento(estado); !errors.Is(err, ErrSeguimientoInvalido) {
					t.Fatalf("canon raíz aceptó formato ajeno: %v", err)
				}
				if _, err := materialCanonicoEstadoSeguimiento(estado); !errors.Is(err, ErrSeguimientoInvalido) {
					t.Fatalf("canon estado aceptó formato ajeno: %v", err)
				}
				if _, err := RehidratarSeguimiento(definicion, estado); !errors.Is(err, ErrSeguimientoInvalido) {
					t.Fatalf("restauración aceptó formato ajeno: %v", err)
				}
			}
		})
	}
}

func TestSeguimientoReferenciasRestoConservaFormatoEstricto(t *testing.T) {
	definicion := definicionSeguimientoValida(t, false)
	incorporado := seguimientoIncorporado(t, definicion)
	for _, valor := range []string{organizacionSeguimientoExistente, expedienteSeguimientoExistente, "12345678Z", "ref:" + strings.Repeat("0", 64)} {
		if referenciaOpacaSeguimientoValida(valor) {
			t.Fatal("se relajó el validador general de seguimiento")
		}
		for _, asignar := range []func(*AltaSeguimiento){
			func(a *AltaSeguimiento) { a.Referencia = valor },
			func(a *AltaSeguimiento) { a.RelacionRef = valor },
		} {
			alta := altaReferenciasSeguimientoPrueba(t, definicion)
			asignar(&alta)
			if _, err := NuevoSeguimiento(definicion, alta); !errors.Is(err, ErrSeguimientoInvalido) {
				t.Fatalf("referencia o relación dejaron de exigir ref:hash: %v", err)
			}
		}
		for _, asignar := range []func(*DatosTransicionSeguimiento){
			func(d *DatosTransicionSeguimiento) { d.ActuacionRef = valor },
			func(d *DatosTransicionSeguimiento) { d.ActorRef = valor },
			func(d *DatosTransicionSeguimiento) { d.UnidadRef = valor },
			func(d *DatosTransicionSeguimiento) { d.ReciboRef = valor },
			func(d *DatosTransicionSeguimiento) { d.CorrelacionRef = valor },
			func(d *DatosTransicionSeguimiento) { d.RectificaActuacionRef = valor },
			func(d *DatosTransicionSeguimiento) { d.Documentos[0].Referencia = valor },
			func(d *DatosTransicionSeguimiento) {
				d.Calendario = calendarioSeguimiento(d.RegistradaEn)
				d.Calendario.Referencia = valor
			},
		} {
			datos := incorporado.Actuaciones()[0].datos()
			asignar(&datos)
			if _, err := normalizarDatosTransicionSeguimiento(datos); !errors.Is(err, ErrTransicionInvalida) {
				t.Fatalf("otra referencia de transición dejó de exigir ref:hash: %v", err)
			}
		}
		referencia := definicion.Referencia()
		referencia.Referencia = valor
		if referencia.Validar() == nil {
			t.Fatal("definición dejó de exigir ref:hash")
		}
		for _, asignar := range []func(*EstadoPersistidoSeguimiento){
			func(e *EstadoPersistidoSeguimiento) { e.Referencia = valor },
			func(e *EstadoPersistidoSeguimiento) { e.RelacionRef = valor },
			func(e *EstadoPersistidoSeguimiento) { e.PeriodosResultantes[0].ActuacionRef = valor },
			func(e *EstadoPersistidoSeguimiento) { e.CeseEfectivo.ActuacionRef = valor },
		} {
			estado := seguimientoCesado(t, definicion).Estado()
			asignar(&estado)
			if _, err := materialCanonicoEstadoSeguimiento(estado); !errors.Is(err, ErrSeguimientoInvalido) {
				t.Fatalf("otra referencia del canon dejó de exigir ref:hash: %v", err)
			}
		}
	}
}

func altaReferenciasSeguimientoPrueba(t *testing.T, definicion DefinicionSeguimiento) AltaSeguimiento {
	t.Helper()
	estado := seguimientoNuevoValido(t, definicion).Estado()
	return AltaSeguimiento{
		Referencia: estado.Referencia, OrganizacionRef: estado.OrganizacionRef,
		ExpedienteRef: estado.ExpedienteRef, RelacionRef: estado.RelacionRef,
		PeriodoPrevisto: estado.PeriodoPrevisto, CreadoEn: estado.CreadoEn,
	}
}

func comprobarRestauracionReferenciasSeguimiento(t *testing.T, definicion DefinicionSeguimiento, original Seguimiento) []byte {
	t.Helper()
	estado := original.Estado()
	material, err := SerializarEstadoSeguimientoCanonico(definicion, estado)
	if err != nil {
		t.Fatal(err)
	}
	codificado, err := json.Marshal(estado)
	if err != nil {
		t.Fatal(err)
	}
	var persistido EstadoPersistidoSeguimiento
	if err := json.Unmarshal(codificado, &persistido); err != nil {
		t.Fatal(err)
	}
	restaurado, err := RehidratarSeguimiento(definicion, persistido)
	if err != nil {
		t.Fatal(err)
	}
	recuperado := restaurado.Estado()
	if recuperado.OrganizacionRef != estado.OrganizacionRef ||
		recuperado.ExpedienteRef != estado.ExpedienteRef || recuperado.Version != estado.Version ||
		recuperado.HuellaRaizSHA256 != estado.HuellaRaizSHA256 {
		t.Fatal("restauración cambió referencias, versión o raíz")
	}
	materialRestaurado, err := SerializarEstadoSeguimientoCanonico(definicion, recuperado)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(material, materialRestaurado) {
		t.Fatal("restauración cambió bytes o huellas canónicas")
	}
	return material
}
