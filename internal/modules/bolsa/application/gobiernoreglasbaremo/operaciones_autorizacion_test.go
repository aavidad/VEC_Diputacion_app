package gobiernoreglasbaremo

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
)

func TestOperacionesYPoliticasCoincidenConContratoReservado(t *testing.T) {
	t.Parallel()
	type caso struct {
		operacion OperacionGobiernoReglasBaremoV2
		nombre    string
		accion    string
		finalidad string
		campos    []string
	}
	casos := []caso{
		{OperacionAltaBorrador, "alta_borrador", "bolsa.reglas_baremo.borrador.crear", finalidadGobiernoReglas, camposGobiernoReglas},
		{OperacionPublicar, "publicar", "bolsa.reglas_baremo.publicar", finalidadGobiernoReglas, camposGobiernoReglas},
		{OperacionActivar, "activar", "bolsa.reglas_baremo.activar", finalidadGobiernoReglas, camposGobiernoReglas},
		{OperacionSustituir, "sustituir", "bolsa.reglas_baremo.sustituir", finalidadGobiernoReglas, camposGobiernoReglas},
		{OperacionRetirar, "retirar", "bolsa.reglas_baremo.retirar", finalidadGobiernoReglas, camposGobiernoReglas},
		{OperacionDescartar, "descartar", "bolsa.reglas_baremo.descartar", finalidadGobiernoReglas, camposGobiernoReglas},
		{OperacionConsultaExacta, "consultar_version_exacta", "bolsa.reglas_baremo.version.consultar", finalidadConsultaReglas, camposConsultaReglas},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			nombre, err := caso.operacion.nombreCanonico()
			especificacion, errEspecificacion := especificacionPara(caso.operacion)
			if err != nil || errEspecificacion != nil || nombre != caso.nombre ||
				especificacion.accion != caso.accion ||
				especificacion.finalidad != caso.finalidad ||
				!reflect.DeepEqual(especificacion.campos, caso.campos) {
				t.Fatalf("contrato divergente: nombre=%q especificacion=%#v errores=%v/%v", nombre, especificacion, err, errEspecificacion)
			}
			if _, err := aplicacionvec.NuevaPoliticaUsoDecisionAutorizacion(
				caso.accion,
				moduloBolsaGobiernoReglas,
				tipoRecursoReglasGobernadas,
				caso.finalidad,
				caso.campos,
				aplicacionvec.PerfilProteccionUsoAutorizacionInternoAlto,
			); err != nil {
				t.Fatalf("la politica interna-alta exacta no compila: %v", err)
			}
		})
	}
	for _, operacion := range []OperacionGobiernoReglasBaremoV2{
		OperacionNoDeclarada,
		255,
	} {
		if operacion.valida() {
			t.Fatalf("operacion libre aceptada: %d", operacion)
		}
		if _, err := operacion.nombreCanonico(); !errors.Is(err, ErrOperacionInvalida) {
			t.Fatalf("operacion libre sin error cerrado: %d, %v", operacion, err)
		}
	}
}

