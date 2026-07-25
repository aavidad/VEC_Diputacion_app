package cobertura

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestPreimagenesOperacionDecisionCoberturaDistinguenSemanticaExacta(
	t *testing.T,
) {
	base := identidadRectificacionDecisionCoberturaPrueba()
	preimagenesBase, err := NuevasPreimagenesOperacionDecisionCobertura(base)
	if err != nil {
		t.Fatal(err)
	}
	ambitoBase, _ := preimagenesBase.BytesAmbito()
	semanticaBase, _ := preimagenesBase.BytesSemantica()
	otraHuella := strings.Repeat("7", 64)
	casos := map[string]func(*DatosIdentidadOperacionDecisionCobertura){
		"tipo y acción": func(d *DatosIdentidadOperacionDecisionCobertura) {
			d.tipo = domain.DecisionCoberturaInicial
			d.accion = domain.AccionDecidirCoberturaGobernada
			d.motivo = domain.MotivoGobernadoDecisionCobertura{}
			d.predecesoraRef = ""
			d.predecesoraHuella = ""
		},
		"versión": func(d *DatosIdentidadOperacionDecisionCobertura) {
			d.versionExpediente++
		},
		"vía": func(d *DatosIdentidadOperacionDecisionCobertura) {
			d.viaElegida = "via_distinta"
		},
		"identidad semántica": func(d *DatosIdentidadOperacionDecisionCobertura) {
			d.identidadSemantica.HuellaSHA256 = otraHuella
			d.identidadSemantica.Referencia =
				"propuesta-cobertura-semantica:sha256:" + otraHuella
		},
		"motivo funcional": func(d *DatosIdentidadOperacionDecisionCobertura) {
			d.motivo = motivoOperacionDecisionCoberturaPrueba("8")
		},
		"predecesora": func(d *DatosIdentidadOperacionDecisionCobertura) {
			d.predecesoraHuella = otraHuella
			d.predecesoraRef = "decision-cobertura:sha256:" + otraHuella
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			datos := base
			mutar(&datos)
			preimagenes, err := NuevasPreimagenesOperacionDecisionCobertura(datos)
			if err != nil {
				t.Fatalf("mutación válida rechazada: %v", err)
			}
			ambito, _ := preimagenes.BytesAmbito()
			semantica, _ := preimagenes.BytesSemantica()
			if !bytes.Equal(ambito, ambitoBase) ||
				bytes.Equal(semantica, semanticaBase) {
				t.Fatal("la identidad no separó ámbito estable y semántica exacta")
			}
		})
	}
}

func TestPreimagenesOperacionDecisionCoberturaSonDefensivasYOpacas(t *testing.T) {
	preimagenes, err := NuevasPreimagenesOperacionDecisionCobertura(
		identidadOperacionDecisionCoberturaPrueba(),
	)
	if err != nil {
		t.Fatal(err)
	}
	primera, err := preimagenes.BytesSemantica()
	if err != nil {
		t.Fatal(err)
	}
	primera[0] ^= 1
	segunda, err := preimagenes.BytesSemantica()
	if err != nil || bytes.Equal(primera, segunda) {
		t.Fatal("la preimagen compartió memoria mutable")
	}
	if _, err := json.Marshal(preimagenes); !errors.Is(
		err, ErrSerializacionOperacionDecisionCoberturaProhibida,
	) {
		t.Fatalf("serialización aceptada: %v", err)
	}
	texto := fmt.Sprintf("%v %#v", preimagenes, preimagenes)
	if texto != redaccionOperacionDecisionCobertura+" "+
		redaccionOperacionDecisionCobertura ||
		strings.Contains(texto, "actor_rrhh") {
		t.Fatalf("formato no redactado: %q", texto)
	}
}

func TestSellosOperacionDecisionCoberturaExigenGeneracionesAlineadas(t *testing.T) {
	sellos := sellosOperacionDecisionCoberturaPrueba(t)
	ambitoActivo, semanticaActiva, err := sellos.parActivo()
	if err != nil || !sellos.contienePar(ambitoActivo, semanticaActiva) {
		t.Fatalf("par activo rechazado: %v", err)
	}
	ambitos, _ := sellos.AmbitosIdempotenciaHMAC.Datos()
	semanticas, _ := sellos.HuellasSemanticasHMAC.Datos()
	if sellos.contienePar(
		ambitos.Activo.Valor,
		semanticas.Retenidos[0].Valor,
	) {
		t.Fatal("se combinaron generaciones distintas")
	}
	desalineada, err := ports.NuevaColeccionSellosHMAC(
		selloOperacionDecisionCoberturaPrueba(
			dominioSemanticaOperacionDecisionCobertura, 3, "e",
		),
		[]string{selloOperacionDecisionCoberturaPrueba(
			dominioSemanticaOperacionDecisionCobertura, 1, "d",
		)},
	)
	if err != nil {
		t.Fatal(err)
	}
	sellos.HuellasSemanticasHMAC = desalineada
	if sellos.Validar() == nil {
		t.Fatal("historias generacionales desalineadas aceptadas")
	}
}

