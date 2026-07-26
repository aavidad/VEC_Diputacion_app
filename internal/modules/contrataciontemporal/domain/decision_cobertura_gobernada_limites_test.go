package domain

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestHistoriaDecisionRechazaNoOpReselladoCoherente(t *testing.T) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	inicial := adoptarDecisionGobernada(t, base, propuesta)
	nueva := propuestaDecisionParaExpediente(
		t,
		inicial,
		inicial.ActualizadoEn.Add(time.Minute),
	)
	datos := datosRectificacion(inicial)
	datos.ViaElegida = "via_alternativa_configurable"
	acto := actuacionDecision(
		string(AccionRectificarCoberturaGobernada),
		"actor_rrhh_rectificador_02",
		inicial.FaseActual,
		inicial.ActualizadoEn.Add(2*time.Minute),
	)
	rectificado, err := inicial.RectificarDecisionCoberturaGobernada(
		inicial.Version,
		datos,
		nueva,
		acto,
	)
	if err != nil {
		t.Fatalf("preparar rectificación: %v", err)
	}

	adulterado := rectificado.Clonar()
	publicacion := adulterado.DecisionesCobertura[1]
	publicacion.ViaElegida = adulterado.DecisionesCobertura[0].ViaElegida
	resellarDecisionCoberturaPrueba(t, &publicacion)
	adulterado.DecisionesCobertura[1] = publicacion
	adulterado.ViaCobertura.ViaClave = publicacion.ViaElegida
	adulterado.ViaCobertura.DecisionGobernada = &publicacion
	if !errors.Is(adulterado.Validar(), ErrExpedienteInvalido) {
		t.Fatal("aceptó rectificación re-sellada sin cambio funcional de vía")
	}
}

func TestCanonDecisionAdmiteExtremosTemporalesInteroperables(
	t *testing.T,
) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	decidido := adoptarDecisionGobernada(t, base, propuesta)
	original := decidido.DecisionesCobertura[0]
	extremos := []struct {
		nombre   string
		instante time.Time
	}{
		{
			nombre: "mínimo",
			instante: time.Date(
				1, 1, 1, 0, 0, 0, 1000, time.UTC,
			),
		},
		{
			nombre: "máximo",
			instante: time.Date(
				9999, 12, 31, 23, 59, 59, 999999000, time.UTC,
			),
		},
	}
	for _, extremo := range extremos {
		t.Run(extremo.nombre, func(t *testing.T) {
			publicacion := original
			publicacion.DecididaEn = extremo.instante
			publicacion.Actuacion.RealizadaEn = extremo.instante
			resellarDecisionCoberturaPrueba(t, &publicacion)
			restaurada, err := RestaurarDecisionCoberturaGobernada(publicacion)
			if err != nil || restaurada.Publicacion() != publicacion {
				t.Fatalf("restaurar extremo %s: %v", extremo.nombre, err)
			}
		})
	}
}

func TestCanonDecisionRechazaAniosFueraDelRangoInteroperable(
	t *testing.T,
) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	decidido := adoptarDecisionGobernada(t, base, propuesta)
	casos := []struct {
		nombre   string
		instante time.Time
	}{
		{
			nombre:   "año cero",
			instante: time.Date(0, 12, 31, 23, 59, 59, 999999000, time.UTC),
		},
		{
			nombre:   "año diez mil",
			instante: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			publicacion := decidido.DecisionesCobertura[0]
			publicacion.DecididaEn = caso.instante
			publicacion.Actuacion.RealizadaEn = caso.instante
			if _, err := RestaurarDecisionCoberturaGobernada(publicacion); !errors.Is(err, ErrDatoInvalido) {
				t.Fatalf("restauró instante fuera de rango: %v", err)
			}
		})
	}
}

func TestExpedienteRehidrataDecisionEnMaximoInteroperable(t *testing.T) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	original := adoptarDecisionGobernada(t, base, propuesta)
	maximo := time.Date(
		9999, 12, 31, 23, 59, 59, 999999000, time.UTC,
	)
	publicacion := original.DecisionesCobertura[0]
	publicacion.DecididaEn = maximo
	publicacion.Actuacion.RealizadaEn = maximo
	resellarDecisionCoberturaPrueba(t, &publicacion)

	original.DecisionesCobertura[0] = publicacion
	original.ViaCobertura.DecisionGobernada = &publicacion
	ultima := len(original.Actuaciones) - 1
	original.Actuaciones[ultima].RealizadaEn = maximo
	original.ActualizadoEn = maximo
	contenido, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("serializar extremo: %v", err)
	}
	var restaurado Expediente
	if err := json.Unmarshal(contenido, &restaurado); err != nil {
		t.Fatalf("decodificar extremo: %v", err)
	}
	if err := restaurado.Validar(); err != nil {
		t.Fatalf("rehidratar extremo interoperable: %v", err)
	}

	fuera := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	restaurado.DecisionesCobertura[0].DecididaEn = fuera
	restaurado.DecisionesCobertura[0].Actuacion.RealizadaEn = fuera
	restaurado.ViaCobertura.DecisionGobernada =
		&restaurado.DecisionesCobertura[0]
	restaurado.Actuaciones[ultima].RealizadaEn = fuera
	restaurado.ActualizadoEn = fuera
	if !errors.Is(restaurado.Validar(), ErrExpedienteInvalido) {
		t.Fatal("rehidrató decisión fuera del rango interoperable")
	}
}