func TestContratoUsaRecursoExactoConAlcanceMinimoYCopiaMapas(t *testing.T) {
	t.Parallel()
	huella := strings.Repeat("a", 64)
	version := borradorPrueba(t)
	alcance, err := nuevoAlcanceAutorizacionDesdeVersion(version)
	if err != nil {
		t.Fatal(err)
	}
	contrato, err := nuevoContratoAutorizacionV2(OperacionPublicar, huella, alcance)
	if err != nil {
		t.Fatal(err)
	}
	recurso, err := contrato.Recurso()
	if err != nil || recurso.Referencia != "reglas-baremo:"+huella ||
		recurso.ModuloID != "bolsa" || recurso.Tipo != "version_reglas_baremo_gobernada" ||
		len(recurso.Ambitos) != 2 || len(recurso.Atributos) != 0 ||
		recurso.Ambitos[ambitoConvocatoriaRef] != referenciaConvocatoriaPlanPrueba ||
		recurso.Ambitos[ambitoExpedienteRef] != referenciaExpedientePlanPrueba {
		t.Fatalf("recurso inesperado: %#v, %v", recurso, err)
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || !huellaSHA256Valida(huellaContexto) {
		t.Fatalf("contexto acotado no canonico: %q, %v", huellaContexto, err)
	}
	recurso.Ambitos["sujeto_ref"] = "dni:12345678z"
	recurso.Atributos["nombre"] = "persona"
	segunda, err := contrato.Recurso()
	if err != nil || len(segunda.Ambitos) != 2 || len(segunda.Atributos) != 0 ||
		segunda.Ambitos[ambitoConvocatoriaRef] != referenciaConvocatoriaPlanPrueba {
		t.Fatalf("el contrato compartio sus mapas: %#v, %v", segunda, err)
	}
	if _, err := contrato.Politica(); err != nil {
		t.Fatalf("politica cerrada no disponible: %v", err)
	}

	for _, invalida := range []string{"", strings.Repeat("A", 64), strings.Repeat("a", 63), strings.Repeat("g", 64)} {
		if _, err := nuevoContratoAutorizacionV2(OperacionPublicar, invalida, alcance); !errors.Is(err, ErrContratoAutorizacionInvalido) {
			t.Fatalf("huella libre aceptada: %q, %v", invalida, err)
		}
	}
}

func TestConsultaExactaConservaTodoElVinculo(t *testing.T) {
	t.Parallel()
	version := borradorPrueba(t)
	vinculo := vinculoPrueba(t, version)
	conjunto, err := version.Conjunto()
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := nuevoDescriptorConsultaExactaReglasBaremoV2(
		vinculo, conjunto.Identidad(),
	)
	if err != nil {
		t.Fatal(err)
	}
	consulta, err := nuevaConsultaExactaReglasBaremoV2(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	recuperado, err := consulta.Vinculo()
	if err != nil || !vinculosIguales(vinculo, recuperado) {
		t.Fatalf("vinculo alterado: %#v, %v", recuperado, err)
	}
	identidadRecuperada, err := consulta.Identidad()
	if err != nil || identidadRecuperada.Referencia() != conjunto.Identidad().Referencia() ||
		identidadRecuperada.Version() != conjunto.Identidad().Version() ||
		identidadRecuperada.ConvocatoriaRef() != conjunto.Identidad().ConvocatoriaRef() ||
		identidadRecuperada.ExpedienteRef() != conjunto.Identidad().ExpedienteRef() {
		t.Fatalf("identidad alterada: %#v, %v", identidadRecuperada, err)
	}
	contrato, err := consulta.ContratoAutorizacionV2()
	if err != nil {
		t.Fatal(err)
	}
	recurso, err := contrato.Recurso()
	if err != nil || recurso.Referencia != "reglas-baremo:"+vinculo.HuellaEstadoSHA256() ||
		recurso.Ambitos[ambitoConvocatoriaRef] != conjunto.Identidad().ConvocatoriaRef() ||
		recurso.Ambitos[ambitoExpedienteRef] != conjunto.Identidad().ExpedienteRef() {
		t.Fatalf("recurso de consulta no exacto: %#v, %v", recurso, err)
	}
	if _, err := nuevaConsultaExactaReglasBaremoV2(DescriptorConsultaExactaReglasBaremoV2{}); !errors.Is(err, ErrConsultaExactaInvalida) {
		t.Fatalf("descriptor cero aceptado: %v", err)
	}
	mezclada, err := reglas.NuevaIdentidadConjuntoReglasBaremo(
		"rgl_dddddddddddddddddddddddddddddddd", 1,
		"con_eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		"exp_ffffffffffffffffffffffffffffffff",
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nuevoDescriptorConsultaExactaReglasBaremoV2(vinculo, mezclada); !errors.Is(err, ErrConsultaExactaInvalida) {
		t.Fatalf("mezcla de vinculo y alcance aceptada: %v", err)
	}
}