func TestTokenPropietarioOperacionDecisionCoberturaEsCSPRNGOpaco(t *testing.T) {
	primero, err := GenerarTokenPropietarioOperacionDecisionCobertura()
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := GenerarTokenPropietarioOperacionDecisionCobertura()
	if err != nil {
		t.Fatal(err)
	}
	huellaPrimero, err := primero.HuellaSHA256()
	huellaSegundo, errSegundo := segundo.HuellaSHA256()
	if err != nil || errSegundo != nil || huellaPrimero == huellaSegundo ||
		!primero.CoincideConHuellaSHA256(huellaPrimero) ||
		primero.CoincideConHuellaSHA256(huellaSegundo) {
		t.Fatalf("token/hash incoherente: %v %v", err, errSegundo)
	}
	if _, err := json.Marshal(primero); !errors.Is(
		err, ErrSerializacionOperacionDecisionCoberturaProhibida,
	) {
		t.Fatalf("token serializable: %v", err)
	}
	texto := fmt.Sprintf("%v %#v", primero, primero)
	if texto != redaccionOperacionDecisionCobertura+" "+
		redaccionOperacionDecisionCobertura ||
		strings.Contains(texto, huellaPrimero) {
		t.Fatalf("token no redactado: %q", texto)
	}
	var cero TokenPropietarioOperacionDecisionCobertura
	if _, err := cero.HuellaSHA256(); !errors.Is(
		err, ErrTokenPropietarioOperacionDecisionCoberturaInvalido,
	) {
		t.Fatalf("token cero aceptado: %v", err)
	}
}

func TestDatosIdentidadOperacionDecisionCoberturaRechazanAtaques(t *testing.T) {
	base := identidadRectificacionDecisionCoberturaPrueba()
	casos := map[string]func(*DatosIdentidadOperacionDecisionCobertura){
		"clave": func(d *DatosIdentidadOperacionDecisionCobertura) {
			d.claveIdempotencia = "no-es-uuid-v4"
		},
		"versión cero": func(d *DatosIdentidadOperacionDecisionCobertura) {
			d.versionExpediente = 0
		},
		"versión no interoperable": func(d *DatosIdentidadOperacionDecisionCobertura) {
			d.versionExpediente = MaximoEnteroSeguroOperacionDecisionCobertura
		},
		"acción cruzada": func(d *DatosIdentidadOperacionDecisionCobertura) {
			d.accion = domain.AccionDecidirCoberturaGobernada
		},
		"motivo ausente": func(d *DatosIdentidadOperacionDecisionCobertura) {
			d.motivo = domain.MotivoGobernadoDecisionCobertura{}
		},
		"predecesora parcial": func(d *DatosIdentidadOperacionDecisionCobertura) {
			d.predecesoraHuella = ""
		},
		"huella semántica": func(d *DatosIdentidadOperacionDecisionCobertura) {
			d.identidadSemantica.HuellaSHA256 = strings.Repeat("0", 64)
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			datos := base
			mutar(&datos)
			if datos.Validar() == nil {
				t.Fatal("identidad insegura aceptada")
			}
		})
	}
}

func TestTiposOpacosOperacionDecisionCoberturaNoExponenCamposSecretos(t *testing.T) {
	tipoToken := reflect.TypeOf(TokenPropietarioOperacionDecisionCobertura{})
	if campo, existe := tipoToken.FieldByName("Secreto"); existe || campo.Name != "" {
		t.Fatal("el secreto del token quedó exportado")
	}
	tipoPreimagenes := reflect.TypeOf(PreimagenesOperacionDecisionCobertura{})
	if _, existe := tipoPreimagenes.FieldByName("Ambito"); existe {
		t.Fatal("la preimagen de ámbito quedó exportada")
	}
}

func FuzzDatosIdentidadOperacionDecisionCoberturaNoEntraEnPanico(f *testing.F) {
	f.Add(uint64(1), "bolsa_vigente")
	f.Add(MaximoEnteroSeguroOperacionDecisionCobertura, "via")
	f.Fuzz(func(t *testing.T, version uint64, via string) {
		datos := identidadOperacionDecisionCoberturaPrueba()
		datos.versionExpediente = version
		datos.viaElegida = domain.ClaveCatalogo(via)
		_ = datos.Validar()
		_, _ = NuevasPreimagenesOperacionDecisionCobertura(datos)
	})
}
